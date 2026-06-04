import { FetchState, FetchStateCallbackPromise, useFetchState } from 'mod-arch-core';
import React from 'react';
import { CatalogSourceList, CatalogSourceListParams } from '~/app/shared/types/catalogTypes';
import { ModelCatalogAPIState } from './useModelCatalogAPIState';

export const useCatalogSources = (
  apiState: ModelCatalogAPIState,
  listParams?: CatalogSourceListParams,
): FetchState<CatalogSourceList> => {
  const call = React.useCallback<FetchStateCallbackPromise<CatalogSourceList>>(
    (opts) => {
      if (!apiState.apiAvailable) {
        return Promise.reject(new Error('API not yet available'));
      }
      return apiState.api.getListSources(opts, listParams).then((data) => ({
        ...data,
        items: data.items ?? [],
      }));
    },
    [apiState, listParams],
  );
  return useFetchState(
    call,
    { items: [], size: 0, pageSize: 0, nextPageToken: '' },
    { initialPromisePurity: true },
  );
};
