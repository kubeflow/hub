package repositories

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/kubeflow/hub/ui/bff/internal/constants"
	k8s "github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

var (
	ErrMcpCatalogSourceNotFound      = errors.New("mcp catalog source not found")
	ErrMcpCatalogSourceAlreadyExist  = errors.New("mcp catalog source already exists")
	ErrMcpCatalogSourceIdRequired    = errors.New("mcp catalog source ID is required")
	ErrMcpCatalogSourceConflict      = errors.New("mcp catalog source was modified by another request")
	ErrMcpCatalogCannotChangeDefault = errors.New("cannot change the default MCP source")
	ErrMcpCatalogCannotDeleteDefault = errors.New("cannot delete the default MCP source")
	ErrMcpCatalogValidationFailed    = errors.New("validation failed")
	ErrMcpCatalogCannotChangeType    = errors.New("cannot change MCP catalog source type")
)

type McpCatalogSettingsRepository struct {
}

func NewMcpCatalogSettingsRepository() *McpCatalogSettingsRepository {
	return &McpCatalogSettingsRepository{}
}

func (r *McpCatalogSettingsRepository) GetAllMcpCatalogSourceConfigs(ctx context.Context, client k8s.KubernetesClientInterface, namespace string) (*models.McpCatalogSourceConfigList, error) {
	defaultCM, userCM, err := client.GetAllMcpCatalogSourceConfigs(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch MCP catalog source configmaps: %w", err)
	}

	catalogMap := make(map[string]models.McpCatalogSourceConfig)

	if raw, ok := defaultCM.Data[k8s.McpCatalogSourceKey]; ok {
		defaultCatalogSources, err := ParseMcpCatalogYaml(raw, true)
		if err != nil {
			return nil, fmt.Errorf("failed to parse default MCP catalogs: %w", err)
		}
		for _, catalog := range defaultCatalogSources {
			catalogMap[catalog.Id] = catalog
		}
	}

	if raw, ok := userCM.Data[k8s.McpCatalogSourceKey]; ok {
		userManagedSources, err := ParseMcpCatalogYaml(raw, false)
		if err != nil {
			return nil, fmt.Errorf("failed to parse user managed MCP catalogs: %w", err)
		}
		for _, userCatalogSource := range userManagedSources {
			if existingSource, exist := catalogMap[userCatalogSource.Id]; exist {
				mergedCatalogSources := mergeMcpCatalogSourceConfigs(existingSource, userCatalogSource)
				catalogMap[userCatalogSource.Id] = mergedCatalogSources
			} else {
				catalogMap[userCatalogSource.Id] = userCatalogSource
			}
		}
	}

	catalogSources := &models.McpCatalogSourceConfigList{
		Catalogs: make([]models.McpCatalogSourceConfig, 0),
	}

	for _, c := range catalogMap {
		catalogSources.Catalogs = append(catalogSources.Catalogs, c)
	}

	return catalogSources, nil
}

func (r *McpCatalogSettingsRepository) GetMcpCatalogSourceConfig(ctx context.Context, client k8s.KubernetesClientInterface, namespace string, sourceID string) (*models.McpCatalogSourceConfig, error) {
	defaultCM, userCM, err := client.GetAllMcpCatalogSourceConfigs(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch MCP catalog source configmaps: %w", err)
	}

	defaultSource := FindMcpCatalogSourceById(defaultCM.Data[k8s.McpCatalogSourceKey], sourceID, true)
	userSource := FindMcpCatalogSourceById(userCM.Data[k8s.McpCatalogSourceKey], sourceID, false)

	var result *models.McpCatalogSourceConfig

	if userSource != nil {
		if defaultSource != nil {
			merged := mergeMcpCatalogSourceConfigs(*defaultSource, *userSource)
			result = &merged
		} else {
			result = userSource
		}
	} else if defaultSource != nil {
		result = defaultSource
	} else {
		return nil, fmt.Errorf("%w, %s", ErrMcpCatalogSourceNotFound, sourceID)
	}

	yamlFilePath := FindMcpCatalogSourceProperties(userCM.Data[k8s.McpCatalogSourceKey], sourceID)
	if yamlFilePath == "" {
		yamlFilePath = FindMcpCatalogSourceProperties(defaultCM.Data[k8s.McpCatalogSourceKey], sourceID)
	}

	if result.Type == McpCatalogTypeYaml {
		if yamlFilePath != "" {
			result.YamlCatalogPath = &yamlFilePath
			if yamlContent, ok := userCM.Data[yamlFilePath]; ok {
				result.Yaml = &yamlContent
			} else if yamlContent, ok := defaultCM.Data[yamlFilePath]; ok {
				result.Yaml = &yamlContent
			} else if result.IsDefault == nil || !*result.IsDefault {
				sessionLogger := ctx.Value(constants.TraceLoggerKey).(*slog.Logger)
				sessionLogger.Warn("MCP yaml catalog content missing from configmap",
					"catalogId", sourceID,
					"expectedPath", yamlFilePath,
				)
			}
		}
	}

	return result, nil
}

