/* eslint-disable camelcase */
import { mockModArchResponse } from 'mod-arch-core';
import { mockModelRegistry } from '~/__mocks__/mockModelRegistry';
import { mockRegisteredModel } from '~/__mocks__/mockRegisteredModel';
import { mockModelVersionList } from '~/__mocks__/mockModelVersionList';
import { mockModelVersion } from '~/__mocks__/mockModelVersion';
import { ModelRegistryMetadataType, ModelState, type ModelRegistry } from '~/app/types';
import { MODEL_REGISTRY_API_VERSION } from '~/__tests__/cypress/cypress/support/commands/api';
import { modelDetailsCard } from '~/__tests__/cypress/cypress/pages/modelRegistryView/modelDetailsCard';
import { appChrome } from '~/__tests__/cypress/cypress/pages/appChrome';

const mockRegisteredModelWithData = mockRegisteredModel({
  id: '1',
  name: 'Test Model',
  description: 'Test model description',
  owner: 'test-owner',
  customProperties: {
    label1: {
      metadataType: ModelRegistryMetadataType.STRING,
      string_value: '',
    },
    label2: {
      metadataType: ModelRegistryMetadataType.STRING,
      string_value: '',
    },
    property1: {
      metadataType: ModelRegistryMetadataType.STRING,
      string_value: 'value1',
    },
    property2: {
      metadataType: ModelRegistryMetadataType.STRING,
      string_value: 'value2',
    },
    'url-property': {
      metadataType: ModelRegistryMetadataType.STRING,
      string_value: 'https://example.com',
    },
  },
});

type HandlersProps = {
  modelRegistries?: ModelRegistry[];
};

