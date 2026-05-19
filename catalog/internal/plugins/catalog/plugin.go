package catalog

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang/glog"

	"github.com/kubeflow/hub/catalog/internal/catalog"
	mcpcatalogmodels "github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog/models"
	modelcatalogmodels "github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	"github.com/kubeflow/hub/catalog/internal/db/models"
	"github.com/kubeflow/hub/catalog/internal/db/service"
	"github.com/kubeflow/hub/catalog/internal/plugin"
	"github.com/kubeflow/hub/catalog/internal/server/openapi"
	"github.com/kubeflow/hub/internal/platform/datastore"
)

// Plugin wraps both the model catalog and MCP catalog under a single plugin.
// The two catalogs share a unified Loader with common leader state and file watching.
type Plugin struct {
	loader   *catalog.Loader
	services service.Services
	typeMap  map[string]int32
}

func (p *Plugin) Name() string                    { return "catalog" }
func (p *Plugin) Version() string                  { return "v1alpha1" }
func (p *Plugin) Description() string              { return "Unified model and MCP catalog" }
func (p *Plugin) BasePath() string                 { return "/api/catalog/v1alpha1" }
func (p *Plugin) Healthy() bool                    { return true }
func (p *Plugin) Migrations() []plugin.Migration   { return nil }

func (p *Plugin) Init(_ context.Context, cfg plugin.Config) error {
	p.typeMap = cfg.TypeMap

	services := service.NewServices(
		getRepo[modelcatalogmodels.CatalogModelRepository](cfg.RepoSet),
		getRepo[models.CatalogArtifactRepository](cfg.RepoSet),
		getRepo[modelcatalogmodels.CatalogModelArtifactRepository](cfg.RepoSet),
		getRepo[modelcatalogmodels.CatalogMetricsArtifactRepository](cfg.RepoSet),
		getRepo[models.CatalogSourceRepository](cfg.RepoSet),
		getRepo[models.PropertyOptionsRepository](cfg.RepoSet),
		getRepo[mcpcatalogmodels.MCPServerRepository](cfg.RepoSet),
		getRepo[mcpcatalogmodels.MCPServerToolRepository](cfg.RepoSet),
	)
	p.services = services

	p.loader = catalog.NewLoader(services, cfg.ConfigPaths)

	if len(cfg.PerformanceMetricsPath) > 0 {
		perfLoader, err := catalog.NewPerformanceMetricsLoader(
			cfg.PerformanceMetricsPath,
			services.CatalogModelRepository,
			services.CatalogMetricsArtifactRepository,
			p.typeMap,
		)
		if err != nil {
			return fmt.Errorf("initializing performance metrics: %w", err)
		}
		p.loader.RegisterEventHandler(perfLoader.Load)
	}

	return nil
}

func (p *Plugin) RegisterRoutes(router chi.Router) error {
	mcpSources := p.loader.MCPSources()

	svc := openapi.NewModelCatalogServiceAPIService(
		catalog.NewDBCatalog(p.services, p.loader.Sources()),
		p.loader.Sources(),
		mcpSources,
		p.loader.Labels(),
		p.services.CatalogSourceRepository,
	)
	ctrl := openapi.NewModelCatalogServiceAPIController(svc)

	mcpProvider := catalog.NewDBMCPCatalog(p.services, mcpSources, func(name string) (map[string]catalog.FieldFilter, bool) {
		return mcpSources.GetNamedQuery(name)
	})
	mcpSvc := openapi.NewMCPCatalogServiceAPIService(mcpProvider, mcpSources)
	mcpCtrl := openapi.NewMCPCatalogServiceAPIController(mcpSvc)

	for _, route := range ctrl.OrderedRoutes() {
		router.Method(route.Method, route.Pattern, route.HandlerFunc)
	}
	for _, route := range mcpCtrl.OrderedRoutes() {
		router.Method(route.Method, route.Pattern, route.HandlerFunc)
	}

	return nil
}

func (p *Plugin) Start(ctx context.Context) error {
	glog.Info("Starting catalog plugin in read-only mode (standby)")
	return p.loader.StartReadOnly(ctx)
}

func (p *Plugin) OnBecomeLeader(ctx context.Context) error {
	glog.Info("Catalog plugin becoming leader")
	poRefresher := models.NewPropertyOptionsRefresher(ctx, p.services.PropertyOptionsRepository, time.Second)
	p.loader.RegisterEventHandler(func(_ context.Context, _ catalog.ModelProviderRecord) error {
		poRefresher.Trigger()
		return nil
	})
	return p.loader.StartLeader(ctx)
}

func (p *Plugin) Stop(_ context.Context) error {
	return p.loader.Shutdown()
}

func getRepo[T any](repoSet datastore.RepoSet) T {
	repo, err := repoSet.Repository(reflect.TypeFor[T]())
	if err != nil {
		panic(fmt.Sprintf("unable to get repository: %v", err))
	}
	return repo.(T)
}
