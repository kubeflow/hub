export const SKILL_CATALOG_TITLE = 'Skill Catalog';

export const skillCatalogUrl = (): string => '/skill-catalog';

export const skillDetailsUrl = (skillId: string | number): string =>
  `${skillCatalogUrl()}/${encodeURIComponent(String(skillId))}`;
