import * as React from 'react';
import {
  Button,
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
import { useThemeContext } from 'mod-arch-kubeflow';
import { ThemeAwareSearchInput } from 'mod-arch-shared';
import { RESET_ALL_FILTERS_LABEL } from './constants';
import './catalogToolbar.css';

export type CatalogSourceLabelSelectorProps = {
  searchTerm: string;
  onSearch: (term: string) => void;
  onClearSearch: () => void;
  onResetAllFilters: () => void;
  /** Drives default clear-all visibility and conditional active-filter rendering. */
  hasFiltersApplied: boolean;
  /**
   * Explicit override for Toolbar clearAllFilters.
   * Defaults to `hasFiltersApplied`.
   */
  showResetAllButton?: boolean;
  /**
   * Remount Toolbar when filter state toggles (MCP/Agents key trick).
   * Default true. Model sets false.
   */
  remountToolbarOnFilterChange?: boolean;
  searchPlaceholder: string;
  searchInputTestId: string;
  searchButtonTestId: string;
  searchAriaLabel?: string;
  /** Render active filter chips in the toolbar. */
  renderActiveFilters?: () => React.ReactNode;
  /**
   * Always render active filters even when none are applied.
   * Model needs this for the PF ToolbarFilter mount workaround.
   */
  alwaysMountActiveFilters?: boolean;
  /** Category toggle blocks row. */
  renderSourceLabelBlocks?: () => React.ReactNode;
  /** Right-side content in the category row (e.g. sort dropdown). */
  sourceLabelRowExtra?: React.ReactNode;
  /** Extra StackItems after toolbar/category row (e.g. performance sections). */
  additionalSections?: React.ReactNode;
};

const CatalogSourceLabelSelector: React.FC<CatalogSourceLabelSelectorProps> = ({
  searchTerm,
  onSearch,
  onClearSearch,
  onResetAllFilters,
  hasFiltersApplied,
  showResetAllButton,
  remountToolbarOnFilterChange = true,
  searchPlaceholder,
  searchInputTestId,
  searchButtonTestId,
  searchAriaLabel = 'Search with submit button',
  renderActiveFilters,
  alwaysMountActiveFilters = false,
  renderSourceLabelBlocks,
  sourceLabelRowExtra,
  additionalSections,
}) => {
  const [inputValue, setInputValue] = React.useState(searchTerm || '');
  const { isMUITheme } = useThemeContext();

  const shouldShowResetAll = showResetAllButton ?? hasFiltersApplied;
  const shouldShowActiveFilters =
    Boolean(renderActiveFilters) && (alwaysMountActiveFilters || hasFiltersApplied);

  React.useEffect(() => {
    setInputValue(searchTerm || '');
  }, [searchTerm]);

  const handleClearAllFilters = React.useCallback(() => {
    if (shouldShowResetAll) {
      onResetAllFilters();
    }
  }, [shouldShowResetAll, onResetAllFilters]);

  const handleSearch = React.useCallback(() => {
    if (inputValue.trim() !== searchTerm) {
      onSearch(inputValue.trim());
    }
  }, [inputValue, searchTerm, onSearch]);

  const handleSearchInputChange = React.useCallback((value: string) => {
    setInputValue(value);
  }, []);

  const handleSearchInputSearch = React.useCallback(
    (_: React.SyntheticEvent<HTMLButtonElement>, value: string) => {
      onSearch(value.trim());
    },
    [onSearch],
  );

  const toolbarClearAllProps = shouldShowResetAll
    ? {
        clearAllFilters: handleClearAllFilters,
        clearFiltersButtonText: RESET_ALL_FILTERS_LABEL,
      }
    : undefined;

  return (
    <Stack hasGutter>
      <StackItem>
        <Toolbar
          className="pf-v6-u-pb-0"
          {...(remountToolbarOnFilterChange
            ? { key: hasFiltersApplied ? 'has-filters' : 'no-filters' }
            : {})}
          {...(toolbarClearAllProps ?? {})}
        >
          <ToolbarContent rowWrap={{ default: 'wrap' }}>
            <Flex style={{ flex: 1 }}>
              <ToolbarToggleGroup style={{ flex: 1 }} breakpoint="md" toggleIcon={<FilterIcon />}>
                <ToolbarGroup
                  style={{ flex: 1 }}
                  variant="filter-group"
                  gap={{ default: 'gapMd' }}
                  alignItems="center"
                  className="toolbar-fieldset-wrapper"
                >
                  <ToolbarItem style={{ flex: 1 }}>
                    <ThemeAwareSearchInput
                      data-testid={searchInputTestId}
                      aria-label={searchAriaLabel}
                      className="toolbar-fieldset-wrapper"
                      placeholder={searchPlaceholder}
                      value={inputValue}
                      onChange={handleSearchInputChange}
                      onSearch={handleSearchInputSearch}
                      onClear={onClearSearch}
                    />
                  </ToolbarItem>
                  <ToolbarItem>
                    {isMUITheme && (
                      <Button
                        isInline
                        aria-label="arrow-right-button"
                        data-testid={searchButtonTestId}
                        variant="link"
                        icon={<ArrowRightIcon />}
                        iconPosition="right"
                        onClick={handleSearch}
                      />
                    )}
                  </ToolbarItem>
                </ToolbarGroup>
              </ToolbarToggleGroup>
              {shouldShowActiveFilters && renderActiveFilters?.()}
            </Flex>
          </ToolbarContent>
        </Toolbar>
      </StackItem>
      {renderSourceLabelBlocks && (
        <StackItem>
          <Flex
            justifyContent={{ default: 'justifyContentSpaceBetween' }}
            alignItems={{ default: 'alignItemsCenter' }}
          >
            {renderSourceLabelBlocks()}
            {sourceLabelRowExtra}
          </Flex>
        </StackItem>
      )}
      {additionalSections}
    </Stack>
  );
};

export default CatalogSourceLabelSelector;
