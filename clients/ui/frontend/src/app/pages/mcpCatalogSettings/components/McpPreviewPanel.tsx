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
import {
  MCP_PAGE_TITLES,
  MCP_ERROR_MESSAGES,
  MCP_EMPTY_STATE_TEXT,
  MCP_BUTTON_LABELS,
} from '~/app/pages/mcpCatalogSettings/constants';
import {
  UseMcpSourcePreviewResult,
  McpPreviewTab,
} from '~/app/pages/mcpCatalogSettings/useMcpSourcePreview';

type McpPreviewPanelProps = {
  preview: UseMcpSourcePreviewResult;
};

const McpPreviewPanel: React.FC<McpPreviewPanelProps> = ({ preview }) => {
  const {
    previewState,
    handlePreview: onPreview,
    handleTabChange,
    handleLoadMore: onLoadMore,
    hasFormChanged,
    canPreview,
  } = preview;
  const { isLoadingInitial, isLoadingMore, activeTab, summary, tabStates, error } = previewState;
  const { items, hasMore } = tabStates[activeTab];

  const handleTabSelect = (_event: React.MouseEvent, tabIndex: string | number) => {
    handleTabChange(tabIndex === 0 ? McpPreviewTab.INCLUDED : McpPreviewTab.EXCLUDED);
  };

  const renderEmptyState = () => {
    if (error) {
      return (
        <EmptyState
          icon={TimesCircleIcon}
          titleText={MCP_ERROR_MESSAGES.PREVIEW_FAILED}
          variant={EmptyStateVariant.sm}
        >
          <EmptyStateBody>{error.message}</EmptyStateBody>
          <EmptyStateFooter>
            <EmptyStateActions>
              <Button
                variant="link"
                onClick={onPreview}
                isDisabled={!canPreview}
                isLoading={isLoadingInitial}
                data-testid="mcp-preview-button-panel-retry"
              >
                {MCP_BUTTON_LABELS.PREVIEW}
              </Button>
            </EmptyStateActions>
          </EmptyStateFooter>
        </EmptyState>
      );
    }

    return (
      <EmptyState titleText={MCP_PAGE_TITLES.PREVIEW_SERVERS} variant={EmptyStateVariant.sm}>
        <EmptyStateBody>
          To view the MCP servers from this source that will appear in the MCP catalog with your
          current configuration, complete all required fields, then click <strong>Preview</strong>.
        </EmptyStateBody>
        <EmptyStateFooter>
          <EmptyStateActions>
            <Button
              variant="link"
              onClick={onPreview}
              isDisabled={!canPreview}
              isLoading={isLoadingInitial}
              data-testid="mcp-preview-button-panel"
            >
              {MCP_BUTTON_LABELS.PREVIEW}
            </Button>
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

    if ((!items.length && !summary) || error) {
      return renderEmptyState();
    }

    return (
      <>
        <Tabs
          activeKey={activeTab === McpPreviewTab.INCLUDED ? 0 : 1}
          onSelect={handleTabSelect}
          aria-label="MCP preview tabs"
        >
          <Tab eventKey={0} title={<TabTitleText>MCP servers included</TabTitleText>} />
          <Tab eventKey={1} title={<TabTitleText>MCP servers excluded</TabTitleText>} />
        </Tabs>
        <div className="pf-v6-u-mt-md">
          {hasFormChanged && (
            <Alert
              variant="info"
              isInline
              title="Source configuration changed. Refresh the preview."
              className="pf-v6-u-mb-md"
              actionLinks={
                <AlertActionLink onClick={onPreview} data-testid="mcp-refresh-preview-link">
                  Refresh preview
                </AlertActionLink>
              }
            />
          )}
          {items.length > 0 ? (
            <>
              <strong>
                {activeTab === McpPreviewTab.INCLUDED
                  ? `${summary?.includedAssets ?? 0} of ${summary?.totalAssets ?? 0} MCP servers included:`
                  : `${summary?.excludedAssets ?? 0} of ${summary?.totalAssets ?? 0} MCP servers excluded:`}
              </strong>
              <List isPlain className="pf-v6-u-mt-md">
                {items.map((server) => (
                  <ListItem
                    key={server.name}
                    icon={
                      server.included ? (
                        <CheckCircleIcon color="green" />
                      ) : (
                        <TimesCircleIcon color="red" />
                      )
                    }
                  >
                    {server.name}
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
                activeTab === McpPreviewTab.INCLUDED
                  ? MCP_EMPTY_STATE_TEXT.NO_SERVERS_INCLUDED
                  : MCP_EMPTY_STATE_TEXT.NO_SERVERS_EXCLUDED
              }
            >
              <EmptyStateBody>
                {activeTab === McpPreviewTab.INCLUDED
                  ? MCP_EMPTY_STATE_TEXT.NO_SERVERS_INCLUDED_BODY
                  : MCP_EMPTY_STATE_TEXT.NO_SERVERS_EXCLUDED_BODY}
              </EmptyStateBody>
            </EmptyState>
          )}
        </div>
      </>
    );
  };

  return (
    <div data-testid="mcp-preview-panel" className="pf-v6-u-h-100">
      <Flex
        justifyContent={{ default: 'justifyContentSpaceBetween' }}
        alignItems={{ default: 'alignItemsCenter' }}
        className="pf-v6-u-mb-md"
      >
        <FlexItem>
          <Title headingLevel="h2" size="lg">
            {MCP_PAGE_TITLES.MCP_CATALOG_PREVIEW}
          </Title>
        </FlexItem>
        <FlexItem>
          <Button
            variant="secondary"
            onClick={onPreview}
            isDisabled={!canPreview}
            isLoading={isLoadingInitial}
            data-testid="mcp-preview-button-header"
          >
            {MCP_BUTTON_LABELS.PREVIEW}
          </Button>
        </FlexItem>
      </Flex>
      {renderContent()}
    </div>
  );
};

export default McpPreviewPanel;
