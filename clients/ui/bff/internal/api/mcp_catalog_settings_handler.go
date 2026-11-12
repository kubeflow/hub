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

type McpCatalogSettingsSourceConfigEnvelope Envelope[*models.McpCatalogSourceConfig, None]
type McpCatalogSettingsSourceConfigListEnvelope Envelope[*models.McpCatalogSourceConfigList, None]
type McpCatalogSourcePayloadEnvelope Envelope[*models.McpCatalogSourceConfigPayload, None]

// GetAllMcpCatalogSourceConfigsHandler lists MCP catalog source configurations.
//
//	@Summary		List all MCP catalog sources
//	@Description	Not available in Kubeflow integration mode.
//	@Tags			MCPCatalogService
//	@ID				getMcpCatalogSources
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace	query		string	true	"Catalog namespace"
//	@Success		200			{object}	McpCatalogSettingsSourceConfigListEnvelope
//	@Failure		400			{object}	ErrorEnvelope
//	@Failure		403			{object}	ErrorEnvelope
//	@Failure		500			{object}	ErrorEnvelope
//	@Router			/api/v1/settings/mcp_catalog/source_configs [get]
func (app *App) GetAllMcpCatalogSourceConfigsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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

	configs, err := app.repositories.McpCatalogSettingsRepository.GetAllMcpCatalogSourceConfigs(ctx, client, namespace)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	envelope := McpCatalogSettingsSourceConfigListEnvelope{
		Data: configs,
	}

	if err = app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// GetMcpCatalogSourceConfigHandler returns an MCP catalog source configuration.
//
//	@Summary		Get an MCP Catalog Source
//	@Description	Not available in Kubeflow integration mode.
//	@Tags			MCPCatalogService
//	@ID				getMcpCatalogSource
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace	query		string	true	"Catalog namespace"
//	@Param			source_id	path		string	true	"Catalog source ID"
//	@Success		200			{object}	McpCatalogSettingsSourceConfigEnvelope
//	@Failure		400			{object}	ErrorEnvelope
//	@Failure		403			{object}	ErrorEnvelope
//	@Failure		404			{object}	ErrorEnvelope
//	@Failure		500			{object}	ErrorEnvelope
//	@Router			/api/v1/settings/mcp_catalog/source_configs/{source_id} [get]
func (app *App) GetMcpCatalogSourceConfigHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()

	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
		return
	}

	sourceID := ps.ByName(CatalogSourceId)

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, errors.New("catalog client not found"))
		return
	}

	config, err := app.repositories.McpCatalogSettingsRepository.GetMcpCatalogSourceConfig(ctx, client, namespace, sourceID)
	if err != nil {
		if errors.Is(err, repositories.ErrMcpCatalogSourceNotFound) {
			app.notFoundResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	envelope := McpCatalogSettingsSourceConfigEnvelope{
		Data: config,
	}

	if err = app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// CreateMcpCatalogSourceConfigHandler creates an MCP catalog source configuration.
//
//	@Summary		Create an MCP Catalog source
//	@Description	Not available in Kubeflow integration mode.
//	@Tags			MCPCatalogService
//	@ID				createMcpCatalogSource
//	@Accept			json
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace	query		string							true	"Catalog namespace"
//	@Param			body		body		McpCatalogSourcePayloadEnvelope	true	"MCP catalog source configuration"
//	@Success		201			{object}	McpCatalogSettingsSourceConfigEnvelope
//	@Header			201			{string}	Location	"URL of the created MCP catalog source"
//	@Failure		400			{object}	ErrorEnvelope
//	@Failure		403			{object}	ErrorEnvelope
//	@Failure		409			{object}	ErrorEnvelope
//	@Failure		500			{object}	ErrorEnvelope
//	@Router			/api/v1/settings/mcp_catalog/source_configs [post]
func (app *App) CreateMcpCatalogSourceConfigHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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

	var envelope McpCatalogSourcePayloadEnvelope
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("error decoding JSON: %w", err))
		return
	}

	if envelope.Data == nil {
		app.badRequestResponse(w, r, fmt.Errorf("missing required field: data"))
		return
	}

	created, err := app.repositories.McpCatalogSettingsRepository.CreateMcpCatalogSourceConfig(ctx, client, namespace, *envelope.Data)
	if err != nil {
		if errors.Is(err, repositories.ErrMcpCatalogSourceAlreadyExist) ||
			errors.Is(err, repositories.ErrMcpCatalogSourceIdRequired) ||
			errors.Is(err, repositories.ErrMcpCatalogValidationFailed) {
			app.badRequestResponse(w, r, err)
		} else if errors.Is(err, repositories.ErrMcpCatalogSourceConflict) {
			app.conflictResponse(w, r, err.Error())
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	result := McpCatalogSettingsSourceConfigEnvelope{
		Data: created,
	}

	w.Header().Set("Location", r.URL.JoinPath(result.Data.Id).String())
	if err = app.WriteJSON(w, http.StatusCreated, result, nil); err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error writing JSON"))
	}
}

