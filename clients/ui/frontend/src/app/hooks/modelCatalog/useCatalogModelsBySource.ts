import { useFetchState, FetchStateCallbackPromise } from 'mod-arch-core';
import React from 'react';
import {
  CatalogFilterOptionsList,
  CatalogModel,
  CatalogModelList,
  ModelCatalogFilterStates,
} from '~/app/modelCatalogTypes';
import { useModelCatalogAPI } from './useModelCatalogAPI';

type PaginatedCatalogModelList = {
  items: CatalogModel[];
  size: number;
  pageSize: number;
  nextPageToken: string;
  loadMore: () => void;
  isLoadingMore: boolean;
  hasMore: boolean;
  refresh: () => void;
};

type ModelList = {
  catalogModels: PaginatedCatalogModelList;
  catalogModelsLoaded: boolean;
  catalogModelsLoadError: Error | undefined;
  refresh: () => void;
};

export const useCatalogModelsBySources = (
  sourceId?: string,
  sourceLabel?: string,
  pageSize = 10,
  searchQuery = '',
  filterData?: ModelCatalogFilterStates,
  filterOptions?: CatalogFilterOptionsList | null,
  filterQuery?: string,
  sortBy?: string | null,
  sortOrder?: string,
  performanceParams?: {
    targetRPS?: number;
    latencyProperty?: string;
    orderBy?: string;
  },
): ModelList => {
  const { api, apiAvailable } = useModelCatalogAPI();

  const [allItems, setAllItems] = React.useState<CatalogModel[]>([]);
  const [totalSize, setTotalSize] = React.useState(0);
  const [nextPageToken, setNextPageToken] = React.useState('');
  const [isLoadingMore, setIsLoadingMore] = React.useState(false);

  const fetchModels = React.useCallback<FetchStateCallbackPromise<CatalogModelList>>(
    (opts) => {
      if (!apiAvailable) {
        return Promise.reject(new Error('API not yet available'));
      }

      return api.getCatalogModelsBySource(
        opts,
        sourceId,
        sourceLabel,
        {
          pageSize: pageSize.toString(),
          ...(sortBy && { orderBy: sortBy }),
          ...(sortOrder && { sortOrder }),
        },
        searchQuery.trim() || undefined,
        filterData,
        filterOptions,
        filterQuery,
        performanceParams,
      );
    },
    [
      api,
      apiAvailable,
      sourceId,
      pageSize,
      searchQuery,
      filterData,
      filterOptions,
      sourceLabel,
      filterQuery,
      sortBy,
      sortOrder,
      performanceParams,
    ],
  );

  const [firstPageData, loaded, error, refetch] = useFetchState(
    fetchModels,
    { items: [], size: 0, pageSize: 10, nextPageToken: '' },
    { initialPromisePurity: true },
  );

  React.useEffect(() => {
    if (loaded && !error && firstPageData.items.length > 0) {
      setAllItems(firstPageData.items);
      setTotalSize(firstPageData.size);
      setNextPageToken(firstPageData.nextPageToken);
    }
  }, [firstPageData, loaded, error]);

  const loadMore = React.useCallback(async () => {
    if (!nextPageToken || isLoadingMore || !apiAvailable) {
      return;
    }

    setIsLoadingMore(true);

    try {
      const response = await api.getCatalogModelsBySource(
        {},
        sourceId,
        sourceLabel,
        {
          pageSize: pageSize.toString(),
          nextPageToken,
          ...(sortBy && { orderBy: sortBy }),
          ...(sortOrder && { sortOrder }),
        },
        searchQuery.trim() || undefined,
        filterData,
        filterOptions,
        filterQuery,
        performanceParams,
      );

      setAllItems((prev) => [...prev, ...response.items]);
      setTotalSize(response.size);
      setNextPageToken(response.nextPageToken);
    } catch (err) {
      throw new Error(
        `Failed to load more models: ${err instanceof Error ? err.message : String(err)}`,
      );
    } finally {
      setIsLoadingMore(false);
    }
  }, [
    api,
    apiAvailable,
    sourceId,
    pageSize,
    searchQuery,
    nextPageToken,
    isLoadingMore,
    sourceLabel,
    filterData,
    filterOptions,
    filterQuery,
    sortBy,
    sortOrder,
    performanceParams,
  ]);

  React.useEffect(() => {
    setAllItems([]);
    setTotalSize(0);
    setNextPageToken('');
    setIsLoadingMore(false);
  }, [
    sourceId,
    searchQuery,
    sourceLabel,
    filterData,
    filterOptions,
    filterQuery,
    sortBy,
    sortOrder,
    performanceParams,
  ]);

  const refresh = React.useCallback(() => {
    setAllItems([]);
    setTotalSize(0);
    setNextPageToken('');
    setIsLoadingMore(false);
    refetch();
  }, [refetch]);

  const paginatedData: PaginatedCatalogModelList = {
    items: allItems,
    size: totalSize,
    pageSize: firstPageData.pageSize,
    nextPageToken,
    loadMore,
    isLoadingMore,
    hasMore: Boolean(nextPageToken),
    refresh,
  };

  return {
    catalogModels: paginatedData,
    catalogModelsLoaded: loaded,
    catalogModelsLoadError: error,
    refresh,
  };
};
