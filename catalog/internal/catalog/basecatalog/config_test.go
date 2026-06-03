package basecatalog

import (
	"encoding/json"
	"testing"

	apimodels "github.com/kubeflow/hub/catalog/pkg/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSourceConfig_GetModelCatalogs(t *testing.T) {
	tests := []struct {
		name     string
		config   *SourceConfig
		expected int
	}{
		{
			name: "only model_catalogs",
			config: &SourceConfig{
				ModelCatalogs: []ModelSource{
					{CatalogSource: apimodels.CatalogSource{Id: "model1", Name: "Model 1"}},
					{CatalogSource: apimodels.CatalogSource{Id: "model2", Name: "Model 2"}},
				},
			},
			expected: 2,
		},
		{
			name: "only deprecated catalogs",
			config: &SourceConfig{
				Catalogs: []ModelSource{
					{CatalogSource: apimodels.CatalogSource{Id: "cat1", Name: "Catalog 1"}},
				},
			},
			expected: 1,
		},
		{
			name: "both fields - model_catalogs takes precedence on conflict",
			config: &SourceConfig{
				ModelCatalogs: []ModelSource{
					{CatalogSource: apimodels.CatalogSource{Id: "shared", Name: "New Name"}},
				},
				Catalogs: []ModelSource{
					{CatalogSource: apimodels.CatalogSource{Id: "shared", Name: "Old Name"}},
					{CatalogSource: apimodels.CatalogSource{Id: "unique", Name: "Unique"}},
				},
			},
			expected: 2, // "shared" from model_catalogs + "unique" from catalogs
		},
		{
			name:     "empty config",
			config:   &SourceConfig{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetModelCatalogs()
			assert.Len(t, result, tt.expected)

			// For the precedence test, verify model_catalogs wins
			if tt.name == "both fields - model_catalogs takes precedence on conflict" {
				var sharedSource *ModelSource
				for _, s := range result {
					if s.Id == "shared" {
						sharedSource = &s
						break
					}
				}
				assert.NotNil(t, sharedSource)
				assert.Equal(t, "New Name", sharedSource.Name)
			}
		})
	}
}

