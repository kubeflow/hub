import * as React from 'react';
import { ApplicationsPage, ProjectObjectType, TitleWithIcon } from 'mod-arch-shared';
import { SearchIcon } from '@patternfly/react-icons';
import { SkillCatalogContext } from '~/app/context/skillCatalog/SkillCatalogContext';
import { SKILL_CATALOG_TITLE, SKILL_CATALOG_DESCRIPTION } from '~/app/pages/skillCatalog/const';
import { hasSkillFiltersApplied } from '~/app/pages/skillCatalog/utils/skillCatalogUtils';
import SkillCatalogFilters from '~/app/pages/skillCatalog/components/SkillCatalogFilters';
import { CatalogPageLayout, EmptyCatalogState } from '~/app/shared/components/catalog';
import SkillCatalogSourceLabelSelector from './SkillCatalogSourceLabelSelector';
import SkillCatalogAllSkillsView from './SkillCatalogAllSkillsView';
import SkillCatalogGalleryView from './SkillCatalogGalleryView';

const SkillCatalog: React.FC = () => {
  const {
    searchQuery,
    setSearchQuery,
    clearAllFilters,
    selectedSourceLabel,
    setSelectedSourceLabel,
    filters,
    catalogSources,
    catalogLabels,
    catalogSourcesLoaded,
    emptyCategoryLabels,
    setCategoryCount,
  } = React.useContext(SkillCatalogContext);

  const filtersApplied = hasSkillFiltersApplied(filters, searchQuery);
  const isAllSkillsView = selectedSourceLabel === undefined && !filtersApplied;

  const handleSearch = React.useCallback(
    (term: string) => {
      setSearchQuery(term);
    },
    [setSearchQuery],
  );

  const handleClearSearch = React.useCallback(() => {
    setSearchQuery('');
  }, [setSearchQuery]);

  const handleResetAllFilters = React.useCallback(() => {
    clearAllFilters();
  }, [clearAllFilters]);

  return (
    <ApplicationsPage
      title={
        <TitleWithIcon title={SKILL_CATALOG_TITLE} objectType={ProjectObjectType.mcpCatalog} />
      }
      description={SKILL_CATALOG_DESCRIPTION}
      empty={false}
      loaded
      provideChildrenPadding
    >
      <CatalogPageLayout
        catalogSources={catalogSources}
        catalogLabels={catalogLabels}
        catalogSourcesLoaded={catalogSourcesLoaded}
        selectedSourceLabel={selectedSourceLabel}
        onSelectSourceLabel={setSelectedSourceLabel}
        isAllItemsView={isAllSkillsView}
        emptyCategoryLabels={emptyCategoryLabels}
        setCategoryCount={setCategoryCount}
        renderEmptyCategoriesState={() => (
          <EmptyCatalogState
            testid="empty-skill-catalog-no-categories"
            title="No skills available"
            headerIcon={SearchIcon}
            description="There are no skill categories available. Configure sources in settings to get started."
          />
        )}
        renderFilterSidebar={() => <SkillCatalogFilters />}
        renderToolbar={() => (
          <SkillCatalogSourceLabelSelector
            searchTerm={searchQuery}
            onSearch={handleSearch}
            onClearSearch={handleClearSearch}
          />
        )}
        renderAllItemsView={() => <SkillCatalogAllSkillsView searchTerm={searchQuery} />}
        renderGalleryView={(isSingleCategory, singleCategoryLabel) => (
          <SkillCatalogGalleryView
            handleFilterReset={handleResetAllFilters}
            isSingleCategory={isSingleCategory}
            singleCategoryLabel={singleCategoryLabel}
          />
        )}
      />
    </ApplicationsPage>
  );
};

export default SkillCatalog;
