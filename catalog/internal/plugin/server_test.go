package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// trackingPlugin records lifecycle calls for verifying ordering.
type trackingPlugin struct {
	mockPlugin
	mu        sync.Mutex
	initOrder int32
	startedAt time.Time
	stoppedAt time.Time
	initErr   error
	startErr  error
	stopErr   error
	counter   *atomic.Int32
}

func (p *trackingPlugin) Init(_ context.Context, _ Config) error {
	if p.counter != nil {
		p.initOrder = p.counter.Add(1)
	}
	return p.initErr
}

func (p *trackingPlugin) Start(_ context.Context) error {
	p.mu.Lock()
	p.startedAt = time.Now()
	p.mu.Unlock()
	return p.startErr
}

func (p *trackingPlugin) Stop(_ context.Context) error {
	p.mu.Lock()
	p.stoppedAt = time.Now()
	p.mu.Unlock()
	return p.stopErr
}

// leaderPlugin implements both CatalogPlugin and LeaderAware.
type leaderPlugin struct {
	mockPlugin
	leaderCalled atomic.Bool
	leaderCtx    context.Context
}

func (p *leaderPlugin) OnBecomeLeader(ctx context.Context) error {
	p.leaderCalled.Store(true)
	p.leaderCtx = ctx
	<-ctx.Done()
	return ctx.Err()
}

// routePlugin registers a test route when RegisterRoutes is called.
type routePlugin struct {
	mockPlugin
}

func (p *routePlugin) RegisterRoutes(router chi.Router) error {
	router.Get("/api/test_catalog/v1alpha1/items", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items":[]}`))
	})
	return nil
}

func newTestServer() *Server {
	return NewServer(ServerConfig{})
}

func TestServerZeroPlugins(t *testing.T) {
	Reset()
	defer Reset()

	s := newTestServer()

	require.NoError(t, s.Init(context.Background()))
	assert.Empty(t, s.Plugins())

	router, err := s.MountRoutes()
	require.NoError(t, err)

	require.NoError(t, s.Start(context.Background()))
	require.NoError(t, s.Stop(context.Background()))

	// Health endpoints still work
	assertStatus(t, router, "/healthz", http.StatusOK)
	assertStatus(t, router, "/readyz", http.StatusOK)

	// Readyz with zero plugins is "ready"
	body := getJSON(t, router, "/readyz")
	assert.Equal(t, "ready", body["status"])
}

func TestServerInitLifecycle(t *testing.T) {
	Reset()
	defer Reset()

	counter := &atomic.Int32{}
	p1 := &trackingPlugin{mockPlugin: mockPlugin{name: "first", version: "v1"}, counter: counter}
	p2 := &trackingPlugin{mockPlugin: mockPlugin{name: "second", version: "v1"}, counter: counter}

	Register(p1)
	Register(p2)

	s := newTestServer()
	require.NoError(t, s.Init(context.Background()))
	assert.Len(t, s.Plugins(), 2)

	// Init called in registration order
	assert.Equal(t, int32(1), p1.initOrder)
	assert.Equal(t, int32(2), p2.initOrder)
}

func TestServerInitFailure(t *testing.T) {
	Reset()
	defer Reset()

	good := &trackingPlugin{mockPlugin: mockPlugin{name: "good", version: "v1"}}
	bad := &trackingPlugin{mockPlugin: mockPlugin{name: "bad", version: "v1"}, initErr: errors.New("init boom")}

	Register(good)
	Register(bad)

	s := newTestServer()
	err := s.Init(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad")
	assert.Contains(t, err.Error(), "init boom")

	// Only the first plugin was successfully initialized
	assert.Len(t, s.Plugins(), 1)
}

func TestServerStartFailure(t *testing.T) {
	Reset()
	defer Reset()

	Register(&trackingPlugin{mockPlugin: mockPlugin{name: "ok", version: "v1"}})
	Register(&trackingPlugin{mockPlugin: mockPlugin{name: "broken", version: "v1"}, startErr: errors.New("start boom")})

	s := newTestServer()
	require.NoError(t, s.Init(context.Background()))
	err := s.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "broken")
}

