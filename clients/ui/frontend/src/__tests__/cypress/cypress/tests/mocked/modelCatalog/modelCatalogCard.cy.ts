import { modelCatalog } from '~/__tests__/cypress/cypress/pages/modelCatalog';
import {
  setupModelCatalogIntercepts,
  interceptPerformanceArtifactsWithFilterCheck,
  type ModelCatalogInterceptOptions,
} from '~/__tests__/cypress/cypress/support/interceptHelpers/modelCatalog';
import { PERFORMANCE_FILTER_TEST_IDS } from '~/__tests__/cypress/cypress/support/constants';

/**
 * Initialize intercepts for model catalog card tests.
 * Uses shared intercept helpers to reduce duplication.
 */
const initIntercepts = (options: Partial<ModelCatalogInterceptOptions> = {}) => {
  setupModelCatalogIntercepts({
    includePerformanceArtifacts: options.useValidatedModel ?? false,
    ...options,
  });
};

describe('ModelCatalogCard Component', () => {
  beforeEach(() => {
    initIntercepts({});
    modelCatalog.visit();
  });
  describe('Card Layout and Content', () => {
    it('should render all cards from the mock data', () => {
      modelCatalog.findModelCatalogCards().should('have.length.at.least', 1);
    });

    it('should display correct source labels', () => {
      modelCatalog.findFirstModelCatalogCard().within(() => {
        modelCatalog.findSourceLabel().should('contain.text', 'source 2Text generationprovider1');
      });
    });

    it('should handle cards with logos', () => {
      modelCatalog.findFirstModelCatalogCard().within(() => {
        modelCatalog
          .findModelLogo()
          .should('exist')
          .and('have.attr', 'src')
          .and('include', 'data:image/svg+xml;base64');
      });
    });
  });

  describe('Description Handling', () => {
    it('should display model descriptions', () => {
      modelCatalog.findFirstModelCatalogCard().within(() => {
        modelCatalog
          .findModelCatalogDescription()
          .should(
            'contain.text',
            'Granite-8B-Code-Instruct is a 8B parameter model fine tuned from\nGranite-8B-Code-Base on a combination of permissively licensed instruction\ndata to enhance instruction following capabilities including logical\nreasoning and problem-solving skills.',
          );
      });
    });
  });

  describe('Navigation and Interaction', () => {
    it('should show model metadata correctly', () => {
      modelCatalog.findFirstModelCatalogCard().within(() => {
        // The first card may be from any category section (Sample category 1, Sample category 2, or Community)
        // depending on which section renders first in the DOM
        modelCatalog.findModelCatalogDetailLink().should('exist');
        modelCatalog.findTaskLabel().should('exist');
        modelCatalog.findProviderLabel().should('exist');
      });
    });
  });

  describe('Validated Model', () => {
    describe('Toggle OFF (default)', () => {
      beforeEach(() => {
        initIntercepts({ useValidatedModel: true });
        modelCatalog.visit();
      });

      it('should show description with View benchmarks link when toggle is OFF', () => {
        cy.wait('@getCatalogSourceModelArtifacts');
        modelCatalog.findFirstModelCatalogCard().within(() => {
          // Should show description
          modelCatalog.findModelCatalogDescription().should('be.visible');

          // Should show "View X benchmarks" link
          // Scroll to ensure the link is in view before checking visibility
          modelCatalog.findValidatedModelBenchmarkLink().scrollIntoView();
          modelCatalog.findValidatedModelBenchmarkLink().should('be.visible');
          modelCatalog
            .findValidatedModelBenchmarkLink()
            .should('contain.text', 'View 3 benchmarks');

          // Should NOT show hardware, replicas, TTFT metrics when toggle is OFF
          modelCatalog.findValidatedModelHardware().should('not.exist');
          modelCatalog.findValidatedModelReplicas().should('not.exist');
          modelCatalog.findValidatedModelLatency().should('not.exist');
        });
      });

      it('should navigate to Performance Insights tab when clicking View benchmarks link', () => {
        cy.wait('@getCatalogSourceModelArtifacts');
        modelCatalog.findFirstModelCatalogCard().within(() => {
          modelCatalog.findValidatedModelBenchmarkLink().click();
        });
        cy.url().should('include', 'performance-insights');
      });
    });

    describe('Toggle ON', () => {
      beforeEach(() => {
        initIntercepts({ useValidatedModel: true });
        modelCatalog.visit();
        cy.wait('@getCatalogSourceModelArtifacts');
        // Turn the toggle ON before each test in this block
        modelCatalog.togglePerformanceView();
        // Wait for the page to settle after toggle
        modelCatalog.findLoadingState().should('not.exist');
      });

      it('should show validated model metrics correctly when toggle is ON', () => {
        modelCatalog.findFirstModelCatalogCard().within(() => {
          // Should show hardware, replicas, latency metrics
          // Note: When toggle is ON, default filters set ttft_p90 as the active latency field
          // and values are formatted with "ms" suffix via formatLatency
          modelCatalog.findValidatedModelHardware().should('contain.text', '2 x H100-80');
          modelCatalog.findValidatedModelReplicas().should('contain.text', '7');
          modelCatalog.findValidatedModelLatency().should('contain.text', '51.56 ms');

          // Should NOT show description when toggle is ON
          modelCatalog.findModelCatalogDescription().should('not.exist');

          // Navigate through benchmarks
          modelCatalog.findValidatedModelBenchmarkNext().click();
          modelCatalog.findValidatedModelHardware().should('contain.text', '33 x RTX 4090');
          modelCatalog.findValidatedModelReplicas().should('contain.text', '10');
          modelCatalog.findValidatedModelLatency().should('contain.text', '82.34 ms');

          modelCatalog.findValidatedModelBenchmarkNext().click();
          modelCatalog.findValidatedModelHardware().should('contain.text', '40 x A100');
          modelCatalog.findValidatedModelReplicas().should('contain.text', '15');
          // ttft_p90 value for A100 artifact
          modelCatalog.findValidatedModelLatency().should('contain.text', '58.45 ms');

          modelCatalog.findValidatedModelBenchmarkPrev().click();
          modelCatalog.findValidatedModelHardware().should('contain.text', '33 x RTX 4090');
          modelCatalog.findValidatedModelReplicas().should('contain.text', '10');
          // ttft_p90 value for RTX 4090 artifact
          modelCatalog.findValidatedModelLatency().should('contain.text', '82.34 ms');

          // Click benchmark link to navigate to Performance Insights
          modelCatalog.findValidatedModelBenchmarkLink().click();
        });
        cy.url().should('include', 'performance-insights');
      });

      it('should navigate through benchmarks correctly', () => {
        modelCatalog.findFirstModelCatalogCard().within(() => {
          // Initial state - first benchmark
          modelCatalog.findValidatedModelHardware().should('contain.text', '2 x H100-80');

          // Navigate to next benchmark
          modelCatalog.findValidatedModelBenchmarkNext().click();
          modelCatalog.findValidatedModelHardware().should('contain.text', '33 x RTX 4090');

          // Navigate to next benchmark
          modelCatalog.findValidatedModelBenchmarkNext().click();
          modelCatalog.findValidatedModelHardware().should('contain.text', '40 x A100');

          // Navigate back
          modelCatalog.findValidatedModelBenchmarkPrev().click();
          modelCatalog.findValidatedModelHardware().should('contain.text', '33 x RTX 4090');
        });
      });
    });

    describe('Benchmark count consistency', () => {
      it('should show same benchmark count with toggle OFF as with toggle ON', () => {
        // Use dynamic mock that returns different counts based on filterQuery presence
        // Returns 3 artifacts when filterQuery present, 5 when absent
        setupModelCatalogIntercepts({ useValidatedModel: true });
        interceptPerformanceArtifactsWithFilterCheck();

        modelCatalog.visit();

        // Turn toggle ON to apply default filters
        modelCatalog.togglePerformanceView();
        modelCatalog.findLoadingState().should('not.exist');

        // Verify benchmark count with toggle ON (shows "X of 3 benchmarks")
        cy.wait('@getCatalogSourceModelArtifacts');
        modelCatalog.findFirstModelCatalogCard().within(() => {
          modelCatalog.findValidatedModelBenchmarksCount().should('contain.text', '3 benchmarks');
        });

        // Turn toggle OFF - should still show same count (filters still passed)
        modelCatalog.togglePerformanceView();
        modelCatalog.findLoadingState().should('not.exist');

        // Verify benchmark count with toggle OFF (shows "View 3 benchmarks")
        cy.wait('@getCatalogSourceModelArtifacts');
        modelCatalog.findFirstModelCatalogCard().within(() => {
          modelCatalog
            .findValidatedModelBenchmarkLink()
            .should('contain.text', 'View 3 benchmarks');
        });
      });
    });
  });

  /**
   * NOTE: Detailed latency filter interactions, workload type filter options,
   * and filter reset behavior are comprehensively tested in modelCatalogTabs.cy.ts
   * (on the Performance Insights tab). These tests focus on card-specific behavior.
   */
  describe('Performance Filters on Catalog Landing Page', () => {
    beforeEach(() => {
      initIntercepts({ useValidatedModel: true });
      modelCatalog.visit();
      cy.wait('@getCatalogSourceModelArtifacts');
      modelCatalog.togglePerformanceView();
      modelCatalog.findLoadingState().should('not.exist');
    });

    it('should display performance filters on catalog page when toggle is ON', () => {
      cy.findByTestId(PERFORMANCE_FILTER_TEST_IDS.workloadType).should('be.visible');
      cy.findByTestId(PERFORMANCE_FILTER_TEST_IDS.latency).should('be.visible');
      cy.findByTestId(PERFORMANCE_FILTER_TEST_IDS.maxRps).should('be.visible');
    });

    it('should update card latency display when latency filter changes', () => {
      // Card should show latency value
      modelCatalog.findFirstModelCatalogCard().within(() => {
        modelCatalog.findValidatedModelLatency().should('be.visible');
      });

      // Change latency filter
      modelCatalog.openLatencyFilter();
      modelCatalog.selectLatencyMetric('E2E');
      modelCatalog.clickApplyFilter();

      // Card should still show latency value (updated based on filter)
      modelCatalog.findFirstModelCatalogCard().within(() => {
        modelCatalog.findValidatedModelLatency().should('be.visible');
      });
    });
  });
});
