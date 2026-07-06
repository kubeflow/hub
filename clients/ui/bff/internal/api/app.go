package api

import (
	"context"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"

	k8s "github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	k8mocks "github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes/k8mocks"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	helper "github.com/kubeflow/hub/ui/bff/internal/helpers"

	"github.com/kubeflow/hub/ui/bff/internal/config"
	"github.com/kubeflow/hub/ui/bff/internal/repositories"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/hub/ui/bff/internal/mocks"
)

const (
	Version = "1.0.0"

	PathPrefix                    = "/model-registry"
	ApiPathPrefix                 = "/api/v1"
	ModelRegistryId               = "model_registry_id"
	RegisteredModelId             = "registered_model_id"
	ModelVersionId                = "model_version_id"
	ModelArtifactId               = "model_artifact_id"
	ArtifactId                    = "artifact_id"
	HealthCheckPath               = "/healthcheck"
	UserPath                      = ApiPathPrefix + "/user"
	ModelRegistryListPath         = ApiPathPrefix + "/model_registry"
	ModelRegistryPath             = ModelRegistryListPath + "/:" + ModelRegistryId
	NamespaceListPath             = ApiPathPrefix + "/namespaces"
	SettingsPath                  = ApiPathPrefix + "/settings"
	ModelRegistrySettingsListPath = SettingsPath + "/model_registry"
	ModelRegistrySettingsPath     = ModelRegistrySettingsListPath + "/:" + ModelRegistryId
	CertificatesPath              = SettingsPath + "/certificates"
	RoleBindingListPath           = SettingsPath + "/role_bindings"
	GroupsPath                    = SettingsPath + "/groups"
	SettingsNamespacePath         = SettingsPath + "/namespaces"
	RoleBindingPath               = RoleBindingListPath + "/:" + RoleBindingNameParam

	RegisteredModelListPath      = ModelRegistryPath + "/registered_models"
	RegisteredModelPath          = RegisteredModelListPath + "/:" + RegisteredModelId
	RegisteredModelVersionsPath  = RegisteredModelPath + "/versions"
	ModelVersionListPath         = ModelRegistryPath + "/model_versions"
	ModelVersionPath             = ModelVersionListPath + "/:" + ModelVersionId
	ModelVersionArtifactListPath = ModelVersionPath + "/artifacts"
	ModelArtifactListPath        = ModelRegistryPath + "/model_artifacts"
	ModelArtifactPath            = ModelArtifactListPath + "/:" + ModelArtifactId
	ArtifactListPath             = ModelRegistryPath + "/artifacts"
	ArtifactPath                 = ArtifactListPath + "/:" + ArtifactId

	// model catalog
	CatalogSourceId                     = "source_id"
	CatalogModelName                    = "model_name"
	CatalogPathPrefix                   = ApiPathPrefix + "/model_catalog"
	CatalogModelListPath                = CatalogPathPrefix + "/models"
	CatalogFilterOptionListPath         = CatalogPathPrefix + "/models/filter_options"
	CatalogSourceListPath               = CatalogPathPrefix + "/sources"
	CatalogLabelsPath                   = CatalogPathPrefix + "/labels"
	CatalogSourceModelCatchAllPath      = CatalogPathPrefix + "/sources/:" + CatalogSourceId + "/models/*" + CatalogModelName
	CatalogSourceModelArtifactsCatchAll = CatalogPathPrefix + "/sources/:" + CatalogSourceId + "/artifacts/*" + CatalogModelName
	CatalogModelPerformanceArtifacts    = CatalogPathPrefix + "/sources/:" + CatalogSourceId + "/performance_artifacts/*" + CatalogModelName
	CatalogModelSecurityArtifacts       = CatalogPathPrefix + "/sources/:" + CatalogSourceId + "/security_artifacts/*" + CatalogModelName

	ModelCatalogSettingsPathPrefix           = SettingsPath + "/model_catalog"
	ModelCatalogSettingsSourceConfigListPath = ModelCatalogSettingsPathPrefix + "/source_configs"
	ModelCatalogSettingsSourceConfigPath     = ModelCatalogSettingsSourceConfigListPath + "/:" + CatalogSourceId
	CatalogSourcePreviewPath                 = ModelCatalogSettingsPathPrefix + "/source_preview"

	// Model Transfer Jobs
	ModelTransferJobName       = "job_name"
	ModelTransferJobListPath   = ModelRegistryPath + "/model_transfer_jobs"
	ModelTransferJobPath       = ModelTransferJobListPath + "/:" + ModelTransferJobName
	ModelTransferJobEventsPath = ModelTransferJobPath + "/events"

	// Agent catalog
	AgentId                   = "agent_id"
	AgentCatalogPathPrefix    = ApiPathPrefix + "/agent_catalog"
	AgentListPath             = AgentCatalogPathPrefix + "/agents"
	AgentFilterOptionListPath = AgentCatalogPathPrefix + "/agents_filter_options"
	AgentPath                 = AgentListPath + "/:" + AgentId

	// MCP server catalog
	McpServerId                   = "server_id"
	McpServerCatalogPathPrefix    = ApiPathPrefix + "/mcp_catalog"
	McpServerListPath             = McpServerCatalogPathPrefix + "/mcp_servers"
	McpServerFilterOptionListPath = McpServerCatalogPathPrefix + "/mcp_servers_filter_options"
	McpServerPath                 = McpServerListPath + "/:" + McpServerId
	McpServersToolListPath        = McpServerPath + "/tools"

	// Swagger UI (interactive API docs)
	SwaggerPath    = ApiPathPrefix + "/swagger"
	SwaggerDocPath = SwaggerPath + "/doc.json"

	// MCP catalog settings
	McpCatalogSettingsPathPrefix           = SettingsPath + "/mcp_catalog"
	McpCatalogSettingsSourceConfigListPath = McpCatalogSettingsPathPrefix + "/source_configs"
	McpCatalogSettingsSourceConfigPath     = McpCatalogSettingsSourceConfigListPath + "/:" + CatalogSourceId
)

