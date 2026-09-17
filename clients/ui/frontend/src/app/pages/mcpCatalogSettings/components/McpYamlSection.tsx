import * as React from 'react';
import { UpdateObjectAtPropAndValue } from 'mod-arch-shared';
import { YamlUploadSection } from '~/app/shared/catalogSettings';
import { ManageMcpSourceFormData } from '~/app/pages/mcpCatalogSettings/useManageMcpSourceData';
import {
  MCP_FORM_LABELS,
  MCP_VALIDATION_MESSAGES,
  MCP_ERROR_MESSAGES,
  MCP_EXPECTED_YAML_FORMAT_LABEL,
  MCP_HELPER_TEXT,
} from '~/app/pages/mcpCatalogSettings/constants';

type McpYamlSectionProps = {
  formData: ManageMcpSourceFormData;
  setData: UpdateObjectAtPropAndValue<ManageMcpSourceFormData>;
  onToggleExpectedFormatDrawer?: () => void;
};

const McpYamlSection: React.FC<McpYamlSectionProps> = ({
  formData,
  setData,
  onToggleExpectedFormatDrawer,
}) => (
  <YamlUploadSection
    yamlContent={formData.yamlContent}
    onChange={(value) => setData('yamlContent', value)}
    onToggleExpectedFormatDrawer={onToggleExpectedFormatDrawer}
    fieldId="mcp-yaml-content"
    yamlContentLabel={MCP_FORM_LABELS.YAML_CONTENT}
    expectedFormatLabel={MCP_EXPECTED_YAML_FORMAT_LABEL}
    yamlContentRequiredMessage={MCP_VALIDATION_MESSAGES.YAML_CONTENT_REQUIRED}
    yamlHelperText={MCP_HELPER_TEXT.YAML}
    fileUploadFailedTitle={MCP_ERROR_MESSAGES.FILE_UPLOAD_FAILED}
    fileUploadFailedBody={MCP_ERROR_MESSAGES.FILE_UPLOAD_FAILED_BODY}
    testIds={{
      section: 'mcp-yaml-section',
      contentInput: 'mcp-yaml-content-input',
      contentError: 'mcp-yaml-content-error',
      fileUploadError: 'mcp-yaml-file-upload-error',
      expectedFormatLink: 'mcp-view-expected-yaml-format-link',
    }}
  />
);

export default McpYamlSection;
