import * as React from 'react';
import { SourcePreviewPanel } from '~/app/shared/catalogSettings';
import {
  McpCatalogSourcePreviewAsset,
  McpCatalogSourcePreviewSummary,
} from '~/app/mcpServerCatalogTypes';
import {
  MCP_PAGE_TITLES,
  MCP_ERROR_MESSAGES,
  MCP_EMPTY_STATE_TEXT,
  MCP_BUTTON_LABELS,
} from '~/app/pages/mcpCatalogSettings/constants';
import { UseMcpSourcePreviewResult } from '~/app/pages/mcpCatalogSettings/useMcpSourcePreview';

type McpPreviewPanelProps = {
  preview: UseMcpSourcePreviewResult;
};

const initialEmptyStateBody = (
  <>
    Complete all required fields, then click <strong>Preview</strong> to see which servers will
    appear in the catalog.
  </>
);

const McpPreviewPanel: React.FC<McpPreviewPanelProps> = ({ preview }) => {
  const {
    previewState,
    handlePreview,
    handleTabChange,
    handleLoadMore,
    hasFormChanged,
    canPreview,
  } = preview;
  const { isLoadingInitial, isLoadingMore, activeTab, summary, tabStates, error } = previewState;

  return (
    <SourcePreviewPanel<McpCatalogSourcePreviewAsset, McpCatalogSourcePreviewSummary>
      activeTab={activeTab}
      tabStates={tabStates}
      summary={summary}
      isLoadingInitial={isLoadingInitial}
      isLoadingMore={isLoadingMore}
      error={error}
      hasFormChanged={hasFormChanged}
      canPreview={canPreview}
      onPreview={handlePreview}
      onLoadMore={handleLoadMore}
      onTabChange={handleTabChange}
      pageTitle={MCP_PAGE_TITLES.MCP_CATALOG_PREVIEW}
      previewLabel={MCP_BUTTON_LABELS.PREVIEW}
      tabsAriaLabel="MCP preview tabs"
      includedTabTitle="MCP servers included"
      excludedTabTitle="MCP servers excluded"
      initialEmptyStateTitle={MCP_PAGE_TITLES.PREVIEW_SERVERS}
      initialEmptyStateBody={initialEmptyStateBody}
      errorStateTitle={MCP_ERROR_MESSAGES.PREVIEW_FAILED}
      noIncludedTitle={MCP_EMPTY_STATE_TEXT.NO_SERVERS_INCLUDED}
      noIncludedBody={MCP_EMPTY_STATE_TEXT.NO_SERVERS_INCLUDED_BODY}
      noExcludedTitle={MCP_EMPTY_STATE_TEXT.NO_SERVERS_EXCLUDED}
      noExcludedBody={MCP_EMPTY_STATE_TEXT.NO_SERVERS_EXCLUDED_BODY}
      getTotalCount={(s) => s.totalAssets}
      getIncludedCount={(s) => s.includedAssets}
      getExcludedCount={(s) => s.excludedAssets}
      includedCountLabel={(included, total) => `${included} of ${total} MCP servers included:`}
      excludedCountLabel={(excluded, total) => `${excluded} of ${total} MCP servers excluded:`}
      testIds={{
        panel: 'mcp-preview-panel',
        previewButtonHeader: 'mcp-preview-button-header',
        previewButtonPanel: 'mcp-preview-button-panel',
        previewButtonPanelRetry: 'mcp-preview-button-panel-retry',
        refreshPreviewLink: 'mcp-refresh-preview-link',
      }}
    />
  );
};

export default McpPreviewPanel;
