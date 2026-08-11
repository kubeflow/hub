import * as React from 'react';
import { ToolbarFilter, ToolbarLabel, ToolbarLabelGroup } from '@patternfly/react-core';

export type CatalogActiveFiltersProps<TFilterKey extends string> = {
  filterKeys: readonly TFilterKey[];
  categoryNames: Record<TFilterKey, string>;
  filters: Partial<Record<TFilterKey, string[]>>;
  setFilters: (
    updater:
      | Partial<Record<TFilterKey, string[]>>
      | ((prev: Partial<Record<TFilterKey, string[]>>) => Partial<Record<TFilterKey, string[]>>),
  ) => void;
  /** Optional per-category value → display label map (e.g. Agents framework labels). */
  labelMappings?: Partial<Record<TFilterKey, Record<string, string>>>;
  /**
   * Prefix for data-testid attributes:
   * `${testIdPrefix}-chip-${filterKey}-${value}` and `${testIdPrefix}-container-${filterKey}`.
   */
  testIdPrefix: string;
};

function CatalogActiveFilters<TFilterKey extends string>({
  filterKeys,
  categoryNames,
  filters,
  setFilters,
  labelMappings,
  testIdPrefix,
}: CatalogActiveFiltersProps<TFilterKey>): React.ReactElement {
  const handleRemoveFilter = React.useCallback(
    (categoryKey: TFilterKey, valueKey: string) => {
      setFilters((prev) => {
        const current = prev[categoryKey];
        const arr = Array.isArray(current) ? current : [];
        return { ...prev, [categoryKey]: arr.filter((v) => v !== valueKey) };
      });
    },
    [setFilters],
  );

  const handleClearCategory = React.useCallback(
    (categoryKey: TFilterKey) => {
      setFilters((prev) => ({ ...prev, [categoryKey]: [] }));
    },
    [setFilters],
  );

  return (
    <>
      {filterKeys.map((filterKey) => {
        const filterValue = filters[filterKey];
        const values = Array.isArray(filterValue) ? filterValue : [];
        const hasValue = values.length > 0;
        const labelMapping = labelMappings?.[filterKey];

        const labels: ToolbarLabel[] = hasValue
          ? values.map((value) => ({
              key: value,
              node: (
                <span data-testid={`${testIdPrefix}-chip-${filterKey}-${value}`}>
                  {labelMapping?.[value] || value}
                </span>
              ),
            }))
          : [];

        const categoryLabelGroup: ToolbarLabelGroup = {
          key: filterKey,
          name: categoryNames[filterKey],
        };

        return (
          <ToolbarFilter
            key={filterKey}
            categoryName={categoryLabelGroup}
            labels={labels}
            deleteLabel={(_, label) => {
              const labelKey = typeof label === 'string' ? label : label.key;
              handleRemoveFilter(filterKey, labelKey);
            }}
            deleteLabelGroup={() => handleClearCategory(filterKey)}
            data-testid={`${testIdPrefix}-container-${filterKey}`}
          >
            {null}
          </ToolbarFilter>
        );
      })}
    </>
  );
}

export default CatalogActiveFilters;
