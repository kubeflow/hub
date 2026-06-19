---
name: sync-catalog
description: >
  Sync a catalog plugin's generated code and domain layer after OpenAPI spec
  changes. Regenerates stubs, detects new fields, updates datastore entries,
  entity mappings, and service implementation. Interactive — asks the user
  which new fields to persist and make filterable.
user-invocable: true
---

# Sync Catalog Plugin

Propagate OpenAPI spec changes through all downstream files for a catalog plugin.

## Phase 1: Identify Plugin

Ask the user which plugin to sync. Validate by checking that these exist:
- `api/openapi/src/plugins/<name>.yaml` — the spec
- `catalog/internal/plugins/<name>/plugin.go` — the plugin entry point
- `catalog/internal/catalog/<name>catalog/` — the domain package

## Phase 2: Regenerate Auto-Generated Code

Run these three make targets in order:

```bash
make api/openapi/catalog.yaml
make -C catalog gen/openapi-server
make -C catalog gen/openapi
```

Report what was regenerated. Check `go build ./catalog/...` — if it fails, proceed to Phase 4 (service sync) first since interface mismatches are the most common cause.

**Note:** `catalog/internal/server/openapi/type_asserts_overrides.go` is a hand-maintained file that overrides auto-generated assert functions for polymorphic types (e.g., `AssertCatalogArtifactRequired`, `AssertFilterOptionRequired`). If `gen/openapi-server` regenerates `type_asserts.go` and the build fails with undefined assert function errors, check whether this overrides file needs updating for the new types.

## Phase 3: Sync Datastore Entries + Entity Mappings

### 3a: Detect new fields

Read the plugin's OpenAPI spec (`api/openapi/src/plugins/<name>.yaml`) and extract all properties from each entity schema under `components.schemas`.

Read the current `DatastoreEntries` in `catalog/internal/plugins/<name>/plugin.go` and extract the registered property names (the string arguments to `.AddString()`, `.AddStruct()`, `.AddBoolean()`).

Compute the diff: fields in the spec but not in DatastoreEntries.

Exclude these fields from the diff (they are handled automatically by the schema layer, not as properties):
- `id`, `name`, `externalId`, `external_id`
- `createTimeSinceEpoch`, `lastUpdateTimeSinceEpoch`
- `customProperties`, `custom_properties`
- `nextPageToken`, `pageSize`, `size`, `items` (list-level fields)

### 3b: Ask user which fields to persist

Present the diff to the user. For each new field show:
- Field name
- OpenAPI type
- Suggested `Add*()` call based on type:
  - `string` → `AddString`
  - `boolean` → `AddBoolean`
  - `integer` / `number` → `AddString` (stored as string property by convention)
  - `array` / `object` → `AddStruct`

Ask: "Which of these new fields should be persisted in the datastore?" (multi-select)

### 3c: Apply changes

For each confirmed field, update three files:

**1. DatastoreEntries** in `catalog/internal/plugins/<name>/plugin.go`:
- Find the entity's `DatastoreEntry` block
- Add the new `.Add*("<field_name>")` call before the closing comma

**2. Entity mappings** in `catalog/internal/catalog/<name>catalog/service/entity_mappings_<entity>.go`:
- Add entry to the properties map variable:
  ```go
  "<field_name>": {Location: filter.PropertyTable, ValueType: filter.<ValueType>, Column: "<field_name>"},
  ```
  Where ValueType is:
  - `StringValueType` for string/integer/number
  - `BoolValueType` for boolean
  - `ArrayValueType` for array

**3. Entity mappings test** in `catalog/internal/catalog/<name>catalog/service/entity_mappings_<entity>_test.go`:
- Add entry to the expected properties map with the same definition

### 3d: Ask about filterability

Ask the user: "Which of these fields should be directly filterable as query parameters?" (multi-select from the persisted fields)

For filterable fields:
- Add a field to the `ListOptions` struct in `catalog/internal/catalog/<name>catalog/models/<entity>.go`
- Add a WHERE clause in `apply<Entity>ListFilters` in `catalog/internal/catalog/<name>catalog/service/<entity>.go`:
  ```go
  if listOptions.<FieldName> != nil {
      query = query.Where("<field_name> = ?", *listOptions.<FieldName>)
  }
  ```

