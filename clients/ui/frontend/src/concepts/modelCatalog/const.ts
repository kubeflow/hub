export enum ModelCatalogStringFilterKey {
  TASK = 'tasks',
  PROVIDER = 'provider',
  LICENSE = 'license',
  LANGUAGE = 'language',
  TENSOR_TYPE = 'tensor_type.string_value',
  VALIDATED_CONFIGURATION = 'validatedTasks',
  // Performance filter keys use backend format
  HARDWARE_TYPE = 'artifacts.hardware_type.string_value',
  HARDWARE_CONFIGURATION = 'artifacts.hardware_configuration.string_value',
  USE_CASE = 'artifacts.use_case.string_value',
}

export enum ModelCatalogNumberFilterKey {
  // Performance filter key uses backend format
  MAX_RPS = 'artifacts.requests_per_second.double_value',
}

/**
 * Short property keys for accessing artifact customProperties directly.
 * These correspond to the performance filter keys but without the artifacts.* prefix and suffix.
 */
export const PerformancePropertyKey = {
  HARDWARE_TYPE: 'hardware_type',
  HARDWARE_CONFIGURATION: 'hardware_configuration',
  USE_CASE: 'use_case',
  REQUESTS_PER_SECOND: 'requests_per_second',
} as const;

export type PerformancePropertyKeyType =
  (typeof PerformancePropertyKey)[keyof typeof PerformancePropertyKey];

/**
 * The name of the default performance filters named query.
 * Used to look up default filter values in the namedQueries object.
 */
export const DEFAULT_PERFORMANCE_FILTERS_QUERY_NAME = 'default-performance-filters';

export enum LatencyMetric {
  E2E = 'E2E', // End to End
  TTFT = 'TTFT', // Time To First Token
  // TODO TPS is not technically a latency field, we should consider refactoring how it is handled in types
  //      and revisit the special logic that excludes it from latency filters.
  //      But it does have permutations with the same latency percentiles so leaving it here for now.
  TPS = 'TPS', // Tokens Per Second
  ITL = 'ITL', // Inter Token Latency
}
export const LatencyMetricLabels: Partial<Record<LatencyMetric, string>> = {
  [LatencyMetric.TTFT]: 'TTFT (time to first token)',
  [LatencyMetric.E2E]: 'E2E (end-to-end)',
  [LatencyMetric.ITL]: 'ITL (inter-token latency)',
};

export const latencyMetricDescriptions: Partial<Record<LatencyMetric, string>> = {
  [LatencyMetric.TTFT]: 'Time until the model starts responding. Best for interactive experiences.',
  [LatencyMetric.E2E]:
    'Total time to generate the full response. Best for summarization, batch jobs, and code generation.',
  [LatencyMetric.ITL]:
    'Time between tokens during generation. Important for smooth streaming and audio.',
};

export enum LatencyPercentile {
  Mean = 'Mean',
  P90 = 'P90',
  P95 = 'P95',
  P99 = 'P99',
}

/**
 * Short key format for accessing artifact customProperties (e.g., 'ttft_p90')
 */
export type LatencyPropertyKey = `${Lowercase<LatencyMetric>}_${Lowercase<LatencyPercentile>}`;

/**
 * Full key format for filter state, matching backend namedQueries format
 * (e.g., 'artifacts.ttft_p90.double_value')
 */
export type LatencyFilterKey = `artifacts.${LatencyPropertyKey}.double_value`;

// Keep LatencyMetricFieldName as alias for LatencyFilterKey for backward compatibility during migration
export type LatencyMetricFieldName = LatencyFilterKey;

const isMetricLowercase = (str: string): str is Lowercase<LatencyMetric> =>
  Object.values(LatencyMetric)
    .map((value) => value.toLowerCase())
    .includes(str);

const isPercentileLowercase = (str: string): str is Lowercase<LatencyPercentile> =>
  Object.values(LatencyPercentile)
    .map((value) => value.toLowerCase())
    .includes(str);

/**
 * Gets the short property key for accessing artifact customProperties (e.g., 'ttft_p90')
 */
export const getLatencyPropertyKey = (
  metric: LatencyMetric,
  percentile: LatencyPercentile,
): LatencyPropertyKey => {
  const metricPrefix = metric.toLowerCase();
  const percentileSuffix = percentile.toLowerCase();
  if (!isMetricLowercase(metricPrefix) || !isPercentileLowercase(percentileSuffix)) {
    return 'ttft_mean'; // Default fallback
  }
  return `${metricPrefix}_${percentileSuffix}`;
};

