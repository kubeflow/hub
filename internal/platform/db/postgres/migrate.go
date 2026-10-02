package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"embed"
	"errors"
	"fmt"
	"net"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/stdlib"
	platformdb "github.com/kubeflow/hub/internal/platform/db"
	"github.com/kubeflow/hub/internal/platform/db/types"
	"gorm.io/gorm"
)

// MigrateContext applies migrations on a reserved connection without closing
// the caller's database pool. golang-migrate uses background contexts for
// several queries, including advisory-lock acquisition and version-table setup.
// Closing this connection's transport on cancellation interrupts those queries.
func MigrateContext(ctx context.Context, db *gorm.DB) (result error) {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := conn.Close(); !errors.Is(err, sql.ErrConnDone) {
			result = errors.Join(result, err)
		}
	}()
	var transport net.Conn
	if err := conn.Raw(func(raw any) error {
		pgxConn, ok := raw.(*stdlib.Conn)
		if !ok {
			return fmt.Errorf("context-aware migrations require a pgx connection, got %T", raw)
		}
		transport = pgxConn.Conn().PgConn().Conn()
		return nil
	}); err != nil {
		return err
	}
	interrupted := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		// net.Conn.Close is safe concurrently with the migration's SQL queries.
		_ = transport.Close()
		close(interrupted)
	})
	defer func() {
		if result != nil {
			// Up can time out while its lock-acquisition goroutine still runs.
			// Interrupt it before sql.Conn.Close waits for the running query.
			_ = transport.Close()
		}
		if !stop() {
			<-interrupted
		}
		if ctx.Err() != nil || result != nil {
			err := conn.Raw(func(any) error { return driver.ErrBadConn })
			if err != nil && !errors.Is(err, driver.ErrBadConn) && !errors.Is(err, sql.ErrConnDone) {
				result = errors.Join(result, err)
			}
			result = errors.Join(ctx.Err(), result)
		}
	}()
	dbDriver, err := postgres.WithConnection(ctx, conn, &postgres.Config{})
	if err != nil {
		return err
	}
	source, err := iofs.New(migrations, MigrationDir)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, source.Close()) }()
	m, err := migrate.NewWithInstance("iofs", source, "postgres", dbDriver)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

//go:embed migrations/*.sql
var migrations embed.FS

const (
	MigrationDir = "migrations"
)

func init() {
	platformdb.RegisterMigratorFactory(types.DatabaseTypePostgres, func(db *gorm.DB) (platformdb.DBMigrator, error) {
		return NewPostgresMigrator(db)
	})
}

type PostgresMigrator struct {
	migrator *migrate.Migrate
}

func NewPostgresMigrator(db *gorm.DB) (*PostgresMigrator, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		return nil, err
	}

	// Create a new source instance from the embedded files
	source, err := iofs.New(migrations, MigrationDir)
	if err != nil {
		return nil, err
	}

	m, err := migrate.NewWithInstance(
		"iofs",
		source,
		"postgres",
		driver,
	)
	if err != nil {
		return nil, err
	}

	return &PostgresMigrator{
		migrator: m,
	}, nil
}

func (m *PostgresMigrator) Migrate() error {
	if err := m.Up(nil); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

func (m *PostgresMigrator) Up(steps *int) error {
	if steps == nil {
		return m.migrator.Up()
	}

	if *steps < 0 {
		return fmt.Errorf("steps cannot be negative")
	}

	return m.migrator.Steps(*steps)
}

func (m *PostgresMigrator) Down(steps *int) error {
	if steps == nil {
		return m.migrator.Down()
	}

	if *steps > 0 {
		return fmt.Errorf("steps cannot be positive")
	}

	return m.migrator.Steps(*steps)
}
