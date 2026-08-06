import { FetchState } from 'mod-arch-core';
import { CatalogSourceConfigList } from '~/app/modelCatalogTypes';
import { useSourceConfigs } from '~/app/shared/catalogSettings/hooks/useSourceConfigs';
import { ModelCatalogSettingsAPIState } from './useModelCatalogSettingsAPIState';

export const useCatalogSourceConfigs = (
  apiState: ModelCatalogSettingsAPIState,
): FetchState<CatalogSourceConfigList> =>
  useSourceConfigs(apiState.apiAvailable, apiState.api.getCatalogSourceConfigs, { catalogs: [] });
