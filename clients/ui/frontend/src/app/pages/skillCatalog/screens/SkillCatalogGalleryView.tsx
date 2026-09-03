import * as React from 'react';
import { Button } from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';
import { SkillCatalogContext } from '~/app/context/skillCatalog/SkillCatalogContext';
import { useSkillsBySourceLabelWithAPI } from '~/app/hooks/skillCatalog/useSkillsBySourceLabel';
import { SKILL_CATALOG_GRID_SPAN, OTHER_SKILLS_DISPLAY_NAME } from '~/app/pages/skillCatalog/const';
import { skillFiltersToFilterQuery } from '~/app/pages/skillCatalog/utils/skillCatalogUtils';
import {
  getLabelDisplayName,
  getLabelDescription,
  CatalogGalleryLayout,
  EmptyCatalogState,
} from '~/app/shared/components/catalog';
import SkillCatalogCard from '~/app/pages/skillCatalog/components/SkillCatalogCard';

const PAGE_SIZE = 10;

type SkillCatalogGalleryViewProps = {
  handleFilterReset: () => void;
  isSingleCategory?: boolean;
  singleCategoryLabel?: string;
};

const SkillCatalogGalleryView: React.FC<SkillCatalogGalleryViewProps> = ({
  handleFilterReset,
  isSingleCategory = false,
  singleCategoryLabel,
}) => {
  const {
    skillApiState,
    selectedSourceLabel,
    searchQuery,
    filters,
    catalogLabels,
    catalogLabelsLoaded,
  } = React.useContext(SkillCatalogContext);

  const filterQuery = React.useMemo(() => skillFiltersToFilterQuery(filters), [filters]);

  const { skills, skillsLoaded, skillsLoadError } = useSkillsBySourceLabelWithAPI(skillApiState, {
    sourceLabel: selectedSourceLabel,
    pageSize: PAGE_SIZE,
    searchQuery,
    filterQuery: filterQuery || undefined,
  });

  const loaded = skillsLoaded && catalogLabelsLoaded;

  const effectiveCategoryLabel = singleCategoryLabel || selectedSourceLabel || '';
  const categoryTitle = isSingleCategory
    ? getLabelDisplayName(
        effectiveCategoryLabel,
        catalogLabels,
        OTHER_SKILLS_DISPLAY_NAME,
        'skills',
      )
    : undefined;
  const categoryDescription = isSingleCategory
    ? getLabelDescription(effectiveCategoryLabel, catalogLabels)
    : undefined;

  return (
    <CatalogGalleryLayout
      items={skills.items}
      loaded={loaded}
      loadError={skillsLoadError}
      renderCard={(skill) => <SkillCatalogCard skill={skill} />}
      getItemKey={(skill) => skill.id}
      gridSpans={SKILL_CATALOG_GRID_SPAN}
      hasMore={skills.hasMore && skills.items.length >= PAGE_SIZE}
      isLoadingMore={skills.isLoadingMore}
      onLoadMore={skills.loadMore}
      loadMoreLabel="Load more skills"
      loadingMoreLabel="Loading more skills..."
      loadingLabel="Loading skills..."
      errorTitle="Failed to load skills"
      categoryTitle={categoryTitle}
      categoryDescription={categoryDescription}
      renderEmptyState={() => (
        <EmptyCatalogState
          testid="empty-skill-catalog-state"
          title="No results found"
          headerIcon={SearchIcon}
          description="No skills match your filters. Adjust your filters and try again."
          primaryAction={
            <Button variant="link" onClick={handleFilterReset}>
              Reset filters
            </Button>
          }
        />
      )}
    />
  );
};

export default SkillCatalogGalleryView;
