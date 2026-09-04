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
} from '@patternfly/react-core';
import { CheckCircleIcon, TimesCircleIcon } from '@patternfly/react-icons';
import { CatalogSettingsPreviewTab } from '~/app/shared/catalogSettings/hooks/previewTypes';
import PreviewButton from './PreviewButton';

export type SourcePreviewItem = {
  name: string;
  included: boolean;
};

export type SourcePreviewTabState<TItem> = {
  items: TItem[];
  hasMore: boolean;
};

export type SourcePreviewPanelTestIds = {
  panel?: string;
  previewButtonHeader?: string;
  previewButtonPanel?: string;
  previewButtonPanelRetry?: string;
  refreshPreviewLink?: string;
};

type SourcePreviewPanelProps<TItem extends SourcePreviewItem, TSummary> = {
  activeTab: CatalogSettingsPreviewTab;
  tabStates: Record<CatalogSettingsPreviewTab, SourcePreviewTabState<TItem>>;
  summary?: TSummary;
  isLoadingInitial: boolean;
  isLoadingMore: boolean;
  /** Preview-relevant error; pass `undefined` while in a different mode (e.g. validate). */
  error?: Error;
  hasFormChanged: boolean;
  canPreview: boolean;
  onPreview: () => void;
  onLoadMore: () => void;
  onTabChange: (tab: CatalogSettingsPreviewTab) => void;

  pageTitle: string;
  previewLabel: string;
  tabsAriaLabel: string;
  includedTabTitle: string;
  excludedTabTitle: string;
  initialEmptyStateTitle: string;
  initialEmptyStateBody: React.ReactNode;
  errorStateTitle?: string;
  noIncludedTitle: string;
  noIncludedBody: string;
  noExcludedTitle: string;
  noExcludedBody: string;
  getTotalCount: (summary: TSummary) => number;
  getIncludedCount: (summary: TSummary) => number;
  getExcludedCount: (summary: TSummary) => number;
  includedCountLabel: (included: number, total: number) => string;
  excludedCountLabel: (excluded: number, total: number) => string;
  testIds?: SourcePreviewPanelTestIds;
};

const SourcePreviewPanel = <TItem extends SourcePreviewItem, TSummary>({
  activeTab,
  tabStates,
  summary,
  isLoadingInitial,
  isLoadingMore,
  error,
  hasFormChanged,
  canPreview,
  onPreview,
  onLoadMore,
  onTabChange,
  pageTitle,
  previewLabel,
  tabsAriaLabel,
  includedTabTitle,
  excludedTabTitle,
  initialEmptyStateTitle,
  initialEmptyStateBody,
  errorStateTitle = 'Preview failed',
  noIncludedTitle,
  noIncludedBody,
  noExcludedTitle,
  noExcludedBody,
  getTotalCount,
  getIncludedCount,
  getExcludedCount,
  includedCountLabel,
  excludedCountLabel,
  testIds,
}: SourcePreviewPanelProps<TItem, TSummary>): React.ReactElement => {
  const {
    panel = 'preview-panel',
    previewButtonHeader = 'preview-button-header',
    previewButtonPanel = 'preview-button-panel',
    previewButtonPanelRetry = 'preview-button-panel-retry',
    refreshPreviewLink = 'refresh-preview-link',
  } = testIds ?? {};

  const { items, hasMore } = tabStates[activeTab];

  const handleTabSelect = (_event: React.MouseEvent, tabIndex: string | number) => {
    onTabChange(
      tabIndex === 0 ? CatalogSettingsPreviewTab.INCLUDED : CatalogSettingsPreviewTab.EXCLUDED,
    );
  };

  const renderEmptyState = () => {
    if (error) {
      return (
        <EmptyState
          icon={TimesCircleIcon}
          titleText={errorStateTitle}
          variant={EmptyStateVariant.sm}
        >
          <EmptyStateBody>{error.message}</EmptyStateBody>
          <EmptyStateFooter>
            <EmptyStateActions>
              <PreviewButton
                label={previewLabel}
                onClick={onPreview}
                isDisabled={!canPreview}
                isLoading={isLoadingInitial}
                variant="link"
                testId={previewButtonPanelRetry}
              />
            </EmptyStateActions>
          </EmptyStateFooter>
        </EmptyState>
      );
    }

    return (
      <EmptyState titleText={initialEmptyStateTitle} variant={EmptyStateVariant.sm}>
        <EmptyStateBody>{initialEmptyStateBody}</EmptyStateBody>
        <EmptyStateFooter>
          <EmptyStateActions>
            <PreviewButton
              label={previewLabel}
              onClick={onPreview}
              isDisabled={!canPreview}
              isLoading={isLoadingInitial}
              variant="link"
              testId={previewButtonPanel}
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
    if ((!items.length && !summary) || error) {
      return renderEmptyState();
    }

    const total = summary ? getTotalCount(summary) : 0;
    const included = summary ? getIncludedCount(summary) : 0;
    const excluded = summary ? getExcludedCount(summary) : 0;

    return (
      <>
        <Tabs
          activeKey={activeTab === CatalogSettingsPreviewTab.INCLUDED ? 0 : 1}
          onSelect={handleTabSelect}
          aria-label={tabsAriaLabel}
        >
          <Tab eventKey={0} title={<TabTitleText>{includedTabTitle}</TabTitleText>} />
          <Tab eventKey={1} title={<TabTitleText>{excludedTabTitle}</TabTitleText>} />
        </Tabs>
        <div className="pf-v6-u-mt-md">
          {hasFormChanged && (
            <Alert
              variant="info"
              isInline
              title="Source configuration changed. Refresh the preview."
              className="pf-v6-u-mb-md"
              actionLinks={
                <AlertActionLink onClick={onPreview} data-testid={refreshPreviewLink}>
                  Refresh preview
                </AlertActionLink>
              }
            />
          )}
          {items.length > 0 ? (
            <>
              <strong>
                {activeTab === CatalogSettingsPreviewTab.INCLUDED
                  ? includedCountLabel(included, total)
                  : excludedCountLabel(excluded, total)}
              </strong>
              <List isPlain className="pf-v6-u-mt-md">
                {items.map((item) => (
                  <ListItem
                    key={item.name}
                    icon={
                      item.included ? (
                        <CheckCircleIcon color="green" />
                      ) : (
                        <TimesCircleIcon color="red" />
                      )
                    }
                  >
                    {item.name}
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
                activeTab === CatalogSettingsPreviewTab.INCLUDED ? noIncludedTitle : noExcludedTitle
              }
            >
              <EmptyStateBody>
                {activeTab === CatalogSettingsPreviewTab.INCLUDED ? noIncludedBody : noExcludedBody}
              </EmptyStateBody>
            </EmptyState>
          )}
        </div>
      </>
    );
  };

  return (
    <div data-testid={panel} className="pf-v6-u-h-100">
      <Flex
        justifyContent={{ default: 'justifyContentSpaceBetween' }}
        alignItems={{ default: 'alignItemsCenter' }}
        className="pf-v6-u-mb-md"
      >
        <FlexItem>
          <Title headingLevel="h2" size="lg">
            {pageTitle}
          </Title>
        </FlexItem>
        <FlexItem>
          <PreviewButton
            label={previewLabel}
            onClick={onPreview}
            isDisabled={!canPreview}
            isLoading={isLoadingInitial}
            variant="secondary"
            testId={previewButtonHeader}
          />
        </FlexItem>
      </Flex>
      {renderContent()}
    </div>
  );
};

export default SourcePreviewPanel;
