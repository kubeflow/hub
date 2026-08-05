import {
  isNonEmptyString,
  validateSourceName,
  isSourceNameEmpty,
  validateYamlContent,
} from '~/app/shared/catalogSettings/utils/validation';

describe('isNonEmptyString', () => {
  it('should return true for non-empty strings', () => {
    expect(isNonEmptyString('test')).toBe(true);
    expect(isNonEmptyString(' a ')).toBe(true);
  });

  it('should return false for empty or whitespace-only strings', () => {
    expect(isNonEmptyString('')).toBe(false);
    expect(isNonEmptyString('   ')).toBe(false);
    expect(isNonEmptyString('\t')).toBe(false);
  });
});

describe('validateSourceName', () => {
  it('should return true for valid names within the limit', () => {
    expect(validateSourceName('My Source', 238)).toBe(true);
    expect(validateSourceName('a'.repeat(238), 238)).toBe(true);
  });

  it('should return false for empty or whitespace-only names', () => {
    expect(validateSourceName('', 238)).toBe(false);
    expect(validateSourceName('   ', 238)).toBe(false);
  });

  it('should return false when name exceeds max length', () => {
    expect(validateSourceName('a'.repeat(239), 238)).toBe(false);
  });
});

describe('isSourceNameEmpty', () => {
  it('should return true for empty or whitespace-only strings', () => {
    expect(isSourceNameEmpty('')).toBe(true);
    expect(isSourceNameEmpty('   ')).toBe(true);
  });

  it('should return false for non-empty strings', () => {
    expect(isSourceNameEmpty('test')).toBe(false);
  });
});

describe('validateYamlContent', () => {
  it('should return true for non-empty YAML content', () => {
    expect(validateYamlContent('source: test')).toBe(true);
  });

  it('should return false for empty or whitespace-only content', () => {
    expect(validateYamlContent('')).toBe(false);
    expect(validateYamlContent('   ')).toBe(false);
  });
});
