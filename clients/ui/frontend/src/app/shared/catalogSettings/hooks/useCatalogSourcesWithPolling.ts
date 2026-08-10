import { FetchState, FetchStateCallbackPromise, useFetchState, POLL_INTERVAL } from 'mod-arch-core';
import * as React from 'react';
import { CatalogSourceList } from '~/app/shared/types/catalogTypes';
import type { CatalogSourcesPollingAPIState } from '~/app/shared/catalogSettings/types';

export const useCatalogSourcesWithPolling = (
  apiState: CatalogSourcesPollingAPIState,
): FetchState<CatalogSourceList> => {
  const call = React.useCallback<FetchStateCallbackPromise<CatalogSourceList>>(
    (opts) => {
      if (!apiState.apiAvailable) {
        return Promise.reject(new Error('API not yet available'));
      }

      return apiState.api.getListSources(opts);
    },
    [apiState],
  );
  return useFetchState(
    call,
    { items: [], size: 0, pageSize: 0, nextPageToken: '' },
    { initialPromisePurity: true, refreshRate: POLL_INTERVAL },
  );
};
