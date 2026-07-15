package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/kubeflow/hub/ui/bff/internal/config"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	"github.com/kubeflow/hub/ui/bff/internal/repositories"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/hub/ui/bff/internal/constants"
	helper "github.com/kubeflow/hub/ui/bff/internal/helpers"
	"github.com/rs/cors"
)

func (app *App) RecoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
				app.logger.Error("Recovered from panic", slog.String("stack_trace", string(debug.Stack())))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *App) InjectRequestIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//skip use headers check if we are not on /api/v1 (i.e. we are on /healthcheck and / (static fe files) )
		if !strings.HasPrefix(r.URL.Path, ApiPathPrefix) && !strings.HasPrefix(r.URL.Path, PathPrefix+ApiPathPrefix) {
			next.ServeHTTP(w, r)
			return
		}

		identity, error := app.kubernetesClientFactory.ExtractRequestIdentity(r.Header)
		if error != nil {
			app.badRequestResponse(w, r, error)
			return
		}

		ctx := context.WithValue(r.Context(), constants.RequestIdentityKey, identity)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *App) EnableCORS(next http.Handler) http.Handler {
	if len(app.config.AllowedOrigins) == 0 {
		// CORS is disabled, this middleware becomes a noop.
		return next
	}

	c := cors.New(cors.Options{
		AllowedOrigins:     app.config.AllowedOrigins,
		AllowCredentials:   true,
		AllowedMethods:     []string{"GET", "PUT", "POST", "PATCH", "DELETE"},
		AllowedHeaders:     []string{constants.KubeflowUserIDHeader, constants.KubeflowUserGroupsIdHeader},
		Debug:              app.config.LogLevel == slog.LevelDebug,
		OptionsPassthrough: false,
	})

	return c.Handler(next)
}

func (app *App) EnableTelemetry(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Adds a unique id to the context to allow tracing of requests
		traceId := uuid.NewString()
		ctx := context.WithValue(r.Context(), constants.TraceIdKey, traceId)

		// logger will only be nil in tests.
		if app.logger != nil {
			traceLogger := app.logger.With(slog.String("trace_id", traceId))
			ctx = context.WithValue(ctx, constants.TraceLoggerKey, traceLogger)

			traceLogger.Debug("Incoming HTTP request", slog.Any("request", helper.RequestLogValuer{Request: r}))
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *App) AttachModelCatalogRESTClient(next func(http.ResponseWriter, *http.Request, httprouter.Params)) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		namespace, ok := r.Context().Value(constants.NamespaceHeaderParameterKey).(string)
		if !ok || namespace == "" {
			app.badRequestResponse(w, r, fmt.Errorf("missing namespace in the context"))
			return
		}

		client, err := app.kubernetesClientFactory.GetClient(r.Context())
		if err != nil {
			app.serverErrorResponse(w, r, fmt.Errorf("failed to get Kubernetes client: %w", err))
			return
		}

		modelCatalog, err := app.repositories.ModelCatalog.GetModelCatalogWithMode(r.Context(), client, namespace, app.config.DeploymentMode.IsFederatedMode())
		if err != nil {
			app.notFoundResponse(w, r)
			return
		}
		apiPath := repositories.ModelCatalogAPIPath
		if strings.HasPrefix(r.URL.Path, McpServerCatalogPathPrefix) {
			apiPath = repositories.McpCatalogAPIPath
		} else if strings.HasPrefix(r.URL.Path, AgentCatalogPathPrefix) {
			apiPath = repositories.AgentCatalogAPIPath
		}

		modelCatalogBaseURL := modelCatalog.ServerAddress
		if apiPath != repositories.ModelCatalogAPIPath {
			modelCatalogBaseURL = strings.Replace(modelCatalogBaseURL, repositories.ModelCatalogAPIPath, apiPath, 1)
		}

		// If we are in dev mode, we need to resolve the server address to the local host
		// to allow the client to connect to the model registry via port forwarded from the cluster to the local machine.
		// If you are in federated mode, we do not want to override the server address.
		if app.config.DevMode && !app.config.DeploymentMode.IsFederatedMode() {
			modelCatalogBaseURL = app.repositories.ModelCatalog.ResolveServerAddress("localhost", int32(app.config.DevModeCatalogPort), modelCatalog.IsHTTPS, "", app.config.DeploymentMode.IsFederatedMode(), apiPath)
		}

		// Set up a child logger for the rest client that automatically adds the request id to all statements for
		// tracing.
		restClientLogger := app.logger
		traceId, ok := r.Context().Value(constants.TraceIdKey).(string)
		if app.logger != nil {
			if ok {
				restClientLogger = app.logger.With(slog.String("trace_id", traceId))
			} else {
				app.logger.Warn("Failed to set trace_id for tracing")
			}
		}

		// Prepare headers for the REST client
		headers := http.Header{}

		// If using user token authentication, extract and forward the authorization header
		if app.config.AuthMethod == config.AuthMethodUser {
			identity, ok := r.Context().Value(constants.RequestIdentityKey).(*kubernetes.RequestIdentity)
			if ok && identity != nil && identity.Token != "" {
				// Always send as "Authorization: Bearer <token>" regardless of incoming header format
				// The identity.Token already has any prefix removed by ExtractRequestIdentity
				authHeaderValue := "Bearer " + identity.Token
				headers.Set("Authorization", authHeaderValue)
			}
		}

		restHttpClient, err := httpclient.NewHTTPClient(restClientLogger, modelCatalogBaseURL, headers, app.config.InsecureSkipVerify, app.rootCAs)
		if err != nil {
			app.serverErrorResponse(w, r, fmt.Errorf("failed to create HTTP client: %v", err))
			return
		}
		ctx := context.WithValue(r.Context(), constants.ModelCatalogHttpClientKey, restHttpClient)
		next(w, r.WithContext(ctx), ps)
	}
}

