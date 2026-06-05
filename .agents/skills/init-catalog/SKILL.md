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

This requires **three pieces** — all methods on the repository impl, not standalone functions.
`GenericRepository.List` treats a non-nil `ApplyCustomOrdering` callback as a *replacement*
for its built-in `ApplyStandardPagination`. If the callback doesn't apply pagination,
no LIMIT is applied and all rows are returned regardless of `pageSize`.

**5a. OrderByColumns map** — defines which `orderBy` values the API accepts:

```go
var XxxOrderByColumns = map[string]string{
    "ID":               "id",
    "CREATE_TIME":      "create_time_since_epoch",
    "LAST_UPDATE_TIME": "last_update_time_since_epoch",
    "NAME":             "name",
    "id":               "id",
}
```

**5b. `applyCustomOrdering`** — handle NAME ordering explicitly (it uses a special
helper for cursor-based pagination on the name column), fall back to standard for
everything else:

```go
func (r *XxxRepositoryImpl) applyXxxCustomOrdering(query *gorm.DB, listOptions *models.XxxListOptions) *gorm.DB {
    db := r.GetConfig().DB
    TABLE := utils.GetTableName(db, &SCHEMA_TYPE{})
    orderBy := listOptions.GetOrderBy()

    if orderBy == "NAME" {
        return pagination.ApplyNameOrdering(query, TABLE, listOptions.GetSortOrder(), listOptions.GetNextPageToken(), listOptions.GetPageSize(), false)
    }

    return r.ApplyStandardPagination(query, listOptions, []models.Xxx{})
}
```

Replace TABLE lookup with the correct schema type from the reference table
(e.g., `&schema.Context{}` for context entities).

Add these imports if not already present:
```go
"github.com/kubeflow/hub/catalog/internal/db/pagination"
"github.com/kubeflow/hub/internal/platform/db/utils"
```

**5c. `ApplyStandardPagination` override** — passes the OrderByColumns map to the
GORM pagination scope:

```go
func (r *XxxRepositoryImpl) ApplyStandardPagination(query *gorm.DB, listOptions *models.XxxListOptions, entities any) *gorm.DB {
    pageSize := listOptions.GetPageSize()
    orderBy := listOptions.GetOrderBy()
    sortOrder := listOptions.GetSortOrder()
    nextPageToken := listOptions.GetNextPageToken()

    pag := &dbmodels.Pagination{
        PageSize:      &pageSize,
        OrderBy:       &orderBy,
        SortOrder:     &sortOrder,
        NextPageToken: &nextPageToken,
    }

    return query.Scopes(scopes.PaginateWithOptions(entities, pag, r.GetConfig().DB, "TABLE", XxxOrderByColumns))
}
```

Replace `"TABLE"` with the GORM table name: `"Context"` for context,
`"Artifact"` for artifact, `"Execution"` for execution.

Add this import if not already present:
```go
"github.com/kubeflow/hub/internal/platform/db/scopes"
```

