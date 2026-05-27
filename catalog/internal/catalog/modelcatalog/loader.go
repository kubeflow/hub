package modelcatalog

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/golang/glog"
	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	sharedmodels "github.com/kubeflow/hub/catalog/internal/db/models"
	"github.com/kubeflow/hub/catalog/internal/db/service"  // for type name constants
	mrmodels "github.com/kubeflow/hub/internal/platform/db/entity"
)

// PartiallyAvailableError indicates that a source loaded some models successfully
// but encountered errors with others.
type PartiallyAvailableError struct {
	FailedModels []string
}

func (e *PartiallyAvailableError) Error() string {
	return fmt.Sprintf("Failed to fetch some models, ensure models exist and are accessible with given credentials. Failed models: %v", e.FailedModels)
}

func (e *PartiallyAvailableError) Is(target error) bool {
	_, ok := target.(*PartiallyAvailableError)
	return ok
}

// ErrPartiallyAvailable is used with errors.Is() to check for this error type.
var ErrPartiallyAvailable error = &PartiallyAvailableError{}

// ModelProviderRecord contains one model and its associated artifacts.
type ModelProviderRecord struct {
	Model     models.CatalogModel
	Artifacts []sharedmodels.CatalogArtifact
	// Error can be set here to emit successfully loaded models before updating source status err.
	Error error
}

// ModelProviderFunc emits models and related data in the channel it returns. It is
// expected to spawn a goroutine and return immediately. The returned channel must
// close when the goroutine ends. The goroutine should end when the context is
// canceled, but may end sooner.
//
// The function may emit a record with a nil Model to indicate that the
// complete set of models has been sent.
type ModelProviderFunc func(ctx context.Context, source *basecatalog.ModelSource, reldir string) (<-chan ModelProviderRecord, error)

var registeredModelProviders = map[string]ModelProviderFunc{}

func RegisterModelProvider(name string, callback ModelProviderFunc) error {
	if _, exists := registeredModelProviders[name]; exists {
		return fmt.Errorf("provider type %s already exists", name)
	}
	registeredModelProviders[name] = callback
	return nil
}

// LoaderEventHandler is the definition of a function called after a model is loaded.
type LoaderEventHandler func(ctx context.Context, record ModelProviderRecord) error

// ModelLoader is the delegate loader for model catalogs.
// It uses external state (LoaderState) for leader operations and write tracking.
type ModelLoader struct {
	state basecatalog.LoaderState

	// Sources contains current source information loaded from the configuration files.
	Sources *SourceCollection

	// Labels contains current labels loaded from the configuration files.
	Labels *LabelCollection

	services      Services
	handlers      []LoaderEventHandler
	loadedSources map[string]bool // tracks which source IDs have been loaded
}

// NewModelLoader creates a new ModelLoader with external state
func NewModelLoader(services Services, state basecatalog.LoaderState) *ModelLoader {
	paths := state.Paths()
	// Convert paths to absolute for consistent origin ordering.
	// This matches how loadOne converts paths before calling Merge.
	absPaths := make([]string, 0, len(paths))
	for _, p := range paths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			// Fall back to original path if conversion fails
			absPath = p
		}
		absPaths = append(absPaths, absPath)
	}

	return &ModelLoader{
		state:         state,
		Sources:       NewSourceCollection(absPaths...),
		Labels:        NewLabelCollection(),
		services:      services,
		loadedSources: map[string]bool{},
	}
}

// RegisterEventHandler adds a function that will be called for every
// successfully processed record. This should be called before initialization.
//
// Handlers are called in the order they are registered.
func (l *ModelLoader) RegisterEventHandler(fn LoaderEventHandler) {
	l.handlers = append(l.handlers, fn)
}

// ParseAllConfigs parses all config files into in-memory collections.
// This is called by the unified loader during initialization.
func (l *ModelLoader) ParseAllConfigs() error {
	for _, path := range l.state.Paths() {
		if err := l.parseAndMerge(path); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}
	return nil
}

// PerformLeaderOperations executes database write operations.
// allKnownSourceIDs is the union of model and MCP source IDs, used to prevent
// cross-contamination when cleaning up shared CatalogSource records.
// This is called by the unified loader when becoming leader.
func (l *ModelLoader) PerformLeaderOperations(ctx context.Context, allKnownSourceIDs mapset.Set[string]) error {
	return l.performLeaderWrites(ctx, allKnownSourceIDs)
}

