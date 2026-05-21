import type { gridSpans } from '@patternfly/react-core';

export type CatalogGridSpans = {
  sm?: gridSpans;
  md?: gridSpans;
  lg?: gridSpans;
  xl?: gridSpans;
  xl2?: gridSpans;
};

export const DEFAULT_CATALOG_GRID_SPANS: CatalogGridSpans = {
  sm: 6,
  md: 6,
  lg: 6,
  xl: 6,
  xl2: 3,
};

export type CatalogFilterStringOption<T extends string = string> = {
  type: 'string';
  values?: T[];
};

export type CatalogFilterNumberOption = {
  type: 'number';
  range?: {
    max?: number;
    min?: number;
  };
};
