# Panic Recipes: CRUD Operations (Panics 6–8) + Imports & Save Override

These panics handle delete and query operations on entities.
Use the **Type Reference Table** in SKILL.md to select the correct table names and join fields.

## Panic 6: `DeleteBySource`

Delete all entities of this type from a given source. Use a GORM fluent builder with
`utils.GetTableName()` for database-portable table names.

```go
func (r *XxxRepositoryImpl) DeleteBySource(sourceID string) error {
    config := r.GetConfig()
    tableName := utils.GetTableName(config.DB, &SCHEMA_TYPE{})
    propTableName := utils.GetTableName(config.DB, &SCHEMA_PROPERTY_TYPE{})

    subQuery := config.DB.Table(tableName).
        Select(tableName + ".id").
        Joins("INNER JOIN " + propTableName + " ON " +
            tableName + ".id = " + propTableName + ".JOIN_FIELD").
        Where(propTableName + ".name = ? AND " +
            propTableName + ".string_value = ? AND " +
            tableName + ".type_id = ?",
            "source_id", sourceID, config.TypeID)

    return config.DB.Where("id IN (?)", subQuery).Delete(&SCHEMA_TYPE{}).Error
}
```

Replace SCHEMA_TYPE, SCHEMA_PROPERTY_TYPE, and JOIN_FIELD from the reference table.
Reference: `catalog/internal/catalog/mcpcatalog/service/mcp_server.go` `DeleteBySource`.

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

Query all unique source_id values for this entity type. Use the GORM fluent API with
`utils.GetTableName()` for database-portable table names.

```go
func (r *XxxRepositoryImpl) GetDistinctSourceIDs() ([]string, error) {
    config := r.GetConfig()
    var sourceIDs []string

    propTableName := utils.GetTableName(config.DB, &SCHEMA_PROPERTY_TYPE{})
    tableName := utils.GetTableName(config.DB, &SCHEMA_TYPE{})

    err := config.DB.Table(propTableName + " cp").
        Select("DISTINCT cp.string_value").
        Joins("INNER JOIN " + tableName + " c ON cp.JOIN_FIELD = c.id").
        Where("cp.name = ? AND c.type_id = ?", "source_id", config.TypeID).
        Pluck("string_value", &sourceIDs).Error

    if err != nil {
        err = dbutil.SanitizeDatabaseError(err)
        return nil, fmt.Errorf("error querying distinct source IDs: %w", err)
    }
    return sourceIDs, nil
}
```

Replace SCHEMA_TYPE, SCHEMA_PROPERTY_TYPE, and JOIN_FIELD from the reference table.
Reference: `catalog/internal/catalog/mcpcatalog/service/mcp_server.go` `GetDistinctSourceIDs`.

## Imports to add

After replacing panics, ensure these imports are present in each modified file:

```go
"fmt"
dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
"github.com/kubeflow/hub/internal/platform/db/dbutil"
"github.com/kubeflow/hub/internal/platform/db/utils"
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

Add a `Save` override to each entity's repository impl. The signature differs by datastore type:

### Context entities (1-param Save)

```go
func (r *XxxRepositoryImpl) Save(entity models.Xxx) (models.Xxx, error) {
    config := r.GetConfig()
    if entity.GetTypeID() == nil && config.TypeID > 0 {
        entity.SetTypeID(config.TypeID)
    }
    return r.GenericRepository.Save(entity, nil)
}
```

Reference: `catalog/internal/catalog/modelcatalog/service/catalog_model.go` `Save` method,
`catalog/internal/catalog/mcpcatalog/service/mcp_server.go` `Save` method.

### Artifact and execution entities (2-param Save)

```go
func (r *XxxRepositoryImpl) Save(entity models.Xxx, parentResourceID *int32) (models.Xxx, error) {
    config := r.GetConfig()
    if entity.GetTypeID() == nil && config.TypeID > 0 {
        entity.SetTypeID(config.TypeID)
    }
    return r.GenericRepository.Save(entity, parentResourceID)
}
```

Reference: `catalog/internal/catalog/modelcatalog/service/catalog_model_artifact.go` `Save` method,
`catalog/internal/catalog/mcpcatalog/service/mcp_server_tool.go` `Save` method.
