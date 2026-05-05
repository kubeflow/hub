// Disabling camelcase for this file because the inherent property names are not camelcase
/* eslint-disable camelcase */
import {
  CatalogFilterOptionsList,
  CatalogLabel,
  CatalogLabelList,
  CatalogSource,
  CatalogSourceList,
  ModelCatalogFilterStates,
  CatalogArtifactType,
  CatalogArtifacts,
  MetricsType,
} from '~/app/modelCatalogTypes';
import {
  AllLanguageCode,
  ModelCatalogNumberFilterKey,
  ModelCatalogProvider,
  ModelCatalogStringFilterKey,
  ModelCatalogTask,
  ModelCatalogTensorType,
  UseCaseOptionValue,
  CatalogModelCustomPropertyKey,
  ModelType,
} from '~/concepts/modelCatalog/const';
import {
  CatalogSourceStatus,
  isSourceStatusWithModels,
} from '~/concepts/modelCatalogSettings/const';
import {
  filtersToFilterQuery,
  filterEnabledCatalogSources,
  filterSourcesWithModels,
  hasSourcesWithModels,
  getUniqueSourceLabels,
  hasSourcesWithoutLabels,
  hasFiltersApplied,
  getArchitecturesFromArtifacts,
  getModelName,
  getCatalogModelTypePropertyForRegistration,
  getActiveSourceLabels,
} from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import { mockCatalogModelArtifact } from '~/__mocks__/mockCatalogModelArtifactList';
import { ModelRegistryMetadataType } from '~/app/types';

