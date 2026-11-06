export const SKILL_CATALOG_SETTINGS_PAGE_TITLE = 'Skill catalog sources';
export const SKILL_CATALOG_SETTINGS_DESCRIPTION =
  'Add and manage Git repositories that populate the skill catalog for users in your organization.';

export const SKILL_ADD_SOURCE_TITLE = 'Add source';
export const SKILL_ADD_SOURCE_DESCRIPTION = 'Add a new skill catalog Git repository source.';
export const SKILL_MANAGE_SOURCE_TITLE = 'Manage source';
export const SKILL_MANAGE_SOURCE_DESCRIPTION = 'Configure this skill catalog source.';

export const skillCatalogSettingsUrl = (): string => '/skill-catalog-settings';

export const skillAddSourceUrl = (): string => `${skillCatalogSettingsUrl()}/add-source`;

export const skillManageSourceUrl = (catalogSourceId: string): string =>
  `${skillCatalogSettingsUrl()}/manage-source/${encodeURIComponent(catalogSourceId)}`;
