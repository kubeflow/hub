import * as React from 'react';
import { useSkillsBySourceLabelWithAPI } from '~/app/hooks/skillCatalog/useSkillsBySourceLabel';
import useReportCategoryEmpty from '~/app/hooks/useReportCategoryEmpty';
import {
  getLabelDescription,
  getLabelDisplayName,
  CatalogCategorySection,
} from '~/app/shared/components/catalog';
import { SourceLabel } from '~/app/shared/types/catalogTypes';
import { SkillCatalogContext } from '~/app/context/skillCatalog/SkillCatalogContext';
import { SKILL_CATALOG_GRID_SPAN, OTHER_SKILLS_DISPLAY_NAME } from '~/app/pages/skillCatalog/const';
import SkillCatalogCard from '~/app/pages/skillCatalog/components/SkillCatalogCard';

type SkillCatalogCategorySectionProps = {
  label: string;
  searchTerm: string;
  filterQuery?: string;
  pageSize: number;
  onShowMore: (label: string) => void;
};

const SkillCatalogCategorySection: React.FC<SkillCatalogCategorySectionProps> = ({
  label,
  searchTerm,
  filterQuery,
  pageSize,
  onShowMore,
}) => {
  const { skillApiState, catalogLabels, reportCategoryEmpty } =
    React.useContext(SkillCatalogContext);
  // Don't forward the sentinel "null" label (SourceLabel.other) to the API — pass undefined
  // instead so the catalog service receives no sourceLabel filter.
  const apiSourceLabel = label === SourceLabel.other ? undefined : label;

  const {
    skills: rawSkills,
    skillsLoaded,
    skillsLoadError,
  } = useSkillsBySourceLabelWithAPI(skillApiState, {
    sourceLabel: apiSourceLabel,
    pageSize,
    searchQuery: searchTerm,
    filterQuery,
  });

  // Use skills directly from the catalog service — client-side source ID filtering is skipped
  // because ConfigMap source IDs (admin UI) differ from the catalog service's internal source IDs.
  const { items } = rawSkills;

  const categoryTitle = getLabelDisplayName(
    label,
    catalogLabels,
    OTHER_SKILLS_DISPLAY_NAME,
    'skills',
  );
  const categoryDescription = getLabelDescription(label, catalogLabels);
  const labelSlug = label.toLowerCase().replace(/\s+/g, '-');

  useReportCategoryEmpty(
    reportCategoryEmpty,
    label,
    skillsLoaded,
    items.length,
    searchTerm,
    skillsLoadError,
  );

  if (skillsLoaded && items.length === 0 && !searchTerm) {
    return null;
  }

  return (
    <CatalogCategorySection
      label={label}
      categoryTitle={categoryTitle}
      categoryDescription={categoryDescription}
      items={items}
      loaded={skillsLoaded}
      loadError={skillsLoadError}
      pageSize={pageSize}
      onShowMore={onShowMore}
      renderCard={(skill) => <SkillCatalogCard skill={skill} />}
      getItemKey={(skill) => skill.id}
      gridSpans={SKILL_CATALOG_GRID_SPAN}
      loadingScreenReaderText={`Loading ${label} skills`}
      testIds={{
        title: `skills-category-title-${label}`,
        showMore: `skills-show-all-${labelSlug}`,
        error: `skills-error-state-${label}`,
        skeleton: (index) => `skills-category-skeleton-${labelSlug}-${index}`,
        empty: `empty-skill-catalog-state-${label}`,
      }}
    />
  );
};

export default SkillCatalogCategorySection;
