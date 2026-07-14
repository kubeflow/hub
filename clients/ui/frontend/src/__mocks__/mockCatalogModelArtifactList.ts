/* eslint-disable camelcase */
import {
  CatalogArtifacts,
  CatalogArtifactList,
  CatalogModelArtifact,
  CatalogArtifactType,
  CatalogPerformanceArtifactList,
  MetricsType,
  CatalogPerformanceMetricsArtifact,
  CatalogAccuracyMetricsArtifact,
} from '~/app/modelCatalogTypes';
import { ModelRegistryMetadataType } from '~/app/types';
import { UseCaseOptionValue } from '~/concepts/modelCatalog/const';

export const mockCatalogModelArtifact = (
  partial?: Partial<CatalogModelArtifact>,
): CatalogArtifacts => ({
  artifactType: CatalogArtifactType.modelArtifact,
  createTimeSinceEpoch: '1739210683000',
  lastUpdateTimeSinceEpoch: '1739210683000',
  uri: '',
  customProperties: {},
  ...partial,
});

export const mockCatalogAccuracyMetricsArtifact = (
  partial?: Partial<CatalogAccuracyMetricsArtifact>,
): CatalogAccuracyMetricsArtifact => ({
  artifactType: CatalogArtifactType.metricsArtifact,
  metricsType: MetricsType.accuracyMetrics,
  createTimeSinceEpoch: '1739210683000',
  lastUpdateTimeSinceEpoch: '1739210683000',
  customProperties: {
    overall_average: {
      metadataType: ModelRegistryMetadataType.DOUBLE,
      double_value: 0.582439,
    },
    arc_v1: {
      metadataType: ModelRegistryMetadataType.DOUBLE,
      double_value: 0.659556,
    },
  },
  ...partial,
});

export const mockCatalogPerformanceMetricsArtifact = (
  partial?: Partial<CatalogPerformanceMetricsArtifact>,
): CatalogPerformanceMetricsArtifact => ({
  artifactType: CatalogArtifactType.metricsArtifact,
  metricsType: MetricsType.performanceMetrics,
  createTimeSinceEpoch: '1739210683000',
  lastUpdateTimeSinceEpoch: '1739210683000',
  customProperties: {
    config_id: {
      metadataType: ModelRegistryMetadataType.STRING,
      string_value: '0055d94f6a542f6932cac5dfa5ffdd38',
    },
    hardware_count: {
      metadataType: ModelRegistryMetadataType.INT,
      int_value: '2',
    },
    hardware_type: {
      metadataType: ModelRegistryMetadataType.STRING,
      string_value: 'H100-80',
    },
    hardware_configuration: {
      metadataType: ModelRegistryMetadataType.STRING,
      string_value: '2 x H100-80',
    },
    requests_per_second: {
      metadataType: ModelRegistryMetadataType.DOUBLE,
      double_value: 7,
    },
    ttft_mean: {
      metadataType: ModelRegistryMetadataType.DOUBLE,
      double_value: 35.48818160947744,
    },
    ttft_p90: {
      metadataType: ModelRegistryMetadataType.DOUBLE,
      double_value: 51.55777931213379,
    },
    ttft_p95: {
      metadataType: ModelRegistryMetadataType.DOUBLE,
      double_value: 61.26761436462402,
    },
    ttft_p99: {
      metadataType: ModelRegistryMetadataType.DOUBLE,
      double_value: 72.95823097229004,
    },
    e2e_mean: {
      metadataType: ModelRegistryMetadataType.DOUBLE,
      double_value: 1994.480013381083,
    },
    use_case: {
      metadataType: ModelRegistryMetadataType.STRING,
      // Use CHATBOT as default to match DEFAULT_PERFORMANCE_FILTERS_QUERY_NAME
      string_value: UseCaseOptionValue.CHATBOT,
    },
  },
  ...partial,
});

export const mockCatalogModelArtifactList = (
  partial?: Partial<CatalogModelArtifact>,
): CatalogArtifactList => ({
  items: [mockCatalogModelArtifact({})],
  pageSize: 10,
  size: 15,
  nextPageToken: '',
  ...partial,
});

