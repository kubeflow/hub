---
name: init-catalog
description: >
  Scaffold a new catalog plugin: run catalog-gen, replace all panic("TODO")
  stubs with minimal working implementations, generate OpenAPI server stubs,
  and verify the build compiles. Produces a plugin that starts without crashes.
user-invocable: true
---

# Init Catalog Plugin

Scaffold a new catalog plugin end-to-end so it compiles and starts without panics.

## Phase 1: Gather Inputs

Ask the user for all three in a single prompt:

- **Plugin name** — snake_case (e.g., `agent`, `dataset`, `pipeline`)
- **Description** — human-readable (e.g., "Agent catalog")
- **Entities** — comma-separated `Name:type` pairs where type is `context`, `artifact`, or `execution` (e.g., `CatalogAgent:context,CatalogAgentArtifact:artifact`)

Validate before proceeding:
- Name contains only lowercase letters, digits, and underscores
- At least one entity is provided
- Each entity has exactly one colon separating PascalCase name from type
- Each type is one of: `context`, `artifact`, `execution`

## Phase 2: Run catalog-gen

```bash
make -C catalog gen/catalog-plugin NAME=<name> DESCRIPTION="<desc>" ENTITIES=<entity1>,<entity2>
```

Check exit code. Report files created.

## Phase 3: Replace Panic TODOs

All 8 panics are in the generated entity service files at:
`catalog/internal/catalog/<name>catalog/service/<entity_snake>.go`

**Read each generated service file first** to get the exact function signatures with concrete entity names. Then replace each panic using the recipes below.

### Type Reference Table

Use this table to select the correct types, mappers, and table names for each entity's datastore type:

| Datastore Type | Schema Type | Property Type | ToProperties Mapper | ToEntity Mapper | Table | Property Table | Join Field | Name Type |
|---|---|---|---|---|---|---|---|---|
| context | `schema.Context` | `schema.ContextProperty` | `service.MapPropertiesToContextProperty(prop, entityID, isCustom)` | `service.MapContextPropertyToProperties(prop)` | `"Context"` | `"ContextProperty"` | `context_id` | `string` |
| artifact | `schema.Artifact` | `schema.ArtifactProperty` | `service.MapPropertiesToArtifactProperty(prop, entityID, isCustom)` | `service.MapArtifactPropertyToProperties(prop)` | `"Artifact"` | `"ArtifactProperty"` | `artifact_id` | `*string` |
| execution | `schema.Execution` | `schema.ExecutionProperty` | `service.MapPropertiesToExecutionProperty(prop, entityID, isCustom)` | `service.MapExecutionPropertyToProperties(prop)` | `"Execution"` | `"ExecutionProperty"` | `execution_id` | `*string` |

The `service` alias refers to `github.com/kubeflow/hub/internal/platform/db/repository` (already imported in the generated file).

### Reference implementations

Before writing code, read these files for the exact patterns:

- **Context entities**: `catalog/internal/catalog/mcpcatalog/service/mcp_server.go`
- **Artifact entities**: `catalog/internal/catalog/modelcatalog/service/catalog_model_artifact.go`
- **Execution entities**: `catalog/internal/catalog/mcpcatalog/service/mcp_server_tool.go`

### Panic 1: `mapEntityToSchema`

Map domain entity to database schema struct. Copy ID, TypeID, Name, ExternalID, timestamps with nil-safety.

For **context** entities (Name is `string`, non-pointer):
```go
func mapXxxToContext(entity models.Xxx) schema.Context {
    attrs := entity.GetAttributes()
    ctx := schema.Context{}
    if typeID := entity.GetTypeID(); typeID != nil {
        ctx.TypeID = *typeID
    }
    if entity.GetID() != nil {
        ctx.ID = *entity.GetID()
    }
    if attrs != nil {
        if attrs.Name != nil {
            ctx.Name = *attrs.Name
        }
        ctx.ExternalID = attrs.ExternalID
        if attrs.CreateTimeSinceEpoch != nil {
            ctx.CreateTimeSinceEpoch = *attrs.CreateTimeSinceEpoch
        }
        if attrs.LastUpdateTimeSinceEpoch != nil {
            ctx.LastUpdateTimeSinceEpoch = *attrs.LastUpdateTimeSinceEpoch
        }
    }
    return ctx
}
```

