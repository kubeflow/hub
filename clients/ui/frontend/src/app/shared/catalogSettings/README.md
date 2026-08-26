# Shared catalog settings kit

This folder holds shared data-wiring and presentational shells for catalog
settings (list + add/manage source). Model and MCP already use it; a future
catalog can reuse the same pieces without copy-pasting UI.

Layers:

- **Data / routes** — `createCatalogSettingsContext`, `CatalogSettingsRoutes`,
  preview/polling hooks under `hooks/`, and `createSourceConfigService`.
- **UI shells** — presentational components under `components/`.

Domain differences (copy, testids, extra columns, HF-only fields, etc.) are
injected via props/slots. Shared components never branch on `catalog === 'mcp'`
— express differences as props.

## Adding a new catalog settings page

1. **Types** — define source-config and preview summary types.

2. **Context** — call `createCatalogSettingsContext` with your API hooks and
   host paths (see `context/modelCatalogSettings/ModelCatalogSettingsContext.tsx`).

3. **Preview hook** — wrap `useCatalogSourcePreviewCore` in a domain hook that
   adapts your preview API to `CatalogSettingsPreviewResult` /
   `CatalogSettingsPreviewTabState`.

4. **List page** — compose:
   - `SourceConfigsTable` / `SourceConfigsTableRow` with columns, visibility
     getters, `StatusComponent`, `getManageSourceUrl`, `onToggleUpdate`,
     `deleteModalBody`, optional `testIdPrefix` / `renderExtraCells`.
   - `CatalogSettingsListPage` as the outer shell; pass the table as `children`.

5. **Manage source page** — compose:
   - `ManageSourcePageShell` for breadcrumb + YAML-format drawer + page chrome.
   - A domain `ManageSourceForm` that owns submit/validation defaults and uses:
     - `ManageSourceFormLayout` for sidebar form + preview + sticky footer chrome
     - `YamlUploadSection`, `IncludeExcludeFiltersSection`, `SourcePreviewPanel`,
       `ManageSourceFormFooter`, `ExpectedYamlFormatDrawer`, `CatalogSourceStatus`
     - Domain-only sections (e.g. Model HF credentials) stay local

6. **Wire definition** — export a `CatalogSettingsDefinition`
   (`ContextProvider`, `ListPage`, `ManagePage`) from `definition.ts` and mount
   via `CatalogSettingsRoutes`.

## Preserving testids

Shared components accept either:

- an explicit `testIds` object, or
- a `testIdPrefix` (`''` for Model, `'mcp-'` for MCP) prepended to fixed suffixes.

When migrating, match existing Cypress page-object ids — do not rename testids
as part of the extract.

## What stays domain-owned

- Model-only: `CredentialsSection`, HF source-type radios in `SourceDetailsSection`
- Submit orchestration, validation wiring, and create defaults in each domain's
  `ManageSourceForm` (layout chrome is shared via `ManageSourceFormLayout`)
- Column definitions (`*CatalogSourceConfigsTableColumns.tsx`) — column sets differ
- Thin domain wrappers around shared sections (pass copy/testids) are fine when
  they keep form import paths stable; prefer deleting wrappers only when call
  sites import shared components directly
