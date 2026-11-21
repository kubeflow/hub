package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/hub/pkg/openapi"
	"github.com/kubeflow/hub/ui/bff/internal/constants"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	"github.com/kubeflow/hub/ui/bff/internal/validation"
)

type RegisteredModelEnvelope Envelope[*openapi.RegisteredModel, None]
type RegisteredModelCreateEnvelope Envelope[*models.RegisteredModelCreate, None]
type RegisteredModelListEnvelope Envelope[*openapi.RegisteredModelList, None]
type RegisteredModelUpdateEnvelope Envelope[*openapi.RegisteredModelUpdate, None]

// GetAllRegisteredModelsHandler lists registered models.
//
//	@Summary		List All RegisteredModels
//	@Description	Lists registered models in the selected registry. Pagination tokens are returned in data.nextPageToken; pass that token back to retrieve the next page. The name parameter searches by entity name.
//	@Tags			ModelRegistryService
//	@ID				getRegisteredModels
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace			query		string	true	"Model registry namespace"
//	@Param			model_registry_id	path		string	true	"Model registry name"
//	@Param			pageSize			query		int		false	"Maximum number of registered models in each page"	example(100)
//	@Param			orderBy				query		string	false	"Ordering field supported by the selected registry"	example(ID)
//	@Param			sortOrder			query		string	false	"Sort direction; defaults to ascending"				Enums(ASC,DESC)	default(ASC)	example(DESC)
//	@Param			nextPageToken		query		string	false	"Opaque token returned in data.nextPageToken by the preceding page"
//	@Param			name				query		string	false	"Name of the entity to search"	example(entity-name)
//	@Success		200					{object}	RegisteredModelListEnvelope
//	@Failure		400					{object}	ErrorEnvelope
//	@Failure		403					{object}	ErrorEnvelope
//	@Failure		404					{object}	ErrorEnvelope
//	@Failure		500					{object}	ErrorEnvelope
//	@Router			/api/v1/model_registry/{model_registry_id}/registered_models [get]
func (app *App) GetAllRegisteredModelsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelRegistryHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("REST client not found"))
		return
	}

	modelList, err := app.repositories.ModelRegistryClient.GetAllRegisteredModels(client, r.URL.Query())
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	modelRegistryRes := RegisteredModelListEnvelope{
		Data: modelList,
	}

	err = app.WriteJSON(w, http.StatusOK, modelRegistryRes, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// CreateRegisteredModelHandler creates a registered model.
//
//	@Summary		Create a RegisteredModel
//	@Description	Creates a named registered model in the selected registry. The JSON request must wrap model fields in data. Custom properties use metadataType plus the matching typed value field. The Location header identifies the created model.
//	@Description	Example request: {"data":{"name":"my-model","description":"An example model","customProperties":{"framework":{"metadataType":"MetadataStringValue","string_value":"pytorch"}}}}.
//	@Tags			ModelRegistryService
//	@ID				createRegisteredModel
//	@Accept			json
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace			query		string					true	"Model registry namespace"
//	@Param			model_registry_id	path		string					true	"Model registry name"
//	@Param			body				body		RegisteredModelCreateEnvelope	true	"Registered model to create in data"
//	@Success		201					{object}	RegisteredModelEnvelope
//	@Header			201					{string}	Location	"URL of the created registered model"
//	@Failure		400					{object}	ErrorEnvelope
//	@Failure		403					{object}	ErrorEnvelope
//	@Failure		404					{object}	ErrorEnvelope
//	@Failure		409					{object}	ErrorEnvelope
//	@Failure		500					{object}	ErrorEnvelope
//	@Router			/api/v1/model_registry/{model_registry_id}/registered_models [post]
func (app *App) CreateRegisteredModelHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelRegistryHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("REST client not found"))
		return
	}

	var envelope RegisteredModelCreateEnvelope
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error decoding JSON: %v", err.Error()))
		return
	}

	if envelope.Data == nil {
		app.badRequestResponse(w, r, errors.New("data is required"))
		return
	}
	data := openapi.RegisteredModel(*envelope.Data)

	if err := validation.ValidateRegisteredModel(data); err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("validation error:: %v", err.Error()))
		return
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error marshaling model to JSON: %w", err))
		return
	}

	createdModel, err := app.repositories.ModelRegistryClient.CreateRegisteredModel(client, jsonData)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if createdModel == nil {
		app.serverErrorResponse(w, r, fmt.Errorf("created model is nil"))
		return
	}

	response := RegisteredModelEnvelope{
		Data: createdModel,
	}

	w.Header().Set("Location", r.URL.JoinPath(*createdModel.Id).String())
	err = app.WriteJSON(w, http.StatusCreated, response, nil)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error writing JSON"))
		return
	}
}

