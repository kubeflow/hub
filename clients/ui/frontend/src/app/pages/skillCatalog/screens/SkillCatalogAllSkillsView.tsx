import React from 'react';
import { SkillCatalogContext } from '~/app/context/skillCatalog/SkillCatalogContext';
import { CatalogAllItemsView } from '~/app/shared/components/catalog';
import { skillFiltersToFilterQuery } from '~/app/pages/skillCatalog/utils/skillCatalogUtils';
import SkillCatalogCategorySection from './SkillCatalogCategorySection';

type SkillCatalogAllSkillsViewProps = {
  searchTerm: string;
};

const SkillCatalogAllSkillsView: React.FC<SkillCatalogAllSkillsViewProps> = ({ searchTerm }) => {
  const { catalogSources, catalogLabels, setSelectedSourceLabel, filters } =
    React.useContext(SkillCatalogContext);

  const filterQuery = React.useMemo(() => skillFiltersToFilterQuery(filters), [filters]);

  const handleShowMoreCategory = React.useCallback(
    (categoryLabel: string) => {
      setSelectedSourceLabel(categoryLabel);
    },
    [setSelectedSourceLabel],
  );

  return (
    <CatalogAllItemsView
      searchTerm={searchTerm}
      catalogSources={catalogSources}
      catalogLabels={catalogLabels}
      pageSize={4}
      otherSectionKey="other-skills"
      onShowMore={handleShowMoreCategory}
      renderCategorySection={(label, term, pageSize, onShowMore) => (
        <SkillCatalogCategorySection
          label={label}
          searchTerm={term}
          filterQuery={filterQuery || undefined}
          pageSize={pageSize}
          onShowMore={onShowMore}
        />
      )}
    />
  );
};

export default SkillCatalogAllSkillsView;
