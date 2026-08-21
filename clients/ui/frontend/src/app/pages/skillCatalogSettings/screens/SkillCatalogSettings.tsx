import * as React from 'react';
import { Button, EmptyState, EmptyStateBody, EmptyStateVariant } from '@patternfly/react-core';
import { PlusCircleIcon } from '@patternfly/react-icons';
import { useNavigate } from 'react-router-dom';
import { ApplicationsPage, ProjectObjectType, TitleWithIcon } from 'mod-arch-shared';
import {
  SKILL_CATALOG_SETTINGS_PAGE_TITLE,
  SKILL_CATALOG_SETTINGS_DESCRIPTION,
  skillAddSourceUrl,
  SKILL_ADD_SOURCE_TITLE,
} from '~/app/routes/skillCatalogSettings/skillCatalogSettings';
import { SkillCatalogSettingsContext } from '~/app/context/skillCatalogSettings/SkillCatalogSettingsContext';
import SkillCatalogSourceConfigsTable from './SkillCatalogSourceConfigsTable';

const SkillCatalogSettings: React.FC = () => {
  const navigate = useNavigate();
  const {
    skillCatalogSourceConfigs,
    skillCatalogSourceConfigsLoaded,
    skillCatalogSourceConfigsLoadError,
    apiState,
    refreshSkillCatalogSourceConfigs,
  } = React.useContext(SkillCatalogSettingsContext);

  const configs = skillCatalogSourceConfigs?.catalogs || [];
  const isEmpty = skillCatalogSourceConfigsLoaded && configs.length === 0;

  const handleDeleteSource = React.useCallback(
    async (sourceId: string): Promise<void> => {
      if (!apiState.apiAvailable) {
        throw new Error('API not available');
      }
      await apiState.api.deleteSkillCatalogSourceConfig({}, sourceId);
      refreshSkillCatalogSourceConfigs();
    },
    [apiState.api, apiState.apiAvailable, refreshSkillCatalogSourceConfigs],
  );

  return (
    <ApplicationsPage
      title={
        <TitleWithIcon
          title={SKILL_CATALOG_SETTINGS_PAGE_TITLE}
          objectType={ProjectObjectType.mcpCatalog}
        />
      }
      description={SKILL_CATALOG_SETTINGS_DESCRIPTION}
      empty={isEmpty}
      emptyStatePage={
        <EmptyState
          headingLevel="h5"
          icon={PlusCircleIcon}
          titleText="No skill sources"
          variant={EmptyStateVariant.lg}
          data-testid="skill-catalog-settings-empty-state"
        >
          <EmptyStateBody>
            No skill sources have been configured. Add a source to get started.
          </EmptyStateBody>
          <Button
            variant="primary"
            onClick={() => navigate(skillAddSourceUrl())}
            data-testid="skill-add-source-button-empty"
          >
            {SKILL_ADD_SOURCE_TITLE}
          </Button>
        </EmptyState>
      }
      loaded={skillCatalogSourceConfigsLoaded}
      loadError={skillCatalogSourceConfigsLoadError}
      errorMessage="Unable to load skill catalog source configurations."
      provideChildrenPadding
    >
      <SkillCatalogSourceConfigsTable
        skillCatalogSourceConfigs={configs}
        onAddSource={() => navigate(skillAddSourceUrl())}
        onDeleteSource={handleDeleteSource}
      />
    </ApplicationsPage>
  );
};

export default SkillCatalogSettings;
