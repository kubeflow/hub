import * as React from 'react';
import {
  Badge,
  Checkbox,
  Dropdown,
  Flex,
  FlexItem,
  MenuToggle,
  MenuToggleElement,
  Panel,
  PanelMain,
  SearchInput,
} from '@patternfly/react-core';
import { useHardwareConfigurationFilterState } from '~/app/pages/modelCatalog/hooks/useHardwareConfigurationFilterState';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import { ModelCatalogStringFilterKey } from '~/concepts/modelCatalog/const';

type HardwareConfigurationOption = {
  value: string;
  label: string;
};

const HardwareConfigurationFilter: React.FC = () => {
  const { appliedValues, setAppliedValues } = useHardwareConfigurationFilterState();
  const { filterOptions } = React.useContext(ModelCatalogContext);
  const [isOpen, setIsOpen] = React.useState(false);
  const [searchValue, setSearchValue] = React.useState('');

  // Get hardware configuration options from filterOptions (from API filter_options endpoint)
  // Always use filterOptions - never extract from artifacts per team decision
  const hardwareOptions: HardwareConfigurationOption[] = React.useMemo(() => {
    const hardwareConfigOptions =
      filterOptions?.filters?.[ModelCatalogStringFilterKey.HARDWARE_CONFIGURATION];
    if (!hardwareConfigOptions?.values) {
      return [];
    }
    return hardwareConfigOptions.values.map((config) => ({
      value: config,
      label: config,
    }));
  }, [filterOptions]);

  // Filter options based on search value
  const filteredOptions = React.useMemo(
    () =>
      hardwareOptions.filter(
        (option) =>
          option.label.toLowerCase().includes(searchValue.trim().toLowerCase()) ||
          appliedValues.includes(option.value),
      ),
    [hardwareOptions, searchValue, appliedValues],
  );

  const selectedCount = appliedValues.length;

  const isHardwareSelected = (value: string): boolean => appliedValues.includes(value);

  const toggleHardwareSelection = (value: string, selected: boolean) => {
    if (selected) {
      setAppliedValues([...appliedValues, value]);
    } else {
      setAppliedValues(appliedValues.filter((item) => item !== value));
    }
  };

  const toggle = (toggleRef: React.Ref<MenuToggleElement>) => (
    <MenuToggle
      ref={toggleRef}
      data-testid="hardware-configuration-filter"
      onClick={() => setIsOpen(!isOpen)}
      isExpanded={isOpen}
      isFullHeight
      style={{ minWidth: '200px', width: 'fit-content', height: '56px' }}
      badge={selectedCount > 0 ? <Badge>{selectedCount} selected</Badge> : undefined}
    >
      Hardware
    </MenuToggle>
  );

  const filterContent = (
    <Panel>
      <PanelMain className="pf-v6-u-p-md" style={{ maxHeight: '300px', overflowY: 'auto' }}>
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsSm' }}>
          {/* Search input */}
          <FlexItem>
            <SearchInput
              placeholder="Search hardware"
              value={searchValue}
              onChange={(_event, value) => setSearchValue(value)}
              onClear={() => setSearchValue('')}
            />
          </FlexItem>
          {/* Hardware configuration checkboxes */}
          <FlexItem>
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsXs' }}>
              {filteredOptions.length === 0 ? (
                <FlexItem>No results found</FlexItem>
              ) : (
                filteredOptions.map((option) => (
                  <FlexItem key={option.value}>
                    <Flex alignItems={{ default: 'alignItemsCenter' }}>
                      <FlexItem flex={{ default: 'flex_1' }}>
                        <Checkbox
                          label={option.label}
                          id={option.value}
                          isChecked={isHardwareSelected(option.value)}
                          onChange={(_, checked) => toggleHardwareSelection(option.value, checked)}
                        />
                      </FlexItem>
                    </Flex>
                  </FlexItem>
                ))
              )}
            </Flex>
          </FlexItem>
        </Flex>
      </PanelMain>
    </Panel>
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

export default HardwareConfigurationFilter;
