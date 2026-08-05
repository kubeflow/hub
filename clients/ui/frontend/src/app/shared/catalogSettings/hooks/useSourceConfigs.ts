import { APIOptions, FetchState, FetchStateCallbackPromise, useFetchState } from 'mod-arch-core';
import * as React from 'react';

type SourceConfigsApiState<TConfigList> = {
  apiAvailable: boolean;
  api: {
    getSourceConfigs: (opts: APIOptions) => Promise<TConfigList>;
  };
};

export const useSourceConfigs = <TConfigList>(
  apiState: SourceConfigsApiState<TConfigList>,
  initialValue: TConfigList,
): FetchState<TConfigList> => {
  const call = React.useCallback<FetchStateCallbackPromise<TConfigList>>(
    (opts) => {
      if (!apiState.apiAvailable) {
        return Promise.reject(new Error('API not yet available'));
      }

      return apiState.api.getSourceConfigs(opts);
    },
    [apiState],
  );
  return useFetchState(call, initialValue, { initialPromisePurity: true });
};
