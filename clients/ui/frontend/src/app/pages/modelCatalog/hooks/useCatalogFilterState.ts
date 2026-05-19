import * as React from 'react';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import {
  ModelCatalogStringFilterKey,
  ModelCatalogNumberFilterKey,
} from '~/concepts/modelCatalog/const';
import {
  ModelCatalogFilterStates,
  ModelCatalogStringFilterValueType,
} from '~/app/modelCatalogTypes';
import { useStringFilterState } from '~/app/shared/components/catalog';

export const useCatalogStringFilterState = <K extends ModelCatalogStringFilterKey>(
  filterKey: K,
): {
  selectedValues: string[];
  isSelected: (value: ModelCatalogStringFilterValueType[K]) => boolean;
  setSelected: (value: string, selected: boolean) => void;
} => {
  const { filterData, setFilterData } = React.useContext(ModelCatalogContext);
  const currentValues: string[] = filterData[filterKey];

  const onChange = React.useCallback(
    (values: string[]) => {
      // eslint-disable-next-line @typescript-eslint/consistent-type-assertions -- all string filter keys map to string[] subtypes at runtime
      setFilterData(filterKey, values as ModelCatalogFilterStates[K]);
    },
    [filterKey, setFilterData],
  );

  const shared = useStringFilterState(currentValues, onChange);

  const { isSelected: sharedIsSelected } = shared;
  const isSelected = React.useCallback(
    (value: ModelCatalogStringFilterValueType[K]) => sharedIsSelected(value),
    [sharedIsSelected],
  );

  return {
    selectedValues: shared.selectedValues,
    isSelected,
    setSelected: shared.setSelected,
  };
};

export const useCatalogNumberFilterState = (
  filterKey: ModelCatalogNumberFilterKey,
): {
  value: number | undefined;
  setValue: (value: number | undefined) => void;
} => {
  const { filterData, setFilterData } = React.useContext(ModelCatalogContext);
  const value = filterData[filterKey];
  const setValue = React.useCallback(
    (newValue: number | undefined) => {
      setFilterData(filterKey, newValue);
    },
    [filterKey, setFilterData],
  );
  return { value, setValue };
};
