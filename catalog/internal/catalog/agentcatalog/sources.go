package agentcatalog

import (
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	model "github.com/kubeflow/hub/catalog/pkg/openapi"
)

type agentOriginEntry struct {
	origin  string
	sources map[string]basecatalog.PluginSource
}

// AgentSourceCollection manages agent catalog sources from multiple origins with priority-based merging.
type AgentSourceCollection struct {
	mu                sync.RWMutex
	entries           []agentOriginEntry
	namedQueryEntries map[string]map[string]map[string]basecatalog.FieldFilter // origin -> queryName -> fieldName -> FieldFilter
}

func NewAgentSourceCollection(originOrder ...string) *AgentSourceCollection {
	entries := make([]agentOriginEntry, len(originOrder))
	for i, origin := range originOrder {
		entries[i] = agentOriginEntry{origin: origin, sources: nil}
	}
	return &AgentSourceCollection{
		entries:           entries,
		namedQueryEntries: make(map[string]map[string]map[string]basecatalog.FieldFilter),
	}
}

// Merge adds sources from one origin, completely replacing anything that was
// previously from that origin. Any named queries previously contributed by
// this origin are cleared.
func (sc *AgentSourceCollection) Merge(origin string, sources map[string]basecatalog.PluginSource) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Clear any previously contributed named queries for this origin
	delete(sc.namedQueryEntries, origin)

	return sc.mergeSourcesInternal(origin, sources)
}

// MergeWithNamedQueries adds sources and named queries from one origin.
// Later origins override earlier ones at the field level within a query.
func (sc *AgentSourceCollection) MergeWithNamedQueries(origin string, sources map[string]basecatalog.PluginSource, namedQueries map[string]map[string]basecatalog.FieldFilter) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if err := sc.mergeSourcesInternal(origin, sources); err != nil {
		return err
	}

	// Deep-copy and store named queries for this origin (clears any previously contributed entries)
	sc.namedQueryEntries[origin] = basecatalog.CloneNamedQueries(namedQueries)

	return nil
}

// mergeSourcesInternal performs the source merge. Must be called with lock held.
func (sc *AgentSourceCollection) mergeSourcesInternal(origin string, sources map[string]basecatalog.PluginSource) error {
	for i := range sc.entries {
		if sc.entries[i].origin == origin {
			sc.entries[i].sources = sources
			return nil
		}
	}

	sc.entries = append(sc.entries, agentOriginEntry{origin: origin, sources: sources})
	return nil
}

// mergedNamedQueries computes the merged view of all named queries across origins.
// Later origins (by entry order) override earlier ones at the field level.
// Must be called with lock held.
func (sc *AgentSourceCollection) mergedNamedQueries() map[string]map[string]basecatalog.FieldFilter {
	originOrder := make([]string, len(sc.entries))
	for i, entry := range sc.entries {
		originOrder[i] = entry.origin
	}
	return basecatalog.MergeNamedQueriesInOrder(originOrder, sc.namedQueryEntries)
}

// GetNamedQueries returns a deep copy of all merged named queries.
func (sc *AgentSourceCollection) GetNamedQueries() map[string]map[string]basecatalog.FieldFilter {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return basecatalog.CloneNamedQueries(sc.mergedNamedQueries())
}

func (sc *AgentSourceCollection) merged() map[string]basecatalog.PluginSource {
	result := map[string]basecatalog.PluginSource{}

	for _, entry := range sc.entries {
		for id, source := range entry.sources {
			if existing, ok := result[id]; ok {
				result[id] = mergeAgentSources(existing, source)
			} else {
				result[id] = source
			}
		}
	}

	for id, source := range result {
		result[id] = applyAgentDefaults(source)
	}

	return result
}

func mergeAgentSources(base, override basecatalog.PluginSource) basecatalog.PluginSource {
	result := base

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

	return result
}

func applyAgentDefaults(source basecatalog.PluginSource) basecatalog.PluginSource {
	if source.Enabled == nil {
		source.Enabled = new(true)
	}
	if source.Labels == nil {
		source.Labels = []string{}
	}
	if source.AssetType == nil {
		source.AssetType = model.CATALOGASSETTYPE_AGENTS.Ptr()
	}
	return source
}

func (sc *AgentSourceCollection) AllSources() map[string]basecatalog.PluginSource {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return sc.merged()
}

// ByLabel returns enabled sources that have any of the labels provided.
// Matching is case-insensitive. If a label is "null", sources without labels are returned.
func (sc *AgentSourceCollection) ByLabel(labels []string) []basecatalog.PluginSource {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	labelMap := make(map[string]struct{}, len(labels))
	for _, label := range labels {
		labelMap[strings.ToLower(label)] = struct{}{}
	}

	matches := map[string]basecatalog.PluginSource{}
	sources := sc.merged()

	if _, hasNull := labelMap["null"]; hasNull {
		for _, source := range sources {
			if source.Enabled != nil && !*source.Enabled {
				continue
			}
			if len(source.Labels) == 0 {
				matches[source.ID] = source
			}
		}
	}

OUTER:
	for _, source := range sources {
		if source.Enabled != nil && !*source.Enabled {
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
