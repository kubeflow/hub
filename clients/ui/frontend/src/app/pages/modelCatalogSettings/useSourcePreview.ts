import * as React from 'react';
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
import {
  CatalogSettingsPreviewTab,
  useCatalogSourcePreviewCore,
} from '~/app/shared/catalogSettings';
import { ManageSourceFormData } from './useManageSourceData';

export enum PreviewMode {
  PREVIEW = 'preview',
  VALIDATE = 'validate',
}

export type PreviewTabState = {
  items: CatalogSourcePreviewModel[];
  nextPageToken?: string;
  hasMore: boolean;
};

export type PreviewState = {
  mode?: PreviewMode;
  isLoadingInitial: boolean;
  isLoadingMore: boolean;
  summary?: CatalogSourcePreviewSummary;
  tabStates: Record<CatalogSettingsPreviewTab, PreviewTabState>;
  error?: Error;
  resultDismissed: boolean;
  lastPreviewedData?: CatalogSourcePreviewRequest;
  activeTab: CatalogSettingsPreviewTab;
};

export interface UseSourcePreviewOptions {
  formData: ManageSourceFormData;
  existingSourceConfig?: CatalogSourceConfig;
  apiState: ModelCatalogSettingsAPIState;
  isEditMode: boolean;
}

export interface UseSourcePreviewResult {
  previewState: PreviewState;
  handlePreview: (mode?: PreviewMode) => Promise<void>;
  handleTabChange: (tab: CatalogSettingsPreviewTab) => void;
  handleLoadMore: () => void;
  handleValidate: () => Promise<void>;
  clearValidationSuccess: () => void;
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
  const canPreview = isPreviewReady(formData);
  const [mode, setMode] = React.useState<PreviewMode | undefined>();
  const [resultDismissed, setResultDismissed] = React.useState(false);

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

  const previewApi = React.useCallback(
    (
      opts: Parameters<ModelCatalogSettingsAPIState['api']['previewCatalogSource']>[0],
      data: CatalogSourcePreviewRequest,
      queryParams?: Parameters<ModelCatalogSettingsAPIState['api']['previewCatalogSource']>[2],
    ) => apiState.api.previewCatalogSource(opts, data, queryParams),
    [apiState.api],
  );

  const {
    previewState: corePreviewState,
    handlePreviewInternal,
    handleTabChange,
    handleLoadMore,
    hasFormChanged,
  } = useCatalogSourcePreviewCore<
    CatalogSourcePreviewModel,
    CatalogSourcePreviewSummary,
    CatalogSourcePreviewRequest
  >({
    canPreview,
    isEditMode,
    apiAvailable: apiState.apiAvailable,
    buildPreviewRequest,
    previewApi,
  });

  const previewState: PreviewState = {
    ...corePreviewState,
    mode,
    resultDismissed,
  };

  const isValidating = mode === PreviewMode.VALIDATE && previewState.isLoadingInitial;
  const validationError = mode === PreviewMode.VALIDATE ? previewState.error : undefined;
  const isValidationSuccess =
    mode === PreviewMode.VALIDATE &&
    !previewState.isLoadingInitial &&
    !previewState.error &&
    !resultDismissed;

  const handlePreview = React.useCallback(
    async (nextMode: PreviewMode = PreviewMode.PREVIEW) => {
      setMode(nextMode);
      setResultDismissed(false);
      await handlePreviewInternal();
    },
    [handlePreviewInternal],
  );

  const handleValidate = React.useCallback(async () => {
    await handlePreview(PreviewMode.VALIDATE);
  }, [handlePreview]);

  const clearValidationSuccess = React.useCallback(() => {
    setResultDismissed(true);
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
