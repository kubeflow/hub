import { McpCatalogSourceType } from '~/app/mcpServerCatalogTypes';
import {
  validateMcpSourceName,
  isMcpSourceNameEmpty,
  validateMcpYamlContent,
  isMcpFormValid,
  isMcpPreviewReady,
} from '~/app/pages/mcpCatalogSettings/utils/validation';
import { ManageMcpSourceFormData } from '~/app/pages/mcpCatalogSettings/useManageMcpSourceData';

const baseFormData: ManageMcpSourceFormData = {
  name: '',
  id: '',
  sourceType: McpCatalogSourceType.YAML,
  yamlContent: '',
  includedServers: '',
  excludedServers: '',
  enabled: true,
  isDefault: false,
};

describe('validateMcpSourceName', () => {
  it('should return true for valid names', () => {
    expect(validateMcpSourceName('My Source')).toBe(true);
    expect(validateMcpSourceName('a')).toBe(true);
    expect(validateMcpSourceName('Source with numbers 123')).toBe(true);
  });

  it('should return false for empty name', () => {
    expect(validateMcpSourceName('')).toBe(false);
  });

  it('should return false for whitespace-only name', () => {
    expect(validateMcpSourceName('   ')).toBe(false);
    expect(validateMcpSourceName('\t')).toBe(false);
  });

  it('should return false for name exceeding character limit', () => {
    const longName = 'a'.repeat(239);
    expect(validateMcpSourceName(longName)).toBe(false);
  });

  it('should return true for name at character limit', () => {
    const maxName = 'a'.repeat(238);
    expect(validateMcpSourceName(maxName)).toBe(true);
  });
});

describe('isMcpSourceNameEmpty', () => {
  it('should return true for empty string', () => {
    expect(isMcpSourceNameEmpty('')).toBe(true);
  });

  it('should return true for whitespace-only string', () => {
    expect(isMcpSourceNameEmpty('   ')).toBe(true);
  });

  it('should return false for non-empty string', () => {
    expect(isMcpSourceNameEmpty('test')).toBe(false);
  });
});

describe('validateMcpYamlContent', () => {
  it('should return true for non-empty YAML content', () => {
    expect(validateMcpYamlContent('source: test')).toBe(true);
    expect(validateMcpYamlContent('mcp_servers:\n  - name: srv')).toBe(true);
  });

  it('should return false for empty string', () => {
    expect(validateMcpYamlContent('')).toBe(false);
  });

  it('should return false for whitespace-only string', () => {
    expect(validateMcpYamlContent('   ')).toBe(false);
    expect(validateMcpYamlContent('\n\t')).toBe(false);
  });
});

describe('isMcpFormValid', () => {
  it('should return false when name is empty', () => {
    const data = { ...baseFormData, name: '', yamlContent: 'source: test' };
    expect(isMcpFormValid(data)).toBe(false);
  });

  it('should return false when YAML content is empty', () => {
    const data = { ...baseFormData, name: 'Test', yamlContent: '' };
    expect(isMcpFormValid(data)).toBe(false);
  });

  it('should return false when both name and YAML are empty', () => {
    expect(isMcpFormValid(baseFormData)).toBe(false);
  });

  it('should return true when name and YAML are both filled', () => {
    const data = { ...baseFormData, name: 'Test Source', yamlContent: 'source: test' };
    expect(isMcpFormValid(data)).toBe(true);
  });

  it('should return true for default source regardless of YAML content', () => {
    const data = { ...baseFormData, isDefault: true, name: '', yamlContent: '' };
    expect(isMcpFormValid(data)).toBe(true);
  });

  it('should return true for default source with any fields', () => {
    const data = { ...baseFormData, isDefault: true, name: 'Default', yamlContent: '' };
    expect(isMcpFormValid(data)).toBe(true);
  });
});

describe('isMcpPreviewReady', () => {
  it('should return false when YAML content is empty', () => {
    const data = { ...baseFormData, yamlContent: '' };
    expect(isMcpPreviewReady(data)).toBe(false);
  });

  it('should return true when YAML content is provided', () => {
    const data = { ...baseFormData, yamlContent: 'source: test' };
    expect(isMcpPreviewReady(data)).toBe(true);
  });

  it('should return true for default source regardless of YAML', () => {
    const data = { ...baseFormData, isDefault: true, yamlContent: '' };
    expect(isMcpPreviewReady(data)).toBe(true);
  });

  it('should not require name for preview readiness', () => {
    const data = { ...baseFormData, name: '', yamlContent: 'source: test' };
    expect(isMcpPreviewReady(data)).toBe(true);
  });
});