type App struct {
	config                  config.EnvConfig
	logger                  *slog.Logger
	kubernetesClientFactory k8s.KubernetesClientFactory
	repositories            *repositories.Repositories
	podNamespace            string
	//used only on mocked k8s client
	testEnv *envtest.Environment
	// rootCAs used for outbound TLS connections to Model Registry/Catalog
	rootCAs *x509.CertPool
}

func NewApp(cfg config.EnvConfig, logger *slog.Logger) (*App, error) {
	logger.Debug("Initializing app with config", slog.Any("config", cfg))
	var k8sFactory k8s.KubernetesClientFactory
	var err error
	// used only on mocked k8s client
	var testEnv *envtest.Environment
	var rootCAs *x509.CertPool

	// Initialize CA pool if bundle paths are provided
	if len(cfg.BundlePaths) > 0 {
		// Start with system certs if available
		if pool, err := x509.SystemCertPool(); err == nil {
			rootCAs = pool
		} else {
			rootCAs = x509.NewCertPool()
		}
		var loadedAny bool
		for _, p := range cfg.BundlePaths {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			// Read and append each PEM bundle; ignore errors per file, log at debug
			pemBytes, readErr := os.ReadFile(p)
			if readErr != nil {
				logger.Debug("CA bundle not readable, skipping", slog.String("path", p), slog.Any("error", readErr))
				continue
			}
			if ok := rootCAs.AppendCertsFromPEM(pemBytes); !ok {
				logger.Debug("No certs appended from PEM bundle", slog.String("path", p))
				continue
			}
			loadedAny = true
			logger.Info("Added CA bundle", slog.String("path", p))
		}
		if !loadedAny {
			// If none were loaded successfully, keep rootCAs nil to fall back to default transport behavior
			rootCAs = nil
			logger.Warn("No CA certificates loaded from bundle-paths; falling back to system defaults")
		}
	}

	if cfg.MockK8Client {
		//mock all k8s calls with 'env test'
		var clientset kubernetes.Interface
		ctx, cancel := context.WithCancel(context.Background())
		testEnv, clientset, err = k8mocks.SetupEnvTest(k8mocks.TestEnvInput{
			Logger: logger,
			Ctx:    ctx,
			Cancel: cancel,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to setup envtest: %w", err)
		}
		//create mocked kubernetes client factory
		k8sFactory, err = k8mocks.NewMockedKubernetesClientFactory(clientset, testEnv, cfg, logger)

	} else {
		//create kubernetes client factory
		k8sFactory, err = k8s.NewKubernetesClientFactory(cfg, logger)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	var mrClient repositories.ModelRegistryClientInterface

	if cfg.MockMRClient {
		//mock all model registry calls
		mrClient, err = mocks.NewModelRegistryClient(logger)
	} else {
		mrClient, err = repositories.NewModelRegistryClient(logger)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create ModelRegistry client: %w", err)
	}

	var modelCatalogClient repositories.ModelCatalogClientInterface

	if cfg.MockMRCatalogClient {
		//mock all model registry catalog calls
		modelCatalogClient, err = mocks.NewModelCatalogClientMock(logger)
	} else {
		modelCatalogClient, err = repositories.NewModelCatalogClient(logger)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create ModelRegistry Catalog client: %w", err)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create ModelCatalogSettings client: %w", err)
	}

	app := &App{
		config:                  cfg,
		logger:                  logger,
		kubernetesClientFactory: k8sFactory,
		repositories:            repositories.NewRepositories(mrClient, modelCatalogClient),
		podNamespace:            getPodNamespace(),
		testEnv:                 testEnv,
		rootCAs:                 rootCAs,
	}
	return app, nil
}

func (app *App) Shutdown() error {
	app.logger.Info("shutting down app...")
	if app.testEnv == nil {
		return nil
	}
	//shutdown the envtest control plane when we are in the mock mode.
	app.logger.Info("shutting env test...")
	return app.testEnv.Stop()
}

func getPodNamespace() string {
	ns, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(ns))
}

