import React from 'react';
import { screen, render } from '@testing-library/react';
import { userEvent } from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { CatalogSourcePreviewModel, CatalogSourcePreviewSummary } from '~/app/modelCatalogTypes';
import PreviewPanel from '~/app/pages/modelCatalogSettings/components/PreviewPanel';
import { PREVIEW_ALERTS } from '~/app/pages/modelCatalogSettings/constants';
import {
  UseSourcePreviewResult,
  PreviewState,
  PreviewMode,
} from '~/app/pages/modelCatalogSettings/useSourcePreview';
import { CatalogSettingsPreviewTab } from '~/app/shared/catalogSettings/hooks/previewTypes';

const mockSummary: CatalogSourcePreviewSummary = {
  totalModels: 20,
  includedModels: 15,
  excludedModels: 5,
  hasGatedAccessDeniedModels: false,
};

const mockIncludedItems: CatalogSourcePreviewModel[] = [
  { name: 'model-1', included: true },
  { name: 'model-2', included: true },
  { name: 'model-3', included: true },
];

const mockExcludedItems: CatalogSourcePreviewModel[] = [
  { name: 'excluded-model-1', included: false },
  { name: 'excluded-model-2', included: false },
];

const createMockPreviewState = (overrides: Partial<PreviewState> = {}): PreviewState => ({
  mode: PreviewMode.PREVIEW,
  isLoadingInitial: false,
  isLoadingMore: false,
  summary: mockSummary,
  tabStates: {
    [CatalogSettingsPreviewTab.INCLUDED]: { items: mockIncludedItems, hasMore: false },
    [CatalogSettingsPreviewTab.EXCLUDED]: { items: mockExcludedItems, hasMore: false },
  },
  error: undefined,
  resultDismissed: false,
  activeTab: CatalogSettingsPreviewTab.INCLUDED,
  ...overrides,
});

const createMockPreview = (
  overrides: Partial<UseSourcePreviewResult> = {},
  stateOverrides: Partial<PreviewState> = {},
): UseSourcePreviewResult => ({
  previewState: createMockPreviewState(stateOverrides),
  handlePreview: jest.fn(),
  handleTabChange: jest.fn(),
  handleLoadMore: jest.fn(),
  handleValidate: jest.fn(),
  clearValidationSuccess: jest.fn(),
  hasFormChanged: false,
  isValidating: false,
  validationError: undefined,
  isValidationSuccess: false,
  canPreview: true,
  ...overrides,
});

