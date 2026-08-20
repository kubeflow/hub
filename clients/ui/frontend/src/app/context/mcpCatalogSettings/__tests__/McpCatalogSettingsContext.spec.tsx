import '@testing-library/jest-dom';
import * as React from 'react';
import { renderHook, act } from '@testing-library/react';
import {
  McpCatalogSettingsContextProvider,
  McpCatalogSettingsContext,
} from '~/app/context/mcpCatalogSettings/McpCatalogSettingsContext';
import type { CatalogSourceList } from '~/app/shared/types/catalogTypes';
import { CatalogSourceStatus } from '~/app/shared/types/catalogTypes';

let mockMcpCatalogSources: CatalogSourceList;
const mockRefreshMcpCatalogSources = jest.fn();

jest.mock('mod-arch-core', () => {
  const actual = jest.requireActual('mod-arch-core');
  return {
    ...actual,
    useQueryParamNamespaces: jest.fn(() => ({})),
  };
});

jest.mock('~/app/hooks/modelCatalog/useModelCatalogAPIState', () => ({
  __esModule: true,
  default: jest.fn(() => [
    {
      apiAvailable: true,
      api: { getListSources: jest.fn() },
    },
    jest.fn(),
  ]),
}));

jest.mock('~/app/hooks/mcpCatalogSettings/useMcpCatalogSettingsAPIState', () => ({
  __esModule: true,
  default: jest.fn(() => [
    {
      apiAvailable: true,
      api: {
        getMcpCatalogSourceConfigs: jest.fn(),
        createMcpCatalogSourceConfig: jest.fn(),
        getMcpCatalogSourceConfig: jest.fn(),
        updateMcpCatalogSourceConfig: jest.fn(),
        deleteMcpCatalogSourceConfig: jest.fn(),
        previewMcpCatalogSource: jest.fn(),
      },
    },
    jest.fn(),
  ]),
}));

jest.mock('~/app/hooks/mcpCatalogSettings/useMcpCatalogSourceConfigs', () => ({
  useMcpCatalogSourceConfigs: jest.fn(() => [{ catalogs: [] }, true, undefined, jest.fn()]),
}));

jest.mock('~/app/shared/catalogSettings/hooks/useCatalogSourcesWithPolling', () => ({
  useCatalogSourcesWithPolling: jest.fn(
    () => [mockMcpCatalogSources, true, undefined, mockRefreshMcpCatalogSources] as const,
  ),
}));

const emptySources: CatalogSourceList = {
  items: [],
  size: 0,
  pageSize: 0,
  nextPageToken: '',
};

describe('McpCatalogSettingsContext — pendingSourceIds clearing', () => {
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <McpCatalogSettingsContextProvider>{children}</McpCatalogSettingsContextProvider>
  );

  beforeEach(() => {
    mockMcpCatalogSources = emptySources;
    jest.clearAllMocks();
  });

  it('should skip three stale responses and clear on fourth', () => {
    mockMcpCatalogSources = {
      ...emptySources,
      items: [{ id: 'src-1', name: 'S1', labels: [], status: CatalogSourceStatus.AVAILABLE }],
    };

    const { result, rerender } = renderHook(() => React.useContext(McpCatalogSettingsContext), {
      wrapper,
    });

    act(() => {
      result.current.markSourcePending('src-1', CatalogSourceStatus.AVAILABLE);
    });
    expect(result.current.pendingSourceIds.has('src-1')).toBe(true);

    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const polling = require('~/app/shared/catalogSettings/hooks/useCatalogSourcesWithPolling');
    const errorSources = {
      ...emptySources,
      items: [{ id: 'src-1', name: 'S1', labels: [], status: CatalogSourceStatus.ERROR }],
    };

    // 1st, 2nd, 3rd responses (stale) — all skipped
    for (let i = 0; i < 3; i++) {
      (polling.useCatalogSourcesWithPolling as jest.Mock).mockReturnValue([
        { ...errorSources },
        true,
        undefined,
        mockRefreshMcpCatalogSources,
      ]);
      rerender();
      expect(result.current.pendingSourceIds.has('src-1')).toBe(true);
    }

    // 4th response — accepted, pending clears
    (polling.useCatalogSourcesWithPolling as jest.Mock).mockReturnValue([
      { ...errorSources },
      true,
      undefined,
      mockRefreshMcpCatalogSources,
    ]);
    rerender();
    expect(result.current.pendingSourceIds.has('src-1')).toBe(false);
  });

  it('should skip three stale responses and clear on fourth (same status)', () => {
    mockMcpCatalogSources = {
      ...emptySources,
      items: [{ id: 'src-1', name: 'S1', labels: [], status: CatalogSourceStatus.AVAILABLE }],
    };

    const { result, rerender } = renderHook(() => React.useContext(McpCatalogSettingsContext), {
      wrapper,
    });

    act(() => {
      result.current.markSourcePending('src-1', CatalogSourceStatus.AVAILABLE);
    });
    expect(result.current.pendingSourceIds.has('src-1')).toBe(true);

    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const polling = require('~/app/shared/catalogSettings/hooks/useCatalogSourcesWithPolling');
    const sameSources = {
      ...emptySources,
      items: [{ id: 'src-1', name: 'S1', labels: [], status: CatalogSourceStatus.AVAILABLE }],
    };

    // 1st, 2nd, 3rd responses — all skipped
    for (let i = 0; i < 3; i++) {
      (polling.useCatalogSourcesWithPolling as jest.Mock).mockReturnValue([
        { ...sameSources },
        true,
        undefined,
        mockRefreshMcpCatalogSources,
      ]);
      rerender();
      expect(result.current.pendingSourceIds.has('src-1')).toBe(true);
    }

    // 4th response — accepted, pending clears even with same status
    (polling.useCatalogSourcesWithPolling as jest.Mock).mockReturnValue([
      { ...sameSources },
      true,
      undefined,
      mockRefreshMcpCatalogSources,
    ]);
    rerender();
    expect(result.current.pendingSourceIds.has('src-1')).toBe(false);
  });
});
