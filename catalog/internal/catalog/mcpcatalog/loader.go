package mcpcatalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/golang/glog"
	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog/models"
	mrmodels "github.com/kubeflow/hub/internal/platform/db/entity"
)

// MCPPartiallyAvailableError indicates that a source loaded some MCP servers successfully
// but encountered errors with others.
type MCPPartiallyAvailableError struct {
	FailedServers []string
}

func (e *MCPPartiallyAvailableError) Error() string {
	return fmt.Sprintf("Failed to load some MCP servers: %v", e.FailedServers)
}

// MCPLoaderEventHandler is called after an MCP server is successfully loaded
type MCPLoaderEventHandler func(ctx context.Context, record MCPServerProviderRecord) error

// MCPLoader handles loading MCP servers from YAML configuration files.
// It uses external state (LoaderState) for leader operations and write tracking.
type MCPLoader struct {
	state basecatalog.LoaderState

	// Sources contains current MCP source information loaded from the configuration files.
	Sources *MCPSourceCollection

	services Services
	handlers []MCPLoaderEventHandler

	closerMu sync.Mutex
	closer   func() // cancels in-progress PerformLeaderOperations
}

// setCloser stores a cancel function that aborts the current leader operations.
// If a previous cancel function exists it is called first, preempting the old run.
func (ml *MCPLoader) setCloser(closer func()) {
	ml.closerMu.Lock()
	defer ml.closerMu.Unlock()
	if ml.closer != nil {
		ml.closer()
	}
	ml.closer = closer
}

// UpdateServices replaces the loader's repository references after a database
// reconnect. Safe to call from the OnBecomeLeader callback: the elector
// drains all previous leader callbacks before invoking a new one, so this
// always runs before NotifyLeader starts any leader-mode loading.
func (ml *MCPLoader) UpdateServices(services Services) {
	ml.services = services
}

// NewMCPLoaderWithState creates a new MCP loader with external state
func NewMCPLoaderWithState(services Services, state basecatalog.LoaderState) *MCPLoader {
	paths := state.Paths()
	return &MCPLoader{
		state:    state,
		Sources:  NewMCPSourceCollection(paths...),
		services: services,
	}
}

// RegisterEventHandler adds a function that will be called for every
// successfully processed MCP server record
func (ml *MCPLoader) RegisterEventHandler(fn MCPLoaderEventHandler) {
	ml.handlers = append(ml.handlers, fn)
}

// ParseAllConfigs parses all config files into in-memory collections.
// This is called by the unified loader during initialization.
func (ml *MCPLoader) ParseAllConfigs() error {
	glog.Info("Initializing MCP loader - parsing configs")

	for _, path := range ml.state.Paths() {
		if err := ml.parseAndMerge(path); err != nil {
			return fmt.Errorf("failed to parse MCP config %s: %w", path, err)
		}
	}

	glog.Info("MCP loader config parsing complete")
	return nil
}

// PerformLeaderOperations executes database write operations.
// allKnownSourceIDs is the union of model and MCP source IDs, used to prevent
// cross-contamination when cleaning up shared CatalogSource records.
// This is called by the unified loader when becoming leader.
// It returns immediately after launching background goroutines; those goroutines
// continue watching for file changes until ctx is cancelled.
func (ml *MCPLoader) PerformLeaderOperations(ctx context.Context, allKnownSourceIDs mapset.Set[string]) error {
	glog.Info("MCP loader performing leader operations")

	ctx, cancel := context.WithCancel(ctx)
	ml.setCloser(cancel)
	// Drain in-flight writes from the previous invocation before running cleanup,
	// so a concurrent write from an old goroutine cannot re-insert data that
	// cleanup is about to remove.
	ml.state.WaitForInflightWrites(30 * time.Second)

	// Get all sources from the collection
	allSources := ml.Sources.AllSources()

	ml.loadAllServers(ctx, allSources, allKnownSourceIDs)

	glog.Info("MCP loader leader operations launched")
	return nil
}

// ReloadParsing re-parses all config files into in-memory collections.
// Called by the unified loader before computing combined source IDs for leader writes.
func (ml *MCPLoader) ReloadParsing() error {
	var errs []error
	for _, path := range ml.state.Paths() {
		if err := ml.parseAndMerge(path); err != nil {
			errs = append(errs, fmt.Errorf("unable to reload MCP sources from %s: %w", path, err))
		}
	}
	return errors.Join(errs...)
}

