import * as React from 'react';
import { ActionsColumn, Td, Tr } from '@patternfly/react-table';
import { Button, Switch } from '@patternfly/react-core';
import { useNavigate } from 'react-router-dom';
import type { SkillCatalogSourceConfig } from '~/app/skillCatalogTypes';
import { skillManageSourceUrl } from '~/app/routes/skillCatalogSettings/skillCatalogSettings';
import DeleteModal from '~/app/shared/components/DeleteModal';
import { useNotification } from '~/app/hooks/useNotification';
import SkillCatalogSourceStatus from '~/app/pages/skillCatalogSettings/components/SkillCatalogSourceStatus';

const SKILL_SOURCE_TYPE_LABELS: Record<string, string> = {
  'git-skills-plugin': 'Git repository',
};

type SkillCatalogSourceConfigsTableRowProps = {
  skillCatalogSourceConfig: SkillCatalogSourceConfig;
  onDeleteSource: (sourceId: string) => Promise<void>;
  isUpdatingToggle: boolean;
  onToggleUpdate: (checked: boolean, sourceConfig: SkillCatalogSourceConfig) => void;
};

const SkillCatalogSourceConfigsTableRow: React.FC<SkillCatalogSourceConfigsTableRowProps> = ({
  skillCatalogSourceConfig,
  onDeleteSource,
  isUpdatingToggle,
  onToggleUpdate,
}) => {
  const navigate = useNavigate();
  const notification = useNotification();
  const [isDeleteModalOpen, setIsDeleteModalOpen] = React.useState(false);
  const [isDeleting, setIsDeleting] = React.useState(false);
  const [deleteError, setDeleteError] = React.useState<Error | undefined>();

  const isDefault = skillCatalogSourceConfig.isDefault ?? false;
  const isEnabled = skillCatalogSourceConfig.enabled ?? true;

  const handleEnableToggle = (checked: boolean) => {
    onToggleUpdate(checked, skillCatalogSourceConfig);
  };

  const handleManageSource = () => {
    navigate(skillManageSourceUrl(skillCatalogSourceConfig.id));
  };

  const handleDeleteClick = () => {
    setDeleteError(undefined);
    setIsDeleteModalOpen(true);
  };

  const handleDeleteConfirm = async () => {
    setIsDeleting(true);
    setDeleteError(undefined);

    try {
      await onDeleteSource(skillCatalogSourceConfig.id);
      setIsDeleteModalOpen(false);
      notification.success(`${skillCatalogSourceConfig.name} deleted successfully`);
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
          <span data-testid={`skill-source-name-${skillCatalogSourceConfig.id}`}>
            {skillCatalogSourceConfig.name}
          </span>
        </Td>
        <Td dataLabel="Source type" style={{ verticalAlign: 'middle' }}>
          <span data-testid={`skill-source-type-${skillCatalogSourceConfig.id}`}>
            {SKILL_SOURCE_TYPE_LABELS[skillCatalogSourceConfig.type] ??
              skillCatalogSourceConfig.type}
          </span>
        </Td>
        <Td dataLabel="Enable" style={{ verticalAlign: 'middle' }}>
          <Switch
            data-testid={`skill-enable-toggle-${skillCatalogSourceConfig.id}`}
            id={`skill-enable-toggle-${skillCatalogSourceConfig.id}`}
            aria-label={`Enable ${skillCatalogSourceConfig.name}`}
            isChecked={isEnabled}
            isDisabled={isUpdatingToggle}
            onChange={(_event, checked) => handleEnableToggle(checked)}
          />
        </Td>
        <Td dataLabel="Validation status" style={{ verticalAlign: 'middle' }}>
          <SkillCatalogSourceStatus skillCatalogSourceConfig={skillCatalogSourceConfig} />
        </Td>
        <Td style={{ verticalAlign: 'middle' }}>
          {/* Shipped defaults are read-only: they expose no edit or delete controls. */}
          {!isDefault && (
            <Button
              variant="link"
              onClick={handleManageSource}
              data-testid={`skill-manage-source-button-${skillCatalogSourceConfig.id}`}
            >
              Manage source
            </Button>
          )}
        </Td>
        <Td isActionCell style={{ verticalAlign: 'middle' }}>
          {!isDefault && (
            <ActionsColumn
              items={[{ title: 'Delete source', onClick: handleDeleteClick }]}
              data-testid={`skill-source-actions-${skillCatalogSourceConfig.id}`}
            />
          )}
        </Td>
      </Tr>
      {isDeleteModalOpen && (
        <DeleteModal
          title="Delete a source"
          testId="skill-delete-source-modal"
          onClose={handleCloseDeleteModal}
          deleting={isDeleting}
          onDelete={handleDeleteConfirm}
          deleteName={skillCatalogSourceConfig.name}
          error={deleteError}
        >
          The <strong>{skillCatalogSourceConfig.name}</strong> source will be deleted, and its
          skills will be removed from the skill catalog.
        </DeleteModal>
      )}
    </>
  );
};

export default SkillCatalogSourceConfigsTableRow;
