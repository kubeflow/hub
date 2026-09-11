import { appChrome } from './appChrome';
import { TableRow } from './components/table';
import { Modal } from './components/Modal';

class DeleteSourceModal extends Modal {
  constructor() {
    super('Delete a source');
  }

  find() {
    return cy.findByTestId('delete-source-modal');
  }

  findDeleteButton() {
    return this.findFooter().findByRole('button', { name: 'Delete' });
  }

  findConfirmInput() {
    return cy.findByTestId('delete-modal-input');
  }

  typeConfirmation(text: string) {
    this.findConfirmInput().clear().type(text);
    return this;
  }
}

class CatalogSourceConfigRow extends TableRow {
  findName() {
    return this.find().find('[data-label="Name"]');
  }

  findOrganization() {
    return this.find().find('[data-label="Organization"]');
  }

  findModelVisibility() {
    return this.find().find('[data-label="Model visibility"]');
  }

  findSourceType() {
    return this.find().find('[data-label="Source type"]');
  }

  findEnableToggle() {
    return this.find().find('[data-label="Enable"]').find('input[type="checkbox"]');
  }

  findValidationStatus() {
    return this.find().find('[data-label="Validation status"]');
  }

  findManageSourceButton() {
    return this.find()
      .find('[data-label="Actions"]')
      .findByRole('button', { name: 'Manage source' });
  }

  shouldHaveModelVisibility(visibility: 'Filtered' | 'All models') {
    this.findModelVisibility().contains(visibility);
    return this;
  }

  shouldHaveOrganization(org: string) {
    this.findOrganization().contains(org);
    return this;
  }

  shouldHaveSourceType(type: string) {
    this.findSourceType().contains(type);
    return this;
  }

  toggleEnable() {
    this.findEnableToggle().click({ force: true });
    return this;
  }

  shouldHaveEnableState(enabled: boolean) {
    if (enabled) {
      this.findEnableToggle().should('be.checked');
    } else {
      this.findEnableToggle().should('not.be.checked');
    }
    return this;
  }

  shouldHaveKebab(shouldExist: boolean) {
    if (shouldExist) {
      this.findKebab().should('exist');
    } else {
      this.find().within(() => {
        cy.get('[data-testid*="source-actions"]').should('not.exist');
      });
    }
    return this;
  }

  shouldHaveValidationStatus(status: 'Ready' | 'Failed' | 'Starting' | 'Unknown' | '-') {
    this.findValidationStatus().contains(status);
    return this;
  }

  findValidationStatusErrorLink() {
    return this.findValidationStatus().find('[data-testid*="source-status-error-link"]');
  }

  clickValidationStatusErrorLink() {
    this.findValidationStatusErrorLink().click();
    return this;
  }

  findValidationStatusIndicator(testId: string) {
    return this.findValidationStatus().findByTestId(testId);
  }

  findDeleteSourceMenuItem() {
    return cy.findByRole('menuitem', { name: 'Delete source' });
  }

  clickDeleteSourceMenuItem() {
    this.findDeleteSourceMenuItem().click();
    return this;
  }
}

class CatalogSourceStatusErrorModal extends Modal {
  constructor() {
    super('Source status');
  }

  find() {
    return cy.findByTestId('catalog-source-status-error-modal');
  }

  findAlert() {
    return cy.findByTestId('catalog-source-status-error-alert');
  }

  findMessage() {
    return cy.findByTestId('catalog-source-status-error-message');
  }

  findCloseButton() {
    return this.find().findByRole('button', { name: 'Close' });
  }

  clickClose() {
    this.findCloseButton().click();
    return this;
  }
}

class ModelCatalogSettings {
  visit({ wait = true }: { wait?: boolean } = {}) {
    cy.visit('/model-catalog-settings');
    if (wait) {
      this.wait();
    }
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
    cy.findByTestId('app-page-title').contains('Model catalog settings');
  }

  findNavItem() {
    return appChrome.findNavItem('Model catalog settings', 'Settings');
  }

