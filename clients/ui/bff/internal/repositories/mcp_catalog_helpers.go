package repositories

import (
	"fmt"

	"github.com/kubeflow/hub/ui/bff/internal/models"
	"gopkg.in/yaml.v3"
)

const (
	McpCatalogTypeYaml = "yaml"
)

func ParseMcpCatalogYaml(raw string, isDefault bool) ([]models.McpCatalogSourceConfig, error) {
	var parsed struct {
		Catalogs []struct {
			Name            string                 `yaml:"name"`
			Id              string                 `yaml:"id"`
			Type            string                 `yaml:"type"`
			Enabled         *bool                  `yaml:"enabled"`
			Properties      map[string]interface{} `yaml:"properties"`
			Labels          []string               `yaml:"labels"`
			IncludedServers []string               `yaml:"includedServers"`
			ExcludedServers []string               `yaml:"excludedServers"`
		} `yaml:"mcp_catalogs"`
	}

	if err := yaml.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse MCP catalogs yaml: %w", err)
	}

	catalogs := make([]models.McpCatalogSourceConfig, 0, len(parsed.Catalogs))
	for _, c := range parsed.Catalogs {
		entry := models.McpCatalogSourceConfig{
			Id:              c.Id,
			Name:            c.Name,
			Type:            c.Type,
			Enabled:         c.Enabled,
			Labels:          c.Labels,
			IsDefault:       &isDefault,
			IncludedServers: c.IncludedServers,
			ExcludedServers: c.ExcludedServers,
		}
		catalogs = append(catalogs, entry)
	}

	return catalogs, nil
}

func FindMcpCatalogSourceById(sourceYAML string, catalogId string, isDefault bool) *models.McpCatalogSourceConfig {
	if sourceYAML == "" {
		return nil
	}

	var parsed struct {
		Catalogs []struct {
			Id              string   `yaml:"id"`
			Name            string   `yaml:"name"`
			Type            string   `yaml:"type"`
			Enabled         *bool    `yaml:"enabled"`
			Labels          []string `yaml:"labels"`
			IncludedServers []string `yaml:"includedServers"`
			ExcludedServers []string `yaml:"excludedServers"`
		} `yaml:"mcp_catalogs"`
	}

	if err := yaml.Unmarshal([]byte(sourceYAML), &parsed); err != nil {
		return nil
	}

	for _, catalog := range parsed.Catalogs {
		if catalog.Id == catalogId {
			isDefaultVal := isDefault
			return &models.McpCatalogSourceConfig{
				Id:              catalog.Id,
				Name:            catalog.Name,
				Type:            catalog.Type,
				Enabled:         catalog.Enabled,
				Labels:          catalog.Labels,
				IsDefault:       &isDefaultVal,
				IncludedServers: catalog.IncludedServers,
				ExcludedServers: catalog.ExcludedServers,
			}
		}
	}

	return nil
}

func FindMcpCatalogSourceProperties(sourceYAML string, sourceId string) (yamlPath string) {
	var parsed struct {
		Catalogs []struct {
			Id         string                 `yaml:"id"`
			Properties map[string]interface{} `yaml:"properties"`
		} `yaml:"mcp_catalogs"`
	}

	if err := yaml.Unmarshal([]byte(sourceYAML), &parsed); err != nil {
		return ""
	}

	for _, catalogSource := range parsed.Catalogs {
		if catalogSource.Id == sourceId && catalogSource.Properties != nil {
			yamlPath, _ = catalogSource.Properties["yamlCatalogPath"].(string)
			return
		}
	}
	return ""
}

func AppendMcpCatalogSourceToYaml(existingConfigMapEntry string, newEntry map[string]interface{}) (string, error) {
	var parsed struct {
		Catalogs []map[string]interface{} `yaml:"mcp_catalogs"`
	}

	if existingConfigMapEntry != "" {
		if err := yaml.Unmarshal([]byte(existingConfigMapEntry), &parsed); err != nil {
			return "", fmt.Errorf("failed to parse existing MCP sources.yaml: %w", err)
		}
	} else {
		parsed.Catalogs = []map[string]interface{}{}
	}
	parsed.Catalogs = append(parsed.Catalogs, newEntry)

	updatedBytes, err := yaml.Marshal(parsed)
	if err != nil {
		return "", fmt.Errorf("failed to marshal updated MCP sources.yaml: %w", err)
	}

	return string(updatedBytes), nil
}

func RemoveMcpCatalogSourceFromYAML(existingYAML string, sourceId string) (string, error) {
	var parsed struct {
		Catalogs []map[string]interface{} `yaml:"mcp_catalogs"`
	}

	if err := yaml.Unmarshal([]byte(existingYAML), &parsed); err != nil {
		return "", fmt.Errorf("failed to parse MCP sources.yaml: %w", err)
	}

	filteredCatalogs := make([]map[string]interface{}, 0)
	for _, catalogSource := range parsed.Catalogs {
		if id, ok := catalogSource["id"].(string); ok && id != sourceId {
			filteredCatalogs = append(filteredCatalogs, catalogSource)
		}
	}

	parsed.Catalogs = filteredCatalogs
	updatedBytes, err := yaml.Marshal(parsed)
	if err != nil {
		return "", fmt.Errorf("failed to marshal updated MCP sources.yaml: %w", err)
	}

	return string(updatedBytes), nil
}

