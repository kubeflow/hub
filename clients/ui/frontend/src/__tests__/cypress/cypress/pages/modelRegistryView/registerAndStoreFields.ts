import { appChrome } from '~/__tests__/cypress/cypress/pages/appChrome';
import { FormFieldSelector } from './registerModelPage';

class RegisterAndStoreFields {
  visit(registryNamespace?: string) {
    const preferredModelRegistry = 'modelregistry-sample';
    const query = registryNamespace ? `?namespace=${encodeURIComponent(registryNamespace)}` : '';
    cy.visit(`/model-registry/${preferredModelRegistry}/register/model${query}`);
    appChrome.waitForA11y();
  }

  findNamespaceFormGroup() {
    return cy.findByTestId('namespace-form-group');
  }

  findNamespaceSelector() {
    return cy.findByTestId('form-namespace-selector');
  }

  findNamespaceTextInput() {
    return cy.findByTestId('form-namespace-text-input');
  }

  findNamespaceSelectTrigger() {
    return cy.findByTestId('form-namespace-selector-trigger');
  }

  findNamespaceSelectCombobox() {
    return this.findNamespaceSelector();
  }

  findOriginLocationSection() {
    return cy.findByTestId('model-origin-location-section');
  }

  findDestinationLocationSection() {
    return cy.findByTestId('model-destination-location-section');
  }

  findRegistrationModeToggleGroup() {
    return cy.findByTestId('registration-mode-toggle-group');
  }

  findRegisterToggleButton() {
    return cy.findByTestId('registration-mode-register');
  }

  findRegisterAndStoreToggleButton() {
    return cy.findByTestId('registration-mode-register-and-store');
  }

  selectNamespace(name: string) {
    this.findNamespaceSelectCombobox().should('not.be.disabled');
    this.findNamespaceSelectCombobox().scrollIntoView().click();
    cy.findByRole('listbox').should('be.visible');
    cy.findByRole('option', { name }).click();
  }

  selectRegisterMode() {
    this.findRegisterToggleButton().click();
  }

  selectRegisterAndStoreMode() {
    this.findRegisterAndStoreToggleButton().click();
  }

  shouldShowPlaceholder(placeholder = 'Select a namespace') {
    this.findNamespaceSelector().findByText(placeholder).should('contain.text', placeholder);
    return this;
  }

  shouldHaveNamespaceOptions(namespaces: string[]) {
    this.findNamespaceSelectCombobox().should('not.be.disabled');
    this.findNamespaceSelectCombobox().scrollIntoView().click();
    namespaces.forEach((namespace) => {
      cy.findByRole('option', { name: namespace }).should('exist');
    });
    this.findNamespaceSelectCombobox().scrollIntoView().click();
    return this;
  }

  shouldShowSelectedNamespace(name: string) {
    this.findNamespaceSelector().findByText(name).should('have.text', name);
    return this;
  }

  shouldHideOriginLocationSection() {
    this.findOriginLocationSection().should('not.exist');
    return this;
  }

  shouldHideDestinationLocationSection() {
    this.findDestinationLocationSection().should('not.exist');
    return this;
  }

  shouldShowOriginLocationSection() {
    this.findOriginLocationSection().should('exist');
    return this;
  }

  shouldShowDestinationLocationSection() {
    this.findDestinationLocationSection().should('exist');
    return this;
  }

  shouldHaveRegistrationModeToggle() {
    this.findRegistrationModeToggleGroup().should('exist');
    return this;
  }

  shouldHaveRegisterModeSelected() {
    this.findRegisterToggleButton().find('button').should('have.attr', 'aria-pressed', 'true');
    return this;
  }

  shouldHaveRegisterAndStoreModeSelected() {
    this.findRegisterAndStoreToggleButton()
      .find('button')
      .should('have.attr', 'aria-pressed', 'true');
    return this;
  }

  findNamespaceRegistryAccessAlert() {
    return cy.findByTestId('namespace-registry-access-alert');
  }

  findNamespaceLoadError() {
    return cy.findByTestId('namespace-load-error');
  }

  shouldShowNamespaceLoadError() {
    this.findNamespaceLoadError()
      .should('be.visible')
      .and('contain.text', 'Failed to load namespaces');
    return this;
  }

  findNamespaceAccessCheckError() {
    return cy.findByTestId('namespace-registry-access-error');
  }

  shouldShowNamespaceAccessCheckError() {
    this.findNamespaceAccessCheckError()
      .should('be.visible')
      .and('contain.text', 'Could not verify namespace access');
    return this;
  }

  findNamespaceCannotCheckAlert() {
    return cy.findByTestId('namespace-registry-cannot-check-alert');
  }

  shouldShowCannotCheckAlert(registryName: string) {
    this.findNamespaceCannotCheckAlert()
      .should('be.visible')
      .and('contain.text', 'Cannot check registry access with your permissions')
      .and('contain.text', registryName);
    return this;
  }

  shouldShowNamespaceLabel() {
    this.findNamespaceFormGroup().find('label').should('contain.text', 'Namespace');
    return this;
  }

  shouldBeNamespaceSelectorDisabled() {
    this.findNamespaceSelectTrigger()
      .find('button')
      .should(
        ($btn) =>
          $btn.prop('disabled') === true ||
          $btn.attr('aria-disabled') === 'true' ||
          $btn.hasClass('pf-m-disabled'),
      );
    return this;
  }

