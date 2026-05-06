package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_ModelCatalogs(t *testing.T) {
	yamlData := `
model_catalogs:
  - name: "Organization AI Models"
    id: organization_ai_models
    type: yaml
    enabled: true
    includedModels:
      - "Granite/*"
    excludedModels:
      - "Granite/legacy-merge-beta"
    labels:
      - "test"
    properties:
      yamlCatalogPath: sample-catalog.yaml
`

	configPath := writeTemp(t, yamlData)
	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)

	models := cfg.GetModelCatalogs()
	require.Equal(t, 1, len(models))

	src := models[0]
	assert.Equal(t, "organization_ai_models", src.GetId())
	assert.Equal(t, "Organization AI Models", src.Name)
	assert.Equal(t, "yaml", src.Type)
	assert.True(t, *src.Enabled)
	assert.Equal(t, []string{"Granite/*"}, src.IncludedModels)
	assert.Equal(t, []string{"Granite/legacy-merge-beta"}, src.ExcludedModels)
	assert.Equal(t, []string{"test"}, src.Labels)
	assert.Equal(t, "sample-catalog.yaml", src.Properties["yamlCatalogPath"])
}

func TestLoadConfig_DeprecatedCatalogsList(t *testing.T) {
	yamlData := `
catalogs:
  - name: Sample Catalog
    id: sample_custom_catalog
    type: yaml
    enabled: true
    properties:
      yamlCatalogPath: sample-catalog.yaml
`

	configPath := writeTemp(t, yamlData)
	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)

	models := cfg.GetModelCatalogs()
	require.Equal(t, 1, len(models))
	assert.Equal(t, "sample_custom_catalog", models[0].GetId())
	assert.True(t, cfg.HasDeprecatedCatalogs())
}

func TestLoadConfig_EmptyCatalogsList(t *testing.T) {
	yamlData := `catalogs: []`

	configPath := writeTemp(t, yamlData)
	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)

	assert.Empty(t, cfg.GetModelCatalogs())
	assert.Empty(t, cfg.MCPCatalogs)
}

func TestLoadConfig_MCPCatalogs(t *testing.T) {
	yamlData := `
mcp_catalogs:
  - name: "MCP Server"
    id: mcp_server_1
    type: mcp_server
    enabled: true
    includedServers:
      - "tool-a"
    excludedServers:
      - "tool-b"
    properties:
      endpoint: "http://localhost:8080"
`

	configPath := writeTemp(t, yamlData)
	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)

	require.Equal(t, 1, len(cfg.MCPCatalogs))

	src := cfg.MCPCatalogs[0]
	assert.Equal(t, "mcp_server_1", src.GetId())
	assert.Equal(t, []string{"tool-a"}, src.IncludedServers)
	assert.Equal(t, []string{"tool-b"}, src.ExcludedServers)
}

func TestLoadConfig_Mixed(t *testing.T) {
	yamlData := `
model_catalogs:
  - name: "Models"
    id: models_1
    type: yaml
    properties:
      yamlCatalogPath: models.yaml
mcp_catalogs:
  - name: "MCP"
    id: mcp_1
    type: mcp_server
    properties:
      endpoint: "http://localhost:8080"
labels:
  - name: "production"
    color: "green"
`

	configPath := writeTemp(t, yamlData)
	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)

	assert.Equal(t, 1, len(cfg.GetModelCatalogs()))
	assert.Equal(t, 1, len(cfg.MCPCatalogs))
	assert.Equal(t, 1, len(cfg.Labels))
	assert.Equal(t, "production", cfg.Labels[0]["name"])
}

func TestLoadConfig_DeprecatedMergedWithModelCatalogs(t *testing.T) {
	yamlData := `
model_catalogs:
  - name: "Primary"
    id: primary
    type: yaml
    properties:
      yamlCatalogPath: primary.yaml
  - name: "Shared"
    id: shared
    type: yaml
    properties:
      yamlCatalogPath: primary-shared.yaml
catalogs:
  - name: "Legacy Only"
    id: legacy_only
    type: yaml
    properties:
      yamlCatalogPath: legacy.yaml
  - name: "Shared Legacy"
    id: shared
    type: yaml
    properties:
      yamlCatalogPath: legacy-shared.yaml
`

	configPath := writeTemp(t, yamlData)
	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)

	models := cfg.GetModelCatalogs()
	// model_catalogs (2) + unique from deprecated (1) = 3
	require.Equal(t, 3, len(models))

	byID := make(map[string]string)
	for _, m := range models {
		byID[m.GetId()] = m.Properties["yamlCatalogPath"].(string)
	}

	// model_catalogs entry wins on ID conflict
	assert.Equal(t, "primary-shared.yaml", byID["shared"])

	// unique deprecated entry preserved
	assert.Equal(t, "legacy.yaml", byID["legacy_only"])
}

func TestLoadConfig_MCPNamedQueriesWithEmptySources(t *testing.T) {
	yamlData := `
mcp_catalogs: []
namedQueries:
  production_ready:
    assetType: mcp_servers
    filters:
      verifiedSource:
        operator: "="
        value: true
`

	configPath := writeTemp(t, yamlData)
	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)

	require.Contains(t, cfg.NamedQueries, "production_ready")
	assert.Equal(t, "mcp_servers", cfg.NamedQueries["production_ready"].AssetType)
	assert.Equal(t, "=", cfg.NamedQueries["production_ready"].Filters["verifiedSource"].Operator)
}

func TestLoadConfigs(t *testing.T) {
	base := writeTemp(t, `
model_catalogs:
  - name: "Base"
    id: base-source
    type: yaml
`)

	override := writeTemp(t, `
model_catalogs:
  - name: "Override"
    id: override-source
    type: hf
`)

	configs, err := LoadConfigs([]string{base, override})
	require.NoError(t, err)
	require.Equal(t, 2, len(configs))

	assert.Equal(t, 1, len(configs[0].GetModelCatalogs()))
	assert.Equal(t, "base-source", configs[0].GetModelCatalogs()[0].GetId())

	assert.Equal(t, 1, len(configs[1].GetModelCatalogs()))
	assert.Equal(t, "override-source", configs[1].GetModelCatalogs()[0].GetId())
}

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "sources.yaml")
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
	return path
}
