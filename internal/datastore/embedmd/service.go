package embedmd

import (
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/golang/glog"
	"github.com/kubeflow/hub/internal/db/models"
	"github.com/kubeflow/hub/internal/db/service"
	"github.com/kubeflow/hub/internal/platform/datastore"
	db "github.com/kubeflow/hub/internal/platform/db"
	_ "github.com/kubeflow/hub/internal/platform/db/drivers"
	"github.com/kubeflow/hub/internal/platform/db/types"
	"github.com/kubeflow/hub/internal/platform/tls"
	"gorm.io/gorm"
)

const connectorType = "embedmd"

func init() {
	datastore.Register(connectorType, func(cfg any) (datastore.Connector, error) {
		emdbCfg, ok := cfg.(*EmbedMDConfig)
		if !ok {
			return nil, fmt.Errorf("invalid EmbedMD config type (%T)", cfg)
		}

		if err := emdbCfg.Validate(); err != nil {
			return nil, fmt.Errorf("invalid EmbedMD config: %w", err)
		}

		return NewEmbedMDService(cfg.(*EmbedMDConfig))
	})
}

type EmbedMDConfig struct {
	DatabaseType string
	DatabaseDSN  string
	TLSConfig    *tls.TLSConfig

	// DB is an already connected database instance that, if provided, will
	// be used instead of making a new connection.
	DB *gorm.DB

	// WaitForMigrations instructs Connect to wait for the migrations to
	// run instead of running them itself.
	WaitForMigrations bool
}

func (c *EmbedMDConfig) Validate() error {
	if c.DB == nil {
		if c.DatabaseType != types.DatabaseTypeMySQL && c.DatabaseType != types.DatabaseTypePostgres {
			return fmt.Errorf("unsupported database type: %s. Supported types: %s, %s", c.DatabaseType, types.DatabaseTypeMySQL, types.DatabaseTypePostgres)
		}

		switch c.DatabaseType {
		case types.DatabaseTypeMySQL:
			// Reject DSNs with multiple '?'
			// query components prohibited in database name (e.g. "DB_NAME?tls=preferred")
			// we append "?charset=utf8mb4" automatically to MySQL DSNs
			if strings.Count(c.DatabaseDSN, "?") > 1 {
				return fmt.Errorf("invalid MySQL DSN: database name must not contain '?' characters; do not put query parameters in the database name")
			}

			// Validate that the DSN can be parsed before continuing
			if _, err := mysql.ParseDSN(c.DatabaseDSN); err != nil {
				return fmt.Errorf("invalid MySQL DSN: %w", err)
			}

		case types.DatabaseTypePostgres:
			// Support both URL and key=value DSN formats.
			if strings.HasPrefix(c.DatabaseDSN, "postgres://") || strings.HasPrefix(c.DatabaseDSN, "postgresql://") {
				parsed, err := url.Parse(c.DatabaseDSN)
				if err != nil {
					return fmt.Errorf("invalid PostgreSQL DSN: %w", err)
				}
				// Path is "/dbname"; ensure db name doesn't contain '?'
				dbName := strings.TrimPrefix(parsed.Path, "/")
				if strings.Contains(dbName, "?") {
					return fmt.Errorf("invalid PostgreSQL DSN: database name must not contain '?' characters; do not put query parameters in the database name")
				}
			} else {
				// key=value format: find dbname=...
				for p := range strings.FieldsSeq(c.DatabaseDSN) {
					if name, ok := strings.CutPrefix(p, "dbname="); ok {
						if strings.Contains(name, "?") {
							return fmt.Errorf("invalid PostgreSQL DSN: database name must not contain '?' characters; do not put query parameters in the database name")
						}
						break
					}
				}
			}
		}
	}

	return nil
}

type EmbedMDService struct {
	dbConnector       db.Connector
	waitForMigrations bool
}

func NewEmbedMDService(cfg *EmbedMDConfig) (*EmbedMDService, error) {
	if cfg.DB != nil {
		db.SetDB(cfg.DB)
	} else {
		err := db.Init(cfg.DatabaseType, cfg.DatabaseDSN, cfg.TLSConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize database connector: %w", err)
		}
	}

	dbConnector := db.GetConnector()
	if dbConnector == nil {
		return nil, fmt.Errorf("database connector not initialized")
	}

	return &EmbedMDService{
		dbConnector:       dbConnector,
		waitForMigrations: cfg.WaitForMigrations,
	}, nil
}

