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
import type { CatalogSource } from '~/app/shared/types/catalogTypes';
import type { CatalogModel } from '~/app/modelCatalogTypes';
import { catalogModelDetailsFromModel } from '~/app/routes/modelCatalog/catalogModel';
import { getLabels, getValueLabels } from '~/app/pages/modelRegistry/screens/utils';
import {
  isModelValidated,
  getModelName,
  isHfGatedAccessDenied,
  getHfAccessLabelVariant,
} from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import {
  MODEL_CATALOG_POPOVER_MESSAGES,
  CATALOG_VALUE_LABEL_KEYS,
} from '~/concepts/modelCatalog/const';
import ModelCatalogLabels from './ModelCatalogLabels';
import ModelCatalogCardBody from './ModelCatalogCardBody';
import ModelCatalogAccessLabel from './ModelCatalogAccessLabel';

type ModelCatalogCardProps = {
  model: CatalogModel;
  source: CatalogSource | undefined;
};

const ModelCatalogCard: React.FC<ModelCatalogCardProps> = ({ model, source }) => {
  const allLabels = model.customProperties ? getLabels(model.customProperties) : [];
  const valueLabels = model.customProperties
    ? getValueLabels(model.customProperties, CATALOG_VALUE_LABEL_KEYS)
    : [];
  const isValidated = isModelValidated(model);
  const isGatedAccessDenied = isHfGatedAccessDenied(model);
  const hasAccessLabel = getHfAccessLabelVariant(model) !== null;

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
              ) : hasAccessLabel ? (
                <ModelCatalogAccessLabel model={model} />
              ) : (
                source && <Label data-testid="model-catalog-source-label">{source.name}</Label>
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
      {!isGatedAccessDenied && (
        <>
          <CardBody>
            <ModelCatalogCardBody model={model} isValidated={isValidated} source={source} />
          </CardBody>
          <CardFooter>
            <ModelCatalogLabels
              tasks={model.tasks ?? []}
              validatedTasks={model.validatedTasks}
              provider={model.provider}
              labels={[...allLabels.filter((label) => label !== 'validated'), ...valueLabels]}
              numLabels={isValidated ? 2 : 3}
            />
          </CardFooter>
        </>
      )}
    </Card>
  );
};

export default ModelCatalogCard;
