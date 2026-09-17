/* eslint-disable camelcase */
import type {
  McpServer,
  McpServerList,
  McpToolWithServer,
  McpToolList,
} from '~/app/mcpServerCatalogTypes';
import type { McpCatalogFilterOptionsList } from '~/app/pages/mcpCatalog/types/mcpCatalogFilterOptions';

export const mockMcpServer = (partial?: Partial<McpServer>): McpServer => ({
  id: '1',
  name: 'Kubernetes',
  displayName: 'Kubernetes MCP',
  description: 'Control and inspect Kubernetes clusters.',
  deploymentMode: 'local',
  securityIndicators: { verifiedSource: true, sast: true },
  sourceId: 'sample',
  toolCount: 0,
  license: 'Apache-2.0',
  licenseLink: 'https://opensource.org/licenses/Apache-2.0',
  version: '1.0.0',
  provider: 'Kubernetes',
  sourceCode: 'https://github.com/kubernetes/mcp-server',
  repositoryUrl: 'https://github.com/kubernetes/mcp-server',
  artifacts: [{ uri: 'ghcr.io/kubernetes/mcp-server:latest' }],
  transports: ['http'],
  tags: ['kubernetes', 'infrastructure'],
  readme: '# Kubernetes MCP Server\n\n### Overview\n\nManage clusters with `kubectl`.',
  serverJson: {
    $schema: 'https://static.modelcontextprotocol.io/schemas/2025-12-11/server.schema.json',
    name: 'kubernetes-mcp-server',
    description: 'Control and inspect Kubernetes clusters.',
    title: 'Kubernetes',
    version: '1.0.0',
    websiteUrl: 'https://github.com/kubernetes/mcp-server',
    packages: [
      {
        environmentVariables: [
          {
            description: 'Path to kubeconfig',
            isRequired: true,
            isSecret: true,
            name: 'KUBECONFIG',
          },
        ],
        identifier: 'ghcr.io/kubernetes/mcp-server:latest',
        packageArguments: [
          { type: 'positional', value: 'npx' },
          { type: 'positional', value: '-y' },
          { type: 'positional', value: '@cncf/kubernetes-mcp-server' },
        ],
        registryType: 'oci',
        runtimeHint: 'docker',
        transport: {
          type: 'streamable-http',
          url: 'https://kubernetes-mcp-server.demo-namespace.svc.cluster.local:8080',
        },
      },
    ],
    repository: {
      source: 'github',
      url: 'https://github.com/kubernetes/mcp-server',
    },
  },
  ...partial,
});

export const mockMcpServerList = (partial?: Partial<McpServerList>): McpServerList => ({
  items: [mockMcpServer()],
  pageSize: 10,
  size: 1,
  nextPageToken: '',
  ...partial,
});

export const mockMcpServers = [
  mockMcpServer(),
  mockMcpServer({
    id: '2',
    name: 'GitHub',
    description: 'Integrate with GitHub repositories.',
    deploymentMode: 'remote',
    securityIndicators: { verifiedSource: true, secureEndpoint: true },
    sourceId: 'sample',
    toolCount: 0,
    license: 'MIT',
    version: '2.1.0',
    provider: 'GitHub',
    sourceCode: 'https://github.com/github/mcp-server',
    artifacts: [{ uri: 'ghcr.io/github/mcp-server:latest' }],
    transports: ['sse'],
    tags: ['github', 'vcs'],
  }),
  mockMcpServer({
    id: '3',
    name: 'Custom MCP Server',
    description: 'A custom MCP server without README.',
    deploymentMode: 'local',
    securityIndicators: {},
    sourceId: 'sample',
    toolCount: 0,
    readme: undefined,
  }),
];

export const mockMcpToolWithServer = (
  serverId: string,
  partial?: Partial<McpToolWithServer['tool']>,
): McpToolWithServer => ({
  serverId,
  tool: {
    name: 'test_tool',
    description: 'A test tool',
    accessType: 'read_only',
    parameters: [],
    ...partial,
  },
});

export const mockMcpToolList = (items: McpToolWithServer[]): McpToolList => ({
  items,
  size: items.length,
  pageSize: 25,
  nextPageToken: '',
});

export const mockMcpCatalogFilterOptions = (
  partial?: Partial<McpCatalogFilterOptionsList>,
): McpCatalogFilterOptionsList => ({
  filters: {
    deploymentMode: { type: 'string', values: ['Remote', 'Local'] },
    supportedTransports: { type: 'string', values: ['SSE', 'http'] },
    license: { type: 'string', values: ['MIT', 'Apache-2.0'] },
    labels: {
      type: 'string',
      values: ['kubernetes', 'github', 'database', 'monitoring', 'security', 'automation'],
    },
    securityIndicators: {
      type: 'string',
      values: ['Verified source', 'Secure endpoint', 'SAST', 'Read only tools'],
    },
  },
  ...partial,
});
