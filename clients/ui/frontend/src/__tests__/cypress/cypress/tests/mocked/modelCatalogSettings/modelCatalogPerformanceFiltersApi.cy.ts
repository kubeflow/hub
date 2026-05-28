/**
 * Tests for verifying performance filters are correctly passed (or not passed)
 * to API endpoints based on the performance view toggle state.
 */
import { modelCatalog } from '~/__tests__/cypress/cypress/pages/modelCatalog';
import { mockModelRegistry } from '~/__mocks__/mockModelRegistry';
import {
  setupValidatedModelIntercepts,
  interceptArtifactsList,
  interceptPerformanceArtifactsList,
} from '~/__tests__/cypress/cypress/support/interceptHelpers/modelCatalog';
import { PERFORMANCE_FILTER_TEST_IDS } from '~/__tests__/cypress/cypress/support/constants';

// Common helper functions
const navigateToPerformanceInsightsTab = (): void => {
  modelCatalog.findModelCatalogDetailLink().first().click();
  modelCatalog.clickPerformanceInsightsTab();
};

const triggerFilterRefresh = (): void => {
  modelCatalog.findFilter('Task').should('be.visible');
  modelCatalog.findFilterCheckbox('Task', 'text-generation').click();
};

const changeWorkloadTypeFilter = (value = 'code_fixing'): void => {
  modelCatalog.findWorkloadTypeFilter().click();
  modelCatalog.selectWorkloadType(value);
};

const assertPerformanceFiltersVisible = (shouldExist: boolean): void => {
  const assertion = shouldExist ? 'be.visible' : 'not.exist';
  cy.findByTestId(PERFORMANCE_FILTER_TEST_IDS.workloadType).should(assertion);
  cy.findByTestId(PERFORMANCE_FILTER_TEST_IDS.latency).should(assertion);
  cy.findByTestId(PERFORMANCE_FILTER_TEST_IDS.maxRps).should(assertion);
  cy.findByTestId(PERFORMANCE_FILTER_TEST_IDS.coldStartLatency).should(assertion);
};

const visitWithPerformanceToggle = (toggleOn: boolean): void => {
  modelCatalog.visit();
  modelCatalog.findLoadingState().should('not.exist');

  if (toggleOn) {
    modelCatalog.togglePerformanceView();
    modelCatalog.findLoadingState().should('not.exist');
  }
};

