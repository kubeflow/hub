import * as React from 'react';
import {
  Alert,
  Button,
  Content,
  Flex,
  FlexItem,
  Grid,
  GridItem,
  Skeleton,
  StackItem,
  Title,
} from '@patternfly/react-core';
import { ArrowRightIcon, SearchIcon } from '@patternfly/react-icons';
import { DEFAULT_CATALOG_GRID_SPANS } from './types/catalogFilterTypes';
import type { CatalogGridSpans } from './types/catalogFilterTypes';
import EmptyCatalogState from './EmptyCatalogState';

type CatalogCategorySectionTestIds = {
  title?: string;
  showMore?: string;
  error?: string;
  skeleton?: (index: number) => string;
  empty?: string;
};

type CatalogCategorySectionProps<T> = {
  label: string;
  categoryTitle: string;
  categoryDescription?: string;
  items: T[];
  loaded: boolean;
  loadError: Error | undefined;
  pageSize: number;
  onShowMore: (label: string) => void;
  renderCard: (item: T) => React.ReactNode;
  getItemKey: (item: T) => string;
  gridSpans?: CatalogGridSpans;
  showAllThreshold?: number;
  skeletonCount?: number;
  loadingScreenReaderText?: string;
  testIds?: CatalogCategorySectionTestIds;
};

function CatalogCategorySection<T>({
  label,
  categoryTitle,
  categoryDescription,
  items,
  loaded,
  loadError,
  pageSize,
  onShowMore,
  renderCard,
  getItemKey,
  gridSpans = DEFAULT_CATALOG_GRID_SPANS,
  showAllThreshold,
  skeletonCount,
  loadingScreenReaderText,
  testIds,
}: CatalogCategorySectionProps<T>): React.ReactElement {
  const effectiveThreshold = showAllThreshold ?? pageSize;
  const effectiveSkeletonCount = skeletonCount ?? pageSize;
  const itemsToDisplay = items.slice(0, pageSize);

  return (
    <StackItem className="pf-v6-u-pb-xl">
      <Flex
        alignItems={{ default: 'alignItemsCenter' }}
        justifyContent={{ default: 'justifyContentSpaceBetween' }}
        className="pf-v6-u-mb-md"
      >
        <FlexItem>
          <Title headingLevel="h3" size="lg" data-testid={testIds?.title}>
            {categoryTitle}
          </Title>
          {categoryDescription && (
            <Content component="p" className="pf-v6-u-color-200 pf-v6-u-mt-sm">
              {categoryDescription}
            </Content>
          )}
        </FlexItem>

        {items.length >= effectiveThreshold && (
          <FlexItem>
            <Button
              variant="link"
              size="sm"
              isInline
              icon={<ArrowRightIcon />}
              iconPosition="right"
              data-testid={testIds?.showMore}
              onClick={() => onShowMore(label)}
            >
              Show all {categoryTitle}
            </Button>
          </FlexItem>
        )}
      </Flex>

      {loadError ? (
        <Alert
          variant="danger"
          title={`Failed to load ${categoryTitle}`}
          data-testid={testIds?.error}
        >
          {loadError.message}
        </Alert>
      ) : !loaded ? (
        <Grid hasGutter>
          {Array.from({ length: effectiveSkeletonCount }).map((_, index) => (
            <GridItem key={index} {...gridSpans}>
              <Skeleton
                height="280px"
                width="100%"
                screenreaderText={loadingScreenReaderText ?? `Loading ${label} items`}
                data-testid={testIds?.skeleton?.(index)}
              />
            </GridItem>
          ))}
        </Grid>
      ) : items.length === 0 ? (
        <EmptyCatalogState
          testid={testIds?.empty}
          title="No result found"
          headerIcon={SearchIcon}
          description="Adjust your filters and try again."
        />
      ) : (
        <Grid hasGutter>
          {itemsToDisplay.map((item) => (
            <GridItem key={getItemKey(item)} {...gridSpans}>
              {renderCard(item)}
            </GridItem>
          ))}
        </Grid>
      )}
    </StackItem>
  );
}

export default CatalogCategorySection;
