import * as React from 'react';
import { ApplicationsPage, ProjectObjectType, TitleWithIcon } from 'mod-arch-shared';
import { SearchIcon } from '@patternfly/react-icons';
import ModelCatalogFilters from '~/app/pages/modelCatalog/components/ModelCatalogFilters';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import { CategoryName } from '~/app/modelCatalogTypes';
import { useHasVisibleFiltersApplied } from '~/app/hooks/modelCatalog/useHasVisibleFiltersApplied';
import { CatalogPageLayout, EmptyCatalogState } from '~/app/shared/components/catalog';
import { MODEL_CATALOG_TITLE, MODEL_CATALOG_DESCRIPTION } from '~/app/pages/modelCatalog/const';
import ModelCatalogSourceLabelSelectorNavigator from './ModelCatalogSourceLabelSelectorNavigator';
import ModelCatalogAllModelsView from './ModelCatalogAllModelsView';
import ModelCatalogGalleryView from './ModelCatalogGalleryView';

const ModelCatalog: React.FC = () => {
  const [searchTerm, setSearchTerm] = React.useState('');
  const {
    selectedSourceLabel,
    setSelectedSourceLabel,
    clearAllFilters,
    catalogSources,
    catalogLabels,
    catalogSourcesLoaded,
    emptyCategoryLabels,
    setCategoryCount,
  } = React.useContext(ModelCatalogContext);
  const filtersApplied = useHasVisibleFiltersApplied();

  const isAllModelsView =
    selectedSourceLabel === CategoryName.allModels && !searchTerm && !filtersApplied;

  const handleSearch = React.useCallback((term: string) => {
    setSearchTerm(term);
  }, []);

  const handleClearSearch = React.useCallback(() => {
    setSearchTerm('');
  }, []);

  const handleFilterReset = React.useCallback(() => {
    setSearchTerm('');
    // clearAllFilters clears basic filters to empty and resets performance filters to defaults
    clearAllFilters();
  }, [clearAllFilters]);

  return (
    <ApplicationsPage
      title={
        <TitleWithIcon title={MODEL_CATALOG_TITLE} objectType={ProjectObjectType.modelCatalog} />
      }
      description={MODEL_CATALOG_DESCRIPTION}
      empty={false}
      loaded
      provideChildrenPadding
    >
      <CatalogPageLayout
        catalogSources={catalogSources}
        catalogLabels={catalogLabels}
        catalogSourcesLoaded={catalogSourcesLoaded}
        selectedSourceLabel={selectedSourceLabel}
        onSelectSourceLabel={setSelectedSourceLabel}
        isAllItemsView={isAllModelsView}
        emptyCategoryLabels={emptyCategoryLabels}
        setCategoryCount={setCategoryCount}
        renderEmptyCategoriesState={() => (
          <EmptyCatalogState
            testid="empty-model-catalog-no-categories"
            title="No models available"
            headerIcon={SearchIcon}
            description="There are no model categories available. Configure model sources in settings to get started."
          />
        )}
        renderFilterSidebar={() => <ModelCatalogFilters />}
        renderToolbar={() => (
          <ModelCatalogSourceLabelSelectorNavigator
            searchTerm={searchTerm}
            onSearch={handleSearch}
            onClearSearch={handleClearSearch}
            onResetAllFilters={handleFilterReset}
          />
        )}
        renderAllItemsView={() => <ModelCatalogAllModelsView searchTerm={searchTerm} />}
        renderGalleryView={(isSingleCategory, singleCategoryLabel) => (
          <ModelCatalogGalleryView
            searchTerm={searchTerm}
            handleFilterReset={handleFilterReset}
            isSingleCategory={isSingleCategory}
            singleCategoryLabel={singleCategoryLabel}
          />
        )}
      />
    </ApplicationsPage>
  );
};

export default ModelCatalog;
