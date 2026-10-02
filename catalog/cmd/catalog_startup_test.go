package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kubeflow/hub/catalog/internal/db/service"
	"github.com/kubeflow/hub/catalog/internal/leader"
	"github.com/kubeflow/hub/catalog/internal/plugin"
	"github.com/kubeflow/hub/internal/datastore/embedmd"
	platformdb "github.com/kubeflow/hub/internal/platform/db"
	"github.com/kubeflow/hub/internal/platform/db/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	contpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const catalogStartupHelperEnv = "TEST_CATALOG_STARTUP_HELPER"

// TestCatalogStartupRollingUpgrade exercises the real startup command rather
// than elector health: a replacement must serve HTTP before the old leader
// exits, even when the replacement requires a type the old version lacks.
func TestCatalogStartupRollingUpgrade(t *testing.T) {
	for _, scenario := range []struct {
		name        string
		missingType bool
	}{
		{name: "existing_types"},
		{name: "new_required_type", missingType: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			setupCtx, setupCancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer setupCancel()
			container, err := contpostgres.Run(setupCtx, "postgres:16",
				contpostgres.WithDatabase("catalog_startup"),
				contpostgres.WithUsername("postgres"),
				contpostgres.WithPassword("postgres"),
				testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp")),
			)
			require.NoError(t, err)
			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				assert.NoError(t, container.Terminate(ctx))
			})
			dsn, err := container.ConnectionString(setupCtx, "sslmode=disable")
			require.NoError(t, err)
			gormDB, err := postgres.NewPostgresDBConnector(dsn, nil).WithMaxRetries(1).Connect()
			require.NoError(t, err)
			sqlDB, err := gormDB.DB()
			require.NoError(t, err)
			t.Cleanup(func() { assert.NoError(t, sqlDB.Close()) })

			oldSpec, err := service.DatastoreSpec()
			require.NoError(t, err)
			require.Contains(t, oldSpec.ArtifactTypes, service.CatalogMetricsArtifactTypeName)
			if scenario.missingType {
				delete(oldSpec.ArtifactTypes, service.CatalogMetricsArtifactTypeName)
			}
			ds, err := embedmd.NewEmbedMDService(&embedmd.EmbedMDConfig{DB: gormDB})
			require.NoError(t, err)
			t.Cleanup(platformdb.ClearConnector)
			_, err = ds.Connect(oldSpec)
			require.NoError(t, err)

			typeCount := func() int {
				var count int
				require.NoError(t, sqlDB.QueryRow(`SELECT COUNT(*) FROM "Type" WHERE name = $1`,
					service.CatalogMetricsArtifactTypeName).Scan(&count))
				return count
			}
			if scenario.missingType {
				require.Zero(t, typeCount(), "old specification must not register the replacement's required type")
			} else {
				require.Equal(t, 1, typeCount())
			}

			oldCtx, oldCancel := context.WithCancel(context.Background())
			oldPod, err := leader.NewLeaderElector(gormDB, oldCtx, leaderLockName, 5*time.Second, time.Second)
			if err != nil {
				oldCancel()
				t.Fatal(err)
			}
			oldDone := make(chan error, 1)
			go func() { oldDone <- oldPod.Wait() }()
			oldStopped := false
			t.Cleanup(func() {
				oldCancel()
				if !oldStopped {
					select {
					case err := <-oldDone:
						assert.ErrorIs(t, err, context.Canceled)
					case <-time.After(5 * time.Second):
						t.Error("old leader did not stop")
					}
				}
			})
			var oldLeader atomic.Bool
			oldPod.OnBecomeLeader(func(ctx context.Context) {
				oldLeader.Store(true)
				defer oldLeader.Store(false)
				<-ctx.Done()
			})
			require.Eventually(t, oldLeader.Load, 5*time.Second, 100*time.Millisecond)
			leaseOwner := func() string {
				var owner string
				require.NoError(t, sqlDB.QueryRow(`SELECT owner FROM locks WHERE name = $1`, leaderLockName).Scan(&owner))
				return owner
			}
			oldOwner := leaseOwner()
			require.NotEmpty(t, oldOwner)

			child, baseURL := startCatalogStartupChild(t, dsn)
			client := &http.Client{Timeout: 500 * time.Millisecond}
			defer client.CloseIdleConnections()
			ready := child.waitFor(t, func() bool {
				return catalogStartupHTTPStatus(client, baseURL+"/readyz") == http.StatusOK
			})
			// Check the lease even on failure, so the timeout is attributable to
			// waiting for a type while a live old pod still owns ingestion.
			assert.NoError(t, oldCtx.Err())
			assert.True(t, oldLeader.Load(), "old pod must remain leader until the replacement is ready")
			assert.Equal(t, oldOwner, leaseOwner())
			t.Logf("replacement required type count: %d", typeCount())
			require.True(t, ready, "replacement must become HTTP-ready while the old leader is alive")
			require.Equal(t, 1, typeCount(), "replacement must register its required type")
			require.Equal(t, http.StatusServiceUnavailable, catalogStartupHTTPStatus(client, baseURL+"/test/leader"),
				"replacement must not start ingestion while the old pod owns leadership")

			oldCancel()
			select {
			case err := <-oldDone:
				oldStopped = true
				require.ErrorIs(t, err, context.Canceled)
			case <-time.After(5 * time.Second):
				t.Fatal("old leader did not release leadership")
			}
			require.True(t, child.waitFor(t, func() bool {
				return catalogStartupHTTPStatus(client, baseURL+"/test/leader") == http.StatusOK &&
					leaseOwner() != oldOwner &&
					catalogStartupHTTPStatus(client, baseURL+"/readyz") == http.StatusOK
			}), "replacement must take over ingestion and remain HTTP-ready after the old leader exits")
		})
	}
}