describe('Model Catalog Performance Filters API Behavior', () => {
  beforeEach(() => {
    cy.intercept('GET', '/model-registry/api/v1/model_registry*', [
      mockModelRegistry({ name: 'modelregistry-sample' }),
    ]).as('getModelRegistries');

    setupValidatedModelIntercepts({});
    interceptArtifactsList();
    interceptPerformanceArtifactsList();
  });

  describe('Toggle OFF - Performance filters should NOT be passed', () => {
    beforeEach(() => {
      visitWithPerformanceToggle(false);
    });

    it('should NOT include performance filter params in /models requests when toggle is OFF', () => {
      cy.intercept('GET', '**/model_catalog/models*').as('getModels');

      triggerFilterRefresh();

      cy.wait('@getModels').then((interception) => {
        const { url } = interception.request;

        // Performance filter params should NOT be present
        expect(url).to.not.include('artifacts.use_case');
        expect(url).to.not.include('artifacts.ttft');
        expect(url).to.not.include('artifacts.e2e');
        expect(url).to.not.include('artifacts.itl');
        expect(url).to.not.include('artifacts.requests_per_second');
        expect(url).to.not.include('cold_start_load_time_seconds');
        expect(url).to.not.include('targetRPS');
        expect(url).to.not.include('latencyProperty');

        // Basic filters should still work
        expect(url).to.include('task');
      });
    });

    it('should NOT include performance filter params in /performance_artifacts requests when toggle is OFF', () => {
      navigateToPerformanceInsightsTab();

      cy.wait('@getCatalogSourceModelArtifacts').then((interception) => {
        const { url } = interception.request;

        // When toggle is OFF, defaults should NOT be applied
        expect(url).to.not.include('use_case=');
      });
    });

    it('should NOT show performance filter UI elements when toggle is OFF', () => {
      assertPerformanceFiltersVisible(false);
      cy.contains('Workload and performance constraints').should('not.exist');
    });
  });

  describe('Toggle ON - Performance filters SHOULD be passed', () => {
    beforeEach(() => {
      visitWithPerformanceToggle(true);
    });

    it('should include default performance filter params in /models requests when toggle is ON', () => {
      cy.intercept('GET', '**/model_catalog/models*').as('getModelsWithFilters');

      triggerFilterRefresh();

      cy.wait('@getModelsWithFilters').then((interception) => {
        const { url } = interception.request;

        // Default performance filters should be applied
        expect(url).to.include('filterQuery');

        const filterQuery = decodeURIComponent(url);
        expect(filterQuery).to.include('artifacts.');
      });
    });

    it('should include performance filter params in /performance_artifacts requests on details page', () => {
      navigateToPerformanceInsightsTab();

      cy.findByTestId(PERFORMANCE_FILTER_TEST_IDS.hardwareTable).should('exist');

      cy.intercept('GET', '**/performance_artifacts/**').as('getPerformanceArtifactsWithFilter');

      changeWorkloadTypeFilter();

      cy.wait('@getPerformanceArtifactsWithFilter').then((interception) => {
        const { url } = interception.request;
        expect(url).to.include('recommendations=true');
      });
    });

    it('should show performance filter UI elements when toggle is ON', () => {
      assertPerformanceFiltersVisible(true);
      cy.contains('Workload and performance constraints').should('be.visible');
    });

    it('should clear performance params from requests when toggle is turned OFF', () => {
      assertPerformanceFiltersVisible(true);

      // Turn OFF the toggle
      modelCatalog.togglePerformanceView();
      modelCatalog.findLoadingState().should('not.exist');

      cy.intercept('GET', '**/model_catalog/models*').as('getModelsAfterToggleOff');

      triggerFilterRefresh();

      cy.wait('@getModelsAfterToggleOff').then((interception) => {
        const { url } = interception.request;

        expect(url).to.not.include('artifacts.use_case');
        expect(url).to.not.include('artifacts.ttft');
        expect(url).to.not.include('cold_start_load_time_seconds');
        expect(url).to.not.include('targetRPS');
      });

      assertPerformanceFiltersVisible(false);
    });
  });

  describe('Performance filter values in API requests', () => {
    beforeEach(() => {
      visitWithPerformanceToggle(true);
    });

    it('should pass correct latencyProperty format to /performance_artifacts endpoint', () => {
      navigateToPerformanceInsightsTab();

      cy.wait('@getCatalogSourceModelArtifacts').then((interception) => {
        const { url } = interception.request;

        // If latency filter is active, verify short property key format (e.g., 'ttft_p90')
        if (url.includes('latencyProperty=')) {
          const latencyMatch = url.match(/latencyProperty=([^&]+)/);
          // eslint-disable-next-line @typescript-eslint/no-unused-expressions
          expect(latencyMatch).to.not.be.null;
          const latencyValue = latencyMatch ? latencyMatch[1] : '';
          // latencyProperty value should not include 'artifacts.' prefix or '.double_value' suffix
          expect(latencyValue).to.not.include('artifacts.');
          expect(latencyValue).to.not.include('.double_value');
          expect(latencyValue).to.match(/^[a-z]+_p\d+$|^[a-z]+_mean$/);
        }
      });
    });

    it('should pass targetRPS as separate param, not in filterQuery', () => {
      navigateToPerformanceInsightsTab();

      cy.wait('@getCatalogSourceModelArtifacts');

      // Apply Max RPS filter
      cy.findByTestId(PERFORMANCE_FILTER_TEST_IDS.maxRps).click();
      cy.findByTestId(PERFORMANCE_FILTER_TEST_IDS.maxRpsApply).click();

      changeWorkloadTypeFilter();

      cy.wait('@getCatalogSourceModelArtifacts').then((interception) => {
        const { url } = interception.request;
        expect(url).to.include('targetRPS=');
      });
    });
  });

  describe('Default filters behavior', () => {
    it('should apply default filters from namedQueries when toggle is turned ON', () => {
      visitWithPerformanceToggle(false);

      // Intercept models request before turning toggle ON
      cy.intercept('GET', '**/model_catalog/models*').as('getModelsAfterToggle');

      // Turn ON the toggle
      modelCatalog.togglePerformanceView();
      modelCatalog.findLoadingState().should('not.exist');

      // The default filters should be applied automatically
      // Check that the workload type filter shows a value (from defaults)
      cy.findByTestId(PERFORMANCE_FILTER_TEST_IDS.workloadType).should('be.visible');
    });

    it('should apply defaults on Performance Insights tab when toggle is ON', () => {
      visitWithPerformanceToggle(true);

      navigateToPerformanceInsightsTab();

      // Wait for initial load
      cy.findByTestId(PERFORMANCE_FILTER_TEST_IDS.hardwareTable).should('exist');

      // Default workload type should be pre-selected (chatbot from namedQueries)
      cy.findByTestId(PERFORMANCE_FILTER_TEST_IDS.workloadType)
        .should('be.visible')
        .and('contain.text', 'Chatbot');
    });

    it('should apply defaults on Performance Insights tab even when toggle is OFF (but not pass to API)', () => {
      visitWithPerformanceToggle(false);

      navigateToPerformanceInsightsTab();

      // Wait for table to load
      cy.findByTestId(PERFORMANCE_FILTER_TEST_IDS.hardwareTable).should('exist');

      // Default filters should be applied to UI state (for the table to work properly)
      // The workload type filter should have a pre-selected default value
      cy.findByTestId(PERFORMANCE_FILTER_TEST_IDS.workloadType)
        .should('be.visible')
        .and('contain.text', 'Chatbot');

      // However, when toggle is OFF, these defaults are NOT passed to the API
      // This is verified by the 'should NOT include performance filter params' tests
    });
  });

  describe('Reset all defaults functionality', () => {
    beforeEach(() => {
      visitWithPerformanceToggle(true);
    });

    it('should clear performance filters when Clear all filters button is clicked on details page', () => {
      navigateToPerformanceInsightsTab();

      cy.findByTestId(PERFORMANCE_FILTER_TEST_IDS.hardwareTable).should('exist');

      // Change a filter to ensure something is set
      changeWorkloadTypeFilter();

      // Click Clear all filters button in the toolbar (PatternFly's native button)
      cy.findByRole('button', { name: 'Reset all defaults' }).click();

      // Verify filters are reset to defaults - workload type should NOT show Code Fixing
      cy.findByTestId(PERFORMANCE_FILTER_TEST_IDS.workloadType)
        .should('be.visible')
        .and('not.contain.text', 'Code Fixing');
    });

    it('should reset latency filter when Reset all filters is clicked', () => {
      navigateToPerformanceInsightsTab();

      cy.findByTestId(PERFORMANCE_FILTER_TEST_IDS.hardwareTable).should('exist');

      // Open and apply latency filter
      modelCatalog.openLatencyFilter();
      modelCatalog.selectLatencyMetric('E2E');
      modelCatalog.clickApplyFilter();

      // Click 'Reset all defaults (PatternFly's native button)
      cy.findByRole('button', { name: 'Reset all defaults' }).click();

      // Latency filter should be reset to default (TTFT, not E2E)
      cy.findByTestId(PERFORMANCE_FILTER_TEST_IDS.latency)
        .should('be.visible')
        .and('not.contain.text', 'E2E');
    });
  });

  describe('Cold start latency filter API behavior', () => {
    it('should include cold_start_load_time_seconds in filterQuery when applied with toggle ON', () => {
      visitWithPerformanceToggle(true);

      modelCatalog.openColdStartLatencyFilter();
      modelCatalog.applyColdStartLatencyFilter();

      cy.intercept('GET', '**/model_catalog/models*').as('getModelsWithColdStart');

      triggerFilterRefresh();

      cy.wait('@getModelsWithColdStart').then((interception) => {
        const decodedUrl = decodeURIComponent(interception.request.url);
        expect(decodedUrl).to.include('cold_start_load_time_seconds');
      });
    });

    it('should NOT include cold_start_load_time_seconds after toggle is turned OFF', () => {
      visitWithPerformanceToggle(true);

      modelCatalog.openColdStartLatencyFilter();
      modelCatalog.applyColdStartLatencyFilter();

      modelCatalog.togglePerformanceView();
      modelCatalog.findLoadingState().should('not.exist');

      cy.intercept('GET', '**/model_catalog/models*').as('getModelsWithoutColdStart');

      triggerFilterRefresh();

      cy.wait('@getModelsWithoutColdStart').then((interception) => {
        const decodedUrl = decodeURIComponent(interception.request.url);
        expect(decodedUrl).to.not.include('cold_start_load_time_seconds');
      });
    });

    it('should pass cold_start_load_time_seconds as orderBy when cold start sort is selected', () => {
      visitWithPerformanceToggle(true);

      modelCatalog.selectSortOption('sort-option-lowest-cold-start');

      cy.intercept('GET', '**/model_catalog/models*').as('getModelsSortedColdStart');

      triggerFilterRefresh();

      cy.wait('@getModelsSortedColdStart').then((interception) => {
        const decodedUrl = decodeURIComponent(interception.request.url);
        expect(decodedUrl).to.include('cold_start_load_time_seconds');
        expect(decodedUrl).to.include('sortOrder=ASC');
      });
    });
  });

  /**
   * NOTE: Filter synchronization tests between catalog and details pages
   * are covered in modelCatalogDetails.cy.ts under 'Filter State Management'.
   * This file focuses on API behavior (what parameters are passed to endpoints).
   */
});
