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
	modelcatalogmodels "github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	modelcatalogservice "github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/service"
	"github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog"
	"github.com/kubeflow/hub/catalog/internal/db/models"
	dbservice "github.com/kubeflow/hub/catalog/internal/db/service"
	"github.com/kubeflow/hub/catalog/internal/plugin"
	v1 "github.com/kubeflow/hub/catalog/internal/server/openapi/v1"
	v1alpha1 "github.com/kubeflow/hub/catalog/internal/server/openapi/v1alpha1"
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
	SkillPreviewer() v1.SkillSourcePreviewer
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

	var alphaOpts []v1alpha1.ModelCatalogServiceOption
	var v1Opts []v1.ModelCatalogServiceOption
	if skillPlugin, ok := plugin.Get("skill"); ok {
		if sp, ok := skillPlugin.(skillPreviewProvider); ok {
			previewer := sp.SkillPreviewer()
			alphaOpts = append(alphaOpts, v1alpha1.WithSkillPreviewer(previewer))
			v1Opts = append(v1Opts, v1.WithSkillPreviewer(previewer))
		}
		if ss, ok := skillPlugin.(skillSourceProvider); ok {
			sources := ss.SkillSources()
			alphaOpts = append(alphaOpts, v1alpha1.WithSkillSources(sources))
			v1Opts = append(v1Opts, v1.WithSkillSources(sources))
		}
	}

	dbCatalog := modelcatalog.NewDBCatalog(p.services, p.loader.Sources)

	// Preview path mode reads yamlCatalogPath only from directories the server
	// already loads catalog sources from: the directories of the configured
	// sources files and of every loaded YAML source's catalog file. The path
	// lists are callbacks, so hot-reloaded sources count immediately.
	previewFiles := basecatalog.NewPreviewFiles(
		p.loader.Paths,
		func() []string {
			var paths []string
			for _, src := range p.loader.Sources.AllSources() {
				if path, ok := basecatalog.ResolveYAMLCatalogPath(src.Type, src.Origin, src.Properties); ok {
					paths = append(paths, path)
				}
			}
			if mcpSources != nil {
				for _, src := range mcpSources.AllSources() {
					if path, ok := basecatalog.ResolveYAMLCatalogPath(src.Type, src.Origin, src.Properties); ok {
						paths = append(paths, path)
					}
				}
			}
			return paths
		},
	)
	alphaOpts = append(alphaOpts, v1alpha1.WithPreviewFiles(previewFiles))
	v1Opts = append(v1Opts, v1.WithPreviewFiles(previewFiles))

	// v1alpha1 routes
	alphaSvc := v1alpha1.NewModelCatalogServiceAPIService(
		dbCatalog,
		p.loader.Sources,
		mcpSources,
		agentSources,
		p.loader.Labels,
		p.services.CatalogSourceRepository,
		alphaOpts...,
	)
	alphaCtrl := v1alpha1.NewModelCatalogServiceAPIController(alphaSvc)
	for _, route := range alphaCtrl.OrderedRoutes() {
		router.Method(route.Method, route.Pattern, route.HandlerFunc)
	}

	// v1 routes
	v1Svc := v1.NewModelCatalogServiceAPIService(
		dbCatalog,
		p.loader.Sources,
		mcpSources,
		agentSources,
		p.loader.Labels,
		p.services.CatalogSourceRepository,
		v1Opts...,
	)
	v1Ctrl := v1.NewModelCatalogServiceAPIController(v1Svc)
	for _, route := range v1Ctrl.OrderedRoutes() {
		router.Method(route.Method, route.Pattern, route.HandlerFunc)
	}

	return nil
}
