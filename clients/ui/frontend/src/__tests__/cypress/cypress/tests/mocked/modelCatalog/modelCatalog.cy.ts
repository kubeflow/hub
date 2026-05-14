/* eslint-disable camelcase */
import { mockModArchResponse } from 'mod-arch-core';
import { modelCatalog } from '~/__tests__/cypress/cypress/pages/modelCatalog';
import {
  mockCatalogAccuracyMetricsArtifact,
  mockCatalogLabelList,
  mockCatalogModel,
  mockCatalogModelArtifact,
  mockCatalogModelList,
  mockCatalogPerformanceMetricsArtifact,
  mockCatalogSource,
  mockCatalogSourceList,
  mockDefaultSources,
  mockProviderAndCustomSources,
  mockTwoProviderSources,
} from '~/__mocks__';
import { MODEL_CATALOG_API_VERSION } from '~/__tests__/cypress/cypress/support/commands/api';
import { mockCatalogFilterOptionsList } from '~/__mocks__/mockCatalogFilterOptionsList';
import { SourceLabel, type CatalogSource } from '~/app/modelCatalogTypes';
import { ModelRegistryMetadataType } from '~/app/types';
import { TempDevFeature } from '~/app/hooks/useTempDevFeatureAvailable';
import { ModelCatalogStringFilterKey } from '~/concepts/modelCatalog/const';

type FilteredModelsInterceptConfig = {
  returnModelsForFilters?: boolean;
  modelsToReturn?: ReturnType<typeof mockCatalogModel>[];
};

const setupFilteredModelsIntercept = ({
  returnModelsForFilters = false,
  modelsToReturn = [],
}: FilteredModelsInterceptConfig = {}) => {
  cy.intercept(
    {
      method: 'GET',
      url: '**/model_catalog/models*filterQuery=*',
    },
    (req) => {
      const items = returnModelsForFilters ? modelsToReturn : [];
      req.reply(mockModArchResponse(mockCatalogModelList({ items })));
    },
  ).as('getFilteredModels');
};

const generateMockModels = (
  count: number,
  hasValidated: boolean,
  namePrefix: string,
  sourceId?: string,
): ReturnType<typeof mockCatalogModel>[] => {
  if (hasValidated) {
    return [
      mockCatalogModel({
        name: 'validated-model',
        source_id: sourceId,
        customProperties: {
          validated: { metadataType: ModelRegistryMetadataType.STRING, string_value: '' },
        },
      }),
      ...Array.from({ length: count - 1 }, (_, i) =>
        mockCatalogModel({ name: `${namePrefix}-${i + 1}`, source_id: sourceId }),
      ),
    ];
  }
  return Array.from({ length: count }, (_, i) =>
    mockCatalogModel({ name: `${namePrefix}-${i + 1}`, source_id: sourceId }),
  );
};

type HandlersProps = {
  sources?: CatalogSource[];
  modelsPerCategory?: number;
  hasValidatedModels?: boolean;
  includeAllModelsIntercept?: boolean;
};

const calculateExpectedCategoryCount = (sources: CatalogSource[]): number => {
  const uniqueLabels = new Set<string>();
  sources.forEach((source) => {
    source.labels.forEach((label) => {
      if (label.trim()) {
        uniqueLabels.add(label.trim());
      }
    });
  });

  const hasSourcesWithoutLabels = sources.some(
    (source) =>
      source.enabled !== false &&
      (source.labels.length === 0 || source.labels.every((label) => !label.trim())),
  );

  return uniqueLabels.size + (hasSourcesWithoutLabels ? 1 : 0);
};

