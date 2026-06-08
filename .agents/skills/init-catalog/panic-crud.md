# Panic Recipes: CRUD Operations (Panics 6–8) + Imports & Save Override

These panics handle delete and query operations on entities.
Use the **Type Reference Table** in SKILL.md to select the correct table names and join fields.

## Panic 6: `DeleteBySource`

Delete all entities of this type from a given source. Use a transaction with a subquery joining the property table.

```go
func (r *XxxRepositoryImpl) DeleteBySource(sourceID string) error {
    config := r.GetConfig()
    return config.DB.Transaction(func(tx *gorm.DB) error {
        query := `DELETE FROM "TABLE" WHERE id IN (
            SELECT "TABLE".id FROM "TABLE"
            INNER JOIN "PROPERTY_TABLE" ON "TABLE".id="PROPERTY_TABLE".JOIN_FIELD
            AND "PROPERTY_TABLE".name='source_id'
            WHERE "PROPERTY_TABLE".string_value=? AND "TABLE".type_id=?
        )`
        return tx.Exec(query, sourceID, config.TypeID).Error
    })
}
```

Replace TABLE, PROPERTY_TABLE, and JOIN_FIELD from the reference table.

## Panic 7: `DeleteByID`

Delete a single entity by ID.

```go
func (r *XxxRepositoryImpl) DeleteByID(id int32) error {
    config := r.GetConfig()
    result := config.DB.Where("id = ? AND type_id = ?", id, config.TypeID).Delete(&SCHEMA_TYPE{})
    if result.Error != nil {
        return result.Error
    }
    if result.RowsAffected == 0 {
        return fmt.Errorf("%w: id %d", config.NotFoundError, id)
    }
    return nil
}
```

Replace SCHEMA_TYPE with the schema type from the reference table (e.g., `schema.Context{}`).

## Panic 8: `GetDistinctSourceIDs`

Query all unique source_id values for this entity type.

```go
func (r *XxxRepositoryImpl) GetDistinctSourceIDs() ([]string, error) {
    config := r.GetConfig()
    var sourceIDs []string
    query := `SELECT DISTINCT "PROPERTY_TABLE".string_value FROM "PROPERTY_TABLE"
        INNER JOIN "TABLE" ON "PROPERTY_TABLE".JOIN_FIELD = "TABLE".id
        WHERE "PROPERTY_TABLE".name='source_id' AND "TABLE".type_id=?`
    rows, err := config.DB.Raw(query, config.TypeID).Rows()
    if err != nil {
        return nil, fmt.Errorf("error querying distinct source IDs: %w", dbutil.SanitizeDatabaseError(err))
    }
    defer rows.Close()
    for rows.Next() {
        var id string
        if err := rows.Scan(&id); err != nil {
            return nil, fmt.Errorf("error scanning source ID: %w", dbutil.SanitizeDatabaseError(err))
        }
        sourceIDs = append(sourceIDs, id)
    }
    return sourceIDs, rows.Err()
}
```

Replace TABLE, PROPERTY_TABLE, and JOIN_FIELD from the reference table.

## Imports to add

After replacing panics, ensure these imports are present in each modified file:

```go
"fmt"
dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
"github.com/kubeflow/hub/internal/platform/db/dbutil"
```

The following should already be present from the template:
```go
"github.com/kubeflow/hub/internal/platform/db/schema"
service "github.com/kubeflow/hub/internal/platform/db/repository"
"gorm.io/gorm"
```

Remove any unused imports after edits.

## Save override for TypeID

The `GenericRepository.Save` does NOT automatically set `TypeID` on entities. Without this,
entities saved by the YAML loader get `type_id = 0` and won't be found by List/Get queries.

Add a `Save` override to each entity's repository impl:

```go
func (r *XxxRepositoryImpl) Save(entity models.Xxx, parentResourceID *int32) (models.Xxx, error) {
    config := r.GetConfig()
    if entity.GetTypeID() == nil && config.TypeID > 0 {
        entity.SetTypeID(config.TypeID)
    }
    return r.GenericRepository.Save(entity, parentResourceID)
}
```

Reference: `catalog/internal/catalog/modelcatalog/service/catalog_model.go` `Save` method.
