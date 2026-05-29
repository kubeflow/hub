import * as React from 'react';
import { SearchIcon } from '@patternfly/react-icons';
import { ToolbarFilter, FilterState, FilterConfigMap } from 'mod-arch-shared';
import { ModelVersion, RegisteredModel } from '~/app/types';
import { filterRegisteredModels, getTextValue } from '~/app/pages/modelRegistry/screens/utils';
import EmptyModelRegistryState from '~/app/pages/modelRegistry/screens/components/EmptyModelRegistryState';
import {
  ModelRegistryFilterDataType,
  ModelRegistryFilterOptions,
} from '~/app/pages/modelRegistry/screens/const';
import RegisteredModelsArchiveTable from './RegisteredModelsArchiveTable';

type RegisteredModelsArchiveListViewProps = {
  registeredModels: RegisteredModel[];
  modelVersions: ModelVersion[];
  refresh: () => void;
};

const filterConfig: FilterConfigMap<ModelRegistryFilterOptions> = {
  [ModelRegistryFilterOptions.keyword]: {
    type: 'text',
    label: 'Keyword',
    placeholder: 'Filter by name, description or label',
  },
  [ModelRegistryFilterOptions.owner]: {
    type: 'text',
    label: 'Owner',
    placeholder: 'Filter by owner',
  },
};

const visibleFilterKeys = [
  ModelRegistryFilterOptions.keyword,
  ModelRegistryFilterOptions.owner,
] as const;

const initialFilterValues: FilterState<ModelRegistryFilterOptions> = {
  [ModelRegistryFilterOptions.keyword]: '',
  [ModelRegistryFilterOptions.owner]: '',
};

const RegisteredModelsArchiveListView: React.FC<RegisteredModelsArchiveListViewProps> = ({
  registeredModels: unfilteredRegisteredModels,
  modelVersions,
  refresh,
}) => {
  const [filterValues, setFilterValues] =
    React.useState<FilterState<ModelRegistryFilterOptions>>(initialFilterValues);

  const onFilterChange = React.useCallback(
    (key: ModelRegistryFilterOptions, value: string | string[]) =>
      setFilterValues((prev) => ({ ...prev, [key]: value })),
    [],
  );

  const onClearAllFilters = React.useCallback(() => setFilterValues(initialFilterValues), []);

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
          filterConfig={filterConfig}
          visibleFilterKeys={visibleFilterKeys}
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
