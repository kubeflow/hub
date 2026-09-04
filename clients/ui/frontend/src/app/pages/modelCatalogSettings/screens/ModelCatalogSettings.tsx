import * as React from 'react';
import { useNavigate } from 'react-router-dom';
import { ProjectObjectType, TitleWithIcon } from 'mod-arch-shared';
import { CatalogSettingsListPage } from '~/app/shared/catalogSettings';
import {
  CATALOG_SETTINGS_PAGE_TITLE,
  CATALOG_SETTINGS_DESCRIPTION,
  addSourceUrl,
  ADD_SOURCE_TITLE,
} from '~/app/routes/modelCatalogSettings/modelCatalogSettings';
import { ModelCatalogSettingsContext } from '~/app/context/modelCatalogSettings/ModelCatalogSettingsContext';
import CatalogSourceConfigsTable from './CatalogSourceConfigsTable';

const ModelCatalogSettings: React.FC = () => {
  const navigate = useNavigate();
  const {
    catalogSourceConfigs,
    catalogSourceConfigsLoaded,
    catalogSourceConfigsLoadError,
    apiState,
    refreshCatalogSourceConfigs,
    refreshCatalogSources,
  } = React.useContext(ModelCatalogSettingsContext);

  const configs = catalogSourceConfigs?.catalogs || [];
  const isEmpty = catalogSourceConfigsLoaded && configs.length === 0;

  const handleAddSource = React.useCallback(() => {
    navigate(addSourceUrl());
  }, [navigate]);

  const handleDeleteSource = React.useCallback(
    async (sourceId: string): Promise<void> => {
      if (!apiState.apiAvailable) {
        throw new Error('API not available');
      }
      await apiState.api.deleteCatalogSourceConfig({}, sourceId);
      refreshCatalogSourceConfigs();
      refreshCatalogSources();
    },
    [apiState.api, apiState.apiAvailable, refreshCatalogSourceConfigs, refreshCatalogSources],
  );

  return (
    <CatalogSettingsListPage
      title={
        <TitleWithIcon
          title={CATALOG_SETTINGS_PAGE_TITLE}
          objectType={ProjectObjectType.modelCatalog}
        />
      }
      description={CATALOG_SETTINGS_DESCRIPTION}
      isEmpty={isEmpty}
      loaded={catalogSourceConfigsLoaded}
      loadError={catalogSourceConfigsLoadError}
      errorMessage="Unable to load catalog source configurations."
      emptyStateTitle="No catalog sources"
      emptyStateBody="No catalog sources have been configured. Add a source to get started."
      emptyStateTestId="catalog-settings-empty-state"
      addSourceLabel={ADD_SOURCE_TITLE}
      addSourceButtonTestId="add-source-button-empty"
      onAddSource={handleAddSource}
    >
      <CatalogSourceConfigsTable
        catalogSourceConfigs={configs}
        onAddSource={handleAddSource}
        onDeleteSource={handleDeleteSource}
      />
    </CatalogSettingsListPage>
  );
};

export default ModelCatalogSettings;