const initIntercepts = ({
  sources = mockDefaultSources(),
  modelsPerCategory = 4,
  hasValidatedModels = false,
  includeAllModelsIntercept = true,
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

  cy.intercept(
    {
      method: 'GET',
      url: new RegExp(`/api/${MODEL_CATALOG_API_VERSION}/model_catalog/labels`),
    },
    mockModArchResponse(mockCatalogLabelList()),
  );

  sources.forEach((source) => {
    source.labels.forEach((label) => {
      const models = generateMockModels(
        modelsPerCategory,
        hasValidatedModels,
        `${label.toLowerCase().replace(/\s+/g, '-')}-model`,
        source.id,
      );

      cy.interceptApi(
        `GET /api/:apiVersion/model_catalog/models`,
        {
          path: { apiVersion: MODEL_CATALOG_API_VERSION },
          query: {
            sourceLabel: label,
          },
        },
        mockCatalogModelList({ items: models }),
      );
    });
  });

  // Intercept requests for sources without labels if they exist
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
        items: Array.from({ length: modelsPerCategory }, (_, i) =>
          mockCatalogModel({
            name: `custom-model-${i + 1}`,
            source_id: sources.find((s) => s.labels.length === 0)?.id || 'custom-source',
          }),
        ),
      }),
    );
  }
  if (includeAllModelsIntercept) {
    cy.intercept(
      {
        method: 'GET',
        url: new RegExp(
          `/api/${MODEL_CATALOG_API_VERSION}/model_catalog/models\\?(?!.*sourceLabel=)`,
        ),
      },
      (req) => {
        const models = generateMockModels(modelsPerCategory, hasValidatedModels, 'all-model');
        req.reply(mockModArchResponse(mockCatalogModelList({ items: models })));
      },
    );
  }

  cy.interceptApi(
    `GET /api/:apiVersion/model_catalog/models/filter_options`,
    {
      path: { apiVersion: MODEL_CATALOG_API_VERSION },
      query: { namespace: 'kubeflow' },
    },
    mockCatalogFilterOptionsList(),
  );

  cy.intercept(
    {
      method: 'GET',
      url: new RegExp(
        `/api/${MODEL_CATALOG_API_VERSION}/model_catalog/sources/.*/performance_artifacts/.*`,
      ),
    },
    { items: [], size: 0, pageSize: 10, nextPageToken: '' },
  );

  cy.interceptApi(
    `GET /api/:apiVersion/model_catalog/sources/:sourceId/artifacts/:modelName`,
    {
      path: {
        apiVersion: MODEL_CATALOG_API_VERSION,
        sourceId: 'sample-source',
        modelName: 'repo1/model1',
      },
    },
    {
      items: [
        mockCatalogPerformanceMetricsArtifact({}),
        mockCatalogAccuracyMetricsArtifact({}),
        mockCatalogModelArtifact({}),
      ],
    },
  );
};

