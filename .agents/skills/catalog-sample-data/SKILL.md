---
name: catalog-sample-data
description: >
  Generate or update sample YAML data for a catalog plugin. Creates a data file
  with realistic entries matching the plugin's OpenAPI schema, and adds a source
  entry to sources.yaml. Does not implement the loader — use for plugins that
  already have a YAML provider (model, mcp) or as a data template for new plugins.
user-invocable: true
---

# Catalog Sample Data

Generate sample YAML data for a catalog plugin and register it in sources.yaml.

## Phase 1: Identify Plugin + Target

Ask the user:
- Which plugin to generate data for? Validate that `api/openapi/src/plugins/<name>.yaml` exists.
- Where to put the data? Default: `manifests/kustomize/options/catalog/overlays/e2e/`.
  Other options if asked:
  - `manifests/kustomize/options/catalog/overlays/demo/` — development
  - Custom path

  Do NOT put sample data in `manifests/kustomize/options/catalog/base/` — base is for
  production config only.

## Phase 2: Read the Entity Schema

Read `api/openapi/src/plugins/<name>.yaml` and extract each entity's schema properties from
`components.schemas`. These are the fields each sample record should have.

Also read the DatastoreEntries in `catalog/internal/plugins/<name>/plugin.go` to confirm which
fields are persisted as properties — the data file must use these exact names.

Combine both to build a field list for each entity, noting:
- Field name
- Type (string, boolean, array, integer)
- Whether it's a core field (id, name) or a property (description, provider, etc.)

## Phase 3: Ask About Content

Ask the user:
- How many sample records? (default: 3-5)
- Theme or domain? (e.g., "weather agents", "code review tools") — if not specified, generate
  something realistic based on the plugin name

## Phase 4: Generate the YAML Data File

Generate the data file following the established format. Use existing files as reference:
- Models: `manifests/kustomize/options/catalog/base/sample-catalog.yaml`
- MCP: `manifests/kustomize/options/catalog/overlays/e2e/test-mcp-servers.yaml`

### Format for context entities

```yaml
source: Sample <PascalName>s
<entity_plural>:
  - name: example-name
    description: "A realistic description"
    <string_field>: "value"
    <array_field>:
      - item1
      - item2
    <boolean_field>: true
    customProperties: {}
    createTimeSinceEpoch: "<epoch_millis>"
    lastUpdateTimeSinceEpoch: "<epoch_millis>"
```

### Guidelines for realistic data

- Names should be realistic and varied (e.g., "mistralai/Mistral-7B" not "test-1")
- Descriptions should be 1-2 sentences, distinct per record
- Use current epoch milliseconds for timestamps
- Array fields should have 2-4 items
- Boolean fields should have a mix of true/false across records
- Include `customProperties: {}` on each record (empty is fine)
- source_id is NOT included in the data file — it's set automatically from the sources.yaml id

Write to: `<target_dir>/sample-<name>-catalog.yaml`

If the file already exists, ask whether to overwrite or append.

## Phase 5: Update sources.yaml

Read the existing sources.yaml at the target path. If it doesn't exist, create it with
a minimal structure.

Add an entry for this plugin under `<name>_catalogs`:

```yaml
<name>_catalogs:
  - name: "Sample <PascalName> Catalog"
    id: sample_<name>_catalog
    type: yaml
    enabled: true
    properties:
      yamlCatalogPath: sample-<name>-catalog.yaml
    labels:
      - sample
```

If an entry with `id: sample_<name>_catalog` already exists, ask whether to replace it.

Important: preserve all existing content in sources.yaml — only add or replace the specific
entry. Use `Read` + `Edit` rather than `Write` to avoid clobbering other plugins' entries.

## Phase 6: Report

Print:
- Data file path and record count
- Sources.yaml entry status (added/replaced/unchanged)

Then print a reminder:

```
Note: For this data to load at startup, the plugin needs:
  1. A YAML provider registered (e.g., in <pkg>/providers.go)
  2. PerformLeaderOperations wired to call the provider

Model and MCP plugins have this built-in. For new plugins like 'agent',
these need to be implemented separately.

To test with the catalog server:
  PGHOST=localhost PGPORT=5432 PGDATABASE=catalog PGUSER=catalog PGPASSWORD=catalog \
    go run . catalog --catalogs-path=<target_dir>/sources.yaml
```
