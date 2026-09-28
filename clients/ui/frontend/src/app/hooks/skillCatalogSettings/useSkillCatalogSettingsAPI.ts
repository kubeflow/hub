import React from 'react';
import { SkillCatalogSettingsContext } from '~/app/context/skillCatalogSettings/SkillCatalogSettingsContext';
import { SkillCatalogSettingsAPIState } from './useSkillCatalogSettingsAPIState';

type UseSkillCatalogSettingsAPI = SkillCatalogSettingsAPIState & {
  refreshAllAPI: () => void;
};

export const useSkillCatalogSettingsAPI = (): UseSkillCatalogSettingsAPI => {
  const { apiState, refreshAPIState: refreshAllAPI } = React.useContext(
    SkillCatalogSettingsContext,
  );

  return {
    refreshAllAPI,
    ...apiState,
  };
};
