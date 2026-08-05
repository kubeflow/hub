import { FetchState } from 'mod-arch-core';
import { CatalogSourceConfigList } from '~/app/modelCatalogTypes';
import { useSourceConfigs } from '~/app/shared/catalogSettings';
import { ModelCatalogSettingsAPIState } from './useModelCatalogSettingsAPIState';

export const useCatalogSourceConfigs = (
  apiState: ModelCatalogSettingsAPIState,
): FetchState<CatalogSourceConfigList> =>
  useSourceConfigs(
    {
      apiAvailable: apiState.apiAvailable,
      api: {
        getSourceConfigs: apiState.api.getCatalogSourceConfigs,
      },
    },
    { catalogs: [] },
  );
