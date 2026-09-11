import * as React from 'react';
import {
  EmptyState,
  EmptyStateVariant,
  EmptyStateBody,
  EmptyStateFooter,
  EmptyStateActions,
  Flex,
  FlexItem,
  Title,
  Tabs,
  Tab,
  TabTitleText,
  Alert,
  List,
  ListItem,
  Spinner,
  Button,
  AlertActionLink,
  Icon,
} from '@patternfly/react-core';
import { CheckCircleIcon, ExclamationTriangleIcon, TimesCircleIcon } from '@patternfly/react-icons';
import {
  PAGE_TITLES,
  ERROR_MESSAGES,
  EMPTY_STATE_TEXT,
} from '~/app/pages/modelCatalogSettings/constants';
import { isPreviewModelGatedAccessDenied } from '~/app/pages/modelCatalogSettings/utils/modelCatalogSettingsUtils';
import { CatalogSourcePreviewModel } from '~/app/modelCatalogTypes';
import { UseSourcePreviewResult } from '~/app/pages/modelCatalogSettings/useSourcePreview';
import { CatalogSettingsPreviewTab } from '~/app/shared/catalogSettings/hooks/previewTypes';
import PreviewButton from './PreviewButton';

type PreviewPanelProps = {
  preview: UseSourcePreviewResult;
};

const PreviewPanel: React.FC<PreviewPanelProps> = ({ preview }) => {
  // Derive values from preview
  const {
    previewState,
    handlePreview,
    handleTabChange,
    handleLoadMore,
    hasFormChanged,
    canPreview,
    previewDisabledTooltip,
  } = preview;
  const { isLoadingInitial, isLoadingMore, activeTab, summary, tabStates, error } = previewState;
  const { items, hasMore } = tabStates[activeTab];
  const previewError = error;

  const onPreview = () => handlePreview();
  const onLoadMore = () => handleLoadMore();

  const handleTabSelect = (_event: React.MouseEvent, tabIndex: string | number) => {
    handleTabChange(
      tabIndex === 0 ? CatalogSettingsPreviewTab.INCLUDED : CatalogSettingsPreviewTab.EXCLUDED,
    );
  };

  const renderModelIcon = (model: CatalogSourcePreviewModel) => {
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

  const renderEmptyState = () => {
    if (previewError) {
      return (
        <EmptyState
          icon={TimesCircleIcon}
          titleText={ERROR_MESSAGES.PREVIEW_FAILED}
          variant={EmptyStateVariant.sm}
        >
          <EmptyStateBody>{previewError.message}</EmptyStateBody>
          <EmptyStateFooter>
            <EmptyStateActions>
              <PreviewButton
                onClick={onPreview}
                isDisabled={!canPreview}
                isLoading={isLoadingInitial}
                variant="link"
                testId="preview-button-panel-retry"
                disabledTooltip={previewDisabledTooltip}
              />
            </EmptyStateActions>
          </EmptyStateFooter>
        </EmptyState>
      );
    }

    return (
      <EmptyState titleText={PAGE_TITLES.PREVIEW_MODELS} variant={EmptyStateVariant.sm}>
        <EmptyStateBody>
          To view the models from this source that will appear in the model catalog, complete all
          required fields, then click <strong>Preview</strong>.
        </EmptyStateBody>
        <EmptyStateFooter>
          <EmptyStateActions>
            <PreviewButton
              onClick={onPreview}
              isDisabled={!canPreview}
              isLoading={isLoadingInitial}
              variant="link"
              testId="preview-button-panel"
              disabledTooltip={previewDisabledTooltip}
            />
          </EmptyStateActions>
        </EmptyStateFooter>
      </EmptyState>
    );
  };

  const renderContent = () => {
    if (isLoadingInitial) {
      return (
        <div className="pf-v6-u-text-align-center pf-v6-u-py-xl">
          <Spinner size="xl" aria-label="Loading preview" />
        </div>
      );
    }

    // Show empty state if no items and no summary (never previewed) or if there's an error
    if ((!items.length && !summary) || previewError) {
      return renderEmptyState();
    }

    return (
      <>
        <Tabs
          activeKey={activeTab === CatalogSettingsPreviewTab.INCLUDED ? 0 : 1}
          onSelect={handleTabSelect}
          aria-label="Preview tabs"
        >
          <Tab eventKey={0} title={<TabTitleText>Models included</TabTitleText>} />
          <Tab eventKey={1} title={<TabTitleText>Models excluded</TabTitleText>} />
        </Tabs>
        <div className="pf-v6-u-mt-md">
          {hasFormChanged && (
            <Alert
              variant="info"
              isInline
              title="Source configuration changed. Refresh the preview."
              className="pf-v6-u-mb-md"
              actionLinks={
                <AlertActionLink onClick={onPreview} data-testid="refresh-preview-link">
                  Refresh preview
                </AlertActionLink>
              }
            />
          )}
          {items.length > 0 ? (
            <>
              <strong>
                {activeTab === CatalogSettingsPreviewTab.INCLUDED
                  ? `${summary?.includedModels ?? 0} of ${summary?.totalModels ?? 0} models included:`
                  : `${summary?.excludedModels ?? 0} of ${summary?.totalModels ?? 0} models excluded:`}
              </strong>
              <List isPlain className="pf-v6-u-mt-md">
                {items.map((model) => (
                  <ListItem key={model.name} icon={renderModelIcon(model)}>
                    {model.name}
                  </ListItem>
                ))}
              </List>
              {hasMore && (
                <div className="pf-v6-u-mt-md pf-v6-u-text-align-center">
                  <Button
                    variant="link"
                    onClick={onLoadMore}
                    isLoading={isLoadingMore}
                    isDisabled={isLoadingMore}
                  >
                    {isLoadingMore ? 'Loading...' : 'Load more'}
                  </Button>
                </div>
              )}
            </>
          ) : (
            <EmptyState
              variant={EmptyStateVariant.sm}
              titleText={
                activeTab === CatalogSettingsPreviewTab.INCLUDED
                  ? EMPTY_STATE_TEXT.NO_MODELS_INCLUDED
                  : EMPTY_STATE_TEXT.NO_MODELS_EXCLUDED
              }
            >
              <EmptyStateBody>
                {activeTab === CatalogSettingsPreviewTab.INCLUDED
                  ? EMPTY_STATE_TEXT.NO_MODELS_INCLUDED_BODY
                  : EMPTY_STATE_TEXT.NO_MODELS_EXCLUDED_BODY}
              </EmptyStateBody>
            </EmptyState>
          )}
        </div>
      </>
    );
  };

  return (
    <div data-testid="preview-panel" className="pf-v6-u-h-100">
      <Flex
        justifyContent={{ default: 'justifyContentSpaceBetween' }}
        alignItems={{ default: 'alignItemsCenter' }}
        className="pf-v6-u-mb-md"
      >
        <FlexItem>
          <Title headingLevel="h2" size="lg">
            {PAGE_TITLES.MODEL_CATALOG_PREVIEW}
          </Title>
        </FlexItem>
        <FlexItem>
          <PreviewButton
            onClick={onPreview}
            isDisabled={!canPreview}
            isLoading={isLoadingInitial}
            variant="secondary"
            testId="preview-button-header"
            disabledTooltip={previewDisabledTooltip}
          />
        </FlexItem>
      </Flex>
      {renderContent()}
    </div>
  );
};

export default PreviewPanel;
