import * as React from 'react';
import { useParams } from 'react-router-dom';
import { ManageSourcePageShell } from '~/app/shared/catalogSettings';
import {
  MCP_ADD_SOURCE_TITLE,
  MCP_ADD_SOURCE_DESCRIPTION,
  MCP_MANAGE_SOURCE_TITLE,
  MCP_MANAGE_SOURCE_DESCRIPTION,
  mcpCatalogSettingsUrl,
} from '~/app/routes/mcpCatalogSettings/mcpCatalogSettings';
import McpManageSourceForm from '~/app/pages/mcpCatalogSettings/components/McpManageSourceForm';
import { McpExpectedYamlFormatDrawerPanel } from '~/app/pages/mcpCatalogSettings/components/McpExpectedYamlFormatDrawer';
import { useMcpCatalogSourceConfigBySourceId } from '~/app/hooks/mcpCatalogSettings/useMcpCatalogSourceConfigBySourceId';

const MCP_CATALOG_SOURCES_BREADCRUMB = 'MCP catalog sources';

const McpManageSourcePage: React.FC = () => {
  const { catalogSourceId } = useParams<{ catalogSourceId?: string }>();
  const isAddMode = !catalogSourceId;
  const pageTitle = isAddMode ? MCP_ADD_SOURCE_TITLE : MCP_MANAGE_SOURCE_TITLE;
  const description = isAddMode ? MCP_ADD_SOURCE_DESCRIPTION : MCP_MANAGE_SOURCE_DESCRIPTION;

  const state = useMcpCatalogSourceConfigBySourceId(catalogSourceId || '');
  const [existingSourceConfig, existingSourceConfigLoaded, existingSourceConfigLoadError] = state;
  const [isExpectedFormatDrawerOpen, setIsExpectedFormatDrawerOpen] = React.useState(false);

  return (
    <ManageSourcePageShell
      listPageUrl={mcpCatalogSettingsUrl()}
      listPageLabel={MCP_CATALOG_SOURCES_BREADCRUMB}
      breadcrumbLabel={pageTitle}
      breadcrumbTestId="mcp-breadcrumb-source-action"
      title={pageTitle}
      description={description}
      errorMessage={catalogSourceId ? existingSourceConfigLoadError?.message : undefined}
      empty={catalogSourceId ? !existingSourceConfig : false}
      loaded={catalogSourceId ? existingSourceConfigLoaded : true}
      isExpectedFormatDrawerOpen={isExpectedFormatDrawerOpen}
      drawerPanelContent={
        <McpExpectedYamlFormatDrawerPanel onClose={() => setIsExpectedFormatDrawerOpen(false)} />
      }
    >
      <McpManageSourceForm
        existingSourceConfig={existingSourceConfig || undefined}
        isEditMode={!isAddMode}
        onToggleExpectedFormatDrawer={() => setIsExpectedFormatDrawerOpen((prev) => !prev)}
      />
    </ManageSourcePageShell>
  );
};

export default McpManageSourcePage;