export const mockCatalogPerformanceMetricsArtifactList = (
  partial?: Partial<CatalogPerformanceMetricsArtifact>,
): CatalogPerformanceArtifactList => ({
  items: [
    // First artifact with base values - uses CHATBOT to match default filters
    mockCatalogPerformanceMetricsArtifact({
      customProperties: {
        config_id: {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: 'config-001',
        },
        hardware_count: { metadataType: ModelRegistryMetadataType.INT, int_value: '2' },
        hardware_type: { metadataType: ModelRegistryMetadataType.STRING, string_value: 'H100-80' },
        requests_per_second: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 7 },
        ttft_mean: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 35.48 },
        ttft_p90: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 51.56 },
        ttft_p95: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 61.27 },
        ttft_p99: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 72.96 },
        e2e_mean: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 1994.48 },
        e2e_p90: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 2644.6 },
        itl_mean: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 7.68 },
        itl_p90: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 7.78 },
        use_case: {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: UseCaseOptionValue.CHATBOT,
        },
        ...partial?.customProperties,
      },
    }),
    // Second artifact with different latency values
    mockCatalogPerformanceMetricsArtifact({
      customProperties: {
        config_id: {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: 'config-002',
        },
        hardware_count: { metadataType: ModelRegistryMetadataType.INT, int_value: '4' },
        hardware_type: { metadataType: ModelRegistryMetadataType.STRING, string_value: 'RTX 4090' },
        requests_per_second: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 10 },
        ttft_mean: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 67.15 },
        ttft_p90: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 82.34 },
        ttft_p95: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 95.67 },
        ttft_p99: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 112.45 },
        e2e_mean: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 2450.32 },
        e2e_p90: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 3200.11 },
        itl_mean: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 9.1 },
        itl_p90: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 11.23 },
        use_case: {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: UseCaseOptionValue.RAG,
        },
      },
    }),
    // Third artifact with CODE_FIXING workload type for testing filter changes
    mockCatalogPerformanceMetricsArtifact({
      customProperties: {
        config_id: {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: 'config-003',
        },
        hardware_count: { metadataType: ModelRegistryMetadataType.INT, int_value: '8' },
        hardware_type: { metadataType: ModelRegistryMetadataType.STRING, string_value: 'A100' },
        requests_per_second: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 15 },
        ttft_mean: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 42.12 },
        ttft_p90: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 58.45 },
        ttft_p95: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 68.78 },
        ttft_p99: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 80.91 },
        e2e_mean: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 1850.67 },
        e2e_p90: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 2400.55 },
        itl_mean: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 6.8 },
        itl_p90: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 8.12 },
        use_case: {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: UseCaseOptionValue.CODE_FIXING,
        },
      },
    }),
  ],
  pageSize: 10,
  size: 3,
  nextPageToken: '',
});

export const mockCatalogAccuracyMetricsArtifactList = (
  partial?: Partial<CatalogAccuracyMetricsArtifact>,
): CatalogArtifactList => ({
  items: [mockCatalogAccuracyMetricsArtifact({}), mockCatalogModelArtifact({})],
  pageSize: 10,
  size: 15,
  nextPageToken: '',
  ...partial,
});

// Performance artifact with computed properties (when targetRPS is provided)
export const mockCatalogPerformanceMetricsArtifactWithRPS = (
  targetRPS: number,
  partial?: Partial<CatalogPerformanceMetricsArtifact>,
): CatalogPerformanceMetricsArtifact => {
  const baseArtifact = mockCatalogPerformanceMetricsArtifact(partial);
  const rps = baseArtifact.customProperties?.requests_per_second?.double_value || 7;
  const replicas = Math.ceil(targetRPS / rps);

  return {
    ...baseArtifact,
    customProperties: {
      ...baseArtifact.customProperties,
      replicas: {
        metadataType: ModelRegistryMetadataType.INT,
        int_value: String(replicas),
      },
      total_requests_per_second: {
        metadataType: ModelRegistryMetadataType.DOUBLE,
        double_value: replicas * rps,
      },
    },
  };
};

// Mock for Pareto-filtered (orderBy=RECOMMENDED) performance artifacts
export const mockParetoFilteredPerformanceArtifactList = (
  targetRPS?: number,
): CatalogPerformanceArtifactList => ({
  items: [
    mockCatalogPerformanceMetricsArtifactWithRPS(targetRPS || 100, {
      customProperties: {
        config_id: {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: 'pareto-optimal-1',
        },
        hardware_count: { metadataType: ModelRegistryMetadataType.INT, int_value: '1' },
        hardware_type: { metadataType: ModelRegistryMetadataType.STRING, string_value: 'H100-80' },
        requests_per_second: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 50 },
        ttft_p90: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 35 },
        use_case: {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: UseCaseOptionValue.CHATBOT,
        },
      },
    }),
    mockCatalogPerformanceMetricsArtifactWithRPS(targetRPS || 100, {
      customProperties: {
        config_id: {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: 'pareto-optimal-2',
        },
        hardware_count: { metadataType: ModelRegistryMetadataType.INT, int_value: '2' },
        hardware_type: { metadataType: ModelRegistryMetadataType.STRING, string_value: 'A100-80' },
        requests_per_second: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 30 },
        ttft_p90: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 28 },
        use_case: {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: UseCaseOptionValue.RAG,
        },
      },
    }),
  ],
  pageSize: 10,
  size: 2,
  nextPageToken: '',
});

