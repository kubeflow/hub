import '@testing-library/jest-dom';
import * as React from 'react';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import SkillCatalogCard from '~/app/pages/skillCatalog/components/SkillCatalogCard';
import type { Skill } from '~/app/skillCatalogTypes';

const wrapper = ({ children }: { children: React.ReactNode }) => (
  <MemoryRouter>{children}</MemoryRouter>
);

const mockSkill: Skill = {
  id: '1',
  name: 'diagnosing-bugs',
  displayName: 'Diagnosing Bugs',
  description: 'Test description for the skill.',
  sourceId: 'matt-pocock-skills',
  repository: 'https://github.com/mattpocock/skills.git',
  path: 'skills/engineering/diagnosing-bugs',
  version: 'v1.2.3',
  labels: ['popular', 'debugging'],
  trustTier: 'communityContributed',
};

describe('SkillCatalogCard', () => {
  it('abbreviates a full commit SHA version but keeps the full ref in the title', () => {
    const sha = '9f8c1a2b3d4e5f60718293a4b5c6d7e8f9012345';
    render(<SkillCatalogCard skill={{ ...mockSkill, version: sha }} />, { wrapper });

    const version = screen.getByTestId('skill-catalog-card-version-1');
    expect(version).toHaveTextContent('9f8c1a2');
    expect(version).toHaveAttribute('title', sha);
  });

  it('renders provider and version under the name', () => {
    render(<SkillCatalogCard skill={{ ...mockSkill, provider: 'Matt Pocock' }} />, { wrapper });
    expect(screen.getByTestId('skill-catalog-card-provider-1')).toHaveTextContent('Matt Pocock');
    expect(screen.getByTestId('skill-catalog-card-version-1')).toHaveTextContent('v1.2.3');
  });

  it('renders version alone when the skill has no provider', () => {
    render(<SkillCatalogCard skill={mockSkill} />, { wrapper });
    expect(screen.queryByTestId('skill-catalog-card-provider-1')).not.toBeInTheDocument();
    expect(screen.getByTestId('skill-catalog-card-version-1')).toHaveTextContent('v1.2.3');
  });

  it('omits the line entirely when neither provider nor version is set', () => {
    render(<SkillCatalogCard skill={{ ...mockSkill, version: undefined }} />, { wrapper });
    expect(screen.queryByTestId('skill-catalog-card-provider-1')).not.toBeInTheDocument();
    expect(screen.queryByTestId('skill-catalog-card-version-1')).not.toBeInTheDocument();
  });

  it('renders skill name and description', () => {
    render(<SkillCatalogCard skill={mockSkill} />, { wrapper });
    expect(screen.getByTestId('skill-catalog-card-name-1')).toHaveTextContent('Diagnosing Bugs');
    expect(screen.getByTestId('skill-catalog-card-description-1')).toHaveTextContent(
      'Test description for the skill.',
    );
  });

  it('renders labels after the description', () => {
    render(<SkillCatalogCard skill={mockSkill} />, { wrapper });
    expect(screen.getByText('popular')).toBeInTheDocument();
    expect(screen.getByText('debugging')).toBeInTheDocument();
  });

  it('does not render labels when none are present', () => {
    render(<SkillCatalogCard skill={{ ...mockSkill, id: '2', labels: undefined }} />, {
      wrapper,
    });
    expect(screen.queryByText('popular')).not.toBeInTheDocument();
  });

  it('always renders the default cube icon', () => {
    render(<SkillCatalogCard skill={mockSkill} />, { wrapper });
    expect(screen.getByTestId('skill-catalog-card-icon-1')).toBeInTheDocument();
  });

  it('renders the trust tier as a green label', () => {
    render(<SkillCatalogCard skill={mockSkill} />, { wrapper });
    const trustTierLabel = screen.getByTestId('skill-catalog-card-trust-tier-1');
    expect(trustTierLabel).toHaveTextContent('Community Contributed');
  });

  it('renders an "Unrated" grey label when trustTier is absent', () => {
    render(<SkillCatalogCard skill={{ ...mockSkill, id: '3', trustTier: undefined }} />, {
      wrapper,
    });
    expect(screen.getByTestId('skill-catalog-card-trust-tier-3')).toHaveTextContent('Unrated');
  });

  it('renders the slash-prefixed skill name in the footer next to the install button', () => {
    render(<SkillCatalogCard skill={mockSkill} />, { wrapper });
    expect(screen.getByTestId('skill-catalog-card-footer-name-1')).toHaveTextContent(
      '/diagnosing-bugs',
    );
  });

  it('renders clickable card name as link to details page', () => {
    render(<SkillCatalogCard skill={mockSkill} />, { wrapper });
    const link = screen.getByTestId('skill-catalog-card-detail-link-1');
    expect(link.tagName).toBe('A');
    expect(link).toHaveAttribute('href', '/skill-catalog/1');
  });
});
