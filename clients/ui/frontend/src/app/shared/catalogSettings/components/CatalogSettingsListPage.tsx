import * as React from 'react';
import { Button, EmptyState, EmptyStateBody, EmptyStateVariant } from '@patternfly/react-core';
import { PlusCircleIcon } from '@patternfly/react-icons';
import { ApplicationsPage } from 'mod-arch-shared';

type CatalogSettingsListPageProps = {
  title: React.ReactNode;
  description: string;
  isEmpty: boolean;
  loaded: boolean;
  loadError?: Error;
  errorMessage: string;
  emptyStateTitle: string;
  emptyStateBody: string;
  emptyStateTestId: string;
  addSourceLabel: string;
  addSourceButtonTestId: string;
  onAddSource: () => void;
  children: React.ReactNode;
};

const CatalogSettingsListPage: React.FC<CatalogSettingsListPageProps> = ({
  title,
  description,
  isEmpty,
  loaded,
  loadError,
  errorMessage,
  emptyStateTitle,
  emptyStateBody,
  emptyStateTestId,
  addSourceLabel,
  addSourceButtonTestId,
  onAddSource,
  children,
}) => (
  <ApplicationsPage
    title={title}
    description={description}
    empty={isEmpty}
    emptyStatePage={
      <EmptyState
        headingLevel="h5"
        icon={PlusCircleIcon}
        titleText={emptyStateTitle}
        variant={EmptyStateVariant.lg}
        data-testid={emptyStateTestId}
      >
        <EmptyStateBody>{emptyStateBody}</EmptyStateBody>
        <Button variant="primary" onClick={onAddSource} data-testid={addSourceButtonTestId}>
          {addSourceLabel}
        </Button>
      </EmptyState>
    }
    loaded={loaded}
    loadError={loadError}
    errorMessage={errorMessage}
    provideChildrenPadding
  >
    {children}
  </ApplicationsPage>
);

export default CatalogSettingsListPage;
