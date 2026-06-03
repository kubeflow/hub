package leader_test

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/kubeflow/hub/catalog/internal/testhelpers"
	"github.com/kubeflow/hub/catalog/internal/leader"
	"github.com/kubeflow/hub/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	os.Exit(testutils.TestMainPostgresHelper(m))
}

func TestLeaderElector_Run(t *testing.T) {
	db, cleanup := testutils.SetupPostgresWithMigrations(t, testhelpers.MustDatastoreSpec(t))
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var becameLeader atomic.Bool
	var lostLeadership atomic.Bool

	onBecomeLeader := func(ctx context.Context) {
		becameLeader.Store(true)
		<-ctx.Done() // Block until leadership is lost
		lostLeadership.Store(true)
	}

	elector, err := leader.NewLeaderElector(
		db,
		ctx,
		"test-lock",
		10*time.Second,
		1*time.Second,
	)
	require.NoError(t, err)
	elector.OnBecomeLeader(onBecomeLeader)

	// Wait for leadership acquisition (background goroutine auto-started)
	require.Eventually(t, func() bool {
		return becameLeader.Load()
	}, 3*time.Second, 100*time.Millisecond, "Should acquire leadership")

	// Cancel context to trigger leadership loss
	cancel()

	// Wait for graceful shutdown using Wait()
	err = elector.Wait()
	assert.ErrorIs(t, err, context.Canceled)

	assert.True(t, lostLeadership.Load(), "Should have lost leadership gracefully")
}

// TestMultiPodLeaderElection verifies that only one pod can hold the leader lock at a time.
// This simulates a multi-pod deployment scenario where:
// - Multiple pods attempt to acquire leadership
// - Only one pod becomes leader at a time
// - When the leader releases the lock, another pod can acquire it
func TestMultiPodLeaderElection(t *testing.T) {
	db, cleanup := testutils.SetupPostgresWithMigrations(t, testhelpers.MustDatastoreSpec(t))
	defer cleanup()

	// Use a longer timeout to account for container startup and leader transitions
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Track which "pod" is currently the leader
	var currentLeader atomic.Int32
	var pod1BecameLeader, pod2BecameLeader atomic.Bool

	// Pod 1 callback
	onBecomeLeaderPod1 := func(ctx context.Context) {
		currentLeader.Store(1)
		pod1BecameLeader.Store(true)
		t.Log("Pod 1 became leader")
		<-ctx.Done() // Block until leadership is lost
		t.Log("Pod 1 lost leadership")
	}

	// Pod 2 callback
	onBecomeLeaderPod2 := func(ctx context.Context) {
		currentLeader.Store(2)
		pod2BecameLeader.Store(true)
		t.Log("Pod 2 became leader")
		<-ctx.Done() // Block until leadership is lost
		t.Log("Pod 2 lost leadership")
	}

	// Create contexts for each pod
	ctx1, cancel1 := context.WithCancel(ctx)
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(ctx)
	defer cancel2()

	// Create two leader electors (simulating two pods)
	// Background goroutines start automatically
	elector1, err := leader.NewLeaderElector(
		db,
		ctx1,
		"catalog-leader",
		5*time.Second, // Lock duration
		1*time.Second, // Heartbeat frequency
	)
	require.NoError(t, err)
	elector1.OnBecomeLeader(onBecomeLeaderPod1)

	// Wait for pod 1 to become leader
	require.Eventually(t, func() bool {
		return pod1BecameLeader.Load()
	}, 3*time.Second, 100*time.Millisecond, "Pod 1 should acquire leadership")

	assert.Equal(t, int32(1), currentLeader.Load(), "Pod 1 should be the leader")

	// Start pod 2 attempting to acquire leadership (should block since pod 1 holds the lock)
	elector2, err := leader.NewLeaderElector(
		db,
		ctx2,
		"catalog-leader", // Same lock name
		5*time.Second,
		1*time.Second,
	)
	require.NoError(t, err)
	elector2.OnBecomeLeader(onBecomeLeaderPod2)

	// Give pod 2 some time to attempt acquisition - it should not become leader
	time.Sleep(2 * time.Second)
	assert.False(t, pod2BecameLeader.Load(), "Pod 2 should not acquire leadership while pod 1 holds the lock")
	assert.Equal(t, int32(1), currentLeader.Load(), "Pod 1 should still be the leader")

	// Release pod 1's leadership
	t.Log("Releasing pod 1's leadership")
	cancel1()

	// Wait for pod 1's Wait to complete
	err = elector1.Wait()
	assert.ErrorIs(t, err, context.Canceled, "Pod 1 should exit with context.Canceled")

	// Now pod 2 should be able to acquire leadership
	t.Log("Waiting for pod 2 to acquire leadership...")
	require.Eventually(t, func() bool {
		acquired := pod2BecameLeader.Load()
		if !acquired {
			t.Log("Pod 2 has not yet acquired leadership, waiting...")
		}
		return acquired
	}, 10*time.Second, 200*time.Millisecond, "Pod 2 should acquire leadership after pod 1 releases")

	assert.Equal(t, int32(2), currentLeader.Load(), "Pod 2 should be the leader")

	// Clean up pod 2
	cancel2()
	err = elector2.Wait()
	assert.ErrorIs(t, err, context.Canceled, "Pod 2 should exit with context.Canceled")
}