**Update the constructor** to use the method receiver:
```go
ApplyCustomOrdering: r.applyXxxCustomOrdering,
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

### Save override for TypeID

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

## Phase 5: Implement DB Provider

The generated `db_<name>.go` has a TODO stub. Replace it with a working implementation
that queries the repository and maps results to OpenAPI models.

1. **Export the type**: rename the struct from `db<PascalName>CatalogImpl` to `DB<PascalName>Catalog`
   so the service implementation can reference it.

2. **Add a `List<PascalName>sParams` struct** with fields: Name, SourceIDs, FilterQuery,
   OrderBy, SortOrder, NextPageToken, PageSize.

3. **Add `List<Entity>s` method**: build `ListOptions` from params, call repository `.List()`,
   map each result via a mapping function, return the API list type with pagination.
   For pointer fields on the list struct (check `catalog/pkg/openapi/model_<entity>_list.go`),
   use `&variable` or `apiutils.Of()`.
   The `NextPageToken` from the repository is a plain `string`; only assign to the response
   if non-empty (the API model field is `*string`).

4. **Add `Get<Entity>` method**: parse ID string to int32, call repository `.GetByID()`,
   map via the mapping function, return or 404.

5. **Add `FindSources` method**: iterate `sources.AllSources()`, build `CatalogSourceList`.

6. **Implement `mapDB<Entity>ToAPI` mapping function**: map ID (format int64 → string),
   Name from attributes, then loop over Properties matching by name:
   - String fields: `res.<Field> = prop.StringValue`
   - Boolean fields: `res.<Field> = prop.BoolValue`
   - Array fields: `json.Unmarshal` from `prop.StringValue` into `[]string`

   Reference: `catalog/internal/catalog/modelcatalog/db_catalog.go` function `mapDBModelToAPIModel`.

7. **Remove** the `var _ sharedmodels.CatalogSourceRepository` placeholder and the TODO comments.

## Phase 6: Create OpenAPI Service Implementation

After `gen/openapi-server` runs, an interface `<PascalName>CatalogServiceAPIServicer` exists in
`catalog/internal/server/openapi/api_<name>.go`. Create a service implementation that calls
through to the DB provider.

1. **Read** `catalog/internal/server/openapi/api_<name>.go` to get the exact interface methods
   and their signatures.

2. **Create** `catalog/internal/server/openapi/api_<name>_catalog_service_service.go` with:
   - A struct holding a `*<pkg>.DB<PascalName>Catalog` provider field
   - A constructor `New<PascalName>CatalogServiceAPIService(provider)` that takes the provider
   - Each interface method delegates to the provider:
     - `Find<Entity>s` → call `provider.List<Entity>s(ctx, params)`, return 200 or error
     - `Get<Entity>` → call `provider.Get<Entity>(ctx, id)`, return 200 or 404
     - `GetFilterOptions` → call `provider.GetFilterOptions(ctx)`
     - `FindSources` → call `provider.FindSources(ctx)`
   - For errors, use `api.ErrNotFound` / `api.ErrBadRequest` checks from `github.com/kubeflow/hub/pkg/api`

   Match the exact method signatures from the interface.

## Phase 7: Implement YAML Provider + Wire Loader

Add a YAML provider so the plugin can load data from YAML files at startup.

### 7a: Create the YAML provider

Create `catalog/internal/catalog/<name>catalog/yaml_provider.go` with:

1. A YAML struct matching the data file format (using the primary entity's field names).
   **CRITICAL: Every field MUST have both `yaml` and `json` struct tags.**
   The `k8s.io/apimachinery/pkg/util/yaml.Unmarshal` used to read YAML data files converts
   YAML→JSON internally, then uses `encoding/json` to populate the struct. Without `json`
   tags, fields with underscores or camelCase names silently fail to parse (Go's JSON
   decoder only falls back to case-insensitive matching, which doesn't handle underscores).

   ```go
   type yaml<PascalName> struct {
       Name        string            `yaml:"name" json:"name"`
       Description *string           `yaml:"description,omitempty" json:"description,omitempty"`
       // ... string/bool/array fields from the spec
       CustomProperties map[string]any `yaml:"customProperties,omitempty" json:"customProperties,omitempty"`
   }

   type yaml<PascalName>Catalog struct {
       Source string           `yaml:"source" json:"source"`
       Items  []yaml<PascalName> `yaml:"<entity_plural>" json:"<entity_plural>"`
   }
   ```

2. A provider function that reads a YAML file and converts each entry to a domain entity
   (using the `models.<Entity>Impl` type with Properties from the YAML fields).

3. Registration via `init()` — but since this is a generic plugin using `PluginSource`,
   the loader already knows the source type from `source.Type`. The provider should be
   called when `source.Type == "yaml"`.

### 7b: Wire PerformLeaderOperations in the loader

Update the generated `loader.go` to replace the TODO in `PerformLeaderOperations`:

```go
func (l *<PascalName>Loader) PerformLeaderOperations(ctx context.Context, allKnownSourceIDs mapset.Set[string]) error {
    glog.Infof("%s loader performing leader operations", "<name>")

    ctx, cancel := context.WithCancel(ctx)
    l.setCloser(cancel)

    allSources := l.Sources.AllSources()

    for id, source := range allSources {
        if !source.IsEnabled() {
            basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, id, basecatalog.SourceStatusDisabled, "")
            continue
        }

        if source.Type != "yaml" {
            glog.Warningf("unknown %s provider type: %s", "<name>", source.Type)
            basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, id, basecatalog.SourceStatusError, "unknown provider type: "+source.Type)
            continue
        }

        if err := l.loadFromYAML(ctx, id, source); err != nil {
            glog.Errorf("error loading %s from source %s: %v", "<name>", id, err)
            basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, id, basecatalog.SourceStatusError, err.Error())
            continue
        }

        basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, id, basecatalog.SourceStatusAvailable, "")
    }

    glog.Infof("%s loader leader operations complete", "<name>")
    return nil
}
```

The `loadFromYAML` method reads the YAML file from `source.Properties["yamlCatalogPath"]`
(resolving relative paths via `source.Origin`), parses each entry, converts to a domain entity,
and saves via `l.services.<Entity>Repository.Save(entity, nil)`.

Reference: `catalog/internal/catalog/mcpcatalog/providers.go` for path resolution and
`catalog/internal/catalog/mcpcatalog/loader.go` `updateDatabase` for the save pattern.

## Phase 8: Wire RegisterRoutes

Update `catalog/internal/plugins/<name>/plugin.go`:

1. **Add import**: `"github.com/kubeflow/hub/catalog/internal/server/openapi"`

2. **Replace** the `RegisterRoutes` method:

```go
func (p *Plugin) RegisterRoutes(router chi.Router) error {
    provider := <pkg>.NewDB<PascalName>Catalog(p.services, p.loader.Sources)
    svc := openapi.New<PascalName>CatalogServiceAPIService(provider)
    ctrl := openapi.New<PascalName>CatalogServiceAPIController(svc)

    for _, route := range ctrl.OrderedRoutes() {
        router.Method(route.Method, route.Pattern, route.HandlerFunc)
    }

    return nil
}
```

## Phase 9: Verify Compilation

```bash
go build ./catalog/...
```

If compilation fails, read the error output, fix the issues (typically unused or missing imports,
pointer vs value mismatches on list struct fields), and retry.

## Phase 10: Report

Print a summary with:

1. **Created** — list all files catalog-gen created
2. **Modified** — entity service panics replaced, db_provider wired, loader implemented, plugin.go wired
3. **Generated** — OpenAPI server stubs + service implementation
4. **Build** — pass/fail

Then print **Next Steps** the developer must complete manually:

```
Next steps:
  1. Edit the OpenAPI spec (api/openapi/src/plugins/<name>.yaml) to add entity-specific fields.
     IMPORTANT: The generated spec uses a flat `type: object` schema. For consistency with
     model and MCP plugins, update it to use `allOf` with `$ref: BaseResource` so the entity
     inherits customProperties, externalId, and timestamp fields:

       CatalogXxx:
         description: ...
         allOf:
           - $ref: "#/components/schemas/BaseResource"
           - type: object
             properties:
               name:
                 type: string
               # ... entity-specific fields ...

     BaseResource is defined in api/openapi/src/lib/common.yaml and provides:
       customProperties, description, externalId, createTimeSinceEpoch, lastUpdateTimeSinceEpoch

     For URL fields use `format: uri`. Follow MCP's camelCase naming for new fields
     (e.g., repositoryUrl, not repository_url) — exception: source_id stays snake_case
     because it's a cross-plugin convention.

  2. Run /sync-catalog to propagate spec changes
  3. Run /catalog-sample-data to generate test data
  4. Add list filter logic in apply<Entity>ListFilters if needed
  5. Add entity-specific properties to *_entity_mappings.go
```
