import * as React from 'react';
import { Content, ContentVariants } from '@patternfly/react-core';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import {
  ModelCatalogStringFilterKey,
  MODEL_CATALOG_FILTER_CATEGORY_NAMES,
  MODEL_CATALOG_TASK_NAME_MAPPING,
  MODEL_CATALOG_PROVIDER_NAME_MAPPING,
  MODEL_CATALOG_VALIDATED_CONFIGURATION_NAME_MAPPING,
  MODEL_CATALOG_ASIAN_LANGUAGES_DETAILS,
  MODEL_CATALOG_EUROPEAN_LANGUAGES_DETAILS,
  MODEL_CATALOG_MIDDLE_EASTERN_AND_OTHER_LANGUAGES_DETAILS,
  ModelCatalogTensorType,
} from '~/concepts/modelCatalog/const';
import {
  CatalogFilterPanel,
  useCatalogFilterConfigs,
  type FilterPanelItem,
} from '~/app/shared/components/catalog';
import ModelPerformanceViewToggleCard from './ModelPerformanceViewToggleCard';

const BASIC_STRING_FILTER_KEYS: ModelCatalogStringFilterKey[] = [
  ModelCatalogStringFilterKey.TASK,
  ModelCatalogStringFilterKey.VALIDATED_CONFIGURATION,
  ModelCatalogStringFilterKey.PROVIDER,
  ModelCatalogStringFilterKey.LICENSE,
  ModelCatalogStringFilterKey.LANGUAGE,
  ModelCatalogStringFilterKey.TENSOR_TYPE,
];

const LANGUAGE_NAME_MAPPING: Record<string, string> = {
  ...MODEL_CATALOG_EUROPEAN_LANGUAGES_DETAILS,
  ...MODEL_CATALOG_ASIAN_LANGUAGES_DETAILS,
  ...MODEL_CATALOG_MIDDLE_EASTERN_AND_OTHER_LANGUAGES_DETAILS,
};

const LABEL_MAPPINGS: Record<string, Record<string, string>> = {
  [ModelCatalogStringFilterKey.TASK]: MODEL_CATALOG_TASK_NAME_MAPPING,
  [ModelCatalogStringFilterKey.PROVIDER]: MODEL_CATALOG_PROVIDER_NAME_MAPPING,
  [ModelCatalogStringFilterKey.VALIDATED_CONFIGURATION]:
    MODEL_CATALOG_VALIDATED_CONFIGURATION_NAME_MAPPING,
  [ModelCatalogStringFilterKey.LANGUAGE]: LANGUAGE_NAME_MAPPING,
  [ModelCatalogStringFilterKey.TENSOR_TYPE]: ModelCatalogTensorType,
};

const ModelCatalogFilters: React.FC = () => {
  const { filterOptions, filterOptionsLoaded, filterOptionsLoadError, filterData, setFilterData } =
    React.useContext(ModelCatalogContext);
  const onFilterChange = React.useCallback(
    (key: string, values: string[]) => {
      const match = BASIC_STRING_FILTER_KEYS.find((k) => k === key);
      if (match) {
        setFilterData(match, values);
      }
    },
    [setFilterData],
  );

  const selectedStringFilters = React.useMemo(() => {
    const result: Record<string, string[] | undefined> = {};
    for (const key of BASIC_STRING_FILTER_KEYS) {
      result[key] = filterData[key];
    }
    return result;
  }, [filterData]);

  const baseFilterItems = useCatalogFilterConfigs({
    filterKeys: BASIC_STRING_FILTER_KEYS,
    filterNames: MODEL_CATALOG_FILTER_CATEGORY_NAMES,
    filterOptions: filterOptions?.filters,
    selectedFilters: selectedStringFilters,
    onFilterChange,
    labelMappings: LABEL_MAPPINGS,
  });

  const filterPanelItems = React.useMemo((): FilterPanelItem[] => {
    const validatedConfigKey = ModelCatalogStringFilterKey.VALIDATED_CONFIGURATION;
    return baseFilterItems.map((item) => {
      const itemWithTestIds: FilterPanelItem = {
        ...item,
        testIdBase: `${item.title}-filter`,
        getCheckboxTestId: (value: string) => `${item.title}-${value}-checkbox`,
      };
      if (item.key === validatedConfigKey) {
        const hasMultiple = item.filterValues.length > 1;
        const hasSelection = item.selectedValues.length > 0;
        return {
          ...itemWithTestIds,
          footer:
            hasMultiple && hasSelection ? (
              <Content component={ContentVariants.small} className="pf-v6-u-mt-sm">
                Showing models with all selected configurations
              </Content>
            ) : undefined,
        };
      }
      return itemWithTestIds;
    });
  }, [baseFilterItems]);

  return (
    <CatalogFilterPanel
      loaded={filterOptionsLoaded}
      loadError={filterOptionsLoadError}
      filters={filterPanelItems}
      extraContent={<ModelPerformanceViewToggleCard />}
      testIdPrefix="model-catalog-filter"
    />
  );
};

export default ModelCatalogFilters;
