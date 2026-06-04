import {
  Alert,
  AlertActionCloseButton,
  Button,
  Content,
  Flex,
  Stack,
  StackItem,
  Toolbar,
  ToolbarContent,
  ToolbarGroup,
  ToolbarItem,
  ToolbarToggleGroup,
} from '@patternfly/react-core';
import { ArrowRightIcon, FilterIcon } from '@patternfly/react-icons';
import React from 'react';
import { useThemeContext } from 'mod-arch-kubeflow';
import { BASIC_FILTER_KEYS } from '~/concepts/modelCatalog/const';
import ModelCatalogActiveFilters from '~/app/pages/modelCatalog/components/ModelCatalogActiveFilters';
import HardwareConfigurationFilterToolbar from '~/app/pages/modelCatalog/components/HardwareConfigurationFilterToolbar';
import ThemeAwareSearchInput from '~/app/pages/modelRegistry/screens/components/ThemeAwareSearchInput';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import { getActiveSourceLabels } from '~/app/shared/components/catalog';
import { hasFiltersApplied } from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import ModelCatalogSortDropdown from '~/app/pages/modelCatalog/components/ModelCatalogSortDropdown';
import ModelCatalogSourceLabelBlocks from './ModelCatalogSourceLabelBlocks';

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
  const [inputValue, setInputValue] = React.useState(searchTerm || '');
  const { isMUITheme } = useThemeContext();
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

  const handleClearAllFilters = React.useCallback(() => {
    if (hasActiveFilters && onResetAllFilters) {
      onResetAllFilters();
    }
  }, [hasActiveFilters, onResetAllFilters]);

  React.useEffect(() => {
    setInputValue(searchTerm || '');
  }, [searchTerm]);

  const handleModelSearch = () => {
    if (onSearch && inputValue.trim() !== searchTerm) {
      onSearch(inputValue.trim());
    }
  };

  const handleClear = () => {
    if (onClearSearch) {
      onClearSearch();
    }
  };

  const handleSearchInputChange = (value: string) => {
    setInputValue(value);
  };

  const handleSearchInputSearch = (_: React.SyntheticEvent<HTMLButtonElement>, value: string) => {
    if (onSearch) {
      onSearch(value.trim());
    }
  };

  return (
    <Stack hasGutter>
      <StackItem>
        <Toolbar
          className="pf-v6-u-pb-0"
          // Use PatternFly's native clearAllFilters - it automatically shows/hides based on ToolbarFilter labels
          // When performance view is OFF, show reset button for basic filters
          // When performance view is ON, the HardwareConfigurationFilterToolbar handles resetting
          {...(onResetAllFilters && !performanceViewEnabled && hasBasicFiltersApplied
            ? {
                clearAllFilters: handleClearAllFilters,
                clearFiltersButtonText: 'Reset all filters',
              }
            : {})}
        >
          <ToolbarContent rowWrap={{ default: 'wrap' }}>
            <Flex style={{ flex: 1 }}>
              <ToolbarToggleGroup style={{ flex: 1 }} breakpoint="md" toggleIcon={<FilterIcon />}>
                <ToolbarGroup
                  variant="filter-group"
                  style={{ flex: 1 }}
                  gap={{ default: 'gapMd' }}
                  alignItems="center"
                  className="toolbar-fieldset-wrapper"
                >
                  <ToolbarItem style={{ flex: 1 }}>
                    <ThemeAwareSearchInput
                      data-testid="search-input"
                      aria-label="Search with submit button"
                      className="toolbar-fieldset-wrapper"
                      placeholder="Filter by name, description and provider"
                      value={inputValue}
                      onChange={handleSearchInputChange}
                      onSearch={handleSearchInputSearch}
                      onClear={handleClear}
                    />
                  </ToolbarItem>
                  <ToolbarItem>
                    {isMUITheme && (
                      <Button
                        isInline
                        aria-label="arrow-right-button"
                        data-testid="search-button"
                        variant="link"
                        icon={<ArrowRightIcon />}
                        iconPosition="right"
                        onClick={handleModelSearch}
                      />
                    )}
                  </ToolbarItem>
                </ToolbarGroup>
              </ToolbarToggleGroup>
              {/* When toggle is OFF, show basic filter chips in the main toolbar */}
              {/* When toggle is ON, keep ToolbarFilters mounted with empty labels to work around */}
              {/* PF ToolbarFilter not cleaning up filter count on unmount (PF#12247) */}
              {onResetAllFilters && (
                <ModelCatalogActiveFilters
                  filtersToShow={filtersToShow}
                  forceHideLabels={performanceViewEnabled}
                />
              )}
            </Flex>
          </ToolbarContent>
        </Toolbar>
      </StackItem>
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
      {hasMultipleCategories && (
        <StackItem>
          <Flex
            justifyContent={{ default: 'justifyContentSpaceBetween' }}
            alignItems={{ default: 'alignItemsCenter' }}
          >
            <ModelCatalogSourceLabelBlocks />
            <ModelCatalogSortDropdown performanceViewEnabled={performanceViewEnabled} />
          </Flex>
        </StackItem>
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
    </Stack>
  );
};

export default ModelCatalogSourceLabelSelector;
