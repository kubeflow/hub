/* eslint-disable camelcase */
import { agentsCatalog, agentDetailsPage } from '~/__tests__/cypress/cypress/pages/agentsCatalog';
import { mockAgent } from '~/__mocks__';
import { initAgentsCatalogIntercepts, interceptSingleAgent } from './agentsCatalogTestUtils';

describe('Agents Catalog Page', () => {
  beforeEach(() => {
    initAgentsCatalogIntercepts();
  });

  it('Agents Catalog tab should be enabled in nav', () => {
    agentsCatalog.visit();
    agentsCatalog.tabEnabled();
  });
});

describe('Agent Details Page', () => {
  describe('Breadcrumb', () => {
    it('should display catalog link in breadcrumb', () => {
      initAgentsCatalogIntercepts();
      const agent = mockAgent({ name: 'test-agent', displayName: 'Test Agent' });
      interceptSingleAgent(agent);

      agentDetailsPage.visit('test-agent');
      agentDetailsPage.findBreadcrumbCatalogLink().should('be.visible');
    });

    it('should display agent displayName in breadcrumb when present', () => {
      initAgentsCatalogIntercepts();
      const agent = mockAgent({ name: 'test-agent', displayName: 'My Test Agent' });
      interceptSingleAgent(agent);

      agentDetailsPage.visit('test-agent');
      agentDetailsPage.findBreadcrumbAgentName().should('contain', 'My Test Agent');
    });

    it('should fall back to agent name in breadcrumb when displayName is absent', () => {
      initAgentsCatalogIntercepts();
      const agent = mockAgent({ name: 'fallback-agent', displayName: undefined });
      interceptSingleAgent(agent);

      agentDetailsPage.visit('fallback-agent');
      agentDetailsPage.findBreadcrumbAgentName().should('contain', 'fallback-agent');
    });

    it('should navigate back to catalog when clicking breadcrumb link', () => {
      initAgentsCatalogIntercepts();
      const agent = mockAgent({ name: 'nav-agent' });
      interceptSingleAgent(agent);

      agentDetailsPage.visit('nav-agent');
      agentDetailsPage.findBreadcrumbCatalogLink().click();
      agentsCatalog.findPageTitle().should('be.visible');
    });
  });

  describe('Header', () => {
    it('should display agent displayName in page title', () => {
      initAgentsCatalogIntercepts();
      const agent = mockAgent({ name: 'title-agent', displayName: 'Title Agent' });
      interceptSingleAgent(agent);

      agentDetailsPage.visit('title-agent');
      agentDetailsPage.findPageTitle().should('contain', 'Title Agent');
    });

    it('should show fallback icon when no logo', () => {
      initAgentsCatalogIntercepts();
      const agent = mockAgent({ name: 'no-logo', logo: undefined });
      interceptSingleAgent(agent);

      agentDetailsPage.visit('no-logo');
      agentDetailsPage.findDefaultIcon().should('exist');
    });
  });

  describe('Description card', () => {
    it('should display description text when present', () => {
      initAgentsCatalogIntercepts();
      const agent = mockAgent({ name: 'desc-agent', description: 'This is a test description.' });
      interceptSingleAgent(agent);

      agentDetailsPage.visit('desc-agent');
      agentDetailsPage.findDescription().should('be.visible');
    });

    it('should show "No description" when description is undefined', () => {
      initAgentsCatalogIntercepts();
      const agent = mockAgent({ name: 'no-desc', description: undefined });
      interceptSingleAgent(agent);

      agentDetailsPage.visit('no-desc');
      agentDetailsPage.findDescription().should('contain', 'No description');
    });
  });

  describe('README card', () => {
    it('should render README markdown when present', () => {
      initAgentsCatalogIntercepts();
      const agent = mockAgent({ name: 'readme-agent', readme: '# Test README\n\nSample content.' });
      interceptSingleAgent(agent);

      agentDetailsPage.visit('readme-agent');
      agentDetailsPage.findReadmeMarkdown().should('exist');
    });

    it('should show "No README available" when readme is undefined', () => {
      initAgentsCatalogIntercepts();
      const agent = mockAgent({ name: 'no-readme', readme: undefined });
      interceptSingleAgent(agent);

      agentDetailsPage.visit('no-readme');
      agentDetailsPage.findNoReadme().should('contain', 'No README available');
    });
  });

  describe('Action buttons', () => {
    it('should show Open GitHub button when repositoryUrl is present', () => {
      initAgentsCatalogIntercepts();
      const agent = mockAgent({ name: 'gh-agent', repositoryUrl: 'https://github.com/test' });
      interceptSingleAgent(agent);

      agentDetailsPage.visit('gh-agent');
      agentDetailsPage.findGitHubButton().should('be.visible');
      agentDetailsPage
        .findGitHubButton()
        .should('have.attr', 'href', 'https://github.com/test')
        .and('have.attr', 'target', '_blank');
    });

    it('should not show Open GitHub button when repositoryUrl is absent', () => {
      initAgentsCatalogIntercepts();
      const agent = mockAgent({ name: 'no-gh', repositoryUrl: undefined });
      interceptSingleAgent(agent);

      agentDetailsPage.visit('no-gh');
      agentDetailsPage.findGitHubButton().should('not.exist');
    });
  });

  describe('Template details', () => {
    it('should show mapped framework display name in details panel', () => {
      initAgentsCatalogIntercepts();
      const agent = mockAgent({ name: 'sidebar-fw', framework: 'langgraph' });
      interceptSingleAgent(agent);

      agentDetailsPage.visit('sidebar-fw');
      agentDetailsPage.findFramework().should('contain', 'LangGraph');
    });

    it('should not show framework when absent', () => {
      initAgentsCatalogIntercepts();
      const agent = mockAgent({ name: 'min-agent', framework: undefined });
      interceptSingleAgent(agent);

      agentDetailsPage.visit('min-agent');
      agentDetailsPage.findFramework().should('not.exist');
    });
  });

  describe('Not found state', () => {
    it('should show agent-not-found when API returns 404', () => {
      initAgentsCatalogIntercepts();
      cy.intercept(
        {
          method: 'GET',
          url: new RegExp(`/model-registry/api/.*/agent_catalog/agents/.*`),
        },
        { statusCode: 404 },
      ).as('get404');

      agentDetailsPage.visit('nonexistent-agent');
      agentDetailsPage.findAgentNotFound().should('be.visible');
    });

    it('should have return button to catalog in not found state', () => {
      initAgentsCatalogIntercepts();
      cy.intercept(
        {
          method: 'GET',
          url: new RegExp(`/model-registry/api/.*/agent_catalog/agents/.*`),
        },
        { statusCode: 404 },
      );

      agentDetailsPage.visit('missing-agent');
      agentDetailsPage.findAgentNotFound().should('exist');
      cy.contains('Return to Agents Catalog').should('exist');
    });
  });

  describe('Route redirect', () => {
    it('should redirect from :agentName to :agentName/overview', () => {
      initAgentsCatalogIntercepts();
      interceptSingleAgent(mockAgent({ name: 'redirect-test' }));
      cy.visit('/agents-catalog/redirect-test');
      cy.url().should('include', '/agents-catalog/redirect-test/overview');
    });
  });
});