describe('PreviewPanel', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders empty state when no items and no summary', () => {
    const preview = createMockPreview(
      {},
      {
        summary: undefined,
        tabStates: {
          [CatalogSettingsPreviewTab.INCLUDED]: { items: [], hasMore: false },
          [CatalogSettingsPreviewTab.EXCLUDED]: { items: [], hasMore: false },
        },
      },
    );
    render(<PreviewPanel preview={preview} isSourceEnabled />);

    expect(screen.getByText('Preview models')).toBeInTheDocument();
    expect(
      screen.getByText(/To view the models from this source that will appear in the model catalog/),
    ).toBeInTheDocument();
  });

  it('renders loading spinner when isLoadingInitial is true', () => {
    const preview = createMockPreview(
      {},
      {
        isLoadingInitial: true,
        tabStates: {
          [CatalogSettingsPreviewTab.INCLUDED]: { items: [], hasMore: false },
          [CatalogSettingsPreviewTab.EXCLUDED]: { items: [], hasMore: false },
        },
      },
    );
    render(<PreviewPanel preview={preview} isSourceEnabled />);

    expect(screen.getByLabelText('Loading preview')).toBeInTheDocument();
  });

  it('renders error state with retry button when previewError exists', () => {
    const preview = createMockPreview(
      {},
      {
        error: new Error('Failed to fetch preview'),
        mode: PreviewMode.PREVIEW,
        tabStates: {
          [CatalogSettingsPreviewTab.INCLUDED]: { items: [], hasMore: false },
          [CatalogSettingsPreviewTab.EXCLUDED]: { items: [], hasMore: false },
        },
      },
    );
    render(<PreviewPanel preview={preview} isSourceEnabled />);

    expect(screen.getByText('Preview failed')).toBeInTheDocument();
    expect(screen.getByText('Failed to fetch preview')).toBeInTheDocument();
    expect(screen.getByTestId('preview-button-panel-retry')).toBeInTheDocument();
  });

  it('renders error state when preview failed during edit auto-preview', () => {
    const preview = createMockPreview(
      {},
      {
        error: new Error('invalid Hugging Face API credentials'),
        summary: undefined,
        tabStates: {
          [CatalogSettingsPreviewTab.INCLUDED]: { items: [], hasMore: false },
          [CatalogSettingsPreviewTab.EXCLUDED]: { items: [], hasMore: false },
        },
      },
    );
    render(<PreviewPanel preview={preview} isSourceEnabled />);

    expect(screen.getByText('Preview failed')).toBeInTheDocument();
    expect(screen.getByText('invalid Hugging Face API credentials')).toBeInTheDocument();
  });

  it('renders tabs for included/excluded models', () => {
    const preview = createMockPreview();
    render(<PreviewPanel preview={preview} isSourceEnabled />);

    expect(screen.getByText('Models included')).toBeInTheDocument();
    expect(screen.getByText('Models excluded')).toBeInTheDocument();
  });

  it('calls handleTabChange when switching tabs', async () => {
    const user = userEvent.setup();
    const handleTabChange = jest.fn();
    const preview = createMockPreview({ handleTabChange });

    render(<PreviewPanel preview={preview} isSourceEnabled />);

    await user.click(screen.getByText('Models excluded'));

    expect(handleTabChange).toHaveBeenCalledWith(CatalogSettingsPreviewTab.EXCLUDED);
  });

  it('displays correct count text for included tab', () => {
    const preview = createMockPreview({}, { activeTab: CatalogSettingsPreviewTab.INCLUDED });
    render(<PreviewPanel preview={preview} isSourceEnabled />);

    expect(screen.getByText('15 of 20 models included:')).toBeInTheDocument();
  });

  it('displays correct count text for excluded tab', () => {
    const preview = createMockPreview({}, { activeTab: CatalogSettingsPreviewTab.EXCLUDED });
    render(<PreviewPanel preview={preview} isSourceEnabled />);

    expect(screen.getByText('5 of 20 models excluded:')).toBeInTheDocument();
  });

  it('renders Load more button when hasMore is true', () => {
    const preview = createMockPreview(
      {},
      {
        tabStates: {
          [CatalogSettingsPreviewTab.INCLUDED]: { items: mockIncludedItems, hasMore: true },
          [CatalogSettingsPreviewTab.EXCLUDED]: { items: mockExcludedItems, hasMore: false },
        },
      },
    );
    render(<PreviewPanel preview={preview} isSourceEnabled />);

    expect(screen.getByText('Load more')).toBeInTheDocument();
  });

  it('does not render Load more button when hasMore is false', () => {
    const preview = createMockPreview();
    render(<PreviewPanel preview={preview} isSourceEnabled />);

    expect(screen.queryByText('Load more')).not.toBeInTheDocument();
  });

  it('calls handleLoadMore when Load more button clicked', async () => {
    const user = userEvent.setup();
    const handleLoadMore = jest.fn();
    const preview = createMockPreview(
      { handleLoadMore },
      {
        tabStates: {
          [CatalogSettingsPreviewTab.INCLUDED]: { items: mockIncludedItems, hasMore: true },
          [CatalogSettingsPreviewTab.EXCLUDED]: { items: mockExcludedItems, hasMore: false },
        },
      },
    );

    render(<PreviewPanel preview={preview} isSourceEnabled />);

    await user.click(screen.getByText('Load more'));

    expect(handleLoadMore).toHaveBeenCalled();
  });

  it('shows loading state on Load more button when isLoadingMore is true', () => {
    const preview = createMockPreview(
      {},
      {
        isLoadingMore: true,
        tabStates: {
          [CatalogSettingsPreviewTab.INCLUDED]: { items: mockIncludedItems, hasMore: true },
          [CatalogSettingsPreviewTab.EXCLUDED]: { items: mockExcludedItems, hasMore: false },
        },
      },
    );
    render(<PreviewPanel preview={preview} isSourceEnabled />);

    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  it('shows gated access alert when preview summary reports gated models', () => {
    const preview = createMockPreview(
      {},
      {
        summary: { ...mockSummary, hasGatedAccessDeniedModels: true },
        tabStates: {
          [CatalogSettingsPreviewTab.INCLUDED]: {
            items: [{ name: 'org/model-a', included: true }],
            hasMore: false,
          },
          [CatalogSettingsPreviewTab.EXCLUDED]: { items: [], hasMore: false },
        },
      },
    );
    render(<PreviewPanel preview={preview} isSourceEnabled />);

    expect(screen.getByTestId('preview-gated-access-alert')).toBeInTheDocument();
    expect(screen.getByText(PREVIEW_ALERTS.GATED_ACCESS_REQUIRED_TITLE)).toBeInTheDocument();
    expect(screen.getByText(PREVIEW_ALERTS.GATED_ACCESS_REQUIRED_BODY)).toBeInTheDocument();
  });

  it('shows gated access alert when summary has gated models but loaded page items are not gated', () => {
    const preview = createMockPreview(
      {},
      {
        summary: {
          ...mockSummary,
          totalModels: 50,
          includedModels: 40,
          hasGatedAccessDeniedModels: true,
        },
        tabStates: {
          [CatalogSettingsPreviewTab.INCLUDED]: {
            items: [
              { name: 'org/public-model', included: true, hfAccessType: 'public' },
              { name: 'org/private-model', included: true, hfAccessType: 'private' },
            ],
            hasMore: true,
          },
          [CatalogSettingsPreviewTab.EXCLUDED]: { items: [], hasMore: false },
        },
      },
    );
    render(<PreviewPanel preview={preview} isSourceEnabled />);

    expect(screen.getByTestId('preview-gated-access-alert')).toBeInTheDocument();
    expect(screen.queryByLabelText('Gated access warning')).not.toBeInTheDocument();
  });

  it('shows gated access alert when summary has gated models and included tab has no items yet', () => {
    const preview = createMockPreview(
      {},
      {
        summary: {
          ...mockSummary,
          totalModels: 10,
          includedModels: 0,
          hasGatedAccessDeniedModels: true,
        },
        tabStates: {
          [CatalogSettingsPreviewTab.INCLUDED]: { items: [], hasMore: false },
          [CatalogSettingsPreviewTab.EXCLUDED]: {
            items: [{ name: 'org/gated-later', included: false, hfAccessType: 'gated_auto' }],
            hasMore: false,
          },
        },
        activeTab: CatalogSettingsPreviewTab.INCLUDED,
      },
    );
    render(<PreviewPanel preview={preview} isSourceEnabled />);

    expect(screen.getByTestId('preview-gated-access-alert')).toBeInTheDocument();
  });

  it('does not show gated access alert when preview summary has no gated models', () => {
    const preview = createMockPreview(
      {},
      {
        summary: { ...mockSummary, hasGatedAccessDeniedModels: false },
        tabStates: {
          [CatalogSettingsPreviewTab.INCLUDED]: {
            items: [{ name: 'org/model-a', included: true }],
            hasMore: false,
          },
          [CatalogSettingsPreviewTab.EXCLUDED]: { items: [], hasMore: false },
        },
      },
    );
    render(<PreviewPanel preview={preview} isSourceEnabled />);

    expect(screen.queryByTestId('preview-gated-access-alert')).not.toBeInTheDocument();
  });

  it('does not show gated access alert when summary reports no gated-without-access models even if loaded items are gated with access', () => {
    const preview = createMockPreview(
      {},
      {
        summary: { ...mockSummary, hasGatedAccessDeniedModels: false },
        tabStates: {
          [CatalogSettingsPreviewTab.INCLUDED]: {
            items: [
              {
                name: 'org/gated-granted',
                included: true,
                hfAccessType: 'gated_auto',
                hfGatedAccessGranted: true,
              },
              {
                name: 'org/gated-manual-granted',
                included: true,
                hfAccessType: 'gated_manual',
                hfGatedAccessGranted: true,
              },
            ],
            hasMore: false,
          },
          [CatalogSettingsPreviewTab.EXCLUDED]: { items: [], hasMore: false },
        },
      },
    );
    render(<PreviewPanel preview={preview} isSourceEnabled />);

    expect(screen.queryByTestId('preview-gated-access-alert')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Gated access warning')).not.toBeInTheDocument();
    expect(screen.getAllByLabelText('Included model')).toHaveLength(2);
  });

  it('shows refresh alert when hasFormChanged is true', () => {
    const preview = createMockPreview({ hasFormChanged: true });
    render(<PreviewPanel preview={preview} isSourceEnabled />);

    expect(
      screen.getByText('Source configuration changed. Refresh the preview.'),
    ).toBeInTheDocument();
    expect(screen.getByTestId('refresh-preview-link')).toBeInTheDocument();
  });

  it('hides refresh alert when preview is disabled', () => {
    const preview = createMockPreview(
      { hasFormChanged: true, canPreview: false },
      {
        summary: mockSummary,
        tabStates: {
          [CatalogSettingsPreviewTab.INCLUDED]: { items: mockIncludedItems, hasMore: false },
          [CatalogSettingsPreviewTab.EXCLUDED]: { items: mockExcludedItems, hasMore: false },
        },
      },
    );
    render(<PreviewPanel preview={preview} isSourceEnabled />);

    expect(
      screen.queryByText('Source configuration changed. Refresh the preview.'),
    ).not.toBeInTheDocument();
    expect(screen.queryByTestId('refresh-preview-link')).not.toBeInTheDocument();
  });

  it('shows preview disabled tooltip when token validation is required', async () => {
    const user = userEvent.setup();
    const preview = createMockPreview({
      canPreview: false,
      previewDisabledTooltip: 'To preview models, validate the access token.',
    });

    render(<PreviewPanel preview={preview} isSourceEnabled />);

    await user.hover(screen.getByTestId('preview-button-header'));
    expect(
      await screen.findByText('To preview models, validate the access token.'),
    ).toBeInTheDocument();
  });

  it('calls handlePreview when refresh link clicked', async () => {
    const user = userEvent.setup();
    const handlePreview = jest.fn();
    const preview = createMockPreview({ handlePreview, hasFormChanged: true });

    render(<PreviewPanel preview={preview} isSourceEnabled />);

    await user.click(screen.getByTestId('refresh-preview-link'));

    expect(handlePreview).toHaveBeenCalled();
  });

  it('renders empty state for included tab with no items but with summary', () => {
    const preview = createMockPreview(
      {},
      {
        activeTab: CatalogSettingsPreviewTab.INCLUDED,
        summary: { ...mockSummary, includedModels: 0 },
        tabStates: {
          [CatalogSettingsPreviewTab.INCLUDED]: { items: [], hasMore: false },
          [CatalogSettingsPreviewTab.EXCLUDED]: { items: mockExcludedItems, hasMore: false },
        },
      },
    );
    render(<PreviewPanel preview={preview} isSourceEnabled />);

    expect(screen.getByText('No models included')).toBeInTheDocument();
    expect(
      screen.getByText(
        'No models from this source are visible in the model catalog. To include models, edit the model visibility settings of this source.',
      ),
    ).toBeInTheDocument();
  });

  it('renders empty state for excluded tab with no items', () => {
    const preview = createMockPreview(
      {},
      {
        activeTab: CatalogSettingsPreviewTab.EXCLUDED,
        summary: { ...mockSummary, excludedModels: 0 },
        tabStates: {
          [CatalogSettingsPreviewTab.INCLUDED]: { items: mockIncludedItems, hasMore: false },
          [CatalogSettingsPreviewTab.EXCLUDED]: { items: [], hasMore: false },
        },
      },
    );
    render(<PreviewPanel preview={preview} isSourceEnabled />);

    expect(screen.getByText('No models excluded')).toBeInTheDocument();
    expect(
      screen.getByText('No models from this source are excluded by this filter'),
    ).toBeInTheDocument();
  });

  it('renders model items in list', () => {
    const preview = createMockPreview();
    render(<PreviewPanel preview={preview} isSourceEnabled />);

    expect(screen.getByText('model-1')).toBeInTheDocument();
    expect(screen.getByText('model-2')).toBeInTheDocument();
    expect(screen.getByText('model-3')).toBeInTheDocument();
  });

  it('does not render panel body preview button when preview results are shown', () => {
    const preview = createMockPreview(
      {
        canPreview: false,
        hasFormChanged: true,
        previewDisabledTooltip: 'To preview models, validate the access token.',
      },
      {
        summary: mockSummary,
        tabStates: {
          [CatalogSettingsPreviewTab.INCLUDED]: { items: mockIncludedItems, hasMore: false },
          [CatalogSettingsPreviewTab.EXCLUDED]: { items: mockExcludedItems, hasMore: false },
        },
      },
    );
    render(<PreviewPanel preview={preview} isSourceEnabled />);

    expect(screen.queryByTestId('preview-button-panel')).not.toBeInTheDocument();
    expect(screen.getByTestId('preview-button-header')).toBeDisabled();
  });

  it('disables preview button when canPreview is false', () => {
    const preview = createMockPreview(
      { canPreview: false },
      {
        summary: undefined,
        tabStates: {
          [CatalogSettingsPreviewTab.INCLUDED]: { items: [], hasMore: false },
          [CatalogSettingsPreviewTab.EXCLUDED]: { items: [], hasMore: false },
        },
      },
    );
    render(<PreviewPanel preview={preview} isSourceEnabled />);

    const previewButton = screen.getByTestId('preview-button-panel');
    expect(previewButton).toBeDisabled();
  });

  it('shows source disabled warning when source is disabled and preview has loaded', () => {
    const preview = createMockPreview();
    render(<PreviewPanel preview={preview} isSourceEnabled={false} />);

    expect(screen.getByTestId('source-disabled-warning')).toBeInTheDocument();
    expect(screen.getByText(PREVIEW_ALERTS.SOURCE_DISABLED_TITLE)).toBeInTheDocument();
    expect(screen.getByText(PREVIEW_ALERTS.SOURCE_DISABLED_BODY)).toBeInTheDocument();
  });

  it('does not show source disabled warning when source is enabled', () => {
    const preview = createMockPreview();
    render(<PreviewPanel preview={preview} isSourceEnabled />);

    expect(screen.queryByTestId('source-disabled-warning')).not.toBeInTheDocument();
  });

  it('does not show source disabled warning before preview has loaded', () => {
    const preview = createMockPreview(
      {},
      {
        summary: undefined,
        tabStates: {
          [CatalogSettingsPreviewTab.INCLUDED]: { items: [], hasMore: false },
          [CatalogSettingsPreviewTab.EXCLUDED]: { items: [], hasMore: false },
        },
      },
    );
    render(<PreviewPanel preview={preview} isSourceEnabled={false} />);

    expect(screen.queryByTestId('source-disabled-warning')).not.toBeInTheDocument();
  });

  it('does not show source disabled warning when preview has an error', () => {
    const preview = createMockPreview(
      {},
      {
        error: new Error('Failed to fetch preview'),
        tabStates: {
          [CatalogSettingsPreviewTab.INCLUDED]: { items: [], hasMore: false },
          [CatalogSettingsPreviewTab.EXCLUDED]: { items: [], hasMore: false },
        },
      },
    );
    render(<PreviewPanel preview={preview} isSourceEnabled={false} />);

    expect(screen.queryByTestId('source-disabled-warning')).not.toBeInTheDocument();
    expect(screen.getByText('Preview failed')).toBeInTheDocument();
  });
});