// TestLeaderElector_MultipleCallbacks verifies that multiple registered callbacks
// are all invoked concurrently when leadership is acquired.
func TestLeaderElector_MultipleCallbacks(t *testing.T) {
	db, cleanup := testutils.SetupPostgresWithMigrations(t, testhelpers.MustDatastoreSpec(t))
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var callback1Started, callback2Started, callback3Started atomic.Bool
	var callback1Done, callback2Done, callback3Done atomic.Bool

	callback1 := func(ctx context.Context) {
		callback1Started.Store(true)
		<-ctx.Done()
		callback1Done.Store(true)
	}

	callback2 := func(ctx context.Context) {
		callback2Started.Store(true)
		<-ctx.Done()
		callback2Done.Store(true)
	}

	callback3 := func(ctx context.Context) {
		callback3Started.Store(true)
		<-ctx.Done()
		callback3Done.Store(true)
	}

	elector, err := leader.NewLeaderElector(
		db,
		ctx,
		"test-lock-multi",
		10*time.Second,
		1*time.Second,
	)
	require.NoError(t, err)

	// Register multiple callbacks
	elector.OnBecomeLeader(callback1)
	elector.OnBecomeLeader(callback2)
	elector.OnBecomeLeader(callback3)

	// Wait for all callbacks to start (background goroutine auto-started)
	require.Eventually(t, func() bool {
		return callback1Started.Load() && callback2Started.Load() && callback3Started.Load()
	}, 3*time.Second, 100*time.Millisecond, "All callbacks should start")

	// Cancel context to trigger leadership loss
	cancel()

	// Wait for graceful shutdown using Wait()
	err = elector.Wait()
	assert.ErrorIs(t, err, context.Canceled)

	// Verify all callbacks completed
	assert.True(t, callback1Done.Load(), "Callback 1 should complete")
	assert.True(t, callback2Done.Load(), "Callback 2 should complete")
	assert.True(t, callback3Done.Load(), "Callback 3 should complete")
}

// TestLeaderElector_CallbackPanic verifies that a panic in one callback
// doesn't affect other callbacks or crash the leader elector.
func TestLeaderElector_CallbackPanic(t *testing.T) {
	db, cleanup := testutils.SetupPostgresWithMigrations(t, testhelpers.MustDatastoreSpec(t))
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var goodCallbackStarted, goodCallbackDone atomic.Bool

	panicCallback := func(ctx context.Context) {
		panic("intentional panic for testing")
	}

	goodCallback := func(ctx context.Context) {
		goodCallbackStarted.Store(true)
		<-ctx.Done()
		goodCallbackDone.Store(true)
	}

	elector, err := leader.NewLeaderElector(
		db,
		ctx,
		"test-lock-panic",
		10*time.Second,
		1*time.Second,
	)
	require.NoError(t, err)

	elector.OnBecomeLeader(panicCallback)
	elector.OnBecomeLeader(goodCallback)

	// Wait for good callback to start (despite panic in other callback)
	require.Eventually(t, func() bool {
		return goodCallbackStarted.Load()
	}, 3*time.Second, 100*time.Millisecond, "Good callback should start despite panic")

	// Cancel context
	cancel()

	// Wait for graceful shutdown using Wait()
	err = elector.Wait()
	assert.ErrorIs(t, err, context.Canceled)

	assert.True(t, goodCallbackDone.Load(), "Good callback should complete")
}