/**
 * Gets the full filter key for filter state, matching backend namedQueries format
 * (e.g., 'artifacts.ttft_p90.double_value')
 */
export const getLatencyFilterKey = (
  metric: LatencyMetric,
  percentile: LatencyPercentile,
): LatencyFilterKey => `artifacts.${getLatencyPropertyKey(metric, percentile)}.double_value`;

/**
 * All possible latency property keys (short format for customProperties access)
 */
export const ALL_LATENCY_PROPERTY_KEYS: LatencyPropertyKey[] = Object.values(LatencyMetric).flatMap(
  (metric) =>
    Object.values(LatencyPercentile).map((percentile) => getLatencyPropertyKey(metric, percentile)),
);

/**
 * All possible latency filter keys (full format for filter state)
 */
export const ALL_LATENCY_FILTER_KEYS: LatencyFilterKey[] = Object.values(LatencyMetric).flatMap(
  (metric) =>
    Object.values(LatencyPercentile).map((percentile) => getLatencyFilterKey(metric, percentile)),
);

// Type guard to check if a string is a valid LatencyFilterKey
export const isLatencyFilterKey = (value: string): value is LatencyFilterKey =>
  ALL_LATENCY_FILTER_KEYS.some((name) => name === value);

/**
 * Parses a LatencyFilterKey to extract the metric and percentile
 */
export const parseLatencyFilterKey = (
  filterKey: LatencyFilterKey,
): { metric: LatencyMetric; percentile: LatencyPercentile; propertyKey: LatencyPropertyKey } => {
  // Extract the property key from artifacts.{propertyKey}.double_value
  const match = filterKey.match(/^artifacts\.([a-z0-9_]+)\.double_value$/);
  const propertyKey = match?.[1] ?? 'ttft_mean';
  const [prefix, suffix] = propertyKey.split('_');
  const metric =
    Object.values(LatencyMetric).find((m) => m.toLowerCase() === prefix) ?? LatencyMetric.TTFT;
  const percentile =
    Object.values(LatencyPercentile).find((p) => p.toLowerCase() === suffix) ??
    LatencyPercentile.Mean;
  return {
    metric,
    percentile,
    propertyKey: getLatencyPropertyKey(metric, percentile),
  };
};

export enum UseCaseOptionValue {
  CHATBOT = 'chatbot',
  CODE_FIXING = 'code_fixing',
  LONG_RAG = 'long_rag',
  RAG = 'rag',
}

export enum ModelCatalogTask {
  AUDIO_TO_TEXT = 'audio-to-text',
  IMAGE_TEXT_TO_TEXT = 'image-text-to-text',
  IMAGE_TO_TEXT = 'image-to-text',
  TEXT_EMBEDDING = 'text-embedding',
  TEXT_GENERATION = 'text-generation',
  TEXT_TO_TEXT = 'text-to-text',
  TOOL_CALLING = 'tool-calling',
  VIDEO_TO_TEXT = 'video-to-text',
}

export const MODEL_CATALOG_TASK_NAME_MAPPING = {
  [ModelCatalogTask.AUDIO_TO_TEXT]: 'Audio-to-text',
  [ModelCatalogTask.IMAGE_TEXT_TO_TEXT]: 'Image-text-to-text',
  [ModelCatalogTask.IMAGE_TO_TEXT]: 'Image-to-text',
  [ModelCatalogTask.TEXT_EMBEDDING]: 'Text embedding',
  [ModelCatalogTask.TEXT_GENERATION]: 'Text generation',
  [ModelCatalogTask.TEXT_TO_TEXT]: 'Text-to-text',
  [ModelCatalogTask.TOOL_CALLING]: 'Tool calling',
  [ModelCatalogTask.VIDEO_TO_TEXT]: 'Video-to-text',
};

export enum ValidatedConfiguration {
  TOOL_CALLING = 'tool-calling',
}

export const MODEL_CATALOG_VALIDATED_CONFIGURATION_NAME_MAPPING: Record<
  ValidatedConfiguration,
  string
> = {
  [ValidatedConfiguration.TOOL_CALLING]: 'Tool calling',
};

