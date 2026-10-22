import { SkillCatalogSettingsContextProvider } from '~/app/context/skillCatalogSettings/SkillCatalogSettingsContext';
import type { CatalogSettingsDefinition } from '~/app/shared/catalogSettings/types';
import SkillCatalogSettings from './screens/SkillCatalogSettings';
import SkillManageSourcePage from './screens/SkillManageSourcePage';

export const skillCatalogSettingsDefinition: CatalogSettingsDefinition = {
  id: 'skill_catalog',
  ContextProvider: SkillCatalogSettingsContextProvider,
  ListPage: SkillCatalogSettings,
  ManagePage: SkillManageSourcePage,
};
