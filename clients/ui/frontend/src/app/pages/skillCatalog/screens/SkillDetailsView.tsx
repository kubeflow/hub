import * as React from 'react';
import {
  Card,
  CardBody,
  CardHeader,
  Content,
  DescriptionList,
  DescriptionListDescription,
  DescriptionListGroup,
  DescriptionListTerm,
  Icon,
  Label,
  LabelGroup,
  List,
  ListItem,
  PageSection,
  Sidebar,
  Tooltip,
  SidebarContent,
  SidebarPanel,
  Stack,
  StackItem,
  Title,
} from '@patternfly/react-core';
import { CodeIcon, ExclamationTriangleIcon, GithubIcon } from '@patternfly/react-icons';
import type { Skill } from '~/app/skillCatalogTypes';
import ExternalLink from '~/app/shared/components/ExternalLink';
import MarkdownComponent from '~/app/shared/markdown/MarkdownComponent';
import SkillFileTree from '~/app/pages/skillCatalog/components/SkillFileTree';
import {
  SKILL_RECOMMENDED_MAX_BODY_LINES,
  SKILL_TRUST_TIER_LABEL_MAPPING,
} from '~/app/pages/skillCatalog/const';
import { buildSupportingFilesTree } from '~/app/pages/skillCatalog/utils/skillSupportingFilesTree';
import { parseAllowedTools } from '~/app/pages/skillCatalog/utils/skillCatalogUtils';

type SkillDetailsViewProps = {
  skill: Skill;
};

const VISIBLE_LABELS = 3;
const MAX_BOX_HEIGHT = '200px';

// Scrolls only once content exceeds MAX_BOX_HEIGHT — a short list sizes to its
// content instead of leaving dead space inside a fixed-height box.
const scrollBoxStyle: React.CSSProperties = {
  maxHeight: MAX_BOX_HEIGHT,
  overflowY: 'auto',
  border: '1px solid var(--pf-t--global--border--color--default)',
  borderRadius: 'var(--pf-t--global--border--radius--small)',
  padding: '4px 8px',
};

// Shortens a repository URL to its "org/repo" form for the sidebar Source field,
// while the full URL remains the link target.
const getRepoShortLabel = (url?: string): string | undefined => {
  if (!url) {
    return undefined;
  }
  try {
    const parts = new URL(url).pathname
      .replace(/\.git$/, '')
      .split('/')
      .filter(Boolean);
    return parts.length >= 2 ? `${parts[parts.length - 2]}/${parts[parts.length - 1]}` : url;
  } catch {
    return url;
  }
};

