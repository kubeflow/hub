package plugin

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockReloader tracks calls and allows configurable errors.
type mockReloader struct {
	mu                     sync.Mutex
	parseAllCalled         atomic.Bool
	reloadParsingCalled    atomic.Int32
	leaderOpsCalled        atomic.Int32
	lastLeaderSourceIDs    mapset.Set[string]
	parseAllErr            error
	leaderOpsErr           error
}

func (m *mockReloader) ParseAllConfigs() error {
	m.parseAllCalled.Store(true)
	return m.parseAllErr
}

func (m *mockReloader) ReloadParsing() {
	m.reloadParsingCalled.Add(1)
}

func (m *mockReloader) PerformLeaderOperations(_ context.Context, allKnownSourceIDs mapset.Set[string]) error {
	m.leaderOpsCalled.Add(1)
	m.mu.Lock()
	m.lastLeaderSourceIDs = allKnownSourceIDs
	m.mu.Unlock()
	return m.leaderOpsErr
}

// mockState provides in-memory lifecycle state.
type mockState struct {
	mu              sync.Mutex
	leader          bool
	shouldWrite     bool
	paths           []string
	watchersSetUp   atomic.Bool
	watchersStopped atomic.Bool
	waitCalled      atomic.Bool
	setupErr        error
}

func newMockState(paths ...string) *mockState {
	return &mockState{paths: paths}
}

func (s *mockState) IsLeader() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.leader
}

func (s *mockState) SetLeader(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.leader = v
	s.shouldWrite = v
}

func (s *mockState) ShouldWriteDatabase() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shouldWrite
}

func (s *mockState) SetupFileWatchers(ctx context.Context) (context.Context, error) {
	s.watchersSetUp.Store(true)
	if s.setupErr != nil {
		return nil, s.setupErr
	}
	return ctx, nil
}

func (s *mockState) StopFileWatchers() {
	s.watchersStopped.Store(true)
}

func (s *mockState) WaitForInflightWrites(_ time.Duration) {
	s.waitCalled.Store(true)
}

func (s *mockState) Paths() []string {
	return s.paths
}

// mockFileWatcher returns controllable channels.
type mockFileWatcher struct {
	mu       sync.Mutex
	channels map[string]chan struct{}
	pathErr  error
}

func newMockFileWatcher() *mockFileWatcher {
	return &mockFileWatcher{channels: make(map[string]chan struct{})}
}

func (w *mockFileWatcher) Path(_ context.Context, path string) (<-chan struct{}, error) {
	if w.pathErr != nil {
		return nil, w.pathErr
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	ch := make(chan struct{}, 1)
	w.channels[path] = ch
	return ch, nil
}

func (w *mockFileWatcher) waitForPath(path string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		w.mu.Lock()
		_, ok := w.channels[path]
		w.mu.Unlock()
		if ok {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}

func (w *mockFileWatcher) send(path string) {
	w.mu.Lock()
	ch := w.channels[path]
	w.mu.Unlock()
	if ch != nil {
		ch <- struct{}{}
		time.Sleep(20 * time.Millisecond)
	}
}

func (w *mockFileWatcher) closeAll() {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, ch := range w.channels {
		close(ch)
	}
}

func newTestPluginBase(state *mockState, loader *mockReloader, watcher *mockFileWatcher) *PluginBase {
	pb := NewPluginBase(PluginBaseConfig{
		Name:        "test",
		State:       state,
		Loader:      loader,
		FileWatcher: watcher,
		SourceIDs:   func() mapset.Set[string] { return mapset.NewSet("src-1", "src-2") },
	})
	return &pb
}

// --- Start tests ---

func TestPluginBaseStart(t *testing.T) {
	state := newMockState("a.yaml", "b.yaml")
	loader := &mockReloader{}
	watcher := newMockFileWatcher()
	pb := newTestPluginBase(state, loader, watcher)

	err := pb.Start(context.Background())
	require.NoError(t, err)
	defer watcher.closeAll()

	assert.True(t, state.watchersSetUp.Load())
	assert.True(t, loader.parseAllCalled.Load())
}

func TestPluginBaseStartSetupWatchersFails(t *testing.T) {
	state := newMockState("a.yaml")
	state.setupErr = errors.New("watcher boom")
	loader := &mockReloader{}
	watcher := newMockFileWatcher()
	pb := newTestPluginBase(state, loader, watcher)

	err := pb.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "watcher boom")
	assert.False(t, loader.parseAllCalled.Load())
}

func TestPluginBaseStartParseAllConfigsFails(t *testing.T) {
	state := newMockState("a.yaml")
	loader := &mockReloader{parseAllErr: errors.New("parse boom")}
	watcher := newMockFileWatcher()
	pb := newTestPluginBase(state, loader, watcher)

	err := pb.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse boom")
	assert.Contains(t, err.Error(), "test config")
}

// --- watchFile tests ---

func TestPluginBaseWatchFileReloadsOnChange(t *testing.T) {
	state := newMockState("a.yaml")
	loader := &mockReloader{}
	watcher := newMockFileWatcher()
	pb := newTestPluginBase(state, loader, watcher)

	require.NoError(t, pb.Start(context.Background()))
	require.True(t, watcher.waitForPath("a.yaml", time.Second), "goroutine should register watcher")

	watcher.send("a.yaml")
	assert.Equal(t, int32(1), loader.reloadParsingCalled.Load())
	assert.Equal(t, int32(0), loader.leaderOpsCalled.Load(), "not leader, should skip leader ops")

	watcher.closeAll()
}

func TestPluginBaseWatchFileLeaderWritesOnChange(t *testing.T) {
	Reset()
	defer Reset()

	state := newMockState("a.yaml")
	loader := &mockReloader{}
	watcher := newMockFileWatcher()
	pb := newTestPluginBase(state, loader, watcher)

	// Register a plugin so CollectAllSourceIDs has something to iterate.
	sourcePlugin := &mockPlugin{name: "other"}
	Register(sourcePlugin)

	state.SetLeader(true)
	require.NoError(t, pb.Start(context.Background()))
	require.True(t, watcher.waitForPath("a.yaml", time.Second), "goroutine should register watcher")

	watcher.send("a.yaml")
	assert.Equal(t, int32(1), loader.reloadParsingCalled.Load())
	assert.Equal(t, int32(1), loader.leaderOpsCalled.Load())

	watcher.closeAll()
}

// --- OnBecomeLeader tests ---

func TestPluginBaseOnBecomeLeader(t *testing.T) {
	Reset()
	defer Reset()

	state := newMockState()
	loader := &mockReloader{}
	watcher := newMockFileWatcher()
	pb := newTestPluginBase(state, loader, watcher)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() { done <- pb.OnBecomeLeader(ctx) }()

	// Give goroutine time to reach the blocking point.
	time.Sleep(50 * time.Millisecond)
	assert.True(t, state.IsLeader())
	assert.Equal(t, int32(1), loader.leaderOpsCalled.Load())

	cancel()

	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("OnBecomeLeader did not return after cancel")
	}

	assert.False(t, state.IsLeader())
	assert.True(t, state.waitCalled.Load())
}

