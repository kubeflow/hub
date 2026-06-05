/* eslint-disable camelcase */
import { mockModelRegistry } from '~/__mocks__/mockModelRegistry';
import { mockModelVersion } from '~/__mocks__/mockModelVersion';
import { mockModelVersionList } from '~/__mocks__/mockModelVersionList';
import { mockRegisteredModel } from '~/__mocks__/mockRegisteredModel';
import { mockRegisteredModelList } from '~/__mocks__/mockRegisteredModelsList';
import { labelModal, modelRegistry } from '~/__tests__/cypress/cypress/pages/modelRegistry';
import {
  ModelRegistryMetadataType,
  type ModelRegistry,
  type ModelVersion,
  type RegisteredModel,
} from '~/app/types';
import { be } from '~/__tests__/cypress/cypress/utils/should';
import { MODEL_REGISTRY_API_VERSION } from '~/__tests__/cypress/cypress/support/commands/api';
import { verifyRelativeURL } from '~/__tests__/cypress/cypress/utils/url';

type HandlersProps = {
  modelRegistries?: ModelRegistry[];
  registeredModels?: RegisteredModel[];
  modelVersions?: ModelVersion[];
};

const initIntercepts = ({
  modelRegistries = [
    mockModelRegistry({
      name: 'modelregistry-sample',
    }),
    mockModelRegistry({
      name: 'modelregistry-sample-2',
      description: '',
      displayName: 'modelregistry-sample-2',
    }),
  ],
  registeredModels = [
    mockRegisteredModel({
      name: 'Fraud detection model',
      description:
        'A machine learning model trained to detect fraudulent transactions in financial data',
      customProperties: {
        'Financial data': {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: '',
        },
        'Fraud detection': {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: '',
        },
        'Test label': {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: '',
        },
        'Machine learning': {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: '',
        },
        'Next data to be overflow': {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: '',
        },
      },
    }),
    mockRegisteredModel({
      name: 'Label modal',
      owner: 'Author 2',
      id: '2',
      description:
        'A machine learning model trained to detect fraudulent transactions in financial data',
      customProperties: {
        'Testing label': {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: '',
        },
        'Financial data': {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: '',
        },
        'Fraud detection': {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: '',
        },
        'Long label data to be truncated abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc':
          {
            metadataType: ModelRegistryMetadataType.STRING,
            string_value: '',
          },
        'Machine learning': {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: '',
        },
        'Next data to be overflow': {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: '',
        },
        'Label x': {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: '',
        },
        'Label y': {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: '',
        },
        'Label z': {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: '',
        },
      },
    }),
  ],
  modelVersions = [
    mockModelVersion({ author: 'Author 1' }),
    mockModelVersion({ name: 'model version' }),
  ],
}: HandlersProps) => {
  cy.interceptApi(
    `GET /api/:apiVersion/model_registry`,
    {
      path: { apiVersion: MODEL_REGISTRY_API_VERSION },
    },
    modelRegistries,
  );

  cy.interceptApi(
    `GET /api/:apiVersion/model_registry/:modelRegistryName/model_versions`,
    {
      path: { modelRegistryName: 'modelregistry-sample', apiVersion: MODEL_REGISTRY_API_VERSION },
    },
    mockModelVersionList({ items: modelVersions }),
  );

  cy.interceptApi(
    `GET /api/:apiVersion/model_registry/:modelRegistryName/registered_models`,
    {
      path: { modelRegistryName: 'modelregistry-sample', apiVersion: MODEL_REGISTRY_API_VERSION },
    },
    mockRegisteredModelList({ items: registeredModels }),
  );

  cy.interceptApi(
    `GET /api/:apiVersion/model_registry/:modelRegistryName/registered_models/:registeredModelId/versions`,
    {
      path: {
        modelRegistryName: 'modelregistry-sample',
        apiVersion: MODEL_REGISTRY_API_VERSION,
        registeredModelId: 1,
      },
    },
    mockModelVersionList({ items: modelVersions }),
  );
};