func (s *EmbedMDService) Connect(spec *datastore.Spec) (datastore.RepoSet, error) {
	glog.Infof("Connecting to EmbedMD service...")

	connectedDB, err := s.dbConnector.Connect()
	if err != nil {
		return nil, err
	}

	glog.Infof("Connected to EmbedMD service")

	if s.waitForMigrations {
		return s.migrationWait(connectedDB, spec)
	}

	err = s.RunMigrations(spec)
	if err != nil {
		return nil, err
	}

	return newRepoSet(connectedDB, spec)
}

func (s *EmbedMDService) RunMigrations(spec *datastore.Spec) error {
	connectedDB, err := s.dbConnector.Connect()
	if err != nil {
		return err
	}

	migrator, err := db.NewDBMigrator(connectedDB)
	if err != nil {
		return err
	}

	glog.Infof("Running migrations...")
	err = migrator.Migrate()
	if err != nil {
		return err
	}
	glog.Infof("Migrations completed")

	glog.Infof("Syncing types...")
	err = s.syncTypes(connectedDB, spec)
	if err != nil {
		return err
	}
	glog.Infof("Syncing types completed")

	return nil
}

func (s *EmbedMDService) migrationWait(connectedDB *gorm.DB, spec *datastore.Spec) (datastore.RepoSet, error) {
	// This may run forever. But in practice, k8s should kill the pod after
	// failing the ready check.
	for {
		rs, err := newRepoSet(connectedDB, spec)
		var mtErr ErrMissingType
		if errors.As(err, &mtErr) {
			glog.Warningf("Unable to get types: %v", err)
			time.Sleep(time.Second)
			continue
		}
		return rs, err
	}
}

func (s EmbedMDService) Type() string {
	return connectorType
}

const (
	executionTypeKind int32 = iota
	artifactTypeKind
	contextTypeKind
)

func (s *EmbedMDService) syncTypes(conn *gorm.DB, spec *datastore.Spec) error {
	idMap := make(map[string]int32, len(spec.ExecutionTypes)+len(spec.ArtifactTypes)+len(spec.ContextTypes))
	var errs []error

	typeRepository := service.NewTypeRepository(conn)
	errs = append(errs, s.createTypes(typeRepository, spec.ExecutionTypes, executionTypeKind, idMap))
	errs = append(errs, s.createTypes(typeRepository, spec.ArtifactTypes, artifactTypeKind, idMap))
	errs = append(errs, s.createTypes(typeRepository, spec.ContextTypes, contextTypeKind, idMap))

	typePropertyRepository := service.NewTypePropertyRepository(conn)
	errs = append(errs, s.createTypeProperties(typePropertyRepository, spec.ExecutionTypes, idMap))
	errs = append(errs, s.createTypeProperties(typePropertyRepository, spec.ArtifactTypes, idMap))
	errs = append(errs, s.createTypeProperties(typePropertyRepository, spec.ContextTypes, idMap))

	return errors.Join(errs...)
}

func (s *EmbedMDService) createTypes(repo models.TypeRepository, types map[string]*datastore.SpecType, kind int32, idMap map[string]int32) error {
	var errs []error
	for _, name := range slices.Sorted(maps.Keys(types)) {
		t, err := repo.Save(&models.TypeImpl{
			Attributes: &models.TypeAttributes{
				Name:     &name,
				TypeKind: &kind,
			},
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: unable to create type: %w", name, err))
			continue
		}

		id := t.GetID()
		if id == nil {
			errs = append(errs, fmt.Errorf("%s: unable to determine type ID", name))
			continue
		}
		idMap[name] = *id
	}

	return errors.Join(errs...)
}

func (s *EmbedMDService) Reconnect(spec *datastore.Spec) (datastore.RepoSet, error) {
	connectedDB, err := s.dbConnector.Connect()
	if err != nil {
		return nil, err
	}
	return newRepoSet(connectedDB, spec)
}

func (s *EmbedMDService) createTypeProperties(repo models.TypePropertyRepository, types map[string]*datastore.SpecType, idMap map[string]int32) error {
	var errs []error
	for _, typeName := range slices.Sorted(maps.Keys(types)) {
		typeSpec := types[typeName]
		typeID := idMap[typeName]
		if typeID == 0 {
			errs = append(errs, fmt.Errorf("%s: unknown type", typeName))
			continue
		}

		for _, name := range slices.Sorted(maps.Keys(typeSpec.Properties)) {
			dataType := typeSpec.Properties[name]
			_, err := repo.Save(&models.TypePropertyImpl{
				TypeID:   typeID,
				Name:     name,
				DataType: new(int32(dataType)),
			})
			if err != nil {
				errs = append(errs, fmt.Errorf("%s-%s: %w", typeName, name, err))
			}
		}
	}

	return errors.Join(errs...)
}
