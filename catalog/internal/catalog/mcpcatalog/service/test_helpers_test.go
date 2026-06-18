package service

import (
	"errors"
	"testing"

	"github.com/kubeflow/hub/internal/platform/datastore"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const (
	testMCPServerTypeName     = "kf.MCPServer"
	testMCPServerToolTypeName = "kf.MCPServerTool"
)

// testDatastoreSpec returns a minimal datastore spec for MCP catalog tests.
// This avoids importing catalog/internal/db/service which would cause an import cycle.
func testDatastoreSpec() *datastore.Spec {
	return datastore.NewSpec().
		AddContext(testMCPServerTypeName, datastore.NewSpecType(NewMCPServerRepository).
			AddString("source_id").
			AddString("base_name").
			AddString("displayName").
			AddString("description").
			AddString("provider").
			AddString("license").
			AddString("license_link").
			AddString("logo").
			AddString("readme").
			AddString("version").
			AddStruct("tags").
			AddStruct("transports").
			AddString("deploymentMode").
			AddBoolean("verifiedSource").
			AddBoolean("secureEndpoint").
			AddBoolean("sast").
			AddBoolean("readOnlyTools"),
		).
		AddExecution(testMCPServerToolTypeName, datastore.NewSpecType(NewMCPServerToolRepository).
			AddString("accessType").
			AddString("description").
			AddString("externalId").
			AddString("parameters"),
		)
}

// getMCPServerTypeID gets the MCPServer type ID from the database
func getMCPServerTypeID(t *testing.T, db *gorm.DB) int32 {
	var typeRecord schema.Type
	err := db.Where("name = ?", testMCPServerTypeName).First(&typeRecord).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			typeRecord = schema.Type{
				Name: testMCPServerTypeName,
			}
			err = db.Create(&typeRecord).Error
			require.NoError(t, err)
		} else {
			require.NoError(t, err)
		}
	}
	return typeRecord.ID
}

// getMCPServerToolTypeID gets the MCPServerTool type ID from the database
func getMCPServerToolTypeID(t *testing.T, db *gorm.DB) int32 {
	var typeRecord schema.Type
	err := db.Where("name = ?", testMCPServerToolTypeName).First(&typeRecord).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			typeRecord = schema.Type{
				Name: testMCPServerToolTypeName,
			}
			err = db.Create(&typeRecord).Error
			require.NoError(t, err)
		} else {
			require.NoError(t, err)
		}
	}
	return typeRecord.ID
}
