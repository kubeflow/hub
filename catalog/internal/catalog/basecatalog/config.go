package basecatalog

import (
	"fmt"
	"os"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// ReadSourceConfig reads, parses, and validates a sources configuration file.
func ReadSourceConfig(path string) (*SourceConfig, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := &SourceConfig{}
	if err = yaml.UnmarshalStrict(bytes, config); err != nil {
		return nil, err
	}

	if config.HasDeprecatedCatalogs() {
		glog.Warningf("Configuration file %s uses deprecated 'catalogs' field. Please rename to 'model_catalogs'.", path)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration in %s: %w", path, err)
	}

	return config, nil
}

// SourceConfig represents the configuration format for model catalogs.
//
// Example:
//
//	model_catalogs:
//	  - name: "Organization AI Models"
//	    id: organization_ai_models
//	    type: yaml
//	    enabled: true
//	    properties:
//	      yamlCatalogPath: dev-organization-models.yaml
//	    labels:
//	      - Organization AI
//
//	# DEPRECATED: Use model_catalogs instead
//	# catalogs: []
type SourceConfig struct {
	// ModelCatalogs contains model catalog source definitions
	ModelCatalogs []ModelSource `yaml:"model_catalogs,omitempty" json:"model_catalogs,omitempty"`

	// MCPCatalogs contains MCP catalog source definitions
	MCPCatalogs []MCPSource `yaml:"mcp_catalogs,omitempty" json:"mcp_catalogs,omitempty"`

	// Labels contains label definitions for the catalogs
	Labels []map[string]any `yaml:"labels,omitempty" json:"labels,omitempty"`

	// NamedQueries contains predefined query filters, optionally scoped by asset type.
	// See NamedQuery for the supported YAML formats.
	NamedQueries map[string]NamedQuery `yaml:"namedQueries,omitempty" json:"namedQueries,omitempty"`

	// DEPRECATED: Use ModelCatalogs instead
	// This field is maintained for backwards compatibility
	Catalogs []ModelSource `yaml:"catalogs,omitempty" json:"catalogs,omitempty"`
}

// GetModelCatalogs returns the merged list of model catalogs, combining the
// new model_catalogs field with the deprecated catalogs field.
// If there are ID conflicts, model_catalogs takes precedence.
func (c *SourceConfig) GetModelCatalogs() []ModelSource {
	if len(c.Catalogs) == 0 {
		return c.ModelCatalogs
	}

	if len(c.ModelCatalogs) == 0 {
		return c.Catalogs
	}

	// Both fields have values. Concatenate the two lists (with
	// ModelCatalogs coming before Catalogs), and remove duplicate entries
	// from Catalogs.

	merged := make([]ModelSource, len(c.ModelCatalogs), len(c.ModelCatalogs)+len(c.Catalogs))
	copy(merged, c.ModelCatalogs)

	mcIDs := make(map[string]struct{}, len(merged))
	for _, catalog := range merged {
		mcIDs[catalog.GetId()] = struct{}{}
	}

	for _, catalog := range c.Catalogs {
		if _, exists := mcIDs[catalog.GetId()]; !exists {
			merged = append(merged, catalog)
		}
	}

	return merged
}

// HasDeprecatedCatalogs returns true if the deprecated "catalogs" field is being used
func (c *SourceConfig) HasDeprecatedCatalogs() bool {
	return len(c.Catalogs) > 0
}

// sourceIdentifiable is satisfied by any source type that has an ID.
type sourceIdentifiable interface {
	GetId() string
}

// validateSourceIDs checks for empty and duplicate IDs within a source section,
// and cross-section collisions against globalSeen.
func validateSourceIDs[T sourceIdentifiable](section string, sources []T, globalSeen map[string]bool) error {
	local := make(map[string]bool, len(sources))
	for _, s := range sources {
		id := s.GetId()
		if id == "" {
			return fmt.Errorf("%s catalog source missing id", section)
		}
		if local[id] {
			return fmt.Errorf("duplicate %s catalog id: %s", section, id)
		}
		if globalSeen[id] {
			return fmt.Errorf("id %q used in multiple catalog types", id)
		}
		local[id] = true
		globalSeen[id] = true
	}
	return nil
}

// Validate checks the configuration for common errors
func (c *SourceConfig) Validate() error {
	seen := make(map[string]bool)

	if err := validateSourceIDs("model", c.GetModelCatalogs(), seen); err != nil {
		return err
	}
	if err := validateSourceIDs("mcp", c.MCPCatalogs, seen); err != nil {
		return err
	}

	if err := ValidateNamedQueries(c.NamedQueries); err != nil {
		return fmt.Errorf("invalid named queries: %w", err)
	}

	return nil
}