func TestServerStopReverseOrder(t *testing.T) {
	Reset()
	defer Reset()

	p1 := &trackingPlugin{mockPlugin: mockPlugin{name: "first", version: "v1"}}
	p2 := &trackingPlugin{mockPlugin: mockPlugin{name: "second", version: "v1"}}

	Register(p1)
	Register(p2)

	s := newTestServer()
	require.NoError(t, s.Init(context.Background()))
	require.NoError(t, s.Start(context.Background()))

	// Add small delay so timestamps are distinguishable
	time.Sleep(time.Millisecond)
	require.NoError(t, s.Stop(context.Background()))

	// Second plugin should stop before first
	assert.False(t, p2.stoppedAt.IsZero(), "second plugin Stop was called")
	assert.False(t, p1.stoppedAt.IsZero(), "first plugin Stop was called")
	assert.True(t, p2.stoppedAt.Before(p1.stoppedAt) || p2.stoppedAt.Equal(p1.stoppedAt),
		"second plugin should stop before or at same time as first")
}

func TestServerStopCollectsErrors(t *testing.T) {
	Reset()
	defer Reset()

	Register(&trackingPlugin{mockPlugin: mockPlugin{name: "fail1", version: "v1"}, stopErr: errors.New("err1")})
	Register(&trackingPlugin{mockPlugin: mockPlugin{name: "fail2", version: "v1"}, stopErr: errors.New("err2")})

	s := newTestServer()
	require.NoError(t, s.Init(context.Background()))
	err := s.Stop(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "err1")
	assert.Contains(t, err.Error(), "err2")
}

func TestServerHealthAggregation(t *testing.T) {
	Reset()
	defer Reset()

	healthy := &mockPlugin{name: "healthy", version: "v1", healthy: true}
	unhealthy := &mockPlugin{name: "unhealthy", version: "v1", healthy: false}

	Register(healthy)
	Register(unhealthy)

	s := newTestServer()
	require.NoError(t, s.Init(context.Background()))
	router, err := s.MountRoutes()
	require.NoError(t, err)

	// One unhealthy → 503
	assertStatus(t, router, "/readyz", http.StatusServiceUnavailable)
	body := getJSON(t, router, "/readyz")
	assert.Equal(t, "not_ready", body["status"])

	plugins := body["plugins"].(map[string]any)
	assert.Equal(t, true, plugins["healthy"])
	assert.Equal(t, false, plugins["unhealthy"])
}

func TestServerHealthAllHealthy(t *testing.T) {
	Reset()
	defer Reset()

	Register(&mockPlugin{name: "a", version: "v1", healthy: true})
	Register(&mockPlugin{name: "b", version: "v1", healthy: true})

	s := newTestServer()
	require.NoError(t, s.Init(context.Background()))
	router, err := s.MountRoutes()
	require.NoError(t, err)

	assertStatus(t, router, "/readyz", http.StatusOK)
	body := getJSON(t, router, "/readyz")
	assert.Equal(t, "ready", body["status"])
}

func TestReadyzBothPluginsUnhealthy(t *testing.T) {
	Reset()
	defer Reset()

	Register(&mockPlugin{name: "model", version: "v1", healthy: false})
	Register(&mockPlugin{name: "mcp", version: "v1", healthy: false})

	s := newTestServer()
	require.NoError(t, s.Init(context.Background()))
	router, err := s.MountRoutes()
	require.NoError(t, err)

	assertStatus(t, router, "/readyz", http.StatusServiceUnavailable)
	body := getJSON(t, router, "/readyz")
	assert.Equal(t, "not_ready", body["status"])

	plugins := body["plugins"].(map[string]any)
	assert.Equal(t, false, plugins["model"])
	assert.Equal(t, false, plugins["mcp"])
}

