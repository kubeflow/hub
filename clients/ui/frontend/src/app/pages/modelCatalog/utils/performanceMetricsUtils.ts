import { asEnumMember } from 'mod-arch-core';
import { CatalogPerformanceMetricsArtifact } from '~/app/modelCatalogTypes';
import { getStringValue, getIntValue } from '~/app/utils';
import {
  UseCaseOptionValue,
  PerformancePropertyKey,
  EMPTY_CUSTOM_PROPERTY_VALUE,
  COLD_START_SUB_TYPE,
} from '~/concepts/modelCatalog/const';
import { getUseCaseOption } from './workloadTypeUtils';

export type SliderRange = {
  minValue: number;
  maxValue: number;
  isSliderDisabled: boolean;
};

export const MAX_RPS_MAX_VALUE = 50;

export const MAX_RPS_RANGE: SliderRange = {
  minValue: 1,
  maxValue: MAX_RPS_MAX_VALUE,
  isSliderDisabled: false,
};

export const FALLBACK_RPS_RANGE: SliderRange = {
  minValue: 1,
  maxValue: 300,
  isSliderDisabled: false,
};

export const FALLBACK_LATENCY_RANGE: SliderRange = {
  minValue: 20,
  maxValue: 893,
  isSliderDisabled: false,
};

export const COLD_START_LOAD_TIME_RANGE: SliderRange = {
  minValue: 15,
  maxValue: 1000,
  isSliderDisabled: false,
};

type CalculateSliderRangeOptions = {
  performanceArtifacts: CatalogPerformanceMetricsArtifact[];
  getArtifactFilterValue: (artifact: CatalogPerformanceMetricsArtifact) => number;
  fallbackRange: SliderRange;
  shouldRound?: boolean;
};

export const formatLatency = (value: number): string => `${value.toFixed(2)} ms`;

export const formatTps = (value: number): string => `${value.toFixed(2)} tok/s`;

export const formatTokenValue = (value: number): string => value.toFixed(0);

export const getWorkloadType = (artifact: CatalogPerformanceMetricsArtifact): string => {
  const useCaseValue = getStringValue(artifact.customProperties, PerformancePropertyKey.USE_CASE);
  if (!useCaseValue) {
    return EMPTY_CUSTOM_PROPERTY_VALUE;
  }
  const useCaseEnum = asEnumMember(useCaseValue, UseCaseOptionValue);
  if (!useCaseEnum) {
    return EMPTY_CUSTOM_PROPERTY_VALUE;
  }
  return getUseCaseOption(useCaseEnum)?.label || EMPTY_CUSTOM_PROPERTY_VALUE;
};

export const isColdStartArtifact = (artifact: CatalogPerformanceMetricsArtifact): boolean =>
  getStringValue(artifact.customProperties, PerformancePropertyKey.PERFORMANCE_SUB_TYPE) ===
  COLD_START_SUB_TYPE;

export const separatePerformanceArtifacts = (
  artifacts: CatalogPerformanceMetricsArtifact[],
): {
  throughputArtifacts: CatalogPerformanceMetricsArtifact[];
  coldStartArtifacts: CatalogPerformanceMetricsArtifact[];
} => {
  const throughputArtifacts: CatalogPerformanceMetricsArtifact[] = [];
  const coldStartArtifacts: CatalogPerformanceMetricsArtifact[] = [];
  artifacts.forEach((artifact) => {
    if (isColdStartArtifact(artifact)) {
      coldStartArtifacts.push(artifact);
    } else {
      throughputArtifacts.push(artifact);
    }
  });
  return { throughputArtifacts, coldStartArtifacts };
};

/**
 * Finds a matching cold-start artifact for a throughput artifact by GPU type.
 * Matches on gpu_type from cold-start artifact against hardware_type or hardware_configuration
 * from the throughput artifact.
 */
export const findMatchingColdStartArtifact = (
  throughputArtifact: CatalogPerformanceMetricsArtifact,
  coldStartArtifacts: CatalogPerformanceMetricsArtifact[],
): CatalogPerformanceMetricsArtifact | undefined => {
  const hwConfig = getStringValue(
    throughputArtifact.customProperties,
    PerformancePropertyKey.HARDWARE_CONFIGURATION,
  );
  const hwType = getStringValue(
    throughputArtifact.customProperties,
    PerformancePropertyKey.HARDWARE_TYPE,
  );
  const hwCount = getIntValue(
    throughputArtifact.customProperties,
    PerformancePropertyKey.HARDWARE_COUNT,
  );

  return coldStartArtifacts.find((cs) => {
    const csGpuType = getStringValue(cs.customProperties, PerformancePropertyKey.GPU_TYPE);
    const csGpuCount = getIntValue(cs.customProperties, PerformancePropertyKey.GPU_COUNT);

    const typeMatches = hwConfig.includes(csGpuType) || csGpuType === hwType;
    const countMatches = csGpuCount === 0 || hwCount === 0 || csGpuCount === hwCount;
    return typeMatches && countMatches;
  });
};

export const getSliderRange = ({
  performanceArtifacts,
  getArtifactFilterValue,
  fallbackRange,
  shouldRound = false,
}: CalculateSliderRangeOptions): SliderRange => {
  if (performanceArtifacts.length === 0) {
    return fallbackRange;
  }

  const values = performanceArtifacts.map(getArtifactFilterValue).filter((value) => value > 0);

  if (values.length === 0) {
    return fallbackRange;
  }

  const minValue = Math.min(...values);
  const maxValue = Math.max(...values);

  const calculatedMin = shouldRound ? Math.round(minValue) : minValue;
  const calculatedMax = shouldRound ? Math.round(maxValue) : maxValue;
  const hasIdenticalValues = calculatedMin === calculatedMax;

  return {
    minValue: calculatedMin,
    maxValue: hasIdenticalValues ? calculatedMin + 1 : calculatedMax,
    isSliderDisabled: hasIdenticalValues,
  };
};
