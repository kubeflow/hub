import React from 'react';
import { CatalogSourceList } from '~/app/modelCatalogTypes';
import { useCatalogModelsBySources } from '~/app/hooks/modelCatalog/useCatalogModelsBySource';
import { CatalogCategorySection } from '~/app/shared/components/catalog';
import {
  getLabelDescription,
  getLabelDisplayName,
  getSourceFromSourceId,
} from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import ModelCatalogCard from '~/app/pages/modelCatalog/components/ModelCatalogCard';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';

type CategorySectionProps = {
  label: string;
  searchTerm: string;
  pageSize: number;
  catalogSources: CatalogSourceList | null;
  onShowMore: (label: string) => void;
};

const ModelCatalogCategorySection: React.FC<CategorySectionProps> = ({
  label,
  searchTerm,
  pageSize,
  catalogSources,
  onShowMore,
}) => {
  const { catalogLabels } = React.useContext(ModelCatalogContext);
  const { catalogModels, catalogModelsLoaded, catalogModelsLoadError } = useCatalogModelsBySources(
    undefined,
    label,
    pageSize,
    searchTerm,
  );

  // Get display name and description from labels API
  const categoryTitle = getLabelDisplayName(label, catalogLabels);
  const categoryDescription = getLabelDescription(label, catalogLabels);
  const labelSlug = label.toLowerCase().replace(/\s+/g, '-');

  return (
    <CatalogCategorySection
      label={label}
      categoryTitle={categoryTitle}
      categoryDescription={categoryDescription}
      items={catalogModels.items}
      loaded={catalogModelsLoaded}
      loadError={catalogModelsLoadError}
      pageSize={pageSize}
      showAllThreshold={4}
      skeletonCount={4}
      onShowMore={onShowMore}
      renderCard={(model) => (
        <ModelCatalogCard
          model={model}
          source={getSourceFromSourceId(model.source_id || '', catalogSources)}
        />
      )}
      getItemKey={(model) => `${model.name}/${model.source_id}`}
      loadingScreenReaderText={`Loading ${label} models`}
      testIds={{
        title: `title ${label}`,
        showMore: `show-more-button ${labelSlug}`,
        error: `error-state ${label}`,
        skeleton: (index) => `category-skeleton-${labelSlug}-${index}`,
        empty: `empty-model-catalog-state ${label}`,
      }}
    />
  );
};
export default ModelCatalogCategorySection;
