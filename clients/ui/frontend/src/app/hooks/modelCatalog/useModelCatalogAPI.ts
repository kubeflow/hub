import React from 'react';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import { ModelCatalogAPIState } from './useModelCatalogAPIState';

type UseModelRegistryAPI = ModelCatalogAPIState & {
  refreshAllAPI: () => void;
};

export const useModelCatalogAPI = (): UseModelRegistryAPI => {
  const { apiState, refreshAPIState: refreshAllAPI } = React.useContext(ModelCatalogContext);

  return {
    refreshAllAPI,
    ...apiState,
  };
};
