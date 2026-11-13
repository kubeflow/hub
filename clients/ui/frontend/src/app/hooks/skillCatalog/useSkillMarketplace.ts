import { FetchState, FetchStateCallbackPromise, useFetchState } from 'mod-arch-core';
import React from 'react';
import { SkillMarketplaceResult } from '~/app/skillCatalogTypes';
import { SkillCatalogContext } from '~/app/context/skillCatalog/SkillCatalogContext';
import type { ModelCatalogAPIState } from '~/app/hooks/modelCatalog/useModelCatalogAPIState';

type State = SkillMarketplaceResult | null;

export const useSkillMarketplaceWithAPI = (apiState: ModelCatalogAPIState): FetchState<State> => {
  const { api, apiAvailable } = apiState;

  const call = React.useCallback<FetchStateCallbackPromise<State>>(
    (opts) => {
      if (!apiAvailable) {
        return Promise.reject(new Error('API not yet available'));
      }
      return api.getSkillMarketplace(opts);
    },
    [api, apiAvailable],
  );
  return useFetchState(call, null, { initialPromisePurity: true });
};

export const useSkillMarketplace = (): FetchState<State> => {
  const { skillApiState } = React.useContext(SkillCatalogContext);
  return useSkillMarketplaceWithAPI(skillApiState);
};
