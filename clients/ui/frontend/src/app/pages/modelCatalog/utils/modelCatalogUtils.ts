import React from 'react';
import { capitalize } from '@patternfly/react-core';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import {
  CatalogArtifacts,
  CatalogArtifactType,
  CatalogFilterOptions,
  CatalogFilterOptionsList,
  CatalogLabel,
  CatalogLabelList,
  CatalogModel,
  CatalogModelArtifact,
  CatalogModelDetailsParams,
  CatalogSource,
  CatalogSourceList,
  ModelCatalogFilterStates,
  ModelCatalogStringFilterValueType,
  MetricsType,
  ModelCatalogFilterKey,
  SourceLabel,
} from '~/app/modelCatalogTypes';
import { getLabels } from '~/app/pages/modelRegistry/screens/utils';
import {
  ModelCatalogStringFilterKey,
  ModelCatalogNumberFilterKey,
  ALL_LATENCY_FILTER_KEYS,
  LatencyMetricFieldName,
  DEFAULT_PERFORMANCE_FILTERS_QUERY_NAME,
  isPerformanceStringFilterKey,
  PERFORMANCE_FILTER_KEYS,
  ModelCatalogSortOption,
  SortOrder,
  SortField,
  CatalogModelCustomPropertyKey,
  ModelType,
  ModelCatalogTask,
  MATCH_ALL_FILTER_KEYS,
} from '~/concepts/modelCatalog/const';
import { isSourceStatusWithModels } from '~/concepts/modelCatalogSettings/const';
import { ModelRegistryCustomProperties, ModelRegistryMetadataType } from '~/app/types';
import {
  buildCustomPropertiesWithModelType,
  getModelTypeStoredValueFromCustomProperties,
} from '~/app/pages/modelRegistry/screens/RegisterModel/registerModelTypeUtils';

/**
 * Prefix used by the backend for artifact-specific filter options.
 * Filter options with this prefix are applicable to the artifacts endpoint.
 */
const ARTIFACTS_FILTER_PREFIX = 'artifacts.';

export const extractVersionTag = (tags?: string[]): string | undefined =>
  tags?.find((tag) => /^\d+\.\d+\.\d+$/.test(tag));
export const filterNonVersionTags = (tags?: string[]): string[] | undefined => {
  const versionTag = extractVersionTag(tags);
  return tags?.filter((tag) => tag !== versionTag);
};

export const getModelName = (modelName: string): string => {
  const index = modelName.indexOf('/');
  if (index === -1) {
    return modelName;
  }
  return modelName.slice(index + 1);
};

export const decodeParams = (
  params: Readonly<CatalogModelDetailsParams>,
): CatalogModelDetailsParams =>
  Object.fromEntries(
    Object.entries(params).map(([key, value]) => [key, decodeURIComponent(value)]),
  );

export const encodeParams = (params: CatalogModelDetailsParams): CatalogModelDetailsParams =>
  Object.fromEntries(
    Object.entries(params).map(([key, value]) => [
      key,
      encodeURIComponent(value).replace(/\./g, '%252E'),
    ]),
  );

export const filterEnabledCatalogSources = (
  catalogSources: CatalogSourceList | null,
): CatalogSourceList | null => {
  if (!catalogSources) {
    return null;
  }

  // Filter sources that are enabled AND have available models (including partially-available)
  const filteredItems = catalogSources.items?.filter(
    (source) => source.enabled !== false && isSourceStatusWithModels(source.status),
  );

  return {
    ...catalogSources,
    items: filteredItems || [],
    size: filteredItems?.length || 0,
  };
};

export const getModelArtifactUri = (artifacts: CatalogArtifacts[]): string => {
  const modelArtifact = findModelArtifact(artifacts);
  return modelArtifact?.uri || '';
};

export const hasModelArtifacts = (artifacts: CatalogArtifacts[]): boolean =>
  artifacts.some((artifact) => artifact.artifactType === CatalogArtifactType.modelArtifact);

/**
 * Finds the model artifact from an array of catalog artifacts.
 * @param artifacts Array of catalog artifacts to search
 * @returns The model artifact if found, undefined otherwise
 */
