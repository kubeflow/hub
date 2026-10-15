import * as React from 'react';
import { ClipboardCopyButton, CodeBlock, CodeBlockCode } from '@patternfly/react-core';

type SkillInstallCodeBlockProps = {
  id: string;
  content: string;
};

const SkillInstallCodeBlock: React.FC<SkillInstallCodeBlockProps> = ({ id, content }) => {
  const [copied, setCopied] = React.useState(false);

  return (
    <div style={{ position: 'relative' }} data-testid={`${id}-code-block`}>
      <CodeBlock>
        <CodeBlockCode id={id} style={{ paddingRight: '2.5rem' }}>
          {content}
        </CodeBlockCode>
      </CodeBlock>
      <ClipboardCopyButton
        id={`${id}-copy-button`}
        textId={id}
        aria-label="Copy to clipboard"
        onClick={() => {
          navigator.clipboard.writeText(content).catch(() => undefined);
          setCopied(true);
          window.setTimeout(() => setCopied(false), 1200);
        }}
        exitDelay={600}
        variant="plain"
        style={{ position: 'absolute', top: '0.25rem', right: '0.25rem' }}
        data-testid={`${id}-copy-button`}
      >
        {copied ? 'Copied' : 'Copy to clipboard'}
      </ClipboardCopyButton>
    </div>
  );
};

export default SkillInstallCodeBlock;
