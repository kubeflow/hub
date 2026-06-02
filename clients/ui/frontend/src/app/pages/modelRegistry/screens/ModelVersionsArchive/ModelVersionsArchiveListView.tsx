import * as React from 'react';
import { SearchIcon } from '@patternfly/react-icons';
import { ToolbarFilter, FilterState } from 'mod-arch-shared';
import { ModelVersion } from '~/app/types';
import { filterModelVersions, getTextValue } from '~/app/pages/modelRegistry/screens/utils';
import EmptyModelRegistryState from '~/app/pages/modelRegistry/screens/components/EmptyModelRegistryState';
import {
  ModelRegistryVersionsFilterDataType,
  ModelRegistryVersionsFilterOptions,
  modelVersionsFilterConfig,
  modelVersionsVisibleFilterKeys,
  modelVersionsInitialFilterValues,
} from '~/app/pages/modelRegistry/screens/const';
import ModelVersionsArchiveTable from './ModelVersionsArchiveTable';

type ModelVersionsArchiveListViewProps = {
  modelVersions: ModelVersion[];
  refresh: () => void;
};

const ModelVersionsArchiveListView: React.FC<ModelVersionsArchiveListViewProps> = ({
  modelVersions: unfilteredmodelVersions,
  refresh,
}) => {
  const [filterValues, setFilterValues] = React.useState<
    FilterState<ModelRegistryVersionsFilterOptions>
  >(modelVersionsInitialFilterValues);

  const onFilterChange = React.useCallback(
    (key: ModelRegistryVersionsFilterOptions, value: string | string[]) =>
      setFilterValues((prev) => ({ ...prev, [key]: value })),
    [],
  );

  const onClearAllFilters = React.useCallback(
    () => setFilterValues(modelVersionsInitialFilterValues),
    [],
  );

  const filterData: ModelRegistryVersionsFilterDataType = {
    [ModelRegistryVersionsFilterOptions.keyword]: getTextValue(
      filterValues[ModelRegistryVersionsFilterOptions.keyword],
    ),
    [ModelRegistryVersionsFilterOptions.author]: getTextValue(
      filterValues[ModelRegistryVersionsFilterOptions.author],
    ),
  };

  const filteredModelVersions = filterModelVersions(unfilteredmodelVersions, filterData);

  if (unfilteredmodelVersions.length === 0) {
    return (
      <EmptyModelRegistryState
        headerIcon={SearchIcon}
        testid="empty-archive-state"
        title="No archived versions"
        description="You can archive the active versions that you no longer use. You can restore archived versions to make it active."
      />
    );
  }

  return (
    <ModelVersionsArchiveTable
      refresh={refresh}
      clearFilters={onClearAllFilters}
      modelVersions={filteredModelVersions}
      toolbarContent={
        <ToolbarFilter
          filterConfig={modelVersionsFilterConfig}
          visibleFilterKeys={modelVersionsVisibleFilterKeys}
          filterValues={filterValues}
          onFilterChange={onFilterChange}
          onClearAllFilters={onClearAllFilters}
          testIdPrefix="model-versions-archive-table"
        />
      }
    />
  );
};

export default ModelVersionsArchiveListView;
