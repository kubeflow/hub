package embedmd

import (
	"context"

	"github.com/golang/glog"
	"github.com/kubeflow/hub/internal/platform/datastore"
	platformdb "github.com/kubeflow/hub/internal/platform/db"
	"github.com/kubeflow/hub/internal/platform/db/postgres"
	"gorm.io/gorm"
)

// Initialize runs migrations and registers required types before constructing
// repositories. PostgreSQL initialization is serialized across processes and
// observes ctx; repositories retain the original database handle, not ctx.
func (s *EmbedMDService) Initialize(ctx context.Context, spec *datastore.Spec) (datastore.RepoSet, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var connectedDB *gorm.DB
	var err error
	if connector, ok := s.dbConnector.(interface {
		ConnectContext(context.Context) (*gorm.DB, error)
	}); ok {
		connectedDB, err = connector.ConnectContext(ctx)
	} else {
		connectedDB, err = s.dbConnector.Connect()
	}
	if err != nil {
		return nil, err
	}
	if err := s.runMigrations(ctx, connectedDB, spec); err != nil {
		return nil, err
	}
	return newRepoSetContext(ctx, connectedDB, spec)
}

func (s *EmbedMDService) runMigrations(ctx context.Context, conn *gorm.DB, spec *datastore.Spec) error {
	initialize := func() error {
		glog.Info("Running migrations...")
		if conn.Name() == "postgres" {
			if err := postgres.MigrateContext(ctx, conn); err != nil {
				return err
			}
		} else {
			migrator, err := platformdb.NewDBMigrator(conn)
			if err != nil {
				return err
			}
			if err := migrator.Migrate(); err != nil {
				return err
			}
		}
		glog.Info("Migrations completed; syncing types...")
		if err := s.syncTypes(conn.WithContext(ctx), spec); err != nil {
			return err
		}
		glog.Info("Syncing types completed")
		return nil
	}
	if conn.Name() == "postgres" {
		return postgres.WithInitializationLock(ctx, conn, initialize)
	}
	return initialize()
}
