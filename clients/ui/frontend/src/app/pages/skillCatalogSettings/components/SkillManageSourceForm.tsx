import * as React from 'react';
import {
  Button,
  Checkbox,
  Form,
  FormGroup,
  FormHelperText,
  HelperText,
  HelperTextItem,
  Label,
  LabelGroup,
  Stack,
  StackItem,
  Sidebar,
  SidebarContent,
  SidebarPanel,
  TextInputGroup,
  TextInputGroupMain,
  TextInputGroupUtilities,
} from '@patternfly/react-core';
import { TimesIcon } from '@patternfly/react-icons';
import { useNavigate } from 'react-router-dom';
import FormSection from '~/app/pages/modelRegistry/components/pf-overrides/FormSection';
import {
  validateSourceName,
  SOURCE_NAME_CHARACTER_LIMIT,
  generateSourceIdFromName,
} from '~/app/shared/catalogSettings';
import { skillCatalogSettingsUrl } from '~/app/routes/skillCatalogSettings/skillCatalogSettings';
import { SkillCatalogSettingsContext } from '~/app/context/skillCatalogSettings/SkillCatalogSettingsContext';
import type { SkillCatalogSourceConfig } from '~/app/skillCatalogTypes';
import { SkillCatalogSourceType } from '~/app/skillCatalogTypes';
import { useManageSkillSourceData } from '~/app/pages/skillCatalogSettings/useManageSkillSourceData';
import type { ManageSkillSourceFormData } from '~/app/pages/skillCatalogSettings/useManageSkillSourceData';
import { useSkillSourcePreview } from '~/app/pages/skillCatalogSettings/useSkillSourcePreview';
import { findConflictingSource } from '~/app/pages/skillCatalogSettings/utils/duplicateSourceName';
import SkillSourceDetailsSection from './SkillSourceDetailsSection';
import SkillRepositoriesSection from './SkillRepositoriesSection';
import SkillPreviewPanel from './SkillPreviewPanel';
import SkillManageSourceFormFooter from './SkillManageSourceFormFooter';

const sourceConfigToFormData = (config: SkillCatalogSourceConfig): ManageSkillSourceFormData => {
  const firstRepo = config.repositories?.[0] ?? { url: '' };
  const remainingRepos = config.repositories?.slice(1) ?? [];
  return {
    id: config.id,
    name: config.name,
    enabled: config.enabled ?? true,
    repositories: [
      {
        ...firstRepo,
        provider: firstRepo.provider ?? config.provider,
        category: firstRepo.category ?? config.category,
        trustTier: firstRepo.trustTier ?? config.trustTier,
      },
      ...remainingRepos,
    ],
    labels: config.labels ?? [],
  };
};

const isFormValid = (formData: ManageSkillSourceFormData): boolean =>
  validateSourceName(formData.name, SOURCE_NAME_CHARACTER_LIMIT) &&
  // A source managed through this form carries exactly one repository, which is what
  // the BFF and the catalog service's inline form accept. A source that already lists
  // several was written outside the UI; saving it here would be rejected, so the form
  // blocks instead of offering a save that cannot succeed.
  formData.repositories.length === 1 &&
  formData.repositories.every((r) => r.url.trim().length > 0) &&
  // The catalog service skips a repository that lists no refs ("no refs configured;
  // skills must be pinned to a tag or commit SHA"), so a source saved without one can
  // only ever sync to Failed. The refs field's own helper text already says at least
  // one is required — enforce it here rather than offering a save that cannot work.
  formData.repositories.every((r) => (r.refs?.length ?? 0) > 0);

type SkillManageSourceFormProps = {
  existingSourceConfig?: SkillCatalogSourceConfig;
  isEditMode: boolean;
};

