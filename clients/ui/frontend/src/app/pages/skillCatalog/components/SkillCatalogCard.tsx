import * as React from 'react';
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
  LabelGroup,
} from '@patternfly/react-core';
import { CubeIcon, PluggedIcon } from '@patternfly/react-icons';
import { Link, type LinkProps } from 'react-router-dom';
import type { Skill } from '~/app/skillCatalogTypes';
import { skillDetailsUrl } from '~/app/routes/skillCatalog/skillCatalog';
import { formatSkillVersion } from '~/app/pages/skillCatalog/utils/skillCatalogUtils';
import { SKILL_TRUST_TIER_LABEL_MAPPING } from '~/app/pages/skillCatalog/const';
import SkillInstallModal from '~/app/pages/skillCatalog/components/SkillInstallModal';

type SkillCatalogCardProps = {
  skill: Skill;
};

const SkillCatalogCard: React.FC<SkillCatalogCardProps> = React.memo(({ skill }) => {
  const [isInstallModalOpen, setIsInstallModalOpen] = React.useState(false);
  const skillName = skill.displayName || skill.name;

  return (
    <Card isFullHeight data-testid={`skill-catalog-card-${skill.id}`}>
      <CardHeader>
        <Flex
          alignItems={{ default: 'alignItemsFlexStart' }}
          justifyContent={{ default: 'justifyContentSpaceBetween' }}
          gap={{ default: 'gapXs' }}
          className="pf-v6-u-mb-md"
        >
          <FlexItem>
            <span
              className="pf-v6-u-display-inline-block pf-v6-u-font-size-2xl pf-v6-u-color-200"
              aria-hidden
              data-testid={`skill-catalog-card-icon-${skill.id}`}
            >
              <CubeIcon />
            </span>
          </FlexItem>
          <FlexItem>
            <Label
              color={skill.trustTier ? 'green' : 'grey'}
              data-testid={`skill-catalog-card-trust-tier-${skill.id}`}
            >
              {skill.trustTier
                ? (SKILL_TRUST_TIER_LABEL_MAPPING[skill.trustTier] ?? skill.trustTier)
                : 'Unrated'}
            </Label>
          </FlexItem>
        </Flex>
        <CardTitle>
          <Button
            data-testid={`skill-catalog-card-detail-link-${skill.id}`}
            variant="link"
            isInline
            component={(props: LinkProps) => <Link {...props} to={skillDetailsUrl(skill.id)} />}
            style={{
              fontSize: 'var(--pf-t--global--font--size--body--default)',
              fontWeight: 'var(--pf-t--global--font--weight--body--bold)',
            }}
          >
            <span
              style={{
                display: 'block',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}
              data-testid={`skill-catalog-card-name-${skill.id}`}
            >
              {skillName}
            </span>
          </Button>
        </CardTitle>
        {(skill.provider || skill.version) && (
          <div
            className="pf-v6-u-mt-xs pf-v6-u-font-size-sm pf-v6-u-color-200"
            style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
          >
            {skill.provider && (
              <span data-testid={`skill-catalog-card-provider-${skill.id}`}>{skill.provider}</span>
            )}
            {skill.provider && skill.version && <span aria-hidden> &middot; </span>}
            {/* Each ref a repository lists becomes its own catalog entry, so several tiles
                can share a name and description. The version is what tells them apart. */}
            {skill.version && (
              // title carries the full ref, so an abbreviated SHA stays recoverable.
              <span title={skill.version} data-testid={`skill-catalog-card-version-${skill.id}`}>
                {formatSkillVersion(skill.version)}
              </span>
            )}
          </div>
        )}
      </CardHeader>
      <CardBody>
        <span
          style={{
            display: '-webkit-box',
            WebkitBoxOrient: 'vertical',
            WebkitLineClamp: 4,
            overflow: 'hidden',
            overflowWrap: 'anywhere',
          }}
          data-testid={`skill-catalog-card-description-${skill.id}`}
        >
          {skill.description ?? ''}
        </span>
        {skill.labels && skill.labels.length > 0 && (
          <LabelGroup className="pf-v6-u-mt-md" numLabels={3}>
            {skill.labels.map((label) => (
              <Label key={label} variant="outline" data-testid={`skill-label-${label}`}>
                {label}
              </Label>
            ))}
          </LabelGroup>
        )}
      </CardBody>
      <CardFooter>
        <Flex
          alignItems={{ default: 'alignItemsCenter' }}
          justifyContent={{ default: 'justifyContentSpaceBetween' }}
          flexWrap={{ default: 'nowrap' }}
        >
          <FlexItem style={{ minWidth: 0 }}>
            <Label
              color="blue"
              data-testid={`skill-catalog-card-footer-name-${skill.id}`}
              style={{ fontFamily: 'monospace', maxWidth: '100%' }}
            >
              <span
                style={{
                  display: 'block',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
              >
                /{skill.name}
              </span>
            </Label>
          </FlexItem>
          <FlexItem>
            <Button
              variant="plain"
              aria-label={`Install ${skillName}`}
              icon={<PluggedIcon />}
              onClick={() => setIsInstallModalOpen(true)}
              data-testid={`skill-catalog-card-install-button-${skill.id}`}
            />
          </FlexItem>
        </Flex>
      </CardFooter>
      {isInstallModalOpen && (
        <SkillInstallModal
          skill={skill}
          isOpen={isInstallModalOpen}
          onClose={() => setIsInstallModalOpen(false)}
        />
      )}
    </Card>
  );
});
SkillCatalogCard.displayName = 'SkillCatalogCard';

export default SkillCatalogCard;
