/* eslint-disable camelcase */
import { mockModArchResponse } from 'mod-arch-core';
import { mockModelRegistry } from '~/__mocks__/mockModelRegistry';
import { mockRegisteredModel } from '~/__mocks__/mockRegisteredModel';
import { mockModelVersionList } from '~/__mocks__/mockModelVersionList';
import { mockModelVersion } from '~/__mocks__/mockModelVersion';
import { ModelRegistryMetadataType, ModelState, type ModelRegistry } from '~/app/types';
import { MODEL_REGISTRY_API_VERSION } from '~/__tests__/cypress/cypress/support/commands/api';
import { modelVersionsCard } from '~/__tests__/cypress/cypress/pages/modelRegistryView/modelVersionsCard';
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
  state: ModelState.LIVE,
});

const mockArchivedRegisteredModelWithData = mockRegisteredModel({
  id: '2',
  name: 'Test Archived Model',
  description: 'Test archived model description',
  owner: 'test-archived-owner',
  state: ModelState.ARCHIVED,
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
      string_value: '',
    },
    property2: {
      metadataType: ModelRegistryMetadataType.STRING,
      string_value: '',
    },
    'url-property': {
      metadataType: ModelRegistryMetadataType.STRING,
      string_value: 'https://example.com',
    },
  },
});

const mockModelVersionListWithData = mockModelVersionList({
  items: [
    mockModelVersion({
      id: '1',
      name: 'Version 1',
      author: 'Author 1',
      registeredModelId: '1',
      createTimeSinceEpoch: '1725282249921',
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
          string_value: '',
        },
        property2: {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: '',
        },
      },
      state: ModelState.LIVE,
    }),
    mockModelVersion({
      id: '2',
      name: 'Version 2',
      author: 'Author 2',
      createTimeSinceEpoch: '1725282249920',
      registeredModelId: '1',
      state: ModelState.LIVE,
    }),
    mockModelVersion({
      id: '3',
      name: 'Version 3',
      author: 'Author 3',
      createTimeSinceEpoch: '1725282249925',
      registeredModelId: '1',
      state: ModelState.LIVE,
    }),
    mockModelVersion({
      id: '4',
      name: 'Version 4',
      author: 'Author 4',
      registeredModelId: '1',
      createTimeSinceEpoch: '1725282349921',
      state: ModelState.LIVE,
    }),
    mockModelVersion({
      id: '5',
      name: 'Version 5',
      author: 'Author 5',
      registeredModelId: '2',
      createTimeSinceEpoch: '1725282349921',
      state: ModelState.ARCHIVED,
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
          string_value: '',
        },
        property2: {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: '',
        },
      },
    }),
    mockModelVersion({
      id: '6',
      name: 'Version 6',
      author: 'Author 6',
      registeredModelId: '2',
      createTimeSinceEpoch: '1725282348921',
      state: ModelState.ARCHIVED,
    }),
    mockModelVersion({
      id: '7',
      name: 'Version 7',
      author: 'Author 7',
      registeredModelId: '2',
      createTimeSinceEpoch: '1725282359921',
      state: ModelState.ARCHIVED,
    }),
    mockModelVersion({
      id: '8',
      name: 'Version 8',
      author: 'Author 8',
      registeredModelId: '2',
      createTimeSinceEpoch: '1725282349925',
      state: ModelState.LIVE,
    }),
  ],
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
    `GET /api/:apiVersion/model_registry/:modelRegistryName/registered_models/:registeredModelId`,
    {
      path: {
        modelRegistryName: 'modelregistry-sample',
        apiVersion: MODEL_REGISTRY_API_VERSION,
        registeredModelId: 2,
      },
    },
    mockArchivedRegisteredModelWithData,
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
    {
      ...mockModelVersionListWithData,
      items: mockModelVersionListWithData.items.filter((mv) => mv.registeredModelId === '1'),
    },
  );

  cy.interceptApi(
    `GET /api/:apiVersion/model_registry/:modelRegistryName/registered_models/:registeredModelId/versions`,
    {
      path: {
        modelRegistryName: 'modelregistry-sample',
        apiVersion: MODEL_REGISTRY_API_VERSION,
        registeredModelId: 2,
      },
    },
    {
      ...mockModelVersionListWithData,
      items: mockModelVersionListWithData.items.filter((mv) => mv.registeredModelId === '2'),
    },
  );
};

