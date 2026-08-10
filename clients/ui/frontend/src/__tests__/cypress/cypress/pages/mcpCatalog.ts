import { appChrome } from './appChrome';

class McpCatalog {
  visit() {
    cy.visit('/mcp-catalog');
    this.wait();
  }

  private wait() {
    cy.contains('MCP Catalog').should('exist');
    cy.testA11y();
  }

  tabEnabled() {
    appChrome.findNavItem('MCP Catalog').should('exist');
    return this;
  }

  findPageTitle() {
    return cy.contains('MCP Catalog');
  }

  findPageDescription() {
    return cy.contains(
      'Browse and deploy MCP servers provided by Red Hat partners and other providers.',
    );
  }

  findFilter(filterKey: string) {
    return cy.findByTestId(`mcp-filter-${filterKey}`);
  }

  findFilterShowMore(filterKey: string) {
    return cy.findByTestId(`mcp-filter-${filterKey}-show-more`);
  }

  findFilterShowLess(filterKey: string) {
    return cy.findByTestId(`mcp-filter-${filterKey}-show-less`);
  }

  findFilterCheckbox(filterKey: string, value: string) {
    return cy.findByTestId(`mcp-filter-${filterKey}-${value}`);
  }

  findMcpCatalogCards() {
    return cy.get('[data-testid^="mcp-catalog-card-"]');
  }

  findMcpCatalogCard(serverId: string) {
    return cy.findByTestId(`mcp-catalog-card-${serverId}`);
  }

  findCardDetailsLink(serverId: string) {
    return cy.findByTestId(`mcp-catalog-card-detail-link-${serverId}`);
  }

  findCardDescription(serverId: string) {
    return cy.findByTestId(`mcp-catalog-card-description-${serverId}`);
  }

  findResetFilters() {
    return cy.findByTestId('mcp-catalog-reset-filters');
  }

  findLoadError() {
    return cy.findByTestId('mcp-catalog-load-error');
  }

  findRetry() {
    return cy.findByTestId('mcp-catalog-retry');
  }

  findMcpCategorySection() {
    return cy.findByTestId('mcp-category-title-organization_mcp_servers');
  }

  findEmptyState() {
    return cy.findByTestId('empty-mcp-catalog-state');
  }
}

class McpServerDetails {
  visit(serverId: string) {
    cy.visit(`/mcp-catalog/${serverId}`);
    this.wait();
  }

  private wait() {
    cy.findByTestId('app-page-title').should('exist');
    cy.testA11y();
  }

  findPageTitle() {
    return cy.findByTestId('app-page-title');
  }

  findBreadcrumbCatalogLink() {
    return cy.get('.pf-v6-c-breadcrumb').contains('MCP Catalog');
  }

  findBreadcrumbServerName() {
    return cy.findByTestId('breadcrumb-server-name');
  }

  findDeployButton() {
    return cy.findByTestId('deploy-mcp-server-button');
  }

  findDescription() {
    return cy.findByTestId('mcp-server-description');
  }

  findReadmeMarkdown() {
    return cy.findByTestId('mcp-server-readme-markdown');
  }

  findNoReadme() {
    return cy.findByTestId('mcp-server-no-readme');
  }

  findServerJsonCard() {
    return cy.findByTestId('mcp-server-json-card');
  }

  findServerJsonCode() {
    return cy.findByTestId('mcp-server-json-code');
  }

  findVersion() {
    return cy.findByTestId('mcp-server-version');
  }

  findDeploymentMode() {
    return cy.findByTestId('mcp-server-deployment-mode');
  }

  findRemoteTitleLabel() {
    return cy.findByTestId('mcp-server-details-remote-label');
  }

  findEndpointCopy() {
    return cy.findByTestId('mcp-server-endpoint-copy');
  }

  findTransportType() {
    return cy.findByTestId('mcp-server-transport-type');
  }

  findProvider() {
    return cy.findByTestId('mcp-server-provider');
  }

  findLicense() {
    return cy.findByTestId('mcp-server-license');
  }

  findLicenseLink() {
    return cy.findByTestId('mcp-server-license-link');
  }

  findLabels() {
    return cy.get('[data-testid="mcp-server-detail-label"]');
  }

  findArtifactCopy() {
    return cy.get('[data-testid="mcp-server-artifact-copy"]');
  }

  findSourceCodeLink() {
    return cy.findByTestId('mcp-server-source-code-link');
  }

  findToolsSection() {
    return cy.findByTestId('mcp-server-tools');
  }

  findToolsError() {
    return cy.findByTestId('mcp-server-tools-error');
  }

  findToolsLoading() {
    return cy.findByTestId('mcp-server-tools-loading');
  }

  findToolsFilter() {
    return cy.findByTestId('mcp-tools-filter');
  }

  findToolsEmptyFilter() {
    return cy.findByTestId('mcp-tools-empty-filter');
  }

  findToolToggle(toolName: string) {
    return cy.findByTestId(`mcp-tool-toggle-${toolName}`);
  }

  findToolsPageIndicator() {
    return cy.findByTestId('mcp-tools-page-indicator');
  }

  findToolsPagePrev() {
    return cy.findByTestId('mcp-tools-page-prev');
  }

  findToolsPageNext() {
    return cy.findByTestId('mcp-tools-page-next');
  }

  findToolRevokedLabel(toolName: string) {
    return cy.findByTestId(`mcp-tool-revoked-${toolName}`);
  }

  findToolRevokedReason(toolName: string) {
    return cy.findByTestId(`mcp-tool-revoked-reason-${toolName}`);
  }

  findMcpNotFound() {
    return cy.findByTestId('mcp-server-not-found', { timeout: 15000 });
  }
}

export const mcpCatalog = new McpCatalog();
export const mcpServerDetails = new McpServerDetails();
