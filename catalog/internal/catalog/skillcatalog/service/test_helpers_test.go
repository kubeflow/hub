package service

import (
	"errors"
	"os"
	"testing"

	"github.com/kubeflow/hub/internal/platform/datastore"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"github.com/kubeflow/hub/internal/testutils"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	os.Exit(testutils.TestMainPostgresHelper(m))
}

func testDatastoreSpec() *datastore.Spec {
	return datastore.NewSpec().
		AddContext(SkillTypeName, datastore.NewSpecType(NewSkillRepository).
			AddString("source_id").
			AddString("name").
			AddString("description").
			AddString("repository").
			AddString("path").
			AddString("version").
			AddString("resolvedCommit").
			AddString("trustTier").
			AddString("provider").
			AddString("category").
			AddString("license").
			AddString("author").
			AddString("compatibility").
			AddString("readme").
			AddString("labels").
			AddString("allowedTools").
			AddString("supportingFiles").
			AddInt("bodyLineCount").
			AddString("configDigest"),
		)
}

func getSkillTypeID(t *testing.T, db *gorm.DB) int32 {
	return getTypeIDByName(t, db, SkillTypeName)
}

func getTypeIDByName(t *testing.T, db *gorm.DB, name string) int32 {
	var typeRecord schema.Type
	err := db.Where("name = ?", name).First(&typeRecord).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			typeRecord = schema.Type{
				Name: name,
			}
			err = db.Create(&typeRecord).Error
			require.NoError(t, err)
		} else {
			require.NoError(t, err)
		}
	}
	return typeRecord.ID
}
