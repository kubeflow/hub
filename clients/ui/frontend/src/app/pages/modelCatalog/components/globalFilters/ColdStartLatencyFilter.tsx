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
import { COLD_START_LOAD_TIME_RANGE } from '~/app/pages/modelCatalog/utils/performanceMetricsUtils';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import SliderWithInput from './SliderWithInput';

const filterKey = ModelCatalogNumberFilterKey.COLD_START_LOAD_TIME;

const ColdStartLatencyFilter: React.FC = () => {
  const { filterOptions } = React.useContext(ModelCatalogContext);
  const { value: filterValue, setValue: setFilterValue } = useCatalogNumberFilterState(filterKey);
  const [isOpen, setIsOpen] = React.useState(false);

  const { minValue, maxValue, isSliderDisabled } = React.useMemo(() => {
    const option = filterOptions?.filters?.[filterKey];
    if (option && option.range) {
      const { min, max } = option.range;
      if (min != null && max != null) {
        const roundedMin = Math.floor(min);
        const roundedMax = Math.ceil(max);
        return {
          minValue: roundedMin,
          maxValue: roundedMax,
          isSliderDisabled: roundedMin === roundedMax,
        };
      }
    }
    return COLD_START_LOAD_TIME_RANGE;
  }, [filterOptions]);

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
          <strong>Cold start load time:</strong> {filterValue} s
        </>
      );
    }
    return 'Cold start load time';
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
      data-testid="cold-start-load-time-filter"
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
      style={{ minWidth: '450px', padding: '16px' }}
    >
      <FlexItem>Cold start load time (seconds)</FlexItem>
      <FlexItem>
        <SliderWithInput
          value={clampedValue}
          min={minValue}
          max={maxValue}
          isDisabled={isSliderDisabled}
          onChange={setLocalValue}
          ariaLabel="Cold start load time value input"
          shouldRound
          showBoundaries
        />
      </FlexItem>
      <FlexItem>
        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
          <FlexItem>
            <Button
              variant="primary"
              onClick={handleApplyFilter}
              isDisabled={isSliderDisabled}
              data-testid="cold-start-load-time-apply-filter"
            >
              Apply filter
            </Button>
          </FlexItem>
          <FlexItem>
            <Button
              variant="link"
              onClick={handleReset}
              data-testid="cold-start-load-time-reset-filter"
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
