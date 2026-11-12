package api

import (
	"errors"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/hub/ui/bff/internal/constants"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"
	"github.com/kubeflow/hub/ui/bff/internal/models"
)

type CatalogFilterOptionsListEnvelope Envelope[*models.FilterOptionsList, None]

// GetCatalogFilterListHandler lists model catalog filter options.
//
//	@Summary	List fields and options available for model filterQuery
//	@Tags		ModelCatalogService
//	@ID			findModelsFilterOptions
//	@Produce	json
//	@Security	KubeflowUserID
//	@Security	BearerAuth
//	@Param		namespace	query		string	true	"Catalog namespace"
//	@Success	200			{object}	CatalogFilterOptionsListEnvelope
//	@Failure	400			{object}	ErrorEnvelope
//	@Failure	403			{object}	ErrorEnvelope
//	@Failure	500			{object}	ErrorEnvelope
//	@Router		/api/v1/model_catalog/models/filter_options [get]
func (app *App) GetCatalogFilterListHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)

	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	catalogFilterOptions, err := app.repositories.ModelCatalogClient.GetCatalogFilterOptions(client)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	catalogFilterOptionList := CatalogFilterOptionsListEnvelope{
		Data: catalogFilterOptions,
	}

	err = app.WriteJSON(w, http.StatusOK, catalogFilterOptionList, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