func TestSourceConfig_HasDeprecatedCatalogs(t *testing.T) {
	tests := []struct {
		name     string
		config   *SourceConfig
		expected bool
	}{
		{
			name: "has deprecated catalogs",
			config: &SourceConfig{
				Catalogs: []ModelSource{
					{CatalogSource: apimodels.CatalogSource{Id: "cat1", Name: "Catalog 1"}},
				},
			},
			expected: true,
		},
		{
			name: "no deprecated catalogs",
			config: &SourceConfig{
				ModelCatalogs: []ModelSource{
					{CatalogSource: apimodels.CatalogSource{Id: "model1", Name: "Model 1"}},
				},
			},
			expected: false,
		},
		{
			name:     "empty config",
			config:   &SourceConfig{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.HasDeprecatedCatalogs()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSourceConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    *SourceConfig
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid config with model catalogs",
			config: &SourceConfig{
				ModelCatalogs: []ModelSource{
					{CatalogSource: apimodels.CatalogSource{Id: "model1", Name: "Model 1"}},
				},
			},
			expectErr: false,
		},
		{
			name: "valid config with MCP catalogs",
			config: &SourceConfig{
				MCPCatalogs: []MCPSource{
					{ID: "mcp1", Name: "MCP 1", Type: "yaml"},
				},
			},
			expectErr: false,
		},
		{
			name: "duplicate model catalog IDs",
			config: &SourceConfig{
				ModelCatalogs: []ModelSource{
					{CatalogSource: apimodels.CatalogSource{Id: "dup", Name: "Model 1"}},
					{CatalogSource: apimodels.CatalogSource{Id: "dup", Name: "Model 2"}},
				},
			},
			expectErr: true,
			errMsg:    "duplicate model catalog id: dup",
		},
		{
			name: "duplicate MCP catalog IDs",
			config: &SourceConfig{
				MCPCatalogs: []MCPSource{
					{ID: "dup", Name: "MCP 1", Type: "yaml"},
					{ID: "dup", Name: "MCP 2", Type: "yaml"},
				},
			},
			expectErr: true,
			errMsg:    "duplicate mcp catalog id: dup",
		},
		{
			name: "missing model catalog ID",
			config: &SourceConfig{
				ModelCatalogs: []ModelSource{
					{CatalogSource: apimodels.CatalogSource{Name: "Model 1"}},
				},
			},
			expectErr: true,
			errMsg:    "model catalog source missing id",
		},
		{
			name: "missing MCP catalog ID",
			config: &SourceConfig{
				MCPCatalogs: []MCPSource{
					{Name: "MCP 1", Type: "yaml"},
				},
			},
			expectErr: true,
			errMsg:    "mcp catalog source missing id",
		},
		{
			name: "cross-type ID collision between model and MCP catalogs",
			config: &SourceConfig{
				ModelCatalogs: []ModelSource{
					{CatalogSource: apimodels.CatalogSource{Id: "shared", Name: "Model 1"}},
				},
				MCPCatalogs: []MCPSource{
					{ID: "shared", Name: "MCP 1", Type: "yaml"},
				},
			},
			expectErr: true,
			errMsg:    `id "shared" used in multiple catalog types`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr {
				if assert.Error(t, err) {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNamedQuery_UnmarshalJSON(t *testing.T) {
	t.Run("legacy flat format", func(t *testing.T) {
		data := []byte(`{"fieldA":{"operator":"=","value":"x"},"fieldB":{"operator":"<=","value":42}}`)
		var nq NamedQuery
		require.NoError(t, json.Unmarshal(data, &nq))
		assert.Equal(t, "", nq.AssetType)
		assert.Equal(t, "=", nq.Filters["fieldA"].Operator)
		assert.Equal(t, "x", nq.Filters["fieldA"].Value)
		assert.Equal(t, "<=", nq.Filters["fieldB"].Operator)
		assert.Equal(t, float64(42), nq.Filters["fieldB"].Value)
	})

	t.Run("new format with assetType and filters", func(t *testing.T) {
		data := []byte(`{"assetType":"mcp_servers","filters":{"verified":{"operator":"=","value":true}}}`)
		var nq NamedQuery
		require.NoError(t, json.Unmarshal(data, &nq))
		assert.Equal(t, AssetTypeMCPServers, nq.AssetType)
		assert.Equal(t, "=", nq.Filters["verified"].Operator)
		assert.Equal(t, true, nq.Filters["verified"].Value)
	})

	t.Run("new format without assetType", func(t *testing.T) {
		data := []byte(`{"filters":{"field":{"operator":">","value":0}}}`)
		var nq NamedQuery
		require.NoError(t, json.Unmarshal(data, &nq))
		assert.Equal(t, "", nq.AssetType)
		assert.Equal(t, ">", nq.Filters["field"].Operator)
	})

	t.Run("legacy format with field named 'filters' is interpreted as new format", func(t *testing.T) {
		// Note: a legacy-format query with a field literally named "filters" triggers the
		// new-format detection path because we use the presence of a "filters" key as the
		// heuristic. In practice, "filters" is not a valid model metadata field name, so
		// this edge case should never occur in real configurations.
		data := []byte(`{"filters":{"operator":"=","value":"x"}}`)
		var nq NamedQuery
		// The "filters" key triggers new-format parsing, but its value {"operator":"=","value":"x"}
		// cannot be decoded as map[string]FieldFilter, so unmarshaling fails.
		err := json.Unmarshal(data, &nq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid filters")
	})
}

func TestFilterNamedQueriesByAssetType(t *testing.T) {
	queries := map[string]NamedQuery{
		"models-query": {
			AssetType: AssetTypeModels,
			Filters:   map[string]FieldFilter{"f": {Operator: "=", Value: "v"}},
		},
		"mcp-query": {
			AssetType: AssetTypeMCPServers,
			Filters:   map[string]FieldFilter{"f": {Operator: "=", Value: "v"}},
		},
		"default-query": {
			// No AssetType — should default to AssetTypeModels
			Filters: map[string]FieldFilter{"f": {Operator: "=", Value: "v"}},
		},
	}

	t.Run("filter for models includes default", func(t *testing.T) {
		result := FilterNamedQueriesByAssetType(queries, AssetTypeModels)
		assert.Len(t, result, 2)
		assert.Contains(t, result, "models-query")
		assert.Contains(t, result, "default-query")
		assert.NotContains(t, result, "mcp-query")
	})

	t.Run("filter for mcp_servers excludes default", func(t *testing.T) {
		result := FilterNamedQueriesByAssetType(queries, AssetTypeMCPServers)
		assert.Len(t, result, 1)
		assert.Contains(t, result, "mcp-query")
		assert.NotContains(t, result, "models-query")
		assert.NotContains(t, result, "default-query")
	})

	t.Run("empty input returns empty map", func(t *testing.T) {
		result := FilterNamedQueriesByAssetType(map[string]NamedQuery{}, AssetTypeModels)
		assert.Empty(t, result)
	})

	t.Run("filters are the same map reference", func(t *testing.T) {
		result := FilterNamedQueriesByAssetType(queries, AssetTypeModels)
		assert.Equal(t, queries["models-query"].Filters, result["models-query"])
	})
}
