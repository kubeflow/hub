import * as React from 'react';
import { useParams, Link } from 'react-router-dom';
import { Breadcrumb, BreadcrumbItem } from '@patternfly/react-core';
import { ApplicationsPage } from 'mod-arch-shared';
import {
  SKILL_ADD_SOURCE_TITLE,
  SKILL_ADD_SOURCE_DESCRIPTION,
  SKILL_MANAGE_SOURCE_TITLE,
  SKILL_MANAGE_SOURCE_DESCRIPTION,
  skillCatalogSettingsUrl,
} from '~/app/routes/skillCatalogSettings/skillCatalogSettings';
import { useSkillCatalogSourceConfigBySourceId } from '~/app/hooks/skillCatalogSettings/useSkillCatalogSourceConfigBySourceId';
import SkillManageSourceForm from '~/app/pages/skillCatalogSettings/components/SkillManageSourceForm';

const SKILL_CATALOG_SOURCES_BREADCRUMB = 'Skill catalog sources';

const SkillManageSourcePage: React.FC = () => {
  const { catalogSourceId } = useParams<{ catalogSourceId?: string }>();
  const isAddMode = !catalogSourceId;
  const pageTitle = isAddMode ? SKILL_ADD_SOURCE_TITLE : SKILL_MANAGE_SOURCE_TITLE;
  const description = isAddMode ? SKILL_ADD_SOURCE_DESCRIPTION : SKILL_MANAGE_SOURCE_DESCRIPTION;

  const [existingSourceConfig, existingSourceConfigLoaded, existingSourceConfigLoadError] =
    useSkillCatalogSourceConfigBySourceId(catalogSourceId || '');

  return (
    <ApplicationsPage
      breadcrumb={
        <Breadcrumb>
          <BreadcrumbItem>
            <Link to={skillCatalogSettingsUrl()}>{SKILL_CATALOG_SOURCES_BREADCRUMB}</Link>
          </BreadcrumbItem>
          <BreadcrumbItem data-testid="skill-breadcrumb-source-action" isActive>
            {pageTitle}
          </BreadcrumbItem>
        </Breadcrumb>
      }
      title={pageTitle}
      description={description}
      errorMessage={catalogSourceId ? existingSourceConfigLoadError?.message : undefined}
      empty={catalogSourceId ? !existingSourceConfig : false}
      loaded={catalogSourceId ? existingSourceConfigLoaded : true}
      provideChildrenPadding
    >
      <SkillManageSourceForm
        existingSourceConfig={existingSourceConfig || undefined}
        isEditMode={!isAddMode}
      />
    </ApplicationsPage>
  );
};

export default SkillManageSourcePage;
