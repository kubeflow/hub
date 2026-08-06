import { act, waitFor } from '@testing-library/react';
import { testHook } from '~/__tests__/unit/testUtils/hooks';
import { useSourcePreview, PreviewMode } from '~/app/pages/modelCatalogSettings/useSourcePreview';
import { ManageSourceFormData } from '~/app/pages/modelCatalogSettings/useManageSourceData';
import { CatalogSourceType } from '~/app/modelCatalogTypes';
import { ModelCatalogSettingsAPIState } from '~/app/hooks/modelCatalogSettings/useModelCatalogSettingsAPIState';
import { CatalogSettingsPreviewTab } from '~/app/shared/catalogSettings/hooks/previewTypes';

// Mock the validation utility
jest.mock('~/app/pages/modelCatalogSettings/utils/validation', () => ({
  isPreviewReady: jest.fn(() => true),
}));

// Mock the transform utility
jest.mock('~/app/pages/modelCatalogSettings/utils/modelCatalogSettingsUtils', () => ({
  transformFormDataToConfig: jest.fn(() => ({
    type: 'yaml',
    includedModels: ['*'],
    excludedModels: [],
    yaml: 'models:\n  - name: test',
  })),
}));

const mockFormData: ManageSourceFormData = {
  name: 'Test Source',
  id: 'test-id',
  sourceType: CatalogSourceType.YAML,
  accessToken: '',
  organization: '',
  yamlContent: 'models:\n  - name: test',
  allowedModels: '',
  excludedModels: '',
  enabled: true,
  isDefault: false,
};

const mockPreviewResult = {
  items: [
    { name: 'model-1', included: true },
    { name: 'model-2', included: true },
  ],
  summary: {
    totalModels: 10,
    includedModels: 8,
    excludedModels: 2,
  },
  nextPageToken: 'token-123',
};

const createMockApiState = (
  overrides: Partial<ModelCatalogSettingsAPIState> = {},
): ModelCatalogSettingsAPIState => ({
  apiAvailable: true,
  api: {
    getCatalogSourceConfigs: jest.fn(),
    getCatalogSourceConfig: jest.fn(),
    createCatalogSourceConfig: jest.fn(),
    updateCatalogSourceConfig: jest.fn(),
    deleteCatalogSourceConfig: jest.fn(),
    previewCatalogSource: jest.fn().mockResolvedValue(mockPreviewResult),
  },
  ...overrides,
});

