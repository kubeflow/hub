import {
  McpCatalogSourceConfigList,
  McpCatalogSourceConfig,
  McpCatalogSourceType,
} from '~/app/mcpServerCatalogTypes';

export const mockMcpCatalogSourceConfig = (
  partial?: Partial<McpCatalogSourceConfig>,
): McpCatalogSourceConfig => ({
  id: 'sample_mcp_source_1',
  name: 'MCP Source 1',
  type: McpCatalogSourceType.YAML,
  enabled: true,
  includedServers: [],
  excludedServers: [],
  isDefault: true,
  yamlCatalogPath: 'community-mcp-servers.yaml',
  ...partial,
});

export const mockMcpCatalogSourceConfigList = (
  partial?: Partial<McpCatalogSourceConfigList>,
): McpCatalogSourceConfigList => ({
  catalogs: [
    mockMcpCatalogSourceConfig({
      id: 'sample_mcp_source_1',
      name: 'Sample MCP source 1',
      isDefault: true,
      yamlCatalogPath: 'community-mcp-servers.yaml',
      includedServers: [],
      excludedServers: [],
    }),
    mockMcpCatalogSourceConfig({
      id: 'mcp_source_2',
      name: 'MCP Source 2',
      isDefault: false,
      yaml: 'source: community\nmcp_servers:\n  - name: Kubernetes\n    description: Manage Kubernetes clusters\n  - name: GitHub\n    description: GitHub integration',
      includedServers: ['server1', 'server2'],
      excludedServers: ['server3'],
      enabled: false,
    }),
    mockMcpCatalogSourceConfig({
      id: 'sample_mcp_source_4',
      name: 'Sample MCP source 4',
      isDefault: false,
      yaml: 'source: organization\nmcp_servers:\n  - name: PostgreSQL\n    description: Database management',
      includedServers: ['server1', 'server2'],
      excludedServers: ['server3'],
    }),
  ],
  ...partial,
});
