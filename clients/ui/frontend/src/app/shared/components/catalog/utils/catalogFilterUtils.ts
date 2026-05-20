export const wrapInQuotes = (v: string): string => `'${v.replace(/'/g, "''")}'`;

export const eqFilter = (k: string, v: string): string => `${k}=${wrapInQuotes(v)}`;

export const inFilter = (k: string, values: string[]): string =>
  `${k} IN (${values.map((v) => wrapInQuotes(v)).join(',')})`;

export const andFilter = (k: string, values: string[]): string =>
  values.map((v) => eqFilter(k, v)).join(' AND ');

/**
 * Computes the next filter selection array after toggling a value.
 * Returns null if checked=true and the value is already present (no-op).
 */
export function toggleFilterValue(
  current: string[],
  value: string,
  checked: boolean,
): string[] | null {
  if (checked) {
    return current.includes(value) ? null : [...current, value];
  }
  return current.filter((x) => x !== value);
}

/**
 * Converts a record of string-array filters into a filterQuery string.
 * Handles single-value equality and multi-value IN (OR) or AND clauses, joined with AND.
 *
 * @param filters - Map of filter keys to selected string values
 * @param keyMapping - Optional frontend-to-backend key remapping
 * @param matchAllKeys - Keys that use AND logic instead of OR (IN) when multiple values selected
 * @returns The filterQuery string for the API, or empty string if no filters active
 */
export function stringFiltersToFilterQuery(
  filters: Record<string, string[] | undefined>,
  keyMapping?: Record<string, string>,
  matchAllKeys?: string[],
): string {
  const clauses: string[] = [];
  for (const [key, values] of Object.entries(filters)) {
    if (!values || values.length === 0) {
      continue;
    }
    const queryKey = keyMapping?.[key] ?? key;
    if (values.length === 1) {
      clauses.push(eqFilter(queryKey, values[0]));
    } else if (matchAllKeys?.includes(key)) {
      clauses.push(andFilter(queryKey, values));
    } else {
      clauses.push(inFilter(queryKey, values));
    }
  }
  return clauses.join(' AND ');
}
