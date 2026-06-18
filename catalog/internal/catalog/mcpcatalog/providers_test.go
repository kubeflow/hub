package mcpcatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	apimodels "github.com/kubeflow/hub/catalog/pkg/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYamlMCPServerConversion(t *testing.T) {
	yamlServer := &yamlMCPServer{
		Name:        "test-server",
		DisplayName: new("Test Server"),
		Description: new("Test MCP server"),
		Provider:    new("Test Provider"),
		Version:     new("1.0.0"),
		License:     new("MIT"),
		Tools: []*yamlMCPTool{
			{Name: "tool1", Description: new("First tool")},
			{Name: "tool2", Description: new("Second tool")},
		},
	}

	record := yamlServer.ToMCPServerProviderRecord()

	assert.NotNil(t, record.Server)
	assert.Nil(t, record.Error)
	assert.Equal(t, 2, len(record.Tools))
	assert.Equal(t, "tool1", record.Tools[0].Name)
	assert.Equal(t, "tool2", record.Tools[1].Name)

	attrs := record.Server.GetAttributes()
	require.NotNil(t, attrs)
	assert.Equal(t, "test-server", *attrs.Name)

	props := record.Server.GetProperties()
	require.NotNil(t, props)

	// Check that basic properties are set
	foundDisplayName := false
	foundDescription := false
	foundProvider := false
	for _, prop := range *props {
		if prop.Name == "displayName" && prop.StringValue != nil {
			assert.Equal(t, "Test Server", *prop.StringValue)
			foundDisplayName = true
		}
		if prop.Name == "description" && prop.StringValue != nil {
			assert.Equal(t, "Test MCP server", *prop.StringValue)
			foundDescription = true
		}
		if prop.Name == "provider" && prop.StringValue != nil {
			assert.Equal(t, "Test Provider", *prop.StringValue)
			foundProvider = true
		}
	}
	assert.True(t, foundDisplayName, "displayName property should be set")
	assert.True(t, foundDescription, "description property should be set")
	assert.True(t, foundProvider, "provider property should be set")
}

func TestYamlMCPServerDisplayNameTooLong(t *testing.T) {
	displayName := strings.Repeat("a", maxMCPDisplayNameLen+1)
	yamlServer := &yamlMCPServer{
		Name:        "test-server",
		DisplayName: &displayName,
	}

	record := yamlServer.ToMCPServerProviderRecord()
	require.Error(t, record.Error)
	assert.Contains(t, record.Error.Error(), "displayName exceeds maximum length")
}

func TestYamlMCPProviderEmitWithDisplayNameKey(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-catalog.yaml")

	yamlContent := `mcp_servers:
  - name: "test-server-1"
    displayName: "Test Server One"
    description: "First test server"
    provider: "Test Provider"
    version: "1.0.0"
`
	err := os.WriteFile(testFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	source := basecatalog.MCPSource{
		Type: "yaml",
		Properties: map[string]any{
			yamlMCPCatalogPathKey: testFile,
		},
	}

	provider, err := NewYamlMCPProvider(source)
	require.NoError(t, err)

	ctx := context.Background()
	recordChan := provider.Servers(ctx)

	record, ok := <-recordChan
	require.True(t, ok)
	require.Nil(t, record.Error)
	require.NotNil(t, record.Server)
	require.NotNil(t, record.Server.Properties)

	var foundDisplayName string
	for _, prop := range *record.Server.Properties {
		if prop.Name == "displayName" && prop.StringValue != nil {
			foundDisplayName = *prop.StringValue
		}
	}
	assert.Equal(t, "Test Server One", foundDisplayName)
}

func TestNewYamlMCPProviderPaths(t *testing.T) {
	// Test with absolute path
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-catalog.yaml")
	err := os.WriteFile(testFile, []byte("mcp_servers: []\n"), 0644)
	require.NoError(t, err)

	source := basecatalog.MCPSource{
		Type: "yaml",
		Properties: map[string]any{
			yamlMCPCatalogPathKey: testFile,
		},
	}

	provider, err := NewYamlMCPProvider(source)
	require.NoError(t, err)
	assert.NotNil(t, provider)

	yamlProvider, ok := provider.(*yamlMCPProvider)
	require.True(t, ok)
	assert.Equal(t, 1, len(yamlProvider.paths))
	assert.Equal(t, testFile, yamlProvider.paths[0])
}

func TestNewYamlMCPProviderRelativePath(t *testing.T) {
	// Test with relative path - should be resolved relative to the config file's directory
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "sources.yaml")

	source := basecatalog.MCPSource{
		Type: "yaml",
		Properties: map[string]any{
			yamlMCPCatalogPathKey: "servers/test.yaml",
		},
		Origin: configFile, // Absolute path to the config file
	}

	provider, err := NewYamlMCPProvider(source)
	require.NoError(t, err)
	assert.NotNil(t, provider)

	yamlProvider, ok := provider.(*yamlMCPProvider)
	require.True(t, ok)
	assert.Equal(t, 1, len(yamlProvider.paths))

	// Should be converted to absolute path
	assert.True(t, filepath.IsAbs(yamlProvider.paths[0]))

	// Should be resolved relative to the config file's directory
	expectedPath := filepath.Join(tmpDir, "servers/test.yaml")
	assert.Equal(t, expectedPath, yamlProvider.paths[0])
}

