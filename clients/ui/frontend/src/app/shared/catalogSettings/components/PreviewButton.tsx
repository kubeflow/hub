import * as React from 'react';
import { Button, ButtonProps } from '@patternfly/react-core';

type PreviewButtonProps = {
  label: string;
  onClick: () => void;
  isDisabled: boolean;
  isLoading?: boolean;
  variant?: ButtonProps['variant'];
  testId?: string;
};

const PreviewButton: React.FC<PreviewButtonProps> = ({
  label,
  onClick,
  isDisabled,
  isLoading = false,
  variant = 'primary',
  testId = 'preview-button',
}) => (
  <Button
    variant={variant}
    onClick={onClick}
    isDisabled={isDisabled}
    isLoading={isLoading}
    data-testid={testId}
  >
    {label}
  </Button>
);

export default PreviewButton;
