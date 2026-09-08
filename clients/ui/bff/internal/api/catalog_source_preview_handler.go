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
