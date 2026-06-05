import { mockModArchResponse } from 'mod-arch-core';
import {
  MOCK_VARIANT_GROUP_ID,
  mockCatalogModel,
  mockCatalogModelList,
  mockCatalogPerformanceMetricsArtifactList,
  mockCatalogSourceList,
  mockModelNoVariantGroup,
  mockVariantModel,
  mockVariantModelNoLogo,
  mockVariantModels,
} from '~/__mocks__';
import { mockCatalogFilterOptionsList } from '~/__mocks__/mockCatalogFilterOptionsList';
import { modelCatalog } from '~/__tests__/cypress/cypress/pages/modelCatalog';
import { appChrome } from '~/__tests__/cypress/cypress/pages/appChrome';
import { MODEL_CATALOG_API_VERSION } from '~/__tests__/cypress/cypress/support/commands/api';
import { ModelRegistryMetadataType } from '~/app/types';

type InterceptConfig = {
  currentModel?: (typeof mockVariantModels)[0];
  variantModels?: typeof mockVariantModels;
  loadError?: boolean;
  emptyResponse?: boolean;
};

const initIntercepts = ({
  currentModel = mockVariantModels[0],
  variantModels = mockVariantModels,
  loadError = false,
  emptyResponse = false,
}: InterceptConfig = {}) => {
  cy.interceptApi(
    'GET /api/:apiVersion/model_catalog/sources',
    { path: { apiVersion: MODEL_CATALOG_API_VERSION } },
    mockCatalogSourceList({}),
  );

  cy.interceptApi(
    'GET /api/:apiVersion/model_catalog/models/filter_options',
    { path: { apiVersion: MODEL_CATALOG_API_VERSION }, query: { namespace: 'kubeflow' } },
    mockCatalogFilterOptionsList(),
  );

  cy.interceptApi(
    'GET /api/:apiVersion/model_catalog/sources/:sourceId/models/:modelName',
    {
      path: {
        apiVersion: MODEL_CATALOG_API_VERSION,
        sourceId: 'sample-source',
        modelName: encodeURIComponent(currentModel.name),
      },
    },
    currentModel,
  );

  cy.interceptApi(
    'GET /api/:apiVersion/model_catalog/sources/:sourceId/artifacts/:modelName',
    {
      path: {
        apiVersion: MODEL_CATALOG_API_VERSION,
        sourceId: 'sample-source',
        modelName: encodeURIComponent(currentModel.name),
      },
    },
    mockCatalogPerformanceMetricsArtifactList({}),
  );

  cy.interceptApi(
    'GET /api/:apiVersion/model_catalog/sources/:sourceId/performance_artifacts/:modelName',
    {
      path: {
        apiVersion: MODEL_CATALOG_API_VERSION,
        sourceId: 'sample-source',
        modelName: encodeURIComponent(currentModel.name),
      },
    },
    mockCatalogPerformanceMetricsArtifactList({}),
  );

  if (loadError) {
    cy.intercept(
      {
        method: 'GET',
        pathname: `/model-registry/api/${MODEL_CATALOG_API_VERSION}/model_catalog/models`,
      },
      { statusCode: 500, body: { message: 'Failed to load variants' } },
    ).as('getVariants');
  } else {
    cy.intercept(
      {
        method: 'GET',
        pathname: `/model-registry/api/${MODEL_CATALOG_API_VERSION}/model_catalog/models`,
      },
      (req) => {
        if (req.query.source === 'sample-source') {
          req.reply(
            mockModArchResponse(
              mockCatalogModelList({
                items: emptyResponse ? [] : variantModels,
                size: emptyResponse ? 0 : variantModels.length,
              }),
            ),
          );
        }
      },
    ).as('getVariants');
  }
};

const visitPerformanceTab = (modelName: string) => {
  cy.visit(`/model-catalog/sample-source/${encodeURIComponent(modelName)}/performance-insights`);
  appChrome.waitForA11y();
};

