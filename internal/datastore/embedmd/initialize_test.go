package embedmd

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/kubeflow/hub/internal/db/models"
	"github.com/kubeflow/hub/internal/db/service"
	"github.com/kubeflow/hub/internal/platform/datastore"
	platformdb "github.com/kubeflow/hub/internal/platform/db"
	"github.com/kubeflow/hub/internal/platform/db/postgres"
	"github.com/kubeflow/hub/internal/platform/db/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	contpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestInitializePostgres(t *testing.T) {
	container, err := contpostgres.Run(t.Context(), "postgres:16",
		contpostgres.WithDatabase("initialization"),
		contpostgres.WithUsername("postgres"),
		contpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp")),
	)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, container.Terminate(context.Background())) })
	dsn, err := container.ConnectionString(t.Context(), "sslmode=disable")
	require.NoError(t, err)
	conn, err := postgres.NewPostgresDBConnector(dsn, nil).WithMaxRetries(1).Connect()
	require.NoError(t, err)
	sqlDB, err := conn.DB()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, sqlDB.Close()) })
	svc := &EmbedMDService{dbConnector: platformdb.ConnectedConnector{ConnectedDB: conn}}
	spec := datastore.NewSpec().AddArtifact("test.NewArtifact",
		datastore.NewSpecType(service.NewModelArtifactRepository).AddString("uri"))
	reset := func(t *testing.T) {
		t.Helper()
		require.NoError(t, conn.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public").Error)
	}

	t.Run("concurrent_initialization", func(t *testing.T) {
		reset(t)
		var wg sync.WaitGroup
		errs := make(chan error, 4)
		start := make(chan struct{})
		for range 4 {
			wg.Go(func() {
				<-start
				_, err := svc.Initialize(t.Context(), spec)
				errs <- err
			})
		}
		close(start)
		wg.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err)
		}
		var count int
		require.NoError(t, conn.Raw(`SELECT COUNT(*) FROM "Type" WHERE name = 'test.NewArtifact'`).Scan(&count).Error)
		require.Equal(t, 1, count)
		require.NoError(t, conn.Raw(`SELECT COUNT(*) FROM "TypeProperty" p JOIN "Type" t ON p.type_id = t.id WHERE t.name = 'test.NewArtifact' AND p.name = 'uri'`).Scan(&count).Error)
		require.Equal(t, 1, count)
	})

	t.Run("mixed_version_initialization", func(t *testing.T) {
		reset(t)
		// Migrations must exist before type registration can run; a legacy
		// pod would have run them too, just without ever taking the
		// initialization advisory lock around type registration afterward.
		require.NoError(t, postgres.MigrateContext(t.Context(), conn))

		repo := repository.NewTypeRepository(conn)
		kind := int32(1)

		// Race TypeRepository.Save (the current, fixed code) against a
		// plain, unconditional insert that emulates a pod still running the
		// previous image -- one that never acquires the initialization
		// advisory lock and has no conflict handling, exactly like Save
		// before this fix. Repeated across many distinct names to make
		// actually interleaving the two lookup-then-insert windows likely;
		// on a pre-fix Save this reliably produces duplicate Type rows
		// within a handful of iterations.
		for i := range 50 {
			typeName := fmt.Sprintf("test.RaceType%d", i)

			var wg sync.WaitGroup
			var saveErr, legacyErr error
			start := make(chan struct{})

			wg.Go(func() {
				<-start
				_, saveErr = repo.Save(&models.TypeImpl{
					Attributes: &models.TypeAttributes{Name: &typeName, TypeKind: &kind},
				})
			})
			wg.Go(func() {
				<-start
				legacyErr = conn.Exec(`INSERT INTO "Type" (name, type_kind) VALUES (?, 1)`, typeName).Error
			})
			close(start)
			wg.Wait()

			require.NoError(t, saveErr, "Save must never fail, whether it wins or loses the race")
			if legacyErr != nil {
				assert.Contains(t, legacyErr.Error(), "duplicate key value violates unique constraint",
					"a losing legacy insert must fail cleanly on the UNIQUE(name) constraint, not silently create a second row")
			}

			var count int
			require.NoError(t, conn.Raw(`SELECT COUNT(*) FROM "Type" WHERE name = ?`, typeName).Scan(&count).Error)
			require.Equal(t, 1, count, "exactly one Type row must exist for %s regardless of which writer won", typeName)
		}
	})

	t.Run("lock_deadline", func(t *testing.T) {
		reset(t)
		locked := make(chan struct{})
		release := make(chan struct{})
		done := make(chan error, 1)
		go func() {
			done <- postgres.WithInitializationLock(t.Context(), conn, func() error {
				close(locked)
				<-release
				return nil
			})
		}()
		defer func() {
			close(release)
			assert.NoError(t, <-done)
		}()
		<-locked
		ctx, cancel := context.WithTimeout(t.Context(), 150*time.Millisecond)
		defer cancel()
		_, err := svc.Initialize(ctx, spec)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		// No migrations or registration may run without the initialization lock.
		var count int
		require.NoError(t, conn.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'").Scan(&count).Error)
		require.Zero(t, count)
	})

	t.Run("cancel_blocked_migration", func(t *testing.T) {
		reset(t)
		// The migration driver's version-table setup will block behind this DDL.
		tx, err := sqlDB.BeginTx(t.Context(), nil)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback() }()
		_, err = tx.ExecContext(t.Context(), "CREATE TABLE schema_migrations (version bigint PRIMARY KEY, dirty boolean NOT NULL)")
		require.NoError(t, err)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		done := make(chan error, 1)
		go func() {
			_, err := svc.Initialize(ctx, spec)
			done <- err
		}()
		require.Eventually(t, func() bool {
			var count int
			err := conn.Raw("SELECT COUNT(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock' AND query LIKE 'CREATE TABLE IF NOT EXISTS%'").Scan(&count).Error
			return err == nil && count > 0
		}, 5*time.Second, 10*time.Millisecond)
		cancel()
		select {
		case err := <-done:
			require.ErrorIs(t, err, context.Canceled)
		case <-time.After(3 * time.Second):
			t.Fatal("initialization ignored cancellation during SQL migrations")
		}
		require.NoError(t, tx.Rollback())
		// A new initializer must acquire the lock and the shared pool must survive.
		_, err = svc.Initialize(t.Context(), spec)
		require.NoError(t, err)
	})

	t.Run("cancel_blocked_type_registration", func(t *testing.T) {
		reset(t)
		require.NoError(t, postgres.MigrateContext(t.Context(), conn))
		tx, err := sqlDB.BeginTx(t.Context(), nil)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback() }()
		_, err = tx.ExecContext(t.Context(), `LOCK TABLE "Type" IN ACCESS EXCLUSIVE MODE`)
		require.NoError(t, err)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		done := make(chan error, 1)
		go func() {
			_, err := svc.Initialize(ctx, spec)
			done <- err
		}()
		require.Eventually(t, func() bool {
			var count int
			err := conn.Raw(`SELECT COUNT(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock' AND query LIKE 'SELECT%"Type"%'`).Scan(&count).Error
			return err == nil && count > 0
		}, 5*time.Second, 10*time.Millisecond)
		cancel()
		select {
		case err := <-done:
			require.ErrorIs(t, err, context.Canceled)
		case <-time.After(3 * time.Second):
			t.Fatal("type registration ignored cancellation")
		}
		require.NoError(t, tx.Rollback())
		var count int
		require.NoError(t, conn.Raw(`SELECT COUNT(*) FROM "Type" WHERE name = 'test.NewArtifact'`).Scan(&count).Error)
		require.Zero(t, count, "canceled registration must not create the type later")
		_, err = svc.Initialize(t.Context(), spec)
		require.NoError(t, err)
	})

	t.Run("recovery_refreshes_type_ids", func(t *testing.T) {
		reset(t)
		repo, err := svc.Initialize(t.Context(), spec)
		require.NoError(t, err)
		oldID := repo.TypeMap()["test.NewArtifact"]
		require.NoError(t, conn.Exec(`DELETE FROM "TypeProperty" WHERE type_id = ?`, oldID).Error)
		require.NoError(t, conn.Exec(`DELETE FROM "Type" WHERE id = ?`, oldID).Error)
		recovered, err := svc.Initialize(t.Context(), spec)
		require.NoError(t, err)
		require.NotEqual(t, oldID, recovered.TypeMap()["test.NewArtifact"])
		// Canceling an initialization context must not poison the repositories.
		ctx, cancel := context.WithCancel(t.Context())
		repo, err = svc.Initialize(ctx, spec)
		require.NoError(t, err)
		cancel()
		artifactRepo, err := repo.Repository(reflect.TypeFor[models.ModelArtifactRepository]())
		require.NoError(t, err)
		_, err = artifactRepo.(models.ModelArtifactRepository).GetByID(0)
		require.ErrorIs(t, err, service.ErrModelArtifactNotFound)
	})
}
