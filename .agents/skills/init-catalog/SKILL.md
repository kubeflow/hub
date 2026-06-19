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

## Phase 3: Register Asset Type

New plugins must register their asset type in the shared catalog infrastructure so that
sources can be scoped to the correct plugin via the model catalog's `/sources` endpoint.

The asset type value is derived from the primary entity's snake_case name, pluralized
(e.g., entity `CatalogAgent` → `catalog_agents`, entity `Agent` → `agents`).

### 3a: Add to the OpenAPI enum

Edit `api/openapi/src/catalog.yaml` — find the `CatalogAssetType` enum and add the new value:

```yaml
CatalogAssetType:
  type: string
  enum:
    - models
    - mcp_servers
    - <asset_type_value>    # e.g., agents
  default: models
```

### 3b: Add the Go constant

Edit `catalog/internal/catalog/basecatalog/source_types.go` — add to the `AssetType` constants:

```go
const (
    AssetTypeModels     = "models"
    AssetTypeMCPServers = "mcp_servers"
    AssetType<PascalAssetType> = "<asset_type_value>"  // e.g., AssetTypeAgents = "agents"
)
```

### 3c: Add to the validation map

Edit `catalog/internal/catalog/basecatalog/validation.go` — add to the `validAssetTypes` map
in `ValidateNamedQueries`:

```go
validAssetTypes := map[string]bool{
    AssetTypeModels:     true,
    AssetTypeMCPServers: true,
    AssetType<PascalAssetType>: true,
}
```

Also update the error message format string to include the new constant.

### 3d: Add AssetType field to PluginSource (if not already present)

Check `catalog/internal/catalog/basecatalog/source_types.go` — the `PluginSource` struct
must have an `AssetType` field. If missing, add it:

```go
type PluginSource struct {
    Name       string                      `json:"name" yaml:"name"`
    ID         string                      `json:"id" yaml:"id"`
    Type       string                      `json:"type" yaml:"type"`
    Enabled    *bool                       `json:"enabled,omitempty" yaml:"enabled,omitempty"`
    Properties map[string]any              `json:"properties" yaml:"properties"`
    Labels     []string                    `json:"labels" yaml:"labels"`
    AssetType  *apimodels.CatalogAssetType `json:"assetType,omitempty" yaml:"assetType,omitempty"`

    Origin string `json:"-" yaml:"-"`
}
```

This matches the pattern used by `MCPSource`.

## Phase 4: Replace Panic TODOs


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

### Panic recipes

Each group of panics is documented in a separate file. **Read the file** for each group before implementing.

- **Panics 1–3** (entity ↔ schema mapping): Read `.agents/skills/init-catalog/panic-mapping.md`
- **Panics 4–5** (filtering & ordering): Read `.agents/skills/init-catalog/panic-ordering.md`
- **Panics 6–8** (delete/query ops, imports, Save override): Read `.agents/skills/init-catalog/panic-crud.md`

## Phase 5: Generate OpenAPI Stubs

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

## Phase 6: Implement DB Provider

The generated `db_<name>.go` has a TODO stub. Replace it with a working implementation
that queries the repository and maps results to OpenAPI models.

1. **Export the type**: rename the struct from `db<PascalName>CatalogImpl` to `DB<PascalName>Catalog`
   so the service implementation can reference it.

2. **Add a `List<PascalName>sParams` struct** with fields: Name, SourceIDs, FilterQuery,
   OrderBy, SortOrder, NextPageToken, PageSize.

3. **Add `List<Entity>s` method**: build `ListOptions` from params, call repository `.List()`,
   map each result via a mapping function, return the API list type with pagination.
   For pointer fields on the list struct (check `catalog/pkg/openapi/model_<entity>_list.go`),
   use `&variable` or `new(value)` (Go 1.26 builtin).
   The `NextPageToken` from the repository is a plain `string`; only assign to the response
   if non-empty (the API model field is `*string`).

4. **Add `Get<Entity>` method**: parse ID string to int32, call repository `.GetByID()`,
   map via the mapping function, return or 404.

5. **Implement `mapDB<Entity>ToAPI` mapping function**: map ID (format int64 → string),
   Name from attributes, then loop over Properties matching by name:
   - String fields: `res.<Field> = prop.StringValue`
   - Boolean fields: `res.<Field> = prop.BoolValue`
   - Array fields: `json.Unmarshal` from `prop.StringValue` into `[]string`

   Reference: `catalog/internal/catalog/modelcatalog/db_catalog.go` function `mapDBModelToAPIModel`.

6. **Remove** the `var _ sharedmodels.CatalogSourceRepository` placeholder and the TODO comments.

## Phase 7: Create OpenAPI Service Implementation

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
   - For errors, use `api.ErrNotFound` / `api.ErrBadRequest` checks from `github.com/kubeflow/hub/pkg/api`
   - Note: Plugins do NOT implement a `FindSources` method — sources are managed by the
     model catalog via the shared `/sources` endpoint with `assetType` filtering.

   Match the exact method signatures from the interface.

## Phase 8: Implement YAML Provider + Wire Loader

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
and saves via the repository. The Save call signature depends on datastore type:
- Context entities: `l.services.<Entity>Repository.Save(entity)` (1 param)
- Artifact/execution entities: `l.services.<Entity>Repository.Save(entity, parentID)` (2 params)

Reference: `catalog/internal/catalog/mcpcatalog/providers.go` for path resolution and
`catalog/internal/catalog/mcpcatalog/loader.go` `updateDatabase` for the save pattern.

## Phase 9: Wire RegisterRoutes

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

## Phase 10: Verify Compilation

```bash
go build ./catalog/...
```

If compilation fails, read the error output, fix the issues (typically unused or missing imports,
pointer vs value mismatches on list struct fields), and retry.

## Phase 11: Report

Print a summary with:

1. **Created** — list all files catalog-gen created
2. **Modified** — entity service panics replaced, db_provider wired, loader implemented, plugin.go wired
3. **Generated** — OpenAPI server stubs + service implementation
4. **Build** — pass/fail

Then print **Next Steps** the developer must complete manually:

```
Next steps:
  1. Edit the OpenAPI spec (api/openapi/src/plugins/<name>.yaml) to add entity-specific fields.
     The generated spec already uses `allOf` with `BaseResource` (which provides
     customProperties, description, externalId, and timestamp fields) and includes
     q, sourceLabel, and filterQuery parameters on the list endpoint.

     Add entity-specific properties under the second `allOf` entry alongside source_id.
     For URL fields use `format: uri`. Follow MCP's camelCase naming for new fields
     (e.g., repositoryUrl, not repository_url) — exception: source_id stays snake_case
     because it's a cross-plugin convention.

  2. Run /sync-catalog to propagate spec changes
  3. Run /catalog-sample-data to generate test data
  4. Add list filter logic in apply<Entity>ListFilters if needed
  5. Add entity-specific properties to *_entity_mappings.go
```
