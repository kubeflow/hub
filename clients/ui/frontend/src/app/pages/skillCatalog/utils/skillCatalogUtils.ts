import { hasFiltersApplied, stringFiltersToFilterQuery } from '~/app/shared/components/catalog';
import { SKILL_FILTER_KEYS } from '~/app/pages/skillCatalog/const';
import type { SkillCatalogFiltersState } from '~/app/pages/skillCatalog/types/skillCatalogFilterOptions';

export const hasSkillFiltersApplied = (
  filters: SkillCatalogFiltersState,
  searchQuery: string,
): boolean => hasFiltersApplied(filters, SKILL_FILTER_KEYS, searchQuery);

export function skillFiltersToFilterQuery(filters: SkillCatalogFiltersState): string {
  return stringFiltersToFilterQuery(filters, {});
}

/**
 * The catalog API's allowedTools array is comma-delimited tool patterns
 * (e.g. "Bash(git diff:*)") that have additionally been split on whitespace
 * upstream, so a single pattern like "Bash(aws logs tail:*)" arrives as three
 * separate array entries. Rejoining with spaces reconstructs the original
 * comma-delimited string, which is then split on "," to recover one entry
 * per tool pattern.
 */
export function parseAllowedTools(allowedTools?: string[]): string[] {
  if (!allowedTools || allowedTools.length === 0) {
    return [];
  }
  return allowedTools
    .join(' ')
    .split(',')
    .map((tool) => tool.trim())
    .filter(Boolean);
}

/**
 * A skill's version is the ref it was indexed at — a tag, a release, or a commit SHA.
 * A full SHA is 40 characters and would dominate a gallery card, so it is abbreviated
 * to the 7 characters git itself uses for short hashes.
 *
 * Only an exact 40-character hex string is abbreviated. Shorter hex-looking values are
 * left alone: a tag may legitimately be named something like "abc1234", and truncating
 * it would misreport the version.
 */
export function formatSkillVersion(version: string): string {
  return /^[0-9a-f]{40}$/i.test(version) ? version.slice(0, 7) : version;
}
