import * as React from 'react';
import { Button, Label, Spinner, Stack, StackItem, Truncate } from '@patternfly/react-core';
import { InProgressIcon } from '@patternfly/react-icons';
import { McpCatalogSourceConfig } from '~/app/mcpServerCatalogTypes';
import { McpCatalogSettingsContext } from '~/app/context/mcpCatalogSettings/McpCatalogSettingsContext';
import { CatalogSourceStatus } from '~/app/shared/types/catalogTypes';
import CatalogSourceStatusErrorModal from '~/app/pages/modelCatalogSettings/components/CatalogSourceStatusErrorModal';

type McpCatalogSourceStatusProps = {
  mcpCatalogSourceConfig: McpCatalogSourceConfig;
};

const McpCatalogSourceStatus: React.FC<McpCatalogSourceStatusProps> = ({
  mcpCatalogSourceConfig,
}) => {
  const {
    mcpCatalogSources,
    mcpCatalogSourcesLoaded,
    mcpCatalogSourcesLoadError,
    pendingSourceIds,
  } = React.useContext(McpCatalogSettingsContext);
  const [isErrorModalOpen, setIsErrorModalOpen] = React.useState(false);

  if (!mcpCatalogSourceConfig.enabled || mcpCatalogSourceConfig.isDefault) {
    return <>-</>;
  }

  if (!mcpCatalogSourcesLoaded) {
    return (
      <Spinner size="md" data-testid={`mcp-source-status-loading-${mcpCatalogSourceConfig.id}`} />
    );
  }

  const startingOrUnknownLabel = (
    <Label
      color="grey"
      variant="outline"
      icon={<InProgressIcon />}
      data-testid={`mcp-source-status-${mcpCatalogSourcesLoadError ? 'unknown' : 'starting'}-${mcpCatalogSourceConfig.id}`}
    >
      {mcpCatalogSourcesLoadError ? 'Unknown' : 'Starting'}
    </Label>
  );

  if (pendingSourceIds.has(mcpCatalogSourceConfig.id)) {
    return startingOrUnknownLabel;
  }

  const matchingSource = mcpCatalogSources?.items?.find(
    (source) => source.id === mcpCatalogSourceConfig.id,
  );

  if (!matchingSource || !matchingSource.status) {
    return startingOrUnknownLabel;
  }

  switch (matchingSource.status) {
    case CatalogSourceStatus.AVAILABLE:
      return (
        <Label
          status="success"
          variant="outline"
          data-testid={`mcp-source-status-connected-${mcpCatalogSourceConfig.id}`}
        >
          Ready
        </Label>
      );

    case CatalogSourceStatus.ERROR: {
      const errorMessage = matchingSource.error || 'Unknown error occurred';

      return (
        <>
          <Stack hasGutter>
            <StackItem>
              <Label
                status="danger"
                variant="outline"
                data-testid={`mcp-source-status-failed-${mcpCatalogSourceConfig.id}`}
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
                data-testid={`mcp-source-status-error-link-${mcpCatalogSourceConfig.id}`}
              >
                <Truncate content={errorMessage} tooltipProps={{ hidden: true }} />
              </Button>
            </StackItem>
          </Stack>
          <CatalogSourceStatusErrorModal
            isOpen={isErrorModalOpen}
            onClose={() => setIsErrorModalOpen(false)}
            errorMessage={errorMessage}
          />
        </>
      );
    }

    case CatalogSourceStatus.DISABLED:
      return startingOrUnknownLabel;
    default:
      return startingOrUnknownLabel;
  }
};

export default McpCatalogSourceStatus;
