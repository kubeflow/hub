import { SortableData } from 'mod-arch-shared';
import {
  CatalogPerformanceMetricsArtifact,
  PerformanceMetricsCustomProperties,
} from '~/app/modelCatalogTypes';
import { getWorkloadType } from '~/app/pages/modelCatalog/utils/performanceMetricsUtils';
import { getDoubleValue, getIntValue, getStringValue } from '~/app/utils';
import {
  LatencyMetric,
  LatencyPercentile,
  getLatencyPropertyKey,
  PerformancePropertyKey,
} from '~/concepts/modelCatalog/const';

export type HardwareConfigColumnField =
  | keyof PerformanceMetricsCustomProperties
  | 'cold_start_load_time'
  | 'runtime_command';

export type HardwareConfigColumn = Omit<
  SortableData<CatalogPerformanceMetricsArtifact>,
  'field'
> & { field: HardwareConfigColumnField };

/*Non-breaking space constant (U+00A0) used to selectively control word wrap in column labels.
This prevents word wrapping into 3 lines (e.g., keeps "TTFT Latency" together instead of "TTFT\nLatency\nMean").
*/
const NBSP = '\u00A0';

const TPS_INFO = {
  popover:
    'Throughput measured in tokens per second (tok/s). Higher values indicate faster token generation.',
  popoverProps: {
    position: 'left' as const,
  },
};

