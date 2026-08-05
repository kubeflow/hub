import { generateSourceIdFromName } from '~/app/shared/catalogSettings/utils/generateSourceIdFromName';

describe('generateSourceIdFromName', () => {
  it('should trim extra spaces', () => {
    expect(generateSourceIdFromName('  testname')).toBe('testname');
  });

  it('should replace - with _', () => {
    expect(generateSourceIdFromName('test-name')).toBe('test_name');
  });

  it('should Remove anything that is NOT alphanumeric and NOT underscore and replace it with _', () => {
    expect(generateSourceIdFromName('Test-Name!')).toBe('test_name');
  });

  it('should convert upper case to lower case', () => {
    expect(generateSourceIdFromName('TestName')).toBe('testname');
  });

  it('should replace multiple spaces with underscores', () => {
    expect(generateSourceIdFromName('My MCP Source')).toBe('my_mcp_source');
  });

  it('should handle special characters', () => {
    expect(generateSourceIdFromName('Source #1 (test)')).toBe('source_1_test');
  });
});
