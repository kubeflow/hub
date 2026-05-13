import * as React from 'react';
import {
  Button,
  Card,
  CardBody,
  CardExpandableContent,
  CardHeader,
  Content,
  DescriptionList,
  DescriptionListDescription,
  DescriptionListGroup,
  DescriptionListTerm,
  Icon,
  Label,
  LabelGroup,
  PageSection,
  Popover,
  Sidebar,
  SidebarContent,
  SidebarPanel,
  Spinner,
  Alert,
  Stack,
  StackItem,
  Title,
} from '@patternfly/react-core';
import { OutlinedClockIcon, HelpIcon } from '@patternfly/react-icons';
import { InlineTruncatedClipboardCopy } from 'mod-arch-shared';
import text from '@patternfly/react-styles/css/utilities/Text/text';
import { CatalogArtifactList, CatalogModel } from '~/app/modelCatalogTypes';
import {
  getLabels,
  getValidatedOnPlatforms,
  getCustomPropString,
} from '~/app/pages/modelRegistry/screens/utils';
import ModelCatalogLabels from '~/app/pages/modelCatalog/components/ModelCatalogLabels';
import ExternalLink from '~/app/shared/components/ExternalLink';
import MarkdownComponent from '~/app/shared/markdown/MarkdownComponent';
import ModelTimestamp from '~/app/pages/modelRegistry/screens/components/ModelTimestamp';
import {
  getArchitecturesFromArtifacts,
  getModelArtifactUri,
  hasModelArtifacts,
  isModelValidated,
  hasValidatedToolCalling,
  formatModelTypeDisplay,
} from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import { CatalogModelCustomPropertyKey } from '~/concepts/modelCatalog/const';
import { useTempDevFeatureAvailable, TempDevFeature } from '~/app/hooks/useTempDevFeatureAvailable';
import CodeBlockComponent from '~/app/shared/markdown/components/CodeBlockComponent';

type ModelDetailsViewProps = {
  model: CatalogModel;
  artifacts: CatalogArtifactList;
  artifactLoaded: boolean;
  artifactsLoadError: Error | undefined;
};