// Mock for performance artifacts filtered by a specific workload type (use_case)
export const mockFilteredPerformanceArtifactsByWorkloadType = (
  workloadType: UseCaseOptionValue,
): CatalogPerformanceArtifactList => ({
  items: [
    mockCatalogPerformanceMetricsArtifact({
      customProperties: {
        config_id: {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: `filtered-${workloadType}-1`,
        },
        hardware_count: { metadataType: ModelRegistryMetadataType.INT, int_value: '2' },
        hardware_type: { metadataType: ModelRegistryMetadataType.STRING, string_value: 'H100-80' },
        requests_per_second: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 7 },
        ttft_mean: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 35.48 },
        ttft_p90: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 51.55 },
        use_case: {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: workloadType,
        },
      },
    }),
    mockCatalogPerformanceMetricsArtifact({
      customProperties: {
        config_id: {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: `filtered-${workloadType}-2`,
        },
        hardware_count: { metadataType: ModelRegistryMetadataType.INT, int_value: '4' },
        hardware_type: { metadataType: ModelRegistryMetadataType.STRING, string_value: 'A100-80' },
        requests_per_second: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 12 },
        ttft_mean: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 28.32 },
        ttft_p90: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 42.18 },
        use_case: {
          metadataType: ModelRegistryMetadataType.STRING,
          string_value: workloadType,
        },
      },
    }),
  ],
  pageSize: 10,
  size: 2,
  nextPageToken: '',
});

// Mock for performance artifacts with multiple workload types (unfiltered list)
export const mockMultipleWorkloadTypePerformanceArtifactList =
  (): CatalogPerformanceArtifactList => ({
    items: [
      mockCatalogPerformanceMetricsArtifact({
        customProperties: {
          config_id: {
            metadataType: ModelRegistryMetadataType.STRING,
            string_value: 'multi-workload-1',
          },
          hardware_count: { metadataType: ModelRegistryMetadataType.INT, int_value: '2' },
          hardware_type: {
            metadataType: ModelRegistryMetadataType.STRING,
            string_value: 'H100-80',
          },
          requests_per_second: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 7 },
          ttft_p90: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 51.55 },
          use_case: {
            metadataType: ModelRegistryMetadataType.STRING,
            string_value: UseCaseOptionValue.CODE_FIXING,
          },
        },
      }),
      mockCatalogPerformanceMetricsArtifact({
        customProperties: {
          config_id: {
            metadataType: ModelRegistryMetadataType.STRING,
            string_value: 'multi-workload-2',
          },
          hardware_count: { metadataType: ModelRegistryMetadataType.INT, int_value: '4' },
          hardware_type: {
            metadataType: ModelRegistryMetadataType.STRING,
            string_value: 'A100-80',
          },
          requests_per_second: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 12 },
          ttft_p90: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 42.18 },
          use_case: {
            metadataType: ModelRegistryMetadataType.STRING,
            string_value: UseCaseOptionValue.CHATBOT,
          },
        },
      }),
      mockCatalogPerformanceMetricsArtifact({
        customProperties: {
          config_id: {
            metadataType: ModelRegistryMetadataType.STRING,
            string_value: 'multi-workload-3',
          },
          hardware_count: { metadataType: ModelRegistryMetadataType.INT, int_value: '1' },
          hardware_type: { metadataType: ModelRegistryMetadataType.STRING, string_value: 'L40S' },
          requests_per_second: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 5 },
          ttft_p90: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 65.23 },
          use_case: {
            metadataType: ModelRegistryMetadataType.STRING,
            string_value: UseCaseOptionValue.RAG,
          },
        },
      }),
      mockCatalogPerformanceMetricsArtifact({
        customProperties: {
          config_id: {
            metadataType: ModelRegistryMetadataType.STRING,
            string_value: 'multi-workload-4',
          },
          hardware_count: { metadataType: ModelRegistryMetadataType.INT, int_value: '2' },
          hardware_type: { metadataType: ModelRegistryMetadataType.STRING, string_value: 'MI300X' },
          requests_per_second: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 9 },
          ttft_p90: { metadataType: ModelRegistryMetadataType.DOUBLE, double_value: 48.77 },
          use_case: {
            metadataType: ModelRegistryMetadataType.STRING,
            string_value: UseCaseOptionValue.LONG_RAG,
          },
        },
      }),
    ],
    pageSize: 10,
    size: 4,
    nextPageToken: '',
  });