const initIntercepts = ({
  modelRegistries = [
    mockModelRegistry({
      name: 'modelregistry-sample',
      description: 'New model registry',
      displayName: 'Model Registry Sample',
    }),
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
    `GET /api/:apiVersion/model_registry/:modelRegistryName/registered_models/:registeredModelId`,
    {
      path: {
        modelRegistryName: 'modelregistry-sample',
        apiVersion: MODEL_REGISTRY_API_VERSION,
        registeredModelId: 1,
      },
    },
    mockRegisteredModelWithData,
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
    mockModelVersionList({
      items: [mockModelVersion({ name: 'Version 1', author: 'Author 1', registeredModelId: '1' })],
    }),
  );

  cy.interceptApi(
    `PATCH /api/:apiVersion/model_registry/:modelRegistryName/registered_models/:registeredModelId`,
    {
      path: {
        modelRegistryName: 'modelregistry-sample',
        apiVersion: MODEL_REGISTRY_API_VERSION,
        registeredModelId: 1,
      },
    },
    mockRegisteredModelWithData,
  ).as('patchRegisteredModel');
};

describe('Model Details Card', () => {
  beforeEach(() => {
    initIntercepts({});
    mockModArchResponse({});
  });

  it('displays model details correctly', () => {
    cy.visit('/model-registry/modelregistry-sample/registered-models/1/overview');
    appChrome.waitForA11y();

    cy.contains('Model details').should('be.visible');

    cy.findByText('Test model description').should('be.visible');

    modelDetailsCard.findOwner().should('contain', 'test-owner');

    cy.contains('Model ID').should('be.visible');
    cy.findByTestId('registered-model-id-clipboard-copy').should('exist');

    cy.contains('Last modified').should('be.visible');
    cy.contains('Created').should('be.visible');
  });

  it('displays labels section correctly', () => {
    cy.visit('/model-registry/modelregistry-sample/registered-models/1/overview');
    appChrome.waitForA11y();

    cy.contains('Labels').should('be.visible');
    cy.contains('label1').should('be.visible');
    cy.contains('label2').should('be.visible');

    cy.contains('Labels').parent().find('button').should('exist');
  });

  it('displays properties in expandable section', () => {
    cy.visit('/model-registry/modelregistry-sample/registered-models/1/overview');
    appChrome.waitForA11y();

    cy.contains('Properties').should('be.visible');
    cy.contains('Properties').parent().find('.pf-v6-c-badge').should('contain', '3'); // property1, property2, url-property

    cy.contains('Properties').click();

    modelDetailsCard.findPropertiesTable().should('be.visible');

    cy.contains('property1').should('be.visible');
    cy.contains('value1').should('be.visible');
    cy.contains('property2').should('be.visible');
    cy.contains('value2').should('be.visible');

    cy.contains('url-property').should('be.visible');
    cy.get('a[href="https://example.com"]').should('exist');
  });

  it('shows add property button and validates input', () => {
    cy.visit('/model-registry/modelregistry-sample/registered-models/1/overview');
    appChrome.waitForA11y();

    cy.contains('Properties').click();

    modelDetailsCard.findAddPropertyButton().should('be.visible');

    modelDetailsCard.findAddPropertyButton().click();

    modelDetailsCard.findAddPropertyKeyInput().should('be.visible');
    modelDetailsCard.findAddPropertyValueInput().should('be.visible');

    modelDetailsCard.findAddPropertyKeyInput().type('property1'); // Already exists
    modelDetailsCard.findAddPropertyValueInput().type('someValue');

    cy.contains('Key must not match an existing property key or label').should('be.visible');
    modelDetailsCard.findSavePropertyButton().should('be.disabled');
  });

  it('validates property key length correctly', () => {
    cy.visit('/model-registry/modelregistry-sample/registered-models/1/overview');
    appChrome.waitForA11y();

    cy.contains('Properties').click();

    modelDetailsCard.findAddPropertyButton().click();

    modelDetailsCard.findAddPropertyKeyInput().type('a'.repeat(64)); // Too long
    modelDetailsCard.findAddPropertyValueInput().type('someValue');

    cy.contains("Key text can't exceed 63 characters").should('be.visible');
    modelDetailsCard.findSavePropertyButton().should('be.disabled');
  });

  it('handles expand/collapse for many properties', () => {
    const customPropsEntries = Array.from({ length: 10 }, (_, i) => [
      `property${i + 1}`,
      {
        metadataType: ModelRegistryMetadataType.STRING as const,
        string_value: `value${i + 1}`,
      },
    ]);
    const customProps = Object.fromEntries(customPropsEntries);

    const manyPropertiesModel = mockRegisteredModel({
      id: '1',
      customProperties: customProps,
    });

    cy.interceptApi(
      `GET /api/:apiVersion/model_registry/:modelRegistryName/registered_models/:registeredModelId`,
      {
        path: {
          modelRegistryName: 'modelregistry-sample',
          apiVersion: MODEL_REGISTRY_API_VERSION,
          registeredModelId: 1,
        },
      },
      manyPropertiesModel,
    );

    cy.visit('/model-registry/modelregistry-sample/registered-models/1/overview');
    appChrome.waitForA11y();

    cy.contains('Properties').click();

    cy.contains('property1').should('be.visible');
    cy.contains('property5').should('be.visible');
    cy.contains('property6').should('not.exist');

    modelDetailsCard.findExpandControlButton().click();

    cy.contains('property6').should('be.visible');
    cy.contains('property10').should('exist');

    modelDetailsCard.findExpandControlButton().should('contain', 'Show fewer properties');
  });

  it('handles archived model state correctly', () => {
    const archivedModel = mockRegisteredModel({
      id: '1',
      state: ModelState.ARCHIVED,
      customProperties: {
        property1: {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: 'value1',
        },
      },
    });

    cy.interceptApi(
      `GET /api/:apiVersion/model_registry/:modelRegistryName/registered_models/:registeredModelId`,
      {
        path: {
          modelRegistryName: 'modelregistry-sample',
          apiVersion: MODEL_REGISTRY_API_VERSION,
          registeredModelId: 1,
        },
      },
      archivedModel,
    );

    cy.visit('/model-registry/modelregistry-sample/registered-models/1/overview');
    appChrome.waitForA11y();

    cy.contains('Properties').click();

    modelDetailsCard.findAddPropertyButton().should('not.exist');
  });

  it('shows the correct tab structure and navigation', () => {
    cy.visit('/model-registry/modelregistry-sample/registered-models/1/overview');
    appChrome.waitForA11y();

    cy.findByTestId('model-versions-page-tabs').should('exist');
    cy.findByTestId('model-overview-tab').should('exist');

    cy.findByTestId('model-overview-tab').should('have.attr', 'aria-selected', 'true');

    cy.findByTestId('model-details-tab-content').should('be.visible');
  });

  // TODO: Pending tests for complex interactions with mod-arch-shared components
  // These tests need investigation of exact DOM structure of mod-arch-shared components

  it('allows editing model description', () => {
    cy.visit('/model-registry/modelregistry-sample/registered-models/1/overview');
    appChrome.waitForA11y();

    cy.findByText('Test model description').should('be.visible');

    modelDetailsCard.findDescriptionEditButton().click();

    modelDetailsCard.findDescriptionTextArea().should('be.visible');

    modelDetailsCard.findDescriptionTextArea().clear();
    modelDetailsCard.findDescriptionTextArea().type('Updated model description for testing');

    modelDetailsCard.findDescriptionSaveButton().click();

    cy.wait('@patchRegisteredModel').then((interception) => {
      expect(interception.request.body.data.description).to.equal(
        'Updated model description for testing',
      );
    });
  });
});
