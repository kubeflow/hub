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
//	@Summary	Search catalog models across sources
//	@Tags		ModelCatalogService
//	@ID			findModels
//	@Produce	json
//	@Security	KubeflowUserID
//	@Security	BearerAuth
//	@Param		namespace		query		string	true	"Catalog namespace"
//	@Param		pageSize		query		int		false	"Maximum number of results"
//	@Param		orderBy			query		string	false	"Field to order by"
//	@Param		sortOrder		query		string	false	"Sort direction"	Enums(ASC,DESC)
//	@Param		nextPageToken	query		string	false	"Pagination token"
//	@Param		q				query		string	false	"Search query"
//	@Param		source			query		string	false	"Source IDs to search, comma-separated"
//	@Param		sourceLabel		query		string	false	"Source label filter"
//	@Param		filterQuery		query		string	false	"Model filter expression"
//	@Success	200				{object}	CatalogModelListEnvelope
//	@Failure	400				{object}	ErrorEnvelope
//	@Failure	403				{object}	ErrorEnvelope
//	@Failure	500				{object}	ErrorEnvelope
//	@Router		/api/v1/model_catalog/models [get]
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