const findModelArtifact = (artifacts: CatalogArtifacts[]): CatalogModelArtifact | undefined =>
  artifacts.find(
    (artifact): artifact is CatalogModelArtifact =>
      artifact.artifactType === CatalogArtifactType.modelArtifact,
  );

/**
 * Extracts architecture values from the model artifact's custom properties.
 * The architecture custom property should be a JSON-encoded array of architecture strings.
 * Architectures are normalized to lowercase and deduplicated.
 *
 * @param artifacts Array of catalog artifacts to search
 * @returns Array of architecture strings, or empty array if none found or invalid
 */
export const getArchitecturesFromArtifacts = (artifacts: CatalogArtifacts[]): string[] => {
  const modelArtifact = findModelArtifact(artifacts);

  if (!modelArtifact) {
    return [];
  }

  const architectureProp =
    modelArtifact.customProperties?.[CatalogModelCustomPropertyKey.ARCHITECTURE];

  if (!architectureProp || architectureProp.metadataType !== ModelRegistryMetadataType.STRING) {
    return [];
  }

  const architectureString = architectureProp.string_value;

  try {
    if (!architectureString) {
      return [];
    }
    const parsed = JSON.parse(architectureString);

    // Handle both non-array and array cases in one flow
    const items = Array.isArray(parsed) ? parsed : [];

    // Filter strings, normalize to lowercase, and deduplicate
    return [
      ...new Set(
        items
          .filter((item): item is string => typeof item === 'string')
          .map((item) => item.toLowerCase()),
      ),
    ];
  } catch (error) {
    // Invalid JSON - return empty array
    if (process.env.NODE_ENV === 'development') {
      // eslint-disable-next-line no-console
      console.warn('Failed to parse architecture JSON:', architectureString, error);
    }
    return [];
  }
};

export const hasPerformanceArtifacts = (artifacts: CatalogArtifacts[]): boolean =>
  artifacts.some(
    (artifact) =>
      artifact.artifactType === CatalogArtifactType.metricsArtifact &&
      'metricsType' in artifact &&
      artifact.metricsType === MetricsType.performanceMetrics,
  );

// Utility function to check if a model is validated
export const isModelValidated = (model: CatalogModel): boolean => {
  if (!model.customProperties) {
    return false;
  }
  const labels = getLabels(model.customProperties);
  return labels.includes('validated');
};

export const shouldShowValidatedInsights = (
  model: CatalogModel,
  artifacts: CatalogArtifacts[],
): boolean => isModelValidated(model) && hasPerformanceArtifacts(artifacts);

export const hasValidatedToolCalling = (model: CatalogModel): boolean =>
  model.validatedTasks?.includes(ModelCatalogTask.TOOL_CALLING) === true &&
  model.servingConfig?.toolCalling != null;

export const useCatalogStringFilterState = <K extends ModelCatalogStringFilterKey>(
  filterKey: K,
): {
  isSelected: (value: ModelCatalogStringFilterValueType[K]) => boolean;
  setSelected: (value: string, selected: boolean) => void;
} => {
  type Value = ModelCatalogStringFilterValueType[K];
  const { filterData, setFilterData } = React.useContext(ModelCatalogContext);
  const selections: string[] = filterData[filterKey];
  const isValidStringState = (state: string[]): state is ModelCatalogFilterStates[K] =>
    Object.values(ModelCatalogStringFilterKey).includes(filterKey);
  const isSelected = React.useCallback((value: Value) => selections.includes(value), [selections]);
  const setSelected = (value: string, selected: boolean) => {
    const nextState = selected
      ? [...selections, value]
      : selections.filter((item) => item !== value);
    if (isValidStringState(nextState)) {
      setFilterData(filterKey, nextState);
    }
  };

  return { isSelected, setSelected };
};

