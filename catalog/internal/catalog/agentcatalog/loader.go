package agentcatalog

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/golang/glog"
	agentmodels "github.com/kubeflow/hub/catalog/internal/catalog/agentcatalog/models"
	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	"github.com/kubeflow/hub/catalog/internal/db/models"
)

// AgentLoader handles loading agent data from YAML configuration files.
type AgentLoader struct {
	state basecatalog.LoaderState

	Sources  *AgentSourceCollection
	services Services

	closerMu sync.Mutex
	closer   func()
}

func (l *AgentLoader) setCloser(closer func()) {
	l.closerMu.Lock()
	defer l.closerMu.Unlock()
	if l.closer != nil {
		l.closer()
	}
	l.closer = closer
}

func NewAgentLoader(services Services, state basecatalog.LoaderState) *AgentLoader {
	paths := state.Paths()
	return &AgentLoader{
		state:    state,
		Sources:  NewAgentSourceCollection(paths...),
		services: services,
	}
}

func (l *AgentLoader) ParseAllConfigs() error {
	glog.Infof("Initializing %s loader - parsing configs", "agent")

	for _, path := range l.state.Paths() {
		if err := l.parseAndMerge(path); err != nil {
			return fmt.Errorf("failed to parse agent config %s: %w", path, err)
		}
	}

	glog.Infof("%s loader config parsing complete", "agent")
	return nil
}

func (l *AgentLoader) PerformLeaderOperations(ctx context.Context, allKnownSourceIDs mapset.Set[string]) error {
	glog.Infof("%s loader performing leader operations", "agent")

	ctx, cancel := context.WithCancel(ctx)
	l.setCloser(cancel)
	// Drain in-flight writes from the previous invocation before running cleanup,
	// so a concurrent write from an old goroutine cannot re-insert data that
	// cleanup is about to remove.
	l.state.WaitForInflightWrites(30 * time.Second)

	if err := l.removeAgentsFromMissingSources(allKnownSourceIDs); err != nil {
		glog.Errorf("error removing agents from missing sources: %v", err)
	}

	allSources := l.Sources.AllSources()
	for id, source := range allSources {
		if !source.IsEnabled() {
			basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, id, basecatalog.SourceStatusDisabled, "")
			continue
		}
		if source.Type != "yaml" {
			glog.Warningf("unknown %s provider type: %s", "agent", source.Type)
			basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, id, basecatalog.SourceStatusError, "unknown provider type: "+source.Type)
			continue
		}
		sourceID := id
		src := source
		l.state.TrackWrite() // released after the initial load completes
		go l.watchAndLoadFromYAML(ctx, sourceID, src)
	}

	glog.Infof("%s loader leader operations launched", "agent")
	return nil
}

func (l *AgentLoader) loadFromYAML(ctx context.Context, sourceID string, source basecatalog.PluginSource) error {
	yamlPath, err := resolveYAMLPath(source)
	if err != nil {
		return err
	}

	catalog, err := readYAMLAgentCatalog(yamlPath)
	if err != nil {
		return err
	}

	validNames := mapset.NewSet[string]()
	successCount := 0

	for _, ya := range catalog.Agents {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		namespacedName := sourceID + ":" + ya.Name
		validNames.Add(namespacedName)

		if !l.state.ShouldWriteDatabase() {
			glog.Info("No longer leader, stopping agent database writes")
			return nil
		}

		func() {
			l.state.TrackWrite()
			defer l.state.WriteComplete()

			entity := yamlAgentToEntity(ya, sourceID)
			saved, err := l.services.AgentRepository.Save(entity)
			if err != nil {
				glog.Errorf("error saving agent %s from source %s: %v", ya.Name, sourceID, err)
				return
			}

			agentID := saved.GetID()
			if agentID == nil {
				glog.Errorf("saved agent %s has no ID", ya.Name)
				return
			}

			if l.services.AgentTemplateArtifactRepository != nil {
				if err := l.services.AgentTemplateArtifactRepository.DeleteByParentID(*agentID); err != nil {
					glog.Errorf("error deleting existing template artifacts for agent %s: %v", ya.Name, err)
				}
				for _, tmpl := range ya.Templates {
					tmplEntity := yamlTemplateToEntity(tmpl, ya.Name, sourceID)
					if _, err := l.services.AgentTemplateArtifactRepository.Save(tmplEntity, agentID); err != nil {
						glog.Errorf("error saving template artifact for agent %s: %v", ya.Name, err)
					}
				}
			}

			successCount++
		}()
	}

	if ctx.Err() == nil {
		glog.Infof("%s: loaded %d agents", sourceID, successCount)

		if _, err := l.removeOrphanedAgentsFromSource(sourceID, validNames); err != nil {
			glog.Errorf("error removing orphaned agents from source %s: %v", sourceID, err)
		}
	}

	return nil
}

