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
	loader    *skillcatalog.SkillLoader
	services  skillcatalog.Services
	previewer *skillcatalog.SkillPreviewer
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
			TypeName: skillservice.SkillTypeName,
			Category: "context",
			// Property names/kinds come from skillcatalog's shared field table so the
			// datastore schema can't drift from the loader's writer and API mapper.
			Spec: skillcatalog.AddSkillProperties(
				datastore.NewSpecType(skillservice.NewSkillRepository),
			),
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

	// Sync fan-out and per-clone limits default to their compiled-in values and
	// are overridable via SKILL_CATALOG_* environment variables (see env_config.go).
	// Private-repo credentials are read at clone time from token files in the mounted
	// git-credentials directory, named by each repository's credentialRef.
	resolveLimits := skillcatalog.ResolveLimitsFromEnv()
	syncLimits := skillcatalog.SyncLimitsFromEnv()
	credentialsDir := skillcatalog.CredentialsDirFromEnv()

	loaderOpts := []skillcatalog.LoaderOption{
		skillcatalog.WithSyncLimits(syncLimits),
		skillcatalog.WithResolveLimits(resolveLimits),
		skillcatalog.WithCredentialsDir(credentialsDir),
	}
	p.loader = skillcatalog.NewSkillLoader(p.services, base, loaderOpts...)

	// The previewer resolves a pasted source config on demand (for the shared
	// sources/preview endpoint). It uses the same clone limits and credentials
	// mount as the loader so a preview behaves like the sync that would follow it.
	p.previewer = skillcatalog.NewSkillPreviewer(resolveLimits, syncLimits, credentialsDir)

	p.PluginBase = plugin.NewPluginBase(plugin.PluginBaseConfig{
		Name:        "skill",
		State:       base,
		Loader:      p.loader,
		FileWatcher: basecatalog.GetMonitor(),
		SourceIDs: func() mapset.Set[string] {
			ids := mapset.NewSet[string]()
			for id := range p.loader.AllSources() {
				ids.Add(id)
			}
			return ids
		},
	})

	return nil
}

// SkillPreviewer returns the skill source previewer for cross-plugin access. The
// model plugin injects it into the shared sources/preview endpoint so an
// assetType: skills preview resolves through the skill catalog's own resolver.
func (p *Plugin) SkillPreviewer() openapi.SkillSourcePreviewer { return p.previewer }

// SkillSources returns the loader's source collection for cross-plugin access.
// The model plugin injects it into FindSources so skill sources appear alongside
// model/MCP/agent sources when queried with assetType: skills.
func (p *Plugin) SkillSources() *skillcatalog.SkillSourceCollection {
	return p.loader.SourceCollection()
}

func (p *Plugin) RegisterRoutes(router chi.Router) error {
	provider := skillcatalog.NewDBSkillCatalog(p.services)
	ctrl := openapi.NewSkillCatalogServiceAPIController(
		openapi.NewSkillCatalogServiceAPIService(provider, p.loader.SourceCollection()),
	)

	for _, route := range ctrl.OrderedRoutes() {
		router.Method(route.Method, route.Pattern, route.HandlerFunc)
	}

	// The marketplace endpoint renders indexed skills as a Claude Code plugin
	// marketplace. It is a per-consumer-format view (Claude's schema, not the
	// catalog's typed API), so it is mounted as an ad-hoc route rather than
	// generated from the OpenAPI spec — mirroring the MCP plugin's logo handler.
	router.Get(p.BasePath()+"/claude/marketplace.json", MarketplaceHandler(provider, skillcatalog.MarketplaceConfigFromEnv()))

	return nil
}
