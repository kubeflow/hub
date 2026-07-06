import { Alert, Bullseye } from '@patternfly/react-core';
import { useThemeContext } from 'mod-arch-kubeflow';
import {
  ApplicationsPage,
  KubeflowDocs,
  ProjectObjectType,
  TitleWithIcon,
  typedEmptyImage,
  WhosMyAdministrator,
} from 'mod-arch-shared';
import * as React from 'react';
import { Outlet } from 'react-router-dom';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import { EmptyCatalogState, hasSourcesWithModels } from '~/app/shared/components/catalog';
import { MODEL_CATALOG_TITLE, MODEL_CATALOG_DESCRIPTION } from '~/app/pages/modelCatalog/const';

const ModelCatalogCoreLoader: React.FC = () => {
  const { catalogSources, catalogSourcesLoaded, catalogSourcesLoadError } =
    React.useContext(ModelCatalogContext);

  const { isMUITheme } = useThemeContext();

  if (catalogSourcesLoadError) {
    return (
      <ApplicationsPage
        title={
          <TitleWithIcon title={MODEL_CATALOG_TITLE} objectType={ProjectObjectType.modelCatalog} />
        }
        description={MODEL_CATALOG_DESCRIPTION}
        headerContent={null}
        empty
        emptyStatePage={
          <Bullseye>
            <Alert title="Model catalog source load error" variant="danger" isInline>
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
          <TitleWithIcon title={MODEL_CATALOG_TITLE} objectType={ProjectObjectType.modelCatalog} />
        }
        description={MODEL_CATALOG_DESCRIPTION}
        headerContent={null}
        empty
        emptyStatePage={<Bullseye>Loading catalog sources...</Bullseye>}
        loaded={false}
      />
    );
  }
  // Show empty state if there are no sources, or if all sources have no models (e.g., disabled)
  if (catalogSources?.items?.length === 0 || !hasSourcesWithModels(catalogSources)) {
    return (
      <ApplicationsPage
        title={
          <TitleWithIcon title={MODEL_CATALOG_TITLE} objectType={ProjectObjectType.modelCatalog} />
        }
        description={MODEL_CATALOG_DESCRIPTION}
        empty
        emptyStatePage={
          <EmptyCatalogState
            testid="empty-model-catalog-state"
            title={isMUITheme ? 'Deploy a model catalog' : 'Model catalog configuration required'}
            description={
              isMUITheme
                ? 'To deploy model catalog, follow the instructions in the docs below.'
                : 'There are no models to display. Request that your administrator configure model sources for the catalog.'
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

export default ModelCatalogCoreLoader;