// TODO: Implement performance filters.
describe('filtersToFilterQuery', () => {
  const mockFormData = ({
    tasks = [],
    license = [],
    provider = [],
    language = [],
    hardware_type = [],
    hardware_configuration = [],
    use_case = [],
    tensor_type = [],
    rps_mean = undefined,
    ttft_mean = undefined,
  }: {
    tasks?: ModelCatalogTask[];
    license?: string[];
    provider?: ModelCatalogProvider[];
    language?: AllLanguageCode[];
    hardware_type?: string[];
    hardware_configuration?: string[];
    use_case?: UseCaseOptionValue[];
    rps_mean?: number;
    tensor_type?: ModelCatalogTensorType[];
    ttft_mean?: number;
  }): ModelCatalogFilterStates => ({
    [ModelCatalogStringFilterKey.TASK]: tasks,
    [ModelCatalogStringFilterKey.PROVIDER]: provider,
    [ModelCatalogStringFilterKey.LICENSE]: license,
    [ModelCatalogStringFilterKey.LANGUAGE]: language,
    [ModelCatalogStringFilterKey.HARDWARE_TYPE]: hardware_type,
    [ModelCatalogStringFilterKey.HARDWARE_CONFIGURATION]: hardware_configuration,
    [ModelCatalogStringFilterKey.USE_CASE]: use_case,
    [ModelCatalogNumberFilterKey.MAX_RPS]: rps_mean,
    [ModelCatalogStringFilterKey.TENSOR_TYPE]: tensor_type,
    'artifacts.ttft_mean.double_value': ttft_mean,
  });

  const mockFilterOptions: CatalogFilterOptionsList = {
    filters: {
      [ModelCatalogStringFilterKey.TASK]: {
        type: 'string',
        values: [
          ModelCatalogTask.AUDIO_TO_TEXT,
          ModelCatalogTask.IMAGE_TEXT_TO_TEXT,
          ModelCatalogTask.IMAGE_TO_TEXT,
          ModelCatalogTask.TEXT_EMBEDDING,
          ModelCatalogTask.TEXT_GENERATION,
          ModelCatalogTask.TEXT_TO_TEXT,
          ModelCatalogTask.TOOL_CALLING,
          ModelCatalogTask.VIDEO_TO_TEXT,
        ],
      },
      [ModelCatalogStringFilterKey.PROVIDER]: {
        type: 'string',
        values: [
          ModelCatalogProvider.ALIBABA_CLOUD,
          ModelCatalogProvider.DEEPSEEK,
          ModelCatalogProvider.GOOGLE,
          ModelCatalogProvider.IBM,
          ModelCatalogProvider.META,
          ModelCatalogProvider.MISTRAL_AI,
          ModelCatalogProvider.MOONSHOT_AI,
          ModelCatalogProvider.NEURAL_MAGIC,
          ModelCatalogProvider.NVIDIA,
          ModelCatalogProvider.NVIDIA_ALTERNATE,
          ModelCatalogProvider.RED_HAT,
        ],
      },
      [ModelCatalogStringFilterKey.LICENSE]: {
        type: 'string',
        values: ['Apache 2.0', 'Gemma', 'Llama 3.3', 'Llama 3.1', 'Llama 4', 'MIT', 'Modified MIT'],
      },
      [ModelCatalogStringFilterKey.LANGUAGE]: {
        type: 'string',
        values: [
          AllLanguageCode.BG,
          AllLanguageCode.CA,
          AllLanguageCode.CS,
          AllLanguageCode.DA,
          AllLanguageCode.DE,
          AllLanguageCode.EL,
          AllLanguageCode.EN,
          AllLanguageCode.ES,
          AllLanguageCode.FI,
          AllLanguageCode.FR,
          AllLanguageCode.HR,
          AllLanguageCode.HU,
          AllLanguageCode.IS,
          AllLanguageCode.IT,
          AllLanguageCode.NL,
          AllLanguageCode.NLD,
          AllLanguageCode.NO,
          AllLanguageCode.PL,
          AllLanguageCode.PT,
          AllLanguageCode.RO,
          AllLanguageCode.RU,
          AllLanguageCode.SK,
          AllLanguageCode.SL,
          AllLanguageCode.SR,
          AllLanguageCode.SV,
          AllLanguageCode.UK,
          AllLanguageCode.JA,
          AllLanguageCode.KO,
          AllLanguageCode.ZH,
          AllLanguageCode.HI,
          AllLanguageCode.TH,
          AllLanguageCode.VI,
          AllLanguageCode.ID,
          AllLanguageCode.MS,
          AllLanguageCode.ZSM,
          AllLanguageCode.AR,
          AllLanguageCode.FA,
          AllLanguageCode.HE,
          AllLanguageCode.TR,
          AllLanguageCode.UR,
          AllLanguageCode.TL,
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
          min: 0,
          max: 300,
        },
      },
      'artifacts.ttft_mean.double_value': {
        type: 'number',
        range: {
          min: 0,
          max: 1000,
        },
      },
    },
  };

  describe('multi-selection values', () => {
    it('handles no data', () => {
      expect(filtersToFilterQuery(mockFormData({}), mockFilterOptions)).toBe('');
    });

    it('returns empty string when options.filters is undefined', () => {
      const optionsWithoutFilters = { namedQueries: {} } as CatalogFilterOptionsList;
      expect(
        filtersToFilterQuery(
          mockFormData({ tasks: [ModelCatalogTask.TEXT_TO_TEXT] }),
          optionsWithoutFilters,
        ),
      ).toBe('');
    });

    it('handles a single array of a single data point', () => {
      expect(
        filtersToFilterQuery(
          mockFormData({ tasks: [ModelCatalogTask.TEXT_TO_TEXT] }),
          mockFilterOptions,
        ),
      ).toBe("tasks='text-to-text'");
      expect(
        filtersToFilterQuery(
          mockFormData({ provider: [ModelCatalogProvider.GOOGLE] }),
          mockFilterOptions,
        ),
      ).toBe("provider='Google'");
      expect(
        filtersToFilterQuery(mockFormData({ license: ['Apache 2.0'] }), mockFilterOptions),
      ).toBe("license='Apache 2.0'");
      expect(
        filtersToFilterQuery(mockFormData({ language: [AllLanguageCode.CA] }), mockFilterOptions),
      ).toBe("language='ca'");
      expect(
        filtersToFilterQuery(
          mockFormData({ use_case: [UseCaseOptionValue.CHATBOT] }),
          mockFilterOptions,
        ),
      ).toBe("artifacts.use_case.string_value='chatbot'");
      expect(
        filtersToFilterQuery(
          mockFormData({ tensor_type: [ModelCatalogTensorType.FP16] }),
          mockFilterOptions,
        ),
      ).toBe("tensor_type.string_value='FP16'");
    });

    it('handles multiple arrays of a single data point', () => {
      expect(
        filtersToFilterQuery(
          mockFormData({
            tasks: [ModelCatalogTask.TEXT_TO_TEXT],
            license: ['Apache 2.0'],
          }),
          mockFilterOptions,
        ),
      ).toBe("tasks='text-to-text' AND license='Apache 2.0'");
      expect(
        filtersToFilterQuery(
          mockFormData({ provider: [ModelCatalogProvider.GOOGLE], language: [AllLanguageCode.CA] }),
          mockFilterOptions,
        ),
      ).toBe("provider='Google' AND language='ca'");
    });

    it('handles a single array with multiple data points', () => {
      expect(
        filtersToFilterQuery(
          mockFormData({ tasks: [ModelCatalogTask.TEXT_TO_TEXT, ModelCatalogTask.IMAGE_TO_TEXT] }),
          mockFilterOptions,
        ),
      ).toBe("tasks IN ('text-to-text','image-to-text')");
      expect(
        filtersToFilterQuery(
          mockFormData({ provider: [ModelCatalogProvider.GOOGLE, ModelCatalogProvider.DEEPSEEK] }),
          mockFilterOptions,
        ),
      ).toBe("provider IN ('Google','DeepSeek')");
      expect(
        filtersToFilterQuery(mockFormData({ license: ['Apache 2.0', 'MIT'] }), mockFilterOptions),
      ).toBe("license IN ('Apache 2.0','MIT')");
      expect(
        filtersToFilterQuery(
          mockFormData({ language: [AllLanguageCode.CA, AllLanguageCode.PT] }),
          mockFilterOptions,
        ),
      ).toBe("language IN ('ca','pt')");
      expect(
        filtersToFilterQuery(
          mockFormData({ tensor_type: [ModelCatalogTensorType.FP16, ModelCatalogTensorType.FP8] }),
          mockFilterOptions,
        ),
      ).toBe("tensor_type.string_value IN ('FP16','FP8')");
      // Note: use_case is now single-select, so multi-select test is not applicable
    });

    it('handles all tensor type enum values', () => {
      expect(
        filtersToFilterQuery(
          mockFormData({
            tensor_type: [
              ModelCatalogTensorType.FP16,
              ModelCatalogTensorType.FP8,
              ModelCatalogTensorType.INT4,
              ModelCatalogTensorType.INT8,
              ModelCatalogTensorType.MXFP4,
            ],
          }),
          mockFilterOptions,
        ),
      ).toBe("tensor_type.string_value IN ('FP16','FP8','INT4','INT8','MXFP4')");
    });

    it('handles multiple arrays with mixed count of data points', () => {
      expect(
        filtersToFilterQuery(
          mockFormData({
            tasks: [ModelCatalogTask.TEXT_TO_TEXT, ModelCatalogTask.IMAGE_TO_TEXT],
            provider: [ModelCatalogProvider.GOOGLE],
            license: ['MIT'],
            language: [
              AllLanguageCode.CA,
              AllLanguageCode.PT,
              AllLanguageCode.VI,
              AllLanguageCode.ZSM,
            ],
          }),
          mockFilterOptions,
        ),
      ).toBe(
        "tasks IN ('text-to-text','image-to-text') AND provider='Google' AND license='MIT' AND language IN ('ca','pt','vi','zsm')",
      );
    });

    it('handles tensor type combined with other basic filters', () => {
      expect(
        filtersToFilterQuery(
          mockFormData({
            tasks: [ModelCatalogTask.TEXT_TO_TEXT],
            tensor_type: [ModelCatalogTensorType.FP16, ModelCatalogTensorType.INT8],
            provider: [ModelCatalogProvider.GOOGLE],
          }),
          mockFilterOptions,
        ),
      ).toBe(
        "tasks='text-to-text' AND provider='Google' AND tensor_type.string_value IN ('FP16','INT8')",
      );
    });
  });

  // TODO: Implement performance filters.
  //   describe('less than values', () => {
  //     it('handles TimeToFirstToken - ttft', () => {
  //       expect(filtersToFilterQuery(mockFormData({ ttft_mean: 100 }), mockFilterOptions)).toBe(
  //         'ttft_mean < 100',
  //       );
  //     });
  //   });

  //   describe('greater than values', () => {
  //     it('handles TimeToFirstToken - ttft', () => {
  //       expect(filtersToFilterQuery(mockFormData({ rps_mean: 7 }), mockFilterOptions)).toBe(
  //         'rps_mean > 7',
  //       );
  //     });
  //   });

  //   describe('mixture of multiple types of values', () => {
  //     it('handles TimeToFirstToken - ttft', () => {
  //       expect(
  //         filtersToFilterQuery(
  //           mockFormData({
  //             ttft_mean: 100,
  //             tasks: [ModelCatalogTask.TEXT_TO_TEXT],
  //             license: ['Apache 2.0', 'MIT'],
  //             rps_mean: 3,
  //           }),
  //           mockFilterOptions,
  //         ),
  //       ).toBe(
  //         "tasks='text-to-text' AND license IN ('Apache 2.0','MIT') AND ttft_mean < 100 AND rps_mean > 3",
  //       );
  //     });
  //   });
});

