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
