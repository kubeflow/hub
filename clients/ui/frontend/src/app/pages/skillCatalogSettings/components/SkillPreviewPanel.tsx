import * as React from 'react';
import {
  EmptyState,
  EmptyStateVariant,
  EmptyStateBody,
  EmptyStateFooter,
  EmptyStateActions,
  ExpandableSection,
  Flex,
  FlexItem,
  Title,
  Tabs,
  Tab,
  TabTitleText,
  Alert,
  List,
  ListItem,
  Spinner,
  Button,
  AlertActionLink,
} from '@patternfly/react-core';
import { CheckCircleIcon, TimesCircleIcon } from '@patternfly/react-icons';
import { UseSkillSourcePreviewResult } from '~/app/pages/skillCatalogSettings/useSkillSourcePreview';
import { CatalogSettingsPreviewTab } from '~/app/shared/catalogSettings/hooks/previewTypes';

type SkillPreviewPanelProps = {
  preview: UseSkillSourcePreviewResult;
};

const SkillPreviewPanel: React.FC<SkillPreviewPanelProps> = ({ preview }) => {
  const {
    previewState,
    handlePreview: onPreview,
    handleTabChange,
    handleLoadMore: onLoadMore,
    hasFormChanged,
    canPreview,
  } = preview;
  const { isLoadingInitial, isLoadingMore, activeTab, summary, tabStates, error } = previewState;
  const { items, hasMore } = tabStates[activeTab];

  const handleTabSelect = (_event: React.MouseEvent, tabIndex: string | number) => {
    handleTabChange(
      tabIndex === 0 ? CatalogSettingsPreviewTab.INCLUDED : CatalogSettingsPreviewTab.EXCLUDED,
    );
  };

  const renderEmptyState = () => {
    if (error) {
      return (
        <EmptyState
          icon={TimesCircleIcon}
          titleText="Preview failed"
          variant={EmptyStateVariant.sm}
        >
          <EmptyStateBody>
            {/* Preview runs without the Auth token field: buildPreviewRequest strips it,
                and the catalog service resolves credentials only through a stored
                credentialRef. So a private repository can only be previewed once the
                source has been saved — telling the admin to fill in the token here would
                send them down a dead end. */}
            <p>
              We couldn&apos;t load skills from this repository. Double-check that the repository
              URL above is correct and reachable from the cluster. Previewing a private repository
              requires its token to be stored first — save the source, then preview it from the edit
              form.
            </p>
            <ExpandableSection
              toggleText="Show error details"
              className="pf-v6-u-mt-sm pf-v6-u-text-align-left"
              data-testid="skill-preview-error-details"
            >
              <p className="pf-v6-u-color-200 pf-v6-u-font-size-sm">{error.message}</p>
            </ExpandableSection>
          </EmptyStateBody>
          <EmptyStateFooter>
            <EmptyStateActions>
              <Button
                variant="link"
                onClick={onPreview}
                isDisabled={!canPreview}
                isLoading={isLoadingInitial}
                data-testid="skill-preview-button-panel-retry"
              >
                Preview
              </Button>
            </EmptyStateActions>
          </EmptyStateFooter>
        </EmptyState>
      );
    }

    return (
      <EmptyState titleText="Preview skills" variant={EmptyStateVariant.sm}>
        <EmptyStateBody>
          Complete all required fields, then click <strong>Preview</strong> to see which skills will
          appear in the catalog.
        </EmptyStateBody>
        <EmptyStateFooter>
          <EmptyStateActions>
            <Button
              variant="link"
              onClick={onPreview}
              isDisabled={!canPreview}
              isLoading={isLoadingInitial}
              data-testid="skill-preview-button-panel"
            >
              Preview
            </Button>
          </EmptyStateActions>
        </EmptyStateFooter>
      </EmptyState>
    );
  };

  const renderContent = () => {
    if (isLoadingInitial) {
      return (
        <div className="pf-v6-u-text-align-center pf-v6-u-py-xl">
          <Spinner size="xl" aria-label="Loading preview" />
        </div>
      );
    }

    if ((!items.length && !summary) || error) {
      return renderEmptyState();
    }

    return (
      <>
        <Tabs
          activeKey={activeTab === CatalogSettingsPreviewTab.INCLUDED ? 0 : 1}
          onSelect={handleTabSelect}
          aria-label="Skill preview tabs"
        >
          <Tab eventKey={0} title={<TabTitleText>Skills included</TabTitleText>} />
          <Tab eventKey={1} title={<TabTitleText>Skills excluded</TabTitleText>} />
        </Tabs>
        <div className="pf-v6-u-mt-md">
          {hasFormChanged && (
            <Alert
              variant="info"
              isInline
              title="Source configuration changed. Refresh the preview."
              className="pf-v6-u-mb-md"
              actionLinks={
                <AlertActionLink onClick={onPreview} data-testid="skill-refresh-preview-link">
                  Refresh preview
                </AlertActionLink>
              }
            />
          )}
          {items.length > 0 ? (
            <>
              <strong>
                {activeTab === CatalogSettingsPreviewTab.INCLUDED
                  ? `${summary?.includedAssets ?? 0} of ${summary?.totalAssets ?? 0} skills included:`
                  : `${summary?.excludedAssets ?? 0} of ${summary?.totalAssets ?? 0} skills excluded:`}
              </strong>
              <List isPlain className="pf-v6-u-mt-md">
                {items.map((skill) => (
                  <ListItem
                    key={skill.name}
                    icon={
                      skill.included ? (
                        <CheckCircleIcon color="green" />
                      ) : (
                        <TimesCircleIcon color="red" />
                      )
                    }
                  >
                    {skill.name}
                  </ListItem>
                ))}
              </List>
              {hasMore && (
                <div className="pf-v6-u-mt-md pf-v6-u-text-align-center">
                  <Button
                    variant="link"
                    onClick={onLoadMore}
                    isLoading={isLoadingMore}
                    isDisabled={isLoadingMore}
                  >
                    {isLoadingMore ? 'Loading...' : 'Load more'}
                  </Button>
                </div>
              )}
            </>
          ) : (
            <EmptyState
              variant={EmptyStateVariant.sm}
              titleText={
                activeTab === CatalogSettingsPreviewTab.INCLUDED
                  ? 'No skills included'
                  : 'No skills excluded'
              }
            >
              <EmptyStateBody>
                {activeTab === CatalogSettingsPreviewTab.INCLUDED
                  ? 'No skills match the current filter settings.'
                  : 'All skills from this repository are included.'}
              </EmptyStateBody>
            </EmptyState>
          )}
        </div>
      </>
    );
  };

  return (
    <div data-testid="skill-preview-panel" className="pf-v6-u-h-100">
      <Flex
        justifyContent={{ default: 'justifyContentSpaceBetween' }}
        alignItems={{ default: 'alignItemsCenter' }}
        className="pf-v6-u-mb-md"
      >
        <FlexItem>
          <Title headingLevel="h2" size="lg">
            Skill catalog preview
          </Title>
        </FlexItem>
        <FlexItem>
          <Button
            variant="secondary"
            onClick={onPreview}
            isDisabled={!canPreview}
            isLoading={isLoadingInitial}
            data-testid="skill-preview-button-header"
          >
            Preview
          </Button>
        </FlexItem>
      </Flex>
      {renderContent()}
    </div>
  );
};

export default SkillPreviewPanel;