func (app *App) AttachModelRegistryRESTClient(next func(http.ResponseWriter, *http.Request, httprouter.Params)) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		modelRegistryID := ps.ByName(ModelRegistryId)

		namespace, ok := r.Context().Value(constants.NamespaceHeaderParameterKey).(string)
		if !ok || namespace == "" {
			app.badRequestResponse(w, r, fmt.Errorf("missing namespace in the context"))
		}

		client, err := app.kubernetesClientFactory.GetClient(r.Context())
		if err != nil {
			app.serverErrorResponse(w, r, fmt.Errorf("failed to get Kubernetes client: %w", err))
			return
		}

		modelRegistry, err := app.repositories.ModelRegistry.GetModelRegistryWithMode(r.Context(), client, namespace, modelRegistryID, app.config.DeploymentMode.IsFederatedMode())
		if err != nil {
			app.notFoundResponse(w, r)
			return
		}
		modelRegistryBaseURL := modelRegistry.ServerAddress

		// If we are in dev mode, we need to resolve the server address to the local host
		// to allow the client to connect to the model registry via port forwarded from the cluster to the local machine.
		// If you are in federated mode, we do not want to override the server address.
		if app.config.DevMode && !app.config.DeploymentMode.IsFederatedMode() {
			modelRegistryBaseURL = app.repositories.ModelRegistry.ResolveServerAddress("localhost", int32(app.config.DevModeModelRegistryPort), modelRegistry.IsHTTPS, "", app.config.DeploymentMode.IsFederatedMode())
		}

		// Set up a child logger for the rest client that automatically adds the request id to all statements for
		// tracing.
		restClientLogger := app.logger
		traceId, ok := r.Context().Value(constants.TraceIdKey).(string)
		if app.logger != nil {
			if ok {
				restClientLogger = app.logger.With(slog.String("trace_id", traceId))
			} else {
				app.logger.Warn("Failed to set trace_id for tracing")
			}
		}

		// Prepare headers for the REST client
		headers := http.Header{}

		// If using user token authentication, extract and forward the authorization header
		if app.config.AuthMethod == config.AuthMethodUser {
			identity, ok := r.Context().Value(constants.RequestIdentityKey).(*kubernetes.RequestIdentity)
			if ok && identity != nil && identity.Token != "" {
				// Always send as "Authorization: Bearer <token>" regardless of incoming header format
				// The identity.Token already has any prefix removed by ExtractRequestIdentity
				authHeaderValue := "Bearer " + identity.Token
				headers.Set("Authorization", authHeaderValue)
			}
		}

		restHttpClient, err := httpclient.NewHTTPClient(restClientLogger, modelRegistryBaseURL, headers, app.config.InsecureSkipVerify, app.rootCAs)
		if err != nil {
			app.serverErrorResponse(w, r, fmt.Errorf("failed to create HTTP client: %v", err))
			return
		}
		ctx := context.WithValue(r.Context(), constants.ModelRegistryHttpClientKey, restHttpClient)
		next(w, r.WithContext(ctx), ps)
	}
}

