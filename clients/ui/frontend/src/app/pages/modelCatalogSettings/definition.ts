import { ModelCatalogSettingsContextProvider } from '~/app/context/modelCatalogSettings/ModelCatalogSettingsContext';
import type { CatalogSettingsDefinition } from '~/app/shared/catalogSettings/types';
import ModelCatalogSettings from './screens/ModelCatalogSettings';
import ManageSourcePage from './screens/ManageSourcePage';

export const modelCatalogSettingsDefinition: CatalogSettingsDefinition = {
  ContextProvider: ModelCatalogSettingsContextProvider,
  ListPage: ModelCatalogSettings,
  ManagePage: ManageSourcePage,
};
