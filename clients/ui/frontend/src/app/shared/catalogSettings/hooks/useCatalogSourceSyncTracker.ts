import * as React from 'react';
import type { CatalogSource, CatalogSourceList } from '~/app/shared/types/catalogTypes';

/**
 * Poll rate for the catalog source list while at least one source is waiting for
 * the catalog to pick up an edit. Only in effect while something is pending; the
 * list falls back to POLL_INTERVAL once everything has settled.
 */
export const SYNC_PENDING_POLL_INTERVAL = 5000;

/**
 * How long a source keeps reporting "syncing" when the catalog never tells us
 * anything changed.
 *
 * Saving a source writes a ConfigMap; the catalog then re-reads it and re-indexes
 * its repositories in the background, which can take minutes. `CatalogSource`
 * carries no sync timestamp — only id, name, enabled, labels, status and error —
 * so a status flip or a label edit is observable from here, but an
 * include/exclude filter edit is not: the source reports "available" throughout
 * and the new content simply appears. For those the only honest end to the
 * pending state is a timeout.
 */
export const SYNC_PENDING_TIMEOUT = 3 * 60 * 1000;

type PendingSync = { since: number; fingerprint: string };

/**
 * Everything the catalog tells us about a source. Any difference means the
 * catalog has re-read the source since it was marked pending.
 */
const fingerprintSource = (source: CatalogSource | undefined): string =>
  source
    ? JSON.stringify([
        source.status ?? '',
        source.error ?? '',
        source.enabled ?? true,
        source.labels,
      ])
    : '';

export type CatalogSourceSyncTracker = {
  /** True while `sourceId` has been edited and the catalog has not confirmed a reload. */
  isSyncPending: (sourceId: string) => boolean;
  /** Call after a successful create/update/toggle to start showing "syncing". */
  markSyncPending: (sourceId: string) => void;
  /** Feed each polled source list back in so pending marks can be cleared. */
  reconcileSyncPending: (catalogSources: CatalogSourceList | null) => void;
  /** True while any source is pending; used to raise the poll rate. */
  hasPendingSyncs: boolean;
};

export const useCatalogSourceSyncTracker = (): CatalogSourceSyncTracker => {
  const [pending, setPending] = React.useState<Record<string, PendingSync>>({});
  const latestSourcesRef = React.useRef<CatalogSourceList | null>(null);

  const markSyncPending = React.useCallback((sourceId: string) => {
    const source = latestSourcesRef.current?.items?.find((s) => s.id === sourceId);
    setPending((prev) => ({
      ...prev,
      [sourceId]: { since: Date.now(), fingerprint: fingerprintSource(source) },
    }));
  }, []);

  const reconcileSyncPending = React.useCallback((catalogSources: CatalogSourceList | null) => {
    latestSourcesRef.current = catalogSources;
    setPending((prev) => {
      const entries = Object.entries(prev);
      if (entries.length === 0) {
        return prev;
      }
      const kept = entries.filter(([sourceId, entry]) => {
        const source = catalogSources?.items?.find((s) => s.id === sourceId);
        return fingerprintSource(source) === entry.fingerprint;
      });
      // Returning `prev` unchanged when nothing resolved keeps this out of a
      // render loop with the effect that reconciles on every poll.
      return kept.length === entries.length ? prev : Object.fromEntries(kept);
    });
  }, []);

  // Expire pending marks the catalog never gave us a signal for.
  React.useEffect(() => {
    const entries = Object.values(pending);
    if (entries.length === 0) {
      return undefined;
    }
    const earliest = Math.min(...entries.map((entry) => entry.since));
    const timer = window.setTimeout(
      () => {
        const cutoff = Date.now() - SYNC_PENDING_TIMEOUT;
        setPending((prev) => {
          const kept = Object.entries(prev).filter(([, entry]) => entry.since > cutoff);
          return kept.length === Object.keys(prev).length ? prev : Object.fromEntries(kept);
        });
      },
      Math.max(0, earliest + SYNC_PENDING_TIMEOUT - Date.now()),
    );
    return () => window.clearTimeout(timer);
  }, [pending]);

  const hasPendingSyncs = Object.keys(pending).length > 0;

  return React.useMemo(
    () => ({
      isSyncPending: (sourceId: string) => sourceId in pending,
      markSyncPending,
      reconcileSyncPending,
      hasPendingSyncs,
    }),
    [pending, markSyncPending, reconcileSyncPending, hasPendingSyncs],
  );
};
