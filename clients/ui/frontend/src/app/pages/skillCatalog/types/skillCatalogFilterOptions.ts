import type { CatalogFilterStringOption } from '~/app/shared/components/catalog';

export type SkillFilterCategoryKey = 'provider' | 'trustTier' | 'category';

export type SkillCatalogFiltersState = {
  [K in SkillFilterCategoryKey]?: string[];
};

export type SkillCatalogFilterOptions = {
  [key in SkillFilterCategoryKey]?: CatalogFilterStringOption;
};

export type SkillCatalogFilterOptionsList = {
  filters?: SkillCatalogFilterOptions;
};
