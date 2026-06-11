import { asEnumMember } from 'mod-arch-core';
import { CatalogPerformanceMetricsArtifact } from '~/app/modelCatalogTypes';
import { getStringValue } from '~/app/utils';
import {
  UseCaseOptionValue,
  PerformancePropertyKey,
  EMPTY_CUSTOM_PROPERTY_VALUE,
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
