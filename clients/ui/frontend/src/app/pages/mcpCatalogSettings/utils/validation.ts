import { ManageMcpSourceFormData } from '~/app/pages/mcpCatalogSettings/useManageMcpSourceData';
import {
  isSourceNameEmpty,
  SOURCE_NAME_CHARACTER_LIMIT,
  validateSourceName as validateSharedSourceName,
  validateYamlContent,
} from '~/app/shared/catalogSettings';

export const validateMcpSourceName = (name: string): boolean =>
  validateSharedSourceName(name, SOURCE_NAME_CHARACTER_LIMIT);

export const isMcpSourceNameEmpty = isSourceNameEmpty;

export const validateMcpYamlContent = validateYamlContent;

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