const SkillDetailsView: React.FC<SkillDetailsViewProps> = ({ skill }) => {
  const repositoryUrl = skill.repository || skill.repositoryUrl;
  const repoShortLabel = getRepoShortLabel(repositoryUrl);
  const supportingFilesTree = buildSupportingFilesTree(skill);
  const allowedTools = parseAllowedTools(skill.allowedTools);

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
                    <p data-testid="skill-description">{skill.description || 'No description'}</p>
                  </Content>
                </CardBody>
              </Card>
            </StackItem>
            <StackItem>
              <Card>
                <CardHeader>
                  <Title headingLevel="h2" size="lg">
                    <Icon isInline style={{ marginRight: '4px' }}>
                      <GithubIcon />
                    </Icon>
                    README
                  </Title>
                </CardHeader>
                <CardBody>
                  {!skill.readme && (
                    <Content component="p" data-testid="skill-no-readme">
                      No README available
                    </Content>
                  )}
                  {skill.readme && (
                    <MarkdownComponent
                      data={skill.readme}
                      dataTestId={`skill-readme-${skill.id}`}
                      maxHeading={3}
                    />
                  )}
                </CardBody>
              </Card>
            </StackItem>
          </Stack>
        </SidebarContent>
        <SidebarPanel width={{ default: 'width_33' }}>
          <Card>
            <CardHeader>
              <Title headingLevel="h2" size="lg">
                Skill details
              </Title>
            </CardHeader>
            <CardBody>
              <DescriptionList>
                {skill.category && (
                  <DescriptionListGroup>
                    <DescriptionListTerm>Category</DescriptionListTerm>
                    <DescriptionListDescription data-testid="skill-category">
                      {skill.category}
                    </DescriptionListDescription>
                  </DescriptionListGroup>
                )}
                <DescriptionListGroup>
                  <DescriptionListTerm>Provider</DescriptionListTerm>
                  <DescriptionListDescription data-testid="skill-provider">
                    {skill.provider || 'N/A'}
                  </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                  <DescriptionListTerm>Trust tier</DescriptionListTerm>
                  <DescriptionListDescription>
                    <Label
                      color={skill.trustTier ? 'green' : 'grey'}
                      data-testid="skill-detail-trust-tier"
                    >
                      {skill.trustTier
                        ? (SKILL_TRUST_TIER_LABEL_MAPPING[skill.trustTier] ?? skill.trustTier)
                        : 'Unrated'}
                    </Label>
                  </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                  <DescriptionListTerm>Version</DescriptionListTerm>
                  <DescriptionListDescription data-testid="skill-version">
                    {skill.version || 'N/A'}
                  </DescriptionListDescription>
                </DescriptionListGroup>
                {skill.bodyLineCount !== undefined && (
                  <DescriptionListGroup>
                    <DescriptionListTerm>Skill length</DescriptionListTerm>
                    <DescriptionListDescription>
                      {/* An agent loads the whole SKILL.md body into its context, so length
                          is a cost signal. Shown for every skill; the colour and tooltip
                          only escalate past the recommended maximum. */}
                      <Tooltip
                        content={
                          skill.bodyLineCount > SKILL_RECOMMENDED_MAX_BODY_LINES
                            ? `This skill's body exceeds the recommended maximum of ${SKILL_RECOMMENDED_MAX_BODY_LINES} lines. An agent loads the whole body into its context, so longer skills leave less room for your own work.`
                            : `Length of the skill's SKILL.md body. The recommended maximum is ${SKILL_RECOMMENDED_MAX_BODY_LINES} lines, because an agent loads the whole body into its context.`
                        }
                      >
                        <Label
                          color={
                            skill.bodyLineCount > SKILL_RECOMMENDED_MAX_BODY_LINES
                              ? 'orange'
                              : 'green'
                          }
                          icon={
                            skill.bodyLineCount > SKILL_RECOMMENDED_MAX_BODY_LINES ? (
                              <ExclamationTriangleIcon />
                            ) : undefined
                          }
                          data-testid="skill-body-line-count"
                        >
                          {skill.bodyLineCount} lines
                        </Label>
                      </Tooltip>
                    </DescriptionListDescription>
                  </DescriptionListGroup>
                )}
                <DescriptionListGroup>
                  <DescriptionListTerm>Invocation</DescriptionListTerm>
                  <DescriptionListDescription data-testid="skill-invocation">
                    /{skill.name}
                  </DescriptionListDescription>
                </DescriptionListGroup>
                {skill.labels && skill.labels.length > 0 && (
                  <DescriptionListGroup>
                    <DescriptionListTerm>Labels</DescriptionListTerm>
                    <DescriptionListDescription>
                      <LabelGroup numLabels={VISIBLE_LABELS} isCompact>
                        {skill.labels.map((label) => (
                          <Label key={label} variant="outline" data-testid="skill-detail-label">
                            {label}
                          </Label>
                        ))}
                      </LabelGroup>
                    </DescriptionListDescription>
                  </DescriptionListGroup>
                )}
                <DescriptionListGroup>
                  <DescriptionListTerm>License</DescriptionListTerm>
                  <DescriptionListDescription data-testid="skill-license">
                    {skill.license || 'N/A'}
                  </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                  <DescriptionListTerm>Author</DescriptionListTerm>
                  <DescriptionListDescription data-testid="skill-author">
                    {skill.author || 'N/A'}
                  </DescriptionListDescription>
                </DescriptionListGroup>
                {skill.compatibility && (
                  <DescriptionListGroup>
                    <DescriptionListTerm>Compatibility</DescriptionListTerm>
                    <DescriptionListDescription data-testid="skill-compatibility">
                      {skill.compatibility}
                    </DescriptionListDescription>
                  </DescriptionListGroup>
                )}
                {repositoryUrl && (
                  <DescriptionListGroup>
                    <DescriptionListTerm>Source</DescriptionListTerm>
                    <DescriptionListDescription>
                      <ExternalLink
                        text={repoShortLabel || repositoryUrl}
                        to={repositoryUrl}
                        testId="skill-source-code-link"
                      />
                    </DescriptionListDescription>
                  </DescriptionListGroup>
                )}
                {supportingFilesTree && (
                  <DescriptionListGroup>
                    <DescriptionListTerm>Supporting files</DescriptionListTerm>
                    <DescriptionListDescription>
                      <div style={scrollBoxStyle} data-testid="skill-supporting-files-box">
                        <SkillFileTree root={supportingFilesTree} />
                      </div>
                    </DescriptionListDescription>
                  </DescriptionListGroup>
                )}
                {allowedTools.length > 0 && (
                  <DescriptionListGroup>
                    <DescriptionListTerm>Allowed tools</DescriptionListTerm>
                    <DescriptionListDescription>
                      <div style={scrollBoxStyle} data-testid="skill-allowed-tools-box">
                        <List
                          isPlain
                          data-testid="skill-allowed-tools-list"
                          style={{ fontSize: '13px' }}
                        >
                          {allowedTools.map((tool) => (
                            <ListItem
                              key={tool}
                              icon={
                                <Icon isInline style={{ fontSize: '12px' }}>
                                  <CodeIcon />
                                </Icon>
                              }
                              data-testid={`skill-allowed-tool-${tool}`}
                              style={{ lineHeight: '22px' }}
                            >
                              {tool}
                            </ListItem>
                          ))}
                        </List>
                      </div>
                    </DescriptionListDescription>
                  </DescriptionListGroup>
                )}
              </DescriptionList>
            </CardBody>
          </Card>
        </SidebarPanel>
      </Sidebar>
    </PageSection>
  );
};

export default SkillDetailsView;
