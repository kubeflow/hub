import {
  APIOptions,
  FetchState,
  FetchStateCallbackPromise,
  NotReadyError,
  useFetchState,
} from 'mod-arch-core';
import * as React from 'react';

type SourceConfigByIdApiState<TConfig> = {
  apiAvailable: boolean;
  api: {
    getSourceConfig: (opts: APIOptions, sourceId: string) => Promise<TConfig>;
  };
};

export const useSourceConfigById = <TConfig>(
  apiState: SourceConfigByIdApiState<TConfig>,
  sourceId: string,
): FetchState<TConfig | null> => {
  const call = React.useCallback<FetchStateCallbackPromise<TConfig | null>>(
    (opts) => {
      if (!apiState.apiAvailable) {
        return Promise.reject(new Error('API not yet available'));
      }
      if (!sourceId) {
        return Promise.reject(new NotReadyError('No source id'));
      }

      return apiState.api.getSourceConfig(opts, sourceId);
    },
    [apiState, sourceId],
  );
  return useFetchState(call, null, { initialPromisePurity: true });
};
