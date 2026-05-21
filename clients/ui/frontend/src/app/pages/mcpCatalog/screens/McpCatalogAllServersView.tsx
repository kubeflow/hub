import React from 'react';
import { McpCatalogContext } from '~/app/context/mcpCatalog/McpCatalogContext';
import { CatalogAllItemsView } from '~/app/shared/components/catalog';
import McpCatalogCategorySection from './McpCatalogCategorySection';

type McpCatalogAllServersViewProps = {
  searchTerm: string;
};

const McpCatalogAllServersView: React.FC<McpCatalogAllServersViewProps> = ({ searchTerm }) => {
  const { catalogSources, catalogLabels, setSelectedSourceLabel } =
    React.useContext(McpCatalogContext);

  const handleShowMoreCategory = React.useCallback(
    (categoryLabel: string) => {
      setSelectedSourceLabel(categoryLabel);
    },
    [setSelectedSourceLabel],
  );

  return (
    <CatalogAllItemsView
      searchTerm={searchTerm}
      catalogSources={catalogSources}
      catalogLabels={catalogLabels}
      pageSize={4}
      otherSectionKey="other-servers"
      onShowMore={handleShowMoreCategory}
      renderCategorySection={(label, term, pageSize, onShowMore) => (
        <McpCatalogCategorySection
          label={label}
          searchTerm={term}
          pageSize={pageSize}
          onShowMore={onShowMore}
        />
      )}
    />
  );
};

export default McpCatalogAllServersView;
