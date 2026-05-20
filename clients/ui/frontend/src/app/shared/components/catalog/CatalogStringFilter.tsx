import * as React from 'react';
import { Button, Checkbox, Content, ContentVariants, SearchInput } from '@patternfly/react-core';
import { CATALOG_STRING_FILTER_MAX_VISIBLE } from './constants';

export type CatalogStringFilterProps = {
  title: string;
  filterValues: string[];
  selectedValues: string[];
  onToggle: (value: string, checked: boolean) => void;
  getLabel?: (value: string) => string;
  testIdBase: string;
  getCheckboxTestId?: (value: string) => string;
};

const CatalogStringFilter: React.FC<CatalogStringFilterProps> = ({
  title,
  filterValues,
  selectedValues,
  onToggle,
  getLabel = (v) => v,
  testIdBase,
  getCheckboxTestId = (v) => `${testIdBase}-${v}`,
}) => {
  const [showMore, setShowMore] = React.useState(false);
  const [searchValue, setSearchValue] = React.useState('');

  const valuesMatchingSearch = React.useMemo(
    () =>
      filterValues.filter((value) => {
        const label = getLabel(value).toLowerCase();
        const q = searchValue.trim().toLowerCase();
        return (
          !q ||
          value.toLowerCase().includes(q) ||
          label.includes(q) ||
          selectedValues.includes(value)
        );
      }),
    [filterValues, getLabel, searchValue, selectedValues],
  );

  const visibleValues = showMore
    ? valuesMatchingSearch
    : valuesMatchingSearch.slice(0, CATALOG_STRING_FILTER_MAX_VISIBLE);
  const hasMoreThanMax = valuesMatchingSearch.length > CATALOG_STRING_FILTER_MAX_VISIBLE;
  const showSearch = filterValues.length > CATALOG_STRING_FILTER_MAX_VISIBLE;

  const isSelected = React.useCallback(
    (value: string) => selectedValues.includes(value),
    [selectedValues],
  );

  return (
    <Content data-testid={testIdBase}>
      <Content component={ContentVariants.h6}>{title}</Content>
      {showSearch && (
        <SearchInput
          placeholder="Search"
          value={searchValue}
          onChange={(_event, value) => setSearchValue(value)}
          data-testid={`${testIdBase}-search`}
          className="pf-v6-u-mb-sm"
        />
      )}
      {visibleValues.length === 0 && (
        <div data-testid={`${testIdBase}-empty`}>No results found</div>
      )}
      {visibleValues.map((value) => (
        <Checkbox
          key={value}
          id={`${testIdBase}-${value}`}
          label={getLabel(value)}
          isChecked={isSelected(value)}
          onChange={(_event, checked) => onToggle(value, checked)}
          data-testid={getCheckboxTestId(value)}
        />
      ))}
      {!showMore && hasMoreThanMax && (
        <Button
          variant="link"
          onClick={() => setShowMore(true)}
          data-testid={`${testIdBase}-show-more`}
        >
          Show more
        </Button>
      )}
      {showMore && hasMoreThanMax && (
        <Button
          variant="link"
          onClick={() => setShowMore(false)}
          data-testid={`${testIdBase}-show-less`}
        >
          Show less
        </Button>
      )}
    </Content>
  );
};

export default CatalogStringFilter;
