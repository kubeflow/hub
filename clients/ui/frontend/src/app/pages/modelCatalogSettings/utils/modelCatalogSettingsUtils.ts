import {
  CatalogSourceConfig,
  CatalogSourceConfigPayload,
  CatalogSourceType,
} from '~/app/modelCatalogTypes';
import { ManageSourceFormData } from '~/app/pages/modelCatalogSettings/useManageSourceData';
import { generateSourceIdFromName } from '~/app/shared/catalogSettings/utils/generateSourceIdFromName';
import { parseCommaSeparatedList } from '~/app/shared/catalogSettings/utils/parseCommaSeparatedList';

export const catalogSourceConfigToFormData = (
  sourceConfig: CatalogSourceConfig,
): Partial<ManageSourceFormData> => {
  const common: Partial<ManageSourceFormData> = {
    name: sourceConfig.name,
    sourceType: sourceConfig.type,
    enabled: sourceConfig.enabled ?? true,
    allowedModels: (sourceConfig.includedModels || []).join(', '),
    excludedModels: (sourceConfig.excludedModels || []).join(', '),
    isDefault: sourceConfig.isDefault,
    id: sourceConfig.id,
  };

  if (sourceConfig.type === CatalogSourceType.YAML) {
    return {
      ...common,
      yamlContent: sourceConfig.yaml ?? '',
      accessToken: '',
      organization: '',
    };
  }

  return {
    ...common,
    accessToken: sourceConfig.apiKey ?? '',
    organization: sourceConfig.allowedOrganization ?? '',
    yamlContent: '',
  };
};

export const transformFormDataToConfig = (
  formData: ManageSourceFormData,
  existingSourceConfig?: CatalogSourceConfig,
): CatalogSourceConfig => {
  const commonFields = {
    id: formData.id || generateSourceIdFromName(formData.name),
    name: formData.name,
    enabled: formData.enabled,
    isDefault: formData.isDefault,
    includedModels: parseCommaSeparatedList(formData.allowedModels),
    excludedModels: parseCommaSeparatedList(formData.excludedModels),
  };

  if (formData.sourceType === CatalogSourceType.YAML) {
    return {
      ...commonFields,
      type: CatalogSourceType.YAML,
      yaml: formData.yamlContent,
      yamlCatalogPath:
        existingSourceConfig?.type === CatalogSourceType.YAML
          ? existingSourceConfig.yamlCatalogPath
          : undefined,
    };
  }

  return {
    ...commonFields,
    type: CatalogSourceType.HUGGING_FACE,
    apiKey: formData.accessToken,
    allowedOrganization: formData.organization,
  };
};

export const getPayloadForConfig = (
  sourceConfig: CatalogSourceConfig,
  isEditMode = false,
): CatalogSourceConfigPayload => {
  if (sourceConfig.isDefault) {
    return {
      enabled: sourceConfig.enabled,
      includedModels: sourceConfig.includedModels,
      excludedModels: sourceConfig.excludedModels,
    };
  }

  if (isEditMode) {
    return {
      name: sourceConfig.name,
      type: sourceConfig.type,
      enabled: sourceConfig.enabled,
      isDefault: sourceConfig.isDefault,
      includedModels: sourceConfig.includedModels,
      excludedModels: sourceConfig.excludedModels,
      ...(sourceConfig.type === CatalogSourceType.YAML && { yaml: sourceConfig.yaml }),
      ...(sourceConfig.type === CatalogSourceType.HUGGING_FACE && {
        apiKey: sourceConfig.apiKey,
        allowedOrganization: sourceConfig.allowedOrganization,
      }),
    };
  }

  return sourceConfig;
};
