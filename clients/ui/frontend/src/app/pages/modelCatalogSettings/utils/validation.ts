import { CatalogSourceType } from '~/app/modelCatalogTypes';
import { ManageSourceFormData } from '~/app/pages/modelCatalogSettings/useManageSourceData';
import {
  isNonEmptyString,
  isSourceNameEmpty,
  SOURCE_NAME_CHARACTER_LIMIT,
  validateSourceName as validateSharedSourceName,
  validateYamlContent,
} from '~/app/shared/catalogSettings';

export { isSourceNameEmpty, validateYamlContent };

export const validateSourceName = (name: string): boolean =>
  validateSharedSourceName(name, SOURCE_NAME_CHARACTER_LIMIT);

export const validateOrganization = (organization: string): boolean =>
  isNonEmptyString(organization);

export const validateHuggingFaceCredentials = (data: ManageSourceFormData): boolean => {
  if (data.sourceType !== CatalogSourceType.HUGGING_FACE) {
    return true;
  }
  return validateOrganization(data.organization);
};

export const validateYamlMode = (data: ManageSourceFormData): boolean => {
  if (data.sourceType !== CatalogSourceType.YAML || data.isDefault) {
    return true;
  }
  return validateYamlContent(data.yamlContent);
};

export const isFormValid = (data: ManageSourceFormData): boolean =>
  validateSourceName(data.name) && validateHuggingFaceCredentials(data) && validateYamlMode(data);

export const isPreviewReady = (data: ManageSourceFormData): boolean => {
  if (data.sourceType === CatalogSourceType.HUGGING_FACE) {
    return validateHuggingFaceCredentials(data);
  }
  return validateYamlMode(data);
};