func (app *App) Routes() http.Handler {
	// Router for /api/v1/*
	apiRouter := httprouter.New()

	apiRouter.NotFound = http.HandlerFunc(app.notFoundResponse)
	apiRouter.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	// Model Registry HTTP client routes (requests that we forward to Model Registry API)
	// on those, we perform SAR or SSAR on Specific Service on a given namespace
	apiRouter.GET(RegisteredModelListPath, app.AttachNamespace(app.RequireAccessToMRService(app.AttachModelRegistryRESTClient(app.GetAllRegisteredModelsHandler))))
	apiRouter.GET(RegisteredModelPath, app.AttachNamespace(app.RequireAccessToMRService(app.AttachModelRegistryRESTClient(app.GetRegisteredModelHandler))))
	apiRouter.POST(RegisteredModelListPath, app.AttachNamespace(app.RequireAccessToMRService(app.AttachModelRegistryRESTClient(app.CreateRegisteredModelHandler))))
	apiRouter.PATCH(RegisteredModelPath, app.AttachNamespace(app.RequireAccessToMRService(app.AttachModelRegistryRESTClient(app.UpdateRegisteredModelHandler))))
	apiRouter.GET(RegisteredModelVersionsPath, app.AttachNamespace(app.RequireAccessToMRService(app.AttachModelRegistryRESTClient(app.GetAllModelVersionsForRegisteredModelHandler))))
	apiRouter.POST(RegisteredModelVersionsPath, app.AttachNamespace(app.RequireAccessToMRService(app.AttachModelRegistryRESTClient(app.CreateModelVersionForRegisteredModelHandler))))
	apiRouter.POST(ModelVersionListPath, app.AttachNamespace(app.RequireAccessToMRService(app.AttachModelRegistryRESTClient(app.CreateModelVersionHandler))))
	apiRouter.GET(ModelVersionListPath, app.AttachNamespace(app.RequireAccessToMRService(app.AttachModelRegistryRESTClient(app.GetAllModelVersionHandler))))
	apiRouter.GET(ModelVersionPath, app.AttachNamespace(app.RequireAccessToMRService(app.AttachModelRegistryRESTClient(app.GetModelVersionHandler))))
	apiRouter.PATCH(ModelVersionPath, app.AttachNamespace(app.RequireAccessToMRService(app.AttachModelRegistryRESTClient(app.UpdateModelVersionHandler))))
	apiRouter.GET(ArtifactListPath, app.AttachNamespace(app.RequireAccessToMRService(app.AttachModelRegistryRESTClient(app.GetAllArtifactsHandler))))
	apiRouter.GET(ArtifactPath, app.AttachNamespace(app.RequireAccessToMRService(app.AttachModelRegistryRESTClient(app.GetArtifactHandler))))
	apiRouter.POST(ArtifactListPath, app.AttachNamespace(app.RequireAccessToMRService(app.AttachModelRegistryRESTClient(app.CreateArtifactHandler))))
	apiRouter.GET(ModelVersionArtifactListPath, app.AttachNamespace(app.RequireAccessToMRService(app.AttachModelRegistryRESTClient(app.GetAllModelArtifactsByModelVersionHandler))))
	apiRouter.POST(ModelVersionArtifactListPath, app.AttachNamespace(app.RequireAccessToMRService(app.AttachModelRegistryRESTClient(app.CreateModelArtifactByModelVersionHandler))))
	apiRouter.PATCH(ModelRegistryPath, app.AttachNamespace(app.RequireAccessToMRService(app.AttachModelRegistryRESTClient(app.UpdateModelVersionHandler))))
	apiRouter.PATCH(ModelArtifactPath, app.AttachNamespace(app.RequireAccessToMRService(app.AttachModelRegistryRESTClient(app.UpdateModelArtifactHandler))))

	// Model Transfer Jobs
	apiRouter.GET(ModelTransferJobListPath, app.AttachNamespace(app.RequireAccessToMRService(app.handlerWithOverride(HandlerIDModelTransferJobList, func() httprouter.Handle { return app.GetAllModelTransferJobsHandler }))))
	apiRouter.GET(ModelTransferJobPath, app.AttachNamespace(app.RequireAccessToMRService(app.GetModelTransferJobHandler)))
	apiRouter.GET(ModelTransferJobEventsPath, app.AttachNamespace(app.RequireAccessToMRService(app.GetModelTransferJobEventsHandler)))
	apiRouter.POST(ModelTransferJobListPath, app.AttachNamespace(app.RequireAccessToMRService(app.CreateModelTransferJobHandler)))
	apiRouter.PATCH(ModelTransferJobPath, app.AttachNamespace(app.RequireAccessToMRService(app.UpdateModelTransferJobHandler)))
	apiRouter.DELETE(ModelTransferJobPath, app.AttachNamespace(app.RequireAccessToMRService(app.DeleteModelTransferJobHandler)))

	// Model catalog HTTP client routes (requests that we forward to Model Catalog API)
	apiRouter.GET(CatalogModelListPath, app.AttachNamespace(app.AttachModelCatalogRESTClient(app.GetAllCatalogModelsAcrossSourcesHandler)))
	apiRouter.GET(CatalogSourceListPath, app.AttachNamespace(app.AttachModelCatalogRESTClient(app.GetAllCatalogSourcesHandler)))
	apiRouter.GET(CatalogLabelsPath, app.AttachNamespace(app.AttachModelCatalogRESTClient(app.GetCatalogLabelsHandler)))
	apiRouter.GET(CatalogFilterOptionListPath, app.AttachNamespace(app.AttachModelCatalogRESTClient(app.GetCatalogFilterListHandler)))
	apiRouter.GET(CatalogSourceModelCatchAllPath, app.AttachNamespace(app.AttachModelCatalogRESTClient(app.GetCatalogSourceModelHandler)))
	apiRouter.GET(CatalogSourceModelArtifactsCatchAll, app.AttachNamespace(app.AttachModelCatalogRESTClient(app.GetCatalogSourceModelArtifactsHandler)))
	apiRouter.GET(CatalogModelPerformanceArtifacts, app.AttachNamespace(app.AttachModelCatalogRESTClient(app.GetCatalogModelPerformanceArtifactsHandler)))
	apiRouter.GET(CatalogModelSecurityArtifacts, app.AttachNamespace(app.AttachModelCatalogRESTClient(app.GetCatalogModelSecurityArtifactsHandler)))
	// Kubernetes routes
	apiRouter.GET(UserPath, app.UserHandler)
	apiRouter.POST(CheckNamespaceRegistryAccessPath, app.CheckNamespaceRegistryAccessHandler)
	apiRouter.GET(ModelRegistryListPath, app.AttachNamespace(app.RequireListServiceAccessInNamespace(app.GetAllModelRegistriesHandler)))

	// Swagger UI (interactive API docs) — only in dev mode
	if app.config.DevMode {
		apiRouter.GET(SwaggerPath+"/*filepath", app.GetSwaggerHandler)
	}

	// Enable these routes in all cases except Kubeflow integration mode
	// (Kubeflow integration mode is when DeploymentMode is kubeflow)
	isKubeflowIntegrationMode := app.config.DeploymentMode.IsKubeflowMode()
	if !isKubeflowIntegrationMode {
		// This namespace endpoint is used on standalone mode to simulate
		// Kubeflow Central Dashboard namespace selector dropdown on our standalone web app
		apiRouter.GET(NamespaceListPath, app.GetNamespacesHandler)

		// SettingsPath endpoints are used to manage the model registry settings and create new model registries
		// We are still discussing the best way to create model registries in the community
		// But in the meantime, those endpoints are STUBs endpoints used to unblock the frontend development
		apiRouter.GET(ModelRegistrySettingsListPath, app.AttachNamespace(app.GetAllModelRegistriesSettingsHandler))
		apiRouter.POST(ModelRegistrySettingsListPath, app.AttachNamespace(app.CreateModelRegistrySettingsHandler))
		apiRouter.GET(ModelRegistrySettingsPath, app.AttachNamespace(app.GetModelRegistrySettingsHandler))
		apiRouter.PATCH(ModelRegistrySettingsPath, app.AttachNamespace(app.UpdateModelRegistrySettingsHandler))
		apiRouter.DELETE(ModelRegistrySettingsPath, app.AttachNamespace(app.DeleteModelRegistrySettingsHandler))

		//SettingsPath: Certificate endpoints
		apiRouter.GET(CertificatesPath, app.AttachNamespace(app.GetCertificatesHandler))

		//SettingsPath: Role Binding endpoints
		apiRouter.GET(RoleBindingListPath, app.AttachNamespace(app.GetRoleBindingsHandler))
		apiRouter.POST(RoleBindingListPath, app.AttachNamespace(app.CreateRoleBindingHandler))
		apiRouter.PATCH(RoleBindingPath, app.AttachNamespace(app.PatchRoleBindingHandler))
		apiRouter.DELETE(RoleBindingPath, app.AttachNamespace(app.DeleteRoleBindingHandler))

		//SettingsPath Groups endpoints
		apiRouter.GET(GroupsPath, app.GetGroupsHandler)

		//SettingsPath Namespace endpoints
		//This namespace endpoint is used to get the namespaces for the current user inside the model registry settings
		apiRouter.GET(SettingsNamespacePath, app.GetNamespacesHandler)

		// Model catalog settings page
		apiRouter.GET(ModelCatalogSettingsSourceConfigListPath, app.AttachNamespace(app.GetAllCatalogSourceConfigsHandler))
		apiRouter.POST(ModelCatalogSettingsSourceConfigListPath, app.AttachNamespace(app.CreateCatalogSourceConfigHandler))
		apiRouter.GET(ModelCatalogSettingsSourceConfigPath, app.AttachNamespace(app.GetCatalogSourceConfigHandler))
		apiRouter.PATCH(ModelCatalogSettingsSourceConfigPath, app.AttachNamespace(app.UpdateCatalogSourceConfigHandler))
		apiRouter.DELETE(ModelCatalogSettingsSourceConfigPath, app.AttachNamespace(app.DeleteCatalogSourceConfigHandler))
		apiRouter.POST(CatalogSourcePreviewPath, app.AttachNamespace(app.AttachModelCatalogRESTClient(app.CreateCatalogSourcePreviewHandler)))

		// Agent catalog endpoints
		apiRouter.GET(AgentListPath, app.AttachNamespace(app.AttachModelCatalogRESTClient(app.GetAllAgentsHandler)))
		apiRouter.GET(AgentFilterOptionListPath, app.AttachNamespace(app.AttachModelCatalogRESTClient(app.GetAgentsFiltersHandler)))
		apiRouter.GET(AgentPath, app.AttachNamespace(app.AttachModelCatalogRESTClient(app.GetAgentHandler)))

		// MCP server catalog endpoints
		apiRouter.GET(McpServerListPath, app.AttachNamespace(app.AttachModelCatalogRESTClient(app.GetAllMcpServersHandler)))
		apiRouter.GET(McpServerFilterOptionListPath, app.AttachNamespace(app.AttachModelCatalogRESTClient(app.GetMcpServersFiltersHandler)))
		apiRouter.GET(McpServerPath, app.AttachNamespace(app.AttachModelCatalogRESTClient(app.GetMcpServerHandler)))
		apiRouter.GET(McpServersToolListPath, app.AttachNamespace(app.AttachModelCatalogRESTClient(app.GetMcpServersToolsHandler)))

		// MCP catalog settings page
		apiRouter.GET(McpCatalogSettingsSourceConfigListPath, app.AttachNamespace(app.GetAllMcpCatalogSourceConfigsHandler))
		apiRouter.POST(McpCatalogSettingsSourceConfigListPath, app.AttachNamespace(app.CreateMcpCatalogSourceConfigHandler))
		apiRouter.GET(McpCatalogSettingsSourceConfigPath, app.AttachNamespace(app.GetMcpCatalogSourceConfigHandler))
		apiRouter.PATCH(McpCatalogSettingsSourceConfigPath, app.AttachNamespace(app.UpdateMcpCatalogSourceConfigHandler))
		apiRouter.DELETE(McpCatalogSettingsSourceConfigPath, app.AttachNamespace(app.DeleteMcpCatalogSourceConfigHandler))
	}

	// App Router
	appMux := http.NewServeMux()

	// handler for api calls
	appMux.Handle(ApiPathPrefix+"/", apiRouter)
	appMux.Handle(PathPrefix+ApiPathPrefix+"/", http.StripPrefix(PathPrefix, apiRouter))

	// file server for the frontend file and SPA routes
	staticDir := http.Dir(app.config.StaticAssetsDir)
	fileServer := http.FileServer(staticDir)
	appMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctxLogger := helper.GetContextLoggerFromReq(r)
		// Check if the requested file exists
		if _, err := staticDir.Open(r.URL.Path); err == nil {
			ctxLogger.Debug("Serving static file", slog.String("path", r.URL.Path))
			// Serve the file if it exists
			fileServer.ServeHTTP(w, r)
			return
		}

		// Fallback to index.html for SPA routes
		ctxLogger.Debug("Static asset not found, serving index.html", slog.String("path", r.URL.Path))
		http.ServeFile(w, r, path.Join(app.config.StaticAssetsDir, "index.html"))
	})

	// Create a mux for the healthcheck endpoint
	healthcheckMux := http.NewServeMux()
	healthcheckRouter := httprouter.New()
	healthcheckRouter.GET(HealthCheckPath, app.HealthcheckHandler)
	healthcheckMux.Handle(HealthCheckPath, app.RecoverPanic(app.EnableTelemetry(healthcheckRouter)))

	// Combines the healthcheck endpoint with the rest of the routes
	combinedMux := http.NewServeMux()
	combinedMux.Handle(HealthCheckPath, healthcheckMux)
	combinedMux.Handle("/", app.RecoverPanic(app.EnableTelemetry(app.EnableCORS(app.InjectRequestIdentity(appMux)))))

	return combinedMux
}
