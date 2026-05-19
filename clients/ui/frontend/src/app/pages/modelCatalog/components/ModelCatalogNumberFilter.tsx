import { Content, ContentVariants, FormGroup, NumberInput } from '@patternfly/react-core';
import * as React from 'react';
import { ModelCatalogNumberFilterKey } from '~/concepts/modelCatalog/const';
import { useCatalogNumberFilterState } from '~/app/pages/modelCatalog/hooks/useCatalogFilterState';
import { CatalogFilterNumberOption } from '~/app/modelCatalogTypes';

type ModelCatalogNumberFilterProps<K extends ModelCatalogNumberFilterKey> = {
  title: string;
  filterKey: K;
  filters: CatalogFilterNumberOption;
  placeholder?: string;
};

const ModelCatalogNumberFilter = <K extends ModelCatalogNumberFilterKey>({
  title,
  filterKey,
  filters,
  placeholder,
}: ModelCatalogNumberFilterProps<K>): JSX.Element => {
  const { value, setValue } = useCatalogNumberFilterState(filterKey);

  const handleValueChange = React.useCallback(
    (event: React.FormEvent<HTMLInputElement>) => {
      const { target } = event;
      if (target instanceof HTMLInputElement) {
        const inputValue = Number(target.value);
        setValue(inputValue || undefined);
      }
    },
    [setValue],
  );

  const handleValueChangeByButton = React.useCallback(
    (inputValue: number) => {
      setValue(inputValue || undefined);
    },
    [setValue],
  );

  const minValue = filters.range?.min;
  const maxValue = filters.range?.max;

  return (
    <Content data-testid={`${title}-filter`}>
      <Content component={ContentVariants.h6}>{title}</Content>
      <FormGroup>
        <NumberInput
          value={value || ''}
          placeholder={placeholder || `Enter ${title.toLowerCase()}`}
          data-testid={`${title}-filter-input`}
          onMinus={() =>
            handleValueChangeByButton(Math.max(minValue ?? 0, ((value || minValue) ?? 0) - 1))
          }
          onPlus={() => handleValueChangeByButton(Math.min(maxValue ?? Infinity, (value || 0) + 1))}
          onChange={handleValueChange}
          min={minValue}
          max={maxValue}
          inputName={`${filterKey}-input`}
          inputAriaLabel={`${title} filter input`}
          minusBtnAriaLabel={`Decrease ${title} value`}
          plusBtnAriaLabel={`Increase ${title} value`}
          widthChars={10}
        />
        {filters.range && minValue !== undefined && maxValue !== undefined && (
          <div className="pf-v6-u-font-size-sm pf-v6-u-color-200 pf-v6-u-mt-xs">
            Range: {minValue} - {maxValue}
          </div>
        )}
      </FormGroup>
    </Content>
  );
};

export default ModelCatalogNumberFilter;