// GetRegisteredModelHandler returns a registered model.
//
//	@Summary		Get a RegisteredModel
//	@Description	Returns a registered model's metadata from the selected registry. The response uses the same data envelope as creation and update.
//	@Tags			ModelRegistryService
//	@ID				getRegisteredModel
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace			query		string	true	"Model registry namespace"
//	@Param			model_registry_id	path		string	true	"Model registry name"
//	@Param			registered_model_id	path		string	true	"Registered model ID"
//	@Success		200					{object}	RegisteredModelEnvelope
//	@Failure		400					{object}	ErrorEnvelope
//	@Failure		403					{object}	ErrorEnvelope
//	@Failure		404					{object}	ErrorEnvelope
//	@Failure		500					{object}	ErrorEnvelope
//	@Router			/api/v1/model_registry/{model_registry_id}/registered_models/{registered_model_id} [get]
func (app *App) GetRegisteredModelHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelRegistryHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("REST client not found"))
		return
	}

	model, err := app.repositories.ModelRegistryClient.GetRegisteredModel(client, ps.ByName(RegisteredModelId))
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if _, ok := model.GetIdOk(); !ok {
		app.notFoundResponse(w, r)
		return
	}

	result := RegisteredModelEnvelope{
		Data: model,
	}

	err = app.WriteJSON(w, http.StatusOK, result, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// UpdateRegisteredModelHandler updates a registered model.
//
//	@Summary		Update a RegisteredModel
//	@Description	Partially updates registered model metadata. Supply only fields to change inside data; create-time requirements must not be applied to the PATCH payload.
//	@Tags			ModelRegistryService
//	@ID				updateRegisteredModel
//	@Accept			json
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace			query		string							true	"Model registry namespace"
//	@Param			model_registry_id	path		string							true	"Model registry name"
//	@Param			registered_model_id	path		string							true	"Registered model ID"
//	@Param			body				body		RegisteredModelUpdateEnvelope	true	"Registered model fields to update"
//	@Success		200					{object}	RegisteredModelEnvelope
//	@Failure		400					{object}	ErrorEnvelope
//	@Failure		403					{object}	ErrorEnvelope
//	@Failure		404					{object}	ErrorEnvelope
//	@Failure		409					{object}	ErrorEnvelope
//	@Failure		500					{object}	ErrorEnvelope
//	@Router			/api/v1/model_registry/{model_registry_id}/registered_models/{registered_model_id} [patch]
func (app *App) UpdateRegisteredModelHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelRegistryHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("REST client not found"))
		return
	}

	var envelope RegisteredModelUpdateEnvelope
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error decoding JSON: %v", err.Error()))
		return
	}

	data := *envelope.Data

	//TODO Validate body, note validation requirements are different to POST (fields are optional)

	jsonData, err := json.Marshal(data)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error marshaling model to JSON: %w", err))
		return
	}

	patchedModel, err := app.repositories.ModelRegistryClient.UpdateRegisteredModel(client, ps.ByName(RegisteredModelId), jsonData)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if patchedModel == nil {
		app.serverErrorResponse(w, r, fmt.Errorf("patched model is nil"))
		return
	}

	responseBody := RegisteredModelEnvelope{
		Data: patchedModel,
	}

	err = app.WriteJSON(w, http.StatusOK, responseBody, nil)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error writing JSON"))
		return
	}
}

