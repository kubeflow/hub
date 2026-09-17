import * as React from 'react';
import { UpdateObjectAtPropAndValue } from 'mod-arch-shared';
import { IncludeExcludeFiltersSection } from '~/app/shared/catalogSettings';
import { ManageMcpSourceFormData } from '~/app/pages/mcpCatalogSettings/useManageMcpSourceData';
import { MCP_FORM_LABELS, MCP_DESCRIPTION_TEXT } from '~/app/pages/mcpCatalogSettings/constants';

type McpServerFiltersSectionProps = {
  formData: ManageMcpSourceFormData;
  setData: UpdateObjectAtPropAndValue<ManageMcpSourceFormData>;
  isDefaultExpanded?: boolean;
};

const McpServerFiltersSection: React.FC<McpServerFiltersSectionProps> = ({
  formData,
  setData,
  isDefaultExpanded = false,
}) => (
  <IncludeExcludeFiltersSection
    includedValue={formData.includedServers}
    onIncludedChange={(value) => setData('includedServers', value)}
    excludedValue={formData.excludedServers}
    onExcludedChange={(value) => setData('excludedServers', value)}
    isDefaultExpanded={isDefaultExpanded}
    sectionTitle={MCP_FORM_LABELS.SERVER_FILTERS}
    sectionTitleId="mcp-server-filters-title"
    sectionDescription={MCP_DESCRIPTION_TEXT.FILTER_INFO}
    includedFieldId="mcp-included-servers"
    excludedFieldId="mcp-excluded-servers"
    includedLabel={MCP_FORM_LABELS.INCLUDED_SERVERS}
    excludedLabel={MCP_FORM_LABELS.EXCLUDED_SERVERS}
    includedDescription={MCP_DESCRIPTION_TEXT.INCLUDED_SERVERS}
    excludedDescription={MCP_DESCRIPTION_TEXT.EXCLUDED_SERVERS}
    testIds={{
      section: 'mcp-server-filters-section',
      includedInput: 'mcp-included-servers-input',
      excludedInput: 'mcp-excluded-servers-input',
    }}
  />
);

export default McpServerFiltersSection;
