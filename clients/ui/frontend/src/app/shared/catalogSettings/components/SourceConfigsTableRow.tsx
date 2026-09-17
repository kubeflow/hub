import * as React from 'react';
import { ActionsColumn, Td, Tr } from '@patternfly/react-table';
import { Button, Label, LabelProps, Switch } from '@patternfly/react-core';
import { useNavigate } from 'react-router-dom';
import DeleteModal from '~/app/shared/components/DeleteModal';
import { useNotification } from '~/app/hooks/useNotification';

export type SourceConfigRowBase = {
  id: string;
  name: string;
  enabled?: boolean;
  isDefault?: boolean;
};

export type SourceVisibilityLabelInfo = {
  text: string;
  color: LabelProps['color'];
  variant?: LabelProps['variant'];
  testId: string;
};

type SourceConfigsTableRowProps<TConfig extends SourceConfigRowBase> = {
  sourceConfig: TConfig;
  isUpdatingToggle: boolean;
  onToggleUpdate: (checked: boolean, config: TConfig) => void;
  onDeleteSource: (sourceId: string) => Promise<void>;
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

const SourceConfigsTableRow = <TConfig extends SourceConfigRowBase>({
  sourceConfig,
  isUpdatingToggle,
  onToggleUpdate,
  onDeleteSource,
  getManageSourceUrl,
  renderExtraCells,
  visibilityColumnLabel,
  getVisibilityLabel,
  getSourceTypeLabel,
  StatusComponent,
  testIdPrefix = '',
  deleteModalTitle = 'Delete a source',
  deleteModalBody,
}: SourceConfigsTableRowProps<TConfig>): React.ReactElement => {
  const navigate = useNavigate();
  const notification = useNotification();
  const [isDeleteModalOpen, setIsDeleteModalOpen] = React.useState(false);
  const [isDeleting, setIsDeleting] = React.useState(false);
  const [deleteError, setDeleteError] = React.useState<Error | undefined>();

  const isDefault = sourceConfig.isDefault ?? false;
  const isEnabled = sourceConfig.enabled ?? true;
  const visibilityLabel = getVisibilityLabel(sourceConfig);

  const handleEnableToggle = (checked: boolean) => {
    onToggleUpdate(checked, sourceConfig);
  };

  const handleManageSource = () => {
    navigate(getManageSourceUrl(sourceConfig.id));
  };

  const handleDeleteClick = () => {
    setDeleteError(undefined);
    setIsDeleteModalOpen(true);
  };

  const handleDeleteConfirm = async () => {
    setIsDeleting(true);
    setDeleteError(undefined);

    try {
      await onDeleteSource(sourceConfig.id);
      setIsDeleteModalOpen(false);
      notification.success(`${sourceConfig.name} deleted successfully`);
    } catch (error) {
      setDeleteError(error instanceof Error ? error : new Error('Failed to delete source'));
    } finally {
      setIsDeleting(false);
    }
  };

  const handleCloseDeleteModal = () => {
    if (!isDeleting) {
      setIsDeleteModalOpen(false);
    }
  };

  return (
    <>
      <Tr>
        <Td dataLabel="Name" style={{ verticalAlign: 'middle' }}>
          <span data-testid={`${testIdPrefix}source-name-${sourceConfig.id}`}>
            {sourceConfig.name}
          </span>
        </Td>
        {renderExtraCells?.(sourceConfig)}
        <Td dataLabel={visibilityColumnLabel} style={{ verticalAlign: 'middle' }}>
          <Label
            color={visibilityLabel.color}
            variant={visibilityLabel.variant}
            data-testid={visibilityLabel.testId}
          >
            {visibilityLabel.text}
          </Label>
        </Td>
        <Td dataLabel="Source type" style={{ verticalAlign: 'middle' }}>
          <span data-testid={`${testIdPrefix}source-type-${sourceConfig.id}`}>
            {getSourceTypeLabel(sourceConfig)}
          </span>
        </Td>
        <Td dataLabel="Enable" style={{ verticalAlign: 'middle' }}>
          <Switch
            data-testid={`${testIdPrefix}enable-toggle-${sourceConfig.id}`}
            id={`${testIdPrefix}enable-toggle-${sourceConfig.id}`}
            aria-label={`Enable ${sourceConfig.name}`}
            isChecked={isEnabled}
            isDisabled={isUpdatingToggle}
            onChange={(_event, checked) => handleEnableToggle(checked)}
          />
        </Td>
        <Td dataLabel="Validation status" style={{ verticalAlign: 'middle' }}>
          <StatusComponent sourceConfig={sourceConfig} />
        </Td>
        <Td dataLabel="Actions" style={{ verticalAlign: 'middle' }}>
          <Button
            variant="link"
            onClick={handleManageSource}
            data-testid={`${testIdPrefix}manage-source-button-${sourceConfig.id}`}
          >
            Manage source
          </Button>
        </Td>
        <Td isActionCell style={{ verticalAlign: 'middle' }}>
          {!isDefault && (
            <ActionsColumn
              items={[{ title: 'Delete source', onClick: handleDeleteClick }]}
              data-testid={`${testIdPrefix}source-actions-${sourceConfig.id}`}
            />
          )}
        </Td>
      </Tr>
      {isDeleteModalOpen && (
        <DeleteModal
          title={deleteModalTitle}
          testId={`${testIdPrefix}delete-source-modal`}
          onClose={handleCloseDeleteModal}
          deleting={isDeleting}
          onDelete={handleDeleteConfirm}
          deleteName={sourceConfig.name}
          error={deleteError}
        >
          {deleteModalBody(sourceConfig)}
        </DeleteModal>
      )}
    </>
  );
};

export default SourceConfigsTableRow;
