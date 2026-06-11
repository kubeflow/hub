/**
 * Shared constants for Model Catalog Cypress tests.
 * These test IDs and constants are used across multiple test files.
 */

/**
 * Test IDs for performance filter components
 */
export const PERFORMANCE_FILTER_TEST_IDS = {
  workloadType: 'workload-type-filter',
  latency: 'latency-filter',
  latencyContent: 'latency-filter-content',
  latencyMetricSelect: 'latency-metric-select',
  latencyPercentileSelect: 'latency-percentile-select',
  latencyApply: 'latency-apply-filter',
  latencyReset: 'latency-reset-filter',
  maxRps: 'max-rps-filter',
  maxRpsApply: 'max-rps-apply-filter',
  coldStartLoadTime: 'cold-start-load-time-filter',
  coldStartLoadTimeApply: 'cold-start-load-time-apply-filter',
  coldStartLoadTimeReset: 'cold-start-load-time-reset-filter',
  hardwareTable: 'hardware-configuration-table',
  clearAllFilters: 'clear-all-filters-button',
} as const;

/**
 * Test IDs for model catalog card components
 */
export const MODEL_CARD_TEST_IDS = {
  card: 'model-catalog-card',
  detailLink: 'model-catalog-detail-link',
  description: 'model-catalog-card-description',
  benchmarkLink: 'validated-model-benchmark-link',
  benchmarkNext: 'validated-model-benchmark-next',
  benchmarkPrev: 'validated-model-benchmark-prev',
  hardware: 'validated-model-hardware',
  replicas: 'validated-model-replicas',
  latency: 'validated-model-latency',
} as const;

/**
 * Test IDs for model catalog details page components
 */
export const MODEL_DETAILS_TEST_IDS = {
  tabs: 'model-details-page-tabs',
  overviewTab: 'model-overview-tab',
  performanceInsightsTab: 'performance-insights-tab',
  overviewTabContent: 'model-overview-tab-content',
  performanceInsightsTabContent: 'performance-insights-tab-content',
  longDescription: 'model-long-description',
} as const;

/**
 * Test IDs for alerts
 */
export const ALERT_TEST_IDS = {
  performanceFiltersUpdated: 'performance-filters-updated-alert',
} as const;

/**
 * Non-breaking space character used in table column headers
 */
export const NBSP = '\u00A0';
