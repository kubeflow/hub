import { useQueryParamNamespaces } from 'mod-arch-core';
import useGenericObjectState from 'mod-arch-core/dist/utilities/useGenericObjectState';
import * as React from 'react';
import { useLocation } from 'react-router-dom';
import {
  createCatalogContext,
  CatalogContextValue,
  CatalogProviderState,
  CatalogCommonData,
} from '~/app/context/catalogContext/createCatalogContext';
import { useCatalogFilterOptionList } from '~/app/hooks/modelCatalog/useCatalogFilterOptionList';
import { useCatalogLabels } from '~/app/hooks/modelCatalog/useCatalogLabels';
import { useCatalogSources } from '~/app/hooks/modelCatalog/useCatalogSources';
import useModelCatalogAPIState, {
  ModelCatalogAPIState,
} from '~/app/hooks/modelCatalog/useModelCatalogAPIState';
import type { CatalogSource } from '~/app/shared/types/catalogTypes';
import {
  CatalogFilterOptionsList,
  CategoryName,
  ModelCatalogFilterStates,
  NamedQuery,
} from '~/app/modelCatalogTypes';
import {
  ModelDetailsTab,
  ModelCatalogStringFilterKey,
  ModelCatalogNumberFilterKey,
  DEFAULT_PERFORMANCE_FILTERS_QUERY_NAME,
  ALL_LATENCY_FILTER_KEYS,
  isLatencyFilterKey,
  ModelCatalogSortOption,
} from '~/concepts/modelCatalog/const';
import {
  getSingleFilterDefault,
  applyFilterValue,
  getDefaultFiltersFromNamedQuery,
} from '~/app/pages/modelCatalog/utils/performanceFilterUtils';
import { getEffectiveSortBy } from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import { BFF_API_VERSION, URL_PREFIX } from '~/app/utilities/const';

const MODEL_CATALOG_HOST_PATH = `${URL_PREFIX}/api/${BFF_API_VERSION}/model_catalog`;

const INITIAL_FILTERS: ModelCatalogFilterStates = {
  [ModelCatalogStringFilterKey.TASK]: [],
  [ModelCatalogStringFilterKey.PROVIDER]: [],
  [ModelCatalogStringFilterKey.LICENSE]: [],
  [ModelCatalogStringFilterKey.LANGUAGE]: [],
  [ModelCatalogStringFilterKey.HARDWARE_TYPE]: [],
  [ModelCatalogStringFilterKey.HARDWARE_CONFIGURATION]: [],
  [ModelCatalogStringFilterKey.USE_CASE]: [],
  [ModelCatalogNumberFilterKey.MAX_RPS]: undefined,
  [ModelCatalogNumberFilterKey.COLD_START_LATENCY]: undefined,
  [ModelCatalogStringFilterKey.TENSOR_TYPE]: [],
  [ModelCatalogStringFilterKey.VALIDATED_CONFIGURATION]: [],
};

type ModelCatalogExtension = {
  selectedSource: CatalogSource | undefined;
  updateSelectedSource: (source: CatalogSource | undefined) => void;
  apiState: ModelCatalogAPIState;
  refreshAPIState: () => void;
  filters: ModelCatalogFilterStates;
  setFilters: (
    updater:
      | ModelCatalogFilterStates
      | ((prev: ModelCatalogFilterStates) => ModelCatalogFilterStates),
  ) => void;
  performanceViewEnabled: boolean;
  setPerformanceViewEnabled: (enabled: boolean) => void;
  performanceFiltersChangedOnDetailsPage: boolean;
  setPerformanceFiltersChangedOnDetailsPage: (changed: boolean) => void;
  lastViewedModelName: string | null;
  setLastViewedModelName: (modelName: string | null) => void;
  resetPerformanceFiltersToDefaults: () => void;
  resetSinglePerformanceFilterToDefault: (filterKey: keyof ModelCatalogFilterStates) => void;
  getPerformanceFilterDefaultValue: (
    filterKey: keyof ModelCatalogFilterStates,
  ) => string | number | string[] | undefined;
  sortBy: ModelCatalogSortOption | null;
  setSortBy: (sortBy: ModelCatalogSortOption | null) => void;
};

export type ModelCatalogContextType = CatalogContextValue<CatalogFilterOptionsList> &
  ModelCatalogExtension;

