import type * as React from 'react';
import type { ModelCatalogAPIState } from '~/app/hooks/modelCatalog/useModelCatalogAPIState';
import type { CatalogContextValue } from '~/app/context/catalogContext/createCatalogContext';
import type {
  SkillCatalogFiltersState,
  SkillCatalogFilterOptionsList,
} from '~/app/pages/skillCatalog/types/skillCatalogFilterOptions';

export type SkillCatalogPaginationState = {
  page: number;
  pageSize: number;
  totalItems: number;
};

export type SkillCatalogExtension = {
  searchQuery: string;
  setSearchQuery: (query: string) => void;
  filters: SkillCatalogFiltersState;
  setFilters: React.Dispatch<React.SetStateAction<SkillCatalogFiltersState>>;
  pagination: SkillCatalogPaginationState;
  setPage: (page: number) => void;
  setPageSize: (pageSize: number) => void;
  setTotalItems: (totalItems: number) => void;
  clearAllFilters: () => void;
  skillApiState: ModelCatalogAPIState;
  emptyCategoryLabels: Set<string>;
  categoriesResolved: boolean;
  reportCategoryEmpty: (label: string, isEmpty: boolean) => void;
  setCategoryCount: (count: number) => void;
};

export type SkillCatalogContextType = CatalogContextValue<SkillCatalogFilterOptionsList> &
  SkillCatalogExtension;
