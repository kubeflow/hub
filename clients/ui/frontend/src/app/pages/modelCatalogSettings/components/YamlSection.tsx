import * as React from 'react';
import { UpdateObjectAtPropAndValue } from 'mod-arch-shared';
import { YamlUploadSection } from '~/app/shared/catalogSettings';
import { ManageSourceFormData } from '~/app/pages/modelCatalogSettings/useManageSourceData';
import {
  FORM_LABELS,
  VALIDATION_MESSAGES,
  ERROR_MESSAGES,
  EXPECTED_YAML_FORMAT_LABEL,
  HELPER_TEXT,
} from '~/app/pages/modelCatalogSettings/constants';

type YamlSectionProps = {
  formData: ManageSourceFormData;
  setData: UpdateObjectAtPropAndValue<ManageSourceFormData>;
  onToggleExpectedFormatDrawer?: () => void;
};

const YamlSection: React.FC<YamlSectionProps> = ({
  formData,
  setData,
  onToggleExpectedFormatDrawer,
}) => (
  <YamlUploadSection
    yamlContent={formData.yamlContent}
    onChange={(value) => setData('yamlContent', value)}
    onToggleExpectedFormatDrawer={onToggleExpectedFormatDrawer}
    yamlContentLabel={FORM_LABELS.YAML_CONTENT}
    expectedFormatLabel={EXPECTED_YAML_FORMAT_LABEL}
    yamlContentRequiredMessage={VALIDATION_MESSAGES.YAML_CONTENT_REQUIRED}
    yamlHelperText={HELPER_TEXT.YAML}
    fileUploadFailedTitle={ERROR_MESSAGES.FILE_UPLOAD_FAILED}
    fileUploadFailedBody={ERROR_MESSAGES.FILE_UPLOAD_FAILED_BODY}
  />
);

export default YamlSection;