export const useCatalogNumberFilterState = (
  filterKey: ModelCatalogNumberFilterKey,
): {
  value: number | undefined;
  setValue: (value: number | undefined) => void;
} => {
  const { filterData, setFilterData } = React.useContext(ModelCatalogContext);
  const value = filterData[filterKey];
  const setValue = React.useCallback(
    (newValue: number | undefined) => {
      setFilterData(filterKey, newValue);
    },
    [filterKey, setFilterData],
  );
  return { value, setValue };
};

const isArrayOfSelections = (
  filterOption: CatalogFilterOptions[keyof CatalogFilterOptions],
  data: unknown,
): data is string[] =>
  filterOption?.type === 'string' && Array.isArray(filterOption.values) && Array.isArray(data);

/**
 * Filter IDs that use numeric comparison (latency filters + Max RPS).
 */
const KNOWN_NUMERIC_FILTER_IDS: string[] = [
  ...ALL_LATENCY_FILTER_KEYS,
  ModelCatalogNumberFilterKey.MAX_RPS,
];

/**
 * Type guard to check if a filter is a known numeric filter with a number value.
 */
const isKnownNumericFilter = (
  filterOption: CatalogFilterOptions[keyof CatalogFilterOptions],
  filterId: string,
  data: unknown,
): data is number =>
  filterOption?.type === 'number' &&
  KNOWN_NUMERIC_FILTER_IDS.includes(filterId) &&
  typeof data === 'number';

/**
 * Gets the comparison operator for a numeric filter from namedQueries.
 * Looks up the operator in the default performance filters namedQuery,
 * falls back to '<' if not found.
 */
const getNumericFilterOperator = (options: CatalogFilterOptionsList, filterId: string): string => {
  const defaultQuery = options.namedQueries?.[DEFAULT_PERFORMANCE_FILTERS_QUERY_NAME];
  if (defaultQuery && filterId in defaultQuery) {
    const fieldFilter = defaultQuery[filterId];
    // Return the operator from the namedQuery (e.g., '<=', '<', '>')
    return fieldFilter.operator;
  }
  // Fall back to '<' if this filter isn't in the namedQuery
  return '<';
};

const isFilterIdInMap = (
  filterId: unknown,
  filters: CatalogFilterOptions,
): filterId is keyof CatalogFilterOptions => typeof filterId === 'string' && filterId in filters;

/**
 * Gets the active latency field name from the filter state (if any)
 */
export const getActiveLatencyFieldName = (
  filterData: ModelCatalogFilterStates,
): LatencyMetricFieldName | undefined => {
  for (const fieldName of ALL_LATENCY_FILTER_KEYS) {
    const value = filterData[fieldName];
    if (value !== undefined && typeof value === 'number') {
      return fieldName;
    }
  }
  return undefined;
};

export const getEffectiveSortBy = (
  sortBy: ModelCatalogSortOption | null,
  performanceViewEnabled: boolean,
): ModelCatalogSortOption => {
  if (sortBy) {
    return sortBy;
  }
  return performanceViewEnabled
    ? ModelCatalogSortOption.LOWEST_LATENCY
    : ModelCatalogSortOption.RECENT_PUBLISH;
};

/**
 * Gets the sort parameters for API requests based on sort option and filter state.
 * @param sortBy - The selected sort option (or null for default)
 * @param performanceViewEnabled - Whether performance view is enabled
 * @param activeLatencyField - The active latency field name (if any)
 * @returns Object with orderBy and sortOrder for API requests
 */
export const getSortParams = (
  sortBy: ModelCatalogSortOption | null,
  performanceViewEnabled: boolean,
  activeLatencyField: LatencyMetricFieldName | undefined,
): { orderBy: string; sortOrder: string } => {
  const effectiveSortBy = getEffectiveSortBy(sortBy, performanceViewEnabled);
  const recentPublishSort = {
    orderBy: SortField.LAST_UPDATE_TIME,
    sortOrder: SortOrder.DESC,
  } as const;

  if (effectiveSortBy === ModelCatalogSortOption.RECENT_PUBLISH) {
    return recentPublishSort;
  }

  // effectiveSortBy must be LOWEST_LATENCY at this point
  if (!activeLatencyField) {
    // Fallback to recent publish if no latency field is available
    return recentPublishSort;
  }

  // activeLatencyField is already in the correct format: artifacts.{metric}_{percentile}.double_value
  // (e.g., artifacts.ttft_p90.double_value, artifacts.e2e_mean.double_value, artifacts.itl_p95.double_value)
  // This matches the filter key format used in filterQuery, so we can use it directly
  return {
    orderBy: activeLatencyField,
    sortOrder: SortOrder.ASC, // Lowest first (ascending)
  };
};

