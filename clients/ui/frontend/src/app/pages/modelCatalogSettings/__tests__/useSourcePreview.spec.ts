import { act, waitFor } from '@testing-library/react';
import { testHook } from '~/__tests__/unit/testUtils/hooks';
import {
  useSourcePreview,
  isPreviewEnabled,
  getPreviewDisabledTooltip,
  UseSourcePreviewOptions,
} from '~/app/pages/modelCatalogSettings/useSourcePreview';
import { ManageSourceFormData } from '~/app/pages/modelCatalogSettings/useManageSourceData';
import { CatalogSourceType } from '~/app/modelCatalogTypes';
import { ModelCatalogSettingsAPIState } from '~/app/hooks/modelCatalogSettings/useModelCatalogSettingsAPIState';
import { CatalogSettingsPreviewTab } from '~/app/shared/catalogSettings/hooks/previewTypes';

// Mock the validation utility
jest.mock('~/app/pages/modelCatalogSettings/utils/validation', () => ({
  isPreviewReady: jest.fn(() => true),
}));

// Mock the transform utility; keep resolveHuggingFaceApiKeyField real for preview payload tests
jest.mock('~/app/pages/modelCatalogSettings/utils/modelCatalogSettingsUtils', () => ({
  ...jest.requireActual('~/app/pages/modelCatalogSettings/utils/modelCatalogSettingsUtils'),
  transformFormDataToConfig: jest.fn((formData: ManageSourceFormData) => {
    if (formData.sourceType === CatalogSourceType.HUGGING_FACE) {
      return {
        type: CatalogSourceType.HUGGING_FACE,
        includedModels: [],
        excludedModels: [],
        allowedOrganization: formData.organization,
        apiKey: formData.accessToken,
      };
    }

    return {
      type: CatalogSourceType.YAML,
      includedModels: ['*'],
      excludedModels: [],
      yaml: formData.yamlContent || 'models:\n  - name: test',
    };
  }),
}));

const mockFormData: ManageSourceFormData = {
  name: 'Test Source',
  id: 'test-id',
  sourceType: CatalogSourceType.YAML,
  accessToken: '',
  organization: '',
  tokenModified: true,
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
    deleteCatalogSourceCredentials: jest.fn(),
    previewCatalogSource: jest.fn().mockResolvedValue(mockPreviewResult),
  },
  ...overrides,
});

const createHookParams = (
  formData: ManageSourceFormData = mockFormData,
  overrides: Partial<Omit<UseSourcePreviewOptions, 'formData'>> & {
    apiState?: ModelCatalogSettingsAPIState;
  } = {},
): UseSourcePreviewOptions => ({
  formData,
  existingSourceConfig: undefined,
  apiState: overrides.apiState ?? createMockApiState(),
  isEditMode: false,
  ...overrides,
});

const hfFormData: ManageSourceFormData = {
  ...mockFormData,
  sourceType: CatalogSourceType.HUGGING_FACE,
  organization: 'qwen',
  accessToken: 'hf_test_token',
};

describe('isPreviewEnabled', () => {
  it('allows preview for HF when organization is valid and token is empty', () => {
    expect(isPreviewEnabled({ ...hfFormData, accessToken: '' }, 'unknown')).toBe(true);
  });

  it('disables preview for HF with token before validation', () => {
    expect(isPreviewEnabled(hfFormData, 'unknown')).toBe(false);
  });

  it('allows preview for HF after successful validation', () => {
    expect(isPreviewEnabled(hfFormData, 'valid')).toBe(true);
  });

  it('disables preview for HF when token validation failed', () => {
    expect(isPreviewEnabled(hfFormData, 'invalid')).toBe(false);
  });

  it('disables preview for HF with token in edit mode until validated', () => {
    expect(isPreviewEnabled(hfFormData, 'unknown')).toBe(false);
    expect(isPreviewEnabled(hfFormData, 'valid')).toBe(true);
  });
});

