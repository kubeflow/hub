import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import {
  McpCatalogSettingsContext,
  McpCatalogSettingsContextType,
} from '~/app/context/mcpCatalogSettings/McpCatalogSettingsContext';
import { McpCatalogSourceConfig, McpCatalogSourceType } from '~/app/mcpServerCatalogTypes';
import McpCatalogSourceStatus from '~/app/pages/mcpCatalogSettings/components/McpCatalogSourceStatus';
import { CatalogSourceStatus as CatalogSourceStatusEnum } from '~/app/shared/types/catalogTypes';

const mockConfig: McpCatalogSourceConfig = {
  id: 'test-source',
  name: 'Test Source',
  type: McpCatalogSourceType.YAML,
  enabled: true,
};

const defaultPagination = { size: 0, pageSize: 10, nextPageToken: '' };

const renderWithContext = (
  config: McpCatalogSourceConfig,
  contextOverrides: Partial<McpCatalogSettingsContextType>,
) => {
  const defaultContext: McpCatalogSettingsContextType = {
    apiState: {
      apiAvailable: false,
      api: null as unknown as McpCatalogSettingsContextType['apiState']['api'],
    },
    refreshAPIState: jest.fn(),
    mcpCatalogSourceConfigs: null,
    mcpCatalogSourceConfigsLoaded: false,
    mcpCatalogSourceConfigsLoadError: undefined,
    refreshMcpCatalogSourceConfigs: jest.fn(),
    mcpCatalogSources: null,
    mcpCatalogSourcesLoaded: true,
    mcpCatalogSourcesLoadError: undefined,
    refreshMcpCatalogSources: jest.fn(),
    pendingSourceIds: new Map(),
    markSourcePending: jest.fn(),
    ...contextOverrides,
  };

  return render(
    <McpCatalogSettingsContext.Provider value={defaultContext}>
      <McpCatalogSourceStatus mcpCatalogSourceConfig={config} />
    </McpCatalogSettingsContext.Provider>,
  );
};

describe('McpCatalogSourceStatus', () => {
  it('renders "Ready" label with outline variant', () => {
    renderWithContext(mockConfig, {
      mcpCatalogSources: {
        ...defaultPagination,
        items: [
          {
            id: 'test-source',
            name: 'Test',
            labels: [],
            status: CatalogSourceStatusEnum.AVAILABLE,
          },
        ],
      },
      mcpCatalogSourcesLoaded: true,
    });

    const label = screen.getByTestId('mcp-source-status-connected-test-source');
    expect(screen.getByText('Ready')).toBeVisible();
    expect(label.className).toMatch(/outline/);
    expect(label.className).not.toMatch(/filled/);
  });

  it('renders "Failed" label with outline variant', () => {
    renderWithContext(mockConfig, {
      mcpCatalogSources: {
        ...defaultPagination,
        items: [
          {
            id: 'test-source',
            name: 'Test',
            labels: [],
            status: CatalogSourceStatusEnum.ERROR,
            error: 'Connection refused',
          },
        ],
      },
      mcpCatalogSourcesLoaded: true,
    });

    const label = screen.getByTestId('mcp-source-status-failed-test-source');
    expect(screen.getByText('Failed')).toBeVisible();
    expect(label.className).toMatch(/outline/);
    expect(label.className).not.toMatch(/filled/);
  });

  it('renders "Starting" label with outline variant when source has no status', () => {
    renderWithContext(mockConfig, {
      mcpCatalogSources: {
        ...defaultPagination,
        items: [{ id: 'test-source', name: 'Test', labels: [] }],
      },
      mcpCatalogSourcesLoaded: true,
    });

    const label = screen.getByTestId('mcp-source-status-starting-test-source');
    expect(screen.getByText('Starting')).toBeVisible();
    expect(label.className).toMatch(/outline/);
  });

  it('renders "Starting" label when source is pending after mutation', () => {
    renderWithContext(mockConfig, {
      mcpCatalogSources: {
        ...defaultPagination,
        items: [
          {
            id: 'test-source',
            name: 'Test',
            labels: [],
            status: CatalogSourceStatusEnum.AVAILABLE,
          },
        ],
      },
      mcpCatalogSourcesLoaded: true,
      pendingSourceIds: new Map([['test-source', CatalogSourceStatusEnum.AVAILABLE]]),
    });

    expect(screen.getByTestId('mcp-source-status-starting-test-source')).toBeInTheDocument();
    expect(screen.getByText('Starting')).toBeVisible();
  });

  it('renders "Unknown" label with outline variant when there is a load error', () => {
    renderWithContext(mockConfig, {
      mcpCatalogSources: null,
      mcpCatalogSourcesLoaded: true,
      mcpCatalogSourcesLoadError: new Error('API error'),
    });

    const label = screen.getByTestId('mcp-source-status-unknown-test-source');
    expect(screen.getByText('Unknown')).toBeVisible();
    expect(label.className).toMatch(/outline/);
  });

  it('renders "-" for default sources', () => {
    renderWithContext({ ...mockConfig, isDefault: true }, { mcpCatalogSourcesLoaded: true });
    expect(screen.getByText('-')).toBeVisible();
  });

  it('renders "-" for disabled sources', () => {
    renderWithContext({ ...mockConfig, enabled: false }, { mcpCatalogSourcesLoaded: true });
    expect(screen.getByText('-')).toBeVisible();
  });
});
