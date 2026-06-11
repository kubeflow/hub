import * as React from 'react';
import { Alert, Divider, Spinner, Stack, StackItem } from '@patternfly/react-core';
import CatalogStringFilter from './CatalogStringFilter';
import { isCustomFilterItem, type FilterPanelItem } from './hooks/useCatalogFilterConfigs';

type CatalogFilterPanelProps = {
  loaded: boolean;
  loadError?: Error;
  filters: FilterPanelItem[];
  extraContent?: React.ReactNode;
  testIdPrefix?: string;
};

const CatalogFilterPanel: React.FC<CatalogFilterPanelProps> = ({
  loaded,
  loadError,
  filters,
  extraContent,
  testIdPrefix = 'catalog-filter',
}) => {
  if (!loaded) {
    return <Spinner />;
  }

  if (loadError) {
    return (
      <Alert variant="danger" title="Failed to load filter options" isInline>
        {loadError.message}
      </Alert>
    );
  }

  const visibleFilters = filters.filter((f) => f.visible !== false);

  return (
    <Stack hasGutter>
      {extraContent}
      {visibleFilters.map((item, index) => (
        <React.Fragment key={item.key}>
          <StackItem>
            {isCustomFilterItem(item) ? (
              item.customContent
            ) : (
              <>
                <CatalogStringFilter
                  title={item.title}
                  filterValues={item.filterValues}
                  selectedValues={item.selectedValues}
                  onToggle={item.onToggle}
                  getLabel={item.getLabel}
                  testIdBase={item.testIdBase ?? `${testIdPrefix}-${item.key}`}
                  getCheckboxTestId={item.getCheckboxTestId}
                />
                {item.footer}
              </>
            )}
          </StackItem>
          {index < visibleFilters.length - 1 && <Divider />}
        </React.Fragment>
      ))}
    </Stack>
  );
};

export default CatalogFilterPanel;
