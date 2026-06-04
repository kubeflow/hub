import { Button, EmptyStateVariant } from '@patternfly/react-core';
import { ChartBarIcon, SearchIcon } from '@patternfly/react-icons';
import React from 'react';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import { useCatalogModelsBySources } from '~/app/hooks/modelCatalog/useCatalogModelsBySource';
import { SourceLabel } from '~/app/shared/types/catalogTypes';
import { CategoryName, type CatalogModel } from '~/app/modelCatalogTypes';
import ModelCatalogCard from '~/app/pages/modelCatalog/components/ModelCatalogCard';
import {
  getLabelDisplayName,
  getLabelDescription,
  CatalogGalleryLayout,
  EmptyCatalogState,
} from '~/app/shared/components/catalog';
import {
  getSourceFromSourceId,
  getBasicFiltersOnly,
  getActiveLatencyFieldName,
  getSortParams,
  generateCategoryName,
  hasFiltersApplied,
  isValueDifferentFromDefault,
} from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import ModelCatalogSortDropdown from '~/app/pages/modelCatalog/components/ModelCatalogSortDropdown';
import {
  ModelCatalogNumberFilterKey,
  ModelCatalogStringFilterKey,
  parseLatencyFilterKey,
  BASIC_FILTER_KEYS,
  PERFORMANCE_FILTER_KEYS,
} from '~/concepts/modelCatalog/const';

type ModelCatalogPageProps = {
  searchTerm: string;
  handleFilterReset: () => void;
  isSingleCategory?: boolean;
  singleCategoryLabel?: string;
};

