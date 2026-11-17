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

type (
	CatalogSourceListEnvelope         Envelope[*models.CatalogSourceList, None]
	CatalogModelEnvelope              Envelope[*models.CatalogModel, None]
	catalogModelArtifactsListEnvelope Envelope[*models.CatalogModelArtifactList, None]
	CatalogLabelListEnvelope          Envelope[*models.CatalogLabelList, None]
)

// GetAllCatalogSourcesHandler lists catalog sources.
//
//	@Summary		List All CatalogSources
//	@Description	Returns configured catalog sources, their labels, enabled state, and loading/authentication status. Use assetType to select sources for models, MCP servers, or agents.
//	@Tags			ModelCatalogService
//	@ID				findSources
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace		query		string	true	"Catalog namespace"
//	@Param			pageSize		query		int		false	"Maximum number of sources in each page"	minimum(1)	default(10)	example(100)
//	@Param			orderBy			query		string	false	"Field to order by"
//	@Param			sortOrder		query		string	false	"Sort direction; defaults to ascending"	Enums(ASC,DESC)	default(ASC)	example(DESC)
//	@Param			nextPageToken	query		string	false	"Opaque token from the previous response; reuse it to retrieve the next page"
//	@Param			name			query		string	false	"Name of the catalog source to search"				example(entity-name)
//	@Param			assetType		query		string	false	"Type of catalog assets whose sources are returned"	Enums(models,mcp_servers,agents,skills)	default(models)	example(models)
//	@Success		200				{object}	CatalogSourceListEnvelope
//	@Failure		400				{object}	ErrorEnvelope
//	@Failure		403				{object}	ErrorEnvelope
//	@Failure		500				{object}	ErrorEnvelope
//	@Router			/api/v1/model_catalog/sources [get]
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
//	@Summary		Get a CatalogModel
//	@Description	Returns model metadata, Markdown documentation, license information, supported tasks, and any validated serving configuration. Model names may contain slashes, for example ibm-granite/granite-3.1-8b-base.
//	@Tags			ModelCatalogService
//	@ID				getModel
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace	query		string									true	"Catalog namespace"
//	@Param			source_id	path		string									true	"Catalog source ID"
//	@Param			model_name	path		string									true	"Model name; may contain slashes (for example org/model)"
//	@Success		200			{object}	CatalogModelEnvelope	"Catalog model metadata"
//	@Failure		400			{object}	ErrorEnvelope
//	@Failure		403			{object}	ErrorEnvelope
//	@Failure		500			{object}	ErrorEnvelope
//	@Router			/api/v1/model_catalog/sources/{source_id}/models/{model_name} [get]
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
//	@Summary		List CatalogModelArtifacts
//	@Description	Lists downloadable model artifacts and associated metrics for a catalog model. filterQuery applies a SQL-like filter to artifact fields and custom properties.
//	@Description	Filter operators: =, !=, <>, >, <, >=, <=, LIKE, ILIKE, IN, AND, OR, and parentheses. Examples: uri LIKE '%s3.amazonaws.com%'; format.string_value = 'pytorch'; ttft_mean > 90.
//	@Description	orderBy accepts ID, NAME, CREATE_TIME, LAST_UPDATE_TIME, or <property>.<value_type>, where value_type is double_value, int_value, or string_value. Examples: mmlu.double_value, framework_type.string_value, hardware_count.int_value.
//	@Description	Invalid custom-property value types and empty property names return an upstream error. An unrecognized ordering format falls back to ID ordering. Artifacts with the selected property sort before artifacts without it.
//	@Tags			ModelCatalogService
//	@ID				getAllModelArtifacts
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace		query		string													true	"Catalog namespace"
//	@Param			source_id		path		string													true	"Catalog source ID"
//	@Param			model_name		path		string													true	"Model name; may contain slashes (for example org/model)"
//	@Param			pageSize		query		int														false	"Maximum number of artifacts in each page"									minimum(1)	default(10)	example(100)
//	@Param			orderBy			query		string													false	"Standard field or <property>.<value_type>; see the operation description"	example(mmlu.double_value)
//	@Param			sortOrder		query		string													false	"Sort direction; defaults to ascending"										Enums(ASC,DESC)	default(ASC)	example(DESC)
//	@Param			nextPageToken	query		string													false	"Opaque token from the previous response; reuse it to retrieve the next page"
//	@Param artifactType query []string false "Artifact categories; comma-separated" Enums(model-artifact,metrics-artifact) collectionFormat(csv)
//	@Param			filterQuery		query		string													false	"SQL-like filter over artifact fields and custom properties"	example(format.string_value='pytorch')
//	@Success		200				{object}	catalogModelArtifactsListEnvelope	"Page of model and metrics artifacts"
//	@Failure		400				{object}	ErrorEnvelope
//	@Failure		403				{object}	ErrorEnvelope
//	@Failure		500				{object}	ErrorEnvelope
//	@Router			/api/v1/model_catalog/sources/{source_id}/artifacts/{model_name} [get]
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
//	@Summary		List CatalogModelPerformanceArtifacts
//	@Description	Returns performance evaluation artifacts for a model. Set targetRPS to calculate replicas and total_requests_per_second custom properties for the requested throughput.
//	@Description	recommendations filters out configurations that are less optimal in approximate cost and latency. Metric-property parameters select the throughput, latency, hardware-count, and hardware-grouping properties used in those calculations.
//	@Description	orderBy accepts ID, NAME, CREATE_TIME, LAST_UPDATE_TIME, or <property>.<value_type>, where value_type is double_value, int_value, or string_value. Examples: mmlu.double_value, framework_type.string_value, hardware_count.int_value.
//	@Description	Invalid custom-property value types and empty property names return an upstream error. Unrecognized ordering formats fall back to ID ordering; artifacts with the selected property sort before those without it.
//	@Description	filterQuery supports SQL-like comparisons, LIKE/ILIKE, IN, AND/OR, parentheses, and typed custom-property access. Example: ttft_mean.double_value < 90 AND hardware_count.int_value = 1.
//	@Tags			ModelCatalogService
//	@ID				getPerformanceArtifacts
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace				query		string													true	"Catalog namespace"
//	@Param			source_id				path		string													true	"Catalog source ID"
//	@Param			model_name				path		string													true	"Model name; may contain slashes (for example org/model)"
//	@Param			pageSize				query		int														false	"Maximum number of artifacts in each page"									minimum(1)	default(10)	example(100)
//	@Param			orderBy					query		string													false	"Standard field or <property>.<value_type>; see the operation description"	example(ttft_mean.double_value)
//	@Param			sortOrder				query		string													false	"Sort direction; defaults to ascending"										Enums(ASC,DESC)	default(ASC)	example(DESC)
//	@Param			nextPageToken			query		string													false	"Opaque token from the previous response; reuse it to retrieve the next page"
//	@Param			filterQuery				query		string													false	"SQL-like filter over performance metrics"															example(hardware_count.int_value=1)
//	@Param			targetRPS				query		int														false	"Target requests per second; enables calculated replicas and total_requests_per_second properties"	example(100)
//	@Param			recommendations			query		boolean													false	"Filter out less optimal configurations based on approximate cost and latency"						default(false)
//	@Param			rpsProperty				query		string													false	"Custom property for the requests-per-second metric"												default(requests_per_second)	example(requests_per_second)
//	@Param			latencyProperty			query		string													false	"Custom property for the latency metric, for example ttft_p90 or p90_latency"						default(ttft_p90)				example(ttft_p90)
//	@Param			hardwareCountProperty	query		string													false	"Custom property for hardware count"																default(hardware_count)
//	@Param			hardwareTypeProperty	query		string													false	"Custom property used to group configurations by hardware type"										default(hardware_type)
//	@Success		200						{object}	catalogModelArtifactsListEnvelope	"Page of performance metrics artifacts"
//	@Failure		400						{object}	ErrorEnvelope
//	@Failure		403						{object}	ErrorEnvelope
//	@Failure		500						{object}	ErrorEnvelope
//	@Router			/api/v1/model_catalog/sources/{source_id}/performance_artifacts/{model_name} [get]
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
//	@Summary		List CatalogModelSecurityArtifacts
//	@Description	Lists security evaluation artifacts by applying metricsType.string_value="security-metrics" upstream. The BFF replaces any caller-supplied filterQuery with that fixed filter.
//	@Tags			ModelCatalogService
//	@ID				getSecurityArtifacts
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace		query		string													true	"Catalog namespace"
//	@Param			source_id		path		string													true	"Catalog source ID"
//	@Param			model_name		path		string													true	"Model name; may contain slashes (for example org/model)"
//	@Param			pageSize		query		int														false	"Maximum number of artifacts in each page"									minimum(1)	default(10)	example(100)
//	@Param			orderBy			query		string													false	"Standard artifact field or a custom property with a typed value suffix"	example(CREATE_TIME)
//	@Param			sortOrder		query		string													false	"Sort direction; defaults to ascending"										Enums(ASC,DESC)	default(ASC)	example(DESC)
//	@Param			nextPageToken	query		string													false	"Opaque token from the previous response; reuse it to retrieve the next page"
//	@Success		200				{object}	catalogModelArtifactsListEnvelope	"Page of security evaluation artifacts"
//	@Failure		400				{object}	ErrorEnvelope
//	@Failure		403				{object}	ErrorEnvelope
//	@Failure		500				{object}	ErrorEnvelope
//	@Router			/api/v1/model_catalog/sources/{source_id}/security_artifacts/{model_name} [get]
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
//	@Summary		List All CatalogLabels
//	@Description	Returns labels used to categorize catalog sources. Use assetType to list labels for models, MCP servers, or agents.
//	@Tags			ModelCatalogService
//	@ID				findLabels
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace		query		string	true	"Catalog namespace"
//	@Param			pageSize		query		int		false	"Maximum number of labels in each page"	minimum(1)	default(10)	example(100)
//	@Param			orderBy			query		string	false	"Field to order by"
//	@Param			sortOrder		query		string	false	"Sort direction; defaults to ascending"	Enums(ASC,DESC)	default(ASC)	example(DESC)
//	@Param			nextPageToken	query		string	false	"Opaque token from the previous response; reuse it to retrieve the next page"
//	@Param			assetType		query		string	false	"Type of catalog assets whose labels are returned"	Enums(models,mcp_servers,agents,skills)	default(models)	example(models)
//	@Success		200				{object}	CatalogLabelListEnvelope
//	@Failure		400				{object}	ErrorEnvelope
//	@Failure		403				{object}	ErrorEnvelope
//	@Failure		500				{object}	ErrorEnvelope
//	@Router			/api/v1/model_catalog/labels [get]
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
