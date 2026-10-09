import React from 'react';
import { CatalogSourceList } from '~/app/shared/types/catalogTypes';
import { useCatalogModelsBySources } from '~/app/hooks/modelCatalog/useCatalogModelsBySource';
import useReportCategoryEmpty from '~/app/hooks/useReportCategoryEmpty';
import {
  CatalogCategorySection,
  getLabelDescription,
  getLabelDisplayName,
} from '~/app/shared/components/catalog';
import { getSourceFromSourceId } from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
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
  const { catalogLabels, reportCategoryEmpty } = React.useContext(ModelCatalogContext);
  const { catalogModels, catalogModelsLoaded, catalogModelsLoadError } = useCatalogModelsBySources(
    undefined,
    label,
    pageSize,
    searchTerm,
  );

  const categoryTitle = getLabelDisplayName(label, catalogLabels);
  const categoryDescription = getLabelDescription(label, catalogLabels);
  const labelSlug = label.toLowerCase().replace(/\s+/g, '-');

  useReportCategoryEmpty(
    reportCategoryEmpty,
    label,
    catalogModelsLoaded,
    catalogModels.items.length,
    searchTerm,
    catalogModelsLoadError,
  );

  if (catalogModelsLoaded && catalogModels.items.length === 0 && !searchTerm) {
    return null;
  }

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
          source={getSourceFromSourceId(model.sourceId || '', catalogSources)}
        />
      )}
      getItemKey={(model) => `${model.name}/${model.sourceId}`}
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
