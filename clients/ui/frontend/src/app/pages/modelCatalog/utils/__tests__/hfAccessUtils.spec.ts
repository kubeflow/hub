import type { CatalogModel } from '~/app/modelCatalogTypes';
import { HfAccessType } from '~/concepts/modelCatalog/const';
import {
  getHfAccessLabelVariant,
  getHfAccessType,
  isHfGatedAccessDeniedFromFields,
} from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import { isPreviewModelGatedAccessDenied } from '~/app/pages/modelCatalogSettings/utils/modelCatalogSettingsUtils';
import { createHfAccessCatalogModel } from '~/__tests__/utils/createHfAccessModel';

describe('HF access utilities', () => {
  it('returns null for models without hf_access_type', () => {
    const model: CatalogModel = { name: 'org/model' };
    expect(getHfAccessType(model)).toBeNull();
    expect(getHfAccessLabelVariant(model)).toBeNull();
  });

  it('returns private for private HF models', () => {
    const model = createHfAccessCatalogModel({ hfAccessType: HfAccessType.PRIVATE });
    expect(getHfAccessLabelVariant(model)).toBe('private');
  });

  it('returns gated for gated models with access granted', () => {
    const autoGranted = createHfAccessCatalogModel({
      hfAccessType: HfAccessType.GATED_AUTO,
      hfGatedAccessGranted: 'true',
    });
    const manualGranted = createHfAccessCatalogModel({
      hfAccessType: HfAccessType.GATED_MANUAL,
      hfGatedAccessGranted: 'true',
    });

    expect(getHfAccessLabelVariant(autoGranted)).toBe('gated');
    expect(getHfAccessLabelVariant(manualGranted)).toBe('gated');
  });

  it('returns gated for gated models regardless of access grant metadata', () => {
    const autoWithoutGrant = createHfAccessCatalogModel({
      hfAccessType: HfAccessType.GATED_AUTO,
      hfGatedAccessGranted: 'false',
    });
    const manualWithoutGrant = createHfAccessCatalogModel({
      hfAccessType: HfAccessType.GATED_MANUAL,
      hfGatedAccessGranted: 'false',
    });
    const missingGrant = createHfAccessCatalogModel({ hfAccessType: HfAccessType.GATED_AUTO });

    expect(getHfAccessLabelVariant(autoWithoutGrant)).toBe('gated');
    expect(getHfAccessLabelVariant(manualWithoutGrant)).toBe('gated');
    expect(getHfAccessLabelVariant(missingGrant)).toBe('gated');
  });

  it('returns null for public HF models', () => {
    const model = createHfAccessCatalogModel({ hfAccessType: HfAccessType.PUBLIC });
    expect(getHfAccessLabelVariant(model)).toBeNull();
  });
});

describe('isHfGatedAccessDeniedFromFields', () => {
  it('returns true for gated models without granted access', () => {
    expect(isHfGatedAccessDeniedFromFields('gated_auto', false)).toBe(true);
    expect(isHfGatedAccessDeniedFromFields('gated_manual', undefined)).toBe(true);
  });

  it('returns false for gated models with granted access', () => {
    expect(isHfGatedAccessDeniedFromFields('gated_auto', true)).toBe(false);
  });

  it('returns false for non-gated access types', () => {
    expect(isHfGatedAccessDeniedFromFields('public', false)).toBe(false);
    expect(isHfGatedAccessDeniedFromFields(undefined, false)).toBe(false);
  });
});

describe('isPreviewModelGatedAccessDenied', () => {
  it('uses the shared gated access rule for preview fields', () => {
    expect(
      isPreviewModelGatedAccessDenied({
        name: 'sample-source/included-model-2',
        included: true,
        hfAccessType: 'gated_manual',
        hfGatedAccessGranted: false,
      }),
    ).toBe(true);

    expect(
      isPreviewModelGatedAccessDenied({
        name: 'sample-source/included-model-1',
        included: true,
        hfAccessType: 'gated_auto',
        hfGatedAccessGranted: true,
      }),
    ).toBe(false);
  });
});