func (r *McpCatalogSettingsRepository) CreateMcpCatalogSourceConfig(
	ctx context.Context,
	client k8s.KubernetesClientInterface,
	namespace string,
	payload models.McpCatalogSourceConfigPayload,
) (*models.McpCatalogSourceConfig, error) {
	if err := validateMcpCatalogSourceConfigPayload(payload); err != nil {
		return nil, err
	}

	defaultCM, userCM, err := client.GetAllMcpCatalogSourceConfigs(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch MCP catalog source configmaps: %w", err)
	}

	if FindMcpCatalogSourceById(defaultCM.Data[k8s.McpCatalogSourceKey], payload.Id, true) != nil {
		return nil, fmt.Errorf("%w: '%s' already exists in default sources", ErrMcpCatalogSourceAlreadyExist, payload.Id)
	}

	if FindMcpCatalogSourceById(userCM.Data[k8s.McpCatalogSourceKey], payload.Id, false) != nil {
		return nil, fmt.Errorf("%w: '%s' already exists in user managed sources", ErrMcpCatalogSourceAlreadyExist, payload.Id)
	}

	yamlFileName := fmt.Sprintf("%s.yaml", payload.Id)
	yamlContent := make(map[string]string)
	yamlContent[yamlFileName] = *payload.Yaml

	newEntry := ConvertMcpSourceConfigToYamlEntry(payload, yamlFileName)

	existingConfigMapEntry := userCM.Data[k8s.McpCatalogSourceKey]

	updatedConfigMapEntry, err := AppendMcpCatalogSourceToYaml(existingConfigMapEntry, newEntry)
	if err != nil {
		return nil, fmt.Errorf("failed to append MCP catalog to yaml: %w", err)
	}

	if userCM.Data == nil {
		userCM.Data = make(map[string]string)
	}
	userCM.Data[k8s.McpCatalogSourceKey] = updatedConfigMapEntry

	for key, value := range yamlContent {
		userCM.Data[key] = value
	}

	err = client.UpdateMcpCatalogSourceConfig(ctx, namespace, &userCM)
	if err != nil {
		if apierrors.IsConflict(err) {
			return nil, fmt.Errorf("%w: %v", ErrMcpCatalogSourceConflict, err)
		}
		return nil, fmt.Errorf("failed to update user MCP configmap: %w", err)
	}

	return r.GetMcpCatalogSourceConfig(ctx, client, namespace, payload.Id)
}