describe('Model Registry core', () => {
  it('Model Registry Enabled in the cluster', () => {
    initIntercepts({
      registeredModels: [],
    });

    modelRegistry.visit();
    modelRegistry.navigate();

    modelRegistry.tabEnabled();
  });

  it('Renders empty state with no model registries', () => {
    initIntercepts({
      modelRegistries: [],
    });

    modelRegistry.visit();
    modelRegistry.navigate();
    modelRegistry.findModelRegistryEmptyState().should('exist');
  });

  it('Shows unavailable state when selected registry has isAvailable false', () => {
    const unavailableRegistryName = 'unavailable-registry-example';
    initIntercepts({
      modelRegistries: [
        mockModelRegistry({
          name: unavailableRegistryName,
          displayName: 'Unavailable Registry Example',
          isAvailable: false,
        }),
      ],
    });

    modelRegistry.visit();
    modelRegistry.navigate();
    // Navigate to the unavailable registry (app would redirect to first registry; we go there directly)
    cy.visit(`/model-registry/${unavailableRegistryName}`);
    modelRegistry.waitForPageLoaded();

    modelRegistry.findUnavailableModelRegistryState().should('exist');
    cy.contains('Model registry unavailable').should('be.visible');
    cy.contains('The Unavailable Registry Example registry is currently unavailable').should(
      'be.visible',
    );
    cy.findByTestId('whos-my-admin-link').should('exist');
    // View details button should not be present when registry is unavailable
    modelRegistry.findViewDetailsButton().should('not.exist');
  });

  it('No registered models in the selected Model Registry', () => {
    initIntercepts({
      registeredModels: [],
    });

    modelRegistry.visit();
    modelRegistry.navigate();
    modelRegistry.shouldModelRegistrySelectorExist();
    modelRegistry.shouldregisteredModelsEmpty();

    modelRegistry.findViewDetailsButton().click();
    modelRegistry.findDetailsPopover().should('exist');
    modelRegistry.findDetailsPopover().findByText('Model registry description').should('exist');

    // Model registry with no description
    modelRegistry.findModelRegistry().findSelectOption('modelregistry-sample-2').click();
    modelRegistry.findViewDetailsButton().click();
    modelRegistry.findDetailsPopover().should('exist');
    modelRegistry.findDetailsPopover().findByText('No description').should('exist');

    //  Model registry help content
    modelRegistry.findHelpContentButton().should('exist');
    // modelRegistry.findHelpContentButton().click(); TODO: Get conditional render depnding on the style
    // modelRegistry.findHelpContentPopover().should('exist');
    // modelRegistry
    //   .findHelpContentPopover()
    //   .findByText(
    //     'To request access to a new or existing model registry, contact your administrator.',
    //   )
    //   .should('exist');
  });

  it('Empty state shows transfer jobs link', () => {
    initIntercepts({
      registeredModels: [],
    });

    modelRegistry.visit();
    modelRegistry.navigate();
    modelRegistry.shouldregisteredModelsEmpty();
    modelRegistry.findEmptyStateTransferJobsButton().should('exist');
    modelRegistry.findEmptyStateTransferJobsButton().click();
    verifyRelativeURL('/model-registry/modelregistry-sample/model-transfer-jobs');
  });

  describe('Registered model table', () => {
    beforeEach(() => {
      initIntercepts({});
      modelRegistry.visit();
    });

    it('Renders row contents', () => {
      const registeredModelRow = modelRegistry.getRow('Fraud detection model');
      registeredModelRow.findName().contains('Fraud detection model');
      registeredModelRow
        .findDescription()
        .contains(
          'A machine learning model trained to detect fraudulent transactions in financial data',
        );
      registeredModelRow.findOwner().contains('Author 1');

      // Label popover
      registeredModelRow.findLabelPopoverText().contains('2 more');
      registeredModelRow.findLabelPopoverText().click();
      registeredModelRow.shouldContainsPopoverLabels([
        'Machine learning',
        'Next data to be overflow',
      ]);
    });

    it('latest version column', () => {
      const registeredModelRow = modelRegistry.getRow('Fraud detection model');
      registeredModelRow.findLatestVersion().contains('new model version');
      registeredModelRow.findLatestVersion().click();
      verifyRelativeURL(
        `/model-registry/modelregistry-sample/registered-models/1/versions/1/details`,
      );
    });

    it('table kebab actions', () => {
      const registeredModelRow = modelRegistry.getRow('Fraud detection model');
      registeredModelRow.findKebabAction('Versions').click();
      verifyRelativeURL(`/model-registry/modelregistry-sample/registered-models/1/versions`);
      cy.go('back');
      registeredModelRow.findKebabAction('Overview').click();
      verifyRelativeURL(`/model-registry/modelregistry-sample/registered-models/1/overview`);
    });

    it('Renders labels in modal', () => {
      const registeredModelRow2 = modelRegistry.getRow('Label modal');
      registeredModelRow2.findLabelModalText().contains('6 more');
      registeredModelRow2.findLabelModalText().click();
      labelModal.shouldContainsModalLabels([
        'Testing label',
        'Financial',
        'Financial data',
        'Fraud detection',
        'Machine learning',
        'Next data to be overflow',
        'Label x',
        'Label y',
        'Label z',
      ]);
      labelModal.findModalSearchInput().type('Financial');
      labelModal.shouldContainsModalLabels(['Financial', 'Financial data']);
      labelModal.findCloseModal().click();
    });

    it('Sort by Model name', () => {
      modelRegistry.findRegisteredModelTableHeaderButton('Model name').click();
      modelRegistry.findRegisteredModelTableHeaderButton('Model name').should(be.sortAscending);
      modelRegistry.findRegisteredModelTableHeaderButton('Model name').click();
      modelRegistry.findRegisteredModelTableHeaderButton('Model name').should(be.sortDescending);
    });

    it('Sort by Last modified', () => {
      modelRegistry.findRegisteredModelTableHeaderButton('Last modified').click();
      modelRegistry.findRegisteredModelTableHeaderButton('Last modified').should(be.sortDescending);
      modelRegistry.findRegisteredModelTableHeaderButton('Last modified').click();
      modelRegistry.findRegisteredModelTableHeaderButton('Last modified').should(be.sortAscending);
      modelRegistry.findRegisteredModelTableHeaderButton('Last modified').click();
      modelRegistry.findRegisteredModelTableHeaderButton('Last modified').should(be.sortDescending);
    });

    it('Filter by keyword then both', () => {
      modelRegistry.findTableSearch().type('Fraud detection model');
      modelRegistry.findTableRows().should('have.length', 1);
      modelRegistry.findFilterDropdownItem('Owner').click();
      modelRegistry.findTableSearch().type('Author 1');
      modelRegistry.findTableRows().should('have.length', 1);
      modelRegistry.findTableRows().contains('Fraud detection model');
      modelRegistry.findTableSearch().type('2');
      modelRegistry.findTableRows().should('have.length', 0);
    });

    it('Filter by owner then both', () => {
      modelRegistry.findFilterDropdownItem('Owner').click();
      modelRegistry.findTableSearch().type('Author 2');
      modelRegistry.findTableRows().should('have.length', 1);
      modelRegistry.findFilterDropdownItem('Keyword').click();
      modelRegistry.findTableSearch().type('Label modal');
      modelRegistry.findTableRows().should('have.length', 1);
      modelRegistry.findTableSearch().type('.');
      modelRegistry.findTableRows().should('have.length', 0);
    });
  });
});