// GetAllModelVersionsForRegisteredModelHandler lists versions of a registered model.
//
//	@Summary		List All RegisteredModel's ModelVersions
//	@Description	Lists versions belonging to the selected registered model. Use data.nextPageToken from the response to request the next page.
//	@Tags			ModelRegistryService
//	@ID				getRegisteredModelVersions
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace			query		string	true	"Model registry namespace"
//	@Param			model_registry_id	path		string	true	"Model registry name"
//	@Param			registered_model_id	path		string	true	"Registered model ID"
//	@Param			pageSize			query		int		false	"Maximum number of results"
//	@Param			orderBy				query		string	false	"Field to order by"
//	@Param			sortOrder			query		string	false	"Sort direction"	Enums(ASC,DESC)
//	@Param			nextPageToken		query		string	false	"Pagination token"
//	@Param			name				query		string	false	"Filter by name"
//	@Success		200					{object}	ModelVersionListEnvelope
//	@Failure		400					{object}	ErrorEnvelope
//	@Failure		403					{object}	ErrorEnvelope
//	@Failure		404					{object}	ErrorEnvelope
//	@Failure		500					{object}	ErrorEnvelope
//	@Router			/api/v1/model_registry/{model_registry_id}/registered_models/{registered_model_id}/versions [get]
func (app *App) GetAllModelVersionsForRegisteredModelHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelRegistryHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("REST client not found"))
		return
	}

	versionList, err := app.repositories.ModelRegistryClient.GetAllModelVersionsForRegisteredModel(client, ps.ByName(RegisteredModelId), r.URL.Query())

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	responseBody := ModelVersionListEnvelope{
		Data: versionList,
	}

	err = app.WriteJSON(w, http.StatusOK, responseBody, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// CreateModelVersionForRegisteredModelHandler creates a version in a registered model.
//
//	@Summary		Create a ModelVersion in RegisteredModel
//	@Description	Creates a version associated with the registered model identified in the path. Supply version fields inside data; the Location header points to the created version.
//	@Tags			ModelRegistryService
//	@ID				createRegisteredModelVersion
//	@Accept			json
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace			query		string					true	"Model registry namespace"
//	@Param			model_registry_id	path		string					true	"Model registry name"
//	@Param			registered_model_id	path		string					true	"Registered model ID"
//	@Param			body				body		ModelVersionCreateForRegisteredModelEnvelope	true	"Model version to create"
//	@Success		201					{object}	ModelVersionEnvelope
//	@Header			201					{string}	Location	"URL of the created model version"
//	@Failure		400					{object}	ErrorEnvelope
//	@Failure		403					{object}	ErrorEnvelope
//	@Failure		404					{object}	ErrorEnvelope
//	@Failure		409					{object}	ErrorEnvelope
//	@Failure		500					{object}	ErrorEnvelope
//	@Router			/api/v1/model_registry/{model_registry_id}/registered_models/{registered_model_id}/versions [post]
func (app *App) CreateModelVersionForRegisteredModelHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelRegistryHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("REST client not found"))
		return
	}

	var envelope ModelVersionCreateForRegisteredModelEnvelope
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("error decoding JSON: %v", err.Error()))
		return
	}

	if envelope.Data == nil {
		app.badRequestResponse(w, r, errors.New("data is required"))
		return
	}
	data := openapi.ModelVersion(*envelope.Data)

	if err := validation.ValidateModelVersion(data); err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("validation error:: %v", err.Error()))
		return
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error marshaling model to JSON: %w", err))
		return
	}

	createdVersion, err := app.repositories.ModelRegistryClient.CreateModelVersionForRegisteredModel(client, ps.ByName(RegisteredModelId), jsonData)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if createdVersion == nil {
		app.serverErrorResponse(w, r, fmt.Errorf("created model version is nil"))
		return
	}

	responseBody := ModelVersionEnvelope{
		Data: createdVersion,
	}

	w.Header().Set("Location", ParseURLTemplate(ModelVersionPath, map[string]string{ModelRegistryId: ps.ByName(ModelRegistryId), ModelVersionId: createdVersion.GetId()}))
	err = app.WriteJSON(w, http.StatusCreated, responseBody, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
