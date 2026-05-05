import {
  ModelRegistryCustomProperties,
  ModelRegistryCustomPropertyString,
  ModelRegistryMetadataType,
} from '~/app/types';
import { ModelType } from '~/concepts/modelCatalog/const';
import {
  buildCustomPropertiesWithModelType,
  getModelTypeRawStringFromCustomProperties,
  getModelTypeStoredValueFromCustomProperties,
  MODEL_TYPE_CUSTOM_PROPERTY_KEY,
} from '~/app/pages/modelRegistry/screens/RegisterModel/registerModelTypeUtils';

const stringProp = (value: string): ModelRegistryCustomPropertyString => ({
  metadataType: ModelRegistryMetadataType.STRING,
  // eslint-disable-next-line camelcase
  string_value: value,
});

describe('registerModelTypeUtils', () => {
  describe('getModelTypeRawStringFromCustomProperties', () => {
    it('returns null when custom properties are undefined', () => {
      expect(getModelTypeRawStringFromCustomProperties(undefined)).toBeNull();
    });

    it('returns null when model_type is missing', () => {
      expect(getModelTypeRawStringFromCustomProperties({})).toBeNull();
    });

    it('returns null when metadata is not STRING', () => {
      expect(
        getModelTypeRawStringFromCustomProperties({
          [MODEL_TYPE_CUSTOM_PROPERTY_KEY]: {
            metadataType: ModelRegistryMetadataType.INT,
            // eslint-disable-next-line camelcase
            int_value: '1',
          },
        }),
      ).toBeNull();
    });

    it('returns null when string_value is empty or whitespace', () => {
      expect(
        getModelTypeRawStringFromCustomProperties({
          [MODEL_TYPE_CUSTOM_PROPERTY_KEY]: stringProp(''),
        }),
      ).toBeNull();
      expect(
        getModelTypeRawStringFromCustomProperties({
          [MODEL_TYPE_CUSTOM_PROPERTY_KEY]: stringProp('   '),
        }),
      ).toBeNull();
    });

    it('returns trimmed raw string without normalizing case', () => {
      expect(
        getModelTypeRawStringFromCustomProperties({
          [MODEL_TYPE_CUSTOM_PROPERTY_KEY]: stringProp('  Generative  '),
        }),
      ).toBe('Generative');
    });
  });

  describe('getModelTypeStoredValueFromCustomProperties', () => {
    it('returns undefined when custom properties are undefined', () => {
      expect(getModelTypeStoredValueFromCustomProperties(undefined)).toBeUndefined();
    });

    it('returns undefined when model_type is missing or not STRING', () => {
      expect(getModelTypeStoredValueFromCustomProperties({})).toBeUndefined();
      expect(
        getModelTypeStoredValueFromCustomProperties({
          [MODEL_TYPE_CUSTOM_PROPERTY_KEY]: {
            metadataType: ModelRegistryMetadataType.INT,
            // eslint-disable-next-line camelcase
            int_value: '1',
          },
        }),
      ).toBeUndefined();
    });

    it('returns unknown when string matches ModelType.UNKNOWN', () => {
      expect(
        getModelTypeStoredValueFromCustomProperties({
          [MODEL_TYPE_CUSTOM_PROPERTY_KEY]: stringProp(ModelType.UNKNOWN),
        }),
      ).toBe(ModelType.UNKNOWN);
    });

    it('returns undefined for unrecognized STRING values', () => {
      expect(
        getModelTypeStoredValueFromCustomProperties({
          [MODEL_TYPE_CUSTOM_PROPERTY_KEY]: stringProp('other'),
        }),
      ).toBeUndefined();
    });

    it('returns generative or predictive when string matches after lowercasing and trim', () => {
      expect(
        getModelTypeStoredValueFromCustomProperties({
          [MODEL_TYPE_CUSTOM_PROPERTY_KEY]: stringProp(ModelType.GENERATIVE),
        }),
      ).toBe(ModelType.GENERATIVE);
      expect(
        getModelTypeStoredValueFromCustomProperties({
          [MODEL_TYPE_CUSTOM_PROPERTY_KEY]: stringProp('  GENERATIVE  '),
        }),
      ).toBe(ModelType.GENERATIVE);
      expect(
        getModelTypeStoredValueFromCustomProperties({
          [MODEL_TYPE_CUSTOM_PROPERTY_KEY]: stringProp(ModelType.PREDICTIVE),
        }),
      ).toBe(ModelType.PREDICTIVE);
    });
  });

  describe('buildCustomPropertiesWithModelType', () => {
    it('returns empty object when base is undefined and model type is cleared', () => {
      expect(buildCustomPropertiesWithModelType(undefined, undefined)).toEqual({});
    });

    it('sets model_type when next is generative or predictive', () => {
      expect(buildCustomPropertiesWithModelType(undefined, ModelType.GENERATIVE)).toEqual({
        [MODEL_TYPE_CUSTOM_PROPERTY_KEY]: stringProp(ModelType.GENERATIVE),
      });
      expect(buildCustomPropertiesWithModelType(undefined, ModelType.PREDICTIVE)).toEqual({
        [MODEL_TYPE_CUSTOM_PROPERTY_KEY]: stringProp(ModelType.PREDICTIVE),
      });
    });

    it('merges with base properties and overwrites model_type', () => {
      const base: ModelRegistryCustomProperties = {
        otherKey: stringProp('keep-me'),
        [MODEL_TYPE_CUSTOM_PROPERTY_KEY]: stringProp(ModelType.GENERATIVE),
      };
      expect(buildCustomPropertiesWithModelType(base, ModelType.PREDICTIVE)).toEqual({
        otherKey: stringProp('keep-me'),
        [MODEL_TYPE_CUSTOM_PROPERTY_KEY]: stringProp(ModelType.PREDICTIVE),
      });
    });

    it('removes model_type when next is undefined and preserves other keys', () => {
      const base: ModelRegistryCustomProperties = {
        otherKey: stringProp('keep-me'),
        [MODEL_TYPE_CUSTOM_PROPERTY_KEY]: stringProp(ModelType.GENERATIVE),
      };
      expect(buildCustomPropertiesWithModelType(base, undefined)).toEqual({
        otherKey: stringProp('keep-me'),
      });
    });

    it('does not mutate the base object', () => {
      const base: ModelRegistryCustomProperties = {
        [MODEL_TYPE_CUSTOM_PROPERTY_KEY]: stringProp(ModelType.GENERATIVE),
      };
      const copy = { ...base };
      buildCustomPropertiesWithModelType(base, ModelType.PREDICTIVE);
      expect(base).toEqual(copy);
    });
  });
});
