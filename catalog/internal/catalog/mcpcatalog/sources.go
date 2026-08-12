package mcpcatalog

import (
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	model "github.com/kubeflow/hub/catalog/pkg/openapi"
)

// mcpOriginEntry holds MCP sources from a single origin (config file).
type mcpOriginEntry struct {
	origin  string
	sources map[string]basecatalog.MCPSource
}

// MCPSourceCollection manages MCP catalog sources from multiple origins with priority-based merging.
// Later entries in the slice take precedence over earlier ones.
type MCPSourceCollection struct {
	mu                sync.RWMutex
	entries           []mcpOriginEntry
	namedQueryEntries map[string]map[string]map[string]basecatalog.FieldFilter // origin -> queryName -> fieldName -> FieldFilter
}

// NewMCPSourceCollection creates a new MCPSourceCollection with the given origin order.
// Origins listed later in the order take precedence over earlier ones.
// For example, if originOrder is ["default.yaml", "user.yaml"], sources from
// "user.yaml" will override sources with the same ID from "default.yaml".
func NewMCPSourceCollection(originOrder ...string) *MCPSourceCollection {
	entries := make([]mcpOriginEntry, len(originOrder))
	for i, origin := range originOrder {
		entries[i] = mcpOriginEntry{origin: origin, sources: nil}
	}
	return &MCPSourceCollection{
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
func (msc *MCPSourceCollection) Merge(origin string, sources map[string]basecatalog.MCPSource) error {
	msc.mu.Lock()
	defer msc.mu.Unlock()

	// Clear any previously contributed named queries for this origin
	delete(msc.namedQueryEntries, origin)

	return msc.mergeSourcesInternal(origin, sources)
}

// MergeWithNamedQueries adds sources and named queries from one origin.
// Later origins override earlier ones at the field level within a query.
func (msc *MCPSourceCollection) MergeWithNamedQueries(origin string, sources map[string]basecatalog.MCPSource, namedQueries map[string]map[string]basecatalog.FieldFilter) error {
	msc.mu.Lock()
	defer msc.mu.Unlock()

	if err := msc.mergeSourcesInternal(origin, sources); err != nil {
		return err
	}

	// Deep-copy and store named queries for this origin (clears any previously contributed entries)
	msc.namedQueryEntries[origin] = basecatalog.CloneNamedQueries(namedQueries)

	return nil
}

// mergedNamedQueries computes the merged view of all named queries across origins.
// Later origins (by entry order) override earlier ones at the field level.
// Must be called with lock held.
func (msc *MCPSourceCollection) mergedNamedQueries() map[string]map[string]basecatalog.FieldFilter {
	originOrder := make([]string, len(msc.entries))
	for i, entry := range msc.entries {
		originOrder[i] = entry.origin
	}
	return basecatalog.MergeNamedQueriesInOrder(originOrder, msc.namedQueryEntries)
}

// GetNamedQuery returns a copy of a single named query by name.
// Returns (filters, true) if found, or (nil, false) if unknown.
// Slice values within each FieldFilter are cloned to prevent callers from
// accidentally mutating internal state.
func (msc *MCPSourceCollection) GetNamedQuery(name string) (map[string]basecatalog.FieldFilter, bool) {
	msc.mu.RLock()
	defer msc.mu.RUnlock()

	merged := msc.mergedNamedQueries()
	fieldFilters, ok := merged[name]
	if !ok {
		return nil, false
	}
	return basecatalog.CloneFieldFilters(fieldFilters), true
}

// GetNamedQueries returns a deep copy of all merged named queries.
func (msc *MCPSourceCollection) GetNamedQueries() map[string]map[string]basecatalog.FieldFilter {
	msc.mu.RLock()
	defer msc.mu.RUnlock()

	return basecatalog.CloneNamedQueries(msc.mergedNamedQueries())
}

// mergeSourcesInternal performs the source merge. Must be called with lock held.
func (msc *MCPSourceCollection) mergeSourcesInternal(origin string, sources map[string]basecatalog.MCPSource) error {
	// Find existing entry for this origin
	for i := range msc.entries {
		if msc.entries[i].origin == origin {
			msc.entries[i].sources = sources
			return nil
		}
	}

	// Origin not found, append it (dynamic registration)
	msc.entries = append(msc.entries, mcpOriginEntry{origin: origin, sources: sources})
	return nil
}

// mergeMCPSources performs field-level merging of two MCPSource structs.
// Fields from 'override' take precedence over 'base' when they are explicitly set.
// A field is considered "set" if:
// - For strings: non-empty
// - For pointers: non-nil
// - For slices: non-nil (empty slice is considered explicitly set to "no items")
// - For maps: non-nil (empty map is considered explicitly set)
func mergeMCPSources(base, override basecatalog.MCPSource) basecatalog.MCPSource {
	result := base

	// ID is always taken from override (it's the key)
	result.ID = override.ID

	// Merge shared fields using the common helper
	common := basecatalog.MergeCommonSourceFields(
		basecatalog.CommonSourceFields{Name: base.Name, Enabled: base.Enabled, Labels: base.Labels, Type: base.Type, Properties: base.Properties, Origin: base.Origin, AssetType: base.AssetType},
		basecatalog.CommonSourceFields{Name: override.Name, Enabled: override.Enabled, Labels: override.Labels, Type: override.Type, Properties: override.Properties, Origin: override.Origin, AssetType: override.AssetType},
	)
	result.Name = common.Name
	result.Enabled = common.Enabled
	result.Labels = common.Labels
	result.Type = common.Type
	result.Properties = common.Properties
	result.Origin = common.Origin
	result.AssetType = common.AssetType

	if override.IncludedServers != nil {
		result.IncludedServers = override.IncludedServers
	}
	if override.ExcludedServers != nil {
		result.ExcludedServers = override.ExcludedServers
	}

	return result
}

// applyMCPDefaults applies default values to an MCPSource for fields that are not set.
func applyMCPDefaults(source basecatalog.MCPSource) basecatalog.MCPSource {
	if source.Enabled == nil {
		source.Enabled = new(true)
	}
	if source.Labels == nil {
		source.Labels = []string{}
	}
	if source.AssetType == nil {
		source.AssetType = model.CATALOGASSETTYPE_MCP_SERVERS.Ptr()
	}
	return source
}

// merged computes the merged view of all MCP sources with field-level merging.
// Must be called with lock held.
func (msc *MCPSourceCollection) merged() map[string]basecatalog.MCPSource {
	result := map[string]basecatalog.MCPSource{}

	for _, entry := range msc.entries {
		for id, source := range entry.sources {
			if existing, ok := result[id]; ok {
				// Field-level merge: existing is base, source is override
				result[id] = mergeMCPSources(existing, source)
			} else {
				result[id] = source
			}
		}
	}

	// Apply defaults to all merged sources
	for id, source := range result {
		result[id] = applyMCPDefaults(source)
	}

	return result
}

// AllSources returns all merged MCP sources including Type and Properties.
// This is used by the loader to get complete source information.
// All sources are returned regardless of enabled status.
func (msc *MCPSourceCollection) AllSources() map[string]basecatalog.MCPSource {
	msc.mu.RLock()
	defer msc.mu.RUnlock()

	return msc.merged()
}

// ByLabel returns enabled sources that have any of the labels provided. The matching
// is case insensitive.
//
// If a label is "null", every source without a label is returned.
func (msc *MCPSourceCollection) ByLabel(labels []string) []basecatalog.MCPSource {
	msc.mu.RLock()
	defer msc.mu.RUnlock()

	labelMap := make(map[string]struct{}, len(labels))
	for _, label := range labels {
		labelMap[strings.ToLower(label)] = struct{}{}
	}

	matches := map[string]basecatalog.MCPSource{}
	sources := msc.merged()

	if _, hasNull := labelMap["null"]; hasNull {
		for _, source := range sources {
			if source.Enabled == nil || !*source.Enabled {
				continue
			}
			if len(source.Labels) == 0 {
				matches[source.ID] = source
			}
		}
	}

OUTER:
	for _, source := range sources {
		if source.Enabled == nil || !*source.Enabled {
			continue
		}
		for _, label := range source.Labels {
			if _, match := labelMap[strings.ToLower(label)]; match {
				matches[source.ID] = source
				continue OUTER
			}
		}
	}

	return slices.Collect(maps.Values(matches))
}
