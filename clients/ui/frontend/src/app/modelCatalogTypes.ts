import { APIOptions } from 'mod-arch-core';
import {
  ModelCatalogTask,
  ModelCatalogProvider,
  AllLanguageCode,
  ModelCatalogStringFilterKey,
  ModelCatalogNumberFilterKey,
  LatencyMetricFieldName,
  LatencyPropertyKey,
  UseCaseOptionValue,
  ModelCatalogFilterKey,
  ModelCatalogTensorType,
} from '~/concepts/modelCatalog/const';
import {
  ModelRegistryCustomProperties,
  ModelRegistryCustomPropertyString,
  ModelRegistryCustomPropertyInt,
  ModelRegistryCustomPropertyDouble,
} from './types';
import {
  McpServer,
  McpServerList,
  McpServerListParams,
  McpToolList,
} from './mcpServerCatalogTypes';

export type CatalogSource = {
  id: string;
  name: string;
  labels: string[];
  enabled?: boolean;
  status?: 'available' | 'partially-available' | 'error' | 'disabled';
  error?: string;
};

export type CatalogSourceList = PaginationParams & { items?: CatalogSource[] };

export type ToolCallingConfig = {
  args?: string;
};

export type ServingConfig = {
  toolCalling?: ToolCallingConfig;
};

export type CatalogModel = {
  source_id?: string;
  name: string;
  provider?: string;
  readme?: string;
  maturity?: string;
  language?: string[];
  logo?: string;
  tasks?: string[];
  validatedTasks?: string[];
  libraryName?: string;
  license?: string;
  licenseLink?: string;
  description?: string;
  createTimeSinceEpoch?: string;
  lastUpdateTimeSinceEpoch?: string;
  customProperties?: ModelRegistryCustomProperties;
  servingConfig?: ServingConfig;
};

export type PaginationParams = {
  size: number;
  pageSize: number;
  nextPageToken: string;
};

export type CatalogAssetType = 'models' | 'mcp_servers';

export type CatalogSourceListParams = {
  assetType?: CatalogAssetType;
};

export type CatalogModelList = PaginationParams & { items: CatalogModel[] };

export enum CatalogArtifactType {
  modelArtifact = 'model-artifact',
  metricsArtifact = 'metrics-artifact',
}

export enum MetricsType {
  accuracyMetrics = 'accuracy-metrics',
  performanceMetrics = 'performance-metrics',
}

export enum CategoryName {
  allModels = 'All models',
  otherModels = 'Other models',
}

export enum SourceLabel {
  other = 'null',
}

export type CatalogLabel = {
  name: string | null;
  displayName?: string;
  description?: string;
};

export type CatalogLabelList = PaginationParams & { items: CatalogLabel[] };

export enum CatalogSourceType {
  YAML = 'yaml',
  HUGGING_FACE = 'hf',
}

export type CatalogArtifactBase = {
  createTimeSinceEpoch?: string;
  lastUpdateTimeSinceEpoch?: string;
  customProperties?: ModelRegistryCustomProperties;
};

export type CatalogModelArtifact = CatalogArtifactBase & {
  artifactType: CatalogArtifactType.modelArtifact;
  uri: string;
};

export type PerformanceMetricsCustomProperties = {
  config_id?: ModelRegistryCustomPropertyString;
  hardware_configuration?: ModelRegistryCustomPropertyString;
  /** @deprecated Use hardware_configuration instead. Should not be used for filtering or display. */
  hardware_type?: ModelRegistryCustomPropertyString;
  /** @deprecated Use hardware_configuration instead. Should not be used for filtering or display. */
  hardware_count?: ModelRegistryCustomPropertyInt;
  requests_per_second?: ModelRegistryCustomPropertyDouble;
  // Token metrics
  mean_input_tokens?: ModelRegistryCustomPropertyDouble;
  mean_output_tokens?: ModelRegistryCustomPropertyDouble;
  // Use case information
  use_case?: ModelRegistryCustomPropertyString;
  // Framework information
  framework?: ModelRegistryCustomPropertyString;
  framework_version?: ModelRegistryCustomPropertyString;
  // Additional fields from ADR (excluded from display per requirements)
  docker_image?: ModelRegistryCustomPropertyString;
  entrypoint?: ModelRegistryCustomPropertyString;
  inserted_at?: ModelRegistryCustomPropertyString;
  created_at?: ModelRegistryCustomPropertyString;
  updated_at?: ModelRegistryCustomPropertyString;
  model_hf_repo_name?: ModelRegistryCustomPropertyString;
  scenario_id?: ModelRegistryCustomPropertyString;
  // Computed properties when targetRPS is provided
  replicas?: ModelRegistryCustomPropertyInt;
  total_requests_per_second?: ModelRegistryCustomPropertyDouble;
} & Partial<Record<LatencyPropertyKey, ModelRegistryCustomPropertyDouble>>;

