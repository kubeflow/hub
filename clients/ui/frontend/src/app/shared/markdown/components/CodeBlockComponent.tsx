import React from 'react';
import {
  CodeBlock,
  CodeBlockAction,
  CodeBlockCode,
  ClipboardCopyButton,
  Panel,
  PanelMain,
  PanelMainBody,
} from '@patternfly/react-core';

type CodeBlockComponentProps = {
  children: string;
  /** When set, wraps only the code in a scrollable panel; copy actions stay fixed above it. */
  maxHeight?: string;
  scrollTestId?: string;
  codeTestId?: string;
};

const CodeBlockComponent: React.FC<CodeBlockComponentProps> = ({
  children,
  maxHeight,
  scrollTestId,
  codeTestId,
}) => {
  const [copied, setCopied] = React.useState(false);
  const id = React.useId();

  const handleCopy = React.useCallback(async () => {
    try {
      await navigator.clipboard.writeText(children);
      setCopied(true);
    } catch {
      // clipboard write failed — don't show success
    }
  }, [children]);

  const actions = (
    <CodeBlockAction>
      <ClipboardCopyButton
        id={`copy-${id}`}
        aria-label="Copy to clipboard"
        onClick={handleCopy}
        onTooltipHidden={() => setCopied(false)}
        exitDelay={copied ? 1500 : 600}
        variant="plain"
      >
        {copied ? 'Successfully copied to clipboard!' : 'Copy to clipboard'}
      </ClipboardCopyButton>
    </CodeBlockAction>
  );

  const code = <CodeBlockCode>{children}</CodeBlockCode>;

  return (
    <CodeBlock actions={actions}>
      {maxHeight ? (
        <Panel isScrollable>
          <PanelMain maxHeight={maxHeight} tabIndex={0} data-testid={scrollTestId}>
            <PanelMainBody data-testid={codeTestId}>{code}</PanelMainBody>
          </PanelMain>
        </Panel>
      ) : (
        code
      )}
    </CodeBlock>
  );
};

export default CodeBlockComponent;