// parseAndMerge parses a config file and merges its MCP sources into the collection.
func (ml *MCPLoader) parseAndMerge(path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %v", path, err)
	}

	config, err := basecatalog.ReadSourceConfig(path)
	if err != nil {
		return err
	}

	return ml.updateSources(path, config)
}

// updateSources merges MCP catalog sources from the config into the Sources collection.
func (ml *MCPLoader) updateSources(path string, config *basecatalog.SourceConfig) error {
	sources := make(map[string]basecatalog.MCPSource, len(config.MCPCatalogs))

	for _, source := range config.MCPCatalogs {
		glog.Infof("reading MCP catalog config type %s...", source.Type)
		if source.ID == "" {
			return fmt.Errorf("invalid MCP source: missing id")
		}
		if _, exists := sources[source.ID]; exists {
			return fmt.Errorf("invalid MCP source: duplicate id %s", source.ID)
		}

		if err := ValidateServerFilters(source.IncludedServers, source.ExcludedServers); err != nil {
			return fmt.Errorf("invalid MCP source %s: %w", source.ID, err)
		}

		// Set the origin path so relative paths in properties can be resolved
		// relative to this config file's directory
		source.Origin = path
		sources[source.ID] = source
		glog.Infof("loaded MCP source %s of type %s", source.ID, source.Type)
	}

	if config.NamedQueries != nil {
		filtered := basecatalog.FilterNamedQueriesByAssetType(config.NamedQueries, basecatalog.AssetTypeMCPServers)
		if len(filtered) > 0 {
			return ml.Sources.MergeWithNamedQueries(path, sources, filtered)
		}
	}
	return ml.Sources.Merge(path, sources)
}

// loadAllServers loads MCP servers from all configured sources.
// It returns immediately after launching per-source goroutines. Each goroutine
// continues watching for file changes until ctx is cancelled.
func (ml *MCPLoader) loadAllServers(ctx context.Context, sources map[string]basecatalog.MCPSource, allKnownSourceIDs mapset.Set[string]) {
	// First pass: classify sources and handle disabled/unknown types synchronously.
	enabledSourceIDs := mapset.NewSet[string]()
	allSourceIDs := mapset.NewSet[string]()

	type providerEntry struct {
		source   basecatalog.MCPSource
		provider MCPProvider
		filter   *ServerFilter
	}
	var toLoad []providerEntry

	for _, source := range sources {
		allSourceIDs.Add(source.ID)

		if !source.IsEnabled() {
			glog.Infof("Skipping disabled MCP source: %s", source.Name)
			basecatalog.SaveSourceStatus(ml.services.CatalogSourceRepository, source.ID, basecatalog.SourceStatusDisabled, "")
			continue
		}

		providerFunc, ok := GetMCPProvider(source.Type)
		if !ok {
			glog.Warningf("Unknown MCP provider type: %s (source: %s)", source.Type, source.Name)
			basecatalog.SaveSourceStatus(ml.services.CatalogSourceRepository, source.ID, basecatalog.SourceStatusError, fmt.Sprintf("unknown MCP provider type: %s", source.Type))
			continue
		}

		provider, err := providerFunc(source)
		if err != nil {
			glog.Errorf("Error creating MCP provider for source %s: %v", source.Name, err)
			basecatalog.SaveSourceStatus(ml.services.CatalogSourceRepository, source.ID, basecatalog.SourceStatusError, err.Error())
			continue
		}

		filter, err := NewServerFilterFromSource(&source)
		if err != nil {
			glog.Errorf("Error building server filter for source %s: %v", source.Name, err)
			basecatalog.SaveSourceStatus(ml.services.CatalogSourceRepository, source.ID, basecatalog.SourceStatusError, err.Error())
			continue
		}

		// Mark as enabled before launching goroutines, so removeServersFromMissingSources
		// does not bulk-delete this source's data before the goroutine has a chance to
		// load it. If all servers fail, finalizeBatch calls removeOrphanedServersFromSource
		// which cleans up individual stale entries.
		enabledSourceIDs.Add(source.ID)
		toLoad = append(toLoad, providerEntry{source, provider, filter})
	}

	// Clean up servers from sources that are no longer configured or enabled.
	if err := ml.removeServersFromMissingSources(enabledSourceIDs, allSourceIDs, allKnownSourceIDs); err != nil {
		glog.Errorf("failed to remove servers from missing sources: %v", err)
	}

	// Launch a goroutine per source. Each goroutine gets its own child context so
	// it can cancel the provider (closing the channel) instead of draining it.
	// TrackWrite is called before launching so that WaitForInflightWrites cannot
	// return before the goroutine's initial batch is done.
	for _, entry := range toLoad {
		source := entry.source
		provider := entry.provider
		filter := entry.filter
		sourceCtx, sourceCancel := context.WithCancel(ctx)
		ml.state.TrackWrite()
		glog.Infof("Loading MCP servers from source: %s (id: %s)", source.Name, source.ID)
		go ml.loadServersFromProvider(sourceCtx, sourceCancel, source.ID, provider, filter)
	}
}

