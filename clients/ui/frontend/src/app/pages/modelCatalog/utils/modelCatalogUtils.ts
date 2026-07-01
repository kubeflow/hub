import { capitalize } from '@patternfly/react-core';
import { CatalogSource, CatalogSourceList } from '~/app/shared/types/catalogTypes';
import {
  CatalogArtifacts,
  CatalogArtifactType,
  CatalogFilterOptions,
  CatalogFilterOptionsList,
  CatalogModel,
  CatalogModelArtifact,
  CatalogModelDetailsParams,
  MetricsType,
  ModelCatalogFilterStates,
  ToolCallingConfig,
} from '~/app/modelCatalogTypes';
import { getLabels, getCustomPropString } from '~/app/pages/modelRegistry/screens/utils';
import { getDoubleValue } from '~/app/utils';
import {
  ModelCatalogStringFilterKey,
  ModelCatalogNumberFilterKey,
  ModelCatalogFilterKey,
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
import { ModelRegistryCustomProperties, ModelRegistryMetadataType } from '~/app/types';
import {
  buildCustomPropertiesWithModelType,
  getModelTypeStoredValueFromCustomProperties,
} from '~/app/pages/modelRegistry/screens/RegisterModel/registerModelTypeUtils';
import { eqFilter, inFilter, andFilter } from '~/app/shared/components/catalog';

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
  !!model.servingConfig?.toolCalling?.toolCallParser;

export const getToolCallingArgs = (config?: ToolCallingConfig): string => {
  const parts: string[] = [];
  if (config?.enableAutoToolChoice) {
    parts.push('--enable-auto-tool-choice');
  }
  if (config?.toolCallParser) {
    parts.push(`--tool-call-parser ${config.toolCallParser}`);
  }
  if (config?.chatTemplate) {
    parts.push(`--chat-template ${config.chatTemplate}`);
  }
  if (config?.requiredArgs) {
    parts.push(...config.requiredArgs);
  }
  return parts.join(' \\\n');
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
  ModelCatalogNumberFilterKey.COLD_START_LOAD_TIME,
  ModelCatalogNumberFilterKey.MIN_VRAM,
  ModelCatalogNumberFilterKey.IMAGE_SIZE,
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
  // Fall back to '<=' if this filter isn't in the namedQuery
  return '<=';
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

  if (effectiveSortBy === ModelCatalogSortOption.LOWEST_COLD_START) {
    return {
      orderBy: ModelCatalogNumberFilterKey.COLD_START_LOAD_TIME,
      sortOrder: SortOrder.ASC,
    };
  }

  // effectiveSortBy must be LOWEST_LATENCY at this point
  if (!activeLatencyField) {
    return recentPublishSort;
  }

  return {
    orderBy: activeLatencyField,
    sortOrder: SortOrder.ASC,
  };
};

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
 * - For models: Include all filters except RPS (which is passed as a separate param).
 *   Cold-start load time is excluded when includeColdStart is false.
 * - For artifacts: Only include filters that have the artifacts.* prefix.
 *   The includeColdStart parameter has no effect on the artifacts target.
 */
const shouldIncludeFilter = (
  filterId: string,
  target: FilterQueryTarget,
  includeColdStart: boolean,
): boolean => {
  if (filterId === ModelCatalogNumberFilterKey.MAX_RPS) {
    return false;
  }

  if (
    filterId === ModelCatalogNumberFilterKey.COLD_START_LOAD_TIME &&
    target === 'models' &&
    !includeColdStart
  ) {
    return false;
  }

  if (target === 'models') {
    return true;
  }

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
 * @param includeColdStart - Whether to include the cold-start filter in the AND clause.
 *   Should be true only when performance view is enabled.
 *
 * Note: RPS is NOT included in filterQuery for either target - it's passed as targetRPS param.
 */
export const filtersToFilterQuery = (
  filterData: ModelCatalogFilterStates,
  options: CatalogFilterOptionsList,
  target: FilterQueryTarget = 'models',
  includeColdStart = true,
): string =>
  Object.entries(filterData)
    .filter(([filterId]) => shouldIncludeFilter(filterId, target, includeColdStart))
    .map(([filterId, data]) => serializeFilterEntry(filterId, data, options, target))
    .filter((v) => !!v)
    .join(' AND ');

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

export const generateCategoryName = (name: string): string =>
  name.toLowerCase().endsWith('models') ? name : `${name} models`;

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
  const stored = getModelTypeStoredValueFromCustomProperties(customProperties) ?? ModelType.UNKNOWN;
  return buildCustomPropertiesWithModelType(undefined, stored);
};

export const getModelSizeFromCustomProperties = (
  customProperties?: ModelRegistryCustomProperties,
): string => {
  if (!customProperties) {
    return '';
  }
  const doubleVal = getDoubleValue(customProperties, 'modelcar_image_size');
  if (doubleVal > 0) {
    return `${doubleVal.toFixed(2)} GB`;
  }
  return getCustomPropString(customProperties, CatalogModelCustomPropertyKey.MODEL_SIZE);
};

export const getMinimumVramFromCustomProperties = (
  customProperties?: ModelRegistryCustomProperties,
): string => {
  if (!customProperties) {
    return '';
  }
  const doubleVal = getDoubleValue(customProperties, CatalogModelCustomPropertyKey.MINIMUM_VRAM);
  if (doubleVal > 0) {
    return `${doubleVal.toFixed(2)} GB`;
  }
  return getCustomPropString(customProperties, CatalogModelCustomPropertyKey.MINIMUM_VRAM);
};
