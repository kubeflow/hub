/* eslint-disable camelcase */
import { CatalogFilterOptionsList, NamedQuery, FilterOperator } from '~/app/modelCatalogTypes';
import {
  ModelCatalogStringFilterKey,
  ModelCatalogNumberFilterKey,
  ModelCatalogProvider,
  ModelCatalogTask,
  ValidatedConfiguration,
  AllLanguageCode,
  UseCaseOptionValue,
  DEFAULT_PERFORMANCE_FILTERS_QUERY_NAME,
  ModelCatalogTensorType,
} from '~/concepts/modelCatalog/const';

export const mockNamedQueries: Record<string, NamedQuery> = {
  // Default performance filters applied when performance toggle is turned on
  // Uses enum values which now match backend format
  [DEFAULT_PERFORMANCE_FILTERS_QUERY_NAME]: {
    [ModelCatalogStringFilterKey.USE_CASE]: {
      operator: FilterOperator.EQUALS,
      value: UseCaseOptionValue.CHATBOT,
    },
    'artifacts.ttft_p90.double_value': {
      operator: FilterOperator.LESS_THAN_OR_EQUAL,
      value: 'max', // 'max' means use the max value from the range in filters
    },
    [ModelCatalogNumberFilterKey.MAX_RPS]: {
      operator: FilterOperator.LESS_THAN_OR_EQUAL,
      value: 'max', // 'max' means use the max value from the range in filters (300 in mock)
    },
  },
  high_performance_gpu: {
    [ModelCatalogStringFilterKey.HARDWARE_TYPE]: {
      operator: FilterOperator.IN,
      value: ['H100-80', 'A100-80'],
    },
    [ModelCatalogNumberFilterKey.MAX_RPS]: {
      operator: FilterOperator.GREATER_THAN_OR_EQUAL,
      value: 50,
    },
  },
  low_latency: {
    'artifacts.ttft_p90.double_value': { operator: FilterOperator.LESS_THAN, value: 100 },
    'artifacts.e2e_p90.double_value': { operator: FilterOperator.LESS_THAN, value: 500 },
  },
  chatbot_optimized: {
    [ModelCatalogStringFilterKey.USE_CASE]: {
      operator: FilterOperator.EQUALS,
      value: UseCaseOptionValue.CHATBOT,
    },
  },
  rag_optimized: {
    [ModelCatalogStringFilterKey.USE_CASE]: {
      operator: FilterOperator.IN,
      value: [UseCaseOptionValue.RAG, UseCaseOptionValue.LONG_RAG],
    },
  },
  cost_effective: {
    'hardware_count.int_value': { operator: FilterOperator.LESS_THAN_OR_EQUAL, value: 2 },
  },
};

