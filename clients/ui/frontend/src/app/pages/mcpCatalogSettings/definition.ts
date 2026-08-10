import { McpCatalogSettingsContextProvider } from '~/app/context/mcpCatalogSettings/McpCatalogSettingsContext';
import McpCatalogSettings from './screens/McpCatalogSettings';
import McpManageSourcePage from './screens/McpManageSourcePage';
import type { CatalogSettingsDefinition } from '~/app/shared/catalogSettings/types';

export const mcpCatalogSettingsDefinition: CatalogSettingsDefinition = {
  id: 'mcp_servers',
  ContextProvider: McpCatalogSettingsContextProvider,
  ListPage: McpCatalogSettings,
  ManagePage: McpManageSourcePage,
};