const initInterceptsForVersion = (modelVersionId: string) => {
  cy.interceptApi(
    `GET /api/:apiVersion/model_registry/:modelRegistryName/model_versions/:modelVersionId`,
    {
      path: {
        modelRegistryName: 'modelregistry-sample',
        apiVersion: MODEL_REGISTRY_API_VERSION,
        modelVersionId: Number(modelVersionId),
      },
    },
    mockModelVersionListWithData.items.find((mv) => mv.id === modelVersionId),
  );
};

describe('Model Versions Card', () => {
  beforeEach(() => {
    initIntercepts({});
    mockModArchResponse({});
  });

  it('does not display model versions list if there are no live model versions', () => {
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
        items: [],
      }),
    );
    cy.visit('/model-registry/modelregistry-sample/registered-models/1/overview');
    appChrome.waitForA11y();

    modelVersionsCard.findNoVersionsText().should('be.visible');
  });

  it('does not display model versions list if there are no archived model versions', () => {
    cy.interceptApi(
      `GET /api/:apiVersion/model_registry/:modelRegistryName/registered_models/:registeredModelId/versions`,
      {
        path: {
          modelRegistryName: 'modelregistry-sample',
          apiVersion: MODEL_REGISTRY_API_VERSION,
          registeredModelId: 2,
        },
      },
      mockModelVersionList({
        items: [],
      }),
    );
    cy.visit('/model-registry/modelregistry-sample/registered-models/archive/2/overview');
    appChrome.waitForA11y();

    modelVersionsCard.findNoVersionsText().should('be.visible');
  });

  it('should display live model versions list correctly', () => {
    cy.visit('/model-registry/modelregistry-sample/registered-models/1/overview');
    appChrome.waitForA11y();

    modelVersionsCard.findModelVersion('1').should('exist');

    modelVersionsCard.findModelVersionProperty('1', 'label1').should('exist');
    modelVersionsCard.findModelVersionProperty('1', 'label2').should('exist');
    modelVersionsCard.findModelVersionProperty('1', 'property1').should('exist');
    modelVersionsCard.findModelVersionLink('1').should('exist');

    modelVersionsCard.findModelVersion('2').should('not.exist');
    modelVersionsCard.findModelVersion('3').should('exist');
    modelVersionsCard.findModelVersion('4').should('exist');

    modelVersionsCard.findViewAllVersionsLink().click();
    cy.url().should('include', '/model-registry/modelregistry-sample/registered-models/1/versions');
  });

  it('should display archived model versions list correctly', () => {
    cy.visit('/model-registry/modelregistry-sample/registered-models/archive/2/overview');
    appChrome.waitForA11y();

    modelVersionsCard.findModelVersion('5').should('exist');

    modelVersionsCard.findModelVersionProperty('5', 'label1').should('exist');
    modelVersionsCard.findModelVersionProperty('5', 'label2').should('exist');
    modelVersionsCard.findModelVersionProperty('5', 'property1').should('exist');
    modelVersionsCard.findModelVersionLink('5').should('exist');

    modelVersionsCard.findModelVersion('6').should('not.exist');
    modelVersionsCard.findModelVersion('7').should('exist');
    modelVersionsCard.findModelVersion('8').should('exist');

    modelVersionsCard.findViewAllVersionsLink().click();
    cy.url().should(
      'include',
      '/model-registry/modelregistry-sample/registered-models/archive/2/versions',
    );
  });

  it('should have the correct link to the live model version', () => {
    initInterceptsForVersion('4');
    cy.visit('/model-registry/modelregistry-sample/registered-models/1/overview');
    appChrome.waitForA11y();

    modelVersionsCard.findModelVersionLink('4').click();
    cy.url().should(
      'include',
      '/model-registry/modelregistry-sample/registered-models/1/versions/4/details',
    );
    cy.contains('Version 4').should('be.visible');
  });

  it('should have the correct link to the archived model version', () => {
    initInterceptsForVersion('8');
    cy.visit('/model-registry/modelregistry-sample/registered-models/archive/2/overview');
    appChrome.waitForA11y();

    modelVersionsCard.findModelVersionLink('8').click();
    cy.url().should(
      'include',
      '/model-registry/modelregistry-sample/registered-models/archive/2/versions/8/details',
    );
    cy.contains('Version 8').should('be.visible');
  });
});
