import * as React from 'react';
import { Button, Label, Spinner, Stack, StackItem, Truncate } from '@patternfly/react-core';
import { InProgressIcon } from '@patternfly/react-icons';
import {
  CatalogSourceStatus as CatalogSourceStatusEnum,
  CatalogSourceList,
} from '~/app/shared/types/catalogTypes';
import CatalogSourceStatusErrorModal from './CatalogSourceStatusErrorModal';

export type CatalogSourceStatusConfig = {
  id: string;
  enabled?: boolean;
  isDefault?: boolean;
};

type CatalogSourceStatusProps<TConfig extends CatalogSourceStatusConfig> = {
  catalogSourceConfig: TConfig;
  catalogSources: CatalogSourceList | null;
  catalogSourcesLoaded: boolean;
  catalogSourcesLoadError?: Error;
  /** Map of sourceId -> previous status, for sources with an in-flight optimistic update. */
  pendingSourceIds?: Map<string, string>;
  /** Prefix applied to every testid rendered by this component (e.g. `'mcp-'`). */
  testIdPrefix?: string;
  validationFailedTitle?: string;
  validationFailedBody?: string;
};

const CatalogSourceStatus = <TConfig extends CatalogSourceStatusConfig>({
  catalogSourceConfig,
  catalogSources,
  catalogSourcesLoaded,
  catalogSourcesLoadError,
  pendingSourceIds,
  testIdPrefix = '',
  validationFailedTitle,
  validationFailedBody,
}: CatalogSourceStatusProps<TConfig>): React.ReactElement => {
  const [isErrorModalOpen, setIsErrorModalOpen] = React.useState(false);

  // Don't render status for default sources
  if (catalogSourceConfig.isDefault) {
    return <>-</>;
  }
  // If source is disabled, render "-"
  if (!catalogSourceConfig.enabled) {
    return <>-</>;
  }

  // Show loading spinner while fetching sources
  if (!catalogSourcesLoaded) {
    return (
      <Spinner
        size="md"
        data-testid={`${testIdPrefix}source-status-loading-${catalogSourceConfig.id}`}
      />
    );
  }

  const startingOrUnknownLabel = (
    <Label
      color="grey"
      variant="outline"
      icon={<InProgressIcon />}
      data-testid={`${testIdPrefix}source-status-${catalogSourcesLoadError ? 'unknown' : 'starting'}-${catalogSourceConfig.id}`}
    >
      {catalogSourcesLoadError ? 'Unknown' : 'Starting'}
    </Label>
  );

  // Show "Starting" for sources with a pending mutation (optimistic update)
  if (pendingSourceIds?.has(catalogSourceConfig.id)) {
    return startingOrUnknownLabel;
  }

  // Find the matching source from the catalog sources list
  const matchingSource = catalogSources?.items?.find(
    (source) => source.id === catalogSourceConfig.id,
  );

  if (!matchingSource || !matchingSource.status) {
    return startingOrUnknownLabel;
  }

  // Render based on status
  switch (matchingSource.status) {
    case CatalogSourceStatusEnum.AVAILABLE:
      return (
        <Label
          status="success"
          variant="outline"
          data-testid={`${testIdPrefix}source-status-connected-${catalogSourceConfig.id}`}
        >
          Ready
        </Label>
      );

    case CatalogSourceStatusEnum.ERROR: {
      const errorMessage = matchingSource.error || 'Unknown error occurred';

      return (
        <>
          <Stack hasGutter>
            <StackItem>
              <Label
                status="danger"
                variant="outline"
                data-testid={`${testIdPrefix}source-status-failed-${catalogSourceConfig.id}`}
              >
                Failed
              </Label>
            </StackItem>
            <StackItem>
              <Button
                variant="link"
                isInline
                isDanger
                onClick={() => setIsErrorModalOpen(true)}
                data-testid={`${testIdPrefix}source-status-error-link-${catalogSourceConfig.id}`}
              >
                <Truncate content={errorMessage} tooltipProps={{ hidden: true }} />
              </Button>
            </StackItem>
          </Stack>
          <CatalogSourceStatusErrorModal
            isOpen={isErrorModalOpen}
            onClose={() => setIsErrorModalOpen(false)}
            errorMessage={errorMessage}
            validationFailedTitle={validationFailedTitle}
            validationFailedBody={validationFailedBody}
          />
        </>
      );
    }

    case CatalogSourceStatusEnum.DISABLED:
      // If we reach here, config.enabled is true - line above
      // But status is still DISABLED, so show "Starting" (re-enable case)
      return startingOrUnknownLabel;
    default:
      return startingOrUnknownLabel;
  }
};

export default CatalogSourceStatus;
