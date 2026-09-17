package repositories

import (
	"context"
	"fmt"

	"github.com/kubeflow/hub/ui/bff/internal/constants"
	helper "github.com/kubeflow/hub/ui/bff/internal/helpers"
	k8s "github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/models"
)

type ModelRegistryRepository struct{}

func NewModelRegistryRepository() *ModelRegistryRepository {
	return &ModelRegistryRepository{}
}

func (m *ModelRegistryRepository) GetAllModelRegistries(sessionCtx context.Context, client k8s.KubernetesClientInterface, namespace string) ([]models.ModelRegistryModel, error) {
	// Default to non-federated mode for backward compatibility
	return m.GetAllModelRegistriesWithMode(sessionCtx, client, namespace, false)
}

// GetAllModelRegistriesWithMode fetches all model registries with support for federated mode
func (m *ModelRegistryRepository) GetAllModelRegistriesWithMode(sessionCtx context.Context, client k8s.KubernetesClientInterface, namespace string, isFederatedMode bool) ([]models.ModelRegistryModel, error) {
	logger := helper.GetContextLogger(sessionCtx)
	logger.Debug("GetAllModelRegistriesWithMode called", "namespace", namespace, "isFederatedMode", isFederatedMode)

	var resources []k8s.ServiceDetails
	var err error

	// Check if we have authorization context from the middleware
	if authCtx, ok := sessionCtx.Value(constants.ServiceAuthorizationContextKey).(*models.ServiceAuthorizationContext); ok {
		if authCtx.AllowList {
			logger.Debug("User can get list services ")
			resources, err = client.GetServiceDetails(sessionCtx, namespace)
		} else {
			logger.Debug("User has limited access - we need use Rule base access",
				"serviceCount", len(authCtx.AllowedServiceNames),
				"services", authCtx.AllowedServiceNames)
			resources, err = m.getSpecificServiceDetails(sessionCtx, client, namespace, authCtx.AllowedServiceNames)
		}
	} else {
		logger.Warn("No authorization context found - using fallback behavior")
		resources, err = client.GetServiceDetails(sessionCtx, namespace)
	}

	if err != nil {
		logger.Error("Error fetching service details", "error", err, "namespace", namespace)
		return nil, fmt.Errorf("error fetching model registries: %w", err)
	}

	// Per-registry GetServiceEndpoints (no list): non-admin users typically lack list Endpoints permission. On fetch error we assume available to avoid new permission requirements upstream.
	var registries = []models.ModelRegistryModel{}
	for _, s := range resources {
		serverAddress := m.ResolveServerAddress(s.ClusterIP, s.HTTPPort, s.IsHTTPS, s.ExternalAddressRest, isFederatedMode)
		isAvailable := m.isRegistryAvailable(sessionCtx, client, namespace, s.Name)
		registry := m.buildModelRegistryModel(s, serverAddress, isAvailable)
		registries = append(registries, registry)
	}

	return registries, nil
}

// isRegistryAvailable returns true if the service's Endpoints have at least one ready address. On fetch error (e.g. Forbidden) returns true so we do not require new RBAC upstream.
func (m *ModelRegistryRepository) isRegistryAvailable(sessionCtx context.Context, client k8s.KubernetesClientInterface, namespace, serviceName string) bool {
	logger := helper.GetContextLogger(sessionCtx)
	endpoints, err := client.GetServiceEndpoints(sessionCtx, namespace, serviceName)
	if err != nil {
		logger.Debug("assuming registry available (endpoints fetch failed, may be permission)", "serviceName", serviceName, "namespace", namespace, "error", err)
		return true
	}
	return k8s.EndpointsHasReadyAddresses(endpoints)
}

// buildModelRegistryModel maps service details and availability into a ModelRegistryModel (DRY for list and get-by-name).
func (m *ModelRegistryRepository) buildModelRegistryModel(s k8s.ServiceDetails, serverAddress string, isAvailable bool) models.ModelRegistryModel {
	return models.ModelRegistryModel{
		Name:          s.Name,
		Description:   s.Description,
		DisplayName:   s.DisplayName,
		ServerAddress: serverAddress,
		IsHTTPS:       s.IsHTTPS,
		IsAvailable:   isAvailable,
	}
}

// getSpecificServiceDetails fetches details for specific services by name
func (m *ModelRegistryRepository) getSpecificServiceDetails(sessionCtx context.Context, client k8s.KubernetesClientInterface, namespace string, serviceNames []string) ([]k8s.ServiceDetails, error) {
	logger := helper.GetContextLogger(sessionCtx)
	logger.Debug("getSpecificServiceDetails called", "namespace", namespace, "serviceNames", serviceNames)

	var resources []k8s.ServiceDetails

	for _, serviceName := range serviceNames {
		logger.Debug("Fetching service details", "serviceName", serviceName, "namespace", namespace)
		// Validate if service is a model registry service by passing the component label value
		serviceDetail, err := client.GetServiceDetailsByName(sessionCtx, namespace, serviceName, k8s.ComponentLabelValue)
		if err != nil {
			logger.Warn("Failed to get service details, skipping",
				"serviceName", serviceName,
				"namespace", namespace,
				"error", err)
			// Log the error but continue with other services
			continue
		}
		logger.Debug("Service details retrieved successfully", "serviceName", serviceName)
		resources = append(resources, serviceDetail)
	}
	return resources, nil
}

func (m *ModelRegistryRepository) GetModelRegistry(sessionCtx context.Context, client k8s.KubernetesClientInterface, namespace string, modelRegistryID string) (models.ModelRegistryModel, error) {
	// Default to non-federated mode for backward compatibility
	return m.GetModelRegistryWithMode(sessionCtx, client, namespace, modelRegistryID, false)
}

// GetModelRegistryWithMode fetches a specific model registry with support for federated mode
func (m *ModelRegistryRepository) GetModelRegistryWithMode(sessionCtx context.Context, client k8s.KubernetesClientInterface, namespace string, modelRegistryID string, isFederatedMode bool) (models.ModelRegistryModel, error) {

	s, err := client.GetServiceDetailsByName(sessionCtx, namespace, modelRegistryID, k8s.ComponentLabelValue)
	if err != nil {
		return models.ModelRegistryModel{}, fmt.Errorf("error fetching model registry: %w", err)
	}

	isAvailable := m.isRegistryAvailable(sessionCtx, client, namespace, modelRegistryID)
	serverAddress := m.ResolveServerAddress(s.ClusterIP, s.HTTPPort, s.IsHTTPS, s.ExternalAddressRest, isFederatedMode)

	return m.buildModelRegistryModel(s, serverAddress, isAvailable), nil
}

func (m *ModelRegistryRepository) ResolveServerAddress(clusterIP string, httpPort int32, isHTTPS bool, externalAddressRest string, isFederatedMode bool) string {
	// Default behavior - use cluster IP and port
	protocol := "http"
	if isHTTPS {
		protocol = "https"
	}
	// In federated mode, if external address is available, use it
	if isFederatedMode && externalAddressRest != "" {
		// External address is assumed to be HTTPS
		url := fmt.Sprintf("%s://%s/api/model_registry/v1", protocol, externalAddressRest)
		return url
	}

	url := fmt.Sprintf("%s://%s:%d/api/model_registry/v1", protocol, clusterIP, httpPort)
	return url
}
