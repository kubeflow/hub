package service

import (
	"fmt"

	"github.com/kubeflow/hub/internal/platform/db/dbutil"
	"gorm.io/gorm"
)

// GetDistinctSourceIDs returns all unique source_id property values for catalog
// entities of the given typeID. entityTable and propTable are the MLMD schema
// table names; propJoinColumn is the column on propTable that references the
// entity (e.g. "context_id"). A custom property that happens to be named
// "source_id" is not the entity's source and is ignored.
func GetDistinctSourceIDs(db *gorm.DB, entityTable, propTable, propJoinColumn string, typeID int32) ([]string, error) {
	var sourceIDs []string
	err := db.Table(propTable+" p").
		Select("DISTINCT p.string_value").
		Joins("INNER JOIN "+entityTable+" e ON p."+propJoinColumn+" = e.id").
		Where("p.name = ? AND p.is_custom_property = ? AND e.type_id = ?", "source_id", false, typeID).
		Pluck("string_value", &sourceIDs).Error
	if err != nil {
		err = dbutil.SanitizeDatabaseError(err)
		return nil, fmt.Errorf("error querying distinct source IDs: %w", err)
	}
	return sourceIDs, nil
}
