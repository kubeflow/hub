import * as React from 'react';
import { McpCatalogContext } from '~/app/context/mcpCatalog/McpCatalogContext';
import { CatalogSourceLabelToggle } from '~/app/shared/components/catalog';
import { getLabelDisplayName } from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import { OTHER_MCP_SERVERS_DISPLAY_NAME } from '~/app/pages/mcpCatalog/const';

const ALL_SERVERS_LABEL = 'All MCP servers';

const McpCatalogSourceLabelBlocks: React.FC = () => {
  const { catalogSources, catalogLabels, selectedSourceLabel, setSelectedSourceLabel } =
    React.useContext(McpCatalogContext);

  const getLabelDisplayNameForMcp = React.useCallback(
    (label: string) =>
      getLabelDisplayName(label, catalogLabels, OTHER_MCP_SERVERS_DISPLAY_NAME, 'servers'),
    [catalogLabels],
  );

  return (
    <CatalogSourceLabelToggle
      catalogSources={catalogSources}
      catalogLabels={catalogLabels}
      selectedSourceLabel={selectedSourceLabel}
      onSelectSourceLabel={setSelectedSourceLabel}
      allBlockLabel={undefined}
      allBlockDisplayName={ALL_SERVERS_LABEL}
      testId="mcp-catalog-category-toggle"
      ariaLabel="MCP category selection"
      hideWhenSingleCategory
      getLabelDisplayNameOverride={getLabelDisplayNameForMcp}
      getTestId={(blockId) => `mcp-category-${blockId}`}
    />
  );
};

export default McpCatalogSourceLabelBlocks;
