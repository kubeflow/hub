import React, { useState } from 'react';
import {
  Button,
  Content,
  ContentVariants,
  Flex,
  Popover,
  Skeleton,
  Stack,
  StackItem,
} from '@patternfly/react-core';
import { Link } from 'react-router-dom';
import { HelpIcon, AngleLeftIcon, AngleRightIcon, ArrowRightIcon } from '@patternfly/react-icons';
import { TruncatedText } from 'mod-arch-shared';
import type { CatalogSource } from '~/app/shared/types/catalogTypes';
import type { CatalogModel } from '~/app/modelCatalogTypes';
import {
  extractValidatedModelMetrics,
  getLatencyValue,
} from '~/app/pages/modelCatalog/utils/validatedModelUtils';
import { catalogModelDetailsTabFromModel } from '~/app/routes/modelCatalog/catalogModel';
import {
  ModelDetailsTab,
  ModelCatalogNumberFilterKey,
  LatencyMetric,
  LatencyMetricLabels,
  latencyMetricDescriptions,
  parseLatencyFilterKey,
  SortOrder,
} from '~/concepts/modelCatalog/const';
import { useCatalogPerformanceArtifacts } from '~/app/hooks/modelCatalog/useCatalogPerformanceArtifacts';
import {
  getActiveLatencyFieldName,
  stripArtifactsPrefix,
} from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import { formatLatency } from '~/app/pages/modelCatalog/utils/performanceMetricsUtils';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import { useNotification } from '~/app/hooks/useNotification';

type ModelCatalogCardBodyProps = {
  model: CatalogModel;
  isValidated: boolean;
  source: CatalogSource | undefined;
};

