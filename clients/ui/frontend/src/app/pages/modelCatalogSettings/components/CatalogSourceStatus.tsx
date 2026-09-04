import * as React from 'react';
import { CatalogSourceStatus as SharedCatalogSourceStatus } from '~/app/shared/catalogSettings';
import { CatalogSourceConfig } from '~/app/modelCatalogTypes';
import { ModelCatalogSettingsContext } from '~/app/context/modelCatalogSettings/ModelCatalogSettingsContext';

type CatalogSourceStatusProps = {
  catalogSourceConfig: CatalogSourceConfig;
};

const CatalogSourceStatus: React.FC<CatalogSourceStatusProps> = ({ catalogSourceConfig }) => {
  const { catalogSources, catalogSourcesLoaded, catalogSourcesLoadError, pendingSourceIds } =
    React.useContext(ModelCatalogSettingsContext);

  return (
    <SharedCatalogSourceStatus
      catalogSourceConfig={catalogSourceConfig}
      catalogSources={catalogSources}
      catalogSourcesLoaded={catalogSourcesLoaded}
      catalogSourcesLoadError={catalogSourcesLoadError}
      pendingSourceIds={pendingSourceIds}
    />
  );
};

export default CatalogSourceStatus;
