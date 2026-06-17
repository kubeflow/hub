package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/kubeflow/hub/ui/bff/internal/api"
	"github.com/kubeflow/hub/ui/bff/internal/config"

	"log/slog"
	"net/http"
	"os"
	"time"
)

func main() {
	var cfg config.EnvConfig
	var certFile, keyFile string

	flag.IntVar(&cfg.Port, "port", getEnvAsInt("PORT", 8080), "API server port")
	flag.StringVar(&certFile, "cert-file", "", "Path to TLS certificate file")
	flag.StringVar(&keyFile, "key-file", "", "Path to TLS key file")
	flag.BoolVar(&cfg.MockK8Client, "mock-k8s-client", false, "Use mock Kubernetes client")
	flag.BoolVar(&cfg.MockMRClient, "mock-mr-client", false, "Use mock Model Registry client")
	flag.BoolVar(&cfg.MockMRCatalogClient, "mock-mr-catalog-client", false, "Use mock Model Registry Catalog client")
	flag.BoolVar(&cfg.DevMode, "dev-mode", false, "Use development mode for access to local K8s cluster")
	flag.IntVar(&cfg.DevModeModelRegistryPort, "dev-mode-model-registry-port", getEnvAsInt("DEV_MODE_MODEL_REGISTRY_PORT", 8080), "Use port when in development mode for model registry")
	flag.IntVar(&cfg.DevModeCatalogPort, "dev-mode-catalog-port", getEnvAsInt("DEV_MODE_CATALOG_PORT", 8081), "Use port when in development mode for catalog")

	// New deployment mode flag
	flag.Var(&cfg.DeploymentMode, "deployment-mode", "Deployment mode (kubeflow, federated, or standalone)")

	flag.StringVar(&cfg.StaticAssetsDir, "static-assets-dir", "./static", "Configure frontend static assets root directory")
	flag.TextVar(&cfg.LogLevel, "log-level", parseLevel(getEnvAsString("LOG_LEVEL", "INFO")), "Sets server log level, possible values: error, warn, info, debug")
	flag.Func("allowed-origins", "Sets allowed origins for CORS purposes, accepts a comma separated list of origins or * to allow all, default none", newOriginParser(&cfg.AllowedOrigins, getEnvAsString("ALLOWED_ORIGINS", "")))
	// bundle-paths accepts a comma-separated list of CA bundle file paths to trust for outbound TLS.
	// If not provided via flag, it can be set via BUNDLE_PATHS env var (comma-separated). Defaults to empty.
	defaultBundlePaths := getEnvAsString("BUNDLE_PATHS", "")
	flag.Func("bundle-paths", "Comma-separated list of PEM CA bundle file paths to trust for outbound TLS (optional)", newOriginParser(&cfg.BundlePaths, defaultBundlePaths))
	flag.StringVar(&cfg.AuthMethod, "auth-method", "internal", "Authentication method (internal or user_token)")
	flag.StringVar(&cfg.AuthTokenHeader, "auth-token-header", getEnvAsString("AUTH_TOKEN_HEADER", config.DefaultAuthTokenHeader), "Header used to extract the token (e.g., Authorization)")
	flag.StringVar(&cfg.AuthTokenPrefix, "auth-token-prefix", getEnvAsString("AUTH_TOKEN_PREFIX", config.DefaultAuthTokenPrefix), "Prefix used in the token header (e.g., 'Bearer ')")

	// TLS configuration flags
	flag.BoolVar(&cfg.InsecureSkipVerify, "insecure-skip-verify", getEnvAsBool("INSECURE_SKIP_VERIFY", false), "Skip TLS certificate verification (useful for development, default: false)")

	// Deprecated flags - kept for backward compatibility
	flag.BoolVar(&cfg.StandaloneMode, "standalone-mode", false, "DEPRECATED: Use -deployment-mode=standalone instead")
	flag.BoolVar(&cfg.FederatedPlatform, "federated-platform", false, "DEPRECATED: Use -deployment-mode=federated instead")

	flag.Parse()

	// Handle backward compatibility: if old flags are used, override deployment mode
	if cfg.StandaloneMode {
		cfg.DeploymentMode = config.DeploymentModeStandalone
	} else if cfg.FederatedPlatform {
		cfg.DeploymentMode = config.DeploymentModeFederated
	}

	// Ensure the deprecated boolean fields are consistent with the new deployment mode
	cfg.StandaloneMode = cfg.DeploymentMode.IsStandaloneMode()
	cfg.FederatedPlatform = cfg.DeploymentMode.IsFederatedMode()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	//validate auth method
	if cfg.AuthMethod != config.AuthMethodInternal && cfg.AuthMethod != config.AuthMethodUser {
		logger.Error("invalid auth method: (must be internal or user_token)", "authMethod", cfg.AuthMethod)
		os.Exit(1)
	}

	// Only use for logging errors about logging configuration.
	slog.SetDefault(logger)

	app, err := api.NewApp(cfg, slog.New(logger.Handler()))
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      app.Routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	// Start the server in a goroutine
	go func() {
		logger.Info("starting server", "addr", srv.Addr, "TLS enabled", (certFile != "" && keyFile != ""))
		var err error
		if certFile != "" && keyFile != "" {
			// Configure TLS if both cert and key files are provided
			tlsConfig := &tls.Config{
				MinVersion: tls.VersionTLS13,
				NextProtos: []string{"h2", "http/1.1"},
			}
			srv.TLSConfig = tlsConfig
			err = srv.ListenAndServeTLS(certFile, keyFile)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server ListenAndServe", "error", err)
		}
	}()

	// Graceful shutdown setup
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Wait for shutdown signal
	<-shutdownCh
	logger.Info("shutting down gracefully...")

	// Create a context with timeout for the shutdown process
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown the HTTP server gracefully
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server shutdown failed", "error", err)
	}

	// Shutdown the App gracefully
	if err := app.Shutdown(); err != nil {
		logger.Error("failed to shutdown Kubernetes manager", "error", err)
	}

	logger.Info("server stopped")
	os.Exit(0)
}
