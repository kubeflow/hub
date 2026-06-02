import { KeyValuePair } from 'mod-arch-core';
import {
  ModelRegistry,
  ModelRegistryCustomProperties,
  ModelRegistryCustomProperty,
  ModelRegistryCustomPropertyDouble,
  ModelRegistryCustomPropertyInt,
  ModelRegistryCustomPropertyString,
  ModelRegistryMetadataType,
  ModelVersion,
  RegisteredModel,
} from '~/app/types';
import { COMPANY_URI } from '~/app/utilities/const';
import { getLastCreatedItem } from '~/app/utils';
import {
  ModelRegistryFilterDataType,
  ModelRegistryVersionsFilterDataType,
} from '~/app/pages/modelRegistry/screens/const';
import {
  CatalogModelCustomPropertyKey,
  DEPLOYMENT_RESOURCE_PREFIXES,
} from '~/concepts/modelCatalog/const';

export type ObjectStorageFields = {
  endpoint: string;
  bucket: string;
  region?: string;
  path: string;
};

// Type for properties that can be displayed/edited in the UI (string, int, double)
export type ModelRegistryEditableCustomProperties = Record<
  string,
  | ModelRegistryCustomPropertyString
  | ModelRegistryCustomPropertyInt
  | ModelRegistryCustomPropertyDouble
>;

// Retrieves the labels from customProperties that have non-empty string_value.
export const getLabels = <T extends ModelRegistryCustomProperties>(customProperties: T): string[] =>
  Object.keys(customProperties).filter((key) => {
    const prop = customProperties[key];
    return prop.metadataType === ModelRegistryMetadataType.STRING && prop.string_value === '';
  });

// Extracts display labels from custom properties whose values (not keys) should be shown.
// For each key in the list, if the property exists and has a non-empty string value,
// that value is returned as a label.
export const getValueLabels = (
  customProperties: ModelRegistryCustomProperties,
  keys: string[],
): string[] =>
  keys
    .filter((key) => key in customProperties)
    .flatMap((key) => {
      const prop = customProperties[key];
      if (prop.metadataType === ModelRegistryMetadataType.STRING && prop.string_value) {
        return [prop.string_value];
      }
      return [];
    });

// Returns the customProperties object with an updated set of labels (non-empty string_value) without affecting other properties.
export const mergeUpdatedLabels = (
  customProperties: ModelRegistryCustomProperties,
  updatedLabels: string[],
): ModelRegistryCustomProperties => {
  const existingLabels = getLabels(customProperties);
  const addedLabels = updatedLabels.filter((label) => !existingLabels.includes(label));
  const removedLabels = existingLabels.filter((label) => !updatedLabels.includes(label));
  const customPropertiesCopy = { ...customProperties };
  removedLabels.forEach((label) => {
    delete customPropertiesCopy[label];
  });
  addedLabels.forEach((label) => {
    customPropertiesCopy[label] = {
      // eslint-disable-next-line camelcase
      string_value: '',
      metadataType: ModelRegistryMetadataType.STRING,
    };
  });
  return customPropertiesCopy;
};

// Extracts the value from a custom property as a string, regardless of its type
export const getPropertyValue = (
  prop:
    | ModelRegistryCustomPropertyString
    | ModelRegistryCustomPropertyInt
    | ModelRegistryCustomPropertyDouble,
): string => {
  switch (prop.metadataType) {
    case ModelRegistryMetadataType.STRING:
      return prop.string_value;
    case ModelRegistryMetadataType.INT:
      return prop.int_value;
    case ModelRegistryMetadataType.DOUBLE:
      return String(prop.double_value);
    default:
      return '';
  }
};

