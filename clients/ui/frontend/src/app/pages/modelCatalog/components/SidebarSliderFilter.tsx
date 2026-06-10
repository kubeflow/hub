import * as React from 'react';
import { Button, Flex, FlexItem, Title } from '@patternfly/react-core';
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

  const range = React.useMemo(() => {
    const option = filterOptions?.filters?.[filterKey];
    if (option?.range?.min != null && option.range.max != null) {
      return { min: option.range.min, max: option.range.max };
    }
    return { min: fallbackMin, max: fallbackMax };
  }, [filterOptions, filterKey, fallbackMin, fallbackMax]);

  const isDisabled = !filterOptionsLoaded || range.min === range.max;

  const [localValue, setLocalValue] = React.useState<number>(() => filterValue ?? range.max);

  React.useEffect(() => {
    setLocalValue(filterValue ?? range.max);
  }, [filterValue, range.max]);

  const hasActiveFilter = filterValue !== undefined;

  if (!performanceViewEnabled || !filterOptionsLoaded || !filterOptions?.filters?.[filterKey]) {
    return null;
  }

  const handleApply = () => {
    setFilterValue(localValue);
  };

  const handleReset = () => {
    setFilterValue(undefined);
    setLocalValue(range.max);
  };

  return (
    <Flex
      direction={{ default: 'column' }}
      gap={{ default: 'gapSm' }}
      style={{ padding: '0 16px 16px' }}
    >
      <FlexItem>
        <Title headingLevel="h5" size="md">
          {label}
        </Title>
      </FlexItem>
      <FlexItem>
        <SliderWithInput
          value={Math.min(Math.max(localValue, range.min), range.max)}
          min={range.min}
          max={range.max}
          isDisabled={isDisabled}
          onChange={setLocalValue}
          suffix={suffix}
          ariaLabel={`${label} filter value`}
          showBoundaries
        />
      </FlexItem>
      <FlexItem>
        <Flex gap={{ default: 'gapSm' }}>
          <FlexItem>
            <Button
              variant="primary"
              size="sm"
              onClick={handleApply}
              isDisabled={isDisabled}
              data-testid={`${label.toLowerCase().replace(/\s+/g, '-')}-apply-filter`}
            >
              Apply
            </Button>
          </FlexItem>
          {hasActiveFilter && (
            <FlexItem>
              <Button
                variant="link"
                size="sm"
                onClick={handleReset}
                data-testid={`${label.toLowerCase().replace(/\s+/g, '-')}-reset-filter`}
              >
                Reset
              </Button>
            </FlexItem>
          )}
        </Flex>
      </FlexItem>
    </Flex>
  );
};

export default SidebarSliderFilter;
