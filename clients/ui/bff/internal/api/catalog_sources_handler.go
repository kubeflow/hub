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

// GetAllCatalogSourcesHandler lists catalog sources.
//
//	@Summary	List All CatalogSources
//	@Tags		ModelCatalogService
//	@ID			findSources
//	@Produce	json
//	@Security	KubeflowUserID
//	@Security	BearerAuth
//	@Param		namespace		query		string	true	"Catalog namespace"
//	@Param		pageSize		query		int		false	"Maximum number of results"
//	@Param		orderBy			query		string	false	"Field to order by"
//	@Param		sortOrder		query		string	false	"Sort direction"	Enums(ASC,DESC)
//	@Param		nextPageToken	query		string	false	"Pagination token"
//	@Param		name			query		string	false	"Filter by source name"
//	@Param		sourceLabel		query		string	false	"Source label filter"
//	@Success	200				{object}	CatalogSourceListEnvelope
//	@Failure	400				{object}	ErrorEnvelope
//	@Failure	403				{object}	ErrorEnvelope
//	@Failure	500				{object}	ErrorEnvelope
//	@Router		/api/v1/model_catalog/sources [get]
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

// GetCatalogSourceModelHandler returns a model from a catalog source.
//
//	@Summary	Get a CatalogModel
//	@Tags		ModelCatalogService
//	@ID			getModel
//	@Produce	json
//	@Security	KubeflowUserID
//	@Security	BearerAuth
//	@Param		namespace	query		string	true	"Catalog namespace"
//	@Param		source_id	path		string	true	"Catalog source ID"
//	@Param		model_name	path		string	true	"Model name; may contain slashes (for example org/model)"
//	@Success	200			{object}	CatalogModelEnvelope
//	@Failure	400			{object}	ErrorEnvelope
//	@Failure	403			{object}	ErrorEnvelope
//	@Failure	500			{object}	ErrorEnvelope
//	@Router		/api/v1/model_catalog/sources/{source_id}/models/{model_name} [get]
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

// GetCatalogSourceModelArtifactsHandler lists a catalog model's artifacts.
//
//	@Summary	List CatalogModelArtifacts
//	@Tags		ModelCatalogService
//	@ID			getAllModelArtifacts
//	@Produce	json
//	@Security	KubeflowUserID
//	@Security	BearerAuth
//	@Param		namespace		query		string	true	"Catalog namespace"
//	@Param		source_id		path		string	true	"Catalog source ID"
//	@Param		model_name		path		string	true	"Model name; may contain slashes (for example org/model)"
//	@Param		pageSize		query		int		false	"Maximum number of results"
//	@Param		orderBy			query		string	false	"Field to order by"
//	@Param		sortOrder		query		string	false	"Sort direction"	Enums(ASC,DESC)
//	@Param		nextPageToken	query		string	false	"Pagination token"
//	@Param		artifactType	query		string	false	"Artifact type filter"
//	@Param		filterQuery		query		string	false	"Artifact filter expression"
//	@Success	200				{object}	catalogModelArtifactsListEnvelope
//	@Failure	400				{object}	ErrorEnvelope
//	@Failure	403				{object}	ErrorEnvelope
//	@Failure	500				{object}	ErrorEnvelope
//	@Router		/api/v1/model_catalog/sources/{source_id}/artifacts/{model_name} [get]
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

// GetCatalogModelPerformanceArtifactsHandler lists performance artifacts.
//
//	@Summary	List CatalogModelPerformanceArtifacts
//	@Tags		ModelCatalogService
//	@ID			getPerformanceArtifacts
//	@Produce	json
//	@Security	KubeflowUserID
//	@Security	BearerAuth
//	@Param		namespace				query		string	true	"Catalog namespace"
//	@Param		source_id				path		string	true	"Catalog source ID"
//	@Param		model_name				path		string	true	"Model name; may contain slashes (for example org/model)"
//	@Param		pageSize				query		int		false	"Maximum number of results"
//	@Param		orderBy					query		string	false	"Field to order by"
//	@Param		sortOrder				query		string	false	"Sort direction"	Enums(ASC,DESC)
//	@Param		nextPageToken			query		string	false	"Pagination token"
//	@Param		filterQuery				query		string	false	"Performance artifact filter expression"
//	@Param		targetRPS				query		number	false	"Target requests per second"
//	@Param		recommendations			query		boolean	false	"Return recommended configurations"
//	@Param		rpsProperty				query		string	false	"RPS metric property"
//	@Param		latencyProperty			query		string	false	"Latency metric property"
//	@Param		hardwareCountProperty	query		string	false	"Hardware count property"
//	@Param		hardwareTypeProperty	query		string	false	"Hardware type property"
//	@Success	200						{object}	catalogModelArtifactsListEnvelope
//	@Failure	400						{object}	ErrorEnvelope
//	@Failure	403						{object}	ErrorEnvelope
//	@Failure	500						{object}	ErrorEnvelope
//	@Router		/api/v1/model_catalog/sources/{source_id}/performance_artifacts/{model_name} [get]
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

// GetCatalogModelSecurityArtifactsHandler lists security artifacts.
//
//	@Summary	List CatalogModelSecurityArtifacts
//	@Tags		ModelCatalogService
//	@ID			getSecurityArtifacts
//	@Produce	json
//	@Security	KubeflowUserID
//	@Security	BearerAuth
//	@Param		namespace		query		string	true	"Catalog namespace"
//	@Param		source_id		path		string	true	"Catalog source ID"
//	@Param		model_name		path		string	true	"Model name; may contain slashes (for example org/model)"
//	@Param		pageSize		query		int		false	"Maximum number of results"
//	@Param		orderBy			query		string	false	"Field to order by"
//	@Param		sortOrder		query		string	false	"Sort direction"	Enums(ASC,DESC)
//	@Param		nextPageToken	query		string	false	"Pagination token"
//	@Param		filterQuery		query		string	false	"Security artifact filter expression"
//	@Success	200				{object}	catalogModelArtifactsListEnvelope
//	@Failure	400				{object}	ErrorEnvelope
//	@Failure	403				{object}	ErrorEnvelope
//	@Failure	500				{object}	ErrorEnvelope
//	@Router		/api/v1/model_catalog/sources/{source_id}/security_artifacts/{model_name} [get]
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

// GetCatalogLabelsHandler lists catalog source labels.
//
//	@Summary	List All CatalogLabels
//	@Tags		ModelCatalogService
//	@ID			findLabels
//	@Produce	json
//	@Security	KubeflowUserID
//	@Security	BearerAuth
//	@Param		namespace		query		string	true	"Catalog namespace"
//	@Param		pageSize		query		int		false	"Maximum number of results"
//	@Param		orderBy			query		string	false	"Field to order by"
//	@Param		sortOrder		query		string	false	"Sort direction"	Enums(ASC,DESC)
//	@Param		nextPageToken	query		string	false	"Pagination token"
//	@Success	200				{object}	CatalogLabelListEnvelope
//	@Failure	400				{object}	ErrorEnvelope
//	@Failure	403				{object}	ErrorEnvelope
//	@Failure	500				{object}	ErrorEnvelope
//	@Router		/api/v1/model_catalog/labels [get]
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
