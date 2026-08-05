import { McpCatalogSourceConfig, McpCatalogSourceConfigPayload } from '~/app/mcpServerCatalogTypes';
import { ManageMcpSourceFormData } from '~/app/pages/mcpCatalogSettings/useManageMcpSourceData';
import {
  generateSourceIdFromName,
  parseCommaSeparatedList,
} from '~/app/shared/catalogSettings';

export const mcpSourceConfigToFormData = (
  sourceConfig: McpCatalogSourceConfig,
): Partial<ManageMcpSourceFormData> => ({
  name: sourceConfig.name,
  id: sourceConfig.id,
  sourceType: sourceConfig.type,
  enabled: sourceConfig.enabled ?? true,
  yamlContent: sourceConfig.yaml ?? '',
  includedServers: (sourceConfig.includedServers || []).join(', '),
  excludedServers: (sourceConfig.excludedServers || []).join(', '),
  isDefault: sourceConfig.isDefault ?? false,
});

export const transformMcpFormDataToConfig = (
  formData: ManageMcpSourceFormData,
  existingSourceConfig?: McpCatalogSourceConfig,
): McpCatalogSourceConfig => ({
  id: formData.id || existingSourceConfig?.id || generateSourceIdFromName(formData.name),
  name: formData.name,
  type: formData.sourceType,
  enabled: formData.enabled,
  isDefault: formData.isDefault,
  yaml: formData.yamlContent || undefined,
  yamlCatalogPath: existingSourceConfig?.yamlCatalogPath,
  includedServers: parseCommaSeparatedList(formData.includedServers),
  excludedServers: parseCommaSeparatedList(formData.excludedServers),
});

export const getMcpPayloadForConfig = (
  sourceConfig: McpCatalogSourceConfig,
  isEditMode = false,
): McpCatalogSourceConfigPayload => {
  if (sourceConfig.isDefault) {
    return {
      enabled: sourceConfig.enabled,
      includedServers: sourceConfig.includedServers,
      excludedServers: sourceConfig.excludedServers,
    };
  }

  if (isEditMode) {
    return {
      name: sourceConfig.name,
      type: sourceConfig.type,
      enabled: sourceConfig.enabled,
      isDefault: sourceConfig.isDefault,
      yaml: sourceConfig.yaml,
      includedServers: sourceConfig.includedServers,
      excludedServers: sourceConfig.excludedServers,
    };
  }

  return sourceConfig;
};