For **artifact** entities (Name is `*string`, pointer; also has URI and State):
```go
func mapXxxToArtifact(entity models.Xxx) schema.Artifact {
    attrs := entity.GetAttributes()
    art := schema.Artifact{}
    if typeID := entity.GetTypeID(); typeID != nil {
        art.TypeID = *typeID
    }
    if entity.GetID() != nil {
        art.ID = *entity.GetID()
    }
    if attrs != nil {
        art.Name = attrs.Name
        art.ExternalID = attrs.ExternalID
        if attrs.CreateTimeSinceEpoch != nil {
            art.CreateTimeSinceEpoch = *attrs.CreateTimeSinceEpoch
        }
        if attrs.LastUpdateTimeSinceEpoch != nil {
            art.LastUpdateTimeSinceEpoch = *attrs.LastUpdateTimeSinceEpoch
        }
    }
    return art
}
```

For **execution** entities (Name is `*string`, pointer):
```go
func mapXxxToExecution(entity models.Xxx) schema.Execution {
    attrs := entity.GetAttributes()
    exec := schema.Execution{}
    if typeID := entity.GetTypeID(); typeID != nil {
        exec.TypeID = *typeID
    }
    if entity.GetID() != nil {
        exec.ID = *entity.GetID()
    }
    if attrs != nil {
        exec.Name = attrs.Name
        if attrs.CreateTimeSinceEpoch != nil {
            exec.CreateTimeSinceEpoch = *attrs.CreateTimeSinceEpoch
        }
        if attrs.LastUpdateTimeSinceEpoch != nil {
            exec.LastUpdateTimeSinceEpoch = *attrs.LastUpdateTimeSinceEpoch
        }
    }
    return exec
}
```

### Panic 2: `mapSchemaToEntity`

Reverse mapping — build domain entity from schema + properties. Split properties into Properties and CustomProperties using the ToEntity mapper from the reference table.

```go
func mapSchemaToXxx(schemaEntity SCHEMA_TYPE, props []PROPERTY_TYPE) models.Xxx {
    entity := &models.XxxImpl{
        ID:     &schemaEntity.ID,
        TypeID: &schemaEntity.TypeID,
        Attributes: &models.XxxAttributes{
            Name:                     NAME_FIELD,  // &schemaEntity.Name for context, schemaEntity.Name for artifact/execution
            ExternalID:               schemaEntity.ExternalID,
            CreateTimeSinceEpoch:     &schemaEntity.CreateTimeSinceEpoch,
            LastUpdateTimeSinceEpoch: &schemaEntity.LastUpdateTimeSinceEpoch,
        },
    }

    properties := []dbmodels.Properties{}
    customProperties := []dbmodels.Properties{}
    for _, prop := range props {
        mapped := TO_ENTITY_MAPPER(prop)  // e.g. service.MapContextPropertyToProperties(prop)
        if prop.IsCustomProperty {
            customProperties = append(customProperties, mapped)
        } else {
            properties = append(properties, mapped)
        }
    }
    entity.Properties = &properties
    entity.CustomProperties = &customProperties
    return entity
}
```

For context entities, Name field is: `&schemaEntity.Name` (take address of string).
For artifact/execution entities, Name field is: `schemaEntity.Name` (already a pointer).

### Panic 3: `mapEntityToProperties`

Iterate entity properties and custom properties, convert each using the ToProperties mapper from the reference table.

```go
func mapXxxToProperties(entity models.Xxx, entityID int32) []PROPERTY_TYPE {
    var properties []PROPERTY_TYPE
    if entity.GetProperties() != nil {
        for _, prop := range *entity.GetProperties() {
            properties = append(properties, TO_PROPERTIES_MAPPER(prop, entityID, false))
        }
    }
    if entity.GetCustomProperties() != nil {
        for _, prop := range *entity.GetCustomProperties() {
            properties = append(properties, TO_PROPERTIES_MAPPER(prop, entityID, true))
        }
    }
    return properties
}
```

### Panic 4: `applyListFilters`

No-op — return query unchanged. Filtering can be added later.

```go
func applyXxxListFilters(query *gorm.DB, _ *models.XxxListOptions) *gorm.DB {
    return query
}
```

### Panic 5: `applyCustomOrdering`

No-op — return query unchanged. Falls back to GenericRepository standard pagination.

```go
func applyXxxCustomOrdering(query *gorm.DB, _ *models.XxxListOptions) *gorm.DB {
    return query
}
```

### Panic 6: `DeleteBySource`

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

### Panic 7: `DeleteByID`

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

### Panic 8: `GetDistinctSourceIDs`

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

### Imports to add

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