export const MODEL_CATALOG_TASK_DESCRIPTION = {
  [ModelCatalogTask.AUDIO_TO_TEXT]: 'Audio transcription and speech recognition models',
  [ModelCatalogTask.IMAGE_TEXT_TO_TEXT]: 'Multimodal models that process both images and text',
  [ModelCatalogTask.IMAGE_TO_TEXT]: 'Image captioning and visual understanding models',
  [ModelCatalogTask.TEXT_EMBEDDING]: 'Models that convert text into vector embeddings',
  [ModelCatalogTask.TEXT_GENERATION]: 'Large language models for text generation',
  [ModelCatalogTask.TEXT_TO_TEXT]: 'Text transformation and translation models',
  [ModelCatalogTask.TOOL_CALLING]: 'Models optimized for function/tool calling',
  [ModelCatalogTask.VIDEO_TO_TEXT]: 'Video analysis and description models',
};

export enum ModelCatalogProvider {
  ALIBABA_CLOUD = 'Alibaba Cloud',
  DEEPSEEK = 'DeepSeek',
  GOOGLE = 'Google',
  IBM = 'IBM',
  META = 'Meta',
  MISTRAL_AI = 'Mistral AI',
  MOONSHOT_AI = 'Moonshot AI',
  NEURAL_MAGIC = 'Neural Magic',
  NVIDIA = 'NVIDIA',
  NVIDIA_ALTERNATE = 'Nvidia', // alternate casing
  RED_HAT = 'Red Hat',
}

export const MODEL_CATALOG_PROVIDER_NAME_MAPPING = {
  [ModelCatalogProvider.ALIBABA_CLOUD]: 'Alibaba Cloud',
  [ModelCatalogProvider.DEEPSEEK]: 'DeepSeek',
  [ModelCatalogProvider.GOOGLE]: 'Google',
  [ModelCatalogProvider.IBM]: 'IBM',
  [ModelCatalogProvider.META]: 'Meta',
  [ModelCatalogProvider.MISTRAL_AI]: 'Mistral AI',
  [ModelCatalogProvider.MOONSHOT_AI]: 'Moonshot AI',
  [ModelCatalogProvider.NEURAL_MAGIC]: 'Neural Magic',
  [ModelCatalogProvider.NVIDIA]: 'NVIDIA',
  [ModelCatalogProvider.NVIDIA_ALTERNATE]: 'NVIDIA',
  [ModelCatalogProvider.RED_HAT]: 'Red Hat',
};

export const MODEL_CATALOG_PROVIDER_NOTABLE_MODELS = {
  [ModelCatalogProvider.ALIBABA_CLOUD]: 'Qwen series models',
  [ModelCatalogProvider.DEEPSEEK]: 'DeepSeek reasoning models',
  [ModelCatalogProvider.GOOGLE]: 'Gemma series models',
  [ModelCatalogProvider.IBM]: 'Granite series models',
  [ModelCatalogProvider.META]: 'Llama series models',
  [ModelCatalogProvider.MISTRAL_AI]: 'Mistral series models',
  [ModelCatalogProvider.MOONSHOT_AI]: 'Kimi series models',
  [ModelCatalogProvider.NEURAL_MAGIC]: 'Quantized model variants',
  [ModelCatalogProvider.NVIDIA]: 'NVIDIA research models',
  [ModelCatalogProvider.NVIDIA_ALTERNATE]: 'NVIDIA research models',
  [ModelCatalogProvider.RED_HAT]: 'Red Hat optimized models',
};

export enum ModelCatalogTensorType {
  FP16 = 'FP16',
  FP8 = 'FP8',
  INT4 = 'INT4',
  INT8 = 'INT8',
  MXFP4 = 'MXFP4',
}

export const MODEL_CATALOG_POPOVER_MESSAGES = {
  VALIDATED:
    'Validated models are benchmarked for performance and quality using leading open source evaluation datasets.',
} as const;

export enum CatalogModelCustomPropertyKey {
  VALIDATED_ON = 'validated_on',
  TENSOR_TYPE = 'tensor_type',
  SIZE = 'size',
  ARCHITECTURE = 'architecture',
  MODEL_TYPE = 'model_type',
}

export enum ModelType {
  GENERATIVE = 'generative',
  PREDICTIVE = 'predictive',
  UNKNOWN = 'unknown',
}

export enum EuropeanLanguagesCode {
  BG = 'bg',
  CA = 'ca',
  CS = 'cs',
  DA = 'da',
  DE = 'de',
  EL = 'el',
  EN = 'en',
  ES = 'es',
  FI = 'fi',
  FR = 'fr',
  HR = 'hr',
  HU = 'hu',
  IS = 'is',
  IT = 'it',
  NL = 'nl',
  NLD = 'nld',
  NO = 'no',
  PL = 'pl',
  PT = 'pt',
  RO = 'ro',
  RU = 'ru',
  SK = 'sk',
  SL = 'sl',
  SR = 'sr',
  SV = 'sv',
  UK = 'uk',
}

