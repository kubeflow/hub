import { act, waitFor } from '@testing-library/react';
import { testHook } from '~/__tests__/unit/testUtils/hooks';
import {
  useMcpSourcePreview,
  McpPreviewTab,
} from '~/app/pages/mcpCatalogSettings/useMcpSourcePreview';
import { ManageMcpSourceFormData } from '~/app/pages/mcpCatalogSettings/useManageMcpSourceData';
import { McpCatalogSourceType } from '~/app/mcpServerCatalogTypes';
import { McpCatalogSettingsAPIState } from '~/app/hooks/mcpCatalogSettings/useMcpCatalogSettingsAPIState';

jest.mock('~/app/pages/mcpCatalogSettings/utils/validation', () => ({
  isPreviewReady: jest.fn(() => true),
  isMcpPreviewReady: jest.fn(() => true),
}));

jest.mock('~/app/pages/mcpCatalogSettings/utils/mcpCatalogSettingsUtils', () => ({
  transformMcpFormDataToConfig: jest.fn(() => ({
    type: 'yaml',
    includedServers: ['*'],
    excludedServers: [],
    yaml: 'source: test\nmcp_servers:\n  - name: Kubernetes',
  })),
}));

const mockFormData: ManageMcpSourceFormData = {
  name: 'Test MCP Source',
  id: 'test-mcp-id',
  sourceType: McpCatalogSourceType.YAML,
  yamlContent: 'source: test\nmcp_servers:\n  - name: Kubernetes',
  includedServers: '',
  excludedServers: '',
  enabled: true,
  isDefault: false,
};

const mockPreviewResult = {
  items: [
    { name: 'Kubernetes', included: true },
    { name: 'GitHub', included: true },
  ],
  summary: {
    totalAssets: 10,
    includedAssets: 8,
    excludedAssets: 2,
  },
  nextPageToken: 'token-123',
};

const createMockApiState = (
  overrides: Partial<McpCatalogSettingsAPIState> = {},
): McpCatalogSettingsAPIState => ({
  apiAvailable: true,
  api: {
    getMcpCatalogSourceConfigs: jest.fn(),
    getMcpCatalogSourceConfig: jest.fn(),
    createMcpCatalogSourceConfig: jest.fn(),
    updateMcpCatalogSourceConfig: jest.fn(),
    deleteMcpCatalogSourceConfig: jest.fn(),
    previewMcpCatalogSource: jest.fn().mockResolvedValue(mockPreviewResult),
  },
  ...overrides,
});

describe('useMcpSourcePreview', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should return initial state', () => {
    const apiState = createMockApiState();

    const { result } = testHook(useMcpSourcePreview)({
      formData: mockFormData,
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    expect(result.current.previewState.isLoadingInitial).toBe(false);
    expect(result.current.previewState.isLoadingMore).toBe(false);
    expect(result.current.previewState.activeTab).toBe(McpPreviewTab.INCLUDED);
    expect(result.current.canPreview).toBe(true);
    expect(result.current.hasFormChanged).toBe(false);
  });

  it('should set error when API is not available', async () => {
    const apiState = createMockApiState({ apiAvailable: false });

    const { result } = testHook(useMcpSourcePreview)({
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

    const { result } = testHook(useMcpSourcePreview)({
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

    expect(apiState.api.previewMcpCatalogSource).toHaveBeenCalledWith(
      {},
      expect.any(Object),
      expect.objectContaining({
        filterStatus: McpPreviewTab.INCLUDED,
        pageSize: 20,
      }),
    );
    expect(result.current.previewState.tabStates[McpPreviewTab.INCLUDED].items).toHaveLength(2);
    expect(result.current.previewState.summary?.totalAssets).toBe(10);
  });

  it('should handle load more and append items', async () => {
    const apiState = createMockApiState();

    const { result } = testHook(useMcpSourcePreview)({
      formData: mockFormData,
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    await act(async () => {
      await result.current.handlePreview();
    });

    (apiState.api.previewMcpCatalogSource as jest.Mock).mockResolvedValueOnce({
      items: [{ name: 'PostgreSQL', included: true }],
      summary: mockPreviewResult.summary,
      nextPageToken: undefined,
    });

    await act(async () => {
      result.current.handleLoadMore();
    });

    await waitFor(() => {
      expect(result.current.previewState.isLoadingMore).toBe(false);
    });

    expect(result.current.previewState.tabStates[McpPreviewTab.INCLUDED].items).toHaveLength(3);
  });

  it('should lazy-load tab when switching to unloaded tab', async () => {
    const apiState = createMockApiState();

    const { result } = testHook(useMcpSourcePreview)({
      formData: mockFormData,
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    await act(async () => {
      await result.current.handlePreview();
    });

    await act(async () => {
      result.current.handleTabChange(McpPreviewTab.EXCLUDED);
    });

    await waitFor(() => {
      expect(result.current.previewState.activeTab).toBe(McpPreviewTab.EXCLUDED);
    });

    expect(apiState.api.previewMcpCatalogSource).toHaveBeenCalledTimes(2);
    expect(apiState.api.previewMcpCatalogSource).toHaveBeenLastCalledWith(
      {},
      expect.any(Object),
      expect.objectContaining({
        filterStatus: McpPreviewTab.EXCLUDED,
      }),
    );
  });

  it('should detect form changes after preview', async () => {
    const apiState = createMockApiState();

    const { result } = testHook(useMcpSourcePreview)({
      formData: mockFormData,
      existingSourceConfig: undefined,
      apiState,
      isEditMode: false,
    });

    await act(async () => {
      await result.current.handlePreview();
    });

    expect(result.current.hasFormChanged).toBe(false);
  });

  it('should handle API errors', async () => {
    const apiState = createMockApiState();
    (apiState.api.previewMcpCatalogSource as jest.Mock).mockRejectedValueOnce(
      new Error('Network error'),
    );

    const { result } = testHook(useMcpSourcePreview)({
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
