import '@testing-library/jest-dom';
import * as React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import SkillPreviewPanel from '~/app/pages/skillCatalogSettings/components/SkillPreviewPanel';
import { CatalogSettingsPreviewTab } from '~/app/shared/catalogSettings/hooks/previewTypes';
import type {
  UseSkillSourcePreviewResult,
  SkillPreviewState,
} from '~/app/pages/skillCatalogSettings/useSkillSourcePreview';

const emptyTabState = { items: [], hasMore: false };

const buildPreviewState = (overrides: Partial<SkillPreviewState> = {}): SkillPreviewState => ({
  isLoadingInitial: false,
  isLoadingMore: false,
  activeTab: CatalogSettingsPreviewTab.INCLUDED,
  tabStates: {
    [CatalogSettingsPreviewTab.INCLUDED]: emptyTabState,
    [CatalogSettingsPreviewTab.EXCLUDED]: emptyTabState,
  },
  ...overrides,
});

const buildPreview = (
  overrides: Partial<UseSkillSourcePreviewResult> = {},
): UseSkillSourcePreviewResult => ({
  previewState: buildPreviewState(),
  handlePreview: jest.fn(),
  handleTabChange: jest.fn(),
  handleLoadMore: jest.fn(),
  hasFormChanged: false,
  canPreview: true,
  ...overrides,
});

describe('SkillPreviewPanel', () => {
  it('shows friendly guidance — check the URL and provide a token — when preview fails', () => {
    const rawError = new Error(
      "failed to resolve any repository: https://github.com/x/y.git@main: fatal: could not read Username for 'https://github.com': terminal prompts disabled",
    );
    render(
      <SkillPreviewPanel
        preview={buildPreview({
          previewState: buildPreviewState({ error: rawError }),
        })}
      />,
    );

    expect(screen.getByText(/Double-check that the repository URL/)).toBeInTheDocument();
    // Preview strips authToken and the catalog resolves credentials only via a stored
    // credentialRef, so the copy must not send the admin to the Auth token field.
    expect(
      screen.getByText(/save the source, then preview it from the edit form/),
    ).toBeInTheDocument();
    // The raw technical error is available but tucked behind the collapsed expandable
    // section, not shown as the primary message.
    expect(screen.getByText(rawError.message)).not.toBeVisible();
  });

  it('reveals the raw technical error when "Show error details" is expanded', () => {
    const rawError = new Error('fatal: repository not found');
    render(
      <SkillPreviewPanel
        preview={buildPreview({
          previewState: buildPreviewState({ error: rawError }),
        })}
      />,
    );

    fireEvent.click(screen.getByText('Show error details'));
    expect(screen.getByText(rawError.message)).toBeInTheDocument();
  });
});
