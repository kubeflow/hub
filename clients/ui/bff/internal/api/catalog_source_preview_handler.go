package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/hub/ui/bff/internal/constants"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	"github.com/kubeflow/hub/ui/bff/internal/repositories"
)

type CatalogSourcePreviewEnvelope Envelope[*models.CatalogSourcePreviewResult, None]

// CreateCatalogSourcePreviewHandler previews catalog source configuration effects.
//
//	@Summary		Preview catalog source configuration effects
//	@Description	Tests a source configuration without saving it and returns assets with their inclusion/exclusion status. Accepts a JSON data envelope, not multipart; the BFF translates it to config/catalogData multipart parts for the catalog service.
//	@Description	For YAML sources, provide catalog content in data.properties.yaml or a configured data.properties.yamlCatalogPath. For Hugging Face sources, properties may include apiKey and allowedOrganization; when editing a saved source, stored credentials can be resolved by data.id.
//	@Description	assetType=models uses includedModels/excludedModels patterns and model summary counts. assetType=mcp_servers uses includedServers/excludedServers and asset summary counts. Hugging Face results may include hfAccessType and hfGatedAccessGranted; the summary reports inaccessible gated models across all pages.
//	@Description	Not available in Kubeflow integration mode.
//	@Tags			ModelCatalogService
//	@ID				previewCatalogSource
//	@Accept			json
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace		query		string												true	"Catalog namespace"
//	@Param			pageSize		query		int													false	"Maximum number of assets in each preview page"	minimum(1)	default(10)	example(100)
//	@Param			nextPageToken	query		string												false	"Opaque token from the preceding preview page"
//	@Param			filterStatus	query		string												false	"all returns every evaluated asset; included/excluded returns only that inclusion status"	Enums(all,included,excluded)	default(all)
//	@Param			assetType		query		string												false	"Asset type to preview; selects the inclusion/exclusion fields and summary counters"		Enums(models,mcp_servers)		default(models)
//	@Param			body			body		Envelope[models.CatalogSourcePreviewRequest,None]	true	"Source preview configuration in data"
//	@Success		200				{object}	CatalogSourcePreviewEnvelope	"Evaluated assets, pagination token, and source-wide summary"
//	@Failure		400				{object}	ErrorEnvelope
//	@Failure		403				{object}	ErrorEnvelope
//	@Failure		422				{object}	ErrorEnvelope "Invalid configuration or catalog content"
//	@Failure		500				{object}	ErrorEnvelope
//	@Router			/api/v1/settings/model_catalog/source_preview [post]
func (app *App) CreateCatalogSourcePreviewHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := r.Context()

	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
		return
	}

	k8sClient, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, errors.New("catalog client not found"))
		return
	}

	client, ok := ctx.Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	var requestBody struct {
		Data models.CatalogSourcePreviewRequest `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error decoding JSON: %v", err.Error()))
		return
	}
	sourcePreviewPayload := requestBody.Data

	if err := app.repositories.ModelCatalogSettingsRepository.PrepareCatalogSourcePreviewRequest(
		ctx, k8sClient, namespace, &sourcePreviewPayload,
	); err != nil {
		if errors.Is(err, repositories.ErrValidationFailed) ||
			errors.Is(err, repositories.ErrCatalogSourceIdRequired) ||
			errors.Is(err, repositories.ErrCatalogIdInvalid) ||
			errors.Is(err, repositories.ErrCatalogIDTooLong) {
			app.badRequestResponse(w, r, err)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	sourcePreview, err := app.repositories.ModelCatalogClient.CreateCatalogSourcePreview(client, sourcePreviewPayload, r.URL.Query())

	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	catalogSourcePreview := CatalogSourcePreviewEnvelope{
		Data: sourcePreview,
	}

	err = app.WriteJSON(w, http.StatusOK, catalogSourcePreview, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