// loadServersFromProvider consumes records from provider until the channel closes
// or ctx is cancelled. A zero-value record (sentinel) marks the end of a batch:
// orphan cleanup and source status are updated at that point, and batch state
// resets for the next batch. This allows the provider to be long-lived and
// re-emit records when the underlying file changes.
//
// The caller must have called ml.state.TrackWrite() before launching this as a
// goroutine. This function calls ml.state.WriteComplete() exactly once, when the
// initial batch is done, so that WaitForInflightWrites can unblock tests that
// call it after PerformLeaderOperations returns.
//
// cancel is the CancelFunc for ctx. Calling it stops the provider goroutine
// (which selects on ctx.Done()) and causes the provider channel to close,
// allowing this function to return without draining the channel.
func (ml *MCPLoader) loadServersFromProvider(ctx context.Context, cancel context.CancelFunc, sourceID string, provider MCPProvider, filter *ServerFilter) {
	defer cancel() // ensure the provider goroutine exits when this function returns

	recordChan := provider.Servers(ctx)

	validServerNames := mapset.NewSet[string]()
	var failedServers []string
	successCount := 0

	// releaseInitialBatch is called exactly once to release the TrackWrite slot
	// reserved by the caller. It fires on the first sentinel or channel-close,
	// signalling that the initial batch of writes is complete.
	releaseInitialBatch := sync.OnceFunc(ml.state.WriteComplete)
	defer releaseInitialBatch() // ensure TrackWrite is released even on panic or unexpected return

	finalizeBatch := func() {
		if ctx.Err() != nil {
			return
		}
		// Skip orphan cleanup on complete failure to preserve stale-but-functional
		// data until the error is fixed.
		completeFailure := validServerNames.Cardinality() == 0 && len(failedServers) > 0
		if !completeFailure {
			ml.state.TrackWrite()
			err := ml.removeOrphanedServersFromSource(sourceID, validServerNames)
			ml.state.WriteComplete()
			if err != nil {
				glog.Warningf("Failed to remove orphaned servers from source %s: %v", sourceID, err)
			}
		}
		if len(failedServers) > 0 {
			if successCount > 0 {
				basecatalog.SaveSourceStatus(ml.services.CatalogSourceRepository, sourceID, basecatalog.SourceStatusPartiallyAvailable,
					(&MCPPartiallyAvailableError{FailedServers: failedServers}).Error())
			} else {
				basecatalog.SaveSourceStatus(ml.services.CatalogSourceRepository, sourceID, basecatalog.SourceStatusError,
					fmt.Sprintf("All MCP servers failed to load from source %s (failed: %v)", sourceID, failedServers))
			}
		} else {
			basecatalog.SaveSourceStatus(ml.services.CatalogSourceRepository, sourceID, basecatalog.SourceStatusAvailable, "")
		}
	}

	for record := range recordChan {
		if ctx.Err() != nil {
			glog.Info("Context cancelled, stopping MCP server processing")
			releaseInitialBatch()
			return
		}

		if !ml.state.ShouldWriteDatabase() {
			glog.Info("No longer leader, stopping MCP server processing")
			releaseInitialBatch()
			// cancel() via defer will close ctx, which causes the provider to exit
			// and close the channel — no need to drain here.
			return
		}

		// Sentinel: end of a batch. Finalize, reset state, and continue watching.
		if record.Server == nil && record.Error == nil {
			glog.Infof("%s: loaded %d MCP servers", sourceID, successCount)
			finalizeBatch()
			releaseInitialBatch() // signal that the initial batch of writes is done
			validServerNames = mapset.NewSet[string]()
			failedServers = nil
			successCount = 0
			continue
		}

		if record.Error != nil {
			glog.Errorf("Error from MCP provider: %v", record.Error)
			failedServers = append(failedServers, fmt.Sprintf("(provider error: %v)", record.Error))
			continue
		}

		ml.setServerSourceID(record.Server, sourceID)

		serverName := ""
		if record.Server.GetAttributes() != nil && record.Server.GetAttributes().Name != nil {
			serverName = *record.Server.GetAttributes().Name
		}

		if !filter.Allows(serverName) {
			if serverName == "" {
				glog.Warningf("MCP server with no name was dropped by includedServers/excludedServers filter; check includedServers/excludedServers configuration")
			} else {
				glog.V(2).Infof("Skipping excluded MCP server: %s", serverName)
			}
			continue
		}

		if serverName != "" {
			validServerNames.Add(serverName)
		}

		if err := ml.updateDatabase(record); err != nil {
			glog.Errorf("Error saving MCP server: %v", err)
			if serverName != "" {
				failedServers = append(failedServers, serverName)
			} else {
				failedServers = append(failedServers, "(unknown)")
			}
			continue
		}

		successCount++

		for _, handler := range ml.handlers {
			if err := handler(ctx, record); err != nil {
				glog.Warningf("Event handler error: %v", err)
			}
		}
	}

	// Channel closed without a sentinel (one-shot provider or context cancelled).
	// Finalize the current batch if any records were processed.
	if successCount > 0 || len(failedServers) > 0 {
		finalizeBatch()
	}
	releaseInitialBatch()
}