  shouldShowNoAccessWarning() {
    this.findNamespaceRegistryAccessAlert()
      .should('be.visible')
      .and('contain.text', 'The selected namespace does not have access to this model registry');
    return this;
  }

  shouldShowNoNamespacesMessage() {
    this.findNamespaceRegistryAccessAlert()
      .should('be.visible')
      .and('contain.text', 'You do not have access to any namespaces');
    return this;
  }

  findWhoIsMyAdminTrigger() {
    return cy.findByTestId('who-is-my-admin-trigger');
  }

  openWhoIsMyAdminPopover() {
    this.findWhoIsMyAdminTrigger().click();
    return this;
  }

  shouldShowWhoIsMyAdminPopoverWithNamespaceWording() {
    cy.contains('Your administrator might be').should('be.visible');
    cy.contains('namespaces that you have permission to access').should('be.visible');
    return this;
  }

  findCreateButton() {
    return cy.findByTestId('create-button');
  }

  shouldHaveCreateButtonDisabled() {
    this.findCreateButton().should('be.disabled');
    return this;
  }

  // Destination field finders (using id selectors since these inputs use id, not data-testid)
  findDestinationOciRegistryInput() {
    return cy.get('#destination-oci-registry');
  }

  findDestinationOciUriInput() {
    return cy.get('#destination-oci-uri');
  }

  findDestinationOciUsernameInput() {
    return cy.get('#destination-oci-username');
  }

  findDestinationOciPasswordInput() {
    return cy.get('#destination-oci-password');
  }

  // Source credential field finders
  findSourceS3AccessKeyIdInput() {
    return cy.get('#location-s3-access-key-id');
  }

  findSourceS3SecretAccessKeyInput() {
    return cy.get('#location-s3-secret-access-key');
  }

  // Submit button
  findSubmitButton() {
    return cy.findByTestId('create-button');
  }

  // Fill form helper methods
  fillModelName(name: string) {
    cy.get(FormFieldSelector.MODEL_NAME).clear();
    cy.get(FormFieldSelector.MODEL_NAME).type(name);
  }

  fillVersionName(name: string) {
    cy.get(FormFieldSelector.VERSION_NAME).clear();
    cy.get(FormFieldSelector.VERSION_NAME).type(name);
  }

  fillJobName(name: string) {
    cy.get(FormFieldSelector.JOB_NAME).clear();
    cy.get(FormFieldSelector.JOB_NAME).type(name);
  }

  fillSourceEndpoint(endpoint: string) {
    cy.get(FormFieldSelector.LOCATION_ENDPOINT).clear();
    cy.get(FormFieldSelector.LOCATION_ENDPOINT).type(endpoint);
  }

  fillSourceBucket(bucket: string) {
    cy.get(FormFieldSelector.LOCATION_BUCKET).clear();
    cy.get(FormFieldSelector.LOCATION_BUCKET).type(bucket);
  }

  fillSourcePath(path: string) {
    cy.get(FormFieldSelector.LOCATION_PATH).clear();
    cy.get(FormFieldSelector.LOCATION_PATH).type(path);
  }

  fillDestinationOciRegistry(registry: string) {
    this.findDestinationOciRegistryInput().clear();
    this.findDestinationOciRegistryInput().type(registry);
  }

  fillDestinationOciUri(uri: string) {
    this.findDestinationOciUriInput().clear();
    this.findDestinationOciUriInput().type(uri);
  }

  fillDestinationOciUsername(username: string) {
    this.findDestinationOciUsernameInput().clear();
    this.findDestinationOciUsernameInput().type(username);
  }

  fillDestinationOciPassword(password: string) {
    this.findDestinationOciPasswordInput().clear();
    this.findDestinationOciPasswordInput().type(password);
  }

  fillSourceS3AccessKeyId(accessKeyId: string) {
    this.findSourceS3AccessKeyIdInput().clear();
    this.findSourceS3AccessKeyIdInput().type(accessKeyId);
  }

  fillSourceS3SecretAccessKey(secretAccessKey: string) {
    this.findSourceS3SecretAccessKeyInput().clear();
    this.findSourceS3SecretAccessKeyInput().type(secretAccessKey);
  }

  /** Sets model type (required on register page). Uses Predictive by default. */
  selectModelType(
    optionName: 'Predictive Model' | 'Generative AI model (Example, LLM)' = 'Predictive Model',
  ) {
    cy.get('#register-model-type-toggle').click();
    cy.findByRole('option', { name: optionName }).click();
  }

  // Convenience method to fill all required fields for submission
  fillAllRequiredFields() {
    this.fillModelName('test-model');
    this.selectModelType();
    this.fillVersionName('v1.0.0');
    this.fillJobName('my-transfer-job');
    this.fillSourceEndpoint('https://s3.amazonaws.com');
    this.fillSourceBucket('test-bucket');
    this.fillSourcePath('models/test');
    this.fillSourceS3AccessKeyId('AKIAIOSFODNN7EXAMPLE');
    this.fillSourceS3SecretAccessKey('wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY');
    this.fillDestinationOciRegistry('quay.io');
    this.fillDestinationOciUri('quay.io/my-org/my-model:v1');
    this.fillDestinationOciUsername('testuser');
    this.fillDestinationOciPassword('testpassword123');
  }
}

export const registerAndStoreFields = new RegisterAndStoreFields();
