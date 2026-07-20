package basecatalog

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
)

// LoaderState provides access to shared loader state for child loaders.
// This interface allows delegate loaders to share leader state, write tracking,
// and file paths without embedding BaseLoader.
type LoaderState interface {
	IsLeader() bool
	ShouldWriteDatabase() bool
	TrackWrite()
	WriteComplete()
	WaitForInflightWrites(timeout time.Duration)
	SetCloser(closer func())
	Paths() []string
}

// BaseLoader provides common functionality for all catalog loaders.
// It manages leader state, inflight write tracking, file watchers, and lifecycle operations.
type BaseLoader struct {
	paths []string

	// Leader state management
	leaderMu sync.RWMutex
	isLeader bool           // true when in leader mode
	writesWG sync.WaitGroup // tracks number of database write operations in progress

	// File watcher state
	watchersMu      sync.Mutex
	watchersStarted bool
	watchersCancel  context.CancelFunc // cancels file watchers on shutdown

	// Closer for current operations
	closersMu sync.Mutex
	closer    func() // cancels the current loading goroutines
}

// NewBaseLoader creates a new BaseLoader with the given config paths
func NewBaseLoader(paths []string) *BaseLoader {
	return &BaseLoader{
		paths: paths,
	}
}

// IsLeader returns true if currently in leader mode
func (bl *BaseLoader) IsLeader() bool {
	bl.leaderMu.RLock()
	defer bl.leaderMu.RUnlock()
	return bl.isLeader
}

// SetLeader sets the leader state
func (bl *BaseLoader) SetLeader(leader bool) {
	bl.leaderMu.Lock()
	defer bl.leaderMu.Unlock()
	bl.isLeader = leader
}

// ShouldWriteDatabase returns true when in leader mode
func (bl *BaseLoader) ShouldWriteDatabase() bool {
	bl.leaderMu.RLock()
	defer bl.leaderMu.RUnlock()
	return bl.isLeader
}

// TrackWrite increments the inflight write counter.
// Should be called before starting a database write operation.
func (bl *BaseLoader) TrackWrite() {
	bl.writesWG.Add(1)
}

// WriteComplete decrements the inflight write counter.
// Should be called after completing a database write operation.
func (bl *BaseLoader) WriteComplete() {
	bl.writesWG.Done()
}

// WaitForInflightWrites waits for all inflight database writes to complete
// with a timeout.
func (bl *BaseLoader) WaitForInflightWrites(timeout time.Duration) {
	glog.Info("Waiting for inflight writes to complete...")

	done := make(chan struct{})
	go func() {
		bl.writesWG.Wait()
		close(done)
	}()

	select {
	case <-done:
		glog.Info("All inflight writes completed")
	case <-time.After(timeout):
		glog.Warningf("Timeout waiting for inflight writes to complete")
	}
}

// SetupFileWatchers initializes file watchers for the given paths.
// Returns a context that should be used for file watching operations.
// Should be called during StartReadOnly initialization.
func (bl *BaseLoader) SetupFileWatchers(ctx context.Context) (context.Context, error) {
	bl.watchersMu.Lock()
	defer bl.watchersMu.Unlock()

	if bl.watchersStarted {
		return nil, fmt.Errorf("file watchers already started")
	}

	watcherCtx, cancel := context.WithCancel(ctx)
	bl.watchersCancel = cancel
	bl.watchersStarted = true

	return watcherCtx, nil
}

// StopFileWatchers cancels the file watcher context.
// Should be called during shutdown.
func (bl *BaseLoader) StopFileWatchers() {
	bl.watchersMu.Lock()
	defer bl.watchersMu.Unlock()

	if bl.watchersCancel != nil {
		bl.watchersCancel()
		bl.watchersCancel = nil
	}
}

// SetCloser sets the closer function that cancels current loading operations.
// Should be called when starting new load operations.
func (bl *BaseLoader) SetCloser(closer func()) {
	bl.closersMu.Lock()
	defer bl.closersMu.Unlock()

	// Cancel any previous operations
	if bl.closer != nil {
		bl.closer()
	}
	bl.closer = closer
}

// Paths returns the configuration file paths
func (bl *BaseLoader) Paths() []string {
	return bl.paths
}
