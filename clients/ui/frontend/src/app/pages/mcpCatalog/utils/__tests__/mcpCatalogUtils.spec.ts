import {
  getMcpServerPrimaryEndpoint,
  getSecurityIndicatorLabels,
  hasMcpFiltersApplied,
  isMcpRemoteDeploymentMode,
  toDisplayServerJson,
} from '~/app/pages/mcpCatalog/utils/mcpCatalogUtils';
import type { McpCatalogFiltersState } from '~/app/pages/mcpCatalog/types/mcpCatalogFilterOptions';

describe('isMcpRemoteDeploymentMode', () => {
  it('returns true when mode is remote', () => {
    expect(isMcpRemoteDeploymentMode('remote')).toBe(true);
  });

  it('returns false when mode is local or undefined', () => {
    expect(isMcpRemoteDeploymentMode('local')).toBe(false);
    expect(isMcpRemoteDeploymentMode(undefined)).toBe(false);
  });
});

describe('getMcpServerPrimaryEndpoint', () => {
  it('returns undefined when endpoints missing or null', () => {
    expect(getMcpServerPrimaryEndpoint(undefined)).toBeUndefined();
    expect(getMcpServerPrimaryEndpoint(null)).toBeUndefined();
    expect(getMcpServerPrimaryEndpoint({})).toBeUndefined();
  });

  it('returns trimmed http when set', () => {
    expect(getMcpServerPrimaryEndpoint({ http: '  host:8080  ' })).toBe('host:8080');
  });

  it('prefers http over sse', () => {
    expect(getMcpServerPrimaryEndpoint({ http: 'https://a', sse: 'https://b' })).toBe('https://a');
  });

  it('falls back to sse when http empty', () => {
    expect(getMcpServerPrimaryEndpoint({ http: '', sse: 'https://sse' })).toBe('https://sse');
    expect(getMcpServerPrimaryEndpoint({ http: '   ', sse: 'https://sse' })).toBe('https://sse');
  });
});

describe('getSecurityIndicatorLabels', () => {
  it('returns empty array when securityIndicators is undefined or null', () => {
    expect(getSecurityIndicatorLabels(undefined)).toEqual([]);
    expect(getSecurityIndicatorLabels(null)).toEqual([]);
  });

  it('returns labels for true boolean flags', () => {
    expect(getSecurityIndicatorLabels({ verifiedSource: true, sast: true })).toEqual([
      'Verified source',
      'SAST',
    ]);
    expect(getSecurityIndicatorLabels({ secureEndpoint: true })).toEqual(['Secure endpoint']);
    expect(getSecurityIndicatorLabels({ readOnlyTools: true })).toEqual(['Read only tools']);
  });

  it('ignores false or undefined flags', () => {
    expect(
      getSecurityIndicatorLabels({
        verifiedSource: false,
        secureEndpoint: true,
        sast: undefined,
      }),
    ).toEqual(['Secure endpoint']);
  });
});

describe('hasMcpFiltersApplied', () => {
  it('returns false when filters are empty and searchQuery is empty', () => {
    expect(hasMcpFiltersApplied({}, '')).toBe(false);
    expect(hasMcpFiltersApplied({}, '   ')).toBe(false);
  });

  it('returns true when searchQuery has non-empty trimmed content', () => {
    expect(hasMcpFiltersApplied({}, 'q')).toBe(true);
    expect(hasMcpFiltersApplied({}, '  query  ')).toBe(true);
  });

  it('returns false when all filter keys have empty arrays or are missing', () => {
    const filters: McpCatalogFiltersState = {
      deploymentMode: [],
      supportedTransports: [],
      license: [],
      labels: [],
      securityIndicators: [],
    };
    expect(hasMcpFiltersApplied(filters, '')).toBe(false);
  });

  it('returns true when deploymentMode has values', () => {
    expect(hasMcpFiltersApplied({ deploymentMode: ['Local'] }, '')).toBe(true);
  });

  it('returns true when supportedTransports has values', () => {
    expect(hasMcpFiltersApplied({ supportedTransports: ['stdio'] }, '')).toBe(true);
  });

  it('returns true when license has values', () => {
    expect(hasMcpFiltersApplied({ license: ['MIT'] }, '')).toBe(true);
  });

  it('returns true when labels has values', () => {
    expect(hasMcpFiltersApplied({ labels: ['Red Hat'] }, '')).toBe(true);
  });

  it('returns true when securityIndicators has values', () => {
    expect(hasMcpFiltersApplied({ securityIndicators: ['Verified'] }, '')).toBe(true);
  });

  it('returns true when multiple filter keys have values', () => {
    const filters: McpCatalogFiltersState = {
      deploymentMode: ['Local'],
      license: ['Apache-2.0'],
    };
    expect(hasMcpFiltersApplied(filters, '')).toBe(true);
  });

  it('ignores non-array filter values', () => {
    expect(hasMcpFiltersApplied({ deploymentMode: 'Local' as unknown as string[] }, '')).toBe(
      false,
    );
  });
});

describe('toDisplayServerJson', () => {
  it('maps packages.transport into remotes and drops heavy fields', () => {
    expect(
      toDisplayServerJson({
        $schema: 'https://static.modelcontextprotocol.io/schemas/2025-12-11/server.schema.json',
        name: 'org.mariadb/mariadb-mcp',
        description: 'MariaDB MCP',
        title: 'MariaDB MCP Server',
        version: '1.0.0',
        websiteUrl: 'https://github.com/mariadb/mcp',
        packages: [
          {
            identifier: 'quay.io/example/mariadb-mcp:latest',
            environmentVariables: [{ name: 'DB_USER', isRequired: true }],
            packageArguments: [{ type: 'positional', value: 'python' }],
            registryType: 'oci',
            runtimeHint: 'docker',
            transport: {
              type: 'streamable-http',
              url: 'http://localhost:9001/mcp',
            },
          },
        ],
        repository: { source: 'github', url: 'https://github.com/mariadb/mcp' },
      }),
    ).toEqual({
      $schema: 'https://static.modelcontextprotocol.io/schemas/2025-12-11/server.schema.json',
      name: 'org.mariadb/mariadb-mcp',
      description: 'MariaDB MCP',
      title: 'MariaDB MCP Server',
      version: '1.0.0',
      websiteUrl: 'https://github.com/mariadb/mcp',
      remotes: [
        {
          type: 'streamable-http',
          url: 'http://localhost:9001/mcp',
        },
      ],
    });
  });

  it('prefers existing remotes over packages', () => {
    expect(
      toDisplayServerJson({
        name: 'dynatrace-mcp-server',
        remotes: [
          {
            type: 'streamable-http',
            url: 'https://dynatrace-mcp-server.demo-namespace.svc.cluster.local:8080',
          },
        ],
        packages: [
          {
            transport: {
              type: 'streamable-http',
              url: 'http://should-not-use',
            },
          },
        ],
      }),
    ).toEqual({
      name: 'dynatrace-mcp-server',
      remotes: [
        {
          type: 'streamable-http',
          url: 'https://dynatrace-mcp-server.demo-namespace.svc.cluster.local:8080',
        },
      ],
    });
  });

  it('omits remotes when neither remotes nor package transports are present', () => {
    expect(
      toDisplayServerJson({
        name: 'example',
        packages: [{ identifier: 'quay.io/example' }],
        repository: { source: 'github', url: 'https://example.com' },
      }),
    ).toEqual({
      name: 'example',
    });
  });
});
