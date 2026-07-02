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
	mu           sync.RWMutex
	entries      []agentOriginEntry
	namedQueries map[string]map[string]basecatalog.FieldFilter
}

func NewAgentSourceCollection(originOrder ...string) *AgentSourceCollection {
	entries := make([]agentOriginEntry, len(originOrder))
	for i, origin := range originOrder {
		entries[i] = agentOriginEntry{origin: origin, sources: nil}
	}
	return &AgentSourceCollection{
		entries:      entries,
		namedQueries: make(map[string]map[string]basecatalog.FieldFilter),
	}
}

func (sc *AgentSourceCollection) Merge(origin string, sources map[string]basecatalog.PluginSource) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	for i := range sc.entries {
		if sc.entries[i].origin == origin {
			sc.entries[i].sources = sources
			return nil
		}
	}

	sc.entries = append(sc.entries, agentOriginEntry{origin: origin, sources: sources})
	return nil
}

func (sc *AgentSourceCollection) MergeWithNamedQueries(origin string, sources map[string]basecatalog.PluginSource, namedQueries map[string]map[string]basecatalog.FieldFilter) error {
	if err := sc.Merge(origin, sources); err != nil {
		return err
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()
	for queryName, fieldFilters := range namedQueries {
		if sc.namedQueries[queryName] == nil {
			sc.namedQueries[queryName] = make(map[string]basecatalog.FieldFilter)
		}
		maps.Copy(sc.namedQueries[queryName], fieldFilters)
	}
	return nil
}

func (sc *AgentSourceCollection) GetNamedQueries() map[string]map[string]basecatalog.FieldFilter {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	result := make(map[string]map[string]basecatalog.FieldFilter, len(sc.namedQueries))
	for queryName, fieldFilters := range sc.namedQueries {
		result[queryName] = make(map[string]basecatalog.FieldFilter, len(fieldFilters))
		maps.Copy(result[queryName], fieldFilters)
	}
	return result
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