describe('catalog source filtering utilities', () => {
  const createMockSource = (overrides: Partial<CatalogSource> = {}): CatalogSource => ({
    id: 'source-1',
    name: 'Test Source',
    labels: ['Red Hat'],
    enabled: true,
    status: CatalogSourceStatus.AVAILABLE,
    ...overrides,
  });

  const createMockSourceList = (items: CatalogSource[] = []): CatalogSourceList => ({
    items,
    size: items.length,
    pageSize: 10,
    nextPageToken: '',
  });

  describe('filterEnabledCatalogSources', () => {
    it('returns null when catalogSources is null', () => {
      expect(filterEnabledCatalogSources(null)).toBeNull();
    });

    it('returns empty list when no sources are enabled and available', () => {
      const sources = createMockSourceList([
        createMockSource({ id: '1', enabled: false, status: CatalogSourceStatus.DISABLED }),
        createMockSource({ id: '2', enabled: true, status: CatalogSourceStatus.ERROR }),
      ]);
      const result = filterEnabledCatalogSources(sources);
      expect(result?.items).toHaveLength(0);
    });

    it('filters out disabled sources', () => {
      const sources = createMockSourceList([
        createMockSource({ id: '1', enabled: true, status: CatalogSourceStatus.AVAILABLE }),
        createMockSource({ id: '2', enabled: false, status: CatalogSourceStatus.AVAILABLE }),
      ]);
      const result = filterEnabledCatalogSources(sources);
      expect(result?.items).toHaveLength(1);
      expect(result?.items?.[0].id).toBe('1');
    });

    it('filters out sources without available status', () => {
      const sources = createMockSourceList([
        createMockSource({ id: '1', enabled: true, status: CatalogSourceStatus.AVAILABLE }),
        createMockSource({ id: '2', enabled: true, status: CatalogSourceStatus.ERROR }),
        createMockSource({ id: '3', enabled: true, status: CatalogSourceStatus.DISABLED }),
      ]);
      const result = filterEnabledCatalogSources(sources);
      expect(result?.items).toHaveLength(1);
      expect(result?.items?.[0].id).toBe('1');
    });

    it('returns all sources when all are enabled and available', () => {
      const sources = createMockSourceList([
        createMockSource({ id: '1', enabled: true, status: CatalogSourceStatus.AVAILABLE }),
        createMockSource({ id: '2', enabled: true, status: CatalogSourceStatus.AVAILABLE }),
      ]);
      const result = filterEnabledCatalogSources(sources);
      expect(result?.items).toHaveLength(2);
    });

    it('includes partially-available sources', () => {
      const sources = createMockSourceList([
        createMockSource({ id: '1', enabled: true, status: CatalogSourceStatus.AVAILABLE }),
        createMockSource({
          id: '2',
          enabled: true,
          status: CatalogSourceStatus.PARTIALLY_AVAILABLE,
        }),
        createMockSource({ id: '3', enabled: true, status: CatalogSourceStatus.ERROR }),
      ]);
      const result = filterEnabledCatalogSources(sources);
      expect(result?.items).toHaveLength(2);
      expect(result?.items?.map((s) => s.id)).toEqual(['1', '2']);
    });
  });

  describe('filterSourcesWithModels', () => {
    it('returns null when catalogSources is null', () => {
      expect(filterSourcesWithModels(null)).toBeNull();
    });

    it('returns only sources with available status', () => {
      const sources = createMockSourceList([
        createMockSource({ id: '1', status: CatalogSourceStatus.AVAILABLE }),
        createMockSource({ id: '2', status: CatalogSourceStatus.ERROR }),
        createMockSource({ id: '3', status: CatalogSourceStatus.DISABLED }),
      ]);
      const result = filterSourcesWithModels(sources);
      expect(result?.items).toHaveLength(1);
      expect(result?.items?.[0].id).toBe('1');
    });

    it('returns empty list when no sources have available status', () => {
      const sources = createMockSourceList([
        createMockSource({ id: '1', status: CatalogSourceStatus.ERROR }),
        createMockSource({ id: '2', status: CatalogSourceStatus.DISABLED }),
      ]);
      const result = filterSourcesWithModels(sources);
      expect(result?.items).toHaveLength(0);
    });

    it('includes partially-available sources', () => {
      const sources = createMockSourceList([
        createMockSource({ id: '1', status: CatalogSourceStatus.AVAILABLE }),
        createMockSource({ id: '2', status: CatalogSourceStatus.PARTIALLY_AVAILABLE }),
        createMockSource({ id: '3', status: CatalogSourceStatus.ERROR }),
      ]);
      const result = filterSourcesWithModels(sources);
      expect(result?.items).toHaveLength(2);
      expect(result?.items?.map((s) => s.id)).toEqual(['1', '2']);
    });
  });

  describe('hasSourcesWithModels', () => {
    it('returns false when catalogSources is null', () => {
      expect(hasSourcesWithModels(null)).toBe(false);
    });

    it('returns false when catalogSources has no items', () => {
      expect(hasSourcesWithModels(createMockSourceList([]))).toBe(false);
    });

    it('returns false when no sources have available status', () => {
      const sources = createMockSourceList([
        createMockSource({ id: '1', status: CatalogSourceStatus.ERROR }),
        createMockSource({ id: '2', status: CatalogSourceStatus.DISABLED }),
      ]);
      expect(hasSourcesWithModels(sources)).toBe(false);
    });

    it('returns true when at least one source has available status', () => {
      const sources = createMockSourceList([
        createMockSource({ id: '1', status: CatalogSourceStatus.ERROR }),
        createMockSource({ id: '2', status: CatalogSourceStatus.AVAILABLE }),
      ]);
      expect(hasSourcesWithModels(sources)).toBe(true);
    });

    it('returns true when all sources have available status', () => {
      const sources = createMockSourceList([
        createMockSource({ id: '1', status: CatalogSourceStatus.AVAILABLE }),
        createMockSource({ id: '2', status: CatalogSourceStatus.AVAILABLE }),
      ]);
      expect(hasSourcesWithModels(sources)).toBe(true);
    });

    it('returns true when a source has partially-available status', () => {
      const sources = createMockSourceList([
        createMockSource({ id: '1', status: CatalogSourceStatus.ERROR }),
        createMockSource({ id: '2', status: CatalogSourceStatus.PARTIALLY_AVAILABLE }),
      ]);
      expect(hasSourcesWithModels(sources)).toBe(true);
    });
  });

  describe('getUniqueSourceLabels', () => {
    it('returns empty array when catalogSources is null', () => {
      expect(getUniqueSourceLabels(null)).toEqual([]);
    });

    it('returns empty array when catalogSources has no items', () => {
      expect(getUniqueSourceLabels(createMockSourceList([]))).toEqual([]);
    });

    it('returns unique labels from enabled and available sources', () => {
      const sources = createMockSourceList([
        createMockSource({
          id: '1',
          labels: ['Red Hat', 'Enterprise'],
          enabled: true,
          status: CatalogSourceStatus.AVAILABLE,
        }),
        createMockSource({
          id: '2',
          labels: ['Red Hat', 'Community'],
          enabled: true,
          status: CatalogSourceStatus.AVAILABLE,
        }),
      ]);
      const labels = getUniqueSourceLabels(sources);
      expect(labels).toHaveLength(3);
      expect(labels).toContain('Red Hat');
      expect(labels).toContain('Enterprise');
      expect(labels).toContain('Community');
    });

    it('excludes labels from disabled sources', () => {
      const sources = createMockSourceList([
        createMockSource({
          id: '1',
          labels: ['Red Hat'],
          enabled: true,
          status: CatalogSourceStatus.AVAILABLE,
        }),
        createMockSource({
          id: '2',
          labels: ['Excluded'],
          enabled: false,
          status: CatalogSourceStatus.AVAILABLE,
        }),
      ]);
      const labels = getUniqueSourceLabels(sources);
      expect(labels).toEqual(['Red Hat']);
    });

    it('excludes labels from sources without available status', () => {
      const sources = createMockSourceList([
        createMockSource({
          id: '1',
          labels: ['Red Hat'],
          enabled: true,
          status: CatalogSourceStatus.AVAILABLE,
        }),
        createMockSource({
          id: '2',
          labels: ['Error Source'],
          enabled: true,
          status: CatalogSourceStatus.ERROR,
        }),
      ]);
      const labels = getUniqueSourceLabels(sources);
      expect(labels).toEqual(['Red Hat']);
    });

    it('includes labels from partially-available sources', () => {
      const sources = createMockSourceList([
        createMockSource({
          id: '1',
          labels: ['Red Hat'],
          enabled: true,
          status: CatalogSourceStatus.AVAILABLE,
        }),
        createMockSource({
          id: '2',
          labels: ['Partial Source'],
          enabled: true,
          status: CatalogSourceStatus.PARTIALLY_AVAILABLE,
        }),
      ]);
      const labels = getUniqueSourceLabels(sources);
      expect(labels).toHaveLength(2);
      expect(labels).toContain('Red Hat');
      expect(labels).toContain('Partial Source');
    });

    it('trims whitespace from labels', () => {
      const sources = createMockSourceList([
        createMockSource({
          id: '1',
          labels: ['  Red Hat  ', 'Enterprise'],
          enabled: true,
          status: CatalogSourceStatus.AVAILABLE,
        }),
      ]);
      const labels = getUniqueSourceLabels(sources);
      expect(labels).toContain('Red Hat');
    });

    it('excludes empty or whitespace-only labels', () => {
      const sources = createMockSourceList([
        createMockSource({
          id: '1',
          labels: ['Red Hat', '', '   '],
          enabled: true,
          status: CatalogSourceStatus.AVAILABLE,
        }),
      ]);
      const labels = getUniqueSourceLabels(sources);
      expect(labels).toEqual(['Red Hat']);
    });
  });

  describe('hasSourcesWithoutLabels', () => {
    it('returns false when catalogSources is null', () => {
      expect(hasSourcesWithoutLabels(null)).toBe(false);
    });

    it('returns false when catalogSources has no items', () => {
      expect(hasSourcesWithoutLabels(createMockSourceList([]))).toBe(false);
    });

    it('returns true when an enabled and available source has no labels', () => {
      const sources = createMockSourceList([
        createMockSource({
          id: '1',
          labels: ['Red Hat'],
          enabled: true,
          status: CatalogSourceStatus.AVAILABLE,
        }),
        createMockSource({
          id: '2',
          labels: [],
          enabled: true,
          status: CatalogSourceStatus.AVAILABLE,
        }),
      ]);
      expect(hasSourcesWithoutLabels(sources)).toBe(true);
    });

    it('returns true when an enabled and available source has only whitespace labels', () => {
      const sources = createMockSourceList([
        createMockSource({
          id: '1',
          labels: ['   ', ''],
          enabled: true,
          status: CatalogSourceStatus.AVAILABLE,
        }),
      ]);
      expect(hasSourcesWithoutLabels(sources)).toBe(true);
    });

    it('returns false when all enabled and available sources have labels', () => {
      const sources = createMockSourceList([
        createMockSource({
          id: '1',
          labels: ['Red Hat'],
          enabled: true,
          status: CatalogSourceStatus.AVAILABLE,
        }),
        createMockSource({
          id: '2',
          labels: ['Community'],
          enabled: true,
          status: CatalogSourceStatus.AVAILABLE,
        }),
      ]);
      expect(hasSourcesWithoutLabels(sources)).toBe(false);
    });

    it('ignores disabled sources without labels', () => {
      const sources = createMockSourceList([
        createMockSource({
          id: '1',
          labels: ['Red Hat'],
          enabled: true,
          status: CatalogSourceStatus.AVAILABLE,
        }),
        createMockSource({
          id: '2',
          labels: [],
          enabled: false,
          status: CatalogSourceStatus.AVAILABLE,
        }),
      ]);
      expect(hasSourcesWithoutLabels(sources)).toBe(false);
    });

    it('ignores sources without available status that have no labels', () => {
      const sources = createMockSourceList([
        createMockSource({
          id: '1',
          labels: ['Red Hat'],
          enabled: true,
          status: CatalogSourceStatus.AVAILABLE,
        }),
        createMockSource({
          id: '2',
          labels: [],
          enabled: true,
          status: CatalogSourceStatus.ERROR,
        }),
      ]);
      expect(hasSourcesWithoutLabels(sources)).toBe(false);
    });

    it('returns true when a partially-available source has no labels', () => {
      const sources = createMockSourceList([
        createMockSource({
          id: '1',
          labels: ['Red Hat'],
          enabled: true,
          status: CatalogSourceStatus.AVAILABLE,
        }),
        createMockSource({
          id: '2',
          labels: [],
          enabled: true,
          status: CatalogSourceStatus.PARTIALLY_AVAILABLE,
        }),
      ]);
      expect(hasSourcesWithoutLabels(sources)).toBe(true);
    });
  });
});