// UpdateMcpCatalogSourceConfigHandler updates an MCP catalog source configuration.
//
//	@Summary		Update an MCP Catalog Source
//	@Description	Not available in Kubeflow integration mode.
//	@Tags			MCPCatalogService
//	@ID				updateMcpCatalogSource
//	@Accept			json
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace	query		string							true	"Catalog namespace"
//	@Param			source_id	path		string							true	"Catalog source ID"
//	@Param			body		body		McpCatalogSourcePayloadEnvelope	true	"MCP catalog source fields to update"
//	@Success		200			{object}	McpCatalogSettingsSourceConfigEnvelope
//	@Failure		400			{object}	ErrorEnvelope
//	@Failure		403			{object}	ErrorEnvelope
//	@Failure		404			{object}	ErrorEnvelope
//	@Failure		409			{object}	ErrorEnvelope
//	@Failure		500			{object}	ErrorEnvelope
//	@Router			/api/v1/settings/mcp_catalog/source_configs/{source_id} [patch]
func (app *App) UpdateMcpCatalogSourceConfigHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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

	var envelope McpCatalogSourcePayloadEnvelope
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("error decoding JSON: %w", err))
		return
	}

	if envelope.Data == nil {
		app.badRequestResponse(w, r, fmt.Errorf("missing required field: data"))
		return
	}

	sourceID := ps.ByName(CatalogSourceId)
	if sourceID == "" {
		sourceID = envelope.Data.Id
	}

	updated, err := app.repositories.McpCatalogSettingsRepository.UpdateMcpCatalogSourceConfig(ctx, client, namespace, sourceID, *envelope.Data)
	if err != nil {
		if errors.Is(err, repositories.ErrMcpCatalogSourceNotFound) {
			app.notFoundResponse(w, r)
		} else if errors.Is(err, repositories.ErrMcpCatalogCannotChangeDefault) ||
			errors.Is(err, repositories.ErrMcpCatalogCannotChangeType) {
			app.forbiddenResponse(w, r, err.Error())
		} else if errors.Is(err, repositories.ErrMcpCatalogSourceConflict) {
			app.conflictResponse(w, r, err.Error())
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	result := McpCatalogSettingsSourceConfigEnvelope{
		Data: updated,
	}

	if err = app.WriteJSON(w, http.StatusOK, result, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// DeleteMcpCatalogSourceConfigHandler deletes an MCP catalog source configuration.
//
//	@Summary		Delete an MCP Catalog source
//	@Description	Returns the deleted source with HTTP 200. Not available in Kubeflow integration mode.
//	@Tags			MCPCatalogService
//	@ID				deleteMcpCatalogSource
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace	query		string	true	"Catalog namespace"
//	@Param			source_id	path		string	true	"Catalog source ID"
//	@Success		200			{object}	McpCatalogSettingsSourceConfigEnvelope
//	@Failure		400			{object}	ErrorEnvelope
//	@Failure		403			{object}	ErrorEnvelope
//	@Failure		404			{object}	ErrorEnvelope
//	@Failure		409			{object}	ErrorEnvelope
//	@Failure		500			{object}	ErrorEnvelope
//	@Router			/api/v1/settings/mcp_catalog/source_configs/{source_id} [delete]
func (app *App) DeleteMcpCatalogSourceConfigHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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

	sourceID := ps.ByName(CatalogSourceId)

	deletedConfig, err := app.repositories.McpCatalogSettingsRepository.DeleteMcpCatalogSourceConfig(ctx, client, namespace, sourceID)
	if err != nil {
		if errors.Is(err, repositories.ErrMcpCatalogCannotDeleteDefault) {
			app.forbiddenResponse(w, r, err.Error())
		} else if errors.Is(err, repositories.ErrMcpCatalogSourceNotFound) {
			app.notFoundResponse(w, r)
		} else if errors.Is(err, repositories.ErrMcpCatalogSourceConflict) {
			app.conflictResponse(w, r, err.Error())
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	result := McpCatalogSettingsSourceConfigEnvelope{
		Data: deletedConfig,
	}

	if err = app.WriteJSON(w, http.StatusOK, result, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
