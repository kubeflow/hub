import * as React from 'react';
import {
  SkillCatalogSourceConfig,
  SkillCatalogSourcePreviewRequest,
  SkillCatalogSourcePreviewAsset,
  SkillCatalogSourcePreviewSummary,
  SkillCatalogSourceType,
} from '~/app/skillCatalogTypes';
import { SkillCatalogSettingsAPIState } from '~/app/hooks/skillCatalogSettings/useSkillCatalogSettingsAPIState';
import { CatalogSettingsPreviewTab } from '~/app/shared/catalogSettings/hooks/previewTypes';
import { useCatalogSourcePreviewCore } from '~/app/shared/catalogSettings/hooks/useCatalogSourcePreviewCore';
import { ManageSkillSourceFormData } from './useManageSkillSourceData';

export type SkillPreviewTabState = {
  items: SkillCatalogSourcePreviewAsset[];
  nextPageToken?: string;
  hasMore: boolean;
};

export type SkillPreviewState = {
  isLoadingInitial: boolean;
  isLoadingMore: boolean;
  summary?: SkillCatalogSourcePreviewSummary;
  tabStates: Record<CatalogSettingsPreviewTab, SkillPreviewTabState>;
  error?: Error;
  lastPreviewedData?: SkillCatalogSourcePreviewRequest;
  activeTab: CatalogSettingsPreviewTab;
};

export interface UseSkillSourcePreviewOptions {
  formData: ManageSkillSourceFormData;
  existingSourceConfig?: SkillCatalogSourceConfig;
  apiState: SkillCatalogSettingsAPIState;
  isEditMode: boolean;
}

export interface UseSkillSourcePreviewResult {
  previewState: SkillPreviewState;
  handlePreview: () => Promise<void>;
  handleTabChange: (tab: CatalogSettingsPreviewTab) => void;
  handleLoadMore: () => void;
  hasFormChanged: boolean;
  canPreview: boolean;
}

export const useSkillSourcePreview = ({
  formData,
  apiState,
  isEditMode,
}: UseSkillSourcePreviewOptions): UseSkillSourcePreviewResult => {
  const canPreview =
    formData.repositories.length > 0 && formData.repositories[0].url.trim().length > 0;

  const buildPreviewRequest = React.useCallback((): SkillCatalogSourcePreviewRequest => {
    const repositories = formData.repositories
      .filter((r) => r.url.trim().length > 0)
      .map((r) => {
        const repo = { ...r };
        delete repo.authToken;
        delete repo.provider;
        delete repo.category;
        delete repo.trustTier;
        return repo;
      });

    return {
      type: SkillCatalogSourceType.GIT_SKILLS,
      properties: {
        repositories,
      },
    };
  }, [formData]);

  const previewApi = React.useCallback(
    (
      opts: Parameters<SkillCatalogSettingsAPIState['api']['previewSkillCatalogSource']>[0],
      data: SkillCatalogSourcePreviewRequest,
      queryParams?: Parameters<SkillCatalogSettingsAPIState['api']['previewSkillCatalogSource']>[2],
    ) => apiState.api.previewSkillCatalogSource(opts, data, queryParams),
    [apiState.api],
  );

  const { previewState, handlePreviewInternal, handleTabChange, handleLoadMore, hasFormChanged } =
    useCatalogSourcePreviewCore<
      SkillCatalogSourcePreviewAsset,
      SkillCatalogSourcePreviewSummary,
      SkillCatalogSourcePreviewRequest
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
