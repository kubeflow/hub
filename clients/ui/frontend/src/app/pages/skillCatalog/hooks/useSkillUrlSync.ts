import * as React from 'react';
import { useSearchParams } from 'react-router-dom';
import { SKILL_FILTER_KEYS } from '~/app/pages/skillCatalog/const';
import type { SkillCatalogFiltersState } from '~/app/pages/skillCatalog/types/skillCatalogFilterOptions';

const SEARCH_PARAM = 'q';
const SOURCE_PARAM = 'source';

function filtersFromParams(params: URLSearchParams): SkillCatalogFiltersState {
  const filters: SkillCatalogFiltersState = {};
  for (const key of SKILL_FILTER_KEYS) {
    const raw = params.get(key);
    if (raw) {
      filters[key] = raw.split(',');
    }
  }
  return filters;
}

function filtersToParams(filters: SkillCatalogFiltersState, params: URLSearchParams): void {
  for (const key of SKILL_FILTER_KEYS) {
    const values = filters[key];
    if (values && values.length > 0) {
      params.set(key, values.join(','));
    } else {
      params.delete(key);
    }
  }
}

type UrlState = {
  searchQuery: string;
  filters: SkillCatalogFiltersState;
  selectedSourceLabel: string | undefined;
};

type UseSkillUrlSyncReturn = {
  initialState: UrlState;
  syncToUrl: (state: UrlState) => void;
};

export function useSkillUrlSync(): UseSkillUrlSyncReturn {
  const [searchParams, setSearchParams] = useSearchParams();

  const initialState = React.useMemo<UrlState>(
    () => ({
      searchQuery: searchParams.get(SEARCH_PARAM) || '',
      filters: filtersFromParams(searchParams),
      selectedSourceLabel: searchParams.get(SOURCE_PARAM) || undefined,
    }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [],
  );

  const syncToUrl = React.useCallback(
    (state: UrlState) => {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          if (state.searchQuery) {
            next.set(SEARCH_PARAM, state.searchQuery);
          } else {
            next.delete(SEARCH_PARAM);
          }
          if (state.selectedSourceLabel) {
            next.set(SOURCE_PARAM, state.selectedSourceLabel);
          } else {
            next.delete(SOURCE_PARAM);
          }
          filtersToParams(state.filters, next);
          return next;
        },
        { replace: true },
      );
    },
    [setSearchParams],
  );

  return { initialState, syncToUrl };
}
