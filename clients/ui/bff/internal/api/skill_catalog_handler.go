package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/hub/ui/bff/internal/constants"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	"github.com/kubeflow/hub/ui/bff/internal/repositories"
)

// catalogServicePort is the port the catalog Service exposes
// (manifests/kustomize/options/catalog/base/service.yaml).
const catalogServicePort = 8080

type SkillListEnvelope Envelope[*models.SkillList, None]
type SkillFilterOptionsListEnvelope Envelope[*models.FilterOptionsList, None]
type SkillEnvelope Envelope[*models.Skill, None]

func (app *App) GetAllSkillsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	skills, err := app.repositories.ModelCatalogClient.GetAllSkills(client, r.URL.Query())
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if err = app.WriteJSON(w, http.StatusOK, SkillListEnvelope{Data: skills}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) GetSkillsFiltersHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := r.Context()

	catalogClient, ok := ctx.Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	filters, err := app.repositories.ModelCatalogClient.GetSkillsFilter(catalogClient)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Filter options come solely from the catalog service, which enumerates them from
	// indexed skills. The BFF used to merge in provider/category/trustTier from the source
	// ConfigMaps as well; that was redundant once a source synced (buildSkillEntity stamps
	// those onto every skill, so they are already enumerated) and actively misleading
	// before it did, offering filter values that matched nothing.

	if err = app.WriteJSON(w, http.StatusOK, SkillFilterOptionsListEnvelope{Data: filters}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) GetSkillHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	skillId := ps.ByName(SkillId)
	if skillId == "" {
		app.badRequestResponse(w, r, fmt.Errorf("skill_id is required"))
		return
	}

	skill, err := app.repositories.ModelCatalogClient.GetSkill(client, skillId, r.URL.Query())
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if err = app.WriteJSON(w, http.StatusOK, SkillEnvelope{Data: skill}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// SkillMarketplaceMetadata carries the URL an agent should hand to
// `/plugin marketplace add`. It is not the URL of this BFF endpoint: this route is
// namespace-scoped and authenticated, which an agent cannot satisfy, while the catalog
// service serves marketplace.json unauthenticated.
type SkillMarketplaceMetadata struct {
	MarketplaceURL string `json:"marketplaceUrl"`
	// External is true when an operator configured an externally reachable URL. When
	// false the URL resolves only inside the cluster, and the UI says so.
	External bool `json:"external"`
}

type SkillMarketplaceEnvelope Envelope[*models.SkillMarketplace, SkillMarketplaceMetadata]

// skillMarketplaceURL returns the marketplace.json URL to advertise.
//
// The default is the catalog service's in-cluster DNS address, which is what agents
// running in the cluster need. It is built from constants rather than a Service lookup
// on purpose: GetModelCatalogWithMode matches on a bare `component` label that this
// repo's manifests do not set, so the lookup cannot be relied on here.
//
// An operator who puts the catalog behind a Route or Ingress sets
// SkillCatalogMarketplaceURL, and that value wins.
func (app *App) skillMarketplaceURL(userNamespace string) (string, bool) {
	if configured := app.config.SkillCatalogMarketplaceURL; configured != "" {
		return configured, true
	}
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d%s/claude/marketplace.json",
		repositories.ModelCatalogServiceName,
		app.skillCatalogNamespace(userNamespace),
		catalogServicePort,
		repositories.SkillCatalogAPIPath,
	), false
}

// GetSkillMarketplaceHandler proxies to the catalog service's claude/marketplace.json
// endpoint, giving the UI the real, backend-computed plugin/marketplace names and
// pinned commit for building install commands, instead of guessing them client-side.
func (app *App) GetSkillMarketplaceHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	marketplace, err := app.repositories.ModelCatalogClient.GetSkillMarketplace(client, r.URL.Query())
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	namespace, _ := r.Context().Value(constants.NamespaceHeaderParameterKey).(string)
	marketplaceURL, external := app.skillMarketplaceURL(namespace)

	if err = app.WriteJSON(w, http.StatusOK, SkillMarketplaceEnvelope{
		Data:     marketplace,
		Metadata: SkillMarketplaceMetadata{MarketplaceURL: marketplaceURL, External: external},
	}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
