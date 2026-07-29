package skill

import (
	"context"

	"github.com/go-chi/chi/v5"

	mapset "github.com/deckarep/golang-set/v2"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog"
	skillmodels "github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog/models"
	skillservice "github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog/service"
	"github.com/kubeflow/hub/catalog/internal/db/models"
	"github.com/kubeflow/hub/catalog/internal/plugin"
	"github.com/kubeflow/hub/catalog/internal/server/openapi"
	"github.com/kubeflow/hub/internal/platform/datastore"
)

type Plugin struct {
	*plugin.PluginBase
	loader   *skillcatalog.SkillLoader
	services skillcatalog.Services
}

func (p *Plugin) Name() string                   { return "skill" }
func (p *Plugin) Version() string                { return "v1alpha1" }
func (p *Plugin) Description() string            { return "Skill catalog" }
func (p *Plugin) BasePath() string               { return "/api/skill_catalog/v1alpha1" }
func (p *Plugin) Healthy() bool                  { return true }
func (p *Plugin) Migrations() []plugin.Migration { return nil }

func (p *Plugin) DatastoreEntries() []plugin.DatastoreEntry {
	return []plugin.DatastoreEntry{
		{
			TypeName: "kf.Skill",
			Category: "context",
			Spec: datastore.NewSpecType(skillservice.NewSkillRepository).
				AddString("source_id"),
		},
	}
}

func (p *Plugin) Init(_ context.Context, cfg plugin.Config) error {
	p.services = skillcatalog.Services{
		SkillRepository:           plugin.GetRepo[skillmodels.SkillRepository](cfg.RepoSet),
		CatalogSourceRepository:   plugin.GetRepo[models.CatalogSourceRepository](cfg.RepoSet),
		PropertyOptionsRepository: plugin.GetRepo[models.PropertyOptionsRepository](cfg.RepoSet),
	}

	base := basecatalog.NewBaseLoader(cfg.ConfigPaths)
	p.loader = skillcatalog.NewSkillLoader(p.services, base)

	p.PluginBase = plugin.NewPluginBase(plugin.PluginBaseConfig{
		Name:        "skill",
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
	provider := skillcatalog.NewDBSkillCatalog(p.services)
	ctrl := openapi.NewSkillCatalogServiceAPIController(
		openapi.NewSkillCatalogServiceAPIService(provider),
	)

	for _, route := range ctrl.OrderedRoutes() {
		router.Method(route.Method, route.Pattern, route.HandlerFunc)
	}

	return nil
}
