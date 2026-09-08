package repositories

import (
	"fmt"
	"net/url"

	"github.com/kubeflow/hub/ui/bff/internal/models"
	"gopkg.in/yaml.v3"
)

func FilterPageValues(values url.Values) url.Values {
	result := url.Values{}

	if v := values.Get("pageSize"); v != "" {
		result.Set("pageSize", v)
	}
	if v := values.Get("orderBy"); v != "" {
		result.Set("orderBy", v)
	}
	if v := values.Get("sortOrder"); v != "" {
		result.Set("sortOrder", v)
	}
	if v := values.Get("nextPageToken"); v != "" {
		result.Set("nextPageToken", v)
	}
	if v := values.Get("name"); v != "" {
		result.Set("name", v)
	}
	if v := values.Get("q"); v != "" {
		result.Set("q", v)
	}
	if v := values.Get("source"); v != "" {
		result.Set("source", v)
	}
	if v := values.Get("sourceLabel"); v != "" {
		result.Set("sourceLabel", v)
	}
	if v := values.Get("filterQuery"); v != "" {
		result.Set("filterQuery", v)
	}
	if v := values.Get("artifactType"); v != "" {
		result.Set("artifactType", v)
	}
	if v := values.Get("targetRPS"); v != "" {
		result.Set("targetRPS", v)
	}
	if v := values.Get("recommendations"); v != "" {
		result.Set("recommendations", v)
	}
	if v := values.Get("rpsProperty"); v != "" {
		result.Set("rpsProperty", v)
	}
	if v := values.Get("latencyProperty"); v != "" {
		result.Set("latencyProperty", v)
	}
	if v := values.Get("hardwareCountProperty"); v != "" {
		result.Set("hardwareCountProperty", v)
	}
	if v := values.Get("hardwareTypeProperty"); v != "" {
		result.Set("hardwareTypeProperty", v)
	}
	if v := values.Get("filterStatus"); v != "" {
		result.Set("filterStatus", v)
	}
	if v := values.Get("assetType"); v != "" {
		result.Set("assetType", v)
	}
	if v := values.Get("includeTools"); v != "" {
		result.Set("includeTools", v)
	}

	return result
}

func UrlWithParams(url string, values url.Values) string {
	queryString := values.Encode()
	if queryString == "" {
		return url
	}
	return fmt.Sprintf("%s?%s", url, queryString)
}

func UrlWithPageParams(url string, values url.Values) string {
	pageValues := FilterPageValues(values)
	return UrlWithParams(url, pageValues)
}

func ParseCatalogYaml(raw string, isDefault bool) ([]models.CatalogSourceConfig, error) {
	// Internal struct to match YAML structure
	var parsed struct {
		Catalogs []struct {
			Name           string                 `yaml:"name"`
			Id             string                 `yaml:"id"`
			Type           string                 `yaml:"type"`
			Enabled        *bool                  `yaml:"enabled"`
			Properties     map[string]interface{} `yaml:"properties"`
			Labels         []string               `yaml:"labels"`
			IncludedModels []string               `yaml:"includedModels"`
			ExcludedModels []string               `yaml:"excludedModels"`
		} `yaml:"catalogs"`
	}

	if err := yaml.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse catalogs yaml: %w", err)
	}

	catalogs := make([]models.CatalogSourceConfig, 0, len(parsed.Catalogs))
	for _, c := range parsed.Catalogs {
		entry := models.CatalogSourceConfig{
			Id:             c.Id,
			Name:           c.Name,
			Type:           c.Type,
			Enabled:        c.Enabled,
			Labels:         c.Labels,
			IsDefault:      &isDefault,
			IncludedModels: c.IncludedModels,
			ExcludedModels: c.ExcludedModels,
		}

		if c.Properties != nil {
			if apiKey, ok := c.Properties[ApiKey].(string); ok {
				entry.ApiKey = &apiKey
			}

			if allowedOrganization, ok := c.Properties["allowedOrganization"].(string); ok {
				entry.AllowedOrganization = &allowedOrganization
			}
		}
		catalogs = append(catalogs, entry)
	}

	return catalogs, nil
}

