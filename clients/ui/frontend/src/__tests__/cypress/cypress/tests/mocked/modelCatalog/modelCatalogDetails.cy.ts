/* eslint-disable camelcase */
import { modelCatalog } from '~/__tests__/cypress/cypress/pages/modelCatalog';
import { mockModelRegistry } from '~/__mocks__/mockModelRegistry';
import {
  setupModelCatalogIntercepts,
  setupValidatedModelIntercepts,
  interceptArtifactsList,
  interceptPerformanceArtifactsList,
} from '~/__tests__/cypress/cypress/support/interceptHelpers/modelCatalog';
import { mockCatalogModelArtifact, mockCatalogModel } from '~/__mocks__';
import { ModelRegistryMetadataType } from '~/app/types';
import { MODEL_CATALOG_API_VERSION } from '~/__tests__/cypress/cypress/support/commands/api';
import { TempDevFeature } from '~/app/hooks/useTempDevFeatureAvailable';
import { ModelCatalogTask } from '~/concepts/modelCatalog/const';

describe('Model Catalog Details Page', () => {
  beforeEach(() => {
    // Mock model registries for register button functionality
    cy.intercept('GET', '/model-registry/api/v1/model_registry*', [
      mockModelRegistry({ name: 'modelregistry-sample' }),
    ]).as('getModelRegistries');

    setupModelCatalogIntercepts({});
    modelCatalog.visit();
  });

  it('navigates to details and shows header, breadcrumb and description', () => {
    modelCatalog.findLoadingState().should('not.exist');
    modelCatalog.findModelCatalogDetailLink().first().click();
    modelCatalog.findBreadcrumb().should('exist');
    modelCatalog.findDetailsProviderText().should('be.visible');
    modelCatalog.findDetailsDescription().should('exist');
  });

  it('shows formatted model type in details', () => {
    modelCatalog.findLoadingState().should('not.exist');
    modelCatalog.findModelCatalogDetailLink().first().click();
    modelCatalog.findModelType().should('be.visible');
    modelCatalog.findModelType().should('contain.text', 'Generative AI model (Example, LLM)');
  });

  it('does not show architecture field when no architectures are available', () => {
    modelCatalog.findLoadingState().should('not.exist');
    modelCatalog.findModelCatalogDetailLink().first().click();
    // Architecture field should not exist when no valid architectures
    modelCatalog.findModelArchitecture().should('not.exist');
  });
});

describe('Model Catalog Details Page - Architecture Field', () => {
  beforeEach(() => {
    // Mock model registries for register button functionality
    cy.intercept('GET', '/model-registry/api/v1/model_registry*', [
      mockModelRegistry({ name: 'modelregistry-sample' }),
    ]).as('getModelRegistries');

    setupModelCatalogIntercepts({});
  });

  it('shows architecture field with valid architectures', () => {
    // Set up intercept with architecture data before navigation
    interceptArtifactsList({
      items: [
        mockCatalogModelArtifact({
          customProperties: {
            architecture: {
              string_value: '["amd64", "arm64", "s390x"]',
              metadataType: ModelRegistryMetadataType.STRING,
            },
          },
        }),
      ],
      size: 1,
      pageSize: 10,
      nextPageToken: '',
    });

    modelCatalog.visit();
    modelCatalog.findLoadingState().should('not.exist');
    modelCatalog.findModelCatalogDetailLink().first().click();
    modelCatalog.findBreadcrumb().should('exist');

    // Architecture field should exist and show correct values
    modelCatalog.findModelArchitecture().should('be.visible');
    modelCatalog.findModelArchitecture().should('contain.text', 'amd64, arm64, s390x');
  });

  it('shows architecture field with uppercase architectures normalized to lowercase', () => {
    // Set up intercept with uppercase architecture data before navigation
    interceptArtifactsList({
      items: [
        mockCatalogModelArtifact({
          customProperties: {
            architecture: {
              string_value: '["AMD64", "ARM64"]',
              metadataType: ModelRegistryMetadataType.STRING,
            },
          },
        }),
      ],
      size: 1,
      pageSize: 10,
      nextPageToken: '',
    });

    modelCatalog.visit();
    modelCatalog.findLoadingState().should('not.exist');
    modelCatalog.findModelCatalogDetailLink().first().click();

    // Architecture should be normalized to lowercase
    modelCatalog.findModelArchitecture().should('be.visible');
    modelCatalog.findModelArchitecture().should('contain.text', 'amd64, arm64');
  });

  it('shows architecture field with custom architecture values', () => {
    // Set up intercept with custom architecture data before navigation
    interceptArtifactsList({
      items: [
        mockCatalogModelArtifact({
          customProperties: {
            architecture: {
              string_value: '["custom-arch", "unknown"]',
              metadataType: ModelRegistryMetadataType.STRING,
            },
          },
        }),
      ],
      size: 1,
      pageSize: 10,
      nextPageToken: '',
    });

    modelCatalog.visit();
    modelCatalog.findLoadingState().should('not.exist');
    modelCatalog.findModelCatalogDetailLink().first().click();

    // Architecture field should display all architecture values without validation
    modelCatalog.findModelArchitecture().should('be.visible');
    modelCatalog.findModelArchitecture().should('contain.text', 'custom-arch, unknown');
  });
});

