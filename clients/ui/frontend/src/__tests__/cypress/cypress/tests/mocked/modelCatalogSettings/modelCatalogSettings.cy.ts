import {
  modelCatalogSettings,
  manageSourcePage,
  deleteSourceModal,
} from '~/__tests__/cypress/cypress/pages/modelCatalogSettings';
import { MODEL_CATALOG_API_VERSION } from '~/__tests__/cypress/cypress/support/commands/api';
import {
  mockCatalogSourceConfigList,
  mockYamlCatalogSourceConfig,
  mockHuggingFaceCatalogSourceConfig,
  mockCatalogSource,
  mockCatalogSourceList,
} from '~/__mocks__';
import type { CatalogSource } from '~/app/shared/types/catalogTypes';
import { CatalogSourceType, type CatalogSourceConfigList } from '~/app/modelCatalogTypes';
import { EMPTY_CUSTOM_PROPERTY_VALUE } from '~/concepts/modelCatalog/const';

const NAMESPACE = 'kubeflow';
const userMock = {
  data: {
    userId: 'user@example.com',
    clusterAdmin: true,
  },
};

const setupMocks = (
  sources: CatalogSource[] = [],
  sourceConfigs: CatalogSourceConfigList = { catalogs: [] },
) => {
  cy.intercept('GET', '/model-registry/api/v1/namespaces', {
    data: [{ metadata: { name: NAMESPACE } }],
  });
  cy.intercept('GET', '/model-registry/api/v1/user', userMock);

  cy.intercept('GET', '/model-registry/api/v1/settings/model_catalog/source_configs', {
    data: sourceConfigs,
  });

  cy.interceptApi(
    `GET /api/:apiVersion/model_catalog/sources`,
    {
      path: { apiVersion: MODEL_CATALOG_API_VERSION },
    },
    mockCatalogSourceList({
      items: sources,
    }),
  );
};

function selectNamespaceIfPresent() {
  cy.get('body').then(($body) => {
    if ($body.find('[data-testid="namespace-select"]').length) {
      cy.get('[data-testid="namespace-select"]').click();
      cy.findByText(NAMESPACE).click();
    }
  });
}

describe('Model Catalog Settings', () => {
  beforeEach(() => {
    setupMocks([], mockCatalogSourceConfigList({}));
  });

  it('should display the settings page', () => {
    modelCatalogSettings.visit();
    modelCatalogSettings.findHeading();
    modelCatalogSettings.findDescription();
  });

  it('should navigate to settings page from nav', () => {
    selectNamespaceIfPresent();
    cy.get('body').then(($body) => {
      if ($body.find('#page-sidebar').length > 0) {
        modelCatalogSettings.navigate();
        modelCatalogSettings.findHeading();
      } else {
        cy.log('Sidebar not available, skipping nav test');
      }
    });
  });

  it('should display add source button', () => {
    modelCatalogSettings.visit();
    modelCatalogSettings.findAddSourceButton().should('be.visible');
    modelCatalogSettings.findAddSourceButton().should('contain', 'Add a source');
  });

  it('should navigate to add source page when button is clicked', () => {
    modelCatalogSettings.visit();
    modelCatalogSettings.findAddSourceButton().click();
    manageSourcePage.findAddSourceTitle();
    manageSourcePage.findAddSourceDescription();
    manageSourcePage.findBreadcrumb().should('exist');
    manageSourcePage.findBreadcrumbAction().should('contain', 'Add a source');
  });
});