// Retrieves the customProperties that are not special (_registeredFrom/model_type) or labels (they have a defined string_value).
// Now includes INT and DOUBLE types in addition to STRING
export const getProperties = <T extends ModelRegistryCustomProperties>(
  customProperties: T,
): ModelRegistryEditableCustomProperties => {
  const initial: ModelRegistryEditableCustomProperties = {};
  return Object.keys(customProperties).reduce((acc, key) => {
    // _lastModified was used by a previous workaround and may still exist on older models; filter it out so users don't see it
    if (
      key === '_lastModified' ||
      key === CatalogModelCustomPropertyKey.MODEL_TYPE ||
      /^_registeredFrom/.test(key)
    ) {
      return acc;
    }

    const prop = customProperties[key];
    // Include STRING (non-empty), INT, and DOUBLE types
    // Exclude labels (STRING with empty value) and complex types (STRUCT, PROTO, BOOL)
    if (prop.metadataType === ModelRegistryMetadataType.STRING && prop.string_value !== '') {
      return { ...acc, [key]: prop };
    }
    if (
      prop.metadataType === ModelRegistryMetadataType.INT ||
      prop.metadataType === ModelRegistryMetadataType.DOUBLE
    ) {
      return { ...acc, [key]: prop };
    }
    return acc;
  }, initial);
};

const INTEGER_INPUT_PATTERN = /^-?\d+$/;
const DECIMAL_INPUT_PATTERN = /^-?\d+\.\d+$/;

// "007", "01", "-01" match INTEGER_INPUT_PATTERN but should stay STRING so leading zeros are preserved.
const hasLeadingZeroIntegerForm = (value: string): boolean => {
  const unsigned = value.startsWith('-') ? value.slice(1) : value;
  return unsigned.length > 1 && unsigned.startsWith('0');
};

// Returns the customProperties object with a single property added, updated or deleted
// Detects numeric types from value string and saves with appropriate type
export const mergeUpdatedProperty = (
  args: { customProperties: ModelRegistryCustomProperties } & (
    | { op: 'create'; newPair: KeyValuePair }
    | { op: 'update'; oldKey: string; newPair: KeyValuePair }
    | { op: 'delete'; oldKey: string }
  ),
): ModelRegistryCustomProperties => {
  const { op } = args;
  const customPropertiesCopy = { ...args.customProperties };
  if (op === 'delete' || (op === 'update' && args.oldKey !== args.newPair.key)) {
    delete customPropertiesCopy[args.oldKey];
  }
  if (op === 'create' || op === 'update') {
    const { key, value } = args.newPair;

    // Detect type from value string: INTEGER_INPUT_PATTERN → INT, DECIMAL_INPUT_PATTERN → DOUBLE, else STRING
    if (INTEGER_INPUT_PATTERN.test(value) && !hasLeadingZeroIntegerForm(value)) {
      // Integer value
      customPropertiesCopy[key] = {
        // eslint-disable-next-line camelcase
        int_value: value,
        metadataType: ModelRegistryMetadataType.INT,
      };
    } else if (DECIMAL_INPUT_PATTERN.test(value)) {
      // Decimal value
      customPropertiesCopy[key] = {
        // eslint-disable-next-line camelcase
        double_value: parseFloat(value),
        metadataType: ModelRegistryMetadataType.DOUBLE,
      };
    } else {
      // String value (default)
      customPropertiesCopy[key] = {
        // eslint-disable-next-line camelcase
        string_value: value,
        metadataType: ModelRegistryMetadataType.STRING,
      };
    }
  }
  return customPropertiesCopy;
};

export const getCustomPropString = <
  T extends Record<string, ModelRegistryCustomProperty | undefined>,
>(
  customProperties: T,
  key: string,
): string => {
  const prop = customProperties[key];

  if (prop?.metadataType === 'MetadataStringValue') {
    return prop.string_value;
  }
  return '';
};

const isMatchVersionKeyword = (mv: ModelVersion, keywordFilter: string): boolean =>
  mv.name.toLowerCase().includes(keywordFilter) ||
  (mv.description && mv.description.toLowerCase().includes(keywordFilter)) ||
  getLabels(mv.customProperties).some((label) => label.toLowerCase().includes(keywordFilter));

export const filterModelVersions = (
  unfilteredModelVersions: ModelVersion[],
  filterData: ModelRegistryVersionsFilterDataType,
): ModelVersion[] => {
  const keywordFilter = filterData.Keyword?.toLowerCase();
  const authorFilter = filterData.Author?.toLowerCase();

  return unfilteredModelVersions.filter((mv: ModelVersion) => {
    if (!keywordFilter && !authorFilter) {
      return true;
    }

    const doesNotMatchVersion = keywordFilter && !isMatchVersionKeyword(mv, keywordFilter);

    if (doesNotMatchVersion) {
      return false;
    }

    return !authorFilter || mv.author?.toLowerCase().includes(authorFilter);
  });
};

