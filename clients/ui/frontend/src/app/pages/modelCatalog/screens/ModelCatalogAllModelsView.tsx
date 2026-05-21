import React from 'react';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import { CategoryName } from '~/app/modelCatalogTypes';
import { CatalogAllItemsView } from '~/app/shared/components/catalog';
import ModelCatalogCategorySection from './ModelCatalogCategorySection';

type ModelCatalogAllModelsViewProps = {
  searchTerm: string;
};

const ModelCatalogAllModelsView: React.FC<ModelCatalogAllModelsViewProps> = ({ searchTerm }) => {
  const { catalogSources, catalogLabels, updateSelectedSourceLabel } =
    React.useContext(ModelCatalogContext);

  const handleShowMoreCategory = React.useCallback(
    (categoryLabel: string) => {
      updateSelectedSourceLabel(categoryLabel);
    },
    [updateSelectedSourceLabel],
  );

  return (
    <CatalogAllItemsView
      searchTerm={searchTerm}
      catalogSources={catalogSources}
      catalogLabels={catalogLabels}
      pageSize={4}
      otherSectionKey={CategoryName.otherModels}
      onShowMore={handleShowMoreCategory}
      renderCategorySection={(label, term, pageSize, onShowMore) => (
        <ModelCatalogCategorySection
          label={label}
          searchTerm={term}
          pageSize={pageSize}
          catalogSources={catalogSources}
          onShowMore={onShowMore}
        />
      )}
    />
  );
};

export default ModelCatalogAllModelsView;
