import { FilterConfigMap, FilterState } from 'mod-arch-shared';

export enum ModelRegistryFilterOptions {
  keyword = 'Keyword',
  owner = 'Owner',
}

export type ModelRegistryFilterDataType = Record<ModelRegistryFilterOptions, string | undefined>;

export const registeredModelsFilterConfig: FilterConfigMap<ModelRegistryFilterOptions> = {
  [ModelRegistryFilterOptions.keyword]: {
    type: 'text',
    label: 'Keyword',
    placeholder: 'Filter by name, description or label',
  },
  [ModelRegistryFilterOptions.owner]: {
    type: 'text',
    label: 'Owner',
    placeholder: 'Filter by owner',
  },
};

export const registeredModelsVisibleFilterKeys = [
  ModelRegistryFilterOptions.keyword,
  ModelRegistryFilterOptions.owner,
] as const;

export const registeredModelsInitialFilterValues: FilterState<ModelRegistryFilterOptions> = {
  [ModelRegistryFilterOptions.keyword]: '',
  [ModelRegistryFilterOptions.owner]: '',
};

export enum ModelRegistryVersionsFilterOptions {
  keyword = 'Keyword',
  author = 'Author',
}

export type ModelRegistryVersionsFilterDataType = Record<
  ModelRegistryVersionsFilterOptions,
  string | undefined
>;

export const modelVersionsFilterConfig: FilterConfigMap<ModelRegistryVersionsFilterOptions> = {
  [ModelRegistryVersionsFilterOptions.keyword]: {
    type: 'text',
    label: 'Keyword',
    placeholder: 'Filter by name, description or label',
  },
  [ModelRegistryVersionsFilterOptions.author]: {
    type: 'text',
    label: 'Author',
    placeholder: 'Filter by author',
  },
};

export const modelVersionsVisibleFilterKeys = [
  ModelRegistryVersionsFilterOptions.keyword,
  ModelRegistryVersionsFilterOptions.author,
] as const;

export const modelVersionsInitialFilterValues: FilterState<ModelRegistryVersionsFilterOptions> = {
  [ModelRegistryVersionsFilterOptions.keyword]: '',
  [ModelRegistryVersionsFilterOptions.author]: '',
};

export enum RegistrationMode {
  Register = 'register',
  RegisterAndStore = 'registerAndStore',
}
