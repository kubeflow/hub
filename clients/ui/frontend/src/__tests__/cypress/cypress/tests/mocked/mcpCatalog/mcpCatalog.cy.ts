import { mockModArchResponse } from 'mod-arch-core';
import { mockCatalogLabel, mockCatalogSourceList, mockCatalogSource } from '~/__mocks__';
import { mcpCatalog } from '~/__tests__/cypress/cypress/pages/mcpCatalog';
import { MODEL_CATALOG_API_VERSION } from '~/__tests__/cypress/cypress/support/commands/api';
import {
  initMcpCatalogIntercepts,
  MCP_FILTER_OPTIONS_PATH,
  mockMcpCatalogFilterOptions,
} from './mcpCatalogTestUtils';

describe('MCP Catalog Page', () => {
  beforeEach(() => {
    initMcpCatalogIntercepts();
  });

  it('MCP Catalog tab should be enabled in nav', () => {
    mcpCatalog.visit();
    mcpCatalog.tabEnabled();
  });

  it('should display page title and description', () => {
    mcpCatalog.visit();
    mcpCatalog.findPageTitle().should('be.visible');
    mcpCatalog.findPageDescription().should('be.visible');
  });

  it('should display MCP server cards', () => {
    mcpCatalog.visit();
    mcpCatalog.findMcpCatalogCards().should('have.length.at.least', 1, { timeout: 15000 });
  });

  it('should display sidebar filters', () => {
    mcpCatalog.visit();
    mcpCatalog.findFilter('deploymentMode').should('be.visible');
    mcpCatalog.findFilter('supportedTransports').should('be.visible');
    mcpCatalog.findFilter('license').should('be.visible');
    mcpCatalog.findFilter('labels').should('be.visible');
    mcpCatalog.findFilter('securityIndicators').should('be.visible');
  });

  it('should display Deployment mode filter with Local and Remote options', () => {
    mcpCatalog.visit();
    mcpCatalog.findFilterCheckbox('deploymentMode', 'Local').should('be.visible');
    mcpCatalog.findFilterCheckbox('deploymentMode', 'Remote').should('be.visible');
  });

  it('filter Show more should expand labels list', () => {
    mcpCatalog.visit();
    mcpCatalog.findFilterShowMore('labels').scrollIntoView();
    mcpCatalog.findFilterShowMore('labels').click();
    mcpCatalog.findFilterShowLess('labels').scrollIntoView();
    mcpCatalog.findFilterShowLess('labels').should('be.visible');
  });

  it('should display category sections with server cards', () => {
    mcpCatalog.visit();
    cy.findByTestId('mcp-category-title-community_mcp_servers', { timeout: 15000 }).should(
      'be.visible',
    );
    mcpCatalog.findMcpCategorySection().should('be.visible');
    mcpCatalog.findMcpCatalogCards().should('have.length.at.least', 4);
  });

  it('card description should be truncated', () => {
    mcpCatalog.visit();
    mcpCatalog
      .findMcpCatalogCards()
      .first()
      .should('be.visible', { timeout: 15000 })
      .find('[data-testid^="mcp-catalog-card-description-"]')
      .should('have.css', '-webkit-line-clamp', '4');
  });
});

describe('MCP Catalog Empty State', () => {
  it('should show empty state when no sources are configured', () => {
    cy.interceptApi(
      `GET /api/:apiVersion/model_catalog/sources`,
      {
        path: { apiVersion: MODEL_CATALOG_API_VERSION },
        query: { assetType: 'mcp_servers' },
      },
      mockCatalogSourceList({ items: [] }),
    );
    cy.intercept(
      {
        method: 'GET',
        url: new RegExp(`/api/${MODEL_CATALOG_API_VERSION}/model_catalog/labels`),
      },
      mockModArchResponse({ items: [], size: 0, pageSize: 10, nextPageToken: '' }),
    );
    cy.intercept(
      { method: 'GET', pathname: MCP_FILTER_OPTIONS_PATH },
      mockModArchResponse(mockMcpCatalogFilterOptions()),
    );
    mcpCatalog.visit();
    mcpCatalog.findEmptyState().should('be.visible');
  });
});

describe('MCP Catalog Empty Category Hiding', () => {
  it('should hide categories that have 0 servers from toggle', () => {
    const sources = [
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
          }),
          mockCatalogLabel({
            name: 'organization_mcp_servers',
            displayName: 'Organization MCP Servers',
          }),
        ],
        size: 2,
        pageSize: 10,
        nextPageToken: '',
      }),
    );

    cy.interceptApi(
      `GET /api/:apiVersion/mcp_catalog/mcp_servers`,
      {
        path: { apiVersion: MODEL_CATALOG_API_VERSION },
        query: { sourceLabel: 'community_mcp_servers' },
      },
      { items: [], size: 0, pageSize: 10, nextPageToken: '' },
    );

    cy.interceptApi(
      `GET /api/:apiVersion/mcp_catalog/mcp_servers`,
      {
        path: { apiVersion: MODEL_CATALOG_API_VERSION },
        query: { sourceLabel: 'organization_mcp_servers' },
      },
      {
        items: [
          {
            id: 'server-1',
            name: 'Test Server',
            description: 'test',
            sourceId: 'org-mcp-source',
          },
        ],
        size: 1,
        pageSize: 10,
        nextPageToken: '',
      },
    );

    cy.intercept(
      { method: 'GET', pathname: MCP_FILTER_OPTIONS_PATH },
      mockModArchResponse(mockMcpCatalogFilterOptions()),
    );

    mcpCatalog.visit();

    // With only 1 effective category (community is empty), the page auto-selects
    // into gallery view where the title renders via CatalogGalleryLayout
    cy.findByTestId('single-category-header').should('be.visible');
    cy.findByTestId('mcp-category-title-community_mcp_servers').should('not.exist');
  });
});

describe('MCP Catalog Error State', () => {
  it('should show error state when sources fail to load', () => {
    cy.intercept(
      {
        method: 'GET',
        url: new RegExp(
          `/api/${MODEL_CATALOG_API_VERSION}/model_catalog/sources.*assetType=mcp_servers`,
        ),
      },
      { statusCode: 500, body: 'Internal Server Error' },
    );
    cy.intercept(
      {
        method: 'GET',
        url: new RegExp(`/api/${MODEL_CATALOG_API_VERSION}/model_catalog/labels`),
      },
      mockModArchResponse({ items: [], size: 0, pageSize: 10, nextPageToken: '' }),
    );
    cy.intercept(
      { method: 'GET', pathname: MCP_FILTER_OPTIONS_PATH },
      mockModArchResponse(mockMcpCatalogFilterOptions()),
    );
    mcpCatalog.visit();
    cy.contains('MCP catalog source load error', { timeout: 15000 }).should('be.visible');
  });
});
