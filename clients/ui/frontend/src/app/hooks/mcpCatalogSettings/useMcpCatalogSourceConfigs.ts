import { FetchState } from 'mod-arch-core';
import { McpCatalogSourceConfigList } from '~/app/mcpServerCatalogTypes';
import { useSourceConfigs } from '~/app/shared/catalogSettings/hooks/useSourceConfigs';
import { McpCatalogSettingsAPIState } from './useMcpCatalogSettingsAPIState';

export const useMcpCatalogSourceConfigs = (
  apiState: McpCatalogSettingsAPIState,
): FetchState<McpCatalogSourceConfigList> =>
  useSourceConfigs(apiState.apiAvailable, apiState.api.getMcpCatalogSourceConfigs, {
    catalogs: [],
  });
