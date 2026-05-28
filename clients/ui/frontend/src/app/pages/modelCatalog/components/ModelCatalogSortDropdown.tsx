import * as React from 'react';
import {
  Flex,
  MenuToggle,
  MenuToggleElement,
  Select,
  SelectList,
  SelectOption,
} from '@patternfly/react-core';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import { ModelCatalogSortOption } from '~/concepts/modelCatalog/const';
import { getActiveLatencyFieldName } from '~/app/pages/modelCatalog/utils/modelCatalogUtils';

type ModelCatalogSortDropdownProps = {
  performanceViewEnabled: boolean;
};

const ModelCatalogSortDropdown: React.FC<ModelCatalogSortDropdownProps> = ({
  performanceViewEnabled,
}) => {
  const { sortBy, setSortBy, filterData } = React.useContext(ModelCatalogContext);
  const [isOpen, setIsOpen] = React.useState(false);

  const activeLatencyField = getActiveLatencyFieldName(filterData);
  // Disable latency sort if performance view is disabled or there's no active latency field
  // Without an active latency field, sorting by latency would fallback to sorting by publish date
  const isLatencySortDisabled = !performanceViewEnabled || activeLatencyField === undefined;

  // Hide dropdown when performance view is disabled
  if (!performanceViewEnabled) {
    return null;
  }

  const isValidSortOption = (value: unknown): value is ModelCatalogSortOption =>
    typeof value === 'string' &&
    (value === ModelCatalogSortOption.RECENT_PUBLISH ||
      value === ModelCatalogSortOption.LOWEST_LATENCY ||
      value === ModelCatalogSortOption.LOWEST_COLD_START);

  const handleSelect = (
    _event: React.SyntheticEvent<Element, Event> | undefined,
    value: string | number | undefined,
  ) => {
    if (isValidSortOption(value)) {
      setSortBy(value);
    }
    setIsOpen(false);
  };

  const getDisplayValue = (): string => {
    if (sortBy === ModelCatalogSortOption.LOWEST_LATENCY) {
      return 'Latency (Lowest → Highest)';
    }
    if (sortBy === ModelCatalogSortOption.LOWEST_COLD_START) {
      return 'Cold start latency (Lowest → Highest)';
    }
    return 'Publish date (Newest → Oldest)';
  };

  const currentSort = sortBy || ModelCatalogSortOption.RECENT_PUBLISH;

  return (
    <Flex gap={{ default: 'gapSm' }} alignItems={{ default: 'alignItemsCenter' }}>
      <span>Sort:</span>
      <Select
        isOpen={isOpen}
        onOpenChange={setIsOpen}
        selected={currentSort}
        onSelect={handleSelect}
        toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
          <MenuToggle
            ref={toggleRef}
            onClick={() => setIsOpen(!isOpen)}
            isExpanded={isOpen}
            aria-label="Sort options"
            data-testid="model-catalog-sort-dropdown"
          >
            {getDisplayValue()}
          </MenuToggle>
        )}
      >
        <SelectList>
          <SelectOption
            value={ModelCatalogSortOption.RECENT_PUBLISH}
            data-testid="sort-option-recent-publish"
          >
            Publish date (Newest → Oldest)
          </SelectOption>
          <SelectOption
            value={ModelCatalogSortOption.LOWEST_LATENCY}
            isDisabled={isLatencySortDisabled}
            data-testid="sort-option-lowest-latency"
          >
            Latency (Lowest → Highest)
          </SelectOption>
          <SelectOption
            value={ModelCatalogSortOption.LOWEST_COLD_START}
            data-testid="sort-option-lowest-cold-start"
          >
            Cold start latency (Lowest → Highest)
          </SelectOption>
        </SelectList>
      </Select>
    </Flex>
  );
};

export default ModelCatalogSortDropdown;
