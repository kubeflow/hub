package plugin

import (
	"context"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
)

// Reloader defines the operations a concrete loader must provide
// for the shared plugin lifecycle (PluginBase) to work.
// Both modelcatalog.ModelLoader and mcpcatalog.MCPLoader satisfy this.
type Reloader interface {
	ParseAllConfigs() error
	ReloadParsing()
	PerformLeaderOperations(ctx context.Context, allKnownSourceIDs mapset.Set[string]) error
}

// LifecycleState provides shared state management for a plugin's lifecycle.
// basecatalog.BaseLoader satisfies this interface.
type LifecycleState interface {
	IsLeader() bool
	SetLeader(bool)
	ShouldWriteDatabase() bool
	SetupFileWatchers(ctx context.Context) (context.Context, error)
	StopFileWatchers()
	WaitForInflightWrites(timeout time.Duration)
	Paths() []string
}

// FileWatcher provides file change notifications.
// basecatalog's monitor singleton satisfies this interface.
type FileWatcher interface {
	Path(ctx context.Context, path string) (<-chan struct{}, error)
}
