import React from 'react';
import {
  EmptyState,
  EmptyStateActions,
  EmptyStateBody,
  EmptyStateFooter,
  EmptyStateVariant,
} from '@patternfly/react-core';
import { PlusCircleIcon } from '@patternfly/react-icons';

type EmptyCatalogStateProps = {
  testid?: string;
  className?: string;
  title: string;
  description: React.ReactNode;
  headerIcon?: React.ComponentType;
  children?: React.ReactNode;
  primaryAction?: React.ReactNode;
  secondaryAction?: React.ReactNode;
  variant?: EmptyStateVariant;
};

const EmptyCatalogState: React.FC<EmptyCatalogStateProps> = ({
  testid,
  className,
  title,
  description,
  headerIcon,
  children,
  primaryAction,
  secondaryAction,
  variant = EmptyStateVariant.sm,
}) => (
  <EmptyState
    className={className}
    icon={headerIcon ?? PlusCircleIcon}
    titleText={title}
    variant={variant}
    data-testid={testid}
  >
    <EmptyStateBody>{description}</EmptyStateBody>
    {children}
    <EmptyStateFooter>
      {primaryAction && <EmptyStateActions>{primaryAction}</EmptyStateActions>}
      {secondaryAction && <EmptyStateActions>{secondaryAction}</EmptyStateActions>}
    </EmptyStateFooter>
  </EmptyState>
);

export default EmptyCatalogState;
