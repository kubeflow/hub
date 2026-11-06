import { APIState, useAPIState } from 'mod-arch-core';
import React from 'react';
import {
  createSkillCatalogSourceConfig,
  deleteSkillCatalogSourceConfig,
  getSkillCatalogSourceConfig,
  getSkillCatalogSourceConfigs,
  updateSkillCatalogSourceConfig,
  previewSkillCatalogSource,
} from '~/app/api/skillCatalogSettings/service';
import { SkillCatalogSettingsAPIs } from '~/app/skillCatalogTypes';

export type SkillCatalogSettingsAPIState = APIState<SkillCatalogSettingsAPIs>;

const useSkillCatalogSettingsAPIState = (
  hostPath: string | null,
  queryParameters?: Record<string, unknown>,
  previewHostPath?: string | null,
): [apiState: SkillCatalogSettingsAPIState, refreshAPIState: () => void] => {
  const createAPI = React.useCallback(
    (path: string) => ({
      getSkillCatalogSourceConfigs: getSkillCatalogSourceConfigs(path, queryParameters),
      createSkillCatalogSourceConfig: createSkillCatalogSourceConfig(path, queryParameters),
      getSkillCatalogSourceConfig: getSkillCatalogSourceConfig(path, queryParameters),
      updateSkillCatalogSourceConfig: updateSkillCatalogSourceConfig(path, queryParameters),
      deleteSkillCatalogSourceConfig: deleteSkillCatalogSourceConfig(path, queryParameters),
      previewSkillCatalogSource: previewSkillCatalogSource(
        previewHostPath || path,
        queryParameters,
      ),
    }),
    [queryParameters, previewHostPath],
  );

  return useAPIState(hostPath, createAPI);
};

export default useSkillCatalogSettingsAPIState;
