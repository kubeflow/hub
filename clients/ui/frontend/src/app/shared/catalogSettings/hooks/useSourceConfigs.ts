import { APIOptions, FetchState, FetchStateCallbackPromise, useFetchState } from 'mod-arch-core';
import * as React from 'react';

export const useSourceConfigs = <TConfigList>(
  apiAvailable: boolean,
  getSourceConfigs: (opts: APIOptions) => Promise<TConfigList>,
  initialValue: TConfigList,
): FetchState<TConfigList> => {
  const call = React.useCallback<FetchStateCallbackPromise<TConfigList>>(
    (opts) => {
      if (!apiAvailable) {
        return Promise.reject(new Error('API not yet available'));
      }

      return getSourceConfigs(opts);
    },
    [apiAvailable, getSourceConfigs],
  );
  return useFetchState(call, initialValue, { initialPromisePurity: true });
};
