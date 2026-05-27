package plugin

import (
	"context"
	"fmt"
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
// and KnownSourceIDs for free.
type PluginBase struct {
	cfg PluginBaseConfig
}

// NewPluginBase creates a PluginBase with the given configuration.
func NewPluginBase(cfg PluginBaseConfig) PluginBase {
	return PluginBase{cfg: cfg}
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
		glog.Errorf("unable to watch file (%s): %v", path, err)
		return
	}

	for range changes {
		glog.Infof("Config file changed, reloading %s sources: %s", pb.cfg.Name, path)
		pb.cfg.Loader.ReloadParsing()

		if pb.cfg.State.ShouldWriteDatabase() {
			allKnownSourceIDs := CollectAllSourceIDs()
			if err := pb.cfg.Loader.PerformLeaderOperations(ctx, allKnownSourceIDs); err != nil {
				glog.Errorf("unable to perform %s leader writes on reload: %v", pb.cfg.Name, err)
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
		pb.cfg.State.SetLeader(false)
		return fmt.Errorf("%s leader operations: %w", pb.cfg.Name, err)
	}

	if pb.cfg.OnLeaderReady != nil {
		if err := pb.cfg.OnLeaderReady(ctx); err != nil {
			pb.cfg.State.SetLeader(false)
			return fmt.Errorf("%s post-leader setup: %w", pb.cfg.Name, err)
		}
	}

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
