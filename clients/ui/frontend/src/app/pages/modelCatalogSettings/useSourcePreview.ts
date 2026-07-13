import * as React from 'react';
import { isEqual } from 'lodash-es';
import { isPreviewReady } from '~/app/pages/modelCatalogSettings/utils/validation';
import { transformFormDataToConfig } from '~/app/pages/modelCatalogSettings/utils/modelCatalogSettingsUtils';
import {
  CatalogSourceConfig,
  CatalogSourceType,
  CatalogSourcePreviewRequest,
  CatalogSourcePreviewModel,
  CatalogSourcePreviewSummary,
} from '~/app/modelCatalogTypes';
import { ModelCatalogSettingsAPIState } from '~/app/hooks/modelCatalogSettings/useModelCatalogSettingsAPIState';
import { ManageSourceFormData } from './useManageSourceData';

export enum PreviewMode {
  PREVIEW = 'preview',
  VALIDATE = 'validate',
}

export enum PreviewTab {
  INCLUDED = 'included',
  EXCLUDED = 'excluded',
}

const DEFAULT_PREVIEW_PAGE_SIZE = 20;

const getTargetTab = (
  isFreshPreview: boolean,
  switchToTab: PreviewTab | undefined,
  activeTab: PreviewTab,
): PreviewTab => {
  if (isFreshPreview) {
    return PreviewTab.INCLUDED;
  }
  return switchToTab ?? activeTab;
};

export type PreviewTabState = {
  items: CatalogSourcePreviewModel[];
  nextPageToken?: string;
  hasMore: boolean;
};

const initialTabState: PreviewTabState = {
  items: [],
  nextPageToken: undefined,
  hasMore: false,
};

export type PreviewState = {
  mode?: PreviewMode;
  isLoadingInitial: boolean;
  isLoadingMore: boolean;
  summary?: CatalogSourcePreviewSummary;
  tabStates: Record<PreviewTab, PreviewTabState>;
  error?: Error;
  resultDismissed: boolean;
  lastPreviewedData?: CatalogSourcePreviewRequest;
  activeTab: PreviewTab;
};

export interface UseSourcePreviewOptions {
  formData: ManageSourceFormData;
  existingSourceConfig?: CatalogSourceConfig;
  apiState: ModelCatalogSettingsAPIState;
  isEditMode: boolean;
}

export interface UseSourcePreviewResult {
  // State
  previewState: PreviewState;

  // Actions
  handlePreview: (mode?: PreviewMode) => Promise<void>;
  handleTabChange: (tab: PreviewTab) => void;
  handleLoadMore: () => void;
  handleValidate: () => Promise<void>;
  clearValidationSuccess: () => void;

  // Derived
  hasFormChanged: boolean;
  isValidating: boolean;
  validationError?: Error;
  isValidationSuccess: boolean;
  canPreview: boolean;
}

