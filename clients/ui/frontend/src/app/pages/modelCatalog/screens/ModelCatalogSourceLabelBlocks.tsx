import React from 'react';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import { CategoryName } from '~/app/modelCatalogTypes';
import { CatalogSourceLabelToggle } from '~/app/shared/components/catalog';

const ModelCatalogSourceLabelBlocks: React.FC = () => {
  const {
    catalogSources,
    catalogLabels,
    setSelectedSourceLabel,
    selectedSourceLabel,
    emptyCategoryLabels,
    categoriesResolved,
  } = React.useContext(ModelCatalogContext);

  if (!categoriesResolved) {
    return null;
  }

  return (
    <CatalogSourceLabelToggle
      catalogSources={catalogSources}
      catalogLabels={catalogLabels}
      selectedSourceLabel={selectedSourceLabel}
      onSelectSourceLabel={(label) => setSelectedSourceLabel(label ?? CategoryName.allModels)}
      allBlockLabel={CategoryName.allModels}
      allBlockDisplayName={CategoryName.allModels}
      emptyCategoryLabels={emptyCategoryLabels}
      className="pf-v6-u-pb-0"
      ariaLabel="Source label selection"
      hideWhenSingleCategory
    />
  );
};

export default ModelCatalogSourceLabelBlocks;
