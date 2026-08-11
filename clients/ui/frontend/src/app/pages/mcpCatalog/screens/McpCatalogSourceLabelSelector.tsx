import * as React from 'react';
import { CatalogActiveFilters, CatalogSourceLabelSelector } from '~/app/shared/components/catalog';
import { McpCatalogContext } from '~/app/context/mcpCatalog/McpCatalogContext';
import { hasMcpFiltersApplied } from '~/app/pages/mcpCatalog/utils/mcpCatalogUtils';
import { MCP_FILTER_KEYS, MCP_FILTER_CATEGORY_NAMES } from '~/app/pages/mcpCatalog/const';
import McpCatalogSourceLabelBlocks from './McpCatalogSourceLabelBlocks';

type McpCatalogSourceLabelSelectorProps = {
  searchTerm: string;
  onSearch: (term: string) => void;
  onClearSearch: () => void;
  onResetAllFilters: () => void;
};

const McpCatalogSourceLabelSelector: React.FC<McpCatalogSourceLabelSelectorProps> = ({
  searchTerm,
  onSearch,
  onClearSearch,
  onResetAllFilters,
}) => {
  const { filters, setFilters } = React.useContext(McpCatalogContext);
  const hasFiltersAppliedValue = hasMcpFiltersApplied(filters, searchTerm);

  return (
    <CatalogSourceLabelSelector
      searchTerm={searchTerm}
      onSearch={onSearch}
      onClearSearch={onClearSearch}
      onResetAllFilters={onResetAllFilters}
      hasFiltersApplied={hasFiltersAppliedValue}
      searchPlaceholder="Search by name, keyword, or description"
      searchInputTestId="mcp-catalog-search-input"
      searchButtonTestId="mcp-search-button"
      renderActiveFilters={() => (
        <CatalogActiveFilters
          filterKeys={MCP_FILTER_KEYS}
          categoryNames={MCP_FILTER_CATEGORY_NAMES}
          filters={filters}
          setFilters={setFilters}
          testIdPrefix="mcp-filter"
        />
      )}
      renderSourceLabelBlocks={() => <McpCatalogSourceLabelBlocks />}
    />
  );
};

export default McpCatalogSourceLabelSelector;
