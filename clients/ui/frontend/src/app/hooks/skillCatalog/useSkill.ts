import { FetchState, FetchStateCallbackPromise, NotReadyError, useFetchState } from 'mod-arch-core';
import React from 'react';
import { Skill } from '~/app/skillCatalogTypes';
import { SkillCatalogContext } from '~/app/context/skillCatalog/SkillCatalogContext';
import type { ModelCatalogAPIState } from '~/app/hooks/modelCatalog/useModelCatalogAPIState';

type State = Skill | null;

export const useSkillWithAPI = (
  apiState: ModelCatalogAPIState,
  skillId: string,
): FetchState<State> => {
  const { api, apiAvailable } = apiState;

  const call = React.useCallback<FetchStateCallbackPromise<State>>(
    (opts) => {
      if (!apiAvailable) {
        return Promise.reject(new Error('API not yet available'));
      }
      if (!skillId) {
        return Promise.reject(new NotReadyError('No skill id'));
      }
      return api.getSkill(opts, skillId);
    },
    [api, apiAvailable, skillId],
  );
  return useFetchState(call, null, { initialPromisePurity: true });
};

export const useSkill = (skillId: string): FetchState<State> => {
  const { skillApiState } = React.useContext(SkillCatalogContext);
  return useSkillWithAPI(skillApiState, skillId);
};