// setServerSourceID sets the source_id property on an MCP server
func (ml *MCPLoader) setServerSourceID(server *models.MCPServerImpl, sourceID string) {
	sourceIDProp := mrmodels.NewStringProperty("source_id", sourceID, false)

	if server.Properties == nil {
		props := []mrmodels.Properties{sourceIDProp}
		server.Properties = &props
	} else {
		// Check if source_id already exists and update it, otherwise append
		found := false
		props := *server.Properties
		for i := range props {
			if props[i].Name == "source_id" {
				props[i] = sourceIDProp
				found = true
				break
			}
		}
		if !found {
			props = append(props, sourceIDProp)
			server.Properties = &props
		}
	}
}

// updateDatabase saves an MCP server and its tools to the database
func (ml *MCPLoader) updateDatabase(record MCPServerProviderRecord) error {
	ml.state.TrackWrite()
	defer ml.state.WriteComplete()

	saved, err := ml.services.MCPServerRepository.Save(record.Server)
	if err != nil {
		return fmt.Errorf("error saving MCP server: %w", err)
	}

	serverID := saved.GetID()
	if serverID == nil {
		return fmt.Errorf("saved MCP server has no ID")
	}

	// Replace tools: remove existing then insert current set
	if err := ml.services.MCPServerToolRepository.DeleteByParentID(*serverID); err != nil {
		return fmt.Errorf("error deleting existing tools for MCP server: %w", err)
	}

	for _, toolRecord := range record.Tools {
		tool := buildMCPServerTool(saved, toolRecord)
		if _, err := ml.services.MCPServerToolRepository.Save(tool, serverID); err != nil {
			glog.Errorf("Error saving MCP server tool %s: %v", toolRecord.Name, err)
		}
	}

	return nil
}

// buildMCPServerTool constructs a MCPServerTool entity from a MCPToolRecord.
// The tool name is qualified with the server's composite name (base_name@version).
func buildMCPServerTool(server models.MCPServer, toolRecord MCPToolRecord) models.MCPServerTool {
	name := toolRecord.Name

	attr := server.GetAttributes()
	if attr != nil && attr.Name != nil {
		// Build qualified name: base_name@version:tool_name
		serverName := *attr.Name
		if props := server.GetProperties(); props != nil {
			for _, prop := range *props {
				if prop.Name == "version" && prop.StringValue != nil && *prop.StringValue != "" {
					serverName = fmt.Sprintf("%s@%s", serverName, *prop.StringValue)
					break
				}
			}
		}
		name = fmt.Sprintf("%s:%s", serverName, toolRecord.Name)
	}

	impl := &models.MCPServerToolImpl{
		Attributes: &models.MCPServerToolAttributes{
			Name: &name,
		},
	}

	var properties []mrmodels.Properties
	if toolRecord.AccessType != nil {
		properties = append(properties, mrmodels.NewStringProperty("accessType", *toolRecord.AccessType, false))
	}
	if toolRecord.Description != nil {
		properties = append(properties, mrmodels.NewStringProperty("description", *toolRecord.Description, false))
	}
	if toolRecord.Schema != nil {
		properties = append(properties, mrmodels.NewStringProperty("schema", *toolRecord.Schema, false))
	}
	if len(toolRecord.Parameters) > 0 {
		if jsonBytes, err := json.Marshal(toolRecord.Parameters); err == nil {
			properties = append(properties, mrmodels.NewStringProperty("parameters", string(jsonBytes), false))
		}
	}
	impl.Properties = &properties

	return impl
}

