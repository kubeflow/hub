import { FetchState } from 'mod-arch-core';
import * as React from 'react';
import { CatalogSourceConfig } from '~/app/modelCatalogTypes';
import { ModelCatalogSettingsContext } from '~/app/context/modelCatalogSettings/ModelCatalogSettingsContext';
import { useSourceConfigById } from '~/app/shared/catalogSettings';

export const useCatalogSourceConfigBySourceId = (
  sourceId: string,
): FetchState<CatalogSourceConfig | null> => {
  const { apiState } = React.useContext(ModelCatalogSettingsContext);
  return useSourceConfigById(
    {
      apiAvailable: apiState.apiAvailable,
      api: {
        getSourceConfig: apiState.api.getCatalogSourceConfig,
      },
    },
    sourceId,
  );
};
