import { APIState, useAPIState } from 'mod-arch-core';
import React from 'react';
import {
  createCatalogSourceConfig,
  deleteCatalogSourceConfig,
  deleteCatalogSourceCredentials,
  getCatalogSourceConfig,
  getCatalogSourceConfigs,
  updateCatalogSourceConfig,
  previewCatalogSource,
} from '~/app/api/modelCatalogSettings/service';
import { ModelCatalogSettingsAPIs } from '~/app/modelCatalogTypes';

export type ModelCatalogSettingsAPIState = APIState<ModelCatalogSettingsAPIs>;

const useModelCatalogSettingsAPIState = (
  hostPath: string | null,
  queryParameters?: Record<string, unknown>,
): [apiState: ModelCatalogSettingsAPIState, refreshAPIState: () => void] => {
  const createAPI = React.useCallback(
    (path: string) => ({
      getCatalogSourceConfigs: getCatalogSourceConfigs(path, queryParameters),
      createCatalogSourceConfig: createCatalogSourceConfig(path, queryParameters),
      getCatalogSourceConfig: getCatalogSourceConfig(path, queryParameters),
      updateCatalogSourceConfig: updateCatalogSourceConfig(path, queryParameters),
      deleteCatalogSourceConfig: deleteCatalogSourceConfig(path, queryParameters),
      deleteCatalogSourceCredentials: deleteCatalogSourceCredentials(path, queryParameters),
      previewCatalogSource: previewCatalogSource(path, queryParameters),
    }),
    [queryParameters],
  );

  return useAPIState(hostPath, createAPI);
};

export default useModelCatalogSettingsAPIState;
