package mcpcatalog

import (
	"context"
	"fmt"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	model "github.com/kubeflow/hub/catalog/pkg/openapi"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// MCPPreviewConfig represents the parsed preview request configuration for MCP sources.
type MCPPreviewConfig struct {
	Type            string         `json:"type" yaml:"type"`
	AssetType       string         `json:"assetType,omitempty" yaml:"assetType,omitempty"`
	IncludedServers []string       `json:"includedServers,omitempty" yaml:"includedServers,omitempty"`
	ExcludedServers []string       `json:"excludedServers,omitempty" yaml:"excludedServers,omitempty"`
	Properties      map[string]any `json:"properties,omitempty" yaml:"properties,omitempty"`

	// Files reads yamlCatalogPath files in path mode, confined to the server's
	// catalog directories. It is set by the server, never parsed from the
	// uploaded config.
	Files *basecatalog.PreviewFiles `json:"-" yaml:"-"`
}

// ParseMCPPreviewConfig parses the uploaded config bytes into an MCPPreviewConfig.
// Extra fields (like name, id, enabled) are ignored so users can paste
// a full source config entry directly for preview.
func ParseMCPPreviewConfig(configBytes []byte) (*MCPPreviewConfig, error) {
	var config MCPPreviewConfig
	if err := yaml.Unmarshal(configBytes, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if config.Type == "" {
		return nil, fmt.Errorf("missing required field: type")
	}

	if err := ValidateServerFilters(config.IncludedServers, config.ExcludedServers); err != nil {
		return nil, err
	}

	return &config, nil
}

// PreviewSourceServers loads MCP servers from the source configuration and returns
// preview results showing which servers would be included or excluded.
// If catalogDataBytes is provided, it will be used directly instead of reading from yamlCatalogPath.
// Otherwise, path mode reads yamlCatalogPath through config.Files, confined to
// the server's catalog directories.
func PreviewSourceServers(ctx context.Context, config *MCPPreviewConfig, catalogDataBytes []byte) ([]model.AssetPreviewResult, error) {
	serverNames, err := loadServerNamesFromSource(ctx, config, catalogDataBytes)
	if err != nil {
		return nil, err
	}

	filter, err := NewServerFilter(config.IncludedServers, config.ExcludedServers)
	if err != nil {
		return nil, fmt.Errorf("invalid filter configuration: %w", err)
	}

	results := make([]model.AssetPreviewResult, 0, len(serverNames))
	for _, name := range serverNames {
		included := filter == nil || filter.Allows(name)
		results = append(results, model.AssetPreviewResult{
			Name:     name,
			Included: included,
		})
	}

	return results, nil
}

func loadServerNamesFromSource(ctx context.Context, config *MCPPreviewConfig, catalogDataBytes []byte) ([]string, error) {
	switch config.Type {
	case "yaml":
		return loadYamlServerNames(ctx, config, catalogDataBytes)
	default:
		return nil, fmt.Errorf("unsupported source type for MCP preview: %s", config.Type)
	}
}

func loadYamlServerNames(ctx context.Context, config *MCPPreviewConfig, catalogDataBytes []byte) ([]string, error) {
	var catalogBytes []byte

	if len(catalogDataBytes) > 0 {
		catalogBytes = catalogDataBytes
	} else {
		path, ok := config.Properties[yamlMCPCatalogPathKey].(string)
		if !ok || path == "" {
			return nil, fmt.Errorf("missing required property: %s (provide catalogData, or a yamlCatalogPath inside a catalog source directory)", yamlMCPCatalogPathKey)
		}

		var err error
		catalogBytes, err = config.Files.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}
	}

	var catalog yamlMCPCatalog
	if err := yaml.Unmarshal(catalogBytes, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse catalog file: %w", err)
	}

	names := make([]string, 0, len(catalog.MCPServers))
	for _, s := range catalog.MCPServers {
		names = append(names, s.Name)
	}

	return names, nil
}