describe('isSourceStatusWithModels', () => {
  it('returns true for available status', () => {
    expect(isSourceStatusWithModels(CatalogSourceStatus.AVAILABLE)).toBe(true);
  });

  it('returns true for partially-available status', () => {
    expect(isSourceStatusWithModels(CatalogSourceStatus.PARTIALLY_AVAILABLE)).toBe(true);
  });

  it('returns false for error status', () => {
    expect(isSourceStatusWithModels(CatalogSourceStatus.ERROR)).toBe(false);
  });

  it('returns false for disabled status', () => {
    expect(isSourceStatusWithModels(CatalogSourceStatus.DISABLED)).toBe(false);
  });
});

describe('hasFiltersApplied', () => {
  const mockFormData = ({
    tasks = [],
    license = [],
    provider = [],
    language = [],
    hardware_type = [],
    hardware_configuration = [],
    use_case = [],
    tensor_type = [],
    rps_mean = undefined,
    ttft_mean = undefined,
  }: {
    tasks?: ModelCatalogTask[];
    license?: string[];
    provider?: ModelCatalogProvider[];
    language?: AllLanguageCode[];
    hardware_type?: string[];
    hardware_configuration?: string[];
    use_case?: UseCaseOptionValue[];
    tensor_type?: ModelCatalogTensorType[];
    rps_mean?: number;
    ttft_mean?: number;
  }): ModelCatalogFilterStates => ({
    [ModelCatalogStringFilterKey.TASK]: tasks,
    [ModelCatalogStringFilterKey.PROVIDER]: provider,
    [ModelCatalogStringFilterKey.LICENSE]: license,
    [ModelCatalogStringFilterKey.LANGUAGE]: language,
    [ModelCatalogStringFilterKey.HARDWARE_TYPE]: hardware_type,
    [ModelCatalogStringFilterKey.HARDWARE_CONFIGURATION]: hardware_configuration,
    [ModelCatalogStringFilterKey.USE_CASE]: use_case,
    [ModelCatalogNumberFilterKey.MAX_RPS]: rps_mean,
    [ModelCatalogStringFilterKey.TENSOR_TYPE]: tensor_type,
    'artifacts.ttft_mean.double_value': ttft_mean,
  });

  describe('without filterKeys parameter (checks all filters)', () => {
    it('returns false when no filters are applied', () => {
      expect(hasFiltersApplied(mockFormData({}))).toBe(false);
    });

    it('returns false when all arrays are empty and number filters are undefined', () => {
      expect(
        hasFiltersApplied(
          mockFormData({
            tasks: [],
            license: [],
            provider: [],
            language: [],
            tensor_type: [],
            hardware_type: [],
            use_case: [],
            rps_mean: undefined,
            ttft_mean: undefined,
          }),
        ),
      ).toBe(false);
    });

    it('returns true when a string filter array has values', () => {
      expect(hasFiltersApplied(mockFormData({ tasks: [ModelCatalogTask.TEXT_TO_TEXT] }))).toBe(
        true,
      );
      expect(hasFiltersApplied(mockFormData({ tensor_type: [ModelCatalogTensorType.FP16] }))).toBe(
        true,
      );
      expect(hasFiltersApplied(mockFormData({ provider: [ModelCatalogProvider.GOOGLE] }))).toBe(
        true,
      );
      expect(hasFiltersApplied(mockFormData({ license: ['MIT'] }))).toBe(true);
      expect(hasFiltersApplied(mockFormData({ language: [AllLanguageCode.EN] }))).toBe(true);
      expect(hasFiltersApplied(mockFormData({ hardware_type: ['GPU'] }))).toBe(true);
      expect(hasFiltersApplied(mockFormData({ use_case: [UseCaseOptionValue.CHATBOT] }))).toBe(
        true,
      );
    });

    it('returns true when multiple string filter arrays have values', () => {
      expect(
        hasFiltersApplied(
          mockFormData({
            tasks: [ModelCatalogTask.TEXT_TO_TEXT],
            provider: [ModelCatalogProvider.GOOGLE, ModelCatalogProvider.META],
          }),
        ),
      ).toBe(true);
    });

    it('returns true when a number filter is defined', () => {
      expect(hasFiltersApplied(mockFormData({ rps_mean: 50 }))).toBe(true);
      expect(hasFiltersApplied(mockFormData({ ttft_mean: 100 }))).toBe(true);
    });

    it('returns true when number filter is 0 (edge case)', () => {
      expect(hasFiltersApplied(mockFormData({ rps_mean: 0 }))).toBe(true);
      expect(hasFiltersApplied(mockFormData({ ttft_mean: 0 }))).toBe(true);
    });

    it('returns true when both string and number filters are applied', () => {
      expect(
        hasFiltersApplied(
          mockFormData({
            tasks: [ModelCatalogTask.TEXT_TO_TEXT],
            rps_mean: 50,
            ttft_mean: 100,
          }),
        ),
      ).toBe(true);
    });

    it('returns false when tensor_type is an empty array', () => {
      expect(hasFiltersApplied(mockFormData({ tensor_type: [] }))).toBe(false);
    });
  });

  describe('with filterKeys parameter (checks only specific filters)', () => {
    it('returns false when specified filters are not applied', () => {
      const filterData = mockFormData({
        tasks: [ModelCatalogTask.TEXT_TO_TEXT], // This filter is applied
      });
      // But we only check for provider filter
      expect(hasFiltersApplied(filterData, [ModelCatalogStringFilterKey.PROVIDER])).toBe(false);
    });

    it('returns true when one of the specified filters is applied', () => {
      const filterData = mockFormData({
        tasks: [ModelCatalogTask.TEXT_TO_TEXT],
        provider: [ModelCatalogProvider.GOOGLE],
      });
      expect(hasFiltersApplied(filterData, [ModelCatalogStringFilterKey.TASK])).toBe(true);
      expect(hasFiltersApplied(filterData, [ModelCatalogStringFilterKey.PROVIDER])).toBe(true);
      expect(hasFiltersApplied(filterData, [ModelCatalogStringFilterKey.TENSOR_TYPE])).toBe(false);

      const filterDataWithTensorType = mockFormData({
        tensor_type: [ModelCatalogTensorType.FP16],
      });
      expect(
        hasFiltersApplied(filterDataWithTensorType, [ModelCatalogStringFilterKey.TENSOR_TYPE]),
      ).toBe(true);
    });

    it('returns true when checking multiple keys and at least one is applied', () => {
      const filterData = mockFormData({
        tasks: [ModelCatalogTask.TEXT_TO_TEXT],
      });
      expect(
        hasFiltersApplied(filterData, [
          ModelCatalogStringFilterKey.TASK,
          ModelCatalogStringFilterKey.PROVIDER,
        ]),
      ).toBe(true);
    });

    it('returns false when checking multiple keys and none are applied', () => {
      const filterData = mockFormData({
        tasks: [ModelCatalogTask.TEXT_TO_TEXT],
      });
      expect(
        hasFiltersApplied(filterData, [
          ModelCatalogStringFilterKey.PROVIDER,
          ModelCatalogStringFilterKey.LICENSE,
        ]),
      ).toBe(false);
    });

    it('correctly checks number filters when specified', () => {
      const filterData = mockFormData({
        rps_mean: 50,
      });
      expect(hasFiltersApplied(filterData, [ModelCatalogNumberFilterKey.MAX_RPS])).toBe(true);
      expect(hasFiltersApplied(filterData, [ModelCatalogStringFilterKey.TASK])).toBe(false);
    });

    it('correctly checks latency filters when specified', () => {
      const filterData = mockFormData({
        ttft_mean: 100,
      });
      expect(hasFiltersApplied(filterData, ['artifacts.ttft_mean.double_value'])).toBe(true);
      expect(hasFiltersApplied(filterData, ['artifacts.e2e_mean.double_value'])).toBe(false);
    });

    it('handles mixed filter types in filterKeys', () => {
      const filterData = mockFormData({
        tasks: [ModelCatalogTask.TEXT_TO_TEXT],
        rps_mean: 50,
        ttft_mean: 100,
      });
      expect(
        hasFiltersApplied(filterData, [
          ModelCatalogStringFilterKey.TASK,
          ModelCatalogNumberFilterKey.MAX_RPS,
          'artifacts.ttft_mean.double_value',
        ]),
      ).toBe(true);
    });

    it('returns false with empty filterKeys array', () => {
      const filterData = mockFormData({
        tasks: [ModelCatalogTask.TEXT_TO_TEXT],
        rps_mean: 50,
      });
      expect(hasFiltersApplied(filterData, [])).toBe(false);
    });
  });
});

