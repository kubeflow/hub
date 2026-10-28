import React from 'react';
import { useParams } from 'react-router';
import { Link } from 'react-router-dom';
import {
  Breadcrumb,
  BreadcrumbItem,
  Button,
  Content,
  ContentVariants,
  EmptyState,
  EmptyStateBody,
  EmptyStateFooter,
  Flex,
  FlexItem,
  Label,
  Stack,
  StackItem,
} from '@patternfly/react-core';
import { CubeIcon, SearchIcon } from '@patternfly/react-icons';
import { ApplicationsPage } from 'mod-arch-shared';
import { useSkillWithAPI } from '~/app/hooks/skillCatalog/useSkill';
import { SkillCatalogContext } from '~/app/context/skillCatalog/SkillCatalogContext';
import { skillCatalogUrl } from '~/app/routes/skillCatalog/skillCatalog';
import {
  SKILL_CATALOG_TITLE,
  SKILL_TRUST_TIER_LABEL_MAPPING,
} from '~/app/pages/skillCatalog/const';
import ScrollViewOnMount from '~/app/shared/components/ScrollViewOnMount';
import SkillDetailsView from './SkillDetailsView';

const SkillDetailsPage: React.FC = () => {
  const { skillId = '' } = useParams<{ skillId: string }>();
  const { skillApiState } = React.useContext(SkillCatalogContext);
  const [skill, skillLoaded, skillLoadError] = useSkillWithAPI(skillApiState, skillId);

  const isNotFound = !skill && (skillLoaded || !!skillLoadError);
  const skillName = skill?.displayName || skill?.name;

  return (
    <>
      <ScrollViewOnMount shouldScroll scrollToTop />
      <ApplicationsPage
        breadcrumb={
          <Breadcrumb>
            <BreadcrumbItem>
              <Link to={skillCatalogUrl()}>{SKILL_CATALOG_TITLE}</Link>
            </BreadcrumbItem>
            <BreadcrumbItem isActive data-testid="breadcrumb-skill-name">
              {skillName || 'Details'}
            </BreadcrumbItem>
          </Breadcrumb>
        }
        title={
          skill ? (
            <Flex
              spaceItems={{ default: 'spaceItemsMd' }}
              alignItems={{ default: 'alignItemsCenter' }}
            >
              <FlexItem
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  height: '48px',
                  width: '48px',
                  borderRadius: '50%',
                  backgroundColor: 'var(--pf-t--global--background--color--secondary--default)',
                }}
              >
                <CubeIcon style={{ fontSize: '24px' }} data-testid="skill-default-icon" />
              </FlexItem>
              <Stack>
                <StackItem>{skillName}</StackItem>
                <StackItem>
                  <Flex
                    gap={{ default: 'gapSm' }}
                    alignItems={{ default: 'alignItemsCenter' }}
                    flexWrap={{ default: 'wrap' }}
                  >
                    <FlexItem>
                      <Label
                        color={skill.trustTier ? 'green' : 'grey'}
                        data-testid="skill-details-trust-tier"
                      >
                        {skill.trustTier
                          ? (SKILL_TRUST_TIER_LABEL_MAPPING[skill.trustTier] ?? skill.trustTier)
                          : 'Unrated'}
                      </Label>
                    </FlexItem>
                    {skill.provider && (
                      <FlexItem>
                        <Content component={ContentVariants.small}>
                          Provider: {skill.provider}
                        </Content>
                      </FlexItem>
                    )}
                  </Flex>
                </StackItem>
              </Stack>
            </Flex>
          ) : null
        }
        empty={isNotFound}
        emptyStatePage={
          isNotFound ? (
            <EmptyState icon={SearchIcon} titleText="Skill not found" data-testid="skill-not-found">
              <EmptyStateBody>The requested skill could not be found.</EmptyStateBody>
              <EmptyStateFooter>
                <Button
                  variant="primary"
                  component={(props) => <Link {...props} to={skillCatalogUrl()} />}
                >
                  Return to {SKILL_CATALOG_TITLE}
                </Button>
              </EmptyStateFooter>
            </EmptyState>
          ) : undefined
        }
        loadError={isNotFound ? undefined : skillLoadError}
        loaded={isNotFound || skillLoaded}
        errorMessage="Unable to load skill details"
        provideChildrenPadding
      >
        {skill && <SkillDetailsView skill={skill} />}
      </ApplicationsPage>
    </>
  );
};

export default SkillDetailsPage;