/**
 * NOTE: Performance Insights Tab Navigation, Hardware Configuration Table,
 * Workload Type Filter, and Latency Filter tests are covered in modelCatalogTabs.cy.ts.
 * This file focuses on filter state management across pages.
 */

describe('Model Catalog Details Page - Filter State Management', () => {
  beforeEach(() => {
    cy.intercept('GET', '/model-registry/api/v1/model_registry*', [
      mockModelRegistry({ name: 'modelregistry-sample' }),
    ]).as('getModelRegistries');

    // Use validated model intercepts which include performance artifacts
    setupValidatedModelIntercepts({});
    interceptArtifactsList();
    interceptPerformanceArtifactsList();
  });

  it('should persist filter state when navigating between Overview and Performance Insights tabs', () => {
    modelCatalog.visit();
    modelCatalog.findLoadingState().should('not.exist');
    modelCatalog.togglePerformanceView();
    modelCatalog.findLoadingState().should('not.exist');

    modelCatalog.findModelCatalogDetailLink().first().click();
    modelCatalog.clickPerformanceInsightsTab();

    // Change a filter
    modelCatalog.findWorkloadTypeFilter().click();
    modelCatalog.selectWorkloadType('code_fixing');

    // Navigate to Overview tab
    modelCatalog.clickOverviewTab();
    modelCatalog.findOverviewTabContent().should('be.visible');

    // Navigate back to Performance Insights
    modelCatalog.clickPerformanceInsightsTab();

    // Filter should still show the selected value (capitalized as "Code Fixing")
    modelCatalog.findWorkloadTypeFilter().should('contain.text', 'Code Fixing');
  });

  it('should sync filter changes back to catalog page', () => {
    modelCatalog.visit();
    modelCatalog.findLoadingState().should('not.exist');
    modelCatalog.togglePerformanceView();
    modelCatalog.findLoadingState().should('not.exist');

    modelCatalog.findModelCatalogDetailLink().first().click();
    modelCatalog.clickPerformanceInsightsTab();

    // Change a filter on details page
    modelCatalog.findWorkloadTypeFilter().click();
    modelCatalog.selectWorkloadType('rag');

    // Go back to catalog page
    cy.go('back');
    cy.go('back');
    modelCatalog.findLoadingState().should('not.exist');

    // The alert should show indicating filters were updated
    modelCatalog.findPerformanceFiltersUpdatedAlert().should('be.visible');
  });
});

