import * as React from 'react';
import { SourceConfigsTable, SourceVisibilityLabelInfo } from '~/app/shared/catalogSettings';
import { McpCatalogSourceConfig } from '~/app/mcpServerCatalogTypes';
import { McpCatalogSettingsContext } from '~/app/context/mcpCatalogSettings/McpCatalogSettingsContext';
import {
  MCP_ADD_SOURCE_TITLE,
  mcpManageSourceUrl,
} from '~/app/routes/mcpCatalogSettings/mcpCatalogSettings';
import {
  McpServerVisibilityBadgeColor,
  MCP_SOURCE_TYPE_LABELS,
} from '~/app/pages/mcpCatalogSettings/const';
import McpCatalogSourceStatus from '~/app/pages/mcpCatalogSettings/components/McpCatalogSourceStatus';
import { mcpCatalogSourceConfigsColumns } from './McpCatalogSourceConfigsTableColumns';

type McpCatalogSourceConfigsTableProps = {
  mcpCatalogSourceConfigs: McpCatalogSourceConfig[];
  onAddSource: () => void;
  onDeleteSource: (sourceId: string) => Promise<void>;
};

const hasFilters = (config: McpCatalogSourceConfig): boolean =>
  (config.includedServers?.length ?? 0) > 0 || (config.excludedServers?.length ?? 0) > 0;

const getVisibilityLabel = (config: McpCatalogSourceConfig): SourceVisibilityLabelInfo =>
  hasFilters(config)
    ? {
        text: 'Filtered',
        color: McpServerVisibilityBadgeColor.FILTERED,
        testId: `mcp-server-visibility-filtered-${config.id}`,
      }
    : {
        text: 'All servers',
        color: McpServerVisibilityBadgeColor.UNFILTERED,
        variant: 'outline',
        testId: `mcp-server-visibility-unfiltered-${config.id}`,
      };

const StatusComponent: React.FC<{ sourceConfig: McpCatalogSourceConfig }> = ({ sourceConfig }) => (
  <McpCatalogSourceStatus mcpCatalogSourceConfig={sourceConfig} />
);

const McpCatalogSourceConfigsTable: React.FC<McpCatalogSourceConfigsTableProps> = ({
  mcpCatalogSourceConfigs,
  onAddSource,
  onDeleteSource,
}) => {
  const { apiState, refreshMcpCatalogSourceConfigs, mcpCatalogSourcesLoadError } =
    React.useContext(McpCatalogSettingsContext);

  const handleToggleUpdate = React.useCallback(
    async (checked: boolean, config: McpCatalogSourceConfig) => {
      await apiState.api.updateMcpCatalogSourceConfig({}, config.id, { enabled: checked });
      refreshMcpCatalogSourceConfigs();
    },
    [apiState.api, refreshMcpCatalogSourceConfigs],
  );

  const deleteModalBody = React.useCallback(
    (config: McpCatalogSourceConfig) => (
      <>
        The <strong>{config.name}</strong> source will be deleted, and its MCP servers will be
        removed from the MCP catalog.
      </>
    ),
    [],
  );

  return (
    <SourceConfigsTable
      sourceConfigs={mcpCatalogSourceConfigs}
      columns={mcpCatalogSourceConfigsColumns}
      onAddSource={onAddSource}
      addSourceLabel={MCP_ADD_SOURCE_TITLE}
      onDeleteSource={onDeleteSource}
      apiAvailable={apiState.apiAvailable}
      onToggleUpdate={handleToggleUpdate}
      loadError={mcpCatalogSourcesLoadError}
      getManageSourceUrl={mcpManageSourceUrl}
      visibilityColumnLabel="Server visibility"
      getVisibilityLabel={getVisibilityLabel}
      getSourceTypeLabel={(config) => MCP_SOURCE_TYPE_LABELS[config.type] ?? config.type}
      StatusComponent={StatusComponent}
      testIdPrefix="mcp-"
      deleteModalBody={deleteModalBody}
    />
  );
};

export default McpCatalogSourceConfigsTable;
