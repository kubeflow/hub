/* eslint-disable camelcase */
import {
  CatalogPerformanceMetricsArtifact,
  CatalogAccuracyMetricsArtifact,
  CatalogArtifactType,
  MetricsType,
  CatalogModel,
} from '~/app/modelCatalogTypes';
import { ModelRegistryMetadataType } from '~/app/types';
import {
  extractPerformanceMetrics,
  // calculateAverageAccuracy, // NOTE: overall_average is currently omitted from the API and will be restored
  extractValidatedModelMetrics,
  sortModelsWithCurrentFirst,
} from '~/app/pages/modelCatalog/utils/validatedModelUtils';
import {
  LatencyMetric,
  LatencyPercentile,
  getLatencyFilterKey,
} from '~/concepts/modelCatalog/const';

describe('validatedModelUtils', () => {
  const createMockPerformanceArtifact = (
    hardwareType: string,
    hardwareCount: number,
    rpsPerReplica: number,
    ttftMean: number,
  ): CatalogPerformanceMetricsArtifact => ({
    artifactType: CatalogArtifactType.metricsArtifact,
    metricsType: MetricsType.performanceMetrics,
    createTimeSinceEpoch: '1739210683000',
    lastUpdateTimeSinceEpoch: '1739210683000',
    customProperties: {
      hardware_type: {
        metadataType: ModelRegistryMetadataType.STRING,
        string_value: hardwareType,
      },
      hardware_count: {
        metadataType: ModelRegistryMetadataType.INT,
        int_value: hardwareCount.toString(),
      },
      requests_per_second: {
        metadataType: ModelRegistryMetadataType.DOUBLE,
        double_value: rpsPerReplica,
      },
      ttft_mean: {
        metadataType: ModelRegistryMetadataType.DOUBLE,
        double_value: ttftMean,
      },
    },
  });

  const createMockAccuracyArtifact =
    () // overallAverage: number, // NOTE: overall_average is currently omitted from the API and will be restored
    : CatalogAccuracyMetricsArtifact => ({
      artifactType: CatalogArtifactType.metricsArtifact,
      metricsType: MetricsType.accuracyMetrics,
      createTimeSinceEpoch: '1739210683000',
      lastUpdateTimeSinceEpoch: '1739210683000',
      customProperties: {
        // overall_average: { // NOTE: overall_average is currently omitted from the API and will be restored
        //   metadataType: ModelRegistryMetadataType.DOUBLE,
        //   double_value: overallAverage,
        // },
      },
    });

  describe('extractPerformanceMetrics', () => {
    it('should extract performance metrics from a single artifact', () => {
      const artifact = createMockPerformanceArtifact('H100-80', 2, 3.5, 1200);

      const result = extractPerformanceMetrics(artifact);

      expect(result).toMatchObject({
        hardwareType: 'H100-80',
        hardwareCount: '2',
        rpsPerReplica: 3.5,
        ttftMean: 1200,
      });
      expect(result.latencyMetrics).toEqual({
        [getLatencyFilterKey(LatencyMetric.TTFT, LatencyPercentile.Mean)]: 1200,
      });
    });

    it('should handle missing properties with default values', () => {
      const artifact: CatalogPerformanceMetricsArtifact = {
        artifactType: CatalogArtifactType.metricsArtifact,
        metricsType: MetricsType.performanceMetrics,
        createTimeSinceEpoch: '1739210683000',
        lastUpdateTimeSinceEpoch: '1739210683000',
        customProperties: {},
      };

      const result = extractPerformanceMetrics(artifact);

      expect(result).toMatchObject({
        hardwareType: 'H100-80',
        hardwareCount: '1',
        rpsPerReplica: 1,
        ttftMean: 1428,
      });
      expect(result.latencyMetrics).toEqual({});
    });
  });

  // NOTE: overall_average is currently omitted from the API and will be restored
  // describe('calculateAverageAccuracy', () => {
  //   it('should calculate average accuracy from multiple artifacts', () => {
  //     const artifacts = [
  //       createMockAccuracyArtifact(50.0),
  //       createMockAccuracyArtifact(60.0),
  //       createMockAccuracyArtifact(70.0),
  //     ];
  //
  //     const result = calculateAverageAccuracy(artifacts);
  //
  //     expect(result).toBe(60.0);
  //   });
  //
  //   it('should handle single artifact', () => {
  //     const artifacts = [createMockAccuracyArtifact(75.5)];
  //
  //     const result = calculateAverageAccuracy(artifacts);
  //
  //     expect(result).toBe(75.5);
  //   });
  //
  //   it('should handle empty array with default fallback', () => {
  //     const result = calculateAverageAccuracy([]);
  //
  //     expect(result).toBe(53.9);
  //   });
  //
  //   it('should round to 1 decimal place', () => {
  //     const artifacts = [
  //       createMockAccuracyArtifact(50.0),
  //       createMockAccuracyArtifact(60.0),
  //       createMockAccuracyArtifact(70.0),
  //       createMockAccuracyArtifact(80.0),
  //     ];
  //
  //     const result = calculateAverageAccuracy(artifacts);
  //
  //     expect(result).toBe(65.0);
  //   });
  //
  //   it('should handle artifacts with missing accuracy values', () => {
  //     const artifacts = [
  //       createMockAccuracyArtifact(50.0),
  //       {
  //         ...createMockAccuracyArtifact(0),
  //         customProperties: {},
  //       },
  //       createMockAccuracyArtifact(70.0),
  //     ];
  //
  //     const result = calculateAverageAccuracy(artifacts);
  //
  //     expect(result).toBe(40.0); // (50 + 0 + 70) / 3
  //   });
  // });

  describe('extractValidatedModelMetrics', () => {
    it('should extract metrics from arrays of artifacts with specific performance index', () => {
      const performanceArtifacts = [
        createMockPerformanceArtifact('A100-80', 1, 2.0, 1000),
        createMockPerformanceArtifact('H100-80', 2, 3.5, 1200),
        createMockPerformanceArtifact('A100-40', 4, 5.0, 1500),
      ];

      const accuracyArtifacts = [
        createMockAccuracyArtifact(/* 50.0 */),
        createMockAccuracyArtifact(/* 60.0 */),
        createMockAccuracyArtifact(/* 70.0 */),
      ];

      const result = extractValidatedModelMetrics(performanceArtifacts, accuracyArtifacts, 1);

      expect(result).toMatchObject({
        // accuracy: 60.0, // NOTE: overall_average is currently omitted from the API and will be restored
        hardwareType: 'H100-80',
        hardwareCount: '2',
        rpsPerReplica: 3.5,
        ttftMean: 1200,
      });
    });

    it('should use first performance artifact by default', () => {
      const performanceArtifacts = [
        createMockPerformanceArtifact('A100-80', 1, 2.0, 1000),
        createMockPerformanceArtifact('H100-80', 2, 3.5, 1200),
      ];

      const accuracyArtifacts = [createMockAccuracyArtifact(/* 75.0 */)];

      const result = extractValidatedModelMetrics(performanceArtifacts, accuracyArtifacts);

      expect(result).toMatchObject({
        // accuracy: 75.0, // NOTE: overall_average is currently omitted from the API and will be restored
        hardwareType: 'A100-80',
        hardwareCount: '1',
        rpsPerReplica: 2.0,
        ttftMean: 1000,
      });
    });

    it('should return null when performance artifacts array is empty', () => {
      const performanceArtifacts: CatalogPerformanceMetricsArtifact[] = [];
      const accuracyArtifacts = [createMockAccuracyArtifact(/* 80.0 */)];

      const result = extractValidatedModelMetrics(performanceArtifacts, accuracyArtifacts);

      expect(result).toBeNull();
    });

    it('should return null when performance index is invalid', () => {
      const performanceArtifacts = [createMockPerformanceArtifact('A100-80', 1, 2.0, 1000)];
      const accuracyArtifacts = [createMockAccuracyArtifact(/* 90.0 */)];

      const result = extractValidatedModelMetrics(performanceArtifacts, accuracyArtifacts, 5);

      expect(result).toBeNull();
    });
  });

  describe('sortModelsWithCurrentFirst', () => {
    const createMockModel = (name: string): CatalogModel =>
      ({
        name,
        sourceId: 'test-source',
      }) as CatalogModel;

    it('should return empty array when given empty array', () => {
      const result = sortModelsWithCurrentFirst([], 'model-1');

      expect(result).toEqual([]);
    });

    it('should put current model first', () => {
      const models = [
        createMockModel('model-a'),
        createMockModel('model-b'),
        createMockModel('current-model'),
        createMockModel('model-c'),
      ];

      const result = sortModelsWithCurrentFirst(models, 'current-model');

      expect(result[0].name).toBe('current-model');
    });

    it('should limit results to specified limit', () => {
      const models = [
        createMockModel('model-1'),
        createMockModel('model-2'),
        createMockModel('model-3'),
        createMockModel('model-4'),
        createMockModel('model-5'),
        createMockModel('model-6'),
      ];

      const result = sortModelsWithCurrentFirst(models, 'model-3', 4);

      expect(result).toHaveLength(4);
    });

    it('should use default limit of 4 when not specified', () => {
      const models = [
        createMockModel('model-1'),
        createMockModel('model-2'),
        createMockModel('model-3'),
        createMockModel('model-4'),
        createMockModel('model-5'),
      ];

      const result = sortModelsWithCurrentFirst(models, 'model-1');

      expect(result).toHaveLength(4);
    });

    it('should include current model in limited results even if it was at end', () => {
      const models = [
        createMockModel('model-1'),
        createMockModel('model-2'),
        createMockModel('model-3'),
        createMockModel('model-4'),
        createMockModel('model-5'),
        createMockModel('current-model'),
      ];

      const result = sortModelsWithCurrentFirst(models, 'current-model', 4);

      expect(result[0].name).toBe('current-model');
      expect(result).toHaveLength(4);
    });

    it('should preserve relative order of other models after current', () => {
      const models = [
        createMockModel('model-a'),
        createMockModel('model-b'),
        createMockModel('current-model'),
        createMockModel('model-c'),
      ];

      const result = sortModelsWithCurrentFirst(models, 'current-model');

      expect(result.map((m) => m.name)).toEqual(['current-model', 'model-a', 'model-b', 'model-c']);
    });

    it('should handle case when current model is not in the list', () => {
      const models = [
        createMockModel('model-a'),
        createMockModel('model-b'),
        createMockModel('model-c'),
      ];

      const result = sortModelsWithCurrentFirst(models, 'non-existent');

      expect(result).toHaveLength(3);
      expect(result.map((m) => m.name)).toEqual(['model-a', 'model-b', 'model-c']);
    });

    it('should not mutate original array', () => {
      const models = [
        createMockModel('model-a'),
        createMockModel('current-model'),
        createMockModel('model-b'),
      ];
      const originalOrder = models.map((m) => m.name);

      sortModelsWithCurrentFirst(models, 'current-model');

      expect(models.map((m) => m.name)).toEqual(originalOrder);
    });

    it('should handle single model that is current', () => {
      const models = [createMockModel('only-model')];

      const result = sortModelsWithCurrentFirst(models, 'only-model');

      expect(result).toHaveLength(1);
      expect(result[0].name).toBe('only-model');
    });

    it('should handle single model that is not current', () => {
      const models = [createMockModel('only-model')];

      const result = sortModelsWithCurrentFirst(models, 'different-model');

      expect(result).toHaveLength(1);
      expect(result[0].name).toBe('only-model');
    });
  });
});