const ModelCatalogCardBody: React.FC<ModelCatalogCardBodyProps> = ({
  model,
  isValidated,
  source,
}) => {
  const [currentPerformanceIndex, setCurrentPerformanceIndex] = useState(0);
  const { filters, filterOptions, performanceViewEnabled } = React.useContext(ModelCatalogContext);
  const notification = useNotification();
  const errorNotificationShown = React.useRef(false);

  const handlePreviousBenchmark = () => {
    setCurrentPerformanceIndex((prev) => (prev > 0 ? prev - 1 : performanceMetrics.length - 1));
  };

  const handleNextBenchmark = () => {
    setCurrentPerformanceIndex((prev) => (prev < performanceMetrics.length - 1 ? prev + 1 : 0));
  };

  // Get performance-specific filter params for the /performance_artifacts endpoint
  // Always apply performance filters to get accurate benchmark counts, regardless of toggle state
  const targetRPS = filters[ModelCatalogNumberFilterKey.MAX_RPS];
  // Get full filter key for display purposes
  const latencyFieldName = getActiveLatencyFieldName(filters);
  // Use short property key (e.g., 'ttft_p90') for the catalog API, not the full filter key
  const latencyProperty = latencyFieldName
    ? parseLatencyFilterKey(latencyFieldName).propertyKey
    : undefined;

  // Fetch performance artifacts from the new endpoint with server-side filtering
  // Always pass filterData so perf filters are applied - this ensures accurate benchmark counts
  // The toggle only controls display, not the artifact fetch filters
  const [performanceArtifactsList, performanceArtifactsLoaded, performanceArtifactsError] =
    useCatalogPerformanceArtifacts(
      source?.id || '',
      model.name,
      {
        targetRPS,
        latencyProperty,
        recommendations: true,
        // TODO this is a temporary workaround to avoid capping performance artifacts with a default page size of 20.
        //      we need to implement proper cursor-based pagination as the user clicks through artifacts on a card.
        pageSize: '999',
        // If a latency filter is applied, sort artifacts on the card by lowest latency.
        ...(latencyFieldName && {
          orderBy: stripArtifactsPrefix(latencyFieldName),
          sortOrder: SortOrder.ASC,
        }),
      },
      filters,
      filterOptions,
      isValidated, // Only fetch if validated
    );

  const performanceMetrics = performanceArtifactsList.items;

  // NOTE: Accuracy metrics are not currently returned by the /performance_artifacts endpoint.
  // This is kept as a placeholder for when accuracy metrics support is restored.
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const accuracyMetrics: any[] = [];

  const isLoading = isValidated && !performanceArtifactsLoaded;

  // Show toast notification when error occurs, then fall back to description
  React.useEffect(() => {
    if (performanceArtifactsError && isValidated && !errorNotificationShown.current) {
      notification.error(
        `Error loading performance artifacts for ${model.name}`,
        performanceArtifactsError.message,
      );
      errorNotificationShown.current = true;
    }
    if (!performanceArtifactsError) {
      errorNotificationShown.current = false;
    }
  }, [performanceArtifactsError, isValidated, model.name, notification]);

  if (isLoading) {
    return <Skeleton width="100%" screenreaderText="Loading model artifacts" />;
  }

  // On error, fall back to showing description
  if (performanceArtifactsError && isValidated) {
    return (
      <TruncatedText
        content={model.description || ''}
        maxLines={4}
        tooltipPosition="left"
        data-testid="model-catalog-card-description"
      />
    );
  }

  if (isValidated && performanceMetrics.length > 0) {
    // When performance view toggle is OFF, show description with a link to benchmarks
    if (!performanceViewEnabled) {
      return (
        <Stack hasGutter>
          <StackItem>
            <TruncatedText
              content={model.description || ''}
              maxLines={4}
              tooltipPosition="left"
              data-testid="model-catalog-card-description"
            />
          </StackItem>
          <StackItem>
            <Link
              to={catalogModelDetailsTabFromModel(
                ModelDetailsTab.PERFORMANCE_INSIGHTS,
                model.name,
                source?.id,
              )}
            >
              <Button
                variant="link"
                isInline
                tabIndex={-1}
                icon={<ArrowRightIcon />}
                iconPosition="end"
                style={{ padding: 0, fontSize: 'inherit' }}
                data-testid="validated-model-benchmark-link"
              >
                View {performanceMetrics.length} benchmark
                {performanceMetrics.length !== 1 ? 's' : ''}
              </Button>
            </Link>
          </StackItem>
        </Stack>
      );
    }

    // When performance view toggle is ON, show hardware, latency and replicas data
    const metrics = extractValidatedModelMetrics(
      performanceMetrics,
      accuracyMetrics,
      currentPerformanceIndex,
    );

    // If no valid metrics, fall back to showing description
    if (!metrics) {
      return (
        <TruncatedText
          content={model.description || ''}
          maxLines={4}
          tooltipPosition="left"
          data-testid="model-catalog-card-description"
        />
      );
    }

    // Get the selected latency metric from filters, or default to TTFT
    const activeLatencyField = latencyFieldName;
    const latencyValue =
      getLatencyValue(metrics.latencyMetrics, activeLatencyField) ?? metrics.ttftMean;
    const latencyLabel: LatencyMetric = activeLatencyField
      ? parseLatencyFilterKey(activeLatencyField).metric
      : LatencyMetric.TTFT;

    const coldStartValue = metrics.coldStartTimeToLoadSeconds;

    return (
      <Stack hasGutter>
        <StackItem>
          <Flex justifyContent={{ default: 'justifyContentSpaceBetween' }}>
            <Flex direction={{ default: 'column' }}>
              <span className="pf-v6-u-font-weight-bold" data-testid="validated-model-hardware">
                {metrics.hardwareConfiguration}
              </span>
              <Content component={ContentVariants.small}>Hardware</Content>
            </Flex>
            <Flex direction={{ default: 'column' }}>
              <span className="pf-v6-u-font-weight-bold" data-testid="validated-model-replicas">
                {metrics.replicas !== undefined ? metrics.replicas : metrics.rpsPerReplica}
              </span>
              <Content component={ContentVariants.small}>
                {metrics.replicas !== undefined ? 'Replicas' : 'RPS/rep.'}
              </Content>
            </Flex>
            <Flex direction={{ default: 'column' }}>
              <span className="pf-v6-u-font-weight-bold" data-testid="validated-model-latency">
                {formatLatency(latencyValue)}
              </span>
              <Flex alignItems={{ default: 'alignItemsBaseline' }} gap={{ default: 'gapXs' }}>
                <Content component={ContentVariants.small}>{latencyLabel}</Content>
                <Popover
                  bodyContent={
                    <div>
                      <p>
                        <strong>{LatencyMetricLabels[latencyLabel] ?? latencyLabel}:</strong>{' '}
                        {latencyMetricDescriptions[latencyLabel] ?? ''}
                      </p>
                    </div>
                  }
                >
                  <Button
                    icon={<HelpIcon />}
                    hasNoPadding
                    aria-label="More info for latency"
                    variant="plain"
                  />
                </Popover>
              </Flex>
            </Flex>
            {coldStartValue !== undefined && coldStartValue > 0 && (
              <Flex direction={{ default: 'column' }}>
                <span className="pf-v6-u-font-weight-bold" data-testid="validated-model-cold-start">
                  {coldStartValue.toFixed(2)} s
                </span>
                <Flex alignItems={{ default: 'alignItemsBaseline' }} gap={{ default: 'gapXs' }}>
                  <Content component={ContentVariants.small}>Cold start load time</Content>
                  <Popover
                    bodyContent={
                      <div>
                        <p>
                          The time it takes for vLLM to load the model. This does not include the
                          time it takes to download the model.
                        </p>
                      </div>
                    }
                  >
                    <Button
                      icon={<HelpIcon />}
                      hasNoPadding
                      aria-label="More info for cold start load time"
                      variant="plain"
                    />
                  </Popover>
                </Flex>
              </Flex>
            )}
          </Flex>
        </StackItem>

        <StackItem>
          <Flex
            alignItems={{ default: 'alignItemsCenter' }}
            justifyContent={{ default: 'justifyContentSpaceBetween' }}
          >
            <span data-testid="validated-model-benchmarks">
              {currentPerformanceIndex + 1} of {performanceMetrics.length}{' '}
              <Link
                to={catalogModelDetailsTabFromModel(
                  ModelDetailsTab.PERFORMANCE_INSIGHTS,
                  model.name,
                  source?.id,
                )}
              >
                <Button
                  variant="link"
                  isInline
                  tabIndex={-1}
                  style={{ padding: 0, fontSize: 'inherit' }}
                  data-testid="validated-model-benchmark-link"
                >
                  benchmarks
                </Button>
              </Link>
            </span>
            <Flex gap={{ default: 'gapSm' }} alignItems={{ default: 'alignItemsCenter' }}>
              <Button
                variant="plain"
                icon={<AngleLeftIcon />}
                aria-label="Previous benchmark"
                data-testid="validated-model-benchmark-prev"
                onClick={handlePreviousBenchmark}
                isDisabled={performanceMetrics.length <= 1}
              />
              <Button
                variant="plain"
                icon={<AngleRightIcon />}
                aria-label="Next benchmark"
                data-testid="validated-model-benchmark-next"
                onClick={handleNextBenchmark}
                isDisabled={performanceMetrics.length <= 1}
              />
            </Flex>
          </Flex>
        </StackItem>
      </Stack>
    );
  }

  // Standard card body for non-validated models
  return (
    <TruncatedText
      content={model.description || ''}
      maxLines={4}
      tooltipPosition="left"
      data-testid="model-catalog-card-description"
    />
  );
};

export default ModelCatalogCardBody;
