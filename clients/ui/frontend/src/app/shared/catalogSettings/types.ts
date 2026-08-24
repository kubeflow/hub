import * as React from 'react';
import type { APIOptions, FetchState } from 'mod-arch-core';
import type { CatalogSourceList } from '~/app/shared/types/catalogTypes';

/** Minimal API state for catalog source list polling (model & MCP settings share the browse API). */
export type CatalogSourcesPollingAPIState = {
  apiAvailable: boolean;
  api: {
    getListSources: (opts: APIOptions) => Promise<CatalogSourceList>;
  };
};

/** Generic context value produced by `createCatalogSettingsContext`. */
export type CatalogSettingsContextValue<TAPIState, TConfigList> = {
  apiState: TAPIState;
  refreshAPIState: () => void;
  sourceConfigs: TConfigList | null;
  sourceConfigsLoaded: boolean;
  sourceConfigsLoadError?: Error;
  refreshSourceConfigs: () => void;
  catalogSources: CatalogSourceList | null;
  catalogSourcesLoaded: boolean;
  catalogSourcesLoadError?: Error;
  refreshCatalogSources: () => void;
  /** True while a source has been edited and the catalog has not confirmed the reload. */
  isSyncPending: (sourceId: string) => boolean;
  /** Mark a source as awaiting a catalog reload after a successful write. */
  markSyncPending: (sourceId: string) => void;
};

/**
 * Definition passed to `createCatalogSettingsContext`.
 * Captures host paths, extra query params (e.g. MCP assetType), and the
 * domain-specific hooks so the factory is fully generic.
 */
export type CatalogSettingsContextDefinition<TAPIState, TConfigList> = {
  /** Full settings CRUD host path (e.g. `.../settings/model_catalog`). */
  settingsHostPath: string;
  /** Full catalog listing / polling host path (e.g. `.../model_catalog`). */
  catalogHostPath: string;
  /** Extra query params merged into the catalog polling request (MCP adds `assetType`). */
  catalogExtraQueryParams?: Record<string, unknown>;
  /** Override preview host — MCP quirk: preview must call the model_catalog settings host. */
  previewHostPath?: string;
  useSettingsAPIState: (
    hostPath: string | null,
    queryParams?: Record<string, unknown>,
    previewHostPath?: string | null,
  ) => [TAPIState, () => void];
  useSourceConfigsList: (apiState: TAPIState) => FetchState<TConfigList>;
};

/**
 * Minimal shape consumed by `CatalogSettingsRoutes`.
 * Each catalog's `definition.ts` satisfies this type.
 */
export type CatalogSettingsDefinition = {
  id: string;
  ContextProvider: React.ComponentType<{ children: React.ReactNode }>;
  ListPage: React.ComponentType;
  ManagePage: React.ComponentType;
};
