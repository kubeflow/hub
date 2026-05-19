import * as React from 'react';
import CatalogStringFilter from '~/app/shared/components/catalog/CatalogStringFilter';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import {
  ModelCatalogStringFilterOptions,
  ModelCatalogStringFilterValueType,
} from '~/app/modelCatalogTypes';
import { ModelCatalogStringFilterKey } from '~/concepts/modelCatalog/const';
import { useCatalogStringFilterState } from '~/app/pages/modelCatalog/hooks/useCatalogFilterState';

function isFilterMappingKey<K extends string>(obj: Partial<Record<K, string>>, s: string): s is K {
  return Object.hasOwn(obj, s);
}

type ModelCatalogStringFilterProps<K extends ModelCatalogStringFilterKey> = {
  title: string;
  filterKey: K;
  filterToNameMapping?: Partial<Record<ModelCatalogStringFilterValueType[K], string>>;
  filters: ModelCatalogStringFilterOptions[K];
};

const ModelCatalogStringFilter = <K extends ModelCatalogStringFilterKey>({
  title,
  filterKey,
  filterToNameMapping = {},
  filters,
}: ModelCatalogStringFilterProps<K>): JSX.Element => {
  const { filterData } = React.useContext(ModelCatalogContext);
  const { setSelected } = useCatalogStringFilterState(filterKey);
  const selectedValues = filterData[filterKey];

  const getLabel = React.useCallback(
    (value: string): string =>
      isFilterMappingKey(filterToNameMapping, value)
        ? (filterToNameMapping[value] ?? value)
        : value,
    [filterToNameMapping],
  );

  const filterValues = React.useMemo(() => filters?.values ?? [], [filters?.values]);

  return (
    <CatalogStringFilter
      title={title}
      filterValues={filterValues}
      selectedValues={selectedValues}
      onToggle={setSelected}
      getLabel={getLabel}
      testIdBase={`${title}-filter`}
      getCheckboxTestId={(value) => `${title}-${value}-checkbox`}
    />
  );
};

export default ModelCatalogStringFilter;
