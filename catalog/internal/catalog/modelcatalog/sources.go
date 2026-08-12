package modelcatalog

import (
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	model "github.com/kubeflow/hub/catalog/pkg/openapi"
)

// originEntry holds sources from a single origin (config file).
type originEntry struct {
	origin  string
	sources map[string]basecatalog.ModelSource
}

// SourceCollection manages catalog sources from multiple origins with priority-based merging.
// Later entries in the slice take precedence over earlier ones.
type SourceCollection struct {
	mu                sync.RWMutex
	entries           []originEntry
	namedQueryEntries map[string]map[string]map[string]basecatalog.FieldFilter // origin -> queryName -> fieldName -> FieldFilter
}

// NewSourceCollection creates a new SourceCollection with the given origin order.
// Origins listed later in the order take precedence over earlier ones.
// For example, if originOrder is ["default.yaml", "user.yaml"], sources from
// "user.yaml" will override sources with the same ID from "default.yaml".
func NewSourceCollection(originOrder ...string) *SourceCollection {
	entries := make([]originEntry, len(originOrder))
	for i, origin := range originOrder {
		entries[i] = originEntry{origin: origin, sources: nil}
	}
	return &SourceCollection{
		entries:           entries,
		namedQueryEntries: make(map[string]map[string]map[string]basecatalog.FieldFilter),
	}
}

// Merge adds sources from one origin (ordinarily, a file path--but any unique
// string will do), completely replacing anything that was previously from that
// origin.
//
// If a source with the same ID exists in multiple origins, fields from
// higher-priority origins (listed later in entries) override fields from
// lower-priority origins. Fields that are not set (zero value for strings,
// nil for pointers/slices/maps) in the override are inherited from the base.
func (sc *SourceCollection) Merge(origin string, sources map[string]basecatalog.ModelSource) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Clear any previously contributed named queries for this origin
	delete(sc.namedQueryEntries, origin)

	return sc.mergeSourcesInternal(origin, sources)
}

// MergeWithNamedQueries adds sources and named queries from one origin.
func (sc *SourceCollection) MergeWithNamedQueries(origin string, sources map[string]basecatalog.ModelSource, namedQueries map[string]map[string]basecatalog.FieldFilter) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Merge sources using existing logic
	if err := sc.mergeSourcesInternal(origin, sources); err != nil {
		return err
	}

	// Deep-copy and store named queries for this origin (clears any previously contributed entries)
	sc.namedQueryEntries[origin] = basecatalog.CloneNamedQueries(namedQueries)

	return nil
}

// mergeSourcesInternal extracts the internal logic from Merge
func (sc *SourceCollection) mergeSourcesInternal(origin string, sources map[string]basecatalog.ModelSource) error {
	// Find existing entry for this origin
	for i := range sc.entries {
		if sc.entries[i].origin == origin {
			sc.entries[i].sources = sources
			return nil
		}
	}

	// Origin not found, append it (dynamic registration)
	sc.entries = append(sc.entries, originEntry{origin: origin, sources: sources})
	return nil
}

// mergedNamedQueries computes the merged view of all named queries across origins.
// Later origins (by entry order) override earlier ones at the field level.
// Must be called with lock held.
func (sc *SourceCollection) mergedNamedQueries() map[string]map[string]basecatalog.FieldFilter {
	originOrder := make([]string, len(sc.entries))
	for i, entry := range sc.entries {
		originOrder[i] = entry.origin
	}
	return basecatalog.MergeNamedQueriesInOrder(originOrder, sc.namedQueryEntries)
}

// GetNamedQueries returns all merged named queries
func (sc *SourceCollection) GetNamedQueries() map[string]map[string]basecatalog.FieldFilter {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return basecatalog.CloneNamedQueries(sc.mergedNamedQueries())
}