const wrapInQuotes = (v: string): string => `'${v.replace(/'/g, "''")}'`;

const eqFilter = (k: string, v: string) => `${k}=${wrapInQuotes(v)}`;
const inFilter = (k: string, values: string[]) =>
  `${k} IN (${values.map((v) => wrapInQuotes(v)).join(',')})`;
const andFilter = (k: string, values: string[]) => values.map((v) => eqFilter(k, v)).join(' AND ');

const isMatchAllFilter = (filterId: string): boolean =>
  MATCH_ALL_FILTER_KEYS.some((key) => key === filterId);

/**
 * Check if a filter key has the artifacts.* prefix.
 * Filter keys with this prefix are artifact-specific filters.
 */
const hasArtifactsPrefix = (filterId: string): boolean =>
  filterId.startsWith(ARTIFACTS_FILTER_PREFIX);

/**
 * Strips the 'artifacts.' prefix from a filter key if present.
 * Used when constructing filterQuery for artifacts endpoint.
 * Example: 'artifacts.use_case.string_value' -> 'use_case.string_value'
 */
export const stripArtifactsPrefix = (filterId: string): string => {
  if (hasArtifactsPrefix(filterId)) {
    return filterId.substring(ARTIFACTS_FILTER_PREFIX.length);
  }
  return filterId;
};

/**
 * Target endpoint type for filter query construction.
 * - 'models': Include all filters (except RPS), use filter keys directly
 * - 'artifacts': Only include artifact-prefixed filters, strip the prefix in output
 */
export type FilterQueryTarget = 'models' | 'artifacts';

/**
 * Determines if a filter should be included based on the target endpoint.
 * - For models: Include all filters except RPS (which is passed as a separate param)
 * - For artifacts: Only include filters that have the artifacts.* prefix
 */
const shouldIncludeFilter = (filterId: string, target: FilterQueryTarget): boolean => {
  // RPS is always passed as a separate param, not in filterQuery
  if (filterId === ModelCatalogNumberFilterKey.MAX_RPS) {
    return false;
  }

  if (target === 'models') {
    // For models, include all filters (except RPS which is already excluded)
    return true;
  }

  // For artifacts, only include filters with the artifacts.* prefix
  return hasArtifactsPrefix(filterId);
};

/**
 * Gets the query key to use in the filterQuery string.
 * - For models: Use the filter ID directly (it already includes artifacts.* prefix if needed)
 * - For artifacts: Strip the artifacts.* prefix (the endpoint doesn't need it)
 */
const getQueryKey = (filterId: string, target: FilterQueryTarget): string => {
  if (target === 'artifacts') {
    return stripArtifactsPrefix(filterId);
  }
  return filterId;
};

/**
 * Serializes a single filter entry to a filter query clause.
 * Handles string arrays (IN/equality) and numeric filters (comparison operators).
 */
const serializeFilterEntry = (
  filterId: string,
  data: ModelCatalogFilterStates[keyof ModelCatalogFilterStates],
  options: CatalogFilterOptionsList,
  target: FilterQueryTarget,
): string => {
  if (typeof data === 'undefined') {
    return '';
  }

  // Get the filter option from the options map
  const filterOption =
    options.filters && isFilterIdInMap(filterId, options.filters)
      ? options.filters[filterId]
      : undefined;

  if (!filterOption) {
    return '';
  }

  const queryKey = getQueryKey(filterId, target);

  // Handle string array filters (multi-select)
  if (isArrayOfSelections(filterOption, data)) {
    switch (data.length) {
      case 0:
        return '';
      case 1:
        return eqFilter(queryKey, data[0]);
      default:
        return isMatchAllFilter(filterId) ? andFilter(queryKey, data) : inFilter(queryKey, data);
    }
  }

  // Handle numeric filters
  if (isKnownNumericFilter(filterOption, filterId, data)) {
    const operator = getNumericFilterOperator(options, filterId);
    return `${queryKey} ${operator} ${data}`;
  }

  return '';
};

