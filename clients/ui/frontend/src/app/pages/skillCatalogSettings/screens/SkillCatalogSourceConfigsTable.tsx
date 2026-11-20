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
import type { SkillCatalogSourceConfig } from '~/app/skillCatalogTypes';
import { SkillCatalogSettingsContext } from '~/app/context/skillCatalogSettings/SkillCatalogSettingsContext';
import { SKILL_ADD_SOURCE_TITLE } from '~/app/routes/skillCatalogSettings/skillCatalogSettings';
import { skillCatalogSourceConfigsColumns } from './SkillCatalogSourceConfigsTableColumns';
import SkillCatalogSourceConfigsTableRow from './SkillCatalogSourceConfigsTableRow';

type SkillCatalogSourceConfigsTableProps = {
  skillCatalogSourceConfigs: SkillCatalogSourceConfig[];
  onAddSource: () => void;
  onDeleteSource: (sourceId: string) => Promise<void>;
};

const SkillCatalogSourceConfigsTable: React.FC<SkillCatalogSourceConfigsTableProps> = ({
  skillCatalogSourceConfigs,
  onAddSource,
  onDeleteSource,
}) => {
  const [toggleError, setToggleError] = React.useState<Error | undefined>(undefined);
  const [updatingToggleId, setUpdatingToggleId] = React.useState<string | null>(null);
  const {
    apiState,
    refreshSkillCatalogSourceConfigs,
    refreshSkillCatalogSources,
    markSkillSourceSyncPending,
  } = React.useContext(SkillCatalogSettingsContext);

  const handleEnableToggle = async (
    checked: boolean,
    catalogSourceConfig: SkillCatalogSourceConfig,
  ) => {
    if (!apiState.apiAvailable) {
      setToggleError(new Error('API is not available'));
      return;
    }
    setUpdatingToggleId(catalogSourceConfig.id);
    setToggleError(undefined);

    try {
      await apiState.api.updateSkillCatalogSourceConfig({}, catalogSourceConfig.id, {
        enabled: checked,
      });
      markSkillSourceSyncPending(catalogSourceConfig.id);
      refreshSkillCatalogSourceConfigs();
      refreshSkillCatalogSources();
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
      <StackItem>
        <Table
          data-testid="skill-catalog-source-configs-table"
          data={skillCatalogSourceConfigs}
          columns={skillCatalogSourceConfigsColumns}
          toolbarContent={
            <Flex direction={{ default: 'column' }}>
              <FlexItem>
                <Toolbar>
                  <ToolbarContent>
                    <ToolbarItem>
                      <Button
                        variant="primary"
                        onClick={onAddSource}
                        data-testid="skill-add-source-button"
                      >
                        {SKILL_ADD_SOURCE_TITLE}
                      </Button>
                    </ToolbarItem>
                  </ToolbarContent>
                </Toolbar>
              </FlexItem>
              {toggleError && (
                <FlexItem>
                  <Alert
                    variant="danger"
                    data-testid="skill-toggle-alert"
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
            <SkillCatalogSourceConfigsTableRow
              key={config.id}
              skillCatalogSourceConfig={config}
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

export default SkillCatalogSourceConfigsTable;
