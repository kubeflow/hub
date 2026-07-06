import React from 'react';
import { useParams } from 'react-router';
import { Link } from 'react-router-dom';
import {
  Breadcrumb,
  BreadcrumbItem,
  Button,
  Content,
  ContentVariants,
  EmptyState,
  EmptyStateBody,
  EmptyStateFooter,
  Flex,
  FlexItem,
  Label,
  Stack,
  StackItem,
} from '@patternfly/react-core';
import { ApplicationsIcon, SearchIcon } from '@patternfly/react-icons';
import { ApplicationsPage } from 'mod-arch-shared';
import { useMcpServerWithAPI } from '~/app/hooks/mcpServerCatalog/useMcpServer';
import { McpCatalogContext } from '~/app/context/mcpCatalog/McpCatalogContext';
import { mcpCatalogUrl } from '~/app/routes/mcpCatalog/mcpCatalog';
import { MCP_CATALOG_TITLE } from '~/app/pages/mcpCatalog/const';
import ScrollViewOnMount from '~/app/shared/components/ScrollViewOnMount';
import {
  McpCardIconType,
  getMcpCardIconConfig,
} from '~/app/pages/mcpCatalog/components/McpCatalogCardIcons';
import { isMcpRemoteDeploymentMode } from '~/app/pages/mcpCatalog/utils/mcpCatalogUtils';
import McpServerDetailsView from './McpServerDetailsView';

const McpServerDetailsPage: React.FC = () => {
  const { serverId = '' } = useParams<{ serverId: string }>();
  const { mcpApiState } = React.useContext(McpCatalogContext);
  const [server, serverLoaded, serverLoadError] = useMcpServerWithAPI(mcpApiState, serverId);

  const isNotFound = !server && (serverLoaded || !!serverLoadError);

  return (
    <>
      <ScrollViewOnMount shouldScroll scrollToTop />
      <ApplicationsPage
        breadcrumb={
          <Breadcrumb>
            <BreadcrumbItem>
              <Link to={mcpCatalogUrl()}>{MCP_CATALOG_TITLE}</Link>
            </BreadcrumbItem>
            <BreadcrumbItem isActive data-testid="breadcrumb-server-name">
              {server?.displayName || server?.name || 'Details'}
            </BreadcrumbItem>
          </Breadcrumb>
        }
        title={
          server ? (
            <Flex
              spaceItems={{ default: 'spaceItemsMd' }}
              alignItems={{ default: 'alignItemsCenter' }}
            >
              {server.logo ? (
                <img
                  src={server.logo}
                  alt="server logo"
                  style={{ height: '56px', width: '56px' }}
                />
              ) : (
                <ApplicationsIcon
                  style={{ fontSize: '56px' }}
                  data-testid="mcp-server-default-icon"
                />
              )}
              <Stack>
                <StackItem>
                  <Flex
                    gap={{ default: 'gapSm' }}
                    alignItems={{ default: 'alignItemsCenter' }}
                    flexWrap={{ default: 'wrap' }}
                  >
                    <FlexItem>{server.displayName || server.name}</FlexItem>
                    {isMcpRemoteDeploymentMode(server.deploymentMode) && (
                      <FlexItem>
                        <Label data-testid="mcp-server-details-remote-label">
                          {getMcpCardIconConfig(McpCardIconType.REMOTE).label}
                        </Label>
                      </FlexItem>
                    )}
                  </Flex>
                </StackItem>
                {server.provider && (
                  <StackItem>
                    <Content component={ContentVariants.small}>Provider: {server.provider}</Content>
                  </StackItem>
                )}
              </Stack>
            </Flex>
          ) : null
        }
        empty={isNotFound}
        emptyStatePage={
          isNotFound ? (
            <EmptyState
              icon={SearchIcon}
              titleText="MCP server not found"
              data-testid="mcp-server-not-found"
            >
              <EmptyStateBody>The requested MCP server could not be found.</EmptyStateBody>
              <EmptyStateFooter>
                <Button
                  variant="primary"
                  component={(props) => <Link {...props} to={mcpCatalogUrl()} />}
                >
                  Return to {MCP_CATALOG_TITLE}
                </Button>
              </EmptyStateFooter>
            </EmptyState>
          ) : undefined
        }
        loadError={isNotFound ? undefined : serverLoadError}
        loaded={isNotFound || serverLoaded}
        errorMessage="Unable to load MCP server details"
        provideChildrenPadding
      >
        {server && <McpServerDetailsView server={server} />}
      </ApplicationsPage>
    </>
  );
};

export default McpServerDetailsPage;
