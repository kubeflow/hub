/* eslint-disable camelcase */
import { mockModArchResponse } from 'mod-arch-core';
import {
  mockCatalogPerformanceMetricsArtifact,
  mockFilteredPerformanceArtifactsByWorkloadType,
} from '~/__mocks__/mockCatalogModelArtifactList';
import { modelCatalog } from '~/__tests__/cypress/cypress/pages/modelCatalog';
import { appChrome } from '~/__tests__/cypress/cypress/pages/appChrome';
import { mockModelRegistry } from '~/__mocks__/mockModelRegistry';
import { MODEL_CATALOG_API_VERSION } from '~/__tests__/cypress/cypress/support/commands/api';
import { UseCaseOptionValue } from '~/concepts/modelCatalog/const';
import { ModelRegistryMetadataType } from '~/app/types';
import {
  setupModelCatalogIntercepts,
  interceptPerformanceArtifactsList,
  interceptArtifactsList,
  type ModelCatalogInterceptOptions,
} from '~/__tests__/cypress/cypress/support/interceptHelpers/modelCatalog';
import { NBSP } from '~/__tests__/cypress/cypress/support/constants';

/**
 * Initialize intercepts for model catalog tabs tests.
 * Uses shared intercept helpers to reduce duplication.
 */
const initIntercepts = (options: Partial<ModelCatalogInterceptOptions> = {}) => {
  const resolvedOptions = {
    useValidatedModel: true,
    includePerformanceArtifacts: true,
    ...options,
  };

  setupModelCatalogIntercepts(resolvedOptions);

  // Additional intercepts for tabs tests:
  // - /artifacts/ endpoint is used to determine if tabs should show
  // - /performance_artifacts/ with regex for flexible matching
  // Only add artifact intercepts if includePerformanceArtifacts is true
  if (resolvedOptions.includePerformanceArtifacts) {
    interceptArtifactsList();
    interceptPerformanceArtifactsList();
  } else {
    // Return empty artifacts list when performance artifacts should not be included
    interceptArtifactsList({ items: [], size: 0, pageSize: 10, nextPageToken: '' });
    interceptPerformanceArtifactsList({
      items: [],
      size: 0,
      pageSize: 10,
      nextPageToken: '',
    });
  }
};

