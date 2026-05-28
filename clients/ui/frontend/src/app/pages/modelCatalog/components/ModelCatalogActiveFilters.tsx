import React from 'react';
import { ToolbarFilter, ToolbarLabelGroup, ToolbarLabel } from '@patternfly/react-core';
import { isEnumMember } from 'mod-arch-core';
import { Theme } from 'mod-arch-kubeflow';
import { STYLE_THEME } from '~/app/utilities/const';
import './ModelCatalogActiveFilters.css';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import {
  ModelCatalogStringFilterKey,
  MODEL_CATALOG_PROVIDER_NAME_MAPPING,
  MODEL_CATALOG_TASK_NAME_MAPPING,
  AllLanguageCodesMap,
  MODEL_CATALOG_FILTER_CATEGORY_NAMES,
  MODEL_CATALOG_FILTER_CHIP_PREFIXES,
  ModelCatalogProvider,
  ModelCatalogTask,
  AllLanguageCode,
  ModelCatalogNumberFilterKey,
  ValidatedConfiguration,
  MODEL_CATALOG_VALIDATED_CONFIGURATION_NAME_MAPPING,
  isCatalogFilterKey,
  isPerformanceFilterKey,
  parseLatencyFilterKey,
  isLatencyFilterKey,
  LatencyFilterKey,
} from '~/concepts/modelCatalog/const';
import { ModelCatalogFilterKey } from '~/app/modelCatalogTypes';
import {
  isUseCaseOptionValue,
  getUseCaseDisplayLabel,
} from '~/app/pages/modelCatalog/utils/workloadTypeUtils';
import { isValueDifferentFromDefault } from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import { formatLatency } from '~/app/pages/modelCatalog/utils/performanceMetricsUtils';

type ModelCatalogActiveFiltersProps = {
  filtersToShow: ModelCatalogFilterKey[];
  /** When true, all ToolbarFilter labels are forced to empty arrays. This keeps the
   *  ToolbarFilter components mounted so their parent Toolbar's internal filter count
   *  stays at zero — working around PF's missing componentWillUnmount cleanup
   *  (https://github.com/patternfly/patternfly-react/issues/12247). */
  forceHideLabels?: boolean;
};