describe('getArchitecturesFromArtifacts', () => {
  it('should return empty array when artifacts array is empty', () => {
    const result = getArchitecturesFromArtifacts([]);
    expect(result).toEqual([]);
  });

  it('should return empty array when no model artifact exists', () => {
    const metricsArtifact: CatalogArtifacts = {
      artifactType: CatalogArtifactType.metricsArtifact,
      metricsType: MetricsType.performanceMetrics,
    };
    const result = getArchitecturesFromArtifacts([metricsArtifact]);
    expect(result).toEqual([]);
  });

  it('should return empty array when model artifact has no customProperties', () => {
    const artifact = mockCatalogModelArtifact({ customProperties: undefined });
    const result = getArchitecturesFromArtifacts([artifact]);
    expect(result).toEqual([]);
  });

  it('should return empty array when model artifact has empty customProperties', () => {
    const artifact = mockCatalogModelArtifact({ customProperties: {} });
    const result = getArchitecturesFromArtifacts([artifact]);
    expect(result).toEqual([]);
  });

  it('should return empty array when architecture property does not exist', () => {
    const artifact = mockCatalogModelArtifact({
      customProperties: {
        other_property: {
          string_value: 'some value',
          metadataType: ModelRegistryMetadataType.STRING,
        },
      },
    });
    const result = getArchitecturesFromArtifacts([artifact]);
    expect(result).toEqual([]);
  });

  it('should return empty array when architecture property has empty string value', () => {
    const artifact = mockCatalogModelArtifact({
      customProperties: {
        architecture: {
          string_value: '',
          metadataType: ModelRegistryMetadataType.STRING,
        },
      },
    });
    const result = getArchitecturesFromArtifacts([artifact]);
    expect(result).toEqual([]);
  });

  it('should return empty array when architecture property has wrong metadata type', () => {
    const artifact = mockCatalogModelArtifact({
      customProperties: {
        architecture: {
          int_value: '123',
          metadataType: ModelRegistryMetadataType.INT,
        },
      },
    });
    const result = getArchitecturesFromArtifacts([artifact]);
    expect(result).toEqual([]);
  });

  it('should return empty array when architecture contains invalid JSON', () => {
    const artifact = mockCatalogModelArtifact({
      customProperties: {
        architecture: {
          string_value: 'not valid json',
          metadataType: ModelRegistryMetadataType.STRING,
        },
      },
    });
    const result = getArchitecturesFromArtifacts([artifact]);
    expect(result).toEqual([]);
  });

  it('should log a warning when architecture JSON is invalid', () => {
    const originalEnv = process.env.NODE_ENV;
    process.env.NODE_ENV = 'development';

    const warnSpy = jest.spyOn(console, 'warn').mockImplementation();
    const artifact = mockCatalogModelArtifact({
      customProperties: {
        architecture: {
          string_value: 'not valid json',
          metadataType: ModelRegistryMetadataType.STRING,
        },
      },
    });

    getArchitecturesFromArtifacts([artifact]);

    expect(warnSpy).toHaveBeenCalledWith(
      'Failed to parse architecture JSON:',
      'not valid json',
      expect.any(Error),
    );
    warnSpy.mockRestore();
    process.env.NODE_ENV = originalEnv;
  });

  it('should return empty array when architecture JSON is not an array', () => {
    const artifact = mockCatalogModelArtifact({
      customProperties: {
        architecture: {
          string_value: '{"arch": "amd64"}',
          metadataType: ModelRegistryMetadataType.STRING,
        },
      },
    });
    const result = getArchitecturesFromArtifacts([artifact]);
    expect(result).toEqual([]);
  });

  it('should return single valid architecture', () => {
    const artifact = mockCatalogModelArtifact({
      customProperties: {
        architecture: {
          string_value: '["amd64"]',
          metadataType: ModelRegistryMetadataType.STRING,
        },
      },
    });
    const result = getArchitecturesFromArtifacts([artifact]);
    expect(result).toEqual(['amd64']);
  });

  it('should return multiple valid architectures', () => {
    const artifact = mockCatalogModelArtifact({
      customProperties: {
        architecture: {
          string_value: '["amd64", "arm64", "s390x", "ppc64le"]',
          metadataType: ModelRegistryMetadataType.STRING,
        },
      },
    });
    const result = getArchitecturesFromArtifacts([artifact]);
    expect(result).toEqual(['amd64', 'arm64', 's390x', 'ppc64le']);
  });

  it('should normalize architectures to lowercase', () => {
    const artifact = mockCatalogModelArtifact({
      customProperties: {
        architecture: {
          string_value: '["AMD64", "ARM64", "S390X", "PPC64LE"]',
          metadataType: ModelRegistryMetadataType.STRING,
        },
      },
    });
    const result = getArchitecturesFromArtifacts([artifact]);
    expect(result).toEqual(['amd64', 'arm64', 's390x', 'ppc64le']);
  });

  it('should handle mixed case architectures', () => {
    const artifact = mockCatalogModelArtifact({
      customProperties: {
        architecture: {
          string_value: '["Amd64", "aRm64", "S390x"]',
          metadataType: ModelRegistryMetadataType.STRING,
        },
      },
    });
    const result = getArchitecturesFromArtifacts([artifact]);
    expect(result).toEqual(['amd64', 'arm64', 's390x']);
  });

  it('should return all architecture types without validation', () => {
    const artifact = mockCatalogModelArtifact({
      customProperties: {
        architecture: {
          string_value: '["amd64", "custom-arch", "arm64", "unknown"]',
          metadataType: ModelRegistryMetadataType.STRING,
        },
      },
    });
    const result = getArchitecturesFromArtifacts([artifact]);
    expect(result).toEqual(['amd64', 'custom-arch', 'arm64', 'unknown']);
  });

  it('should filter out non-string items from JSON array', () => {
    const artifact = mockCatalogModelArtifact({
      customProperties: {
        architecture: {
          string_value: '["amd64", 123, null, "arm64", true, "s390x"]',
          metadataType: ModelRegistryMetadataType.STRING,
        },
      },
    });
    const result = getArchitecturesFromArtifacts([artifact]);
    expect(result).toEqual(['amd64', 'arm64', 's390x']);
  });

  it('should return all architecture strings without validation', () => {
    const artifact = mockCatalogModelArtifact({
      customProperties: {
        architecture: {
          string_value: '["custom1", "custom2", "unknown"]',
          metadataType: ModelRegistryMetadataType.STRING,
        },
      },
    });
    const result = getArchitecturesFromArtifacts([artifact]);
    expect(result).toEqual(['custom1', 'custom2', 'unknown']);
  });

  it('should return empty array when array contains only non-string types', () => {
    const artifact = mockCatalogModelArtifact({
      customProperties: {
        architecture: {
          string_value: '[123, null, true, {}]',
          metadataType: ModelRegistryMetadataType.STRING,
        },
      },
    });
    const result = getArchitecturesFromArtifacts([artifact]);
    expect(result).toEqual([]);
  });

  it('should handle customProperties with multiple properties including architecture', () => {
    const artifact = mockCatalogModelArtifact({
      customProperties: {
        provider: {
          string_value: 'Red Hat',
          metadataType: ModelRegistryMetadataType.STRING,
        },
        architecture: {
          string_value: '["amd64", "arm64"]',
          metadataType: ModelRegistryMetadataType.STRING,
        },
        size: {
          string_value: '8GB',
          metadataType: ModelRegistryMetadataType.STRING,
        },
      },
    });
    const result = getArchitecturesFromArtifacts([artifact]);
    expect(result).toEqual(['amd64', 'arm64']);
  });

  it('should find model artifact among multiple artifacts', () => {
    const metricsArtifact: CatalogArtifacts = {
      artifactType: CatalogArtifactType.metricsArtifact,
      metricsType: MetricsType.performanceMetrics,
    };
    const modelArtifact = mockCatalogModelArtifact({
      customProperties: {
        architecture: {
          string_value: '["amd64", "arm64"]',
          metadataType: ModelRegistryMetadataType.STRING,
        },
      },
    });
    const result = getArchitecturesFromArtifacts([metricsArtifact, modelArtifact]);
    expect(result).toEqual(['amd64', 'arm64']);
  });

  it('should use first model artifact when multiple model artifacts exist', () => {
    const modelArtifact1 = mockCatalogModelArtifact({
      customProperties: {
        architecture: {
          string_value: '["amd64"]',
          metadataType: ModelRegistryMetadataType.STRING,
        },
      },
    });
    const modelArtifact2 = mockCatalogModelArtifact({
      customProperties: {
        architecture: {
          string_value: '["arm64"]',
          metadataType: ModelRegistryMetadataType.STRING,
        },
      },
    });
    const result = getArchitecturesFromArtifacts([modelArtifact1, modelArtifact2]);
    // Should return architectures from first model artifact
    expect(result).toEqual(['amd64']);
  });

  it('should handle empty JSON array', () => {
    const artifact = mockCatalogModelArtifact({
      customProperties: {
        architecture: {
          string_value: '[]',
          metadataType: ModelRegistryMetadataType.STRING,
        },
      },
    });
    const result = getArchitecturesFromArtifacts([artifact]);
    expect(result).toEqual([]);
  });

  it('should handle whitespace in JSON', () => {
    const artifact = mockCatalogModelArtifact({
      customProperties: {
        architecture: {
          string_value: '  [ "amd64" , "arm64" ]  ',
          metadataType: ModelRegistryMetadataType.STRING,
        },
      },
    });
    const result = getArchitecturesFromArtifacts([artifact]);
    expect(result).toEqual(['amd64', 'arm64']);
  });

  it('should return all common CPU architecture types', () => {
    const artifact = mockCatalogModelArtifact({
      customProperties: {
        architecture: {
          string_value: '["amd64", "arm64", "s390x", "ppc64le"]',
          metadataType: ModelRegistryMetadataType.STRING,
        },
      },
    });
    const result = getArchitecturesFromArtifacts([artifact]);
    expect(result).toHaveLength(4);
    expect(result).toContain('amd64');
    expect(result).toContain('arm64');
    expect(result).toContain('s390x');
    expect(result).toContain('ppc64le');
  });

  it('should deduplicate architectures when duplicates exist', () => {
    const artifact = mockCatalogModelArtifact({
      customProperties: {
        architecture: {
          string_value: '["amd64", "AMD64", "amd64"]',
          metadataType: ModelRegistryMetadataType.STRING,
        },
      },
    });
    const result = getArchitecturesFromArtifacts([artifact]);
    expect(result).toEqual(['amd64']); // Should return only one instance
  });

  it('should deduplicate mixed case architectures across multiple values', () => {
    const artifact = mockCatalogModelArtifact({
      customProperties: {
        architecture: {
          string_value: '["amd64", "arm64", "AMD64", "ARM64", "amd64", "s390x"]',
          metadataType: ModelRegistryMetadataType.STRING,
        },
      },
    });
    const result = getArchitecturesFromArtifacts([artifact]);
    expect(result).toEqual(['amd64', 'arm64', 's390x']);
  });
});

