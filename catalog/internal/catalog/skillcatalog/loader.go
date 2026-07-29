package skillcatalog

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/golang/glog"
	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
)

// SkillLoader handles loading skill data from YAML configuration files.
type SkillLoader struct {
	state basecatalog.LoaderState

	Sources  *SkillSourceCollection
	services Services

	closerMu sync.Mutex
	closer   func()
}

func (l *SkillLoader) setCloser(closer func()) {
	l.closerMu.Lock()
	defer l.closerMu.Unlock()
	if l.closer != nil {
		l.closer()
	}
	l.closer = closer
}

func NewSkillLoader(services Services, state basecatalog.LoaderState) *SkillLoader {
	paths := state.Paths()
	return &SkillLoader{
		state:    state,
		Sources:  NewSkillSourceCollection(paths...),
		services: services,
	}
}

func (l *SkillLoader) ParseAllConfigs() error {
	glog.Infof("Initializing %s loader - parsing configs", "skill")

	for _, path := range l.state.Paths() {
		if err := l.parseAndMerge(path); err != nil {
			return fmt.Errorf("failed to parse skill config %s: %w", path, err)
		}
	}

	glog.Infof("%s loader config parsing complete", "skill")
	return nil
}

func (l *SkillLoader) PerformLeaderOperations(ctx context.Context, allKnownSourceIDs mapset.Set[string]) error {
	glog.Infof("%s loader performing leader operations", "skill")

	ctx, cancel := context.WithCancel(ctx)
	l.setCloser(cancel)

	allSources := l.Sources.AllSources()

	// TODO: Implement leader write operations for each source.
	// Pattern: iterate sources, load data from provider, save to DB via services.
	_ = ctx
	_ = allSources
	_ = allKnownSourceIDs

	glog.Infof("%s loader leader operations complete", "skill")
	return nil
}

func (l *SkillLoader) ReloadParsing() error {
	var errs []error
	for _, path := range l.state.Paths() {
		if err := l.parseAndMerge(path); err != nil {
			errs = append(errs, fmt.Errorf("unable to reload skill sources from %s: %w", path, err))
		}
	}
	return errors.Join(errs...)
}

func (l *SkillLoader) parseAndMerge(path string) error {
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

func (l *SkillLoader) updateSources(path string, config *basecatalog.SourceConfig) error {
	sources := make(map[string]basecatalog.PluginSource, len(config.SkillCatalogs))

	for _, source := range config.SkillCatalogs {
		glog.Infof("reading skill catalog config type %s...", source.Type)
		if source.GetId() == "" {
			return fmt.Errorf("invalid skill source: missing id")
		}
		if _, exists := sources[source.GetId()]; exists {
			return fmt.Errorf("invalid skill source: duplicate id %s", source.GetId())
		}

		source.Origin = path
		sources[source.GetId()] = source
		glog.Infof("loaded skill source %s of type %s", source.GetId(), source.Type)
	}

	return l.Sources.Merge(path, sources)
}