// Run startup in a separate process to isolate signals and plugin registration.
func TestCatalogStartupHelper(t *testing.T) {
	if os.Getenv(catalogStartupHelperEnv) != "1" {
		return
	}
	plugin.Register(&startupLeadershipObserver{})
	catalogCfg.ListenAddress = os.Getenv("TEST_CATALOG_LISTEN")
	catalogCfg.ConfigPath = []string{os.Getenv("TEST_CATALOG_SOURCES")}
	CatalogCmd.SetContext(t.Context())
	err := runCatalogServer(CatalogCmd, nil)
	if os.Getenv("TEST_CATALOG_EXPECT_DEADLINE") == "1" {
		require.ErrorIs(t, err, context.DeadlineExceeded)
	} else if expected := os.Getenv("TEST_CATALOG_EXPECT_ERROR"); expected != "" {
		require.ErrorContains(t, err, expected)
	} else {
		require.NoError(t, err)
	}
}

type startupLeadershipObserver struct {
	leader atomic.Bool
}

func (*startupLeadershipObserver) Name() string        { return "startup-test" }
func (*startupLeadershipObserver) Version() string     { return "v1" }
func (*startupLeadershipObserver) Description() string { return "Startup test leadership observer" }
func (*startupLeadershipObserver) Init(ctx context.Context, _ plugin.Config) error {
	if os.Getenv("TEST_CATALOG_BLOCK_INIT") == "1" {
		if path := os.Getenv("TEST_CATALOG_INIT_STARTED"); path != "" {
			if err := os.WriteFile(path, nil, 0600); err != nil {
				return err
			}
		}
		<-ctx.Done()
		return errors.New("startup test init failure after cancel")
	}
	return nil
}
func (*startupLeadershipObserver) Start(context.Context) error { return nil }
func (*startupLeadershipObserver) Stop(context.Context) error {
	if path := os.Getenv("TEST_CATALOG_STOPPED"); path != "" {
		if err := os.WriteFile(path, nil, 0600); err != nil {
			return err
		}
	}
	if os.Getenv("TEST_CATALOG_STOP_CANCELED") == "1" {
		return context.Canceled
	}
	return nil
}
func (*startupLeadershipObserver) Healthy() bool                  { return true }
func (*startupLeadershipObserver) Migrations() []plugin.Migration { return nil }

func (p *startupLeadershipObserver) OnBecomeLeader(ctx context.Context) error {
	p.leader.Store(true)
	defer p.leader.Store(false)
	<-ctx.Done()
	return ctx.Err()
}

func (*startupLeadershipObserver) Reconnect(context.Context, plugin.Config) error {
	if os.Getenv("TEST_CATALOG_FAIL_RECONNECT") == "1" {
		return fmt.Errorf("startup test reconnect failure: %w", context.Canceled)
	}
	return nil
}