// eslint-disable-next-line @typescript-eslint/no-unused-vars
function useModelCatalogSetup(providerState: CatalogProviderState) {
  const queryParams = useQueryParamNamespaces();
  const [apiState, refreshAPIState] = useModelCatalogAPIState(MODEL_CATALOG_HOST_PATH, queryParams);

  const [catalogSources, catalogSourcesLoaded, catalogSourcesLoadError] =
    useCatalogSources(apiState);
  const [catalogLabels, catalogLabelsLoaded, catalogLabelsLoadError] = useCatalogLabels(apiState);
  const [filterOptions, filterOptionsLoaded, filterOptionsLoadError] =
    useCatalogFilterOptionList(apiState);

  const [selectedSource, setSelectedSource] = React.useState<CatalogSource | undefined>(undefined);
  const [filterState, baseSetFilterData, , replaceFilterState] =
    useGenericObjectState<ModelCatalogFilterStates>(INITIAL_FILTERS);
  const [basePerformanceViewEnabled, setBasePerformanceViewEnabled] = React.useState(false);
  const [performanceFiltersChangedOnDetailsPage, setPerformanceFiltersChangedOnDetailsPage] =
    React.useState(false);
  const [lastViewedModelName, setLastViewedModelName] = React.useState<string | null>(null);
  const [sortBy, setSortBy] = React.useState<ModelCatalogSortOption | null>(null);

  const location = useLocation();
  const isOnDetailsPage = location.pathname.includes(ModelDetailsTab.PERFORMANCE_INSIGHTS);

  /**
   * Applies filter values from a named query to the filter state.
   * Uses getDefaultFiltersFromNamedQuery to parse the namedQuery and applyFilterValue to set each filter.
   */
  const applyNamedQueryDefaults = React.useCallback(
    (namedQuery: NamedQuery) => {
      const defaults = getDefaultFiltersFromNamedQuery(filterOptions, namedQuery);
      Object.entries(defaults).forEach(([filterKey, value]) => {
        applyFilterValue(baseSetFilterData, filterKey, value);
      });
    },
    [baseSetFilterData, filterOptions],
  );

  /**
   * Resets performance filters to their default values from namedQueries.
   * Performance filters should always have values (defaults when not explicitly set).
   * This is the single function for "clearing" or "resetting" performance filters.
   */
  const resetPerformanceFiltersToDefaults = React.useCallback(() => {
    // First, clear ALL latency filters (only one should be active at a time)
    // This ensures any non-default latency filter is removed before applying defaults
    ALL_LATENCY_FILTER_KEYS.forEach((latencyKey) => {
      baseSetFilterData(latencyKey, undefined);
    });
    baseSetFilterData(ModelCatalogStringFilterKey.HARDWARE_CONFIGURATION, []);

    // Then apply all defaults from namedQueries
    const defaultQuery = filterOptions?.namedQueries?.[DEFAULT_PERFORMANCE_FILTERS_QUERY_NAME];
    if (defaultQuery) {
      applyNamedQueryDefaults(defaultQuery);
    }

    if (!isOnDetailsPage) {
      setPerformanceFiltersChangedOnDetailsPage(false);
    }
  }, [filterOptions?.namedQueries, applyNamedQueryDefaults, baseSetFilterData, isOnDetailsPage]);

  /**
   * Clears basic filters (Task, Provider, License, Language, Tensor Type, Validated Configuration) to empty.
   * Note: BASIC_FILTER_KEYS in const.ts should be updated if basic filters change.
   */
  const clearBasicFilters = React.useCallback(() => {
    baseSetFilterData(ModelCatalogStringFilterKey.TASK, []);
    baseSetFilterData(ModelCatalogStringFilterKey.PROVIDER, []);
    baseSetFilterData(ModelCatalogStringFilterKey.LICENSE, []);
    baseSetFilterData(ModelCatalogStringFilterKey.LANGUAGE, []);
    baseSetFilterData(ModelCatalogStringFilterKey.TENSOR_TYPE, []);
    baseSetFilterData(ModelCatalogStringFilterKey.VALIDATED_CONFIGURATION, []);
  }, [baseSetFilterData]);

  /**
   * Clears all filters: basic filters to empty, performance filters to defaults.
   */
  const clearAllFilters = React.useCallback(() => {
    clearBasicFilters();
    resetPerformanceFiltersToDefaults();
  }, [clearBasicFilters, resetPerformanceFiltersToDefaults]);

  const setPerformanceViewEnabled = React.useCallback(
    (enabled: boolean) => {
      setBasePerformanceViewEnabled(enabled);
      // Performance filters always have values (defaults).
      // When toggle changes, ensure defaults are applied.
      // When toggle is OFF, filters are just not passed in API calls or shown as chips.
      resetPerformanceFiltersToDefaults();

      // Update sort to default for the new toggle state, preserving user selection if it doesn't match the opposite default
      const defaultSort = getEffectiveSortBy(null, enabled);
      const oppositeDefault = getEffectiveSortBy(null, !enabled);
      setSortBy((currentSortBy) => {
        if (currentSortBy === null || currentSortBy === oppositeDefault) {
          return defaultSort;
        }
        return currentSortBy;
      });
      setPerformanceFiltersChangedOnDetailsPage(false);
    },
    [resetPerformanceFiltersToDefaults],
  );

  /**
   * Resets a single performance filter to its default value from namedQueries.
   * Used when clicking the undo button on individual performance filter chips.
   *
   * For latency filters: Only one latency filter can be active at a time.
   * When closing any latency chip, we clear ALL latency filters and apply the DEFAULT latency filter.
   * This ensures proper reset behavior (e.g., closing ITL chip resets to the default TTFT filter).
   */
  const resetSinglePerformanceFilterToDefault = React.useCallback(
    (filterKey: keyof ModelCatalogFilterStates) => {
      if (isLatencyFilterKey(filterKey)) {
        // For latency filters: clear ALL latency filters first
        ALL_LATENCY_FILTER_KEYS.forEach((latencyKey) => {
          baseSetFilterData(latencyKey, undefined);
        });

        // Then apply the default latency filter (which may be a different key, e.g., TTFT when closing ITL)
        const defaultQuery = filterOptions?.namedQueries?.[DEFAULT_PERFORMANCE_FILTERS_QUERY_NAME];
        if (defaultQuery) {
          // Find the default latency filter from namedQueries
          for (const latencyKey of ALL_LATENCY_FILTER_KEYS) {
            const { hasDefault, value } = getSingleFilterDefault(filterOptions, latencyKey);
            if (hasDefault && value !== undefined) {
              applyFilterValue(baseSetFilterData, latencyKey, value);
              break; // Only apply the first (and should be only) default latency filter
            }
          }
        }
      } else {
        // Non-latency filters: just reset to default
        const { value } = getSingleFilterDefault(filterOptions, filterKey);
        applyFilterValue(baseSetFilterData, filterKey, value);
      }

      if (!isOnDetailsPage) {
        setPerformanceFiltersChangedOnDetailsPage(false);
      }
    },
    [filterOptions, baseSetFilterData, isOnDetailsPage],
  );

  /**
   * Gets the default value for a performance filter from namedQueries.
   * Wrapper around the utility function that provides filterOptions from context.
   */
  const getDefaultValueForPerformanceFilter = React.useCallback(
    (filterKey: keyof ModelCatalogFilterStates): string | number | string[] | undefined => {
      const { value } = getSingleFilterDefault(filterOptions, filterKey);
      if (Array.isArray(value) || typeof value === 'string' || typeof value === 'number') {
        return value;
      }
      return undefined;
    },
    [filterOptions],
  );

  const setFilters = React.useCallback(
    (
      updater:
        | ModelCatalogFilterStates
        | ((prev: ModelCatalogFilterStates) => ModelCatalogFilterStates),
    ) => {
      const newValue = typeof updater === 'function' ? updater(filterState) : updater;
      replaceFilterState(newValue);

      if (isOnDetailsPage) {
        setPerformanceFiltersChangedOnDetailsPage(true);
      } else {
        setPerformanceFiltersChangedOnDetailsPage(false);
      }
    },
    [replaceFilterState, filterState, isOnDetailsPage],
  );

  React.useEffect(() => {
    if (
      filterOptionsLoaded &&
      filterOptions?.namedQueries?.[DEFAULT_PERFORMANCE_FILTERS_QUERY_NAME] &&
      filterState[ModelCatalogStringFilterKey.USE_CASE].length === 0
    ) {
      resetPerformanceFiltersToDefaults();
    }
  }, [
    filterOptionsLoaded,
    filterOptions?.namedQueries,
    filterState,
    resetPerformanceFiltersToDefaults,
  ]);

  const catalogData = React.useMemo<CatalogCommonData<CatalogFilterOptionsList>>(
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
      selectedSource: selectedSource ?? undefined,
      updateSelectedSource: setSelectedSource,
      apiState,
      refreshAPIState,
      filters: filterState,
      setFilters,
      clearAllFilters,
      performanceViewEnabled: basePerformanceViewEnabled,
      setPerformanceViewEnabled,
      performanceFiltersChangedOnDetailsPage,
      setPerformanceFiltersChangedOnDetailsPage,
      lastViewedModelName,
      setLastViewedModelName,
      resetPerformanceFiltersToDefaults,
      resetSinglePerformanceFilterToDefault,
      getPerformanceFilterDefaultValue: getDefaultValueForPerformanceFilter,
      sortBy,
      setSortBy,
    }),
    [
      selectedSource,
      apiState,
      refreshAPIState,
      filterState,
      setFilters,
      clearAllFilters,
      basePerformanceViewEnabled,
      setPerformanceViewEnabled,
      performanceFiltersChangedOnDetailsPage,
      lastViewedModelName,
      resetPerformanceFiltersToDefaults,
      resetSinglePerformanceFilterToDefault,
      getDefaultValueForPerformanceFilter,
      sortBy,
    ],
  );

  return { catalogData, extension };
}

const {
  Context: ModelCatalogContext,
  Provider: ModelCatalogContextProvider,
  useContext: useModelCatalogContext,
} = createCatalogContext<CatalogFilterOptionsList, ModelCatalogExtension>({
  displayName: 'ModelCatalogContextProvider',
  initialSelectedSourceLabel: CategoryName.allModels,
  useSetup: useModelCatalogSetup,
});

export { ModelCatalogContext, ModelCatalogContextProvider, useModelCatalogContext };
