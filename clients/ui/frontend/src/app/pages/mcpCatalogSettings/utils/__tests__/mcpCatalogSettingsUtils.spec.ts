import { mockMcpCatalogSourceConfig } from '~/__mocks__';
import { McpCatalogSourceType } from '~/app/mcpServerCatalogTypes';
import {
  mcpSourceConfigToFormData,
  generateMcpSourceIdFromName,
  getMcpPayloadForConfig,
  transformMcpFormDataToConfig,
} from '~/app/pages/mcpCatalogSettings/utils/mcpCatalogSettingsUtils';
import { ManageMcpSourceFormData } from '~/app/pages/mcpCatalogSettings/useManageMcpSourceData';

const defaultSourceConfig = mockMcpCatalogSourceConfig({});
const nonDefaultSourceConfig = mockMcpCatalogSourceConfig({
  id: 'mcp_source_2',
  name: 'MCP Source 2',
  isDefault: false,
  yaml: 'source: test\nmcp_servers:\n  - name: Kubernetes',
  includedServers: ['Kubernetes', 'GitHub'],
  excludedServers: ['*preview*'],
  enabled: false,
});

const defaultFormData: ManageMcpSourceFormData = {
  name: 'MCP Source 1',
  id: 'sample_mcp_source_1',
  sourceType: McpCatalogSourceType.YAML,
  yamlContent: '',
  includedServers: '',
  excludedServers: '',
  enabled: true,
  isDefault: true,
};

const nonDefaultFormData: ManageMcpSourceFormData = {
  name: 'MCP Source 2',
  id: 'mcp_source_2',
  sourceType: McpCatalogSourceType.YAML,
  yamlContent: 'source: test\nmcp_servers:\n  - name: Kubernetes',
  includedServers: 'Kubernetes, GitHub',
  excludedServers: '*preview*',
  enabled: false,
  isDefault: false,
};

describe('generateMcpSourceIdFromName', () => {
  it('should trim extra spaces', () => {
    expect(generateMcpSourceIdFromName('  testname')).toBe('testname');
  });

  it('should replace hyphens with underscores', () => {
    expect(generateMcpSourceIdFromName('test-name')).toBe('test_name');
  });

  it('should remove non-alphanumeric characters except underscores', () => {
    expect(generateMcpSourceIdFromName('Test-Name!')).toBe('test_name');
  });

  it('should convert to lowercase', () => {
    expect(generateMcpSourceIdFromName('TestName')).toBe('testname');
  });

  it('should replace multiple spaces with underscores', () => {
    expect(generateMcpSourceIdFromName('My MCP Source')).toBe('my_mcp_source');
  });

  it('should handle special characters', () => {
    expect(generateMcpSourceIdFromName('Source #1 (test)')).toBe('source_1_test');
  });
});

describe('mcpSourceConfigToFormData', () => {
  it('should convert default source config to form data', () => {
    expect(mcpSourceConfigToFormData(defaultSourceConfig)).toEqual(defaultFormData);
  });

  it('should convert non-default source config to form data', () => {
    expect(mcpSourceConfigToFormData(nonDefaultSourceConfig)).toEqual(nonDefaultFormData);
  });

  it('should handle empty server lists', () => {
    const config = mockMcpCatalogSourceConfig({
      includedServers: [],
      excludedServers: [],
    });
    const result = mcpSourceConfigToFormData(config);
    expect(result.includedServers).toBe('');
    expect(result.excludedServers).toBe('');
  });

  it('should join server lists with commas', () => {
    const config = mockMcpCatalogSourceConfig({
      includedServers: ['server1', 'server2', 'server3'],
      excludedServers: ['excluded1'],
    });
    const result = mcpSourceConfigToFormData(config);
    expect(result.includedServers).toBe('server1, server2, server3');
    expect(result.excludedServers).toBe('excluded1');
  });

  it('should handle undefined yaml gracefully', () => {
    const config = mockMcpCatalogSourceConfig({ yaml: undefined });
    const result = mcpSourceConfigToFormData(config);
    expect(result.yamlContent).toBe('');
  });

  it('should handle undefined enabled (defaults to true)', () => {
    const config = mockMcpCatalogSourceConfig({ enabled: undefined });
    const result = mcpSourceConfigToFormData(config);
    expect(result.enabled).toBe(true);
  });
});

