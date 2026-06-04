import * as React from 'react';
import { useQueryParamNamespaces } from 'mod-arch-core';
import useModelCatalogSettingsAPIState, {
  ModelCatalogSettingsAPIState,
} from '~/app/hooks/modelCatalogSettings/useModelCatalogSettingsAPIState';
import { useCatalogSourceConfigs } from '~/app/hooks/modelCatalogSettings/useCatalogSourceConfigs';
import type { CatalogSourceList } from '~/app/shared/types/catalogTypes';
import type { CatalogSourceConfigList } from '~/app/modelCatalogTypes';
import { BFF_API_VERSION, URL_PREFIX } from '~/app/utilities/const';
import useModelCatalogAPIState from '~/app/hooks/modelCatalog/useModelCatalogAPIState';
import { useCatalogSourcesWithPolling } from '~/app/hooks/modelCatalogSettings/useCatalogSourcesWithPolling';

export type ModelCatalogSettingsContextType = {
  apiState: ModelCatalogSettingsAPIState;
  refreshAPIState: () => void;
  catalogSourceConfigs: CatalogSourceConfigList | null;
  catalogSourceConfigsLoaded: boolean;
  catalogSourceConfigsLoadError?: Error;
  refreshCatalogSourceConfigs: () => void;
  catalogSources: CatalogSourceList | null;
  catalogSourcesLoaded: boolean;
  catalogSourcesLoadError?: Error;
  refreshCatalogSources: () => void;
};

type ModelCatalogSettingsContextProviderProps = {
  children: React.ReactNode;
};

export const ModelCatalogSettingsContext = React.createContext<ModelCatalogSettingsContextType>({
  // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
  apiState: { apiAvailable: false, api: null as unknown as ModelCatalogSettingsAPIState['api'] },
  refreshAPIState: () => undefined,
  catalogSourceConfigs: null,
  catalogSourceConfigsLoaded: false,
  catalogSourceConfigsLoadError: undefined,
  refreshCatalogSourceConfigs: () => undefined,
  catalogSources: null,
  catalogSourcesLoaded: false,
  catalogSourcesLoadError: undefined,
  refreshCatalogSources: () => undefined,
});

export const ModelCatalogSettingsContextProvider: React.FC<
  ModelCatalogSettingsContextProviderProps
> = ({ children }) => {
  const hostPath = `${URL_PREFIX}/api/${BFF_API_VERSION}/settings/model_catalog`;
  const catalogHostPath = `${URL_PREFIX}/api/${BFF_API_VERSION}/model_catalog`;
  const queryParams = useQueryParamNamespaces();
  const [apiState, refreshAPIState] = useModelCatalogSettingsAPIState(hostPath, queryParams);
  const [catalogAPIState] = useModelCatalogAPIState(catalogHostPath, queryParams);
  const [
    catalogSourceConfigs,
    catalogSourceConfigsLoaded,
    catalogSourceConfigsLoadError,
    refreshCatalogSourceConfigs,
  ] = useCatalogSourceConfigs(apiState);

  // Fetch catalog sources with polling for status updates
  const [catalogSources, catalogSourcesLoaded, catalogSourcesLoadError, refreshCatalogSources] =
    useCatalogSourcesWithPolling(catalogAPIState);

  const contextValue = React.useMemo(
    () => ({
      apiState,
      refreshAPIState,
      catalogSourceConfigs,
      catalogSourceConfigsLoaded,
      catalogSourceConfigsLoadError,
      refreshCatalogSourceConfigs,
      catalogSources,
      catalogSourcesLoaded,
      catalogSourcesLoadError,
      refreshCatalogSources,
    }),
    [
      apiState,
      refreshAPIState,
      catalogSourceConfigs,
      catalogSourceConfigsLoaded,
      catalogSourceConfigsLoadError,
      refreshCatalogSourceConfigs,
      catalogSources,
      catalogSourcesLoaded,
      catalogSourcesLoadError,
      refreshCatalogSources,
    ],
  );

  return (
    <ModelCatalogSettingsContext.Provider value={contextValue}>
      {children}
    </ModelCatalogSettingsContext.Provider>
  );
};
