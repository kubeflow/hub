import { FetchState, FetchStateCallbackPromise, useFetchState } from 'mod-arch-core';
import * as React from 'react';
import { CatalogFilterOptionsList } from '~/app/modelCatalogTypes';
import type { ModelCatalogAPIState } from '~/app/hooks/modelCatalog/useModelCatalogAPIState';
import {
  BACKEND_TO_FRONTEND_SKILL_FILTER_KEY,
  SKILL_FILTER_KEYS,
} from '~/app/pages/skillCatalog/const';
import type { CatalogFilterStringOption } from '~/app/shared/components/catalog';
import type {
  SkillCatalogFilterOptionsList,
  SkillFilterCategoryKey,
} from '~/app/pages/skillCatalog/types/skillCatalogFilterOptions';

function isSkillFilterCategoryKey(s: string): s is SkillFilterCategoryKey {
  return SKILL_FILTER_KEYS.some((k) => k === s);
}

function isFilterStringOption(v: unknown): v is CatalogFilterStringOption {
  if (typeof v !== 'object' || v === null || !('type' in v)) {
    return false;
  }
  const typeVal = Object.getOwnPropertyDescriptor(v, 'type')?.value;
  return typeVal === 'string';
}

export function mapBackendFilterOptions(
  raw: CatalogFilterOptionsList,
): SkillCatalogFilterOptionsList {
  if (!raw.filters) {
    return { filters: undefined };
  }
  const mapped: SkillCatalogFilterOptionsList['filters'] = {};
  for (const [key, value] of Object.entries(raw.filters)) {
    const frontendKey = BACKEND_TO_FRONTEND_SKILL_FILTER_KEY[key] ?? key;
    if (isSkillFilterCategoryKey(frontendKey) && isFilterStringOption(value)) {
      mapped[frontendKey] = value;
    }
  }
  return { filters: mapped };
}

type State = SkillCatalogFilterOptionsList | null;

export const useSkillFilterOptionListWithAPI = (
  apiState: ModelCatalogAPIState,
): FetchState<State> => {
  const { api, apiAvailable } = apiState;
  const call = React.useCallback<FetchStateCallbackPromise<State>>(
    (opts) => {
      if (!apiAvailable) {
        return Promise.reject(new Error('API not yet available'));
      }
      return api.getSkillFilterOptionList(opts).then(mapBackendFilterOptions);
    },
    [api, apiAvailable],
  );
  return useFetchState(call, null, { initialPromisePurity: true });
};
