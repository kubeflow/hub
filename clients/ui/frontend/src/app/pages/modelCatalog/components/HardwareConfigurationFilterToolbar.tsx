import * as React from 'react';
import {
  Button,
  Popover,
  Toolbar,
  ToolbarContent,
  ToolbarGroup,
  ToolbarItem,
} from '@patternfly/react-core';
import { HelpIcon } from '@patternfly/react-icons';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import {
  getPerformanceFiltersToShow,
  getAllFiltersToShow,
  BASIC_FILTER_KEYS,
} from '~/concepts/modelCatalog/const';
import { isValueDifferentFromDefault } from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import WorkloadTypeFilter from './globalFilters/WorkloadTypeFilter';
import HardwareConfigurationFilter from './globalFilters/HardwareConfigurationFilter';
import MaxRpsFilter from './globalFilters/MaxRpsFilter';
import ColdStartLatencyFilter from './globalFilters/ColdStartLatencyFilter';
import LatencyFilter from './globalFilters/LatencyFilter';
import ModelCatalogActiveFilters from './ModelCatalogActiveFilters';

type HardwareConfigurationFilterToolbarProps = {
  onResetAllFilters?: () => void;
  /** If true, shows basic filter chips. Defaults to false (only show on landing page when toggle is ON). */
  includeBasicFilters?: boolean;
  /** If true, shows performance filter chips. Landing page passes performanceViewEnabled, details page passes true. */
  includePerformanceFilters?: boolean;
  /** Optional content to render at the end of the toolbar (e.g., manage columns button). */
  toolbarActions?: React.ReactNode;
};

const HardwareConfigurationFilterToolbar: React.FC<HardwareConfigurationFilterToolbarProps> = ({
  onResetAllFilters,
  includeBasicFilters = false,
  includePerformanceFilters = true,
  toolbarActions,
}) => {
  const {
    filterOptions,
    filterOptionsLoaded,
    filterOptionsLoadError,
    filterData,
    getPerformanceFilterDefaultValue,
  } = React.useContext(ModelCatalogContext);

  // Get filter keys to show in the chip bar based on props
  // - includeBasicFilters: show basic filters (Task, Provider, License, Language)
  // - includePerformanceFilters: show performance filters (Workload type, Hardware type, Max RPS, Latency)
  const filtersToShow = React.useMemo(() => {
    if (includeBasicFilters && includePerformanceFilters) {
      return getAllFiltersToShow(filterData);
    }
    if (includePerformanceFilters) {
      return getPerformanceFiltersToShow(filterData);
    }
    if (includeBasicFilters) {
      return BASIC_FILTER_KEYS;
    }
    return [];
  }, [filterData, includeBasicFilters, includePerformanceFilters]);

  // Check if there are any visible filter chips (to control "Clear all filters" button visibility)
  // A chip is visible if:
  // - For filters with defaults: has a value different from the default
  // - For filters without defaults: has a non-empty value
  const hasVisibleChips = React.useMemo(
    () =>
      filtersToShow.some((filterKey) => {
        const filterValue = filterData[filterKey];

        // Skip if no value is set
        if (!filterValue) {
          return false;
        }

        // For array values (string filters), skip if empty
        if (Array.isArray(filterValue) && filterValue.length === 0) {
          return false;
        }

        // For any filter with a default value, check if current value differs from default
        const defaultValue = getPerformanceFilterDefaultValue(filterKey);
        if (defaultValue !== undefined) {
          return isValueDifferentFromDefault(filterValue, defaultValue);
        }

        // For filters without defaults, any non-empty value means visible
        return true;
      }),
    [filtersToShow, filterData, getPerformanceFilterDefaultValue],
  );

  if (!filterOptionsLoaded || filterOptionsLoadError || !filterOptions) {
    return null;
  }

  return (
    <Toolbar
      className="pf-v6-u-pb-0"
      {...(onResetAllFilters && hasVisibleChips
        ? { clearAllFilters: onResetAllFilters, clearFiltersButtonText: 'Reset all defaults' }
        : {})}
    >
      <ToolbarContent rowWrap={{ default: 'wrap' }}>
        <ToolbarGroup rowWrap={{ default: 'wrap' }}>
          <ToolbarItem>
            <WorkloadTypeFilter />
            <Popover
              bodyContent="Select a predefined scenario used to measure and compare model performance."
              footerContent="Each scenario uses fixed input and output token lengths. Scenario names approximate common use cases and do not describe model capabilities."
              appendTo={() => document.body}
            >
              <Button
                variant="plain"
                aria-label="More info for workload type"
                className="pf-v6-u-p-xs"
                icon={<HelpIcon />}
              />
            </Popover>
          </ToolbarItem>
          <ToolbarItem variant="separator" />
          <ToolbarItem>
            <LatencyFilter />
            <Popover
              bodyContent={
                <>
                  Filter models performance benchmarks by measured latency.
                  <ul style={{ listStyleType: 'disc', paddingLeft: '20px', marginTop: '8px' }}>
                    <li>
                      <strong>Metric:</strong> Select the latency metric (TTFT, E2E, or ITL) to
                      evaluate.
                    </li>
                    <li>
                      <strong>Percentile:</strong> Choose how strictly the model must meet the
                      target. For example, P90 means 90% of requests must meet the selected
                      threshold.
                    </li>
                    <li>
                      <strong>Threshold:</strong> Set the maximum latency in milliseconds. Models
                      exceeding this value are excluded.
                    </li>
                  </ul>
                </>
              }
              appendTo={() => document.body}
            >
              <Button
                variant="plain"
                aria-label="More info for latency"
                className="pf-v6-u-p-xs"
                icon={<HelpIcon />}
              />
            </Popover>
          </ToolbarItem>
          <ToolbarItem>
            <MaxRpsFilter />
            <Popover
              bodyContent="Set your target traffic load in requests per second (RPS). This value is used to calculate the optimal deployment size (number of replicas) for reliable performance. "
              appendTo={() => document.body}
              position="top"
            >
              <Button
                variant="plain"
                aria-label="More info for max RPS"
                className="pf-v6-u-p-xs"
                icon={<HelpIcon />}
              />
            </Popover>
          </ToolbarItem>
          <ToolbarItem>
            <ColdStartLatencyFilter />
            <Popover
              bodyContent="The estimated time required to provision hardware resources and initialize the container before the model can accept traffic."
              appendTo={() => document.body}
              position="top"
            >
              <Button
                variant="plain"
                aria-label="More info for cold start latency"
                className="pf-v6-u-p-xs"
                icon={<HelpIcon />}
              />
            </Popover>
          </ToolbarItem>
          <ToolbarItem variant="separator" />
          <ToolbarItem>
            <HardwareConfigurationFilter />
            <Popover
              bodyContent="Select the desired hardware configuration used for benchmarking. The format of hardware is [GPU type] x [number of GPUs per replica]. Example, A100 x 1"
              appendTo={() => document.body}
              position="top"
            >
              <Button
                variant="plain"
                aria-label="More info for latency"
                className="pf-v6-u-p-xs"
                icon={<HelpIcon />}
              />
            </Popover>
          </ToolbarItem>
          {toolbarActions && (
            <>
              <ToolbarItem variant="separator" />
              <ToolbarItem>{toolbarActions}</ToolbarItem>
            </>
          )}
        </ToolbarGroup>
        <ModelCatalogActiveFilters filtersToShow={filtersToShow} />
      </ToolbarContent>
    </Toolbar>
  );
};

export default HardwareConfigurationFilterToolbar;
