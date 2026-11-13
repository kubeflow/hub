package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/hub/ui/bff/internal/constants"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	"github.com/kubeflow/hub/ui/bff/internal/repositories"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

type (
	McpServerListEnvelope              Envelope[*models.McpServerList, None]
	McpServerFilterOptionsListEnvelope Envelope[*models.FilterOptionsList, None]
	McpServerEnvelope                  Envelope[*models.McpServer, None]
	McpServerToolsListEnvelope         Envelope[*models.McpToolList, None]
)

func (app *App) GetAllMcpServersHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)

	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	mcpServers, err := app.repositories.ModelCatalogClient.GetAllMcpServers(client, r.URL.Query())
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	mcpServerList := McpServerListEnvelope{
		Data: mcpServers,
	}

	err = app.WriteJSON(w, http.StatusOK, mcpServerList, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) GetMcpServersFiltersHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)

	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	mcpServerFilterOptions, err := app.repositories.ModelCatalogClient.GetMcpServersFilter(client)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	mcpServerFilterOptionsList := McpServerFilterOptionsListEnvelope{
		Data: mcpServerFilterOptions,
	}

	err = app.WriteJSON(w, http.StatusOK, mcpServerFilterOptionsList, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) GetMcpServerHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)

	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	serverId := ps.ByName(McpServerId)

	if serverId == "" {
		app.badRequestResponse(w, r, fmt.Errorf("server_id is required"))
		return
	}

	server, err := app.repositories.ModelCatalogClient.GetMcpServer(client, serverId, r.URL.Query())
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	mcpServer := McpServerEnvelope{
		Data: server,
	}

	err = app.WriteJSON(w, http.StatusOK, mcpServer, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) GetMcpServersToolsHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)

	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	serverId := ps.ByName(McpServerId)

	if serverId == "" {
		app.badRequestResponse(w, r, fmt.Errorf("server_id is required"))
		return
	}

	mcpServerTools, err := app.repositories.ModelCatalogClient.GetMcpServersTools(client, serverId)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	mcpServerToolList := McpServerToolsListEnvelope{
		Data: mcpServerTools,
	}

	err = app.WriteJSON(w, http.StatusOK, mcpServerToolList, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) GetMcpServerLogoHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)

	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	serverId := ps.ByName(McpServerId)

	if serverId == "" {
		app.badRequestResponse(w, r, fmt.Errorf("server_id is required"))
		return
	}

	resp, err := app.repositories.ModelCatalogClient.GetMcpServerLogo(client, serverId)
	// GetMcpServerLogo uses GETRaw, which only returns an error for transport-level
	// failures (never for upstream error statuses), so any non-nil err is a server error.
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Faithfully proxy the catalog logo response. For plain external-URL logos the
	// catalog replies with a redirect, which we pass through so the browser follows it;
	// for inline data-URI logos it replies with the decoded image bytes.
	if resp.StatusCode >= http.StatusMultipleChoices && resp.StatusCode < http.StatusBadRequest {
		if location := resp.Header.Get("Location"); location != "" {
			w.Header().Set("Location", location)
		}
		w.WriteHeader(resp.StatusCode)
		return
	}

	// GETRaw passes upstream error statuses through untouched, so translate any 4xx/5xx
	// into the structured JSON error the rest of the API returns instead of forwarding
	// the catalog's plaintext body (e.g. "404 page not found") verbatim.
	if resp.StatusCode >= http.StatusBadRequest {
		app.errorResponse(w, r, httpErrorFromRaw(resp))
		return
	}

	// Forward an allowlist of upstream headers. Besides Content-Type and Cache-Control,
	// this MUST include the catalog's protective headers (X-Content-Type-Options: nosniff
	// and the SVG Content-Security-Policy): since we re-serve these bytes under the
	// dashboard's own origin, dropping them would re-open the stored-XSS vector on
	// SVG logos that those headers exist to prevent.
	for _, h := range []string{
		"Content-Type",
		"Cache-Control",
		"X-Content-Type-Options",
		"Content-Security-Policy",
		"Content-Disposition",
	} {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}

	w.WriteHeader(resp.StatusCode)
	if _, err := w.Write(resp.Body); err != nil {
		app.logger.Error("failed to write mcp server logo response", "error", err)
	}
}

func (app *App) ClearMcpCatalogSourceStatusHandler(
	w http.ResponseWriter,
	r *http.Request,
	ps httprouter.Params,
) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("namespace is required"))
		return
	}

	sourceID := ps.ByName(CatalogSourceId)
	if sourceID == "" || sourceID == "." || sourceID == ".." ||
		strings.ContainsAny(sourceID, `/\`) {
		app.badRequestResponse(w, r, fmt.Errorf("invalid source ID"))
		return
	}

	identity, ok := ctx.Value(constants.RequestIdentityKey).(*kubernetes.RequestIdentity)
	if !ok || identity == nil {
		app.errorResponse(w, r, &httpclient.HTTPError{
			StatusCode: http.StatusUnauthorized,
			ErrorResponse: httpclient.ErrorResponse{
				Code:    "401",
				Message: "Authentication is required",
			},
		})
		return
	}

	kubeClient, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	allowed, err := kubeClient.CanUpdateMcpCatalogSources(
		ctx,
		identity,
		namespace,
	)
	if err != nil {
		if apierrors.IsForbidden(err) {
			app.forbiddenResponse(w, r, "Cannot verify MCP source update permission")
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	if !allowed {
		app.forbiddenResponse(w, r, "MCP source update permission is required")
		return
	}

	// Prevent calls on arbitrary model/agent source IDs through this route.
	_, err = app.repositories.McpCatalogSettingsRepository.
		GetMcpCatalogSourceConfig(ctx, kubeClient, namespace, sourceID)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrMcpCatalogSourceNotFound):
			app.notFoundResponse(w, r)
		case apierrors.IsForbidden(err):
			app.forbiddenResponse(w, r, "Cannot read MCP source configuration")
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	catalogClient, ok := ctx.Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)
	if !ok || catalogClient == nil {
		app.serverErrorResponse(w, r, fmt.Errorf("catalog REST client not found"))
		return
	}

	err = app.repositories.ModelCatalogClient.
		ClearCatalogSourceStatus(ctx, catalogClient, sourceID)
	if err != nil {
		var upstreamError *httpclient.HTTPError
		if errors.As(err, &upstreamError) &&
			upstreamError.StatusCode >= 400 &&
			upstreamError.StatusCode <= 599 {
			app.errorResponse(w, r, upstreamError)
			return
		}

		app.LogError(r, err)
		app.errorResponse(w, r, &httpclient.HTTPError{
			StatusCode: http.StatusBadGateway,
			ErrorResponse: httpclient.ErrorResponse{
				Code:    "502",
				Message: "Could not clear catalog source status",
			},
		})
		return
	}

	if err := app.WriteJSON(
		w,
		http.StatusOK,
		map[string]any{"data": nil},
		nil,
	); err != nil {
		app.LogError(r, err)
	}
}
