import * as React from 'react';
import {
  FormFieldGroupExpandable,
  FormFieldGroupHeader,
  TextArea,
  FormHelperText,
  HelperText,
  HelperTextItem,
} from '@patternfly/react-core';
import { UpdateObjectAtPropAndValue, ThemeAwareFormGroupWrapper } from 'mod-arch-shared';
import FormSection from '~/app/pages/modelRegistry/components/pf-overrides/FormSection';
import { ManageSourceFormData } from '~/app/pages/modelCatalogSettings/useManageSourceData';
import {
  FORM_LABELS,
  PLACEHOLDERS,
  DESCRIPTION_TEXT,
  getFilterInfoWithOrg,
  getAllowedModelsHelp,
  getExcludedModelsHelp,
  getIncludedModelsFieldHelperText,
  getExcludedModelsFieldHelperText,
} from '~/app/pages/modelCatalogSettings/constants';
import { CatalogSourceType } from '~/app/modelCatalogTypes';

type ModelVisibilitySectionProps = {
  formData: ManageSourceFormData;
  setData: UpdateObjectAtPropAndValue<ManageSourceFormData>;
  isDefaultExpanded?: boolean;
};

const ModelVisibilitySection: React.FC<ModelVisibilitySectionProps> = ({
  formData,
  setData,
  isDefaultExpanded = false,
}) => {
  const isHuggingFaceMode = formData.sourceType === CatalogSourceType.HUGGING_FACE;
  const organization = isHuggingFaceMode ? formData.organization : undefined;

  const sectionDescription =
    isHuggingFaceMode && organization
      ? getFilterInfoWithOrg(organization)
      : DESCRIPTION_TEXT.FILTER_INFO_GENERIC;

  const allowedModelsInput = (
    <TextArea
      id="allowed-models"
      name="allowed-models"
      data-testid="allowed-models-input"
      value={formData.allowedModels}
      onChange={(_event, value) => setData('allowedModels', value)}
      rows={3}
      resizeOrientation="vertical"
      placeholder={PLACEHOLDERS.ALLOWED_MODELS}
    />
  );

  const allowedModelsDescriptionTxtNode = (
    <FormHelperText>
      <HelperText>
        <HelperTextItem>{getAllowedModelsHelp(organization)}</HelperTextItem>
      </HelperText>
    </FormHelperText>
  );

  const allowedModelsHelperTxtNode = (
    <FormHelperText>
      <HelperText>
        <HelperTextItem>{getIncludedModelsFieldHelperText}</HelperTextItem>
      </HelperText>
    </FormHelperText>
  );

  const excludedModelsInput = (
    <TextArea
      id="excluded-models"
      name="excluded-models"
      data-testid="excluded-models-input"
      value={formData.excludedModels}
      onChange={(_event, value) => setData('excludedModels', value)}
      rows={3}
      resizeOrientation="vertical"
      placeholder={PLACEHOLDERS.EXCLUDED_MODELS}
    />
  );

  const excludedModelsDescriptionTxtNode = (
    <FormHelperText>
      <HelperText>
        <HelperTextItem>{getExcludedModelsHelp(organization)}</HelperTextItem>
      </HelperText>
    </FormHelperText>
  );

  const excludedModelsHelperTxtNode = (
    <FormHelperText>
      <HelperText>
        <HelperTextItem>{getExcludedModelsFieldHelperText}</HelperTextItem>
      </HelperText>
    </FormHelperText>
  );

  return (
    <FormSection>
      <FormFieldGroupExpandable
        toggleAriaLabel="Model visibility"
        header={
          <FormFieldGroupHeader
            titleText={{ text: FORM_LABELS.MODEL_VISIBILITY, id: 'model-visibility-title' }}
            titleDescription={sectionDescription}
          />
        }
        isExpanded={isDefaultExpanded}
        data-testid="model-visibility-section"
      >
        <ThemeAwareFormGroupWrapper
          label={FORM_LABELS.ALLOWED_MODELS}
          fieldId="allowed-models"
          helperTextNode={allowedModelsHelperTxtNode}
        >
          {allowedModelsDescriptionTxtNode}
          {allowedModelsInput}
        </ThemeAwareFormGroupWrapper>

        <ThemeAwareFormGroupWrapper
          label={FORM_LABELS.EXCLUDED_MODELS}
          fieldId="excluded-models"
          helperTextNode={excludedModelsHelperTxtNode}
        >
          {excludedModelsDescriptionTxtNode}
          {excludedModelsInput}
        </ThemeAwareFormGroupWrapper>
      </FormFieldGroupExpandable>
    </FormSection>
  );
};

export default ModelVisibilitySection;
