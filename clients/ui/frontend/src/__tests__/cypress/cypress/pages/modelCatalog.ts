import { appChrome } from './appChrome';

class ModelCatalogFilter {
  constructor(private title: string) {
    this.title = title;
  }

  find() {
    return cy.findByTestId(`${this.title}-filter`);
  }

  findCheckbox(value: string) {
    return this.find().findByTestId(`${this.title}-${value}-checkbox`);
  }

  findShowMoreButton() {
    return this.find().findByTestId(`${this.title}-filter-show-more`);
  }

  findShowLessButton() {
    return this.find().findByTestId(`${this.title}-filter-show-less`);
  }

  findSearch() {
    return this.find().findByTestId(`${this.title}-filter-search`);
  }

  findEmpty() {
    return this.find().findByTestId(`${this.title}-filter-empty`);
  }
}

class ModelCatalog {
  visit() {
    cy.visit('/model-catalog');
    this.wait();
  }

  private wait() {
    cy.findByTestId('app-page-title').should('exist');
    cy.findByTestId('app-page-title').contains('Model Catalog');
    cy.testA11y();
  }

  waitForPerformanceInsightsTab() {
    this.findPerformanceInsightsTabContent().should('be.visible');
    cy.testA11y();
  }

  findFilter(title: string) {
    return new ModelCatalogFilter(title).find();
  }

  findFilterSearch(title: string) {
    return new ModelCatalogFilter(title).findSearch();
  }

  findFilterEmpty(title: string) {
    return new ModelCatalogFilter(title).findEmpty();
  }

  findFilterShowMoreButton(title: string) {
    return new ModelCatalogFilter(title).findShowMoreButton();
  }

  findFilterShowLessButton(title: string) {
    return new ModelCatalogFilter(title).findShowLessButton();
  }

  findFilterCheckbox(title: string, value: string) {
    return new ModelCatalogFilter(title).findCheckbox(value);
  }

  tabEnabled() {
    appChrome.findNavItem('Model Catalog').should('exist');
    return this;
  }

  findAllModelsToggle() {
    return cy.findByTestId('all');
  }

  findCategoryToggle(category: string) {
    return cy.findByTestId(category);
  }

  findCategoryTitle(category: string) {
    return cy.findByTestId(['title', category]);
  }

  findShowMoreModelsLink(category: string) {
    return cy.findByTestId(['show-more-button', category]);
  }

  findErrorState(category: string) {
    return cy.findByTestId(['error-state', category]);
  }

  findEmptyState(category: string) {
    return cy.findByTestId(['empty-model-catalog-state', category]);
  }

  findModelCatalogEmptyState() {
    return cy.findByTestId('empty-model-catalog-state');
  }

  findModelCatalogCards() {
    return cy.findAllByTestId('model-catalog-card');
  }

  findFirstModelCatalogCard() {
    return this.findModelCatalogCards().first().should('be.visible');
  }

  findModelCatalogDetailLink() {
    return cy.findAllByTestId('model-catalog-detail-link');
  }

  findValidatedModelBenchmarkLink() {
    return cy.findAllByTestId('validated-model-benchmark-link');
  }

  findValidatedModelBenchmarkNext() {
    return cy.findAllByTestId('validated-model-benchmark-next');
  }

  findValidatedModelBenchmarkPrev() {
    return cy.findAllByTestId('validated-model-benchmark-prev');
  }

  findModelCatalogDescription() {
    return cy.findByTestId('model-catalog-card-description');
  }

  findSourceLabel() {
    return cy.get('.pf-v6-c-label');
  }

  findModelLogo() {
    return cy.get('img[alt="model logo"]');
  }

  findTaskLabel() {
    return cy.contains('Text generation');
  }

  findProviderLabel() {
    return cy.contains('provider1');
  }

  findLoadingState() {
    return cy.contains('Loading model catalog...');
  }

  findPageTitle() {
    return cy.contains('Model Catalog');
  }

  findPageDescription() {
    return cy.contains('Discover models that are available for your organization');
  }

  // Details page helpers
  findBreadcrumb() {
    return cy.contains('Model Catalog');
  }

  findDetailsProviderText() {
    return cy.contains('Provided by');
  }

  findDetailsDescription() {
    return cy.findByTestId('model-long-description');
  }

  findModelCardMarkdown() {
    return cy.findByTestId('model-card-markdown');
  }

  findModelArchitecture() {
    return cy.findByTestId('model-architecture');
  }

  findModelType() {
    return cy.findByTestId('model-type');
  }

  // Tabs functionality
  findModelDetailsTabs() {
    return cy.findByTestId('model-details-page-tabs');
  }

  findOverviewTab() {
    return cy.findByTestId('model-overview-tab');
  }

  findPerformanceInsightsTab() {
    return cy.findByTestId('performance-insights-tab');
  }

