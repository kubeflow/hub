import * as React from 'react';
import { useQueryParamNamespaces } from 'mod-arch-core';
import useMcpCatalogSettingsAPIState, {
  McpCatalogSettingsAPIState,
} from '~/app/hooks/mcpCatalogSettings/useMcpCatalogSettingsAPIState';
import { useMcpCatalogSourceConfigs } from '~/app/hooks/mcpCatalogSettings/useMcpCatalogSourceConfigs';
import type { McpCatalogSourceConfigList } from '~/app/mcpServerCatalogTypes';
import type { CatalogSourceList } from '~/app/shared/types/catalogTypes';
import { BFF_API_VERSION, URL_PREFIX } from '~/app/utilities/const';
import useModelCatalogAPIState from '~/app/hooks/modelCatalog/useModelCatalogAPIState';
import { useMcpCatalogSourcesWithPolling } from '~/app/hooks/mcpCatalogSettings/useMcpCatalogSourcesWithPolling';

export type McpCatalogSettingsContextType = {
  apiState: McpCatalogSettingsAPIState;
  refreshAPIState: () => void;
  mcpCatalogSourceConfigs: McpCatalogSourceConfigList | null;
  mcpCatalogSourceConfigsLoaded: boolean;
  mcpCatalogSourceConfigsLoadError?: Error;
  refreshMcpCatalogSourceConfigs: () => void;
  mcpCatalogSources: CatalogSourceList | null;
  mcpCatalogSourcesLoaded: boolean;
  mcpCatalogSourcesLoadError?: Error;
  refreshMcpCatalogSources: () => void;
};

type McpCatalogSettingsContextProviderProps = {
  children: React.ReactNode;
};

export const McpCatalogSettingsContext = React.createContext<McpCatalogSettingsContextType>({
  // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
  apiState: { apiAvailable: false, api: null as unknown as McpCatalogSettingsAPIState['api'] },
  refreshAPIState: () => undefined,
  mcpCatalogSourceConfigs: null,
  mcpCatalogSourceConfigsLoaded: false,
  mcpCatalogSourceConfigsLoadError: undefined,
  refreshMcpCatalogSourceConfigs: () => undefined,
  mcpCatalogSources: null,
  mcpCatalogSourcesLoaded: false,
  mcpCatalogSourcesLoadError: undefined,
  refreshMcpCatalogSources: () => undefined,
});

export const McpCatalogSettingsContextProvider: React.FC<
  McpCatalogSettingsContextProviderProps
> = ({ children }) => {
  const hostPath = `${URL_PREFIX}/api/${BFF_API_VERSION}/settings/mcp_catalog`;
  const mcpCatalogHostPath = `${URL_PREFIX}/api/${BFF_API_VERSION}/model_catalog`;
  const previewHostPath = `${URL_PREFIX}/api/${BFF_API_VERSION}/settings/model_catalog`;
  const queryParams = useQueryParamNamespaces();
  const [apiState, refreshAPIState] = useMcpCatalogSettingsAPIState(
    hostPath,
    queryParams,
    previewHostPath,
  );
  const mcpCatalogQueryParams = React.useMemo(
    () => ({ ...queryParams, assetType: 'mcp_servers' }),
    [queryParams],
  );
  const [mcpCatalogAPIState] = useModelCatalogAPIState(mcpCatalogHostPath, mcpCatalogQueryParams);
  const [
    mcpCatalogSourceConfigs,
    mcpCatalogSourceConfigsLoaded,
    mcpCatalogSourceConfigsLoadError,
    refreshMcpCatalogSourceConfigs,
  ] = useMcpCatalogSourceConfigs(apiState);

  const [
    mcpCatalogSources,
    mcpCatalogSourcesLoaded,
    mcpCatalogSourcesLoadError,
    refreshMcpCatalogSources,
  ] = useMcpCatalogSourcesWithPolling(mcpCatalogAPIState);

  const contextValue = React.useMemo(
    () => ({
      apiState,
      refreshAPIState,
      mcpCatalogSourceConfigs,
      mcpCatalogSourceConfigsLoaded,
      mcpCatalogSourceConfigsLoadError,
      refreshMcpCatalogSourceConfigs,
      mcpCatalogSources,
      mcpCatalogSourcesLoaded,
      mcpCatalogSourcesLoadError,
      refreshMcpCatalogSources,
    }),
    [
      apiState,
      refreshAPIState,
      mcpCatalogSourceConfigs,
      mcpCatalogSourceConfigsLoaded,
      mcpCatalogSourceConfigsLoadError,
      refreshMcpCatalogSourceConfigs,
      mcpCatalogSources,
      mcpCatalogSourcesLoaded,
      mcpCatalogSourcesLoadError,
      refreshMcpCatalogSources,
    ],
  );

  return (
    <McpCatalogSettingsContext.Provider value={contextValue}>
      {children}
    </McpCatalogSettingsContext.Provider>
  );
};
