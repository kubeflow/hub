/* eslint-disable camelcase */
import { CatalogModel } from '~/app/modelCatalogTypes';
import { CatalogModelCustomPropertyKey, HfAccessType } from '~/concepts/modelCatalog/const';
import { ModelRegistryMetadataType } from '~/app/types';
import {
  getHfAccessLabelVariant,
  getHfAccessType,
  getHfGatedAccessGranted,
  getHuggingFaceModelUrl,
  isHfGatedAccessDenied,
} from '~/app/pages/modelCatalog/utils/modelCatalogUtils';

const createHfModel = (accessType: string, gatedAccessGranted?: string): CatalogModel => ({
  name: 'org/model',
  customProperties: {
    [CatalogModelCustomPropertyKey.HF_ACCESS_TYPE]: {
      string_value: accessType,
      metadataType: ModelRegistryMetadataType.STRING,
    },
    ...(gatedAccessGranted !== undefined
      ? {
          [CatalogModelCustomPropertyKey.HF_GATED_ACCESS_GRANTED]: {
            string_value: gatedAccessGranted,
            metadataType: ModelRegistryMetadataType.STRING,
          },
        }
      : {}),
  },
});

describe('HF access utilities', () => {
  it('returns null for models without hf_access_type', () => {
    const model: CatalogModel = { name: 'org/model' };
    expect(getHfAccessType(model)).toBeNull();
    expect(getHfAccessLabelVariant(model)).toBeNull();
    expect(isHfGatedAccessDenied(model)).toBe(false);
  });

  it('returns private for private HF models', () => {
    const model = createHfModel(HfAccessType.PRIVATE);
    expect(getHfAccessLabelVariant(model)).toBe('private');
    expect(isHfGatedAccessDenied(model)).toBe(false);
  });

  it('returns gated for gated models with access granted', () => {
    const autoGranted = createHfModel(HfAccessType.GATED_AUTO, 'true');
    const manualGranted = createHfModel(HfAccessType.GATED_MANUAL, 'true');

    expect(getHfAccessLabelVariant(autoGranted)).toBe('gated');
    expect(getHfAccessLabelVariant(manualGranted)).toBe('gated');
    expect(isHfGatedAccessDenied(autoGranted)).toBe(false);
  });

  it('returns gated-denied for gated models without access', () => {
    const autoDenied = createHfModel(HfAccessType.GATED_AUTO, 'false');
    const manualDenied = createHfModel(HfAccessType.GATED_MANUAL, 'false');

    expect(getHfAccessLabelVariant(autoDenied)).toBe('gated-denied');
    expect(getHfAccessLabelVariant(manualDenied)).toBe('gated-denied');
    expect(isHfGatedAccessDenied(autoDenied)).toBe(true);
    expect(isHfGatedAccessDenied(manualDenied)).toBe(true);
  });

  it('returns gated-denied when hf_gated_access_granted is missing on gated models', () => {
    const model = createHfModel(HfAccessType.GATED_AUTO);

    expect(getHfGatedAccessGranted(model)).toBe(false);
    expect(getHfAccessLabelVariant(model)).toBe('gated-denied');
    expect(isHfGatedAccessDenied(model)).toBe(true);
  });

  it('reads hf_gated_access_granted from boolean metadata', () => {
    const model: CatalogModel = {
      name: 'org/model',
      customProperties: {
        [CatalogModelCustomPropertyKey.HF_ACCESS_TYPE]: {
          string_value: HfAccessType.GATED_AUTO,
          metadataType: ModelRegistryMetadataType.STRING,
        },
        [CatalogModelCustomPropertyKey.HF_GATED_ACCESS_GRANTED]: {
          bool_value: true,
          metadataType: ModelRegistryMetadataType.BOOL,
        },
      },
    };

    expect(getHfGatedAccessGranted(model)).toBe(true);
    expect(getHfAccessLabelVariant(model)).toBe('gated');
  });

  it('returns null for public HF models', () => {
    const model = createHfModel(HfAccessType.PUBLIC);
    expect(getHfAccessLabelVariant(model)).toBeNull();
  });

  it('builds the Hugging Face model URL from the model name', () => {
    const model: CatalogModel = { name: 'meta-llama/Llama-3.1-8B-Instruct-INT8' };
    expect(getHuggingFaceModelUrl(model)).toBe(
      'https://huggingface.co/meta-llama/Llama-3.1-8B-Instruct-INT8',
    );
  });
});
