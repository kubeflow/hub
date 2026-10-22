package modelcatalog

import (
	"context"
	"fmt"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	model "github.com/kubeflow/hub/catalog/pkg/openapi"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// PreviewConfig represents the parsed preview request configuration.
type PreviewConfig struct {
	Type           string         `json:"type" yaml:"type"`
	IncludedModels []string       `json:"includedModels,omitempty" yaml:"includedModels,omitempty"`
	ExcludedModels []string       `json:"excludedModels,omitempty" yaml:"excludedModels,omitempty"`
	Properties     map[string]any `json:"properties,omitempty" yaml:"properties,omitempty"`

	// Files reads yamlCatalogPath files in path mode, confined to the server's
	// catalog directories. It is set by the server, never parsed from the
	// uploaded config.
	Files *basecatalog.PreviewFiles `json:"-" yaml:"-"`
}

// ParsePreviewConfig parses the uploaded config bytes into a PreviewConfig.
// Extra fields (like name, id, enabled) are ignored so users can paste
// a full source config entry directly for preview.
func ParsePreviewConfig(configBytes []byte) (*PreviewConfig, error) {
	var config PreviewConfig
	if err := yaml.Unmarshal(configBytes, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if config.Type == "" {
		return nil, fmt.Errorf("missing required field: type")
	}

	// Validate filter patterns early
	if err := ValidateSourceFilters(config.IncludedModels, config.ExcludedModels); err != nil {
		return nil, err
	}

	return &config, nil
}

// PreviewSourceModels loads models from the source configuration and returns
// preview results showing which models would be included or excluded.
// If catalogDataBytes is provided, it will be used directly instead of reading from yamlCatalogPath.
// Otherwise, path mode reads yamlCatalogPath through config.Files, confined to
// the server's catalog directories.
func PreviewSourceModels(ctx context.Context, config *PreviewConfig, catalogDataBytes []byte) ([]model.ModelPreviewResult, error) {
	switch config.Type {
	case "hf", "huggingface":
		return previewHFModels(ctx, config)
	default:
		return previewModelsByName(ctx, config, catalogDataBytes)
	}
}

// previewModelsByName is the name-only preview path used for YAML and other
// non-HF sources where no extra metadata is available.
func previewModelsByName(ctx context.Context, config *PreviewConfig, catalogDataBytes []byte) ([]model.ModelPreviewResult, error) {
	modelNames, err := loadModelNamesFromSource(ctx, config, catalogDataBytes)
	if err != nil {
		return nil, err
	}

	filter, err := NewModelFilter(config.IncludedModels, config.ExcludedModels)
	if err != nil {
		return nil, fmt.Errorf("invalid filter configuration: %w", err)
	}

	results := make([]model.ModelPreviewResult, 0, len(modelNames))
	for _, name := range modelNames {
		included := filter == nil || filter.Allows(name)
		results = append(results, model.ModelPreviewResult{
			Name:     name,
			Included: included,
		})
	}

	return results, nil
}

// previewHFModels fetches enriched model data from the HuggingFace API
// and populates HfAccessType / HfGatedAccessGranted on each result.
func previewHFModels(ctx context.Context, config *PreviewConfig) ([]model.ModelPreviewResult, error) {
	hfModels, err := loadHFModels(ctx, config)
	if err != nil {
		return nil, err
	}

	filter, err := NewModelFilter(config.IncludedModels, config.ExcludedModels)
	if err != nil {
		return nil, fmt.Errorf("invalid filter configuration: %w", err)
	}

	results := make([]model.ModelPreviewResult, 0, len(hfModels))
	for _, m := range hfModels {
		included := filter == nil || filter.Allows(m.Name)
		r := model.ModelPreviewResult{
			Name:     m.Name,
			Included: included,
		}
		if m.AccessType != "" {
			r.HfAccessType = &m.AccessType
		}
		r.HfGatedAccessGranted = m.GatedAccessGranted
		results = append(results, r)
	}

	return results, nil
}

// loadModelNamesFromSource loads model names from the specified source type.
// If catalogDataBytes is provided, it takes precedence over reading from file paths.
func loadModelNamesFromSource(ctx context.Context, config *PreviewConfig, catalogDataBytes []byte) ([]string, error) {
	switch config.Type {
	case "yaml":
		return loadYamlModelNames(ctx, config, catalogDataBytes)
	default:
		return nil, fmt.Errorf("unsupported source type: %s", config.Type)
	}
}

// loadHFModels fetches enriched model info from the HuggingFace API for preview.
func loadHFModels(ctx context.Context, config *PreviewConfig) ([]hfPreviewModelResult, error) {
	if config.Properties == nil {
		config.Properties = make(map[string]any)
	}

	provider, err := NewHFPreviewProvider(config)
	if err != nil {
		return nil, err
	}

	return provider.FetchModelsForPreview(ctx, config.IncludedModels)
}

// loadYamlModelNames loads model names from a YAML catalog.
// If catalogDataBytes is provided (stateless mode), it is used directly.
// Otherwise, the catalog is read from yamlCatalogPath (path mode) through
// config.Files, which confines reads to the server's catalog directories.
func loadYamlModelNames(ctx context.Context, config *PreviewConfig, catalogDataBytes []byte) ([]string, error) {
	var catalogBytes []byte

	if len(catalogDataBytes) > 0 {
		// Stateless mode: use provided catalog data directly
		catalogBytes = catalogDataBytes
	} else {
		// Path mode: read from yamlCatalogPath
		path, ok := config.Properties[yamlCatalogPathKey].(string)
		if !ok || path == "" {
			return nil, fmt.Errorf("missing required property: %s (provide catalogData, or a yamlCatalogPath inside a catalog source directory)", yamlCatalogPathKey)
		}

		var err error
		catalogBytes, err = config.Files.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}
	}

	var catalog yamlCatalog
	if err := yaml.UnmarshalStrict(catalogBytes, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse catalog file: %w", err)
	}

	// Extract model names
	names := make([]string, 0, len(catalog.Models))
	for _, m := range catalog.Models {
		names = append(names, m.Name)
	}

	return names, nil
}
