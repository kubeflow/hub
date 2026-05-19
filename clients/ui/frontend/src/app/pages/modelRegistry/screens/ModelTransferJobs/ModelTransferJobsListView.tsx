import * as React from 'react';
import { SearchIcon } from '@patternfly/react-icons';
import { ToolbarFilter, FilterState, FilterConfigMap } from 'mod-arch-shared';
import { ModelTransferJob } from '~/app/types';
import EmptyModelRegistryState from '~/app/pages/modelRegistry/screens/components/EmptyModelRegistryState';
import ModelTransferJobsTable from './ModelTransferJobsTable';

enum ModelTransferJobsFilterOptions {
  jobName = 'jobName',
  modelName = 'modelName',
  versionName = 'versionName',
  namespace = 'namespace',
  author = 'author',
  status = 'status',
}

const modelTransferJobsFilterConfig: FilterConfigMap<ModelTransferJobsFilterOptions> = {
  [ModelTransferJobsFilterOptions.jobName]: {
    type: 'text',
    label: 'Job name',
    placeholder: 'Filter by job name',
  },
  [ModelTransferJobsFilterOptions.modelName]: {
    type: 'text',
    label: 'Model name',
    placeholder: 'Filter by model name',
  },
  [ModelTransferJobsFilterOptions.versionName]: {
    type: 'text',
    label: 'Version name',
    placeholder: 'Filter by version name',
  },
  [ModelTransferJobsFilterOptions.namespace]: {
    type: 'text',
    label: 'Namespace',
    placeholder: 'Filter by namespace',
  },
  [ModelTransferJobsFilterOptions.author]: {
    type: 'text',
    label: 'Author',
    placeholder: 'Filter by author',
  },
  [ModelTransferJobsFilterOptions.status]: {
    type: 'text',
    label: 'Status',
    placeholder: 'Filter by status',
  },
};

const visibleFilterKeys = [
  ModelTransferJobsFilterOptions.jobName,
  ModelTransferJobsFilterOptions.modelName,
  ModelTransferJobsFilterOptions.versionName,
  ModelTransferJobsFilterOptions.namespace,
  ModelTransferJobsFilterOptions.author,
  ModelTransferJobsFilterOptions.status,
] as const;

const initialFilterValues: FilterState<ModelTransferJobsFilterOptions> = {
  [ModelTransferJobsFilterOptions.jobName]: '',
  [ModelTransferJobsFilterOptions.modelName]: '',
  [ModelTransferJobsFilterOptions.versionName]: '',
  [ModelTransferJobsFilterOptions.namespace]: '',
  [ModelTransferJobsFilterOptions.author]: '',
  [ModelTransferJobsFilterOptions.status]: '',
};

type ModelTransferJobsListViewProps = {
  jobs: ModelTransferJob[];
  onRequestDelete?: (job: ModelTransferJob) => void;
  onRequestRetry?: (job: ModelTransferJob) => void;
};

const ModelTransferJobsListView: React.FC<ModelTransferJobsListViewProps> = ({
  jobs,
  onRequestDelete,
  onRequestRetry,
}) => {
  const [filterValues, setFilterValues] =
    React.useState<FilterState<ModelTransferJobsFilterOptions>>(initialFilterValues);

  const onFilterChange = React.useCallback(
    (key: ModelTransferJobsFilterOptions, value: string | string[]) =>
      setFilterValues((prev) => ({ ...prev, [key]: value })),
    [],
  );

  const onClearAllFilters = React.useCallback(() => setFilterValues(initialFilterValues), []);

  const filteredJobs = React.useMemo(() => {
    const getValue = (key: ModelTransferJobsFilterOptions): string => {
      const v = filterValues[key];
      return typeof v === 'string' ? v.toLowerCase() : '';
    };
    const jobNameFilter = getValue(ModelTransferJobsFilterOptions.jobName);
    const modelNameFilter = getValue(ModelTransferJobsFilterOptions.modelName);
    const versionNameFilter = getValue(ModelTransferJobsFilterOptions.versionName);
    const namespaceFilter = getValue(ModelTransferJobsFilterOptions.namespace);
    const authorFilter = getValue(ModelTransferJobsFilterOptions.author);
    const statusFilter = getValue(ModelTransferJobsFilterOptions.status);

    return jobs.filter((job) => {
      const jobNameForDisplay = (job.jobDisplayName || job.name || '').toLowerCase();
      if (jobNameFilter && !jobNameForDisplay.includes(jobNameFilter)) {
        return false;
      }
      if (modelNameFilter && !job.registeredModelName?.toLowerCase().includes(modelNameFilter)) {
        return false;
      }
      if (versionNameFilter && !job.modelVersionName?.toLowerCase().includes(versionNameFilter)) {
        return false;
      }
      if (namespaceFilter && !job.namespace.toLowerCase().includes(namespaceFilter)) {
        return false;
      }
      if (authorFilter && !job.author?.toLowerCase().includes(authorFilter)) {
        return false;
      }
      if (statusFilter && !job.status.toLowerCase().includes(statusFilter)) {
        return false;
      }
      return true;
    });
  }, [jobs, filterValues]);

  if (jobs.length === 0) {
    return (
      <EmptyModelRegistryState
        testid="empty-model-transfer-jobs"
        title="No model transfer jobs"
        headerIcon={SearchIcon}
        description="Model transfer jobs are created when you choose to store model artifacts during model registration."
      />
    );
  }

  return (
    <ModelTransferJobsTable
      jobs={filteredJobs}
      clearFilters={onClearAllFilters}
      toolbarContent={
        <ToolbarFilter
          filterConfig={modelTransferJobsFilterConfig}
          visibleFilterKeys={visibleFilterKeys}
          filterValues={filterValues}
          onFilterChange={onFilterChange}
          onClearAllFilters={onClearAllFilters}
          testIdPrefix="model-transfer-jobs"
        />
      }
      onRequestDelete={onRequestDelete}
      onRequestRetry={onRequestRetry}
    />
  );
};

export default ModelTransferJobsListView;