export const MODEL_CATALOG_EUROPEAN_LANGUAGES_DETAILS = {
  [EuropeanLanguagesCode.BG]: 'Bulgarian',
  [EuropeanLanguagesCode.CA]: 'Catalan',
  [EuropeanLanguagesCode.CS]: 'Czech',
  [EuropeanLanguagesCode.DA]: 'Danish',
  [EuropeanLanguagesCode.DE]: 'German',
  [EuropeanLanguagesCode.EL]: 'Greek',
  [EuropeanLanguagesCode.EN]: 'English',
  [EuropeanLanguagesCode.ES]: 'Spanish',
  [EuropeanLanguagesCode.FI]: 'Finnish',
  [EuropeanLanguagesCode.FR]: 'French',
  [EuropeanLanguagesCode.HR]: 'Croatian',
  [EuropeanLanguagesCode.HU]: 'Hungarian',
  [EuropeanLanguagesCode.IS]: 'Icelandic',
  [EuropeanLanguagesCode.IT]: 'Italian',
  [EuropeanLanguagesCode.NL]: 'Dutch',
  [EuropeanLanguagesCode.NLD]: 'Dutch - variant',
  [EuropeanLanguagesCode.NO]: 'Norwegian',
  [EuropeanLanguagesCode.PL]: 'Polish',
  [EuropeanLanguagesCode.PT]: 'Portuguese',
  [EuropeanLanguagesCode.RO]: 'Romanian',
  [EuropeanLanguagesCode.RU]: 'Russian',
  [EuropeanLanguagesCode.SK]: 'Slovak',
  [EuropeanLanguagesCode.SL]: 'Slovenian',
  [EuropeanLanguagesCode.SR]: 'Serbian',
  [EuropeanLanguagesCode.SV]: 'Swedish',
  [EuropeanLanguagesCode.UK]: 'Ukrainian',
};

export enum AsianLanguagesCode {
  JA = 'ja',
  KO = 'ko',
  ZH = 'zh',
  HI = 'hi',
  TH = 'th',
  VI = 'vi',
  ID = 'id',
  MS = 'ms',
  ZSM = 'zsm',
}

export const MODEL_CATALOG_ASIAN_LANGUAGES_DETAILS = {
  [AsianLanguagesCode.JA]: 'Japanese',
  [AsianLanguagesCode.KO]: 'Korean',
  [AsianLanguagesCode.ZH]: 'Chinese',
  [AsianLanguagesCode.HI]: 'Hindi',
  [AsianLanguagesCode.TH]: 'Thai',
  [AsianLanguagesCode.VI]: 'Vietnamese',
  [AsianLanguagesCode.ID]: 'Indonesian',
  [AsianLanguagesCode.MS]: 'Malay',
  [AsianLanguagesCode.ZSM]: 'Standard Malay',
};

export enum MiddleEasternAndOtherLanguagesCode {
  AR = 'ar',
  FA = 'fa',
  HE = 'he',
  TR = 'tr',
  UR = 'ur',
  TL = 'tl',
}

export const MODEL_CATALOG_MIDDLE_EASTERN_AND_OTHER_LANGUAGES_DETAILS = {
  [MiddleEasternAndOtherLanguagesCode.AR]: 'Arabic',
  [MiddleEasternAndOtherLanguagesCode.FA]: 'Persian',
  [MiddleEasternAndOtherLanguagesCode.HE]: 'Hebrew',
  [MiddleEasternAndOtherLanguagesCode.TR]: 'Turkish',
  [MiddleEasternAndOtherLanguagesCode.UR]: 'Urdu',
  [MiddleEasternAndOtherLanguagesCode.TL]: 'Tagalog',
};

export const AllLanguageCodesMap = {
  ...MODEL_CATALOG_EUROPEAN_LANGUAGES_DETAILS,
  ...MODEL_CATALOG_ASIAN_LANGUAGES_DETAILS,
  ...MODEL_CATALOG_MIDDLE_EASTERN_AND_OTHER_LANGUAGES_DETAILS,
};

