import * as React from 'react';
import { DashboardEmptyTableView, Table, ManageColumnsModal } from 'mod-arch-shared';
import { Button, Spinner } from '@patternfly/react-core';
import { ColumnsIcon } from '@patternfly/react-icons';
import { InnerScrollContainer } from '@patternfly/react-table';
import { CatalogPerformanceMetricsArtifact } from '~/app/modelCatalogTypes';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import { getActiveLatencyFieldName } from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import { SortOrder } from '~/concepts/modelCatalog/const';
import {
  useHardwareConfigColumns,
  ControlledTableSortProps,
} from '~/app/pages/modelCatalog/hooks/useHardwareConfigColumns';
import HardwareConfigurationTableRow from './HardwareConfigurationTableRow';
import HardwareConfigurationFilterToolbar from './HardwareConfigurationFilterToolbar';

type HardwareConfigurationTableProps = {
  performanceArtifacts: CatalogPerformanceMetricsArtifact[];
  isLoading?: boolean;
  onSortChange?: (sort: { orderBy?: string; sortOrder?: string }) => void;
};

const HardwareConfigurationTable: React.FC<HardwareConfigurationTableProps> = ({
  performanceArtifacts,
  isLoading = false,
  onSortChange,
}) => {
  const { filters, resetPerformanceFiltersToDefaults } = React.useContext(ModelCatalogContext);

  // Note: Filtering is now done server-side via the /performance_artifacts endpoint.
  // The performanceArtifacts prop contains pre-filtered data from the server.

  // Get the active latency filter field name (if any)
  const activeLatencyField = getActiveLatencyFieldName(filters);

  // Use the custom hook that combines manage columns with the latency filter + sort logic
  const {
    columns,
    manageColumnsResult,
    sortState: {
      sortIndex,
      sortDirection,
      sortColumnField,
      onSortIndexChange,
      onSortDirectionChange,
    },
  } = useHardwareConfigColumns(activeLatencyField);

  React.useEffect(() => {
    if (sortColumnField === null) {
      onSortChange?.({});
      return;
    }

    onSortChange?.({
      orderBy: sortColumnField,
      sortOrder: sortDirection === 'asc' ? SortOrder.ASC : SortOrder.DESC,
    });
  }, [onSortChange, sortColumnField, sortDirection]);

  if (isLoading) {
    return <Spinner size="lg" />;
  }

  const handleClearFilters = () => {
    // On details page, reset performance filters to defaults (not basic filters from landing page)
    resetPerformanceFiltersToDefaults();
  };

  const manageColumnsButton = (
    <Button
      variant="link"
      icon={<ColumnsIcon />}
      onClick={manageColumnsResult.openModal}
      data-testid="manage-columns-button"
    >
      Customize columns
    </Button>
  );

  const toolbarContent = (
    <HardwareConfigurationFilterToolbar
      onResetAllFilters={handleClearFilters}
      includePerformanceFilters
      toolbarActions={manageColumnsButton}
    />
  );

  const hasActiveSort = sortColumnField !== null && sortIndex >= 0;

  // Keep callbacks wired for sort interactions, but only control index/direction when active.
  const controlledSortProps: Partial<ControlledTableSortProps> = hasActiveSort
    ? {
        sortIndex,
        sortDirection,
        onSortIndexChange,
        onSortDirectionChange,
      }
    : {
        onSortIndexChange,
        onSortDirectionChange,
      };

  return (
    <>
      {toolbarContent}
      <InnerScrollContainer>
        <Table
          data-testid="hardware-configuration-table"
          variant="compact"
          data={performanceArtifacts}
          columns={columns}
          onClearFilters={handleClearFilters}
          {...(hasActiveSort ? { defaultSortColumn: sortIndex } : {})}
          {...controlledSortProps}
          emptyTableView={<DashboardEmptyTableView onClearFilters={handleClearFilters} />}
          rowRenderer={(artifact: CatalogPerformanceMetricsArtifact) => (
            <HardwareConfigurationTableRow
              key={artifact.customProperties?.config_id?.string_value}
              performanceArtifact={artifact}
              columns={columns}
            />
          )}
        />
      </InnerScrollContainer>
      <ManageColumnsModal
        manageColumnsResult={manageColumnsResult}
        description="Manage the columns that appear in the hardware configuration table."
        dataTestId="hardware-config-manage-columns"
      />
    </>
  );
};

export default HardwareConfigurationTable;
