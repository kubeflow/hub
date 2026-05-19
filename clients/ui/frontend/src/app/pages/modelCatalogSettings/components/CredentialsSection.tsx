import * as React from 'react';
import {
  TextInput,
  FormHelperText,
  HelperText,
  HelperTextItem,
  Button,
  ActionList,
  Alert,
  AlertActionCloseButton,
} from '@patternfly/react-core';
import { UpdateObjectAtPropAndValue, ThemeAwareFormGroupWrapper } from 'mod-arch-shared';
import PasswordInput from '~/app/shared/components/PasswordInput';
import FormSection from '~/app/pages/modelRegistry/components/pf-overrides/FormSection';
import { ManageSourceFormData } from '~/app/pages/modelCatalogSettings/useManageSourceData';
import { validateOrganization } from '~/app/pages/modelCatalogSettings/utils/validation';
import {
  FORM_LABELS,
  VALIDATION_MESSAGES,
  DESCRIPTION_TEXT,
  HELPER_TEXT,
  PLACEHOLDERS,
  ERROR_MESSAGES,
  SUCCESS_MESSAGES,
} from '~/app/pages/modelCatalogSettings/constants';
import { TempDevFeature, useTempDevFeatureAvailable } from '~/app/hooks/useTempDevFeatureAvailable';

type CredentialsSectionProps = {
  formData: ManageSourceFormData;
  setData: UpdateObjectAtPropAndValue<ManageSourceFormData>;
  onValidate: () => Promise<void>;
  isValidating: boolean;
  validationError?: Error;
  isValidationSuccess: boolean;
  onClearValidationSuccess: () => void;
};

const CredentialsSection: React.FC<CredentialsSectionProps> = ({
  formData,
  setData,
  onValidate,
  isValidating,
  validationError,
  isValidationSuccess,
  onClearValidationSuccess,
}) => {
  const [isOrganizationTouched, setIsOrganizationTouched] = React.useState(false);

  const isOrganizationValid = validateOrganization(formData.organization);

  const organizationInput = (
    <TextInput
      isRequired
      type="text"
      id="organization"
      name="organization"
      data-testid="organization-input"
      placeholder={PLACEHOLDERS.ORGANIZATION}
      value={formData.organization}
      onChange={(_event, value) => setData('organization', value)}
      onBlur={() => setIsOrganizationTouched(true)}
      validated={isOrganizationTouched && !isOrganizationValid ? 'error' : 'default'}
    />
  );

  const organizationDescriptionTxtNode = (
    <>
      <FormHelperText>
        <HelperText>
          <HelperTextItem>{DESCRIPTION_TEXT.ORGANIZATION}</HelperTextItem>
        </HelperText>
      </FormHelperText>
    </>
  );

  const organizationHelperTxtNode =
    isOrganizationTouched && !isOrganizationValid ? (
      <>
        <FormHelperText>
          <HelperText>
            <HelperTextItem variant="error" data-testid="organization-error">
              {VALIDATION_MESSAGES.ORGANIZATION_REQUIRED}
            </HelperTextItem>
          </HelperText>
        </FormHelperText>
      </>
    ) : undefined;

  const organizationFormGroup = (
    <>
      <ThemeAwareFormGroupWrapper
        label={FORM_LABELS.ORGANIZATION}
        fieldId="organization"
        isRequired
        hasError={!!organizationHelperTxtNode}
        helperTextNode={organizationHelperTxtNode}
      >
        {organizationDescriptionTxtNode}
        {organizationInput}
      </ThemeAwareFormGroupWrapper>
      <FormHelperText>
        <HelperText>
          <HelperTextItem>{HELPER_TEXT.ORGANIZATION_SLUG}</HelperTextItem>
        </HelperText>
      </FormHelperText>
    </>
  );

  const accessTokenInput = (
    <PasswordInput
      isRequired
      id="access-token"
      name="access-token"
      data-testid="access-token-input"
      value={formData.accessToken}
      onChange={(_event, value) => setData('accessToken', value)}
      ariaLabelShow="Show access token"
      ariaLabelHide="Hide access token"
    />
  );

  const accessTokenDescriptionTxtNode = (
    <FormHelperText>
      <HelperText>
        <HelperTextItem>{DESCRIPTION_TEXT.ACCESS_TOKEN}</HelperTextItem>
      </HelperText>
    </FormHelperText>
  );

  const accessTokenHelperTxtNode = (
    <FormHelperText>
      <HelperText>
        <HelperTextItem>{HELPER_TEXT.ACCESS_TOKEN}</HelperTextItem>
      </HelperText>
    </FormHelperText>
  );

  const accessTokenFormGroup = (
    <>
      <ThemeAwareFormGroupWrapper
        label={FORM_LABELS.ACCESS_TOKEN}
        fieldId="access-token"
        helperTextNode={accessTokenHelperTxtNode}
      >
        {accessTokenDescriptionTxtNode}
        {accessTokenInput}
      </ThemeAwareFormGroupWrapper>
      {validationError && (
        <Alert
          isInline
          variant="danger"
          title={ERROR_MESSAGES.VALIDATION_FAILED}
          className="pf-v6-u-mt-md"
        >
          {validationError.message}
        </Alert>
      )}
      {isValidationSuccess && (
        <Alert
          isInline
          variant="success"
          className="pf-v6-u-mt-md"
          title={SUCCESS_MESSAGES.VALIDATION_SUCCESSFUL}
          actionClose={<AlertActionCloseButton onClose={onClearValidationSuccess} />}
        >
          {SUCCESS_MESSAGES.VALIDATION_SUCCESSFUL_BODY}
        </Alert>
      )}

      <ActionList className="pf-v6-u-mt-md">
        <Button
          isDisabled={!isOrganizationValid || isValidating}
          variant="link"
          onClick={onValidate}
          isLoading={isValidating}
        >
          Validate
        </Button>
      </ActionList>
    </>
  );

  const accessTokenFeatureAvailable = useTempDevFeatureAvailable(
    TempDevFeature.CatalogHuggingFaceApiKey,
  );

  return (
    <FormSection
      title={accessTokenFeatureAvailable ? FORM_LABELS.CREDENTIALS : undefined}
      data-testid="credentials-section"
    >
      {organizationFormGroup}
      {accessTokenFeatureAvailable && accessTokenFormGroup}
    </FormSection>
  );
};

export default CredentialsSection;