func TestPluginBaseOnBecomeLeaderAlreadyLeader(t *testing.T) {
	state := newMockState()
	state.SetLeader(true)
	loader := &mockReloader{}
	watcher := newMockFileWatcher()
	pb := newTestPluginBase(state, loader, watcher)

	err := pb.OnBecomeLeader(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already in leader mode")
	assert.Equal(t, int32(0), loader.leaderOpsCalled.Load())
}

func TestPluginBaseOnBecomeLeaderOpsFail(t *testing.T) {
	Reset()
	defer Reset()

	state := newMockState()
	loader := &mockReloader{leaderOpsErr: errors.New("db down")}
	watcher := newMockFileWatcher()
	pb := newTestPluginBase(state, loader, watcher)

	err := pb.OnBecomeLeader(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db down")
	assert.False(t, state.IsLeader(), "leader state should revert on failure")
}

func TestPluginBaseOnBecomeLeaderHookCalled(t *testing.T) {
	Reset()
	defer Reset()

	state := newMockState()
	loader := &mockReloader{}
	watcher := newMockFileWatcher()

	hookCalled := atomic.Bool{}
	pb := NewPluginBase(PluginBaseConfig{
		Name:        "test",
		State:       state,
		Loader:      loader,
		FileWatcher: watcher,
		SourceIDs:   func() mapset.Set[string] { return mapset.NewSet[string]() },
		OnLeaderReady: func(_ context.Context) error {
			hookCalled.Store(true)
			return nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- pb.OnBecomeLeader(ctx) }()

	time.Sleep(50 * time.Millisecond)
	assert.True(t, hookCalled.Load())

	cancel()
	<-done
}

func TestPluginBaseOnBecomeLeaderHookFails(t *testing.T) {
	Reset()
	defer Reset()

	state := newMockState()
	loader := &mockReloader{}
	watcher := newMockFileWatcher()

	pb := NewPluginBase(PluginBaseConfig{
		Name:        "test",
		State:       state,
		Loader:      loader,
		FileWatcher: watcher,
		SourceIDs:   func() mapset.Set[string] { return mapset.NewSet[string]() },
		OnLeaderReady: func(_ context.Context) error {
			return errors.New("hook failed")
		},
	})

	err := pb.OnBecomeLeader(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hook failed")
	assert.False(t, state.IsLeader(), "leader state should revert on hook failure")
}

func TestPluginBaseOnBecomeLeaderNilHook(t *testing.T) {
	Reset()
	defer Reset()

	state := newMockState()
	loader := &mockReloader{}
	watcher := newMockFileWatcher()
	pb := newTestPluginBase(state, loader, watcher)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- pb.OnBecomeLeader(ctx) }()

	time.Sleep(50 * time.Millisecond)
	assert.True(t, state.IsLeader())

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("should return after cancel with nil hook")
	}
}

// --- Stop tests ---

func TestPluginBaseStop(t *testing.T) {
	state := newMockState()
	loader := &mockReloader{}
	watcher := newMockFileWatcher()
	pb := newTestPluginBase(state, loader, watcher)

	err := pb.Stop(context.Background())
	require.NoError(t, err)
	assert.True(t, state.watchersStopped.Load())
	assert.True(t, state.waitCalled.Load())
}

// --- KnownSourceIDs tests ---

func TestPluginBaseKnownSourceIDs(t *testing.T) {
	state := newMockState()
	loader := &mockReloader{}
	watcher := newMockFileWatcher()
	pb := newTestPluginBase(state, loader, watcher)

	ids := pb.KnownSourceIDs()
	assert.True(t, ids.Contains("src-1"))
	assert.True(t, ids.Contains("src-2"))
	assert.Equal(t, 2, ids.Cardinality())
}
