import * as React from 'react';
import {
  Alert,
  Bullseye,
  Button,
  Content,
  EmptyState,
  Flex,
  Grid,
  GridItem,
  Spinner,
  Title,
} from '@patternfly/react-core';
import ScrollViewOnMount from '~/app/shared/components/ScrollViewOnMount';
import { DEFAULT_CATALOG_GRID_SPANS } from './types/catalogFilterTypes';
import type { CatalogGridSpans } from './types/catalogFilterTypes';

type CatalogGalleryLayoutProps<T> = {
  items: T[];
  loaded: boolean;
  loadError: Error | undefined;
  renderCard: (item: T) => React.ReactNode;
  getItemKey: (item: T) => string;
  gridSpans?: CatalogGridSpans;
  hasMore?: boolean;
  isLoadingMore?: boolean;
  onLoadMore?: () => void;
  loadMoreLabel?: string;
  loadingMoreLabel?: string;
  loadingLabel?: string;
  errorTitle?: string;
  renderEmptyState?: () => React.ReactNode;
  renderExtraEmptyStates?: () => React.ReactNode | null;
  categoryTitle?: string;
  categoryDescription?: string;
  headerExtra?: React.ReactNode;
};

function CatalogGalleryLayout<T>({
  items,
  loaded,
  loadError,
  renderCard,
  getItemKey,
  gridSpans: gridSpansProp = DEFAULT_CATALOG_GRID_SPANS,
  hasMore = false,
  isLoadingMore = false,
  onLoadMore,
  loadMoreLabel = 'Load more',
  loadingMoreLabel = 'Loading more...',
  loadingLabel = 'Loading...',
  errorTitle = 'Failed to load',
  renderEmptyState,
  renderExtraEmptyStates,
  categoryTitle,
  categoryDescription,
  headerExtra,
}: CatalogGalleryLayoutProps<T>): React.ReactElement | null {
  if (loadError) {
    return (
      <Alert variant="danger" title={errorTitle} isInline>
        {loadError.message}
      </Alert>
    );
  }

  if (!loaded) {
    return (
      <EmptyState>
        <Spinner />
        <Title headingLevel="h4" size="lg">
          {loadingLabel}
        </Title>
      </EmptyState>
    );
  }

  const extraEmptyState = renderExtraEmptyStates?.();
  if (extraEmptyState) {
    return <>{extraEmptyState}</>;
  }

  if (items.length === 0 && renderEmptyState) {
    return <>{renderEmptyState()}</>;
  }

  return (
    <>
      <ScrollViewOnMount shouldScroll scrollToTop />
      {categoryTitle && (
        <div className="pf-v6-u-mb-lg" data-testid="single-category-header">
          <Flex
            justifyContent={{ default: 'justifyContentSpaceBetween' }}
            alignItems={{ default: 'alignItemsCenter' }}
          >
            <Title headingLevel="h3" size="lg">
              {categoryTitle}
            </Title>
            {headerExtra}
          </Flex>
          {categoryDescription && (
            <Content component="p" className="pf-v6-u-color-200 pf-v6-u-mt-sm">
              {categoryDescription}
            </Content>
          )}
        </div>
      )}
      <Grid hasGutter>
        {items.map((item) => (
          <GridItem key={getItemKey(item)} {...gridSpansProp}>
            {renderCard(item)}
          </GridItem>
        ))}
      </Grid>
      {hasMore && (
        <Bullseye className="pf-v6-u-mt-lg">
          {isLoadingMore ? (
            <Flex
              direction={{ default: 'column' }}
              alignItems={{ default: 'alignItemsCenter' }}
              gap={{ default: 'gapMd' }}
            >
              <Spinner size="lg" />
              <Title size="lg" headingLevel="h5">
                {loadingMoreLabel}
              </Title>
            </Flex>
          ) : (
            <Button variant="tertiary" onClick={onLoadMore} size="lg">
              {loadMoreLabel}
            </Button>
          )}
        </Bullseye>
      )}
    </>
  );
}

export default CatalogGalleryLayout;
