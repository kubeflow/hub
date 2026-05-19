import * as React from 'react';
import { ModelCatalogStringFilterKey } from '~/concepts/modelCatalog/const';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';

const filterKey = ModelCatalogStringFilterKey.HARDWARE_CONFIGURATION;

export const useHardwareConfigurationFilterState = (): {
  appliedValues: string[];
  setAppliedValues: (values: string[]) => void;
  clearFilters: () => void;
} => {
  const { filterData, setFilterData } = React.useContext(ModelCatalogContext);
  const appliedValues: string[] = filterData[filterKey];

  const setAppliedValues = React.useCallback(
    (values: string[]) => {
      setFilterData(filterKey, values);
    },
    [setFilterData],
  );

  const clearFilters = React.useCallback(() => {
    setFilterData(filterKey, []);
  }, [setFilterData]);

  return { appliedValues, setAppliedValues, clearFilters };
};
