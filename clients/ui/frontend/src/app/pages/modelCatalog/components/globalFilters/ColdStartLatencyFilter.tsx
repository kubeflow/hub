import * as React from 'react';
import {
  Button,
  Dropdown,
  Flex,
  FlexItem,
  MenuToggle,
  MenuToggleElement,
} from '@patternfly/react-core';
import { ModelCatalogNumberFilterKey } from '~/concepts/modelCatalog/const';
import { useCatalogNumberFilterState } from '~/app/pages/modelCatalog/hooks/useCatalogFilterState';
import { COLD_START_LATENCY_RANGE } from '~/app/pages/modelCatalog/utils/performanceMetricsUtils';
import SliderWithInput from './SliderWithInput';

const filterKey = ModelCatalogNumberFilterKey.COLD_START_LATENCY;

const ColdStartLatencyFilter: React.FC = () => {
  const { value: filterValue, setValue: setFilterValue } = useCatalogNumberFilterState(filterKey);
  const [isOpen, setIsOpen] = React.useState(false);

  const { minValue, maxValue, isSliderDisabled } = COLD_START_LATENCY_RANGE;

  const [localValue, setLocalValue] = React.useState<number>(() => filterValue ?? maxValue);

  const clampedValue = React.useMemo(
    () => Math.min(Math.max(localValue, minValue), maxValue),
    [localValue, minValue, maxValue],
  );

  React.useEffect(() => {
    if (isOpen) {
      setLocalValue(filterValue ?? maxValue);
    }
  }, [isOpen, filterValue, maxValue]);

  const hasActiveFilter = filterValue !== undefined;

  const getDisplayText = (): React.ReactNode => {
    if (hasActiveFilter) {
      return (
        <>
          <strong>Cold start latency:</strong> {filterValue} ms
        </>
      );
    }
    return 'Cold start latency';
  };

  const handleApplyFilter = () => {
    setFilterValue(localValue);
    setIsOpen(false);
  };

  const handleReset = () => {
    setLocalValue(filterValue ?? maxValue);
  };

  const toggle = (toggleRef: React.Ref<MenuToggleElement>) => (
    <MenuToggle
      ref={toggleRef}
      data-testid="cold-start-latency-filter"
      onClick={() => setIsOpen(!isOpen)}
      isExpanded={isOpen}
      isFullHeight
      style={{ minWidth: '200px', width: 'fit-content', height: '56px' }}
    >
      {getDisplayText()}
    </MenuToggle>
  );

  const filterContent = (
    <Flex
      direction={{ default: 'column' }}
      spaceItems={{ default: 'spaceItemsSm' }}
      flexWrap={{ default: 'wrap' }}
      style={{ minWidth: '400px', padding: '16px' }}
    >
      <FlexItem>Cold start latency (ms)</FlexItem>
      <FlexItem>
        <SliderWithInput
          value={clampedValue}
          min={minValue}
          max={maxValue}
          isDisabled={isSliderDisabled}
          onChange={setLocalValue}
          ariaLabel="Cold start latency value input"
        />
      </FlexItem>
      <FlexItem>
        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
          <FlexItem>
            <Button
              variant="primary"
              onClick={handleApplyFilter}
              isDisabled={isSliderDisabled}
              data-testid="cold-start-latency-apply-filter"
            >
              Apply filter
            </Button>
          </FlexItem>
          <FlexItem>
            <Button
              variant="link"
              onClick={handleReset}
              data-testid="cold-start-latency-reset-filter"
            >
              Reset
            </Button>
          </FlexItem>
        </Flex>
      </FlexItem>
    </Flex>
  );

  return (
    <Dropdown
      isOpen={isOpen}
      onOpenChange={setIsOpen}
      toggle={toggle}
      shouldFocusToggleOnSelect={false}
    >
      {filterContent}
    </Dropdown>
  );
};

export default ColdStartLatencyFilter;
