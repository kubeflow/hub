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

export type ExpectedYamlFormatDrawerTestIds = {
  title?: string;
  close?: string;
  copyButton?: string;
};

type ExpectedYamlFormatDrawerProps = {
  onClose: () => void;
  title: string;
  sampleYaml: string;
  /** Optional intro copy shown above the code block (MCP includes usage guidance). */
  intro?: React.ReactNode;
  /** Whether to show a "Copy to clipboard" action above the code block. */
  showCopyButton?: boolean;
  testIds?: ExpectedYamlFormatDrawerTestIds;
};

export const ExpectedYamlFormatDrawer: React.FC<ExpectedYamlFormatDrawerProps> = ({
  onClose,
  title,
  sampleYaml,
  intro,
  showCopyButton = false,
  testIds,
}) => {
  const [copied, setCopied] = React.useState(false);
  const {
    title: titleTestId = 'expected-format-drawer-title',
    close: closeTestId = 'expected-format-drawer-close',
    copyButton: copyButtonTestId = 'yaml-copy-button',
  } = testIds ?? {};
  const codeId = `${copyButtonTestId}-code`;

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(sampleYaml);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // clipboard API unavailable (insecure context or permission denied)
    }
  };

  return (
    <>
      <DrawerHead>
        <span data-testid={titleTestId}>{title}</span>
        <DrawerActions>
          <DrawerCloseButton
            onClose={onClose}
            aria-label="Close drawer"
            data-testid={closeTestId}
          />
        </DrawerActions>
      </DrawerHead>
      <DrawerPanelBody hasNoPadding={!intro}>
        {intro}
        <CodeBlock
          actions={
            showCopyButton ? (
              <CodeBlockAction>
                <ClipboardCopyButton
                  id={copyButtonTestId}
                  textId={codeId}
                  aria-label="Copy to clipboard"
                  onClick={handleCopy}
                  variant="plain"
                  data-testid={copyButtonTestId}
                >
                  {copied ? 'Copied' : 'Copy to clipboard'}
                </ClipboardCopyButton>
              </CodeBlockAction>
            ) : undefined
          }
        >
          <CodeBlockCode id={codeId}>{sampleYaml}</CodeBlockCode>
        </CodeBlock>
      </DrawerPanelBody>
    </>
  );
};

export default ExpectedYamlFormatDrawer;