describe('getModelName', () => {
  it('should return the part after the slash when model name contains a slash', () => {
    const result = getModelName('repo1/granite-8b-code-instruct');
    expect(result).toBe('granite-8b-code-instruct');
  });

  it('should return the original name when no slash is present', () => {
    const result = getModelName('granite-8b-code-instruct');
    expect(result).toBe('granite-8b-code-instruct');
  });

  it('should return empty string when given an empty string', () => {
    const result = getModelName('');
    expect(result).toBe('');
  });

  it('should handle multiple slashes and return everything after the first slash', () => {
    const result = getModelName('org/repo/model-name');
    expect(result).toBe('repo/model-name');
  });

  it('should return empty string when model name ends with a slash', () => {
    const result = getModelName('repo/');
    expect(result).toBe('');
  });

  it('should return the part after slash when model name starts with a slash', () => {
    const result = getModelName('/model-name');
    expect(result).toBe('model-name');
  });

  it('should handle model names with special characters', () => {
    const result = getModelName('repo1/granite-8b-code-instruct-quantized.w4a16');
    expect(result).toBe('granite-8b-code-instruct-quantized.w4a16');
  });

  it('should handle model names with hyphens and underscores', () => {
    const result = getModelName('my_org/my-model_v1');
    expect(result).toBe('my-model_v1');
  });
});

