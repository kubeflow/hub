import { CatalogLabel, CatalogLabelList } from '~/app/modelCatalogTypes';

export const mockCatalogLabel = (partial?: Partial<CatalogLabel>): CatalogLabel => ({
  name: 'Sample category 1',
  displayName: 'Sample Category 1',
  description: 'This is a sample category description',
  ...partial,
});

export const mockCatalogLabelList = (partial?: Partial<CatalogLabelList>): CatalogLabelList => ({
  items: [
    mockCatalogLabel({
      name: 'Red Hat AI',
      displayName: 'Red Hat AI models',
      description:
        'Red Hat AI models are curated and optimized for performance on Red Hat platforms.',
    }),
    mockCatalogLabel({
      name: 'Red Hat AI Validated',
      displayName: 'Red Hat AI Validated models',
      description:
        'Validated models are benchmarked for performance and quality using leading open source evaluation datasets. Some of these include tested runtime arguments for enabling additional capabilities.',
    }),
    mockCatalogLabel({
      name: 'Sample category 1',
      displayName: 'Sample Category 1',
      description: 'Sample category 1 description',
    }),
    mockCatalogLabel({
      name: 'Sample category 2',
      displayName: 'Sample Category 2',
      description: 'Sample category 2 description',
    }),
    mockCatalogLabel({
      name: 'Community',
      displayName: 'Community models',
      description: 'Community contributed models from various sources.',
    }),
    mockCatalogLabel({
      name: null,
      displayName: 'Other models',
      description: 'Models without a specific category label.',
    }),
  ],
  size: 6,
  pageSize: 10,
  nextPageToken: '',
  ...partial,
});