func (r *McpCatalogSettingsRepository) UpdateMcpCatalogSourceConfig(
	ctx context.Context,
	client k8s.KubernetesClientInterface,
	namespace string,
	sourceID string,
	payload models.McpCatalogSourceConfigPayload,
) (*models.McpCatalogSourceConfig, error) {
	defaultCM, userCM, err := client.GetAllMcpCatalogSourceConfigs(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch MCP catalog source configmaps: %w", err)
	}

	existingUserSource := FindMcpCatalogSourceById(userCM.Data[k8s.McpCatalogSourceKey], sourceID, false)
	existingDefaultSource := FindMcpCatalogSourceById(defaultCM.Data[k8s.McpCatalogSourceKey], sourceID, true)

	var existingCatalog *models.McpCatalogSourceConfig
	var isOverridingDefault bool

	if existingUserSource != nil {
		existingCatalog = existingUserSource
		isOverridingDefault = existingDefaultSource != nil
	} else if existingDefaultSource != nil {
		existingCatalog = existingDefaultSource
		isOverridingDefault = true
	} else {
		return nil, fmt.Errorf("%w: '%s'", ErrMcpCatalogSourceNotFound, sourceID)
	}

	if payload.Type != "" && payload.Type != existingCatalog.Type {
		return nil, fmt.Errorf(
			"%w: cannot change from '%s' to '%s'",
			ErrMcpCatalogCannotChangeType,
			existingCatalog.Type,
			payload.Type,
		)
	}

	if isOverridingDefault {
		if err := validateMcpUpdatePayloadForDefaultOverride(payload); err != nil {
			return nil, err
		}
	}

	var yamlFilePath string
	if !isOverridingDefault || existingUserSource != nil {
		yamlFilePath = FindMcpCatalogSourceProperties(userCM.Data[k8s.McpCatalogSourceKey], sourceID)
		if yamlFilePath == "" {
			yamlFilePath = FindMcpCatalogSourceProperties(defaultCM.Data[k8s.McpCatalogSourceKey], sourceID)
		}
	}

	if userCM.Data == nil {
		userCM.Data = make(map[string]string)
	}

	if !isOverridingDefault || existingUserSource != nil {
		if payload.Yaml != nil && *payload.Yaml != "" {
			if yamlFilePath == "" {
				yamlFilePath = fmt.Sprintf("%s.yaml", sourceID)
			}
			userCM.Data[yamlFilePath] = *payload.Yaml
		}
	}

	if existingUserSource != nil {
		updatedYAML, err := UpdateMcpCatalogSourceInYAML(
			userCM.Data[k8s.McpCatalogSourceKey],
			sourceID,
			payload,
			yamlFilePath,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to update MCP catalog in yaml: %w", err)
		}
		userCM.Data[k8s.McpCatalogSourceKey] = updatedYAML
	} else {
		overrideEntry := BuildOverrideEntryForDefaultMcpSource(sourceID, payload)
		updatedYAML, err := AppendMcpCatalogSourceToYaml(userCM.Data[k8s.McpCatalogSourceKey], overrideEntry)
		if err != nil {
			return nil, fmt.Errorf("failed to append override entry: %w", err)
		}
		userCM.Data[k8s.McpCatalogSourceKey] = updatedYAML
	}

	err = client.UpdateMcpCatalogSourceConfig(ctx, namespace, &userCM)
	if err != nil {
		if apierrors.IsConflict(err) {
			return nil, fmt.Errorf("%w: %v", ErrMcpCatalogSourceConflict, err)
		}
		return nil, fmt.Errorf("failed to update user MCP configmap: %w", err)
	}

	return r.GetMcpCatalogSourceConfig(ctx, client, namespace, sourceID)
}

func (r *McpCatalogSettingsRepository) DeleteMcpCatalogSourceConfig(
	ctx context.Context,
	client k8s.KubernetesClientInterface,
	namespace string,
	sourceID string,
) (*models.McpCatalogSourceConfig, error) {
	defaultCM, userCM, err := client.GetAllMcpCatalogSourceConfigs(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch MCP catalog source configmaps: %w", err)
	}

	if FindMcpCatalogSourceById(defaultCM.Data[k8s.McpCatalogSourceKey], sourceID, true) != nil {
		return nil, fmt.Errorf("%w: '%s' is a default source", ErrMcpCatalogCannotDeleteDefault, sourceID)
	}

	catalogSourceToDelete := FindMcpCatalogSourceById(userCM.Data[k8s.McpCatalogSourceKey], sourceID, false)
	if catalogSourceToDelete == nil {
		return nil, fmt.Errorf("%w: '%s' not found in user sources", ErrMcpCatalogSourceNotFound, sourceID)
	}

	yamlFilePath := FindMcpCatalogSourceProperties(userCM.Data[k8s.McpCatalogSourceKey], sourceID)
	if catalogSourceToDelete.Type == McpCatalogTypeYaml && yamlFilePath != "" {
		delete(userCM.Data, yamlFilePath)
	}

	updatedYAML, err := RemoveMcpCatalogSourceFromYAML(userCM.Data[k8s.McpCatalogSourceKey], sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to remove MCP catalog from sources.yaml: %w", err)
	}
	userCM.Data[k8s.McpCatalogSourceKey] = updatedYAML

	err = client.UpdateMcpCatalogSourceConfig(ctx, namespace, &userCM)
	if err != nil {
		if apierrors.IsConflict(err) {
			return nil, fmt.Errorf("%w: %v", ErrMcpCatalogSourceConflict, err)
		}
		return nil, fmt.Errorf("failed to update configmap after deletion: %w", err)
	}

	return catalogSourceToDelete, nil
}