// mergeSources performs field-level merging of two Source structs.
// Fields from 'override' take precedence over 'base' when they are explicitly set.
// A field is considered "set" if:
// - For strings: non-empty
// - For pointers: non-nil
// - For slices: non-nil (empty slice is considered explicitly set to "no items")
// - For maps: non-nil (empty map is considered explicitly set)
func mergeSources(base, override basecatalog.ModelSource) basecatalog.ModelSource {
	result := base

	// Id is always taken from override (it's the key)
	result.Id = override.Id

	// Merge shared fields using the common helper
	common := basecatalog.MergeCommonSourceFields(
		basecatalog.CommonSourceFields{Name: base.Name, Enabled: base.Enabled, Labels: base.Labels, Type: base.Type, Properties: base.Properties, Origin: base.Origin},
		basecatalog.CommonSourceFields{Name: override.Name, Enabled: override.Enabled, Labels: override.Labels, Type: override.Type, Properties: override.Properties, Origin: override.Origin},
	)
	result.Name = common.Name
	result.Enabled = common.Enabled
	result.Labels = common.Labels
	result.Type = common.Type
	result.Properties = common.Properties
	result.Origin = common.Origin

	// Model-specific fields
	if override.IncludedModels != nil {
		result.IncludedModels = override.IncludedModels
	}
	if override.ExcludedModels != nil {
		result.ExcludedModels = override.ExcludedModels
	}

	return result
}

// applyDefaults applies default values to an Source for fields that are not set.
func applyDefaults(source basecatalog.ModelSource) basecatalog.ModelSource {
	// Default Enabled to true if not set
	if source.Enabled == nil {
		source.Enabled = new(true)
	}

	// Default Labels to empty slice if not set
	if source.Labels == nil {
		source.Labels = []string{}
	}

	return source
}

// merged computes the merged view of all sources with field-level merging.
// Must be called with lock held.
func (sc *SourceCollection) merged() map[string]basecatalog.ModelSource {
	result := map[string]basecatalog.ModelSource{}

	for _, entry := range sc.entries {
		for id, source := range entry.sources {
			if existing, ok := result[id]; ok {
				// Field-level merge: existing is base, source is override
				result[id] = mergeSources(existing, source)
			} else {
				result[id] = source
			}
		}
	}

	// Apply defaults to all merged sources
	for id, source := range result {
		result[id] = applyDefaults(source)
	}

	return result
}

// AllSources returns all merged sources including Type and Properties.
// This is used by the loader to get complete source information.
// All sources are returned regardless of enabled status.
func (sc *SourceCollection) AllSources() map[string]basecatalog.ModelSource {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	result := map[string]basecatalog.ModelSource{}
	maps.Copy(result, sc.merged())
	return result
}

// All returns all sources as CatalogSource (for the API).
// This excludes internal fields like Type and Properties.
func (sc *SourceCollection) All() map[string]model.CatalogSource {
	result := map[string]model.CatalogSource{}
	for id, source := range sc.AllSources() {
		result[id] = source.CatalogSource
	}
	return result
}

// Get returns a source by name if it exists and is enabled.
func (sc *SourceCollection) Get(name string) (src model.CatalogSource, ok bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// Get from merged view (which includes field-level merging and defaults)
	merged := sc.merged()
	source, exists := merged[name]
	if !exists {
		return model.CatalogSource{}, false
	}

	// Only return if enabled
	if source.Enabled != nil && *source.Enabled {
		return source.CatalogSource, true
	}
	return model.CatalogSource{}, false
}

// ByLabel returns enabled sources that have any of the labels provided. The matching
// is case insensitive.
//
// If a label is "null", every source without a label is returned.
func (sc *SourceCollection) ByLabel(labels []string) []model.CatalogSource {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	labelMap := make(map[string]struct{}, len(labels))
	for _, label := range labels {
		labelMap[strings.ToLower(label)] = struct{}{}
	}

	matches := map[string]model.CatalogSource{}
	sources := sc.merged()

	if _, hasNull := labelMap["null"]; hasNull {
		for _, source := range sources {
			// Skip disabled sources
			if source.Enabled == nil || !*source.Enabled {
				continue
			}
			if len(source.Labels) == 0 {
				matches[source.Id] = source.CatalogSource
			}
		}
	}

OUTER:
	for _, source := range sources {
		// Skip disabled sources
		if source.Enabled == nil || !*source.Enabled {
			continue
		}
		for _, label := range source.Labels {
			if _, match := labelMap[strings.ToLower(label)]; match {
				matches[source.Id] = source.CatalogSource
				continue OUTER
			}
		}
	}

	return slices.Collect(maps.Values(matches))
}
