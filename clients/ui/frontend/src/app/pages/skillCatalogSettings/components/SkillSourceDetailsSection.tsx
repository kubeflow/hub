import * as React from 'react';
import {
  FormGroup,
  TextInput,
  FormHelperText,
  HelperText,
  HelperTextItem,
} from '@patternfly/react-core';
import { UpdateObjectAtPropAndValue, ThemeAwareFormGroupWrapper } from 'mod-arch-shared';
import FormSection from '~/app/pages/modelRegistry/components/pf-overrides/FormSection';
import {
  validateSourceName,
  isSourceNameEmpty,
  SOURCE_NAME_CHARACTER_LIMIT,
} from '~/app/shared/catalogSettings';
import type { ManageSkillSourceFormData } from '~/app/pages/skillCatalogSettings/useManageSkillSourceData';

type SkillSourceDetailsSectionProps = {
  formData: ManageSkillSourceFormData;
  setData: UpdateObjectAtPropAndValue<ManageSkillSourceFormData>;
  isEditMode: boolean;
  /**
   * Set when the name would derive a source id that an existing source already uses.
   * Surfaced on the name field because the id is derived from the name and is not
   * editable: without this the collision only appears as a "already exists" error
   * after submitting.
   */
  duplicateNameError?: string;
};

const SkillSourceDetailsSection: React.FC<SkillSourceDetailsSectionProps> = ({
  formData,
  setData,
  isEditMode,
  duplicateNameError,
}) => {
  const [isNameTouched, setIsNameTouched] = React.useState(false);
  const isNameValid = validateSourceName(formData.name, SOURCE_NAME_CHARACTER_LIMIT);
  const hasNameError = isNameTouched && !isNameValid;
  // A duplicate shows as soon as it is known, without waiting for a blur: the admin has
  // typed a complete name that cannot be saved, and the sooner that is visible the less
  // of the rest of the form they fill in first.
  const showDuplicate = Boolean(duplicateNameError) && !hasNameError;

  const nameInput = (
    <TextInput
      isRequired
      readOnlyVariant={isEditMode ? 'plain' : undefined}
      type="text"
      id="skill-source-name"
      name="skill-source-name"
      data-testid="skill-source-name-input"
      value={formData.name}
      onChange={(_event, value) => setData('name', value)}
      onBlur={() => setIsNameTouched(true)}
      validated={hasNameError || showDuplicate ? 'error' : 'default'}
    />
  );

  const nameHelperTextNode = showDuplicate ? (
    <FormHelperText>
      <HelperText>
        <HelperTextItem variant="error" data-testid="skill-source-name-duplicate">
          {duplicateNameError}
        </HelperTextItem>
      </HelperText>
    </FormHelperText>
  ) : hasNameError ? (
    <FormHelperText>
      <HelperText>
        <HelperTextItem variant="error" data-testid="skill-source-name-error">
          {isSourceNameEmpty(formData.name)
            ? 'Name is required'
            : `Cannot exceed ${SOURCE_NAME_CHARACTER_LIMIT} characters`}
        </HelperTextItem>
      </HelperText>
    </FormHelperText>
  ) : undefined;

  return (
    <FormSection>
      <ThemeAwareFormGroupWrapper
        label="Name"
        fieldId="skill-source-name"
        isRequired
        hasError={hasNameError}
        helperTextNode={nameHelperTextNode}
      >
        {nameInput}
      </ThemeAwareFormGroupWrapper>

      <FormGroup label="Source type" fieldId="skill-source-type">
        <TextInput
          readOnlyVariant="plain"
          type="text"
          id="skill-source-type"
          data-testid="skill-source-type"
          value="Git repository"
        />
      </FormGroup>
    </FormSection>
  );
};

export default SkillSourceDetailsSection;
