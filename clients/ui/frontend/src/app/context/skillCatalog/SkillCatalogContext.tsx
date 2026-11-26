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
import { useSkillFilterOptionListWithAPI } from '~/app/hooks/skillCatalog/useSkillFilterOptionList';
import useEmptyCategoryTracking from '~/app/hooks/useEmptyCategoryTracking';
import type {
  SkillCatalogFiltersState,
  SkillCatalogFilterOptionsList,
} from '~/app/pages/skillCatalog/types/skillCatalogFilterOptions';
import { useSkillUrlSync } from '~/app/pages/skillCatalog/hooks/useSkillUrlSync';
import type { SkillCatalogExtension, SkillCatalogPaginationState } from './types';

export type {
  SkillCatalogContextType,
  SkillCatalogExtension,
  SkillCatalogPaginationState,
} from './types';
export type { SkillCatalogFiltersState } from '~/app/pages/skillCatalog/types/skillCatalogFilterOptions';

const SKILL_CATALOG_PATH = `${URL_PREFIX}/api/${BFF_API_VERSION}/skill_catalog`;
const MODEL_CATALOG_PATH = `${URL_PREFIX}/api/${BFF_API_VERSION}/model_catalog`;

const defaultPagination: SkillCatalogPaginationState = {
  page: 1,
  pageSize: 10,
  totalItems: 0,
};

function useSkillCatalogSetup(providerState: CatalogProviderState) {
  const queryParams = useQueryParamNamespaces();
  const [apiStateSkillCatalog] = useModelCatalogAPIState(SKILL_CATALOG_PATH, queryParams);
  // Sources and labels are served by the generic model_catalog plugin, discriminated by
  // assetType — the same route MCP and models use, rather than a skill-specific proxy.
  const [apiStateModelCatalog] = useModelCatalogAPIState(MODEL_CATALOG_PATH, queryParams);

  const skillListParams = React.useMemo(() => ({ assetType: 'skills' as const }), []);
  const [catalogSources, catalogSourcesLoaded, catalogSourcesLoadError] = useCatalogSources(
    apiStateModelCatalog,
    skillListParams,
  );
  const [catalogLabels, catalogLabelsLoaded, catalogLabelsLoadError] = useCatalogLabels(
    apiStateModelCatalog,
    skillListParams,
  );

  const [filterOptions, filterOptionsLoaded, filterOptionsLoadError] =
    useSkillFilterOptionListWithAPI(apiStateSkillCatalog);

  const { initialState, syncToUrl } = useSkillUrlSync();

  const [filters, setFilters] = React.useState<SkillCatalogFiltersState>(initialState.filters);
  const [searchQuery, setSearchQuery] = React.useState(initialState.searchQuery);
  const [pagination, setPaginationState] =
    React.useState<SkillCatalogPaginationState>(defaultPagination);

  const { setSelectedSourceLabel } = providerState;
  const { emptyCategoryLabels, categoriesResolved, reportCategoryEmpty, setCategoryCount } =
    useEmptyCategoryTracking();

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
  }, []);

  const catalogData = React.useMemo<CatalogCommonData<SkillCatalogFilterOptionsList>>(
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
      searchQuery,
      setSearchQuery,
      filters,
      setFilters,
      pagination,
      setPage,
      setPageSize,
      setTotalItems,
      clearAllFilters,
      skillApiState: apiStateSkillCatalog,
      emptyCategoryLabels,
      categoriesResolved,
      reportCategoryEmpty,
      setCategoryCount,
    }),
    [
      apiStateSkillCatalog,
      searchQuery,
      filters,
      pagination,
      setPage,
      setPageSize,
      setTotalItems,
      clearAllFilters,
      emptyCategoryLabels,
      categoriesResolved,
      reportCategoryEmpty,
      setCategoryCount,
    ],
  );

  return { catalogData, extension };
}

const {
  Context: SkillCatalogContext,
  Provider: SkillCatalogContextProvider,
  useContext: useSkillCatalogContext,
} = createCatalogContext<SkillCatalogFilterOptionsList, SkillCatalogExtension>({
  displayName: 'SkillCatalogContextProvider',
  useSetup: useSkillCatalogSetup,
});

export { SkillCatalogContext, SkillCatalogContextProvider, useSkillCatalogContext };
