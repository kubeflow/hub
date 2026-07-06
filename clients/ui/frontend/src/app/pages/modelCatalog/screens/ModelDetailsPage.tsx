import React from 'react';
import { useParams, useNavigate } from 'react-router';
import { Link } from 'react-router-dom';
import {
  ActionList,
  Breadcrumb,
  BreadcrumbItem,
  Content,
  ContentVariants,
  Flex,
  FlexItem,
  Stack,
  StackItem,
  Button,
  Popover,
  Tooltip,
  ActionListGroup,
  Skeleton,
  Label,
} from '@patternfly/react-core';
import { CheckCircleIcon } from '@patternfly/react-icons';
import { ApplicationsPage } from 'mod-arch-shared';
import {
  decodeParams,
  getModelName,
  hasModelArtifacts,
  isModelValidated,
} from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import { useCatalogModel } from '~/app/hooks/modelCatalog/useCatalogModel';
import { ModelRegistrySelectorContext } from '~/app/context/ModelRegistrySelectorContext';
import { getRegisterCatalogModelRoute } from '~/app/routes/modelCatalog/catalogModelRegister';
import { CatalogModelDetailsParams } from '~/app/modelCatalogTypes';
import { useCatalogModelArtifacts } from '~/app/hooks/modelCatalog/useCatalogModelArtifacts';
import { modelCatalogUrl } from '~/app/routes/modelCatalog/catalogModel';
import ScrollViewOnMount from '~/app/shared/components/ScrollViewOnMount';
import { ModelDetailsTab, MODEL_CATALOG_POPOVER_MESSAGES } from '~/concepts/modelCatalog/const';
import { MODEL_CATALOG_TITLE } from '~/app/pages/modelCatalog/const';
import ModelDetailsTabs from './ModelDetailsTabs';

type ModelDetailsPageProps = {
  tab: ModelDetailsTab;
};

const ModelDetailsPage: React.FC<ModelDetailsPageProps> = ({ tab }) => {
  const params = useParams<CatalogModelDetailsParams>();
  const decodedParams = decodeParams(params);
  const navigate = useNavigate();
  const state = useCatalogModel(
    decodedParams.sourceId || '',
    encodeURIComponent(`${decodedParams.modelName}`),
  );
  const [model, modelLoaded, modelLoadError] = state;
  const { modelRegistries, modelRegistriesLoadError, modelRegistriesLoaded } = React.useContext(
    ModelRegistrySelectorContext,
  );

  const [artifacts, artifactLoaded, artifactsLoadError] = useCatalogModelArtifacts(
    decodedParams.sourceId || '',
    encodeURIComponent(`${decodedParams.modelName}`),
  );

  const registerButtonTooltip = (headerContent: string, bodyContent: string) => (
    <Tooltip
      content={
        headerContent ? (
          <div>
            <strong>{headerContent}</strong>
            <div>{bodyContent}</div>
          </div>
        ) : (
          bodyContent
        )
      }
      data-testid="register-catalog-model-tooltip"
    >
      <Button variant="primary" isAriaDisabled data-testid="register-model-button">
        Register model
      </Button>
    </Tooltip>
  );

  const registerModelButton = () => {
    if (!modelRegistriesLoaded || modelRegistriesLoadError) {
      return null;
    }

    if (artifactsLoadError) {
      return registerButtonTooltip(
        'Unable to load model artifacts',
        'Model registration is unavailable due to an error loading model artifacts. Please try again later.',
      );
    }

    if (!artifactLoaded) {
      return (
        <Button variant="primary" data-testid="register-model-button" isLoading isAriaDisabled>
          Register model
        </Button>
      );
    }

    return modelRegistries.length === 0 ? (
      registerButtonTooltip(
        'Request access to a model registry',
        'To request a new model registry, or to request permission to access an existing model registry, contact your administrator.',
      )
    ) : artifacts.items.length === 0 || !hasModelArtifacts(artifacts.items) ? (
      registerButtonTooltip('', 'Model location is unavailable')
    ) : (
      <Button
        data-testid="register-model-button"
        variant="primary"
        onClick={() => {
          navigate(getRegisterCatalogModelRoute(decodedParams.sourceId, decodedParams.modelName));
        }}
      >
        Register model
      </Button>
    );
  };

  return (
    <>
      <ScrollViewOnMount shouldScroll scrollToTop />
      <ApplicationsPage
        breadcrumb={
          <Breadcrumb>
            <BreadcrumbItem>
              <Link to={modelCatalogUrl()}>{MODEL_CATALOG_TITLE}</Link>
            </BreadcrumbItem>
            <BreadcrumbItem isActive>{getModelName(model?.name || '') || 'Details'}</BreadcrumbItem>
          </Breadcrumb>
        }
        title={
          model ? (
            <Flex
              spaceItems={{ default: 'spaceItemsMd' }}
              alignItems={{ default: 'alignItemsCenter' }}
            >
              {model.logo ? (
                <img src={model.logo} alt="model logo" style={{ height: '56px', width: '56px' }} />
              ) : (
                <Skeleton
                  shape="square"
                  width="56px"
                  height="56px"
                  screenreaderText="Brand image loading"
                />
              )}
              <Stack>
                <StackItem>
                  <Flex
                    spaceItems={{ default: 'spaceItemsMd' }}
                    alignItems={{ default: 'alignItemsCenter' }}
                  >
                    <FlexItem>{getModelName(model.name)}</FlexItem>
                    {isModelValidated(model) && (
                      <Popover bodyContent={MODEL_CATALOG_POPOVER_MESSAGES.VALIDATED}>
                        <Label
                          variant="outline"
                          isClickable
                          status="success"
                          icon={<CheckCircleIcon />}
                        >
                          Validated
                        </Label>
                      </Popover>
                    )}
                  </Flex>
                </StackItem>
                <StackItem>
                  <Content component={ContentVariants.small}>Provided by {model.provider}</Content>
                </StackItem>
              </Stack>
            </Flex>
          ) : null
        }
        empty={!model}
        emptyStatePage={
          !model ? (
            <div>
              Details not found. Return to <Link to={modelCatalogUrl()}>{MODEL_CATALOG_TITLE}</Link>
            </div>
          ) : undefined
        }
        loadError={modelLoadError}
        loaded={modelLoaded}
        errorMessage="Unable to load model catalog"
        provideChildrenPadding
        headerAction={
          modelLoaded &&
          !modelLoadError &&
          model && (
            <ActionList>
              <ActionListGroup>{registerModelButton()}</ActionListGroup>
            </ActionList>
          )
        }
      >
        {model && (
          <ModelDetailsTabs
            model={model}
            tab={tab}
            artifacts={artifacts}
            artifactLoaded={artifactLoaded}
            artifactsLoadError={artifactsLoadError}
          />
        )}
      </ApplicationsPage>
    </>
  );
};

export default ModelDetailsPage;
