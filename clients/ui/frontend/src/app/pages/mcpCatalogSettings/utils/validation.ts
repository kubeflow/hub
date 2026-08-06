import { ManageMcpSourceFormData } from '~/app/pages/mcpCatalogSettings/useManageMcpSourceData';
import { SOURCE_NAME_CHARACTER_LIMIT } from '~/app/shared/catalogSettings/const';
import {
  validateSourceName as validateSharedSourceName,
  validateYamlContent,
} from '~/app/shared/catalogSettings/utils/validation';

export const validateMcpSourceName = (name: string): boolean =>
  validateSharedSourceName(name, SOURCE_NAME_CHARACTER_LIMIT);

export const isMcpFormValid = (data: ManageMcpSourceFormData): boolean => {
  if (data.isDefault) {
    return true;
  }
  return validateMcpSourceName(data.name) && validateYamlContent(data.yamlContent);
};

export const isMcpPreviewReady = (data: ManageMcpSourceFormData): boolean => {
  if (data.isDefault) {
    return true;
  }
  return validateYamlContent(data.yamlContent);
};