// ReloadParsing re-parses all config files into in-memory collections.
// Called by the unified loader before computing combined source IDs for leader writes.
func (l *ModelLoader) ReloadParsing() {
	for _, path := range l.state.Paths() {
		if err := l.parseAndMerge(path); err != nil {
			glog.Errorf("unable to reload model sources from %s: %v", path, err)
		}
	}
}

// performLeaderWrites executes database write operations: removing orphaned
// models and loading all models from sources.
func (l *ModelLoader) performLeaderWrites(ctx context.Context, allKnownSourceIDs mapset.Set[string]) error {
	if err := l.removeModelsFromMissingSources(allKnownSourceIDs); err != nil {
		return fmt.Errorf("failed to remove models from missing sources: %w", err)
	}
	return l.loadAllModels(ctx)
}

// parseAndMerge parses a config file and merges its sources/labels into the collections.
func (l *ModelLoader) parseAndMerge(path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %v", path, err)
	}

	config, err := l.read(path)
	if err != nil {
		return err
	}

	if err = l.updateSources(path, config); err != nil {
		return err
	}

	return l.updateLabels(path, config)
}

func (l *ModelLoader) loadAllModels(ctx context.Context) error {
	l.loadedSources = map[string]bool{}

	return l.updateDatabase(ctx)
}

func (l *ModelLoader) read(path string) (*basecatalog.SourceConfig, error) {
	config, err := basecatalog.ReadSourceConfig(path)
	if err != nil {
		return nil, err
	}

	// Validate named queries if present
	if config.NamedQueries != nil {
		if err := basecatalog.ValidateNamedQueries(config.NamedQueries); err != nil {
			return nil, fmt.Errorf("invalid named queries in %s: %w", path, err)
		}
	}

	// Note: We intentionally do NOT filter disabled sources or apply defaults here.
	// This allows field-level merging in SourceCollection to work correctly:
	// - A base source with enabled=false can be enabled by a user override with just id + enabled=true
	// - Defaults are applied after merging in SourceCollection.merged()

	return config, nil
}

func (l *ModelLoader) updateSources(path string, config *basecatalog.SourceConfig) error {
	modelCatalogs := config.GetModelCatalogs()
	sources := make(map[string]basecatalog.ModelSource, len(modelCatalogs))

	for _, source := range modelCatalogs {
		glog.Infof("reading config type %s...", source.Type)
		id := source.GetId()
		if len(id) == 0 {
			return fmt.Errorf("invalid source: missing id")
		}
		if _, exists := sources[id]; exists {
			return fmt.Errorf("invalid source: duplicate id %s", id)
		}

		// Validate includedModels/excludedModels patterns early (only if set)
		if err := ValidateSourceFilters(source.IncludedModels, source.ExcludedModels); err != nil {
			return fmt.Errorf("invalid source %s: %w", id, err)
		}

		// Set the origin path so relative paths in properties can be resolved
		// relative to this config file's directory
		source.Origin = path
		sources[id] = source
		glog.Infof("loaded source %s of type %s", id, source.Type)
	}

	// Use MergeWithNamedQueries for named queries scoped to models; skip others.
	if config.NamedQueries != nil {
		filtered := basecatalog.FilterNamedQueriesByAssetType(config.NamedQueries, basecatalog.AssetTypeModels)
		if len(filtered) > 0 {
			return l.Sources.MergeWithNamedQueries(path, sources, filtered)
		}
	}
	return l.Sources.Merge(path, sources)
}

func (l *ModelLoader) updateLabels(path string, config *basecatalog.SourceConfig) error {
	// Merge labels from config into the label collection
	if config.Labels == nil {
		// No labels in config, but we still need to clear any previous labels from this origin
		return l.Labels.Merge(path, []map[string]any{})
	}

	// Validate that each label has a required "name" field
	for i, label := range config.Labels {
		if name, ok := label["name"]; !ok || name == "" {
			return fmt.Errorf("invalid label at index %d: missing required 'name' field", i)
		}
	}

	return l.Labels.Merge(path, config.Labels)
}

