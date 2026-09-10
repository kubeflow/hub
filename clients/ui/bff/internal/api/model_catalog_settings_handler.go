package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/hub/ui/bff/internal/constants"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	"github.com/kubeflow/hub/ui/bff/internal/repositories"
)

type ModelCatalogSettingsSourceConfigEnvelope Envelope[*models.CatalogSourceConfig, None]
type ModelCatalogSettingsSourceConfigListEnvelope Envelope[*models.CatalogSourceConfigList, None]
type ModelCatalogSourcePayloadEnvelope Envelope[*models.CatalogSourceConfigPayload, None]

func (app *App) GetAllCatalogSourceConfigsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := r.Context()

	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, errors.New("catalog client not found"))
		return
	}
	catalogSourceConfigs, err := app.repositories.ModelCatalogSettingsRepository.GetAllCatalogSourceConfigs(ctx, client, namespace)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	modelCatalogSource := ModelCatalogSettingsSourceConfigListEnvelope{
		Data: catalogSourceConfigs,
	}

	err = app.WriteJSON(w, http.StatusOK, modelCatalogSource, nil)

	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

func (app *App) GetCatalogSourceConfigHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()

	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
		return
	}

	catalogSourceId := ps.ByName(CatalogSourceId)

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, errors.New("catalog client not found"))
		return
	}

	catalogSourceConfig, err := app.repositories.ModelCatalogSettingsRepository.GetCatalogSourceConfig(ctx, client, namespace, catalogSourceId)

	if err != nil {
		if errors.Is(err, repositories.ErrCatalogSourceNotFound) {
			app.notFoundResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	modelCatalogSource := ModelCatalogSettingsSourceConfigEnvelope{
		Data: catalogSourceConfig,
	}

	err = app.WriteJSON(w, http.StatusOK, modelCatalogSource, nil)

	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

func (app *App) CreateCatalogSourceConfigHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := r.Context()

	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, errors.New("catalog client not found"))
		return
	}

	var envelope ModelCatalogSourcePayloadEnvelope
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error decoding JSON: %v", err.Error()))
		return
	}

	newCatalogSource, err := app.repositories.ModelCatalogSettingsRepository.CreateCatalogSourceConfig(ctx, client, namespace, *envelope.Data)

	if err != nil {
		if errors.Is(err, repositories.ErrCatalogSourceAlreadyExist) ||
			errors.Is(err, repositories.ErrCatalogSourceIdRequired) ||
			errors.Is(err, repositories.ErrUnsupportedCatalogType) ||
			errors.Is(err, repositories.ErrValidationFailed) {
			app.badRequestResponse(w, r, err)
		} else if errors.Is(err, repositories.ErrCatalogSourceConflict) {
			app.conflictResponse(w, r, err.Error())
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	modelCatalogSource := ModelCatalogSettingsSourceConfigEnvelope{
		Data: newCatalogSource,
	}

	w.Header().Set("Location", r.URL.JoinPath(modelCatalogSource.Data.Id).String())
	writeErr := app.WriteJSON(w, http.StatusCreated, modelCatalogSource, nil)
	if writeErr != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error writing JSON"))
		return
	}

}

func (app *App) UpdateCatalogSourceConfigHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()

	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, errors.New("catalog client not found"))
		return
	}

	var envelope ModelCatalogSourcePayloadEnvelope
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error decoding JSON: %v", err.Error()))
		return
	}

	catalogSourceId := ps.ByName(CatalogSourceId)
	if catalogSourceId == "" {
		catalogSourceId = envelope.Data.Id
	}
	updatedCatalogSource, err := app.repositories.ModelCatalogSettingsRepository.UpdateCatalogSourceConfig(ctx, client, namespace, catalogSourceId, *envelope.Data)

	if err != nil {
		if errors.Is(err, repositories.ErrCatalogSourceNotFound) {
			app.notFoundResponse(w, r)
		} else if errors.Is(err, repositories.ErrCannotChangeDefaultSource) ||
			errors.Is(err, repositories.ErrCannotChangeType) {
			app.forbiddenResponse(w, r, err.Error())
		} else if errors.Is(err, repositories.ErrCatalogSourceConflict) {
			app.conflictResponse(w, r, err.Error())
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	modelCatalogSource := ModelCatalogSettingsSourceConfigEnvelope{
		Data: updatedCatalogSource,
	}

	err = app.WriteJSON(w, http.StatusOK, modelCatalogSource, nil)

	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) DeleteCatalogSourceConfigHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()

	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, errors.New("catalog client not found"))
		return
	}

	catalogSourceId := ps.ByName(CatalogSourceId)

	deletedCatalogSource, err := app.repositories.ModelCatalogSettingsRepository.DeleteCatalogSourceConfig(ctx, client, namespace, catalogSourceId)

	if err != nil {
		if errors.Is(err, repositories.ErrCannotDeleteDefaultSource) {
			app.forbiddenResponse(w, r, err.Error())
		} else if errors.Is(err, repositories.ErrCatalogSourceNotFound) {
			app.notFoundResponse(w, r)
		} else if errors.Is(err, repositories.ErrCatalogSourceConflict) {
			app.conflictResponse(w, r, err.Error())
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	modelCatalogSource := ModelCatalogSettingsSourceConfigEnvelope{
		Data: deletedCatalogSource,
	}

	err = app.WriteJSON(w, http.StatusOK, modelCatalogSource, nil)

	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) ClearCatalogSourceCredentialsHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()

	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, errors.New("catalog client not found"))
		return
	}

	catalogSourceId := ps.ByName(CatalogSourceId)
	if catalogSourceId == "" {
		app.badRequestResponse(w, r, repositories.ErrCatalogSourceIdRequired)
		return
	}

	err = app.repositories.ModelCatalogSettingsRepository.ClearHuggingFaceCatalogSourceCredentials(
		ctx, client, namespace, catalogSourceId,
	)
	if err != nil {
		if errors.Is(err, repositories.ErrCatalogSourceIdRequired) ||
			errors.Is(err, repositories.ErrCatalogIdInvalid) ||
			errors.Is(err, repositories.ErrCatalogIDTooLong) {
			app.badRequestResponse(w, r, err)
		} else if errors.Is(err, repositories.ErrCatalogSourceConflict) {
			app.conflictResponse(w, r, err.Error())
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Return JSON (not 204) so mod-arch-core restDELETE can parse the response body.
	if err := app.WriteJSON(w, http.StatusOK, struct{}{}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
