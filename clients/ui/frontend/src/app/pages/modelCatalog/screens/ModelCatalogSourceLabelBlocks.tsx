import React from 'react';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import { CategoryName } from '~/app/modelCatalogTypes';
import { CatalogSourceLabelToggle } from '~/app/shared/components/catalog';

const ModelCatalogSourceLabelBlocks: React.FC = () => {
  const { catalogSources, catalogLabels, updateSelectedSourceLabel, selectedSourceLabel } =
    React.useContext(ModelCatalogContext);

  return (
    <CatalogSourceLabelToggle
      catalogSources={catalogSources}
      catalogLabels={catalogLabels}
      selectedSourceLabel={selectedSourceLabel}
      onSelectSourceLabel={(label) => updateSelectedSourceLabel(label ?? CategoryName.allModels)}
      allBlockLabel={CategoryName.allModels}
      allBlockDisplayName={CategoryName.allModels}
      className="pf-v6-u-pb-0"
      ariaLabel="Source label selection"
    />
  );
};

export default ModelCatalogSourceLabelBlocks;