export type AccuracyMetricsCustomProperties = {
  // overall_average?: ModelRegistryCustomPropertyDouble; // NOTE: overall_average is currently omitted from the API and will be restored
  arc_v1?: ModelRegistryCustomPropertyDouble;
} & Record<string, ModelRegistryCustomPropertyDouble>;

export type CatalogPerformanceMetricsArtifact = Omit<CatalogArtifactBase, 'customProperties'> & {
  artifactType: CatalogArtifactType.metricsArtifact;
  metricsType: MetricsType.performanceMetrics;
  customProperties?: PerformanceMetricsCustomProperties;
};

export type CatalogAccuracyMetricsArtifact = Omit<CatalogArtifactBase, 'customProperties'> & {
  artifactType: CatalogArtifactType.metricsArtifact;
  metricsType: MetricsType.accuracyMetrics;
  customProperties?: AccuracyMetricsCustomProperties;
};

export type CatalogMetricsArtifact =
  | CatalogPerformanceMetricsArtifact
  | CatalogAccuracyMetricsArtifact;

export type CatalogArtifacts = CatalogModelArtifact | CatalogMetricsArtifact;

export type CatalogArtifactList = PaginationParams & { items: CatalogArtifacts[] };

export type CatalogPerformanceArtifactList = PaginationParams & {
  items: CatalogPerformanceMetricsArtifact[];
};

export type CatalogFilterNumberOption = {
  type: 'number';
  range?: {
    max?: number;
    min?: number;
  };
};

export type CatalogFilterStringOption<T extends string> = {
  type: 'string';
  values?: T[];
};

export type GetCatalogModelsBySource = (
  opts: APIOptions,
  sourceId?: string,
  sourceLabel?: string,
  paginationParams?: {
    pageSize?: string;
    nextPageToken?: string;
    orderBy?: string;
    sortOrder?: string;
  },
  searchKeyword?: string,
  filterData?: ModelCatalogFilterStates,
  filterOptions?: CatalogFilterOptionsList | null,
  filterQuery?: string,
  performanceParams?: {
    targetRPS?: number;
    latencyProperty?: string;
    recommendations?: boolean;
  },
) => Promise<CatalogModelList>;

export type GetListSources = (
  opts: APIOptions,
  listParams?: CatalogSourceListParams,
) => Promise<CatalogSourceList>;

export type GetCatalogModel = (
  opts: APIOptions,
  sourceId: string,
  modelName: string,
) => Promise<CatalogModel>;

export type GetListCatalogModelArtifacts = (
  opts: APIOptions,
  sourceId: string,
  modelName: string,
  filterQuery?: string,
) => Promise<CatalogArtifactList>;

export type GetPerformanceArtifacts = (
  opts: APIOptions,
  sourceId: string,
  modelName: string,
  params?: PerformanceArtifactsParams,
  filterData?: ModelCatalogFilterStates,
  filterOptions?: CatalogFilterOptionsList | null,
) => Promise<CatalogPerformanceArtifactList>;

export type GetArtifactFilterOptions = (
  opts: APIOptions,
  sourceId: string,
  modelName: string,
) => Promise<CatalogFilterOptionsList>;

export type GetCatalogFilterOptionList = (opts: APIOptions) => Promise<CatalogFilterOptionsList>;

export type CatalogLabelListParams = {
  assetType?: CatalogAssetType;
};

export type GetCatalogLabels = (
  opts: APIOptions,
  listParams?: CatalogLabelListParams,
) => Promise<CatalogLabelList>;

