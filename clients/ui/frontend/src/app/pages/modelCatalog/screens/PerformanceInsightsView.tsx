import * as React from 'react';
import {
  PageSection,
  Card,
  CardBody,
  Title,
  Flex,
  FlexItem,
  Alert,
  Button,
  Stack,
  StackItem,
} from '@patternfly/react-core';
import { useParams } from 'react-router-dom';
import HardwareConfigurationTable from '~/app/pages/modelCatalog/components/HardwareConfigurationTable';
import {
  CatalogModel,
  CatalogModelDetailsParams,
  PerformanceArtifactsParams,
} from '~/app/modelCatalogTypes';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import { usePaginatedCatalogPerformanceArtifacts } from '~/app/hooks/modelCatalog/useCatalogPerformanceArtifacts';
import {
  ModelCatalogNumberFilterKey,
  ModelCatalogStringFilterKey,
  DEFAULT_PERFORMANCE_FILTERS_QUERY_NAME,
  parseLatencyFilterKey,
} from '~/concepts/modelCatalog/const';
import {
  decodeParams,
  getActiveLatencyFieldName,
  filterRegularPerformanceArtifacts,
  resolveHardwareConfigurations,
} from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import { getDefaultFiltersFromNamedQuery } from '~/app/pages/modelCatalog/utils/performanceFilterUtils';
import TensorTypeComparisonCard from './TensorTypeComparisonCard';

const HARDWARE_CONFIG_PAGE_SIZE = 20;

type PerformanceInsightsViewProps = {
  model: CatalogModel;
};

const PerformanceInsightsView: React.FC<PerformanceInsightsViewProps> = ({ model }) => {
  const params = useParams<CatalogModelDetailsParams>();
  const decodedParams = decodeParams(params);
  const {
    filters,
    filterOptions,
    filterOptionsLoaded,
    setFilters,
    setLastViewedModelName,
    setPerformanceFiltersChangedOnDetailsPage,
  } = React.useContext(ModelCatalogContext);

  // Apply default performance filters on mount if they don't have values yet
  // Details page should always have default filters applied (regardless of toggle state)
  React.useEffect(() => {
    if (!filterOptionsLoaded || !filterOptions?.namedQueries) {
      return;
    }

    // Check if any performance filter already has a value
    const hasUseCaseValue = filters[ModelCatalogStringFilterKey.USE_CASE].length > 0;
    const hasRpsValue = filters[ModelCatalogNumberFilterKey.MAX_RPS] !== undefined;
    const hasLatencyValue = getActiveLatencyFieldName(filters) !== undefined;

    // If no performance filters are set, apply defaults
    if (!hasUseCaseValue && !hasRpsValue && !hasLatencyValue) {
      const defaultQuery = filterOptions.namedQueries[DEFAULT_PERFORMANCE_FILTERS_QUERY_NAME];
      const defaults = getDefaultFiltersFromNamedQuery(filterOptions, defaultQuery);
      setFilters((prev) => {
        const next = { ...prev };
        Object.entries(defaults).forEach(([filterKey, value]) => {
          // eslint-disable-next-line @typescript-eslint/consistent-type-assertions -- dynamic filter key from namedQueries
          (next as Record<string, unknown>)[filterKey] = value;
        });
        return next;
      });
    }
    // Only run on mount when filterOptions become available
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filterOptionsLoaded]);

  // Get performance-specific filter params for the /performance_artifacts endpoint
  const targetRPS = filters[ModelCatalogNumberFilterKey.MAX_RPS];

  // Get full filter key and convert to short property key for the catalog API
  const latencyFieldName = getActiveLatencyFieldName(filters);

  const latencyProperty = latencyFieldName
    ? parseLatencyFilterKey(latencyFieldName).propertyKey
    : undefined;
  const [tableSort, setTableSort] = React.useState<
    Pick<PerformanceArtifactsParams, 'orderBy' | 'sortOrder'>
  >({});

  // Fetch performance artifacts from server with filtering and cursor-based pagination
  const {
    performanceArtifacts,
    performanceArtifactsLoaded,
    performanceArtifactsLoadError: performanceArtifactsError,
  } = usePaginatedCatalogPerformanceArtifacts(
    decodedParams.sourceId || '',
    encodeURIComponent(`${decodedParams.modelName}`),
    {
      targetRPS,
      latencyProperty,
      recommendations: true,
      pageSize: String(HARDWARE_CONFIG_PAGE_SIZE),
      orderBy: tableSort.orderBy,
      sortOrder: tableSort.sortOrder,
    },
    filters,
    filterOptions,
  );

  // After defaults are applied on the details page, clear the "changed on details page" flag
  // and record the last viewed model name for the landing page alert content.
  React.useEffect(() => {
    if (!filterOptionsLoaded) {
      return;
    }

    setPerformanceFiltersChangedOnDetailsPage(false);
    setLastViewedModelName(model.name);
  }, [
    filterOptionsLoaded,
    setPerformanceFiltersChangedOnDetailsPage,
    setLastViewedModelName,
    model.name,
  ]);

  const regularPerformanceArtifacts = React.useMemo(
    () => filterRegularPerformanceArtifacts(performanceArtifacts.items),
    [performanceArtifacts.items],
  );

  const hardwareConfigurations = React.useMemo(
    () => resolveHardwareConfigurations(performanceArtifacts.items, model.customProperties),
    [performanceArtifacts.items, model.customProperties],
  );

  if (performanceArtifactsError) {
    return (
      <PageSection padding={{ default: 'noPadding' }}>
        <Alert variant="danger" isInline title="Error loading performance data">
          {performanceArtifactsError.message}
        </Alert>
      </PageSection>
    );
  }

  return (
    <Stack hasGutter>
      <StackItem>
        <Card>
          <CardBody>
            <Flex direction={{ default: 'column' }} gap={{ default: 'gapLg' }}>
              <FlexItem>
                <Flex direction={{ default: 'column' }} gap={{ default: 'gapSm' }}>
                  <FlexItem>
                    <Title headingLevel="h2" size="lg">
                      Hardware Configuration
                    </Title>
                  </FlexItem>
                  <FlexItem>
                    <p>
                      Compare the performance metrics of hardware configuration to determine the
                      most suitable option for deployment.
                    </p>
                  </FlexItem>
                </Flex>
              </FlexItem>
              <FlexItem>
                <HardwareConfigurationTable
                  performanceArtifacts={regularPerformanceArtifacts}
                  hardwareConfigurations={hardwareConfigurations}
                  isLoading={!performanceArtifactsLoaded && performanceArtifacts.items.length === 0}
                  onSortChange={setTableSort}
                />
              </FlexItem>
              {performanceArtifacts.loadMoreError && (
                <FlexItem>
                  <Alert
                    variant="danger"
                    isInline
                    title="Error loading more hardware configurations"
                  >
                    {performanceArtifacts.loadMoreError.message}
                  </Alert>
                </FlexItem>
              )}
              {performanceArtifacts.hasMore && (
                <FlexItem>
                  <Flex justifyContent={{ default: 'justifyContentCenter' }}>
                    <Button
                      data-testid="hardware-config-load-more-button"
                      variant="secondary"
                      onClick={performanceArtifacts.loadMore}
                      isLoading={performanceArtifacts.isLoadingMore}
                    >
                      Load more
                    </Button>
                  </Flex>
                </FlexItem>
              )}
            </Flex>
          </CardBody>
        </Card>
      </StackItem>
      <StackItem>
        <TensorTypeComparisonCard model={model} />
      </StackItem>
    </Stack>
  );
};

export default PerformanceInsightsView;
