import { FetchState, FetchStateCallbackPromise, NotReadyError, useFetchState } from 'mod-arch-core';
import React from 'react';
import {
  CatalogFilterOptionsList,
  CatalogPerformanceArtifactList,
  CatalogPerformanceMetricsArtifact,
  ModelCatalogFilterStates,
  PerformanceArtifactsParams,
} from '~/app/modelCatalogTypes';
import { useModelCatalogAPI } from './useModelCatalogAPI';

const useNormalizedPerformanceParams = (params?: PerformanceArtifactsParams) =>
  React.useMemo(
    (): PerformanceArtifactsParams => ({
      targetRPS: params?.targetRPS,
      rpsProperty: params?.rpsProperty,
      latencyProperty: params?.latencyProperty,
      hardwareCountProperty: params?.hardwareCountProperty,
      hardwareTypeProperty: params?.hardwareTypeProperty,
      filterQuery: params?.filterQuery,
      pageSize: params?.pageSize,
      orderBy: params?.orderBy,
      sortOrder: params?.sortOrder,
      nextPageToken: params?.nextPageToken,
    }),
    [
      params?.targetRPS,
      params?.rpsProperty,
      params?.latencyProperty,
      params?.hardwareCountProperty,
      params?.hardwareTypeProperty,
      params?.filterQuery,
      params?.pageSize,
      params?.orderBy,
      params?.sortOrder,
      params?.nextPageToken,
    ],
  );

/**
 * Hook for fetching performance artifacts from the /performance_artifacts endpoint.
 * This endpoint returns only performance artifacts and supports server-side filtering,
 * sorting, and pagination.
 *
 * @param sourceId - The catalog source ID
 * @param modelName - The model name
 * @param params - Performance-specific parameters (targetRPS, latencyProperty, orderBy, pagination)
 * @param filterData - Current filter state for building filterQuery
 * @param filterOptions - Filter options from the API for validation
 * @param enabled - Whether to enable fetching (default: true)
 */
export const useCatalogPerformanceArtifacts = (
  sourceId: string,
  modelName: string,
  params?: PerformanceArtifactsParams,
  filterData?: ModelCatalogFilterStates,
  filterOptions?: CatalogFilterOptionsList | null,
  enabled = true,
): FetchState<CatalogPerformanceArtifactList> => {
  const { api, apiAvailable } = useModelCatalogAPI();

  const performanceParams = useNormalizedPerformanceParams(params);

  const call = React.useCallback<FetchStateCallbackPromise<CatalogPerformanceArtifactList>>(
    (opts) => {
      if (!apiAvailable) {
        return Promise.reject(new Error('API not yet available'));
      }
      if (!sourceId) {
        return Promise.reject(new NotReadyError('No source id'));
      }
      if (!modelName) {
        return Promise.reject(new NotReadyError('No model name'));
      }
      if (!enabled) {
        return Promise.reject(new NotReadyError('Fetching is disabled'));
      }
      return api.getPerformanceArtifacts(
        opts,
        sourceId,
        modelName,
        performanceParams,
        filterData,
        filterOptions,
      );
    },
    [apiAvailable, sourceId, modelName, enabled, api, performanceParams, filterData, filterOptions],
  );

  return useFetchState(
    call,
    { items: [], size: 0, pageSize: 0, nextPageToken: '' },
    {
      initialPromisePurity: true,
    },
  );
};

type PaginatedPerformanceArtifactList = {
  items: CatalogPerformanceMetricsArtifact[];
  size: number;
  pageSize: number;
  nextPageToken: string;
  loadMore: () => void;
  isLoadingMore: boolean;
  hasMore: boolean;
  loadMoreError?: Error;
  refresh: () => void;
};

type PaginatedPerformanceArtifactsResult = {
  performanceArtifacts: PaginatedPerformanceArtifactList;
  performanceArtifactsLoaded: boolean;
  performanceArtifactsLoadError: Error | undefined;
  refresh: () => void;
};

