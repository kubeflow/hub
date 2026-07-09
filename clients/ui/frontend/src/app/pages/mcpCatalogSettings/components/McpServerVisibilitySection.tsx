import * as React from 'react';
import {
  FormFieldGroupExpandable,
  FormFieldGroupHeader,
  TextArea,
  FormHelperText,
  HelperText,
  HelperTextItem,
} from '@patternfly/react-core';
import { UpdateObjectAtPropAndValue, ThemeAwareFormGroupWrapper } from 'mod-arch-shared';
import FormSection from '~/app/pages/modelRegistry/components/pf-overrides/FormSection';
import { ManageMcpSourceFormData } from '~/app/pages/mcpCatalogSettings/useManageMcpSourceData';
import {
  MCP_FORM_LABELS,
  MCP_PLACEHOLDERS,
  MCP_DESCRIPTION_TEXT,
} from '~/app/pages/mcpCatalogSettings/constants';

type McpServerVisibilitySectionProps = {
  formData: ManageMcpSourceFormData;
  setData: UpdateObjectAtPropAndValue<ManageMcpSourceFormData>;
  isDefaultExpanded?: boolean;
};

const McpServerVisibilitySection: React.FC<McpServerVisibilitySectionProps> = ({
  formData,
  setData,
  isDefaultExpanded = false,
}) => {
  const includedServersInput = (
    <TextArea
      id="mcp-included-servers"
      name="mcp-included-servers"
      data-testid="mcp-included-servers-input"
      value={formData.includedServers}
      onChange={(_event, value) => setData('includedServers', value)}
      rows={3}
      resizeOrientation="vertical"
      placeholder={MCP_PLACEHOLDERS.INCLUDED_SERVERS}
    />
  );

  const includedServersDescriptionTxtNode = (
    <FormHelperText>
      <HelperText>
        <HelperTextItem>{MCP_DESCRIPTION_TEXT.INCLUDED_SERVERS}</HelperTextItem>
      </HelperText>
    </FormHelperText>
  );

  const excludedServersInput = (
    <TextArea
      id="mcp-excluded-servers"
      name="mcp-excluded-servers"
      data-testid="mcp-excluded-servers-input"
      value={formData.excludedServers}
      onChange={(_event, value) => setData('excludedServers', value)}
      rows={3}
      resizeOrientation="vertical"
      placeholder={MCP_PLACEHOLDERS.EXCLUDED_SERVERS}
    />
  );

  const excludedServersDescriptionTxtNode = (
    <FormHelperText>
      <HelperText>
        <HelperTextItem>{MCP_DESCRIPTION_TEXT.EXCLUDED_SERVERS}</HelperTextItem>
      </HelperText>
    </FormHelperText>
  );

  return (
    <FormSection>
      <FormFieldGroupExpandable
        toggleAriaLabel="Server visibility"
        header={
          <FormFieldGroupHeader
            titleText={{
              text: MCP_FORM_LABELS.SERVER_VISIBILITY,
              id: 'mcp-server-visibility-title',
            }}
            titleDescription={MCP_DESCRIPTION_TEXT.FILTER_INFO}
          />
        }
        isExpanded={isDefaultExpanded}
        data-testid="mcp-server-visibility-section"
      >
        <ThemeAwareFormGroupWrapper
          label={MCP_FORM_LABELS.INCLUDED_SERVERS}
          fieldId="mcp-included-servers"
          descriptionTextNode={includedServersDescriptionTxtNode}
        >
          {includedServersInput}
        </ThemeAwareFormGroupWrapper>

        <ThemeAwareFormGroupWrapper
          label={MCP_FORM_LABELS.EXCLUDED_SERVERS}
          fieldId="mcp-excluded-servers"
          descriptionTextNode={excludedServersDescriptionTxtNode}
        >
          {excludedServersInput}
        </ThemeAwareFormGroupWrapper>
      </FormFieldGroupExpandable>
    </FormSection>
  );
};

export default McpServerVisibilitySection;