  findDescription() {
    return cy.contains(
      'Add and manage model sources that populate the model catalog for users in your organization.',
    );
  }

  findAddSourceButton() {
    return cy.findByTestId('add-source-button');
  }

  findToggleAlert() {
    return cy.findByTestId('toggle-alert');
  }

  findTable() {
    return cy.findByTestId('catalog-source-configs-table');
  }

  findEmptyState() {
    return cy.findByTestId('catalog-settings-empty-state');
  }

  getRow(name: string) {
    return new CatalogSourceConfigRow(() =>
      this.findTable().find('tbody').find('tr').contains(name).parents('tr'),
    );
  }

  findRows() {
    return this.findTable().find('tbody tr');
  }

  shouldHaveSourceConfigs() {
    this.findTable().should('exist');
    this.findRows().should('have.length.at.least', 1);
    return this;
  }

  shouldBeEmpty() {
    this.findEmptyState().should('exist');
    return this;
  }

  findSourceStatusErrorAlert() {
    return cy.findByTestId('source-status-error-alert');
  }

  shouldHaveSourceStatusErrorAlert() {
    this.findSourceStatusErrorAlert().should('exist');
    return this;
  }
}

class ManageSourcePage {
  visitAddSource({ wait = true }: { wait?: boolean } = {}) {
    cy.visit('/model-catalog-settings/add-source');
    if (wait) {
      this.wait();
    }
  }

  visitManageSource(catalogSourceId: string, { wait = true }: { wait?: boolean } = {}) {
    cy.visit(`/model-catalog-settings/manage-source/${encodeURIComponent(catalogSourceId)}`);
    if (wait) {
      this.wait();
    }
  }

  private wait() {
    this.findHeading();
    cy.testA11y();
  }

  findHeading() {
    cy.findByTestId('app-page-title').should('exist');
  }

  findBreadcrumb() {
    return cy.get('a[href="/model-catalog-settings"]').contains('Model catalog settings');
  }

  findBreadcrumbAction() {
    return cy.findByTestId('breadcrumb-source-action');
  }

  findAddSourceTitle() {
    return cy.findByTestId('app-page-title').contains('Add a source');
  }

  findManageSourceTitle() {
    return cy.findByTestId('app-page-title').contains('Manage source');
  }

  findAddSourceDescription() {
    return cy.contains('Add a new model catalog source to your organization.');
  }

  findManageSourceDescription() {
    return cy.contains('Manage the selected model catalog source.');
  }

  // Form field methods
  findNameInput() {
    return cy.findByTestId('source-name-input');
  }

  findNameError() {
    return cy.findByTestId('source-name-error');
  }

  findSourceTypeHuggingFace() {
    return cy.findByTestId('source-type-huggingface');
  }

  findSourceTypeYaml() {
    return cy.findByTestId('source-type-yaml');
  }

  findSourceTypeHuggingFaceLabel() {
    return cy.get('label[for="source-type-huggingface"]');
  }

  findSourceTypeYamlLabel() {
    return cy.get('label[for="source-type-yaml"]');
  }

  findCredentialsSection() {
    return cy.findByTestId('credentials-section');
  }

  findAccessTokenInput() {
    return cy.findByTestId('access-token-input');
  }

  findOrganizationInput() {
    return cy.findByTestId('organization-input');
  }

  findOrganizationError() {
    return cy.findByTestId('organization-error');
  }

  findYamlSection() {
    return cy.findByTestId('yaml-section');
  }

  findYamlContentInput() {
    return cy.findByTestId('yaml-content-input').find('textarea');
  }

  findYamlContentError() {
    return cy.findByTestId('yaml-content-error');
  }

  findModelVisibilitySection() {
    return cy.findByTestId('model-visibility-section');
  }

  toggleModelVisibility() {
    this.findModelVisibilitySection().find('button').first().click();
  }

  findAllowedModelsInput() {
    return cy.findByTestId('allowed-models-input');
  }

  findExcludedModelsInput() {
    return cy.findByTestId('excluded-models-input');
  }