const ModelCatalogGalleryView: React.FC<ModelCatalogPageProps> = ({
  searchTerm,
  handleFilterReset,
  isSingleCategory = false,
  singleCategoryLabel,
}) => {
  const {
    selectedSourceLabel,
    filters,
    filterOptions,
    filterOptionsLoaded,
    filterOptionsLoadError,
    catalogSources,
    catalogLabels,
    catalogLabelsLoaded,
    catalogLabelsLoadError,
    setPerformanceViewEnabled,
    setSelectedSourceLabel,
    performanceViewEnabled,
    sortBy,
    getPerformanceFilterDefaultValue,
  } = React.useContext(ModelCatalogContext);

  // When performance view is disabled, exclude performance filters from API queries
  // Memoize to prevent infinite re-fetching
  const effectiveFilterData = React.useMemo(
    () => (performanceViewEnabled ? filters : getBasicFiltersOnly(filters)),
    [performanceViewEnabled, filters],
  );

  // Optimize: Only track the active latency field instead of entire filterData
  // This prevents unnecessary recalculations when non-latency filters change
  const activeLatencyField = React.useMemo(() => getActiveLatencyFieldName(filters), [filters]);

  const sortParams = React.useMemo(
    () => getSortParams(sortBy, performanceViewEnabled, activeLatencyField),
    [sortBy, performanceViewEnabled, activeLatencyField],
  );

  // Derive performance params to pass to the models API when performance view is enabled
  const performanceParams = React.useMemo(() => {
    if (!performanceViewEnabled) {
      return undefined;
    }

    const targetRPS = filters[ModelCatalogNumberFilterKey.MAX_RPS];
    const latencyProperty = activeLatencyField
      ? parseLatencyFilterKey(activeLatencyField).propertyKey
      : undefined;

    return {
      targetRPS,
      latencyProperty,
      recommendations: true,
    };
  }, [performanceViewEnabled, filters, activeLatencyField]);

  const { catalogModels, catalogModelsLoaded, catalogModelsLoadError } = useCatalogModelsBySources(
    '',
    selectedSourceLabel === CategoryName.allModels ? undefined : selectedSourceLabel,
    10,
    searchTerm,
    effectiveFilterData,
    filterOptions,
    undefined, // filterQuery - will be computed from filterData and filterOptions
    sortParams.orderBy,
    sortParams.sortOrder,
    performanceParams,
  );

  const loaded = catalogModelsLoaded && filterOptionsLoaded && catalogLabelsLoaded;
  const loadError = catalogModelsLoadError || filterOptionsLoadError || catalogLabelsLoadError;

  const isNoLabelsSection = selectedSourceLabel === SourceLabel.other;

  // Check if basic filters are applied
  const hasBasicFiltersApplied = React.useMemo(
    () => hasFiltersApplied(filters, BASIC_FILTER_KEYS),
    [filters],
  );

  // Check if Hardware Configuration filter is applied
  const hasHardwareConfigurationApplied = React.useMemo(() => {
    const hardwareConfig = filters[ModelCatalogStringFilterKey.HARDWARE_CONFIGURATION];
    return Array.isArray(hardwareConfig) && hardwareConfig.length > 0;
  }, [filters]);

  // When performance view is enabled, performance filters have default values.
  const hasPerformanceFiltersChanged = React.useMemo(() => {
    if (!performanceViewEnabled) {
      return false;
    }
    return PERFORMANCE_FILTER_KEYS.some((filterKey) => {
      const filterValue = filters[filterKey];
      const defaultValue = getPerformanceFilterDefaultValue(filterKey);

      if (filterValue === undefined) {
        return false;
      }

      if (Array.isArray(filterValue) && filterValue.length === 0) {
        return false;
      }

      return isValueDifferentFromDefault(filterValue, defaultValue);
    });
  }, [performanceViewEnabled, filters, getPerformanceFilterDefaultValue]);

  const noUserFiltersOrSearch = React.useMemo(
    () =>
      !hasBasicFiltersApplied &&
      !hasHardwareConfigurationApplied &&
      !hasPerformanceFiltersChanged &&
      !searchTerm,
    [
      hasBasicFiltersApplied,
      hasHardwareConfigurationApplied,
      hasPerformanceFiltersChanged,
      searchTerm,
    ],
  );

  const shouldShowPerformanceEmptyState = React.useMemo(() => {
    const isEmptyResult = catalogModels.items.length === 0;
    const isNotAllModelsCategory = selectedSourceLabel !== CategoryName.allModels;
    const isPerformanceExcludedSection = isNoLabelsSection || noUserFiltersOrSearch;

    return (
      performanceViewEnabled &&
      isEmptyResult &&
      isNotAllModelsCategory &&
      isPerformanceExcludedSection
    );
  }, [
    performanceViewEnabled,
    catalogModels.items.length,
    selectedSourceLabel,
    isNoLabelsSection,
    noUserFiltersOrSearch,
  ]);

  const handleDisablePerformanceView = () => {
    setPerformanceViewEnabled(false);
  };

  const handleSelectAllModels = () => {
    setSelectedSourceLabel(CategoryName.allModels);
  };

  const effectiveCategoryLabel = singleCategoryLabel || selectedSourceLabel || '';
  const categoryTitle = isSingleCategory
    ? getLabelDisplayName(effectiveCategoryLabel, catalogLabels)
    : undefined;
  const categoryDescription = isSingleCategory
    ? getLabelDescription(effectiveCategoryLabel, catalogLabels)
    : undefined;

  return (
    <CatalogGalleryLayout
      items={catalogModels.items}
      loaded={loaded}
      loadError={loadError}
      renderCard={(model: CatalogModel) => (
        <ModelCatalogCard
          model={model}
          source={getSourceFromSourceId(model.source_id || '', catalogSources)}
        />
      )}
      getItemKey={(model: CatalogModel) => `${model.name}/${model.source_id}`}
      hasMore={catalogModels.hasMore}
      isLoadingMore={catalogModels.isLoadingMore}
      onLoadMore={catalogModels.loadMore}
      loadMoreLabel="Load more models"
      loadingMoreLabel="Loading more catalog models..."
      loadingLabel="Loading model catalog..."
      errorTitle="Failed to load model catalog"
      categoryTitle={categoryTitle}
      categoryDescription={categoryDescription}
      headerExtra={
        isSingleCategory && performanceViewEnabled ? (
          <ModelCatalogSortDropdown performanceViewEnabled={performanceViewEnabled} />
        ) : undefined
      }
      renderExtraEmptyStates={() => {
        if (shouldShowPerformanceEmptyState) {
          return (
            <EmptyCatalogState
              testid="performance-empty-state"
              title={
                isSingleCategory
                  ? 'No performance data available'
                  : 'No performance data available in selected category'
              }
              headerIcon={ChartBarIcon}
              variant={EmptyStateVariant.lg}
              description={
                isSingleCategory ? (
                  'No models have performance data available. Turn off model performance view to see all models.'
                ) : (
                  <>
                    No models in the{' '}
                    <strong>
                      {selectedSourceLabel === 'null'
                        ? CategoryName.otherModels
                        : generateCategoryName(selectedSourceLabel || '')}
                    </strong>{' '}
                    category have performance data. Select another model category, or turn off model
                    performance view to see models in the selected category.
                  </>
                )
              }
              primaryAction={
                isSingleCategory ? undefined : (
                  <Button variant="primary" onClick={handleSelectAllModels}>
                    View all models with performance data
                  </Button>
                )
              }
              secondaryAction={
                <Button variant="link" onClick={handleDisablePerformanceView}>
                  Turn off model performance view
                </Button>
              }
            />
          );
        }

        if (catalogModels.items.length === 0 && noUserFiltersOrSearch) {
          return (
            <EmptyCatalogState
              testid="empty-model-catalog-state"
              title="No models available"
              headerIcon={SearchIcon}
              description="No models are available in this category"
            />
          );
        }

        return null;
      }}
      renderEmptyState={() => (
        <EmptyCatalogState
          testid="empty-model-catalog-state"
          title="No results found"
          headerIcon={SearchIcon}
          description="Adjust your filters and try again."
          primaryAction={
            <Button variant="link" onClick={handleFilterReset}>
              {performanceViewEnabled && hasPerformanceFiltersChanged
                ? 'Reset all defaults'
                : 'Reset all filters'}
            </Button>
          }
        />
      )}
    />
  );
};

export default ModelCatalogGalleryView;
