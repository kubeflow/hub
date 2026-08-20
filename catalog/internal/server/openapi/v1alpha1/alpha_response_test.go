package v1alpha1

import (
	"encoding/json"
	"testing"

	model "github.com/kubeflow/hub/catalog/pkg/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlphaResponseJSONRenaming(t *testing.T) {
	sourceID := "test-source-1"
	licenseLink := "https://example.com/license"

	t.Run("AlphaCatalogModel renames sourceId to source_id", func(t *testing.T) {
		m := model.CatalogModel{
			Name:     "test-model",
			SourceId: &sourceID,
		}
		wrapped := wrapCatalogModel(&m)

		data, err := json.Marshal(wrapped)
		require.NoError(t, err)

		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"source_id":"test-source-1"`)
		assert.NotContains(t, jsonStr, `"sourceId"`)
	})

	t.Run("AlphaCatalogModelList renames sourceId in items", func(t *testing.T) {
		m := model.CatalogModel{
			Name:     "test-model",
			SourceId: &sourceID,
		}
		list := model.CatalogModelList{
			Items: []model.CatalogModel{m},
			Size:  1,
		}
		wrapped := wrapCatalogModelList(&list)

		data, err := json.Marshal(wrapped)
		require.NoError(t, err)

		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"source_id":"test-source-1"`)
		assert.NotContains(t, jsonStr, `"sourceId"`)
	})

	t.Run("AlphaMCPServer renames sourceId and licenseLink", func(t *testing.T) {
		s := model.MCPServer{
			Name:        "test-mcp",
			SourceId:    &sourceID,
			LicenseLink: &licenseLink,
		}
		wrapped := wrapMCPServer(&s)

		data, err := json.Marshal(wrapped)
		require.NoError(t, err)

		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"source_id":"test-source-1"`)
		assert.Contains(t, jsonStr, `"license_link":"https://example.com/license"`)
		assert.NotContains(t, jsonStr, `"sourceId"`)
		assert.NotContains(t, jsonStr, `"licenseLink"`)
	})

	t.Run("AlphaMCPServerList renames sourceId and licenseLink in items", func(t *testing.T) {
		s := model.MCPServer{
			Name:        "test-mcp",
			SourceId:    &sourceID,
			LicenseLink: &licenseLink,
		}
		list := model.MCPServerList{
			Items: []model.MCPServer{s},
			Size:  1,
		}
		wrapped := wrapMCPServerList(&list)

		data, err := json.Marshal(wrapped)
		require.NoError(t, err)

		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"source_id":"test-source-1"`)
		assert.Contains(t, jsonStr, `"license_link":"https://example.com/license"`)
		assert.NotContains(t, jsonStr, `"sourceId"`)
		assert.NotContains(t, jsonStr, `"licenseLink"`)
	})

	t.Run("AlphaAgent renames sourceId to source_id", func(t *testing.T) {
		a := model.Agent{
			Name:     "test-agent",
			SourceId: &sourceID,
		}
		wrapped := wrapAgent(&a)

		data, err := json.Marshal(wrapped)
		require.NoError(t, err)

		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"source_id":"test-source-1"`)
		assert.NotContains(t, jsonStr, `"sourceId"`)
	})

	t.Run("AlphaAgentList renames sourceId in items", func(t *testing.T) {
		a := model.Agent{
			Name:     "test-agent",
			SourceId: &sourceID,
		}
		list := model.AgentList{
			Items: []model.Agent{a},
			Size:  1,
		}
		wrapped := wrapAgentList(&list)

		data, err := json.Marshal(wrapped)
		require.NoError(t, err)

		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"source_id":"test-source-1"`)
		assert.NotContains(t, jsonStr, `"sourceId"`)
	})

	t.Run("AlphaCatalogModel preserves keys in nested maps", func(t *testing.T) {
		m := model.CatalogModel{
			Name:     "test-model",
			SourceId: &sourceID,
			CustomProperties: map[string]model.MetadataValue{
				"sourceId": {
					MetadataStringValue: &model.MetadataStringValue{
						StringValue: "nested-val",
					},
				},
			},
		}
		wrapped := wrapCatalogModel(&m)

		data, err := json.Marshal(wrapped)
		require.NoError(t, err)

		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"source_id":"test-source-1"`)
		assert.Contains(t, jsonStr, `"customProperties":{"sourceId":{`)
	})
}
