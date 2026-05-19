import { FetchState, FetchStateCallbackPromise, useFetchState } from 'mod-arch-core';
import React from 'react';
import { CatalogFilterOptionsList } from '~/app/modelCatalogTypes';
import { McpCatalogContext } from '~/app/context/mcpCatalog/McpCatalogContext';
import type { ModelCatalogAPIState } from '~/app/hooks/modelCatalog/useModelCatalogAPIState';
import { BACKEND_TO_FRONTEND_FILTER_KEY, MCP_FILTER_KEYS } from '~/app/pages/mcpCatalog/const';
import type { CatalogFilterStringOption } from '~/app/shared/components/catalog';
import type {
  McpCatalogFilterOptionsList,
  McpFilterCategoryKey,
} from '~/app/pages/mcpCatalog/types/mcpCatalogFilterOptions';

function isMcpFilterCategoryKey(s: string): s is McpFilterCategoryKey {
  return MCP_FILTER_KEYS.some((k) => k === s);
}

function isMcpFilterStringOption(v: unknown): v is CatalogFilterStringOption {
  if (typeof v !== 'object' || v === null || !('type' in v)) {
    return false;
  }
  const typeVal = Object.getOwnPropertyDescriptor(v, 'type')?.value;
  return typeVal === 'string';
}

export function mapBackendFilterOptions(
  raw: CatalogFilterOptionsList,
): McpCatalogFilterOptionsList {
  if (!raw.filters) {
    return { filters: undefined };
  }
  const mapped: McpCatalogFilterOptionsList['filters'] = {};
  for (const [key, value] of Object.entries(raw.filters)) {
    const frontendKey = BACKEND_TO_FRONTEND_FILTER_KEY[key] ?? key;
    if (isMcpFilterCategoryKey(frontendKey) && isMcpFilterStringOption(value)) {
      mapped[frontendKey] = value;
    }
  }
  return { filters: mapped };
}

type State = McpCatalogFilterOptionsList | null;

export const useMcpServerFilterOptionListWithAPI = (
  apiState: ModelCatalogAPIState,
): FetchState<State> => {
  const { api, apiAvailable } = apiState;
  const call = React.useCallback<FetchStateCallbackPromise<State>>(
    (opts) => {
      if (!apiAvailable) {
        return Promise.reject(new Error('API not yet available'));
      }

      return api.getMcpServerFilterOptionList(opts).then(mapBackendFilterOptions);
    },
    [api, apiAvailable],
  );
  return useFetchState(call, null, { initialPromisePurity: true });
};

export const useMcpServerFilterOptionList = (): FetchState<State> => {
  const { mcpApiState } = React.useContext(McpCatalogContext);
  return useMcpServerFilterOptionListWithAPI(mcpApiState);
};
