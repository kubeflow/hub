import * as React from 'react';
import type { CatalogLabelList, CatalogSourceList } from '~/app/shared/types/catalogTypes';

export type CatalogCommonData<TFilterOptions> = {
  catalogSources: CatalogSourceList | null;
  catalogSourcesLoaded: boolean;
  catalogSourcesLoadError?: Error;
  catalogLabels: CatalogLabelList | null;
  catalogLabelsLoaded: boolean;
  catalogLabelsLoadError?: Error;
  filterOptions: TFilterOptions | null;
  filterOptionsLoaded: boolean;
  filterOptionsLoadError?: Error;
};

export type CatalogProviderState = {
  selectedSourceLabel: string | undefined;
  setSelectedSourceLabel: (label: string | undefined) => void;
};

export type CatalogContextValue<TFilterOptions> = CatalogCommonData<TFilterOptions> &
  CatalogProviderState & {
    clearAllFilters: () => void;
  };

export type CatalogContextConfig<TFilterOptions, TExtension> = {
  displayName?: string;
  initialSelectedSourceLabel?: string;
  useSetup: (providerState: CatalogProviderState) => {
    catalogData: CatalogCommonData<TFilterOptions>;
    extension: TExtension & { clearAllFilters: () => void };
  };
};

type CatalogContextResult<TFilterOptions, TExtension> = {
  Context: React.Context<CatalogContextValue<TFilterOptions> & TExtension>;
  Provider: React.FC<{ children: React.ReactNode }>;
  useContext: () => CatalogContextValue<TFilterOptions> & TExtension;
};

export function createCatalogContext<TFilterOptions, TExtension>(
  config: CatalogContextConfig<TFilterOptions, TExtension>,
): CatalogContextResult<TFilterOptions, TExtension> {
  type FullContextType = CatalogContextValue<TFilterOptions> & TExtension;

  const Context = React.createContext<FullContextType | null>(null);

  const Provider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    const [selectedSourceLabel, setSelectedSourceLabel] = React.useState<string | undefined>(
      config.initialSelectedSourceLabel,
    );

    const providerState = React.useMemo<CatalogProviderState>(
      () => ({
        selectedSourceLabel,
        setSelectedSourceLabel,
      }),
      [selectedSourceLabel],
    );

    const { catalogData, extension } = config.useSetup(providerState);

    const value = React.useMemo<FullContextType>(
      () => ({
        ...catalogData,
        ...extension,
        selectedSourceLabel,
        setSelectedSourceLabel,
      }),
      [catalogData, extension, selectedSourceLabel],
    );

    return <Context.Provider value={value}>{children}</Context.Provider>;
  };

  Provider.displayName = config.displayName ?? 'CatalogContextProvider';

  const useCtx = (): FullContextType => {
    const context = React.useContext(Context);
    if (context === null) {
      throw new Error(`${config.displayName ?? 'CatalogContext'} must be used within its Provider`);
    }
    return context;
  };

  return {
    // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
    Context: Context as unknown as React.Context<FullContextType>,
    Provider,
    useContext: useCtx,
  };
}
