import '@testing-library/jest-dom';
import * as React from 'react';
import { render, screen } from '@testing-library/react';
import SkillSourceDetailsSection from '~/app/pages/skillCatalogSettings/components/SkillSourceDetailsSection';
import type { ManageSkillSourceFormData } from '~/app/pages/skillCatalogSettings/useManageSkillSourceData';

const formData: ManageSkillSourceFormData = {
  id: '',
  name: 'Acme Skills',
  enabled: true,
  repositories: [{ url: 'https://github.com/acme/skills' }],
  labels: [],
};

const renderSection = (duplicateNameError?: string) =>
  render(
    <SkillSourceDetailsSection
      formData={formData}
      setData={jest.fn()}
      isEditMode={false}
      duplicateNameError={duplicateNameError}
    />,
  );

describe('SkillSourceDetailsSection duplicate name', () => {
  it('shows the collision on the name field without waiting for a blur', () => {
    // The id is derived from the name and is not editable, so the admin cannot resolve
    // a collision except by renaming — surfacing it late means re-doing the form.
    renderSection('The source "Acme Skills" already uses the identifier "acme_skills".');

    expect(screen.getByTestId('skill-source-name-duplicate')).toHaveTextContent(
      'already uses the identifier "acme_skills"',
    );
  });

  it('shows no duplicate error when the name is free', () => {
    renderSection(undefined);
    expect(screen.queryByTestId('skill-source-name-duplicate')).not.toBeInTheDocument();
  });
});
