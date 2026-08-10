import * as React from 'react';
import { useQueryParamNamespaces } from 'mod-arch-core';
import useModelCatalogAPIState from '~/app/hooks/modelCatalog/useModelCatalogAPIState';
import { useCatalogSourcesWithPolling } from './hooks/useCatalogSourcesWithPolling';
import type {
  CatalogSettingsContextDefinition,
  CatalogSettingsContextValue,
} from './types';

/**
 * Factory that builds a reusable context-value hook for a catalog settings
 * section.  Call once at module level (outside any component) and use the
 * returned `useCatalogSettingsValue` hook inside your context provider.
 *
 * The returned hook handles:
 *   - settings API state (CRUD)
 *   - source-config list
 *   - catalog source polling (for status updates)
 *
 * Domain-specific context types (Model / MCP) are responsible for mapping the
 * generic field names (`sourceConfigs`, `catalogSources`, …) to their own
 * exported field names.
 */
export const createCatalogSettingsContext = <TAPIState, TConfigList>(
  contextDef: CatalogSettingsContextDefinition<TAPIState, TConfigList>,
): { useCatalogSettingsValue: () => CatalogSettingsContextValue<TAPIState, TConfigList> } => {
  const useCatalogSettingsValue = (): CatalogSettingsContextValue<TAPIState, TConfigList> => {
    const queryParams = useQueryParamNamespaces();

    const catalogQueryParams = React.useMemo(
      () => ({ ...queryParams, ...contextDef.catalogExtraQueryParams }),
      // contextDef is a module-level constant — only queryParams changes
      // eslint-disable-next-line react-hooks/exhaustive-deps
      [queryParams],
    );

    const [apiState, refreshAPIState] = contextDef.useSettingsAPIState(
      contextDef.settingsHostPath,
      queryParams,
      contextDef.previewHostPath ?? null,
    );

    const [catalogAPIState] = useModelCatalogAPIState(
      contextDef.catalogHostPath,
      catalogQueryParams,
    );

    const [
      sourceConfigs,
      sourceConfigsLoaded,
      sourceConfigsLoadError,
      refreshSourceConfigs,
    ] = contextDef.useSourceConfigsList(apiState);

    const [
      catalogSources,
      catalogSourcesLoaded,
      catalogSourcesLoadError,
      refreshCatalogSources,
    ] = useCatalogSourcesWithPolling(catalogAPIState);

    return React.useMemo(
      () => ({
        apiState,
        refreshAPIState,
        sourceConfigs,
        sourceConfigsLoaded,
        sourceConfigsLoadError,
        refreshSourceConfigs,
        catalogSources,
        catalogSourcesLoaded,
        catalogSourcesLoadError,
        refreshCatalogSources,
      }),
      [
        apiState,
        refreshAPIState,
        sourceConfigs,
        sourceConfigsLoaded,
        sourceConfigsLoadError,
        refreshSourceConfigs,
        catalogSources,
        catalogSourcesLoaded,
        catalogSourcesLoadError,
        refreshCatalogSources,
      ],
    );
  };

  return { useCatalogSettingsValue };
};