export const hardwareConfigColumns: HardwareConfigColumn[] = [
  {
    field: PerformancePropertyKey.HARDWARE_CONFIGURATION,
    label: 'Hardware configuration',
    info: {
      popover:
        'The hardware configuration used for benchmarking, including the GPU type and the number of GPUs per replica.',
      popoverProps: {
        position: 'left',
      },
    },
    sortable: (
      a: CatalogPerformanceMetricsArtifact,
      b: CatalogPerformanceMetricsArtifact,
    ): number =>
      getStringValue(
        a.customProperties,
        PerformancePropertyKey.HARDWARE_CONFIGURATION,
      ).localeCompare(
        getStringValue(b.customProperties, PerformancePropertyKey.HARDWARE_CONFIGURATION),
      ),
    isStickyColumn: true,
    stickyMinWidth: '200px',
    stickyLeftOffset: '0',
    modifier: 'wrap',
  },
  {
    field: PerformancePropertyKey.USE_CASE,
    label: 'Workload type',
    sortable: (
      a: CatalogPerformanceMetricsArtifact,
      b: CatalogPerformanceMetricsArtifact,
    ): number => getWorkloadType(a).localeCompare(getWorkloadType(b)),
    isStickyColumn: true,
    stickyMinWidth: '132px',
    stickyLeftOffset: '200px',
    modifier: 'wrap',
    hasRightBorder: true,
  },
  {
    field: 'replicas',
    label: 'Replicas',
    info: {
      popover:
        'The number of replicas required to support the specified maximum requests per second (Max RPS).',
      popoverProps: {
        position: 'left',
      },
    },
    sortable: (
      a: CatalogPerformanceMetricsArtifact,
      b: CatalogPerformanceMetricsArtifact,
    ): number =>
      getIntValue(a.customProperties, 'replicas') - getIntValue(b.customProperties, 'replicas'),
    width: 20,
    modifier: 'wrap',
  },
  {
    field: PerformancePropertyKey.REQUESTS_PER_SECOND,
    label: `RPS${NBSP}per Replica`,
    sortable: (
      a: CatalogPerformanceMetricsArtifact,
      b: CatalogPerformanceMetricsArtifact,
    ): number =>
      getDoubleValue(a.customProperties, PerformancePropertyKey.REQUESTS_PER_SECOND) -
      getDoubleValue(b.customProperties, PerformancePropertyKey.REQUESTS_PER_SECOND),
    width: 20,
    modifier: 'wrap',
  },
  {
    field: 'total_requests_per_second',
    label: 'Total RPS',
    info: {
      popover:
        'The total traffic capacity, measured in requests per second, supported by the recommended number of replicas.',
      popoverProps: {
        position: 'left',
        footerContent:
          'Note: Because we cannot use fractional GPUs, requirements are rounded up (example, 1.2 becomes 2), which might result in a higher total RPS than your defined maximum.',
      },
    },
    sortable: (
      a: CatalogPerformanceMetricsArtifact,
      b: CatalogPerformanceMetricsArtifact,
    ): number =>
      getDoubleValue(a.customProperties, 'total_requests_per_second') -
      getDoubleValue(b.customProperties, 'total_requests_per_second'),
    width: 20,
    modifier: 'wrap',
  },
  {
    field: 'ttft_mean',
    label: `TTFT${NBSP}Latency Mean`,
    sortable: (
      a: CatalogPerformanceMetricsArtifact,
      b: CatalogPerformanceMetricsArtifact,
    ): number =>
      getDoubleValue(a.customProperties, 'ttft_mean') -
      getDoubleValue(b.customProperties, 'ttft_mean'),
    width: 20,
    modifier: 'wrap',
  },
  {
    field: 'ttft_p90',
    label: `TTFT${NBSP}Latency P90`,
    sortable: (
      a: CatalogPerformanceMetricsArtifact,
      b: CatalogPerformanceMetricsArtifact,
    ): number =>
      getDoubleValue(a.customProperties, 'ttft_p90') -
      getDoubleValue(b.customProperties, 'ttft_p90'),
    width: 20,
    modifier: 'wrap',
  },
  {
    field: 'ttft_p95',
    label: `TTFT${NBSP}Latency P95`,
    sortable: (
      a: CatalogPerformanceMetricsArtifact,
      b: CatalogPerformanceMetricsArtifact,
    ): number =>
      getDoubleValue(a.customProperties, 'ttft_p95') -
      getDoubleValue(b.customProperties, 'ttft_p95'),
    width: 20,
    modifier: 'wrap',
  },
  {
    field: 'ttft_p99',
    label: `TTFT${NBSP}Latency P99`,
    sortable: (
      a: CatalogPerformanceMetricsArtifact,
      b: CatalogPerformanceMetricsArtifact,
    ): number =>
      getDoubleValue(a.customProperties, 'ttft_p99') -
      getDoubleValue(b.customProperties, 'ttft_p99'),
    width: 20,
    modifier: 'wrap',
  },
  {
    field: 'e2e_mean',
    label: `E2E${NBSP}Latency Mean`,
    sortable: (
      a: CatalogPerformanceMetricsArtifact,
      b: CatalogPerformanceMetricsArtifact,
    ): number =>
      getDoubleValue(a.customProperties, 'e2e_mean') -
      getDoubleValue(b.customProperties, 'e2e_mean'),
    width: 20,
    modifier: 'wrap',
  },
  {
    field: 'e2e_p90',
    label: `E2E${NBSP}Latency P90`,
    sortable: (
      a: CatalogPerformanceMetricsArtifact,
      b: CatalogPerformanceMetricsArtifact,
    ): number =>
      getDoubleValue(a.customProperties, 'e2e_p90') - getDoubleValue(b.customProperties, 'e2e_p90'),
    width: 20,
    modifier: 'wrap',
  },
  {
    field: 'e2e_p95',
    label: `E2E${NBSP}Latency P95`,
    sortable: (
      a: CatalogPerformanceMetricsArtifact,
      b: CatalogPerformanceMetricsArtifact,
    ): number =>
      getDoubleValue(a.customProperties, 'e2e_p95') - getDoubleValue(b.customProperties, 'e2e_p95'),
    width: 20,
    modifier: 'wrap',
  },
  {
    field: 'e2e_p99',
    label: `E2E${NBSP}Latency P99`,
    sortable: (
      a: CatalogPerformanceMetricsArtifact,
      b: CatalogPerformanceMetricsArtifact,
    ): number =>
      getDoubleValue(a.customProperties, 'e2e_p99') - getDoubleValue(b.customProperties, 'e2e_p99'),
    width: 20,
    modifier: 'wrap',
  },
  {
    field: 'itl_mean',
    label: `ITL${NBSP}Latency Mean`,
    sortable: (
      a: CatalogPerformanceMetricsArtifact,
      b: CatalogPerformanceMetricsArtifact,
    ): number =>
      getDoubleValue(a.customProperties, 'itl_mean') -
      getDoubleValue(b.customProperties, 'itl_mean'),
    width: 20,
    modifier: 'wrap',
  },
  {
    field: 'itl_p90',
    label: `ITL${NBSP}Latency P90`,
    sortable: (
      a: CatalogPerformanceMetricsArtifact,
      b: CatalogPerformanceMetricsArtifact,
    ): number =>
      getDoubleValue(a.customProperties, 'itl_p90') - getDoubleValue(b.customProperties, 'itl_p90'),
    width: 20,
    modifier: 'wrap',
  },
  {
    field: 'itl_p95',
    label: `ITL${NBSP}Latency P95`,
    sortable: (
      a: CatalogPerformanceMetricsArtifact,
      b: CatalogPerformanceMetricsArtifact,
    ): number =>
      getDoubleValue(a.customProperties, 'itl_p95') - getDoubleValue(b.customProperties, 'itl_p95'),
    width: 20,
    modifier: 'wrap',
  },
  {
    field: 'itl_p99',
    label: `ITL${NBSP}Latency P99`,
    sortable: (
      a: CatalogPerformanceMetricsArtifact,
      b: CatalogPerformanceMetricsArtifact,
    ): number =>
      getDoubleValue(a.customProperties, 'itl_p99') - getDoubleValue(b.customProperties, 'itl_p99'),
    width: 20,
    modifier: 'wrap',
  },
  {
    field: 'tps_mean',
    label: `TPS${NBSP}Mean`,
    info: TPS_INFO,
    sortable: (
      a: CatalogPerformanceMetricsArtifact,
      b: CatalogPerformanceMetricsArtifact,
    ): number =>
      getDoubleValue(a.customProperties, 'tps_mean') -
      getDoubleValue(b.customProperties, 'tps_mean'),
    width: 20,
    modifier: 'wrap',
  },
  {
    field: 'tps_p90',
    label: `TPS${NBSP}P90`,
    info: TPS_INFO,
    sortable: (
      a: CatalogPerformanceMetricsArtifact,
      b: CatalogPerformanceMetricsArtifact,
    ): number =>
      getDoubleValue(a.customProperties, 'tps_p90') - getDoubleValue(b.customProperties, 'tps_p90'),
    width: 20,
    modifier: 'wrap',
  },
  {
    field: 'tps_p95',
    label: `TPS${NBSP}P95`,
    info: TPS_INFO,
    sortable: (
      a: CatalogPerformanceMetricsArtifact,
      b: CatalogPerformanceMetricsArtifact,
    ): number =>
      getDoubleValue(a.customProperties, 'tps_p95') - getDoubleValue(b.customProperties, 'tps_p95'),
    width: 20,
    modifier: 'wrap',
  },
  {
    field: 'tps_p99',
    label: `TPS${NBSP}P99`,
    info: TPS_INFO,
    sortable: (
      a: CatalogPerformanceMetricsArtifact,
      b: CatalogPerformanceMetricsArtifact,
    ): number =>
      getDoubleValue(a.customProperties, 'tps_p99') - getDoubleValue(b.customProperties, 'tps_p99'),
    width: 20,
    modifier: 'wrap',
  },
  {
    field: 'mean_input_tokens',
    label: `Mean${NBSP}Input Tokens`,
    sortable: (
      a: CatalogPerformanceMetricsArtifact,
      b: CatalogPerformanceMetricsArtifact,
    ): number =>
      getDoubleValue(a.customProperties, 'mean_input_tokens') -
      getDoubleValue(b.customProperties, 'mean_input_tokens'),
    width: 20,
    modifier: 'wrap',
  },
  {
    field: 'mean_output_tokens',
    label: `Mean${NBSP}Output Tokens`,
    sortable: (
      a: CatalogPerformanceMetricsArtifact,
      b: CatalogPerformanceMetricsArtifact,
    ): number =>
      getDoubleValue(a.customProperties, 'mean_output_tokens') -
      getDoubleValue(b.customProperties, 'mean_output_tokens'),
    width: 20,
    modifier: 'wrap',
  },
  {
    field: 'framework_version',
    label: 'vLLM Version',
    info: {
      popover: 'The vLLM inference engine version used to generate these performance results.',
      popoverProps: {
        position: 'left',
      },
    },
    sortable: (
      a: CatalogPerformanceMetricsArtifact,
      b: CatalogPerformanceMetricsArtifact,
    ): number => {
      const versionA = getStringValue(a.customProperties, 'framework_version');
      const versionB = getStringValue(b.customProperties, 'framework_version');
      return versionA.localeCompare(versionB);
    },
    width: 20,
    modifier: 'wrap',
  },
  {
    field: 'cold_start_load_time',
    label: `Cold start\nlatency`,
    info: {
      popover:
        'The estimated time required to provision hardware resources and initialize the container before the model can accept traffic.',
      popoverProps: {
        position: 'left',
      },
    },
    sortable: false,
    width: 20,
    modifier: 'wrap',
  },
  {
    field: 'runtime_command',
    label: 'Runtime Command',
    info: {
      popover: 'Runtime configuration used to validate this model.',
      popoverProps: {
        position: 'left',
      },
    },
    sortable: false,
    width: 20,
    modifier: 'wrap',
  },
];

