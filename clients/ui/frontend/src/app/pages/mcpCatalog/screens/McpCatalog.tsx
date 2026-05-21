import * as React from 'react';
import { ApplicationsPage, ProjectObjectType, TitleWithIcon } from 'mod-arch-shared';
import { SearchIcon } from '@patternfly/react-icons';
import { McpCatalogContext } from '~/app/context/mcpCatalog/McpCatalogContext';
import { hasMcpFiltersApplied } from '~/app/pages/mcpCatalog/utils/mcpCatalogUtils';
import McpCatalogFilters from '~/app/pages/mcpCatalog/components/McpCatalogFilters';
import { MCP_CATALOG_TITLE, MCP_CATALOG_DESCRIPTION } from '~/app/pages/mcpCatalog/const';
import { CatalogPageLayout, EmptyCatalogState } from '~/app/shared/components/catalog';
import McpCatalogSourceLabelSelector from './McpCatalogSourceLabelSelector';
import McpCatalogAllServersView from './McpCatalogAllServersView';
import McpCatalogGalleryView from './McpCatalogGalleryView';

const McpCatalog: React.FC = () => {
  const {
    searchQuery,
    setSearchQuery,
    clearAllFilters,
    selectedSourceLabel,
    setSelectedSourceLabel,
    filters,
    catalogSources,
    catalogLabels,
    catalogSourcesLoaded,
  } = React.useContext(McpCatalogContext);

  const filtersApplied = hasMcpFiltersApplied(filters, searchQuery);
  const isAllServersView = selectedSourceLabel === undefined && !filtersApplied;

  const handleSearch = React.useCallback(
    (term: string) => {
      setSearchQuery(term);
    },
    [setSearchQuery],
  );

  const handleClearSearch = React.useCallback(() => {
    setSearchQuery('');
  }, [setSearchQuery]);

  const handleResetAllFilters = React.useCallback(() => {
    clearAllFilters();
  }, [clearAllFilters]);

  return (
    <ApplicationsPage
      title={<TitleWithIcon title={MCP_CATALOG_TITLE} objectType={ProjectObjectType.mcpCatalog} />}
      description={MCP_CATALOG_DESCRIPTION}
      empty={false}
      loaded
      provideChildrenPadding
    >
      <CatalogPageLayout
        catalogSources={catalogSources}
        catalogLabels={catalogLabels}
        catalogSourcesLoaded={catalogSourcesLoaded}
        selectedSourceLabel={selectedSourceLabel}
        onSelectSourceLabel={setSelectedSourceLabel}
        isAllItemsView={isAllServersView}
        renderEmptyCategoriesState={() => (
          <EmptyCatalogState
            testid="empty-mcp-catalog-no-categories"
            title="No MCP servers available"
            headerIcon={SearchIcon}
            description="There are no MCP server categories available. Configure sources in settings to get started."
          />
        )}
        renderFilterSidebar={() => <McpCatalogFilters />}
        renderToolbar={() => (
          <McpCatalogSourceLabelSelector
            searchTerm={searchQuery}
            onSearch={handleSearch}
            onClearSearch={handleClearSearch}
            onResetAllFilters={handleResetAllFilters}
          />
        )}
        renderAllItemsView={() => <McpCatalogAllServersView searchTerm={searchQuery} />}
        renderGalleryView={(isSingleCategory, singleCategoryLabel) => (
          <McpCatalogGalleryView
            handleFilterReset={handleResetAllFilters}
            isSingleCategory={isSingleCategory}
            singleCategoryLabel={singleCategoryLabel}
          />
        )}
      />
    </ApplicationsPage>
  );
};

export default McpCatalog;
