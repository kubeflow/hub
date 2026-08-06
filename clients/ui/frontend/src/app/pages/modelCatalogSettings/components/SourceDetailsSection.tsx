import * as React from 'react';
import {
  FormGroup,
  TextInput,
  Radio,
  FormHelperText,
  HelperText,
  HelperTextItem,
  Flex,
  FlexItem,
} from '@patternfly/react-core';
import { UpdateObjectAtPropAndValue, ThemeAwareFormGroupWrapper } from 'mod-arch-shared';
import FormSection from '~/app/pages/modelRegistry/components/pf-overrides/FormSection';
import { ManageSourceFormData } from '~/app/pages/modelCatalogSettings/useManageSourceData';
import { validateSourceName } from '~/app/pages/modelCatalogSettings/utils/validation';
import {
  FORM_LABELS,
  SOURCE_TYPE_LABELS,
  VALIDATION_MESSAGES,
  SOURCE_NAME_CHARACTER_LIMIT,
} from '~/app/pages/modelCatalogSettings/constants';
import { isSourceNameEmpty } from '~/app/shared/catalogSettings/utils/validation';
import { CatalogSourceType } from '~/app/modelCatalogTypes';

type SourceDetailsSectionProps = {
  formData: ManageSourceFormData;
  setData: UpdateObjectAtPropAndValue<ManageSourceFormData>;
  isEditMode: boolean;
};

const SourceDetailsSection: React.FC<SourceDetailsSectionProps> = ({
  formData,
  setData,
  isEditMode,
}) => {
  const [isNameTouched, setIsNameTouched] = React.useState(false);
  const isNameValid = validateSourceName(formData.name);
  const hasNameError = isNameTouched && !isNameValid;

  const nameInput = (
    <TextInput
      isRequired
      readOnlyVariant={formData.isDefault ? 'plain' : undefined}
      type="text"
      id="source-name"
      name="source-name"
      data-testid="source-name-input"
      value={formData.name}
      onChange={(_event, value) => setData('name', value)}
      onBlur={() => setIsNameTouched(true)}
      validated={hasNameError ? 'error' : 'default'}
    />
  );

  const nameHelperTextNode = hasNameError ? (
    <FormHelperText>
      <HelperText>
        <HelperTextItem variant="error" data-testid="source-name-error">
          {isSourceNameEmpty(formData.name)
            ? VALIDATION_MESSAGES.NAME_REQUIRED
            : formData.name.length > SOURCE_NAME_CHARACTER_LIMIT
              ? `Cannot exceed ${SOURCE_NAME_CHARACTER_LIMIT} characters`
              : null}
        </HelperTextItem>
      </HelperText>
    </FormHelperText>
  ) : undefined;

  return (
    <FormSection>
      <ThemeAwareFormGroupWrapper
        label={FORM_LABELS.NAME}
        fieldId="source-name"
        isRequired
        hasError={hasNameError}
        helperTextNode={nameHelperTextNode}
      >
        {nameInput}
      </ThemeAwareFormGroupWrapper>

      <FormGroup
        label={FORM_LABELS.SOURCE_TYPE}
        isRequired
        fieldId="source-type"
        role="radiogroup"
        aria-labelledby="source-type-label"
      >
        {isEditMode ? (
          <span>
            {formData.sourceType === CatalogSourceType.HUGGING_FACE
              ? SOURCE_TYPE_LABELS.HUGGING_FACE
              : SOURCE_TYPE_LABELS.YAML}
          </span>
        ) : (
          <Flex spaceItems={{ default: 'spaceItemsMd' }}>
            <FlexItem>
              <Radio
                isChecked={formData.sourceType === CatalogSourceType.HUGGING_FACE}
                name="source-type"
                onChange={() => setData('sourceType', CatalogSourceType.HUGGING_FACE)}
                label={SOURCE_TYPE_LABELS.HUGGING_FACE}
                id="source-type-huggingface"
                data-testid="source-type-huggingface"
              />
            </FlexItem>
            <FlexItem>
              <Radio
                isChecked={formData.sourceType === CatalogSourceType.YAML}
                name="source-type"
                onChange={() => setData('sourceType', CatalogSourceType.YAML)}
                label={SOURCE_TYPE_LABELS.YAML}
                id="source-type-yaml"
                data-testid="source-type-yaml"
              />
            </FlexItem>
          </Flex>
        )}
      </FormGroup>
    </FormSection>
  );
};

export default SourceDetailsSection;