/**
 * Fields that should always be visible (sticky columns).
 * These will not appear in the ManageColumnsModal.
 */
export const STICKY_COLUMN_FIELDS: HardwareConfigColumnField[] = [
  PerformancePropertyKey.HARDWARE_CONFIGURATION,
  PerformancePropertyKey.USE_CASE,
];

/**
 * Default visible column fields (excluding sticky - they're always visible).
 * Core metrics: Replicas, Total RPS, RPS/replica, P90 latencies, tokens, vLLM version.
 * NOTE: The latency and TPS will be overridden on mount to match the active latency filter (if any).
 *       See useHardwareConfigColumns.ts for that logic.
 */
export const DEFAULT_VISIBLE_COLUMN_FIELDS: HardwareConfigColumnField[] = [
  'replicas',
  'total_requests_per_second',
  'requests_per_second',
  'ttft_p90',
  'tps_p90',
  'mean_input_tokens',
  'mean_output_tokens',
  'framework_version',
];

/**
 * All latency column fields (TTFT, E2E, ITL - excluding TPS which measures throughput).
 */
export const LATENCY_COLUMN_FIELDS: HardwareConfigColumnField[] = Object.values(LatencyMetric)
  .filter((metric) => metric !== LatencyMetric.TPS)
  .flatMap((metric) =>
    Object.values(LatencyPercentile).map((percentile) => getLatencyPropertyKey(metric, percentile)),
  );

/**
 * All TPS column fields.
 */
export const TPS_COLUMN_FIELDS: HardwareConfigColumnField[] = Object.values(LatencyPercentile).map(
  (percentile) => getLatencyPropertyKey(LatencyMetric.TPS, percentile),
);

/**
 * Storage key for localStorage persistence of column visibility.
 */
export const HARDWARE_CONFIG_COLUMNS_STORAGE_KEY = 'hardware-config-table-columns';