describe('Catalog Source Configs Table', () => {
  const defaultYamlSource = mockYamlCatalogSourceConfig({
    id: 'default-yaml',
    name: 'Default Catalog',
    isDefault: true,
    enabled: true,
    includedModels: [],
    excludedModels: [],
  });

  const huggingFaceSource = mockHuggingFaceCatalogSourceConfig({
    id: 'hf-google',
    name: 'HuggingFace Google',
    isDefault: false,
    enabled: true,
    allowedOrganization: 'Google',
    includedModels: ['model1', 'model2'],
  });

  const customYamlSource = mockYamlCatalogSourceConfig({
    id: 'custom-yaml',
    name: 'Custom YAML',
    isDefault: false,
    enabled: false,
    excludedModels: ['excluded-model'],
  });
  beforeEach(() => {
    setupMocks([], { catalogs: [defaultYamlSource, huggingFaceSource, customYamlSource] });
  });

  it('should display empty state when no source configs exist', () => {
    setupMocks([], { catalogs: [] });
    modelCatalogSettings.visit();
    modelCatalogSettings.shouldBeEmpty();
    modelCatalogSettings.findEmptyState().should('contain', 'No catalog sources');
  });

  it('should display table with source configs', () => {
    modelCatalogSettings.visit();
    modelCatalogSettings.shouldHaveSourceConfigs();
    modelCatalogSettings.findRows().should('have.length', 3);
  });

  it('should render table column headers correctly', () => {
    modelCatalogSettings.visit();
    modelCatalogSettings.findTable().should('be.visible');
    modelCatalogSettings.findTable().contains('th', 'Source name').should('be.visible');
    modelCatalogSettings.findTable().contains('th', 'Organization').should('be.visible');
    modelCatalogSettings.findTable().contains('th', 'Model visibility').should('be.visible');
    modelCatalogSettings.findTable().contains('th', 'Source type').should('be.visible');
    modelCatalogSettings.findTable().contains('th', 'Enable').should('be.visible');
    modelCatalogSettings.findTable().contains('th', 'Validation status').should('be.visible');
  });

  describe('Table row rendering', () => {
    it('should render default YAML source correctly', () => {
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('Default Catalog');
      row.findName().should('be.visible').and('contain', 'Default Catalog');
      row.shouldHaveOrganization(EMPTY_CUSTOM_PROPERTY_VALUE);
      row.shouldHaveModelVisibility('All models');
      row.shouldHaveSourceType('YAML file');
    });

    it('should render Hugging Face source correctly', () => {
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('HuggingFace Google');
      row.findName().should('be.visible').and('contain', 'HuggingFace Google');
      row.shouldHaveOrganization('Google');
      row.shouldHaveModelVisibility('Filtered');
      row.shouldHaveSourceType('Hugging Face');
      row.shouldHaveEnableState(true);
    });

    it('should render custom YAML source correctly', () => {
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('Custom YAML');
      row.findName().should('be.visible').and('contain', 'Custom YAML');
      row.shouldHaveOrganization(EMPTY_CUSTOM_PROPERTY_VALUE);
      row.shouldHaveModelVisibility('Filtered');
      row.shouldHaveSourceType('YAML file');
      row.shouldHaveEnableState(false);
    });
  });

  describe('Enable toggle functionality', () => {
    it('should disable the source when toggle is clicked', () => {
      cy.intercept('PATCH', '/model-registry/api/v1/settings/model_catalog/source_configs/*', {
        statusCode: 200,
        body: {
          data: mockYamlCatalogSourceConfig({ id: 'source_2', isDefault: false }),
        },
      }).as('manageToggle');
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('HuggingFace Google');
      row.findName().should('be.visible');
      row.findEnableToggle().should('exist').and('be.checked');

      row.toggleEnable();
      cy.wait('@manageToggle').then((interception) => {
        expect(interception.request.body).to.eql({
          data: {
            enabled: false,
          },
        });
      });
    });

    it('should enable the source when toggle is clicked', () => {
      cy.intercept('PATCH', '/model-registry/api/v1/settings/model_catalog/source_configs/*', {
        statusCode: 200,
        body: {
          data: mockYamlCatalogSourceConfig({ id: 'source_2', isDefault: false }),
        },
      }).as('manageToggle');
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('Custom YAML');
      row.findName().should('be.visible');
      row.findEnableToggle().should('exist').and('not.be.checked');

      row.toggleEnable();
      cy.wait('@manageToggle').then((interception) => {
        expect(interception.request.body).to.eql({
          data: {
            enabled: true,
          },
        });
      });
    });

    it('should show error, if the patch call to toggle fails', () => {
      cy.intercept(
        'PATCH',
        '/model-registry/api/v1/settings/model_catalog/source_configs/*',
        (req) => {
          req.reply({
            statusCode: 404,
          });
        },
      ).as('manageToggle');
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('Custom YAML');
      row.findName().should('be.visible');
      row.findEnableToggle().should('exist').and('not.be.checked');

      row.toggleEnable();
      modelCatalogSettings.findToggleAlert().should('exist');
      modelCatalogSettings
        .findToggleAlert()
        .should('have.text', 'Danger alert:Error enabling/disabling source Custom YAML');
    });

    it('should disable the toggle, when the request is processing', () => {
      cy.intercept(
        'PATCH',
        '/model-registry/api/v1/settings/model_catalog/source_configs/*',
        (req) => {
          req.reply({
            statusCode: 200,
            delay: 1000,
          });
        },
      ).as('manageToggle');
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('Custom YAML');
      row.findName().should('be.visible');
      row.findEnableToggle().should('exist').and('not.be.checked');

      row.toggleEnable();
      row.findEnableToggle().should('be.disabled');
    });
  });

  describe('Manage source button', () => {
    it('should navigate to manage source page when button is clicked', () => {
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('HuggingFace Google');
      row.findName().should('be.visible');
      row.findManageSourceButton().should('be.visible').click();
      cy.url().should('include', '/model-catalog-settings/manage-source/hf-google');
      manageSourcePage.findManageSourceTitle();
    });

    it('should navigate to correct manage source page for each row', () => {
      modelCatalogSettings.visit();
      const customRow = modelCatalogSettings.getRow('Custom YAML');
      customRow.findName().should('be.visible');
      customRow.findManageSourceButton().should('be.visible').click();
      cy.url().should('include', '/model-catalog-settings/manage-source/custom-yaml');
    });
  });

  describe('Kebab menu actions', () => {
    it('should show kebab with delete action for non-default sources', () => {
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('HuggingFace Google');
      row.findName().should('be.visible');
      row.shouldHaveKebab(true);
      row.findKebab().should('be.visible').click();
      cy.findByRole('menuitem', { name: 'Delete source' })
        .should('be.visible')
        .and('not.be.disabled');
    });

    it('should not show kebab menu for default sources', () => {
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('Default Catalog');
      row.findName().should('be.visible');
      row.shouldHaveKebab(false);
    });
  });

  describe('Delete source functionality', () => {
    beforeEach(() => {
      cy.intercept(
        'DELETE',
        `/model-registry/api/${MODEL_CATALOG_API_VERSION}/settings/model_catalog/source_configs/*`,
        {
          statusCode: 200,
        },
      ).as('deleteSource');
    });

    it('should open delete modal when delete action is clicked', () => {
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('HuggingFace Google');
      row.findKebab().click();
      cy.findByRole('menuitem', { name: 'Delete source' }).click();

      deleteSourceModal.shouldBeOpen();
      deleteSourceModal.find().should('contain', 'HuggingFace Google');
      deleteSourceModal.find().should('contain', 'repository will be deleted');
    });

    it('should require typing source name to enable delete button', () => {
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('HuggingFace Google');
      row.findKebab().click();
      cy.findByRole('menuitem', { name: 'Delete source' }).click();

      deleteSourceModal.shouldBeOpen();
      deleteSourceModal.findDeleteButton().should('be.disabled');

      deleteSourceModal.typeConfirmation('wrong name');
      deleteSourceModal.findDeleteButton().should('be.disabled');

      deleteSourceModal.typeConfirmation('HuggingFace Google');
      deleteSourceModal.findDeleteButton().should('not.be.disabled');
    });

    it('should close modal when cancel is clicked', () => {
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('HuggingFace Google');
      row.findKebab().click();
      cy.findByRole('menuitem', { name: 'Delete source' }).click();

      deleteSourceModal.shouldBeOpen();
      deleteSourceModal.findCancelButton().click();
      deleteSourceModal.shouldBeOpen(false);
    });

    it('should disable delete button while deleting', () => {
      cy.intercept(
        'DELETE',
        `/model-registry/api/${MODEL_CATALOG_API_VERSION}/settings/model_catalog/source_configs/*`,
        (req) => {
          req.reply({
            statusCode: 200,
            delay: 1000,
          });
        },
      ).as('deleteSourceSlow');

      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('Custom YAML');
      row.findKebab().click();
      cy.findByRole('menuitem', { name: 'Delete source' }).click();

      deleteSourceModal.shouldBeOpen();
      deleteSourceModal.typeConfirmation('Custom YAML');
      deleteSourceModal.findDeleteButton().should('not.be.disabled').click();

      // Check that the button is disabled (it will show "Loading... Delete")
      deleteSourceModal.findFooter().find('button').first().should('be.disabled');
    });
  });

  describe('Model visibility badges', () => {
    it('should show "Filtered" badge when source has included models', () => {
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('HuggingFace Google');
      row.findName().should('be.visible');
      row.findModelVisibility().should('be.visible').and('contain', 'Filtered');
      row
        .findModelVisibility()
        .find('[data-testid*="model-visibility-filtered"]')
        .should('be.visible');
    });

    it('should show "Filtered" badge when source has excluded models', () => {
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('Custom YAML');
      row.findName().should('be.visible');
      row.findModelVisibility().should('be.visible').and('contain', 'Filtered');
    });

    it('should show "All models" badge when source has no filters', () => {
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('Default Catalog');
      row.findName().should('be.visible');
      row.findModelVisibility().should('be.visible').and('contain', 'All models');
      row
        .findModelVisibility()
        .find('[data-testid*="model-visibility-unfiltered"]')
        .should('be.visible');
    });
  });

  describe('Validation status column', () => {
    it('should show "-" for default sources', () => {
      setupMocks([], { catalogs: [defaultYamlSource, huggingFaceSource] });
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('Default Catalog');
      row.findName().should('be.visible');
      row.shouldHaveValidationStatus(EMPTY_CUSTOM_PROPERTY_VALUE);
    });

    it('should show "-" for disabled sources', () => {
      setupMocks([], { catalogs: [defaultYamlSource, huggingFaceSource] });
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('Default Catalog');
      row.findName().should('be.visible');
      row.shouldHaveValidationStatus(EMPTY_CUSTOM_PROPERTY_VALUE);
    });

    it('should show "Connected" status for available sources', () => {
      const availableSource = mockCatalogSource({
        id: 'hf-google',
        name: 'HuggingFace Google',
        status: 'available',
      });
      setupMocks([availableSource], { catalogs: [huggingFaceSource] });
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('HuggingFace Google');
      row.findName().should('be.visible');
      row.shouldHaveValidationStatus('Ready');
      row.findValidationStatus().findByTestId('source-status-connected-hf-google').should('exist');
    });

    it('should show "Starting" status when no matching source found', () => {
      setupMocks([], { catalogs: [huggingFaceSource] });
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('HuggingFace Google');
      row.findName().should('be.visible');
      row.shouldHaveValidationStatus('Starting');
      row.findValidationStatus().findByTestId('source-status-starting-hf-google').should('exist');
    });

    it('should show "Starting" status when source has no status field', () => {
      const startingSource = mockCatalogSource({
        id: 'hf-google',
        name: 'HuggingFace Google',
        status: undefined,
      });
      setupMocks([startingSource], { catalogs: [huggingFaceSource] });
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('HuggingFace Google');
      row.findName().should('be.visible');
      row.shouldHaveValidationStatus('Starting');
    });

    it('should show "Starting" status when re-enabling source (enabled=true but status=disabled)', () => {
      const reenablingSource = mockCatalogSource({
        id: 'hf-google',
        name: 'HuggingFace Google',
        status: 'disabled',
      });
      setupMocks([reenablingSource], { catalogs: [huggingFaceSource] }); // huggingFaceSource has enabled=true
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('HuggingFace Google');
      row.findName().should('be.visible');
      row.shouldHaveValidationStatus('Starting');
      row.findValidationStatus().findByTestId('source-status-starting-hf-google').should('exist');
    });

    it('should show "Failed" status with error message for error sources', () => {
      const errorSource = mockCatalogSource({
        id: 'hf-google',
        name: 'HuggingFace Google',
        status: 'error',
        error: 'The provided API key is invalid or has expired. Please update your credentials.',
      });
      setupMocks([errorSource], { catalogs: [huggingFaceSource] });
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('HuggingFace Google');
      row.findName().should('be.visible');
      row.shouldHaveValidationStatus('Failed');
      row.findValidationStatus().findByTestId('source-status-failed-hf-google').should('exist');
      row.findValidationStatusErrorLink().should('exist');
    });

    it('should show truncated error message for long errors', () => {
      const longErrorMessage =
        'The specified organization "invalid-org" does not exist or you don\'t have access to it. Please verify the organization name and ensure you have the necessary permissions to access models from this organization.';
      const errorSource = mockCatalogSource({
        id: 'hf-google',
        name: 'HuggingFace Google',
        status: 'error',
        error: longErrorMessage,
      });
      setupMocks([errorSource], { catalogs: [huggingFaceSource] });
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('HuggingFace Google');
      row.findName().should('be.visible');
      row.findValidationStatusErrorLink().find('.pf-v6-c-truncate').should('exist');
      row.findValidationStatusErrorLink().should('contain', longErrorMessage);
    });

    it('should open error modal when clicking error message', () => {
      const errorSource = mockCatalogSource({
        id: 'hf-google',
        name: 'HuggingFace Google',
        status: 'error',
        error: 'The provided API key is invalid or has expired.',
      });
      setupMocks([errorSource], { catalogs: [huggingFaceSource] });
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('HuggingFace Google');
      row.findName().should('be.visible');
      row.clickValidationStatusErrorLink();

      // Check modal is displayed
      cy.findByTestId('catalog-source-status-error-modal').should('exist');
      cy.findByTestId('catalog-source-status-error-modal')
        .contains('Source status')
        .should('exist');
      cy.findByTestId('catalog-source-status-error-modal').contains('Failed').should('exist');
      cy.findByTestId('catalog-source-status-error-alert').should('exist');
      cy.findByTestId('catalog-source-status-error-alert')
        .contains('Validation failed')
        .should('exist');
      cy.findByTestId('catalog-source-status-error-message').should(
        'contain',
        'The provided API key is invalid or has expired.',
      );
    });

    it('should close error modal when clicking close button', () => {
      const errorSource = mockCatalogSource({
        id: 'hf-google',
        name: 'HuggingFace Google',
        status: 'error',
        error: 'The provided API key is invalid.',
      });
      setupMocks([errorSource], { catalogs: [huggingFaceSource] });
      modelCatalogSettings.visit();
      const row = modelCatalogSettings.getRow('HuggingFace Google');
      row.clickValidationStatusErrorLink();

      cy.findByTestId('catalog-source-status-error-modal').should('exist');
      cy.findByTestId('catalog-source-status-error-modal')
        .findByRole('button', { name: 'Close' })
        .click();
      cy.findByTestId('catalog-source-status-error-modal').should('not.exist');
    });
  });
});

