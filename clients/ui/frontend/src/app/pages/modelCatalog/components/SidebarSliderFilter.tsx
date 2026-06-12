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
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import SliderWithInput from './globalFilters/SliderWithInput';

type SidebarSliderFilterProps = {
  filterKey: ModelCatalogNumberFilterKey;
  label: string;
  suffix?: string;
  fallbackMin: number;
  fallbackMax: number;
};

const SidebarSliderFilter: React.FC<SidebarSliderFilterProps> = ({
  filterKey,
  label,
  suffix = 'GB',
  fallbackMin,
  fallbackMax,
}) => {
  const { filterOptions, filterOptionsLoaded, performanceViewEnabled } =
    React.useContext(ModelCatalogContext);
  const { value: filterValue, setValue: setFilterValue } = useCatalogNumberFilterState(filterKey);
  const [isOpen, setIsOpen] = React.useState(false);

  const range = React.useMemo(() => {
    const option = filterOptions?.filters?.[filterKey];
    if (option && option.range) {
      const { min, max } = option.range;
      if (min != null && max != null) {
        return { min: Math.floor(min), max: Math.ceil(max) };
      }
    }
    return { min: fallbackMin, max: fallbackMax };
  }, [filterOptions, filterKey, fallbackMin, fallbackMax]);

  const isDisabled = !filterOptionsLoaded || range.min === range.max;

  const [localValue, setLocalValue] = React.useState<number>(() => filterValue ?? range.max);

  React.useEffect(() => {
    if (isOpen) {
      setLocalValue(filterValue ?? range.max);
    }
  }, [isOpen, filterValue, range.max]);

  if (!performanceViewEnabled || !filterOptionsLoaded || !filterOptions?.filters?.[filterKey]) {
    return null;
  }

  const hasActiveFilter = filterValue !== undefined;

  const getDisplayText = (): React.ReactNode => {
    if (hasActiveFilter) {
      return (
        <>
          <strong>{label}:</strong> ≤ {filterValue} {suffix}
        </>
      );
    }
    return label;
  };

  const handleApply = () => {
    setFilterValue(localValue);
    setIsOpen(false);
  };

  const handleReset = () => {
    setLocalValue(filterValue ?? range.max);
  };

  const clampedValue = Math.min(Math.max(localValue, range.min), range.max);

  const toggle = (toggleRef: React.Ref<MenuToggleElement>) => (
    <MenuToggle
      ref={toggleRef}
      data-testid={`${label.toLowerCase().replace(/\s+/g, '-')}-filter`}
      onClick={() => setIsOpen(!isOpen)}
      isExpanded={isOpen}
      isFullWidth
    >
      {getDisplayText()}
    </MenuToggle>
  );

  return (
    <Dropdown
      isOpen={isOpen}
      onOpenChange={setIsOpen}
      toggle={toggle}
      shouldFocusToggleOnSelect={false}
    >
      <Flex
        direction={{ default: 'column' }}
        spaceItems={{ default: 'spaceItemsSm' }}
        style={{ minWidth: '400px', padding: '16px' }}
      >
        <FlexItem>{label}</FlexItem>
        <FlexItem>
          <SliderWithInput
            value={clampedValue}
            min={range.min}
            max={range.max}
            isDisabled={isDisabled}
            onChange={setLocalValue}
            suffix={suffix}
            ariaLabel={`${label} filter value`}
            showBoundaries
            shouldRound
          />
        </FlexItem>
        <FlexItem>
          <Flex spaceItems={{ default: 'spaceItemsSm' }}>
            <FlexItem>
              <Button
                variant="primary"
                onClick={handleApply}
                isDisabled={isDisabled}
                data-testid={`${label.toLowerCase().replace(/\s+/g, '-')}-apply-filter`}
              >
                Apply
              </Button>
            </FlexItem>
            <FlexItem>
              <Button
                variant="link"
                onClick={handleReset}
                data-testid={`${label.toLowerCase().replace(/\s+/g, '-')}-reset-filter`}
              >
                Reset
              </Button>
            </FlexItem>
          </Flex>
        </FlexItem>
      </Flex>
    </Dropdown>
  );
};

export default SidebarSliderFilter;
