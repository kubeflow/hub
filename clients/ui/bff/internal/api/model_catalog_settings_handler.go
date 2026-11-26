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

type (
	ModelCatalogSettingsSourceConfigEnvelope     Envelope[*models.CatalogSourceConfig, None]
	ModelCatalogSettingsSourceConfigListEnvelope Envelope[*models.CatalogSourceConfigList, None]
	ModelCatalogSourcePayloadEnvelope            Envelope[*models.CatalogSourceConfigPayload, None]
)

type ModelCatalogSourceCreateEnvelope struct {
	Data     json.RawMessage `json:"data" validate:"required" swaggertype:"object" oneOf:"models.YamlCatalogSourceCreate,models.HFCatalogSourceCreate"`
	Metadata None            `json:"metadata,omitempty"`
}

// GetAllCatalogSourceConfigsHandler lists model catalog source configurations.
//
//	@Summary		List all catalog sources
//	@Description	Lists default and user-defined model catalog source configurations, including labels, inclusion/exclusion patterns, and effective credential-configuration indicators. Not available in Kubeflow integration mode.
//	@Tags			ModelCatalogService
//	@ID				getCatalogSources
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace	query		string	true	"Catalog namespace"
//	@Success		200			{object}	ModelCatalogSettingsSourceConfigListEnvelope
//	@Failure		400			{object}	ErrorEnvelope
//	@Failure		403			{object}	ErrorEnvelope
//	@Failure		500			{object}	ErrorEnvelope
//	@Router			/api/v1/settings/model_catalog/source_configs [get]
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

// GetCatalogSourceConfigHandler returns a model catalog source configuration.
//
//	@Summary		Get a Model Catalog Source
//	@Description	Returns one model catalog source configuration. hasConfiguredApiKey indicates whether an effective Hugging Face API-key reference is configured; it does not prove the Secret exists or authentication succeeded. Not available in Kubeflow integration mode.
//	@Tags			ModelCatalogService
//	@ID				getCatalogSource
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace	query		string	true	"Catalog namespace"
//	@Param			source_id	path		string	true	"Catalog source ID"
//	@Success		200			{object}	ModelCatalogSettingsSourceConfigEnvelope
//	@Failure		400			{object}	ErrorEnvelope
//	@Failure		403			{object}	ErrorEnvelope
//	@Failure		404			{object}	ErrorEnvelope
//	@Failure		500			{object}	ErrorEnvelope
//	@Router			/api/v1/settings/model_catalog/source_configs/{source_id} [get]
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

// CreateCatalogSourceConfigHandler creates a model catalog source configuration.
//
//	@Summary		Create a Catalog source
//	@Description	Creates a yaml or hf catalog source. YAML sources require complete catalog content in yaml; Hugging Face sources require allowedOrganization and optionally accept apiKey. Labels and includedModels/excludedModels control discovery and inclusion. Not available in Kubeflow integration mode.
//	@Tags			ModelCatalogService
//	@ID				createCatalogSource
//	@Accept			json
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace	query		string								true	"Catalog namespace"
//	@Param			body		body		ModelCatalogSourceCreateEnvelope	true	"Catalog source configuration"
//	@Success		201			{object}	ModelCatalogSettingsSourceConfigEnvelope
//	@Header			201			{string}	Location	"URL of the created catalog source"
//	@Failure		400			{object}	ErrorEnvelope
//	@Failure		403			{object}	ErrorEnvelope
//	@Failure		409			{object}	ErrorEnvelope
//	@Failure		500			{object}	ErrorEnvelope
//	@Router			/api/v1/settings/model_catalog/source_configs [post]
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

	var envelope ModelCatalogSourceCreateEnvelope
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error decoding JSON: %v", err.Error()))
		return
	}

	payload, err := models.DecodeCatalogSourceCreate(envelope.Data)
	if err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("error decoding catalog source: %w", err))
		return
	}
	newCatalogSource, err := app.repositories.ModelCatalogSettingsRepository.CreateCatalogSourceConfig(ctx, client, namespace, payload)
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

// UpdateCatalogSourceConfigHandler updates a model catalog source configuration.
//
//	@Summary		Update a Model Catalog Source
//	@Description	Updates source configuration and model inclusion/exclusion patterns. Default-source and provider-type changes are restricted; concurrent updates may return 409. Not available in Kubeflow integration mode.
//	@Tags			ModelCatalogService
//	@ID				updateModelCatalogSource
//	@Accept			json
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace	query		string								true	"Catalog namespace"
//	@Param			source_id	path		string								true	"Catalog source ID"
//	@Param			body		body		ModelCatalogSourcePayloadEnvelope	true	"Catalog source fields to update"
//	@Success		200			{object}	ModelCatalogSettingsSourceConfigEnvelope
//	@Failure		400			{object}	ErrorEnvelope
//	@Failure		403			{object}	ErrorEnvelope
//	@Failure		404			{object}	ErrorEnvelope
//	@Failure		409			{object}	ErrorEnvelope
//	@Failure		500			{object}	ErrorEnvelope
//	@Router			/api/v1/settings/model_catalog/source_configs/{source_id} [patch]
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

// DeleteCatalogSourceConfigHandler deletes a model catalog source configuration.
//
//	@Summary		Delete a Model Catalog source
//	@Description	Returns the deleted source with HTTP 200. Not available in Kubeflow integration mode.
//	@Tags			ModelCatalogService
//	@ID				deleteModelCatalogSource
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace	query		string	true	"Catalog namespace"
//	@Param			source_id	path		string	true	"Catalog source ID"
//	@Success		200			{object}	ModelCatalogSettingsSourceConfigEnvelope
//	@Failure		400			{object}	ErrorEnvelope
//	@Failure		403			{object}	ErrorEnvelope
//	@Failure		404			{object}	ErrorEnvelope
//	@Failure		409			{object}	ErrorEnvelope
//	@Failure		500			{object}	ErrorEnvelope
//	@Router			/api/v1/settings/model_catalog/source_configs/{source_id} [delete]
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

// ClearCatalogSourceCredentialsHandler clears stored catalog credentials.
//
//	@Summary		Clear catalog source credentials
//	@Description	Removes stored Hugging Face credentials and returns an empty JSON object. Not available in Kubeflow integration mode.
//	@Tags			ModelCatalogService
//	@ID				clearCatalogSourceCredentials
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace	query		string	true	"Catalog namespace"
//	@Param			source_id	path		string	true	"Catalog source ID"
//	@Success		200			{object}	models.EmptyObject
//	@Failure		400			{object}	ErrorEnvelope
//	@Failure		403			{object}	ErrorEnvelope
//	@Failure		409			{object}	ErrorEnvelope
//	@Failure		500			{object}	ErrorEnvelope
//	@Router			/api/v1/settings/model_catalog/source_configs/{source_id}/credentials [delete]
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
