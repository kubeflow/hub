import * as React from 'react';
import { Button } from '@patternfly/react-core';
import { useNavigate } from 'react-router-dom';
import {
  ProjectObjectType,
  typedEmptyImage,
  ToolbarFilter,
  FilterState,
  FilterConfigMap,
} from 'mod-arch-shared';
import { ModelVersion, RegisteredModel } from '~/app/types';
import { ModelRegistrySelectorContext } from '~/app/context/ModelRegistrySelectorContext';
import {
  modelTransferJobsUrl,
  registeredModelArchiveUrl,
  registerModelUrl,
} from '~/app/pages/modelRegistry/screens/routeUtils';
import EmptyModelRegistryState from '~/app/pages/modelRegistry/screens/components/EmptyModelRegistryState';
import { filterRegisteredModels } from '~/app/pages/modelRegistry/screens/utils';
import { filterArchiveModels, filterLiveModels } from '~/app/utils';
import {
  ModelRegistryFilterDataType,
  ModelRegistryFilterOptions,
} from '~/app/pages/modelRegistry/screens/const';
import RegisteredModelTable from './RegisteredModelTable';
import RegisteredModelsToolbarActions from './RegisteredModelsToolbarActions';

type RegisteredModelListViewProps = {
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

const RegisteredModelListView: React.FC<RegisteredModelListViewProps> = ({
  registeredModels,
  modelVersions,
  refresh,
}) => {
  const navigate = useNavigate();
  const { preferredModelRegistry } = React.useContext(ModelRegistrySelectorContext);
  const [filterValues, setFilterValues] =
    React.useState<FilterState<ModelRegistryFilterOptions>>(initialFilterValues);
  const unfilteredRegisteredModels = filterLiveModels(registeredModels);
  const archiveRegisteredModels = filterArchiveModels(registeredModels);

  const onFilterChange = React.useCallback(
    (key: ModelRegistryFilterOptions, value: string | string[]) =>
      setFilterValues((prev) => ({ ...prev, [key]: value })),
    [],
  );

  const onClearAllFilters = React.useCallback(() => setFilterValues(initialFilterValues), []);

  if (unfilteredRegisteredModels.length === 0) {
    return (
      <EmptyModelRegistryState
        testid="empty-registered-models"
        title="No models in selected registry"
        headerIcon={() => (
          <img
            src={typedEmptyImage(ProjectObjectType.registeredModels, 'MissingModel')}
            alt="missing model"
          />
        )}
        description={`${
          preferredModelRegistry?.name ?? ''
        } has no active registered models. Register a model in this registry, or select a different registry.`}
        primaryActionText="Register model"
        secondaryActionText={
          archiveRegisteredModels.length !== 0 ? 'View archived models' : undefined
        }
        primaryActionOnClick={() => {
          navigate(registerModelUrl(preferredModelRegistry?.name));
        }}
        secondaryActionOnClick={() => {
          navigate(registeredModelArchiveUrl(preferredModelRegistry?.name));
        }}
        customAction={
          <Button
            data-testid="empty-model-registry-transfer-jobs-action"
            variant="link"
            onClick={() => navigate(modelTransferJobsUrl(preferredModelRegistry?.name))}
          >
            View model transfer jobs
          </Button>
        }
      />
    );
  }

  const getTextValue = (v: string | string[]): string | undefined => {
    const s = typeof v === 'string' ? v : v[0];
    return s || undefined;
  };

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

  return (
    <RegisteredModelTable
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
          toolbarActions={<RegisteredModelsToolbarActions />}
          testIdPrefix="registered-models-table"
        />
      }
    />
  );
};

export default RegisteredModelListView;
