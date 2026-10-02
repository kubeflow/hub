package cmd

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/kubeflow/hub/internal/platform/db/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	contpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestCatalogStartupLifecycle(t *testing.T) {
	container, err := contpostgres.Run(t.Context(), "postgres:16",
		contpostgres.WithDatabase("catalog_lifecycle"),
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
	for _, scenario := range []string{"signal_during_initialization", "initialization_timeout", "signal_during_election_setup", "election_setup_timeout", "signal_during_plugin_init", "route_failure_cleanup", "route_failure_canceled_cleanup", "http_failure_stops_election", "recovery_failure", "database_recovery"} {
		t.Run(scenario, func(t *testing.T) {
			require.NoError(t, conn.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public").Error)
			switch scenario {
			case "recovery_failure":
				child, _ := startCatalogStartupChild(t, dsn, "TEST_CATALOG_FAIL_RECONNECT=1", "TEST_CATALOG_EXPECT_ERROR=startup test reconnect failure")
				waitForCatalogExit(t, child)
			case "signal_during_initialization", "initialization_timeout", "signal_during_election_setup", "election_setup_timeout":
				tx, err := sqlDB.BeginTx(t.Context(), nil)
				require.NoError(t, err)
				defer func() { assert.NoError(t, tx.Rollback()) }()
				ddl := "CREATE TABLE schema_migrations (version bigint PRIMARY KEY, dirty boolean NOT NULL)"
				queryPattern := "CREATE TABLE IF NOT EXISTS%"
				if scenario == "signal_during_election_setup" || scenario == "election_setup_timeout" {
					ddl = "CREATE TABLE locks (name varchar(255) PRIMARY KEY, record_version_number bigint, data bytea, owner varchar(255))"
					queryPattern = "%CREATE TABLE IF NOT EXISTS locks%"
				}
				_, err = tx.ExecContext(t.Context(), ddl)
				require.NoError(t, err)
				var env []string
				if scenario == "initialization_timeout" || scenario == "election_setup_timeout" {
					env = []string{"CATALOG_INITIALIZATION_TIMEOUT=2s", "TEST_CATALOG_EXPECT_DEADLINE=1"}
				}
				child, _ := startCatalogStartupChild(t, dsn, env...)
				require.True(t, child.waitFor(t, func() bool {
					var count int
					err := conn.Raw("SELECT COUNT(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock' AND query LIKE ?", queryPattern).Scan(&count).Error
					return err == nil && count > 0
				}))
				if scenario == "signal_during_initialization" || scenario == "signal_during_election_setup" {
					require.NoError(t, child.process.Signal(syscall.SIGTERM))
				}
				waitForCatalogExit(t, child)
			case "signal_during_plugin_init":
				// A plugin Init returning a non-Canceled error after a shutdown
				// signal must still exit cleanly: startupError normalizes any
				// startup error once a signal has been observed, not only
				// errors that unwrap to context.Canceled.
				initStarted := filepath.Join(t.TempDir(), "init_started")
				child, _ := startCatalogStartupChild(t, dsn, "TEST_CATALOG_BLOCK_INIT=1", "TEST_CATALOG_INIT_STARTED="+initStarted)
				require.True(t, child.waitFor(t, func() bool {
					_, err := os.Stat(initStarted)
					return err == nil
				}))
				require.NoError(t, child.process.Signal(syscall.SIGTERM))
				waitForCatalogExit(t, child)
			case "route_failure_cleanup", "route_failure_canceled_cleanup", "http_failure_stops_election":
				stopped := filepath.Join(t.TempDir(), "stopped")
				env := []string{"TEST_CATALOG_STOPPED=" + stopped}
				if scenario == "route_failure_cleanup" || scenario == "route_failure_canceled_cleanup" {
					env = append(env, "TEST_CATALOG_FAIL_ROUTES=1", "TEST_CATALOG_EXPECT_ERROR=startup test route failure")
					if scenario == "route_failure_canceled_cleanup" {
						env = append(env, "TEST_CATALOG_STOP_CANCELED=1")
					}
				} else {
					listener, err := net.Listen("tcp", "127.0.0.1:0")
					require.NoError(t, err)
					defer func() { assert.NoError(t, listener.Close()) }()
					env = append(env, "TEST_CATALOG_LISTEN="+listener.Addr().String(), "TEST_CATALOG_EXPECT_ERROR=HTTP server failed")
				}
				child, _ := startCatalogStartupChild(t, dsn, env...)
				waitForCatalogExit(t, child)
				_, err := os.Stat(stopped)
				require.NoError(t, err, "successfully initialized plugins must stop on startup failure")
			case "database_recovery":
				child, baseURL := startCatalogStartupChild(t, dsn)
				client := &http.Client{Timeout: 500 * time.Millisecond}
				defer client.CloseIdleConnections()
				require.True(t, child.waitFor(t, func() bool {
					return catalogStartupHTTPStatus(client, baseURL+"/test/leader") == http.StatusOK
				}))
				require.NoError(t, conn.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public").Error)
				require.True(t, child.waitFor(t, func() bool {
					return catalogStartupHTTPStatus(client, baseURL+"/test/leader") == http.StatusServiceUnavailable
				}), "leadership must stop after database loss")
				require.True(t, child.waitFor(t, func() bool {
					return catalogStartupHTTPStatus(client, baseURL+"/test/leader") == http.StatusOK &&
						catalogStartupHTTPStatus(client, baseURL+"/readyz") == http.StatusOK
				}), "recovery must reinitialize the datastore and resume ingestion")
				var count int
				require.NoError(t, conn.Raw(`SELECT COUNT(*) FROM "Type" WHERE name = 'kf.CatalogMetricsArtifact'`).Scan(&count).Error)
				require.Equal(t, 1, count)
			}
		})
	}
}

func waitForCatalogExit(t *testing.T, child *catalogStartupChild) {
	t.Helper()
	select {
	case <-child.done:
		require.NoError(t, child.err)
	case <-time.After(5 * time.Second):
		t.Fatal("catalog did not exit promptly")
	}
}
