import * as React from 'react';
import {
  FormGroup,
  TextInput,
  FormHelperText,
  HelperText,
  HelperTextItem,
  Content,
} from '@patternfly/react-core';
import { UpdateObjectAtPropAndValue, ThemeAwareFormGroupWrapper } from 'mod-arch-shared';
import FormSection from '~/app/pages/modelRegistry/components/pf-overrides/FormSection';
import { ManageMcpSourceFormData } from '~/app/pages/mcpCatalogSettings/useManageMcpSourceData';
import {
  validateMcpSourceName,
  isMcpSourceNameEmpty,
} from '~/app/pages/mcpCatalogSettings/utils/validation';
import {
  MCP_FORM_LABELS,
  MCP_VALIDATION_MESSAGES,
  MCP_SOURCE_NAME_CHARACTER_LIMIT,
} from '~/app/pages/mcpCatalogSettings/constants';
import { McpCatalogSourceConfig } from '~/app/mcpServerCatalogTypes';

type McpSourceDetailsSectionProps = {
  formData: ManageMcpSourceFormData;
  setData: UpdateObjectAtPropAndValue<ManageMcpSourceFormData>;
  isEditMode: boolean;
  existingSourceConfig?: McpCatalogSourceConfig;
  serverCount?: number;
};

const McpSourceDetailsSection: React.FC<McpSourceDetailsSectionProps> = ({
  formData,
  setData,
  isEditMode,
  existingSourceConfig,
  serverCount,
}) => {
  const [isNameTouched, setIsNameTouched] = React.useState(false);
  const isNameValid = validateMcpSourceName(formData.name);
  const hasNameError = isNameTouched && !isNameValid;

  const nameInput = (
    <TextInput
      isRequired
      readOnlyVariant={formData.isDefault ? 'plain' : undefined}
      type="text"
      id="mcp-source-name"
      name="mcp-source-name"
      data-testid="mcp-source-name-input"
      value={formData.name}
      onChange={(_event, value) => setData('name', value)}
      onBlur={() => setIsNameTouched(true)}
      validated={hasNameError ? 'error' : 'default'}
    />
  );

  const nameHelperTextNode = hasNameError ? (
    <FormHelperText>
      <HelperText>
        <HelperTextItem variant="error" data-testid="mcp-source-name-error">
          {isMcpSourceNameEmpty(formData.name)
            ? MCP_VALIDATION_MESSAGES.NAME_REQUIRED
            : formData.name.length > MCP_SOURCE_NAME_CHARACTER_LIMIT
              ? `Cannot exceed ${MCP_SOURCE_NAME_CHARACTER_LIMIT} characters`
              : null}
        </HelperTextItem>
      </HelperText>
    </FormHelperText>
  ) : undefined;

  return (
    <FormSection>
      <ThemeAwareFormGroupWrapper
        label={MCP_FORM_LABELS.NAME}
        fieldId="mcp-source-name"
        isRequired={!formData.isDefault}
        hasError={hasNameError}
        helperTextNode={nameHelperTextNode}
      >
        {nameInput}
      </ThemeAwareFormGroupWrapper>

      {isEditMode &&
        existingSourceConfig &&
        formData.isDefault &&
        existingSourceConfig.yamlCatalogPath && (
          <FormGroup label={MCP_FORM_LABELS.CATALOG_YAML_FILE} fieldId="mcp-catalog-yaml-file">
            <TextInput
              readOnlyVariant="plain"
              type="text"
              id="mcp-catalog-yaml-file"
              data-testid="mcp-catalog-yaml-file"
              value={existingSourceConfig.yamlCatalogPath}
            />
          </FormGroup>
        )}

      {isEditMode && formData.isDefault && serverCount !== undefined && (
        <FormGroup label={MCP_FORM_LABELS.MCP_SERVERS} fieldId="mcp-servers-count">
          <Content component="p" data-testid="mcp-servers-count">
            {serverCount} servers
          </Content>
        </FormGroup>
      )}
    </FormSection>
  );
};

export default McpSourceDetailsSection;
