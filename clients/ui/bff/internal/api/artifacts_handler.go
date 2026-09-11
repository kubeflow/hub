package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/hub/pkg/openapi-v1"
	"github.com/kubeflow/hub/ui/bff/internal/constants"
)

type ArtifactListEnvelope Envelope[*openapi.ArtifactList, None]
type ArtifactEnvelope Envelope[*openapi.Artifact, None]

func (app *App) CreateArtifactHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelRegistryHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("REST client not found"))
		return
	}

	var envelope ArtifactEnvelope
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error decoding JSON: %v", err.Error()))
		return
	}

	data := *envelope.Data

	jsonData, err := json.Marshal(data)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error marshaling model to JSON: %w", err))
		return
	}

	createdArtifact, err := app.repositories.ModelRegistryClient.CreateArtifact(client, jsonData)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if createdArtifact == nil || (createdArtifact.DocArtifact == nil && createdArtifact.ModelArtifact == nil) {
		app.serverErrorResponse(w, r, fmt.Errorf("created artifact is nil or does not contain valid data"))
		return
	}

	response := ArtifactEnvelope{
		Data: createdArtifact,
	}

	if createdArtifact.DocArtifact != nil && createdArtifact.DocArtifact.Id != nil {
		w.Header().Set("Location", r.URL.JoinPath(*createdArtifact.DocArtifact.Id).String())
	} else if createdArtifact.ModelArtifact != nil && createdArtifact.ModelArtifact.Id != nil {
		w.Header().Set("Location", r.URL.JoinPath(*createdArtifact.ModelArtifact.Id).String())
	} else {
		app.serverErrorResponse(w, r, fmt.Errorf("artifact ID is missing"))
		return
	}

	err = app.WriteJSON(w, http.StatusCreated, response, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) GetArtifactHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelRegistryHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("REST client not found"))
		return
	}

	model, err := app.repositories.ModelRegistryClient.GetArtifact(client, ps.ByName(ArtifactId))
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	result := ArtifactEnvelope{
		Data: model,
	}

	err = app.WriteJSON(w, http.StatusOK, result, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) GetAllArtifactsHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelRegistryHttpClientKey).(httpclient.HTTPClientInterface)

	if !ok {
		app.serverErrorResponse(w, r, errors.New("REST client not found"))
		return
	}

	artifactList, err := app.repositories.ModelRegistryClient.GetAllArtifacts(client, r.URL.Query())

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	artifactsRes := ArtifactListEnvelope{
		Data: artifactList,
	}

	err = app.WriteJSON(w, http.StatusOK, artifactsRes, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