func ConvertMcpSourceConfigToYamlEntry(payload models.McpCatalogSourceConfigPayload, yamlFileName string) map[string]interface{} {
	entry := map[string]interface{}{
		"id":      payload.Id,
		"name":    payload.Name,
		"type":    payload.Type,
		"enabled": payload.Enabled,
	}

	if len(payload.Labels) > 0 {
		entry["labels"] = payload.Labels
	}

	properties := make(map[string]interface{})
	if yamlFileName != "" {
		properties["yamlCatalogPath"] = yamlFileName
	}
	if len(properties) > 0 {
		entry["properties"] = properties
	}

	if len(payload.IncludedServers) > 0 {
		entry["includedServers"] = payload.IncludedServers
	}
	if len(payload.ExcludedServers) > 0 {
		entry["excludedServers"] = payload.ExcludedServers
	}

	return entry
}

func UpdateMcpCatalogSourceInYAML(
	existingYAML string,
	catalogId string,
	payload models.McpCatalogSourceConfigPayload,
	yamlFilePath string,
) (string, error) {
	var parsed struct {
		Catalogs []map[string]interface{} `yaml:"mcp_catalogs"`
	}

	if existingYAML == "" {
		return "", fmt.Errorf("no existing yaml to update")
	}

	if err := yaml.Unmarshal([]byte(existingYAML), &parsed); err != nil {
		return "", fmt.Errorf("failed to parse sources.yaml: %w", err)
	}

	found := false
	for i, catalogSource := range parsed.Catalogs {
		if id, ok := catalogSource["id"].(string); ok && id == catalogId {
			found = true

			if payload.Name != "" {
				catalogSource["name"] = payload.Name
			}
			if len(payload.Labels) > 0 {
				catalogSource["labels"] = payload.Labels
			}
			if payload.Enabled != nil {
				catalogSource["enabled"] = *payload.Enabled
			}

			if payload.IncludedServers != nil {
				if len(payload.IncludedServers) > 0 {
					catalogSource["includedServers"] = payload.IncludedServers
				} else {
					delete(catalogSource, "includedServers")
				}
			}
			if payload.ExcludedServers != nil {
				if len(payload.ExcludedServers) > 0 {
					catalogSource["excludedServers"] = payload.ExcludedServers
				} else {
					delete(catalogSource, "excludedServers")
				}
			}

			properties, _ := catalogSource["properties"].(map[string]interface{})
			if properties == nil {
				properties = make(map[string]interface{})
			}
			if yamlFilePath != "" && payload.Yaml != nil {
				properties["yamlCatalogPath"] = yamlFilePath
			}
			if len(properties) > 0 {
				catalogSource["properties"] = properties
			}

			parsed.Catalogs[i] = catalogSource
			break
		}
	}

	if !found {
		return "", fmt.Errorf("MCP catalog '%s' not found in yaml", catalogId)
	}

	updatedBytes, err := yaml.Marshal(parsed)
	if err != nil {
		return "", fmt.Errorf("failed to marshal updated sources.yaml: %w", err)
	}

	return string(updatedBytes), nil
}

func BuildOverrideEntryForDefaultMcpSource(catalogId string, payload models.McpCatalogSourceConfigPayload) map[string]interface{} {
	entry := map[string]interface{}{
		"id": catalogId,
	}

	if payload.Enabled != nil {
		entry["enabled"] = *payload.Enabled
	}

	if len(payload.IncludedServers) > 0 {
		entry["includedServers"] = payload.IncludedServers
	}
	if len(payload.ExcludedServers) > 0 {
		entry["excludedServers"] = payload.ExcludedServers
	}

	return entry
}

func mergeMcpCatalogSourceConfigs(defaultCatalog models.McpCatalogSourceConfig, userCatalog models.McpCatalogSourceConfig) models.McpCatalogSourceConfig {
	mergedSource := defaultCatalog

	if userCatalog.Name != "" {
		mergedSource.Name = userCatalog.Name
	}

	if userCatalog.Type != "" {
		mergedSource.Type = userCatalog.Type
	}

	if userCatalog.Enabled != nil {
		mergedSource.Enabled = userCatalog.Enabled
	}

	if userCatalog.IncludedServers != nil {
		mergedSource.IncludedServers = userCatalog.IncludedServers
	}

	if userCatalog.ExcludedServers != nil {
		mergedSource.ExcludedServers = userCatalog.ExcludedServers
	}

	if userCatalog.Yaml != nil {
		mergedSource.Yaml = userCatalog.Yaml
	}

	return mergedSource
}

func validateMcpCatalogSourceConfigPayload(payload models.McpCatalogSourceConfigPayload) error {
	if payload.Id == "" {
		return fmt.Errorf("%w", ErrMcpCatalogSourceIdRequired)
	}

	if err := validateCatalogId(payload.Id); err != nil {
		return err
	}

	if payload.Name == "" {
		return fmt.Errorf("%w: name is required", ErrMcpCatalogValidationFailed)
	}

	if payload.Type == "" {
		return fmt.Errorf("%w: type is required", ErrMcpCatalogValidationFailed)
	}

	if payload.Type != McpCatalogTypeYaml {
		return fmt.Errorf("%w: unsupported MCP catalog type: %s (supported: yaml)", ErrMcpCatalogValidationFailed, payload.Type)
	}

	if payload.Yaml == nil || *payload.Yaml == "" {
		return fmt.Errorf("%w: yaml field is required for yaml-type sources", ErrMcpCatalogValidationFailed)
	}

	return nil
}

func validateMcpUpdatePayloadForDefaultOverride(payload models.McpCatalogSourceConfigPayload) error {
	if payload.Name != "" {
		return fmt.Errorf("%w: cannot change 'name'", ErrMcpCatalogCannotChangeDefault)
	}

	if len(payload.Labels) > 0 {
		return fmt.Errorf("%w: cannot change 'labels'", ErrMcpCatalogCannotChangeDefault)
	}

	if payload.Yaml != nil && *payload.Yaml != "" {
		return fmt.Errorf("%w: cannot change 'yaml' content", ErrMcpCatalogCannotChangeDefault)
	}

	return nil
}