func TestNewYamlMCPProviderMissingPath(t *testing.T) {
	source := basecatalog.MCPSource{
		Type:       "yaml",
		Properties: map[string]any{},
	}

	_, err := NewYamlMCPProvider(source)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "yamlCatalogPath property is required")
}

func TestYamlMCPProviderEmit(t *testing.T) {
	// Create a temporary YAML file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-catalog.yaml")

	yamlContent := `mcp_servers:
  - name: "test-server-1"
    description: "First test server"
    provider: "Test Provider"
    version: "1.0.0"
    tools:
      - name: "test-tool"
        description: "Test tool"
  - name: "test-server-2"
    description: "Second test server"
    provider: "Test Provider"
    version: "2.0.0"
`
	err := os.WriteFile(testFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	source := basecatalog.MCPSource{
		Type: "yaml",
		Properties: map[string]any{
			yamlMCPCatalogPathKey: testFile,
		},
	}

	provider, err := NewYamlMCPProvider(source)
	require.NoError(t, err)

	ctx := context.Background()
	recordChan := provider.Servers(ctx)

	servers := make([]string, 0)
	for record := range recordChan {
		assert.Nil(t, record.Error)
		assert.NotNil(t, record.Server)
		attrs := record.Server.GetAttributes()
		require.NotNil(t, attrs)
		servers = append(servers, *attrs.Name)
	}

	assert.Equal(t, 2, len(servers))
	assert.Contains(t, servers, "test-server-1")
	assert.Contains(t, servers, "test-server-2")
}

func TestYamlMCPProviderEmitWithCancellation(t *testing.T) {
	// Create a temporary YAML file with many servers
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-catalog.yaml")

	// Create YAML with multiple servers
	var yamlContent strings.Builder
	yamlContent.WriteString("mcp_servers:\n")
	for i := range 10 {
		_, err := fmt.Fprintf(&yamlContent, "  - name: \"test-server-%c\"\n    description: \"Test server\"\n", rune('0'+i))
		require.NoError(t, err)
	}
	err := os.WriteFile(testFile, []byte(yamlContent.String()), 0644)
	require.NoError(t, err)

	source := basecatalog.MCPSource{
		Type: "yaml",
		Properties: map[string]any{
			yamlMCPCatalogPathKey: testFile,
		},
	}

	provider, err := NewYamlMCPProvider(source)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	recordChan := provider.Servers(ctx)

	// Cancel after receiving first record
	recordCount := 0
	for range recordChan {
		recordCount++
		if recordCount == 1 {
			cancel()
		}
	}

	// Should have received at least 1 record before cancellation
	assert.GreaterOrEqual(t, recordCount, 1)
}

func TestRegisterMCPProvider(t *testing.T) {
	// Test registering a new provider
	testProviderFunc := func(source basecatalog.MCPSource) (MCPProvider, error) {
		return nil, nil
	}

	err := RegisterMCPProvider("test-provider", testProviderFunc)
	assert.NoError(t, err)
	t.Cleanup(func() { unregisterMCPProvider("test-provider") })

	// Test registering duplicate provider
	err = RegisterMCPProvider("test-provider", testProviderFunc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	// Verify provider can be retrieved
	provider, exists := GetMCPProvider("test-provider")
	assert.True(t, exists)
	assert.NotNil(t, provider)

	// Verify unknown provider returns false
	_, exists = GetMCPProvider("unknown-provider")
	assert.False(t, exists)
}

func TestYamlMCPProviderInvalidFile(t *testing.T) {
	// Test with non-existent file
	source := basecatalog.MCPSource{
		Type: "yaml",
		Properties: map[string]any{
			yamlMCPCatalogPathKey: "/nonexistent/path/catalog.yaml",
		},
	}

	provider, err := NewYamlMCPProvider(source)
	require.NoError(t, err)

	ctx := context.Background()
	recordChan := provider.Servers(ctx)

	// Should emit exactly one error record so the caller can mark the source as errored.
	var records []MCPServerProviderRecord
	for record := range recordChan {
		records = append(records, record)
	}
	require.Len(t, records, 1, "expected one error record for unreadable file")
	assert.NotNil(t, records[0].Error, "error record should have non-nil Error")
	assert.Nil(t, records[0].Server, "error record should have nil Server")
}

func TestYamlMCPEndpointsValidation(t *testing.T) {
	tests := []struct {
		name        string
		server      *yamlMCPServer
		wantErr     bool
		errContains string
	}{
		{
			name: "remote server with valid HTTP endpoint passes",
			server: &yamlMCPServer{
				Name:           "remote-http",
				DeploymentMode: new("remote"),
				Endpoints:      &yamlMCPEndpoints{HTTP: new("https://api.example.com/mcp")},
			},
		},
		{
			name: "remote server with valid SSE endpoint passes",
			server: &yamlMCPServer{
				Name:           "remote-sse",
				DeploymentMode: new("remote"),
				Endpoints:      &yamlMCPEndpoints{SSE: new("https://api.example.com/mcp/events")},
			},
		},
		{
			name: "remote server with both endpoints passes",
			server: &yamlMCPServer{
				Name:           "remote-both",
				DeploymentMode: new("remote"),
				Endpoints: &yamlMCPEndpoints{
					HTTP: new("https://api.example.com/mcp"),
					SSE:  new("https://api.example.com/mcp/events"),
				},
			},
		},
		{
			name: "remote server with no endpoints is rejected",
			server: &yamlMCPServer{
				Name:           "remote-no-endpoints",
				DeploymentMode: new("remote"),
			},
			wantErr:     true,
			errContains: "no endpoints defined",
		},
		{
			name: "remote server with empty endpoints struct is rejected",
			server: &yamlMCPServer{
				Name:           "remote-empty-endpoints",
				DeploymentMode: new("remote"),
				Endpoints:      &yamlMCPEndpoints{},
			},
			wantErr:     true,
			errContains: "no endpoints defined",
		},
		{
			name: "local server without endpoints passes",
			server: &yamlMCPServer{
				Name:           "local-no-endpoints",
				DeploymentMode: new("local"),
			},
		},
		{
			name: "local server with endpoints passes (hybrid)",
			server: &yamlMCPServer{
				Name:           "local-with-endpoints",
				DeploymentMode: new("local"),
				Endpoints:      &yamlMCPEndpoints{HTTP: new("https://api.example.com/mcp")},
			},
		},
		{
			name: "non-HTTP URL in HTTP endpoint is rejected",
			server: &yamlMCPServer{
				Name:           "bad-url-scheme",
				DeploymentMode: new("remote"),
				Endpoints:      &yamlMCPEndpoints{HTTP: new("ftp://example.com/mcp")},
			},
			wantErr:     true,
			errContains: "must be a valid URL",
		},
		{
			name: "malformed URL is rejected",
			server: &yamlMCPServer{
				Name:           "malformed-url",
				DeploymentMode: new("remote"),
				Endpoints:      &yamlMCPEndpoints{HTTP: new("not-a-url-at-all")},
			},
			wantErr:     true,
			errContains: "must be a valid URL",
		},
		{
			name: "plain http URL is accepted",
			server: &yamlMCPServer{
				Name:           "plain-http",
				DeploymentMode: new("remote"),
				Endpoints:      &yamlMCPEndpoints{HTTP: new("http://api.example.com/mcp")},
			},
		},
		{
			name: "websocket-only remote server passes",
			server: &yamlMCPServer{
				Name:           "remote-ws",
				DeploymentMode: new("remote"),
				Endpoints:      &yamlMCPEndpoints{WebSocket: new("wss://api.example.com/ws")},
			},
		},
		{
			name: "ws scheme accepted for websocket endpoint",
			server: &yamlMCPServer{
				Name:           "remote-ws-plain",
				DeploymentMode: new("remote"),
				Endpoints:      &yamlMCPEndpoints{WebSocket: new("ws://api.example.com/ws")},
			},
		},
		{
			name: "ftp scheme rejected for websocket endpoint",
			server: &yamlMCPServer{
				Name:           "bad-ws-scheme",
				DeploymentMode: new("remote"),
				Endpoints:      &yamlMCPEndpoints{WebSocket: new("ftp://example.com/ws")},
			},
			wantErr:     true,
			errContains: "must be a valid URL",
		},
		{
			name: "server with no deploymentMode and endpoints passes",
			server: &yamlMCPServer{
				Name:      "no-mode-with-endpoints",
				Endpoints: &yamlMCPEndpoints{HTTP: new("https://api.example.com/mcp")},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			record := tc.server.ToMCPServerProviderRecord()
			if tc.wantErr {
				assert.NotNil(t, record.Error, "expected an error but got none")
				if tc.errContains != "" {
					assert.Contains(t, record.Error.Error(), tc.errContains)
				}
			} else {
				assert.Nil(t, record.Error, "expected no error but got: %v", record.Error)
				assert.NotNil(t, record.Server)
			}
		})
	}
}

func TestYamlMCPEndpointsJSONKeys(t *testing.T) {
	endpoints := &yamlMCPEndpoints{
		HTTP: new("https://api.example.com/mcp"),
		SSE:  new("https://api.example.com/mcp/events"),
	}

	jsonBytes, err := json.Marshal(endpoints)
	require.NoError(t, err)

	jsonStr := string(jsonBytes)
	assert.Contains(t, jsonStr, `"http":`, "JSON key should be lowercase 'http'")
	assert.Contains(t, jsonStr, `"sse":`, "JSON key should be lowercase 'sse'")
	assert.NotContains(t, jsonStr, `"HTTP":`, "JSON key must not be uppercase 'HTTP'")
	assert.NotContains(t, jsonStr, `"SSE":`, "JSON key must not be uppercase 'SSE'")
}

func TestYamlMCPArtifactURIValidation(t *testing.T) {
	tests := []struct {
		name        string
		server      *yamlMCPServer
		wantErr     bool
		errContains string
	}{
		{
			name: "artifact with valid oci URI passes",
			server: &yamlMCPServer{
				Name: "server-oci-artifact",
				Artifacts: []apimodels.MCPArtifact{
					{Uri: "oci://registry.example.com/image:v1"},
				},
			},
		},
		{
			name: "artifact with valid https URI passes",
			server: &yamlMCPServer{
				Name: "server-https-artifact",
				Artifacts: []apimodels.MCPArtifact{
					{Uri: "https://example.com/model.bin"},
				},
			},
		},
		{
			name: "artifact with valid s3 URI passes",
			server: &yamlMCPServer{
				Name: "server-s3-artifact",
				Artifacts: []apimodels.MCPArtifact{
					{Uri: "s3://bucket/path/to/weights"},
				},
			},
		},
		{
			name: "artifact with invalid URI (no scheme) is rejected",
			server: &yamlMCPServer{
				Name: "server-invalid-uri",
				Artifacts: []apimodels.MCPArtifact{
					{Uri: "not-a-valid-uri"},
				},
			},
			wantErr:     true,
			errContains: "must have a scheme",
		},
		{
			name: "artifact with unsupported scheme is rejected",
			server: &yamlMCPServer{
				Name: "server-ftp-artifact",
				Artifacts: []apimodels.MCPArtifact{
					{Uri: "ftp://example.com/model.bin"},
				},
			},
			wantErr:     true,
			errContains: "unsupported scheme",
		},
		{
			name: "artifact with empty URI is rejected",
			server: &yamlMCPServer{
				Name: "server-empty-uri",
				Artifacts: []apimodels.MCPArtifact{
					{Uri: ""},
				},
			},
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name: "multiple artifacts with one invalid URI is rejected",
			server: &yamlMCPServer{
				Name: "server-mixed-artifacts",
				Artifacts: []apimodels.MCPArtifact{
					{Uri: "s3://bucket/valid"},
					{Uri: "invalid-uri"},
				},
			},
			wantErr:     true,
			errContains: "artifact 1",
		},
		{
			name: "server with no artifacts passes",
			server: &yamlMCPServer{
				Name: "server-no-artifacts",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			record := tc.server.ToMCPServerProviderRecord()
			if tc.wantErr {
				assert.NotNil(t, record.Error, "expected an error but got none")
				if tc.errContains != "" {
					assert.Contains(t, record.Error.Error(), tc.errContains)
				}
			} else {
				assert.Nil(t, record.Error, "expected no error but got: %v", record.Error)
				assert.NotNil(t, record.Server)
			}
		})
	}
}

func TestYamlMCPArtifactTimestampRoundTrip(t *testing.T) {
	createTime := "1753292581000"
	updateTime := "1774534350000"
	server := &yamlMCPServer{
		Name: "server-with-timestamps",
		Artifacts: []apimodels.MCPArtifact{
			{Uri: "oci://registry.example.com/image:v1", CreateTimeSinceEpoch: &createTime, LastUpdateTimeSinceEpoch: &updateTime},
		},
	}

	record := server.ToMCPServerProviderRecord()
	require.Nil(t, record.Error)
	require.NotNil(t, record.Server.Properties)

	var artifactsJSON string
	for _, prop := range *record.Server.Properties {
		if prop.Name == "artifacts" && prop.StringValue != nil {
			artifactsJSON = *prop.StringValue
		}
	}
	require.NotEmpty(t, artifactsJSON, "artifacts property should be set")

	var artifacts []apimodels.MCPArtifact
	err := json.Unmarshal([]byte(artifactsJSON), &artifacts)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	assert.Equal(t, "oci://registry.example.com/image:v1", artifacts[0].Uri)
	require.NotNil(t, artifacts[0].CreateTimeSinceEpoch, "createTimeSinceEpoch should survive round-trip")
	assert.Equal(t, "1753292581000", *artifacts[0].CreateTimeSinceEpoch)
	require.NotNil(t, artifacts[0].LastUpdateTimeSinceEpoch, "lastUpdateTimeSinceEpoch should survive round-trip")
	assert.Equal(t, "1774534350000", *artifacts[0].LastUpdateTimeSinceEpoch)
}

func TestYamlMCPServerLicenseTransformation(t *testing.T) {
	tests := []struct {
		name            string
		inputLicense    string
		expectedLicense string
	}{
		{
			name:            "apache-2.0 SPDX transforms to human-readable",
			inputLicense:    "apache-2.0",
			expectedLicense: "Apache 2.0",
		},
		{
			name:            "MIT SPDX transforms to human-readable",
			inputLicense:    "mit",
			expectedLicense: "MIT",
		},
		{
			name:            "GPL-3.0 SPDX transforms to human-readable",
			inputLicense:    "gpl-3.0",
			expectedLicense: "GPL 3.0",
		},
		{
			name:            "BSD-3-Clause SPDX transforms to human-readable",
			inputLicense:    "bsd-3-clause",
			expectedLicense: "BSD 3-Clause",
		},
		{
			name:            "Llama3.1 custom license transforms",
			inputLicense:    "llama3.1",
			expectedLicense: "Llama 3.1 Community License",
		},
		{
			name:            "Llama3 custom license transforms",
			inputLicense:    "llama3",
			expectedLicense: "Llama 3 Community License",
		},
		{
			name:            "Gemma custom license transforms",
			inputLicense:    "gemma",
			expectedLicense: "Gemma License",
		},
		{
			name:            "Creative Commons CC-BY-4.0 transforms",
			inputLicense:    "cc-by-4.0",
			expectedLicense: "Creative Commons Attribution 4.0 International",
		},
		{
			name:            "Creative Commons CC0 transforms",
			inputLicense:    "cc0-1.0",
			expectedLicense: "Creative Commons Zero v1.0 Universal",
		},
		{
			name:            "OpenRAIL license transforms",
			inputLicense:    "openrail",
			expectedLicense: "OpenRAIL License",
		},
		{
			name:            "BigScience OpenRAIL-M transforms",
			inputLicense:    "bigscience-openrail-m",
			expectedLicense: "BigScience OpenRAIL-M License",
		},
		{
			name:            "Uppercase APACHE-2.0 normalizes and transforms",
			inputLicense:    "APACHE-2.0",
			expectedLicense: "Apache 2.0",
		},
		{
			name:            "Mixed case Mit normalizes and transforms",
			inputLicense:    "Mit",
			expectedLicense: "MIT",
		},
		{
			name:            "GPL-2.0 SPDX transforms to human-readable",
			inputLicense:    "gpl-2.0",
			expectedLicense: "GPL 2.0",
		},
		{
			name:            "LGPL-3.0 SPDX transforms to human-readable",
			inputLicense:    "lgpl-3.0",
			expectedLicense: "LGPL 3.0",
		},
		{
			name:            "Unknown license passes through unchanged",
			inputLicense:    "custom-proprietary-license-1.0",
			expectedLicense: "custom-proprietary-license-1.0",
		},
		{
			name:            "Empty license results in no license property",
			inputLicense:    "",
			expectedLicense: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yamlServer *yamlMCPServer
			if tt.inputLicense == "" {
				yamlServer = &yamlMCPServer{
					Name:     "test-server",
					License:  nil, // No license field
					Provider: new("Test Provider"),
				}
			} else {
				yamlServer = &yamlMCPServer{
					Name:     "test-server",
					License:  &tt.inputLicense,
					Provider: new("Test Provider"),
				}
			}

			record := yamlServer.ToMCPServerProviderRecord()

			assert.Nil(t, record.Error, "expected no error converting server")
			assert.NotNil(t, record.Server, "expected server to be created")

			props := record.Server.GetProperties()
			require.NotNil(t, props, "expected properties to be set")

			// Find the license property
			var foundLicense *string
			for _, prop := range *props {
				if prop.Name == "license" && prop.StringValue != nil {
					foundLicense = prop.StringValue
					break
				}
			}

			if tt.expectedLicense == "" {
				assert.Nil(t, foundLicense, "expected no license property for empty input")
			} else {
				require.NotNil(t, foundLicense, "expected license property to be set")
				assert.Equal(t, tt.expectedLicense, *foundLicense,
					"license should be transformed from %q to %q", tt.inputLicense, tt.expectedLicense)
			}
		})
	}
}

func TestYamlMCPServerLicenseConsistencyWithModels(t *testing.T) {
	// This test verifies that MCP servers apply the same license transformation
	// as models to ensure API consistency
	testLicenses := []string{
		"apache-2.0",
		"mit",
		"llama3.1",
		"cc-by-4.0",
		"gpl-3.0",
	}

	for _, license := range testLicenses {
		t.Run(license, func(t *testing.T) {
			// Get the expected human-readable transformation
			expectedHumanReadable := basecatalog.TransformLicenseToHumanReadable(license)

			// Create MCP server with this license
			yamlServer := &yamlMCPServer{
				Name:    "test-server",
				License: &license,
			}

			record := yamlServer.ToMCPServerProviderRecord()
			require.Nil(t, record.Error)
			require.NotNil(t, record.Server)

			// Extract license from properties
			props := record.Server.GetProperties()
			require.NotNil(t, props)

			var actualLicense *string
			for _, prop := range *props {
				if prop.Name == "license" && prop.StringValue != nil {
					actualLicense = prop.StringValue
					break
				}
			}

			require.NotNil(t, actualLicense, "license property should be set")
			assert.Equal(t, expectedHumanReadable, *actualLicense,
				"MCP server license transformation should match model catalog transformation")
		})
	}
}

func TestYamlMCPToolAccessTypePreserved(t *testing.T) {
	accessType := "read_write"
	paramDesc := "Project key"
	yamlServer := &yamlMCPServer{
		Name: "test-server",
		Tools: []*yamlMCPTool{
			{
				Name:       "create_issue",
				AccessType: &accessType,
				Parameters: []yamlMCPParameter{
					{Name: "project", Type: "string", Description: &paramDesc, Required: true},
					{Name: "summary", Type: "string", Required: true},
				},
			},
			{
				Name: "search_issues",
				// no AccessType set - should be nil in the record
			},
		},
	}

	record := yamlServer.ToMCPServerProviderRecord()

	require.Len(t, record.Tools, 2)

	createTool := record.Tools[0]
	require.NotNil(t, createTool.AccessType, "accessType should be preserved")
	assert.Equal(t, "read_write", *createTool.AccessType)
	require.Len(t, createTool.Parameters, 2)
	assert.Equal(t, "project", createTool.Parameters[0].Name)
	assert.Equal(t, "string", createTool.Parameters[0].Type)
	assert.True(t, createTool.Parameters[0].Required)
	require.NotNil(t, createTool.Parameters[0].Description)
	assert.Equal(t, "Project key", *createTool.Parameters[0].Description)
	assert.Equal(t, "summary", createTool.Parameters[1].Name)
	assert.True(t, createTool.Parameters[1].Required)

	searchTool := record.Tools[1]
	assert.Nil(t, searchTool.AccessType, "nil accessType should stay nil")
	assert.Empty(t, searchTool.Parameters)
}

func TestYamlMCPServerRuntimeMetadataConversion(t *testing.T) {
	defaultPort := int32(8080)
	mcpPath := "/mcp"
	saRequired := true
	saHint := "Needs 'view' ClusterRole"
	saName := "mcp-viewer"
	cmMountAsFile := true
	cmMountPath := "/etc/mcp-config"
	cmKeyRequired := false

	yamlServer := &yamlMCPServer{
		Name: "runtime-meta-server",
		RuntimeMetadata: &apimodels.MCPRuntimeMetadata{
			DefaultPort: &defaultPort,
			McpPath:     &mcpPath,
			Prerequisites: &apimodels.MCPPrerequisites{
				ServiceAccount: &apimodels.MCPServiceAccountRequirement{
					Required:      &saRequired,
					Hint:          &saHint,
					SuggestedName: &saName,
				},
				ConfigMaps: []apimodels.MCPConfigMapRequirement{
					{
						Name:        "server-config",
						Description: "Config files",
						MountAsFile: &cmMountAsFile,
						MountPath:   &cmMountPath,
						Keys: []apimodels.MCPConfigMapKey{
							{
								Key:         "config.toml",
								Description: "Main config",
								Required:    &cmKeyRequired,
							},
						},
					},
				},
			},
		},
	}

	record := yamlServer.ToMCPServerProviderRecord()
	assert.Nil(t, record.Error)
	assert.NotNil(t, record.Server)

	props := record.Server.GetProperties()
	require.NotNil(t, props)

	var runtimeJSON *string
	for _, prop := range *props {
		if prop.Name == "runtimeMetadata" && prop.StringValue != nil {
			runtimeJSON = prop.StringValue
			break
		}
	}
	require.NotNil(t, runtimeJSON, "runtimeMetadata property should be set")

	var parsed apimodels.MCPRuntimeMetadata
	err := json.Unmarshal([]byte(*runtimeJSON), &parsed)
	require.NoError(t, err)

	assert.Equal(t, int32(8080), *parsed.DefaultPort)
	assert.Equal(t, "/mcp", *parsed.McpPath)
	require.NotNil(t, parsed.Prerequisites)
	require.NotNil(t, parsed.Prerequisites.ServiceAccount)
	assert.True(t, *parsed.Prerequisites.ServiceAccount.Required)
	assert.Equal(t, "mcp-viewer", *parsed.Prerequisites.ServiceAccount.SuggestedName)
	require.Len(t, parsed.Prerequisites.ConfigMaps, 1)
	assert.Equal(t, "server-config", parsed.Prerequisites.ConfigMaps[0].Name)
}

func TestYamlMCPServerRuntimeMetadataNil(t *testing.T) {
	yamlServer := &yamlMCPServer{
		Name: "no-runtime-meta",
	}

	record := yamlServer.ToMCPServerProviderRecord()
	assert.Nil(t, record.Error)

	props := record.Server.GetProperties()
	require.NotNil(t, props)

	for _, prop := range *props {
		assert.NotEqual(t, "runtimeMetadata", prop.Name,
			"runtimeMetadata property should not be set when RuntimeMetadata is nil")
	}
}

func TestYamlMCPProviderEmitWithRuntimeMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-catalog.yaml")

	yamlContent := `mcp_servers:
  - name: "k8s-mcp"
    provider: "Test"
    version: "1.0.0"
    runtimeMetadata:
      defaultPort: 8080
      mcpPath: /mcp
      prerequisites:
        serviceAccount:
          required: true
          hint: "Needs view ClusterRole"
          suggestedName: mcp-viewer
        secrets:
          - name: api-creds
            description: "API credentials"
            keys:
              - key: api-key
                description: "API key"
                envVarName: API_KEY
                required: true
            mountAsFile: false
`
	err := os.WriteFile(testFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	source := basecatalog.MCPSource{
		Type: "yaml",
		Properties: map[string]any{
			yamlMCPCatalogPathKey: testFile,
		},
	}

	provider, err := NewYamlMCPProvider(source)
	require.NoError(t, err)

	ctx := context.Background()
	recordChan := provider.Servers(ctx)

	var record MCPServerProviderRecord
	for r := range recordChan {
		record = r
	}

	assert.Nil(t, record.Error)
	assert.NotNil(t, record.Server)

	props := record.Server.GetProperties()
	require.NotNil(t, props)

	var runtimeJSON *string
	for _, prop := range *props {
		if prop.Name == "runtimeMetadata" && prop.StringValue != nil {
			runtimeJSON = prop.StringValue
			break
		}
	}
	require.NotNil(t, runtimeJSON, "runtimeMetadata should be parsed from YAML file")

	var parsed apimodels.MCPRuntimeMetadata
	err = json.Unmarshal([]byte(*runtimeJSON), &parsed)
	require.NoError(t, err)

	assert.Equal(t, int32(8080), *parsed.DefaultPort)
	assert.Equal(t, "/mcp", *parsed.McpPath)
	require.NotNil(t, parsed.Prerequisites)
	require.NotNil(t, parsed.Prerequisites.ServiceAccount)
	assert.True(t, *parsed.Prerequisites.ServiceAccount.Required)
	assert.Equal(t, "mcp-viewer", *parsed.Prerequisites.ServiceAccount.SuggestedName)
	require.Len(t, parsed.Prerequisites.Secrets, 1)
	assert.Equal(t, "api-creds", parsed.Prerequisites.Secrets[0].Name)
	require.Len(t, parsed.Prerequisites.Secrets[0].Keys, 1)
	assert.Equal(t, "API_KEY", *parsed.Prerequisites.Secrets[0].Keys[0].EnvVarName)
}

func TestYamlMCPServerSecurityIndicatorsToStandardProperties(t *testing.T) {
	trueVal := true
	falseVal := false
	yamlServer := &yamlMCPServer{
		Name: "test-server",
		SecurityIndicators: &yamlMCPSecurityIndicator{
			VerifiedSource: &trueVal,
			SecureEndpoint: &falseVal,
			Sast:           &trueVal,
			ReadOnlyTools:  &falseVal,
		},
	}

	record := yamlServer.ToMCPServerProviderRecord()

	require.NotNil(t, record.Server)
	require.Nil(t, record.Error)

	// Security indicators must land in standard properties, not custom properties
	props := record.Server.GetProperties()
	require.NotNil(t, props)

	propMap := make(map[string]bool)
	for _, p := range *props {
		if p.BoolValue != nil {
			propMap[p.Name] = *p.BoolValue
		}
	}

	assert.True(t, propMap["verifiedSource"], "verifiedSource should be true in standard properties")
	assert.False(t, propMap["secureEndpoint"], "secureEndpoint should be false in standard properties")
	assert.True(t, propMap["sast"], "sast should be true in standard properties")
	assert.False(t, propMap["readOnlyTools"], "readOnlyTools should be false in standard properties")

	// Security indicators must NOT appear in custom properties
	customProps := record.Server.GetCustomProperties()
	if customProps != nil {
		for _, p := range *customProps {
			assert.NotEqual(t, "verifiedSource", p.Name)
			assert.NotEqual(t, "secureEndpoint", p.Name)
			assert.NotEqual(t, "sast", p.Name)
			assert.NotEqual(t, "readOnlyTools", p.Name)
		}
	}
}

func TestYamlMCPServerSecurityIndicatorsNil(t *testing.T) {
	yamlServer := &yamlMCPServer{
		Name:               "test-server",
		SecurityIndicators: nil,
	}

	record := yamlServer.ToMCPServerProviderRecord()

	require.NotNil(t, record.Server)
	require.Nil(t, record.Error)

	props := record.Server.GetProperties()
	if props != nil {
		for _, p := range *props {
			assert.Nil(t, p.BoolValue, "no bool properties should exist when SecurityIndicators is nil (prop: %s)", p.Name)
		}
	}
}

func TestYamlMCPServerSecurityIndicatorsPartial(t *testing.T) {
	trueVal := true
	yamlServer := &yamlMCPServer{
		Name: "test-server",
		SecurityIndicators: &yamlMCPSecurityIndicator{
			VerifiedSource: &trueVal,
			// SecureEndpoint, Sast, ReadOnlyTools intentionally unset
		},
	}

	record := yamlServer.ToMCPServerProviderRecord()

	require.NotNil(t, record.Server)
	require.Nil(t, record.Error)

	props := record.Server.GetProperties()
	require.NotNil(t, props)

	boolProps := make(map[string]bool)
	for _, p := range *props {
		if p.BoolValue != nil {
			boolProps[p.Name] = *p.BoolValue
		}
	}

	assert.True(t, boolProps["verifiedSource"], "verifiedSource should be true")
	_, hasSecureEndpoint := boolProps["secureEndpoint"]
	assert.False(t, hasSecureEndpoint, "secureEndpoint should not be present when unset")
	_, hasSast := boolProps["sast"]
	assert.False(t, hasSast, "sast should not be present when unset")
	_, hasReadOnlyTools := boolProps["readOnlyTools"]
	assert.False(t, hasReadOnlyTools, "readOnlyTools should not be present when unset")
}