describe('Model Catalog Details Tabs', () => {
  describe('Validated Models with performance artifacts (with tabs)', () => {
    beforeEach(() => {
      // Mock model registries for register button functionality
      cy.intercept('GET', '/model-registry/api/v1/model_registry*', [
        mockModelRegistry({ name: 'modelregistry-sample' }),
      ]).as('getModelRegistries');

      initIntercepts({ useValidatedModel: true, includePerformanceArtifacts: true });
      modelCatalog.visit();
    });

    describe('Tab Navigation', () => {
      it('should display tabs on model details page', () => {
        modelCatalog.findLoadingState().should('not.exist');
        modelCatalog.findModelCatalogDetailLink().first().click();
        appChrome.waitForA11y();

        // Verify tabs are present
        modelCatalog.findModelDetailsTabs().should('be.visible');
        modelCatalog.findOverviewTab().should('be.visible');
        modelCatalog.findPerformanceInsightsTab().should('be.visible');
      });

      it('should show Overview tab as active by default', () => {
        modelCatalog.findModelCatalogDetailLink().first().click();
        appChrome.waitForA11y();

        // Overview tab should be active and content should be visible
        modelCatalog.findOverviewTab().should('have.attr', 'aria-selected', 'true');
        modelCatalog.findOverviewTabContent().should('be.visible');
        modelCatalog.findDetailsDescription().should('be.visible');
        cy.url().should('include', '/model-catalog/source-2/validated-model/overview');
      });

      it('should switch to Performance Insights tab when clicked', () => {
        modelCatalog.findModelCatalogDetailLink().first().click();
        appChrome.waitForA11y();

        cy.url().should('include', '/model-catalog/source-2/validated-model/overview');

        // Click Performance Insights tab
        modelCatalog.clickPerformanceInsightsTab();

        // Verify tab switch
        modelCatalog.findPerformanceInsightsTab().should('have.attr', 'aria-selected', 'true');
        modelCatalog.findOverviewTab().should('have.attr', 'aria-selected', 'false');
        modelCatalog.findPerformanceInsightsTabContent().should('be.visible');
        cy.url().should('include', '/model-catalog/source-2/validated-model/performance-insights');
      });

      it('should switch back to Overview tab when clicked', () => {
        modelCatalog.findModelCatalogDetailLink().first().click();
        appChrome.waitForA11y();

        cy.url().should('include', '/model-catalog/source-2/validated-model/overview');

        // First switch to Performance Insights
        modelCatalog.clickPerformanceInsightsTab();
        modelCatalog.findPerformanceInsightsTab().should('have.attr', 'aria-selected', 'true');

        // Then switch back to Overview
        modelCatalog.clickOverviewTab();
        modelCatalog.findOverviewTab().should('have.attr', 'aria-selected', 'true');
        modelCatalog.findPerformanceInsightsTab().should('have.attr', 'aria-selected', 'false');
        modelCatalog.findOverviewTabContent().should('be.visible');
        cy.url().should('include', '/model-catalog/source-2/validated-model/overview');
      });
    });

    describe('Tab Content', () => {
      it('should display Hardware Configuration content in Performance Insights tab', () => {
        modelCatalog.findModelCatalogDetailLink().first().click();
        appChrome.waitForA11y();

        // Switch to Performance Insights tab
        modelCatalog.clickPerformanceInsightsTab();
        modelCatalog.waitForPerformanceInsightsTab();

        // Verify Hardware Configuration content is displayed
        modelCatalog.findHardwareConfigurationTitle().should('be.visible');
        modelCatalog.findHardwareConfigurationDescription().should('be.visible');
        modelCatalog.findHardwareConfigurationTable().should('be.visible');
      });

      it('should display Workload type column as the second column in hardware configuration table', () => {
        modelCatalog.findModelCatalogDetailLink().first().click();
        appChrome.waitForA11y();
        modelCatalog.clickPerformanceInsightsTab();

        modelCatalog
          .findHardwareConfigurationTableHeaders()
          .eq(1)
          .should('contain.text', 'Workload type');
        // First row should contain formatted workload type (Chatbot, not chatbot)
        modelCatalog
          .findHardwareConfigurationColumn('Workload type')
          .first()
          .should('contain.text', 'Chatbot')
          .should('not.contain.text', 'chatbot');
      });
    });

    describe('Workload Type Filter', () => {
      it('should display workload type filter in the toolbar', () => {
        modelCatalog.findModelCatalogDetailLink().first().click();
        appChrome.waitForA11y();
        modelCatalog.clickPerformanceInsightsTab();

        modelCatalog
          .findWorkloadTypeFilter()
          .should('be.visible')
          .should('contain.text', 'Scenario');
      });

      it('should show workload type options when clicked', () => {
        modelCatalog.findModelCatalogDetailLink().first().click();
        appChrome.waitForA11y();
        modelCatalog.clickPerformanceInsightsTab();
        modelCatalog.findWorkloadTypeFilter().click();
        modelCatalog.findWorkloadTypeOption('chatbot').should('be.visible');
        modelCatalog.findWorkloadTypeOption('code_fixing').should('be.visible');
        modelCatalog.findWorkloadTypeOption('long_rag').should('be.visible');
        modelCatalog.findWorkloadTypeOption('rag').should('be.visible');
      });

      it('should filter hardware configuration table by selected workload type', () => {
        // Note: This test verifies UI behavior after server-side filter is applied.
        // Server-side filtering is verified by the 'Server-Side Filtering' tests below.
        modelCatalog.findModelCatalogDetailLink().first().click();
        appChrome.waitForA11y();
        modelCatalog.clickPerformanceInsightsTab();
        modelCatalog.findHardwareConfigurationTableRows().should('have.length.at.least', 1);
        modelCatalog.findWorkloadTypeFilter().click();
        modelCatalog.selectWorkloadType('code_fixing');
        // Verify filter is applied (single-select shows selected value in toggle)
        modelCatalog
          .findWorkloadTypeFilter()
          .should('contain.text', 'Scenario')
          .should('contain.text', 'Code Fixing');
        // Table should still exist (server-side filtering returns mock data)
        modelCatalog.findHardwareConfigurationTableRows().should('exist');
      });

      it('should change workload type selection when clicking a different option', () => {
        modelCatalog.findModelCatalogDetailLink().first().click();
        appChrome.waitForA11y();
        modelCatalog.clickPerformanceInsightsTab();
        modelCatalog.findWorkloadTypeFilter().click();
        modelCatalog.selectWorkloadType('code_fixing');
        // Single-select shows selected value in toggle
        modelCatalog
          .findWorkloadTypeFilter()
          .should('contain.text', 'Scenario')
          .should('contain.text', 'Code Fixing');

        // Re-open dropdown and select a different option
        modelCatalog.findWorkloadTypeFilter().click();
        modelCatalog.selectWorkloadType('chatbot');
        modelCatalog.findWorkloadTypeFilter().should('contain.text', 'Scenario');
        modelCatalog.findWorkloadTypeFilter().should('contain.text', 'Chatbot');
        modelCatalog.findWorkloadTypeFilter().should('not.contain.text', 'Code Fixing');
      });
    });

    describe('Accessibility', () => {
      it('should have proper ARIA attributes for tabs', () => {
        modelCatalog.findModelCatalogDetailLink().first().click();
        appChrome.waitForA11y();

        // Check tab container has proper role
        modelCatalog.findModelDetailsTabs().should('have.attr', 'role', 'region');
        modelCatalog
          .findModelDetailsTabs()
          .should('have.attr', 'aria-label', 'Model details page tabs');

        // Check individual tabs have proper attributes
        modelCatalog.findOverviewTab().should('have.attr', 'aria-label', 'Model overview tab');
        modelCatalog
          .findPerformanceInsightsTab()
          .should('have.attr', 'aria-label', 'Performance insights tab');
      });
    });

    describe('Tab State Management', () => {
      it('should maintain tab state when switching between tabs', () => {
        modelCatalog.findModelCatalogDetailLink().first().click();
        appChrome.waitForA11y();

        cy.url().should('include', '/model-catalog/source-2/validated-model/overview');

        // Switch to Performance Insights
        modelCatalog.clickPerformanceInsightsTab();
        modelCatalog.findPerformanceInsightsTab().should('have.attr', 'aria-selected', 'true');

        // Switch back to Overview
        modelCatalog.clickOverviewTab();
        modelCatalog.findOverviewTab().should('have.attr', 'aria-selected', 'true');

        // Switch to Performance Insights again
        modelCatalog.clickPerformanceInsightsTab();
        modelCatalog.findPerformanceInsightsTab().should('have.attr', 'aria-selected', 'true');
      });
    });
  });

  describe('Validated Models without performance artifacts (without tabs)', () => {
    beforeEach(() => {
      cy.intercept('GET', '/model-registry/api/v1/model_registry*', [
        mockModelRegistry({ name: 'modelregistry-sample' }),
      ]).as('getModelRegistries');

      initIntercepts({ useValidatedModel: true, includePerformanceArtifacts: false });
      modelCatalog.visit();
    });

    it('should not display tabs for validated models without performance artifacts', () => {
      modelCatalog.findLoadingState().should('not.exist');
      modelCatalog.findModelCatalogDetailLink().first().click();
      appChrome.waitForA11y();
      modelCatalog.findModelDetailsTabs().should('not.exist');
      modelCatalog.findOverviewTab().should('not.exist');
      modelCatalog.findPerformanceInsightsTab().should('not.exist');

      modelCatalog.findOverviewTabContent().should('be.visible');
      modelCatalog.findDetailsDescription().should('be.visible');
    });
  });

  describe('Non-Validated Models (without tabs)', () => {
    beforeEach(() => {
      // Mock model registries for register button functionality
      cy.intercept('GET', '/model-registry/api/v1/model_registry*', [
        mockModelRegistry({ name: 'modelregistry-sample' }),
      ]).as('getModelRegistries');

      initIntercepts({ useValidatedModel: false, includePerformanceArtifacts: false });
      modelCatalog.visit();
    });

    it('should not display tabs for non-validated models', () => {
      modelCatalog.findLoadingState().should('not.exist');
      modelCatalog.findModelCatalogDetailLink().first().click();
      appChrome.waitForA11y();

      // Tabs should not be present
      modelCatalog.findModelDetailsTabs().should('not.exist');
      modelCatalog.findOverviewTab().should('not.exist');
      modelCatalog.findPerformanceInsightsTab().should('not.exist');

      // But overview content should still be visible
      modelCatalog.findOverviewTabContent().should('be.visible');
      modelCatalog.findDetailsDescription().should('be.visible');
    });
  });

  describe('Latency Filter Column Visibility', () => {
    beforeEach(() => {
      cy.intercept('GET', '/model-registry/api/v1/model_registry*', [
        mockModelRegistry({ name: 'modelregistry-sample' }),
      ]).as('getModelRegistries');

      initIntercepts({ useValidatedModel: true, includePerformanceArtifacts: true });
      modelCatalog.visit();
      // Enable performance toggle to apply default filters
      modelCatalog.togglePerformanceView();
    });

    describe('Default State (with default latency filter)', () => {
      it('should show only TTFT P90 and TPS P90 columns when default latency filter is applied', () => {
        // Note: Default performance filters are automatically applied when
        // toggle is ON and user navigates to Performance Insights tab

        modelCatalog.findModelCatalogDetailLink().first().click();
        appChrome.waitForA11y();
        modelCatalog.clickPerformanceInsightsTab();

        // TTFT P90 column should be visible (from default filter)
        modelCatalog
          .findHardwareConfigurationTableHeaders()
          .should('contain.text', `TTFT${NBSP}Latency P90`);

        // TPS P90 column should be visible (matching percentile)
        modelCatalog
          .findHardwareConfigurationTableHeaders()
          .should('contain.text', `TPS${NBSP}P90`);
      });
    });

    describe('With Latency Filter Applied', () => {
      it('should show only the selected latency column and matching TPS column when TTFT P90 filter is applied', () => {
        modelCatalog.findModelCatalogDetailLink().first().click();
        appChrome.waitForA11y();
        modelCatalog.clickPerformanceInsightsTab();

        // Open latency filter dropdown
        modelCatalog.openLatencyFilter();

        // Apply the default TTFT P90 filter
        modelCatalog.clickApplyFilter();

        // TTFT P90 column should be visible
        modelCatalog
          .findHardwareConfigurationTableHeaders()
          .should('contain.text', `TTFT${NBSP}Latency P90`);

        // TPS P90 column should be visible (matching percentile)
        modelCatalog
          .findHardwareConfigurationTableHeaders()
          .should('contain.text', `TPS${NBSP}P90`);

        // E2E and ITL columns should be hidden
        modelCatalog.findHardwareConfigurationTableHeaders().should('not.contain.text', 'E2E');
        modelCatalog.findHardwareConfigurationTableHeaders().should('not.contain.text', 'ITL');
      });

      it('should show only E2E mean column and TPS mean column when E2E mean filter is applied', () => {
        modelCatalog.findModelCatalogDetailLink().first().click();
        appChrome.waitForA11y();
        modelCatalog.clickPerformanceInsightsTab();

        // Open latency filter dropdown
        modelCatalog.openLatencyFilter();

        // Change metric to E2E
        modelCatalog.selectLatencyMetric('E2E');

        // Change percentile to Mean
        modelCatalog.selectLatencyPercentile('Mean');

        // Apply filter
        modelCatalog.clickApplyFilter();

        // E2E mean column should be visible
        modelCatalog
          .findHardwareConfigurationTableHeaders()
          .should('contain.text', `E2E${NBSP}Latency Mean`);

        // TPS Mean column should be visible (matching percentile)
        modelCatalog
          .findHardwareConfigurationTableHeaders()
          .should('contain.text', `TPS${NBSP}Mean`);

        // TTFT and ITL columns should be hidden
        modelCatalog.findHardwareConfigurationTableHeaders().should('not.contain.text', 'TTFT');
        modelCatalog.findHardwareConfigurationTableHeaders().should('not.contain.text', 'ITL');
      });

      it('should reset to applied values when changes were made in the dropdown', () => {
        modelCatalog.findModelCatalogDetailLink().first().click();
        appChrome.waitForA11y();
        modelCatalog.clickPerformanceInsightsTab();

        // Apply a non-default filter first (E2E Mean)
        modelCatalog.openLatencyFilter();
        modelCatalog.selectLatencyMetric('E2E');
        modelCatalog.selectLatencyPercentile('Mean');
        modelCatalog.clickApplyFilter();

        // Verify E2E Mean columns are shown
        modelCatalog
          .findHardwareConfigurationTableHeaders()
          .should('contain.text', `E2E${NBSP}Latency Mean`);
        modelCatalog.findHardwareConfigurationTableHeaders().should('not.contain.text', 'TTFT');

        modelCatalog.openLatencyFilter();
        modelCatalog.selectLatencyMetric('TTFT');
        modelCatalog.selectLatencyPercentile('P90');

        // Click reset - should restore to applied values (E2E Mean), not apply filter
        modelCatalog.clickResetFilter();
        cy.get('body').click(0, 0);

        modelCatalog
          .findHardwareConfigurationTableHeaders()
          .should('contain.text', `E2E${NBSP}Latency Mean`);
        modelCatalog.findHardwareConfigurationTableHeaders().should('not.contain.text', 'TTFT');
      });

      it('should keep non-latency columns visible when latency filter is applied', () => {
        modelCatalog.findModelCatalogDetailLink().first().click();
        appChrome.waitForA11y();
        modelCatalog.clickPerformanceInsightsTab();

        // Apply a latency filter
        modelCatalog.openLatencyFilter();
        modelCatalog.clickApplyFilter();

        // Non-latency columns should still be visible
        modelCatalog
          .findHardwareConfigurationTableHeaders()
          .should('contain.text', 'Hardware configuration');
        modelCatalog
          .findHardwareConfigurationTableHeaders()
          .should('contain.text', 'Workload type');
        modelCatalog.findHardwareConfigurationTableHeaders().should('contain.text', 'RPS');
        modelCatalog.findHardwareConfigurationTableHeaders().should('contain.text', 'Replicas');
      });
    });
  });
});

