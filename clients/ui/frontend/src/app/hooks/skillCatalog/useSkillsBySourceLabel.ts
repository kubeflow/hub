import React from 'react';
import { FetchStateCallbackPromise, useFetchState } from 'mod-arch-core';
import { Skill, SkillList, SkillListParams } from '~/app/skillCatalogTypes';
import { SkillCatalogContext } from '~/app/context/skillCatalog/SkillCatalogContext';
import type { ModelCatalogAPIState } from '~/app/hooks/modelCatalog/useModelCatalogAPIState';

type PaginatedSkillList = {
  items: Skill[];
  size: number;
  pageSize: number;
  nextPageToken: string;
  loadMore: () => void;
  isLoadingMore: boolean;
  hasMore: boolean;
  refresh: () => void;
  loadMoreError?: Error;
};

export type SkillsResult = {
  skills: PaginatedSkillList;
  skillsLoaded: boolean;
  skillsLoadError: Error | undefined;
  refresh: () => void;
};

type UseSkillsBySourceLabelParams = {
  sourceLabel?: string;
  pageSize?: number;
  searchQuery?: string;
  filterQuery?: string;
};

export function useSkillsBySourceLabelWithAPI(
  apiState: ModelCatalogAPIState,
  params: UseSkillsBySourceLabelParams,
): SkillsResult {
  const { sourceLabel, pageSize = 10, searchQuery = '', filterQuery } = params;
  const { api, apiAvailable } = apiState;

  const [allItems, setAllItems] = React.useState<Skill[]>([]);
  const [totalSize, setTotalSize] = React.useState(0);
  const [nextPageTokenValue, setNextPageTokenValue] = React.useState('');
  const [isLoadingMore, setIsLoadingMore] = React.useState(false);
  const [loadMoreError, setLoadMoreError] = React.useState<Error | undefined>();

  const buildSkillListParams = React.useCallback(
    (nextPageToken?: string): SkillListParams => ({
      sourceLabel,
      pageSize: pageSize.toString(),
      ...(nextPageToken !== undefined && nextPageToken !== '' && { nextPageToken }),
      q: searchQuery.trim() || undefined,
      filterQuery,
    }),
    [sourceLabel, pageSize, searchQuery, filterQuery],
  );

  const fetchSkills = React.useCallback<FetchStateCallbackPromise<SkillList>>(
    (opts) => {
      if (!apiAvailable) {
        return Promise.reject(new Error('API not yet available'));
      }
      return api.getSkillList(opts, buildSkillListParams());
    },
    [api, apiAvailable, buildSkillListParams],
  );

  const [firstPageData, loaded, error, refetch] = useFetchState(
    fetchSkills,
    { items: [], size: 0, pageSize: 10, nextPageToken: '' },
    { initialPromisePurity: true },
  );

  React.useEffect(() => {
    if (loaded && !error) {
      setAllItems(firstPageData.items ?? []);
      setTotalSize(firstPageData.size);
      setNextPageTokenValue(firstPageData.nextPageToken);
    }
  }, [firstPageData, loaded, error]);

  const loadMore = React.useCallback(async () => {
    if (!nextPageTokenValue || isLoadingMore || !apiAvailable) {
      return;
    }
    setIsLoadingMore(true);
    setLoadMoreError(undefined);
    try {
      const response = await api.getSkillList({}, buildSkillListParams(nextPageTokenValue));
      setAllItems((prev) => [...prev, ...(response.items ?? [])]);
      setTotalSize(response.size);
      setNextPageTokenValue(response.nextPageToken);
    } catch (err) {
      setLoadMoreError(
        new Error(
          `Failed to load more skills: ${err instanceof Error ? err.message : String(err)}`,
        ),
      );
    } finally {
      setIsLoadingMore(false);
    }
  }, [api, apiAvailable, buildSkillListParams, isLoadingMore, nextPageTokenValue]);

  React.useEffect(() => {
    setAllItems([]);
    setTotalSize(0);
    setNextPageTokenValue('');
    setIsLoadingMore(false);
    setLoadMoreError(undefined);
  }, [sourceLabel, pageSize, searchQuery, filterQuery]);

  const refresh = React.useCallback(() => {
    setAllItems([]);
    setTotalSize(0);
    setNextPageTokenValue('');
    setIsLoadingMore(false);
    setLoadMoreError(undefined);
    refetch();
  }, [refetch]);

  return {
    skills: {
      items: allItems,
      size: totalSize,
      pageSize: firstPageData.pageSize,
      nextPageToken: nextPageTokenValue,
      loadMore,
      isLoadingMore,
      hasMore: Boolean(nextPageTokenValue),
      refresh,
      loadMoreError,
    },
    skillsLoaded: loaded,
    skillsLoadError: error,
    refresh,
  };
}

export const useSkillsBySourceLabel = (
  sourceLabel?: string,
  pageSize = 10,
  searchQuery = '',
  filterQuery?: string,
): SkillsResult => {
  const { skillApiState } = React.useContext(SkillCatalogContext);
  return useSkillsBySourceLabelWithAPI(skillApiState, {
    sourceLabel,
    pageSize,
    searchQuery,
    filterQuery,
  });
};
