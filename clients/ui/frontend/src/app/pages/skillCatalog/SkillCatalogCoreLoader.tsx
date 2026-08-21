import * as React from 'react';
import { Alert, Bullseye } from '@patternfly/react-core';
import {
  ApplicationsPage,
  KubeflowDocs,
  ProjectObjectType,
  TitleWithIcon,
  typedEmptyImage,
  WhosMyAdministrator,
} from 'mod-arch-shared';
import { useThemeContext } from 'mod-arch-kubeflow';
import { Outlet } from 'react-router-dom';
import { SkillCatalogContext } from '~/app/context/skillCatalog/SkillCatalogContext';
import { EmptyCatalogState, hasSourcesWithModels } from '~/app/shared/components/catalog';
import { SKILL_CATALOG_TITLE, SKILL_CATALOG_DESCRIPTION } from '~/app/pages/skillCatalog/const';

const SkillCatalogCoreLoader: React.FC = () => {
  const { catalogSources, catalogSourcesLoaded, catalogSourcesLoadError } =
    React.useContext(SkillCatalogContext);
  const { isMUITheme } = useThemeContext();

  if (catalogSourcesLoadError) {
    return (
      <ApplicationsPage
        title={
          <TitleWithIcon title={SKILL_CATALOG_TITLE} objectType={ProjectObjectType.mcpCatalog} />
        }
        description={SKILL_CATALOG_DESCRIPTION}
        headerContent={null}
        empty
        emptyStatePage={
          <Bullseye>
            <Alert title="Skill catalog source load error" variant="danger" isInline>
              {catalogSourcesLoadError.message}
            </Alert>
          </Bullseye>
        }
        loaded
      />
    );
  }

  if (!catalogSourcesLoaded) {
    return (
      <ApplicationsPage
        title={
          <TitleWithIcon title={SKILL_CATALOG_TITLE} objectType={ProjectObjectType.mcpCatalog} />
        }
        description={SKILL_CATALOG_DESCRIPTION}
        headerContent={null}
        empty
        emptyStatePage={<Bullseye>Loading catalog sources...</Bullseye>}
        loaded={false}
      />
    );
  }

  if (catalogSources?.items?.length === 0 || !hasSourcesWithModels(catalogSources)) {
    return (
      <ApplicationsPage
        title={
          <TitleWithIcon title={SKILL_CATALOG_TITLE} objectType={ProjectObjectType.mcpCatalog} />
        }
        description={SKILL_CATALOG_DESCRIPTION}
        empty
        emptyStatePage={
          <EmptyCatalogState
            testid="empty-skill-catalog-state"
            title="Skill catalog configuration required"
            description={
              isMUITheme
                ? 'To discover skills, follow the instructions in the docs below.'
                : 'There are no skill sources to display. Request that your administrator configure skill sources for the catalog.'
            }
            headerIcon={() => (
              <img src={typedEmptyImage(ProjectObjectType.modelRegistrySettings)} alt="" />
            )}
            primaryAction={isMUITheme ? <KubeflowDocs /> : <WhosMyAdministrator />}
          />
        }
        headerContent={null}
        loaded
        provideChildrenPadding
      />
    );
  }

  return <Outlet />;
};

export default SkillCatalogCoreLoader;
