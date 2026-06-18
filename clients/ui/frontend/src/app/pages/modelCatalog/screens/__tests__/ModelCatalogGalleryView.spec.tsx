import '@testing-library/jest-dom';
import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import type { ModelCatalogContextType } from '~/app/context/modelCatalog/ModelCatalogContext';
import { useCatalogModelsBySources } from '~/app/hooks/modelCatalog/useCatalogModelsBySource';
import { CategoryName } from '~/app/modelCatalogTypes';
import {
  ModelCatalogStringFilterKey,
  ModelCatalogNumberFilterKey,
} from '~/concepts/modelCatalog/const';
import ModelCatalogGalleryView from '~/app/pages/modelCatalog/screens/ModelCatalogGalleryView';

jest.mock('~/app/hooks/modelCatalog/useCatalogModelsBySource', () => ({
  useCatalogModelsBySources: jest.fn(),
}));

const mockUseCatalogModelsBySources = jest.mocked(useCatalogModelsBySources);

const INITIAL_FILTERS = {
  [ModelCatalogStringFilterKey.TASK]: [],
  [ModelCatalogStringFilterKey.PROVIDER]: [],
  [ModelCatalogStringFilterKey.LICENSE]: [],
  [ModelCatalogStringFilterKey.LANGUAGE]: [],
  [ModelCatalogStringFilterKey.HARDWARE_TYPE]: [],
  [ModelCatalogStringFilterKey.HARDWARE_CONFIGURATION]: [],
  [ModelCatalogStringFilterKey.USE_CASE]: [],
  [ModelCatalogNumberFilterKey.MAX_RPS]: undefined,
  [ModelCatalogNumberFilterKey.COLD_START_LOAD_TIME]: undefined,
  [ModelCatalogNumberFilterKey.MIN_VRAM]: undefined,
  [ModelCatalogNumberFilterKey.IMAGE_SIZE]: undefined,
  [ModelCatalogStringFilterKey.TENSOR_TYPE]: [],
  [ModelCatalogStringFilterKey.VALIDATED_CONFIGURATION]: [],
} as ModelCatalogContextType['filters'];

const defaultContext: ModelCatalogContextType = {
  catalogSources: null,
  catalogSourcesLoaded: true,
  catalogSourcesLoadError: undefined,
  catalogLabels: null,
  catalogLabelsLoaded: true,
  catalogLabelsLoadError: undefined,
  filterOptions: null,
  filterOptionsLoaded: true,
  filterOptionsLoadError: undefined,
  selectedSourceLabel: CategoryName.allModels,
  setSelectedSourceLabel: jest.fn(),
  clearAllFilters: jest.fn(),
  selectedSource: undefined,
  updateSelectedSource: jest.fn(),
  apiState: {} as ModelCatalogContextType['apiState'],
  refreshAPIState: jest.fn(),
  filters: { ...INITIAL_FILTERS },
  setFilters: jest.fn(),
  performanceViewEnabled: false,
  setPerformanceViewEnabled: jest.fn(),
  performanceFiltersChangedOnDetailsPage: false,
  setPerformanceFiltersChangedOnDetailsPage: jest.fn(),
  lastViewedModelName: null,
  setLastViewedModelName: jest.fn(),
  resetPerformanceFiltersToDefaults: jest.fn(),
  resetSinglePerformanceFilterToDefault: jest.fn(),
  getPerformanceFilterDefaultValue: jest.fn().mockReturnValue(undefined),
  sortBy: null,
  setSortBy: jest.fn(),
};

const defaultHookResult = {
  catalogModels: {
    items: [] as never[],
    size: 0,
    pageSize: 10,
    nextPageToken: '',
    loadMore: jest.fn(),
    isLoadingMore: false,
    hasMore: false,
    refresh: jest.fn(),
  },
  catalogModelsLoaded: true,
  catalogModelsLoadError: undefined,
  refresh: jest.fn(),
};

const handleFilterReset = jest.fn();

const renderWithContext = (contextOverrides: Partial<ModelCatalogContextType> = {}) => {
  mockUseCatalogModelsBySources.mockReturnValue(
    defaultHookResult as ReturnType<typeof useCatalogModelsBySources>,
  );
  const ctx = { ...defaultContext, ...contextOverrides };
  return render(
    <MemoryRouter>
      <ModelCatalogContext.Provider value={ctx}>
        <ModelCatalogGalleryView searchTerm="test" handleFilterReset={handleFilterReset} />
      </ModelCatalogContext.Provider>
    </MemoryRouter>,
  );
};

describe('ModelCatalogGalleryView', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    Element.prototype.scrollTo = jest.fn();
  });

  describe('empty state reset button label', () => {
    it('should show "Reset all defaults" when performance view is disabled', () => {
      renderWithContext();
      expect(screen.getByText('Reset all defaults')).toBeInTheDocument();
    });

    it('should show "Reset all defaults" when performance view is enabled', () => {
      renderWithContext({ performanceViewEnabled: true });
      expect(screen.getByText('Reset all defaults')).toBeInTheDocument();
    });

    it('should call handleFilterReset when reset button is clicked', () => {
      renderWithContext();
      fireEvent.click(screen.getByText('Reset all defaults'));
      expect(handleFilterReset).toHaveBeenCalledTimes(1);
    });
  });
});