func (p *startupLeadershipObserver) RegisterRoutes(router chi.Router) error {
	if os.Getenv("TEST_CATALOG_FAIL_ROUTES") == "1" {
		return errors.New("startup test route failure")
	}
	router.Get("/test/leader", func(w http.ResponseWriter, _ *http.Request) {
		if p.leader.Load() {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	})
	return nil
}

type catalogStartupChild struct {
	done    chan struct{}
	err     error // Read only after done closes.
	process *os.Process
}

func startCatalogStartupChild(t *testing.T, dsn string, extraEnv ...string) (*catalogStartupChild, string) {
	t.Helper()
	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	password, _ := parsed.User.Password()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())
	tmpDir := t.TempDir()
	sourcesPath := filepath.Join(tmpDir, "sources.yaml")
	require.NoError(t, os.WriteFile(sourcesPath, []byte("model_catalogs: []\nmcp_catalogs: []\nagent_catalogs: []\nskill_catalogs: []\n"), 0600))
	outputPath := filepath.Join(tmpDir, "catalog.log")
	output, err := os.Create(outputPath)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, output.Close()) })
	executable, err := os.Executable()
	require.NoError(t, err)
	command := exec.Command(executable, "-test.run=^TestCatalogStartupHelper$", "-test.timeout=60s", "-logtostderr=true")
	for _, env := range os.Environ() {
		// Isolate libpq configuration and catalog flags from the developer's
		// environment, while retaining PATH and the test runtime environment.
		if !strings.HasPrefix(env, "PG") && !strings.HasPrefix(env, "CATALOG_") && !strings.HasPrefix(env, "TEST_CATALOG_") {
			command.Env = append(command.Env, env)
		}
	}
	command.Env = append(command.Env,
		catalogStartupHelperEnv+"=1",
		"TEST_CATALOG_LISTEN="+address,
		"TEST_CATALOG_SOURCES="+sourcesPath,
		"PGHOST="+parsed.Hostname(), "PGPORT="+parsed.Port(),
		"PGUSER="+parsed.User.Username(), "PGPASSWORD="+password,
		"PGDATABASE="+strings.TrimPrefix(parsed.Path, "/"), "PGSSLMODE=disable",
		envLeaderLockDuration+"=5s", envLeaderHeartbeat+"=1s",
	)
	command.Env = append(command.Env, extraEnv...)
	command.Stdout = output
	command.Stderr = output
	require.NoError(t, command.Start())
	child := &catalogStartupChild{done: make(chan struct{}), process: command.Process}
	go func() {
		child.err = command.Wait()
		close(child.done)
	}()
	t.Cleanup(func() {
		select {
		case <-child.done:
		default:
			if err := command.Process.Signal(syscall.SIGTERM); err != nil && err != os.ErrProcessDone {
				t.Errorf("signal catalog subprocess: %v", err)
			}
			select {
			case <-child.done:
			case <-time.After(3 * time.Second):
				assert.NoError(t, command.Process.Kill())
				select {
				case <-child.done:
				case <-time.After(3 * time.Second):
					t.Error("catalog subprocess was not reaped")
				}
			}
		}
		if t.Failed() {
			logs, err := os.ReadFile(outputPath)
			if assert.NoError(t, err) {
				t.Logf("catalog subprocess output:\n%s", logs)
			}
		}
	})
	return child, "http://" + address
}

func (c *catalogStartupChild) waitFor(t *testing.T, condition func() bool) bool {
	t.Helper()
	timeout := time.NewTimer(15 * time.Second)
	defer timeout.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-c.done:
			t.Fatalf("catalog subprocess exited before the startup condition was met: %v", c.err)
		default:
		}
		if condition() {
			return true
		}
		select {
		case <-c.done:
			t.Fatalf("catalog subprocess exited before the startup condition was met: %v", c.err)
		case <-timeout.C:
			return false
		case <-ticker.C:
		}
	}
}

func catalogStartupHTTPStatus(client *http.Client, address string) int {
	response, err := client.Get(address)
	if err != nil {
		return 0
	}
	defer response.Body.Close() //nolint:errcheck // The status is sufficient for polling.
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		return 0
	}
	return response.StatusCode
}