export const sortModelVersionsByCreateTime = (registeredModels: ModelVersion[]): ModelVersion[] =>
  registeredModels.toSorted((a, b) => {
    const first = parseInt(a.createTimeSinceEpoch);
    const second = parseInt(b.createTimeSinceEpoch);
    return new Date(second).getTime() - new Date(first).getTime();
  });

export const filterRegisteredModels = (
  unfilteredRegisteredModels: RegisteredModel[],
  unfilteredModelVersions: ModelVersion[],
  filterData: ModelRegistryFilterDataType,
): RegisteredModel[] => {
  const keywordFilter = filterData.Keyword?.toLowerCase();
  const ownerFilter = filterData.Owner?.toLowerCase();

  return unfilteredRegisteredModels.filter((rm: RegisteredModel) => {
    if (!keywordFilter && !ownerFilter) {
      return true;
    }
    const modelVersions = unfilteredModelVersions.filter((mv) => mv.registeredModelId === rm.id);
    const doesNotMatchModel =
      keywordFilter &&
      !(
        rm.name.toLowerCase().includes(keywordFilter) ||
        (rm.description && rm.description.toLowerCase().includes(keywordFilter)) ||
        getLabels(rm.customProperties).some((label) => label.toLowerCase().includes(keywordFilter))
      );

    const doesNotMatchVersions =
      keywordFilter &&
      !modelVersions.some((mv: ModelVersion) => isMatchVersionKeyword(mv, keywordFilter));

    if (doesNotMatchModel && doesNotMatchVersions) {
      return false;
    }

    return !ownerFilter || rm.owner?.toLowerCase().includes(ownerFilter);
  });
};

export const getTextValue = (v: string | string[]): string | undefined => {
  const s = typeof v === 'string' ? v : v.filter(Boolean).join(' ');
  return s || undefined;
};

export const getServerAddress = (resource: ModelRegistry): string => resource.serverAddress || '';

export const isValidHttpUrl = (value: string): boolean => {
  try {
    const url = new URL(value);
    const isHttp = url.protocol === 'http:' || url.protocol === 'https:';
    // Domain validation
    const domainPattern = /^(?!-)[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)*\.[A-Za-z]{2,}$/;

    return isHttp && domainPattern.test(url.hostname);
  } catch {
    return false;
  }
};

export const isCompanyUri = (uri: string): boolean => uri.startsWith(`${COMPANY_URI}/`);

export const getLatestVersionForRegisteredModel = (
  modelVersions: ModelVersion[],
  rmId: string,
): ModelVersion | undefined => {
  const filteredVersions = modelVersions.filter((mv) => mv.registeredModelId === rmId);
  const latestVersion = getLastCreatedItem(filteredVersions);
  return latestVersion;
};

const isDeploymentResource = (entry: string): boolean =>
  DEPLOYMENT_RESOURCE_PREFIXES.some((prefix) => entry.toLowerCase().startsWith(prefix));

const getValidatedOnEntries = <T extends ModelRegistryCustomProperties>(
  customProperties: T | undefined,
): string[] => {
  if (!customProperties) {
    return [];
  }

  const validatedOnString = getCustomPropString(
    customProperties,
    CatalogModelCustomPropertyKey.VALIDATED_ON,
  );

  if (!validatedOnString) {
    return [];
  }

  try {
    const parsed = JSON.parse(validatedOnString);
    if (Array.isArray(parsed)) {
      return parsed
        .filter((item) => typeof item === 'string')
        .map((item) => item.trim())
        .filter((item) => item.length > 0);
    }
    return [];
  } catch {
    return [];
  }
};

export const getValidatedOnPlatforms = <T extends ModelRegistryCustomProperties>(
  customProperties: T | undefined,
): string[] =>
  getValidatedOnEntries(customProperties).filter((entry) => !isDeploymentResource(entry));

export const getValidatedDeploymentResources = <T extends ModelRegistryCustomProperties>(
  customProperties: T | undefined,
): string[] => getValidatedOnEntries(customProperties).filter(isDeploymentResource);
