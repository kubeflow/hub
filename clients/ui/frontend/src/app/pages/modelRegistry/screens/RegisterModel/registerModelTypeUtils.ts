import { CatalogModelCustomPropertyKey, ModelType } from '~/concepts/modelCatalog/const';
import { ModelRegistryCustomProperties, ModelRegistryMetadataType } from '~/app/types';

export const MODEL_TYPE_CUSTOM_PROPERTY_KEY = CatalogModelCustomPropertyKey.MODEL_TYPE;

export type RegisterableModelType = ModelType.GENERATIVE | ModelType.PREDICTIVE | ModelType.UNKNOWN;

/** Raw `model_type` string for display (any non-empty STRING metadata), or null if unset. */
export const getModelTypeRawStringFromCustomProperties = (
  customProperties: ModelRegistryCustomProperties | undefined,
): string | null => {
  const prop = customProperties?.[MODEL_TYPE_CUSTOM_PROPERTY_KEY];
  if (!prop || prop.metadataType !== ModelRegistryMetadataType.STRING) {
    return null;
  }
  const v = prop.string_value.trim();
  return v || null;
};

export const getModelTypeStoredValueFromCustomProperties = (
  props: ModelRegistryCustomProperties | undefined,
): RegisterableModelType | undefined => {
  const prop = props?.[MODEL_TYPE_CUSTOM_PROPERTY_KEY];
  if (!prop || prop.metadataType !== ModelRegistryMetadataType.STRING) {
    return undefined;
  }
  const v = prop.string_value.toLowerCase().trim();
  if (v === ModelType.GENERATIVE || v === ModelType.PREDICTIVE || v === ModelType.UNKNOWN) {
    return v;
  }
  return undefined;
};

export const buildCustomPropertiesWithModelType = (
  base: ModelRegistryCustomProperties | undefined,
  next: RegisterableModelType | undefined,
): ModelRegistryCustomProperties => {
  const result = { ...(base ?? {}) };
  if (!next) {
    delete result[MODEL_TYPE_CUSTOM_PROPERTY_KEY];
  } else {
    result[MODEL_TYPE_CUSTOM_PROPERTY_KEY] = {
      metadataType: ModelRegistryMetadataType.STRING,
      // eslint-disable-next-line camelcase
      string_value: next,
    };
  }
  return result;
};