describe('Server-Side Filtering', () => {
  beforeEach(() => {
    cy.intercept('GET', '/model-registry/api/v1/model_registry*', [
      mockModelRegistry({ name: 'modelregistry-sample' }),
    ]).as('getModelRegistries');

    // Use initIntercepts for common setup
    initIntercepts({ useValidatedModel: true, includePerformanceArtifacts: true });
  });

  describe('Filtered Response Handling', () => {
    it('should display only artifacts matching the selected workload type from server response', () => {
      // Initial request returns default-filtered results (chatbot is the default)
      cy.intercept(
        {
          method: 'GET',
          pathname: new RegExp(
            `/api/${MODEL_CATALOG_API_VERSION}/model_catalog/sources/.*/performance_artifacts/.*`,
          ),
        },
        mockModArchResponse(
          mockFilteredPerformanceArtifactsByWorkloadType(UseCaseOptionValue.CHATBOT),
        ),
      ).as('getDefaultFilteredArtifacts');

      modelCatalog.visit();
      modelCatalog.findLoadingState().should('not.exist');
      // Enable performance toggle to apply filters to API requests
      modelCatalog.togglePerformanceView();
      modelCatalog.findModelCatalogDetailLink().first().click();
      appChrome.waitForA11y();
      modelCatalog.clickPerformanceInsightsTab();

      cy.wait('@getDefaultFilteredArtifacts');

      // Verify initial table shows default-filtered results (chatbot)
      modelCatalog.findHardwareConfigurationTableRows().should('have.length', 2);

      // Server returns filtered response when filter is changed to code_fixing
      cy.intercept(
        {
          method: 'GET',
          pathname: new RegExp(
            `/api/${MODEL_CATALOG_API_VERSION}/model_catalog/sources/.*/performance_artifacts/.*`,
          ),
        },
        mockModArchResponse(
          mockFilteredPerformanceArtifactsByWorkloadType(UseCaseOptionValue.CODE_FIXING),
        ),
      ).as('getFilteredPerformanceArtifacts');

      // Apply workload type filter
      modelCatalog.findWorkloadTypeFilter().click();
      modelCatalog.selectWorkloadType('code_fixing');

      cy.wait('@getFilteredPerformanceArtifacts');

      // Verify table shows only filtered results (2 items for code_fixing)
      modelCatalog.findHardwareConfigurationTableRows().should('have.length', 2);

      // Verify all displayed rows have the correct workload type
      modelCatalog.findHardwareConfigurationColumn('Workload type').each(($el) => {
        cy.wrap($el).should('contain.text', 'Code Fixing');
      });
    });

    it('should update table when server returns different filtered results for chatbot', () => {
      // Initial request returns code_fixing results (we'll change default first)
      cy.intercept(
        {
          method: 'GET',
          pathname: new RegExp(
            `/api/${MODEL_CATALOG_API_VERSION}/model_catalog/sources/.*/performance_artifacts/.*`,
          ),
        },
        mockModArchResponse(
          mockFilteredPerformanceArtifactsByWorkloadType(UseCaseOptionValue.CODE_FIXING),
        ),
      ).as('getCodeFixingArtifacts');

      modelCatalog.visit();
      modelCatalog.findLoadingState().should('not.exist');
      // Enable performance toggle
      modelCatalog.togglePerformanceView();
      // Pre-select code_fixing on landing page so it's not the default chatbot
      modelCatalog.findWorkloadTypeFilter().click();
      modelCatalog.selectWorkloadType('code_fixing');
      modelCatalog.findModelCatalogDetailLink().first().click();
      appChrome.waitForA11y();
      modelCatalog.clickPerformanceInsightsTab();

      cy.wait('@getCodeFixingArtifacts');

      // Server returns filtered response for chatbot
      cy.intercept(
        {
          method: 'GET',
          pathname: new RegExp(
            `/api/${MODEL_CATALOG_API_VERSION}/model_catalog/sources/.*/performance_artifacts/.*`,
          ),
        },
        mockModArchResponse(
          mockFilteredPerformanceArtifactsByWorkloadType(UseCaseOptionValue.CHATBOT),
        ),
      ).as('getFilteredChatbotArtifacts');

      // Apply chatbot workload type filter
      modelCatalog.findWorkloadTypeFilter().click();
      modelCatalog.selectWorkloadType('chatbot');

      cy.wait('@getFilteredChatbotArtifacts');

      // Verify table shows filtered results with Chatbot workload type
      modelCatalog.findHardwareConfigurationColumn('Workload type').each(($el) => {
        cy.wrap($el).should('contain.text', 'Chatbot');
      });
    });

    it('should refetch data when workload type filter is changed', () => {
      // Initial request returns default-filtered results (chatbot)
      cy.intercept(
        {
          method: 'GET',
          pathname: new RegExp(
            `/api/${MODEL_CATALOG_API_VERSION}/model_catalog/sources/.*/performance_artifacts/.*`,
          ),
        },
        mockModArchResponse(
          mockFilteredPerformanceArtifactsByWorkloadType(UseCaseOptionValue.CHATBOT),
        ),
      ).as('getDefaultFilteredArtifacts');

      modelCatalog.visit();
      modelCatalog.findLoadingState().should('not.exist');
      // Enable performance toggle
      modelCatalog.togglePerformanceView();
      modelCatalog.findModelCatalogDetailLink().first().click();
      appChrome.waitForA11y();
      modelCatalog.clickPerformanceInsightsTab();

      cy.wait('@getDefaultFilteredArtifacts');
      modelCatalog.findHardwareConfigurationTableRows().should('have.length', 2);

      // Apply filter - server returns filtered response for code_fixing
      cy.intercept(
        {
          method: 'GET',
          pathname: new RegExp(
            `/api/${MODEL_CATALOG_API_VERSION}/model_catalog/sources/.*/performance_artifacts/.*`,
          ),
        },
        mockModArchResponse(
          mockFilteredPerformanceArtifactsByWorkloadType(UseCaseOptionValue.CODE_FIXING),
        ),
      ).as('getFilteredPerformanceArtifacts');

      modelCatalog.findWorkloadTypeFilter().click();
      modelCatalog.selectWorkloadType('code_fixing');
      cy.wait('@getFilteredPerformanceArtifacts');
      modelCatalog.findHardwareConfigurationTableRows().should('have.length', 2);

      // Change filter to long_rag - server returns long_rag-filtered response
      cy.intercept(
        {
          method: 'GET',
          pathname: new RegExp(
            `/api/${MODEL_CATALOG_API_VERSION}/model_catalog/sources/.*/performance_artifacts/.*`,
          ),
        },
        mockModArchResponse(
          mockFilteredPerformanceArtifactsByWorkloadType(UseCaseOptionValue.LONG_RAG),
        ),
      ).as('getFilteredByLongRag');

      // Re-open dropdown and select a different workload type (single-select replaces the value)
      modelCatalog.findWorkloadTypeFilter().click();
      modelCatalog.selectWorkloadType('long_rag');

      cy.wait('@getFilteredByLongRag');

      // Verify table shows long_rag-filtered items
      modelCatalog.findHardwareConfigurationTableRows().should('have.length', 2);
    });
  });
});