describe('getCatalogModelTypePropertyForRegistration', () => {
  it('returns model_type metadata when catalog has generative or predictive', () => {
    expect(
      getCatalogModelTypePropertyForRegistration({
        [CatalogModelCustomPropertyKey.MODEL_TYPE]: {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: ModelType.GENERATIVE,
        },
      }),
    ).toEqual({
      [CatalogModelCustomPropertyKey.MODEL_TYPE]: {
        metadataType: ModelRegistryMetadataType.STRING,
        string_value: ModelType.GENERATIVE,
      },
    });
  });

  it('returns model_type metadata when catalog has unknown', () => {
    expect(
      getCatalogModelTypePropertyForRegistration({
        [CatalogModelCustomPropertyKey.MODEL_TYPE]: {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: ModelType.UNKNOWN,
        },
      }),
    ).toEqual({
      [CatalogModelCustomPropertyKey.MODEL_TYPE]: {
        metadataType: ModelRegistryMetadataType.STRING,
        string_value: ModelType.UNKNOWN,
      },
    });
  });

  it('returns empty object when model_type is absent or not a recognized value', () => {
    expect(getCatalogModelTypePropertyForRegistration(undefined)).toEqual({});
    expect(getCatalogModelTypePropertyForRegistration({})).toEqual({});
  });
});

