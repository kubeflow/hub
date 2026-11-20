import { GenericObjectState } from 'mod-arch-core';
import useGenericObjectState from 'mod-arch-core/dist/utilities/useGenericObjectState';
import type { SkillRepository } from '~/app/skillCatalogTypes';

export type ManageSkillSourceFormData = {
  id: string;
  name: string;
  enabled: boolean;
  repositories: SkillRepository[];
  labels: string[];
};

const defaultFormData: ManageSkillSourceFormData = {
  id: '',
  name: '',
  enabled: true,
  repositories: [{ url: '' }],
  labels: [],
};

export const useManageSkillSourceData = (
  existingData?: Partial<ManageSkillSourceFormData>,
): GenericObjectState<ManageSkillSourceFormData> =>
  useGenericObjectState<ManageSkillSourceFormData>({
    ...defaultFormData,
    ...existingData,
  });
