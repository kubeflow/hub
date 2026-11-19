package repositories

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAPIPathsTargetV1(t *testing.T) {
	assert.Equal(t, "/api/model_registry/v1", ModelRegistryAPIPath)
	assert.Equal(t, "/api/model_catalog/v1", ModelCatalogAPIPath)
	assert.Equal(t, "/api/mcp_catalog/v1", McpCatalogAPIPath)
	assert.Equal(t, "/api/agent_catalog/v1", AgentCatalogAPIPath)
}

func TestModelRegistryResolveServerAddress(t *testing.T) {
	tests := []struct {
		name                string
		isHTTPS             bool
		externalAddressRest string
		isFederatedMode     bool
		expected            string
	}{
		{name: "cluster IP over http", expected: "http://10.0.0.1:8080/api/model_registry/v1"},
		{name: "cluster IP over https", isHTTPS: true, expected: "https://10.0.0.1:8080/api/model_registry/v1"},
		{name: "federated with external address", isHTTPS: true, externalAddressRest: "mr.example.com", isFederatedMode: true, expected: "https://mr.example.com/api/model_registry/v1"},
		{name: "federated without external address falls back to cluster IP", isFederatedMode: true, expected: "http://10.0.0.1:8080/api/model_registry/v1"},
		{name: "external address ignored when not federated", externalAddressRest: "mr.example.com", expected: "http://10.0.0.1:8080/api/model_registry/v1"},
	}

	repo := NewModelRegistryRepository()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repo.ResolveServerAddress("10.0.0.1", 8080, tt.isHTTPS, tt.externalAddressRest, tt.isFederatedMode)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestModelCatalogResolveServerAddress(t *testing.T) {
	tests := []struct {
		name                string
		isHTTPS             bool
		externalAddressRest string
		isFederatedMode     bool
		apiPath             string
		expected            string
	}{
		{name: "model catalog over cluster IP", apiPath: ModelCatalogAPIPath, expected: "http://10.0.0.1:8080/api/model_catalog/v1"},
		{name: "mcp catalog over https", isHTTPS: true, apiPath: McpCatalogAPIPath, expected: "https://10.0.0.1:8080/api/mcp_catalog/v1"},
		{name: "agent catalog federated with external address", isHTTPS: true, externalAddressRest: "catalog.example.com", isFederatedMode: true, apiPath: AgentCatalogAPIPath, expected: "https://catalog.example.com/api/agent_catalog/v1"},
		{name: "federated without external address falls back to cluster IP", isFederatedMode: true, apiPath: ModelCatalogAPIPath, expected: "http://10.0.0.1:8080/api/model_catalog/v1"},
	}

	repo := &ModelCatalogRepository{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repo.ResolveServerAddress("10.0.0.1", 8080, tt.isHTTPS, tt.externalAddressRest, tt.isFederatedMode, tt.apiPath)
			assert.Equal(t, tt.expected, got)
		})
	}
}
