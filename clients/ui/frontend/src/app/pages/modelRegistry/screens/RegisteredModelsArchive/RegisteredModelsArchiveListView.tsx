import * as React from 'react';
import { SearchIcon } from '@patternfly/react-icons';
import { ToolbarFilter, FilterState } from 'mod-arch-shared';
import { ModelVersion, RegisteredModel } from '~/app/types';
import { filterRegisteredModels, getTextValue } from '~/app/pages/modelRegistry/screens/utils';
import EmptyModelRegistryState from '~/app/pages/modelRegistry/screens/components/EmptyModelRegistryState';
import {
  ModelRegistryFilterDataType,
  ModelRegistryFilterOptions,
  registeredModelsFilterConfig,
  registeredModelsVisibleFilterKeys,
  registeredModelsInitialFilterValues,
} from '~/app/pages/modelRegistry/screens/const';
import RegisteredModelsArchiveTable from './RegisteredModelsArchiveTable';

type RegisteredModelsArchiveListViewProps = {
  registeredModels: RegisteredModel[];
  modelVersions: ModelVersion[];
  refresh: () => void;
};

const RegisteredModelsArchiveListView: React.FC<RegisteredModelsArchiveListViewProps> = ({
  registeredModels: unfilteredRegisteredModels,
  modelVersions,
  refresh,
}) => {
  const [filterValues, setFilterValues] = React.useState<FilterState<ModelRegistryFilterOptions>>(
    registeredModelsInitialFilterValues,
  );

  const onFilterChange = React.useCallback(
    (key: ModelRegistryFilterOptions, value: string | string[]) =>
      setFilterValues((prev) => ({ ...prev, [key]: value })),
    [],
  );

  const onClearAllFilters = React.useCallback(
    () => setFilterValues(registeredModelsInitialFilterValues),
    [],
  );

  const filterData: ModelRegistryFilterDataType = {
    [ModelRegistryFilterOptions.keyword]: getTextValue(
      filterValues[ModelRegistryFilterOptions.keyword],
    ),
    [ModelRegistryFilterOptions.owner]: getTextValue(
      filterValues[ModelRegistryFilterOptions.owner],
    ),
  };

  const filteredRegisteredModels = filterRegisteredModels(
    unfilteredRegisteredModels,
    modelVersions,
    filterData,
  );

  if (unfilteredRegisteredModels.length === 0) {
    return (
      <EmptyModelRegistryState
        headerIcon={SearchIcon}
        testid="empty-archive-model-state"
        title="No archived models"
        description="You can archive the active models that you no longer use. You can restore an archived
      model to make it active."
      />
    );
  }

  return (
    <RegisteredModelsArchiveTable
      refresh={refresh}
      clearFilters={onClearAllFilters}
      registeredModels={filteredRegisteredModels}
      modelVersions={modelVersions}
      toolbarContent={
        <ToolbarFilter
          filterConfig={registeredModelsFilterConfig}
          visibleFilterKeys={registeredModelsVisibleFilterKeys}
          filterValues={filterValues}
          onFilterChange={onFilterChange}
          onClearAllFilters={onClearAllFilters}
          testIdPrefix="registered-models-archive-table"
        />
      }
    />
  );
};

export default RegisteredModelsArchiveListView;
