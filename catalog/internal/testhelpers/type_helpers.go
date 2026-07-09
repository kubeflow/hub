package testhelpers

import (
	"testing"

	agentcatalogservice "github.com/kubeflow/hub/catalog/internal/catalog/agentcatalog/service"
	"github.com/kubeflow/hub/catalog/internal/db/service"
	"github.com/kubeflow/hub/internal/platform/datastore"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// MustDatastoreSpec calls service.DatastoreSpec and fails the test on error.
func MustDatastoreSpec(t *testing.T) *datastore.Spec {
	t.Helper()
	spec, err := service.DatastoreSpec()
	require.NoError(t, err)
	return spec
}

// GetCatalogModelTypeIDForDBTest retrieves the CatalogModel type ID for testing
func GetCatalogModelTypeIDForDBTest(t *testing.T, db *gorm.DB) int32 {
	var typeRecord schema.Type
	err := db.Where("name = ?", service.CatalogModelTypeName).First(&typeRecord).Error
	require.NoError(t, err, "Failed to query CatalogModel type")
	return typeRecord.ID
}

// GetCatalogModelArtifactTypeIDForDBTest retrieves the CatalogModelArtifact type ID for testing
func GetCatalogModelArtifactTypeIDForDBTest(t *testing.T, db *gorm.DB) int32 {
	var typeRecord schema.Type
	err := db.Where("name = ?", service.CatalogModelArtifactTypeName).First(&typeRecord).Error
	require.NoError(t, err, "Failed to query CatalogModelArtifact type")
	return typeRecord.ID
}

// GetCatalogMetricsArtifactTypeIDForDBTest retrieves the CatalogMetricsArtifact type ID for testing
func GetCatalogMetricsArtifactTypeIDForDBTest(t *testing.T, db *gorm.DB) int32 {
	var typeRecord schema.Type
	err := db.Where("name = ?", service.CatalogMetricsArtifactTypeName).First(&typeRecord).Error
	require.NoError(t, err, "Failed to query CatalogMetricsArtifact type")
	return typeRecord.ID
}

// GetCatalogSourceTypeIDForDBTest retrieves the CatalogSource type ID for testing
func GetCatalogSourceTypeIDForDBTest(t *testing.T, db *gorm.DB) int32 {
	var typeRecord schema.Type
	err := db.Where("name = ?", service.CatalogSourceTypeName).First(&typeRecord).Error
	require.NoError(t, err, "Failed to query CatalogSource type")
	return typeRecord.ID
}

// GetMCPServerTypeIDForDBTest retrieves the MCPServer type ID for testing
func GetMCPServerTypeIDForDBTest(t *testing.T, db *gorm.DB) int32 {
	var typeRecord schema.Type
	err := db.Where("name = ?", service.MCPServerTypeName).First(&typeRecord).Error
	require.NoError(t, err, "Failed to query MCPServer type")
	return typeRecord.ID
}

// GetMCPServerToolTypeIDForDBTest retrieves the MCPServerTool type ID for testing
func GetMCPServerToolTypeIDForDBTest(t *testing.T, db *gorm.DB) int32 {
	var typeRecord schema.Type
	err := db.Where("name = ?", service.MCPServerToolTypeName).First(&typeRecord).Error
	require.NoError(t, err, "Failed to query MCPServerTool type")
	return typeRecord.ID
}

// GetAgentTypeIDForDBTest retrieves the Agent type ID for testing
func GetAgentTypeIDForDBTest(t *testing.T, db *gorm.DB) int32 {
	var typeRecord schema.Type
	err := db.Where("name = ?", agentcatalogservice.AgentTypeName).First(&typeRecord).Error
	require.NoError(t, err, "Failed to query Agent type")
	return typeRecord.ID
}

// GetAgentTemplateArtifactTypeIDForDBTest retrieves the AgentTemplateArtifact type ID for testing
func GetAgentTemplateArtifactTypeIDForDBTest(t *testing.T, db *gorm.DB) int32 {
	var typeRecord schema.Type
	err := db.Where("name = ?", agentcatalogservice.AgentTemplateArtifactTypeName).First(&typeRecord).Error
	require.NoError(t, err, "Failed to query AgentTemplateArtifact type")
	return typeRecord.ID
}