export type GetMcpServerList = (
  opts: APIOptions,
  listParams?: McpServerListParams,
) => Promise<McpServerList>;

export type GetMcpServerFilterOptionList = (opts: APIOptions) => Promise<CatalogFilterOptionsList>;

export type GetMcpServer = (opts: APIOptions, serverId: string) => Promise<McpServer>;

export type GetMcpServerToolList = (opts: APIOptions, serverId: string) => Promise<McpToolList>;

export type ModelCatalogAPIs = {
  getCatalogModelsBySource: GetCatalogModelsBySource;
  getListSources: GetListSources;
  getCatalogModel: GetCatalogModel;
  getListCatalogModelArtifacts: GetListCatalogModelArtifacts;
  getCatalogFilterOptionList: GetCatalogFilterOptionList;
  getPerformanceArtifacts: GetPerformanceArtifacts;
  getCatalogLabels: GetCatalogLabels;
  getMcpServerList: GetMcpServerList;
  getMcpServerFilterOptionList: GetMcpServerFilterOptionList;
  getMcpServer: GetMcpServer;
  getMcpServerToolList: GetMcpServerToolList;
};

export type CatalogModelDetailsParams = {
  sourceId?: string;
  modelName?: string;
};

export type { ModelCatalogFilterKey };

// Not used for a run time value, just for mapping other types
export type ModelCatalogStringFilterValueType = {
  [ModelCatalogStringFilterKey.TASK]: ModelCatalogTask;
  [ModelCatalogStringFilterKey.PROVIDER]: ModelCatalogProvider;
  [ModelCatalogStringFilterKey.LICENSE]: string;
  [ModelCatalogStringFilterKey.LANGUAGE]: AllLanguageCode;
  [ModelCatalogStringFilterKey.TENSOR_TYPE]: ModelCatalogTensorType;
  [ModelCatalogStringFilterKey.VALIDATED_CONFIGURATION]: string;
  [ModelCatalogStringFilterKey.HARDWARE_TYPE]: string;
  [ModelCatalogStringFilterKey.HARDWARE_CONFIGURATION]: string;
  [ModelCatalogStringFilterKey.USE_CASE]: UseCaseOptionValue;
};

export type ModelCatalogStringFilterOptions = {
  [key in ModelCatalogStringFilterKey]?: CatalogFilterStringOption<
    ModelCatalogStringFilterValueType[key]
  >;
};

export type CatalogFilterOptions = ModelCatalogStringFilterOptions & {
  [key in ModelCatalogNumberFilterKey]?: CatalogFilterNumberOption;
} & {
  [key in LatencyMetricFieldName]?: CatalogFilterNumberOption;
};

export enum FilterOperator {
  LESS_THAN = '<',
  EQUALS = '=',
  GREATER_THAN = '>',
  LESS_THAN_OR_EQUAL = '<=',
  GREATER_THAN_OR_EQUAL = '>=',
  NOT_EQUAL = '!=',
  IN = 'IN',
  LIKE = 'LIKE',
  ILIKE = 'ILIKE',
}

export type FieldFilter = {
  operator: FilterOperator;
  value: string | number | boolean | (string | number)[];
};

export type NamedQuery = Record<string, FieldFilter>;

export type CatalogFilterOptionsList = {
  filters?: CatalogFilterOptions;
  namedQueries?: Record<string, NamedQuery>;
};

export type PerformanceArtifactsParams = {
  targetRPS?: number;
  recommendations?: boolean;
  rpsProperty?: string;
  latencyProperty?: string;
  hardwareCountProperty?: string;
  hardwareTypeProperty?: string;
  filterQuery?: string;
  pageSize?: string;
  orderBy?: string;
  sortOrder?: string;
  nextPageToken?: string;
};

export type ComputedPerformanceProperties = {
  replicas?: number;
  total_requests_per_second?: number;
};

