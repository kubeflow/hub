import * as React from 'react';
import {
  Form,
  FormGroup,
  Checkbox,
  Stack,
  StackItem,
  Sidebar,
  SidebarPanel,
  SidebarContent,
} from '@patternfly/react-core';
import { useNavigate } from 'react-router-dom';
import FormSection from '~/app/pages/modelRegistry/components/pf-overrides/FormSection';
import { catalogSettingsUrl } from '~/app/routes/modelCatalogSettings/modelCatalogSettings';
import { isFormValid } from '~/app/pages/modelCatalogSettings/utils/validation';
import { useManageSourceData } from '~/app/pages/modelCatalogSettings/useManageSourceData';
import { useSourcePreview } from '~/app/pages/modelCatalogSettings/useSourcePreview';
import {
  FORM_LABELS,
  DESCRIPTION_TEXT,
  ERROR_MESSAGES,
} from '~/app/pages/modelCatalogSettings/constants';
import { ModelCatalogSettingsContext } from '~/app/context/modelCatalogSettings/ModelCatalogSettingsContext';
import {
  catalogSourceConfigToFormData,
  getPayloadForConfig,
  transformFormDataToConfig,
} from '~/app/pages/modelCatalogSettings/utils/modelCatalogSettingsUtils';
import { CatalogSourceConfig, CatalogSourceType } from '~/app/modelCatalogTypes';
import SourceDetailsSection from './SourceDetailsSection';
import CredentialsSection from './CredentialsSection';
import YamlSection from './YamlSection';
import ModelVisibilitySection from './ModelVisibilitySection';
import PreviewPanel from './PreviewPanel';
import ManageSourceFormFooter from './ManageSourceFormFooter';

type ManageSourceFormProps = {
  existingSourceConfig?: CatalogSourceConfig;
  isEditMode: boolean;
  onToggleExpectedFormatDrawer?: () => void;
};

const ManageSourceForm: React.FC<ManageSourceFormProps> = ({
  existingSourceConfig,
  isEditMode,
  onToggleExpectedFormatDrawer,
}) => {
  const navigate = useNavigate();
  const existingData = existingSourceConfig
    ? catalogSourceConfigToFormData(existingSourceConfig)
    : undefined;
  const [formData, setData] = useManageSourceData(existingData);
  const [isSubmitting, setIsSubmitting] = React.useState(false);
  const [submitError, setSubmitError] = React.useState<Error | undefined>(undefined);
  const {
    apiState,
    catalogSources,
    refreshCatalogSourceConfigs,
    refreshCatalogSources,
    markSourcePending,
  } = React.useContext(ModelCatalogSettingsContext);

  // Use the preview hook
  const preview = useSourcePreview({
    formData,
    existingSourceConfig,
    apiState,
    isEditMode,
  });

  const isHuggingFaceMode = formData.sourceType === CatalogSourceType.HUGGING_FACE;
  const isFormComplete = isFormValid(formData);

  const handleSubmit = async () => {
    if (!apiState.apiAvailable) {
      setSubmitError(new Error('API is not available'));
      return;
    }
    setIsSubmitting(true);
    setSubmitError(undefined);

    try {
      const sourceConfig = transformFormDataToConfig(formData, existingSourceConfig);
      const payload = getPayloadForConfig(sourceConfig, isEditMode);

      if (isEditMode && existingData) {
        const previousStatus =
          catalogSources?.items?.find((s) => s.id === formData.id)?.status ?? '';
        await apiState.api.updateCatalogSourceConfig({}, formData.id, payload);
        const validationFieldsChanged =
          existingData!.sourceType !== formData.sourceType ||
          existingData!.yamlContent !== formData.yamlContent ||
          existingData!.accessToken !== formData.accessToken ||
          existingData!.organization !== formData.organization ||
          existingData!.allowedModels !== formData.allowedModels ||
          existingData!.excludedModels !== formData.excludedModels ||
          existingData!.enabled !== formData.enabled;
        if (validationFieldsChanged) {
          markSourcePending(formData.id, previousStatus);
        }
      } else {
        await apiState.api.createCatalogSourceConfig({}, payload);
      }

      refreshCatalogSourceConfigs();
      refreshCatalogSources();
      navigate(catalogSettingsUrl());
    } catch (error) {
      setSubmitError(error instanceof Error ? error : new Error(ERROR_MESSAGES.SAVE_FAILED));
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleCancel = () => {
    navigate(catalogSettingsUrl());
  };

  return (
    <>
      <Sidebar hasBorder isPanelRight hasGutter>
        <SidebarContent>
          <Form isWidthLimited>
            <Stack hasGutter>
              <StackItem>
                <SourceDetailsSection
                  formData={formData}
                  setData={setData}
                  isEditMode={isEditMode}
                />
              </StackItem>

              {isHuggingFaceMode && (
                <StackItem>
                  <CredentialsSection
                    formData={formData}
                    setData={setData}
                    onValidate={preview.handleValidate}
                    isValidating={preview.isValidating}
                    validationError={preview.validationError}
                    isValidationSuccess={preview.isValidationSuccess}
                    onClearValidationSuccess={preview.clearValidationSuccess}
                  />
                </StackItem>
              )}

              {!formData.isDefault && !isHuggingFaceMode && (
                <StackItem>
                  <YamlSection
                    formData={formData}
                    setData={setData}
                    onToggleExpectedFormatDrawer={onToggleExpectedFormatDrawer}
                  />
                </StackItem>
              )}

              <StackItem>
                <FormSection>
                  <FormGroup fieldId="enable-source">
                    <Checkbox
                      label={
                        <span className="pf-v6-c-form__label-text">
                          {FORM_LABELS.ENABLE_SOURCE}
                        </span>
                      }
                      id="enable-source"
                      name="enable-source"
                      data-testid="enable-source-checkbox"
                      description={DESCRIPTION_TEXT.ENABLE_SOURCE}
                      isChecked={formData.enabled}
                      onChange={(_event, checked) => setData('enabled', checked)}
                    />
                  </FormGroup>
                </FormSection>
              </StackItem>
              <StackItem>
                <ModelVisibilitySection
                  formData={formData}
                  setData={setData}
                  isDefaultExpanded={
                    existingData?.isDefault ||
                    !!existingData?.allowedModels ||
                    !!existingData?.excludedModels
                  }
                />
              </StackItem>
            </Stack>
          </Form>
        </SidebarContent>
        <SidebarPanel width={{ default: 'width_50' }}>
          <PreviewPanel preview={preview} />
        </SidebarPanel>
      </Sidebar>
      <ManageSourceFormFooter
        submitLabel={isEditMode ? 'Save' : 'Add'}
        submitError={submitError}
        isSubmitDisabled={!isFormComplete || isSubmitting}
        isSubmitting={isSubmitting}
        onSubmit={handleSubmit}
        onCancel={handleCancel}
        isPreviewDisabled={!preview.canPreview}
        isPreviewLoading={preview.previewState.isLoadingInitial}
        onPreview={() => preview.handlePreview()}
      />
    </>
  );
};

export default ManageSourceForm;