describe('Agents Catalog Gallery', () => {
  it('should render agent cards when agents are available', () => {
    initAgentsCatalogIntercepts({ agentsPerCategory: 3 });
    agentsCatalog.visit();
    agentsCatalog.findCards().should('have.length.greaterThan', 0);
  });

  it('should display agent name and description in cards', () => {
    initAgentsCatalogIntercepts({ agentsPerCategory: 2 });
    agentsCatalog.visit();
    agentsCatalog.findCardDetailLink('agent_templates-agent-1').should('be.visible');
    agentsCatalog.findCardDescription('agent_templates-agent-1').should('be.visible');
  });

  it('should navigate to agent details when clicking card link', () => {
    initAgentsCatalogIntercepts({ agentsPerCategory: 1 });
    agentsCatalog.visit();
    agentsCatalog.findCardDetailLink('agent_templates-agent-1').click();
    cy.url().should('include', '/agents-catalog/agent_templates-agent-1/overview');
    cy.findByTestId('app-page-title').should('exist');
  });

  it('should display labels on agent cards with mapped display names', () => {
    // mockAgent defaults: framework='langgraph', labels=['Web search','General purpose']
    // Card renders cardLabels = [frameworkDisplayName, ...labels] = 3 labels total
    // 'langgraph' maps to 'LangGraph' via AGENT_FRAMEWORK_LABEL_MAPPING
    initAgentsCatalogIntercepts({ agentsPerCategory: 1 });
    agentsCatalog.visit();

    const agentId = 'agent_templates-agent-1';
    agentsCatalog.findCardLabels(agentId).should('have.length', 3);
    agentsCatalog.findCardLabels(agentId).eq(0).should('contain.text', 'LangGraph');
    agentsCatalog.findCardLabels(agentId).eq(1).should('contain.text', 'Web search');
    agentsCatalog.findCardLabels(agentId).eq(2).should('contain.text', 'General purpose');
  });
});

describe('Agents Catalog Empty States', () => {
  it('should show empty state when no sources configured', () => {
    initAgentsCatalogIntercepts({ sources: [], agentsPerCategory: 0 });
    agentsCatalog.visit();
    agentsCatalog.findEmptyState().should('be.visible');
  });
});