/**
 * Converts filter data into a filter query string.
 *
 * @param filterData - The current filter state
 * @param options - Filter options from the server (includes namedQueries for operators)
 * @param target - The target endpoint:
 *   - 'models': Include all filters (except RPS), use filter keys directly
 *   - 'artifacts': Only include artifact-prefixed filters, strip the prefix in output
 *
 * Note: RPS is NOT included in filterQuery for either target - it's passed as targetRPS param.
 */
export const filtersToFilterQuery = (
  filterData: ModelCatalogFilterStates,
  options: CatalogFilterOptionsList,
  target: FilterQueryTarget = 'models',
): string => {
  const serializedFilters: string[] = Object.entries(filterData)
    .filter(([filterId]) => shouldIncludeFilter(filterId, target))
    .map(([filterId, data]) => serializeFilterEntry(filterId, data, options, target));

  const nonEmptyFilters = serializedFilters.filter((v) => !!v);
  return nonEmptyFilters.length === 0 ? '' : nonEmptyFilters.join(' AND ');
};

/**
 * Returns a copy of filterData with only basic (non-performance) filters.
 * Used when performance view is disabled to exclude performance filters from API queries.
 * Performance filters are cleared to their empty state ([] for arrays, undefined for numbers).
 */
export const getBasicFiltersOnly = (
  filterData: ModelCatalogFilterStates,
): ModelCatalogFilterStates => {
  // Start with a copy of filterData
  const result: ModelCatalogFilterStates = { ...filterData };

  // Clear all performance filters using the centralized list
  PERFORMANCE_FILTER_KEYS.forEach((perfKey) => {
    if (isPerformanceStringFilterKey(perfKey)) {
      // String filters clear to empty array
      result[perfKey] = [];
    } else {
      // Number filters (MAX_RPS and latency) clear to undefined
      result[perfKey] = undefined;
    }
  });
  result[ModelCatalogStringFilterKey.HARDWARE_CONFIGURATION] = [];

  return result;
};

export const getUniqueSourceLabels = (catalogSources: CatalogSourceList | null): string[] => {
  if (!catalogSources || !catalogSources.items) {
    return [];
  }

  const allLabels = new Set<string>();

  catalogSources.items.forEach((source) => {
    // Only include labels from sources that are enabled AND have models (available or partially-available)
    if (source.enabled && isSourceStatusWithModels(source.status) && source.labels.length > 0) {
      source.labels.forEach((label) => {
        if (label.trim()) {
          allLabels.add(label.trim());
        }
      });
    }
  });

  return Array.from(allLabels);
};

export const hasSourcesWithoutLabels = (catalogSources: CatalogSourceList | null): boolean => {
  if (!catalogSources || !catalogSources.items) {
    return false;
  }

  return catalogSources.items.some((source) => {
    // Only consider sources that are enabled AND have models (available or partially-available)
    if (source.enabled !== false && isSourceStatusWithModels(source.status)) {
      // Check if source has no labels or only empty/whitespace labels
      return source.labels.length === 0 || source.labels.every((label) => !label.trim());
    }
    return false;
  });
};

export const getSourceFromSourceId = (
  sourceId: string,
  catalogSources: CatalogSourceList | null,
): CatalogSource | undefined => {
  if (!catalogSources || !sourceId || !catalogSources.items) {
    return undefined;
  }

  return catalogSources.items.find((source) => source.id === sourceId);
};

/**
 * Checks if any filters are applied. If filterKeys is provided, only checks those specific filters.
 * Otherwise checks all filters.
 */
export const hasFiltersApplied = (
  filterData: ModelCatalogFilterStates,
  filterKeys?: ModelCatalogFilterKey[],
): boolean =>
  Object.entries(filterData).some(([key, value]) => {
    if (filterKeys && !filterKeys.some((k) => k === key)) {
      return false;
    }
    if (Array.isArray(value)) {
      return value.length > 0;
    }
    return value !== undefined;
  });

