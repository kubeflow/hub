import * as React from 'react';
import { isMcpPreviewReady } from '~/app/pages/mcpCatalogSettings/utils/validation';
import { transformMcpFormDataToConfig } from '~/app/pages/mcpCatalogSettings/utils/mcpCatalogSettingsUtils';
import {
  McpCatalogSourceConfig,
  McpCatalogSourcePreviewRequest,
  McpCatalogSourcePreviewAsset,
  McpCatalogSourcePreviewSummary,
} from '~/app/mcpServerCatalogTypes';
import { McpCatalogSettingsAPIState } from '~/app/hooks/mcpCatalogSettings/useMcpCatalogSettingsAPIState';
import {
  CatalogSettingsPreviewTab,
  useCatalogSourcePreviewCore,
} from '~/app/shared/catalogSettings';
import { ManageMcpSourceFormData } from './useManageMcpSourceData';

export type McpPreviewTabState = {
  items: McpCatalogSourcePreviewAsset[];
  nextPageToken?: string;
  hasMore: boolean;
};

export type McpPreviewState = {
  isLoadingInitial: boolean;
  isLoadingMore: boolean;
  summary?: McpCatalogSourcePreviewSummary;
  tabStates: Record<CatalogSettingsPreviewTab, McpPreviewTabState>;
  error?: Error;
  lastPreviewedData?: McpCatalogSourcePreviewRequest;
  activeTab: CatalogSettingsPreviewTab;
};

export interface UseMcpSourcePreviewOptions {
  formData: ManageMcpSourceFormData;
  existingSourceConfig?: McpCatalogSourceConfig;
  apiState: McpCatalogSettingsAPIState;
  isEditMode: boolean;
}

export interface UseMcpSourcePreviewResult {
  previewState: McpPreviewState;
  handlePreview: () => Promise<void>;
  handleTabChange: (tab: CatalogSettingsPreviewTab) => void;
  handleLoadMore: () => void;
  hasFormChanged: boolean;
  canPreview: boolean;
}

export const useMcpSourcePreview = ({
  formData,
  existingSourceConfig,
  apiState,
  isEditMode,
}: UseMcpSourcePreviewOptions): UseMcpSourcePreviewResult => {
  const canPreview = isMcpPreviewReady(formData);

  const buildPreviewRequest = React.useCallback((): McpCatalogSourcePreviewRequest => {
    const config = transformMcpFormDataToConfig(formData, existingSourceConfig);
    return {
      type: config.type,
      includedServers: config.includedServers,
      excludedServers: config.excludedServers,
      properties: {
        yaml: config.yaml,
        yamlCatalogPath: config.yamlCatalogPath,
      },
    };
  }, [formData, existingSourceConfig]);

  const previewApi = React.useCallback(
    (
      opts: Parameters<McpCatalogSettingsAPIState['api']['previewMcpCatalogSource']>[0],
      data: McpCatalogSourcePreviewRequest,
      queryParams?: Parameters<McpCatalogSettingsAPIState['api']['previewMcpCatalogSource']>[2],
    ) => apiState.api.previewMcpCatalogSource(opts, data, queryParams),
    [apiState.api],
  );

  const {
    previewState,
    handlePreviewInternal,
    handleTabChange,
    handleLoadMore,
    hasFormChanged,
  } = useCatalogSourcePreviewCore<
    McpCatalogSourcePreviewAsset,
    McpCatalogSourcePreviewSummary,
    McpCatalogSourcePreviewRequest
  >({
    canPreview,
    isEditMode,
    apiAvailable: apiState.apiAvailable,
    buildPreviewRequest,
    previewApi,
  });

  const handlePreview = React.useCallback(async () => {
    await handlePreviewInternal();
  }, [handlePreviewInternal]);

  return {
    previewState,
    handlePreview,
    handleTabChange,
    handleLoadMore,
    hasFormChanged,
    canPreview,
  };
};