func (app *App) AttachNamespace(next func(http.ResponseWriter, *http.Request, httprouter.Params)) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		namespace := r.URL.Query().Get(string(constants.NamespaceHeaderParameterKey))
		if namespace == "" {
			app.badRequestResponse(w, r, fmt.Errorf("missing required query parameter: %s", constants.NamespaceHeaderParameterKey))
			return
		}

		ctx := context.WithValue(r.Context(), constants.NamespaceHeaderParameterKey, namespace)
		r = r.WithContext(ctx)

		next(w, r, ps)
	}
}

func (app *App) RequireListServiceAccessInNamespace(next func(http.ResponseWriter, *http.Request, httprouter.Params)) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		ctx := r.Context()
		identity, ok := ctx.Value(constants.RequestIdentityKey).(*kubernetes.RequestIdentity)
		if !ok || identity == nil {
			app.badRequestResponse(w, r, fmt.Errorf("missing RequestIdentity in context"))
			return
		}

		if err := app.kubernetesClientFactory.ValidateRequestIdentity(identity); err != nil {
			app.badRequestResponse(w, r, err)
			return
		}

		namespace, ok := r.Context().Value(constants.NamespaceHeaderParameterKey).(string)
		if !ok || namespace == "" {
			app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
			return
		}

		client, err := app.kubernetesClientFactory.GetClient(ctx)
		if err != nil {
			app.serverErrorResponse(w, r, fmt.Errorf("failed to get Kubernetes client: %w", err))
			return
		}

		// Validate if user can list services in namespace
		allowList, err := client.CanListServicesInNamespace(ctx, identity, namespace)
		if err != nil {
			app.forbiddenResponse(w, r, fmt.Sprintf("SAR or SelfSAR failed for namespace %s: %v", namespace, err))
			return
		}

		// If user can't list all services, validate if user can access specific services
		var allowedServiceNames []string
		if !allowList {
			// Use SelfSubjectRulesReview to get specific service names the user can access
			allowedServiceNames, err = client.GetSelfSubjectRulesReview(ctx, identity, namespace)
			if err != nil {
				app.forbiddenResponse(w, r, fmt.Sprintf("Failed to get rules review for namespace %s: %v", namespace, err))
				return
			}

			// If user has no specific service access and can't list all, deny access
			if len(allowedServiceNames) == 0 {
				app.forbiddenResponse(w, r, fmt.Sprintf("No service access granted in namespace %s", namespace))
				return
			}
		}

		//Add authorization context for passing to downstream handler GetAllModelRegistriesHandler
		authCtx := &models.ServiceAuthorizationContext{
			AllowList:           allowList,
			AllowedServiceNames: allowedServiceNames,
			Namespace:           namespace,
		}

		ctx = context.WithValue(ctx, constants.ServiceAuthorizationContextKey, authCtx)
		next(w, r.WithContext(ctx), ps)
	}
}

func (app *App) RequireAccessToMRService(next func(http.ResponseWriter, *http.Request, httprouter.Params)) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		ctx := r.Context()
		identity, ok := ctx.Value(constants.RequestIdentityKey).(*kubernetes.RequestIdentity)
		if !ok || identity == nil {
			app.badRequestResponse(w, r, fmt.Errorf("missing RequestIdentity in context"))
			return
		}

		if err := app.kubernetesClientFactory.ValidateRequestIdentity(identity); err != nil {
			app.badRequestResponse(w, r, err)
			return
		}

		namespace, ok := r.Context().Value(constants.NamespaceHeaderParameterKey).(string)
		if !ok || namespace == "" {
			app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
			return
		}

		serviceName := ps.ByName(ModelRegistryId)
		if !ok || serviceName == "" {
			app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
			return
		}

		client, err := app.kubernetesClientFactory.GetClient(ctx)
		if err != nil {
			app.serverErrorResponse(w, r, fmt.Errorf("failed to get Kubernetes client: %w", err))
			return
		}

		allowed, err := client.CanAccessServiceInNamespace(r.Context(), identity, namespace, serviceName)

		if err != nil {
			app.forbiddenResponse(w, r, fmt.Sprintf("SAR or SelfSAR AccessReview failed for service %s in namespace %s: %v", serviceName, namespace, err))
			return
		}
		if !allowed {
			app.forbiddenResponse(w, r, fmt.Sprintf("SAR or SelfSAR AccessReview denied access to service %s in namespace %s", serviceName, namespace))
			return
		}

		next(w, r, ps)
	}
}
