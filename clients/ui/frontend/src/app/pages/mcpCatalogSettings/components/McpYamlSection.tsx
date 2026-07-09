import * as React from 'react';
import {
  Alert,
  Button,
  Flex,
  FlexItem,
  FormGroup,
  FileUpload,
  FormHelperText,
  HelperText,
  HelperTextItem,
} from '@patternfly/react-core';
import { OpenDrawerRightIcon } from '@patternfly/react-icons';
import { UpdateObjectAtPropAndValue } from 'mod-arch-shared';
import { useThemeContext } from 'mod-arch-kubeflow';
import FormSection from '~/app/pages/modelRegistry/components/pf-overrides/FormSection';
import ThemeAwareFieldset from '~/app/pages/modelRegistry/screens/components/ThemeAwareFieldset';
import { ManageMcpSourceFormData } from '~/app/pages/mcpCatalogSettings/useManageMcpSourceData';
import { validateMcpYamlContent } from '~/app/pages/mcpCatalogSettings/utils/validation';
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
}) => {
  const { isMUITheme } = useThemeContext();
  const [isYamlTouched, setIsYamlTouched] = React.useState(false);
  const [filename, setFilename] = React.useState('');
  const [fileUploadError, setFileUploadError] = React.useState<string | undefined>(undefined);
  const isYamlContentValid = validateMcpYamlContent(formData.yamlContent);

  const handleFileChange = (
    _event: React.DragEvent<HTMLElement> | React.ChangeEvent<HTMLInputElement> | Event,
    file: File,
  ) => {
    setFilename(file.name);
    setFileUploadError(undefined);
    const reader = new FileReader();
    reader.onload = () => {
      const text = typeof reader.result === 'string' ? reader.result : '';
      setData('yamlContent', text);
      setIsYamlTouched(true);
    };
    reader.onerror = () => {
      setFileUploadError(MCP_ERROR_MESSAGES.FILE_UPLOAD_FAILED_BODY);
    };
    reader.readAsText(file);
  };

  const handleTextChange = (_event: React.ChangeEvent<HTMLTextAreaElement>, value: string) => {
    setData('yamlContent', value);
  };

  const handleClear = () => {
    setFilename('');
    setData('yamlContent', '');
    setIsYamlTouched(true);
    setFileUploadError(undefined);
  };

  const yamlInput = (
    <div data-testid="mcp-yaml-content-input">
      <FileUpload
        id="mcp-yaml-content"
        type="text"
        value={formData.yamlContent}
        filename={filename}
        filenamePlaceholder="Drag and drop a YAML file or upload one"
        onFileInputChange={handleFileChange}
        onTextChange={handleTextChange}
        onClearClick={handleClear}
        onBlur={() => setIsYamlTouched(true)}
        validated={isYamlTouched && !isYamlContentValid ? 'error' : 'default'}
        browseButtonText="Upload"
        allowEditingUploadedText
        dropzoneProps={{
          accept: { 'text/yaml': ['.yaml', '.yml'] },
        }}
      />
    </div>
  );

  const yamlHelperTxtNode =
    isYamlTouched && !isYamlContentValid ? (
      <FormHelperText>
        <HelperText>
          <HelperTextItem variant="error" data-testid="mcp-yaml-content-error">
            {MCP_VALIDATION_MESSAGES.YAML_CONTENT_REQUIRED}
          </HelperTextItem>
        </HelperText>
      </FormHelperText>
    ) : (
      <FormHelperText>
        <HelperText>
          <HelperTextItem>{MCP_HELPER_TEXT.YAML}</HelperTextItem>
        </HelperText>
      </FormHelperText>
    );

  const expectedFormatButton = onToggleExpectedFormatDrawer ? (
    <Button
      variant="link"
      isInline
      onClick={onToggleExpectedFormatDrawer}
      data-testid="mcp-view-expected-yaml-format-link"
      icon={<OpenDrawerRightIcon />}
      iconPosition="end"
    >
      {MCP_EXPECTED_YAML_FORMAT_LABEL}
    </Button>
  ) : null;

  return (
    <FormSection data-testid="mcp-yaml-section">
      {fileUploadError && (
        <Alert
          variant="danger"
          isInline
          title={MCP_ERROR_MESSAGES.FILE_UPLOAD_FAILED}
          className="pf-v6-u-mb-md"
          data-testid="mcp-yaml-file-upload-error"
        >
          {fileUploadError}
        </Alert>
      )}
      {isMUITheme && (
        <Flex
          justifyContent={{ default: 'justifyContentSpaceBetween' }}
          alignItems={{ default: 'alignItemsCenter' }}
          className="pf-v6-u-mb-sm"
        >
          <FlexItem>{MCP_FORM_LABELS.YAML_CONTENT}</FlexItem>
          <FlexItem>{expectedFormatButton}</FlexItem>
        </Flex>
      )}
      <FormGroup
        label={!isMUITheme ? MCP_FORM_LABELS.YAML_CONTENT : undefined}
        labelInfo={!isMUITheme ? expectedFormatButton : undefined}
        isRequired
        fieldId="mcp-yaml-content"
      >
        <ThemeAwareFieldset field="YAML">{yamlInput}</ThemeAwareFieldset>
        {yamlHelperTxtNode}
      </FormGroup>
    </FormSection>
  );
};

export default McpYamlSection;
