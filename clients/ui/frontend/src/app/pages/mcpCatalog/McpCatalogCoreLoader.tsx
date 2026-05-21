import * as React from 'react';
import { Alert, Bullseye } from '@patternfly/react-core';
import {
  ApplicationsPage,
  KubeflowDocs,
  ProjectObjectType,
  TitleWithIcon,
  typedEmptyImage,
  WhosMyAdministrator,
} from 'mod-arch-shared';
import { useThemeContext } from 'mod-arch-kubeflow';
import { Outlet } from 'react-router-dom';
import { McpCatalogContext } from '~/app/context/mcpCatalog/McpCatalogContext';
import { EmptyCatalogState } from '~/app/shared/components/catalog';
import { hasSourcesWithModels } from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import { MCP_CATALOG_TITLE, MCP_CATALOG_DESCRIPTION } from '~/app/pages/mcpCatalog/const';

const McpCatalogCoreLoader: React.FC = () => {
  const { catalogSources, catalogSourcesLoaded, catalogSourcesLoadError } =
    React.useContext(McpCatalogContext);
  const { isMUITheme } = useThemeContext();

  if (catalogSourcesLoadError) {
    return (
      <ApplicationsPage
        title={
          <TitleWithIcon title={MCP_CATALOG_TITLE} objectType={ProjectObjectType.mcpCatalog} />
        }
        description={MCP_CATALOG_DESCRIPTION}
        headerContent={null}
        empty
        emptyStatePage={
          <Bullseye>
            <Alert title="MCP catalog source load error" variant="danger" isInline>
              {catalogSourcesLoadError.message}
            </Alert>
          </Bullseye>
        }
        loaded
      />
    );
  }

  if (!catalogSourcesLoaded) {
    return (
      <ApplicationsPage
        title={
          <TitleWithIcon title={MCP_CATALOG_TITLE} objectType={ProjectObjectType.mcpCatalog} />
        }
        description={MCP_CATALOG_DESCRIPTION}
        headerContent={null}
        empty
        emptyStatePage={<Bullseye>Loading catalog sources...</Bullseye>}
        loaded={false}
      />
    );
  }

  if (catalogSources?.items?.length === 0 || !hasSourcesWithModels(catalogSources)) {
    return (
      <ApplicationsPage
        title={
          <TitleWithIcon title={MCP_CATALOG_TITLE} objectType={ProjectObjectType.mcpCatalog} />
        }
        description={MCP_CATALOG_DESCRIPTION}
        empty
        emptyStatePage={
          <EmptyCatalogState
            testid="empty-mcp-catalog-state"
            title="MCP catalog configuration required"
            description={
              isMUITheme
                ? 'To discover MCP servers, follow the instructions in the docs below.'
                : 'There are no MCP sources to display. Request that your administrator configure MCP sources for the catalog.'
            }
            headerIcon={() => (
              <img src={typedEmptyImage(ProjectObjectType.modelRegistrySettings)} alt="" />
            )}
            primaryAction={isMUITheme ? <KubeflowDocs /> : <WhosMyAdministrator />}
          />
        }
        headerContent={null}
        loaded
        provideChildrenPadding
      />
    );
  }

  return <Outlet />;
};

export default McpCatalogCoreLoader;
