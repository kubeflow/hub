import * as React from 'react';
import { useNavigate } from 'react-router-dom';
import { ProjectObjectType, TitleWithIcon } from 'mod-arch-shared';
import { CatalogSettingsListPage } from '~/app/shared/catalogSettings';
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
  } = React.useContext(McpCatalogSettingsContext);

  const configs = mcpCatalogSourceConfigs?.catalogs || [];
  const isEmpty = mcpCatalogSourceConfigsLoaded && configs.length === 0;

  const handleAddSource = React.useCallback(() => {
    navigate(mcpAddSourceUrl());
  }, [navigate]);

  const handleDeleteSource = React.useCallback(
    async (sourceId: string): Promise<void> => {
      if (!apiState.apiAvailable) {
        throw new Error('API not available');
      }
      await apiState.api.deleteMcpCatalogSourceConfig({}, sourceId);
      refreshMcpCatalogSourceConfigs();
    },
    [apiState.api, apiState.apiAvailable, refreshMcpCatalogSourceConfigs],
  );

  return (
    <CatalogSettingsListPage
      title={
        <TitleWithIcon
          title={MCP_CATALOG_SETTINGS_PAGE_TITLE}
          objectType={ProjectObjectType.mcpCatalog}
        />
      }
      description={MCP_CATALOG_SETTINGS_DESCRIPTION}
      isEmpty={isEmpty}
      loaded={mcpCatalogSourceConfigsLoaded}
      loadError={mcpCatalogSourceConfigsLoadError}
      errorMessage="Unable to load MCP catalog source configurations."
      emptyStateTitle="No MCP sources"
      emptyStateBody="No MCP sources have been configured. Add a source to get started."
      emptyStateTestId="mcp-catalog-settings-empty-state"
      addSourceLabel={MCP_ADD_SOURCE_TITLE}
      addSourceButtonTestId="mcp-add-source-button-empty"
      onAddSource={handleAddSource}
    >
      <McpCatalogSourceConfigsTable
        mcpCatalogSourceConfigs={configs}
        onAddSource={handleAddSource}
        onDeleteSource={handleDeleteSource}
      />
    </CatalogSettingsListPage>
  );
};

export default McpCatalogSettings;