// removeServersFromMissingSources removes servers from sources that are no longer enabled,
// and cleans up CatalogSource status records for sources completely removed from config.
// allKnownSourceIDs is the union of model and MCP source IDs to prevent cross-contamination
// when cleaning up shared CatalogSource records.
func (ml *MCPLoader) removeServersFromMissingSources(enabledSourceIDs, allSourceIDs, allKnownSourceIDs mapset.Set[string]) error {
	// Get all source IDs from the database
	dbSourceIDs, err := ml.services.MCPServerRepository.GetDistinctSourceIDs()
	if err != nil {
		return fmt.Errorf("error getting distinct source IDs: %w", err)
	}

	// Find source IDs in DB that are not in the enabled set (disabled or removed)
	for _, dbSourceID := range dbSourceIDs {
		if !enabledSourceIDs.Contains(dbSourceID) {
			glog.Infof("Removing MCP servers from source: %s", dbSourceID)
			// List servers from this source to delete their tools first
			listOptions := models.MCPServerListOptions{
				SourceIDs: &[]string{dbSourceID},
			}
			result, listErr := ml.services.MCPServerRepository.List(listOptions)
			if listErr == nil && result != nil {
				for _, server := range result.Items {
					if server.GetID() != nil {
						if err := ml.services.MCPServerToolRepository.DeleteByParentID(*server.GetID()); err != nil {
							glog.Errorf("Error deleting tools for server during source cleanup: %v", err)
						}
					}
				}
			}
			// Now safe to delete servers
			if err := ml.services.MCPServerRepository.DeleteBySource(dbSourceID); err != nil {
				glog.Errorf("Error deleting servers from source %s: %v", dbSourceID, err)
			}

			// Only delete the shared CatalogSource status record if the source is absent
			// from ALL plugin configs — not just the MCP config. allKnownSourceIDs is
			// the union of every plugin's configured source IDs, so if another plugin
			// (model, skill, agent) owns this source we must not delete its status row.
			if !allSourceIDs.Contains(dbSourceID) && !allKnownSourceIDs.Contains(dbSourceID) {
				glog.Infof("Removing status for MCP source %s (no longer in any config)", dbSourceID)
				if delErr := ml.services.CatalogSourceRepository.Delete(dbSourceID); delErr != nil {
					glog.Errorf("failed to delete status for MCP source %s: %v", dbSourceID, delErr)
				}
			}
		}
	}

	// Clean up CatalogSource records for sources no longer in any loader's config.
	// The protected set is the union of this loader's own source IDs and the combined
	// known source IDs from all loaders, preventing cross-contamination.
	protectedSourceIDs := allSourceIDs.Union(allKnownSourceIDs)
	if err := basecatalog.CleanupOrphanedCatalogSources(ml.services.CatalogSourceRepository, protectedSourceIDs); err != nil {
		glog.Errorf("failed to cleanup orphaned MCP catalog sources: %v", err)
	}

	return nil
}

// removeOrphanedServersFromSource removes servers that are no longer in the source
func (ml *MCPLoader) removeOrphanedServersFromSource(sourceID string, validServerNames mapset.Set[string]) error {
	// Get all servers from this source
	listOptions := models.MCPServerListOptions{
		SourceIDs: &[]string{sourceID},
	}

	result, err := ml.services.MCPServerRepository.List(listOptions)
	if err != nil {
		return fmt.Errorf("error listing servers from source %s: %w", sourceID, err)
	}

	if result == nil {
		return nil
	}

	// Delete servers that are no longer in the valid set
	for _, server := range result.Items {
		attrs := server.GetAttributes()
		if attrs == nil || attrs.Name == nil {
			continue
		}

		if !validServerNames.Contains(*attrs.Name) {
			glog.Infof("Removing orphaned MCP server: %s (source: %s)", *attrs.Name, sourceID)
			if server.GetID() != nil {
				if err := ml.services.MCPServerToolRepository.DeleteByParentID(*server.GetID()); err != nil {
					glog.Errorf("Error deleting tools for server %s: %v", *attrs.Name, err)
				}
				err := ml.services.MCPServerRepository.DeleteByID(*server.GetID())
				if err != nil {
					glog.Errorf("Error deleting server %s: %v", *attrs.Name, err)
				}
			}
		}
	}

	return nil
}