func ExtractStringSlice(value interface{}) []string {
	if arr, ok := value.([]interface{}); ok {
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	if strSlice, ok := value.([]string); ok {
		return strSlice
	}
	return []string{}

}

func FindCatalogSourceById(sourceYAML string, catalogId string, isDefault bool) *models.CatalogSourceConfig {
	if sourceYAML == "" {
		return nil
	}

	var parsed struct {
		Catalogs []struct {
			Id             string                 `yaml:"id"`
			Name           string                 `yaml:"name"`
			Type           string                 `yaml:"type"`
			Enabled        *bool                  `yaml:"enabled"`
			Labels         []string               `yaml:"labels"`
			Properties     map[string]interface{} `yaml:"properties"`
			IncludedModels []string               `yaml:"includedModels"`
			ExcludedModels []string               `yaml:"excludedModels"`
		} `yaml:"catalogs"`
	}

	if err := yaml.Unmarshal([]byte(sourceYAML), &parsed); err != nil {
		return nil
	}

	for _, catalog := range parsed.Catalogs {
		if catalog.Id == catalogId {
			isDefaultVal := isDefault
			result := &models.CatalogSourceConfig{
				Id:             catalog.Id,
				Name:           catalog.Name,
				Type:           catalog.Type,
				Enabled:        catalog.Enabled,
				Labels:         catalog.Labels,
				IsDefault:      &isDefaultVal,
				IncludedModels: catalog.IncludedModels,
				ExcludedModels: catalog.ExcludedModels,
			}

			if catalog.Properties != nil {
				if org, ok := catalog.Properties["allowedOrganization"].(string); ok {
					result.AllowedOrganization = &org
				}
			}

			return result
		}
	}

	return nil
}

func ConvertSourceConfigToYamlEntry(payload models.CatalogSourceConfigPayload,
	yamlFileName string,
	secretName string) map[string]interface{} {
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

	switch payload.Type {
	case CatalogTypeYaml:
		properties["yamlCatalogPath"] = yamlFileName
	case CatalogTypeHuggingFace:
		properties[ApiKey] = secretName
		if payload.AllowedOrganization != nil {
			properties["allowedOrganization"] = *payload.AllowedOrganization
		}

	}

	if len(properties) > 0 {
		entry["properties"] = properties
	}

	if len(payload.IncludedModels) > 0 {
		entry["includedModels"] = payload.IncludedModels
	}
	if len(payload.ExcludedModels) > 0 {
		entry["excludedModels"] = payload.ExcludedModels
	}

	return entry
}

func AppendCatalogSourceToYaml(existingConfigMapEntry string, newEntry map[string]interface{}) (string, error) {
	var parsed struct {
		Catalogs []map[string]interface{} `yaml:"catalogs"`
	}

	if existingConfigMapEntry != "" {
		if err := yaml.Unmarshal([]byte(existingConfigMapEntry), &parsed); err != nil {
			return "", fmt.Errorf("failed to parse existing sources.yaml: %w", err)
		}
	} else {
		parsed.Catalogs = []map[string]interface{}{}
	}
	parsed.Catalogs = append(parsed.Catalogs, newEntry)

	updatedBytes, err := yaml.Marshal(parsed)
	if err != nil {
		return "", fmt.Errorf("failed to marshal updated sources.yaml: %w", err)
	}

	return string(updatedBytes), nil
}

func RemoveCatalogSourceFromYAML(existingYAML string, sourceId string) (string, error) {
	var parsed struct {
		Catalogs []map[string]interface{} `yaml:"catalogs"`
	}

	if err := yaml.Unmarshal([]byte(existingYAML), &parsed); err != nil {
		return "", fmt.Errorf("failed to parse sources.yaml: %w", err)
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
		return "", fmt.Errorf("failed to marshal updated sources.yaml: %w", err)
	}

	return string(updatedBytes), nil
}

func FindCatalogSourceProperties(sourceYAML string, sourceId string) (secretName string, yamlPath string) {
	var parsed struct {
		Catalogs []struct {
			Id         string                 `yaml:"id"`
			Properties map[string]interface{} `yaml:"properties"`
		} `yaml:"catalogs"`
	}

	if err := yaml.Unmarshal([]byte(sourceYAML), &parsed); err != nil {
		return "", ""
	}

	for _, catalogSource := range parsed.Catalogs {
		if catalogSource.Id == sourceId && catalogSource.Properties != nil {
			secretName, _ = catalogSource.Properties[ApiKey].(string)
			yamlPath, _ = catalogSource.Properties["yamlCatalogPath"].(string)
			return
		}
	}
	return "", ""
}

func UpdateCatalogSourceInYAML(
	existingYAML string,
	catalogId string,
	payload models.CatalogSourceConfigPayload,
	secretName string,
	yamlFilePath string,
	clearApiKey bool,
) (string, error) {
	var parsed struct {
		Catalogs []map[string]interface{} `yaml:"catalogs"`
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

			properties, _ := catalogSource["properties"].(map[string]interface{})
			if properties == nil {
				properties = make(map[string]interface{})
			}

			if payload.IncludedModels != nil {
				if len(payload.IncludedModels) > 0 {
					catalogSource["includedModels"] = payload.IncludedModels
				} else {
					delete(catalogSource, "includedModels")
				}
			}
			if payload.ExcludedModels != nil {
				if len(payload.ExcludedModels) > 0 {
					catalogSource["excludedModels"] = payload.ExcludedModels
				} else {
					delete(catalogSource, "excludedModels")
				}
			}
			if payload.AllowedOrganization != nil {
				properties["allowedOrganization"] = *payload.AllowedOrganization
			}
			if clearApiKey {
				delete(properties, ApiKey)
			} else if secretName != "" {
				properties["apiKey"] = secretName
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
		return "", fmt.Errorf("catalog '%s' not found in yaml", catalogId)
	}

	updatedBytes, err := yaml.Marshal(parsed)
	if err != nil {
		return "", fmt.Errorf("failed to marshal updated sources.yaml: %w", err)
	}

	return string(updatedBytes), nil
}

func BuildOverrideEntryForDefaultSource(catalogId string, payload models.CatalogSourceConfigPayload) map[string]interface{} {
	entry := map[string]interface{}{
		"id": catalogId,
	}

	if payload.Enabled != nil {
		entry["enabled"] = *payload.Enabled
	}

	if len(payload.IncludedModels) > 0 {
		entry["includedModels"] = payload.IncludedModels
	}
	if len(payload.ExcludedModels) > 0 {
		entry["excludedModels"] = payload.ExcludedModels
	}

	return entry
}