func (l *ModelLoader) updateDatabase(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	l.state.SetCloser(cancel)

	records := l.readProviderRecords(ctx)

	go func() {
		for record := range records {
			// Check if we're still the leader before each write
			if !l.state.ShouldWriteDatabase() {
				glog.Info("No longer leader, stopping database writes")
				return
			}

			// Check context cancellation
			if ctx.Err() != nil {
				glog.Info("Context cancelled, stopping database writes")
				return
			}

			if record.Model == nil {
				continue
			}
			attr := record.Model.GetAttributes()
			if attr == nil || attr.Name == nil {
				continue
			}

			func() {
				// Track this write operation
				l.state.TrackWrite()
				defer l.state.WriteComplete()

				glog.Infof("Loading model %s with %d artifact(s)", *attr.Name, len(record.Artifacts))

				model, err := l.services.CatalogModelRepository.Save(record.Model)
				if err != nil {
					glog.Errorf("%s: unable to save: %v", *attr.Name, err)
					return
				}

				modelID := model.GetID()
				if modelID == nil {
					glog.Errorf("%s: model has no ID after save", *attr.Name)
					return
				}

				// Remove artifacts that existed before.
				err = l.services.CatalogArtifactRepository.DeleteByParentID(service.CatalogModelArtifactTypeName, *modelID)
				if err != nil {
					glog.Errorf("%s: unable to remove old catalog model artifacts: %v", *attr.Name, err)
				}
				err = l.services.CatalogArtifactRepository.DeleteByParentID(service.CatalogMetricsArtifactTypeName, *modelID)
				if err != nil {
					glog.Errorf("%s: unable to remove old catalog metrics artifacts: %v", *attr.Name, err)
				}

				for i, artifact := range record.Artifacts {
					switch {
					case artifact.CatalogModelArtifact != nil:
						if ma, ok := artifact.CatalogModelArtifact.(models.CatalogModelArtifact); ok {
							_, err = l.services.CatalogModelArtifactRepository.Save(ma, modelID)
						} else {
							err = fmt.Errorf("invalid model artifact type: %T", artifact.CatalogModelArtifact)
						}
					case artifact.CatalogMetricsArtifact != nil:
						if ma, ok := artifact.CatalogMetricsArtifact.(models.CatalogMetricsArtifact); ok {
							_, err = l.services.CatalogMetricsArtifactRepository.Save(ma, modelID)
						} else {
							err = fmt.Errorf("invalid metrics artifact type: %T", artifact.CatalogMetricsArtifact)
						}
					default:
						err = errors.New("unknown artifact type")
					}

					if err != nil {
						glog.Errorf("%s, artifact %d: %v", *attr.Name, i, err)
					}
				}

			for _, handler := range l.handlers {
				if err := handler(ctx, record); err != nil {
					glog.Errorf("%s: event handler error: %v", *attr.Name, err)
				}
			}
			}()
		}
	}()

	return nil
}