describe('getActiveSourceLabels', () => {
  const createSource = (overrides: Partial<CatalogSource> = {}): CatalogSource => ({
    id: 'source-1',
    name: 'Test Source',
    labels: ['Red Hat'],
    enabled: true,
    status: CatalogSourceStatus.AVAILABLE,
    ...overrides,
  });

  const createSourceList = (items: CatalogSource[] = []): CatalogSourceList => ({
    items,
    size: items.length,
    pageSize: 10,
    nextPageToken: '',
  });

  it('returns empty array when catalogSources is null', () => {
    expect(getActiveSourceLabels(null, null)).toEqual([]);
  });

  it('returns empty array when no sources are enabled or available', () => {
    const sources = createSourceList([
      createSource({
        id: '1',
        enabled: false,
        status: CatalogSourceStatus.DISABLED,
      }),
      createSource({
        id: '2',
        enabled: true,
        status: CatalogSourceStatus.ERROR,
      }),
    ]);
    expect(getActiveSourceLabels(sources, null)).toEqual([]);
  });

  it('returns a single label when only one category exists', () => {
    const sources = createSourceList([
      createSource({
        id: '1',
        labels: ['Red Hat'],
        enabled: true,
        status: CatalogSourceStatus.AVAILABLE,
      }),
    ]);
    expect(getActiveSourceLabels(sources, null)).toEqual(['Red Hat']);
  });

  it('returns multiple labels when multiple categories exist', () => {
    const sources = createSourceList([
      createSource({
        id: '1',
        labels: ['Red Hat'],
        enabled: true,
        status: CatalogSourceStatus.AVAILABLE,
      }),
      createSource({
        id: '2',
        labels: ['Community'],
        enabled: true,
        status: CatalogSourceStatus.AVAILABLE,
      }),
    ]);
    const result = getActiveSourceLabels(sources, null);
    expect(result).toHaveLength(2);
    expect(result).toContain('Red Hat');
    expect(result).toContain('Community');
  });

  it('includes "null" label for sources without labels', () => {
    const sources = createSourceList([
      createSource({
        id: '1',
        labels: ['Red Hat'],
        enabled: true,
        status: CatalogSourceStatus.AVAILABLE,
      }),
      createSource({
        id: '2',
        labels: [],
        enabled: true,
        status: CatalogSourceStatus.AVAILABLE,
      }),
    ]);
    const result = getActiveSourceLabels(sources, null);
    expect(result).toContain('Red Hat');
    expect(result).toContain('null');
  });

  it('excludes disabled sources from active categories', () => {
    const sources = createSourceList([
      createSource({
        id: '1',
        labels: ['Red Hat'],
        enabled: true,
        status: CatalogSourceStatus.AVAILABLE,
      }),
      createSource({
        id: '2',
        labels: ['Excluded'],
        enabled: false,
        status: CatalogSourceStatus.AVAILABLE,
      }),
    ]);
    const result = getActiveSourceLabels(sources, null);
    expect(result).toEqual(['Red Hat']);
  });

  it('excludes sources with error status', () => {
    const sources = createSourceList([
      createSource({
        id: '1',
        labels: ['Red Hat'],
        enabled: true,
        status: CatalogSourceStatus.AVAILABLE,
      }),
      createSource({
        id: '2',
        labels: ['Error Source'],
        enabled: true,
        status: CatalogSourceStatus.ERROR,
      }),
    ]);
    const result = getActiveSourceLabels(sources, null);
    expect(result).toEqual(['Red Hat']);
  });

  it('includes partially-available sources', () => {
    const sources = createSourceList([
      createSource({
        id: '1',
        labels: ['Red Hat'],
        enabled: true,
        status: CatalogSourceStatus.AVAILABLE,
      }),
      createSource({
        id: '2',
        labels: ['Partial'],
        enabled: true,
        status: CatalogSourceStatus.PARTIALLY_AVAILABLE,
      }),
    ]);
    const result = getActiveSourceLabels(sources, null);
    expect(result).toHaveLength(2);
    expect(result).toContain('Red Hat');
    expect(result).toContain('Partial');
  });

  it('deduplicates labels from multiple sources with the same label', () => {
    const sources = createSourceList([
      createSource({
        id: '1',
        labels: ['Red Hat'],
        enabled: true,
        status: CatalogSourceStatus.AVAILABLE,
      }),
      createSource({
        id: '2',
        labels: ['Red Hat'],
        enabled: true,
        status: CatalogSourceStatus.AVAILABLE,
      }),
    ]);
    const result = getActiveSourceLabels(sources, null);
    expect(result).toEqual(['Red Hat']);
  });

  it('returns only "null" label when all sources have no labels', () => {
    const sources = createSourceList([
      createSource({
        id: '1',
        labels: [],
        enabled: true,
        status: CatalogSourceStatus.AVAILABLE,
      }),
      createSource({
        id: '2',
        labels: [],
        enabled: true,
        status: CatalogSourceStatus.AVAILABLE,
      }),
    ]);
    const result = getActiveSourceLabels(sources, null);
    expect(result).toEqual(['null']);
  });

  it('orders labels by catalogLabels priority when catalogLabels is provided', () => {
    const sources = createSourceList([
      createSource({
        id: '1',
        labels: ['Community'],
        enabled: true,
        status: CatalogSourceStatus.AVAILABLE,
      }),
      createSource({
        id: '2',
        labels: ['Red Hat'],
        enabled: true,
        status: CatalogSourceStatus.AVAILABLE,
      }),
      createSource({
        id: '3',
        labels: ['Partner'],
        enabled: true,
        status: CatalogSourceStatus.AVAILABLE,
      }),
    ]);

    const createLabel = (name: string | null): CatalogLabel => ({
      name,
      displayName: name ?? undefined,
    });

    const catalogLabels: CatalogLabelList = {
      items: [createLabel('Red Hat'), createLabel('Partner'), createLabel('Community')],
      size: 3,
      pageSize: 10,
      nextPageToken: '',
    };

    const result = getActiveSourceLabels(sources, catalogLabels);
    expect(result).toEqual(['Red Hat', 'Partner', 'Community']);
  });
});
