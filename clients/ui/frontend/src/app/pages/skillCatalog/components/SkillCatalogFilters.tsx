import * as React from 'react';
import { SkillCatalogContext } from '~/app/context/skillCatalog/SkillCatalogContext';
import { CatalogFilterPanel, useCatalogFilterConfigs } from '~/app/shared/components/catalog';
import {
  SKILL_FILTER_KEYS,
  SKILL_FILTER_CATEGORY_NAMES,
  SKILL_LABEL_MAPPINGS,
} from '~/app/pages/skillCatalog/const';

const SkillCatalogFilters: React.FC = () => {
  const { filters, setFilters, filterOptions, filterOptionsLoaded, filterOptionsLoadError } =
    React.useContext(SkillCatalogContext);

  const onFilterChange = React.useCallback(
    (key: string, values: string[]) => {
      setFilters((prev) => ({ ...prev, [key]: values }));
    },
    [setFilters],
  );

  const filterPanelItems = useCatalogFilterConfigs({
    filterKeys: SKILL_FILTER_KEYS,
    filterNames: SKILL_FILTER_CATEGORY_NAMES,
    filterOptions: filterOptions?.filters,
    selectedFilters: filters,
    onFilterChange,
    labelMappings: SKILL_LABEL_MAPPINGS,
  });

  return (
    <CatalogFilterPanel
      loaded={filterOptionsLoaded}
      loadError={filterOptionsLoadError}
      filters={filterPanelItems}
      testIdPrefix="skill-filter"
    />
  );
};

export default SkillCatalogFilters;
