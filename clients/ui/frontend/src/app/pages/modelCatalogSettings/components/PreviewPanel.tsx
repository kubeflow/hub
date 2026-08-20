import * as React from 'react';
import { SourcePreviewPanel } from '~/app/shared/catalogSettings';
import { CatalogSourcePreviewModel, CatalogSourcePreviewSummary } from '~/app/modelCatalogTypes';
import {
  PAGE_TITLES,
  ERROR_MESSAGES,
  EMPTY_STATE_TEXT,
  BUTTON_LABELS,
} from '~/app/pages/modelCatalogSettings/constants';
import {
  UseSourcePreviewResult,
  PreviewMode,
} from '~/app/pages/modelCatalogSettings/useSourcePreview';

type PreviewPanelProps = {
  preview: UseSourcePreviewResult;
};

const initialEmptyStateBody = (
  <>
    To view the models from this source that will appear in the model catalog, complete all required
    fields, then click <strong>Preview</strong>.
  </>
);

const PreviewPanel: React.FC<PreviewPanelProps> = ({ preview }) => {
  const {
    previewState,
    handlePreview,
    handleTabChange,
    handleLoadMore,
    hasFormChanged,
    canPreview,
  } = preview;
  const { isLoadingInitial, isLoadingMore, activeTab, summary, tabStates, error, mode } =
    previewState;
  const previewError = mode === PreviewMode.PREVIEW ? error : undefined;

  return (
    <SourcePreviewPanel<CatalogSourcePreviewModel, CatalogSourcePreviewSummary>
      activeTab={activeTab}
      tabStates={tabStates}
      summary={summary}
      isLoadingInitial={isLoadingInitial}
      isLoadingMore={isLoadingMore}
      error={previewError}
      hasFormChanged={hasFormChanged}
      canPreview={canPreview}
      onPreview={() => handlePreview()}
      onLoadMore={() => handleLoadMore()}
      onTabChange={handleTabChange}
      pageTitle={PAGE_TITLES.MODEL_CATALOG_PREVIEW}
      previewLabel={BUTTON_LABELS.PREVIEW}
      tabsAriaLabel="Preview tabs"
      includedTabTitle="Models included"
      excludedTabTitle="Models excluded"
      initialEmptyStateTitle={PAGE_TITLES.PREVIEW_MODELS}
      initialEmptyStateBody={initialEmptyStateBody}
      errorStateTitle={ERROR_MESSAGES.PREVIEW_FAILED}
      noIncludedTitle={EMPTY_STATE_TEXT.NO_MODELS_INCLUDED}
      noIncludedBody={EMPTY_STATE_TEXT.NO_MODELS_INCLUDED_BODY}
      noExcludedTitle={EMPTY_STATE_TEXT.NO_MODELS_EXCLUDED}
      noExcludedBody={EMPTY_STATE_TEXT.NO_MODELS_EXCLUDED_BODY}
      getTotalCount={(s) => s.totalModels}
      getIncludedCount={(s) => s.includedModels}
      getExcludedCount={(s) => s.excludedModels}
      includedCountLabel={(included, total) => `${included} of ${total} models included:`}
      excludedCountLabel={(excluded, total) => `${excluded} of ${total} models excluded:`}
    />
  );
};

export default PreviewPanel;
