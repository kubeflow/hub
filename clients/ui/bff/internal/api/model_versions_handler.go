package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/hub/pkg/openapi-v1"
	"github.com/kubeflow/hub/ui/bff/internal/constants"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"
	"github.com/kubeflow/hub/ui/bff/internal/validation"
)

type ModelVersionEnvelope Envelope[*openapi.ModelVersion, None]
type ModelVersionListEnvelope Envelope[*openapi.ModelVersionList, None]
type ModelVersionUpdateEnvelope Envelope[*openapi.ModelVersionUpdate, None]

type ModelArtifactListEnvelope Envelope[*openapi.ModelArtifactList, None]
type ModelArtifactEnvelope Envelope[*openapi.ModelArtifact, None]

func (app *App) GetAllModelVersionHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelRegistryHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("REST client not found"))
		return
	}

	versionList, err := app.repositories.ModelRegistryClient.GetAllModelVersions(client)
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

func (app *App) GetModelVersionHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelRegistryHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("REST client not found"))
		return
	}

	model, err := app.repositories.ModelRegistryClient.GetModelVersion(client, ps.ByName(ModelVersionId))
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if _, ok := model.GetIdOk(); !ok {
		app.notFoundResponse(w, r)
		return
	}

	result := ModelVersionEnvelope{
		Data: model,
	}

	err = app.WriteJSON(w, http.StatusOK, result, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) CreateModelVersionHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelRegistryHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("REST client not found"))
		return
	}

	var envelope ModelVersionEnvelope
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error decoding JSON: %v", err.Error()))
		return
	}

	data := *envelope.Data

	if err := validation.ValidateModelVersion(data); err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("validation error:: %v", err.Error()))
		return
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error marshaling ModelVersion to JSON: %w", err))
		return
	}

	createdVersion, err := app.repositories.ModelRegistryClient.CreateModelVersion(client, jsonData)
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
		app.serverErrorResponse(w, r, fmt.Errorf("created ModelVersion is nil"))
		return
	}

	response := ModelVersionEnvelope{
		Data: createdVersion,
	}

	w.Header().Set("Location", r.URL.JoinPath(*createdVersion.Id).String())
	err = app.WriteJSON(w, http.StatusCreated, response, nil)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error writing JSON"))
		return
	}
}

func (app *App) UpdateModelVersionHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelRegistryHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("REST client not found"))
		return
	}

	var envelope ModelVersionUpdateEnvelope
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error decoding JSON: %v", err.Error()))
		return
	}

	data := *envelope.Data

	//TODO add validation - note updating requires different rules to create as fields are optional.

	jsonData, err := json.Marshal(data)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error marshaling ModelVersion to JSON: %w", err))
		return
	}

	patchedModel, err := app.repositories.ModelRegistryClient.UpdateModelVersion(client, ps.ByName(ModelVersionId), jsonData)
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
		app.serverErrorResponse(w, r, fmt.Errorf("patched ModelVersion is nil"))
		return
	}

	responseBody := ModelVersionEnvelope{
		Data: patchedModel,
	}

	err = app.WriteJSON(w, http.StatusOK, responseBody, nil)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error writing JSON"))
		return
	}
}

func (app *App) GetAllModelArtifactsByModelVersionHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelRegistryHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("REST client not found"))
		return
	}

	data, err := app.repositories.ModelRegistryClient.GetModelArtifactsByModelVersion(client, ps.ByName(ModelVersionId), r.URL.Query())
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	result := ModelArtifactListEnvelope{
		Data: data,
	}

	err = app.WriteJSON(w, http.StatusOK, result, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) CreateModelArtifactByModelVersionHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelRegistryHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("REST client not found"))
		return
	}

	var envelope ModelArtifactEnvelope
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error decoding JSON: %v", err.Error()))
		return
	}

	data := *envelope.Data

	if err := validation.ValidateModelArtifact(data); err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("validation error:: %v", err.Error()))
		return
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error marshaling ModelVersion to JSON: %w", err))
		return
	}

	createdArtifact, err := app.repositories.ModelRegistryClient.CreateModelArtifactByModelVersion(client, ps.ByName(ModelVersionId), jsonData)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if createdArtifact == nil {
		app.serverErrorResponse(w, r, fmt.Errorf("created ModelArtifact is nil"))
		return
	}

	response := ModelArtifactEnvelope{
		Data: createdArtifact,
	}

	w.Header().Set("Location", ParseURLTemplate(ModelArtifactPath, map[string]string{
		ModelRegistryId: ps.ByName(ModelRegistryId),
		ModelArtifactId: createdArtifact.GetId(),
	}))
	err = app.WriteJSON(w, http.StatusCreated, response, nil)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error writing JSON"))
		return
	}
}
