import { useQueryParamNamespaces } from 'mod-arch-core';
import useGenericObjectState from 'mod-arch-core/dist/utilities/useGenericObjectState';
import * as React from 'react';
import { useLocation } from 'react-router-dom';
import { useCatalogFilterOptionList } from '~/app/hooks/modelCatalog/useCatalogFilterOptionList';
import { useCatalogLabels } from '~/app/hooks/modelCatalog/useCatalogLabels';
import { useCatalogSources } from '~/app/hooks/modelCatalog/useCatalogSources';
import useModelCatalogAPIState, {
  ModelCatalogAPIState,
} from '~/app/hooks/modelCatalog/useModelCatalogAPIState';
import {
  CatalogFilterOptionsList,
  CatalogLabelList,
  CatalogSource,
  CatalogSourceList,
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

export type ModelCatalogContextType = {
  catalogSourcesLoaded: boolean;
  catalogSourcesLoadError?: Error;
  catalogSources: CatalogSourceList | null;
  catalogLabels: CatalogLabelList | null;
  catalogLabelsLoaded: boolean;
  catalogLabelsLoadError?: Error;
  selectedSource: CatalogSource | undefined;
  updateSelectedSource: (source: CatalogSource | undefined) => void;
  selectedSourceLabel: string | undefined;
  updateSelectedSourceLabel: (sourceLabel: string | undefined) => void;
  apiState: ModelCatalogAPIState;
  refreshAPIState: () => void;
  filterData: ModelCatalogFilterStates;
  setFilterData: <K extends keyof ModelCatalogFilterStates>(
    key: K,
    value: ModelCatalogFilterStates[K],
  ) => void;
  filterOptions: CatalogFilterOptionsList | null;
  filterOptionsLoaded: boolean;
  filterOptionsLoadError?: Error;
  performanceViewEnabled: boolean;
  setPerformanceViewEnabled: (enabled: boolean) => void;
  performanceFiltersChangedOnDetailsPage: boolean;
  setPerformanceFiltersChangedOnDetailsPage: (changed: boolean) => void;
  lastViewedModelName: string | null;
  setLastViewedModelName: (modelName: string | null) => void;
  clearAllFilters: () => void;
  resetPerformanceFiltersToDefaults: () => void;
  resetSinglePerformanceFilterToDefault: (filterKey: keyof ModelCatalogFilterStates) => void;
  getPerformanceFilterDefaultValue: (
    filterKey: keyof ModelCatalogFilterStates,
  ) => string | number | string[] | undefined;
  sortBy: ModelCatalogSortOption | null;
  setSortBy: (sortBy: ModelCatalogSortOption | null) => void;
};

type ModelCatalogContextProviderProps = {
  children: React.ReactNode;
};

export const ModelCatalogContext = React.createContext<ModelCatalogContextType>({
  catalogSourcesLoaded: false,
  catalogSourcesLoadError: undefined,
  catalogSources: null,
  catalogLabels: null,
  catalogLabelsLoaded: false,
  catalogLabelsLoadError: undefined,
  selectedSource: undefined,
  filterData: {
    [ModelCatalogStringFilterKey.TASK]: [],
    [ModelCatalogStringFilterKey.PROVIDER]: [],
    [ModelCatalogStringFilterKey.LICENSE]: [],
    [ModelCatalogStringFilterKey.LANGUAGE]: [],
    [ModelCatalogStringFilterKey.HARDWARE_TYPE]: [],
    [ModelCatalogStringFilterKey.HARDWARE_CONFIGURATION]: [],
    [ModelCatalogStringFilterKey.USE_CASE]: [],
    [ModelCatalogNumberFilterKey.MAX_RPS]: undefined,
    [ModelCatalogStringFilterKey.TENSOR_TYPE]: [],
    [ModelCatalogStringFilterKey.VALIDATED_CONFIGURATION]: [],
  },
  updateSelectedSource: () => undefined,
  selectedSourceLabel: undefined,
  updateSelectedSourceLabel: () => undefined,
  // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
  apiState: { apiAvailable: false, api: null as unknown as ModelCatalogAPIState['api'] },
  refreshAPIState: () => undefined,
  setFilterData: () => undefined,
  filterOptions: null,
  filterOptionsLoaded: false,
  filterOptionsLoadError: undefined,
  performanceViewEnabled: false,
  setPerformanceViewEnabled: () => undefined,
  performanceFiltersChangedOnDetailsPage: false,
  setPerformanceFiltersChangedOnDetailsPage: () => undefined,
  lastViewedModelName: null,
  setLastViewedModelName: () => undefined,
  clearAllFilters: () => undefined,
  resetPerformanceFiltersToDefaults: () => undefined,
  resetSinglePerformanceFilterToDefault: () => undefined,
  getPerformanceFilterDefaultValue: () => undefined,
  sortBy: null,
  setSortBy: () => undefined,
});

export const ModelCatalogContextProvider: React.FC<ModelCatalogContextProviderProps> = ({
  children,
}) => {
  const hostPath = `${URL_PREFIX}/api/${BFF_API_VERSION}/model_catalog`;
  const queryParams = useQueryParamNamespaces();
  const [apiState, refreshAPIState] = useModelCatalogAPIState(hostPath, queryParams);
  const [catalogSources, catalogSourcesLoaded, catalogSourcesLoadError] =
    useCatalogSources(apiState);
  const [catalogLabels, catalogLabelsLoaded, catalogLabelsLoadError] = useCatalogLabels(apiState);
  const [selectedSource, setSelectedSource] =
    React.useState<ModelCatalogContextType['selectedSource']>(undefined);
  const [filterData, baseSetFilterData] = useGenericObjectState<ModelCatalogFilterStates>({
    [ModelCatalogStringFilterKey.TASK]: [],
    [ModelCatalogStringFilterKey.PROVIDER]: [],
    [ModelCatalogStringFilterKey.LICENSE]: [],
    [ModelCatalogStringFilterKey.LANGUAGE]: [],
    [ModelCatalogStringFilterKey.HARDWARE_TYPE]: [],
    [ModelCatalogStringFilterKey.HARDWARE_CONFIGURATION]: [],
    [ModelCatalogStringFilterKey.USE_CASE]: [],
    [ModelCatalogNumberFilterKey.MAX_RPS]: undefined,
    [ModelCatalogStringFilterKey.TENSOR_TYPE]: [],
    [ModelCatalogStringFilterKey.VALIDATED_CONFIGURATION]: [],
  });
  const [filterOptions, filterOptionsLoaded, filterOptionsLoadError] =
    useCatalogFilterOptionList(apiState);
  const [selectedSourceLabel, setSelectedSourceLabel] = React.useState<
    ModelCatalogContextType['selectedSourceLabel']
  >(CategoryName.allModels);
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
   * Clears basic filters (Task, Provider, License, Language, Tensor Type) to empty.
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
      // Return value - the type is already compatible
      if (Array.isArray(value) || typeof value === 'string' || typeof value === 'number') {
        return value;
      }
      return undefined;
    },
    [filterOptions],
  );

  const setFilterData = React.useCallback(
    <K extends keyof ModelCatalogFilterStates>(key: K, value: ModelCatalogFilterStates[K]) => {
      baseSetFilterData(key, value);
      if (isOnDetailsPage) {
        setPerformanceFiltersChangedOnDetailsPage(true);
      } else {
        setPerformanceFiltersChangedOnDetailsPage(false);
      }
    },
    [baseSetFilterData, isOnDetailsPage],
  );

  // Apply default performance filters on initial load if none are set
  React.useEffect(() => {
    if (
      filterOptionsLoaded &&
      filterOptions?.namedQueries?.[DEFAULT_PERFORMANCE_FILTERS_QUERY_NAME] &&
      filterData[ModelCatalogStringFilterKey.USE_CASE].length === 0
    ) {
      resetPerformanceFiltersToDefaults();
    }
  }, [
    filterOptionsLoaded,
    filterOptions?.namedQueries,
    filterData,
    resetPerformanceFiltersToDefaults,
  ]);

  const contextValue = React.useMemo(
    () => ({
      catalogSourcesLoaded,
      catalogSourcesLoadError,
      catalogSources,
      catalogLabels,
      catalogLabelsLoaded,
      catalogLabelsLoadError,
      selectedSource: selectedSource ?? undefined,
      updateSelectedSource: setSelectedSource,
      selectedSourceLabel: selectedSourceLabel ?? undefined,
      updateSelectedSourceLabel: setSelectedSourceLabel,
      apiState,
      refreshAPIState,
      filterData,
      setFilterData,
      filterOptions,
      filterOptionsLoaded,
      filterOptionsLoadError,
      performanceViewEnabled: basePerformanceViewEnabled,
      setPerformanceViewEnabled,
      performanceFiltersChangedOnDetailsPage,
      setPerformanceFiltersChangedOnDetailsPage,
      lastViewedModelName,
      setLastViewedModelName,
      clearAllFilters,
      resetPerformanceFiltersToDefaults,
      resetSinglePerformanceFilterToDefault,
      getPerformanceFilterDefaultValue: getDefaultValueForPerformanceFilter,
      sortBy,
      setSortBy,
    }),
    [
      catalogSourcesLoaded,
      catalogSourcesLoadError,
      catalogSources,
      catalogLabels,
      catalogLabelsLoaded,
      catalogLabelsLoadError,
      selectedSource,
      apiState,
      refreshAPIState,
      filterData,
      setFilterData,
      filterOptions,
      filterOptionsLoaded,
      filterOptionsLoadError,
      selectedSourceLabel,
      basePerformanceViewEnabled,
      setPerformanceViewEnabled,
      performanceFiltersChangedOnDetailsPage,
      setPerformanceFiltersChangedOnDetailsPage,
      lastViewedModelName,
      setLastViewedModelName,
      clearAllFilters,
      resetPerformanceFiltersToDefaults,
      resetSinglePerformanceFilterToDefault,
      getDefaultValueForPerformanceFilter,
      sortBy,
      setSortBy,
    ],
  );

  return (
    <ModelCatalogContext.Provider value={contextValue}>{children}</ModelCatalogContext.Provider>
  );
};
