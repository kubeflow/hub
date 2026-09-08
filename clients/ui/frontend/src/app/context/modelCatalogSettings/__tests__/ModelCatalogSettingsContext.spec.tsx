import '@testing-library/jest-dom';
import * as React from 'react';
import { renderHook, act } from '@testing-library/react';
import {
  ModelCatalogSettingsContextProvider,
  ModelCatalogSettingsContext,
} from '~/app/context/modelCatalogSettings/ModelCatalogSettingsContext';
import type { CatalogSourceList } from '~/app/shared/types/catalogTypes';
import { CatalogSourceStatus } from '~/app/shared/types/catalogTypes';

let mockCatalogSources: CatalogSourceList;
const mockRefreshCatalogSources = jest.fn();

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

jest.mock('~/app/hooks/modelCatalogSettings/useModelCatalogSettingsAPIState', () => ({
  __esModule: true,
  default: jest.fn(() => [
    {
      apiAvailable: true,
      api: {
        getCatalogSourceConfigs: jest.fn(),
        createCatalogSourceConfig: jest.fn(),
        getCatalogSourceConfig: jest.fn(),
        updateCatalogSourceConfig: jest.fn(),
        deleteCatalogSourceConfig: jest.fn(),
        deleteCatalogSourceCredentials: jest.fn(),
        previewCatalogSource: jest.fn(),
      },
    },
    jest.fn(),
  ]),
}));

jest.mock('~/app/hooks/modelCatalogSettings/useCatalogSourceConfigs', () => ({
  useCatalogSourceConfigs: jest.fn(() => [{ catalogs: [] }, true, undefined, jest.fn()]),
}));

jest.mock('~/app/shared/catalogSettings/hooks/useCatalogSourcesWithPolling', () => ({
  useCatalogSourcesWithPolling: jest.fn(
    () => [mockCatalogSources, true, undefined, mockRefreshCatalogSources] as const,
  ),
}));

const emptySources: CatalogSourceList = {
  items: [],
  size: 0,
  pageSize: 0,
  nextPageToken: '',
};

describe('ModelCatalogSettingsContext — pendingSourceIds clearing', () => {
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <ModelCatalogSettingsContextProvider>{children}</ModelCatalogSettingsContextProvider>
  );

  beforeEach(() => {
    mockCatalogSources = emptySources;
    jest.clearAllMocks();
  });

  it('should skip three stale responses and clear on fourth', () => {
    mockCatalogSources = {
      ...emptySources,
      items: [{ id: 'src-1', name: 'S1', labels: [], status: CatalogSourceStatus.AVAILABLE }],
    };

    const { result, rerender } = renderHook(() => React.useContext(ModelCatalogSettingsContext), {
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
        mockRefreshCatalogSources,
      ]);
      rerender();
      expect(result.current.pendingSourceIds.has('src-1')).toBe(true);
    }

    // 4th response — accepted, pending clears
    (polling.useCatalogSourcesWithPolling as jest.Mock).mockReturnValue([
      { ...errorSources },
      true,
      undefined,
      mockRefreshCatalogSources,
    ]);
    rerender();
    expect(result.current.pendingSourceIds.has('src-1')).toBe(false);
  });

  it('should skip three stale responses and clear on fourth (same status)', () => {
    mockCatalogSources = {
      ...emptySources,
      items: [{ id: 'src-1', name: 'S1', labels: [], status: CatalogSourceStatus.AVAILABLE }],
    };

    const { result, rerender } = renderHook(() => React.useContext(ModelCatalogSettingsContext), {
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
        mockRefreshCatalogSources,
      ]);
      rerender();
      expect(result.current.pendingSourceIds.has('src-1')).toBe(true);
    }

    // 4th response — accepted, pending clears even with same status
    (polling.useCatalogSourcesWithPolling as jest.Mock).mockReturnValue([
      { ...sameSources },
      true,
      undefined,
      mockRefreshCatalogSources,
    ]);
    rerender();
    expect(result.current.pendingSourceIds.has('src-1')).toBe(false);
  });
});
