import type { SkillFilterCategoryKey } from '~/app/pages/skillCatalog/types/skillCatalogFilterOptions';

export const SKILL_CATALOG_TITLE = 'Skill Catalog';
export const SKILL_CATALOG_DESCRIPTION = 'Browse and use skills provided by your organization.';

export const OTHER_SKILLS_DISPLAY_NAME = 'Other skills';
export const ALL_SKILLS_LABEL = 'All skills';

export const SKILL_FILTER_KEYS: SkillFilterCategoryKey[] = ['provider', 'trustTier', 'category'];

export const SKILL_FILTER_CATEGORY_NAMES: Record<SkillFilterCategoryKey, string> = {
  provider: 'Provider',
  trustTier: 'Trust tier',
  category: 'Category',
};

// Keys must match the catalog backend's SkillTrustTier enum exactly
// (catalog/pkg/openapi/model_skill_trust_tier.go) — it's a hard enum, so these are the
// only values a skill's trustTier can ever have.
export const SKILL_TRUST_TIER_LABEL_MAPPING: Record<string, string> = {
  platformProvided: 'Platform Provided',
  partnerVerified: 'Partner Verified',
  organizationApproved: 'Organization Approved',
  communityContributed: 'Community Contributed',
};

// Shape matches mod-arch-shared's SimpleSelectOption, so this can be passed to
// <SimpleSelect options={...} /> directly.
export const SKILL_TRUST_TIER_OPTIONS: { key: string; label: string }[] = [
  { key: 'platformProvided', label: SKILL_TRUST_TIER_LABEL_MAPPING.platformProvided },
  { key: 'partnerVerified', label: SKILL_TRUST_TIER_LABEL_MAPPING.partnerVerified },
  { key: 'organizationApproved', label: SKILL_TRUST_TIER_LABEL_MAPPING.organizationApproved },
  { key: 'communityContributed', label: SKILL_TRUST_TIER_LABEL_MAPPING.communityContributed },
];

export const SKILL_LABEL_MAPPINGS: Record<string, Record<string, string>> = {
  trustTier: SKILL_TRUST_TIER_LABEL_MAPPING,
};

export const BACKEND_TO_FRONTEND_SKILL_FILTER_KEY: Record<string, SkillFilterCategoryKey> = {};

export const SKILL_CATALOG_GALLERY = {
  CARDS_PER_ROW: 4,
  PAGE_SIZE: 10,
  SECTION_TITLE: 'Skills',
} as const;

type GridSpan = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12;

const GRID_COLUMNS = 12;
const GRID_SPAN_VALUES: GridSpan[] = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12];

function toGridSpan(cols: number): GridSpan {
  const index = Math.min(Math.max(0, cols - 1), GRID_SPAN_VALUES.length - 1);
  return GRID_SPAN_VALUES[index];
}

export const SKILL_CATALOG_GRID_SPAN: {
  sm: GridSpan;
  md: GridSpan;
  lg: GridSpan;
  xl2: GridSpan;
} = {
  sm: toGridSpan(GRID_COLUMNS),
  md: toGridSpan(GRID_COLUMNS / 2),
  lg: toGridSpan(GRID_COLUMNS / SKILL_CATALOG_GALLERY.CARDS_PER_ROW),
  xl2: toGridSpan(GRID_COLUMNS / SKILL_CATALOG_GALLERY.CARDS_PER_ROW),
};

// Mirrors maxSkillBodyLines in the catalog service's skill parser
// (catalog/internal/catalog/skillcatalog/skillmd_parser.go), which warns when a
// SKILL.md body exceeds it. The service does not expose the threshold, so it is
// duplicated here; it comes from the Agent Skills specification rather than being
// deployment-tunable. If skill validation rules later expose their thresholds through
// the API, this constant should give way to that value.
export const SKILL_RECOMMENDED_MAX_BODY_LINES = 500;