const SkillManageSourceForm: React.FC<SkillManageSourceFormProps> = ({
  existingSourceConfig,
  isEditMode,
}) => {
  const navigate = useNavigate();
  const existingData = existingSourceConfig
    ? sourceConfigToFormData(existingSourceConfig)
    : undefined;
  const [formData, setData] = useManageSkillSourceData(existingData);
  const [isSubmitting, setIsSubmitting] = React.useState(false);
  const [submitError, setSubmitError] = React.useState<Error | undefined>(undefined);
  const [labelInputValue, setLabelInputValue] = React.useState('');
  const {
    apiState,
    skillCatalogSourceConfigs,
    refreshSkillCatalogSourceConfigs,
    refreshSkillCatalogSources,
    markSkillSourceSyncPending,
  } = React.useContext(SkillCatalogSettingsContext);

  // The source id is derived from the name and is not editable, so two sources named
  // alike collide on id. The BFF rejects that with "already exists" — but only after
  // submit, and on create it is the same key the colliding source's git token is stored
  // under. Surfacing it on the name field keeps the admin from getting that far.
  const duplicateNameError = React.useMemo(() => {
    if (isEditMode) {
      return undefined;
    }
    const clash = findConflictingSource(formData.name, skillCatalogSourceConfigs);
    return clash
      ? `The source "${clash.name}" already uses the identifier "${generateSourceIdFromName(formData.name)}". Choose a different name.`
      : undefined;
  }, [isEditMode, formData.name, skillCatalogSourceConfigs]);

  const isComplete = isFormValid(formData) && !duplicateNameError;

  const preview = useSkillSourcePreview({
    formData,
    existingSourceConfig,
    apiState,
    isEditMode,
  });

  const addLabel = () => {
    const trimmed = labelInputValue.trim();
    if (trimmed && !formData.labels.includes(trimmed)) {
      setData('labels', [...formData.labels, trimmed]);
    }
    setLabelInputValue('');
  };

  const removeLabel = (label: string) => {
    setData(
      'labels',
      formData.labels.filter((l) => l !== label),
    );
  };

  const handleSubmit = async () => {
    if (!apiState.apiAvailable) {
      setSubmitError(new Error('API is not available'));
      return;
    }
    setIsSubmitting(true);
    setSubmitError(undefined);

    try {
      // Commit whatever is still sitting in the label box. Requiring Enter before
      // Save silently dropped the label the admin had just typed.
      const pendingLabel = labelInputValue.trim();
      const labels =
        pendingLabel && !formData.labels.includes(pendingLabel)
          ? [...formData.labels, pendingLabel]
          : formData.labels;

      const filledRepos = formData.repositories.filter((r) => r.url.trim().length > 0);
      const firstRepo = filledRepos[0] ?? {};
      // Strip repo-form-only fields before sending; they travel as source-level API fields
      // so the BFF writes them into each repository entry in the YAML.
      const repositories = filledRepos.map((r) => {
        const repo = { ...r };
        delete repo.provider;
        delete repo.category;
        delete repo.trustTier;
        return repo;
      });

      const optionalFields = {
        ...(firstRepo.trustTier ? { trustTier: firstRepo.trustTier } : {}),
        ...(firstRepo.provider ? { provider: firstRepo.provider } : {}),
        ...(firstRepo.category ? { category: firstRepo.category } : {}),
      };

      // generateSourceIdFromName strips characters a Kubernetes object name/Secret data
      // key can't contain (e.g. apostrophes); a plain slugify (lowercase + hyphenate
      // whitespace) lets them through and creating the source fails opaquely once a git
      // token is added, because the id is also used as the credential Secret's data key.
      const savedSourceId = isEditMode ? formData.id : generateSourceIdFromName(formData.name);

      if (isEditMode && existingSourceConfig) {
        await apiState.api.updateSkillCatalogSourceConfig({}, formData.id, {
          enabled: formData.enabled,
          repositories,
          // Always sent on edit, empty array included: the BFF treats an omitted
          // labels field as "leave as-is", so skipping it would make removing the
          // last label a silent no-op.
          labels,
          ...optionalFields,
        });
      } else {
        await apiState.api.createSkillCatalogSourceConfig(
          {},
          {
            id: savedSourceId,
            name: formData.name,
            type: SkillCatalogSourceType.GIT_SKILLS,
            enabled: formData.enabled,
            repositories,
            ...(labels.length > 0 ? { labels } : {}),
            ...optionalFields,
          },
        );
      }

      // The write only lands in a ConfigMap; the catalog reloads asynchronously.
      // Flag the source so the list shows "Syncing" rather than the stale status,
      // and pull the source list now instead of waiting out the poll interval.
      markSkillSourceSyncPending(savedSourceId);
      refreshSkillCatalogSourceConfigs();
      refreshSkillCatalogSources();
      navigate(skillCatalogSettingsUrl());
    } catch (error) {
      setSubmitError(error instanceof Error ? error : new Error('Failed to save source'));
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleCancel = () => {
    navigate(skillCatalogSettingsUrl());
  };

  return (
    <>
      <Sidebar hasBorder isPanelRight hasGutter>
        <SidebarContent>
          <Form isWidthLimited>
            <Stack hasGutter>
              <StackItem>
                <SkillSourceDetailsSection
                  duplicateNameError={duplicateNameError}
                  formData={formData}
                  setData={setData}
                  isEditMode={isEditMode}
                />
              </StackItem>

              <StackItem>
                <SkillRepositoriesSection
                  repositories={formData.repositories}
                  onChange={(repos) => setData('repositories', repos)}
                />
              </StackItem>

              <StackItem>
                <FormSection title="Source metadata">
                  <FormGroup label="Source labels" fieldId="skill-source-labels">
                    {formData.labels.length > 0 && (
                      <LabelGroup style={{ marginBottom: 'var(--pf-t--global--spacer--sm)' }}>
                        {formData.labels.map((label) => (
                          <Label key={label} onClose={() => removeLabel(label)}>
                            {label}
                          </Label>
                        ))}
                      </LabelGroup>
                    )}
                    <TextInputGroup>
                      <TextInputGroupMain
                        id="skill-source-labels-input"
                        data-testid="skill-source-labels-input"
                        value={labelInputValue}
                        onChange={(_event, value) => setLabelInputValue(value)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') {
                            e.preventDefault();
                            addLabel();
                          }
                        }}
                        placeholder="Type a label and press Enter"
                      />
                      {labelInputValue && (
                        <TextInputGroupUtilities>
                          <Button
                            variant="plain"
                            onClick={() => setLabelInputValue('')}
                            aria-label="Clear label input"
                          >
                            <TimesIcon />
                          </Button>
                        </TextInputGroupUtilities>
                      )}
                    </TextInputGroup>
                    <FormHelperText>
                      <HelperText>
                        <HelperTextItem>
                          Group this source in the skill catalog&apos;s source selector. To label
                          the skills themselves, use <b>Skill labels</b> on the repository above.
                        </HelperTextItem>
                      </HelperText>
                    </FormHelperText>
                  </FormGroup>
                </FormSection>
              </StackItem>

              <StackItem>
                <FormSection>
                  <FormGroup fieldId="skill-enable-source">
                    <Checkbox
                      label={<span className="pf-v6-c-form__label-text">Enable source</span>}
                      id="skill-enable-source"
                      name="skill-enable-source"
                      data-testid="skill-enable-source-checkbox"
                      description="Enable users in your organization to view skills from this source in the skill catalog."
                      isChecked={formData.enabled}
                      onChange={(_event, checked) => setData('enabled', checked)}
                    />
                  </FormGroup>
                </FormSection>
              </StackItem>
            </Stack>
          </Form>
        </SidebarContent>
        <SidebarPanel width={{ default: 'width_50' }}>
          <SkillPreviewPanel preview={preview} />
        </SidebarPanel>
      </Sidebar>
      <SkillManageSourceFormFooter
        submitLabel={isEditMode ? 'Save' : 'Add'}
        submitError={submitError}
        isSubmitDisabled={!isComplete || isSubmitting}
        isSubmitting={isSubmitting}
        onSubmit={handleSubmit}
        onCancel={handleCancel}
        isPreviewDisabled={!preview.canPreview}
        isPreviewLoading={preview.previewState.isLoadingInitial}
        onPreview={preview.handlePreview}
      />
    </>
  );
};

export default SkillManageSourceForm;