// readProviderRecords calls the provider for every merged source that hasn't
// been loaded yet, and merges the returned channels together. The returned
// channel is closed when the last provider channel is closed.
func (l *ModelLoader) readProviderRecords(ctx context.Context) <-chan ModelProviderRecord {
	ch := make(chan ModelProviderRecord)
	var wg sync.WaitGroup

	// Get all sources from the merged collection.
	// This allows sparse overrides to work: a user can enable a disabled source
	// with just "id" and "enabled: true", inheriting Type and Properties from the base.
	mergedSources := l.Sources.AllSources()

	for _, source := range mergedSources {
		// Skip disabled sources - only load catalog data from enabled sources
		// Per OpenAPI spec, enabled defaults to true, so nil is treated as enabled
		if source.Enabled != nil && !*source.Enabled {
			// Persist disabled status
			basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, source.Id, basecatalog.SourceStatusDisabled, "")
			continue
		}

		// Skip sources that have already been loaded
		if l.loadedSources[source.Id] {
			continue
		}

		if source.Type == "" {
			glog.Errorf("source %s has no type defined, skipping", source.Id)
			basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, source.Id, basecatalog.SourceStatusError, "source has no type defined")
			continue
		}

		// Mark this source as loaded
		l.loadedSources[source.Id] = true

		glog.Infof("Reading models from %s source %s", source.Type, source.Id)

		registerFunc, ok := registeredModelProviders[source.Type]
		if !ok {
			glog.Errorf("catalog type %s not registered", source.Type)
			basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, source.Id, basecatalog.SourceStatusError, fmt.Sprintf("catalog type %q not registered", source.Type))
			continue
		}

		// Use the source's origin directory for resolving relative paths.
		// This allows sources from different config files (e.g., mounted from
		// different configmaps) to use relative paths correctly.
		sourceDir := filepath.Dir(source.Origin)

		records, err := registerFunc(ctx, &source, sourceDir)
		if err != nil {
			glog.Errorf("error reading catalog type %s with id %s: %v", source.Type, source.Id, err)
			basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, source.Id, basecatalog.SourceStatusError, err.Error())
			continue
		}

		wg.Add(1)
		go func(ctx context.Context, sourceID string) {
			defer wg.Done()

			modelNames := []string{}
			failedModels := []string{}
			statusSaved := false

			for r := range records {
				// Per-model validation errors (Error set, no Model). The Hugging Face
				// provider also sends a nil-Model completion record with
				// ErrPartiallyAvailable; that is handled below as batch completion, not here.
				if r.Error != nil && r.Model == nil && !errors.Is(r.Error, ErrPartiallyAvailable) {
					glog.Errorf("%s: model validation error: %v", sourceID, r.Error)
					failedModels = append(failedModels, r.Error.Error())
					continue
				}

				if r.Model == nil {

					glog.Infof("%s: loaded %d models", sourceID, len(modelNames))

					// Copy the list of model names, then clear it.
					modelNameSet := mapset.NewSet(modelNames...)
					modelNames = modelNames[:0]

					go func() {
						count, err := l.removeOrphanedModelsFromSource(sourceID, modelNameSet)
						if err != nil {
							glog.Errorf("error removing orphaned models: %v", err)
						}
						glog.Infof("%s: cleaned up %d models", sourceID, count)
					}()

					// Only save status if context is still valid (no reload in progress)
					if ctx.Err() == nil {
						successCount := modelNameSet.Cardinality()
						hasPartialFailure := errors.Is(r.Error, ErrPartiallyAvailable)
						hasValidationFailures := len(failedModels) > 0

						if successCount > 0 {
							if hasPartialFailure {
								glog.Warningf("%s: partial error after loading models: %v", sourceID, r.Error)
								basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, sourceID, basecatalog.SourceStatusPartiallyAvailable, r.Error.Error())
							} else if hasValidationFailures {
								errMsg := fmt.Sprintf("Failed to load %d model(s): %v", len(failedModels), failedModels)
								glog.Warningf("%s: %s", sourceID, errMsg)
								basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, sourceID, basecatalog.SourceStatusPartiallyAvailable, errMsg)
							} else {
								basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, sourceID, basecatalog.SourceStatusAvailable, "")
							}
						} else if hasPartialFailure || hasValidationFailures {
							if hasPartialFailure {
								glog.Warningf("%s: all catalog models failed to load from source: %v", sourceID, r.Error)
								basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, sourceID, basecatalog.SourceStatusError, r.Error.Error())
							} else {
								errMsg := fmt.Sprintf("all catalog models failed to load from source %s (failed: %v)", sourceID, failedModels)
								glog.Warningf("%s: %s", sourceID, errMsg)
								basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, sourceID, basecatalog.SourceStatusError, errMsg)
							}
						} else {
							basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, sourceID, basecatalog.SourceStatusAvailable, "")
						}
						statusSaved = true
					}
					continue
				}

				// Set source_id and namespaced name on every returned model.
				l.setModelSourceID(r.Model, sourceID)

				if attr := r.Model.GetAttributes(); attr != nil && attr.Name != nil {
					// Use namespaced name (source_id:model_name)so removeOrphanedModelsFromSource matches DB (which stores namespaced names).
					modelNames = append(modelNames, *attr.Name)
				}

				ch <- r
			}

			// If the channel closed without a nil Model marker and status wasn't already saved,
			// save available status if context is still valid and we processed some models
			if !statusSaved && ctx.Err() == nil && len(modelNames) > 0 {
				basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, sourceID, basecatalog.SourceStatusAvailable, "")
			}
		}(ctx, source.Id)
	}

	go func() {
		defer close(ch)
		wg.Wait()
	}()

	return ch
}

