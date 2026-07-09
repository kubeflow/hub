import { APIState, useAPIState } from 'mod-arch-core';
import React from 'react';
import {
  createMcpCatalogSourceConfig,
  deleteMcpCatalogSourceConfig,
  getMcpCatalogSourceConfig,
  getMcpCatalogSourceConfigs,
  updateMcpCatalogSourceConfig,
  previewMcpCatalogSource,
} from '~/app/api/mcpCatalogSettings/service';
import { McpCatalogSettingsAPIs } from '~/app/mcpServerCatalogTypes';

export type McpCatalogSettingsAPIState = APIState<McpCatalogSettingsAPIs>;

const useMcpCatalogSettingsAPIState = (
  hostPath: string | null,
  queryParameters?: Record<string, unknown>,
  previewHostPath?: string | null,
): [apiState: McpCatalogSettingsAPIState, refreshAPIState: () => void] => {
  const createAPI = React.useCallback(
    (path: string) => ({
      getMcpCatalogSourceConfigs: getMcpCatalogSourceConfigs(path, queryParameters),
      createMcpCatalogSourceConfig: createMcpCatalogSourceConfig(path, queryParameters),
      getMcpCatalogSourceConfig: getMcpCatalogSourceConfig(path, queryParameters),
      updateMcpCatalogSourceConfig: updateMcpCatalogSourceConfig(path, queryParameters),
      deleteMcpCatalogSourceConfig: deleteMcpCatalogSourceConfig(path, queryParameters),
      previewMcpCatalogSource: previewMcpCatalogSource(previewHostPath || path, queryParameters),
    }),
    [queryParameters, previewHostPath],
  );

  return useAPIState(hostPath, createAPI);
};

export default useMcpCatalogSettingsAPIState;