describe('Model Catalog Page', () => {
  it('model catalog tab should be enabled', () => {
    initIntercepts({});
    modelCatalog.visit();
    modelCatalog.tabEnabled();
  });

  it('should show empty state when configmap has empty sources', () => {
    initIntercepts({ sources: [] });
    modelCatalog.visit();
    modelCatalog.findModelCatalogEmptyState().should('exist');
  });

  it('should display model catalog content when data is loaded', () => {
    initIntercepts({});
    modelCatalog.visit();
    modelCatalog.findLoadingState().should('not.exist');
    modelCatalog.findPageTitle().should('be.visible');
    modelCatalog.findPageDescription().should('be.visible');
    modelCatalog.findModelCatalogCards().should('have.length.at.least', 1);
  });

  it('should display model catalog filters', () => {
    initIntercepts({});
    modelCatalog.visit();
    modelCatalog.findFilter('Provider').should('be.visible');
    modelCatalog.findFilter('License').should('be.visible');
    modelCatalog.findFilter('Task').should('be.visible');
    modelCatalog.findFilter('Language').should('be.visible');
    modelCatalog.findFilter('Tensor type').scrollIntoView().should('be.visible');
  });

  it('filters show more and show less button should work', () => {
    initIntercepts({});
    modelCatalog.visit();
    modelCatalog.findFilterShowMoreButton('Task').click({ scrollBehavior: false });
    modelCatalog.findFilterCheckbox('Task', 'text-generation').should('be.visible');
    modelCatalog.findFilterCheckbox('Task', 'text-to-text').should('be.visible');
    modelCatalog.findFilterCheckbox('Task', 'image-to-text').should('be.visible');
    modelCatalog.findFilterCheckbox('Task', 'image-text-to-text').should('be.visible');
    modelCatalog.findFilterCheckbox('Task', 'audio-to-text').should('be.visible');
    modelCatalog.findFilterCheckbox('Task', 'video-to-text').should('be.visible');
    modelCatalog.findFilterShowLessButton('Task').click({ scrollBehavior: false });
    modelCatalog.findFilterCheckbox('Task', 'audio-to-text').should('not.exist');
  });

  it('filters should be searchable', () => {
    initIntercepts({});
    modelCatalog.visit();
    modelCatalog.findFilterSearch('Task').type('audio-to-text');
    modelCatalog.findFilterCheckbox('Task', 'audio-to-text').should('be.visible');
    modelCatalog.findFilterCheckbox('Task', 'video-to-text').should('not.be.exist');
    modelCatalog.findFilterSearch('Task').type('test');
    modelCatalog.findFilterEmpty('Task').should('be.visible');
  });

  it('checkbox should work', () => {
    const expectedCategoryCount = calculateExpectedCategoryCount(mockDefaultSources());

    initIntercepts({ sources: mockDefaultSources(), includeAllModelsIntercept: false });

    setupFilteredModelsIntercept({
      returnModelsForFilters: true,
      modelsToReturn: [mockCatalogModel({})],
    });

    modelCatalog.visit();
    modelCatalog.findFilterShowMoreButton('Task').click();
    modelCatalog.findFilterCheckbox('Task', 'text-generation').click();
    modelCatalog.findFilterCheckbox('Task', 'text-to-text').click();
    modelCatalog.findFilterCheckbox('Provider', 'Google').click();

    // Wait for the expected number of API calls (one per category section when filters are applied)
    const waitCalls = Array.from({ length: expectedCategoryCount }, () => '@getFilteredModels');
    cy.wait(waitCalls).then((interceptions) => {
      const lastInterception = interceptions[interceptions.length - 1];
      expect(lastInterception.request.url).to.include(
        'tasks+IN+%28%27text-generation%27%2C%27text-to-text%27%29+AND+provider%3D%27Google%27',
      );
    });
  });

  it('tensor type filter checkbox should work', () => {
    initIntercepts({ includeAllModelsIntercept: true });

    setupFilteredModelsIntercept({
      returnModelsForFilters: true,
      modelsToReturn: [mockCatalogModel({})],
    });

    modelCatalog.visit();
    modelCatalog.findFilterCheckbox('Tensor type', 'FP16').click();
    cy.wait('@getFilteredModels');

    modelCatalog.findFilterCheckbox('Tensor type', 'INT8').click();

    cy.wait('@getFilteredModels').then((interception) => {
      expect(interception.request.url).to.include(
        'tensor_type.string_value+IN+%28%27FP16%27%2C%27INT8%27%29',
      );
    });
  });

  it('tensor type filter combined with other filters should work', () => {
    const expectedCategoryCount = calculateExpectedCategoryCount(mockDefaultSources());

    initIntercepts({ sources: mockDefaultSources(), includeAllModelsIntercept: false });

    setupFilteredModelsIntercept({
      returnModelsForFilters: true,
      modelsToReturn: [mockCatalogModel({})],
    });

    modelCatalog.visit();
    modelCatalog.findFilterShowMoreButton('Task').click();
    modelCatalog.findFilterCheckbox('Task', 'text-generation').click();
    modelCatalog.findFilterCheckbox('Tensor type', 'FP16').click();
    modelCatalog.findFilterCheckbox('Provider', 'Google').click();

    const waitCalls = Array.from({ length: expectedCategoryCount }, () => '@getFilteredModels');
    cy.wait(waitCalls).then((interceptions) => {
      const lastInterception = interceptions[interceptions.length - 1];
      const { url } = lastInterception.request;
      expect(url).to.include('tasks%3D%27text-generation%27');
      expect(url).to.include('tensor_type.string_value%3D%27FP16%27');
      expect(url).to.include('provider%3D%27Google%27');
    });
  });

  it('should not display validated configuration filter when feature flag is off', () => {
    initIntercepts({});
    modelCatalog.visit();
    modelCatalog.findFilter('Validated configuration').should('not.exist');
  });

  describe('Validated Configuration Filter (feature flag on)', () => {
    beforeEach(() => {
      window.localStorage.setItem(TempDevFeature.ToolCallingConfiguration, 'true');
    });

    afterEach(() => {
      window.localStorage.removeItem(TempDevFeature.ToolCallingConfiguration);
    });

    it('checkbox should filter models', () => {
      initIntercepts({ includeAllModelsIntercept: true });
      setupFilteredModelsIntercept({
        returnModelsForFilters: true,
        modelsToReturn: [mockCatalogModel({})],
      });

      modelCatalog.visit();
      modelCatalog
        .findFilterCheckbox('Validated configuration', 'tool-calling')
        .scrollIntoView()
        .click();

      cy.wait('@getFilteredModels').then((interception) => {
        expect(interception.request.url).to.include('validatedTasks%3D%27tool-calling%27');
      });
    });

    it('should work combined with other filters', () => {
      initIntercepts({ includeAllModelsIntercept: true });
      setupFilteredModelsIntercept({
        returnModelsForFilters: true,
        modelsToReturn: [mockCatalogModel({})],
      });

      modelCatalog.visit();
      modelCatalog
        .findFilterCheckbox('Validated configuration', 'tool-calling')
        .scrollIntoView()
        .click();
      cy.wait('@getFilteredModels');

      modelCatalog.findFilterCheckbox('Provider', 'Google').click();

      cy.wait('@getFilteredModels').then((interception) => {
        const { url } = interception.request;
        expect(url).to.include('validatedTasks%3D%27tool-calling%27');
        expect(url).to.include('provider%3D%27Google%27');
      });
    });

    it('should not show helper text when only one option exists', () => {
      initIntercepts({});
      modelCatalog.visit();
      modelCatalog
        .findFilterCheckbox('Validated configuration', 'tool-calling')
        .scrollIntoView()
        .click();
      cy.contains('Showing models with all selected configurations').should('not.exist');
    });

    describe('with multiple options', () => {
      const multiOptionFilterOptions = mockCatalogFilterOptionsList({
        filters: {
          ...mockCatalogFilterOptionsList().filters,
          [ModelCatalogStringFilterKey.VALIDATED_CONFIGURATION]: {
            type: 'string',
            values: ['tool-calling', 'text-generation', 'question-answering'],
          },
        },
      });

      const initMultiOptionIntercepts = (props: HandlersProps) => {
        initIntercepts(props);
        cy.interceptApi(
          `GET /api/:apiVersion/model_catalog/models/filter_options`,
          {
            path: { apiVersion: MODEL_CATALOG_API_VERSION },
            query: { namespace: 'kubeflow' },
          },
          multiOptionFilterOptions,
        );
      };

      it('should show helper text when an option is selected', () => {
        initMultiOptionIntercepts({});
        modelCatalog.visit();
        modelCatalog
          .findFilterCheckbox('Validated configuration', 'tool-calling')
          .scrollIntoView()
          .click();
        cy.contains('Showing models with all selected configurations').should('be.visible');
      });

      it('should send AND conditions instead of IN for multiple selections', () => {
        initMultiOptionIntercepts({ includeAllModelsIntercept: true });
        setupFilteredModelsIntercept({
          returnModelsForFilters: true,
          modelsToReturn: [mockCatalogModel({})],
        });

        modelCatalog.visit();
        modelCatalog
          .findFilterCheckbox('Validated configuration', 'tool-calling')
          .scrollIntoView()
          .click();
        cy.wait('@getFilteredModels');

        modelCatalog.findFilterCheckbox('Validated configuration', 'text-generation').click();

        cy.wait('@getFilteredModels').then((interception) => {
          const { url } = interception.request;
          expect(url).to.include('validatedTasks%3D%27tool-calling%27');
          expect(url).to.include('AND');
          expect(url).to.include('validatedTasks%3D%27text-generation%27');
          expect(url).to.not.include('IN');
        });
      });
    });
  });
});

