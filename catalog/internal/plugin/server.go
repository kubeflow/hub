package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"

	"github.com/kubeflow/hub/internal/platform/datastore"
	platformmw "github.com/kubeflow/hub/internal/platform/server/middleware"
)

// swappableHandler is an http.Handler whose underlying handler can be
// atomically replaced. Reconnect rebuilds the router from refreshed plugin
// services and swaps it in here, so in-flight requests finish against the
// handler they started with while new requests observe the refreshed one.
type swappableHandler struct {
	h atomic.Pointer[http.Handler]
}

func (s *swappableHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	(*s.h.Load()).ServeHTTP(w, r)
}

func (s *swappableHandler) Store(h http.Handler) {
	s.h.Store(&h)
}

// ServerConfig holds the dependencies needed to create a plugin Server.
type ServerConfig struct {
	DB                     *gorm.DB
	ConfigPaths            []string
	PerformanceMetricsPath []string
	RepoSet                datastore.RepoSet
	Logger                 *slog.Logger
	CORSAllowedOrigins     []string

	// AlphaSunsetDate, when set, enables RFC 8594 deprecation headers on
	// every plugin route served under an alpha API version, pointing at the
	// plugin's v1 base path as the successor.
	AlphaSunsetDate *time.Time
}

// readinessCheck is a named readiness check evaluated by the /readyz handler.
type readinessCheck struct {
	name string
	fn   func() bool
}

// Server manages the lifecycle of catalog plugins and provides a unified HTTP server.
type Server struct {
	cfg             ServerConfig
	mu              sync.RWMutex
	plugins         []CatalogPlugin
	handler         *swappableHandler
	readinessChecks []readinessCheck
	lastReady       map[string]bool
}