  findEnableSourceCheckbox() {
    return cy.findByTestId('enable-source-checkbox');
  }

  findSubmitButton() {
    return cy.findByTestId('submit-button');
  }

  findPreviewButton() {
    return cy.findByTestId('preview-button');
  }

  findCancelButton() {
    return cy.findByTestId('cancel-button');
  }

  fillSourceName(name: string) {
    this.findNameInput().clear().type(name);
  }

  selectSourceType(type: 'huggingface' | 'yaml') {
    if (type === 'huggingface') {
      this.findSourceTypeHuggingFaceLabel().click();
    } else {
      this.findSourceTypeYamlLabel().click();
    }
  }

  fillAccessToken(token: string) {
    this.findAccessTokenInput().clear().type(token);
  }

  fillOrganization(org: string) {
    this.findOrganizationInput().clear().type(org);
  }

  fillYamlContent(yaml: string) {
    this.findYamlContentInput().clear().type(yaml);
  }

  fillAllowedModels(models: string) {
    this.findAllowedModelsInput().clear().type(models);
  }

  fillExcludedModels(models: string) {
    this.findExcludedModelsInput().clear().type(models);
  }

  toggleEnableSource() {
    this.findEnableSourceCheckbox().click();
  }

  findPreviewPanel() {
    return cy.findByTestId('preview-panel');
  }

  findPreviewPanelTitle() {
    return cy.contains('Model catalog preview');
  }

  findPreviewPanelEmptyMessage() {
    return cy.contains('To view the models from this source that will appear');
  }

  findPreviewPanelHeaderButton() {
    return this.findPreviewPanel().findByTestId('preview-button-header');
  }

  findPreviewPanelBodyButton() {
    return this.findPreviewPanel().findByTestId('preview-button-panel');
  }

  findAccessTokenHiddenHelper() {
    return cy.findByTestId('access-token-hidden-helper');
  }

  findClearAccessTokenModal() {
    return cy.findByTestId('clear-access-token-modal');
  }

  findClearAccessTokenConfirmButton() {
    return cy.findByTestId('clear-access-token-confirm-button');
  }

  findRefreshPreviewAlert() {
    return cy.contains('Source configuration changed. Refresh the preview.');
  }

  findRefreshPreviewLink() {
    return cy.findByTestId('refresh-preview-link');
  }

  findValidateButton() {
    return cy.findByRole('button', { name: 'Validate' });
  }

  clickValidate() {
    this.findValidateButton().click();
    return this;
  }

  findShowAccessTokenButton() {
    return cy.findByRole('button', { name: 'Show access token' });
  }

  findHideAccessTokenButton() {
    return cy.findByRole('button', { name: 'Hide access token' });
  }

  clickShowAccessToken() {
    this.findShowAccessTokenButton().click();
    return this;
  }

  clickHideAccessToken() {
    this.findHideAccessTokenButton().click();
    return this;
  }

  findClearAccessTokenButton() {
    return cy.findByRole('button', { name: 'Clear' });
  }

  clickClearAccessToken() {
    this.findClearAccessTokenButton().click();
    return this;
  }

  findClearAccessTokenModalCancelButton() {
    return this.findClearAccessTokenModal().findByRole('button', { name: 'Cancel' });
  }

  clickClearAccessTokenModalCancel() {
    this.findClearAccessTokenModalCancelButton().click();
    return this;
  }

  findValidationSuccessAlert() {
    return cy.contains('Credentials validated');
  }

  findValidationFailedAlert() {
    return cy.contains('Credentials validation failed');
  }

  findPreviewModelsIncludedSummary(count: number, total: number) {
    return cy.contains(`${count} of ${total} models included:`);
  }
}

export const modelCatalogSettings = new ModelCatalogSettings();
export const manageSourcePage = new ManageSourcePage();
export const deleteSourceModal = new DeleteSourceModal();
export const catalogSourceStatusErrorModal = new CatalogSourceStatusErrorModal();
