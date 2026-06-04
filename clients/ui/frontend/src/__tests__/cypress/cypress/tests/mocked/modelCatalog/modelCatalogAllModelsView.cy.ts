import { modelCatalog } from '~/__tests__/cypress/cypress/pages/modelCatalog';
import {
  mockCatalogModel,
  mockCatalogModelList,
  mockCatalogSource,
  mockCatalogSourceList,
} from '~/__mocks__';
import type { CatalogSource } from '~/app/shared/types/catalogTypes';
import { MODEL_CATALOG_API_VERSION } from '~/__tests__/cypress/cypress/support/commands/api';
import { mockCatalogFilterOptionsList } from '~/__mocks__/mockCatalogFilterOptionsList';
import { SourceLabel } from '~/app/shared/types/catalogTypes';

type HandlersProps = {
  sources?: CatalogSource[];
  modelsPerCategory?: number;
  isEmpty?: boolean;
  includeSourcesWithoutLabels?: boolean;
};

const initIntercepts = ({
  sources = [
    mockCatalogSource({ id: 'huggingface', name: 'Hugging Face', labels: ['Hugging Face'] }),
    mockCatalogSource({ id: 'openvino', name: 'OpenVINO', labels: ['OpenVINO'] }),
    mockCatalogSource({ id: 'community', name: 'Community', labels: ['Community'] }),
    mockCatalogSource({ id: 'custom-source', name: 'Custom Source', labels: [] }),
  ],
  modelsPerCategory = 4,
  isEmpty = false,
  includeSourcesWithoutLabels = true,
}: HandlersProps) => {
  cy.interceptApi(
    `GET /api/:apiVersion/model_catalog/sources`,
    {
      path: { apiVersion: MODEL_CATALOG_API_VERSION },
    },
    mockCatalogSourceList({
      items: sources,
    }),
  );

  cy.interceptApi(
    `GET /api/:apiVersion/model_catalog/models/filter_options`,
    {
      path: { apiVersion: MODEL_CATALOG_API_VERSION },
      query: { namespace: 'kubeflow' },
    },
    mockCatalogFilterOptionsList(),
  );

  sources.forEach((source) => {
    source.labels.forEach((label) => {
      cy.interceptApi(
        `GET /api/:apiVersion/model_catalog/models`,
        {
          path: { apiVersion: MODEL_CATALOG_API_VERSION },
          query: { sourceLabel: label },
        },
        mockCatalogModelList({
          items: isEmpty
            ? []
            : Array.from({ length: modelsPerCategory }, (_, i) =>
                mockCatalogModel({
                  name: `${label.toLowerCase()}-model-${i + 1}`,
                  // eslint-disable-next-line camelcase
                  source_id: source.id,
                }),
              ),
        }),
      );
    });
  });

  // Intercept requests for sources without labels (sourceLabel=null)
  if (includeSourcesWithoutLabels) {
    const hasSourcesWithoutLabels = sources.some(
      (source) =>
        source.enabled !== false &&
        (source.labels.length === 0 || source.labels.every((label) => !label.trim())),
    );

    if (hasSourcesWithoutLabels) {
      cy.interceptApi(
        `GET /api/:apiVersion/model_catalog/models`,
        {
          path: { apiVersion: MODEL_CATALOG_API_VERSION },
          query: { sourceLabel: SourceLabel.other },
        },
        mockCatalogModelList({
          items: isEmpty
            ? []
            : Array.from({ length: modelsPerCategory }, (_, i) =>
                mockCatalogModel({
                  name: `custom-model-${i + 1}`,
                  // eslint-disable-next-line camelcase
                  source_id: sources.find((s) => s.labels.length === 0)?.id || 'custom-source',
                }),
              ),
        }),
      );
    }
  }
};

