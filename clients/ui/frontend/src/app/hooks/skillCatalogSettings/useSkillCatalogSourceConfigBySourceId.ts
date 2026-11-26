import { FetchState } from 'mod-arch-core';
import * as React from 'react';
import { SkillCatalogSourceConfig } from '~/app/skillCatalogTypes';
import { SkillCatalogSettingsContext } from '~/app/context/skillCatalogSettings/SkillCatalogSettingsContext';
import { useSourceConfigById } from '~/app/shared/catalogSettings/hooks/useSourceConfigById';

export const useSkillCatalogSourceConfigBySourceId = (
  sourceId: string,
): FetchState<SkillCatalogSourceConfig | null> => {
  const { apiState } = React.useContext(SkillCatalogSettingsContext);
  return useSourceConfigById(
    apiState.apiAvailable,
    apiState.api.getSkillCatalogSourceConfig,
    sourceId,
  );
};
