import * as React from 'react';
import { useManageColumns, UseManageColumnsResult } from 'mod-arch-shared';
import { CatalogPerformanceMetricsArtifact } from '~/app/modelCatalogTypes';
import {
  LatencyMetric,
  LatencyMetricFieldName,
  getLatencyPropertyKey,
  parseLatencyFilterKey,
} from '~/concepts/modelCatalog/const';
import {
  hardwareConfigColumns,
  HardwareConfigColumn,
  STICKY_COLUMN_FIELDS,
  DEFAULT_VISIBLE_COLUMN_FIELDS,
  LATENCY_COLUMN_FIELDS,
  TPS_COLUMN_FIELDS,
  HARDWARE_CONFIG_COLUMNS_STORAGE_KEY,
} from '~/app/pages/modelCatalog/components/HardwareConfigurationTableColumns';

/** Controlled sort props for the Table component */
export type ControlledTableSortProps = {
  sortIndex: number;
  sortDirection: 'asc' | 'desc';
  onSortIndexChange: (index: number) => void;
  onSortDirectionChange: (direction: 'asc' | 'desc') => void;
};

type UseHardwareConfigColumnsResult = {
  /** Final columns to render in the table (sticky + visible managed columns) */
  columns: HardwareConfigColumn[];
  /** Result from useManageColumns hook, to be passed directly to ManageColumnsModal */
  manageColumnsResult: UseManageColumnsResult<CatalogPerformanceMetricsArtifact>;
  /**
   * Lifted sort state.
   * Simplified by reusing the interface we'll use for the Table assertion.
   */
  sortState: ControlledTableSortProps & { sortColumnField: string | null };
};

/**
 * Check if a column field is a latency column (TTFT, E2E, ITL - not TPS)
 */
const isLatencyColumnField = (field: string): boolean =>
  LATENCY_COLUMN_FIELDS.some((f) => f === field);

/**
 * Check if a column field is a TPS column
 */
const isTpsColumnField = (field: string): boolean => TPS_COLUMN_FIELDS.some((f) => f === field);

/**
 * Custom hook that combines useManageColumns with the latency filter effect logic.
 *
 * When the latency filter changes:
 * - The filtered latency column becomes selected
 * - Other latency columns become deselected
 * - The corresponding TPS column becomes selected
 * - Other TPS columns become deselected
 *
 * When the filter is cleared, the current state is preserved.
 */
export const useHardwareConfigColumns = (
  activeLatencyField: LatencyMetricFieldName | undefined,
): UseHardwareConfigColumnsResult => {
  // Track the previous latency filter to detect changes
  const prevLatencyFieldRef = React.useRef<LatencyMetricFieldName | undefined>(undefined);

  // sort state
  const [sortColumnField, setSortColumnField] = React.useState<string | null>(null);
  const [sortDirection, setSortDirection] = React.useState<'asc' | 'desc'>('asc');

  // Separate sticky columns (always visible) from manageable columns
  const { stickyColumns, manageableColumns } = React.useMemo(() => {
    const sticky = hardwareConfigColumns.filter((col) => STICKY_COLUMN_FIELDS.includes(col.field));
    const manageable = hardwareConfigColumns.filter(
      (col) => !STICKY_COLUMN_FIELDS.includes(col.field),
    );
    return { stickyColumns: sticky, manageableColumns: manageable };
  }, []);

  // Use the manage columns hook for manageable columns only
  const manageColumnsResult = useManageColumns<
    CatalogPerformanceMetricsArtifact,
    HardwareConfigColumn
  >({
    allColumns: manageableColumns,
    storageKey: HARDWARE_CONFIG_COLUMNS_STORAGE_KEY,
    defaultVisibleColumnIds: DEFAULT_VISIBLE_COLUMN_FIELDS,
  });

  // Combine sticky + visible managed columns
  const columns = React.useMemo(
    (): HardwareConfigColumn[] => [...stickyColumns, ...manageColumnsResult.visibleColumns],
    [stickyColumns, manageColumnsResult.visibleColumns],
  );

  // Combined effect to update visible columns AND sort when latency filter changes
  React.useEffect(() => {
    // Only react to changes, not initial mount
    if (prevLatencyFieldRef.current === activeLatencyField) {
      return;
    }
    prevLatencyFieldRef.current = activeLatencyField;

    // If filter is cleared, keep current state
    if (!activeLatencyField) {
      return;
    }

    // Parse the active filter to get metric and percentile
    const parsed = parseLatencyFilterKey(activeLatencyField);
    const activePropertyKey = parsed.propertyKey;
    const matchingTpsPropertyKey = getLatencyPropertyKey(LatencyMetric.TPS, parsed.percentile);

    // Build new visible column IDs:
    // - Keep all non-latency/non-TPS columns
    // - Remove all latency and TPS columns
    // - Add the active latency column and matching TPS column
    const newVisibleIds = manageColumnsResult.visibleColumnIds.filter(
      (id) => !isLatencyColumnField(id) && !isTpsColumnField(id),
    );

    // Use a Set to ensure uniqueness without manual .includes() checks
    const updatedIds = Array.from(
      new Set([...newVisibleIds, activePropertyKey, matchingTpsPropertyKey]),
    );

    manageColumnsResult.setVisibleColumnIds(updatedIds);

    setSortColumnField(activePropertyKey);
    setSortDirection('asc');
  }, [activeLatencyField, manageColumnsResult, stickyColumns, manageableColumns]);

  const sortState = React.useMemo(() => {
    const sortIndex =
      sortColumnField !== null ? columns.findIndex((col) => col.field === sortColumnField) : -1;

    // Translate sortIndex back to sortColumnField when Table calls the callback
    const onSortIndexChange = (index: number) => {
      if (index >= 0 && index < columns.length) {
        setSortColumnField(columns[index].field);
      } else {
        setSortColumnField(null);
      }
    };

    return {
      sortIndex,
      sortDirection,
      sortColumnField,
      onSortIndexChange,
      onSortDirectionChange: setSortDirection,
    };
  }, [sortColumnField, sortDirection, columns]);

  return {
    columns,
    manageColumnsResult,
    sortState,
  };
};