const ModelCatalogActiveFilters: React.FC<ModelCatalogActiveFiltersProps> = ({
  filtersToShow,
  forceHideLabels = false,
}) => {
  const {
    filterData,
    setFilterData,
    resetSinglePerformanceFilterToDefault,
    getPerformanceFilterDefaultValue,
  } = React.useContext(ModelCatalogContext);

  const isPatternfly = STYLE_THEME === Theme.Patternfly;

  const handleRemoveFilter = (categoryKey: string, labelKey: string) => {
    if (!isCatalogFilterKey(categoryKey)) {
      return;
    }

    if (isPerformanceFilterKey(categoryKey)) {
      resetSinglePerformanceFilterToDefault(categoryKey);
      return;
    }

    if (isEnumMember(categoryKey, ModelCatalogStringFilterKey)) {
      const currentValues = filterData[categoryKey];
      if (Array.isArray(currentValues)) {
        const newValues = currentValues.filter((v) => String(v) !== String(labelKey));
        setFilterData(categoryKey, newValues);
      }
    } else {
      setFilterData(categoryKey, undefined);
    }
  };

  const handleClearCategory = (categoryKey: string) => {
    if (!isCatalogFilterKey(categoryKey)) {
      return;
    }

    if (isPerformanceFilterKey(categoryKey)) {
      resetSinglePerformanceFilterToDefault(categoryKey);
      return;
    }

    if (isEnumMember(categoryKey, ModelCatalogStringFilterKey)) {
      setFilterData(categoryKey, []);
    } else {
      setFilterData(categoryKey, undefined);
    }
  };

  /**
   * Gets the display label for a filter value based on the filter key type
   */
  const getFilterLabel = (filterKey: ModelCatalogFilterKey, value: string | number): string => {
    if (isEnumMember(filterKey, ModelCatalogStringFilterKey)) {
      const valueStr = String(value);
      switch (filterKey) {
        case ModelCatalogStringFilterKey.PROVIDER: {
          return isEnumMember(valueStr, ModelCatalogProvider)
            ? MODEL_CATALOG_PROVIDER_NAME_MAPPING[valueStr]
            : valueStr;
        }
        case ModelCatalogStringFilterKey.LICENSE: {
          return valueStr;
        }
        case ModelCatalogStringFilterKey.TASK: {
          return isEnumMember(valueStr, ModelCatalogTask)
            ? MODEL_CATALOG_TASK_NAME_MAPPING[valueStr]
            : valueStr;
        }
        case ModelCatalogStringFilterKey.LANGUAGE: {
          return isEnumMember(valueStr, AllLanguageCode) ? AllLanguageCodesMap[valueStr] : valueStr;
        }
        case ModelCatalogStringFilterKey.VALIDATED_CONFIGURATION: {
          return isEnumMember(valueStr, ValidatedConfiguration)
            ? MODEL_CATALOG_VALIDATED_CONFIGURATION_NAME_MAPPING[valueStr]
            : valueStr;
        }
        case ModelCatalogStringFilterKey.USE_CASE: {
          if (isUseCaseOptionValue(valueStr)) {
            return `${MODEL_CATALOG_FILTER_CHIP_PREFIXES.WORKLOAD_TYPE} ${getUseCaseDisplayLabel(valueStr)}`;
          }
          return valueStr;
        }
        default:
          return valueStr;
      }
    }

    if (isEnumMember(filterKey, ModelCatalogNumberFilterKey)) {
      switch (filterKey) {
        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
        case ModelCatalogNumberFilterKey.MAX_RPS:
          return `${MODEL_CATALOG_FILTER_CHIP_PREFIXES.MAX_RPS} ${value}`;
        case ModelCatalogNumberFilterKey.COLD_START_LATENCY:
          return `${MODEL_CATALOG_FILTER_CHIP_PREFIXES.COLD_START_LATENCY} ${value} ms`;
        default:
          return String(value);
      }
    }

    const parsed = parseLatencyFilterKey(filterKey);
    const formattedValue = typeof value === 'number' ? formatLatency(value) : `${value}ms`;
    return `${parsed.metric} | ${parsed.percentile} | ${formattedValue}`;
  };

  return (
    <>
      {filtersToShow.map((filterKey) => {
        const filterValue = filterData[filterKey];

        // Determine whether this filter has visible chips.
        // TODO: PF's ToolbarFilter lacks componentWillUnmount cleanup for its internal
        // filter count (https://github.com/patternfly/patternfly-react/issues/12247).
        // Once fixed upstream, we can return null for empty filters instead of keeping
        // every ToolbarFilter mounted with labels={[]}.
        const hasValue = !!filterValue && !(Array.isArray(filterValue) && filterValue.length === 0);
        const defaultValue = getPerformanceFilterDefaultValue(filterKey);
        const isAtDefault =
          hasValue &&
          defaultValue !== undefined &&
          !isValueDifferentFromDefault(filterValue, defaultValue);
        const isVisible = hasValue && !isAtDefault;

        // Performance filter chips use data-has-default to trigger undo icon styling via CSS
        const filterHasDefault =
          isPatternfly &&
          isPerformanceFilterKey(filterKey) &&
          getPerformanceFilterDefaultValue(filterKey) !== undefined;

        // Latency: 3 separate chips in a group
        if (isLatencyFilterKey(filterKey)) {
          let latencyLabels: ToolbarLabel[] = [];

          if (isVisible) {
            const latencyFilterKey: LatencyFilterKey = filterKey;
            const parsed = parseLatencyFilterKey(latencyFilterKey);
            const formattedValue =
              typeof filterValue === 'number' ? formatLatency(filterValue) : `${filterValue}ms`;

            latencyLabels = [
              {
                key: `${filterKey}-metric`,
                node: (
                  <span data-testid={`${filterKey}-filter-chip-metric`} data-has-default="true">
                    {MODEL_CATALOG_FILTER_CHIP_PREFIXES.LATENCY_METRIC} {parsed.metric}
                  </span>
                ),
              },
              {
                key: `${filterKey}-percentile`,
                node: (
                  <span data-testid={`${filterKey}-filter-chip-percentile`} data-has-default="true">
                    {MODEL_CATALOG_FILTER_CHIP_PREFIXES.LATENCY_PERCENTILE} {parsed.percentile}
                  </span>
                ),
              },
              {
                key: `${filterKey}-threshold`,
                node: (
                  <span data-testid={`${filterKey}-filter-chip-threshold`} data-has-default="true">
                    {MODEL_CATALOG_FILTER_CHIP_PREFIXES.LATENCY_THRESHOLD} {formattedValue}
                  </span>
                ),
              },
            ];
          }

          return (
            <ToolbarFilter
              key={filterKey}
              categoryName={{
                key: filterKey,
                name: MODEL_CATALOG_FILTER_CATEGORY_NAMES[filterKey],
              }}
              labels={forceHideLabels ? [] : latencyLabels}
              deleteLabel={(category) => {
                const categoryKeyValue = typeof category === 'string' ? category : category.key;
                handleClearCategory(categoryKeyValue);
              }}
              deleteLabelGroup={(category) => {
                const categoryKeyValue = typeof category === 'string' ? category : category.key;
                handleClearCategory(categoryKeyValue);
              }}
              data-testid={`${filterKey}-filter-container`}
            >
              {null}
            </ToolbarFilter>
          );
        }

        // All other filters
        const isSingleValuePerformanceFilter =
          filterKey === ModelCatalogStringFilterKey.USE_CASE ||
          filterKey === ModelCatalogNumberFilterKey.MAX_RPS ||
          filterKey === ModelCatalogNumberFilterKey.COLD_START_LATENCY;

        let labels: ToolbarLabel[] = [];

        if (isVisible) {
          const filterValues = Array.isArray(filterValue) ? filterValue : [filterValue];

          const hasDefaultAttr = filterHasDefault
            ? isSingleValuePerformanceFilter
              ? 'single'
              : 'group'
            : undefined;

          labels = filterValues.map((value) => {
            const valueStr = String(value);
            const labelText = getFilterLabel(filterKey, value);
            return {
              key: valueStr,
              node: (
                <span
                  data-testid={`${filterKey}-filter-chip-${valueStr}`}
                  {...(hasDefaultAttr && { 'data-has-default': hasDefaultAttr })}
                >
                  {labelText}
                </span>
              ),
            };
          });
        }

        // Empty name removes the category box/border. PF's LabelGroup only applies
        // the category modifier class when categoryName is truthy.
        const categoryLabelGroup: ToolbarLabelGroup = {
          key: filterKey,
          name: isSingleValuePerformanceFilter
            ? ''
            : MODEL_CATALOG_FILTER_CATEGORY_NAMES[filterKey],
        };

        return (
          <ToolbarFilter
            key={filterKey}
            categoryName={categoryLabelGroup}
            labels={forceHideLabels ? [] : labels}
            deleteLabel={(category, label) => {
              const categoryKeyValue = typeof category === 'string' ? category : category.key;
              const labelKey = typeof label === 'string' ? label : label.key;
              handleRemoveFilter(categoryKeyValue, labelKey);
            }}
            {...(!isSingleValuePerformanceFilter && {
              deleteLabelGroup: (category: string | ToolbarLabelGroup) => {
                const categoryKeyValue = typeof category === 'string' ? category : category.key;
                handleClearCategory(categoryKeyValue);
              },
            })}
            data-testid={`${filterKey}-filter-container`}
          >
            {null}
          </ToolbarFilter>
        );
      })}
    </>
  );
};

export default ModelCatalogActiveFilters;
