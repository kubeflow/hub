import { mockModArchResponse } from 'mod-arch-core';
import { mockMcpServer } from '~/__mocks__';
import { mcpCatalog, mcpServerDetails } from '~/__tests__/cypress/cypress/pages/mcpCatalog';
import { appChrome } from '~/__tests__/cypress/cypress/pages/appChrome';
import {
  initMcpCatalogIntercepts,
  initServerDetailIntercept,
  initServerToolsIntercept,
  initServerToolsErrorIntercept,
  mockMcpToolWithServer,
  mockMcpToolList,
} from './mcpCatalogTestUtils';

const kubernetesServer = mockMcpServer();
const customServer = mockMcpServer({
  id: 'custom-1',
  name: 'Custom MCP Server',
  description: 'A custom MCP server without README.',
  deploymentMode: 'local',
  securityIndicators: {},
  source_id: 'sample', // eslint-disable-line camelcase
  toolCount: 0,
  readme: undefined,
});

describe('MCP Server Details Page', () => {
  beforeEach(() => {
    initMcpCatalogIntercepts();
  });

  describe('Navigation from catalog', () => {
    it('should navigate to details page when clicking server card link', () => {
      initServerDetailIntercept(kubernetesServer);
      mcpCatalog.visit();
      mcpCatalog.findMcpCatalogCards().should('have.length.at.least', 1, { timeout: 15000 });
      cy.get('[data-testid^="mcp-catalog-card-detail-link-"]').first().click();
      cy.url().should('include', '/mcp-catalog/');
    });
  });

  describe('Breadcrumb navigation', () => {
    beforeEach(() => {
      initServerDetailIntercept(kubernetesServer);
    });

    it('should display breadcrumb with MCP Catalog link and server name', () => {
      mcpServerDetails.visit(kubernetesServer.id);
      mcpServerDetails.findBreadcrumbCatalogLink().should('be.visible');
      mcpServerDetails.findBreadcrumbServerName().should('contain.text', kubernetesServer.name);
    });

    it('should navigate back to catalog when clicking breadcrumb link', () => {
      mcpServerDetails.visit(kubernetesServer.id);
      mcpServerDetails.findBreadcrumbCatalogLink().click();
      cy.url().should('include', '/mcp-catalog');
      cy.url().should('not.include', `/${kubernetesServer.id}`);
    });
  });

  describe('Server header and description', () => {
    beforeEach(() => {
      initServerDetailIntercept(kubernetesServer);
    });

    it('should display server name and description', () => {
      mcpServerDetails.visit(kubernetesServer.id);

      cy.findByTestId('app-page-title').should('contain.text', kubernetesServer.name);

      mcpServerDetails.findDescription().should('contain.text', kubernetesServer.description);
    });

    it('should not show Remote title label for local deployment', () => {
      mcpServerDetails.visit(kubernetesServer.id);
      mcpServerDetails.findRemoteTitleLabel().should('not.exist');
    });

    it('should show Remote title label when deploymentMode is remote', () => {
      const remoteServer = mockMcpServer({
        id: 'remote-header-test',
        name: 'GitHub',
        deploymentMode: 'remote',
      });
      initServerDetailIntercept(remoteServer);
      mcpServerDetails.visit(remoteServer.id);
      mcpServerDetails.findRemoteTitleLabel().should('be.visible');
      mcpServerDetails.findRemoteTitleLabel().should('contain.text', 'Remote');
    });
  });

  describe('README card', () => {
    it('should render README with markdown elements', () => {
      initServerDetailIntercept(kubernetesServer);
      mcpServerDetails.visit(kubernetesServer.id);
      mcpServerDetails.findReadmeMarkdown().should('be.visible');
      mcpServerDetails.findReadmeMarkdown().should('contain.text', 'Kubernetes MCP Server');
      mcpServerDetails.findReadmeMarkdown().find('h3').should('exist');
      mcpServerDetails.findReadmeMarkdown().find('code').should('exist');
    });

    it('should display empty state when no README is available', () => {
      initServerDetailIntercept(customServer);
      mcpServerDetails.visit(customServer.id);
      mcpServerDetails.findNoReadme().should('be.visible');
      mcpServerDetails.findNoReadme().should('contain.text', 'No README available');
    });
  });

  describe('Server details sidebar', () => {
    beforeEach(() => {
      initServerDetailIntercept(kubernetesServer);
    });

    it('should not display endpoint when API omits endpoints', () => {
      mcpServerDetails.visit(kubernetesServer.id);
      cy.findByTestId('mcp-server-endpoint-copy').should('not.exist');
    });

    it('should display endpoint with copy when endpoints are present', () => {
      const host = 'splunk-mcp-server.demo-namespace.svc.cluster.local:8080';
      const serverWithEndpoint = mockMcpServer({
        id: 'endpoint-test-server',
        name: 'Splunk MCP',
        deploymentMode: 'remote',
        endpoints: { http: host },
      });
      initServerDetailIntercept(serverWithEndpoint);
      mcpServerDetails.visit(serverWithEndpoint.id);
      mcpServerDetails.findEndpointCopy().should('be.visible');
      mcpServerDetails.findEndpointCopy().find('input').should('have.value', host);
    });

    it('should display labels, license, version, and deployment mode', () => {
      mcpServerDetails.visit(kubernetesServer.id);

      mcpServerDetails.findLabels().should('have.length.at.least', 1);

      mcpServerDetails.findLicenseLink().should('be.visible');
      mcpServerDetails.findLicenseLink().should('contain.text', kubernetesServer.license);

      mcpServerDetails.findVersion().should('contain.text', kubernetesServer.version);

      mcpServerDetails.findDeploymentMode().should('contain.text', 'Local to cluster');
    });

    it('should display artifacts, source code, provider, and transport type', () => {
      mcpServerDetails.visit(kubernetesServer.id);

      mcpServerDetails.findArtifactCopy().should('be.visible');
      mcpServerDetails
        .findArtifactCopy()
        .first()
        .find('input')
        .should('have.value', kubernetesServer.artifacts![0].uri);

      mcpServerDetails.findSourceCodeLink().should('be.visible');

      mcpServerDetails.findProvider().should('contain.text', kubernetesServer.provider);

      mcpServerDetails.findTransportType().should('contain.text', 'http-streaming');
    });
  });

  describe('Error handling', () => {
    it('should show not-found state for invalid server ID', () => {
      cy.intercept(
        { method: 'GET', url: '**/mcp_servers/invalid-id-that-does-not-exist*' },
        {
          statusCode: 404,
          body: {
            error: { code: '404', message: 'the requested resource could not be found' },
          },
        },
      );
      cy.visit('/mcp-catalog/invalid-id-that-does-not-exist');
      mcpServerDetails.findMcpNotFound().should('be.visible');
      appChrome.waitForA11y();
      cy.contains('MCP server not found').should('be.visible');
    });
  });

  describe('Browser navigation', () => {
    beforeEach(() => {
      initServerDetailIntercept(kubernetesServer);
    });

    it('should support browser back navigation', () => {
      mcpCatalog.visit();
      mcpCatalog.findMcpCatalogCards().should('have.length.at.least', 1, { timeout: 15000 });
      cy.get('[data-testid^="mcp-catalog-card-detail-link-"]').first().click();
      cy.url().should('include', '/mcp-catalog/');
      cy.go('back');
      cy.url().should('eq', `${Cypress.config().baseUrl}/mcp-catalog`);
    });
  });

  describe('Tools section', () => {
    const serverId = kubernetesServer.id;

    const threeTools = mockMcpToolList([
      mockMcpToolWithServer(serverId, {
        name: 'query',
        description: 'Execute PromQL queries',
        accessType: 'read_only',
        parameters: [
          { name: 'query', type: 'string', description: 'PromQL expression', required: true },
          { name: 'timeout', type: 'string', description: 'Timeout duration', required: false },
        ],
      }),
      mockMcpToolWithServer(serverId, {
        name: 'deploy_model',
        description: 'Deploy a model to Kubernetes',
        accessType: 'execute',
      }),
      mockMcpToolWithServer(serverId, {
        name: 'create_alert',
        description: 'Create a new alert rule',
        accessType: 'read_write',
      }),
    ]);

    const sevenTools = mockMcpToolList(
      Array.from({ length: 7 }, (_, i) =>
        mockMcpToolWithServer(serverId, {
          name: `tool_${i + 1}`,
          description: `Description for tool ${i + 1}`,
          accessType: 'read_only',
        }),
      ),
    );

    it('should show loading spinner while tools are being fetched', () => {
      initServerDetailIntercept(kubernetesServer);
      cy.intercept({ method: 'GET', pathname: `**/mcp_servers/${serverId}/tools` }, (req) => {
        req.reply({ delay: 2000, body: mockModArchResponse(mockMcpToolList([])) });
      });
      mcpServerDetails.visit(serverId);
      mcpServerDetails.findToolsLoading().should('be.visible');
    });

    it('should hide tools section when API returns empty list', () => {
      initServerDetailIntercept(kubernetesServer);
      initServerToolsIntercept(serverId, mockMcpToolList([]));
      mcpServerDetails.visit(serverId);
      mcpServerDetails.findToolsSection().should('not.exist');
    });

    it('should show error state when tools API fails', () => {
      initServerDetailIntercept(kubernetesServer);
      initServerToolsErrorIntercept(serverId);
      mcpServerDetails.visit(serverId);
      mcpServerDetails.findToolsError().should('be.visible');
      mcpServerDetails.findToolsError().should('contain.text', 'Unable to load tools.');
    });

    it('should display tools with name, access type, and description', () => {
      initServerDetailIntercept(kubernetesServer);
      initServerToolsIntercept(serverId, threeTools);
      mcpServerDetails.visit(serverId);
      mcpServerDetails.findToolsSection().should('be.visible');
      mcpServerDetails.findToolToggle('query').should('contain.text', 'query');
      mcpServerDetails.findToolToggle('query').should('contain.text', 'read-only');
      mcpServerDetails.findToolToggle('query').should('contain.text', 'Execute PromQL queries');
      mcpServerDetails.findToolToggle('deploy_model').should('contain.text', 'execute');
      mcpServerDetails.findToolToggle('create_alert').should('contain.text', 'read/write');
    });

    it('should show parameters when expanding a tool', () => {
      initServerDetailIntercept(kubernetesServer);
      initServerToolsIntercept(serverId, threeTools);
      mcpServerDetails.visit(serverId);
      mcpServerDetails.findToolToggle('query').click();
      cy.findByTestId('mcp-server-tools').within(() => {
        cy.contains('Input Parameters:').should('be.visible');
        cy.contains('query').should('be.visible');
        cy.contains('string').should('be.visible');
        cy.contains('required').should('be.visible');
        cy.contains('timeout').should('be.visible');
        cy.contains('optional').should('be.visible');
      });
    });

    it('should not show pagination when 5 or fewer tools', () => {
      initServerDetailIntercept(kubernetesServer);
      initServerToolsIntercept(serverId, threeTools);
      mcpServerDetails.visit(serverId);
      mcpServerDetails.findToolsSection().should('be.visible');
      mcpServerDetails.findToolsPageIndicator().should('not.exist');
    });

    it('should show pagination when more than 5 tools', () => {
      initServerDetailIntercept(kubernetesServer);
      initServerToolsIntercept(serverId, sevenTools);
      mcpServerDetails.visit(serverId);
      mcpServerDetails.findToolsPageIndicator().should('contain.text', '1 / 2');
      mcpServerDetails.findToolsPagePrev().should('be.disabled');
      mcpServerDetails.findToolsPageNext().should('not.be.disabled');
    });

    it('should navigate between pages', () => {
      initServerDetailIntercept(kubernetesServer);
      initServerToolsIntercept(serverId, sevenTools);
      mcpServerDetails.visit(serverId);
      mcpServerDetails.findToolToggle('tool_1').should('be.visible');
      mcpServerDetails.findToolToggle('tool_6').should('not.exist');
      mcpServerDetails.findToolsPageNext().click();
      mcpServerDetails.findToolsPageIndicator().should('contain.text', '2 / 2');
      mcpServerDetails.findToolToggle('tool_6').should('be.visible');
      mcpServerDetails.findToolToggle('tool_1').should('not.exist');
      mcpServerDetails.findToolsPageNext().should('be.disabled');
      mcpServerDetails.findToolsPagePrev().click();
      mcpServerDetails.findToolsPageIndicator().should('contain.text', '1 / 2');
    });

    it('should filter tools by name', () => {
      initServerDetailIntercept(kubernetesServer);
      initServerToolsIntercept(serverId, threeTools);
      mcpServerDetails.visit(serverId);
      mcpServerDetails.findToolsFilter().type('deploy');
      mcpServerDetails.findToolToggle('deploy_model').should('be.visible');
      mcpServerDetails.findToolToggle('query').should('not.exist');
      mcpServerDetails.findToolToggle('create_alert').should('not.exist');
    });

    it('should filter tools by description', () => {
      initServerDetailIntercept(kubernetesServer);
      initServerToolsIntercept(serverId, threeTools);
      mcpServerDetails.visit(serverId);
      mcpServerDetails.findToolsFilter().type('PromQL');
      mcpServerDetails.findToolToggle('query').should('be.visible');
      mcpServerDetails.findToolToggle('deploy_model').should('not.exist');
    });

    it('should show empty filter message when no tools match', () => {
      initServerDetailIntercept(kubernetesServer);
      initServerToolsIntercept(serverId, threeTools);
      mcpServerDetails.visit(serverId);
      mcpServerDetails.findToolsFilter().type('nonexistent_tool_xyz');
      mcpServerDetails.findToolsEmptyFilter().should('be.visible');
      mcpServerDetails
        .findToolsEmptyFilter()
        .should('contain.text', 'No tools match the filter criteria.');
    });

    it('should reset pagination when filtering', () => {
      initServerDetailIntercept(kubernetesServer);
      initServerToolsIntercept(serverId, sevenTools);
      mcpServerDetails.visit(serverId);
      mcpServerDetails.findToolsPageNext().click();
      mcpServerDetails.findToolsPageIndicator().should('contain.text', '2 / 2');
      mcpServerDetails.findToolsFilter().type('tool_1');
      mcpServerDetails.findToolsPageIndicator().should('not.exist');
      mcpServerDetails.findToolToggle('tool_1').should('be.visible');
    });

    it('should display revoked label and reason for revoked tools', () => {
      const toolsWithRevoked = mockMcpToolList([
        mockMcpToolWithServer(serverId, {
          name: 'legacy_export',
          description: 'Export metrics in legacy format',
          accessType: 'read_only',
          revoked: true,
          revokedReason: 'Deprecated in favor of the new metrics API.',
        }),
        mockMcpToolWithServer(serverId, {
          name: 'active_tool',
          description: 'An active tool',
          accessType: 'read_only',
        }),
      ]);
      initServerDetailIntercept(kubernetesServer);
      initServerToolsIntercept(serverId, toolsWithRevoked);
      mcpServerDetails.visit(serverId);
      mcpServerDetails.findToolRevokedLabel('legacy_export').should('be.visible');
      mcpServerDetails.findToolRevokedLabel('legacy_export').should('contain.text', 'revoked');
      mcpServerDetails.findToolRevokedLabel('active_tool').should('not.exist');
      mcpServerDetails.findToolToggle('legacy_export').click();
      mcpServerDetails.findToolRevokedReason('legacy_export').should('be.visible');
      mcpServerDetails
        .findToolRevokedReason('legacy_export')
        .should('contain.text', 'Deprecated in favor of the new metrics API.');
    });
  });
});
