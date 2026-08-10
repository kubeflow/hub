import * as React from 'react';
import useMcpCatalogSettingsAPIState, {
  McpCatalogSettingsAPIState,
} from '~/app/hooks/mcpCatalogSettings/useMcpCatalogSettingsAPIState';
import { useMcpCatalogSourceConfigs } from '~/app/hooks/mcpCatalogSettings/useMcpCatalogSourceConfigs';
import type { McpCatalogSourceConfigList } from '~/app/mcpServerCatalogTypes';
import type { CatalogSourceList } from '~/app/shared/types/catalogTypes';
import { BFF_API_VERSION, URL_PREFIX } from '~/app/utilities/const';
import { createCatalogSettingsContext } from '~/app/shared/catalogSettings/createCatalogSettingsContext';

// MCP preview quirk: preview calls the model_catalog settings host, not mcp_catalog.
const { useCatalogSettingsValue } = createCatalogSettingsContext<
  McpCatalogSettingsAPIState,
  McpCatalogSourceConfigList
>({
  settingsHostPath: `${URL_PREFIX}/api/${BFF_API_VERSION}/settings/mcp_catalog`,
  catalogHostPath: `${URL_PREFIX}/api/${BFF_API_VERSION}/model_catalog`,
  catalogExtraQueryParams: { assetType: 'mcp_servers' },
  previewHostPath: `${URL_PREFIX}/api/${BFF_API_VERSION}/settings/model_catalog`,
  useSettingsAPIState: useMcpCatalogSettingsAPIState,
  useSourceConfigsList: useMcpCatalogSourceConfigs,
});

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
  const {
    apiState,
    refreshAPIState,
    sourceConfigs: mcpCatalogSourceConfigs,
    sourceConfigsLoaded: mcpCatalogSourceConfigsLoaded,
    sourceConfigsLoadError: mcpCatalogSourceConfigsLoadError,
    refreshSourceConfigs: refreshMcpCatalogSourceConfigs,
    catalogSources: mcpCatalogSources,
    catalogSourcesLoaded: mcpCatalogSourcesLoaded,
    catalogSourcesLoadError: mcpCatalogSourcesLoadError,
    refreshCatalogSources: refreshMcpCatalogSources,
  } = useCatalogSettingsValue();

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
