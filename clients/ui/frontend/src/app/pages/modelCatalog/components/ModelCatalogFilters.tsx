import * as React from 'react';
import { Stack, Spinner, Alert } from '@patternfly/react-core';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import { ModelCatalogStringFilterKey } from '~/concepts/modelCatalog/const';
import { TempDevFeature, useTempDevFeatureAvailable } from '~/app/hooks/useTempDevFeatureAvailable';
import ModelPerformanceViewToggleCard from './ModelPerformanceViewToggleCard';
import TaskFilter from './globalFilters/TaskFilter';
import ProviderFilter from './globalFilters/ProviderFilter';
import LicenseFilter from './globalFilters/LicenseFilter';
import LanguageFilter from './globalFilters/LanguageFilter';
import TensorTypeFilter from './globalFilters/TensorTypeFilter';
import ValidatedConfigurationFilter from './globalFilters/ValidatedConfigurationFilter';

const ModelCatalogFilters: React.FC = () => {
  const { filterOptions, filterOptionsLoaded, filterOptionsLoadError, filterData, setFilterData } =
    React.useContext(ModelCatalogContext);
  const toolCallingFeatureAvailable = useTempDevFeatureAvailable(
    TempDevFeature.ToolCallingConfiguration,
  );

  React.useEffect(() => {
    if (
      !toolCallingFeatureAvailable &&
      filterData[ModelCatalogStringFilterKey.VALIDATED_CONFIGURATION].length > 0
    ) {
      setFilterData(ModelCatalogStringFilterKey.VALIDATED_CONFIGURATION, []);
    }
    // Only react to flag changes, not filterData changes
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [toolCallingFeatureAvailable]);

  const filters = filterOptions?.filters;
  if (!filterOptionsLoaded) {
    return <Spinner />;
  }
  if (filterOptionsLoadError) {
    return (
      <Alert variant="danger" title="Failed to load filter options" isInline>
        {filterOptionsLoadError.message}
      </Alert>
    );
  }

  const getFilterProps = (filterKey: ModelCatalogStringFilterKey) =>
    filters && filterKey in filters ? filters : undefined;

  return (
    <Stack hasGutter>
      <ModelPerformanceViewToggleCard />
      <TaskFilter filters={getFilterProps(ModelCatalogStringFilterKey.TASK)} />
      {toolCallingFeatureAvailable && (
        <ValidatedConfigurationFilter
          filters={getFilterProps(ModelCatalogStringFilterKey.VALIDATED_CONFIGURATION)}
        />
      )}
      <ProviderFilter filters={getFilterProps(ModelCatalogStringFilterKey.PROVIDER)} />
      <LicenseFilter filters={getFilterProps(ModelCatalogStringFilterKey.LICENSE)} />
      <LanguageFilter filters={getFilterProps(ModelCatalogStringFilterKey.LANGUAGE)} />
      <TensorTypeFilter filters={getFilterProps(ModelCatalogStringFilterKey.TENSOR_TYPE)} />
    </Stack>
  );
};

export default ModelCatalogFilters;
