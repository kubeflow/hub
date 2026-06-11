import * as React from 'react';
import { Content, ContentVariants, Flex } from '@patternfly/react-core';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import {
  ModelCatalogNumberFilterKey,
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
  type StringFilterPanelItem,
} from '~/app/shared/components/catalog';
import ModelPerformanceViewToggleCard from './ModelPerformanceViewToggleCard';
import SidebarSliderFilter from './SidebarSliderFilter';

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
  const {
    filterOptions,
    filterOptionsLoaded,
    filterOptionsLoadError,
    filters,
    setFilters,
    performanceViewEnabled,
  } = React.useContext(ModelCatalogContext);
  const onFilterChange = React.useCallback(
    (key: string, values: string[]) => {
      const match = BASIC_STRING_FILTER_KEYS.find((k) => k === key);
      if (match) {
        setFilters((prev) => ({ ...prev, [match]: values }));
      }
    },
    [setFilters],
  );

  const selectedStringFilters = React.useMemo(() => {
    const result: Record<string, string[] | undefined> = {};
    for (const key of BASIC_STRING_FILTER_KEYS) {
      result[key] = filters[key];
    }
    return result;
  }, [filters]);

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
    const items: FilterPanelItem[] = baseFilterItems.map((item) => {
      const itemWithTestIds: StringFilterPanelItem = {
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

    // Insert slider filters as independent panel items after "Validated arguments"
    const validatedIndex = items.findIndex((item) => item.key === validatedConfigKey);
    const insertIndex = validatedIndex >= 0 ? validatedIndex + 1 : items.length;

    const sliderItems: FilterPanelItem[] = [
      {
        key: 'hardware-slider-filters',
        title: 'Hardware filters',
        visible: performanceViewEnabled,
        customContent: (
          <Flex direction={{ default: 'column' }} gap={{ default: 'gapSm' }}>
            <SidebarSliderFilter
              filterKey={ModelCatalogNumberFilterKey.MIN_VRAM}
              label="Minimum vRAM"
              suffix="GB"
              fallbackMin={4}
              fallbackMax={480}
            />
            <SidebarSliderFilter
              filterKey={ModelCatalogNumberFilterKey.IMAGE_SIZE}
              label="Container size"
              suffix="GB"
              fallbackMin={4}
              fallbackMax={500}
            />
          </Flex>
        ),
      },
    ];

    items.splice(insertIndex, 0, ...sliderItems);
    return items;
  }, [baseFilterItems, performanceViewEnabled]);

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