export const mockCatalogFilterOptionsList = (
  partial?: Partial<CatalogFilterOptionsList>,
): CatalogFilterOptionsList => ({
  filters: {
    [ModelCatalogStringFilterKey.PROVIDER]: {
      type: 'string',
      values: [ModelCatalogProvider.RED_HAT, ModelCatalogProvider.IBM, ModelCatalogProvider.GOOGLE],
    },
    [ModelCatalogStringFilterKey.LICENSE]: {
      type: 'string',
      values: ['Apache 2.0', 'MIT'],
    },
    [ModelCatalogStringFilterKey.TASK]: {
      type: 'string',
      values: [
        ModelCatalogTask.TEXT_GENERATION,
        ModelCatalogTask.TEXT_TO_TEXT,
        ModelCatalogTask.TEXT_EMBEDDING,
        ModelCatalogTask.TOOL_CALLING,
        ModelCatalogTask.IMAGE_TO_TEXT,
        ModelCatalogTask.IMAGE_TEXT_TO_TEXT,
        ModelCatalogTask.VIDEO_TO_TEXT,
        ModelCatalogTask.AUDIO_TO_TEXT,
      ],
    },
    [ModelCatalogStringFilterKey.LANGUAGE]: {
      type: 'string',
      values: [
        AllLanguageCode.AR,
        AllLanguageCode.CS,
        AllLanguageCode.DE,
        AllLanguageCode.EN,
        AllLanguageCode.ES,
        AllLanguageCode.FR,
        AllLanguageCode.IT,
        AllLanguageCode.JA,
        AllLanguageCode.KO,
        AllLanguageCode.NL,
        AllLanguageCode.PT,
        AllLanguageCode.ZH,
      ],
    },
    [ModelCatalogStringFilterKey.TENSOR_TYPE]: {
      type: 'string',
      values: [
        ModelCatalogTensorType.FP16,
        ModelCatalogTensorType.FP8,
        ModelCatalogTensorType.INT4,
        ModelCatalogTensorType.INT8,
        ModelCatalogTensorType.MXFP4,
      ],
    },
    [ModelCatalogStringFilterKey.VALIDATED_CONFIGURATION]: {
      type: 'string',
      values: [ValidatedConfiguration.TOOL_CALLING],
    },
    [ModelCatalogStringFilterKey.HARDWARE_TYPE]: {
      type: 'string',
      values: ['GPU', 'CPU', 'TPU', 'FPGA'],
    },
    [ModelCatalogStringFilterKey.USE_CASE]: {
      type: 'string',
      values: [
        UseCaseOptionValue.CHATBOT,
        UseCaseOptionValue.CODE_FIXING,
        UseCaseOptionValue.LONG_RAG,
        UseCaseOptionValue.RAG,
      ],
    },
    [ModelCatalogNumberFilterKey.MAX_RPS]: {
      type: 'number',
      range: {
        min: 1,
        max: 300,
      },
    },
    // All latency metric combinations for dropdown options (using full filter key format)
    'artifacts.ttft_mean.double_value': {
      type: 'number' as const,
      range: { min: 20, max: 893 },
    },
    'artifacts.ttft_p90.double_value': {
      type: 'number' as const,
      range: { min: 25, max: 600 },
    },
    'artifacts.ttft_p95.double_value': {
      type: 'number' as const,
      range: { min: 30, max: 700 },
    },
    'artifacts.ttft_p99.double_value': {
      type: 'number' as const,
      range: { min: 40, max: 893 },
    },
    'artifacts.e2e_mean.double_value': {
      type: 'number' as const,
      range: { min: 50, max: 800 },
    },
    'artifacts.e2e_p90.double_value': {
      type: 'number' as const,
      range: { min: 60, max: 900 },
    },
    'artifacts.e2e_p95.double_value': {
      type: 'number' as const,
      range: { min: 70, max: 1000 },
    },
    'artifacts.e2e_p99.double_value': {
      type: 'number' as const,
      range: { min: 80, max: 1200 },
    },
    'artifacts.tps_mean.double_value': {
      type: 'number' as const,
      range: { min: 10, max: 300 },
    },
    'artifacts.tps_p90.double_value': {
      type: 'number' as const,
      range: { min: 15, max: 350 },
    },
    'artifacts.tps_p95.double_value': {
      type: 'number' as const,
      range: { min: 20, max: 400 },
    },
    'artifacts.tps_p99.double_value': {
      type: 'number' as const,
      range: { min: 25, max: 500 },
    },
    'artifacts.itl_mean.double_value': {
      type: 'number' as const,
      range: { min: 5, max: 100 },
    },
    'artifacts.itl_p90.double_value': {
      type: 'number' as const,
      range: { min: 8, max: 120 },
    },
    'artifacts.itl_p95.double_value': {
      type: 'number' as const,
      range: { min: 10, max: 150 },
    },
    'artifacts.itl_p99.double_value': {
      type: 'number' as const,
      range: { min: 15, max: 200 },
    },
  },
  namedQueries: mockNamedQueries,
  ...partial,
});

// Mock for artifact-specific filter options (performance artifacts endpoint)
// This is a subset of filters relevant to performance artifacts
export const mockArtifactFilterOptionsList = (
  partial?: Partial<CatalogFilterOptionsList>,
): CatalogFilterOptionsList => {
  const base = mockCatalogFilterOptionsList();
  return {
    ...base,
    filters: {
      ...base.filters,
      [ModelCatalogStringFilterKey.HARDWARE_TYPE]: {
        type: 'string',
        values: ['H100-80', 'A100-80', 'L40S', 'MI300X'],
      },
    },
    namedQueries: mockNamedQueries,
    ...partial,
  };
};