describe('Manage Source Page', () => {
  beforeEach(() => {
    setupMocks([], mockCatalogSourceConfigList({}));
  });

  describe('Add Source Mode', () => {
    it('should display add source page', () => {
      manageSourcePage.visitAddSource();
      manageSourcePage.findAddSourceTitle();
      manageSourcePage.findAddSourceDescription();
    });

    it('should display correct breadcrumb for add source', () => {
      manageSourcePage.visitAddSource();
      manageSourcePage.findBreadcrumb().should('exist');
      manageSourcePage.findBreadcrumbAction().should('contain', 'Add a source');
    });

    it('should navigate back to settings from breadcrumb', () => {
      manageSourcePage.visitAddSource();
      manageSourcePage.findBreadcrumb().click({ force: true });
      modelCatalogSettings.findHeading();
    });

    it('should navigate back to settings from cancel button', () => {
      manageSourcePage.visitAddSource();
      manageSourcePage.findCancelButton().click();
      modelCatalogSettings.findHeading();
    });

    it('should display form fields', () => {
      manageSourcePage.visitAddSource();
      manageSourcePage.findNameInput().should('exist');
      manageSourcePage.findSourceTypeHuggingFace().should('exist');
      manageSourcePage.findSourceTypeYaml().should('exist');
      manageSourcePage.findEnableSourceCheckbox().should('exist');
      manageSourcePage.findSubmitButton().should('exist');
      manageSourcePage.findPreviewButton().should('exist');
      manageSourcePage.findCancelButton().should('exist');
    });

    it('should show Hugging Face fields by default', () => {
      manageSourcePage.visitAddSource({ enableTempDevCatalogHuggingFaceApiKeyFeature: true });
      manageSourcePage.findSourceTypeHuggingFace().should('be.checked');
      manageSourcePage.findCredentialsSection().should('exist');
      manageSourcePage.findAccessTokenInput().should('exist');
      manageSourcePage.findOrganizationInput().should('exist');
      manageSourcePage.findYamlSection().should('not.exist');
    });

    it('should show YAML fields when YAML type is selected', () => {
      manageSourcePage.visitAddSource();
      manageSourcePage.selectSourceType('yaml');
      manageSourcePage.findSourceTypeYaml().should('be.checked');
      manageSourcePage.findYamlSection().should('exist');
      manageSourcePage.findYamlContentInput().should('exist');
      manageSourcePage.findCredentialsSection().should('not.exist');
    });

    it('should have Add button disabled by default', () => {
      manageSourcePage.visitAddSource();
      manageSourcePage.findSubmitButton().should('be.disabled');
    });

    it('should have Preview button disabled by default', () => {
      manageSourcePage.visitAddSource();
      manageSourcePage.findPreviewButton().should('be.disabled');
    });

    it('should show validation error when name field is empty and touched', () => {
      manageSourcePage.visitAddSource();
      manageSourcePage.findNameInput().focus().blur();
      manageSourcePage.findNameError().should('exist');
      manageSourcePage.findNameError().should('contain', 'Name is required');
    });

    it('should enable Add button when all required HF fields are filled', () => {
      manageSourcePage.visitAddSource({ enableTempDevCatalogHuggingFaceApiKeyFeature: true });
      manageSourcePage.fillSourceName('Test Source');
      manageSourcePage.fillAccessToken('test-token-123');
      manageSourcePage.fillOrganization('Google');
      manageSourcePage.findSubmitButton().should('not.be.disabled');
    });

    it('should enable Preview button when HF credentials are filled', () => {
      manageSourcePage.visitAddSource({ enableTempDevCatalogHuggingFaceApiKeyFeature: true });
      manageSourcePage.fillAccessToken('test-token-123');
      manageSourcePage.fillOrganization('Google');
      manageSourcePage.findPreviewButton().should('not.be.disabled');
    });

    it('should enable Add button when all required YAML fields are filled', () => {
      manageSourcePage.visitAddSource();
      manageSourcePage.selectSourceType('yaml');
      manageSourcePage.fillSourceName('Test Source');
      manageSourcePage.fillYamlContent('test: yaml\ncontent: here');
      manageSourcePage.findSubmitButton().should('not.be.disabled');
    });

    it('should enable Preview button when YAML content is filled', () => {
      manageSourcePage.visitAddSource();
      manageSourcePage.selectSourceType('yaml');
      manageSourcePage.fillYamlContent('test: yaml\ncontent: here');
      manageSourcePage.findPreviewButton().should('not.be.disabled');
    });

    it('should show validation errors for HF organization when touched', () => {
      manageSourcePage.visitAddSource();

      manageSourcePage.findOrganizationInput().focus().blur();
      manageSourcePage.findOrganizationError().should('exist');
      manageSourcePage.findOrganizationError().should('contain', 'Organization is required');
    });

    it('should show validation error for YAML content when touched', () => {
      manageSourcePage.visitAddSource();
      manageSourcePage.selectSourceType('yaml');
      manageSourcePage.findYamlContentInput().focus().blur();
      manageSourcePage.findYamlContentError().should('exist');
      manageSourcePage.findYamlContentError().should('contain', 'YAML content is required');
    });

    it('should expand and collapse model visibility section', () => {
      manageSourcePage.visitAddSource();
      manageSourcePage.findAllowedModelsInput().should('not.exist');
      manageSourcePage.findExcludedModelsInput().should('not.exist');

      manageSourcePage.toggleModelVisibility();
      manageSourcePage.findAllowedModelsInput().should('exist');
      manageSourcePage.findExcludedModelsInput().should('exist');

      manageSourcePage.toggleModelVisibility();
      manageSourcePage.findAllowedModelsInput().should('not.exist');
      manageSourcePage.findExcludedModelsInput().should('not.exist');
    });

    it('should allow entering filter values', () => {
      manageSourcePage.visitAddSource();
      manageSourcePage.toggleModelVisibility();

      const allowedModels = 'model-1\nmodel-2*\nmodel-3';
      const excludedModels = 'test-model*\ndemo-model';

      manageSourcePage.fillAllowedModels(allowedModels);
      manageSourcePage.fillExcludedModels(excludedModels);

      manageSourcePage.findAllowedModelsInput().should('have.value', allowedModels);
      manageSourcePage.findExcludedModelsInput().should('have.value', excludedModels);
    });

    it('should have enable source checkbox unchecked by default', () => {
      manageSourcePage.visitAddSource();
      manageSourcePage.findEnableSourceCheckbox().should('not.be.checked');
    });

    it('should allow toggling enable source checkbox', () => {
      manageSourcePage.visitAddSource();
      manageSourcePage.findEnableSourceCheckbox().should('not.be.checked');
      manageSourcePage.toggleEnableSource();
      manageSourcePage.findEnableSourceCheckbox().should('be.checked');
      manageSourcePage.toggleEnableSource();
      manageSourcePage.findEnableSourceCheckbox().should('not.be.checked');
    });

    it('should clear validation errors when fields are filled', () => {
      manageSourcePage.visitAddSource();

      // Trigger validation errors
      manageSourcePage.findNameInput().focus().blur();
      manageSourcePage.findOrganizationInput().focus().blur();

      manageSourcePage.findNameError().should('exist');
      manageSourcePage.findOrganizationError().should('exist');

      // Fill fields
      manageSourcePage.fillSourceName('Test Source');
      manageSourcePage.fillOrganization('Google');

      // Errors should be cleared
      manageSourcePage.findNameError().should('not.exist');
      manageSourcePage.findOrganizationError().should('not.exist');
    });

    it('should maintain form state when switching between source types', () => {
      manageSourcePage.visitAddSource({ enableTempDevCatalogHuggingFaceApiKeyFeature: true });

      // Fill name and HF fields
      manageSourcePage.fillSourceName('Test Source');
      manageSourcePage.fillAccessToken('test-token');
      manageSourcePage.fillOrganization('Google');

      // Switch to YAML
      manageSourcePage.selectSourceType('yaml');
      manageSourcePage.findYamlSection().should('exist');

      // Name should be maintained
      manageSourcePage.findNameInput().should('have.value', 'Test Source');

      // Fill YAML
      manageSourcePage.fillYamlContent('test: yaml');

      // Switch back to HF
      manageSourcePage.selectSourceType('huggingface');
      manageSourcePage.findCredentialsSection().should('exist');

      // All values should be maintained
      manageSourcePage.findNameInput().should('have.value', 'Test Source');
      manageSourcePage.findAccessTokenInput().should('have.value', 'test-token');
      manageSourcePage.findOrganizationInput().should('have.value', 'Google');
    });

    it('should dynamically update filter descriptions with organization name', () => {
      manageSourcePage.visitAddSource();
      manageSourcePage.toggleModelVisibility();

      // Before entering organization, should show generic text
      cy.contains(
        'Optionally filter which models from this source appear in the model catalog',
      ).should('exist');

      // Fill organization name
      manageSourcePage.fillOrganization('Google');

      // After entering organization, should show organization-specific text
      cy.contains(
        'Optionally filter which Google models from this source appear in the model catalog',
      ).should('exist');
      cy.contains('all Google models from the source will be visible').should('exist');

      // Change organization name
      manageSourcePage.fillOrganization('Meta');

      // Text should update to new organization
      cy.contains(
        'Optionally filter which Meta models from this source appear in the model catalog',
      ).should('exist');
      cy.contains('all Meta models from the source will be visible').should('exist');
    });

    it('should display model catalog preview panel', () => {
      manageSourcePage.visitAddSource();
      manageSourcePage.findPreviewPanel().should('exist');
      manageSourcePage.findPreviewPanelTitle().should('be.visible');
      manageSourcePage.findPreviewPanelEmptyMessage().should('be.visible');
    });

    it('should have three preview buttons', () => {
      manageSourcePage.visitAddSource();
      // One in the action group (bottom left)
      manageSourcePage.findPreviewButton().should('exist');
      // One in the preview panel header (top right)
      manageSourcePage.findPreviewButtonHeader().should('exist');
      // One in the preview panel body (center)
      manageSourcePage.findPreviewButtonPanel().should('exist');
    });

    it('should have all three preview buttons disabled by default', () => {
      manageSourcePage.visitAddSource();
      manageSourcePage.findPreviewButton().should('be.disabled');
      manageSourcePage.findPreviewButtonHeader().should('be.disabled');
      manageSourcePage.findPreviewButtonPanel().should('be.disabled');
    });

    it('should enable all three preview buttons when credentials are filled', () => {
      manageSourcePage.visitAddSource({ enableTempDevCatalogHuggingFaceApiKeyFeature: true });
      manageSourcePage.fillAccessToken('test-token');
      manageSourcePage.fillOrganization('Google');
      manageSourcePage.findPreviewButton().should('not.be.disabled');
      manageSourcePage.findPreviewButtonHeader().should('not.be.disabled');
      manageSourcePage.findPreviewButtonPanel().should('not.be.disabled');
    });

    it('submit add source form with yaml source type', () => {
      cy.intercept('POST', '/model-registry/api/v1/settings/model_catalog/source_configs', {
        data: mockYamlCatalogSourceConfig({}),
      }).as('addSourcewithYamlType');
      manageSourcePage.visitAddSource();
      manageSourcePage.findNameInput().type('sample source');
      manageSourcePage.selectSourceType('yaml');
      manageSourcePage.findSourceTypeYaml().should('be.checked');

      manageSourcePage.findYamlSection().should('exist');
      manageSourcePage.findYamlContentInput().should('exist');
      manageSourcePage.findYamlContentInput().type('models:\n  - name: model1');

      manageSourcePage.toggleModelVisibility();
      manageSourcePage.findAllowedModelsInput().should('exist');
      manageSourcePage.findAllowedModelsInput().type('model-1-*, model-2-*');
      manageSourcePage.findExcludedModelsInput().should('exist');
      manageSourcePage.findExcludedModelsInput().type('model-3-*, model-4-*');

      manageSourcePage.findSubmitButton().should('be.enabled');
      manageSourcePage.findSubmitButton().click();
      cy.wait('@addSourcewithYamlType').then((interception) => {
        expect(interception.request.body).to.eql({
          data: {
            name: 'sample source',
            id: 'sample_source',
            isDefault: false,
            includedModels: ['model-1-*', 'model-2-*'],
            excludedModels: ['model-3-*', 'model-4-*'],
            enabled: false,
            type: CatalogSourceType.YAML,
            yaml: 'models:\n  - name: model1',
          },
        });
      });
    });

    it('submit the add source form with hugging face source type', () => {
      cy.intercept('POST', '/model-registry/api/v1/settings/model_catalog/source_configs', {
        data: mockHuggingFaceCatalogSourceConfig({}),
      }).as('addSourcewithHuggingFaceType');
      manageSourcePage.visitAddSource({ enableTempDevCatalogHuggingFaceApiKeyFeature: true });
      manageSourcePage.findNameInput().type('sample source');
      manageSourcePage.selectSourceType('huggingface');
      manageSourcePage.findSourceTypeHuggingFace().should('be.checked');

      manageSourcePage.findAccessTokenInput().type('apikey');
      manageSourcePage.findOrganizationInput().type('org1');

      manageSourcePage.toggleModelVisibility();
      manageSourcePage.findAllowedModelsInput().should('exist');
      manageSourcePage.findAllowedModelsInput().type('model-1-*, model-2-*');
      manageSourcePage.findExcludedModelsInput().should('exist');
      manageSourcePage.findExcludedModelsInput().type('model-3-*, model-4-*');

      manageSourcePage.findSubmitButton().should('be.enabled');
      manageSourcePage.findSubmitButton().click();
      cy.wait('@addSourcewithHuggingFaceType').then((interception) => {
        expect(interception.request.body).to.eql({
          data: {
            name: 'sample source',
            id: 'sample_source',
            type: CatalogSourceType.HUGGING_FACE,
            includedModels: ['model-1-*', 'model-2-*'],
            excludedModels: ['model-3-*', 'model-4-*'],
            enabled: false,
            isDefault: false,
            allowedOrganization: 'org1',
            apiKey: 'apikey',
          },
        });
      });
    });
  });

  describe('Manage Source Mode', () => {
    const catalogSourceId = 'test-source-id';

    beforeEach(() => {
      setupMocks([], mockCatalogSourceConfigList({}));
    });

    it('should display manage source page', () => {
      manageSourcePage.visitManageSource(catalogSourceId);
      manageSourcePage.findManageSourceTitle();
      manageSourcePage.findManageSourceDescription();
    });

    it('should display correct breadcrumb for manage source', () => {
      manageSourcePage.visitManageSource(catalogSourceId);
      manageSourcePage.findBreadcrumb().should('exist');
      manageSourcePage.findBreadcrumbAction().should('contain', 'Manage source');
    });

    it('should navigate back to settings from breadcrumb', () => {
      manageSourcePage.visitManageSource(catalogSourceId);
      manageSourcePage.findBreadcrumb().click({ force: true });
      modelCatalogSettings.findHeading();
    });

    it('should show Save button instead of Add button', () => {
      cy.intercept('GET', '/model-registry/api/v1/settings/model_catalog/source_configs/**', {
        data: mockYamlCatalogSourceConfig({
          id: 'source_2',
          name: 'Source 2',
          isDefault: false,
          includedModels: ['model1', 'model2'],
          excludedModels: ['model3'],
          enabled: false,
          yaml: 'models:\n  - name: model1',
        }),
      });
      manageSourcePage.visitManageSource(catalogSourceId);
      manageSourcePage.findSubmitButton().should('exist');
      manageSourcePage.findSubmitButton().should('be.enabled');
      manageSourcePage.findSubmitButton().should('contain', 'Save');
    });

    it('should do the form validation for default source config', () => {
      cy.intercept('GET', '/model-registry/api/v1/settings/model_catalog/source_configs/**', {
        data: mockYamlCatalogSourceConfig({
          id: 'source_2',
          name: 'Source 2',
          isDefault: true,
          includedModels: ['model1', 'model2'],
          excludedModels: ['model3'],
          enabled: false,
        }),
      });
      manageSourcePage.visitManageSource(catalogSourceId);
      manageSourcePage.findNameInput().should('have.value', 'Source 2');
      manageSourcePage.findSubmitButton().should('exist');
      manageSourcePage.findSubmitButton().should('be.enabled');
      manageSourcePage.findSubmitButton().should('contain', 'Save');
    });
  });

  it('should successfully update the source with yaml type', () => {
    cy.intercept('GET', '/model-registry/api/v1/settings/model_catalog/source_configs/**', {
      data: mockYamlCatalogSourceConfig({
        id: 'source_2',
        name: 'Source 2',
        isDefault: false,
        includedModels: ['model1', 'model2'],
        excludedModels: ['model3'],
        enabled: false,
        yaml: 'models:\n  - name: model1',
      }),
    });

    cy.intercept('PATCH', '/model-registry/api/v1/settings/model_catalog/source_configs/*', {
      statusCode: 200,
      body: {
        data: mockYamlCatalogSourceConfig({ id: 'source_2', isDefault: false }),
      },
    }).as('manageSourcewithYamlType');

    manageSourcePage.visitManageSource('source_2');
    cy.url().should('include', '/model-catalog-settings/manage-source/source_2');
    manageSourcePage.findNameInput().should('have.value', 'Source 2');
    manageSourcePage.findSourceTypeHuggingFace().should('not.exist');
    manageSourcePage.findSourceTypeYaml().should('not.exist');

    manageSourcePage.findAllowedModelsInput().should('exist');
    manageSourcePage.findExcludedModelsInput().should('exist');
    manageSourcePage.findAllowedModelsInput().type(', model-1-*, model-2-*');
    manageSourcePage.findExcludedModelsInput().type(', model-3-*, model-4-*');
    manageSourcePage.findEnableSourceCheckbox().should('not.be.checked');
    manageSourcePage.findEnableSourceCheckbox().check();

    manageSourcePage.findSubmitButton().should('be.enabled');
    manageSourcePage.findSubmitButton().should('have.text', 'Save');
    manageSourcePage.findSubmitButton().click();

    cy.wait('@manageSourcewithYamlType').then((interception) => {
      expect(interception.request.body).to.eql({
        data: {
          name: 'Source 2',
          type: CatalogSourceType.YAML,
          includedModels: ['model1', 'model2', 'model-1-*', 'model-2-*'],
          excludedModels: ['model3', 'model-3-*', 'model-4-*'],
          enabled: true,
          isDefault: false,
          yaml: 'models:\n  - name: model1',
        },
      });
    });
  });

  it('should successfully update the source with yaml type and default one', () => {
    cy.intercept('GET', '/model-registry/api/v1/settings/model_catalog/source_configs/**', {
      data: mockYamlCatalogSourceConfig({
        id: 'sample_source_1',
        name: 'Sample source 1',
        isDefault: true,
        includedModels: [],
        excludedModels: [],
      }),
    });

    cy.intercept('PATCH', '/model-registry/api/v1/settings/model_catalog/source_configs/*', {
      statusCode: 200,
      body: {
        data: mockYamlCatalogSourceConfig({ id: 'sample_source_1' }),
      },
    }).as('manageSourcewithYamlType');

    manageSourcePage.visitManageSource('sample_source_1');
    cy.url().should('include', '/model-catalog-settings/manage-source/sample_source_1');
    manageSourcePage.findNameInput().should('have.value', 'Sample source 1');
    manageSourcePage.findSourceTypeHuggingFace().should('not.exist');
    manageSourcePage.findSourceTypeYaml().should('not.exist');

    manageSourcePage.findAllowedModelsInput().should('exist');
    manageSourcePage.findExcludedModelsInput().should('exist');
    manageSourcePage.findAllowedModelsInput().type('model-1-*, model-2-*');
    manageSourcePage.findExcludedModelsInput().type('model-3-*, model-4-*');
    manageSourcePage.findEnableSourceCheckbox().should('be.checked');
    manageSourcePage.findEnableSourceCheckbox().uncheck();

    manageSourcePage.findSubmitButton().should('be.enabled');
    manageSourcePage.findSubmitButton().should('have.text', 'Save');
    manageSourcePage.findSubmitButton().click();

    cy.wait('@manageSourcewithYamlType').then((interception) => {
      expect(interception.request.body).to.eql({
        data: {
          enabled: false,
          includedModels: ['model-1-*', 'model-2-*'],
          excludedModels: ['model-3-*', 'model-4-*'],
        },
      });
    });
  });

  it('should successfully update the source with huggingface type', () => {
    cy.intercept('GET', '/model-registry/api/v1/settings/model_catalog/source_configs/**', {
      data: mockHuggingFaceCatalogSourceConfig({
        id: 'huggingface_source_3',
        name: 'Huggingface source 3',
        allowedOrganization: 'org1',
        isDefault: false,
      }),
    });

    cy.intercept(
      'PATCH',
      `/model-registry/api/${MODEL_CATALOG_API_VERSION}/settings/model_catalog/source_configs/*`,
      {
        statusCode: 200,
        body: {
          data: mockHuggingFaceCatalogSourceConfig({
            id: 'huggingface_source_3',
            name: 'Huggingface source 3',
            allowedOrganization: 'org1',
            isDefault: false,
          }),
        },
      },
    ).as('manageSourcewithHuggingFaceType');

    manageSourcePage.visitManageSource('huggingface_source_3', {
      enableTempDevCatalogHuggingFaceApiKeyFeature: true,
    });
    manageSourcePage.findNameInput().should('have.value', 'Huggingface source 3');

    manageSourcePage.findAccessTokenInput().should('have.value', 'apikey');
    manageSourcePage.findOrganizationInput().should('have.value', 'org1');

    manageSourcePage.toggleModelVisibility();
    manageSourcePage.findAllowedModelsInput().should('exist');
    manageSourcePage.findAllowedModelsInput().type('model-1-*, model-2-*');
    manageSourcePage.findExcludedModelsInput().should('exist');
    manageSourcePage.findExcludedModelsInput().type('model-3-*, model-4-*');
    manageSourcePage.findEnableSourceCheckbox().should('be.checked');
    manageSourcePage.findEnableSourceCheckbox().uncheck();

    manageSourcePage.findSubmitButton().should('be.enabled');
    manageSourcePage.findSubmitButton().should('have.text', 'Save');
    manageSourcePage.findSubmitButton().click();
    cy.wait('@manageSourcewithHuggingFaceType').then((interception) => {
      expect(interception.request.body).to.eql({
        data: {
          name: 'Huggingface source 3',
          apiKey: 'apikey',
          allowedOrganization: 'org1',
          type: CatalogSourceType.HUGGING_FACE,
          includedModels: ['model-1-*', 'model-2-*'],
          excludedModels: ['model-3-*', 'model-4-*'],
          enabled: false,
          isDefault: false,
        },
      });
    });
  });
});