  findOverviewTabContent() {
    return cy.findByTestId('model-overview-tab-content');
  }

  findPerformanceInsightsTabContent() {
    return cy.findByTestId('performance-insights-tab-content');
  }

  clickOverviewTab() {
    this.findOverviewTab().click();
    return this;
  }

  clickPerformanceInsightsTab() {
    this.findPerformanceInsightsTab().click();
    return this;
  }

  // Hardware Configuration functionality
  findHardwareConfigurationTitle() {
    return cy.contains('Hardware Configuration');
  }

  findHardwareConfigurationDescription() {
    return cy.contains(
      'Compare the performance metrics of hardware configuration to determine the most suitable option for deployment.',
    );
  }

  findHardwareConfigurationTable() {
    return cy.findByTestId('hardware-configuration-table');
  }

  findHardwareConfigurationTableHeaders() {
    return cy.get('[data-testid="hardware-configuration-table"] thead th');
  }

  findHardwareConfigurationTableRows() {
    return cy.get('[data-testid="hardware-configuration-table"] tbody tr');
  }

  findHardwareConfigLoadMoreButton() {
    return cy.findByTestId('hardware-config-load-more-button');
  }

  findHardwareConfigurationColumn(columnName: string) {
    return cy.get(`[data-testid="hardware-configuration-table"] [data-label="${columnName}"]`);
  }

  findValidatedModelHardware() {
    return cy.findByTestId('validated-model-hardware');
  }

  findValidatedModelReplicas() {
    return cy.findByTestId('validated-model-replicas');
  }

  findValidatedModelLatency() {
    return cy.findByTestId('validated-model-latency');
  }

  findWorkloadTypeFilter() {
    return cy.findByTestId('workload-type-filter');
  }

  findWorkloadTypeOption(useCaseValue: string) {
    // WorkloadTypeFilter is now single-select dropdown with data-testid
    return cy.findByTestId(`workload-type-filter-${useCaseValue}`);
  }

  selectWorkloadType(useCaseValue: string) {
    this.findWorkloadTypeOption(useCaseValue).click();
  }

  findPerformanceViewToggle() {
    return cy.pfSwitch('model-performance-view-toggle');
  }

  findPerformanceViewToggleValue() {
    return cy.pfSwitchValue('model-performance-view-toggle');
  }

  togglePerformanceView() {
    this.findPerformanceViewToggle().click();
    return this;
  }

  findPerformanceFiltersUpdatedAlert() {
    return cy.findByTestId('performance-filters-updated-alert');
  }

  findPerformanceFiltersUpdatedAlertCloseButton() {
    return this.findPerformanceFiltersUpdatedAlert().find('button[aria-label^="Close"]');
  }

  dismissPerformanceFiltersUpdatedAlert() {
    this.findPerformanceFiltersUpdatedAlertCloseButton().click();
    return this;
  }

  closeFilterChip(filterKey: string, value: string) {
    cy.get(`[data-testid="${filterKey}-filter-chip-${value}"]`)
      .closest('.pf-v6-c-label')
      .find('button')
      .click();
    return this;
  }

  // Model card content helpers for toggle-based display
  findValidatedModelBenchmarksCount() {
    return cy.findAllByTestId('validated-model-benchmarks');
  }

  // Latency filter helpers
  findLatencyFilter() {
    return cy.findByTestId('latency-filter');
  }

  openLatencyFilter() {
    this.findLatencyFilter().click();
    // Wait for dropdown content to appear
    cy.findByTestId('latency-filter-content').should('be.visible');
    return this;
  }

  findLatencyMetricSelect() {
    return cy.findByTestId('latency-metric-select');
  }

  findLatencyPercentileSelect() {
    return cy.findByTestId('latency-percentile-select');
  }

  selectLatencyMetric(metric: string) {
    this.findLatencyMetricSelect().click();
    // Wait for menu to appear and click the option
    cy.findByTestId('latency-metric-options').contains(metric).click();
    return this;
  }

  selectLatencyPercentile(percentile: string) {
    this.findLatencyPercentileSelect().click();
    // Wait for menu to appear and click the option
    cy.findByTestId('latency-percentile-options').contains(percentile).click();
    return this;
  }

  findApplyFilterButton() {
    return cy.findByTestId('latency-apply-filter');
  }

  findResetFilterButton() {
    return cy.findByTestId('latency-reset-filter');
  }

  findEmptyStateResetFiltersButton() {
    return this.findModelCatalogEmptyState().findByRole('button', {
      name: /Reset all (defaults|filters)/i,
    });
  }

  clickApplyFilter() {
    this.findApplyFilterButton().click();
    return this;
  }

  clickResetFilter() {
    this.findResetFilterButton().click();
    return this;
  }