export enum AllLanguageCode {
  BG = 'bg',
  CA = 'ca',
  CS = 'cs',
  DA = 'da',
  DE = 'de',
  EL = 'el',
  EN = 'en',
  ES = 'es',
  FI = 'fi',
  FR = 'fr',
  HR = 'hr',
  HU = 'hu',
  IS = 'is',
  IT = 'it',
  NL = 'nl',
  NLD = 'nld',
  NO = 'no',
  PL = 'pl',
  PT = 'pt',
  RO = 'ro',
  RU = 'ru',
  SK = 'sk',
  SL = 'sl',
  SR = 'sr',
  SV = 'sv',
  UK = 'uk',
  JA = 'ja',
  KO = 'ko',
  ZH = 'zh',
  HI = 'hi',
  TH = 'th',
  VI = 'vi',
  ID = 'id',
  MS = 'ms',
  ZSM = 'zsm',
  AR = 'ar',
  FA = 'fa',
  HE = 'he',
  TR = 'tr',
  UR = 'ur',
  TL = 'tl',
}

export type ModelCatalogFilterKey =
  | ModelCatalogStringFilterKey
  | ModelCatalogNumberFilterKey
  | LatencyMetricFieldName;

/**
 * All possible filter keys combining string, number, and latency field keys
 */
export const ALL_CATALOG_FILTER_KEYS: ModelCatalogFilterKey[] = [
  ...Object.values(ModelCatalogStringFilterKey),
  ...Object.values(ModelCatalogNumberFilterKey),
  ...ALL_LATENCY_FILTER_KEYS,
];

/**
 * Type guard to check if a string is a valid ModelCatalogFilterKey
 */
export const isCatalogFilterKey = (key: string): key is ModelCatalogFilterKey =>
  ALL_CATALOG_FILTER_KEYS.some((k) => String(k) === key);

/**
 * Basic filter keys that appear on the catalog landing page (non-performance filters).
 */
export const BASIC_FILTER_KEYS: ModelCatalogFilterKey[] = [
  ModelCatalogStringFilterKey.PROVIDER,
  ModelCatalogStringFilterKey.LICENSE,
  ModelCatalogStringFilterKey.TASK,
  ModelCatalogStringFilterKey.LANGUAGE,
  ModelCatalogStringFilterKey.TENSOR_TYPE,
  ModelCatalogStringFilterKey.VALIDATED_CONFIGURATION,
];

/**
 * Filters that use AND logic when multiple values are selected.
 * Standard filters use OR (IN operator): items matching ANY selected value.
 * AND filters require items to match ALL selected values.
 * Add a filter key here to switch it from OR to AND behavior.
 */
export const MATCH_ALL_FILTER_KEYS: ModelCatalogStringFilterKey[] = [
  ModelCatalogStringFilterKey.VALIDATED_CONFIGURATION,
];

/**
 * Performance filter keys that are shown when performance view is enabled.
 * These filters should reset to default values (from namedQueries) instead of clearing.
 * Note: HARDWARE_CONFIGURATION is NOT included here because it should clear normally
 * like basic filters, not reset to defaults.
 */
export const PERFORMANCE_FILTER_KEYS: ModelCatalogFilterKey[] = [
  ModelCatalogStringFilterKey.USE_CASE,
  ModelCatalogNumberFilterKey.MAX_RPS,
  ...ALL_LATENCY_FILTER_KEYS,
];

/**
 * Check if a filter key is a performance filter (should reset to default instead of clear).
 */
export const isPerformanceFilterKey = (filterKey: ModelCatalogFilterKey): boolean =>
  PERFORMANCE_FILTER_KEYS.includes(filterKey);

/**
 * Performance string filter keys (arrays of strings).
 * Add new string-based performance filters here.
 */
export const PERFORMANCE_STRING_FILTER_KEYS: ModelCatalogStringFilterKey[] = [
  ModelCatalogStringFilterKey.USE_CASE,
  ModelCatalogStringFilterKey.HARDWARE_TYPE,
];

/**
 * Performance number filter keys (single number values).
 * Add new number-based performance filters here.
 */
export const PERFORMANCE_NUMBER_FILTER_KEYS: ModelCatalogNumberFilterKey[] = [
  ModelCatalogNumberFilterKey.MAX_RPS,
];

/**
 * Check if a filter key is a performance string filter.
 */
export const isPerformanceStringFilterKey = (
  filterKey: string,
): filterKey is ModelCatalogStringFilterKey =>
  PERFORMANCE_STRING_FILTER_KEYS.some((key) => key === filterKey);

/**
 * Check if a filter key is a performance number filter.
 */
