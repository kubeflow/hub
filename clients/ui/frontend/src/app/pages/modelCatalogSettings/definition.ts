import { ModelCatalogSettingsContextProvider } from '~/app/context/modelCatalogSettings/ModelCatalogSettingsContext';
import ModelCatalogSettings from './screens/ModelCatalogSettings';
import ManageSourcePage from './screens/ManageSourcePage';
import type { CatalogSettingsDefinition } from '~/app/shared/catalogSettings/types';

export const modelCatalogSettingsDefinition: CatalogSettingsDefinition = {
  id: 'models',
  ContextProvider: ModelCatalogSettingsContextProvider,
  ListPage: ModelCatalogSettings,
  ManagePage: ManageSourcePage,
};