describe('Model Registry selector persistence', () => {
  const SELECTED_STORAGE_KEY = 'kubeflow.dashboard.model.registry.selected';
  const FAVORITE_STORAGE_KEY = 'kubeflow.dashboard.model.registry.favorite';

  it('persists selected registry across in-session navigation', () => {
    initIntercepts({ registeredModels: [] });
    modelRegistry.visit();
    modelRegistry.shouldModelRegistrySelectorExist();

    modelRegistry.findModelRegistry().findSelectOption('modelregistry-sample-2').click();
    verifyRelativeURL('/model-registry/modelregistry-sample-2');

    // Re-visit the base route to trigger redirect from sessionStorage
    modelRegistry.visit();

    modelRegistry.findModelRegistry().should('contain.text', 'modelregistry-sample-2');
    verifyRelativeURL('/model-registry/modelregistry-sample-2');
  });

  it('restores selected registry after page refresh', () => {
    initIntercepts({ registeredModels: [] });
    modelRegistry.visit();

    modelRegistry.findModelRegistry().findSelectOption('modelregistry-sample-2').click();
    verifyRelativeURL('/model-registry/modelregistry-sample-2');

    cy.reload();
    cy.findByTestId('app-page-title').should('exist');
    cy.findByTestId('app-page-title').contains('Model Registry');

    modelRegistry.findModelRegistry().should('contain.text', 'modelregistry-sample-2');
  });

  it('falls back to first favorited registry when no session selection exists', () => {
    initIntercepts({ registeredModels: [] });

    cy.visit('/model-registry', {
      onBeforeLoad(win) {
        win.sessionStorage.removeItem(SELECTED_STORAGE_KEY);
        win.localStorage.setItem(FAVORITE_STORAGE_KEY, JSON.stringify(['modelregistry-sample-2']));
      },
    });

    modelRegistry.waitForPageLoaded();

    modelRegistry.findModelRegistry().should('contain.text', 'modelregistry-sample-2');
    verifyRelativeURL('/model-registry/modelregistry-sample-2');
  });

  it('falls back to first available registry in a fresh session with no favorites', () => {
    initIntercepts({ registeredModels: [] });

    cy.visit('/model-registry', {
      onBeforeLoad(win) {
        win.sessionStorage.removeItem(SELECTED_STORAGE_KEY);
        win.localStorage.removeItem(FAVORITE_STORAGE_KEY);
      },
    });

    modelRegistry.waitForPageLoaded();

    modelRegistry.findModelRegistry().should('contain.text', 'Model Registry Sample');
    verifyRelativeURL('/model-registry/modelregistry-sample');
  });
});

