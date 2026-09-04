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
  Modal,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalVariant,
} from '@patternfly/react-core';
import { InfoCircleIcon } from '@patternfly/react-icons';
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
  CLEAR_ACCESS_TOKEN_MODAL,
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
  const [isClearModalOpen, setIsClearModalOpen] = React.useState(false);

  const isOrganizationValid = validateOrganization(formData.organization);

  const onClearToken = React.useCallback(() => {
    setData('accessToken', '');
    onClearValidationSuccess();
  }, [setData, onClearValidationSuccess]);

  const accessTokenFeatureAvailable = useTempDevFeatureAvailable(
    TempDevFeature.CatalogHuggingFaceApiKey,
  );

  if (!accessTokenFeatureAvailable) {
    return (
      <>
        <ThemeAwareFormGroupWrapper
          label={FORM_LABELS.ORGANIZATION}
          fieldId="organization"
          isRequired
          hasError={isOrganizationTouched && !isOrganizationValid}
          helperTextNode={
            isOrganizationTouched && !isOrganizationValid ? (
              <FormHelperText>
                <HelperText>
                  <HelperTextItem variant="error" data-testid="organization-error">
                    {VALIDATION_MESSAGES.ORGANIZATION_REQUIRED}
                  </HelperTextItem>
                </HelperText>
              </FormHelperText>
            ) : undefined
          }
          popoverHelpText={DESCRIPTION_TEXT.ORGANIZATION}
        >
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
        </ThemeAwareFormGroupWrapper>
        <FormHelperText>
          <HelperText>
            <HelperTextItem>{HELPER_TEXT.ORGANIZATION_SLUG}</HelperTextItem>
          </HelperText>
        </FormHelperText>
      </>
    );
  }

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

  const formGroupOrgHelpTextNode = (
    <>
      <FormHelperText>
        <HelperText>
          <HelperTextItem>{HELPER_TEXT.ORGANIZATION_SLUG}</HelperTextItem>
        </HelperText>
      </FormHelperText>
    </>
  );

  const organizationFormGroup = (
    <>
      <ThemeAwareFormGroupWrapper
        label={FORM_LABELS.ORGANIZATION}
        fieldId="organization"
        isRequired
        hasError={!!organizationHelperTxtNode}
        helperTextNode={organizationHelperTxtNode}
        popoverHelpText={DESCRIPTION_TEXT.ORGANIZATION}
      >
        {organizationInput}
      </ThemeAwareFormGroupWrapper>
      {formGroupOrgHelpTextNode}
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
      isDisabled={isValidationSuccess}
      hideToggleButton={isValidationSuccess}
      forceHidden={isValidationSuccess}
    />
  );

  const accessTokenHelperTxtNode = isValidationSuccess ? (
    <FormHelperText>
      <HelperText>
        <HelperTextItem data-testid="access-token-hidden-helper" icon={<InfoCircleIcon />}>
          {HELPER_TEXT.ACCESS_TOKEN_HIDDEN}
        </HelperTextItem>
      </HelperText>
    </FormHelperText>
  ) : undefined;

  const tokenValidationBtn = isValidationSuccess ? undefined : (
    <Button
      isDisabled={!isOrganizationValid || isValidating}
      variant="link"
      onClick={onValidate}
      isLoading={isValidating}
    >
      Validate
    </Button>
  );

  const tokenClearBtn = !isValidationSuccess ? undefined : (
    <Button
      isDisabled={!isOrganizationValid || isValidating}
      variant="link"
      onClick={() => setIsClearModalOpen(true)}
    >
      Clear
    </Button>
  );

  const accessTokenFormGroup = (
    <>
      <ThemeAwareFormGroupWrapper
        label={FORM_LABELS.ACCESS_TOKEN}
        fieldId="access-token"
        helperTextNode={accessTokenHelperTxtNode}
        popoverHelpText={DESCRIPTION_TEXT.ACCESS_TOKEN}
      >
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
          className="pf-v6-u"
          title={SUCCESS_MESSAGES.VALIDATION_SUCCESSFUL}
          actionClose={<AlertActionCloseButton onClose={onClearValidationSuccess} />}
        >
          {SUCCESS_MESSAGES.VALIDATION_SUCCESSFUL_BODY}
        </Alert>
      )}

      <ActionList className="pf-v6-u">
        {tokenValidationBtn}
        {tokenClearBtn}
      </ActionList>
    </>
  );

  return (
    <>
      <FormSection title={FORM_LABELS.CREDENTIALS} data-testid="credentials-section">
        {organizationFormGroup}
        {accessTokenFormGroup}
      </FormSection>
      <Modal
        variant={ModalVariant.small}
        isOpen={isClearModalOpen}
        onClose={() => setIsClearModalOpen(false)}
        data-testid="clear-access-token-modal"
      >
        <ModalHeader title={CLEAR_ACCESS_TOKEN_MODAL.MODAL_TITLE} />
        <ModalBody>{CLEAR_ACCESS_TOKEN_MODAL.MODAL_BODY}</ModalBody>
        <ModalFooter>
          <Button
            variant="danger"
            onClick={() => {
              onClearToken();
              setIsClearModalOpen(false);
            }}
            data-testid="clear-access-token-confirm-button"
          >
            {CLEAR_ACCESS_TOKEN_MODAL.CONFIRM_BTN}
          </Button>
          <Button variant="link" onClick={() => setIsClearModalOpen(false)}>
            {CLEAR_ACCESS_TOKEN_MODAL.CANCEL_BTN}
          </Button>
        </ModalFooter>
      </Modal>
    </>
  );
};

export default CredentialsSection;
