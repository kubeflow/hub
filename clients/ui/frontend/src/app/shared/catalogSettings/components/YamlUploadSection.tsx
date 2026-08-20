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
import { useThemeContext } from 'mod-arch-kubeflow';
import FormSection from '~/app/pages/modelRegistry/components/pf-overrides/FormSection';
import ThemeAwareFieldset from '~/app/pages/modelRegistry/screens/components/ThemeAwareFieldset';
import { validateYamlContent } from '~/app/shared/catalogSettings/utils/validation';

export type YamlUploadSectionTestIds = {
  section?: string;
  contentInput?: string;
  contentError?: string;
  fileUploadError?: string;
  expectedFormatLink?: string;
};

type YamlUploadSectionProps = {
  yamlContent: string;
  onChange: (value: string) => void;
  onToggleExpectedFormatDrawer?: () => void;
  /** Base id for the FileUpload input; defaults to 'yaml-content'. */
  fieldId?: string;
  yamlContentLabel: string;
  expectedFormatLabel: string;
  yamlContentRequiredMessage: string;
  yamlHelperText: React.ReactNode;
  fileUploadFailedTitle: string;
  fileUploadFailedBody: string;
  testIds?: YamlUploadSectionTestIds;
};

const YamlUploadSection: React.FC<YamlUploadSectionProps> = ({
  yamlContent,
  onChange,
  onToggleExpectedFormatDrawer,
  fieldId = 'yaml-content',
  yamlContentLabel,
  expectedFormatLabel,
  yamlContentRequiredMessage,
  yamlHelperText,
  fileUploadFailedTitle,
  fileUploadFailedBody,
  testIds,
}) => {
  const { isMUITheme } = useThemeContext();
  const [isYamlTouched, setIsYamlTouched] = React.useState(false);
  const [filename, setFilename] = React.useState('');
  const [fileUploadError, setFileUploadError] = React.useState<string | undefined>(undefined);
  const isYamlContentValid = validateYamlContent(yamlContent);

  const {
    section = 'yaml-section',
    contentInput = 'yaml-content-input',
    contentError = 'yaml-content-error',
    fileUploadError: fileUploadErrorTestId = 'yaml-file-upload-error',
    expectedFormatLink = 'view-expected-yaml-format-link',
  } = testIds ?? {};

  const handleFileChange = (
    _event: React.DragEvent<HTMLElement> | React.ChangeEvent<HTMLInputElement> | Event,
    file: File,
  ) => {
    setFilename(file.name);
    setFileUploadError(undefined);
    const reader = new FileReader();
    reader.onload = () => {
      const text = typeof reader.result === 'string' ? reader.result : '';
      onChange(text);
      setIsYamlTouched(true);
    };
    reader.onerror = () => {
      setFileUploadError(fileUploadFailedBody);
    };
    reader.readAsText(file);
  };

  const handleTextChange = (_event: React.ChangeEvent<HTMLTextAreaElement>, value: string) => {
    onChange(value);
  };

  const handleClear = () => {
    setFilename('');
    onChange('');
    setIsYamlTouched(true);
    setFileUploadError(undefined);
  };

  const yamlInput = (
    <div data-testid={contentInput}>
      <FileUpload
        id={fieldId}
        type="text"
        value={yamlContent}
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
          <HelperTextItem variant="error" data-testid={contentError}>
            {yamlContentRequiredMessage}
          </HelperTextItem>
        </HelperText>
      </FormHelperText>
    ) : (
      <FormHelperText>
        <HelperText>
          <HelperTextItem>{yamlHelperText}</HelperTextItem>
        </HelperText>
      </FormHelperText>
    );

  const expectedFormatButton = onToggleExpectedFormatDrawer ? (
    <Button
      variant="link"
      isInline
      onClick={onToggleExpectedFormatDrawer}
      data-testid={expectedFormatLink}
      icon={<OpenDrawerRightIcon />}
      iconPosition="end"
    >
      {expectedFormatLabel}
    </Button>
  ) : null;

  return (
    <FormSection data-testid={section}>
      {fileUploadError && (
        <Alert
          variant="danger"
          isInline
          title={fileUploadFailedTitle}
          className="pf-v6-u-mb-md"
          data-testid={fileUploadErrorTestId}
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
          <FlexItem>{yamlContentLabel}</FlexItem>
          <FlexItem>{expectedFormatButton}</FlexItem>
        </Flex>
      )}
      <FormGroup
        label={!isMUITheme ? yamlContentLabel : undefined}
        labelInfo={!isMUITheme ? expectedFormatButton : undefined}
        isRequired
        fieldId={fieldId}
      >
        <ThemeAwareFieldset field="YAML">{yamlInput}</ThemeAwareFieldset>
        {yamlHelperTxtNode}
      </FormGroup>
    </FormSection>
  );
};

export default YamlUploadSection;
