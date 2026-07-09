export const MCP_CATALOG_SETTINGS_PAGE_TITLE = 'MCP catalog settings';
export const MCP_CATALOG_SETTINGS_DESCRIPTION =
  'Add and manage MCP catalog sources. Each source is a YAML catalog file that can contain multiple MCP servers for users in your organization.';

export const MCP_ADD_SOURCE_TITLE = 'Add a source';
export const MCP_ADD_SOURCE_DESCRIPTION = 'Add a new MCP catalog source to your organization.';

export const MCP_MANAGE_SOURCE_TITLE = 'Manage source';
export const MCP_MANAGE_SOURCE_DESCRIPTION =
  'Configure which MCP servers from this pre-loaded catalog source are visible in the MCP catalog.';

export const mcpCatalogSettingsUrl = (): string => '/mcp-catalog-settings';

export const mcpAddSourceUrl = (): string => `${mcpCatalogSettingsUrl()}/add-source`;

export const mcpManageSourceUrl = (catalogSourceId: string): string =>
  `${mcpCatalogSettingsUrl()}/manage-source/${encodeURIComponent(catalogSourceId)}`;