func TestReadyzSinglePluginUnhealthy(t *testing.T) {
	Reset()
	defer Reset()

	Register(&mockPlugin{name: "model", version: "v1", healthy: true})
	Register(&mockPlugin{name: "mcp", version: "v1", healthy: false})

	s := newTestServer()
	require.NoError(t, s.Init(context.Background()))
	router, err := s.MountRoutes()
	require.NoError(t, err)

	assertStatus(t, router, "/readyz", http.StatusServiceUnavailable)
	body := getJSON(t, router, "/readyz")
	assert.Equal(t, "not_ready", body["status"])

	plugins := body["plugins"].(map[string]any)
	assert.Equal(t, true, plugins["model"])
	assert.Equal(t, false, plugins["mcp"])
}

func TestServerNotifyLeader(t *testing.T) {
	Reset()
	defer Reset()

	leader := &leaderPlugin{mockPlugin: mockPlugin{name: "leader-aware", version: "v1"}}
	plain := &mockPlugin{name: "plain", version: "v1"}

	Register(leader)
	Register(plain)

	s := newTestServer()
	require.NoError(t, s.Init(context.Background()))

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		s.NotifyLeader(ctx)
		close(done)
	}()

	// Give goroutine time to start
	time.Sleep(50 * time.Millisecond)
	assert.True(t, leader.leaderCalled.Load(), "LeaderAware plugin should be called")

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("NotifyLeader did not return after context cancellation")
	}
}

func TestServerNotifyLeaderNoLeaderPlugins(t *testing.T) {
	Reset()
	defer Reset()

	Register(&mockPlugin{name: "plain", version: "v1"})

	s := newTestServer()
	require.NoError(t, s.Init(context.Background()))

	done := make(chan struct{})
	go func() {
		s.NotifyLeader(context.Background())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("NotifyLeader should return immediately with no LeaderAware plugins")
	}
}

func TestServerMountRoutes(t *testing.T) {
	Reset()
	defer Reset()

	Register(&routePlugin{mockPlugin: mockPlugin{name: "test", version: "v1alpha1", healthy: true}})

	s := newTestServer()
	require.NoError(t, s.Init(context.Background()))
	router, err := s.MountRoutes()
	require.NoError(t, err)

	// Plugin route works
	assertStatus(t, router, "/api/test_catalog/v1alpha1/items", http.StatusOK)

	// Server endpoints coexist
	assertStatus(t, router, "/healthz", http.StatusOK)
	assertStatus(t, router, "/readyz", http.StatusOK)
}

// failingRoutePlugin returns an error from RegisterRoutes.
type failingRoutePlugin struct {
	mockPlugin
}

func (p *failingRoutePlugin) RegisterRoutes(_ chi.Router) error {
	return errors.New("route registration failed")
}

func TestServerMountRoutesFailure(t *testing.T) {
	Reset()
	defer Reset()

	Register(&failingRoutePlugin{mockPlugin: mockPlugin{name: "broken", version: "v1"}})

	s := newTestServer()
	require.NoError(t, s.Init(context.Background()))
	_, err := s.MountRoutes()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "broken")
	assert.Contains(t, err.Error(), "route registration failed")
}

func TestReadyzNoReadinessChecks(t *testing.T) {
	Reset()
	defer Reset()

	Register(&mockPlugin{name: "model", version: "v1", healthy: true})

	s := newTestServer()
	require.NoError(t, s.Init(context.Background()))

	router, err := s.MountRoutes()
	require.NoError(t, err)

	assertStatus(t, router, "/readyz", http.StatusOK)
	body := getJSON(t, router, "/readyz")
	assert.Equal(t, "ready", body["status"])
}

