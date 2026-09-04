import * as React from 'react';
import { UpdateObjectAtPropAndValue } from 'mod-arch-shared';
import { IncludeExcludeFiltersSection } from '~/app/shared/catalogSettings';
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

  return (
    <IncludeExcludeFiltersSection
      includedValue={formData.allowedModels}
      onIncludedChange={(value) => setData('allowedModels', value)}
      excludedValue={formData.excludedModels}
      onExcludedChange={(value) => setData('excludedModels', value)}
      isDefaultExpanded={isDefaultExpanded}
      sectionTitle={FORM_LABELS.MODEL_VISIBILITY}
      sectionTitleId="model-visibility-title"
      sectionDescription={sectionDescription}
      includedFieldId="allowed-models"
      excludedFieldId="excluded-models"
      includedLabel={FORM_LABELS.ALLOWED_MODELS}
      excludedLabel={FORM_LABELS.EXCLUDED_MODELS}
      includedDescription={getAllowedModelsHelp(organization)}
      excludedDescription={getExcludedModelsHelp(organization)}
      includedHelperText={getIncludedModelsFieldHelperText}
      excludedHelperText={getExcludedModelsFieldHelperText}
      includedPlaceholder={PLACEHOLDERS.ALLOWED_MODELS}
      excludedPlaceholder={PLACEHOLDERS.EXCLUDED_MODELS}
      testIds={{
        section: 'model-visibility-section',
        includedInput: 'allowed-models-input',
        excludedInput: 'excluded-models-input',
      }}
    />
  );
};

export default ModelVisibilitySection;
