import * as React from 'react';

export type FilterPanelItem = {
  key: string;
  title: string;
  filterValues: string[];
  selectedValues: string[];
  onToggle: (value: string, checked: boolean) => void;
  getLabel?: (value: string) => string;
  footer?: React.ReactNode;
  visible?: boolean;
  testIdBase?: string;
  getCheckboxTestId?: (value: string) => string;
};

type CatalogFilterConfigsInput = {
  filterKeys: string[];
  filterNames: Record<string, string>;
  filterOptions: Record<string, { type?: string; values?: string[] }> | undefined;
  selectedFilters: Record<string, string[] | undefined>;
  onFilterChange: (key: string, values: string[]) => void;
  labelMappings?: Record<string, Record<string, string>>;
};

/**
 * Builds a FilterPanelItem[] from catalog-specific config.
 * Handles toggle logic, value extraction, and label mappings.
 * The caller passes catalog-specific context data; the hook returns
 * a ready-made array for CatalogFilterPanel.
 */
export function useCatalogFilterConfigs({
  filterKeys,
  filterNames,
  filterOptions,
  selectedFilters,
  onFilterChange,
  labelMappings,
}: CatalogFilterConfigsInput): FilterPanelItem[] {
  const selectedFiltersRef = React.useRef(selectedFilters);
  selectedFiltersRef.current = selectedFilters;

  return React.useMemo(
    () =>
      filterKeys
        .map((filterKey): FilterPanelItem | null => {
          const filterOption = filterOptions?.[filterKey];
          if (!filterOption?.values || filterOption.values.length === 0) {
            return null;
          }

          const selectedValues = selectedFilters[filterKey] ?? [];
          const labelMapping = labelMappings?.[filterKey];

          return {
            key: filterKey,
            title: filterNames[filterKey] ?? filterKey,
            filterValues: filterOption.values,
            selectedValues,
            onToggle: (filterValue: string, isChecked: boolean) => {
              const currentSelectedValues = selectedFiltersRef.current[filterKey] ?? [];
              const updatedValues = isChecked
                ? currentSelectedValues.includes(filterValue)
                  ? currentSelectedValues
                  : [...currentSelectedValues, filterValue]
                : currentSelectedValues.filter((selected) => selected !== filterValue);
              onFilterChange(filterKey, updatedValues);
            },
            getLabel: labelMapping
              ? (filterValue: string) => labelMapping[filterValue] ?? filterValue
              : undefined,
          };
        })
        .filter((item): item is FilterPanelItem => item !== null),
    [filterKeys, filterNames, filterOptions, selectedFilters, onFilterChange, labelMappings],
  );
}
