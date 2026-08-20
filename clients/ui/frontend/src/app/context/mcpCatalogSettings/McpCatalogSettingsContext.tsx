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
  pendingSourceIds: Map<string, string>;
  markSourcePending: (id: string, previousStatus: string) => void;
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
  pendingSourceIds: new Map(),
  markSourcePending: () => undefined,
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

  const [pendingSourceIds, setPendingSourceIds] = React.useState<Map<string, string>>(new Map());
  const pendingSkipCountRef = React.useRef(new Map<string, number>());
  const pollGenerationRef = React.useRef(0);
  const lastSeenGenerationRef = React.useRef(new Map<string, number>());

  const markSourcePending = React.useCallback((id: string, previousStatus: string) => {
    lastSeenGenerationRef.current.set(id, pollGenerationRef.current);
    pendingSkipCountRef.current.set(id, 3);
    setPendingSourceIds((prev) => {
      const next = new Map(prev);
      next.set(id, previousStatus);
      return next;
    });
  }, []);

  React.useEffect(() => {
    pollGenerationRef.current += 1;
    const currentGeneration = pollGenerationRef.current;

    setPendingSourceIds((prev) => {
      if (prev.size === 0) {
        return prev;
      }
      const next = new Map(prev);
      let changed = false;
      for (const [id] of prev) {
        const markedAt = lastSeenGenerationRef.current.get(id) ?? 0;
        const pollsSinceMarked = currentGeneration - markedAt;
        const skipCount = pendingSkipCountRef.current.get(id) ?? 0;

        if (pollsSinceMarked <= skipCount) {
          continue;
        }
        const source = mcpCatalogSources?.items?.find((s) => s.id === id);
        if (!source || source.status) {
          next.delete(id);
          pendingSkipCountRef.current.delete(id);
          lastSeenGenerationRef.current.delete(id);
          changed = true;
        }
      }
      return changed ? next : prev;
    });
  }, [mcpCatalogSources]);

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
      pendingSourceIds,
      markSourcePending,
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
      pendingSourceIds,
      markSourcePending,
    ],
  );

  return (
    <McpCatalogSettingsContext.Provider value={contextValue}>
      {children}
    </McpCatalogSettingsContext.Provider>
  );
};
