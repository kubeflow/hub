import { appChrome } from './appChrome';
import { TableRow } from './components/table';
import { Modal } from './components/Modal';

class McpDeleteSourceModal extends Modal {
  constructor() {
    super('Delete a source');
  }

  find() {
    return cy.findByTestId('mcp-delete-source-modal');
  }

  findDeleteButton() {
    return cy.findByTestId('delete-modal-confirm-button');
  }

  findConfirmInput() {
    return cy.findByTestId('delete-modal-input');
  }

  typeConfirmation(text: string) {
    this.findConfirmInput().clear().type(text);
    return this;
  }
}

class McpCatalogSourceConfigRow extends TableRow {
  findName() {
    return this.find().find('[data-label="Name"]');
  }

  findEnableToggle() {
    return this.find().find('[data-label="Enable"]').find('input[type="checkbox"]');
  }

  findValidationStatus() {
    return this.find().find('[data-label="Validation status"]');
  }

  toggleEnable() {
    this.findEnableToggle().click({ force: true });
    return this;
  }

  shouldHaveValidationStatus(status: 'Ready' | 'Failed' | 'Starting' | 'Unknown' | '-') {
    this.findValidationStatus().contains(status);
    return this;
  }
}

class McpCatalogSettings {
  visit() {
    cy.visit('/mcp-catalog-settings');
    this.wait();
  }

  navigate() {
    cy.get('body').then(($body) => {
      if ($body.find('#page-sidebar').length > 0) {
        this.findNavItem().click();
        this.wait();
      }
    });
  }

  private wait() {
    this.findHeading();
    cy.testA11y();
  }

  findHeading() {
    cy.findByTestId('app-page-title').should('exist');
    cy.findByTestId('app-page-title').contains('MCP catalog sources');
  }

  findNavItem() {
    return appChrome.findNavItem('MCP catalog sources', 'Settings');
  }

  findDescription() {
    return cy.contains('Add and manage MCP catalog sources');
  }

  findEmptyState() {
    return cy.findByTestId('mcp-catalog-settings-empty-state');
  }

  shouldBeEmpty() {
    this.findEmptyState().should('exist');
    return this;
  }

  findAddSourceButton() {
    return cy.findByTestId('mcp-add-source-button-empty');
  }

  findTable() {
    return cy.findByTestId('mcp-catalog-source-configs-table');
  }

  findRows() {
    return this.findTable().find('tbody tr');
  }

  shouldHaveRows() {
    this.findTable().should('exist');
    this.findRows().should('have.length.at.least', 1);
  }

  getRow(name: string) {
    return new McpCatalogSourceConfigRow(() =>
      this.findTable().find('tbody').find('tr').contains(name).parents('tr'),
    );
  }

  findToggleAlert() {
    return cy.findByTestId('mcp-toggle-alert');
  }

  findSourceStatusErrorAlert() {
    return cy.findByTestId('mcp-source-status-error-alert');
  }

  findSortButton(columnLabel: string) {
    return this.findTable().find('thead th').contains(columnLabel).find('button');
  }
}

class McpManageSourcePage {
  visitAddSource() {
    cy.visit('/mcp-catalog-settings/add-source');
    this.wait();
  }

  visitManageSource(catalogSourceId: string) {
    cy.visit(`/mcp-catalog-settings/manage-source/${encodeURIComponent(catalogSourceId)}`);
    this.wait();
  }

  private wait() {
    this.findHeading();
    cy.testA11y();
  }

  findHeading() {
    cy.findByTestId('app-page-title').should('exist');
  }

  findBreadcrumb() {
    return cy.get('a[href="/mcp-catalog-settings"]').contains('MCP catalog sources');
  }

  findBreadcrumbAction() {
    return cy.findByTestId('mcp-breadcrumb-source-action');
  }

  findAddSourceTitle() {
    return cy.findByTestId('app-page-title').contains('Add source');
  }

  findManageSourceTitle() {
    return cy.findByTestId('app-page-title').contains('Manage source');
  }

  findAddSourceDescription() {
    return cy.contains('Add a new MCP catalog source to your organization.');
  }

  findManageSourceDescription() {
    return cy.contains('Configure which servers from this source appear in the MCP catalog.');
  }

  // Form field methods
  findNameInput() {
    return cy.findByTestId('mcp-source-name-input');
  }

  findNameError() {
    return cy.findByTestId('mcp-source-name-error');
  }

  findYamlSection() {
    return cy.findByTestId('mcp-yaml-section');
  }

  findYamlContentInput() {
    return cy.findByTestId('mcp-yaml-content-input').find('textarea');
  }

  findYamlContentError() {
    return cy.findByTestId('mcp-yaml-content-error');
  }

  findServerVisibilitySection() {
    return cy.findByTestId('mcp-server-filters-section');
  }

  toggleServerVisibility() {
    this.findServerVisibilitySection().find('button').first().click();
  }

  findIncludedServersInput() {
    return cy.findByTestId('mcp-included-servers-input');
  }

  findExcludedServersInput() {
    return cy.findByTestId('mcp-excluded-servers-input');
  }

  findEnableSourceCheckbox() {
    return cy.findByTestId('mcp-enable-source-checkbox');
  }

  findSubmitButton() {
    return cy.findByTestId('mcp-submit-button');
  }

  findPreviewButton() {
    return cy.findByTestId('mcp-preview-button');
  }

  findCancelButton() {
    return cy.findByTestId('mcp-cancel-button');
  }

  findPreviewPanel() {
    return cy.findByTestId('mcp-preview-panel');
  }

  findPreviewPanelTitle() {
    return cy.contains('MCP catalog preview');
  }

  findPreviewPanelEmptyMessage() {
    return cy.contains('Complete all required fields, then click');
  }

  findPreviewButtonHeader() {
    return cy.findByTestId('mcp-preview-button-header');
  }

  findPreviewButtonPanel() {
    return cy.findByTestId('mcp-preview-button-panel');
  }

  findExpectedFormatLink() {
    return cy.findByTestId('mcp-view-expected-yaml-format-link');
  }

  findExpectedFormatDrawerTitle() {
    return cy.findByTestId('mcp-expected-format-drawer-title');
  }

  findExpectedFormatDrawerClose() {
    return cy.findByTestId('mcp-expected-format-drawer-close');
  }

  findCatalogYamlFile() {
    return cy.findByTestId('mcp-catalog-yaml-file');
  }

  findServersCount() {
    return cy.findByTestId('mcp-servers-count');
  }

  // Action helpers
  fillSourceName(name: string) {
    this.findNameInput().clear().type(name);
  }

  fillYamlContent(yaml: string) {
    this.findYamlContentInput().clear().type(yaml);
  }

  fillIncludedServers(servers: string) {
    this.findIncludedServersInput().clear().type(servers);
  }

  fillExcludedServers(servers: string) {
    this.findExcludedServersInput().clear().type(servers);
  }

  toggleEnableSource() {
    this.findEnableSourceCheckbox().click();
  }
}

export const mcpCatalogSettings = new McpCatalogSettings();
export const mcpManageSourcePage = new McpManageSourcePage();
export const mcpDeleteSourceModal = new McpDeleteSourceModal();
