package api

import (
	"errors"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/hub/ui/bff/internal/constants"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"
	"github.com/kubeflow/hub/ui/bff/internal/models"
)

type CatalogModelListEnvelope Envelope[*models.CatalogModelList, None]

// GetAllCatalogModelsAcrossSourcesHandler searches models across catalog sources.
//
//	@Summary		Search catalog models across sources
//	@Description	Lists models from configured catalog sources. Use q for keyword search, source/sourceLabel to select a source, and filterQuery for structured filtering.
//	@Description	filterQuery supports =, !=, <>, >, <, >=, <=, LIKE, ILIKE (case-insensitive matching), IN, AND, OR, and parentheses. Values may be quoted strings, numbers, or booleans.
//	@Description	Access custom properties by name or with a typed suffix such as framework.string_value, accuracy.double_value, or enabled.bool_value. Use backticks around property names containing special characters, for example `mlflow.source.type` = 'notebook'.
//	@Description	Examples: name LIKE '%granite%'; accuracy > 0.95; (name = 'model-a' OR name = 'model-b') AND framework.string_value = 'pytorch'.
//	@Description	orderBy accepts standard fields or custom-property paths such as provider.string_value and artifacts.ifeval.double_value. The BFF forwards only the first value of each query parameter; repeated source/sourceLabel parameters are not combined.
//	@Description	With recommendations=true or orderBy=RECOMMENDED, Pareto filtering ranks models by recommended latency; the metric-property parameters tune that ranking.
//	@Tags			ModelCatalogService
//	@ID				findModels
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace		query		string												true	"Catalog namespace"
//	@Param			pageSize		query		int													false	"Maximum number of models in each page"																		minimum(1)	default(10)	example(100)
//	@Param			orderBy			query		string												false	"Ordering field (CREATE_TIME, LAST_UPDATE_TIME, ID, NAME, ACCURACY, RECOMMENDED) or a custom-property path"	example(NAME)
//	@Param			sortOrder		query		string												false	"Sort direction; defaults to ascending"																		Enums(ASC,DESC)	default(ASC)	example(DESC)
//	@Param			nextPageToken	query		string												false	"Opaque token from the previous response; reuse it to retrieve the next page"
//	@Param			q				query		string												false	"Free-form keyword search used to filter the response"											example(granite)
//	@Param			source			query		string												false	"Catalog source ID to search; only the first supplied value is forwarded"						example(huggingface)
//	@Param			sourceLabel		query		string												false	"Label associated with a catalog source; only the first supplied value is forwarded"			example(validated)
//	@Param			filterQuery		query		string												false	"SQL-like filter expression; see the operation description for operators and property access"	example(name='my-model' AND framework.string_value='pytorch')
//	@Param			recommendations			query		boolean	false	"Apply Pareto filtering"		default(false)
//	@Param			targetRPS				query		int		false	"Target requests per second"	example(100)
//	@Param			latencyProperty			query		string	false	"Latency metric property"		default(ttft_p90)
//	@Param			rpsProperty				query		string	false	"Throughput metric property"	default(requests_per_second)
//	@Param			hardwareCountProperty	query		string	false	"Hardware count property"		default(hardware_count)
//	@Param			hardwareTypeProperty	query		string	false	"Hardware grouping property"	default(hardware_type)
//	@Success		200				{object}	CatalogModelListEnvelope	"Page of catalog models, with a token for the next page"
//	@Failure		400				{object}	ErrorEnvelope
//	@Failure		403				{object}	ErrorEnvelope
//	@Failure		500				{object}	ErrorEnvelope
//	@Router			/api/v1/model_catalog/models [get]
func (app *App) GetAllCatalogModelsAcrossSourcesHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	catalogModels, err := app.repositories.ModelCatalogClient.GetAllCatalogModelsAcrossSources(client, r.URL.Query())
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	sourcesList := CatalogModelListEnvelope{
		Data: catalogModels,
	}

	err = app.WriteJSON(w, http.StatusOK, sourcesList, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
