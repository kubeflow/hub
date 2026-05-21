import * as React from 'react';
import { PageSection, Sidebar, SidebarContent, SidebarPanel, Stack } from '@patternfly/react-core';
import type { CatalogLabelList, CatalogSourceList } from '~/app/modelCatalogTypes';
import { getActiveSourceLabels } from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import ScrollViewOnMount from '~/app/shared/components/ScrollViewOnMount';

type CatalogPageLayoutProps = {
  catalogSources: CatalogSourceList | null;
  catalogLabels: CatalogLabelList | null;
  catalogSourcesLoaded: boolean;
  selectedSourceLabel: string | undefined;
  onSelectSourceLabel: (label: string) => void;
  isAllItemsView: boolean;
  renderEmptyCategoriesState: () => React.ReactNode;
  renderFilterSidebar: () => React.ReactNode;
  renderToolbar: () => React.ReactNode;
  renderAllItemsView: () => React.ReactNode;
  renderGalleryView: (
    isSingleCategory: boolean,
    singleCategoryLabel: string | undefined,
  ) => React.ReactNode;
};

const CatalogPageLayout: React.FC<CatalogPageLayoutProps> = ({
  catalogSources,
  catalogLabels,
  catalogSourcesLoaded,
  selectedSourceLabel,
  onSelectSourceLabel,
  isAllItemsView,
  renderEmptyCategoriesState,
  renderFilterSidebar,
  renderToolbar,
  renderAllItemsView,
  renderGalleryView,
}) => {
  const activeCategories = React.useMemo(
    () => getActiveSourceLabels(catalogSources, catalogLabels),
    [catalogSources, catalogLabels],
  );

  const isSingleCategory = activeCategories.length === 1;
  const hasNoCategories = activeCategories.length === 0;

  React.useEffect(() => {
    if (catalogSourcesLoaded && isSingleCategory && selectedSourceLabel !== activeCategories[0]) {
      onSelectSourceLabel(activeCategories[0]);
    }
  }, [
    catalogSourcesLoaded,
    isSingleCategory,
    activeCategories,
    selectedSourceLabel,
    onSelectSourceLabel,
  ]);

  if (catalogSourcesLoaded && hasNoCategories) {
    return (
      <>
        <ScrollViewOnMount shouldScroll scrollToTop />
        {renderEmptyCategoriesState()}
      </>
    );
  }

  return (
    <>
      <ScrollViewOnMount shouldScroll scrollToTop />
      <Sidebar hasBorder hasGutter>
        <SidebarPanel variant="sticky">{renderFilterSidebar()}</SidebarPanel>
        <SidebarContent>
          <Stack hasGutter>
            {renderToolbar()}
            <PageSection isFilled padding={{ default: 'noPadding' }}>
              {isAllItemsView && !isSingleCategory
                ? renderAllItemsView()
                : renderGalleryView(
                    isSingleCategory,
                    isSingleCategory ? activeCategories[0] : undefined,
                  )}
            </PageSection>
          </Stack>
        </SidebarContent>
      </Sidebar>
    </>
  );
};

export default CatalogPageLayout;
