import * as React from 'react';
import {
  Alert,
  AlertActionCloseButton,
  Button,
  Flex,
  FlexItem,
  Stack,
  StackItem,
  Toolbar,
  ToolbarContent,
  ToolbarItem,
} from '@patternfly/react-core';
import { Table } from 'mod-arch-shared';
import { McpCatalogSourceConfig } from '~/app/mcpServerCatalogTypes';
import { McpCatalogSettingsContext } from '~/app/context/mcpCatalogSettings/McpCatalogSettingsContext';
import { MCP_ADD_SOURCE_TITLE } from '~/app/routes/mcpCatalogSettings/mcpCatalogSettings';
import { mcpCatalogSourceConfigsColumns } from './McpCatalogSourceConfigsTableColumns';
import McpCatalogSourceConfigsTableRow from './McpCatalogSourceConfigsTableRow';

type McpCatalogSourceConfigsTableProps = {
  mcpCatalogSourceConfigs: McpCatalogSourceConfig[];
  onAddSource: () => void;
  onDeleteSource: (sourceId: string) => Promise<void>;
};

const McpCatalogSourceConfigsTable: React.FC<McpCatalogSourceConfigsTableProps> = ({
  mcpCatalogSourceConfigs,
  onAddSource,
  onDeleteSource,
}) => {
  const [toggleError, setToggleError] = React.useState<Error | undefined>(undefined);
  const [updatingToggleId, setUpdatingToggleId] = React.useState<string | null>(null);
  const {
    apiState,
    refreshMcpCatalogSourceConfigs,
    refreshMcpCatalogSources,
    mcpCatalogSourcesLoadError,
    mcpCatalogSources,
    markSourcePending,
  } = React.useContext(McpCatalogSettingsContext);

  const handleEnableToggle = async (
    checked: boolean,
    catalogSourceConfig: McpCatalogSourceConfig,
  ) => {
    if (!apiState.apiAvailable) {
      setToggleError(new Error('API is not available'));
      return;
    }
    setUpdatingToggleId(catalogSourceConfig.id);
    setToggleError(undefined);

    if (checked) {
      const previousStatus =
        mcpCatalogSources?.items?.find((s) => s.id === catalogSourceConfig.id)?.status ?? '';
      markSourcePending(catalogSourceConfig.id, previousStatus);
    }

    try {
      await apiState.api.updateMcpCatalogSourceConfig({}, catalogSourceConfig.id, {
        enabled: checked,
      });
      refreshMcpCatalogSourceConfigs();
      if (checked) {
        refreshMcpCatalogSources();
      }
    } catch (e) {
      if (e instanceof Error) {
        setToggleError(new Error(`Error enabling/disabling source ${catalogSourceConfig.name}`));
      }
    } finally {
      setUpdatingToggleId(null);
    }
  };

  return (
    <Stack hasGutter>
      {mcpCatalogSourcesLoadError && (
        <StackItem>
          <Alert
            variant="danger"
            isInline
            title="Error fetching source statuses"
            data-testid="mcp-source-status-error-alert"
          >
            {mcpCatalogSourcesLoadError.message}
          </Alert>
        </StackItem>
      )}
      <StackItem>
        <Table
          data-testid="mcp-catalog-source-configs-table"
          data={mcpCatalogSourceConfigs}
          columns={mcpCatalogSourceConfigsColumns}
          toolbarContent={
            <Flex direction={{ default: 'column' }}>
              <FlexItem>
                <Toolbar>
                  <ToolbarContent>
                    <ToolbarItem>
                      <Button
                        variant="primary"
                        onClick={onAddSource}
                        data-testid="mcp-add-source-button"
                      >
                        {MCP_ADD_SOURCE_TITLE}
                      </Button>
                    </ToolbarItem>
                  </ToolbarContent>
                </Toolbar>
              </FlexItem>
              {toggleError && (
                <FlexItem>
                  <Alert
                    variant="danger"
                    data-testid="mcp-toggle-alert"
                    title={toggleError.message}
                    actionClose={
                      <AlertActionCloseButton onClose={() => setToggleError(undefined)} />
                    }
                  />
                </FlexItem>
              )}
            </Flex>
          }
          rowRenderer={(config) => (
            <McpCatalogSourceConfigsTableRow
              key={config.id}
              mcpCatalogSourceConfig={config}
              isUpdatingToggle={updatingToggleId === config.id}
              onToggleUpdate={handleEnableToggle}
              onDeleteSource={onDeleteSource}
            />
          )}
          variant="compact"
        />
      </StackItem>
    </Stack>
  );
};

export default McpCatalogSourceConfigsTable;
