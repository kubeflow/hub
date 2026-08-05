import {
  APIOptions,
  FetchState,
  FetchStateCallbackPromise,
  NotReadyError,
  useFetchState,
} from 'mod-arch-core';
import * as React from 'react';

export const useSourceConfigById = <TConfig>(
  apiAvailable: boolean,
  getSourceConfig: (opts: APIOptions, sourceId: string) => Promise<TConfig>,
  sourceId: string,
): FetchState<TConfig | null> => {
  const call = React.useCallback<FetchStateCallbackPromise<TConfig | null>>(
    (opts) => {
      if (!apiAvailable) {
        return Promise.reject(new Error('API not yet available'));
      }
      if (!sourceId) {
        return Promise.reject(new NotReadyError('No source id'));
      }

      return getSourceConfig(opts, sourceId);
    },
    [apiAvailable, getSourceConfig, sourceId],
  );
  return useFetchState(call, null, { initialPromisePurity: true });
};
