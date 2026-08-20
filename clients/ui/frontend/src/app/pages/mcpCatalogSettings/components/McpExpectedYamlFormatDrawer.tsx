import * as React from 'react';
import { ExpectedYamlFormatDrawer } from '~/app/shared/catalogSettings';
import sampleMcpCatalogYamlContent from '~/app/pages/mcpCatalogSettings/sample-mcp-catalog.yaml';
import { MCP_EXPECTED_FORMAT_DRAWER_TITLE } from '~/app/pages/mcpCatalogSettings/constants';

type McpExpectedYamlFormatDrawerPanelProps = {
  onClose: () => void;
};

const introText = (
  <p className="pf-v6-u-mb-md">
    MCP catalog sources use a YAML file with an optional <strong>source</strong> label and an{' '}
    <strong>mcp_servers</strong> list. Each server entry maps to fields shown in the MCP catalog
    preview. Comments in the example below describe required and optional fields.
  </p>
);

export const McpExpectedYamlFormatDrawerPanel: React.FC<McpExpectedYamlFormatDrawerPanelProps> = ({
  onClose,
}) => (
  <ExpectedYamlFormatDrawer
    onClose={onClose}
    title={MCP_EXPECTED_FORMAT_DRAWER_TITLE}
    sampleYaml={sampleMcpCatalogYamlContent}
    intro={introText}
    showCopyButton
    testIds={{
      title: 'mcp-expected-format-drawer-title',
      close: 'mcp-expected-format-drawer-close',
      copyButton: 'mcp-yaml-copy-button',
    }}
  />
);