describe('Performance Empty State', () => {
  describe('Community & Custom Section', () => {
    it('should show performance empty state when toggle is ON', () => {
      initIntercepts({
        sources: mockProviderAndCustomSources(),
        hasValidatedModels: true,
      });
      setupFilteredModelsIntercept({ returnModelsForFilters: false });
      modelCatalog.visit();

      modelCatalog.togglePerformanceView();
      modelCatalog.findPerformanceViewToggleValue().should('be.checked');

      modelCatalog.findCategoryToggle('no-labels').click();

      modelCatalog
        .findPerformanceEmptyState()
        .should('be.visible')
        .and('contain.text', 'No performance data available in selected category');
      modelCatalog.findModelCatalogCards().should('not.exist');
    });

    it('should show models when toggle is OFF', () => {
      initIntercepts({
        sources: mockProviderAndCustomSources(),
      });
      modelCatalog.visit();

      modelCatalog.findCategoryToggle('no-labels').click();

      modelCatalog.findModelCatalogCards().should('have.length.at.least', 1);
      modelCatalog.findPerformanceEmptyState().should('not.exist');
    });
  });

  describe('Labeled Section Without Validated Models', () => {
    it('should show performance empty state when toggle is ON and no validated models', () => {
      initIntercepts({
        sources: mockTwoProviderSources(),
        hasValidatedModels: false,
      });
      // No user filters/search; this scenario should hit the special "No performance data" state.
      setupFilteredModelsIntercept({ returnModelsForFilters: false });
      modelCatalog.visit();

      modelCatalog.togglePerformanceView();
      modelCatalog.findPerformanceViewToggleValue().should('be.checked');
      modelCatalog.findCategoryToggle('label-Provider one').click();

      modelCatalog.findPerformanceEmptyState().should('be.visible');
      modelCatalog.findModelCatalogEmptyState().should('not.exist');
    });

    it('should show models when toggle is OFF', () => {
      initIntercepts({
        sources: mockTwoProviderSources(),
        hasValidatedModels: false,
      });
      modelCatalog.visit();

      modelCatalog.findCategoryToggle('label-Provider one').click();

      modelCatalog.findModelCatalogCards().should('have.length.at.least', 1);
    });
  });

  describe('Labeled Section With Validated Models', () => {
    it('should show models when toggle is ON and section has validated models', () => {
      initIntercepts({
        sources: mockTwoProviderSources(),
        hasValidatedModels: true,
      });
      modelCatalog.visit();

      modelCatalog.togglePerformanceView();
      modelCatalog.findCategoryToggle('label-Provider one').click();

      modelCatalog.findModelCatalogCards().should('have.length.at.least', 1);
      modelCatalog.findPerformanceEmptyState().should('not.exist');
    });
  });

  describe('Empty State Actions', () => {
    it('should turn off toggle when clicking "Turn Model performance view off"', () => {
      initIntercepts({
        sources: mockProviderAndCustomSources(),
        hasValidatedModels: true,
      });
      setupFilteredModelsIntercept({ returnModelsForFilters: false });
      modelCatalog.visit();

      modelCatalog.togglePerformanceView();
      modelCatalog.findCategoryToggle('no-labels').click();
      modelCatalog.findPerformanceEmptyState().should('be.visible');

      modelCatalog.findSetPerformanceOffLink().click();

      modelCatalog.findPerformanceViewToggleValue().should('not.be.checked');
      modelCatalog.findModelCatalogCards().should('have.length.at.least', 1);
    });

    it('should navigate to All models when clicking "View all models with performance data"', () => {
      initIntercepts({
        sources: mockProviderAndCustomSources(),
        hasValidatedModels: true,
      });
      setupFilteredModelsIntercept({ returnModelsForFilters: false });
      modelCatalog.visit();

      modelCatalog.togglePerformanceView();
      modelCatalog.findCategoryToggle('no-labels').click();
      modelCatalog.findPerformanceEmptyState().should('be.visible');

      modelCatalog.findSelectAllModelsCategoryButton().click();

      modelCatalog.findAllModelsToggle().find('button').should('have.attr', 'aria-pressed', 'true');
    });

    it('should show performance empty state after clicking Reset filters when toggle is ON', () => {
      initIntercepts({
        sources: mockTwoProviderSources(),
        hasValidatedModels: false,
      });
      setupFilteredModelsIntercept({ returnModelsForFilters: false });
      modelCatalog.visit();

      modelCatalog.togglePerformanceView();
      modelCatalog.findCategoryToggle('label-Provider one').click();
      modelCatalog.findPerformanceEmptyState().should('be.visible');

      modelCatalog.findFilterShowMoreButton('Task').click();
      modelCatalog.findFilterCheckbox('Task', 'audio-to-text').click();

      modelCatalog.findPerformanceEmptyState().should('not.exist');
      modelCatalog.findModelCatalogEmptyState().should('contain.text', 'No results found');

      modelCatalog.findEmptyStateResetFiltersButton().click();
      modelCatalog.findPerformanceEmptyState().should('be.visible');
    });
  });

  describe('Single Category Empty State Actions', () => {
    beforeEach(() => {
      initIntercepts({
        sources: [mockCatalogSource({ labels: ['Provider one'] })],
        hasValidatedModels: false,
      });
      setupFilteredModelsIntercept({ returnModelsForFilters: false });
      modelCatalog.visit();
    });

    it('should show single-category title and description when performance empty state is displayed', () => {
      modelCatalog.togglePerformanceView();
      modelCatalog.findPerformanceEmptyState().should('be.visible');

      modelCatalog
        .findPerformanceEmptyState()
        .should('contain.text', 'No performance data available')
        .and('not.contain.text', 'in selected category');

      modelCatalog
        .findPerformanceEmptyState()
        .should(
          'contain.text',
          'No models have performance data available. Turn off model performance view to see all models.',
        );
    });

    it('should not show "View all models with performance data" button when single category', () => {
      modelCatalog.togglePerformanceView();
      modelCatalog.findPerformanceEmptyState().should('be.visible');

      modelCatalog
        .findPerformanceEmptyState()
        .contains('button', /View all models with performance data/i)
        .should('not.exist');
    });

    it('should turn off toggle when clicking "Turn off model performance view" in single category', () => {
      modelCatalog.togglePerformanceView();
      modelCatalog.findPerformanceEmptyState().should('be.visible');

      modelCatalog.findSetPerformanceOffLink().click();

      modelCatalog.findPerformanceViewToggleValue().should('not.be.checked');
      modelCatalog.findModelCatalogCards().should('have.length.at.least', 1);
    });
  });

  it('should work correctly when toggling performance view', () => {
    initIntercepts({
      sources: mockProviderAndCustomSources(),
      hasValidatedModels: true,
    });
    setupFilteredModelsIntercept({ returnModelsForFilters: false });
    modelCatalog.visit();

    modelCatalog.togglePerformanceView();
    modelCatalog.findPerformanceViewToggleValue().should('be.checked');
    modelCatalog.findCategoryToggle('no-labels').click();
    modelCatalog.findPerformanceEmptyState().should('be.visible');

    modelCatalog.findSetPerformanceOffLink().click();
    modelCatalog.findPerformanceViewToggleValue().should('not.be.checked');
    modelCatalog.findPerformanceEmptyState().should('not.exist');
    modelCatalog.findModelCatalogCards().should('have.length.at.least', 1);

    modelCatalog.togglePerformanceView();
    modelCatalog.findPerformanceViewToggleValue().should('be.checked');
    modelCatalog.findPerformanceEmptyState().should('be.visible');
    modelCatalog.findModelCatalogCards().should('not.exist');
  });

  it('should show "No results found" when toggle is ON and user applies filter that returns 0 results', () => {
    initIntercepts({
      sources: mockTwoProviderSources(),
      hasValidatedModels: true,
    });
    setupFilteredModelsIntercept({ returnModelsForFilters: false });

    modelCatalog.visit();
    modelCatalog.togglePerformanceView();
    modelCatalog.findCategoryToggle('label-Provider one').click();
    modelCatalog.findFilterShowMoreButton('Task').click();
    modelCatalog.findFilterCheckbox('Task', 'audio-to-text').click();
    modelCatalog.findPerformanceEmptyState().should('not.exist');
    modelCatalog.findModelCatalogEmptyState().should('contain.text', 'No results found');
    modelCatalog.findAllModelsToggle().click();
    modelCatalog.findPerformanceEmptyState().should('not.exist');
    modelCatalog.findModelCatalogEmptyState().should('contain.text', 'No results found');
  });
});