func (l *ModelLoader) setModelSourceID(model models.CatalogModel, sourceID string) {
	if model == nil {
		return
	}

	// Add a source_id property to the model's properties list.. the hard
	// way, because we use pointers to slices for some reason.

	props := model.GetProperties()
	if props == nil {
		if modelImpl, ok := model.(*models.CatalogModelImpl); ok {
			newProps := make([]mrmodels.Properties, 0, 1)
			modelImpl.Properties = &newProps
			props = &newProps
		} else {
			// Can't do anything with this.
			return
		}
	}

	found := false
	for i := range *props {
		if (*props)[i].Name == "source_id" {
			// Already has a source_id, just update it
			(*props)[i].StringValue = &sourceID
			found = true
			break
		}
	}
	if !found {
		*props = append(*props, mrmodels.NewStringProperty("source_id", sourceID, false))
	}

	// Prepend sourceId to the model name so DB uniqueness is (sourceId, modelName).
	// Format: sourceId:modelName — must run for every model so DB and removeOrphanedModelsFromSource stay consistent.
	attr := model.GetAttributes()
	if attr != nil && attr.Name != nil && *attr.Name != "" {
		namespacedName := sourceID + ":" + *attr.Name
		attr.Name = &namespacedName
	}
}

func (l *ModelLoader) removeModelsFromMissingSources(allKnownSourceIDs mapset.Set[string]) error {
	enabledSourceIDs := mapset.NewSet[string]()
	modelSourceIDs := mapset.NewSet[string]()
	for id, source := range l.Sources.AllSources() {
		modelSourceIDs.Add(id)
		if source.Enabled == nil || *source.Enabled {
			enabledSourceIDs.Add(id)
		}
	}

	existingSourceIDs, err := l.services.CatalogModelRepository.GetDistinctSourceIDs()
	if err != nil {
		return fmt.Errorf("unable to retrieve existing source IDs: %w", err)
	}

	for oldSource := range mapset.NewSet(existingSourceIDs...).Difference(enabledSourceIDs).Iter() {
		glog.Infof("Removing models from source %s", oldSource)

		err = l.services.CatalogModelRepository.DeleteBySource(oldSource)
		if err != nil {
			return fmt.Errorf("unable to remove models from source %q: %w", oldSource, err)
		}

		// If the source is completely gone from model config (not just disabled), remove its status too.
		// We check model-only IDs here since existingSourceIDs comes from CatalogModelRepository.
		if !modelSourceIDs.Contains(oldSource) {
			glog.Infof("Removing status for source %s (no longer in model config)", oldSource)
			if delErr := l.services.CatalogSourceRepository.Delete(oldSource); delErr != nil {
				glog.Errorf("failed to delete status for source %s: %v", oldSource, delErr)
			}
		}
	}

	// Clean up CatalogSource records for sources no longer in any loader's config.
	// The protected set is the union of this loader's own source IDs and the combined
	// known source IDs from all loaders, preventing cross-contamination.
	protectedSourceIDs := modelSourceIDs.Union(allKnownSourceIDs)
	if err := basecatalog.CleanupOrphanedCatalogSources(l.services.CatalogSourceRepository, protectedSourceIDs); err != nil {
		glog.Errorf("failed to cleanup orphaned catalog sources: %v", err)
	}

	return nil
}

func (l *ModelLoader) removeOrphanedModelsFromSource(sourceID string, valid mapset.Set[string]) (int, error) {
	list, err := l.services.CatalogModelRepository.List(models.CatalogModelListOptions{
		SourceIDs: &[]string{sourceID},
	})
	if err != nil {
		return 0, fmt.Errorf("unable to list models from source %q: %w", sourceID, err)
	}

	count := 0
	for _, model := range list.Items {
		attr := model.GetAttributes()
		if attr == nil || attr.Name == nil || model.GetID() == nil {
			continue
		}

		if valid.Contains(*attr.Name) {
			continue
		}

		glog.Infof("Removing %s model %s", sourceID, *attr.Name)

		err = l.services.CatalogModelRepository.DeleteByID(*model.GetID())
		if err != nil {
			return count, fmt.Errorf("unable to remove model %d (%s from source %s): %w", *model.GetID(), *attr.Name, sourceID, err)
		}
		count++
	}

	return count, nil
}