## Phase 4: Generate OpenAPI Stubs

Three make targets must run in order:

```bash
# 1. Rebuild the merged catalog spec to include the new plugin
make api/openapi/catalog.yaml

# 2. Generate server stubs (controller, routes, type_asserts)
make -C catalog gen/openapi-server

# 3. Generate client model types (type_asserts.go references these)
make -C catalog gen/openapi
```

Step 3 is critical — `gen/openapi-server` produces `type_asserts.go` which references
client model types (e.g., `model.Agent`, `model.AgentList`) from `catalog/pkg/openapi/`.
Without `gen/openapi`, those types don't exist and the build fails.

Verify that these files were created or updated:
- `catalog/internal/server/openapi/api_<name>_catalog_service.go`
- `catalog/internal/server/openapi/api_<name>.go`
- `catalog/pkg/openapi/model_<entity_snake>.go` (one per entity)

## Phase 5: Create OpenAPI Service Implementation

After `gen/openapi-server` runs, an interface `<PascalName>CatalogServiceAPIServicer` exists in
`catalog/internal/server/openapi/api_<name>.go`. Create a minimal service implementation
that satisfies this interface and returns empty/stub responses.

1. **Read** `catalog/internal/server/openapi/api_<name>.go` to get the exact interface methods and their signatures.

2. **Read** `catalog/pkg/openapi/model_<primary_entity_snake>.go` and the `_list.go` variant to check whether list struct fields are pointers or values (e.g., `Size *int32` vs `Size int32`). For pointer fields, use `apiutils.Of(int32(0))`.

3. **Create** `catalog/internal/server/openapi/api_<name>_catalog_service_service.go` with:

```go
package openapi

import (
    "context"
    "net/http"

    model "github.com/kubeflow/hub/catalog/pkg/openapi"
    "github.com/kubeflow/hub/internal/platform/apiutils"
)

type <PascalName>CatalogServiceAPIService struct{}

func New<PascalName>CatalogServiceAPIService() *<PascalName>CatalogServiceAPIService {
    return &<PascalName>CatalogServiceAPIService{}
}
```

Then implement each method from the interface:

- **Find<Entity>s** — return an empty list with pagination fields. Use `apiutils.Of()` for pointer fields.
- **Get<Entity>** — return 404 with `model.Error{Code: "not_found", Message: "<entity> not found"}`.
- **GetFilterOptions** — return empty `model.FilterOptionsList{Filters: &options}` where `options` is an empty map.
- **FindSources** — return empty `model.CatalogSourceList{Items: []model.CatalogSource{}}`.

Match the exact method signatures from the interface — parameter types and counts vary depending on the OpenAPI spec (query params like name, source, orderBy, sortOrder, etc.).

Remove the `apiutils` import if no pointer fields are needed.

## Phase 6: Wire RegisterRoutes

Update `catalog/internal/plugins/<name>/plugin.go`:

1. **Add import**: `"github.com/kubeflow/hub/catalog/internal/server/openapi"`

2. **Replace** the `RegisterRoutes` method:

```go
func (p *Plugin) RegisterRoutes(router chi.Router) error {
    svc := openapi.New<PascalName>CatalogServiceAPIService()
    ctrl := openapi.New<PascalName>CatalogServiceAPIController(svc)

    for _, route := range ctrl.OrderedRoutes() {
        router.Method(route.Method, route.Pattern, route.HandlerFunc)
    }

    return nil
}
```

## Phase 7: Verify Compilation

```bash
go build ./catalog/...
```

If compilation fails, read the error output, fix the issues (typically unused or missing imports, pointer vs value mismatches on list struct fields), and retry.

## Phase 8: Report

Print a summary with:

1. **Created** — list all files catalog-gen created
2. **Modified** — list entity service files where panics were replaced, plugin.go wiring
3. **Generated** — OpenAPI server stubs + service implementation
4. **Build** — pass/fail

Then print **Next Steps** the developer must complete manually:

```
Next steps:
  1. Edit the OpenAPI spec (api/openapi/src/plugins/<name>.yaml) to add entity-specific fields
  2. Re-run: make api/openapi/catalog.yaml && make -C catalog gen/openapi-server && make -C catalog gen/openapi
  3. Connect the service implementation to the DB provider (replace stub responses with real queries)
  4. Add list filter logic in apply<Entity>ListFilters if needed
  5. Add entity-specific properties to *_entity_mappings.go
  6. Implement PerformLeaderOperations in the loader
```
