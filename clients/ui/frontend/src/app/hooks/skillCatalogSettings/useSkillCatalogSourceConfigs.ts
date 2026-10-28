import { FetchState } from 'mod-arch-core';
import { SkillCatalogSourceConfigList } from '~/app/skillCatalogTypes';
import { useSourceConfigs } from '~/app/shared/catalogSettings/hooks/useSourceConfigs';
import { SkillCatalogSettingsAPIState } from './useSkillCatalogSettingsAPIState';

export const useSkillCatalogSourceConfigs = (
  apiState: SkillCatalogSettingsAPIState,
): FetchState<SkillCatalogSourceConfigList> =>
  useSourceConfigs(apiState.apiAvailable, apiState.api.getSkillCatalogSourceConfigs, {
    catalogs: [],
  });
