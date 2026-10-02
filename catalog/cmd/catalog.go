package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/kubeflow/hub/catalog/internal/db/service"
	"github.com/kubeflow/hub/catalog/internal/leader"
	"github.com/kubeflow/hub/catalog/internal/plugin"
	"github.com/kubeflow/hub/internal/datastore/embedmd"
	"github.com/kubeflow/hub/internal/platform/db/postgres"
	"github.com/kubeflow/hub/internal/platform/server/middleware"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	_ "github.com/kubeflow/hub/catalog/internal/plugins/agent"
	_ "github.com/kubeflow/hub/catalog/internal/plugins/mcp"
	_ "github.com/kubeflow/hub/catalog/internal/plugins/model"
	_ "github.com/kubeflow/hub/catalog/internal/plugins/skill"
)

var catalogCfg = struct {
	ListenAddress          string
	ConfigPath             []string
	PerformanceMetricsPath []string
	CORSAllowedOrigins     []string
	AlphaSunsetDate        string
}{
	ListenAddress:          "0.0.0.0:8080",
	ConfigPath:             []string{"sources.yaml"},
	PerformanceMetricsPath: []string{},
}

const (
	leaderLockName = "catalog-leader"

	defaultLeaderLockDuration = 60 * time.Second
	defaultLeaderHeartbeat    = 15 * time.Second

	envLeaderLockDuration        = "CATALOG_LEADER_LOCK_DURATION"
	envLeaderHeartbeat           = "CATALOG_LEADER_HEARTBEAT"
	envInitializationTimeout     = "CATALOG_INITIALIZATION_TIMEOUT"
	defaultInitializationTimeout = 5 * time.Minute
)

func getInitializationTimeout() (time.Duration, error) {
	value := os.Getenv(envInitializationTimeout)
	if value == "" {
		return defaultInitializationTimeout, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("invalid %s %q: must be a positive duration", envInitializationTimeout, value)
	}
	return duration, nil
}

// parseDurationEnv parses a duration from an environment variable,
// falling back to a default value if unset or invalid.
func parseDurationEnv(envName string, defaultVal time.Duration) time.Duration {
	if envVal := os.Getenv(envName); envVal != "" {
		if parsed, err := time.ParseDuration(envVal); err == nil {
			glog.Infof("Using %s: %v", envName, parsed)
			return parsed
		}
		glog.Warningf("Invalid %s value %q, using default %v", envName, envVal, defaultVal)
	}
	return defaultVal
}

// getLeaderElectionConfig reads leader election configuration from environment
// variables, falling back to defaults when unset or invalid.
func getLeaderElectionConfig() (lockDuration, heartbeat time.Duration) {
	lockDuration = parseDurationEnv(envLeaderLockDuration, defaultLeaderLockDuration)
	heartbeat = parseDurationEnv(envLeaderHeartbeat, defaultLeaderHeartbeat)

	// Validate pglock requirement: heartbeat <= lockDuration/2
	if heartbeat > lockDuration/2 {
		glog.Warningf("Heartbeat (%v) exceeds half of lock duration (%v), required by pglock. Using defaults.", heartbeat, lockDuration)
		return defaultLeaderLockDuration, defaultLeaderHeartbeat
	}

	return lockDuration, heartbeat
}

var CatalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Catalog API server",
	Long: `Launch the API server for the model catalog. Use PostgreSQL's
	environment variables
	(https://www.postgresql.org/docs/current/libpq-envars.html) to
	configure the database connection.`,
	RunE: runCatalogServer,
}

func init() {
	fs := CatalogCmd.Flags()
	fs.StringVarP(&catalogCfg.ListenAddress, "listen", "l", catalogCfg.ListenAddress, "Address to listen on")
	fs.StringSliceVar(&catalogCfg.ConfigPath, "catalogs-path", catalogCfg.ConfigPath, "Path to catalog source configuration file")
	fs.StringSliceVar(&catalogCfg.PerformanceMetricsPath, "performance-metrics", catalogCfg.PerformanceMetricsPath, "Path to performance metrics data directory")
	fs.StringSliceVar(&catalogCfg.CORSAllowedOrigins, "cors-allowed-origins", nil,
		"Comma-separated list of allowed CORS origins. If empty (default), CORS is disabled. Can also be set via CATALOG_CORS_ALLOWED_ORIGINS environment variable.")
	fs.StringVar(&catalogCfg.AlphaSunsetDate, "alpha-sunset-date", "",
		"Sunset date (YYYY-MM-DD) for the deprecated v1alpha1 catalog APIs, per RFC 8594. "+
			"If empty (default), no deprecation headers are added to v1alpha1 responses. Can also be set via CATALOG_ALPHA_SUNSET_DATE environment variable.")
}

