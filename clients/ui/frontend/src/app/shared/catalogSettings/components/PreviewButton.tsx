import * as React from 'react';
import { Button, ButtonProps, Tooltip } from '@patternfly/react-core';

type PreviewButtonProps = {
  label: string;
  onClick: () => void;
  isDisabled: boolean;
  isLoading?: boolean;
  variant?: ButtonProps['variant'];
  testId?: string;
  disabledTooltip?: string;
};

const PreviewButton: React.FC<PreviewButtonProps> = ({
  label,
  onClick,
  isDisabled,
  isLoading = false,
  variant = 'primary',
  testId = 'preview-button',
  disabledTooltip,
}) => {
  const button = (
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

  if (disabledTooltip && isDisabled) {
    return (
      <Tooltip content={disabledTooltip}>
        <span className="pf-v6-u-display-inline-block">{button}</span>
      </Tooltip>
    );
  }

  return button;
};

export default PreviewButton;
