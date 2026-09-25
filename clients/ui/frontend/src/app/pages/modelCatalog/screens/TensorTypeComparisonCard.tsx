import {
  Alert,
  Card,
  CardBody,
  Divider,
  Flex,
  FlexItem,
  Label,
  Skeleton,
  Spinner,
  Title,
  Truncate,
  TruncateProps,
} from '@patternfly/react-core';
import React from 'react';
import { Link } from 'react-router';
import { useCatalogModelsBySources } from '~/app/hooks/modelCatalog/useCatalogModelsBySource';
import { CatalogModel } from '~/app/modelCatalogTypes';
import { catalogModelDetailsTabFromModel } from '~/app/routes/modelCatalog/catalogModel';
import { getStringValue } from '~/app/utils';
import { getModelName } from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import { EMPTY_CUSTOM_PROPERTY_VALUE, ModelDetailsTab } from '~/concepts/modelCatalog/const';
import { sortModelsWithCurrentFirst } from '~/app/pages/modelCatalog/utils/validatedModelUtils';

type TensorTypeComparisonCardProps = {
  model: CatalogModel;
};
const TensorTypeComparisonCard: React.FC<TensorTypeComparisonCardProps> = ({ model }) => {
  const variantGroupId = getStringValue(model.customProperties ?? {}, 'variant_group_id');

  const variantFilterQuery = `variant_group_id.string_value="${variantGroupId}"`;

  const { catalogModels, catalogModelsLoaded, catalogModelsLoadError } = useCatalogModelsBySources(
    model.sourceId || '',
    undefined,
    10,
    '',
    undefined,
    undefined,
    variantFilterQuery,
  );

  const sortedModels = React.useMemo(
    () => sortModelsWithCurrentFirst(catalogModels.items, model.name, 4),
    [catalogModels.items, model.name],
  );

  // Hide the card if there's no variant group ID
  if (!variantGroupId || variantGroupId === EMPTY_CUSTOM_PROPERTY_VALUE) {
    return null;
  }

  // Hide the card if there's only one variant (the current model)
  if (catalogModelsLoaded && !catalogModelsLoadError && sortedModels.length <= 1) {
    return null;
  }

  return (
    <Card data-testid="compression-comparison-card">
      <CardBody>
        <Flex direction={{ default: 'column' }} gap={{ default: 'gapLg' }}>
          <FlexItem>
            <Flex direction={{ default: 'column' }} gap={{ default: 'gapSm' }}>
              <FlexItem>
                <Title headingLevel="h2" size="lg">
                  Model variants by tensor type
                </Title>
              </FlexItem>
              <FlexItem>
                <p>
                  Compare benchmark performance across tensor types to understand accuracy and
                  efficiency tradeoffs.
                </p>
              </FlexItem>
            </Flex>
          </FlexItem>
          <FlexItem style={{ overflowX: 'auto' }}>
            {catalogModelsLoadError ? (
              <Alert
                variant="danger"
                isInline
                title="Error loading performance data"
                data-testid="compression-comparison-error"
              >
                {catalogModelsLoadError.message || 'An error occurred'}
              </Alert>
            ) : !catalogModelsLoaded ? (
              <Spinner size="lg" data-testid="compression-comparison-loading" />
            ) : (
              <Flex
                gap={{ default: 'gapMd' }}
                flexWrap={{ default: 'nowrap' }}
                justifyContent={{ default: 'justifyContentSpaceEvenly' }}
              >
                {sortedModels.map((variant, index) => {
                  const tensorType = getStringValue(variant.customProperties ?? {}, 'tensor_type');
                  const isCurrent = variant.name === model.name;
                  const modelDisplayName = getModelName(variant.name || '');

                  const truncateProps: Pick<
                    TruncateProps,
                    'content' | 'position' | 'tooltipPosition' | 'maxCharsDisplayed'
                  > = {
                    content: modelDisplayName,
                    position: 'middle',
                    maxCharsDisplayed: 20,
                    tooltipPosition: 'top',
                  };

                  return (
                    <React.Fragment key={`${variant.name}-${index}`}>
                      {index > 0 && (
                        <Divider
                          orientation={{ default: 'vertical' }}
                          data-testid={`compression-divider-${index}`}
                        />
                      )}
                      <FlexItem
                        data-testid={`compression-variant-${index}`}
                        style={{
                          minWidth: '180px',
                        }}
                      >
                        <Flex
                          alignItems={{ default: 'alignItemsFlexStart' }}
                          gap={{ default: 'gapSm' }}
                          flexWrap={{ default: 'nowrap' }}
                        >
                          <FlexItem flex={{ default: 'flexNone' }}>
                            {variant.logo ? (
                              <img
                                src={variant.logo}
                                alt="model logo"
                                style={{ height: '56px', width: '56px' }}
                                data-testid={`compression-logo-${index}`}
                              />
                            ) : (
                              <Skeleton
                                shape="square"
                                width="56px"
                                height="56px"
                                screenreaderText="Brand image loading"
                                data-testid={`compression-skeleton-${index}`}
                              />
                            )}
                          </FlexItem>
                          <FlexItem>
                            <Flex
                              direction={{ default: 'column' }}
                              spaceItems={{ default: 'spaceItemsXs' }}
                            >
                              <FlexItem>
                                {isCurrent ? (
                                  <Truncate
                                    {...truncateProps}
                                    data-testid="compression-current-model-name"
                                  />
                                ) : (
                                  <Link
                                    to={catalogModelDetailsTabFromModel(
                                      ModelDetailsTab.PERFORMANCE_INSIGHTS,
                                      variant.name || '',
                                      variant.sourceId,
                                    )}
                                    data-testid={`compression-link-${index}`}
                                  >
                                    <Truncate
                                      {...truncateProps}
                                      style={{ textDecoration: 'underline' }}
                                    />
                                  </Link>
                                )}
                              </FlexItem>
                              <FlexItem>
                                <Flex
                                  spaceItems={{ default: 'spaceItemsXs' }}
                                  alignItems={{ default: 'alignItemsCenter' }}
                                >
                                  <FlexItem>
                                    {tensorType && tensorType !== EMPTY_CUSTOM_PROPERTY_VALUE && (
                                      <Label
                                        color="green"
                                        isCompact
                                        data-testid={`compression-tensor-type-${index}`}
                                      >
                                        {tensorType}
                                      </Label>
                                    )}
                                  </FlexItem>
                                  {isCurrent && (
                                    <FlexItem>
                                      <Label
                                        isCompact
                                        variant="outline"
                                        data-testid="compression-current-label"
                                      >
                                        Current model
                                      </Label>
                                    </FlexItem>
                                  )}
                                </Flex>
                              </FlexItem>
                            </Flex>
                          </FlexItem>
                        </Flex>
                      </FlexItem>
                    </React.Fragment>
                  );
                })}
              </Flex>
            )}
          </FlexItem>
        </Flex>
      </CardBody>
    </Card>
  );
};

export default TensorTypeComparisonCard;