// NewServer creates a new plugin server.
func NewServer(cfg ServerConfig) *Server {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &Server{
		cfg:       cfg,
		plugins:   make([]CatalogPlugin, 0),
		lastReady: make(map[string]bool),
		handler:   &swappableHandler{},
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

	for _, p := range registered {
		basePath := computeBasePath(p)

		pluginCfg := Config{
			DB:                     s.cfg.DB,
			BasePath:               basePath,
			ConfigPaths:            s.cfg.ConfigPaths,
			RepoSet:                s.cfg.RepoSet,
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

// MountRoutes builds the HTTP router with all plugin routes and server
// endpoints, then returns a stable handler backed by it. The returned
// handler stays valid across calls to Reconnect, which rebuilds the router
// from refreshed plugin services and swaps it in underneath.
func (s *Server) MountRoutes() (http.Handler, error) {
	router, err := s.buildRouter()
	if err != nil {
		return nil, err
	}
	s.handler.Store(router)
	return s.handler, nil
}

// buildRouter constructs a fresh router from the plugins' current routes and
// services. Called once by MountRoutes and again by Reconnect after plugin
// services have been refreshed with a new RepoSet, so the router — and every
// provider it mounts — observes the refreshed repositories.
func (s *Server) buildRouter() (chi.Router, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(platformmw.CORSMiddleware(s.cfg.CORSAllowedOrigins))

	if s.cfg.AlphaSunsetDate != nil {
		paths := alphaDeprecatedPaths(s.plugins)
		if len(paths) > 0 {
			router.Use(platformmw.DeprecationMiddleware(platformmw.DeprecationConfig{
				SunsetDate: *s.cfg.AlphaSunsetDate,
				Paths:      paths,
			}))
			for _, p := range paths {
				s.cfg.Logger.Info("alpha API deprecation headers enabled", "prefix", p.Prefix, "successor", p.Successor, "sunset", s.cfg.AlphaSunsetDate.Format("2006-01-02"))
			}
		}
	}

	for _, p := range s.plugins {
		s.cfg.Logger.Info("mounting plugin routes", "plugin", p.Name())
		if err := p.RegisterRoutes(router); err != nil {
			return nil, fmt.Errorf("plugin %s failed to register routes: %w", p.Name(), err)
		}
	}

	router.Get("/healthz", s.healthHandler)
	router.Get("/readyz", s.readyHandler)

	return router, nil
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

// Reconnect updates the server's RepoSet, notifies all Reconnectable plugins
// to refresh their cached repository references and type IDs, then rebuilds
// and atomically swaps in the HTTP router so every mounted provider is
// reconstructed from the refreshed services. This must be called after
// RunMigrations recreates the database schema from scratch (e.g., after
// emptyDir data loss).
func (s *Server) Reconnect(ctx context.Context, repoSet datastore.RepoSet) error {
	s.mu.Lock()
	s.cfg.RepoSet = repoSet
	plugins := make([]CatalogPlugin, len(s.plugins))
	copy(plugins, s.plugins)
	s.mu.Unlock()

	var errs []error
	for _, p := range plugins {
		rc, ok := p.(Reconnectable)
		if !ok {
			continue
		}
		cfg := Config{
			DB:                     s.cfg.DB,
			BasePath:               computeBasePath(p),
			ConfigPaths:            s.cfg.ConfigPaths,
			RepoSet:                repoSet,
			PerformanceMetricsPath: s.cfg.PerformanceMetricsPath,
			Logger:                 s.cfg.Logger.With("plugin", p.Name()),
		}
		if err := rc.Reconnect(ctx, cfg); err != nil {
			errs = append(errs, fmt.Errorf("plugin %s reconnect failed: %w", p.Name(), err))
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	// Rebuild the router so every mounted provider is reconstructed from the
	// plugins' refreshed services, then atomically swap it in. Providers
	// built during the original MountRoutes captured their repositories by
	// value; without this, they would keep querying through the pre-recovery
	// RepoSet and its now-stale type IDs even though the plugins themselves
	// were reconnected above.
	router, err := s.buildRouter()
	if err != nil {
		return fmt.Errorf("remounting routes after reconnect: %w", err)
	}
	s.handler.Store(router)
	return nil
}

// Plugins returns the list of initialized plugins.
func (s *Server) Plugins() []CatalogPlugin {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]CatalogPlugin, len(s.plugins))
	copy(result, s.plugins)
	return result
}

// AddReadinessCheck registers a readiness check evaluated by the /readyz
// handler alongside plugin health. Must be called before serving traffic.
func (s *Server) AddReadinessCheck(name string, fn func() bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readinessChecks = append(s.readinessChecks, readinessCheck{name: name, fn: fn})
}

func (s *Server) healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) readyHandler(w http.ResponseWriter, _ *http.Request) {
	// Write lock: readiness check transition tracking updates lastReady.
	s.mu.Lock()
	defer s.mu.Unlock()

	allHealthy := true
	pluginStatus := make(map[string]bool)

	for _, p := range s.plugins {
		healthy := p.Healthy()
		pluginStatus[p.Name()] = healthy
		if !healthy {
			allHealthy = false
		}
	}

	for _, rc := range s.readinessChecks {
		ready := rc.fn()
		if !ready {
			allHealthy = false
		}
		prev, seen := s.lastReady[rc.name]
		if seen && prev != ready {
			if ready {
				s.cfg.Logger.Info("readiness check recovered", "check", rc.name)
			} else {
				s.cfg.Logger.Warn("readiness check failed", "check", rc.name)
			}
		}
		s.lastReady[rc.name] = ready
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

// alphaDeprecatedPaths returns a deprecated prefix and its v1 successor for
// every plugin whose API version is an alpha (for example "v1alpha1").
// Plugins already on v1 contribute nothing.
func alphaDeprecatedPaths(plugins []CatalogPlugin) []platformmw.DeprecatedPath {
	var paths []platformmw.DeprecatedPath
	for _, p := range plugins {
		version := p.Version()
		if !strings.Contains(version, "alpha") {
			continue
		}
		base := computeBasePath(p)
		versionSegment := "/" + version
		if !strings.HasSuffix(base, versionSegment) {
			continue
		}
		successor := strings.TrimSuffix(base, versionSegment) + "/v1"
		paths = append(paths, platformmw.DeprecatedPath{
			Prefix:    base + "/",
			Successor: successor + "/",
		})
	}
	return paths
}

func computeBasePath(p CatalogPlugin) string {
	if bp, ok := p.(BasePathProvider); ok {
		return bp.BasePath()
	}
	return fmt.Sprintf("/api/%s_catalog/%s", p.Name(), p.Version())
}
