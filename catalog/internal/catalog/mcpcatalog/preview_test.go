package mcpcatalog

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMCPPreviewConfig(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		errorMsg    string
		validate    func(t *testing.T, config *MCPPreviewConfig)
	}{
		{
			name: "valid config with all fields",
			configYAML: `
type: yaml
assetType: mcp_servers
includedServers:
  - "kubernetes*"
  - "prometheus*"
excludedServers:
  - "*-internal"
properties:
  yamlCatalogPath: "/path/to/servers.yaml"
`,
			expectError: false,
			validate: func(t *testing.T, config *MCPPreviewConfig) {
				assert.Equal(t, "yaml", config.Type)
				assert.Equal(t, "mcp_servers", config.AssetType)
				assert.Equal(t, []string{"kubernetes*", "prometheus*"}, config.IncludedServers)
				assert.Equal(t, []string{"*-internal"}, config.ExcludedServers)
				assert.Equal(t, "/path/to/servers.yaml", config.Properties["yamlCatalogPath"])
			},
		},
		{
			name: "valid config with type only",
			configYAML: `
type: yaml
properties:
  yamlCatalogPath: "/path/to/servers.yaml"
`,
			expectError: false,
			validate: func(t *testing.T, config *MCPPreviewConfig) {
				assert.Equal(t, "yaml", config.Type)
				assert.Empty(t, config.IncludedServers)
				assert.Empty(t, config.ExcludedServers)
			},
		},
		{
			name:        "missing type field",
			configYAML:  `includedServers: ["kubernetes*"]`,
			expectError: true,
			errorMsg:    "missing required field: type",
		},
		{
			name: "extra fields from full source config are ignored",
			configYAML: `
name: "My MCP Source"
id: my-mcp-source
type: yaml
enabled: true
includedServers:
  - "kubernetes*"
properties:
  yamlCatalogPath: "/path/to/servers.yaml"
`,
			expectError: false,
			validate: func(t *testing.T, config *MCPPreviewConfig) {
				assert.Equal(t, "yaml", config.Type)
				assert.Equal(t, []string{"kubernetes*"}, config.IncludedServers)
			},
		},
		{
			name: "conflicting patterns - logged but not rejected",
			configYAML: `
type: yaml
includedServers:
  - "kubernetes*"
excludedServers:
  - "kubernetes*"
`,
			expectError: false,
			validate: func(t *testing.T, config *MCPPreviewConfig) {
				assert.Equal(t, "yaml", config.Type)
				assert.Equal(t, []string{"kubernetes*"}, config.IncludedServers)
				assert.Equal(t, []string{"kubernetes*"}, config.ExcludedServers)
			},
		},
		{
			name: "empty pattern",
			configYAML: `
type: yaml
includedServers:
  - ""
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseMCPPreviewConfig([]byte(tt.configYAML))

			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, config)
			if tt.validate != nil {
				tt.validate(t, config)
			}
		})
	}
}

func TestPreviewSourceServers(t *testing.T) {
	ctx := context.Background()

	catalogYAML := `
mcp_servers:
  - name: kubernetes-mcp
    description: "Kubernetes MCP server"
  - name: kubernetes-admin
    description: "Kubernetes admin MCP server"
  - name: prometheus-mcp
    description: "Prometheus MCP server"
  - name: grafana-internal
    description: "Internal Grafana server"
  - name: vault-mcp
    description: "Vault MCP server"
`

	tests := []struct {
		name            string
		config          *MCPPreviewConfig
		catalogData     []byte
		usePath         bool
		expectError     bool
		errorMsg        string
		expectedTotal   int
		expectedInclude int
		expectedExclude int
	}{
		{
			name: "no filters - all servers included",
			config: &MCPPreviewConfig{
				Type: "yaml",
			},
			catalogData:     []byte(catalogYAML),
			expectedTotal:   5,
			expectedInclude: 5,
			expectedExclude: 0,
		},
		{
			name: "include only - kubernetes servers",
			config: &MCPPreviewConfig{
				Type:            "yaml",
				IncludedServers: []string{"kubernetes*"},
			},
			catalogData:     []byte(catalogYAML),
			expectedTotal:   5,
			expectedInclude: 2,
			expectedExclude: 3,
		},
		{
			name: "exclude only - internal servers",
			config: &MCPPreviewConfig{
				Type:            "yaml",
				ExcludedServers: []string{"*-internal"},
			},
			catalogData:     []byte(catalogYAML),
			expectedTotal:   5,
			expectedInclude: 4,
			expectedExclude: 1,
		},
		{
			name: "combined include and exclude - exclusion takes precedence",
			config: &MCPPreviewConfig{
				Type:            "yaml",
				IncludedServers: []string{"kubernetes*"},
				ExcludedServers: []string{"*-admin"},
			},
			catalogData:     []byte(catalogYAML),
			expectedTotal:   5,
			expectedInclude: 1,
			expectedExclude: 4,
		},
		{
			name: "case insensitive matching",
			config: &MCPPreviewConfig{
				Type:            "yaml",
				IncludedServers: []string{"KUBERNETES*"},
			},
			catalogData:     []byte(catalogYAML),
			expectedTotal:   5,
			expectedInclude: 2,
			expectedExclude: 3,
		},
		{
			name: "multiple include patterns",
			config: &MCPPreviewConfig{
				Type:            "yaml",
				IncludedServers: []string{"kubernetes*", "prometheus*"},
			},
			catalogData:     []byte(catalogYAML),
			expectedTotal:   5,
			expectedInclude: 3,
			expectedExclude: 2,
		},
		{
			name: "wildcard in middle",
			config: &MCPPreviewConfig{
				Type:            "yaml",
				IncludedServers: []string{"*-mcp"},
			},
			catalogData:     []byte(catalogYAML),
			expectedTotal:   5,
			expectedInclude: 3,
			expectedExclude: 2,
		},
		{
			name: "path mode with yamlCatalogPath",
			config: &MCPPreviewConfig{
				Type: "yaml",
			},
			usePath:         true,
			expectedTotal:   5,
			expectedInclude: 5,
			expectedExclude: 0,
		},
		{
			name: "stateless mode takes precedence over path",
			config: &MCPPreviewConfig{
				Type: "yaml",
			},
			catalogData: []byte(`
mcp_servers:
  - name: only-one-server
    description: "Single server"
`),
			usePath:         true,
			expectedTotal:   1,
			expectedInclude: 1,
			expectedExclude: 0,
		},
		{
			name: "unsupported source type",
			config: &MCPPreviewConfig{
				Type: "hf",
			},
			catalogData: []byte(catalogYAML),
			expectError: true,
			errorMsg:    "unsupported source type for MCP preview",
		},
		{
			name: "missing yamlCatalogPath with no catalog data",
			config: &MCPPreviewConfig{
				Type: "yaml",
			},
			expectError: true,
			errorMsg:    "missing required property",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalogData := tt.catalogData

			if tt.usePath {
				tmpDir := t.TempDir()
				catalogFile := filepath.Join(tmpDir, "servers.yaml")
				err := os.WriteFile(catalogFile, []byte(catalogYAML), 0644)
				require.NoError(t, err)

				if tt.config.Properties == nil {
					tt.config.Properties = make(map[string]any)
				}
				tt.config.Properties["yamlCatalogPath"] = catalogFile
			}

			results, err := PreviewSourceServers(ctx, tt.config, catalogData)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				return
			}

			require.NoError(t, err)
			assert.Len(t, results, tt.expectedTotal)

			var included, excluded int
			for _, r := range results {
				if r.Included {
					included++
				} else {
					excluded++
				}
			}
			assert.Equal(t, tt.expectedInclude, included, "included count")
			assert.Equal(t, tt.expectedExclude, excluded, "excluded count")
		})
	}
}

func TestPreviewSourceServersNames(t *testing.T) {
	ctx := context.Background()

	catalogData := []byte(`
mcp_servers:
  - name: kubernetes-mcp
    description: "Kubernetes server"
  - name: prometheus-mcp
    description: "Prometheus server"
  - name: vault-internal
    description: "Internal Vault"
`)

	config := &MCPPreviewConfig{
		Type:            "yaml",
		IncludedServers: []string{"*-mcp"},
		ExcludedServers: []string{"*-internal"},
	}

	results, err := PreviewSourceServers(ctx, config, catalogData)
	require.NoError(t, err)
	require.Len(t, results, 3)

	assert.Equal(t, "kubernetes-mcp", results[0].Name)
	assert.True(t, results[0].Included)

	assert.Equal(t, "prometheus-mcp", results[1].Name)
	assert.True(t, results[1].Included)

	assert.Equal(t, "vault-internal", results[2].Name)
	assert.False(t, results[2].Included)
}
