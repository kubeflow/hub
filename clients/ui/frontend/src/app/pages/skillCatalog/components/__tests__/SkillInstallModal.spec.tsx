import '@testing-library/jest-dom';
import * as React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import type { FetchState } from 'mod-arch-core';
import SkillInstallModal from '~/app/pages/skillCatalog/components/SkillInstallModal';
import { useSkillMarketplace } from '~/app/hooks/skillCatalog/useSkillMarketplace';
import type { Skill, SkillMarketplace, SkillMarketplaceResult } from '~/app/skillCatalogTypes';

jest.mock('~/app/hooks/skillCatalog/useSkillMarketplace', () => ({
  useSkillMarketplace: jest.fn(),
}));

const mockUseSkillMarketplace = jest.mocked(useSkillMarketplace);

const mockSkill: Skill = {
  id: '1',
  name: 'diagnosing-bugs',
  displayName: 'Diagnosing Bugs',
  sourceId: 'matt-pocock-skills',
  repository: 'https://github.com/mattpocock/skills.git',
  path: 'skills/engineering/diagnosing-bugs',
  version: 'v1.2.3',
  trustTier: 'communityContributed',
};

const marketplaceResult = (
  marketplace: SkillMarketplace,
  overrides: Partial<SkillMarketplaceResult> = {},
): SkillMarketplaceResult => ({
  marketplace,
  marketplaceUrl:
    'http://model-catalog.kubeflow.svc.cluster.local:8080/api/skill_catalog/v1/claude/marketplace.json',
  external: false,
  ...overrides,
});

const mockMarketplace: SkillMarketplace = {
  name: 'kubeflow-skill-catalog',
  plugins: [
    {
      name: 'diagnosing-bugs',
      source: {
        url: 'https://github.com/mattpocock/skills.git',
        path: 'skills/engineering/diagnosing-bugs',
        ref: 'v1.2.3',
        sha: 'abc123',
      },
    },
  ],
};

const buildFetchState = <T,>(data: T, loaded: boolean, error?: Error): FetchState<T> => [
  data,
  loaded,
  error,
  jest.fn(),
];

const originalClipboard = navigator.clipboard;

beforeEach(() => {
  Object.defineProperty(navigator, 'clipboard', {
    value: { writeText: jest.fn().mockResolvedValue(undefined) },
    writable: true,
    configurable: true,
  });
  mockUseSkillMarketplace.mockReturnValue(buildFetchState(null, false));
});

afterEach(() => {
  Object.defineProperty(navigator, 'clipboard', {
    value: originalClipboard,
    writable: true,
    configurable: true,
  });
  jest.clearAllMocks();
});

describe('SkillInstallModal', () => {
  it('defaults to the npx tab with a command scoped to this skill via --skill', () => {
    render(<SkillInstallModal skill={mockSkill} isOpen onClose={jest.fn()} />);
    expect(screen.getByTestId('skill-install-modal-npx-1-code-block')).toHaveTextContent(
      'npx skills add https://github.com/mattpocock/skills.git --skill diagnosing-bugs',
    );
  });

  it('renders the manual git clone command on the Manual tab', () => {
    render(<SkillInstallModal skill={mockSkill} isOpen onClose={jest.fn()} />);
    fireEvent.click(screen.getByTestId('skill-install-modal-tab-manual'));
    expect(screen.getByTestId('skill-install-modal-manual-1-code-block')).toHaveTextContent(
      'git clone https://github.com/mattpocock/skills.git',
    );
  });

  it('shows a loading state on the Claude Code tab while the marketplace is fetching', () => {
    render(<SkillInstallModal skill={mockSkill} isOpen onClose={jest.fn()} />);
    fireEvent.click(screen.getByTestId('skill-install-modal-tab-claude-code'));
    expect(screen.getByTestId('skill-install-modal-claude-code-loading-1')).toBeInTheDocument();
  });

  it('builds the Claude Code install command from the real marketplace API response', () => {
    mockUseSkillMarketplace.mockReturnValue(
      buildFetchState(marketplaceResult(mockMarketplace), true),
    );
    render(<SkillInstallModal skill={mockSkill} isOpen onClose={jest.fn()} />);
    fireEvent.click(screen.getByTestId('skill-install-modal-tab-claude-code'));

    const codeBlock = screen.getByTestId('skill-install-modal-claude-code-1-code-block');
    expect(codeBlock).toHaveTextContent('/plugin marketplace add');
    expect(codeBlock).toHaveTextContent('/plugin install diagnosing-bugs@kubeflow-skill-catalog');
  });

  it('shows an unavailable message when the skill has no matching marketplace entry', () => {
    mockUseSkillMarketplace.mockReturnValue(
      buildFetchState(marketplaceResult({ name: 'kubeflow-skill-catalog', plugins: [] }), true),
    );
    render(<SkillInstallModal skill={mockSkill} isOpen onClose={jest.fn()} />);
    fireEvent.click(screen.getByTestId('skill-install-modal-tab-claude-code'));

    expect(screen.getByTestId('skill-install-modal-claude-code-unavailable-1')).toBeInTheDocument();
  });

  it('renders the trust tier as a green label', () => {
    render(<SkillInstallModal skill={mockSkill} isOpen onClose={jest.fn()} />);
    expect(screen.getByTestId('skill-install-modal-trust-tier-1')).toHaveTextContent(
      'Community Contributed',
    );
  });

  it('does not render a trust tier label when absent', () => {
    render(
      <SkillInstallModal
        skill={{ ...mockSkill, trustTier: undefined }}
        isOpen
        onClose={jest.fn()}
      />,
    );
    expect(screen.queryByTestId('skill-install-modal-trust-tier-1')).not.toBeInTheDocument();
  });
});
