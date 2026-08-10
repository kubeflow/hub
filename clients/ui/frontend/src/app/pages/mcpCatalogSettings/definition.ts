import { McpCatalogSettingsContextProvider } from '~/app/context/mcpCatalogSettings/McpCatalogSettingsContext';
import type { CatalogSettingsDefinition } from '~/app/shared/catalogSettings/types';
import McpCatalogSettings from './screens/McpCatalogSettings';
import McpManageSourcePage from './screens/McpManageSourcePage';

export const mcpCatalogSettingsDefinition: CatalogSettingsDefinition = {
  id: 'mcp_servers',
  ContextProvider: McpCatalogSettingsContextProvider,
  ListPage: McpCatalogSettings,
  ManagePage: McpManageSourcePage,
};