describe('Model Catalog Details Page - Edge Cases', () => {
  beforeEach(() => {
    cy.intercept('GET', '/model-registry/api/v1/model_registry*', [
      mockModelRegistry({ name: 'modelregistry-sample' }),
    ]).as('getModelRegistries');
  });

  it('should show "No description" when model has no description', () => {
    const modelWithoutDescription = mockCatalogModel({
      name: 'no-description-model',
      description: undefined,
    });

    setupModelCatalogIntercepts({ customNonValidatedModel: modelWithoutDescription });
    modelCatalog.visit();
    modelCatalog.findLoadingState().should('not.exist');
    modelCatalog.findModelCatalogDetailLink().first().click();
    modelCatalog.findBreadcrumb().should('exist');

    modelCatalog.findDetailsDescription().should('contain.text', 'No description');
  });

  it('should show model card markdown when readme exists', () => {
    setupModelCatalogIntercepts({});
    modelCatalog.visit();
    modelCatalog.findLoadingState().should('not.exist');
    modelCatalog.findModelCatalogDetailLink().first().click();
    modelCatalog.findBreadcrumb().should('exist');

    modelCatalog.findModelCardMarkdown().should('exist');
  });

  it('should show "No model card" when model has no readme', () => {
    const modelWithoutReadme = mockCatalogModel({
      name: 'no-readme-model',
      readme: undefined,
    });

    setupModelCatalogIntercepts({ customNonValidatedModel: modelWithoutReadme });
    modelCatalog.visit();
    modelCatalog.findLoadingState().should('not.exist');
    modelCatalog.findModelCatalogDetailLink().first().click();
    modelCatalog.findBreadcrumb().should('exist');

    cy.contains('No model card').should('be.visible');
  });

  it('should show "N/A" for provider when provider is not set', () => {
    const modelWithoutProvider = mockCatalogModel({
      name: 'no-provider-model',
      provider: undefined,
    });

    setupModelCatalogIntercepts({ customNonValidatedModel: modelWithoutProvider });
    modelCatalog.visit();
    modelCatalog.findLoadingState().should('not.exist');
    modelCatalog.findModelCatalogDetailLink().first().click();
    modelCatalog.findBreadcrumb().should('exist');

    cy.findAllByText('N/A').should('have.length.at.least', 1);
  });

  it('should show error alert when artifacts fail to load', () => {
    setupModelCatalogIntercepts({});

    cy.intercept(
      {
        method: 'GET',
        url: new RegExp(
          `/model-registry/api/${MODEL_CATALOG_API_VERSION}/model_catalog/sources/.*/artifacts/.*`,
        ),
      },
      { statusCode: 500, body: { message: 'Failed to load artifacts' } },
    ).as('getArtifactsError');

    modelCatalog.visit();
    modelCatalog.findLoadingState().should('not.exist');
    modelCatalog.findModelCatalogDetailLink().first().click();
    modelCatalog.findBreadcrumb().should('exist');

    cy.wait('@getArtifactsError');
    cy.get('.pf-v6-c-alert.pf-m-danger').should('be.visible');
  });

  it('should show spinner while artifacts are loading', () => {
    setupModelCatalogIntercepts({});

    cy.intercept(
      {
        method: 'GET',
        url: new RegExp(
          `/model-registry/api/${MODEL_CATALOG_API_VERSION}/model_catalog/sources/.*/artifacts/.*`,
        ),
      },
      (req) => {
        req.on('response', (res) => {
          res.setDelay(10000);
        });
      },
    ).as('getArtifactsSlow');

    modelCatalog.visit();
    modelCatalog.findLoadingState().should('not.exist');
    modelCatalog.findModelCatalogDetailLink().first().click();
    modelCatalog.findBreadcrumb().should('exist');

    cy.findByRole('progressbar').should('exist');
  });
});

describe('Model Catalog Details Page - Validated Configurations Card', () => {
  beforeEach(() => {
    cy.intercept('GET', '/model-registry/api/v1/model_registry*', [
      mockModelRegistry({ name: 'modelregistry-sample' }),
    ]).as('getModelRegistries');
  });

  describe('with feature flag enabled', () => {
    beforeEach(() => {
      window.localStorage.setItem(TempDevFeature.ToolCallingConfiguration, 'true');
      setupValidatedModelIntercepts({});
      interceptArtifactsList();
      modelCatalog.visit();
      modelCatalog.findLoadingState().should('not.exist');
      modelCatalog.findModelCatalogDetailLink().first().click();
      modelCatalog.findBreadcrumb().should('exist');
    });

    it('should display the validated configurations card with tool calling content', () => {
      modelCatalog.findValidatedConfigurationsCard().should('be.visible');
      modelCatalog
        .findValidatedConfigurationsCard()
        .should('contain.text', 'Validated configurations');
      modelCatalog.findToolCallingCard().should('be.visible');
      modelCatalog.findToolCallingCard().should('contain.text', 'Tool Calling');
    });

    it('should show CLI args inside the tool calling card when expanded', () => {
      modelCatalog.findToolCallingToggle().click();
      modelCatalog.findToolCallingCard().should('contain.text', '--enable-auto-tool-choice');
      modelCatalog.findToolCallingCard().should('contain.text', '--tool-call-parser granite');
    });
  });

  describe('with feature flag disabled', () => {
    it('should not display the validated configurations card', () => {
      window.localStorage.removeItem(TempDevFeature.ToolCallingConfiguration);
      setupValidatedModelIntercepts({});
      interceptArtifactsList();
      modelCatalog.visit();
      modelCatalog.findLoadingState().should('not.exist');
      modelCatalog.findModelCatalogDetailLink().first().click();
      modelCatalog.findBreadcrumb().should('exist');
      modelCatalog.findValidatedConfigurationsCard().should('not.exist');
    });
  });

  describe('for a non-validated model', () => {
    it('should not display the validated configurations card', () => {
      window.localStorage.setItem(TempDevFeature.ToolCallingConfiguration, 'true');
      setupModelCatalogIntercepts({
        customNonValidatedModel: mockCatalogModel({
          name: 'non-validated-model',
          tasks: [ModelCatalogTask.TEXT_GENERATION],
        }),
      });
      modelCatalog.visit();
      modelCatalog.findLoadingState().should('not.exist');
      modelCatalog.findModelCatalogDetailLink().first().click();
      modelCatalog.findBreadcrumb().should('exist');
      modelCatalog.findValidatedConfigurationsCard().should('not.exist');
    });
  });
});
