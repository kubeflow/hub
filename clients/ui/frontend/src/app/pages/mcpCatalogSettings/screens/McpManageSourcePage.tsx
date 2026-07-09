import * as React from 'react';
import { useParams, Link } from 'react-router-dom';
import {
  Breadcrumb,
  BreadcrumbItem,
  Drawer,
  DrawerContent,
  DrawerContentBody,
  DrawerPanelContent,
} from '@patternfly/react-core';
import { ApplicationsPage } from 'mod-arch-shared';
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

  const panelContent = (
    <DrawerPanelContent isResizable defaultSize="50%">
      <McpExpectedYamlFormatDrawerPanel onClose={() => setIsExpectedFormatDrawerOpen(false)} />
    </DrawerPanelContent>
  );

  return (
    <Drawer isExpanded={isExpectedFormatDrawerOpen}>
      <DrawerContent panelContent={panelContent}>
        <DrawerContentBody>
          <ApplicationsPage
            breadcrumb={
              <Breadcrumb>
                <BreadcrumbItem>
                  <Link to={mcpCatalogSettingsUrl()}>{MCP_CATALOG_SOURCES_BREADCRUMB}</Link>
                </BreadcrumbItem>
                <BreadcrumbItem data-testid="mcp-breadcrumb-source-action" isActive>
                  {pageTitle}
                </BreadcrumbItem>
              </Breadcrumb>
            }
            title={pageTitle}
            description={description}
            errorMessage={catalogSourceId ? existingSourceConfigLoadError?.message : undefined}
            empty={catalogSourceId ? !existingSourceConfig : false}
            loaded={catalogSourceId ? existingSourceConfigLoaded : true}
            provideChildrenPadding
          >
            <McpManageSourceForm
              existingSourceConfig={existingSourceConfig || undefined}
              isEditMode={!isAddMode}
              onToggleExpectedFormatDrawer={() => setIsExpectedFormatDrawerOpen((prev) => !prev)}
            />
          </ApplicationsPage>
        </DrawerContentBody>
      </DrawerContent>
    </Drawer>
  );
};

export default McpManageSourcePage;