export type ModelCatalogFilterStates = {
  [ModelCatalogStringFilterKey.TASK]: ModelCatalogTask[];
  [ModelCatalogStringFilterKey.PROVIDER]: ModelCatalogProvider[];
  [ModelCatalogStringFilterKey.LICENSE]: string[];
  [ModelCatalogStringFilterKey.LANGUAGE]: AllLanguageCode[];
  [ModelCatalogStringFilterKey.TENSOR_TYPE]: ModelCatalogTensorType[];
  [ModelCatalogStringFilterKey.VALIDATED_CONFIGURATION]: string[];
  [ModelCatalogStringFilterKey.HARDWARE_TYPE]: string[];
  [ModelCatalogStringFilterKey.HARDWARE_CONFIGURATION]: string[];
  [ModelCatalogStringFilterKey.USE_CASE]: UseCaseOptionValue[];
} & {
  [key in ModelCatalogNumberFilterKey]: number | undefined;
} & {
  [key in LatencyMetricFieldName]?: number | undefined;
};

// Model Catalog Settings types
export type CatalogSourceConfigCommon = {
  id: string;
  name: string;
  enabled?: boolean;
  labels?: string[];
  includedModels?: string[];
  excludedModels?: string[];
  isDefault?: boolean;
};

export type YamlCatalogSourceConfig = CatalogSourceConfigCommon & {
  type: CatalogSourceType.YAML;
  /** yaml and yamlCatalogPath will be populated on GET (by ID) requests, not on LIST requests */
  yaml?: string;
  yamlCatalogPath?: string;
};

export type HuggingFaceCatalogSourceConfig = CatalogSourceConfigCommon & {
  type: CatalogSourceType.HUGGING_FACE;
  allowedOrganization?: string;
  /** apiKey will be populated on GET (by ID) requests, not on LIST requests */
  apiKey?: string;
};

export type CatalogSourceConfig = YamlCatalogSourceConfig | HuggingFaceCatalogSourceConfig;

export type CatalogSourceConfigPayload =
  | CatalogSourceConfig
  | Pick<CatalogSourceConfig, 'enabled' | 'includedModels' | 'excludedModels'>;

export type CatalogSourceConfigList = {
  catalogs: CatalogSourceConfig[];
};

export type GetCatalogSourceConfigs = (opts: APIOptions) => Promise<CatalogSourceConfigList>;
export type CreateCatalogSourceConfig = (
  opts: APIOptions,
  data: CatalogSourceConfigPayload,
) => Promise<CatalogSourceConfig>;
export type GetCatalogSourceConfig = (
  opts: APIOptions,
  sourceId: string,
) => Promise<CatalogSourceConfig>;
export type UpdateCatalogSourceConfig = (
  opts: APIOptions,
  sourceId: string,
  data: Partial<CatalogSourceConfigPayload>,
) => Promise<CatalogSourceConfig>;
export type DeleteCatalogSourceConfig = (opts: APIOptions, sourceId: string) => Promise<void>;

// Preview types
export type CatalogSourcePreviewRequest = {
  type: string;
  includedModels?: string[];
  excludedModels?: string[];
  properties?: Record<string, unknown>;
};

export type CatalogSourcePreviewModel = {
  name: string;
  included: boolean;
};

export type CatalogSourcePreviewSummary = {
  totalModels: number;
  includedModels: number;
  excludedModels: number;
};

export type CatalogSourcePreviewResult = {
  items: CatalogSourcePreviewModel[];
  summary: CatalogSourcePreviewSummary;
  nextPageToken: string;
  pageSize: number;
  size: number;
};

export type PreviewCatalogSourceQueryParams = {
  filterStatus?: 'all' | 'included' | 'excluded';
  pageSize?: number;
  nextPageToken?: string;
};

export type PreviewCatalogSource = (
  opts: APIOptions,
  data: CatalogSourcePreviewRequest,
  queryParams?: PreviewCatalogSourceQueryParams,
) => Promise<CatalogSourcePreviewResult>;

export type ModelCatalogSettingsAPIs = {
  getCatalogSourceConfigs: GetCatalogSourceConfigs;
  createCatalogSourceConfig: CreateCatalogSourceConfig;
  getCatalogSourceConfig: GetCatalogSourceConfig;
  updateCatalogSourceConfig: UpdateCatalogSourceConfig;
  deleteCatalogSourceConfig: DeleteCatalogSourceConfig;
  previewCatalogSource: PreviewCatalogSource;
};
