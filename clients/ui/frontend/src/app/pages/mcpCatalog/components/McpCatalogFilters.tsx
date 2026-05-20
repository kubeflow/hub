import * as React from 'react';
import { McpCatalogContext } from '~/app/context/mcpCatalog/McpCatalogContext';
import { CatalogFilterPanel, useCatalogFilterConfigs } from '~/app/shared/components/catalog';
import { MCP_FILTER_KEYS, MCP_FILTER_CATEGORY_NAMES } from '~/app/pages/mcpCatalog/const';

const McpCatalogFilters: React.FC = () => {
  const { filters, setFilters, filterOptions, filterOptionsLoaded, filterOptionsLoadError } =
    React.useContext(McpCatalogContext);

  const onFilterChange = React.useCallback(
    (key: string, values: string[]) => {
      setFilters((prev) => ({ ...prev, [key]: values }));
    },
    [setFilters],
  );

  const filterPanelItems = useCatalogFilterConfigs({
    filterKeys: MCP_FILTER_KEYS,
    filterNames: MCP_FILTER_CATEGORY_NAMES,
    filterOptions: filterOptions?.filters,
    selectedFilters: filters,
    onFilterChange,
  });

  return (
    <CatalogFilterPanel
      loaded={filterOptionsLoaded}
      loadError={filterOptionsLoadError}
      filters={filterPanelItems}
      testIdPrefix="mcp-filter"
    />
  );
};

export default McpCatalogFilters;
