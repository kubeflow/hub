import * as React from 'react';
import { Breadcrumb, BreadcrumbItem } from '@patternfly/react-core';
import { Link, useParams } from 'react-router-dom';
import { ApplicationsPage } from 'mod-arch-shared';
import { ModelRegistryContextProvider } from '~/app/context/ModelRegistryContext';
import {
  ModelRegistrySelectorContextProvider,
  ModelRegistrySelectorContext,
} from '~/app/context/ModelRegistrySelectorContext';
import { useCatalogModel } from '~/app/hooks/modelCatalog/useCatalogModel';
import { CatalogModelDetailsParams } from '~/app/modelCatalogTypes';
import { useCatalogModelArtifacts } from '~/app/hooks/modelCatalog/useCatalogModelArtifacts';
import { getCatalogModelDetailsRoute } from '~/app/routes/modelCatalog/catalogModelDetails';
import { decodeParams, getModelName } from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import { modelCatalogUrl } from '~/app/routes/modelCatalog/catalogModel';
import { MODEL_CATALOG_TITLE } from '~/app/pages/modelCatalog/const';
import RegisterCatalogModelForm from './RegisterCatalogModelForm';

const RegisterCatalogModelPageInner: React.FC = () => {
  const params = useParams<CatalogModelDetailsParams>();
  const decodedParams = decodeParams(params);
  const { modelRegistries, modelRegistriesLoaded } = React.useContext(ModelRegistrySelectorContext);

  const state = useCatalogModel(
    decodedParams.sourceId || '',
    encodeURIComponent(`${decodedParams.modelName}`),
  );
  const [model, modelLoaded, modelLoadError] = state;
  const [artifacts, artifactLoaded, artifactsLoadError] = useCatalogModelArtifacts(
    decodedParams.sourceId || '',
    encodeURIComponent(`${decodedParams.modelName}`),
  );

  const preferredModelRegistry = modelRegistries.length > 0 ? modelRegistries[0] : null;

  // Check to see if data is loaded
  const isDataReady =
    modelLoaded &&
    artifactLoaded &&
    !artifactsLoadError &&
    !modelLoadError &&
    model &&
    modelRegistriesLoaded &&
    modelRegistries.length > 0;

  return (
    <ApplicationsPage
      title={`Register ${getModelName(model?.name || '') || ''} model`}
      description="Create and register the first version of a new model."
      breadcrumb={
        <Breadcrumb>
          <BreadcrumbItem
            render={() => <Link to={modelCatalogUrl()}>{MODEL_CATALOG_TITLE}</Link>}
          />
          <BreadcrumbItem
            data-testid="breadcrumb-model-name"
            render={() =>
              !model?.name ? (
                'Loading...'
              ) : (
                <Link
                  to={getCatalogModelDetailsRoute({
                    sourceId: decodedParams.sourceId,
                    modelName: decodedParams.modelName,
                  })}
                >
                  {getModelName(model.name)}
                </Link>
              )
            }
          />
          <BreadcrumbItem data-testid="breadcrumb-version-name" isActive>
            Register model
          </BreadcrumbItem>
        </Breadcrumb>
      }
      loaded={modelLoaded}
      loadError={modelLoadError}
      empty={false}
    >
      {isDataReady && preferredModelRegistry ? (
        <ModelRegistryContextProvider modelRegistryName={preferredModelRegistry.name}>
          <RegisterCatalogModelForm
            model={model}
            preferredModelRegistry={preferredModelRegistry}
            artifacts={artifacts.items}
            decodedParams={decodedParams}
            removeChildrenTopPadding
          />
        </ModelRegistryContextProvider>
      ) : (
        <div>Loading...</div>
      )}
    </ApplicationsPage>
  );
};

const RegisterCatalogModelPage: React.FC = () => (
  <ModelRegistrySelectorContextProvider>
    <RegisterCatalogModelPageInner />
  </ModelRegistrySelectorContextProvider>
);

export default RegisterCatalogModelPage;
