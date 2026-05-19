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
import { MAX_RPS_RANGE } from '~/app/pages/modelCatalog/utils/performanceMetricsUtils';
import SliderWithInput from './SliderWithInput';

const filterKey = ModelCatalogNumberFilterKey.MAX_RPS;

const MaxRpsFilter: React.FC = () => {
  const { value: rpsFilterValue, setValue: setRpsFilterValue } =
    useCatalogNumberFilterState(filterKey);
  const [isOpen, setIsOpen] = React.useState(false);

  const { minValue, maxValue, isSliderDisabled } = MAX_RPS_RANGE;

  const [localValue, setLocalValue] = React.useState<number>(() => rpsFilterValue ?? maxValue);

  const clampedValue = React.useMemo(
    () => Math.min(Math.max(localValue, minValue), maxValue),
    [localValue, minValue, maxValue],
  );

  React.useEffect(() => {
    if (isOpen) {
      setLocalValue(rpsFilterValue ?? maxValue);
    }
  }, [isOpen, rpsFilterValue, maxValue]);

  const hasActiveFilter = rpsFilterValue !== undefined;

  const getDisplayText = (): React.ReactNode => {
    if (hasActiveFilter) {
      return (
        <>
          <strong>Max RPS:</strong> {rpsFilterValue}
        </>
      );
    }
    return 'Max RPS';
  };

  const handleApplyFilter = () => {
    setRpsFilterValue(localValue);
    setIsOpen(false);
  };

  const handleReset = () => {
    setLocalValue(rpsFilterValue ?? maxValue);
  };

  const toggle = (toggleRef: React.Ref<MenuToggleElement>) => (
    <MenuToggle
      ref={toggleRef}
      data-testid="max-rps-filter"
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
      <FlexItem>Max requests per second (RPS)</FlexItem>
      <FlexItem>
        <SliderWithInput
          value={clampedValue}
          min={minValue}
          max={maxValue}
          isDisabled={isSliderDisabled}
          onChange={setLocalValue}
          ariaLabel="RPS value input"
        />
      </FlexItem>
      <FlexItem>
        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
          <FlexItem>
            <Button
              variant="primary"
              onClick={handleApplyFilter}
              isDisabled={isSliderDisabled}
              data-testid="max-rps-apply-filter"
            >
              Apply filter
            </Button>
          </FlexItem>
          <FlexItem>
            <Button variant="link" onClick={handleReset} data-testid="max-rps-reset-filter">
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

export default MaxRpsFilter;