describe('All Models Section', () => {
  it('should show models in All models section even when toggle is ON', () => {
    initIntercepts({
      sources: mockTwoProviderSources(),
      hasValidatedModels: true,
    });
    modelCatalog.visit();

    modelCatalog.togglePerformanceView();
    modelCatalog.findPerformanceViewToggleValue().should('be.checked');

    modelCatalog.findAllModelsToggle().click();
    modelCatalog.findModelCatalogCards().should('have.length.at.least', 1);
    modelCatalog.findPerformanceEmptyState().should('not.exist');
  });
});

describe('Single Category Behavior', () => {
  it('should hide category toggle when only one category is active', () => {
    initIntercepts({
      sources: [mockCatalogSource({ labels: ['Provider one'] })],
    });
    modelCatalog.visit();

    cy.findByTestId('label-Provider one').should('not.exist');
    cy.findByTestId('all').should('not.exist');
  });

  it('should auto-select the single active category and show its models', () => {
    initIntercepts({
      sources: [mockCatalogSource({ labels: ['Provider one'] })],
    });
    modelCatalog.visit();

    modelCatalog.findModelCatalogCards().should('have.length.at.least', 1);
  });

  it('should show full grid view for the auto-selected single category', () => {
    initIntercepts({
      sources: [mockCatalogSource({ labels: ['Provider one'] })],
      hasValidatedModels: true,
    });
    modelCatalog.visit();

    modelCatalog.togglePerformanceView();
    modelCatalog.findModelCatalogCards().should('have.length.at.least', 1);
  });
});
