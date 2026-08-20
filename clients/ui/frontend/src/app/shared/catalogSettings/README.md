# Shared catalog settings kit

This folder contains everything needed to build a new "catalog settings" experience
(list page + add/manage source page) without duplicating UI or data-fetching logic
across catalogs (Model, MCP, and future catalogs).

It is split into two layers, corresponding to Story 2 and Story 3 of the
shared-catalog-settings-ui-shells effort:

- **Story 2 — data layer**: `createCatalogSettingsContext`, `CatalogSettingsRoutes`,
  the preview/polling hooks under `hooks/`, and `createSourceConfigService`.
- **Story 3 — UI layer**: the presentational components under `components/`.

Domain differences (copy, testids, extra columns, HF-only fields, etc.) are always
injected via props/slots. Shared components never branch on `catalog === 'mcp'` or
similar — if a shared component needs to behave differently per catalog, that
difference must be expressed as a prop.

## Adding a new catalog settings page

1. **Types** — define your source-config and catalog-source-preview types
   (e.g. `MyCatalogSourceConfig`, `MyCatalogSourcePreviewSummary`).

2. **Context** — call `createCatalogSettingsContext` with your API hooks
   (`useSettingsAPIState`, `useSourceConfigsList`) and host paths. This gives you a
   `Context` + `ContextProvider` with polling for source configs and catalog sources
   already wired up (see `context/modelCatalogSettings/ModelCatalogSettingsContext.tsx`
   for a full example).

3. **Preview hook** — wrap `useCatalogSourcePreviewCore` in a domain hook (e.g.
   `useMySourcePreview`) that adapts your preview API shape to
   `CatalogSettingsPreviewResult` / `CatalogSettingsPreviewTabState`.

4. **List page** — compose:
   - `SourceConfigsTable` (+ implicitly `SourceConfigsTableRow`) for the table,
     supplying `columns`, `getVisibilityLabel`, `getSourceTypeLabel`,
     `StatusComponent`, `getManageSourceUrl`, `onToggleUpdate`, `deleteModalBody`,
     and an optional `testIdPrefix` / `renderExtraCells` for domain-only columns
     (e.g. Model's "Organization" column).
   - `CatalogSettingsListPage` as the outer shell (title, empty state, error state),
     with your table as `children`.

5. **Manage source page** — compose:
   - `ManageSourcePageShell` for the breadcrumb + YAML-format drawer + page shell.
   - Your own `ManageSourceForm` that assembles the shared form sections:
     - `YamlUploadSection` for the paste/upload YAML field.
     - `IncludeExcludeFiltersSection` for included/excluded filters.
     - `SourcePreviewPanel` for the preview tabs (included/excluded items).
     - `ManageSourceFormFooter` for submit/preview/cancel actions.
     - `ExpectedYamlFormatDrawer` for the drawer panel content.
     - `CatalogSourceStatus` (+ `CatalogSourceStatusErrorModal`, used internally)
       for the validation status badge, if your manage page shows it.
   - Keep any domain-only fields (e.g. Model's Hugging Face credentials radios) in
     your own domain form/section components — do not try to generalize those into
     the shared kit.

6. **Wire up the definition** — export a `CatalogSettingsDefinition` (`id`,
   `ContextProvider`, `ListPage`, `ManagePage`) from a `definition.ts` in your
   catalog's page folder, same as
   `pages/modelCatalogSettings/definition.ts` / `pages/mcpCatalogSettings/definition.ts`,
   and register it wherever `CatalogSettingsRoutes` is mounted.

## Preserving testids

Every shared component that renders a `data-testid` accepts either:

- an explicit `testIds` object (e.g. `YamlUploadSectionTestIds`,
  `ExpectedYamlFormatDrawerTestIds`, `SourcePreviewPanelTestIds`,
  `IncludeExcludeFiltersSectionTestIds`), or
- a `testIdPrefix` string (e.g. `''` for Model, `'mcp-'` for MCP) that's prepended to
  a fixed suffix (`SourceConfigsTable`, `SourceConfigsTableRow`, `CatalogSourceStatus`,
  `ManageSourceFormFooter`).

When migrating an existing catalog, check the current Cypress page object /
`data-testid` usages first and pick whichever mechanism reproduces the existing ids
exactly — don't rename testids as part of a shared-component migration.

## What stays domain-owned

- `CredentialsSection` and the Hugging Face source-type radios in
  `SourceDetailsSection` are Model-only and intentionally not shared.
- Overall form orchestration (`ManageSourceForm` / `McpManageSourceForm`) — submit
  handlers, validation wiring, and default values — stays in each domain's
  `pages/<catalog>CatalogSettings/components/` folder. The shared kit only extracts
  the parts that were byte-for-byte (or prop-parameterizably) identical across
  catalogs.
- Column definitions (`*CatalogSourceConfigsTableColumns.tsx`) stay domain-owned since
  column sets differ per catalog (e.g. Model's Organization column).