describe('transformMcpFormDataToConfig', () => {
  it('should transform form data to full config', () => {
    expect(transformMcpFormDataToConfig(nonDefaultFormData)).toEqual({
      id: 'mcp_source_2',
      name: 'MCP Source 2',
      type: McpCatalogSourceType.YAML,
      enabled: false,
      isDefault: false,
      yaml: 'source: test\nmcp_servers:\n  - name: Kubernetes',
      includedServers: ['Kubernetes', 'GitHub'],
      excludedServers: ['*preview*'],
    });
  });

  it('should transform default form data to config', () => {
    expect(transformMcpFormDataToConfig(defaultFormData)).toEqual({
      id: 'sample_mcp_source_1',
      name: 'MCP Source 1',
      type: McpCatalogSourceType.YAML,
      enabled: true,
      isDefault: true,
      yaml: undefined,
      includedServers: [],
      excludedServers: [],
    });
  });

  it('should generate id from name if no id provided', () => {
    const formData: ManageMcpSourceFormData = {
      ...nonDefaultFormData,
      id: '',
    };
    const result = transformMcpFormDataToConfig(formData);
    expect(result.id).toBe('mcp_source_2');
  });

  it('should use existing config id when available', () => {
    const formData: ManageMcpSourceFormData = {
      ...nonDefaultFormData,
      id: '',
    };
    const existingConfig = mockMcpCatalogSourceConfig({ id: 'existing_id' });
    const result = transformMcpFormDataToConfig(formData, existingConfig);
    expect(result.id).toBe('existing_id');
  });

  it('should parse comma-separated server lists', () => {
    const formData: ManageMcpSourceFormData = {
      ...nonDefaultFormData,
      includedServers: 'server1, server2,  server3  ',
      excludedServers: '*preview*,  *test*',
    };
    const result = transformMcpFormDataToConfig(formData);
    expect(result.includedServers).toEqual(['server1', 'server2', 'server3']);
    expect(result.excludedServers).toEqual(['*preview*', '*test*']);
  });

  it('should filter out empty items from server lists', () => {
    const formData: ManageMcpSourceFormData = {
      ...nonDefaultFormData,
      includedServers: 'server1, , server2, ',
      excludedServers: '',
    };
    const result = transformMcpFormDataToConfig(formData);
    expect(result.includedServers).toEqual(['server1', 'server2']);
    expect(result.excludedServers).toEqual([]);
  });
});

describe('getMcpPayloadForConfig', () => {
  it('should return full config for non-default source (create mode)', () => {
    const config = transformMcpFormDataToConfig(nonDefaultFormData);
    expect(getMcpPayloadForConfig(config, false)).toEqual({
      id: 'mcp_source_2',
      name: 'MCP Source 2',
      type: McpCatalogSourceType.YAML,
      enabled: false,
      isDefault: false,
      yaml: 'source: test\nmcp_servers:\n  - name: Kubernetes',
      includedServers: ['Kubernetes', 'GitHub'],
      excludedServers: ['*preview*'],
    });
  });

  it('should omit id for non-default source (edit mode)', () => {
    const config = transformMcpFormDataToConfig(nonDefaultFormData);
    expect(getMcpPayloadForConfig(config, true)).toEqual({
      name: 'MCP Source 2',
      type: McpCatalogSourceType.YAML,
      enabled: false,
      isDefault: false,
      yaml: 'source: test\nmcp_servers:\n  - name: Kubernetes',
      includedServers: ['Kubernetes', 'GitHub'],
      excludedServers: ['*preview*'],
    });
  });

  it('should return only allowed fields for default source', () => {
    const config = transformMcpFormDataToConfig(defaultFormData);
    expect(getMcpPayloadForConfig(config, false)).toEqual({
      enabled: true,
      includedServers: [],
      excludedServers: [],
    });
  });

  it('should include server filters for default source', () => {
    const formData: ManageMcpSourceFormData = {
      ...defaultFormData,
      includedServers: 'Kubernetes, GitHub',
      excludedServers: '*experimental*',
      enabled: false,
    };
    const config = transformMcpFormDataToConfig(formData);
    expect(getMcpPayloadForConfig(config, true)).toEqual({
      enabled: false,
      includedServers: ['Kubernetes', 'GitHub'],
      excludedServers: ['*experimental*'],
    });
  });
});
