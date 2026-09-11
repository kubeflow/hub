package mcpcatalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYamlMCPStoragePersistenceRoundTrip(t *testing.T) {
	_, services, cleanup := setupMCPLoaderTest(t)
	defer cleanup()

	catalog := NewDBMCPCatalog(services, nil, nil)
	for _, tc := range []struct {
		name         string
		yamlMetadata string
		wantJSON     string
	}{
		{
			name: "storage-sources",
			yamlMetadata: `      defaultPort: 8080
      storage:
        - path: /tmp
          permissions: ReadWrite
          source:
            type: EmptyDir
            emptyDir: {}
        - path: /cache
          permissions: ReadWrite
          source:
            type: EmptyDir
            emptyDir:
              medium: Memory
              sizeLimit: 128Mi
        - path: /etc/config
          source:
            type: ConfigMap
            configMap:
              name: server-config
              items:
                - key: config
                  path: config.yaml
        - path: /etc/secrets
          permissions: ReadOnly
          source:
            type: Secret
            secret:
              secretName: server-credentials
              optional: false
`,
			wantJSON: `{
  "defaultPort": 8080,
  "storage": [
    {"path": "/tmp", "permissions": "ReadWrite", "source": {"type": "EmptyDir", "emptyDir": {}}},
    {"path": "/cache", "permissions": "ReadWrite", "source": {"type": "EmptyDir", "emptyDir": {"medium": "Memory", "sizeLimit": "128Mi"}}},
    {"path": "/etc/config", "source": {"type": "ConfigMap", "configMap": {"name": "server-config", "items": [{"key": "config", "path": "config.yaml"}]}}},
    {"path": "/etc/secrets", "permissions": "ReadOnly", "source": {"type": "Secret", "secret": {"secretName": "server-credentials", "optional": false}}}
  ]
}`,
		},
		{
			name:         "without-storage",
			yamlMetadata: "      defaultPort: 8080\n",
			wantJSON:     `{"defaultPort": 8080}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			serversFile := filepath.Join(t.TempDir(), "servers.yaml")
			yamlContent := "mcp_servers:\n  - name: storage-round-trip-" + tc.name + "\n    version: 1.0.0\n    runtimeMetadata:\n" + tc.yamlMetadata
			require.NoError(t, os.WriteFile(serversFile, []byte(yamlContent), 0644))

			provider, err := NewYamlMCPProvider(basecatalog.MCPSource{
				Type: "yaml",
				Properties: map[string]any{
					yamlMCPCatalogPathKey: serversFile,
				},
			})
			require.NoError(t, err)

			record, ok := <-provider.Servers(t.Context())
			require.True(t, ok)
			require.NoError(t, record.Error)
			require.NotNil(t, record.Server)

			saved, err := services.MCPServerRepository.Save(record.Server)
			require.NoError(t, err)
			require.NotNil(t, saved.GetID())

			// Read back from PostgreSQL rather than inspecting the provider's in-memory record.
			persisted, err := services.MCPServerRepository.GetByID(*saved.GetID())
			require.NoError(t, err)
			require.NotNil(t, persisted.GetProperties())
			var runtimeJSON *string
			for _, prop := range *persisted.GetProperties() {
				if prop.Name == "runtimeMetadata" {
					runtimeJSON = prop.StringValue
					break
				}
			}
			require.NotNil(t, runtimeJSON, "runtimeMetadata should be persisted")
			assert.JSONEq(t, tc.wantJSON, *runtimeJSON)

			apiServer, err := catalog.GetMCPServer(t.Context(), fmt.Sprint(*saved.GetID()), false, 0)
			require.NoError(t, err)
			responseJSON, err := json.Marshal(apiServer)
			require.NoError(t, err)
			var response map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(responseJSON, &response))
			require.Contains(t, response, "runtimeMetadata")
			// Compare wire JSON so dropping emptyDir: {} or adding absent fields fails the test.
			assert.JSONEq(t, tc.wantJSON, string(response["runtimeMetadata"]))
		})
	}
}
