import { formatSkillVersion } from '~/app/pages/skillCatalog/utils/skillCatalogUtils';

describe('formatSkillVersion', () => {
  it('abbreviates a full 40-character commit SHA to 7 characters', () => {
    expect(formatSkillVersion('9f8c1a2b3d4e5f60718293a4b5c6d7e8f9012345')).toBe('9f8c1a2');
  });

  it('abbreviates an uppercase SHA too', () => {
    expect(formatSkillVersion('9F8C1A2B3D4E5F60718293A4B5C6D7E8F9012345')).toBe('9F8C1A2');
  });

  it('leaves a semver tag untouched', () => {
    expect(formatSkillVersion('v1.2.3')).toBe('v1.2.3');
  });

  it('leaves a short hex-looking tag untouched', () => {
    // A tag may legitimately be named like an abbreviated SHA; truncating it would
    // misreport the version, so only exact 40-character hashes are shortened.
    expect(formatSkillVersion('abc1234')).toBe('abc1234');
  });

  it('leaves a 40-character non-hex string untouched', () => {
    const value = 'z'.repeat(40);
    expect(formatSkillVersion(value)).toBe(value);
  });
});