// TestLeaderElector_CallbackEarlyExit verifies that if callbacks exit early,
// leadership continues (keeps the lock) until context is cancelled.
// This is the new behavior per the plan - no longer releases lock when callbacks exit.
func TestLeaderElector_CallbackEarlyExit(t *testing.T) {
	db, cleanup := testutils.SetupPostgresWithMigrations(t, testhelpers.MustDatastoreSpec(t))
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var earlyExitStarted atomic.Bool

	earlyExitCallback := func(ctx context.Context) {
		earlyExitStarted.Store(true)
		// Exit immediately without waiting for context
	}

	elector, err := leader.NewLeaderElector(
		db,
		ctx,
		"test-lock-early",
		10*time.Second,
		1*time.Second,
	)
	require.NoError(t, err)

	elector.OnBecomeLeader(earlyExitCallback)

	// Wait for callback to start
	require.Eventually(t, func() bool {
		return earlyExitStarted.Load()
	}, 3*time.Second, 100*time.Millisecond, "Callback should start")

	// Give some time to ensure leadership continues despite early exit
	// In the old behavior, this would cause the lock to be released
	// In the new behavior, the lock is kept until context cancels
	time.Sleep(1 * time.Second)

	// Cancel context to trigger shutdown
	cancel()

	// Wait for graceful shutdown using Wait()
	err = elector.Wait()
	assert.ErrorIs(t, err, context.Canceled)
}

// TestLeaderElector_CallbackRegisteredAfterLeadership verifies that callbacks
// registered after leadership is already acquired are invoked immediately.
func TestLeaderElector_CallbackRegisteredAfterLeadership(t *testing.T) {
	db, cleanup := testutils.SetupPostgresWithMigrations(t, testhelpers.MustDatastoreSpec(t))
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var callback1Started, callback2Started atomic.Bool
	var callback1Done, callback2Done atomic.Bool

	callback1 := func(ctx context.Context) {
		callback1Started.Store(true)
		<-ctx.Done()
		callback1Done.Store(true)
	}

	callback2 := func(ctx context.Context) {
		callback2Started.Store(true)
		<-ctx.Done()
		callback2Done.Store(true)
	}

	elector, err := leader.NewLeaderElector(
		db,
		ctx,
		"test-lock-late-register",
		10*time.Second,
		1*time.Second,
	)
	require.NoError(t, err)

	// Register first callback
	elector.OnBecomeLeader(callback1)

	// Wait for leadership acquisition
	require.Eventually(t, func() bool {
		return callback1Started.Load()
	}, 3*time.Second, 100*time.Millisecond, "Callback 1 should start")

	// Now register a second callback while already leader
	elector.OnBecomeLeader(callback2)

	// The second callback should start immediately
	require.Eventually(t, func() bool {
		return callback2Started.Load()
	}, 1*time.Second, 50*time.Millisecond, "Callback 2 should start immediately after registration")

	// Cancel context
	cancel()

	// Wait for graceful shutdown
	err = elector.Wait()
	assert.ErrorIs(t, err, context.Canceled)

	// Both callbacks should have completed
	assert.True(t, callback1Done.Load(), "Callback 1 should complete")
	assert.True(t, callback2Done.Load(), "Callback 2 should complete")
}

// TestLeaderElector_WaitReturnsCorrectError verifies that Wait() returns
// the correct error from the background goroutine.
func TestLeaderElector_WaitReturnsCorrectError(t *testing.T) {
	db, cleanup := testutils.SetupPostgresWithMigrations(t, testhelpers.MustDatastoreSpec(t))
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	elector, err := leader.NewLeaderElector(
		db,
		ctx,
		"test-lock-wait-error",
		10*time.Second,
		1*time.Second,
	)
	require.NoError(t, err)

	// Cancel immediately
	cancel()

	// Wait should return context.Canceled
	err = elector.Wait()
	assert.ErrorIs(t, err, context.Canceled)
}

// TestLeaderElector_GoroutineStartsImmediately verifies that the background
// goroutine starts immediately from the constructor.
func TestLeaderElector_GoroutineStartsImmediately(t *testing.T) {
	db, cleanup := testutils.SetupPostgresWithMigrations(t, testhelpers.MustDatastoreSpec(t))
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var becameLeader atomic.Bool

	elector, err := leader.NewLeaderElector(
		db,
		ctx,
		"test-lock-immediate",
		10*time.Second,
		1*time.Second,
	)
	require.NoError(t, err)

	// Register callback after construction
	elector.OnBecomeLeader(func(ctx context.Context) {
		becameLeader.Store(true)
		<-ctx.Done()
	})

	// Should acquire leadership without explicitly calling Run()
	require.Eventually(t, func() bool {
		return becameLeader.Load()
	}, 3*time.Second, 100*time.Millisecond, "Should acquire leadership automatically")

	cancel()
	err = elector.Wait()
	assert.ErrorIs(t, err, context.Canceled)
}
