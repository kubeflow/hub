import * as React from 'react';
import { useParams } from 'react-router-dom';
import { ManageSourcePageShell } from '~/app/shared/catalogSettings';
import {
  CATALOG_SETTINGS_PAGE_TITLE,
  ADD_SOURCE_TITLE,
  ADD_SOURCE_DESCRIPTION,
  MANAGE_SOURCE_TITLE,
  MANAGE_SOURCE_DESCRIPTION,
  catalogSettingsUrl,
} from '~/app/routes/modelCatalogSettings/modelCatalogSettings';
import ManageSourceForm from '~/app/pages/modelCatalogSettings/components/ManageSourceForm';
import { ExpectedYamlFormatDrawerPanel } from '~/app/pages/modelCatalogSettings/components/ExpectedYamlFormatDrawer';
import { useCatalogSourceConfigBySourceId } from '~/app/hooks/modelCatalogSettings/useCatalogSourceConfigBySourceId';

const ManageSourcePage: React.FC = () => {
  const { catalogSourceId } = useParams<{ catalogSourceId?: string }>();
  const isAddMode = !catalogSourceId;
  const pageTitle = isAddMode ? ADD_SOURCE_TITLE : MANAGE_SOURCE_TITLE;
  const breadcrumbLabel = isAddMode ? ADD_SOURCE_TITLE : MANAGE_SOURCE_TITLE;
  const description = isAddMode ? ADD_SOURCE_DESCRIPTION : MANAGE_SOURCE_DESCRIPTION;

  const state = useCatalogSourceConfigBySourceId(catalogSourceId || '');
  const [existingSourceConfig, existingSourceConfigLoaded, existingSourceConfigLoadError] = state;
  const [isExpectedFormatDrawerOpen, setIsExpectedFormatDrawerOpen] = React.useState(false);

  return (
    <ManageSourcePageShell
      listPageUrl={catalogSettingsUrl()}
      listPageLabel={CATALOG_SETTINGS_PAGE_TITLE}
      breadcrumbLabel={breadcrumbLabel}
      breadcrumbTestId="breadcrumb-source-action"
      title={pageTitle}
      description={description}
      errorMessage={catalogSourceId ? existingSourceConfigLoadError?.message : undefined}
      empty={catalogSourceId ? !existingSourceConfig : false}
      loaded={catalogSourceId ? existingSourceConfigLoaded : true}
      isExpectedFormatDrawerOpen={isExpectedFormatDrawerOpen}
      drawerPanelContent={
        <ExpectedYamlFormatDrawerPanel onClose={() => setIsExpectedFormatDrawerOpen(false)} />
      }
    >
      <ManageSourceForm
        existingSourceConfig={existingSourceConfig || undefined}
        isEditMode={!isAddMode}
        onToggleExpectedFormatDrawer={() => setIsExpectedFormatDrawerOpen((prev) => !prev)}
      />
    </ManageSourcePageShell>
  );
};

export default ManageSourcePage;