/**
 * Checks if a filter value differs from its default value.
 * Used to determine if a filter chip should be visible.
 */
export const isValueDifferentFromDefault = (
  currentValue: string | number | string[] | undefined,
  defaultValue: string | number | string[] | undefined,
): boolean => {
  if (defaultValue === undefined) {
    // No default defined, show chip if value exists
    if (Array.isArray(currentValue)) {
      return currentValue.length > 0;
    }
    return currentValue !== undefined;
  }

  if (currentValue === undefined) {
    return false;
  }

  // Compare arrays
  if (Array.isArray(currentValue) && Array.isArray(defaultValue)) {
    if (currentValue.length !== defaultValue.length) {
      return true;
    }
    return !currentValue.every((v) => defaultValue.includes(String(v)));
  }

  // Compare single value with array
  if (Array.isArray(currentValue) && !Array.isArray(defaultValue)) {
    if (currentValue.length !== 1) {
      return true;
    }
    return currentValue[0] !== defaultValue;
  }

  // Compare single values
  return currentValue !== defaultValue;
};

/**
 * Filters catalog sources to only include those with discoverable models.
 * A source has models if its status is AVAILABLE or PARTIALLY_AVAILABLE.
 * This is used to filter out disabled sources or sources with errors from the switcher.
 */
export const filterSourcesWithModels = (
  catalogSources: CatalogSourceList | null,
): CatalogSourceList | null => {
  if (!catalogSources) {
    return null;
  }

  const filteredItems = catalogSources.items?.filter((source) =>
    isSourceStatusWithModels(source.status),
  );

  return {
    ...catalogSources,
    items: filteredItems || [],
    size: filteredItems?.length || 0,
  };
};

/**
 * Checks if there are any catalog sources that have models available.
 * Returns true if at least one source has status AVAILABLE or PARTIALLY_AVAILABLE.
 */
export const hasSourcesWithModels = (catalogSources: CatalogSourceList | null): boolean => {
  if (!catalogSources?.items) {
    return false;
  }

  return catalogSources.items.some((source) => isSourceStatusWithModels(source.status));
};

export const generateCategoryName = (name: string): string =>
  name.toLowerCase().endsWith('models') ? name : `${name} models`;

/**
 * Finds a label from the catalog labels list that matches the given source label name.
 * Handles the special case where sourceLabel is 'null' (other/unlabeled sources).
 * @param sourceLabel The label string from a source (or SourceLabel.other for unlabeled sources)
 * @param catalogLabels The list of catalog labels from the API
 * @returns The matching CatalogLabel or undefined if not found
 */
export const findLabelData = (
  sourceLabel: string | undefined,
  catalogLabels: CatalogLabelList | null,
): CatalogLabel | undefined => {
  if (!catalogLabels?.items || !sourceLabel) {
    return undefined;
  }

  // Special case: sourceLabel is 'null' (SourceLabel.other) - look for label with name: null
  if (sourceLabel === SourceLabel.other) {
    return catalogLabels.items.find((label) => label.name === null);
  }

  // Normal case: find label with matching name
  return catalogLabels.items.find((label) => label.name === sourceLabel);
};

/**
 * Gets the display name for a source label, using the catalog labels data if available.
 * Falls back to the raw label name with the given suffix appended if no display name is found.
 * @param sourceLabel The label string from a source (or SourceLabel.other for unlabeled sources)
 * @param catalogLabels The list of catalog labels from the API
 * @param otherFallback Display name for sources without labels (default: 'Other models')
 * @param categorySuffix Suffix appended to the label name when no display name exists (default: 'models')
 * @returns The display name to show in the UI
 */
