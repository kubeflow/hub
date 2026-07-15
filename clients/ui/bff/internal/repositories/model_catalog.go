package repositories

import (
	"context"
	"fmt"

	k8s "github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/models"
)

const (
	ModelCatalogServiceName = "model-catalog"
	ModelCatalogAPIPath     = "/api/model_catalog/v1alpha1"
	McpCatalogAPIPath       = "/api/mcp_catalog/v1alpha1"
	AgentCatalogAPIPath     = "/api/agent_catalog/v1alpha1"
)

type ModelCatalogRepository struct {
}

func NewCatalogRepository() *ModelCatalogRepository {
	return &ModelCatalogRepository{}
}

func (m *ModelCatalogRepository) GetModelCatalogWithMode(sessionCtx context.Context, client k8s.KubernetesClientInterface, namespace string, isFederatedMode bool) (models.ModelCatalogModel, error) {

	s, err := client.GetServiceDetailsByName(sessionCtx, namespace, ModelCatalogServiceName, k8s.ComponentLabelValueCatalog)
	if err != nil {
		return models.ModelCatalogModel{}, fmt.Errorf("error fetching model catalog: %w", err)
	}

	modelCatalog := models.ModelCatalogModel{
		Name:          s.Name,
		Description:   s.Description,
		DisplayName:   s.DisplayName,
		ServerAddress: m.ResolveServerAddress(s.ClusterIP, s.HTTPPort, s.IsHTTPS, s.ExternalAddressRest, isFederatedMode, ModelCatalogAPIPath),
		IsHTTPS:       s.IsHTTPS,
	}
	return modelCatalog, nil
}

func (m *ModelCatalogRepository) ResolveServerAddress(clusterIP string, httpPort int32, isHTTPS bool, externalAddressRest string, isFederatedMode bool, apiPath string) string {
	protocol := "http"
	if isHTTPS {
		protocol = "https"
	}
	if isFederatedMode && externalAddressRest != "" {
		return fmt.Sprintf("%s://%s%s", protocol, externalAddressRest, apiPath)
	}

	return fmt.Sprintf("%s://%s:%d%s", protocol, clusterIP, httpPort, apiPath)
}