describe('Model Catalog All Models View', () => {
  beforeEach(() => {
    initIntercepts({});
    modelCatalog.visit();
  });

  describe('Category Sections', () => {
    it('should display all category sections when sources without labels exist', () => {
      modelCatalog.findAllModelsToggle().should('be.visible');
      modelCatalog.findCategoryToggle('label-Hugging Face').should('be.visible');
      modelCatalog.findCategoryToggle('label-OpenVINO').should('be.visible');
      modelCatalog.findCategoryToggle('label-Community').should('be.visible');
      modelCatalog.findCategoryToggle('no-labels').should('be.visible');
    });

    it('should hide Other models section when no sources without labels exist', () => {
      initIntercepts({
        sources: [
          mockCatalogSource({ id: 'huggingface', name: 'Hugging Face', labels: ['Hugging Face'] }),
          mockCatalogSource({ id: 'openvino', name: 'OpenVINO', labels: ['OpenVINO'] }),
          mockCatalogSource({ id: 'community', name: 'Community', labels: ['Community'] }),
        ],
        includeSourcesWithoutLabels: false,
      });
      modelCatalog.visit();

      modelCatalog.findAllModelsToggle().should('be.visible');
      modelCatalog.findCategoryToggle('label-Hugging Face').should('be.visible');
      modelCatalog.findCategoryToggle('label-OpenVINO').should('be.visible');
      modelCatalog.findCategoryToggle('label-Community').should('be.visible');
      modelCatalog.findCategoryToggle('no-labels').should('not.exist');
    });

    it('should show category titles', () => {
      modelCatalog.findCategoryTitle('OpenVINO').should('contain.text', 'OpenVINO models');
      cy.findByTestId('title Hugging Face').should('contain.text', 'Hugging Face models');
      modelCatalog.findCategoryTitle('Community').should('contain.text', 'Community models');
      modelCatalog.findCategoryTitle('null').should('contain.text', 'Other models');
    });
  });

  describe('Show More Functionality', () => {
    it('should display show more button when category has 4 or more models', () => {
      modelCatalog.findShowMoreModelsLink('hugging-face').scrollIntoView().should('be.visible');
      modelCatalog.findShowMoreModelsLink('hugging-face').click();
      modelCatalog.findAllModelsToggle().click();
      modelCatalog.findShowMoreModelsLink('openvino').scrollIntoView().should('be.visible');
      modelCatalog.findShowMoreModelsLink('openvino').click();
      modelCatalog.findAllModelsToggle().click();
      modelCatalog.findShowMoreModelsLink('community').scrollIntoView().should('be.visible');
      modelCatalog.findAllModelsToggle().click();
      modelCatalog.findShowMoreModelsLink('community').click();
    });
  });

  describe('Error Handling', () => {
    it('should display error message when category fails to load', () => {
      // Setup intercepts with sources without labels
      initIntercepts({
        sources: [
          mockCatalogSource({ id: 'huggingface', name: 'Hugging Face', labels: ['Hugging Face'] }),
          mockCatalogSource({ id: 'openvino', name: 'OpenVINO', labels: ['OpenVINO'] }),
          mockCatalogSource({ id: 'community', name: 'Community', labels: ['Community'] }),
          mockCatalogSource({ id: 'custom-source', name: 'Custom Source', labels: [] }),
        ],
        includeSourcesWithoutLabels: false, // Don't set up success intercept
      });

      // Manually intercept with error response for sourceLabel=null
      cy.intercept(
        {
          method: 'GET',
          pathname: `/model-registry/api/${MODEL_CATALOG_API_VERSION}/model_catalog/models`,
          query: { sourceLabel: SourceLabel.other },
        },
        {
          statusCode: 500,
          body: { error: 'Internal server error' },
        },
      );

      modelCatalog.visit();

      modelCatalog.findErrorState('null').scrollIntoView().should('be.visible');
      modelCatalog.findErrorState('null').should('contain.text', 'Failed to load Other models');
    });
  });

  describe('Empty States', () => {
    it('should show empty state when category has no models', () => {
      initIntercepts({ isEmpty: true });
      modelCatalog.visit();

      modelCatalog.findEmptyState('OpenVINO').scrollIntoView().should('be.visible');
      modelCatalog
        .findEmptyState('OpenVINO')
        .should('contain.text', 'No result foundAdjust your filters and try again.');
    });
  });
});
