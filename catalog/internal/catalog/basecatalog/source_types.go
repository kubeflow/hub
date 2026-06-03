package basecatalog

import (
	"encoding/json"
	"fmt"

	apimodels "github.com/kubeflow/hub/catalog/pkg/openapi"
)

// Source status constants matching the OpenAPI enum values.
const (
	SourceStatusAvailable          = "available"
	SourceStatusPartiallyAvailable = "partially-available"
	SourceStatusError              = "error"
	SourceStatusDisabled           = "disabled"
)

// Asset type constants for named query scoping.
const (
	AssetTypeModels     = "models"
	AssetTypeMCPServers = "mcp_servers"
)

// CommonSourceFields holds the fields shared between ModelSource and MCPSource
// for field-level merge operations.
type CommonSourceFields struct {
	Name       string
	Enabled    *bool
	Labels     []string
	Type       string
	Properties map[string]any
	Origin     string
	AssetType  *apimodels.CatalogAssetType
}

// MergeCommonSourceFields merges two CommonSourceFields structs with field-level override semantics.
// Fields from 'override' take precedence over 'base' when explicitly set (non-zero/non-nil).
func MergeCommonSourceFields(base, override CommonSourceFields) CommonSourceFields {
	result := base
	if override.Name != "" {
		result.Name = override.Name
	}
	if override.Enabled != nil {
		result.Enabled = override.Enabled
	}
	if override.Labels != nil {
		result.Labels = override.Labels
	}
	if override.Type != "" {
		result.Type = override.Type
	}
	if override.Properties != nil {
		result.Properties = override.Properties
	}
	// Origin follows Properties: use override's origin only when Properties are overridden,
	// so relative paths resolve relative to where they were defined.
	if override.Properties != nil && override.Origin != "" {
		result.Origin = override.Origin
	}
	if override.AssetType != nil {
		result.AssetType = override.AssetType
	}
	return result
}

// FieldFilter represents a single field filter within a named query
type FieldFilter struct {
	Operator string `json:"operator" yaml:"operator"`
	Value    any    `json:"value" yaml:"value"`
}

// NamedQuery represents a named query with an optional asset type and its field filters.
// The AssetType field scopes the query to a specific entity type (AssetTypeModels or AssetTypeMCPServers).
// If AssetType is empty, it defaults to AssetTypeModels.
//
// Both the legacy format (flat map of field name → filter) and the new format
// (explicit "assetType" and "filters" keys) are supported during deserialization.
type NamedQuery struct {
	AssetType string                 `json:"assetType,omitempty" yaml:"assetType,omitempty"`
	Filters   map[string]FieldFilter `json:"filters" yaml:"filters"`
}

// UnmarshalJSON supports both the legacy flat format and the new structured format.
//
// Legacy format (field names as keys, defaults to assetType "models"):
//
//	{"fieldName": {"operator": "=", "value": "x"}}
//
// New format (explicit assetType and filters):
//
//	{"assetType": "models", "filters": {"fieldName": {"operator": "=", "value": "x"}}}
//
// Note: format detection uses the presence of a "filters" key. A legacy-format query
// with a field literally named "filters" would be interpreted as the new structured
// format. This is an intentional trade-off since "filters" is not a valid model
// metadata field name in practice.
func (nq *NamedQuery) UnmarshalJSON(data []byte) error {
	// Try to detect the new format by checking for a "filters" key.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if filtersRaw, hasFilters := raw["filters"]; hasFilters {
		// New format: parse assetType and filters explicitly.
		if assetTypeRaw, ok := raw["assetType"]; ok {
			if err := json.Unmarshal(assetTypeRaw, &nq.AssetType); err != nil {
				return fmt.Errorf("invalid assetType: %w", err)
			}
		}
		if err := json.Unmarshal(filtersRaw, &nq.Filters); err != nil {
			return fmt.Errorf("invalid filters: %w", err)
		}
		return nil
	}

	// Legacy format: treat all keys as field name → FieldFilter.
	filters := make(map[string]FieldFilter, len(raw))
	for key, val := range raw {
		var ff FieldFilter
		if err := json.Unmarshal(val, &ff); err != nil {
			return fmt.Errorf("invalid filter for field %q: %w", key, err)
		}
		filters[key] = ff
	}
	nq.Filters = filters
	return nil
}

// FilterNamedQueriesByAssetType returns only the named queries that match the given
// target asset type, stripping the NamedQuery wrapper to return the raw filter maps.
// Queries with an empty AssetType default to AssetTypeModels.
func FilterNamedQueriesByAssetType(queries map[string]NamedQuery, target string) map[string]map[string]FieldFilter {
	result := make(map[string]map[string]FieldFilter)
	for name, nq := range queries {
		effectiveType := nq.AssetType
		if effectiveType == "" {
			effectiveType = AssetTypeModels
		}
		if effectiveType == target {
			result[name] = nq.Filters
		}
	}
	return result
}

// ModelSource is a single entry from the catalog sources YAML file.
type ModelSource struct {
	apimodels.CatalogSource `json:",inline"`

	// Catalog type to use, must match one of the registered types
	Type string `json:"type"`

	// Properties used for configuring the catalog connection based on catalog implementation
	Properties map[string]any `json:"properties,omitempty"`

	// Origin is the absolute path of the config file this source was loaded from.
	// This is set automatically during loading and used for resolving relative paths.
	// It is not read from YAML; it's set programmatically.
	Origin string `json:"-" yaml:"-"`
}

// GetId returns the ID of the source
func (s ModelSource) GetId() string {
	return s.Id
}

// MCPSource represents a source of MCP servers
type MCPSource struct {
	Name       string                      `json:"name" yaml:"name"`
	ID         string                      `json:"id" yaml:"id"`
	Type       string                      `json:"type" yaml:"type"`
	Enabled    *bool                       `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Properties map[string]any              `json:"properties" yaml:"properties"`
	Labels     []string                    `json:"labels" yaml:"labels"`
	AssetType  *apimodels.CatalogAssetType `json:"assetType,omitempty" yaml:"assetType,omitempty"`

	IncludedServers []string `json:"includedServers,omitempty" yaml:"includedServers,omitempty"`
	ExcludedServers []string `json:"excludedServers,omitempty" yaml:"excludedServers,omitempty"`

	// Origin is the absolute path of the config file this source was loaded from.
	// This is set automatically during loading and used for resolving relative paths.
	// It is not read from YAML; it's set programmatically.
	Origin string `json:"-" yaml:"-"`
}

// GetId returns the ID of the source
func (s MCPSource) GetId() string {
	return s.ID
}

// IsEnabled returns true if the source is enabled.
// If Enabled is nil, the source is considered enabled by default.
func (s *MCPSource) IsEnabled() bool {
	return s.Enabled == nil || *s.Enabled
}

// PluginSource is the generic source type for catalog plugins.
// New plugins should use this instead of defining a custom source struct.
type PluginSource struct {
	Name       string         `json:"name" yaml:"name"`
	ID         string         `json:"id" yaml:"id"`
	Type       string         `json:"type" yaml:"type"`
	Enabled    *bool          `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Properties map[string]any `json:"properties" yaml:"properties"`
	Labels     []string       `json:"labels" yaml:"labels"`

	// Origin is the absolute path of the config file this source was loaded from.
	Origin string `json:"-" yaml:"-"`
}

// GetId returns the ID of the source.
func (s PluginSource) GetId() string {
	return s.ID
}

// IsEnabled returns true if the source is enabled.
func (s PluginSource) IsEnabled() bool {
	return s.Enabled == nil || *s.Enabled
}