export const isPerformanceNumberFilterKey = (
  filterKey: string,
): filterKey is ModelCatalogNumberFilterKey =>
  PERFORMANCE_NUMBER_FILTER_KEYS.some((key) => key === filterKey);

/**
 * Gets performance filter keys to show in the hardware configuration toolbar.
 * Only shows performance filters (not basic filters).
 * Includes HARDWARE_CONFIGURATION which is shown in performance toolbar but clears normally.
 */
export const getPerformanceFiltersToShow = (
  filterData: Partial<Record<LatencyMetricFieldName, number | undefined>>,
): ModelCatalogFilterKey[] => {
  const activeLatencyKeys = ALL_LATENCY_FILTER_KEYS.filter((key) => filterData[key] !== undefined);
  // Use Set to deduplicate since PERFORMANCE_FILTER_KEYS already includes latency fields
  return [
    ...new Set([
      ...PERFORMANCE_FILTER_KEYS,
      ModelCatalogStringFilterKey.HARDWARE_CONFIGURATION,
      ...activeLatencyKeys,
    ]),
  ];
};

/**
 * Gets all filter keys to show when performance view is enabled.
 * Includes basic filters, performance filters, and any active latency filters.
 */
export const getAllFiltersToShow = (
  filterData: Partial<Record<LatencyMetricFieldName, number | undefined>>,
): ModelCatalogFilterKey[] => {
  const activeLatencyKeys = ALL_LATENCY_FILTER_KEYS.filter((key) => filterData[key] !== undefined);
  // Use Set to deduplicate since PERFORMANCE_FILTER_KEYS already includes latency fields
  // Include HARDWARE_CONFIGURATION which shows in performance toolbar but clears normally
  return [
    ...new Set([
      ...BASIC_FILTER_KEYS,
      ...PERFORMANCE_FILTER_KEYS,
      ModelCatalogStringFilterKey.HARDWARE_CONFIGURATION,
      ...activeLatencyKeys,
    ]),
  ];
};

/**
 * Display names for filter categories.
 * Includes all ModelCatalogFilterKeys (ModelCatalogStringFilterKey | ModelCatalogNumberFilterKey | LatencyMetricFieldName).
 */
export const MODEL_CATALOG_FILTER_CATEGORY_NAMES: Record<ModelCatalogFilterKey, string> = {
  // String filter keys
  [ModelCatalogStringFilterKey.PROVIDER]: 'Provider',
  [ModelCatalogStringFilterKey.LICENSE]: 'License',
  [ModelCatalogStringFilterKey.TASK]: 'Task',
  [ModelCatalogStringFilterKey.LANGUAGE]: 'Language',
  [ModelCatalogStringFilterKey.HARDWARE_TYPE]: 'Hardware type',
  [ModelCatalogStringFilterKey.HARDWARE_CONFIGURATION]: 'Hardware',
  [ModelCatalogStringFilterKey.USE_CASE]: 'Workload type',
  [ModelCatalogStringFilterKey.TENSOR_TYPE]: 'Tensor type',
  [ModelCatalogStringFilterKey.VALIDATED_CONFIGURATION]: 'Validated configuration',
  // Number filter keys
  [ModelCatalogNumberFilterKey.MAX_RPS]: 'Max RPS',
  // Latency field names - all use "Latency" as category name
  // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
  ...(Object.fromEntries(ALL_LATENCY_FILTER_KEYS.map((field) => [field, 'Latency'])) as Record<
    LatencyMetricFieldName,
    string
  >),
};

export const MODEL_CATALOG_FILTER_CHIP_PREFIXES = {
  WORKLOAD_TYPE: 'Workload type:',
  MAX_RPS: 'Max RPS:',
  LATENCY_METRIC: 'Metric:',
  LATENCY_PERCENTILE: 'Percentile:',
  LATENCY_THRESHOLD: 'Under',
} as const;

export enum ModelDetailsTab {
  OVERVIEW = 'overview',
  PERFORMANCE_INSIGHTS = 'performance-insights',
}

export const EMPTY_CUSTOM_PROPERTY_VALUE = '-';

export enum ModelCatalogSortOption {
  RECENT_PUBLISH = 'recent_publish',
  LOWEST_LATENCY = 'lowest_latency',
}

export enum SortOrder {
  ASC = 'ASC',
  DESC = 'DESC',
}

export enum SortField {
  LAST_UPDATE_TIME = 'LAST_UPDATE_TIME',
}
