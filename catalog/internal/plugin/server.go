package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"gorm.io/gorm"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	"github.com/kubeflow/hub/internal/platform/datastore"
)

// ServerConfig holds the dependencies needed to create a plugin Server.
type ServerConfig struct {
	DB                     *gorm.DB
	ConfigPaths            []string
	PerformanceMetricsPath []string
	RepoSet                datastore.RepoSet
	TypeMap                map[string]int32
	Logger                 *slog.Logger
}

// Server manages the lifecycle of catalog plugins and provides a unified HTTP server.
type Server struct {
	cfg     ServerConfig
	mu      sync.RWMutex
	plugins []CatalogPlugin
	router  chi.Router
}

// NewServer creates a new plugin server.
func NewServer(cfg ServerConfig) *Server {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &Server{
		cfg:     cfg,
		plugins: make([]CatalogPlugin, 0),
	}
}

// Init discovers all registered plugins and initializes them.
// Fails fast on the first plugin Init error.
func (s *Server) Init(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	registered := All()
	if len(registered) == 0 {
		s.cfg.Logger.Info("no plugins registered")
		return nil
	}

	var sourceConfig *basecatalog.SourceConfig
	if len(s.cfg.ConfigPaths) > 0 {
		cfg, err := LoadConfig(s.cfg.ConfigPaths[0])
		if err != nil {
			return fmt.Errorf("loading source config: %w", err)
		}
		sourceConfig = cfg
	}

	for _, p := range registered {
		basePath := computeBasePath(p)

		pluginCfg := Config{
			SourceConfig:           sourceConfig,
			DB:                     s.cfg.DB,
			BasePath:               basePath,
			ConfigPaths:            s.cfg.ConfigPaths,
			RepoSet:                s.cfg.RepoSet,
			TypeMap:                s.cfg.TypeMap,
			PerformanceMetricsPath: s.cfg.PerformanceMetricsPath,
			Logger:                 s.cfg.Logger.With("plugin", p.Name()),
		}

		s.cfg.Logger.Info("initializing plugin",
			"plugin", p.Name(),
			"version", p.Version(),
			"basePath", basePath,
		)

		if err := p.Init(ctx, pluginCfg); err != nil {
			return fmt.Errorf("plugin %s init failed: %w", p.Name(), err)
		}

		s.plugins = append(s.plugins, p)
	}

	s.cfg.Logger.Info("all plugins initialized", "count", len(s.plugins))
	return nil
}

// MountRoutes creates the HTTP router with all plugin routes and server endpoints.
// Returns an error if any plugin fails to register its routes.
func (s *Server) MountRoutes() (chi.Router, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.router = chi.NewRouter()
	s.router.Use(middleware.Logger)
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-PINGOTHER"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	for _, p := range s.plugins {
		s.cfg.Logger.Info("mounting plugin routes", "plugin", p.Name())
		if err := p.RegisterRoutes(s.router); err != nil {
			return nil, fmt.Errorf("plugin %s failed to register routes: %w", p.Name(), err)
		}
	}

	s.router.Get("/healthz", s.healthHandler)
	s.router.Get("/readyz", s.readyHandler)

	return s.router, nil
}

// Start starts all plugins' background operations.
func (s *Server) Start(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, p := range s.plugins {
		s.cfg.Logger.Info("starting plugin", "plugin", p.Name())
		if err := p.Start(ctx); err != nil {
			return fmt.Errorf("plugin %s start failed: %w", p.Name(), err)
		}
	}
	return nil
}

// Stop gracefully shuts down all plugins in reverse order.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var errs []error
	for i := len(s.plugins) - 1; i >= 0; i-- {
		p := s.plugins[i]
		s.cfg.Logger.Info("stopping plugin", "plugin", p.Name())
		if err := p.Stop(ctx); err != nil {
			s.cfg.Logger.Error("plugin stop failed", "plugin", p.Name(), "error", err)
			errs = append(errs, fmt.Errorf("plugin %s: %w", p.Name(), err))
		}
	}
	return errors.Join(errs...)
}

// NotifyLeader notifies all LeaderAware plugins that this pod became leader.
// Each plugin's OnBecomeLeader runs in its own goroutine; this method blocks
// until all return (typically when ctx is cancelled / leadership lost).
func (s *Server) NotifyLeader(ctx context.Context) {
	s.mu.RLock()
	plugins := make([]CatalogPlugin, len(s.plugins))
	copy(plugins, s.plugins)
	s.mu.RUnlock()

	var wg sync.WaitGroup
	for _, p := range plugins {
		la, ok := p.(LeaderAware)
		if !ok {
			continue
		}
		name := p.Name()
		wg.Go(func() {
			s.cfg.Logger.Info("plugin becoming leader", "plugin", name)
			if err := la.OnBecomeLeader(ctx); err != nil && !errors.Is(err, context.Canceled) {
				s.cfg.Logger.Error("leader callback failed", "plugin", name, "error", err)
			}
		})
	}
	wg.Wait()
}

// Plugins returns the list of initialized plugins.
func (s *Server) Plugins() []CatalogPlugin {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]CatalogPlugin, len(s.plugins))
	copy(result, s.plugins)
	return result
}

func (s *Server) healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) readyHandler(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	allHealthy := true
	pluginStatus := make(map[string]bool)

	for _, p := range s.plugins {
		healthy := p.Healthy()
		pluginStatus[p.Name()] = healthy
		if !healthy {
			allHealthy = false
		}
	}

	w.Header().Set("Content-Type", "application/json")

	response := map[string]any{
		"plugins": pluginStatus,
	}

	if allHealthy {
		response["status"] = "ready"
		w.WriteHeader(http.StatusOK)
	} else {
		response["status"] = "not_ready"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	_ = json.NewEncoder(w).Encode(response)
}


func computeBasePath(p CatalogPlugin) string {
	if bp, ok := p.(BasePathProvider); ok {
		return bp.BasePath()
	}
	return fmt.Sprintf("/api/%s_catalog/%s", p.Name(), p.Version())
}