// watchAndLoadFromYAML performs the initial load from a YAML source, then watches
// the data file for changes and reloads on each change until ctx is cancelled.
// The caller must have called l.state.TrackWrite() before launching this as a goroutine;
// this function calls l.state.WriteComplete() once the initial load is done.
func (l *AgentLoader) watchAndLoadFromYAML(ctx context.Context, sourceID string, source basecatalog.PluginSource) {
	releaseInitialLoad := sync.OnceFunc(l.state.WriteComplete)
	defer releaseInitialLoad() // safety net: ensures TrackWrite is released even on panic

	// Set up the watcher before the initial load so that changes arriving
	// during or just after the read are not missed.
	yamlPath, err := resolveYAMLPath(source)
	if err != nil {
		glog.Errorf("unable to resolve YAML path for agent source %s: %v", sourceID, err)
		return
	}

	ch, watchErr := basecatalog.GetMonitor().Path(ctx, yamlPath)
	if watchErr != nil {
		glog.Errorf("unable to watch agent catalog file %s: %v", yamlPath, watchErr)
	}

	doLoad := func() {
		if err := l.loadFromYAML(ctx, sourceID, source); err != nil {
			glog.Errorf("error loading agent from source %s: %v", sourceID, err)
			basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, sourceID, basecatalog.SourceStatusError, err.Error())
			return
		}
		basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, sourceID, basecatalog.SourceStatusAvailable, "")
		if err := l.services.PropertyOptionsRepository.Refresh(models.ContextPropertyOptionType); err != nil {
			glog.Errorf("error refreshing property options after agent load: %v", err)
		}
	}

	doLoad()
	releaseInitialLoad() // release the TrackWrite slot after the initial load

	if ch == nil {
		// Watcher setup failed; no live updates possible.
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-ch:
			if !ok {
				return
			}
			glog.Infof("Reloading agent catalog from %s (file changed)", yamlPath)
			doLoad()
		}
	}
}

func (l *AgentLoader) ReloadParsing() error {
	var errs []error
	for _, path := range l.state.Paths() {
		if err := l.parseAndMerge(path); err != nil {
			errs = append(errs, fmt.Errorf("unable to reload agent sources from %s: %w", path, err))
		}
	}
	return errors.Join(errs...)
}

func (l *AgentLoader) parseAndMerge(path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %v", path, err)
	}

	config, err := basecatalog.ReadSourceConfig(path)
	if err != nil {
		return err
	}

	return l.updateSources(path, config)
}

func (l *AgentLoader) updateSources(path string, config *basecatalog.SourceConfig) error {
	sources := make(map[string]basecatalog.PluginSource, len(config.AgentCatalogs))

	for _, source := range config.AgentCatalogs {
		glog.Infof("reading agent catalog config type %s...", source.Type)
		if source.GetId() == "" {
			return fmt.Errorf("invalid agent source: missing id")
		}
		if _, exists := sources[source.GetId()]; exists {
			return fmt.Errorf("invalid agent source: duplicate id %s", source.GetId())
		}

		source.Origin = path
		sources[source.GetId()] = source
		glog.Infof("loaded agent source %s of type %s", source.GetId(), source.Type)
	}

	if config.NamedQueries != nil {
		filtered := basecatalog.FilterNamedQueriesByAssetType(config.NamedQueries, basecatalog.AssetTypeAgents)
		if len(filtered) > 0 {
			return l.Sources.MergeWithNamedQueries(path, sources, filtered)
		}
	}

	return l.Sources.Merge(path, sources)
}

func (l *AgentLoader) removeAgentsFromMissingSources(allKnownSourceIDs mapset.Set[string]) error {
	enabledSourceIDs := mapset.NewSet[string]()
	agentSourceIDs := mapset.NewSet[string]()
	for id, source := range l.Sources.AllSources() {
		agentSourceIDs.Add(id)
		if source.IsEnabled() {
			enabledSourceIDs.Add(id)
		}
	}

	existingSourceIDs, err := l.services.AgentRepository.GetDistinctSourceIDs()
	if err != nil {
		return fmt.Errorf("unable to retrieve existing agent source IDs: %w", err)
	}

	for oldSource := range mapset.NewSet(existingSourceIDs...).Difference(enabledSourceIDs).Iter() {
		glog.Infof("Removing agents from source %s", oldSource)

		l.state.TrackWrite()
		err = l.services.AgentRepository.DeleteBySource(oldSource)
		l.state.WriteComplete()
		if err != nil {
			return fmt.Errorf("unable to remove agents from source %q: %w", oldSource, err)
		}

		// Only delete the shared CatalogSource status record if the source is absent
		// from ALL plugin configs — not just the agent config. allKnownSourceIDs is
		// the union of every plugin's configured source IDs, so if another plugin
		// (model, skill, mcp) owns this source we must not delete its status row.
		if !agentSourceIDs.Contains(oldSource) && !allKnownSourceIDs.Contains(oldSource) {
			glog.Infof("Removing status for agent source %s (no longer in any config)", oldSource)
			if delErr := l.services.CatalogSourceRepository.Delete(oldSource); delErr != nil {
				glog.Errorf("failed to delete status for agent source %s: %v", oldSource, delErr)
			}
		}
	}

	protectedSourceIDs := agentSourceIDs.Union(allKnownSourceIDs)
	if err := basecatalog.CleanupOrphanedCatalogSources(l.services.CatalogSourceRepository, protectedSourceIDs); err != nil {
		glog.Errorf("failed to cleanup orphaned agent catalog sources: %v", err)
	}

	return nil
}

func (l *AgentLoader) removeOrphanedAgentsFromSource(sourceID string, validNames mapset.Set[string]) (int, error) {
	list, err := l.services.AgentRepository.List(&agentmodels.AgentListOptions{
		SourceIDs: &[]string{sourceID},
	})
	if err != nil {
		return 0, fmt.Errorf("unable to list agents from source %q: %w", sourceID, err)
	}

	count := 0
	for _, agent := range list.Items {
		attr := agent.GetAttributes()
		if attr == nil || attr.Name == nil || agent.GetID() == nil {
			continue
		}

		if validNames.Contains(*attr.Name) {
			continue
		}

		glog.Infof("Removing orphaned agent %s from source %s", *attr.Name, sourceID)

		l.state.TrackWrite()
		err = l.services.AgentRepository.DeleteByID(*agent.GetID())
		l.state.WriteComplete()
		if err != nil {
			return count, fmt.Errorf("unable to remove agent %d (%s from source %s): %w", *agent.GetID(), *attr.Name, sourceID, err)
		}
		count++
	}

	return count, nil
}
