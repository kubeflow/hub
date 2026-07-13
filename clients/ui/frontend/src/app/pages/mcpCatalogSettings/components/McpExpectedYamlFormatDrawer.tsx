import * as React from 'react';
import {
  CodeBlock,
  CodeBlockAction,
  CodeBlockCode,
  ClipboardCopyButton,
  DrawerActions,
  DrawerCloseButton,
  DrawerHead,
  DrawerPanelBody,
} from '@patternfly/react-core';
import sampleMcpCatalogYamlContent from '~/app/pages/mcpCatalogSettings/sample-mcp-catalog.yaml';
import { MCP_EXPECTED_FORMAT_DRAWER_TITLE } from '~/app/pages/mcpCatalogSettings/constants';

type McpExpectedYamlFormatDrawerPanelProps = {
  onClose: () => void;
};

export const McpExpectedYamlFormatDrawerPanel: React.FC<McpExpectedYamlFormatDrawerPanelProps> = ({
  onClose,
}) => {
  const [copied, setCopied] = React.useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(sampleMcpCatalogYamlContent);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // clipboard API unavailable (insecure context or permission denied)
    }
  };

  return (
    <>
      <DrawerHead>
        <span data-testid="mcp-expected-format-drawer-title">
          {MCP_EXPECTED_FORMAT_DRAWER_TITLE}
        </span>
        <DrawerActions>
          <DrawerCloseButton
            onClose={onClose}
            aria-label="Close drawer"
            data-testid="mcp-expected-format-drawer-close"
          />
        </DrawerActions>
      </DrawerHead>
      <DrawerPanelBody>
        <p className="pf-v6-u-mb-md">
          MCP catalog sources use a YAML file with an optional <strong>source</strong> label and an{' '}
          <strong>mcp_servers</strong> list. Each server entry maps to fields shown in the MCP
          catalog preview. Comments in the example below describe required and optional fields.
        </p>
        <CodeBlock
          actions={
            <CodeBlockAction>
              <ClipboardCopyButton
                id="mcp-yaml-copy-button"
                textId="mcp-yaml-code"
                aria-label="Copy to clipboard"
                onClick={handleCopy}
                variant="plain"
                data-testid="mcp-yaml-copy-button"
              >
                {copied ? 'Copied' : 'Copy to clipboard'}
              </ClipboardCopyButton>
            </CodeBlockAction>
          }
        >
          <CodeBlockCode id="mcp-yaml-code">{sampleMcpCatalogYamlContent}</CodeBlockCode>
        </CodeBlock>
      </DrawerPanelBody>
    </>
  );
};
