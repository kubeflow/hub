package model

import (
	"context"
	"fmt"
	"time"

	"github.com/go-chi/chi/v5"

	mapset "github.com/deckarep/golang-set/v2"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog"
	modelcatalogmodels "github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	"github.com/kubeflow/hub/catalog/internal/db/models"
	"github.com/kubeflow/hub/catalog/internal/plugin"
	"github.com/kubeflow/hub/catalog/internal/server/openapi"
)

// mcpSourceProvider is a local interface satisfied by the MCP plugin.
// Used to get MCP sources for the unified FindSources endpoint.
type mcpSourceProvider interface {
	MCPSources() *mcpcatalog.MCPSourceCollection
}

type Plugin struct {
	plugin.PluginBase
	loader   *modelcatalog.ModelLoader
	services modelcatalog.Services
}

func (p *Plugin) Name() string                   { return "model" }
func (p *Plugin) Version() string                { return "v1alpha1" }
func (p *Plugin) Description() string            { return "Model catalog" }
func (p *Plugin) BasePath() string               { return "/api/model_catalog/v1alpha1" }
func (p *Plugin) Healthy() bool                  { return true }
func (p *Plugin) Migrations() []plugin.Migration { return nil }

func (p *Plugin) Init(_ context.Context, cfg plugin.Config) error {
	p.services = modelcatalog.Services{
		CatalogModelRepository:           plugin.GetRepo[modelcatalogmodels.CatalogModelRepository](cfg.RepoSet),
		CatalogArtifactRepository:        plugin.GetRepo[models.CatalogArtifactRepository](cfg.RepoSet),
		CatalogModelArtifactRepository:   plugin.GetRepo[modelcatalogmodels.CatalogModelArtifactRepository](cfg.RepoSet),
		CatalogMetricsArtifactRepository: plugin.GetRepo[modelcatalogmodels.CatalogMetricsArtifactRepository](cfg.RepoSet),
		CatalogSourceRepository:          plugin.GetRepo[models.CatalogSourceRepository](cfg.RepoSet),
		PropertyOptionsRepository:        plugin.GetRepo[models.PropertyOptionsRepository](cfg.RepoSet),
	}

	base := basecatalog.NewBaseLoader(cfg.ConfigPaths)
	p.loader = modelcatalog.NewModelLoader(p.services, base)

	if len(cfg.PerformanceMetricsPath) > 0 {
		perfLoader, err := modelcatalog.NewPerformanceMetricsLoader(
			cfg.PerformanceMetricsPath,
			p.services.CatalogModelRepository,
			p.services.CatalogMetricsArtifactRepository,
			cfg.RepoSet.TypeMap(),
		)
		if err != nil {
			return fmt.Errorf("initializing performance metrics: %w", err)
		}
		p.loader.RegisterEventHandler(perfLoader.Load)
	}

	p.PluginBase = plugin.NewPluginBase(plugin.PluginBaseConfig{
		Name:        "model",
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
		OnLeaderReady: func(ctx context.Context) error {
			poRefresher := models.NewPropertyOptionsRefresher(ctx, p.services.PropertyOptionsRepository, time.Second)
			p.loader.RegisterEventHandler(func(_ context.Context, _ modelcatalog.ModelProviderRecord) error {
				poRefresher.Trigger()
				return nil
			})
			return nil
		},
	})

	return nil
}

func (p *Plugin) RegisterRoutes(router chi.Router) error {
	var mcpSources *mcpcatalog.MCPSourceCollection
	if mcpPlugin, ok := plugin.Get("mcp"); ok {
		if mp, ok := mcpPlugin.(mcpSourceProvider); ok {
			mcpSources = mp.MCPSources()
		}
	}

	svc := openapi.NewModelCatalogServiceAPIService(
		modelcatalog.NewDBCatalog(p.services, p.loader.Sources),
		p.loader.Sources,
		mcpSources,
		p.loader.Labels,
		p.services.CatalogSourceRepository,
	)
	ctrl := openapi.NewModelCatalogServiceAPIController(svc)

	for _, route := range ctrl.OrderedRoutes() {
		router.Method(route.Method, route.Pattern, route.HandlerFunc)
	}

	return nil
}
