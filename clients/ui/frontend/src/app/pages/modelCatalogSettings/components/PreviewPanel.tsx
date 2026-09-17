import * as React from 'react';
import { Alert, Icon } from '@patternfly/react-core';
import { CheckCircleIcon, ExclamationTriangleIcon, TimesCircleIcon } from '@patternfly/react-icons';
import { SourcePreviewPanel } from '~/app/shared/catalogSettings';
import { CatalogSourcePreviewModel, CatalogSourcePreviewSummary } from '~/app/modelCatalogTypes';
import {
  PAGE_TITLES,
  ERROR_MESSAGES,
  EMPTY_STATE_TEXT,
  BUTTON_LABELS,
  PREVIEW_ALERTS,
} from '~/app/pages/modelCatalogSettings/constants';
import {
  isPreviewModelGatedAccessDenied,
  previewHasGatedAccessDeniedModels,
} from '~/app/pages/modelCatalogSettings/utils/modelCatalogSettingsUtils';
import {
  UseSourcePreviewResult,
  PreviewMode,
} from '~/app/pages/modelCatalogSettings/useSourcePreview';
import { CatalogSettingsPreviewTab } from '~/app/shared/catalogSettings/hooks/previewTypes';

type PreviewPanelProps = {
  preview: UseSourcePreviewResult;
  isSourceEnabled: boolean;
};

const initialEmptyStateBody = (
  <>
    To view the models from this source that will appear in the model catalog, complete all required
    fields, then click <strong>Preview</strong>.
  </>
);

const renderListItemIcon = (model: CatalogSourcePreviewModel): React.ReactNode => {
  if (isPreviewModelGatedAccessDenied(model)) {
    return (
      <Icon status="warning">
        <ExclamationTriangleIcon aria-label="Gated access warning" />
      </Icon>
    );
  }

  if (model.included) {
    return <CheckCircleIcon color="green" aria-label="Included model" />;
  }

  return <TimesCircleIcon color="red" aria-label="Excluded model" />;
};

const PreviewPanel: React.FC<PreviewPanelProps> = ({ preview, isSourceEnabled }) => {
  const {
    previewState,
    handlePreview,
    handleTabChange,
    handleLoadMore,
    hasFormChanged,
    canPreview,
    previewDisabledTooltip,
  } = preview;
  const { isLoadingInitial, isLoadingMore, activeTab, summary, tabStates, error, mode } =
    previewState;
  const previewError = mode === PreviewMode.PREVIEW ? error : undefined;

  const showSourceDisabledWarning = !isSourceEnabled && !!summary && !previewError;

  const hasGatedAccessDeniedModels = React.useMemo(
    () =>
      previewHasGatedAccessDeniedModels([
        ...tabStates[CatalogSettingsPreviewTab.INCLUDED].items,
        ...tabStates[CatalogSettingsPreviewTab.EXCLUDED].items,
      ]),
    [tabStates],
  );

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
      onPreview={handlePreview}
      onLoadMore={handleLoadMore}
      onTabChange={handleTabChange}
      previewDisabledTooltip={previewDisabledTooltip}
      renderListItemIcon={renderListItemIcon}
      sourceDisabledWarning={
        showSourceDisabledWarning ? (
          <Alert
            variant="warning"
            isInline
            title={PREVIEW_ALERTS.SOURCE_DISABLED_TITLE}
            className="pf-v6-u-mb-md"
            data-testid="source-disabled-warning"
          >
            {PREVIEW_ALERTS.SOURCE_DISABLED_BODY}
          </Alert>
        ) : undefined
      }
      gatedAccessWarning={
        hasGatedAccessDeniedModels ? (
          <Alert
            variant="warning"
            isInline
            title={PREVIEW_ALERTS.GATED_ACCESS_REQUIRED_TITLE}
            className="pf-v6-u-mb-md"
            data-testid="preview-gated-access-alert"
          >
            {PREVIEW_ALERTS.GATED_ACCESS_REQUIRED_BODY}
          </Alert>
        ) : undefined
      }
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