export const usePaginatedCatalogPerformanceArtifacts = (
  sourceId: string,
  modelName: string,
  params?: PerformanceArtifactsParams,
  filterData?: ModelCatalogFilterStates,
  filterOptions?: CatalogFilterOptionsList | null,
  enabled = true,
): PaginatedPerformanceArtifactsResult => {
  const { api, apiAvailable } = useModelCatalogAPI();

  const [allItems, setAllItems] = React.useState<CatalogPerformanceMetricsArtifact[]>([]);
  const [nextPageToken, setNextPageToken] = React.useState('');
  const [isLoadingMore, setIsLoadingMore] = React.useState(false);
  const isLoadingMoreRef = React.useRef(false);
  const [loadMoreError, setLoadMoreError] = React.useState<Error | undefined>();
  const queryVersionRef = React.useRef(0);
  const queryVersion = queryVersionRef.current;

  const normalizedParams = useNormalizedPerformanceParams(params);

  const firstPageParams: PerformanceArtifactsParams | undefined = React.useMemo(
    () => ({ ...normalizedParams, nextPageToken: undefined }),
    [normalizedParams],
  );

  const fetchFirstPage = React.useCallback<
    FetchStateCallbackPromise<CatalogPerformanceArtifactList>
  >(
    (opts) => {
      if (!apiAvailable) {
        return Promise.reject(new Error('API not yet available'));
      }
      if (!sourceId) {
        return Promise.reject(new NotReadyError('No source id'));
      }
      if (!modelName) {
        return Promise.reject(new NotReadyError('No model name'));
      }
      if (!enabled) {
        return Promise.reject(new NotReadyError('Fetching is disabled'));
      }
      return api.getPerformanceArtifacts(
        opts,
        sourceId,
        modelName,
        firstPageParams,
        filterData,
        filterOptions,
      );
    },
    [api, apiAvailable, sourceId, modelName, firstPageParams, filterData, filterOptions, enabled],
  );

  const [firstPageData, loaded, error, refetch] = useFetchState(
    fetchFirstPage,
    { items: [], size: 0, pageSize: 0, nextPageToken: '' },
    { initialPromisePurity: true },
  );

  React.useEffect(() => {
    if (loaded && !error) {
      setAllItems(firstPageData.items);
      setNextPageToken(firstPageData.nextPageToken);
    }
  }, [firstPageData, loaded, error]);

  const loadMore = React.useCallback(async () => {
    if (!nextPageToken || !apiAvailable || !enabled) {
      return;
    }
    if (queryVersion !== queryVersionRef.current) {
      return;
    }
    if (isLoadingMoreRef.current) {
      return;
    }
    isLoadingMoreRef.current = true;
    setIsLoadingMore(true);
    setLoadMoreError(undefined);

    try {
      const response = await api.getPerformanceArtifacts(
        {},
        sourceId,
        modelName,
        { ...normalizedParams, nextPageToken },
        filterData,
        filterOptions,
      );

      setAllItems((prev) => [...prev, ...response.items]);
      setNextPageToken(response.nextPageToken);
    } catch (err) {
      setLoadMoreError(
        new Error(
          `Failed to load more performance artifacts: ${
            err instanceof Error ? err.message : String(err)
          }`,
        ),
      );
    } finally {
      isLoadingMoreRef.current = false;
      setIsLoadingMore(false);
    }
  }, [
    api,
    apiAvailable,
    sourceId,
    modelName,
    normalizedParams,
    filterData,
    filterOptions,
    nextPageToken,
    queryVersion,
    enabled,
  ]);

  // When query inputs change: reset pagination state, but keep current rows visible until
  // the refreshed first page arrives. This avoids a full table "flash" on sort/filter changes.
  React.useEffect(() => {
    queryVersionRef.current += 1;
    setNextPageToken('');
    isLoadingMoreRef.current = false;
    setIsLoadingMore(false);
    setLoadMoreError(undefined);
  }, [sourceId, modelName, normalizedParams, filterData, filterOptions, enabled]);

  const refresh = React.useCallback(() => {
    setAllItems([]);
    setNextPageToken('');
    isLoadingMoreRef.current = false;
    setIsLoadingMore(false);
    setLoadMoreError(undefined);
    refetch();
  }, [refetch]);

  return {
    performanceArtifacts: {
      items: allItems,
      size: firstPageData.size,
      pageSize: firstPageData.pageSize,
      nextPageToken,
      loadMore,
      isLoadingMore,
      hasMore: Boolean(nextPageToken),
      loadMoreError,
      refresh,
    },
    performanceArtifactsLoaded: loaded,
    performanceArtifactsLoadError: error,
    refresh,
  };
};
