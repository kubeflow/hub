import { modelCatalog } from '~/__tests__/cypress/cypress/pages/modelCatalog';
import { mockModelRegistry } from '~/__mocks__/mockModelRegistry';
import {
  setupValidatedModelIntercepts,
  interceptArtifactsList,
  interceptPerformanceArtifactsList,
  type ModelCatalogInterceptOptions,
} from '~/__tests__/cypress/cypress/support/interceptHelpers/modelCatalog';

/**
 * Initialize intercepts for performance filters alert tests.
 * Uses shared intercept helpers to reduce duplication.
 */
const initIntercepts = (options: Partial<ModelCatalogInterceptOptions> = {}) => {
  setupValidatedModelIntercepts(options);

  // Additional intercepts needed for Performance Insights tab:
  // - /artifacts/ endpoint is used to determine if tabs should show
  // - /performance_artifacts/ with regex for flexible matching
  interceptArtifactsList();
  interceptPerformanceArtifactsList();
};

describe('Model Catalog Performance Filters Alert', () => {
  beforeEach(() => {
    cy.intercept('GET', '/model-registry/api/v1/model_registry*', [
      mockModelRegistry({ name: 'modelregistry-sample' }),
    ]).as('getModelRegistries');

    initIntercepts({});
    modelCatalog.visit();
  });

  describe('Alert Display Logic', () => {
    it('should not show alert initially', () => {
      modelCatalog.findLoadingState().should('not.exist');
      modelCatalog.findPerformanceFiltersUpdatedAlert().should('not.exist');
    });

    it('should not show alert when performance view is disabled', () => {
      modelCatalog.findModelCatalogDetailLink().first().click();
      modelCatalog.clickPerformanceInsightsTab();

      modelCatalog.findWorkloadTypeFilter().click();
      modelCatalog.selectWorkloadType('code_fixing');

      cy.go('back');
      cy.go('back');
      modelCatalog.findLoadingState().should('not.exist');

      modelCatalog.findPerformanceFiltersUpdatedAlert().should('not.exist');
    });

    it('should show alert when returning from details page after changing performance filters', () => {
      modelCatalog.togglePerformanceView();
      modelCatalog.findPerformanceViewToggleValue().should('be.checked');
      // Wait for the models to reload after toggle applies default filters
      modelCatalog.findLoadingState().should('not.exist');

      modelCatalog.findModelCatalogDetailLink().first().click();
      modelCatalog.clickPerformanceInsightsTab();

      modelCatalog.findWorkloadTypeFilter().click();
      modelCatalog.selectWorkloadType('code_fixing');

      cy.go('back');
      cy.go('back');
      modelCatalog.findLoadingState().should('not.exist');

      modelCatalog.findPerformanceFiltersUpdatedAlert().should('be.visible');
      modelCatalog
        .findPerformanceFiltersUpdatedAlert()
        .should(
          'include.text',
          'The performance constraints and results have been updated to match the constraints you applied to the',
        );
    });

    it('should not show alert when no filters were changed on details page', () => {
      modelCatalog.togglePerformanceView();
      modelCatalog.findLoadingState().should('not.exist');

      modelCatalog.findModelCatalogDetailLink().first().click();
      modelCatalog.clickPerformanceInsightsTab();

      cy.go('back');
      cy.go('back');
      modelCatalog.findLoadingState().should('not.exist');

      modelCatalog.findPerformanceFiltersUpdatedAlert().should('not.exist');
    });
  });

  describe('Alert Dismissal', () => {
    it('should dismiss alert when close button is clicked', () => {
      modelCatalog.togglePerformanceView();
      modelCatalog.findLoadingState().should('not.exist');

      modelCatalog.findModelCatalogDetailLink().first().click();
      modelCatalog.clickPerformanceInsightsTab();

      modelCatalog.findWorkloadTypeFilter().click();
      modelCatalog.selectWorkloadType('code_fixing');

      cy.go('back');
      cy.go('back');
      modelCatalog.findLoadingState().should('not.exist');

      modelCatalog.findPerformanceFiltersUpdatedAlert().should('be.visible');

      modelCatalog.dismissPerformanceFiltersUpdatedAlert();

      modelCatalog.findPerformanceFiltersUpdatedAlert().should('not.exist');
    });
  });

  describe('Alert Hidden Scenarios', () => {
    it('should hide alert when performance toggle is turned OFF', () => {
      modelCatalog.togglePerformanceView();
      modelCatalog.findLoadingState().should('not.exist');

      modelCatalog.findModelCatalogDetailLink().first().click();
      modelCatalog.clickPerformanceInsightsTab();

      modelCatalog.findWorkloadTypeFilter().click();
      modelCatalog.selectWorkloadType('code_fixing');

      cy.go('back');
      cy.go('back');
      modelCatalog.findLoadingState().should('not.exist');

      modelCatalog.findPerformanceFiltersUpdatedAlert().should('be.visible');

      modelCatalog.togglePerformanceView();

      modelCatalog.findPerformanceFiltersUpdatedAlert().should('not.exist');
    });

    it('should hide alert when filters change on catalog page', () => {
      modelCatalog.togglePerformanceView();
      modelCatalog.findLoadingState().should('not.exist');

      modelCatalog.findModelCatalogDetailLink().first().click();
      modelCatalog.clickPerformanceInsightsTab();

      modelCatalog.findWorkloadTypeFilter().click();
      modelCatalog.selectWorkloadType('code_fixing');

      cy.go('back');
      cy.go('back');
      modelCatalog.findLoadingState().should('not.exist');

      modelCatalog.findPerformanceFiltersUpdatedAlert().should('be.visible');

      modelCatalog.findFilter('Task').should('be.visible');
      modelCatalog.findFilterCheckbox('Task', 'text-generation').click();

      modelCatalog.findPerformanceFiltersUpdatedAlert().should('not.exist');
    });

    it('should hide alert when Reset all filters is clicked on catalog page', () => {
      modelCatalog.togglePerformanceView();
      modelCatalog.findLoadingState().should('not.exist');

      modelCatalog.findModelCatalogDetailLink().first().click();
      modelCatalog.clickPerformanceInsightsTab();

      modelCatalog.findWorkloadTypeFilter().click();
      modelCatalog.selectWorkloadType('code_fixing');

      cy.go('back');
      cy.go('back');
      modelCatalog.findLoadingState().should('not.exist');

      modelCatalog.findPerformanceFiltersUpdatedAlert().should('be.visible');

      cy.findByRole('button', { name: 'Reset all filters' }).click();

      modelCatalog.findPerformanceFiltersUpdatedAlert().should('not.exist');
    });

    it('should hide alert when a performance filter chip is removed on catalog page', () => {
      modelCatalog.togglePerformanceView();
      modelCatalog.findLoadingState().should('not.exist');

      modelCatalog.findModelCatalogDetailLink().first().click();
      modelCatalog.clickPerformanceInsightsTab();

      modelCatalog.findWorkloadTypeFilter().click();
      modelCatalog.selectWorkloadType('code_fixing');

      cy.go('back');
      cy.go('back');
      modelCatalog.findLoadingState().should('not.exist');

      modelCatalog.findPerformanceFiltersUpdatedAlert().should('be.visible');

      modelCatalog.closeFilterChip('artifacts.use_case.string_value', 'code_fixing');

      modelCatalog.findPerformanceFiltersUpdatedAlert().should('not.exist');
    });
  });

  describe('Multiple Filter Changes', () => {
    it('should show alert after changing multiple performance filters', () => {
      modelCatalog.togglePerformanceView();
      modelCatalog.findLoadingState().should('not.exist');

      modelCatalog.findModelCatalogDetailLink().first().click();
      modelCatalog.clickPerformanceInsightsTab();

      modelCatalog.findWorkloadTypeFilter().click();
      modelCatalog.selectWorkloadType('code_fixing');
      // Re-open dropdown to select second option
      modelCatalog.findWorkloadTypeFilter().click();
      modelCatalog.selectWorkloadType('chatbot');

      cy.go('back');
      cy.go('back');
      modelCatalog.findLoadingState().should('not.exist');

      modelCatalog.findPerformanceFiltersUpdatedAlert().should('be.visible');
    });
  });
});
