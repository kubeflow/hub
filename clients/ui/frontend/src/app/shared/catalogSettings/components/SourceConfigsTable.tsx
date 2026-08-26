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
import { SortableData, Table } from 'mod-arch-shared';
import SourceConfigsTableRow, {
  SourceConfigRowBase,
  SourceVisibilityLabelInfo,
} from './SourceConfigsTableRow';

type SourceConfigsTableProps<TConfig extends SourceConfigRowBase> = {
  sourceConfigs: TConfig[];
  columns: SortableData<TConfig>[];
  onAddSource: () => void;
  addSourceLabel: string;
  onDeleteSource: (sourceId: string) => Promise<void>;
  apiAvailable: boolean;
  /** Performs the enable/disable mutation (and any related refreshes); assumes the API is available. */
  onToggleUpdate: (checked: boolean, config: TConfig) => Promise<void>;
  loadError?: Error;
  getManageSourceUrl: (id: string) => string;
  renderExtraCells?: (config: TConfig) => React.ReactNode;
  visibilityColumnLabel: string;
  getVisibilityLabel: (config: TConfig) => SourceVisibilityLabelInfo;
  getSourceTypeLabel: (config: TConfig) => string;
  StatusComponent: React.ComponentType<{ sourceConfig: TConfig }>;
  testIdPrefix?: string;
  deleteModalTitle?: string;
  deleteModalBody: (config: TConfig) => React.ReactNode;
};

const SourceConfigsTable = <TConfig extends SourceConfigRowBase>({
  sourceConfigs,
  columns,
  onAddSource,
  addSourceLabel,
  onDeleteSource,
  apiAvailable,
  onToggleUpdate,
  loadError,
  getManageSourceUrl,
  renderExtraCells,
  visibilityColumnLabel,
  getVisibilityLabel,
  getSourceTypeLabel,
  StatusComponent,
  testIdPrefix = '',
  deleteModalTitle,
  deleteModalBody,
}: SourceConfigsTableProps<TConfig>): React.ReactElement => {
  const [toggleError, setToggleError] = React.useState<Error | undefined>(undefined);
  const [updatingToggleId, setUpdatingToggleId] = React.useState<string | null>(null);

  const handleEnableToggle = async (checked: boolean, config: TConfig) => {
    if (!apiAvailable) {
      setToggleError(new Error('API is not available'));
      return;
    }
    setUpdatingToggleId(config.id);
    setToggleError(undefined);

    try {
      await onToggleUpdate(checked, config);
    } catch (e) {
      setToggleError(
        e instanceof Error ? e : new Error(`Error enabling/disabling source ${config.name}`),
      );
    } finally {
      setUpdatingToggleId(null);
    }
  };

  return (
    <Stack hasGutter>
      {loadError && (
        <StackItem>
          <Alert
            variant="danger"
            isInline
            title="Error fetching source statuses"
            data-testid={`${testIdPrefix}source-status-error-alert`}
          >
            {loadError.message}
          </Alert>
        </StackItem>
      )}
      <StackItem>
        <Table
          data-testid={`${testIdPrefix}catalog-source-configs-table`}
          data={sourceConfigs}
          columns={columns}
          toolbarContent={
            <Flex direction={{ default: 'column' }}>
              <FlexItem>
                <Toolbar>
                  <ToolbarContent>
                    <ToolbarItem>
                      <Button
                        variant="primary"
                        onClick={onAddSource}
                        data-testid={`${testIdPrefix}add-source-button`}
                      >
                        {addSourceLabel}
                      </Button>
                    </ToolbarItem>
                  </ToolbarContent>
                </Toolbar>
              </FlexItem>
              {toggleError && (
                <FlexItem>
                  <Alert
                    variant="danger"
                    data-testid={`${testIdPrefix}toggle-alert`}
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
            <SourceConfigsTableRow
              key={config.id}
              sourceConfig={config}
              isUpdatingToggle={updatingToggleId === config.id}
              onToggleUpdate={handleEnableToggle}
              onDeleteSource={onDeleteSource}
              getManageSourceUrl={getManageSourceUrl}
              renderExtraCells={renderExtraCells}
              visibilityColumnLabel={visibilityColumnLabel}
              getVisibilityLabel={getVisibilityLabel}
              getSourceTypeLabel={getSourceTypeLabel}
              StatusComponent={StatusComponent}
              testIdPrefix={testIdPrefix}
              deleteModalTitle={deleteModalTitle}
              deleteModalBody={deleteModalBody}
            />
          )}
          variant="compact"
        />
      </StackItem>
    </Stack>
  );
};

export default SourceConfigsTable;
