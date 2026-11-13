package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/hub/ui/bff/internal/constants"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	"github.com/kubeflow/hub/ui/bff/internal/repositories"
)

type McpCatalogSettingsSourceConfigEnvelope Envelope[*models.McpCatalogSourceConfig, None]
type McpCatalogSettingsSourceConfigListEnvelope Envelope[*models.McpCatalogSourceConfigList, None]
type McpCatalogSourcePayloadEnvelope Envelope[*models.McpCatalogSourceConfigPayload, None]

func (app *App) clearMcpCatalogSourceStatus(ctx context.Context, sourceID string) {
	client, ok := ctx.Value(constants.ModelCatalogStatusHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok {
		app.logger.Warn("MCP source saved but status client is unavailable", slog.String("source_id", sourceID))
		return
	}

	if err := app.repositories.ModelCatalogClient.ClearSourceStatus(client, sourceID); err != nil {
		app.logger.Warn("MCP source saved but source status could not be cleared",
			slog.String("source_id", sourceID),
			slog.Any("error", err),
		)
	}
}

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

	previous, err := app.repositories.McpCatalogSettingsRepository.GetMcpCatalogSourceConfig(ctx, client, namespace, sourceID)
	if err != nil {
		if errors.Is(err, repositories.ErrMcpCatalogSourceNotFound) {
			app.notFoundResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
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
	if repositories.ShouldClearMcpCatalogSourceStatusOnUpdate(previous, updated) {
		app.clearMcpCatalogSourceStatus(ctx, updated.Id)
	}

	if err = app.WriteJSON(w, http.StatusOK, result, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

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
