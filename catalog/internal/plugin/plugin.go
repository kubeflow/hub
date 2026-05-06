package plugin

import (
	"context"
	"flag"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
)

// CatalogPlugin defines the interface that all catalog plugins must implement.
// Plugins register themselves via init() using the Register function.
type CatalogPlugin interface {
	// Name returns the plugin name (e.g., "model", "dataset").
	// Used for routing and configuration lookup.
	Name() string

	// Version returns the API version (e.g., "v1alpha1").
	Version() string

	// Description returns a human-readable description of the plugin.
	Description() string

	// Init initializes the plugin with its configuration.
	// Called once during server startup before Start.
	Init(ctx context.Context, cfg Config) error

	// Start begins background operations (hot-reload, watchers, etc.).
	// Called after Init and after database migrations.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the plugin.
	// Called during server shutdown.
	Stop(ctx context.Context) error

	// Healthy returns true if the plugin is functioning correctly.
	// Used for health check endpoints.
	Healthy() bool

	// RegisterRoutes mounts the plugin's HTTP routes on the provided router.
	// The router is already scoped to the plugin's base path.
	RegisterRoutes(router chi.Router) error

	// Migrations returns database migrations for this plugin.
	// Migrations are applied in order during server initialization.
	Migrations() []Migration
}

// BasePathProvider is an optional interface that plugins can implement
// to specify their own API base path. If not implemented, the server
// computes it as /api/{name}_catalog/{version}.
type BasePathProvider interface {
	BasePath() string
}

// SourceKeyProvider is an optional interface that plugins can implement
// to specify which key in sources.yaml they respond to.
// If not implemented, the plugin name is used as the config key.
type SourceKeyProvider interface {
	SourceKey() string
}

// FlagProvider is an optional interface that plugins can implement
// to register custom CLI flags before flag parsing.
type FlagProvider interface {
	RegisterFlags(fs *flag.FlagSet)
}

// CatalogLoader defines the interface for data loading strategies.
type CatalogLoader interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Migration represents a database migration for a plugin.
type Migration struct {
	Version     string
	Description string
	Up          func(db *gorm.DB) error
	Down        func(db *gorm.DB) error
}

// Config is passed to each plugin during Init.
type Config struct {
	// SourceConfig is the parsed sources.yaml configuration.
	// Plugins use basecatalog methods (GetModelCatalogs, etc.) to extract
	// their relevant sections.
	//
	// This couples plugins to basecatalog's schema — adding a new catalog type
	// requires extending SourceConfig. Full decoupling (plugin-defined config
	// types with raw YAML routing) is a server orchestration concern.
	SourceConfig *basecatalog.SourceConfig

	// DB is the shared database connection.
	DB *gorm.DB

	// BasePath is the API base path for this plugin (e.g., "/api/model_catalog/v1alpha1").
	BasePath string

	// ConfigPaths are the paths to all sources.yaml files being used.
	ConfigPaths []string
}
