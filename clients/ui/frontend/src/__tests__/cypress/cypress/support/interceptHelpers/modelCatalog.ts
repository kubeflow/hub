/* eslint-disable camelcase */
/**
 * Shared intercept helpers for Model Catalog tests.
 * These functions consolidate common API intercept patterns used across multiple test files.
 */
import { mockModArchResponse } from 'mod-arch-core';
import {
  mockCatalogLabelList,
  mockCatalogModel,
  mockCatalogModelList,
  mockCatalogPerformanceMetricsArtifact,
  mockCatalogSource,
  mockCatalogSourceList,
  mockNonValidatedModel,
  mockValidatedModel,
} from '~/__mocks__';
import { mockCatalogPerformanceMetricsArtifactList } from '~/__mocks__/mockCatalogModelArtifactList';
import { mockCatalogFilterOptionsList } from '~/__mocks__/mockCatalogFilterOptionsList';
import { mockModelRegistry } from '~/__mocks__/mockModelRegistry';
import type { CatalogLabelList, CatalogModel, CatalogSource } from '~/app/modelCatalogTypes';
import type { ModelRegistryCustomProperties } from '~/app/types';
import { ModelRegistryMetadataType } from '~/app/types';
import { MODEL_CATALOG_API_VERSION } from '~/__tests__/cypress/cypress/support/commands/api';
import { ValidatedConfiguration } from '~/concepts/modelCatalog/const';

/**
 * Options for setting up model catalog intercepts
 */
export type ModelCatalogInterceptOptions = {
  /** Catalog sources to mock. Defaults to two sample sources. */
  sources?: CatalogSource[];
  /** Number of models per category/label. Defaults to 4. */
  modelsPerCategory?: number;
  /** Whether to include a validated model (with performance data). Defaults to false. */
  useValidatedModel?: boolean;
  /** Whether to include performance artifacts. Defaults to false. */
  includePerformanceArtifacts?: boolean;
  /** Whether to include filter options. Defaults to true. */
  includeFilterOptions?: boolean;
  /** Whether to include model registry intercept. Defaults to false. */
  includeModelRegistry?: boolean;
  /** Custom validated model to use. Defaults to mockValidatedModel. */
  customValidatedModel?: CatalogModel;
  /** Custom non-validated model to use. Defaults to mockNonValidatedModel. */
  customNonValidatedModel?: CatalogModel;
};

/**
 * Default sources used when none are provided
 */
export const defaultSources = (): CatalogSource[] => [
  mockCatalogSource({}),
  mockCatalogSource({ id: 'source-2', name: 'source 2' }),
];

/**
 * Intercepts the GET /model_catalog/sources endpoint
 */
