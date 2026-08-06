import { FetchState } from 'mod-arch-core';
import * as React from 'react';
import { CatalogSourceConfig } from '~/app/modelCatalogTypes';
import { ModelCatalogSettingsContext } from '~/app/context/modelCatalogSettings/ModelCatalogSettingsContext';
import { useSourceConfigById } from '~/app/shared/catalogSettings/hooks/useSourceConfigById';

export const useCatalogSourceConfigBySourceId = (
  sourceId: string,
): FetchState<CatalogSourceConfig | null> => {
  const { apiState } = React.useContext(ModelCatalogSettingsContext);
  return useSourceConfigById(apiState.apiAvailable, apiState.api.getCatalogSourceConfig, sourceId);
};
