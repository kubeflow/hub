import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import CodeBlockComponent from '~/app/shared/markdown/components/CodeBlockComponent';

describe('CodeBlockComponent', () => {
  const originalClipboard = navigator.clipboard;

  beforeEach(() => {
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText: jest.fn().mockResolvedValue(undefined) },
      writable: true,
      configurable: true,
    });
  });

  afterEach(() => {
    Object.defineProperty(navigator, 'clipboard', {
      value: originalClipboard,
      writable: true,
      configurable: true,
    });
  });

  it('renders code content correctly', () => {
    const codeContent = 'const example = "test";';
    render(<CodeBlockComponent>{codeContent}</CodeBlockComponent>);

    expect(screen.getByText(codeContent)).toBeInTheDocument();
    expect(screen.getByText(codeContent).closest('.pf-v6-c-code-block')).toBeInTheDocument();
    expect(screen.getByText(codeContent).closest('.pf-v6-c-code-block__code')).toBeInTheDocument();
  });

  it('renders copy to clipboard button', () => {
    render(<CodeBlockComponent>some code</CodeBlockComponent>);

    expect(screen.getByRole('button', { name: 'Copy to clipboard' })).toBeInTheDocument();
  });

  it('copies content to clipboard on click', async () => {
    const codeContent = 'pip install model-registry';
    render(<CodeBlockComponent>{codeContent}</CodeBlockComponent>);

    await userEvent.click(screen.getByRole('button', { name: 'Copy to clipboard' }));

    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(codeContent);
  });

  it('button remains accessible after successful copy', async () => {
    render(<CodeBlockComponent>some code</CodeBlockComponent>);

    await userEvent.click(screen.getByRole('button', { name: 'Copy to clipboard' }));

    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('some code');
    expect(screen.getByRole('button', { name: 'Copy to clipboard' })).toBeInTheDocument();
  });

  it('keeps copy actions outside the scroll region when maxHeight is set', () => {
    render(
      <CodeBlockComponent maxHeight="400px" scrollTestId="scroll-region" codeTestId="code-region">
        {'{ "name": "example" }'}
      </CodeBlockComponent>,
    );

    const scrollRegion = screen.getByTestId('scroll-region');
    expect(scrollRegion).toHaveAttribute('tabIndex', '0');
    expect(screen.getByTestId('code-region')).toHaveTextContent('{ "name": "example" }');
    expect(scrollRegion.querySelector('.pf-v6-c-code-block__header')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Copy to clipboard' })).toBeInTheDocument();
  });

  it('does not show copied state when clipboard write fails', async () => {
    (navigator.clipboard.writeText as jest.Mock).mockRejectedValue(new Error('denied'));

    render(<CodeBlockComponent>some code</CodeBlockComponent>);

    await userEvent.click(screen.getByRole('button', { name: 'Copy to clipboard' }));

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Copy to clipboard' })).toBeInTheDocument();
    });
  });
});
