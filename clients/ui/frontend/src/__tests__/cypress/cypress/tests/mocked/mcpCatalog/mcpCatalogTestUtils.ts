import { mockModArchResponse } from 'mod-arch-core';
import {
  mockCatalogLabel,
  mockCatalogSource,
  mockCatalogSourceList,
  mockMcpCatalogFilterOptions,
  mockMcpServer,
  mockMcpToolWithServer,
  mockMcpToolList,
} from '~/__mocks__';
import type { McpToolList } from '~/app/mcpServerCatalogTypes';
import type { CatalogSource } from '~/app/shared/types/catalogTypes';
import { MODEL_CATALOG_API_VERSION } from '~/__tests__/cypress/cypress/support/commands/api';

export { mockMcpCatalogFilterOptions, mockMcpToolWithServer, mockMcpToolList };

export const MCP_SERVERS_PATH = `/model-registry/api/${MODEL_CATALOG_API_VERSION}/mcp_catalog/mcp_servers`;
export const MCP_FILTER_OPTIONS_PATH = `/model-registry/api/${MODEL_CATALOG_API_VERSION}/mcp_catalog/mcp_servers_filter_options`;

const generateMockServers = (count: number, prefix: string, sourceId: string) =>
  Array.from({ length: count }, (_, i) =>
    mockMcpServer({
      id: `${prefix}-${i + 1}`,
      name: `${prefix} Server ${i + 1}`,
      description: `Description for ${prefix} server ${i + 1}`,
      source_id: sourceId, // eslint-disable-line camelcase
    }),
  );

const defaultSources: CatalogSource[] = [
  mockCatalogSource({
    id: 'community-mcp-source',
    name: 'Community MCP Servers',
    labels: ['community_mcp_servers'],
  }),
  mockCatalogSource({
    id: 'org-mcp-source',
    name: 'Organization MCP Servers',
    labels: ['organization_mcp_servers'],
  }),
];

type InitInterceptsConfig = {
  sources?: CatalogSource[];
  serversPerCategory?: number;
};

export const initMcpCatalogIntercepts = ({
  sources = defaultSources,
  serversPerCategory = 4,
}: InitInterceptsConfig = {}): void => {
  cy.interceptApi(
    `GET /api/:apiVersion/model_catalog/sources`,
    { path: { apiVersion: MODEL_CATALOG_API_VERSION }, query: { assetType: 'mcp_servers' } },
    mockCatalogSourceList({ items: sources }),
  );

  cy.intercept(
    {
      method: 'GET',
      url: new RegExp(`/api/${MODEL_CATALOG_API_VERSION}/model_catalog/labels`),
    },
    mockModArchResponse({
      items: [
        mockCatalogLabel({
          name: 'community_mcp_servers',
          displayName: 'Community MCP Servers',
          description: 'Community contributed MCP servers.',
        }),
        mockCatalogLabel({
          name: 'organization_mcp_servers',
          displayName: 'Organization MCP Servers',
          description: 'MCP servers provided by your organization.',
        }),
      ],
      size: 2,
      pageSize: 10,
      nextPageToken: '',
    }),
  );

  sources.forEach((source) => {
    source.labels.forEach((label) => {
      const servers = generateMockServers(
        serversPerCategory,
        label.toLowerCase().replace(/\s+/g, '-'),
        source.id,
      );

      cy.interceptApi(
        `GET /api/:apiVersion/mcp_catalog/mcp_servers`,
        {
          path: { apiVersion: MODEL_CATALOG_API_VERSION },
          query: { sourceLabel: label },
        },
        { items: servers, size: servers.length, pageSize: 10, nextPageToken: '' },
      );
    });
  });

  cy.intercept(
    { method: 'GET', pathname: MCP_FILTER_OPTIONS_PATH },
    mockModArchResponse(mockMcpCatalogFilterOptions()),
  );
};

export const initServerDetailIntercept = (server: ReturnType<typeof mockMcpServer>): void => {
  cy.intercept(
    {
      method: 'GET',
      pathname: `${MCP_SERVERS_PATH}/${server.id}`,
    },
    mockModArchResponse(server),
  );
};

export const initServerToolsIntercept = (serverId: string, toolList: McpToolList): void => {
  cy.intercept(
    {
      method: 'GET',
      pathname: `${MCP_SERVERS_PATH}/${serverId}/tools`,
    },
    mockModArchResponse(toolList),
  );
};

export const initServerToolsErrorIntercept = (serverId: string): void => {
  cy.intercept(
    {
      method: 'GET',
      pathname: `${MCP_SERVERS_PATH}/${serverId}/tools`,
    },
    { statusCode: 500, body: { error: 'Internal server error' } },
  );
};
