package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/kubeflow/hub/internal/platform/db/postgres"
	"github.com/stretchr/testify/require"
)

// An old migrator can acquire the schema lock after version-table setup but
// before Up. golang-migrate times out without joining its lock-query goroutine;
// cleanup must interrupt that query rather than wait forever in sql.Conn.Close.
func TestMigrateContextLockTimeoutCleanup(t *testing.T) {
	require.NoError(t, postgres.MigrateContext(t.Context(), sharedDB))
	sqlDB, err := sharedDB.DB()
	require.NoError(t, err)
	lockID, err := database.GenerateAdvisoryLockId("testdb", "public", "schema_migrations")
	require.NoError(t, err)
	first, err := sqlDB.Conn(t.Context())
	require.NoError(t, err)
	defer func() { _ = first.Close() }()
	_, err = first.ExecContext(t.Context(), "SELECT pg_advisory_lock($1)", lockID)
	require.NoError(t, err)
	defer func() { _, _ = first.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", lockID) }()
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- postgres.MigrateContext(ctx, sharedDB) }()
	waiting := func(pid int) bool {
		var count int
		query := "SELECT COUNT(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event = 'advisory'"
		var err error
		if pid == 0 {
			err = sqlDB.QueryRowContext(t.Context(), query).Scan(&count)
		} else {
			err = sqlDB.QueryRowContext(t.Context(), query+" AND pid = $1", pid).Scan(&count)
		}
		return err == nil && count > 0
	}
	require.Eventually(t, func() bool { return waiting(0) }, 5*time.Second, 10*time.Millisecond)
	// Queue another session behind version-table setup. PostgreSQL grants it
	// the lock before the migration session's subsequent Up acquisition.
	next, err := sqlDB.Conn(t.Context())
	require.NoError(t, err)
	defer func() { _ = next.Close() }()
	var pid int
	require.NoError(t, next.QueryRowContext(t.Context(), "SELECT pg_backend_pid()").Scan(&pid))
	acquired := make(chan error, 1)
	go func() {
		_, err := next.ExecContext(ctx, "SELECT pg_advisory_lock($1)", lockID)
		acquired <- err
	}()
	defer func() { _, _ = next.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", lockID) }()
	require.Eventually(t, func() bool { return waiting(pid) }, 5*time.Second, 10*time.Millisecond)
	_, err = first.ExecContext(t.Context(), "SELECT pg_advisory_unlock($1)", lockID)
	require.NoError(t, err)
	require.NoError(t, <-acquired)
	select {
	case err := <-done:
		require.ErrorIs(t, err, migrate.ErrLockTimeout)
	case <-time.After(20 * time.Second):
		t.Fatal("migration cleanup hung after the library's lock timeout")
	}
	require.NoError(t, sqlDB.PingContext(t.Context()), "migration cleanup must preserve the shared database pool")
	// No unfinished lock query may survive cleanup.
	require.Eventually(t, func() bool { return !waiting(0) }, 3*time.Second, 10*time.Millisecond)
	require.NoError(t, ctx.Err())
}
