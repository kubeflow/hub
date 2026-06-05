import { appChrome } from '~/__tests__/cypress/cypress/pages/appChrome';
import { TableRow } from './components/table';
import { Modal } from './components/Modal';

class LabelModal extends Modal {
  constructor() {
    super('Labels');
  }

  findModalSearchInput() {
    return cy.findByTestId('label-modal-search');
  }

  findCloseModal() {
    return cy.findByTestId('close-modal');
  }

  shouldContainsModalLabels(labels: string[]) {
    cy.findByTestId('modal-label-group').within(() => labels.map((label) => cy.contains(label)));
    return this;
  }
}

class ModelRegistryTableRow extends TableRow {
  findName() {
    return this.find().findByTestId('model-name');
  }

  findDescription() {
    return this.find().findByTestId('description');
  }

  findLatestVersion() {
    return this.find().findByTestId('latest-version');
  }

  findOwner() {
    return this.find().findByTestId('registered-model-owner');
  }

  findLabelPopoverText() {
    return this.find().findByTestId('popover-label-text');
  }

  findLabelModalText() {
    return this.find().findByTestId('modal-label-text');
  }

  shouldContainsPopoverLabels(labels: string[]) {
    cy.findByTestId('popover-label-group').within(() => labels.map((label) => cy.contains(label)));
    return this;
  }

  findModelVersionName() {
    return this.find().findByTestId('model-version-name');
  }
}

class ModelRegistry {
  landingPage() {
    cy.visit('/');
    this.waitLanding();
  }

  visit() {
    cy.visit(`/model-registry`);
    this.waitForPageLoaded();
  }

  navigate() {
    appChrome.findNavItem('Model Registry').click();
    this.waitForPageLoaded();
  }

  waitForPageLoaded() {
    appChrome.waitForA11y();
    cy.findByTestId('app-page-title').contains('Model Registry');
  }

  private waitLanding() {
    cy.findByTestId('home-page').should('be.visible');
    cy.testA11y();
  }

  shouldBeEmpty() {
    cy.findByTestId('empty-state-title').should('exist');
    return this;
  }

  findModelRegistryEmptyState() {
    return cy.findByTestId('empty-model-registries-state');
  }

  findUnavailableModelRegistryState() {
    return cy.findByTestId('unavailable-model-registry');
  }

  shouldregisteredModelsEmpty() {
    cy.findByTestId('empty-registered-models').should('exist');
  }

  findEmptyStateTransferJobsButton() {
    return cy.findByTestId('empty-model-registry-transfer-jobs-action');
  }

  findViewDetailsButton() {
    return cy.findByTestId('view-details-button');
  }

  findDetailsPopover() {
    return cy.findByTestId('mr-details-popover');
  }

  findHelpContentButton() {
    return cy.findByTestId('model-registry-help-button');
  }

  findHelpContentPopover() {
    return cy.findByTestId('model-registry-help-content');
  }

  shouldModelVersionsEmpty() {
    cy.findByTestId('empty-model-versions').should('exist');
  }

  shouldArchiveModelVersionsEmpty() {
    cy.findByTestId('empty-archive-model-versions').should('exist');
  }

  shouldModelRegistrySelectorExist() {
    cy.findByTestId('model-registry-selector-dropdown').should('exist');
  }

  shouldtableToolbarExist() {
    cy.findByTestId('registered-models-table-toolbar').should('exist');
  }

  tabEnabled() {
    appChrome.findNavItem('Model Registry').should('exist');
    return this;
  }

  tabDisabled() {
    appChrome.findNavItem('Model Registry').should('not.exist');
    return this;
  }

  findTable() {
    return cy.findByTestId('registered-model-table');
  }

  findModelVersionsTable() {
    return cy.findByTestId('model-versions-table');
  }

  findTableRows() {
    return this.findTable().find('tbody tr');
  }

  findModelVersionsTableRows() {
    return this.findModelVersionsTable().find('tbody tr');
  }

  getRow(name: string) {
    return new ModelRegistryTableRow(() =>
      this.findTable().find(`[data-label="Model name"]`).contains(name).parents('tr'),
    );
  }

  getModelVersionRow(name: string) {
    return new ModelRegistryTableRow(() =>
      this.findModelVersionsTable()
        .find(`[data-label="Version name"]`)
        .contains(name)
        .parents('tr'),
    );
  }

  findRegisteredModelTableHeaderButton(name: string) {
    return this.findTable().find('thead').findByRole('button', { name });
  }

  findModelRegistry() {
    return cy.findByTestId('model-registry-selector-dropdown');
  }

  findModelVersionsTableHeaderButton(name: string) {
    return this.findModelVersionsTable().find('thead').findByRole('button', { name });
  }

  findTableSearch() {
    return cy.get('[data-testid$="-toolbar"]').find('[data-testid$="-input"]').filter(':visible');
  }

  findFilterDropdownItem(name: string) {
    return cy
      .get('[data-testid$="-toolbar"]')
      .find('[data-testid$="-dropdown"]')
      .then(($el) => {
        if ($el.attr('aria-expanded') === 'false') {
          cy.wrap($el).click();
        }
        return cy.get('body').findByRole('option', { name });
      });
  }

  findModelVersionsTableToolbar() {
    return cy.findByTestId('model-versions-table-toolbar');
  }

  findModelBreadcrumbItem() {
    return cy.findByTestId('breadcrumb-model');
  }

  findModelVersionsTableKebab() {
    return cy.findByTestId('model-versions-table-kebab-action');
  }

  findModelVersionsHeaderAction() {
    return cy.findByTestId('model-version-action-toggle');
  }

  findModelVersionsTableFilterOption(name: string) {
    return cy.findByTestId('model-versions-table-dropdown').then(($el) => {
      if ($el.attr('aria-expanded') === 'false') {
        cy.wrap($el).click();
      }
      return cy.findByRole('option', { name });
    });
  }

  findRegisterModelButton() {
    return cy.findByRole('button', { name: 'Register model' });
  }
}

export const modelRegistry = new ModelRegistry();
export const labelModal = new LabelModal();