describe('getPreviewDisabledTooltip', () => {
  it('returns validation tooltip when HF token is present and not validated', () => {
    expect(getPreviewDisabledTooltip(hfFormData, 'unknown')).toBe(
      'Validate the access token to preview models.',
    );
    expect(getPreviewDisabledTooltip(hfFormData, 'invalid')).toBe(
      'Validate the access token to preview models.',
    );
  });

  it('returns undefined when preview is allowed or token is empty', () => {
    expect(
      getPreviewDisabledTooltip({ ...hfFormData, accessToken: '' }, 'unknown'),
    ).toBeUndefined();
    expect(getPreviewDisabledTooltip(hfFormData, 'valid')).toBeUndefined();
  });

  it('returns validation tooltip when HF token is present and not validated', () => {
    expect(getPreviewDisabledTooltip(hfFormData, 'unknown')).toBe(
      'Validate the access token to preview models.',
    );
  });
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

  it('should handle validation without populating preview state', async () => {
    const apiState = createMockApiState();

    const { result } = testHook(useSourcePreview)({
      formData: hfFormData,
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    await act(async () => {
      await result.current.handleValidate();
    });

    await waitFor(() => {
      expect(result.current.isValidating).toBe(false);
    });

    expect(result.current.isValidationSuccess).toBe(true);
    expect(
      result.current.previewState.tabStates[CatalogSettingsPreviewTab.INCLUDED].items,
    ).toHaveLength(0);
    expect(result.current.previewState.summary).toBeUndefined();
    expect(apiState.api.previewCatalogSource).toHaveBeenCalledTimes(1);
  });

  it('should auto-preview on mount in edit mode when saved credentials use secret lookup', async () => {
    const apiState = createMockApiState();
    const editFormData: ManageSourceFormData = {
      ...hfFormData,
      accessToken: '',
      tokenModified: false,
    };

    testHook(useSourcePreview)(
      createHookParams(editFormData, { apiState, isEditMode: true, hasExistingApiKey: true }),
    );

    await waitFor(() => {
      expect(apiState.api.previewCatalogSource).toHaveBeenCalledTimes(1);
    });
  });

  it('should auto-preview in edit mode when API becomes available after mount', async () => {
    const apiState = createMockApiState({ apiAvailable: false });
    const editFormData: ManageSourceFormData = {
      ...hfFormData,
      accessToken: '',
      tokenModified: false,
    };
    const initialParams = createHookParams(editFormData, {
      apiState,
      isEditMode: true,
      hasExistingApiKey: true,
    });

    const renderResult = testHook(useSourcePreview)(initialParams);

    expect(apiState.api.previewCatalogSource).not.toHaveBeenCalled();

    renderResult.rerender(
      createHookParams(editFormData, {
        apiState: { ...apiState, apiAvailable: true },
        isEditMode: true,
        hasExistingApiKey: true,
      }),
    );

    await waitFor(() => {
      expect(apiState.api.previewCatalogSource).toHaveBeenCalledTimes(1);
    });
  });

  it('should enable preview on mount in edit mode without a token in the form field', () => {
    const editFormData: ManageSourceFormData = {
      ...hfFormData,
      accessToken: '',
      tokenModified: false,
    };
    const { result } = testHook(useSourcePreview)(
      createHookParams(editFormData, { isEditMode: true, hasExistingApiKey: true }),
    );

    expect(result.current.canPreview).toBe(true);
    expect(result.current.previewDisabledTooltip).toBeUndefined();
  });

  it('should disable preview in edit mode while typing a new access token', () => {
    const { result } = testHook(useSourcePreview)(
      createHookParams(hfFormData, { isEditMode: true }),
    );

    expect(result.current.canPreview).toBe(false);
    expect(result.current.previewDisabledTooltip).toBe(
      'Validate the access token to preview models.',
    );
  });

  it('should enable preview after successful token validation for HF sources', async () => {
    const apiState = createMockApiState();

    const { result } = testHook(useSourcePreview)({
      formData: hfFormData,
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    expect(result.current.canPreview).toBe(false);

    await act(async () => {
      await result.current.handleValidate();
    });

    await waitFor(() => {
      expect(result.current.canPreview).toBe(true);
    });
  });

  it('should disable preview after failed token validation for HF sources', async () => {
    const apiState = createMockApiState();
    (apiState.api.previewCatalogSource as jest.Mock).mockRejectedValueOnce(
      new Error('invalid Hugging Face API credentials'),
    );

    const { result } = testHook(useSourcePreview)({
      formData: hfFormData,
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    expect(result.current.canPreview).toBe(false);

    await act(async () => {
      await result.current.handleValidate();
    });

    await waitFor(() => {
      expect(result.current.canPreview).toBe(false);
    });

    expect(result.current.validationError?.message).toBe('invalid Hugging Face API credentials');
    expect(
      result.current.previewState.tabStates[CatalogSettingsPreviewTab.INCLUDED].items,
    ).toHaveLength(0);
  });

  it('should re-enable preview when token is cleared after failed validation', async () => {
    const apiState = createMockApiState();
    (apiState.api.previewCatalogSource as jest.Mock).mockRejectedValueOnce(
      new Error('invalid Hugging Face API credentials'),
    );

    const { result, rerender } = testHook(useSourcePreview)({
      formData: hfFormData,
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    await act(async () => {
      await result.current.handleValidate();
    });

    await waitFor(() => {
      expect(result.current.canPreview).toBe(false);
    });

    rerender({
      formData: { ...hfFormData, accessToken: '' },
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    expect(result.current.canPreview).toBe(true);
  });

  it('should handle validation mode', async () => {
    const apiState = createMockApiState();

    const { result } = testHook(useSourcePreview)({
      formData: hfFormData,
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    await act(async () => {
      await result.current.handleValidate();
    });

    await waitFor(() => {
      expect(result.current.isValidating).toBe(false);
    });

    expect(result.current.isValidationSuccess).toBe(true);
  });

  it('should clear validation success', async () => {
    const apiState = createMockApiState();

    const { result } = testHook(useSourcePreview)({
      formData: hfFormData,
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

  it('should keep preview results and require refresh when access token changes', async () => {
    const apiState = createMockApiState();

    const { result, rerender } = testHook(useSourcePreview)({
      formData: { ...hfFormData, accessToken: '' },
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    await act(async () => {
      await result.current.handlePreview();
    });

    await waitFor(() => {
      expect(
        result.current.previewState.tabStates[CatalogSettingsPreviewTab.INCLUDED].items,
      ).toHaveLength(2);
    });

    rerender({
      formData: { ...hfFormData, accessToken: 'new-token' },
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    expect(
      result.current.previewState.tabStates[CatalogSettingsPreviewTab.INCLUDED].items,
    ).toHaveLength(2);
    expect(result.current.previewState.summary?.totalModels).toBe(10);
    expect(result.current.hasFormChanged).toBe(true);
    expect(result.current.canPreview).toBe(false);
  });

  it('should keep preview results and require refresh when access token is cleared', async () => {
    const apiState = createMockApiState();

    const { result, rerender } = testHook(useSourcePreview)({
      formData: hfFormData,
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    await act(async () => {
      await result.current.handleValidate();
    });

    await act(async () => {
      await result.current.handlePreview();
    });

    await waitFor(() => {
      expect(
        result.current.previewState.tabStates[CatalogSettingsPreviewTab.INCLUDED].items,
      ).toHaveLength(2);
    });

    rerender({
      formData: { ...hfFormData, accessToken: '' },
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    expect(
      result.current.previewState.tabStates[CatalogSettingsPreviewTab.INCLUDED].items,
    ).toHaveLength(2);
    expect(result.current.previewState.summary?.totalModels).toBe(10);
    expect(result.current.hasFormChanged).toBe(true);
    expect(result.current.canPreview).toBe(true);
  });

  it('should keep preview results and require refresh when organization changes', async () => {
    const apiState = createMockApiState();

    const { result, rerender } = testHook(useSourcePreview)({
      formData: { ...hfFormData, accessToken: '', organization: 'google' },
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    await act(async () => {
      await result.current.handlePreview();
    });

    await waitFor(() => {
      expect(
        result.current.previewState.tabStates[CatalogSettingsPreviewTab.INCLUDED].items,
      ).toHaveLength(2);
    });

    rerender({
      formData: { ...hfFormData, accessToken: '', organization: 'Googl' },
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    expect(
      result.current.previewState.tabStates[CatalogSettingsPreviewTab.INCLUDED].items,
    ).toHaveLength(2);
    expect(result.current.hasFormChanged).toBe(true);
    expect(result.current.canPreview).toBe(true);
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

  it('should assert stability of handlers when hook props are unchanged', () => {
    const hookParams = createHookParams();
    const renderResult = testHook(useSourcePreview)(hookParams);

    expect(renderResult).hookToHaveUpdateCount(1);

    renderResult.rerender(hookParams);

    expect(renderResult).hookToHaveUpdateCount(2);
    expect(renderResult).hookToBeStable({
      handlePreview: true,
      handleValidate: true,
      handleTabChange: true,
      handleLoadMore: true,
      clearValidationSuccess: true,
      canPreview: true,
      hasFormChanged: true,
      previewDisabledTooltip: true,
    });
  });

  it('should update preview gating when access token changes', () => {
    const apiState = createMockApiState();
    const formDataWithoutToken: ManageSourceFormData = { ...hfFormData, accessToken: '' };
    const initialParams = createHookParams(formDataWithoutToken, { apiState });

    const renderResult = testHook(useSourcePreview)(initialParams);

    expect(renderResult).hookToHaveUpdateCount(1);
    expect(renderResult.result.current.canPreview).toBe(true);
    expect(renderResult.result.current.previewDisabledTooltip).toBeUndefined();

    renderResult.rerender(
      createHookParams({ ...formDataWithoutToken, accessToken: 'new-token' }, { apiState }),
    );

    expect(renderResult).hookToHaveUpdateCount(2);
    expect(renderResult.result.current.canPreview).toBe(false);
    expect(renderResult.result.current.previewDisabledTooltip).toBe(
      'Validate the access token to preview models.',
    );
  });

  it('should reset validation state when credentials change', async () => {
    const apiState = createMockApiState();
    const renderResult = testHook(useSourcePreview)(createHookParams(hfFormData, { apiState }));

    expect(renderResult).hookToHaveUpdateCount(1);

    await act(async () => {
      await renderResult.result.current.handleValidate();
    });

    await waitFor(() => {
      expect(renderResult.result.current.isValidationSuccess).toBe(true);
    });

    const updateCountAfterValidate = renderResult.getUpdateCount();

    renderResult.rerender(
      createHookParams({ ...hfFormData, accessToken: 'updated-token' }, { apiState }),
    );

    expect(renderResult.result.current.isValidationSuccess).toBe(false);
    expect(renderResult.result.current.validationError).toBeUndefined();
    expect(renderResult.getUpdateCount()).toBeGreaterThan(updateCountAfterValidate);
  });
});
