import { ManageMcpSourceFormData } from '~/app/pages/mcpCatalogSettings/useManageMcpSourceData';
import { MCP_SOURCE_NAME_CHARACTER_LIMIT } from '~/app/pages/mcpCatalogSettings/constants';

const isNonEmptyString = (value: string): boolean => value.trim().length > 0;

export const validateMcpSourceName = (name: string): boolean =>
  isNonEmptyString(name) && name.length <= MCP_SOURCE_NAME_CHARACTER_LIMIT;

export const isMcpSourceNameEmpty = (name: string): boolean => !isNonEmptyString(name);

export const validateMcpYamlContent = (yamlContent: string): boolean =>
  isNonEmptyString(yamlContent);

export const isMcpFormValid = (data: ManageMcpSourceFormData): boolean => {
  if (data.isDefault) {
    return true;
  }
  return validateMcpSourceName(data.name) && validateMcpYamlContent(data.yamlContent);
};

export const isMcpPreviewReady = (data: ManageMcpSourceFormData): boolean => {
  if (data.isDefault) {
    return true;
  }
  return validateMcpYamlContent(data.yamlContent);
};