## Phase 4: Sync Service Implementation

Read the regenerated service interface from `catalog/internal/server/openapi/api_<name>.go`.
Read the current service implementation from `catalog/internal/server/openapi/api_<name>_catalog_service_service.go`.

Compare method signatures:

- **Changed signatures**: Update the implementation method to match the new interface signature. Keep the existing body logic, just fix the parameters.
- **New methods**: Add a stub implementation returning an empty/not-found response (same pattern as init-catalog Phase 6).
- **Removed methods**: Delete the implementation method.

If the service implementation file doesn't exist yet (e.g., it was deleted during a regen), recreate it following the init-catalog Phase 6 pattern.

## Phase 5: Sync DB Provider + Property-to-API Mapping

The db_provider (`catalog/internal/catalog/<name>catalog/db_<name>.go`) is the bridge between
the repository layer and the OpenAPI models. When the spec gains new fields, the property-to-API
mapping function must be updated to include them.

### 5a: Check if db_provider has real query methods

Read `catalog/internal/catalog/<name>catalog/db_<name>.go`. If it still has the generated TODO
stubs (no `List*` or `Get*` methods), wire it up:

1. **Export the type**: rename `dbXxxCatalogImpl` to `DBXxxCatalog` so the service impl can reference it.

2. **Add List method**: query the repository with `ListOptions`, convert results via mapping function, return the API list type with pagination.

3. **Add Get method**: parse the ID, call `GetByID`, convert via mapping function, return the API type or 404.

4. **Add a `ListParams` struct** for the method parameters (Name, SourceIDs, FilterQuery, OrderBy, SortOrder, NextPageToken, PageSize).

Note: Plugins do NOT implement a `FindSources` method — sources are managed by the
model catalog via the shared `/sources` endpoint with `assetType` filtering.

Reference: `catalog/internal/catalog/modelcatalog/db_catalog.go` (`ListModels`, `GetModel`, `mapDBModelToAPIModel`).

### 5b: Sync the property-to-API mapping function

Read the generated client model struct from `catalog/pkg/openapi/model_<entity>.go` to get all
available fields and their Go types.

Read the current `mapDB<Entity>ToAPI` function (or create it if missing). For each field in the
API model that comes from a stored Property, ensure there's a `case` in the Properties switch:

- **String fields** (`*string`): `res.<Field> = prop.StringValue`
- **Boolean fields** (`*bool`): `res.<Field> = prop.BoolValue`
- **Array fields** (`[]string`): JSON unmarshal from `prop.StringValue`
- **Integer fields** (`*int32`): parse from `prop.IntValue` or `prop.StringValue`

When new fields were added to DatastoreEntries in Phase 3, add corresponding cases here.

### 5c: Update RegisterRoutes if needed

If the db_provider type or constructor changed (e.g., from unexported to exported, or new
constructor parameters), update `RegisterRoutes` in `catalog/internal/plugins/<name>/plugin.go`
to match.

Ensure the service constructor receives the provider:
```go
provider := <pkg>.NewDB<PascalName>Catalog(p.services, p.loader.Sources)
svc := openapi.New<PascalName>CatalogServiceAPIService(provider)
```

## Phase 6: Verify + Report

```bash
go build ./catalog/...
go test ./catalog/internal/catalog/<name>catalog/... -count=1
```

If build fails, read errors and fix. Common issues:
- Unused imports after method removal
- Missing imports after adding filter logic (`fmt`, `utils`)
- Interface signature mismatches
- Pointer vs value mismatches on list fields (`*int32` vs `int32`, `*string` vs `string`)

Print summary:

```
Sync complete for plugin "<name>":

Regenerated:
  - api/openapi/catalog.yaml (merged spec)
  - catalog/internal/server/openapi/api_<name>*.go (server stubs)
  - catalog/pkg/openapi/model_*.go (client types)

Datastore entries updated:
  - <list of added fields with Add* type>

Entity mappings updated:
  - <list of added property definitions>

DB provider:
  - <methods added/updated, mapping cases added>

Service implementation:
  - <methods updated/added/removed>

Build: PASS/FAIL
Tests: PASS/FAIL

Remaining manual work:
  - Add complex filter logic for fields that need more than equality matching
  - Update loader if new fields affect data ingestion
```
