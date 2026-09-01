import { Alert, AlertActionCloseButton, Content, StackItem } from '@patternfly/react-core';
import React from 'react';
import { BASIC_FILTER_KEYS } from '~/concepts/modelCatalog/const';
import ModelCatalogActiveFilters from '~/app/pages/modelCatalog/components/ModelCatalogActiveFilters';
import HardwareConfigurationFilterToolbar from '~/app/pages/modelCatalog/components/HardwareConfigurationFilterToolbar';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import { CatalogSourceLabelSelector, getActiveSourceLabels } from '~/app/shared/components/catalog';
import { hasFiltersApplied } from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import ModelCatalogSortDropdown from '~/app/pages/modelCatalog/components/ModelCatalogSortDropdown';
import ModelCatalogSourceLabelBlocks from './ModelCatalogSourceLabelBlocks';

const noop = (): void => undefined;

type ModelCatalogSourceLabelSelectorProps = {
  searchTerm?: string;
  onSearch?: (term: string) => void;
  onClearSearch?: () => void;
  onResetAllFilters?: () => void;
};

const ModelCatalogSourceLabelSelector: React.FC<ModelCatalogSourceLabelSelectorProps> = ({
  searchTerm,
  onSearch,
  onClearSearch,
  onResetAllFilters,
}) => {
  const {
    catalogSources,
    catalogLabels,
    filters,
    performanceViewEnabled,
    performanceFiltersChangedOnDetailsPage,
    setPerformanceFiltersChangedOnDetailsPage,
    lastViewedModelName,
  } = React.useContext(ModelCatalogContext);

  const hasMultipleCategories = React.useMemo(
    () => getActiveSourceLabels(catalogSources, catalogLabels).length > 1,
    [catalogSources, catalogLabels],
  );

  // Only show basic filters in the main chip bar - performance filters have their own section
  const filtersToShow = BASIC_FILTER_KEYS;

  // Check if any basic filters are applied
  const hasBasicFiltersApplied = React.useMemo(
    () => hasFiltersApplied(filters, filtersToShow),
    [filters, filtersToShow],
  );

  // Check if search term is active
  const hasSearchTerm = Boolean(searchTerm && searchTerm.trim().length > 0);

  // When performance toggle is ON, we need to check if performance filters differ from defaults
  // When toggle is OFF, we just check if any filters have values
  const hasActiveFilters = React.useMemo(() => {
    if (hasSearchTerm) {
      return true;
    }

    if (hasBasicFiltersApplied) {
      return true;
    }

    // When performance view is OFF, only basic filters matter
    if (!performanceViewEnabled) {
      return false;
    }

    // When performance view is ON, check if any performance filters differ from defaults
    // (the HardwareConfigurationFilterToolbar handles showing its own "Clear all filters")
    // The top toolbar should only show "Reset all filters" if basic filters are applied
    // or if there's a search term
    return false;
  }, [hasSearchTerm, hasBasicFiltersApplied, performanceViewEnabled]);

  const shouldShowAlert = performanceViewEnabled && performanceFiltersChangedOnDetailsPage;

  return (
    <CatalogSourceLabelSelector
      searchTerm={searchTerm || ''}
      onSearch={onSearch ?? noop}
      onClearSearch={onClearSearch ?? noop}
      onResetAllFilters={onResetAllFilters ?? noop}
      hasFiltersApplied={hasActiveFilters}
      showResetAllButton={
        Boolean(onResetAllFilters) &&
        !performanceViewEnabled &&
        (hasBasicFiltersApplied || hasSearchTerm)
      }
      remountToolbarOnFilterChange={false}
      searchPlaceholder="Filter by name, description and provider"
      searchInputTestId="search-input"
      searchButtonTestId="search-button"
      alwaysMountActiveFilters={Boolean(onResetAllFilters)}
      renderActiveFilters={() => (
        <ModelCatalogActiveFilters
          filtersToShow={filtersToShow}
          forceHideLabels={performanceViewEnabled}
        />
      )}
      renderSourceLabelBlocks={
        hasMultipleCategories ? () => <ModelCatalogSourceLabelBlocks /> : undefined
      }
      sourceLabelRowExtra={
        hasMultipleCategories ? (
          <ModelCatalogSortDropdown performanceViewEnabled={performanceViewEnabled} />
        ) : undefined
      }
      additionalSections={
        <>
          {performanceViewEnabled && (
            <>
              <StackItem>
                <Content component="h2" className="pf-v6-u-font-weight-bold">
                  Workload and performance constraints
                </Content>
              </StackItem>
              <StackItem>
                <HardwareConfigurationFilterToolbar
                  onResetAllFilters={onResetAllFilters}
                  includeBasicFilters
                  includePerformanceFilters={performanceViewEnabled}
                />
              </StackItem>
            </>
          )}
          {shouldShowAlert && (
            <StackItem>
              <Alert
                variant="info"
                isInline
                className="pf-v6-u-mb-lg"
                title={
                  lastViewedModelName
                    ? `The performance constraints and results have been updated to match the constraints you applied to the ${lastViewedModelName} model details page.`
                    : 'The performance constraints and results have been updated to match the constraints you applied to the model details page.'
                }
                actionClose={
                  <AlertActionCloseButton
                    onClose={() => {
                      setPerformanceFiltersChangedOnDetailsPage(false);
                    }}
                  />
                }
                data-testid="performance-filters-updated-alert"
              />
            </StackItem>
          )}
        </>
      }
    />
  );
};

export default ModelCatalogSourceLabelSelector;
