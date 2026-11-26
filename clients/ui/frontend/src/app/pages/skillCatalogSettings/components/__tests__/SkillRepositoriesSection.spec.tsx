import '@testing-library/jest-dom';
import * as React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import SkillRepositoriesSection from '~/app/pages/skillCatalogSettings/components/SkillRepositoriesSection';
import type { SkillRepository } from '~/app/skillCatalogTypes';

// Mirrors how the real form holds `repositories` as controlled state fed back in via
// onChange, since that round-trip is what originally broke typing a comma.
const ControlledWrapper: React.FC<{
  onLabelsCommitted: (labels: string[] | undefined) => void;
}> = ({ onLabelsCommitted }) => {
  const [repositories, setRepositories] = React.useState<SkillRepository[]>([{ url: '' }]);
  return (
    <SkillRepositoriesSection
      repositories={repositories}
      onChange={(next) => {
        setRepositories(next);
        onLabelsCommitted(next[0]?.labels);
      }}
    />
  );
};

describe('SkillRepositoriesSection', () => {
  it('preserves repositories it cannot show when the first one is edited', () => {
    // A GitOps-authored source can still carry several repositories. Editing the first
    // used to replace the whole list with one entry, silently deleting the rest.
    const onChange = jest.fn();
    const repositories: SkillRepository[] = [
      { url: 'https://github.com/example/one' },
      { url: 'https://github.com/example/two', credentialRef: 'two' },
      { url: 'https://github.com/example/three' },
    ];
    render(<SkillRepositoriesSection repositories={repositories} onChange={onChange} />);

    fireEvent.change(screen.getByTestId('skill-repo-url-0'), {
      target: { value: 'https://github.com/example/renamed' },
    });

    expect(onChange).toHaveBeenLastCalledWith([
      { url: 'https://github.com/example/renamed' },
      { url: 'https://github.com/example/two', credentialRef: 'two' },
      { url: 'https://github.com/example/three' },
    ]);
  });

  it('tells the admin refs must be immutable and non-empty', () => {
    // The previous copy advertised branches and "leave empty to scan the default
    // branch"; both produce a source the catalog service rejects at sync.
    render(
      <SkillRepositoriesSection
        repositories={[{ url: 'https://github.com/example/one' }]}
        onChange={jest.fn()}
      />,
    );

    expect(screen.getByTestId('skill-repo-refs-0')).toHaveAttribute(
      'placeholder',
      'v1.2.3, 9f8c1a2',
    );
    expect(screen.getByText(/Branches and HEAD are not accepted/)).toBeInTheDocument();
    expect(screen.getByText(/At least one ref is required/)).toBeInTheDocument();
  });

  it('blocks editing when the source lists more repositories than the form supports', () => {
    render(
      <SkillRepositoriesSection
        repositories={[
          { url: 'https://github.com/example/one' },
          { url: 'https://github.com/example/two' },
        ]}
        onChange={jest.fn()}
      />,
    );

    expect(screen.getByTestId('skill-multi-repo-warning')).toHaveTextContent(
      'This source lists 2 repositories and cannot be saved here',
    );
  });

  it('shows no multi-repository alert for a single-repository source', () => {
    render(
      <SkillRepositoriesSection
        repositories={[{ url: 'https://github.com/example/one' }]}
        onChange={jest.fn()}
      />,
    );

    expect(screen.queryByTestId('skill-multi-repo-warning')).not.toBeInTheDocument();
  });

  it('lets a user type a comma to start a second label without it being erased', () => {
    const onLabelsCommitted = jest.fn();
    render(<ControlledWrapper onLabelsCommitted={onLabelsCommitted} />);

    const input = screen.getByTestId('skill-repo-labels-0');

    // Simulate typing character-by-character, including the comma+space that used to get
    // silently stripped by re-deriving the field's text from the parsed array each keystroke.
    fireEvent.change(input, { target: { value: 'typescript' } });
    fireEvent.change(input, { target: { value: 'typescript,' } });
    fireEvent.change(input, { target: { value: 'typescript, ' } });
    fireEvent.change(input, { target: { value: 'typescript, testing' } });

    expect(input).toHaveValue('typescript, testing');
    expect(onLabelsCommitted).toHaveBeenLastCalledWith(['typescript', 'testing']);
  });

  it('parses a fully-typed comma-separated list into a trimmed array on commit', () => {
    const onLabelsCommitted = jest.fn();
    render(<ControlledWrapper onLabelsCommitted={onLabelsCommitted} />);

    const input = screen.getByTestId('skill-repo-labels-0');
    fireEvent.change(input, { target: { value: 'typescript,  testing ,e2e' } });

    expect(onLabelsCommitted).toHaveBeenLastCalledWith(['typescript', 'testing', 'e2e']);
  });
});