func runCatalogServer(cmd *cobra.Command, _ []string) (result error) {
	signalCtx, stopSignals := signal.NotifyContext(cmd.Context(), syscall.SIGTERM, syscall.SIGINT)
	defer stopSignals()
	ctx, cancel := context.WithCancelCause(signalCtx)
	defer cancel(nil)
	initializationTimeout, err := getInitializationTimeout()
	if err != nil {
		return err
	}
	// Normalize externally canceled startup before cleanup joins its errors.
	startupError := func(message string, err error) error {
		if signalCtx.Err() != nil {
			return nil
		}
		return fmt.Errorf("%s: %w", message, err)
	}
	if !cmd.Flags().Changed("cors-allowed-origins") {
		if envVal := os.Getenv("CATALOG_CORS_ALLOWED_ORIGINS"); envVal != "" {
			for origin := range strings.SplitSeq(envVal, ",") {
				if o := strings.TrimSpace(origin); o != "" {
					catalogCfg.CORSAllowedOrigins = append(catalogCfg.CORSAllowedOrigins, o)
				}
			}
		}
	}

	if !cmd.Flags().Changed("alpha-sunset-date") {
		if envVal := os.Getenv("CATALOG_ALPHA_SUNSET_DATE"); envVal != "" {
			catalogCfg.AlphaSunsetDate = envVal
		}
	}
	var alphaSunsetDate *time.Time
	if catalogCfg.AlphaSunsetDate != "" {
		parsed, err := time.Parse("2006-01-02", catalogCfg.AlphaSunsetDate)
		if err != nil {
			return fmt.Errorf("invalid --alpha-sunset-date %q: must be YYYY-MM-DD: %w", catalogCfg.AlphaSunsetDate, err)
		}
		alphaSunsetDate = &parsed
		glog.Infof("Alpha API (v1alpha1) deprecation headers enabled; sunset date: %s", catalogCfg.AlphaSunsetDate)
	}

	spec, err := service.DatastoreSpec()
	if err != nil {
		return fmt.Errorf("error building datastore spec: %w", err)
	}
	// Each pod initializes the database independently of ingestion leadership.
	initCtx, initCancel := context.WithTimeout(ctx, initializationTimeout)
	defer initCancel()
	gormDB, err := postgres.NewPostgresDBConnector("", nil).ConnectContext(initCtx)
	if err != nil {
		return startupError("error connecting to database", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return fmt.Errorf("error getting database pool: %w", err)
	}
	defer func() { result = errors.Join(result, sqlDB.Close()) }()
	ds, err := embedmd.NewEmbedMDService(&embedmd.EmbedMDConfig{DB: gormDB})
	if err != nil {
		return fmt.Errorf("error creating datastore: %w", err)
	}

	repoSet, err := ds.Initialize(initCtx, spec)
	if err != nil {
		return startupError("error initializing datastore", err)
	}
	initCancel()

	// Plugin server setup
	pluginServer := plugin.NewServer(plugin.ServerConfig{
		DB:                     gormDB,
		ConfigPaths:            catalogCfg.ConfigPath,
		PerformanceMetricsPath: catalogCfg.PerformanceMetricsPath,
		RepoSet:                repoSet,
		CORSAllowedOrigins:     catalogCfg.CORSAllowedOrigins,
		AlphaSunsetDate:        alphaSunsetDate,
	})

	// Stop every successfully initialized plugin, including on startup errors.
	defer func() {
		cancel(nil)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := pluginServer.Stop(shutdownCtx); err != nil {
			result = errors.Join(result, fmt.Errorf("plugin shutdown error: %w", err))
		}
	}()
	if err := pluginServer.Init(ctx); err != nil {
		return startupError("error initializing plugins", err)
	}

	handler, err := pluginServer.MountRoutes()
	if err != nil {
		return fmt.Errorf("error mounting routes: %w", err)
	}

	if err := pluginServer.Start(ctx); err != nil {
		return startupError("error starting plugins", err)
	}

	// Election starts only after plugins are fully constructed and started.
	// Its context also observes HTTP failures, so g.Wait cannot strand election.
	g, gctx := errgroup.WithContext(ctx)
	lockDuration, heartbeat := getLeaderElectionConfig()
	glog.Infof("Leader election configured: lock duration=%v, heartbeat=%v", lockDuration, heartbeat)
	elector, err := leader.NewLeaderElector(gormDB, gctx, leaderLockName, lockDuration, heartbeat, initializationTimeout)
	if err != nil {
		return startupError("error creating leader elector", err)
	}
	defer func() {
		cancel(nil)
		_ = elector.Wait() // Reported by the errgroup below.
	}()
	pluginServer.AddReadinessCheck("leader_election", elector.Healthy)
	elector.OnBecomeLeader(func(leaderCtx context.Context) {
		recoveryCtx, recoveryCancel := context.WithTimeout(leaderCtx, initializationTimeout)
		defer recoveryCancel()
		newRepoSet, err := ds.Initialize(recoveryCtx, spec)
		if err == nil {
			err = pluginServer.Reconnect(recoveryCtx, newRepoSet)
		}
		if err != nil {
			if leaderCtx.Err() == nil {
				glog.Errorf("unable to initialize leader datastore: %v — canceling to trigger restart", err)
				cancel(fmt.Errorf("leader datastore initialization failed: %w", err))
			}
			return
		}
		recoveryCancel()
		pluginServer.NotifyLeader(leaderCtx)
	})

	server := &http.Server{
		Addr:    catalogCfg.ListenAddress,
		Handler: middleware.ValidationMiddleware(handler),
	}

	g.Go(func() error {
		glog.Infof("Catalog API server listening on %s", catalogCfg.ListenAddress)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("HTTP server failed: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		<-gctx.Done()
		glog.Info("Shutting down HTTP server...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			glog.Errorf("HTTP server shutdown error: %v", err)
		}
		return nil
	})

	g.Go(func() error {
		if err := elector.Wait(); err != nil {
			return fmt.Errorf("leader elector failed: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		result = err
	}
	if cause := context.Cause(ctx); cause != nil && cause != context.Canceled {
		result = errors.Join(result, cause)
	}
	return result
}
