import '@testing-library/jest-dom';
import * as React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { McpCatalogContext } from '~/app/context/mcpCatalog/McpCatalogContext';
import type { McpCatalogContextType } from '~/app/context/mcpCatalog/types';
import type { McpServer } from '~/app/mcpServerCatalogTypes';
import type { McpServersResult } from '~/app/hooks/mcpServerCatalog/useMcpServersBySourceLabel';
import { useMcpServersBySourceLabelWithAPI } from '~/app/hooks/mcpServerCatalog/useMcpServersBySourceLabel';
import McpCatalogGalleryView from '~/app/pages/mcpCatalog/screens/McpCatalogGalleryView';

jest.mock('~/app/hooks/mcpServerCatalog/useMcpServersBySourceLabel', () => ({
  useMcpServersBySourceLabelWithAPI: jest.fn(),
}));

const mockUseMcpServersBySourceLabelWithAPI = jest.mocked(useMcpServersBySourceLabelWithAPI);

const buildServer = (id: string, sourceId: string): McpServer => ({
  id,
  name: `Server ${id}`,
  description: `Description ${id}`,
  deploymentMode: 'local',
  securityIndicators: {},
  source_id: sourceId, // eslint-disable-line camelcase
  toolCount: 0,
});

const defaultContext: McpCatalogContextType = {
  filters: {},
  setFilters: jest.fn(),
  searchQuery: '',
  setSearchQuery: jest.fn(),
  namedQuery: null,
  setNamedQuery: jest.fn(),
  pagination: { page: 1, pageSize: 10, totalItems: 0 },
  setPage: jest.fn(),
  setPageSize: jest.fn(),
  setTotalItems: jest.fn(),
  selectedSourceLabel: undefined,
  setSelectedSourceLabel: jest.fn(),
  clearAllFilters: jest.fn(),
  mcpApiState: { api: {} as McpCatalogContextType['mcpApiState']['api'], apiAvailable: false },
  catalogSources: null,
  catalogSourcesLoaded: true,
  catalogSourcesLoadError: undefined,
  catalogLabels: null,
  catalogLabelsLoaded: true,
  catalogLabelsLoadError: undefined,
  filterOptions: null,
  filterOptionsLoaded: true,
  filterOptionsLoadError: undefined,
};

const defaultHookResult: McpServersResult = {
  mcpServers: {
    items: [],
    size: 0,
    pageSize: 10,
    nextPageToken: '',
    loadMore: jest.fn(),
    isLoadingMore: false,
    hasMore: false,
    refresh: jest.fn(),
  },
  mcpServersLoaded: true,
  mcpServersLoadError: undefined,
  refresh: jest.fn(),
};

const handleFilterReset = jest.fn();

const renderWithContext = (
  contextOverrides: Partial<McpCatalogContextType> = {},
  hookOverrides: Partial<McpServersResult> = {},
) => {
  const hookResult = {
    ...defaultHookResult,
    ...hookOverrides,
    mcpServers: { ...defaultHookResult.mcpServers, ...hookOverrides.mcpServers },
  };
  mockUseMcpServersBySourceLabelWithAPI.mockReturnValue(hookResult);

  const ctx = { ...defaultContext, ...contextOverrides };
  return render(
    <MemoryRouter>
      <McpCatalogContext.Provider value={ctx}>
        <McpCatalogGalleryView handleFilterReset={handleFilterReset} />
      </McpCatalogContext.Provider>
    </MemoryRouter>,
  );
};

describe('McpCatalogGalleryView', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    Element.prototype.scrollTo = jest.fn();
  });

  it('renders loading spinner when not loaded', () => {
    renderWithContext({}, { mcpServersLoaded: false });
    expect(screen.getByText('Loading MCP servers...')).toBeInTheDocument();
  });

  it('renders error alert', () => {
    renderWithContext({}, { mcpServersLoadError: new Error('Network failure') });
    expect(screen.getByText('Failed to load MCP servers')).toBeInTheDocument();
    expect(screen.getByText('Network failure')).toBeInTheDocument();
  });

  it('renders empty state with Reset filters action', () => {
    renderWithContext({}, { mcpServers: { ...defaultHookResult.mcpServers, items: [] } });
    expect(screen.getByTestId('empty-mcp-catalog-state')).toBeInTheDocument();
    expect(screen.getByText('No results found')).toBeInTheDocument();
    fireEvent.click(screen.getByText('Reset filters'));
    expect(handleFilterReset).toHaveBeenCalledTimes(1);
  });

  it('renders server cards', () => {
    const servers = Array.from({ length: 5 }, (_, i) => buildServer(String(i + 1), 'cat-a'));
    renderWithContext({}, { mcpServers: { ...defaultHookResult.mcpServers, items: servers } });
    expect(screen.getAllByTestId(/^mcp-catalog-card-\d+$/)).toHaveLength(5);
  });

  it('renders Load more button when hasMore is true', () => {
    const servers = Array.from({ length: 10 }, (_, i) => buildServer(String(i + 1), 'cat-a'));
    const loadMoreFn = jest.fn();
    renderWithContext(
      {},
      {
        mcpServers: {
          ...defaultHookResult.mcpServers,
          items: servers,
          hasMore: true,
          loadMore: loadMoreFn,
        },
      },
    );
    expect(screen.getByText('Load more servers')).toBeInTheDocument();
    fireEvent.click(screen.getByText('Load more servers'));
    expect(loadMoreFn).toHaveBeenCalledTimes(1);
  });

  it('shows loading spinner when isLoadingMore is true', () => {
    const servers = Array.from({ length: 10 }, (_, i) => buildServer(String(i + 1), 'cat-a'));
    renderWithContext(
      {},
      {
        mcpServers: {
          ...defaultHookResult.mcpServers,
          items: servers,
          hasMore: true,
          isLoadingMore: true,
        },
      },
    );
    expect(screen.getByText('Loading more MCP servers...')).toBeInTheDocument();
    expect(screen.queryByText('Load more servers')).not.toBeInTheDocument();
  });

  it('does not show Load more when hasMore is false', () => {
    const servers = Array.from({ length: 5 }, (_, i) => buildServer(String(i + 1), 'cat-a'));
    renderWithContext(
      {},
      { mcpServers: { ...defaultHookResult.mcpServers, items: servers, hasMore: false } },
    );
    expect(screen.queryByText('Load more servers')).not.toBeInTheDocument();
  });

  it('does not show Load more when hasMore is true but filtered items are fewer than page size', () => {
    const servers = Array.from({ length: 3 }, (_, i) => buildServer(String(i + 1), 'cat-a'));
    renderWithContext(
      {},
      { mcpServers: { ...defaultHookResult.mcpServers, items: servers, hasMore: true } },
    );
    expect(screen.queryByText('Load more servers')).not.toBeInTheDocument();
  });
});