export const useSourcePreview = ({
  formData,
  existingSourceConfig,
  apiState,
  isEditMode,
}: UseSourcePreviewOptions): UseSourcePreviewResult => {
  const [previewState, setPreviewState] = React.useState<PreviewState>({
    isLoadingInitial: false,
    isLoadingMore: false,
    tabStates: {
      [PreviewTab.INCLUDED]: initialTabState,
      [PreviewTab.EXCLUDED]: initialTabState,
    },
    resultDismissed: false,
    activeTab: PreviewTab.INCLUDED,
  });

  const previewStateRef = React.useRef(previewState);
  previewStateRef.current = previewState;

  const canPreview = isPreviewReady(formData);

  const buildPreviewRequest = React.useCallback((): CatalogSourcePreviewRequest => {
    const payload = transformFormDataToConfig(formData, existingSourceConfig);

    const request: CatalogSourcePreviewRequest = {
      type: payload.type,
      includedModels: payload.includedModels,
      excludedModels: payload.excludedModels,
    };

    if (payload.type === CatalogSourceType.HUGGING_FACE) {
      request.properties = {
        allowedOrganization: payload.allowedOrganization,
        apiKey: payload.apiKey,
      };
    } else {
      request.properties = {
        yaml: payload.yaml,
        yamlCatalogPath: payload.yamlCatalogPath,
      };
    }

    return request;
  }, [formData, existingSourceConfig]);

  // Derive whether form has changed since last preview
  const hasFormChanged = React.useMemo(() => {
    if (!previewState.lastPreviewedData) {
      return false;
    }
    const currentRequest = buildPreviewRequest();
    return !isEqual(currentRequest, previewState.lastPreviewedData);
  }, [buildPreviewRequest, previewState.lastPreviewedData]);

  // Derive validation states
  const isValidating = previewState.mode === PreviewMode.VALIDATE && previewState.isLoadingInitial;
  const validationError =
    previewState.mode === PreviewMode.VALIDATE ? previewState.error : undefined;
  const isValidationSuccess =
    previewState.mode === PreviewMode.VALIDATE &&
    !previewState.isLoadingInitial &&
    !previewState.error &&
    !previewState.resultDismissed;

  const handlePreview = React.useCallback(
    async (
      mode: PreviewMode = PreviewMode.PREVIEW,
      options?: {
        loadMore?: boolean;
        switchToTab?: PreviewTab;
      },
    ) => {
      const { loadMore = false, switchToTab } = options ?? {};
      const isFreshPreview = !loadMore && !switchToTab;
      const currentState = previewStateRef.current;
      const targetTab = getTargetTab(isFreshPreview, switchToTab, currentState.activeTab);

      if (!apiState.apiAvailable) {
        setPreviewState((prev) => ({
          ...prev,
          mode,
          isLoadingInitial: false,
          error: new Error('API is not available'),
          resultDismissed: false,
        }));
        return;
      }

      // For fresh preview, reset everything to clean state
      if (isFreshPreview) {
        setPreviewState({
          mode,
          isLoadingInitial: true,
          isLoadingMore: false,
          tabStates: {
            [PreviewTab.INCLUDED]: initialTabState,
            [PreviewTab.EXCLUDED]: initialTabState,
          },
          activeTab: PreviewTab.INCLUDED,
          error: undefined,
          resultDismissed: false,
          summary: undefined,
          lastPreviewedData: undefined,
        });
      } else if (loadMore) {
        setPreviewState((prev) => ({ ...prev, isLoadingMore: true }));
      } else if (switchToTab) {
        setPreviewState((prev) => ({ ...prev, activeTab: switchToTab, isLoadingInitial: true }));
      }

      // Use lastPreviewedData for load more / tab switch, current formData for fresh
      let requestData: CatalogSourcePreviewRequest;
      if (isFreshPreview) {
        requestData = buildPreviewRequest();
      } else if (currentState.lastPreviewedData) {
        requestData = currentState.lastPreviewedData;
      } else {
        // For non-fresh requests, lastPreviewedData must exist (guard against edge case)
        // eslint-disable-next-line no-console
        console.warn(
          'Attempted load more / tab switch without lastPreviewedData, triggering fresh preview',
        );
        return handlePreview(mode);
      }

      // Get token for load more
      const nextPageToken = loadMore ? currentState.tabStates[targetTab].nextPageToken : undefined;

      try {
        const result = await apiState.api.previewCatalogSource({}, requestData, {
          filterStatus: targetTab,
          pageSize: DEFAULT_PREVIEW_PAGE_SIZE,
          nextPageToken,
        });

        // Update state based on operation type
        setPreviewState((prev) => {
          const currentTabState = prev.tabStates[targetTab];
          const newItems = loadMore ? [...currentTabState.items, ...result.items] : result.items;

          return {
            ...prev,
            mode,
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
            resultDismissed: false,
          };
        });
      } catch (error) {
        const err = error instanceof Error ? error : new Error('Failed to preview source');

        setPreviewState((prev) => ({
          ...prev,
          mode,
          isLoadingInitial: false,
          isLoadingMore: false,
          error: err,
          resultDismissed: false,
        }));
      }
    },
    [apiState, buildPreviewRequest],
  );

  const handleTabChange = React.useCallback(
    (newTab: PreviewTab) => {
      const currentState = previewStateRef.current;
      if (newTab === currentState.activeTab) {
        return;
      }

      const tabState = currentState.tabStates[newTab];
      if (tabState.items.length === 0) {
        // Tab not yet loaded, fetch it
        handlePreview(PreviewMode.PREVIEW, { switchToTab: newTab });
      } else {
        // Tab already has data, just switch
        setPreviewState((prev) => ({ ...prev, activeTab: newTab }));
      }
    },
    [handlePreview],
  );

  const handleLoadMore = React.useCallback(() => {
    handlePreview(PreviewMode.PREVIEW, { loadMore: true });
  }, [handlePreview]);

  const handleValidate = React.useCallback(async () => {
    await handlePreview(PreviewMode.VALIDATE);
  }, [handlePreview]);

  const clearValidationSuccess = React.useCallback(() => {
    setPreviewState((prev) => ({ ...prev, resultDismissed: true }));
  }, []);

  // Auto-trigger preview on mount in edit mode
  React.useEffect(() => {
    const hasNoResults = previewState.tabStates[PreviewTab.INCLUDED].items.length === 0;
    if (isEditMode && canPreview && hasNoResults) {
      handlePreview();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return {
    previewState,
    handlePreview,
    handleTabChange,
    handleLoadMore,
    handleValidate,
    clearValidationSuccess,
    hasFormChanged,
    isValidating,
    validationError,
    isValidationSuccess,
    canPreview,
  };
};
