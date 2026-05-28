import React from 'react';
import {
  Button,
  Card,
  CardBody,
  CardFooter,
  CardHeader,
  CardTitle,
  Flex,
  FlexItem,
  Label,
  Popover,
  Skeleton,
} from '@patternfly/react-core';
import { CheckCircleIcon } from '@patternfly/react-icons';
import { Link } from 'react-router-dom';
import { CatalogModel, CatalogSource } from '~/app/modelCatalogTypes';
import { catalogModelDetailsFromModel } from '~/app/routes/modelCatalog/catalogModel';
import { getLabels } from '~/app/pages/modelRegistry/screens/utils';
import { isModelValidated, getModelName } from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import { MODEL_CATALOG_POPOVER_MESSAGES } from '~/concepts/modelCatalog/const';
import ModelCatalogLabels from './ModelCatalogLabels';
import ModelCatalogCardBody from './ModelCatalogCardBody';

type ModelCatalogCardProps = {
  model: CatalogModel;
  source: CatalogSource | undefined;
};

const ModelCatalogCard: React.FC<ModelCatalogCardProps> = ({ model, source }) => {
  const allLabels = model.customProperties ? getLabels(model.customProperties) : [];
  const isValidated = isModelValidated(model);

  return (
    <Card isFullHeight data-testid="model-catalog-card" key={`${model.name}/${model.source_id}`}>
      <CardHeader>
        <CardTitle>
          <Flex alignItems={{ default: 'alignItemsFlexStart' }} className="pf-v6-u-mb-md">
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
            <FlexItem align={{ default: 'alignRight' }}>
              {isValidated ? (
                <Popover bodyContent={MODEL_CATALOG_POPOVER_MESSAGES.VALIDATED}>
                  <Label variant="outline" isClickable status="success" icon={<CheckCircleIcon />}>
                    Validated
                  </Label>
                </Popover>
              ) : (
                source && <Label>{source.name}</Label>
              )}
            </FlexItem>
          </Flex>
          <Link to={catalogModelDetailsFromModel(model.name, source?.id)}>
            <Button
              data-testid="model-catalog-detail-link"
              variant="link"
              tabIndex={-1}
              isInline
              style={{
                fontSize: 'var(--pf-t--global--font--size--body--default)',
                fontWeight: 'var(--pf-t--global--font--weight--body--bold)',
              }}
            >
              <span data-testid="model-catalog-card-name">{getModelName(model.name)}</span>
            </Button>
          </Link>
        </CardTitle>
      </CardHeader>
      <CardBody>
        <ModelCatalogCardBody model={model} isValidated={isValidated} source={source} />
      </CardBody>
      <CardFooter>
        <ModelCatalogLabels
          tasks={model.tasks ?? []}
          validatedTasks={model.validatedTasks}
          provider={model.provider}
          labels={allLabels.filter((label) => label !== 'validated')}
          numLabels={isValidated ? 2 : 3}
        />
      </CardFooter>
    </Card>
  );
};

export default ModelCatalogCard;
