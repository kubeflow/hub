import * as React from 'react';
import { CatalogSourceStatus } from '~/app/shared/catalogSettings';
import { McpCatalogSourceConfig } from '~/app/mcpServerCatalogTypes';
import { McpCatalogSettingsContext } from '~/app/context/mcpCatalogSettings/McpCatalogSettingsContext';

type McpCatalogSourceStatusProps = {
  mcpCatalogSourceConfig: McpCatalogSourceConfig;
};

const McpCatalogSourceStatus: React.FC<McpCatalogSourceStatusProps> = ({
  mcpCatalogSourceConfig,
}) => {
  const { mcpCatalogSources, mcpCatalogSourcesLoaded, mcpCatalogSourcesLoadError } =
    React.useContext(McpCatalogSettingsContext);

  return (
    <CatalogSourceStatus
      catalogSourceConfig={mcpCatalogSourceConfig}
      catalogSources={mcpCatalogSources}
      catalogSourcesLoaded={mcpCatalogSourcesLoaded}
      catalogSourcesLoadError={mcpCatalogSourcesLoadError}
      testIdPrefix="mcp-"
    />
  );
};

export default McpCatalogSourceStatus;