const ModelDetailsView: React.FC<ModelDetailsViewProps> = ({
  model,
  artifacts,
  artifactLoaded,
  artifactsLoadError,
}) => {
  const allLabels = model.customProperties ? getLabels(model.customProperties) : [];
  const isValidated = isModelValidated(model);
  const isToolCallingEnabled = useTempDevFeatureAvailable(TempDevFeature.ToolCallingConfiguration);
  const isToolCallingValidated = isToolCallingEnabled && hasValidatedToolCalling(model);

  const validatedOnPlatforms = getValidatedOnPlatforms(model.customProperties);

  const modelTypeRaw = model.customProperties
    ? getCustomPropString(model.customProperties, CatalogModelCustomPropertyKey.MODEL_TYPE)
    : null;

  const modelTypeDisplay = React.useMemo(
    () => formatModelTypeDisplay(modelTypeRaw),
    [modelTypeRaw],
  );

  const tensorType = model.customProperties
    ? getCustomPropString(model.customProperties, CatalogModelCustomPropertyKey.TENSOR_TYPE)
    : '';
  const size = model.customProperties
    ? getCustomPropString(model.customProperties, CatalogModelCustomPropertyKey.SIZE)
    : '';

  const architectures = React.useMemo(
    () => (artifactLoaded ? getArchitecturesFromArtifacts(artifacts.items) : []),
    [artifactLoaded, artifacts.items],
  );

  const [isToolCallingExpanded, setIsToolCallingExpanded] = React.useState(false);

  return (
    <PageSection hasBodyWrapper={false} isFilled padding={{ default: 'noPadding' }}>
      <Sidebar hasGutter isPanelRight>
        <SidebarContent style={{ minWidth: 0, overflow: 'hidden' }}>
          <Stack hasGutter>
            <StackItem>
              <Card>
                <CardHeader>
                  <Title headingLevel="h2" size="lg">
                    Description
                  </Title>
                </CardHeader>
                <CardBody>
                  <Content className="pf-v6-u-text-break-word">
                    <p data-testid="model-long-description">
                      {model.description || 'No description'}
                    </p>
                  </Content>
                </CardBody>
              </Card>
            </StackItem>
            <StackItem>
              <Card>
                <CardHeader>
                  <Title headingLevel="h2" size="lg">
                    Model card
                  </Title>
                </CardHeader>
                <CardBody>
                  {!model.readme && <p className={text.textColorDisabled}>No model card</p>}
                  {model.readme && (
                    <MarkdownComponent
                      data={model.readme}
                      dataTestId="model-card-markdown"
                      maxHeading={3}
                    />
                  )}
                </CardBody>
              </Card>
            </StackItem>
          </Stack>
        </SidebarContent>
        <SidebarPanel width={{ default: 'width_33' }}>
          <Stack hasGutter>
            {isToolCallingValidated && (
              <StackItem>
                <Card data-testid="validated-configurations-card">
                  <CardHeader>
                    <Title headingLevel="h2" size="lg">
                      Validated configurations
                    </Title>
                  </CardHeader>
                  <CardBody>
                    <Content component="p" className="pf-v6-u-mb-md">
                      These configurations have been tested and validated for this model. Use them
                      as runtime arguments when deploying.
                    </Content>
                    <Card
                      id="tool-calling-card"
                      isExpanded={isToolCallingExpanded}
                      data-testid="tool-calling-card"
                    >
                      <CardHeader
                        onExpand={() => setIsToolCallingExpanded((prev) => !prev)}
                        toggleButtonProps={{
                          id: 'tool-calling-toggle',
                          'aria-label': 'Tool Calling',
                          'aria-expanded': isToolCallingExpanded,
                        }}
                      >
                        <Stack>
                          <StackItem>
                            <Title headingLevel="h3" size="md">
                              Tool Calling
                            </Title>
                          </StackItem>
                          <StackItem>
                            <Content component="small">
                              Enables agentic workflows and function calling capabilities.
                            </Content>
                          </StackItem>
                        </Stack>
                      </CardHeader>
                      <CardExpandableContent>
                        <CardBody>
                          <CodeBlockComponent>
                            {model.servingConfig?.toolCalling?.args ?? ''}
                          </CodeBlockComponent>
                        </CardBody>
                      </CardExpandableContent>
                    </Card>
                  </CardBody>
                </Card>
              </StackItem>
            )}
            <StackItem>
              <Card>
                <CardHeader>
                  <Title headingLevel="h2" size="lg">
                    Model details
                  </Title>
                </CardHeader>
                <CardBody>
                  <DescriptionList>
                    <DescriptionListGroup>
                      <DescriptionListTerm>Labels</DescriptionListTerm>
                      <DescriptionListDescription>
                        <ModelCatalogLabels
                          tasks={model.tasks ?? []}
                          validatedTasks={isToolCallingEnabled ? model.validatedTasks : undefined}
                          labels={allLabels.filter((label) => label !== 'validated')}
                          numLabels={isValidated ? 2 : 3}
                        />
                      </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                      <DescriptionListTerm>Model type</DescriptionListTerm>
                      <DescriptionListDescription data-testid="model-type">
                        {modelTypeDisplay}
                      </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                      <DescriptionListTerm>Tensor type</DescriptionListTerm>
                      <DescriptionListDescription>{tensorType || 'N/A'}</DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                      <DescriptionListTerm>Size</DescriptionListTerm>
                      <DescriptionListDescription>{size || 'N/A'}</DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                      <DescriptionListTerm>License</DescriptionListTerm>
                      <DescriptionListDescription>
                        <ExternalLink
                          text="Agreement"
                          to={model.licenseLink || ''}
                          testId="model-license-link"
                        />
                      </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                      <DescriptionListTerm>Provider</DescriptionListTerm>
                      <DescriptionListDescription>
                        {model.provider || 'N/A'}
                      </DescriptionListDescription>
                    </DescriptionListGroup>
                    {validatedOnPlatforms.length > 0 && (
                      <DescriptionListGroup>
                        <DescriptionListTerm>
                          Certified platforms{' '}
                          <Popover bodyContent="The model has been tested and verified to operate correctly on the specified enterprise platforms.">
                            <Button
                              variant="plain"
                              aria-label="More info for validated on"
                              className="pf-v6-u-p-xs"
                              icon={<HelpIcon />}
                            />
                          </Popover>
                        </DescriptionListTerm>
                        <DescriptionListDescription>
                          <LabelGroup numLabels={5} isCompact>
                            {validatedOnPlatforms.map((platform) => (
                              <Label
                                data-testid="validated-on-label"
                                key={platform}
                                variant="outline"
                              >
                                {platform}
                              </Label>
                            ))}
                          </LabelGroup>
                        </DescriptionListDescription>
                      </DescriptionListGroup>
                    )}
                    <DescriptionListGroup>
                      <DescriptionListTerm>Model location</DescriptionListTerm>
                      {artifactsLoadError ? (
                        <Alert variant="danger" isInline title={artifactsLoadError.name}>
                          {artifactsLoadError.message}
                        </Alert>
                      ) : !artifactLoaded ? (
                        <Spinner size="sm" />
                      ) : artifacts.items.length > 0 && hasModelArtifacts(artifacts.items) ? (
                        <DescriptionListDescription>
                          <InlineTruncatedClipboardCopy
                            testId="source-image-location"
                            textToCopy={getModelArtifactUri(artifacts.items) || ''}
                          />
                        </DescriptionListDescription>
                      ) : (
                        <p className={text.textColorDisabled}>No artifacts available</p>
                      )}
                    </DescriptionListGroup>
                    {architectures.length > 0 && (
                      <DescriptionListGroup>
                        <DescriptionListTerm>Architecture</DescriptionListTerm>
                        <DescriptionListDescription data-testid="model-architecture">
                          {architectures.join(', ')}
                        </DescriptionListDescription>
                      </DescriptionListGroup>
                    )}
                    <DescriptionListGroup>
                      <DescriptionListTerm>Last modified</DescriptionListTerm>
                      <DescriptionListDescription>
                        <Icon isInline style={{ marginRight: 4 }}>
                          <OutlinedClockIcon />
                        </Icon>
                        <ModelTimestamp timeSinceEpoch={model.lastUpdateTimeSinceEpoch} />
                      </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                      <DescriptionListTerm>Published</DescriptionListTerm>
                      <DescriptionListDescription>
                        <Icon isInline style={{ marginRight: 4 }}>
                          <OutlinedClockIcon />
                        </Icon>
                        <ModelTimestamp timeSinceEpoch={model.createTimeSinceEpoch} />
                      </DescriptionListDescription>
                    </DescriptionListGroup>
                  </DescriptionList>
                </CardBody>
              </Card>
            </StackItem>
          </Stack>
        </SidebarPanel>
      </Sidebar>
    </PageSection>
  );
};

export default ModelDetailsView;
