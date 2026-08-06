import { FetchState } from 'mod-arch-core';
import * as React from 'react';
import { McpCatalogSourceConfig } from '~/app/mcpServerCatalogTypes';
import { McpCatalogSettingsContext } from '~/app/context/mcpCatalogSettings/McpCatalogSettingsContext';
import { useSourceConfigById } from '~/app/shared/catalogSettings/hooks/useSourceConfigById';

export const useMcpCatalogSourceConfigBySourceId = (
  sourceId: string,
): FetchState<McpCatalogSourceConfig | null> => {
  const { apiState } = React.useContext(McpCatalogSettingsContext);
  return useSourceConfigById(
    apiState.apiAvailable,
    apiState.api.getMcpCatalogSourceConfig,
    sourceId,
  );
};