  // Hardware slider filter helpers (sidebar)
  findMinVramFilter() {
    return cy.findByTestId('minimum-vram-filter');
  }

  findContainerSizeFilter() {
    return cy.findByTestId('container-size-filter');
  }

  // Cold start latency filter helpers
  findColdStartLatencyFilter() {
    return cy.findByTestId('cold-start-load-time-filter');
  }

  openColdStartLatencyFilter() {
    this.findColdStartLatencyFilter().click();
    return this;
  }

  applyColdStartLatencyFilter() {
    cy.findByTestId('cold-start-load-time-apply-filter').click();
    return this;
  }

  resetColdStartLatencyFilter() {
    cy.findByTestId('cold-start-load-time-reset-filter').click();
    return this;
  }

  // Sort dropdown helpers
  findSortDropdown() {
    return cy.findByTestId('model-catalog-sort-dropdown');
  }

  findCategorySortDropdown() {
    return cy.findByTestId('model-catalog-category-sort-dropdown');
  }

  selectSortOption(testId: string) {
    this.findSortDropdown().click();
    cy.findByTestId(testId).click();
    return this;
  }

  // Compression Comparison Card
  findCompressionComparisonCard() {
    return cy.findByTestId('compression-comparison-card');
  }

  findCompressionComparisonLoading() {
    return cy.findByTestId('compression-comparison-loading');
  }

  findCompressionComparisonError() {
    return cy.findByTestId('compression-comparison-error');
  }

  findCompressionComparisonEmpty() {
    return cy.findByTestId('compression-comparison-empty');
  }

  findCompressionVariant(index: number) {
    return cy.findByTestId(`compression-variant-${index}`);
  }

  findCompressionVariantLogo(index: number) {
    return cy.findByTestId(`compression-logo-${index}`);
  }

  findCompressionVariantSkeleton(index: number) {
    return cy.findByTestId(`compression-skeleton-${index}`);
  }

  findCompressionVariantLink(index: number) {
    return cy.findByTestId(`compression-link-${index}`);
  }

  findCompressionTensorType(index: number) {
    return cy.findByTestId(`compression-tensor-type-${index}`);
  }

  findCompressionCurrentModelName() {
    return cy.findByTestId('compression-current-model-name');
  }

  findCompressionCurrentLabel() {
    return cy.findByTestId('compression-current-label');
  }

  findAllCompressionCurrentLabels() {
    return cy.findAllByTestId('compression-current-label');
  }

  findCompressionDivider(index: number) {
    return cy.findByTestId(`compression-divider-${index}`);
  }

  findAllCompressionVariants() {
    return cy.get('[data-testid^="compression-variant-"]');
  }

  // Validated Configurations Card
  findValidatedConfigurationsCard() {
    return cy.findByTestId('validated-configurations-card');
  }

  findToolCallingCard() {
    return cy.findByTestId('tool-calling-card');
  }

  findToolCallingToggle() {
    return cy.get('#tool-calling-toggle');
  }

  findValidatedDeploymentResourceLabels() {
    return cy.findAllByTestId('validated-deployment-resource-label');
  }

  // Performance Empty State
  findPerformanceEmptyState() {
    return cy.findByTestId('performance-empty-state');
  }

  findSetPerformanceOffLink() {
    return this.findPerformanceEmptyState().contains('button', /Turn off model performance view/i);
  }

  findSelectAllModelsCategoryButton() {
    return this.findPerformanceEmptyState().findByRole('button', {
      name: /View all models with performance data/i,
    });
  }

  findRegisterModelButton() {
    return cy.findByTestId('register-model-button');
  }

  findRegisterCatalogModelTooltip() {
    return cy.findByTestId('register-catalog-model-tooltip');
  }

  findModelTypeSelect() {
    return cy.findByTestId('register-model-type-select');
  }

  findManageColumnsButton() {
    return cy.findByTestId('manage-columns-button');
  }

  findManageColumnsModal() {
    return cy.findByTestId('hardware-config-manage-columns');
  }

  findManageColumnsUpdateButton() {
    return cy.findByTestId('hardware-config-manage-columns-update-button');
  }

  findManageColumnsCancelButton() {
    return cy.findByTestId('hardware-config-manage-columns-cancel-button');
  }

  findManageColumnsRestoreDefaults() {
    return cy.findByTestId('hardware-config-manage-columns-restore-defaults');
  }

  findManageColumnsSearch() {
    return cy.findByTestId('hardware-config-manage-columns-search');
  }

  findManageColumnCheckbox(columnLabel: string) {
    return this.findManageColumnsModal().find(`[aria-label="${columnLabel}"]`).scrollIntoView();
  }

  openManageColumnsModal() {
    this.findManageColumnsButton().click();
    this.findManageColumnsModal().should('be.visible');
    return this;
  }
}

export const modelCatalog = new ModelCatalog();
