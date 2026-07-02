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
	ErrMcpCatalogNotImplemented     = errors.New("mcp catalog settings not implemented yet")
)

type McpCatalogSettingsRepository struct {
}

func NewMcpCatalogSettingsRepository() *McpCatalogSettingsRepository {
	return &McpCatalogSettingsRepository{}
}

func (r *McpCatalogSettingsRepository) GetAllMcpCatalogSourceConfigs(_ context.Context, _ k8s.KubernetesClientInterface, _ string) (*models.McpCatalogSourceConfigList, error) {
	enabled := true
	disabled := false
	isDefault := true
	return &models.McpCatalogSourceConfigList{
		Catalogs: []models.McpCatalogSourceConfig{
			{
				Id:        "community-mcp-source",
				Name:      "Community MCP Servers",
				Type:      "yaml",
				Enabled:   &enabled,
				Labels:    []string{"community_mcp_servers"},
				IsDefault: &isDefault,
			},
			{
				Id:      "organization-mcp-source",
				Name:    "Organization MCP Servers",
				Type:    "yaml",
				Enabled: &enabled,
				Labels:  []string{"organization_mcp_servers"},
			},
			{
				Id:      "standalone-mcp-source",
				Name:    "Other MCP Servers",
				Type:    "yaml",
				Enabled: &enabled,
				Labels:  []string{},
			},
			{
				Id:      "disabled-mcp-source",
				Name:    "Disabled MCP source",
				Type:    "yaml",
				Enabled: &disabled,
				Labels:  []string{"disabled_servers"},
			},
		},
	}, nil
}

func (r *McpCatalogSettingsRepository) GetMcpCatalogSourceConfig(ctx context.Context, client k8s.KubernetesClientInterface, namespace string, sourceID string) (*models.McpCatalogSourceConfig, error) {
	all, err := r.GetAllMcpCatalogSourceConfigs(ctx, client, namespace)
	if err != nil {
		return nil, err
	}
	for i := range all.Catalogs {
		if all.Catalogs[i].Id == sourceID {
			return &all.Catalogs[i], nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrMcpCatalogSourceNotFound, sourceID)
}

func (r *McpCatalogSettingsRepository) CreateMcpCatalogSourceConfig(_ context.Context, _ k8s.KubernetesClientInterface, _ string, payload models.McpCatalogSourceConfigPayload) (*models.McpCatalogSourceConfig, error) {
	if payload.Id == "" {
		return nil, ErrMcpCatalogSourceIdRequired
	}
	enabled := true
	return &models.McpCatalogSourceConfig{
		Id:      payload.Id,
		Name:    payload.Name,
		Type:    payload.Type,
		Enabled: &enabled,
		Labels:  payload.Labels,
	}, nil
}

func (r *McpCatalogSettingsRepository) UpdateMcpCatalogSourceConfig(_ context.Context, _ k8s.KubernetesClientInterface, _ string, sourceID string, payload models.McpCatalogSourceConfigPayload) (*models.McpCatalogSourceConfig, error) {
	enabled := true
	return &models.McpCatalogSourceConfig{
		Id:      sourceID,
		Name:    payload.Name,
		Type:    payload.Type,
		Enabled: &enabled,
		Labels:  payload.Labels,
	}, nil
}

func (r *McpCatalogSettingsRepository) DeleteMcpCatalogSourceConfig(_ context.Context, _ k8s.KubernetesClientInterface, _ string, _ string) (*models.McpCatalogSourceConfig, error) {
	return nil, nil
}
