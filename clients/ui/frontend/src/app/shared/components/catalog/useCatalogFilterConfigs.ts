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
        .map((key): FilterPanelItem | null => {
          const option = filterOptions?.[key];
          if (!option?.values || option.values.length === 0) {
            return null;
          }

          const currentValues = selectedFilters[key] ?? [];
          const mapping = labelMappings?.[key];

          return {
            key,
            title: filterNames[key] ?? key,
            filterValues: option.values,
            selectedValues: currentValues,
            onToggle: (value: string, checked: boolean) => {
              const latest = selectedFiltersRef.current[key] ?? [];
              const next = checked
                ? latest.includes(value)
                  ? latest
                  : [...latest, value]
                : latest.filter((v) => v !== value);
              onFilterChange(key, next);
            },
            getLabel: mapping ? (value: string) => mapping[value] ?? value : undefined,
          };
        })
        .filter((item): item is FilterPanelItem => item !== null),
    [filterKeys, filterNames, filterOptions, selectedFilters, onFilterChange, labelMappings],
  );
}