describe('Performance Insights Pagination', () => {
  beforeEach(() => {
    cy.intercept('GET', '/model-registry/api/v1/model_registry*', [
      mockModelRegistry({ name: 'modelregistry-sample' }),
    ]).as('getModelRegistries');

    initIntercepts({ useValidatedModel: true, includePerformanceArtifacts: true });
  });

  it('loads next page when clicking Load more', () => {
    cy.intercept(
      {
        method: 'GET',
        pathname: new RegExp(
          `/api/${MODEL_CATALOG_API_VERSION}/model_catalog/sources/.*/performance_artifacts/.*`,
        ),
      },
      (req) => {
        const { nextPageToken } = req.query;
        const pageSize = Number(req.query.pageSize) || 20;
        const total = 25;
        const start = nextPageToken ? Number(nextPageToken) : 0;
        const end = Math.min(start + pageSize, total);

        const items = Array.from({ length: end - start }, (_, i) =>
          mockCatalogPerformanceMetricsArtifact({
            customProperties: {
              config_id: {
                metadataType: ModelRegistryMetadataType.STRING,
                string_value: `paginated-config-${start + i + 1}`,
              },
              hardware_count: {
                metadataType: ModelRegistryMetadataType.INT,
                int_value: `${(start + i + 1) % 8 || 1}`,
              },
              hardware_type: {
                metadataType: ModelRegistryMetadataType.STRING,
                string_value: 'H100-80',
              },
              hardware_configuration: {
                metadataType: ModelRegistryMetadataType.STRING,
                string_value: `${(start + i + 1) % 8 || 1} x H100-80`,
              },
              requests_per_second: {
                metadataType: ModelRegistryMetadataType.DOUBLE,
                double_value: 10 + i,
              },
              ttft_p90: {
                metadataType: ModelRegistryMetadataType.DOUBLE,
                double_value: 40 + i,
              },
              use_case: {
                metadataType: ModelRegistryMetadataType.STRING,
                string_value: UseCaseOptionValue.CHATBOT,
              },
            },
          }),
        );

        req.reply(
          mockModArchResponse({
            items,
            size: total,
            pageSize,
            nextPageToken: end < total ? String(end) : '',
          }),
        );
      },
    ).as('getPaginatedPerformanceArtifacts');

    modelCatalog.visit();
    modelCatalog.findLoadingState().should('not.exist');
    modelCatalog.togglePerformanceView();
    modelCatalog.findModelCatalogDetailLink().first().click();
    appChrome.waitForA11y();
    modelCatalog.clickPerformanceInsightsTab();

    cy.wait('@getPaginatedPerformanceArtifacts');
    modelCatalog.findHardwareConfigurationTableRows().should('have.length', 20);

    modelCatalog.findHardwareConfigLoadMoreButton().click();
    cy.wait('@getPaginatedPerformanceArtifacts');

    modelCatalog.findHardwareConfigurationTableRows().should('have.length', 25);
    modelCatalog.findHardwareConfigLoadMoreButton().should('not.exist');
  });
});
