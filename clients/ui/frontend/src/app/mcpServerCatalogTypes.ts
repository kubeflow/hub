import { APIOptions } from 'mod-arch-core';
import { PaginationParams } from '~/app/shared/types/catalogTypes';
import { PreviewCatalogSourceQueryParams } from '~/app/modelCatalogTypes';

export type McpDeploymentMode = 'local' | 'remote';

export type McpTransportType = 'stdio' | 'sse' | 'http';

export type McpToolAccessType = 'read_only' | 'read_write' | 'execute';

export type McpEndpoints = {
  http?: string;
  sse?: string;
};

export type McpArtifact = {
  uri: string;
  createTimeSinceEpoch?: string;
  lastUpdateTimeSinceEpoch?: string;
};

export type McpSecurityIndicator = {
  verifiedSource?: boolean;
  secureEndpoint?: boolean;
  sast?: boolean;
  readOnlyTools?: boolean;
};

export type McpEnvVarMetadata = {
  name: string;
  description: string;
  required?: boolean;
  defaultValue?: string;
  type?: 'string' | 'int' | 'boolean' | 'secret';
  example?: string;
};

export type McpResourceRecommendation = {
  minimal?: { cpu?: string; memory?: string };
  recommended?: { cpu?: string; memory?: string };
  high?: { cpu?: string; memory?: string };
};

export type McpRuntimeMetadataHealthEndpoints = {
  liveness?: string;
  readiness?: string;
};

export type McpRuntimeMetadataCapabilities = {
  requiresNetwork?: boolean;
  requiresFileSystem?: boolean;
  requiresGPU?: boolean;
};

export type McpServiceAccountRequirement = {
  required?: boolean;
  hint?: string;
  suggestedName?: string;
};

export type McpSecretKey = {
  key: string;
  description: string;
  envVarName?: string;
  required?: boolean;
};

export type McpSecretRequirement = {
  name: string;
  description: string;
  keys?: McpSecretKey[];
  mountAsFile?: boolean;
  mountPath?: string;
};

export type McpConfigMapKey = {
  key: string;
  description: string;
  defaultContent?: string;
  envVarName?: string;
  required?: boolean;
};

export type McpConfigMapRequirement = {
  name: string;
  description: string;
  keys?: McpConfigMapKey[];
  mountAsFile?: boolean;
  mountPath?: string;
};

export type McpPrerequisites = {
  serviceAccount?: McpServiceAccountRequirement;
  secrets?: McpSecretRequirement[];
  configMaps?: McpConfigMapRequirement[];
  environmentVariables?: McpEnvVarMetadata[];
  customResources?: string[];
};

export type McpRuntimeMetadata = {
  defaultPort?: number;
  defaultArgs?: string[];
  requiredEnvironmentVariables?: McpEnvVarMetadata[];
  optionalEnvironmentVariables?: McpEnvVarMetadata[];
  recommendedResources?: McpResourceRecommendation;
  healthEndpoints?: McpRuntimeMetadataHealthEndpoints;
  capabilities?: McpRuntimeMetadataCapabilities;
  mcpPath?: string;
  prerequisites?: McpPrerequisites;
};

export type McpToolParameter = {
  name: string;
  type: string;
  description?: string;
  required: boolean;
};

export type McpTool = {
  name: string;
  description?: string;
  accessType: McpToolAccessType;
  parameters?: McpToolParameter[];
  revoked?: boolean;
  revokedReason?: string;
};

export type McpServer = {
  id: string;
  name: string;
  displayName?: string;
  source_id?: string;
  description?: string;
  logo?: string;
  license?: string;
  licenseLink?: string;
  provider?: string;
  version?: string;
  tags?: string[];
  toolCount: number;
  tools?: McpTool[];
  securityIndicators?: McpSecurityIndicator;
  documentationUrl?: string;
  repositoryUrl?: string;
  sourceCode?: string;
  lastUpdated?: string;
  publishedDate?: string;
  artifacts?: McpArtifact[];
  transports?: McpTransportType[];
  readme?: string;
  deploymentMode?: McpDeploymentMode;
  endpoints?: McpEndpoints;
  runtimeMetadata?: McpRuntimeMetadata;
};

export type McpServerList = PaginationParams & { items?: McpServer[] };

export type McpToolWithServer = {
  serverId: string;
  tool: McpTool;
};

export type McpToolList = PaginationParams & { items?: McpToolWithServer[] };

export type McpServerListParams = {
  sourceLabel?: string;
  pageSize?: number | string;
  nextPageToken?: string;
  filterQuery?: string;
  namedQuery?: string;
  includeTools?: boolean;
  toolLimit?: number;
  orderBy?: string;
  sortOrder?: string;
  name?: string;
  q?: string;
};

export enum McpCatalogSourceType {
  YAML = 'yaml',
}

export type McpCatalogSourceConfig = {
  id: string;
  name: string;
  type: McpCatalogSourceType;
  enabled?: boolean;
  labels?: string[];
  isDefault?: boolean;
  yaml?: string;
  yamlCatalogPath?: string;
  includedServers?: string[];
  excludedServers?: string[];
};

export type McpCatalogSourceConfigPayload =
  | McpCatalogSourceConfig
  | Pick<McpCatalogSourceConfig, 'enabled' | 'includedServers' | 'excludedServers'>;

export type McpCatalogSourceConfigList = {
  catalogs: McpCatalogSourceConfig[];
};

export type GetMcpCatalogSourceConfigs = (opts: APIOptions) => Promise<McpCatalogSourceConfigList>;
export type CreateMcpCatalogSourceConfig = (
  opts: APIOptions,
  data: McpCatalogSourceConfigPayload,
) => Promise<McpCatalogSourceConfig>;
export type GetMcpCatalogSourceConfig = (
  opts: APIOptions,
  sourceId: string,
) => Promise<McpCatalogSourceConfig>;
export type UpdateMcpCatalogSourceConfig = (
  opts: APIOptions,
  sourceId: string,
  data: Partial<McpCatalogSourceConfigPayload>,
) => Promise<McpCatalogSourceConfig>;
export type DeleteMcpCatalogSourceConfig = (opts: APIOptions, sourceId: string) => Promise<void>;

export type McpCatalogSourcePreviewRequest = {
  type: McpCatalogSourceType;
  includedServers?: string[];
  excludedServers?: string[];
  properties?: Record<string, unknown>;
};

export type McpCatalogSourcePreviewAsset = {
  name: string;
  included: boolean;
};

export type McpCatalogSourcePreviewSummary = {
  totalAssets: number;
  includedAssets: number;
  excludedAssets: number;
};

export type McpCatalogSourcePreviewResult = {
  items: McpCatalogSourcePreviewAsset[];
  summary: McpCatalogSourcePreviewSummary;
  nextPageToken: string;
  pageSize: number;
  size: number;
};

export type PreviewMcpCatalogSource = (
  opts: APIOptions,
  data: McpCatalogSourcePreviewRequest,
  queryParams?: PreviewCatalogSourceQueryParams,
) => Promise<McpCatalogSourcePreviewResult>;

export type McpCatalogSettingsAPIs = {
  getMcpCatalogSourceConfigs: GetMcpCatalogSourceConfigs;
  createMcpCatalogSourceConfig: CreateMcpCatalogSourceConfig;
  getMcpCatalogSourceConfig: GetMcpCatalogSourceConfig;
  updateMcpCatalogSourceConfig: UpdateMcpCatalogSourceConfig;
  deleteMcpCatalogSourceConfig: DeleteMcpCatalogSourceConfig;
  previewMcpCatalogSource: PreviewMcpCatalogSource;
};
