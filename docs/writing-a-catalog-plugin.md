# Writing a Catalog Plugin

The catalog service is a federated asset registry. Each asset kind — models, MCP
servers, agents, and so on — is served by a **plugin**. A plugin owns its own
OpenAPI spec, domain package, datastore mappings, HTTP routes, and YAML data
loader.

This guide walks through creating a new plugin from an empty scaffold to a
compiling, route-serving component. It relies on the `catalog-gen` tool to
generate boilerplate and on a set of AI-agent skills to fill in the parts that
require judgement.

## Before You Start

- Work from the repository root; all `make` targets below assume it.
- A plugin is defined by three inputs:
  - **Name** — `snake_case`, singular (e.g. `agent`, `dataset`, `pipeline`).
  - **Description** — human-readable (e.g. `"Agent catalog"`).
  - **Entities** — one or more `Name:type` pairs. `Name` is PascalCase; `type`
    is the datastore kind: `context`, `artifact`, or `execution`
    (e.g. `CatalogAgent:context,CatalogAgentArtifact:artifact`).
- Choose the datastore type per entity deliberately:
  - `context` — a top-level, independently listed asset (the common case).
  - `artifact` — a file/blob-like child of a context.
  - `execution` — a run/action associated with a context.

If you use Claude Code (or a compatible assistant), the `/init-catalog` skill
runs this entire flow interactively and fills in the hand-written code for you.
The sections below document what it does so you can run it — or do it manually.

## Step 1 — Scaffold with catalog-gen

`catalog-gen` (`tools/catalog-gen`) generates the plugin skeleton. Invoke it
through the Makefile:

```bash
make -C catalog gen/catalog-plugin \
  NAME=<name> \
  DESCRIPTION="<description>" \
  ENTITIES=<Entity1:type>,<Entity2:type>
```

Example:

```bash
make -C catalog gen/catalog-plugin \
  NAME=dataset \
  DESCRIPTION="Dataset catalog" \
  ENTITIES=CatalogDataset:context,CatalogDatasetArtifact:artifact
```

> Pass `--dry-run` to preview without writing:
> `go run ./tools/catalog-gen init --name=dataset --description="Dataset catalog" --entity=CatalogDataset:context --root=. --dry-run`

This creates:

| Location | Purpose |
|---|---|
| `catalog/internal/plugins/<name>/` | Plugin entry point (`plugin.go`, `register.go`) |
| `catalog/internal/catalog/<name>catalog/` | Domain package — `services.go`, `loader.go`, `db_<name>.go`, `sources.go` |
| `catalog/internal/catalog/<name>catalog/models/` | Entity model per entity |
| `catalog/internal/catalog/<name>catalog/service/` | Entity service + datastore mappings per entity |
| `api/openapi/src/plugins/<name>.yaml` | Plugin OpenAPI source spec |
| `catalog/plugins/<name>/` | OpenAPI codegen ignore + server-stub generation script |

It also edits two shared files: registers the plugin in `catalog/cmd/catalog.go`
(blank import) and wires configuration.

The generated entity services contain `panic("TODO")` stubs — the plugin does not
yet compile into working behavior. The remaining steps replace them.

## Step 2 — Register the Asset Type

So the shared `/sources` endpoint can scope sources to your plugin, register a new
asset-type value (the primary entity's snake_case name, pluralized — e.g.
`CatalogAgent` → `agents`):

1. Add the value to the `CatalogAssetType` enum in `api/openapi/src/catalog.yaml`.
2. Add an `AssetType<Name>` constant in
   `catalog/internal/catalog/basecatalog/source_types.go`.
3. Add it to the `validAssetTypes` map (and error message) in
   `catalog/internal/catalog/basecatalog/validation.go`.

## Step 3 — Implement the Entity Services

Each generated file under `catalog/internal/catalog/<name>catalog/service/` has
eight `panic("TODO")` stubs covering entity↔schema mapping, filtering, ordering,
and CRUD. Use the existing plugins as reference implementations for each
datastore type:

- **context** — `catalog/internal/catalog/mcpcatalog/service/mcp_server.go`
- **artifact** — `catalog/internal/catalog/modelcatalog/service/catalog_model_artifact.go`
- **execution** — `catalog/internal/catalog/mcpcatalog/service/mcp_server_tool.go`

## Step 4 — Generate OpenAPI Server + Client Code

Run these targets in order — the order matters, because the server stubs
reference generated client model types:

```bash
make api/openapi/catalog.yaml      # merge the plugin spec into the catalog spec
make -C catalog gen/openapi-server # server stubs (controller, routes)
make -C catalog gen/openapi        # client model types
```

Regenerate whenever you change the plugin's OpenAPI spec.

## Step 5 — Wire the DB Provider, Service, Loader, and Routes

Fill in the remaining hand-written pieces:

- **DB provider** (`db_<name>.go`) — implement `List<Entity>s` / `Get<Entity>`
  and the DB→API mapping. Reference:
  `catalog/internal/catalog/modelcatalog/db_catalog.go`.
- **Service implementation** — create
  `catalog/internal/server/openapi/api_<name>_catalog_service_service.go`
  delegating to the DB provider. (Plugins do **not** implement `FindSources`;
  sources are handled by the shared `/sources` endpoint.)
- **YAML provider + loader** — add a YAML provider and implement
  `PerformLeaderOperations` in `loader.go` so the plugin loads data from YAML at
  startup. Every YAML struct field needs **both** `yaml` and `json` tags.
  Reference: `catalog/internal/catalog/mcpcatalog/`.
- **Routes** — implement `RegisterRoutes` in
  `catalog/internal/plugins/<name>/plugin.go` to mount the generated controller.

## Step 6 — Build and Verify

```bash
go build ./catalog/...
make -C catalog test
```

## Step 7 — Add Fields and Sample Data

The scaffold ships a minimal spec (`BaseResource` fields plus `q`, `sourceLabel`,
and `filterQuery` on the list endpoint). To flesh the plugin out:

1. Edit `api/openapi/src/plugins/<name>.yaml` to add entity-specific properties.
   Follow the camelCase naming convention (e.g. `repositoryUrl`); the sole
   exception is the cross-plugin `source_id`, which stays snake_case.
2. Re-run the generation targets from Step 4.
3. Propagate the new fields through the datastore mappings and service layer
   (persistence, filtering).
4. Add sample data and register a source in `sources.yaml`.

## Related Skills

If you use an AI assistant, these skills automate the flow above:

| Skill | Use for |
|---|---|
| `/init-catalog` | Run Steps 1–6 end-to-end, interactively. |
| `/catalog-add-route` | Add an endpoint or query parameter to a plugin spec. |
| `/sync-catalog` | Propagate spec changes (Step 7.2–7.3) through generated and hand-written code. |
| `/catalog-sample-data` | Generate sample YAML data and register a source. |

## Reference

- Tool source: `tools/catalog-gen/` (`catalog-gen init --help`)
- Makefile target: `gen/catalog-plugin` in `catalog/Makefile`
- Existing plugins to model after: `catalog/internal/catalog/modelcatalog/`,
  `catalog/internal/catalog/mcpcatalog/`
- YAML source format: [catalog-yaml-reference.md](./catalog-yaml-reference.md)
