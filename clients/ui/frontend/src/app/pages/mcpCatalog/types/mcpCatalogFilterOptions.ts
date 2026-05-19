import type { CatalogFilterStringOption } from '~/app/shared/components/catalog';

export type McpFilterCategoryKey =
  | 'deploymentMode'
  | 'supportedTransports'
  | 'license'
  | 'labels'
  | 'securityIndicators';

export type McpCatalogFiltersState = {
  [K in McpFilterCategoryKey]?: string[];
};

export type McpCatalogFilterOptions = {
  [key in McpFilterCategoryKey]?: CatalogFilterStringOption;
};

export type McpCatalogFilterOptionsList = {
  filters?: McpCatalogFilterOptions;
};
