package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/hub/pkg/openapi"
	"github.com/kubeflow/hub/ui/bff/internal/constants"
)

type ModelArtifactUpdateEnvelope Envelope[*openapi.ModelArtifactUpdate, None]

// UpdateModelArtifactHandler updates a model artifact.
//
//	@Summary	Patch a Model Artifact
//	@Tags		ModelRegistryService
//	@ID			patchModelArtifact
//	@Accept		json
//	@Produce	json
//	@Security	KubeflowUserID
//	@Security	BearerAuth
//	@Param		namespace			query		string						true	"Model registry namespace"
//	@Param		model_registry_id	path		string						true	"Model registry name"
//	@Param		model_artifact_id	path		string						true	"Model artifact ID"
//	@Param		body				body		ModelArtifactUpdateEnvelope	true	"Model artifact fields to update"
//	@Success	200					{object}	ModelArtifactEnvelope
//	@Failure	400					{object}	ErrorEnvelope
//	@Failure	403					{object}	ErrorEnvelope
//	@Failure	404					{object}	ErrorEnvelope
//	@Failure	409					{object}	ErrorEnvelope
//	@Failure	500					{object}	ErrorEnvelope
//	@Router		/api/v1/model_registry/{model_registry_id}/model_artifacts/{model_artifact_id} [patch]
func (app *App) UpdateModelArtifactHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelRegistryHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("REST client not found"))
		return
	}

	var envelope ModelArtifactUpdateEnvelope
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

	patchedModelArtifact, err := app.repositories.ModelRegistryClient.UpdateModelArtifact(client, ps.ByName(ModelArtifactId), jsonData)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if patchedModelArtifact == nil {
		app.serverErrorResponse(w, r, fmt.Errorf("created artifact is nil or does not contain valid data"))
		return
	}

	responseBody := ModelArtifactEnvelope{
		Data: patchedModelArtifact,
	}

	err = app.WriteJSON(w, http.StatusOK, responseBody, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
