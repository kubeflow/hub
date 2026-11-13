import * as React from 'react';
import {
  Alert,
  FormGroup,
  FormHelperText,
  HelperText,
  HelperTextItem,
  TextInput,
  Stack,
  StackItem,
} from '@patternfly/react-core';
import { SimpleSelect } from 'mod-arch-shared';
import FormSection from '~/app/pages/modelRegistry/components/pf-overrides/FormSection';
import type { SkillRepository } from '~/app/skillCatalogTypes';
import { SKILL_TRUST_TIER_OPTIONS } from '~/app/pages/skillCatalog/const';

const arrToComma = (arr?: string[]): string => arr?.join(', ') ?? '';

const commaToArr = (s: string): string[] | undefined => {
  const items = s
    .split(',')
    .map((v) => v.trim())
    .filter(Boolean);
  return items.length > 0 ? items : undefined;
};

type SkillRepositoriesSectionProps = {
  repositories: SkillRepository[];
  onChange: (repositories: SkillRepository[]) => void;
};

const SkillRepositoriesSection: React.FC<SkillRepositoriesSectionProps> = ({
  repositories,
  onChange,
}) => {
  const repo = repositories[0] ?? { url: '' };
  // A source written through this form carries a single repository (the loader rejects
  // an inline list of more than one). A source authored by GitOps before that rule, or
  // hand-edited since, can still arrive with several: keep the ones this form cannot
  // show instead of dropping them, so editing the first repository is not destructive.
  const extraRepositories = repositories.slice(1);

  const handleChange = (patch: Partial<SkillRepository>) => {
    onChange([{ ...repo, ...patch }, ...extraRepositories]);
  };

  // Comma-separated fields keep their own raw text as local state, seeded once from the
  // loaded repo. Deriving the displayed text from the parsed array on every keystroke (via
  // arrToComma(commaToArr(value))) drops a trailing ", " the moment it's typed — split on
  // comma trims a still-being-typed empty segment away, so the next character the user types
  // lands right after the previous label instead of starting a new one.
  const [refsText, setRefsText] = React.useState(() => arrToComma(repo.refs));
  const [scanPathsText, setScanPathsText] = React.useState(() => arrToComma(repo.scanPaths));
  const [includedSkillsText, setIncludedSkillsText] = React.useState(() =>
    arrToComma(repo.includedSkills),
  );
  const [excludedSkillsText, setExcludedSkillsText] = React.useState(() =>
    arrToComma(repo.excludedSkills),
  );
  const [labelsText, setLabelsText] = React.useState(() => arrToComma(repo.labels));

  return (
    <FormSection title="Git repository">
      <Stack hasGutter>
        {extraRepositories.length > 0 && (
          <StackItem>
            <Alert
              isInline
              variant="danger"
              title={`This source lists ${repositories.length} repositories and cannot be saved here`}
              data-testid="skill-multi-repo-warning"
            >
              A source managed through this form carries a single repository. This one was
              configured outside the UI. To edit it, reduce it to one repository in the source
              ConfigMap, register each repository as its own source, or manage it through
              yamlCatalogPath.
            </Alert>
          </StackItem>
        )}
        <StackItem>
          <FormGroup label="Repository URL" fieldId="skill-repo-url-0" isRequired>
            <TextInput
              isRequired
              type="url"
              id="skill-repo-url-0"
              data-testid="skill-repo-url-0"
              value={repo.url}
              placeholder="https://github.com/org/repo"
              onChange={(_event, value) => handleChange({ url: value })}
            />
            <FormHelperText>
              <HelperText>
                <HelperTextItem>URL of a Git repository containing skill files.</HelperTextItem>
              </HelperText>
            </FormHelperText>
          </FormGroup>
        </StackItem>

        <StackItem>
          <FormGroup label="Auth token" fieldId="skill-repo-auth-token-0">
            <TextInput
              type="password"
              id="skill-repo-auth-token-0"
              data-testid="skill-repo-auth-token-0"
              value={repo.authToken ?? ''}
              placeholder={repo.credentialRef ? '••••••••  (unchanged)' : 'ghp_xxxxxxxxxxxx'}
              onChange={(_event, value) => handleChange({ authToken: value || undefined })}
            />
            <FormHelperText>
              <HelperText>
                <HelperTextItem>
                  {repo.credentialRef
                    ? 'A token is already stored for this source. Leave blank to keep it, or enter a new one to replace it.'
                    : 'Personal access token for private repositories. Stored in a Kubernetes secret, never in the catalog config.'}
                </HelperTextItem>
              </HelperText>
            </FormHelperText>
          </FormGroup>
        </StackItem>

        <StackItem>
          <FormGroup label="Refs" fieldId="skill-repo-refs-0">
            <TextInput
              id="skill-repo-refs-0"
              data-testid="skill-repo-refs-0"
              value={refsText}
              placeholder="v1.2.3, 9f8c1a2"
              onChange={(_event, value) => {
                setRefsText(value);
                handleChange({ refs: commaToArr(value) });
              }}
            />
            <FormHelperText>
              <HelperText>
                {/* Both rules are enforced by the catalog service, not here: a ref cannot be
                    told apart from a branch by name, so the resolver asks the remote at sync
                    time. Stating them here keeps an admin from saving a source that will only
                    fail later. Running a preview applies the same check before saving. */}
                <HelperTextItem>
                  Comma-separated Git tags or commit SHAs to scan, one entry per version. Each ref
                  becomes its own set of catalog entries.
                </HelperTextItem>
                <HelperTextItem>
                  Branches and HEAD are not accepted — skills must be pinned to an immutable ref so
                  installs are reproducible. At least one ref is required; a repository with none is
                  skipped at sync.
                </HelperTextItem>
              </HelperText>
            </FormHelperText>
          </FormGroup>
        </StackItem>

        <StackItem>
          <FormGroup label="Scan paths" fieldId="skill-repo-scanpaths-0">
            <TextInput
              id="skill-repo-scanpaths-0"
              data-testid="skill-repo-scanpaths-0"
              value={scanPathsText}
              placeholder="skills/"
              onChange={(_event, value) => {
                setScanPathsText(value);
                handleChange({ scanPaths: commaToArr(value) });
              }}
            />
            <FormHelperText>
              <HelperText>
                <HelperTextItem>
                  Comma-separated directory paths within the repository to scan for skills. Leave
                  empty to scan the entire repository.
                </HelperTextItem>
              </HelperText>
            </FormHelperText>
          </FormGroup>
        </StackItem>

        <StackItem>
          <FormGroup label="Included skills" fieldId="skill-repo-included-0">
            <TextInput
              id="skill-repo-included-0"
              data-testid="skill-repo-included-0"
              value={includedSkillsText}
              placeholder="*"
              onChange={(_event, value) => {
                setIncludedSkillsText(value);
                handleChange({ includedSkills: commaToArr(value) });
              }}
            />
            <FormHelperText>
              <HelperText>
                <HelperTextItem>
                  Comma-separated skill name patterns to include. Use <code>*</code> to include all
                  skills.
                </HelperTextItem>
              </HelperText>
            </FormHelperText>
          </FormGroup>
        </StackItem>

        <StackItem>
          <FormGroup label="Excluded skills" fieldId="skill-repo-excluded-0">
            <TextInput
              id="skill-repo-excluded-0"
              data-testid="skill-repo-excluded-0"
              value={excludedSkillsText}
              placeholder="internal-*"
              onChange={(_event, value) => {
                setExcludedSkillsText(value);
                handleChange({ excludedSkills: commaToArr(value) });
              }}
            />
            <FormHelperText>
              <HelperText>
                <HelperTextItem>Comma-separated skill name patterns to exclude.</HelperTextItem>
              </HelperText>
            </FormHelperText>
          </FormGroup>
        </StackItem>

        <StackItem>
          <FormGroup label="Skill labels" fieldId="skill-repo-labels-0">
            <TextInput
              id="skill-repo-labels-0"
              data-testid="skill-repo-labels-0"
              value={labelsText}
              placeholder="typescript, testing"
              onChange={(_event, value) => {
                setLabelsText(value);
                handleChange({ labels: commaToArr(value) });
              }}
            />
            <FormHelperText>
              <HelperText>
                <HelperTextItem>
                  Comma-separated labels applied to every skill from this repository. They appear on
                  skill cards and can be filtered on in the skill catalog.
                </HelperTextItem>
              </HelperText>
            </FormHelperText>
          </FormGroup>
        </StackItem>

        <StackItem>
          <FormGroup label="Provider" fieldId="skill-repo-provider-0">
            <TextInput
              type="text"
              id="skill-repo-provider-0"
              data-testid="skill-repo-provider-0"
              value={repo.provider ?? ''}
              onChange={(_event, value) => handleChange({ provider: value || undefined })}
              placeholder="e.g. Red Hat"
            />
          </FormGroup>
        </StackItem>

        <StackItem>
          <FormGroup label="Category" fieldId="skill-repo-category-0">
            <TextInput
              type="text"
              id="skill-repo-category-0"
              data-testid="skill-repo-category-0"
              value={repo.category ?? ''}
              onChange={(_event, value) => handleChange({ category: value || undefined })}
              placeholder="e.g. productivity"
            />
          </FormGroup>
        </StackItem>

        <StackItem>
          <FormGroup label="Trust tier" fieldId="skill-repo-trust-tier-0">
            <SimpleSelect
              options={SKILL_TRUST_TIER_OPTIONS}
              value={repo.trustTier}
              onChange={(key) => handleChange({ trustTier: key })}
              placeholder="Select a trust tier"
              isFullWidth
              dataTestId="skill-repo-trust-tier-0"
              previewDescription={false}
              popperProps={{ direction: 'down' }}
              toggleProps={{ id: 'skill-repo-trust-tier-0-toggle' }}
            />
          </FormGroup>
        </StackItem>
      </Stack>
    </FormSection>
  );
};

export default SkillRepositoriesSection;
