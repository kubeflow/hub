import * as React from 'react';
import { useQueryParamNamespaces } from 'mod-arch-core';
import { BFF_API_VERSION, URL_PREFIX } from '~/app/utilities/const';
import {
  createCatalogContext,
  CatalogCommonData,
  CatalogProviderState,
} from '~/app/context/catalogContext/createCatalogContext';
import useModelCatalogAPIState from '~/app/hooks/modelCatalog/useModelCatalogAPIState';
import { useCatalogSources } from '~/app/hooks/modelCatalog/useCatalogSources';
import { useCatalogLabels } from '~/app/hooks/modelCatalog/useCatalogLabels';
import { useMcpServerFilterOptionListWithAPI } from '~/app/hooks/mcpServerCatalog/useMcpServerFilterOptionList';
import type {
  McpCatalogFiltersState,
  McpCatalogFilterOptionsList,
} from '~/app/pages/mcpCatalog/types/mcpCatalogFilterOptions';
import { useMcpUrlSync } from '~/app/pages/mcpCatalog/hooks/useMcpUrlSync';
import type { McpCatalogExtension, McpCatalogPaginationState } from './types';

export type {
  McpCatalogContextType,
  McpCatalogExtension,
  McpCatalogPaginationState,
} from './types';
export type { McpCatalogFiltersState } from '~/app/pages/mcpCatalog/types/mcpCatalogFilterOptions';

const MODEL_CATALOG_PATH = `${URL_PREFIX}/api/${BFF_API_VERSION}/model_catalog`;
const MCP_CATALOG_PATH = `${URL_PREFIX}/api/${BFF_API_VERSION}/mcp_catalog`;

const defaultPagination: McpCatalogPaginationState = {
  page: 1,
  pageSize: 10,
  totalItems: 0,
};

function useMcpCatalogSetup(providerState: CatalogProviderState) {
  const queryParams = useQueryParamNamespaces();
  const [apiStateModelCatalog] = useModelCatalogAPIState(MODEL_CATALOG_PATH, queryParams);
  const [apiStateMcpCatalog] = useModelCatalogAPIState(MCP_CATALOG_PATH, queryParams);

  const mcpListParams = React.useMemo(() => ({ assetType: 'mcp_servers' as const }), []);
  const [catalogSources, catalogSourcesLoaded, catalogSourcesLoadError] = useCatalogSources(
    apiStateModelCatalog,
    mcpListParams,
  );
  const [catalogLabels, catalogLabelsLoaded, catalogLabelsLoadError] = useCatalogLabels(
    apiStateModelCatalog,
    mcpListParams,
  );
  const [filterOptions, filterOptionsLoaded, filterOptionsLoadError] =
    useMcpServerFilterOptionListWithAPI(apiStateMcpCatalog);

  const { initialState, syncToUrl } = useMcpUrlSync();

  const [filters, setFilters] = React.useState<McpCatalogFiltersState>(initialState.filters);
  const [searchQuery, setSearchQuery] = React.useState(initialState.searchQuery);
  const [namedQuery, setNamedQuery] = React.useState<string | null>(null);
  const [pagination, setPaginationState] =
    React.useState<McpCatalogPaginationState>(defaultPagination);

  const { setSelectedSourceLabel } = providerState;

  React.useEffect(() => {
    setSelectedSourceLabel(initialState.selectedSourceLabel);
  }, [setSelectedSourceLabel, initialState.selectedSourceLabel]);

  React.useEffect(() => {
    syncToUrl({
      searchQuery,
      filters,
      selectedSourceLabel: providerState.selectedSourceLabel,
    });
  }, [searchQuery, filters, providerState.selectedSourceLabel, syncToUrl]);

  const setPage = React.useCallback((page: number) => {
    setPaginationState((prev) => ({ ...prev, page }));
  }, []);

  const setPageSize = React.useCallback((pageSize: number) => {
    setPaginationState((prev) => ({ ...prev, pageSize, page: 1 }));
  }, []);

  const setTotalItems = React.useCallback((totalItems: number) => {
    setPaginationState((prev) => ({ ...prev, totalItems }));
  }, []);

  const clearAllFilters = React.useCallback(() => {
    setSearchQuery('');
    setFilters({});
    setNamedQuery(null);
  }, []);

  const catalogData = React.useMemo<CatalogCommonData<McpCatalogFilterOptionsList>>(
    () => ({
      catalogSources,
      catalogSourcesLoaded,
      catalogSourcesLoadError,
      catalogLabels,
      catalogLabelsLoaded,
      catalogLabelsLoadError,
      filterOptions,
      filterOptionsLoaded,
      filterOptionsLoadError,
    }),
    [
      catalogSources,
      catalogSourcesLoaded,
      catalogSourcesLoadError,
      catalogLabels,
      catalogLabelsLoaded,
      catalogLabelsLoadError,
      filterOptions,
      filterOptionsLoaded,
      filterOptionsLoadError,
    ],
  );

  const extension = React.useMemo(
    () => ({
      filters,
      setFilters,
      searchQuery,
      setSearchQuery,
      namedQuery,
      setNamedQuery,
      pagination,
      setPage,
      setPageSize,
      setTotalItems,
      clearAllFilters,
      mcpApiState: apiStateMcpCatalog,
    }),
    [
      apiStateMcpCatalog,
      filters,
      searchQuery,
      namedQuery,
      pagination,
      setPage,
      setPageSize,
      setTotalItems,
      clearAllFilters,
    ],
  );

  return { catalogData, extension };
}

const {
  Context: McpCatalogContext,
  Provider: McpCatalogContextProvider,
  useContext: useMcpCatalogContext,
} = createCatalogContext<McpCatalogFilterOptionsList, McpCatalogExtension>({
  displayName: 'McpCatalogContextProvider',
  useSetup: useMcpCatalogSetup,
});

export { McpCatalogContext, McpCatalogContextProvider, useMcpCatalogContext };
