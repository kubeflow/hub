import '@testing-library/jest-dom';
import * as React from 'react';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import SkillDetailsView from '~/app/pages/skillCatalog/screens/SkillDetailsView';
import type { Skill } from '~/app/skillCatalogTypes';

// react-markdown ships ESM that jest does not transform, so the renderer is stubbed.
// Nothing here asserts on README content.
jest.mock('~/app/shared/markdown/MarkdownComponent', () => ({
  __esModule: true,
  default: () => null,
}));

const wrapper = ({ children }: { children: React.ReactNode }) => (
  <MemoryRouter>{children}</MemoryRouter>
);

const baseSkill: Skill = {
  id: '1',
  name: 'diagnosing-bugs',
  displayName: 'Diagnosing Bugs',
  description: 'Test description.',
  repository: 'https://github.com/mattpocock/skills.git',
  path: 'skills/engineering/diagnosing-bugs',
  version: 'v1.2.3',
};

describe('SkillDetailsView skill length', () => {
  it('shows the line count in green for a body within the recommended maximum', () => {
    render(<SkillDetailsView skill={{ ...baseSkill, bodyLineCount: 120 }} />, { wrapper });

    const label = screen.getByTestId('skill-body-line-count');
    expect(label).toHaveTextContent('120 lines');
    expect(label).toHaveClass('pf-m-green');
  });

  it('shows the line count in orange once it exceeds the recommended maximum', () => {
    render(<SkillDetailsView skill={{ ...baseSkill, bodyLineCount: 1240 }} />, { wrapper });

    const label = screen.getByTestId('skill-body-line-count');
    expect(label).toHaveTextContent('1240 lines');
    expect(label).toHaveClass('pf-m-orange');
  });

  it('omits the row entirely when the catalog did not report a line count', () => {
    render(<SkillDetailsView skill={baseSkill} />, { wrapper });
    expect(screen.queryByTestId('skill-body-line-count')).not.toBeInTheDocument();
  });
});