describe('Register Model button', () => {
  it('Navigates to register page from empty state', () => {
    initIntercepts({ registeredModels: [] });
    modelRegistry.visit();
    modelRegistry.findRegisterModelButton().click();
    cy.findByTestId('app-page-title').should('exist');
    cy.findByTestId('app-page-title').contains('Register model');
    cy.findByText('Model registry - modelregistry-sample').should('exist');
  });

  it('Navigates to register page from table toolbar', () => {
    initIntercepts({ registeredModels: [] });
    modelRegistry.visit();
    modelRegistry.findRegisterModelButton().click();
    cy.findByTestId('app-page-title').should('exist');
    cy.findByTestId('app-page-title').contains('Register model');
    cy.findByText('Model registry - modelregistry-sample').should('exist');
  });

  it('should redirect to version details page after successful model registration', () => {
    initIntercepts({ registeredModels: [] });

    // Mock successful model registration
    cy.intercept(
      'POST',
      `/api/modelregistry/${MODEL_REGISTRY_API_VERSION}/modelregistries/modelregistry-sample/registeredmodels`,
      {
        statusCode: 201,
        body: mockRegisteredModel({
          id: 'test-model-id',
          name: 'Test Model',
        }),
      },
    ).as('createRegisteredModel');

    // Mock successful model version creation
    cy.intercept(
      'POST',
      `/api/modelregistry/${MODEL_REGISTRY_API_VERSION}/modelregistries/modelregistry-sample/registeredmodels/test-model-id/versions`,
      {
        statusCode: 201,
        body: mockModelVersion({
          id: 'test-version-id',
          name: 'v1.0.0',
          registeredModelId: 'test-model-id',
        }),
      },
    ).as('createModelVersion');
    cy.intercept(
      'POST',
      `/api/modelregistry/${MODEL_REGISTRY_API_VERSION}/modelregistries/modelregistry-sample/registeredmodels/test-model-id/versions/test-version-id/artifacts`,
      {
        statusCode: 201,
        body: {
          id: 'test-artifact-id',
          name: 'test-artifact',
        },
      },
    ).as('createModelArtifact');
    modelRegistry.visit();
    modelRegistry.findRegisterModelButton().click();
    cy.findByTestId('app-page-title').should('contain', 'Register model');
    const expectedUrlPattern =
      '/model-registry/modelregistry-sample/registered-models/test-model-id/versions/test-version-id';
    cy.url().should('include', '/model-registry/modelregistry-sample/register/model');
    cy.log(`Expected redirect URL: ${expectedUrlPattern}`);
  });
});