describe('Compression Level Comparison Card', () => {
  describe('When model has variant_group_id', () => {
    beforeEach(() => {
      initIntercepts({});
      visitPerformanceTab(mockVariantModels[0].name);
    });

    it('should display the card with title and description', () => {
      modelCatalog.findCompressionComparisonCard().should('exist');
      modelCatalog
        .findCompressionComparisonCard()
        .should('contain', 'Model variants by tensor type');
      modelCatalog
        .findCompressionComparisonCard()
        .should(
          'contain',
          'Compare benchmark performance across tensor types to understand accuracy and efficiency tradeoffs.',
        );
    });

    it('should display exactly 4 variant models', () => {
      modelCatalog.findAllCompressionVariants().should('have.length', 4);
    });

    it('should show current model as first item', () => {
      modelCatalog.findCompressionVariant(0).should('exist');
      modelCatalog.findCompressionCurrentModelName().should('exist');
      modelCatalog.findCompressionCurrentModelName().should('contain', 'granite-8b-instruct');
    });

    it('should display "Current model" label only for current model', () => {
      modelCatalog.findCompressionCurrentLabel().should('exist');
      modelCatalog.findCompressionCurrentLabel().should('contain', 'Current model');
      modelCatalog.findAllCompressionCurrentLabels().should('have.length', 1);
    });

    it('should display tensor type labels for each variant', () => {
      modelCatalog.findCompressionTensorType(0).should('contain', 'FP16');
      modelCatalog.findCompressionTensorType(1).should('contain', 'INT4');
      modelCatalog.findCompressionTensorType(2).should('contain', 'INT8');
      modelCatalog.findCompressionTensorType(3).should('contain', 'BF16');
    });

    it('should NOT have link for current model', () => {
      modelCatalog.findCompressionVariantLink(0).should('not.exist');
    });

    it('should have links for non-current model variants', () => {
      modelCatalog.findCompressionVariantLink(1).should('exist');
      modelCatalog.findCompressionVariantLink(2).should('exist');
      modelCatalog.findCompressionVariantLink(3).should('exist');
    });

    it('should display vertical dividers between variants', () => {
      modelCatalog.findCompressionDivider(0).should('not.exist');
      modelCatalog.findCompressionDivider(1).should('exist');
      modelCatalog.findCompressionDivider(2).should('exist');
      modelCatalog.findCompressionDivider(3).should('exist');
    });

    it('should navigate to variant model Performance Insights tab when clicking link', () => {
      modelCatalog.findCompressionVariantLink(1).click();
      cy.url().should('include', 'granite-8b-int4');
      cy.url().should('include', 'performance-insights');
    });
  });

  describe('When model does NOT have variant_group_id', () => {
    beforeEach(() => {
      initIntercepts({ currentModel: mockModelNoVariantGroup });
      visitPerformanceTab(mockModelNoVariantGroup.name);
    });

    it('should NOT display the comparison card', () => {
      modelCatalog.findCompressionComparisonCard().should('not.exist');
    });
  });

  describe('Loading state', () => {
    it('should show loading spinner while fetching variants', () => {
      initIntercepts({});
      cy.intercept(
        {
          method: 'GET',
          pathname: `/model-registry/api/${MODEL_CATALOG_API_VERSION}/model_catalog/models`,
        },
        {
          delay: 2000,
          body: mockModArchResponse(
            mockCatalogModelList({
              items: mockVariantModels,
              size: mockVariantModels.length,
            }),
          ),
        },
      ).as('getVariantsDelayed');

      visitPerformanceTab(mockVariantModels[0].name);

      modelCatalog.findCompressionComparisonLoading().should('exist');
    });
  });

  describe('Error state', () => {
    beforeEach(() => {
      initIntercepts({ loadError: true });
      visitPerformanceTab(mockVariantModels[0].name);
    });

    it('should display error alert when loading fails', () => {
      modelCatalog.findCompressionComparisonError().should('exist');
      modelCatalog
        .findCompressionComparisonError()
        .should('contain', 'Error loading performance data');
    });
  });

  describe('Empty state', () => {
    beforeEach(() => {
      initIntercepts({ emptyResponse: true });
      visitPerformanceTab(mockVariantModels[0].name);
    });

    it('should NOT display the card when no variants found', () => {
      modelCatalog.findCompressionComparisonCard().should('not.exist');
    });
  });

  describe('Single variant state', () => {
    beforeEach(() => {
      // Only the current model is returned as a variant
      initIntercepts({ variantModels: [mockVariantModels[0]] });
      visitPerformanceTab(mockVariantModels[0].name);
    });

    it('should NOT display the card when there is only one variant (the current model)', () => {
      modelCatalog.findCompressionComparisonCard().should('not.exist');
    });
  });

  describe('Edge cases', () => {
    it('should show skeleton when variant has no logo', () => {
      initIntercepts({
        variantModels: [mockVariantModels[0], mockVariantModelNoLogo],
      });
      visitPerformanceTab(mockVariantModels[0].name);

      modelCatalog.findCompressionVariantLogo(0).should('exist');
      modelCatalog.findCompressionVariantSkeleton(1).should('exist');
    });

    it('should not show tensor type label when tensor_type is missing', () => {
      const modelNoTensorType1 = mockCatalogModel({
        ...mockVariantModels[1],
        name: 'repo1/no-tensor-1',
        customProperties: {
          // eslint-disable-next-line camelcase
          variant_group_id: {
            metadataType: ModelRegistryMetadataType.STRING,
            // eslint-disable-next-line camelcase
            string_value: MOCK_VARIANT_GROUP_ID,
          },
        },
      });

      const modelNoTensorType2 = mockCatalogModel({
        ...mockVariantModels[2],
        name: 'repo1/no-tensor-2',
        customProperties: {
          // eslint-disable-next-line camelcase
          variant_group_id: {
            metadataType: ModelRegistryMetadataType.STRING,
            // eslint-disable-next-line camelcase
            string_value: MOCK_VARIANT_GROUP_ID,
          },
        },
      });

      const fourModels = [
        mockVariantModels[0],
        modelNoTensorType1,
        modelNoTensorType2,
        mockVariantModels[1],
      ];

      initIntercepts({
        variantModels: fourModels,
      });
      visitPerformanceTab(mockVariantModels[0].name);

      modelCatalog.findCompressionTensorType(0).should('exist');
      modelCatalog.findCompressionTensorType(3).should('exist');
      modelCatalog.findCompressionTensorType(2).should('not.exist');
      modelCatalog.findCompressionTensorType(1).should('not.exist');
    });

    it('should limit to 4 variants even if more are returned', () => {
      const sixVariants = [
        ...mockVariantModels,
        mockVariantModel('repo1/extra-1', 'FP32'),
        mockVariantModel('repo1/extra-2', 'INT2'),
      ];

      initIntercepts({ variantModels: sixVariants });
      visitPerformanceTab(mockVariantModels[0].name);

      modelCatalog.findAllCompressionVariants().should('have.length', 4);
    });
  });
});
