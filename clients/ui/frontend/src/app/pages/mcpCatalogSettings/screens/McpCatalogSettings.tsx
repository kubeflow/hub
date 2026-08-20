import * as React from 'react';
import { Button, EmptyState, EmptyStateBody, EmptyStateVariant } from '@patternfly/react-core';
import { PlusCircleIcon } from '@patternfly/react-icons';
import { useNavigate } from 'react-router-dom';
import { ApplicationsPage, ProjectObjectType, TitleWithIcon } from 'mod-arch-shared';
import {
  MCP_CATALOG_SETTINGS_PAGE_TITLE,
  MCP_CATALOG_SETTINGS_DESCRIPTION,
  mcpAddSourceUrl,
  MCP_ADD_SOURCE_TITLE,
} from '~/app/routes/mcpCatalogSettings/mcpCatalogSettings';
import { McpCatalogSettingsContext } from '~/app/context/mcpCatalogSettings/McpCatalogSettingsContext';
import McpCatalogSourceConfigsTable from './McpCatalogSourceConfigsTable';

const McpCatalogSettings: React.FC = () => {
  const navigate = useNavigate();
  const {
    mcpCatalogSourceConfigs,
    mcpCatalogSourceConfigsLoaded,
    mcpCatalogSourceConfigsLoadError,
    apiState,
    refreshMcpCatalogSourceConfigs,
    refreshMcpCatalogSources,
  } = React.useContext(McpCatalogSettingsContext);

  const configs = mcpCatalogSourceConfigs?.catalogs || [];
  const isEmpty = mcpCatalogSourceConfigsLoaded && configs.length === 0;

  const handleDeleteSource = React.useCallback(
    async (sourceId: string): Promise<void> => {
      if (!apiState.apiAvailable) {
        throw new Error('API not available');
      }
      await apiState.api.deleteMcpCatalogSourceConfig({}, sourceId);
      refreshMcpCatalogSourceConfigs();
      refreshMcpCatalogSources();
    },
    [apiState.api, apiState.apiAvailable, refreshMcpCatalogSourceConfigs, refreshMcpCatalogSources],
  );

  return (
    <ApplicationsPage
      title={
        <TitleWithIcon
          title={MCP_CATALOG_SETTINGS_PAGE_TITLE}
          objectType={ProjectObjectType.mcpCatalog}
        />
      }
      description={MCP_CATALOG_SETTINGS_DESCRIPTION}
      empty={isEmpty}
      emptyStatePage={
        <EmptyState
          headingLevel="h5"
          icon={PlusCircleIcon}
          titleText="No MCP sources"
          variant={EmptyStateVariant.lg}
          data-testid="mcp-catalog-settings-empty-state"
        >
          <EmptyStateBody>
            No MCP sources have been configured. Add a source to get started.
          </EmptyStateBody>
          <Button
            variant="primary"
            onClick={() => navigate(mcpAddSourceUrl())}
            data-testid="mcp-add-source-button-empty"
          >
            {MCP_ADD_SOURCE_TITLE}
          </Button>
        </EmptyState>
      }
      loaded={mcpCatalogSourceConfigsLoaded}
      loadError={mcpCatalogSourceConfigsLoadError}
      errorMessage="Unable to load MCP catalog source configurations."
      provideChildrenPadding
    >
      <McpCatalogSourceConfigsTable
        mcpCatalogSourceConfigs={configs}
        onAddSource={() => navigate(mcpAddSourceUrl())}
        onDeleteSource={handleDeleteSource}
      />
    </ApplicationsPage>
  );
};

export default McpCatalogSettings;
