import { ManageMcpSourceFormData } from '~/app/pages/mcpCatalogSettings/useManageMcpSourceData';
import { MCP_SOURCE_NAME_CHARACTER_LIMIT } from '~/app/pages/mcpCatalogSettings/constants';
import {
  isSourceNameEmpty as isSharedSourceNameEmpty,
  validateSourceName as validateSharedSourceName,
  validateYamlContent as validateSharedYamlContent,
} from '~/app/shared/catalogSettings';

export const validateMcpSourceName = (name: string): boolean =>
  validateSharedSourceName(name, MCP_SOURCE_NAME_CHARACTER_LIMIT);

export const isMcpSourceNameEmpty = isSharedSourceNameEmpty;

export const validateMcpYamlContent = validateSharedYamlContent;

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
