import * as React from 'react';
import { useMcpServersBySourceLabelWithAPI } from '~/app/hooks/mcpServerCatalog/useMcpServersBySourceLabel';
import {
  getLabelDescription,
  getLabelDisplayName,
} from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import { CatalogCategorySection } from '~/app/shared/components/catalog';
import { McpCatalogContext } from '~/app/context/mcpCatalog/McpCatalogContext';
import {
  MCP_CATALOG_GRID_SPAN,
  OTHER_MCP_SERVERS_DISPLAY_NAME,
} from '~/app/pages/mcpCatalog/const';
import McpCatalogCard from '~/app/pages/mcpCatalog/components/McpCatalogCard';

type McpCatalogCategorySectionProps = {
  label: string;
  searchTerm: string;
  pageSize: number;
  onShowMore: (label: string) => void;
};

const McpCatalogCategorySection: React.FC<McpCatalogCategorySectionProps> = ({
  label,
  searchTerm,
  pageSize,
  onShowMore,
}) => {
  const { mcpApiState, catalogLabels } = React.useContext(McpCatalogContext);
  const { mcpServers, mcpServersLoaded, mcpServersLoadError } = useMcpServersBySourceLabelWithAPI(
    mcpApiState,
    {
      sourceLabel: label,
      pageSize,
      searchQuery: searchTerm,
    },
  );

  const categoryTitle = getLabelDisplayName(
    label,
    catalogLabels,
    OTHER_MCP_SERVERS_DISPLAY_NAME,
    'servers',
  );
  const categoryDescription = getLabelDescription(label, catalogLabels);
  const labelSlug = label.toLowerCase().replace(/\s+/g, '-');

  return (
    <CatalogCategorySection
      label={label}
      categoryTitle={categoryTitle}
      categoryDescription={categoryDescription}
      items={mcpServers.items}
      loaded={mcpServersLoaded}
      loadError={mcpServersLoadError}
      pageSize={pageSize}
      onShowMore={onShowMore}
      renderCard={(server) => <McpCatalogCard server={server} />}
      getItemKey={(server) => server.id}
      gridSpans={MCP_CATALOG_GRID_SPAN}
      loadingScreenReaderText={`Loading ${label} servers`}
      testIds={{
        title: `mcp-category-title-${label}`,
        showMore: `mcp-show-all-${labelSlug}`,
        error: `mcp-error-state-${label}`,
        skeleton: (index) => `mcp-category-skeleton-${labelSlug}-${index}`,
        empty: `empty-mcp-catalog-state-${label}`,
      }}
    />
  );
};
export default McpCatalogCategorySection;
