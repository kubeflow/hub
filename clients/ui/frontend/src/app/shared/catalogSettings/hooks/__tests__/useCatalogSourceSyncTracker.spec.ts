import '@testing-library/jest-dom';
import { renderHook, act } from '@testing-library/react';
import {
  useCatalogSourceSyncTracker,
  SYNC_PENDING_TIMEOUT,
} from '~/app/shared/catalogSettings/hooks/useCatalogSourceSyncTracker';
import { CatalogSourceStatus } from '~/app/shared/types/catalogTypes';
import type { CatalogSource, CatalogSourceList } from '~/app/shared/types/catalogTypes';

const sourceList = (...items: CatalogSource[]): CatalogSourceList => ({
  items,
  size: items.length,
  pageSize: 10,
  nextPageToken: '',
});

const availableSource = (id: string, overrides: Partial<CatalogSource> = {}): CatalogSource => ({
  id,
  name: id,
  labels: [],
  enabled: true,
  status: CatalogSourceStatus.AVAILABLE,
  ...overrides,
});

describe('useCatalogSourceSyncTracker', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('reports nothing pending before any write', () => {
    const { result } = renderHook(() => useCatalogSourceSyncTracker());

    expect(result.current.hasPendingSyncs).toBe(false);
    expect(result.current.isSyncPending('skills')).toBe(false);
  });

  it('marks a source pending and raises hasPendingSyncs', () => {
    const { result } = renderHook(() => useCatalogSourceSyncTracker());

    act(() => {
      result.current.reconcileSyncPending(sourceList(availableSource('skills')));
    });
    act(() => {
      result.current.markSyncPending('skills');
    });

    expect(result.current.isSyncPending('skills')).toBe(true);
    expect(result.current.hasPendingSyncs).toBe(true);
  });

  it('leaves a source pending while the catalog keeps reporting the same thing', () => {
    const { result } = renderHook(() => useCatalogSourceSyncTracker());

    act(() => {
      result.current.reconcileSyncPending(sourceList(availableSource('skills')));
    });
    act(() => {
      result.current.markSyncPending('skills');
    });
    // An include/exclude filter edit is invisible in the source list: the catalog
    // reports "available" throughout, so the pending mark must survive the poll.
    act(() => {
      result.current.reconcileSyncPending(sourceList(availableSource('skills')));
    });

    expect(result.current.isSyncPending('skills')).toBe(true);
  });

  it('clears the pending mark when the reported status changes', () => {
    const { result } = renderHook(() => useCatalogSourceSyncTracker());

    act(() => {
      result.current.reconcileSyncPending(sourceList(availableSource('skills')));
    });
    act(() => {
      result.current.markSyncPending('skills');
    });
    act(() => {
      result.current.reconcileSyncPending(
        sourceList(
          availableSource('skills', { status: CatalogSourceStatus.ERROR, error: 'clone failed' }),
        ),
      );
    });

    expect(result.current.isSyncPending('skills')).toBe(false);
    expect(result.current.hasPendingSyncs).toBe(false);
  });

  it('clears the pending mark when the reported labels change', () => {
    const { result } = renderHook(() => useCatalogSourceSyncTracker());

    act(() => {
      result.current.reconcileSyncPending(sourceList(availableSource('skills')));
    });
    act(() => {
      result.current.markSyncPending('skills');
    });
    act(() => {
      result.current.reconcileSyncPending(
        sourceList(availableSource('skills', { labels: ['community'] })),
      );
    });

    expect(result.current.isSyncPending('skills')).toBe(false);
  });

  it('clears the pending mark for a newly created source once the catalog lists it', () => {
    const { result } = renderHook(() => useCatalogSourceSyncTracker());

    act(() => {
      result.current.reconcileSyncPending(sourceList());
    });
    act(() => {
      result.current.markSyncPending('new-skills');
    });
    expect(result.current.isSyncPending('new-skills')).toBe(true);

    act(() => {
      result.current.reconcileSyncPending(sourceList(availableSource('new-skills')));
    });

    expect(result.current.isSyncPending('new-skills')).toBe(false);
  });

  it('expires the pending mark after the timeout when the catalog never signals', () => {
    const { result } = renderHook(() => useCatalogSourceSyncTracker());

    act(() => {
      result.current.reconcileSyncPending(sourceList(availableSource('skills')));
    });
    act(() => {
      result.current.markSyncPending('skills');
    });

    act(() => {
      jest.advanceTimersByTime(SYNC_PENDING_TIMEOUT - 1000);
    });
    expect(result.current.isSyncPending('skills')).toBe(true);

    act(() => {
      jest.advanceTimersByTime(2000);
    });
    expect(result.current.isSyncPending('skills')).toBe(false);
    expect(result.current.hasPendingSyncs).toBe(false);
  });

  it('tracks several sources independently', () => {
    const { result } = renderHook(() => useCatalogSourceSyncTracker());

    act(() => {
      result.current.reconcileSyncPending(sourceList(availableSource('a'), availableSource('b')));
    });
    act(() => {
      result.current.markSyncPending('a');
      result.current.markSyncPending('b');
    });
    act(() => {
      result.current.reconcileSyncPending(
        sourceList(availableSource('a', { labels: ['x'] }), availableSource('b')),
      );
    });

    expect(result.current.isSyncPending('a')).toBe(false);
    expect(result.current.isSyncPending('b')).toBe(true);
    expect(result.current.hasPendingSyncs).toBe(true);
  });
});