export const interceptSources = (sources: CatalogSource[]): void => {
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

/**
 * Intercepts the GET /model_catalog/labels endpoint
 */
export const interceptLabels = (labels?: Partial<CatalogLabelList>): void => {
  cy.interceptApi(
    `GET /api/:apiVersion/model_catalog/labels`,
    {
      path: { apiVersion: MODEL_CATALOG_API_VERSION },
    },
    mockCatalogLabelList(labels),
  );
};

/**
 * Creates mock models for a given label with optional validated model support
 */
export const createMockModelsForLabel = (
  source: CatalogSource,
  label: string,
  modelsPerCategory: number,
  useValidatedModel: boolean,
): ReturnType<typeof mockCatalogModelList> =>
  mockCatalogModelList({
    items: Array.from({ length: modelsPerCategory }, (_, i) => {
      const isValidated = i === 0 && useValidatedModel;
      const customProperties = isValidated
        ? ({
            validated: {
              metadataType: ModelRegistryMetadataType.STRING,
              string_value: '',
            },
          } as ModelRegistryCustomProperties)
        : undefined;
      const name = isValidated ? 'validated-model' : `${label.toLowerCase()}-model-${i + 1}`;

      return mockCatalogModel({
        name,
        source_id: source.id,
        customProperties,
        ...(isValidated && {
          validatedTasks: [ValidatedConfiguration.TOOL_CALLING],
          servingConfig: {
            toolCalling: {
              args: '--enable-auto-tool-choice \\\n--tool-call-parser granite \\\n--chat-template\nopt/app-root/template/tool_chat_template_granite.jinja',
            },
          },
        }),
      });
    }),
  });

/**
 * Intercepts models by label using regex to handle filterQuery parameters
 */
export const interceptModelsByLabel = (
  sources: CatalogSource[],
  modelsPerCategory: number,
  useValidatedModel: boolean,
): void => {
  sources.forEach((source) => {
    source.labels.forEach((label) => {
      const mockModels = createMockModelsForLabel(
        source,
        label,
        modelsPerCategory,
        useValidatedModel,
      );
      const encodedLabel = encodeURIComponent(label);

      // Use regex-based intercept to match requests with this sourceLabel
      // This handles both basic requests and requests with filterQuery
      cy.intercept(
        {
          method: 'GET',
          url: new RegExp(
            `/model-registry/api/${MODEL_CATALOG_API_VERSION}/model_catalog/models.*sourceLabel=${encodedLabel}`,
          ),
        },
        mockModArchResponse(mockModels),
      ).as(`getModels-${label}`);
    });
  });
};

/**
 * Intercepts models requests without sourceLabel (for "All models" / GalleryView)
 */
export const interceptAllModels = (modelsPerCategory: number, useValidatedModel: boolean): void => {
  const allModelsResponse = mockCatalogModelList({
    items: Array.from({ length: modelsPerCategory }, (_, i) => {
      const isValidated = i === 0 && useValidatedModel;
      const customProperties = isValidated
        ? ({
            validated: {
              metadataType: ModelRegistryMetadataType.STRING,
              string_value: '',
            },
          } as ModelRegistryCustomProperties)
        : undefined;
      const name = isValidated ? 'validated-model' : `all-models-model-${i + 1}`;
      return mockCatalogModel({
        name,
        source_id: 'sample-source',
        customProperties,
        ...(isValidated && {
          validatedTasks: [ValidatedConfiguration.TOOL_CALLING],
          servingConfig: {
            toolCalling: {
              args: '--enable-auto-tool-choice \\\n--tool-call-parser granite \\\n--chat-template\nopt/app-root/template/tool_chat_template_granite.jinja',
            },
          },
        }),
      });
    }),
  });

  // Intercept for GalleryView when filters are applied (no sourceLabel, but has filterQuery or pageSize)
  cy.intercept(
    {
      method: 'GET',
      url: new RegExp(
        `/model-registry/api/${MODEL_CATALOG_API_VERSION}/model_catalog/models\\?(?!.*sourceLabel=)`,
      ),
    },
    mockModArchResponse(allModelsResponse),
  ).as('getModelsFiltered');
};

/**
 * Intercepts the filter_options endpoint
 */
export const interceptFilterOptions = (): void => {
  cy.interceptApi(
    `GET /api/:apiVersion/model_catalog/models/filter_options`,
    {
      path: { apiVersion: MODEL_CATALOG_API_VERSION },
      query: { namespace: 'kubeflow' },
    },
    mockCatalogFilterOptionsList(),
  );
};

/**
 * Intercepts a single model detail request
 */
export const interceptSingleModel = (sourceId: string, model: CatalogModel): void => {
  cy.interceptApi(
    `GET /api/:apiVersion/model_catalog/sources/:sourceId/models/:modelName`,
    {
      path: {
        apiVersion: MODEL_CATALOG_API_VERSION,
        sourceId,
        modelName: model.name.replace('/', '%2F'),
      },
    },
    model,
  );
};

/**
 * Intercepts single model using regex to match any source/model combination
 */
export const interceptSingleModelRegex = (model: CatalogModel): void => {
  cy.intercept(
    {
      method: 'GET',
      url: new RegExp(
        `/model-registry/api/${MODEL_CATALOG_API_VERSION}/model_catalog/sources/[^/]+/models/[^/]+$`,
      ),
    },
    mockModArchResponse(model),
  ).as('getSingleModel');
};

/**
 * Intercepts performance artifacts for a specific model using regex
 */
export const interceptPerformanceArtifacts = (modelName = 'validated-model'): void => {
  const performanceArtifactsResponse = {
    items: [
      mockCatalogPerformanceMetricsArtifact({
        customProperties: {
          hardware_configuration: {
            metadataType: ModelRegistryMetadataType.STRING,
            string_value: '2 x H100-80',
          },
          ttft_p90: {
            metadataType: ModelRegistryMetadataType.DOUBLE,
            double_value: 51.56,
          },
          replicas: {
            metadataType: ModelRegistryMetadataType.INT,
            int_value: '7',
          },
        },
      }),
      mockCatalogPerformanceMetricsArtifact({
        customProperties: {
          hardware_type: {
            metadataType: ModelRegistryMetadataType.STRING,
            string_value: 'RTX 4090',
          },
          hardware_count: {
            metadataType: ModelRegistryMetadataType.INT,
            int_value: '33',
          },
          hardware_configuration: {
            metadataType: ModelRegistryMetadataType.STRING,
            string_value: '33 x RTX 4090',
          },
          requests_per_second: {
            metadataType: ModelRegistryMetadataType.DOUBLE,
            double_value: 10,
          },
          ttft_mean: {
            metadataType: ModelRegistryMetadataType.DOUBLE,
            double_value: 67.15,
          },
          ttft_p90: {
            metadataType: ModelRegistryMetadataType.DOUBLE,
            double_value: 82.34,
          },
          replicas: {
            metadataType: ModelRegistryMetadataType.INT,
            int_value: '10',
          },
          use_case: {
            metadataType: ModelRegistryMetadataType.STRING,
            string_value: 'chatbot',
          },
        },
      }),
      mockCatalogPerformanceMetricsArtifact({
        customProperties: {
          hardware_type: {
            metadataType: ModelRegistryMetadataType.STRING,
            string_value: 'A100',
          },
          hardware_count: {
            metadataType: ModelRegistryMetadataType.INT,
            int_value: '40',
          },
          hardware_configuration: {
            metadataType: ModelRegistryMetadataType.STRING,
            string_value: '40 x A100',
          },
          requests_per_second: {
            metadataType: ModelRegistryMetadataType.DOUBLE,
            double_value: 15,
          },
          ttft_mean: {
            metadataType: ModelRegistryMetadataType.DOUBLE,
            double_value: 42.12,
          },
          ttft_p90: {
            metadataType: ModelRegistryMetadataType.DOUBLE,
            double_value: 58.45,
          },
          replicas: {
            metadataType: ModelRegistryMetadataType.INT,
            int_value: '15',
          },
          use_case: {
            metadataType: ModelRegistryMetadataType.STRING,
            string_value: 'chatbot',
          },
        },
      }),
    ],
    pageSize: 10,
    size: 3,
    nextPageToken: '',
  };

  // Use regex to match any source's model performance artifacts requests
  cy.intercept(
    {
      method: 'GET',
      url: new RegExp(
        `/model-registry/api/${MODEL_CATALOG_API_VERSION}/model_catalog/sources/.*/performance_artifacts/${modelName}`,
      ),
    },
    mockModArchResponse(performanceArtifactsResponse),
  ).as('getCatalogSourceModelArtifacts');
};

/**
 * Intercepts performance artifacts using the standard list mock
 * @param overrideArtifacts - Optional override for the artifacts response (e.g., empty list)
 */
export const interceptPerformanceArtifactsList = (overrideArtifacts?: {
  items: unknown[];
  size: number;
  pageSize: number;
  nextPageToken: string;
}): void => {
  const testArtifacts = overrideArtifacts ?? mockCatalogPerformanceMetricsArtifactList({});

  cy.intercept(
    {
      method: 'GET',
      url: new RegExp(
        `/model-registry/api/${MODEL_CATALOG_API_VERSION}/model_catalog/sources/.*/performance_artifacts/.*`,
      ),
    },
    mockModArchResponse(testArtifacts),
  ).as('getCatalogSourceModelArtifacts');
};

/**
 * Intercepts performance artifacts with dynamic response based on filterQuery presence.
 * Returns 3 artifacts when filterQuery is present, 5 when absent.
 * Used to test that perf filters are passed regardless of toggle state.
 */
export const interceptPerformanceArtifactsWithFilterCheck = (): void => {
  cy.intercept(
    {
      method: 'GET',
      url: new RegExp(
        `/model-registry/api/${MODEL_CATALOG_API_VERSION}/model_catalog/sources/.*/performance_artifacts/.*`,
      ),
    },
    (req) => {
      const hasFilterQuery = req.url.includes('filterQuery');
      // Return different counts based on whether filterQuery is present
      const artifactCount = hasFilterQuery ? 3 : 5;
      const items = Array.from({ length: artifactCount }, (_, i) =>
        mockCatalogPerformanceMetricsArtifact({
          customProperties: {
            hardware_configuration: {
              metadataType: ModelRegistryMetadataType.STRING,
              string_value: `${i + 1} x H100-80`,
            },
            ttft_p90: {
              metadataType: ModelRegistryMetadataType.DOUBLE,
              double_value: 50 + i * 10,
            },
            replicas: {
              metadataType: ModelRegistryMetadataType.INT,
              int_value: String(i + 1),
            },
            use_case: {
              metadataType: ModelRegistryMetadataType.STRING,
              string_value: 'chatbot',
            },
          },
        }),
      );
      req.reply(
        mockModArchResponse({
          items,
          size: artifactCount,
          pageSize: 10,
          nextPageToken: '',
        }),
      );
    },
  ).as('getCatalogSourceModelArtifacts');
};

/**
 * Intercepts the /artifacts/ endpoint (used to determine if tabs should show)
 * @param overrideArtifacts - Optional override for the artifacts response (e.g., empty list)
 */
export const interceptArtifactsList = (overrideArtifacts?: {
  items: unknown[];
  size: number;
  pageSize: number;
  nextPageToken: string;
}): void => {
  const testArtifacts = overrideArtifacts ?? mockCatalogPerformanceMetricsArtifactList({});

  cy.intercept(
    {
      method: 'GET',
      url: new RegExp(
        `/model-registry/api/${MODEL_CATALOG_API_VERSION}/model_catalog/sources/.*/artifacts/.*`,
      ),
    },
    mockModArchResponse(testArtifacts),
  ).as('getArtifacts');
};

/**
 * Intercepts model registry endpoint
 */
export const interceptModelRegistry = (): void => {
  cy.interceptApi(
    `GET /api/:apiVersion/model_registry`,
    {
      path: { apiVersion: 'v1' },
    },
    [mockModelRegistry({})],
  );
};

/**
 * Sets up all common model catalog intercepts with sensible defaults.
 * This is the main function to use for setting up test intercepts.
 */
export const setupModelCatalogIntercepts = (options: ModelCatalogInterceptOptions = {}): void => {
  const {
    sources = defaultSources(),
    modelsPerCategory = 4,
    useValidatedModel = false,
    includePerformanceArtifacts = false,
    includeFilterOptions = true,
    includeModelRegistry = false,
    customValidatedModel,
    customNonValidatedModel,
  } = options;

  // Always intercept sources and labels
  interceptSources(sources);
  interceptLabels();

  // Intercept models by label
  interceptModelsByLabel(sources, modelsPerCategory, useValidatedModel);

  // Intercept all models (for GalleryView without sourceLabel)
  interceptAllModels(modelsPerCategory, useValidatedModel);

  // Intercept single model detail
  const testModel = useValidatedModel
    ? (customValidatedModel ?? mockValidatedModel)
    : (customNonValidatedModel ?? mockNonValidatedModel);
  interceptSingleModel('source-2', testModel);
  interceptSingleModelRegex(testModel);

  // Optional intercepts
  if (includeFilterOptions) {
    interceptFilterOptions();
  }

  if (includePerformanceArtifacts) {
    interceptPerformanceArtifacts();
  }

  if (includeModelRegistry) {
    interceptModelRegistry();
  }
};

/**
 * Sets up intercepts specifically for tests that need validated models with performance data
 */
export const setupValidatedModelIntercepts = (
  options: Omit<
    ModelCatalogInterceptOptions,
    'useValidatedModel' | 'includePerformanceArtifacts'
  > = {},
): void => {
  setupModelCatalogIntercepts({
    ...options,
    useValidatedModel: true,
    includePerformanceArtifacts: true,
  });
};

/**
 * Sets up intercepts for model details page tests
 */
export const setupModelDetailsIntercepts = (options: ModelCatalogInterceptOptions = {}): void => {
  setupModelCatalogIntercepts({
    ...options,
    includeModelRegistry: true,
    includePerformanceArtifacts: true,
  });
};
