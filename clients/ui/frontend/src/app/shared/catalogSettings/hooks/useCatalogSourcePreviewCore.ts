import * as React from 'react';
import { isEqual } from 'lodash-es';
import { APIOptions } from 'mod-arch-core';
import { PreviewCatalogSourceQueryParams } from '~/app/modelCatalogTypes';
import {
  CatalogSettingsPreviewResult,
  CatalogSettingsPreviewTab,
  CatalogSettingsPreviewTabState,
  createInitialPreviewTabState,
  DEFAULT_PREVIEW_PAGE_SIZE,
  getTargetPreviewTab,
} from './previewTypes';

export type CatalogSettingsPreviewCoreState<TItem, TSummary, TRequest> = {
  isLoadingInitial: boolean;
  isLoadingMore: boolean;
  summary?: TSummary;
  tabStates: Record<CatalogSettingsPreviewTab, CatalogSettingsPreviewTabState<TItem>>;
  error?: Error;
  lastPreviewedData?: TRequest;
  activeTab: CatalogSettingsPreviewTab;
};

export type UseCatalogSourcePreviewCoreOptions<TItem, TSummary, TRequest> = {
  canPreview: boolean;
  isEditMode: boolean;
  apiAvailable: boolean;
  buildPreviewRequest: () => TRequest;
  previewApi: (
    opts: APIOptions,
    data: TRequest,
    queryParams?: PreviewCatalogSourceQueryParams,
  ) => Promise<CatalogSettingsPreviewResult<TItem, TSummary>>;
};

export type UseCatalogSourcePreviewCoreResult<TItem, TSummary, TRequest> = {
  previewState: CatalogSettingsPreviewCoreState<TItem, TSummary, TRequest>;
  handlePreviewInternal: (options?: {
    loadMore?: boolean;
    switchToTab?: CatalogSettingsPreviewTab;
  }) => Promise<void>;
  handleTabChange: (tab: CatalogSettingsPreviewTab) => void;
  handleLoadMore: () => void;
  hasFormChanged: boolean;
};

const createInitialPreviewState = <
  TItem,
  TSummary,
  TRequest,
>(): CatalogSettingsPreviewCoreState<TItem, TSummary, TRequest> => ({
  isLoadingInitial: false,
  isLoadingMore: false,
  tabStates: {
    [CatalogSettingsPreviewTab.INCLUDED]: createInitialPreviewTabState<TItem>(),
    [CatalogSettingsPreviewTab.EXCLUDED]: createInitialPreviewTabState<TItem>(),
  },
  activeTab: CatalogSettingsPreviewTab.INCLUDED,
});

export const useCatalogSourcePreviewCore = <TItem, TSummary, TRequest>({
  canPreview,
  isEditMode,
  apiAvailable,
  buildPreviewRequest,
  previewApi,
}: UseCatalogSourcePreviewCoreOptions<TItem, TSummary, TRequest>): UseCatalogSourcePreviewCoreResult<
  TItem,
  TSummary,
  TRequest
> => {
  const [previewState, setPreviewState] = React.useState(createInitialPreviewState<TItem, TSummary, TRequest>);

  const previewStateRef = React.useRef(previewState);
  previewStateRef.current = previewState;

  const hasFormChanged = React.useMemo(() => {
    if (!previewState.lastPreviewedData) {
      return false;
    }
    return !isEqual(buildPreviewRequest(), previewState.lastPreviewedData);
  }, [buildPreviewRequest, previewState.lastPreviewedData]);

  const handlePreviewInternal = React.useCallback(
    async (options?: { loadMore?: boolean; switchToTab?: CatalogSettingsPreviewTab }) => {
      const { loadMore = false, switchToTab } = options ?? {};
      const isFreshPreview = !loadMore && !switchToTab;
      const currentState = previewStateRef.current;
      const targetTab = getTargetPreviewTab(
        isFreshPreview,
        switchToTab,
        currentState.activeTab,
        CatalogSettingsPreviewTab.INCLUDED,
      );

      if (!apiAvailable) {
        setPreviewState((prev) => ({
          ...prev,
          isLoadingInitial: false,
          error: new Error('API is not available'),
        }));
        return;
      }

      if (isFreshPreview) {
        setPreviewState({
          ...createInitialPreviewState<TItem, TSummary, TRequest>(),
          isLoadingInitial: true,
        });
      } else if (loadMore) {
        setPreviewState((prev) => ({ ...prev, isLoadingMore: true }));
      } else if (switchToTab) {
        setPreviewState((prev) => ({ ...prev, activeTab: switchToTab, isLoadingInitial: true }));
      }

      let requestData: TRequest;
      if (isFreshPreview) {
        requestData = buildPreviewRequest();
      } else if (currentState.lastPreviewedData) {
        requestData = currentState.lastPreviewedData;
      } else {
        return handlePreviewInternal();
      }

      const nextPageToken = loadMore ? currentState.tabStates[targetTab].nextPageToken : undefined;

      try {
        const result = await previewApi({}, requestData, {
          filterStatus: targetTab,
          pageSize: DEFAULT_PREVIEW_PAGE_SIZE,
          nextPageToken,
        });

        setPreviewState((prev) => {
          const currentTabState = prev.tabStates[targetTab];
          const newItems = loadMore ? [...currentTabState.items, ...result.items] : result.items;

          return {
            ...prev,
            isLoadingInitial: false,
            isLoadingMore: false,
            summary: result.summary,
            lastPreviewedData: isFreshPreview ? requestData : prev.lastPreviewedData,
            tabStates: {
              ...prev.tabStates,
              [targetTab]: {
                items: newItems,
                nextPageToken: result.nextPageToken,
                hasMore: !!result.nextPageToken && result.items.length > 0,
              },
            },
            error: undefined,
          };
        });
      } catch (error) {
        const err = error instanceof Error ? error : new Error('Failed to preview source');
        setPreviewState((prev) => ({
          ...prev,
          isLoadingInitial: false,
          isLoadingMore: false,
          error: err,
        }));
      }
    },
    [apiAvailable, buildPreviewRequest, previewApi],
  );

  const handleTabChange = React.useCallback(
    (newTab: CatalogSettingsPreviewTab) => {
      const currentState = previewStateRef.current;
      if (newTab === currentState.activeTab) {
        return;
      }
      const tabState = currentState.tabStates[newTab];
      if (tabState.items.length === 0) {
        handlePreviewInternal({ switchToTab: newTab });
      } else {
        setPreviewState((prev) => ({ ...prev, activeTab: newTab }));
      }
    },
    [handlePreviewInternal],
  );

  const handleLoadMore = React.useCallback(() => {
    handlePreviewInternal({ loadMore: true });
  }, [handlePreviewInternal]);

  React.useEffect(() => {
    const hasNoResults =
      previewState.tabStates[CatalogSettingsPreviewTab.INCLUDED].items.length === 0;
    if (isEditMode && canPreview && hasNoResults) {
      handlePreviewInternal();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return {
    previewState,
    handlePreviewInternal,
    handleTabChange,
    handleLoadMore,
    hasFormChanged,
  };
};
