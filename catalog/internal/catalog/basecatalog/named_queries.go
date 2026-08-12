package basecatalog

import "maps"

// deepCopyFieldFilter returns a copy of ff where slice values are cloned.
func deepCopyFieldFilter(ff FieldFilter) FieldFilter {
	if vals, ok := ff.Value.([]any); ok {
		cp := make([]any, len(vals))
		copy(cp, vals)
		ff.Value = cp
	}
	return ff
}

// CloneFieldFilters returns a deep copy of a single query's field filters,
// cloning mutable FieldFilter.Value slices so that later caller mutations
// cannot affect stored state. Returns nil for a nil input.
func CloneFieldFilters(src map[string]FieldFilter) map[string]FieldFilter {
	if src == nil {
		return nil
	}
	dst := make(map[string]FieldFilter, len(src))
	for field, ff := range src {
		dst[field] = deepCopyFieldFilter(ff)
	}
	return dst
}

// CloneNamedQueries returns a deep copy of the entire named-queries map,
// including mutable FieldFilter.Value slices, so that later caller
// mutations cannot affect stored state.
func CloneNamedQueries(src map[string]map[string]FieldFilter) map[string]map[string]FieldFilter {
	if src == nil {
		return nil
	}
	dst := make(map[string]map[string]FieldFilter, len(src))
	for queryName, fieldFilters := range src {
		dst[queryName] = CloneFieldFilters(fieldFilters)
	}
	return dst
}

// MergeNamedQueriesInOrder computes the merged view of all named queries across
// origins. originOrder gives priority order (later origins override earlier
// ones at the field level); entriesByOrigin maps origin -> queryName ->
// fieldName -> FieldFilter.
func MergeNamedQueriesInOrder(
	originOrder []string,
	entriesByOrigin map[string]map[string]map[string]FieldFilter,
) map[string]map[string]FieldFilter {
	result := make(map[string]map[string]FieldFilter)
	for _, origin := range originOrder {
		originQueries, ok := entriesByOrigin[origin]
		if !ok {
			continue
		}
		for queryName, fieldFilters := range originQueries {
			if result[queryName] == nil {
				result[queryName] = make(map[string]FieldFilter)
			}
			maps.Copy(result[queryName], fieldFilters)
		}
	}
	return result
}
