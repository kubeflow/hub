package repositories

import (
	"context"
	"errors"
	"fmt"

	k8s "github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/models"
)

var (
	ErrMcpCatalogSourceNotFound     = errors.New("mcp catalog source not found")
	ErrMcpCatalogSourceAlreadyExist = errors.New("mcp catalog source already exists")
	ErrMcpCatalogSourceIdRequired   = errors.New("mcp catalog source ID is required")
	ErrMcpCatalogSourceConflict     = errors.New("mcp catalog source was modified by another request")
)

type McpCatalogSettingsRepository struct {
}

func NewMcpCatalogSettingsRepository() *McpCatalogSettingsRepository {
	return &McpCatalogSettingsRepository{}
}

func (r *McpCatalogSettingsRepository) GetAllMcpCatalogSourceConfigs(_ context.Context, _ k8s.KubernetesClientInterface, _ string) (*models.McpCatalogSourceConfigList, error) {
	return nil, fmt.Errorf("not implemented yet")
}

func (r *McpCatalogSettingsRepository) GetMcpCatalogSourceConfig(_ context.Context, _ k8s.KubernetesClientInterface, _ string, _ string) (*models.McpCatalogSourceConfig, error) {
	return nil, fmt.Errorf("not implemented yet")
}

func (r *McpCatalogSettingsRepository) CreateMcpCatalogSourceConfig(_ context.Context, _ k8s.KubernetesClientInterface, _ string, _ models.McpCatalogSourceConfigPayload) (*models.McpCatalogSourceConfig, error) {
	return nil, fmt.Errorf("not implemented yet")
}

func (r *McpCatalogSettingsRepository) UpdateMcpCatalogSourceConfig(_ context.Context, _ k8s.KubernetesClientInterface, _ string, _ string, _ models.McpCatalogSourceConfigPayload) (*models.McpCatalogSourceConfig, error) {
	return nil, fmt.Errorf("not implemented yet")
}

func (r *McpCatalogSettingsRepository) DeleteMcpCatalogSourceConfig(_ context.Context, _ k8s.KubernetesClientInterface, _ string, _ string) (*models.McpCatalogSourceConfig, error) {
	return nil, fmt.Errorf("not implemented yet")
}