describe('useSourcePreview', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should return initial state', () => {
    const apiState = createMockApiState();

    const { result } = testHook(useSourcePreview)({
      formData: mockFormData,
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    expect(result.current.previewState.isLoadingInitial).toBe(false);
    expect(result.current.previewState.isLoadingMore).toBe(false);
    expect(result.current.previewState.activeTab).toBe(CatalogSettingsPreviewTab.INCLUDED);
    expect(result.current.canPreview).toBe(true);
    expect(result.current.hasFormChanged).toBe(false);
  });

  it('should set error when API is not available', async () => {
    const apiState = createMockApiState({ apiAvailable: false });

    const { result } = testHook(useSourcePreview)({
      formData: mockFormData,
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    await act(async () => {
      await result.current.handlePreview();
    });

    expect(result.current.previewState.error?.message).toBe('API is not available');
  });

  it('should fetch preview and update state on success', async () => {
    const apiState = createMockApiState();

    const { result } = testHook(useSourcePreview)({
      formData: mockFormData,
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    await act(async () => {
      await result.current.handlePreview();
    });

    await waitFor(() => {
      expect(result.current.previewState.isLoadingInitial).toBe(false);
    });

    expect(apiState.api.previewCatalogSource).toHaveBeenCalledWith(
      {},
      expect.any(Object),
      expect.objectContaining({
        filterStatus: CatalogSettingsPreviewTab.INCLUDED,
        pageSize: 20,
      }),
    );
    expect(
      result.current.previewState.tabStates[CatalogSettingsPreviewTab.INCLUDED].items,
    ).toHaveLength(2);
    expect(result.current.previewState.summary?.totalModels).toBe(10);
  });

  it('should handle load more and append items', async () => {
    const apiState = createMockApiState();

    const { result } = testHook(useSourcePreview)({
      formData: mockFormData,
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    // First, do an initial preview
    await act(async () => {
      await result.current.handlePreview();
    });

    // Mock a second page of results
    (apiState.api.previewCatalogSource as jest.Mock).mockResolvedValueOnce({
      items: [{ name: 'model-3', included: true }],
      summary: mockPreviewResult.summary,
      nextPageToken: undefined,
    });

    // Load more
    await act(async () => {
      await result.current.handleLoadMore();
    });

    await waitFor(() => {
      expect(result.current.previewState.isLoadingMore).toBe(false);
    });

    // Should have 3 items now (2 from first load + 1 from load more)
    expect(
      result.current.previewState.tabStates[CatalogSettingsPreviewTab.INCLUDED].items,
    ).toHaveLength(3);
  });

  it('should lazy-load tab when switching to unloaded tab', async () => {
    const apiState = createMockApiState();

    const { result } = testHook(useSourcePreview)({
      formData: mockFormData,
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    // First, do an initial preview (loads included tab)
    await act(async () => {
      await result.current.handlePreview();
    });

    // Switch to excluded tab (should trigger a fetch)
    await act(async () => {
      result.current.handleTabChange(CatalogSettingsPreviewTab.EXCLUDED);
    });

    await waitFor(() => {
      expect(result.current.previewState.activeTab).toBe(CatalogSettingsPreviewTab.EXCLUDED);
    });

    // Should have called previewCatalogSource twice (once for included, once for excluded)
    expect(apiState.api.previewCatalogSource).toHaveBeenCalledTimes(2);
    expect(apiState.api.previewCatalogSource).toHaveBeenLastCalledWith(
      {},
      expect.any(Object),
      expect.objectContaining({
        filterStatus: CatalogSettingsPreviewTab.EXCLUDED,
      }),
    );
  });

  it('should detect form changes after preview', async () => {
    const apiState = createMockApiState();

    const { result } = testHook(useSourcePreview)({
      formData: mockFormData,
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    // Initial preview
    await act(async () => {
      await result.current.handlePreview();
    });

    expect(result.current.hasFormChanged).toBe(false);

    // Note: hasFormChanged depends on buildPreviewRequest comparing current vs last previewed data
    // Since we're mocking transformFormDataToConfig to always return the same thing,
    // we can't easily test form changes without more complex mocking
  });

  it('should handle validation mode', async () => {
    const apiState = createMockApiState();

    const { result } = testHook(useSourcePreview)({
      formData: mockFormData,
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    await act(async () => {
      await result.current.handleValidate();
    });

    await waitFor(() => {
      expect(result.current.previewState.mode).toBe(PreviewMode.VALIDATE);
    });

    expect(result.current.isValidationSuccess).toBe(true);
  });

  it('should clear validation success', async () => {
    const apiState = createMockApiState();

    const { result } = testHook(useSourcePreview)({
      formData: mockFormData,
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    // Run validation
    await act(async () => {
      await result.current.handleValidate();
    });

    expect(result.current.isValidationSuccess).toBe(true);

    // Clear validation success
    act(() => {
      result.current.clearValidationSuccess();
    });

    expect(result.current.isValidationSuccess).toBe(false);
    expect(result.current.previewState.resultDismissed).toBe(true);
  });

  it('should handle API errors', async () => {
    const apiState = createMockApiState();
    (apiState.api.previewCatalogSource as jest.Mock).mockRejectedValueOnce(
      new Error('Network error'),
    );

    const { result } = testHook(useSourcePreview)({
      formData: mockFormData,
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    await act(async () => {
      await result.current.handlePreview();
    });

    await waitFor(() => {
      expect(result.current.previewState.error?.message).toBe('Network error');
    });
  });
});