func TestReadyzWithReadinessCheck(t *testing.T) {
	Reset()
	defer Reset()

	Register(&mockPlugin{name: "model", version: "v1", healthy: true})

	s := newTestServer()
	require.NoError(t, s.Init(context.Background()))

	healthy := true
	s.AddReadinessCheck("leader_election", func() bool { return healthy })

	router, err := s.MountRoutes()
	require.NoError(t, err)

	// All healthy → 200
	assertStatus(t, router, "/readyz", http.StatusOK)
	body := getJSON(t, router, "/readyz")
	assert.Equal(t, "ready", body["status"])

	// Health check fails → 503
	healthy = false
	assertStatus(t, router, "/readyz", http.StatusServiceUnavailable)
	body = getJSON(t, router, "/readyz")
	assert.Equal(t, "not_ready", body["status"])

	// Plugin still healthy
	plugins := body["plugins"].(map[string]any)
	assert.Equal(t, true, plugins["model"])
}

func TestReadyzReadinessCheckRecovery(t *testing.T) {
	Reset()
	defer Reset()

	Register(&mockPlugin{name: "model", version: "v1", healthy: true})

	s := newTestServer()
	require.NoError(t, s.Init(context.Background()))

	healthy := false
	s.AddReadinessCheck("leader_election", func() bool { return healthy })

	router, err := s.MountRoutes()
	require.NoError(t, err)

	// Starts unhealthy
	assertStatus(t, router, "/readyz", http.StatusServiceUnavailable)

	// Recovers
	healthy = true
	assertStatus(t, router, "/readyz", http.StatusOK)
	body := getJSON(t, router, "/readyz")
	assert.Equal(t, "ready", body["status"])
}

// fakeRepoSet is a minimal RepoSet implementation for tests.
type fakeRepoSet struct {
	typeMap map[string]int32
}

func (f *fakeRepoSet) TypeMap() map[string]int32              { return f.typeMap }
func (f *fakeRepoSet) Repository(_ reflect.Type) (any, error) { return nil, nil }

// reconnectablePlugin implements Reconnectable to record reconnect calls.
type reconnectablePlugin struct {
	mockPlugin
	reconnectCalled bool
	reconnectCfg    Config
	reconnectErr    error
}

func (p *reconnectablePlugin) Reconnect(_ context.Context, cfg Config) error {
	p.reconnectCalled = true
	p.reconnectCfg = cfg
	return p.reconnectErr
}

func TestServerReconnect_UpdatesRepoSetAndCallsPlugins(t *testing.T) {
	Reset()
	defer Reset()

	oldRepoSet := &fakeRepoSet{typeMap: map[string]int32{"TypeA": 1}}
	newRepoSet := &fakeRepoSet{typeMap: map[string]int32{"TypeA": 99}}

	plain := &mockPlugin{name: "plain", version: "v1"}
	rc := &reconnectablePlugin{mockPlugin: mockPlugin{name: "rc", version: "v1"}}
	Register(plain)
	Register(rc)

	s := NewServer(ServerConfig{RepoSet: oldRepoSet})
	require.NoError(t, s.Init(context.Background()))

	require.NoError(t, s.Reconnect(context.Background(), newRepoSet))

	assert.Equal(t, newRepoSet, s.cfg.RepoSet, "server RepoSet must be updated")
	assert.True(t, rc.reconnectCalled, "Reconnectable plugin must be called")
	assert.Equal(t, newRepoSet, rc.reconnectCfg.RepoSet, "plugin must receive new RepoSet")
}

func TestServerReconnect_PropagatesError(t *testing.T) {
	Reset()
	defer Reset()

	rc := &reconnectablePlugin{
		mockPlugin:   mockPlugin{name: "rc", version: "v1"},
		reconnectErr: errors.New("reconnect boom"),
	}
	Register(rc)

	s := NewServer(ServerConfig{})
	require.NoError(t, s.Init(context.Background()))

	err := s.Reconnect(context.Background(), &fakeRepoSet{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reconnect boom")
}

// Test helpers

func assertStatus(t *testing.T, handler http.Handler, path string, expectedStatus int) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, expectedStatus, w.Code, "unexpected status for %s", path)
}

func getJSON(t *testing.T, handler http.Handler, path string) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	var result map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	return result
}
