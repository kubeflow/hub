package mcp

import (
	"context"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/go-chi/chi/v5"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog"
	mcpcatalogmodels "github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog/models"
	"github.com/kubeflow/hub/catalog/internal/db/models"
	"github.com/kubeflow/hub/catalog/internal/plugin"
	"github.com/kubeflow/hub/catalog/internal/server/openapi"
)

type Plugin struct {
	plugin.PluginBase
	loader   *mcpcatalog.MCPLoader
	services mcpcatalog.Services
}

func (p *Plugin) Name() string                   { return "mcp" }
func (p *Plugin) Version() string                { return "v1alpha1" }
func (p *Plugin) Description() string            { return "MCP server catalog" }
func (p *Plugin) BasePath() string               { return "/api/mcp_catalog/v1alpha1" }
func (p *Plugin) Healthy() bool                  { return true }
func (p *Plugin) Migrations() []plugin.Migration { return nil }

// MCPSources returns the MCP source collection for cross-plugin access.
func (p *Plugin) MCPSources() *mcpcatalog.MCPSourceCollection {
	return p.loader.Sources
}

func (p *Plugin) Init(_ context.Context, cfg plugin.Config) error {
	p.services = mcpcatalog.Services{
		MCPServerRepository:       plugin.GetRepo[mcpcatalogmodels.MCPServerRepository](cfg.RepoSet),
		MCPServerToolRepository:   plugin.GetRepo[mcpcatalogmodels.MCPServerToolRepository](cfg.RepoSet),
		CatalogSourceRepository:   plugin.GetRepo[models.CatalogSourceRepository](cfg.RepoSet),
		PropertyOptionsRepository: plugin.GetRepo[models.PropertyOptionsRepository](cfg.RepoSet),
	}

	base := basecatalog.NewBaseLoader(cfg.ConfigPaths)
	p.loader = mcpcatalog.NewMCPLoaderWithState(p.services, base)

	p.PluginBase = plugin.NewPluginBase(plugin.PluginBaseConfig{
		Name:        "mcp",
		State:       base,
		Loader:      p.loader,
		FileWatcher: basecatalog.GetMonitor(),
		SourceIDs: func() mapset.Set[string] {
			ids := mapset.NewSet[string]()
			for id := range p.loader.Sources.AllSources() {
				ids.Add(id)
			}
			return ids
		},
	})

	return nil
}

func (p *Plugin) RegisterRoutes(router chi.Router) error {
	mcpProvider := mcpcatalog.NewDBMCPCatalog(p.services, p.loader.Sources, func(name string) (map[string]basecatalog.FieldFilter, bool) {
		return p.loader.Sources.GetNamedQuery(name)
	})
	ctrl := openapi.NewMCPCatalogServiceAPIController(
		openapi.NewMCPCatalogServiceAPIService(mcpProvider, p.loader.Sources),
	)

	for _, route := range ctrl.OrderedRoutes() {
		router.Method(route.Method, route.Pattern, route.HandlerFunc)
	}

	return nil
}
