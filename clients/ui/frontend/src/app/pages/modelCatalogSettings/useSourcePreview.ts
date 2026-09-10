import * as React from 'react';
import { isPreviewReady } from '~/app/pages/modelCatalogSettings/utils/validation';
import {
  transformFormDataToConfig,
  resolveHuggingFaceApiKeyField,
} from '~/app/pages/modelCatalogSettings/utils/modelCatalogSettingsUtils';
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
  DEFAULT_PREVIEW_PAGE_SIZE,
} from '~/app/shared/catalogSettings/hooks/previewTypes';
import { useCatalogSourcePreviewCore } from '~/app/shared/catalogSettings/hooks/useCatalogSourcePreviewCore';
import { TOOLTIP_MESSAGES } from './constants';
import { ManageSourceFormData } from './useManageSourceData';

export enum PreviewMode {
  PREVIEW = 'preview',
}

type CredentialsValidationStatus = 'unknown' | 'valid' | 'invalid';

const isHuggingFaceWithAccessToken = (formData: ManageSourceFormData): boolean =>
  formData.sourceType === CatalogSourceType.HUGGING_FACE && formData.accessToken.trim().length > 0;

export const isPreviewEnabled = (
  formData: ManageSourceFormData,
  credentialsValidationStatus: CredentialsValidationStatus,
): boolean => {
  if (!isPreviewReady(formData)) {
    return false;
  }
  if (isHuggingFaceWithAccessToken(formData)) {
    return credentialsValidationStatus === 'valid';
  }
  return true;
};

export const getPreviewDisabledTooltip = (
  formData: ManageSourceFormData,
  credentialsValidationStatus: CredentialsValidationStatus,
): string | undefined => {
  if (
    isPreviewReady(formData) &&
    isHuggingFaceWithAccessToken(formData) &&
    credentialsValidationStatus !== 'valid'
  ) {
    return TOOLTIP_MESSAGES.PREVIEW_REQUIRES_VALIDATION;
  }
  return undefined;
};

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
  hasExistingApiKey?: boolean;
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
  previewDisabledTooltip?: string;
}

export const useSourcePreview = ({
  formData,
  existingSourceConfig,
  apiState,
  isEditMode,
  hasExistingApiKey = false,
}: UseSourcePreviewOptions): UseSourcePreviewResult => {
  const [credentialsValidationStatus, setCredentialsValidationStatus] =
    React.useState<CredentialsValidationStatus>('unknown');
  const [isValidating, setIsValidating] = React.useState(false);
  const [validationError, setValidationError] = React.useState<Error | undefined>();
  const [resultDismissed, setResultDismissed] = React.useState(false);
  const [mode, setMode] = React.useState<PreviewMode | undefined>();

  const canPreview = isPreviewEnabled(formData, credentialsValidationStatus);
  const previewDisabledTooltip = getPreviewDisabledTooltip(formData, credentialsValidationStatus);

  const buildPreviewRequest = React.useCallback((): CatalogSourcePreviewRequest => {
    const payload = transformFormDataToConfig(formData, existingSourceConfig);

    const request: CatalogSourcePreviewRequest = {
      id: payload.id,
      type: payload.type,
      includedModels: payload.includedModels,
      excludedModels: payload.excludedModels,
    };

    if (payload.type === CatalogSourceType.HUGGING_FACE) {
      request.properties = {
        allowedOrganization: payload.allowedOrganization,
        ...resolveHuggingFaceApiKeyField(payload.apiKey, {
          tokenModified: formData.tokenModified,
          hasExistingApiKey,
          forPreview: true,
        }),
      };
    } else {
      request.properties = {
        yaml: payload.yaml,
        yamlCatalogPath: payload.yamlCatalogPath,
      };
    }

    return request;
  }, [formData, existingSourceConfig, hasExistingApiKey]);

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

  React.useEffect(() => {
    setCredentialsValidationStatus('unknown');
    setValidationError(undefined);
    setResultDismissed(false);
  }, [formData.accessToken, formData.organization]);

  const isValidationSuccess = credentialsValidationStatus === 'valid' && !resultDismissed;

  const handlePreview = React.useCallback(async () => {
    setMode(PreviewMode.PREVIEW);
    await handlePreviewInternal();
  }, [handlePreviewInternal]);

  const handleValidate = React.useCallback(async () => {
    if (!apiState.apiAvailable) {
      setValidationError(new Error('API is not available'));
      setCredentialsValidationStatus('invalid');
      return;
    }

    setIsValidating(true);
    setValidationError(undefined);
    setResultDismissed(false);

    try {
      await previewApi({}, buildPreviewRequest(), {
        filterStatus: CatalogSettingsPreviewTab.INCLUDED,
        pageSize: DEFAULT_PREVIEW_PAGE_SIZE,
      });
      setCredentialsValidationStatus('valid');
    } catch (error) {
      const err = error instanceof Error ? error : new Error('Failed to validate credentials');
      setValidationError(err);
      setCredentialsValidationStatus('invalid');
    } finally {
      setIsValidating(false);
    }
  }, [apiState.apiAvailable, buildPreviewRequest, previewApi]);

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
    previewDisabledTooltip,
  };
};
