package skillcatalog

import (
	"sync"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	model "github.com/kubeflow/hub/catalog/pkg/openapi"
)

type skillOriginEntry struct {
	origin  string
	sources map[string]basecatalog.PluginSource
}

// SkillSourceCollection manages skill catalog sources from multiple origins with priority-based merging.
type SkillSourceCollection struct {
	mu      sync.RWMutex
	entries []skillOriginEntry
}

func NewSkillSourceCollection(originOrder ...string) *SkillSourceCollection {
	entries := make([]skillOriginEntry, len(originOrder))
	for i, origin := range originOrder {
		entries[i] = skillOriginEntry{origin: origin, sources: nil}
	}
	return &SkillSourceCollection{
		entries: entries,
	}
}

func (sc *SkillSourceCollection) Merge(origin string, sources map[string]basecatalog.PluginSource) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	for i := range sc.entries {
		if sc.entries[i].origin == origin {
			sc.entries[i].sources = sources
			return nil
		}
	}

	sc.entries = append(sc.entries, skillOriginEntry{origin: origin, sources: sources})
	return nil
}

func (sc *SkillSourceCollection) merged() map[string]basecatalog.PluginSource {
	result := map[string]basecatalog.PluginSource{}

	for _, entry := range sc.entries {
		for id, source := range entry.sources {
			if existing, ok := result[id]; ok {
				result[id] = mergeSkillSources(existing, source)
			} else {
				result[id] = source
			}
		}
	}

	for id, source := range result {
		result[id] = applySkillDefaults(source)
	}

	return result
}

func mergeSkillSources(base, override basecatalog.PluginSource) basecatalog.PluginSource {
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

func applySkillDefaults(source basecatalog.PluginSource) basecatalog.PluginSource {
	if source.Enabled == nil {
		source.Enabled = new(true)
	}
	if source.Labels == nil {
		source.Labels = []string{}
	}
	if source.AssetType == nil {
		source.AssetType = model.CATALOGASSETTYPE_SKILLS.Ptr()
	}
	return source
}

func (sc *SkillSourceCollection) AllSources() map[string]basecatalog.PluginSource {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return sc.merged()
}
