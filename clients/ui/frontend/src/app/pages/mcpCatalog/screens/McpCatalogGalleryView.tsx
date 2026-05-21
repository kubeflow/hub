import * as React from 'react';
import { Button } from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';
import { McpCatalogContext } from '~/app/context/mcpCatalog/McpCatalogContext';
import { useMcpServersBySourceLabelWithAPI } from '~/app/hooks/mcpServerCatalog/useMcpServersBySourceLabel';
import {
  MCP_CATALOG_GRID_SPAN,
  OTHER_MCP_SERVERS_DISPLAY_NAME,
} from '~/app/pages/mcpCatalog/const';
import { mcpFiltersToFilterQuery } from '~/app/pages/mcpCatalog/utils/mcpCatalogUtils';
import {
  getLabelDisplayName,
  getLabelDescription,
} from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import { CatalogGalleryLayout, EmptyCatalogState } from '~/app/shared/components/catalog';
import McpCatalogCard from '~/app/pages/mcpCatalog/components/McpCatalogCard';

const PAGE_SIZE = 10;

type McpCatalogGalleryViewProps = {
  handleFilterReset: () => void;
  isSingleCategory?: boolean;
  singleCategoryLabel?: string;
};

const McpCatalogGalleryView: React.FC<McpCatalogGalleryViewProps> = ({
  handleFilterReset,
  isSingleCategory = false,
  singleCategoryLabel,
}) => {
  const {
    mcpApiState,
    selectedSourceLabel,
    searchQuery,
    filters,
    catalogLabels,
    catalogLabelsLoaded,
  } = React.useContext(McpCatalogContext);

  const filterQuery = React.useMemo(() => mcpFiltersToFilterQuery(filters), [filters]);

  const { mcpServers, mcpServersLoaded, mcpServersLoadError } = useMcpServersBySourceLabelWithAPI(
    mcpApiState,
    {
      sourceLabel: selectedSourceLabel,
      pageSize: PAGE_SIZE,
      searchQuery,
      filterQuery: filterQuery || undefined,
    },
  );

  const loaded = mcpServersLoaded && catalogLabelsLoaded;

  const effectiveCategoryLabel = singleCategoryLabel || selectedSourceLabel || '';
  const categoryTitle = isSingleCategory
    ? getLabelDisplayName(
        effectiveCategoryLabel,
        catalogLabels,
        OTHER_MCP_SERVERS_DISPLAY_NAME,
        'servers',
      )
    : undefined;
  const categoryDescription = isSingleCategory
    ? getLabelDescription(effectiveCategoryLabel, catalogLabels)
    : undefined;

  return (
    <CatalogGalleryLayout
      items={mcpServers.items}
      loaded={loaded}
      loadError={mcpServersLoadError}
      renderCard={(server) => <McpCatalogCard server={server} />}
      getItemKey={(server) => server.id}
      gridSpans={MCP_CATALOG_GRID_SPAN}
      hasMore={mcpServers.hasMore && mcpServers.items.length >= PAGE_SIZE}
      isLoadingMore={mcpServers.isLoadingMore}
      onLoadMore={mcpServers.loadMore}
      loadMoreLabel="Load more servers"
      loadingMoreLabel="Loading more MCP servers..."
      loadingLabel="Loading MCP servers..."
      errorTitle="Failed to load MCP servers"
      categoryTitle={categoryTitle}
      categoryDescription={categoryDescription}
      renderEmptyState={() => (
        <EmptyCatalogState
          testid="empty-mcp-catalog-state"
          title="No results found"
          headerIcon={SearchIcon}
          description="Adjust your filters and try again."
          primaryAction={
            <Button variant="link" onClick={handleFilterReset}>
              Reset filters
            </Button>
          }
        />
      )}
    />
  );
};

export default McpCatalogGalleryView;
