package model

import (
	"context"
	"fmt"
	"time"

	"github.com/go-chi/chi/v5"

	mapset "github.com/deckarep/golang-set/v2"

	"github.com/kubeflow/hub/catalog/internal/catalog/agentcatalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog"
	modelcatalogmodels "github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	modelcatalogservice "github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/service"
	"github.com/kubeflow/hub/catalog/internal/db/models"
	dbservice "github.com/kubeflow/hub/catalog/internal/db/service"
	"github.com/kubeflow/hub/catalog/internal/plugin"
	"github.com/kubeflow/hub/catalog/internal/server/openapi"
	"github.com/kubeflow/hub/internal/platform/datastore"
)

// mcpSourceProvider is a local interface satisfied by the MCP plugin.
// Used to get MCP sources for the unified FindSources endpoint.
type mcpSourceProvider interface {
	MCPSources() *mcpcatalog.MCPSourceCollection
}

// agentSourceProvider is a local interface satisfied by the agent plugin.
// Used to get agent sources for the unified FindSources endpoint.
type agentSourceProvider interface {
	AgentSources() *agentcatalog.AgentSourceCollection
}

// skillPreviewProvider is a local interface satisfied by the skill plugin. It
// supplies the previewer that lets the shared sources/preview endpoint handle
// assetType: skills without the model service depending on the skill catalog.
type skillPreviewProvider interface {
	SkillPreviewer() openapi.SkillSourcePreviewer
}

// skillSourceProvider is a local interface satisfied by the skill plugin.
// Used to get skill sources for the unified FindSources endpoint.
type skillSourceProvider interface {
	SkillSources() *skillcatalog.SkillSourceCollection
}

type Plugin struct {
	*plugin.PluginBase
	loader     *modelcatalog.ModelLoader
	services   modelcatalog.Services
	perfLoader *modelcatalog.PerformanceMetricsLoader
}

func (p *Plugin) Name() string                   { return "model" }
func (p *Plugin) Version() string                { return "v1alpha1" }
func (p *Plugin) Description() string            { return "Model catalog" }
func (p *Plugin) BasePath() string               { return "/api/model_catalog/v1alpha1" }
func (p *Plugin) Migrations() []plugin.Migration { return nil }

func (p *Plugin) DatastoreEntries() []plugin.DatastoreEntry {
	return []plugin.DatastoreEntry{
		{
			TypeName: dbservice.CatalogModelTypeName,
			Category: "context",
			Spec: datastore.NewSpecType(modelcatalogservice.NewCatalogModelRepository).
				AddString("source_id").
				AddString("description").
				AddString("owner").
				AddString("state").
				AddStruct("language").
				AddString("library_name").
				AddString("license_link").
				AddString("license").
				AddString("logo").
				AddString("maturity").
				AddString("provider").
				AddString("readme").
				AddStruct("tasks"),
		},
		{
			TypeName: dbservice.CatalogModelArtifactTypeName,
			Category: "artifact",
			Spec: datastore.NewSpecType(modelcatalogservice.NewCatalogModelArtifactRepository).
				AddString("uri"),
		},
		{
			TypeName: dbservice.CatalogMetricsArtifactTypeName,
			Category: "artifact",
			Spec: datastore.NewSpecType(modelcatalogservice.NewCatalogMetricsArtifactRepository).
				AddString("metricsType"),
		},
	}
}

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
		p.perfLoader = perfLoader
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
			// Models are fully written before OnLeaderReady is called (WaitForInflightWrites
			// runs first), so the event handler above will not fire for the initial load.
			// Do a synchronous refresh here so filter options are populated immediately.
			if err := p.services.PropertyOptionsRepository.Refresh(models.ContextPropertyOptionType); err != nil {
				return fmt.Errorf("refreshing context property options: %w", err)
			}
			if err := p.services.PropertyOptionsRepository.Refresh(models.ArtifactPropertyOptionType); err != nil {
				return fmt.Errorf("refreshing artifact property options: %w", err)
			}
			return nil
		},
	})

	return nil
}

func (p *Plugin) Reconnect(_ context.Context, cfg plugin.Config) error {
	p.services = modelcatalog.Services{
		CatalogModelRepository:           plugin.GetRepo[modelcatalogmodels.CatalogModelRepository](cfg.RepoSet),
		CatalogArtifactRepository:        plugin.GetRepo[models.CatalogArtifactRepository](cfg.RepoSet),
		CatalogModelArtifactRepository:   plugin.GetRepo[modelcatalogmodels.CatalogModelArtifactRepository](cfg.RepoSet),
		CatalogMetricsArtifactRepository: plugin.GetRepo[modelcatalogmodels.CatalogMetricsArtifactRepository](cfg.RepoSet),
		CatalogSourceRepository:          plugin.GetRepo[models.CatalogSourceRepository](cfg.RepoSet),
		PropertyOptionsRepository:        plugin.GetRepo[models.PropertyOptionsRepository](cfg.RepoSet),
	}
	p.loader.UpdateServices(p.services)
	if p.perfLoader != nil {
		if err := p.perfLoader.UpdateRepos(
			p.services.CatalogModelRepository,
			p.services.CatalogMetricsArtifactRepository,
			cfg.RepoSet.TypeMap(),
		); err != nil {
			return fmt.Errorf("updating performance metrics repos: %w", err)
		}
	}
	return nil
}

func (p *Plugin) RegisterRoutes(router chi.Router) error {
	var mcpSources *mcpcatalog.MCPSourceCollection
	if mcpPlugin, ok := plugin.Get("mcp"); ok {
		if mp, ok := mcpPlugin.(mcpSourceProvider); ok {
			mcpSources = mp.MCPSources()
		}
	}

	var agentSources *agentcatalog.AgentSourceCollection
	if agentPlugin, ok := plugin.Get("agent"); ok {
		if ap, ok := agentPlugin.(agentSourceProvider); ok {
			agentSources = ap.AgentSources()
		}
	}

	var svcOpts []openapi.ModelCatalogServiceOption
	if skillPlugin, ok := plugin.Get("skill"); ok {
		if sp, ok := skillPlugin.(skillPreviewProvider); ok {
			svcOpts = append(svcOpts, openapi.WithSkillPreviewer(sp.SkillPreviewer()))
		}
		if ss, ok := skillPlugin.(skillSourceProvider); ok {
			svcOpts = append(svcOpts, openapi.WithSkillSources(ss.SkillSources()))
		}
	}

	svc := openapi.NewModelCatalogServiceAPIService(
		modelcatalog.NewDBCatalog(p.services, p.loader.Sources),
		p.loader.Sources,
		mcpSources,
		agentSources,
		p.loader.Labels,
		p.services.CatalogSourceRepository,
		svcOpts...,
	)
	ctrl := openapi.NewModelCatalogServiceAPIController(svc)

	for _, route := range ctrl.OrderedRoutes() {
		router.Method(route.Method, route.Pattern, route.HandlerFunc)
	}

	return nil
}
