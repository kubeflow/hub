package plugin

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/golang/glog"
)

// PluginBaseConfig holds the dependencies needed to construct a PluginBase.
type PluginBaseConfig struct {
	// Name is used for log messages and error wrapping.
	Name string

	// State manages leader election, file watchers, and write tracking.
	State LifecycleState

	// Loader performs config parsing and leader write operations.
	Loader Reloader

	// SourceIDs returns the set of source IDs known to this plugin.
	SourceIDs func() mapset.Set[string]

	// FileWatcher provides file change notifications.
	FileWatcher FileWatcher

	// OnLeaderReady is called after leader operations succeed.
	// Optional — nil means no post-leader hook.
	OnLeaderReady func(ctx context.Context) error
}

// PluginBase provides the shared lifecycle implementation for catalog plugins.
// Embed this in concrete plugin structs to get Start, Stop, OnBecomeLeader,
// Healthy, and KnownSourceIDs for free.
type PluginBase struct {
	cfg     PluginBaseConfig
	healthy atomic.Bool
}

// NewPluginBase creates a PluginBase with the given configuration.
func NewPluginBase(cfg PluginBaseConfig) *PluginBase {
	pb := &PluginBase{cfg: cfg}
	pb.healthy.Store(true)
	return pb
}

// Healthy returns true if the plugin has not encountered any runtime errors.
func (pb *PluginBase) Healthy() bool {
	return pb.healthy.Load()
}

// Start sets up file watchers, parses all configs, and launches
// background goroutines to watch for config file changes.
func (pb *PluginBase) Start(ctx context.Context) error {
	glog.Infof("Starting %s plugin in read-only mode (standby)", pb.cfg.Name)

	watcherCtx, err := pb.cfg.State.SetupFileWatchers(ctx)
	if err != nil {
		return err
	}

	if err := pb.cfg.Loader.ParseAllConfigs(); err != nil {
		return fmt.Errorf("%s config: %w", pb.cfg.Name, err)
	}

	for _, path := range pb.cfg.State.Paths() {
		go pb.watchFile(watcherCtx, path)
	}

	glog.Infof("%s plugin read-only mode initialized", pb.cfg.Name)
	return nil
}

func (pb *PluginBase) watchFile(ctx context.Context, path string) {
	changes, err := pb.cfg.FileWatcher.Path(ctx, path)
	if err != nil {
		pb.healthy.Store(false)
		glog.Errorf("unable to watch file (%s): %v", path, err)
		return
	}

	for range changes {
		glog.Infof("Config file changed, reloading %s sources: %s", pb.cfg.Name, path)
		if err := pb.cfg.Loader.ReloadParsing(); err != nil {
			pb.healthy.Store(false)
			glog.Errorf("unable to reload %s config: %v", pb.cfg.Name, err)
			continue
		}
		pb.healthy.Store(true)

		if pb.cfg.State.ShouldWriteDatabase() {
			allKnownSourceIDs := CollectAllSourceIDs()
			if err := pb.cfg.Loader.PerformLeaderOperations(ctx, allKnownSourceIDs); err != nil {
				pb.healthy.Store(false)
				glog.Errorf("unable to perform %s leader writes on reload: %v", pb.cfg.Name, err)
			} else {
				// Wait for the initial load batch to finish before marking healthy,
				// so API clients don't see stale data after a config reload.
				pb.cfg.State.WaitForInflightWrites(30 * time.Second)
				pb.healthy.Store(true)
			}
		}
	}
}

// OnBecomeLeader handles the full leader lifecycle: sets leader state,
// runs leader operations, calls the optional OnLeaderReady hook, then
// blocks until the context is cancelled (leadership lost).
func (pb *PluginBase) OnBecomeLeader(ctx context.Context) error {
	if pb.cfg.State.IsLeader() {
		return fmt.Errorf("already in leader mode")
	}
	pb.cfg.State.SetLeader(true)

	glog.Infof("%s plugin becoming leader", pb.cfg.Name)

	allKnownSourceIDs := CollectAllSourceIDs()
	if err := pb.cfg.Loader.PerformLeaderOperations(ctx, allKnownSourceIDs); err != nil {
		pb.healthy.Store(false)
		pb.cfg.State.SetLeader(false)
		return fmt.Errorf("%s leader operations: %w", pb.cfg.Name, err)
	}
	// Wait for the initial load batch to finish before calling OnLeaderReady and
	// marking healthy, so the plugin is fully populated when readiness probes fire.
	pb.cfg.State.WaitForInflightWrites(30 * time.Second)

	if pb.cfg.OnLeaderReady != nil {
		if err := pb.cfg.OnLeaderReady(ctx); err != nil {
			pb.healthy.Store(false)
			pb.cfg.State.SetLeader(false)
			return fmt.Errorf("%s post-leader setup: %w", pb.cfg.Name, err)
		}
	}

	pb.healthy.Store(true)
	glog.Infof("%s plugin leader mode active", pb.cfg.Name)
	<-ctx.Done()

	pb.cfg.State.WaitForInflightWrites(5 * time.Second)
	pb.cfg.State.SetLeader(false)

	glog.Infof("%s plugin leader mode stopped", pb.cfg.Name)
	return ctx.Err()
}

// Stop cancels file watchers and waits for inflight writes.
func (pb *PluginBase) Stop(_ context.Context) error {
	pb.cfg.State.StopFileWatchers()
	pb.cfg.State.WaitForInflightWrites(10 * time.Second)
	return nil
}

// KnownSourceIDs returns the set of source IDs this plugin manages.
func (pb *PluginBase) KnownSourceIDs() mapset.Set[string] {
	return pb.cfg.SourceIDs()
}
