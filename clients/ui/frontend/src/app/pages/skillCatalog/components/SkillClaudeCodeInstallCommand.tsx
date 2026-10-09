import * as React from 'react';
import { Alert, HelperText, HelperTextItem, Skeleton } from '@patternfly/react-core';
import type { Skill } from '~/app/skillCatalogTypes';
import { useSkillMarketplace } from '~/app/hooks/skillCatalog/useSkillMarketplace';
import SkillInstallCodeBlock from '~/app/pages/skillCatalog/components/SkillInstallCodeBlock';

type SkillClaudeCodeInstallCommandProps = {
  skill: Skill;
  idPrefix: string;
};

const SkillClaudeCodeInstallCommand: React.FC<SkillClaudeCodeInstallCommandProps> = ({
  skill,
  idPrefix,
}) => {
  const [marketplaceResult, marketplaceLoaded, marketplaceLoadError] = useSkillMarketplace();
  const marketplace = marketplaceResult?.marketplace;

  if (!marketplaceLoaded) {
    return (
      <Skeleton
        width="100%"
        height="80px"
        data-testid={`${idPrefix}-claude-code-loading-${skill.id}`}
      />
    );
  }

  const plugin = marketplace?.plugins?.find(
    (p) =>
      p.source.url === skill.repository &&
      p.source.path === skill.path &&
      (!skill.version || p.source.ref === skill.version),
  );

  if (marketplaceLoadError || !marketplace || !plugin) {
    return (
      <Alert
        variant="warning"
        isInline
        title="Install command unavailable"
        data-testid={`${idPrefix}-claude-code-unavailable-${skill.id}`}
      >
        The catalog marketplace endpoint did not return this skill. Try the npx or Manual commands
        instead.
      </Alert>
    );
  }

  const claudeCodeCommand = [
    `/plugin marketplace add ${marketplaceResult.marketplaceUrl}`,
    `/plugin install ${plugin.name}@${marketplace.name}`,
  ].join('\n');

  return (
    <>
      <SkillInstallCodeBlock
        id={`${idPrefix}-claude-code-${skill.id}`}
        content={claudeCodeCommand}
      />
      {!marketplaceResult.external && (
        <HelperText className="pf-v6-u-mt-sm">
          <HelperTextItem data-testid={`${idPrefix}-claude-code-in-cluster-note-${skill.id}`}>
            This marketplace URL resolves inside the cluster, so it works for agents running there.
            To use it from outside, expose the catalog through a Route or Ingress and set the
            SKILL_CATALOG_MARKETPLACE_URL override on the UI backend.
          </HelperTextItem>
        </HelperText>
      )}
    </>
  );
};

export default SkillClaudeCodeInstallCommand;