export const getLabelDisplayName = (
  sourceLabel: string | undefined,
  catalogLabels: CatalogLabelList | null,
  otherFallback = 'Other models',
  categorySuffix = 'models',
): string => {
  if (!sourceLabel) {
    return '';
  }

  const labelData = findLabelData(sourceLabel, catalogLabels);

  if (labelData?.displayName) {
    return labelData.displayName;
  }

  if (sourceLabel === SourceLabel.other) {
    return otherFallback;
  }

  return sourceLabel.toLowerCase().endsWith(categorySuffix)
    ? sourceLabel
    : `${sourceLabel} ${categorySuffix}`;
};

/**
 * Gets the description for a source label from the catalog labels data.
 * @param sourceLabel The label string from a source (or SourceLabel.other for unlabeled sources)
 * @param catalogLabels The list of catalog labels from the API
 * @returns The description text or undefined if not found
 */
export const getLabelDescription = (
  sourceLabel: string | undefined,
  catalogLabels: CatalogLabelList | null,
): string | undefined => {
  const labelData = findLabelData(sourceLabel, catalogLabels);
  return labelData?.description;
};

/**
 * Orders source labels according to the order in the catalog labels list.
 * Labels that appear in catalogLabels are ordered first (in the order they appear in the API),
 * followed by any labels found on sources that don't appear in catalogLabels.
 * @param sourceLabels Array of unique source labels from the sources
 * @param catalogLabels The list of catalog labels from the API
 * @returns Ordered array of source labels
 */
export const orderLabelsByPriority = (
  sourceLabels: string[],
  catalogLabels: CatalogLabelList | null,
): string[] => {
  if (!catalogLabels?.items) {
    return sourceLabels;
  }

  const orderedLabels: string[] = [];
  const remainingLabels = new Set(sourceLabels);

  // First, add labels in the order they appear in catalogLabels
  catalogLabels.items.forEach((catalogLabel) => {
    // Skip the null entry (it's handled separately as "other models")
    if (catalogLabel.name === null) {
      return;
    }

    if (remainingLabels.has(catalogLabel.name)) {
      orderedLabels.push(catalogLabel.name);
      remainingLabels.delete(catalogLabel.name);
    }
  });

  // Then add any remaining labels that weren't in catalogLabels
  orderedLabels.push(...Array.from(remainingLabels));

  return orderedLabels;
};

export const getActiveSourceLabels = (
  catalogSources: CatalogSourceList | null,
  catalogLabels: CatalogLabelList | null,
): string[] => {
  const enabledSources = filterEnabledCatalogSources(catalogSources);
  const uniqueLabels = getUniqueSourceLabels(enabledSources);
  const orderedLabels = orderLabelsByPriority(uniqueLabels, catalogLabels);

  if (hasSourcesWithoutLabels(enabledSources)) {
    return [...orderedLabels, SourceLabel.other];
  }

  return orderedLabels;
};

/**
 * Formats model type value for display in the UI.
 * Converts raw API values (generative, predictive, unknown) to user-friendly display labels.
 *
 * @param modelTypeRaw The raw model type value from customProperties, or null if not set
 * @returns Formatted display string for the model type
 */
export const formatModelTypeDisplay = (modelTypeRaw: string | null): string => {
  if (!modelTypeRaw || modelTypeRaw.trim() === '') {
    return 'Unknown';
  }
  const normalized = modelTypeRaw.toLowerCase().trim();
  if (normalized === ModelType.GENERATIVE) {
    return 'Generative AI model (Example, LLM)';
  }
  if (normalized === ModelType.PREDICTIVE) {
    return 'Predictive Model';
  }
  if (normalized === ModelType.UNKNOWN) {
    return 'Unknown';
  }
  // Fallback: capitalize whatever value we got
  return capitalize(modelTypeRaw.trim());
};

/**
 * Returns model registry customProperties entries to prefill `model_type` when registering
 * from the catalog. Recognized values (generative, predictive, unknown) are copied;
 * unrecognized or missing values yield {}.
 */
export const getCatalogModelTypePropertyForRegistration = (
  customProperties?: ModelRegistryCustomProperties,
): ModelRegistryCustomProperties => {
  const stored = getModelTypeStoredValueFromCustomProperties(customProperties);
  return buildCustomPropertiesWithModelType(undefined, stored);
};
