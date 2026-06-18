package api

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/hub/ui/bff/internal/constants"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"
	"github.com/kubeflow/hub/ui/bff/internal/models"
)

type CatalogSourceListEnvelope Envelope[*models.CatalogSourceList, None]
type CatalogModelEnvelope Envelope[*models.CatalogModel, None]
type catalogModelArtifactsListEnvelope Envelope[*models.CatalogModelArtifactList, None]
type CatalogLabelListEnvelope Envelope[*models.CatalogLabelList, None]

func (app *App) GetAllCatalogSourcesHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	catalogSources, err := app.repositories.ModelCatalogClient.GetAllCatalogSources(client, r.URL.Query())

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	sourcesList := CatalogSourceListEnvelope{
		Data: catalogSources,
	}

	err = app.WriteJSON(w, http.StatusOK, sourcesList, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) GetCatalogSourceModelHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	ps.ByName(CatalogSourceId)
	modelName := strings.TrimPrefix(ps.ByName(CatalogModelName), "/")

	newModelName := url.PathEscape(modelName)

	model, err := app.repositories.ModelCatalogClient.GetCatalogSourceModel(client, ps.ByName(CatalogSourceId), newModelName)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	catalogModel := CatalogModelEnvelope{
		Data: model,
	}

	err = app.WriteJSON(w, http.StatusOK, catalogModel, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) GetCatalogSourceModelArtifactsHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	ps.ByName(CatalogSourceId)
	modelName := strings.TrimPrefix(ps.ByName(CatalogModelName), "/")

	newModelName := url.PathEscape(modelName)

	catalogModelArtifacts, err := app.repositories.ModelCatalogClient.GetCatalogSourceModelArtifacts(client, ps.ByName(CatalogSourceId), newModelName, r.URL.Query())

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	catalogModelArtifactList := catalogModelArtifactsListEnvelope{
		Data: catalogModelArtifacts,
	}

	err = app.WriteJSON(w, http.StatusOK, catalogModelArtifactList, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) GetCatalogModelPerformanceArtifactsHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	modelName := strings.TrimPrefix(ps.ByName(CatalogModelName), "/")

	newModelName := url.PathEscape(modelName)

	catalogModelPerformanceArtifacts, err := app.repositories.ModelCatalogClient.GetCatalogModelPerformanceArtifacts(client, ps.ByName(CatalogSourceId), newModelName, r.URL.Query())

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	catalogModelArtifactList := catalogModelArtifactsListEnvelope{
		Data: catalogModelPerformanceArtifacts,
	}

	err = app.WriteJSON(w, http.StatusOK, catalogModelArtifactList, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) GetCatalogModelSecurityArtifactsHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	modelName := strings.TrimPrefix(ps.ByName(CatalogModelName), "/")

	newModelName := url.PathEscape(modelName)

	catalogModelSecurityArtifacts, err := app.repositories.ModelCatalogClient.GetCatalogModelSecurityArtifacts(client, ps.ByName(CatalogSourceId), newModelName, r.URL.Query())

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	catalogModelArtifactList := catalogModelArtifactsListEnvelope{
		Data: catalogModelSecurityArtifacts,
	}

	err = app.WriteJSON(w, http.StatusOK, catalogModelArtifactList, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) GetCatalogLabelsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	catalogLabels, err := app.repositories.ModelCatalogClient.GetCatalogLabels(client, r.URL.Query())

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	labelsList := CatalogLabelListEnvelope{
		Data: catalogLabels,
	}

	err = app.WriteJSON(w, http.StatusOK, labelsList, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
