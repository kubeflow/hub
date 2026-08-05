import { FetchState } from 'mod-arch-core';
import { McpCatalogSourceConfigList } from '~/app/mcpServerCatalogTypes';
import { useSourceConfigs } from '~/app/shared/catalogSettings';
import { McpCatalogSettingsAPIState } from './useMcpCatalogSettingsAPIState';

export const useMcpCatalogSourceConfigs = (
  apiState: McpCatalogSettingsAPIState,
): FetchState<McpCatalogSourceConfigList> =>
  useSourceConfigs(
    {
      apiAvailable: apiState.apiAvailable,
      api: {
        getSourceConfigs: apiState.api.getMcpCatalogSourceConfigs,
      },
    },
    { catalogs: [] },
  );
