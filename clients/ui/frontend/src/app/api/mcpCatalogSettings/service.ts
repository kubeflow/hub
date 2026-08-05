import {
  McpCatalogSourceConfig,
  McpCatalogSourceConfigList,
  McpCatalogSourceConfigPayload,
  McpCatalogSourcePreviewRequest,
  McpCatalogSourcePreviewResult,
} from '~/app/mcpServerCatalogTypes';
import { createSourceConfigService } from '~/app/shared/catalogSettings';

const service = createSourceConfigService<
  McpCatalogSourceConfig,
  McpCatalogSourceConfigList,
  McpCatalogSourceConfigPayload,
  McpCatalogSourcePreviewRequest,
  McpCatalogSourcePreviewResult
>({
  previewExtraQueryParams: { assetType: 'mcp_servers' },
});

export const getMcpCatalogSourceConfigs = service.getSourceConfigs;
export const createMcpCatalogSourceConfig = service.createSourceConfig;
export const getMcpCatalogSourceConfig = service.getSourceConfig;
export const updateMcpCatalogSourceConfig = service.updateSourceConfig;
export const deleteMcpCatalogSourceConfig = service.deleteSourceConfig;
export const previewMcpCatalogSource = service.previewSource;
