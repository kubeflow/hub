// Package leader provides distributed leader election with PostgreSQL-based locking.
package leader

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"cirello.io/pglock"
	"github.com/golang/glog"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

const defaultUnhealthyThreshold int32 = 3

// isFatalError reports whether err indicates a lost pglock schema (SQLSTATE 42P01,
// "undefined table/sequence"). When resetFunc is available, the run loop attempts
// in-process recovery; otherwise the process exits so Kubernetes can restart it
// and recreate the schema via TryCreateTable.
func isFatalError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42P01"
}

// lockHandle represents a held distributed lock.
type lockHandle interface {
	sendHeartbeat(ctx context.Context) error
	release() error
}

// lockClient abstracts distributed lock acquisition for testability.
type lockClient interface {
	acquireContext(ctx context.Context, name string) (lockHandle, error)
}

// pglockAdapter wraps *pglock.Client to satisfy lockClient.
type pglockAdapter struct {
	client *pglock.Client
}

func (a *pglockAdapter) acquireContext(ctx context.Context, name string) (lockHandle, error) {
	lock, err := a.client.AcquireContext(ctx, name, pglock.FailIfLocked())
	if err != nil {
		return nil, err
	}
	return &pglockHandle{client: a.client, lock: lock}, nil
}

// pglockHandle wraps a held *pglock.Lock to satisfy lockHandle.
type pglockHandle struct {
	client *pglock.Client
	lock   *pglock.Lock
}

func (h *pglockHandle) sendHeartbeat(ctx context.Context) error {
	return h.client.SendHeartbeat(ctx, h.lock)
}

func (h *pglockHandle) release() error {
	return h.client.Release(h.lock)
}

// backoff provides exponential backoff for retry logic.
type backoff struct {
	current time.Duration
	max     time.Duration
}

// newBackoff creates a backoff with 1s initial delay and 30s maximum.
func newBackoff() *backoff {
	return &backoff{
		current: 1 * time.Second,
		max:     30 * time.Second,
	}
}

// next returns the current delay, then doubles it (up to max).
func (b *backoff) next() time.Duration {
	delay := b.current
	b.current = min(b.current*2, b.max)
	return delay
}

func (b *backoff) reset() {
	b.current = 1 * time.Second
}

// LeaderElector manages leader election for distributed services
// using pglock to coordinate leadership across multiple instances.
type LeaderElector struct {
	ctx            context.Context
	lockName       string
	lockDuration   time.Duration
	heartbeatFreq  time.Duration
	onBecomeLeader []func(context.Context)
	locker         lockClient
	mu             sync.Mutex

	// Leadership state (protected by mu)
	isLeader     bool
	leaderCtx    context.Context
	cancelLeader func()

	// Background goroutine tracking
	done chan struct{}
	err  error

	// Dynamic callback tracking
	activeCallbacks sync.WaitGroup

	// Health tracking.
	// dbReachable flips to true on first successful DB contact (acquisition
	// or ErrNotAcquired). Pods that have never contacted the DB are not ready.
	// consecutiveFailures counts consecutive infrastructure errors (DB
	// unreachable, lost heartbeat). Healthy() returns false when the DB has
	// never been reached OR failures exceed the threshold.
	dbReachable         atomic.Bool
	consecutiveFailures atomic.Int32
	unhealthyThreshold  int32

	// retryBackoff overrides the default backoff for testing. nil = use defaults.
	retryBackoff *backoff

	// resetFunc recreates the lock client after a 42P01 schema-loss error.
	// When non-nil, the run loop calls it instead of exiting, allowing
	// in-process recovery without a pod restart. When nil, the fatal exit
	// path is used (backwards-compatible default for tests that construct
	// LeaderElector directly).
	resetFunc func() (lockClient, error)
}

// NewLeaderElector creates a new LeaderElector instance.
//
// Parameters:
//   - gormDB: PostgreSQL database connection
//   - ctx: Base context for leadership contexts
//   - lockName: Unique identifier for the distributed lock
//   - lockDuration: How long the lock is held before expiring
//   - heartbeatFreq: How often to renew the lock while leader
//
// Returns a configured LeaderElector, or an error if client creation fails.
// Use OnBecomeLeader to register callbacks that will be invoked when leadership is acquired.
func NewLeaderElector(
	gormDB *gorm.DB,
	ctx context.Context,
	lockName string,
	lockDuration time.Duration,
	heartbeatFreq time.Duration,
) (*LeaderElector, error) {
	if gormDB.Name() != "postgres" {
		return nil, errors.New("not a postgres database handle")
	}

	db, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("unable to get sql.DB from GORM: %w", err)
	}

	client, err := pglock.UnsafeNew(
		db,
		pglock.WithLeaseDuration(lockDuration),
		pglock.WithHeartbeatFrequency(heartbeatFreq),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create pglock client: %w", err)
	}

	err = client.TryCreateTable()
	if err != nil {
		return nil, err
	}

	e := &LeaderElector{
		ctx:                ctx,
		lockName:           lockName,
		lockDuration:       lockDuration,
		heartbeatFreq:      heartbeatFreq,
		locker:             &pglockAdapter{client: client},
		done:               make(chan struct{}),
		unhealthyThreshold: defaultUnhealthyThreshold,
		resetFunc: func() (lockClient, error) {
			c, err := pglock.UnsafeNew(
				db,
				pglock.WithLeaseDuration(lockDuration),
				pglock.WithHeartbeatFrequency(heartbeatFreq),
			)
			if err != nil {
				return nil, fmt.Errorf("failed to recreate pglock client: %w", err)
			}
			if err := c.TryCreateTable(); err != nil {
				return nil, fmt.Errorf("failed to recreate pglock schema: %w", err)
			}
			return &pglockAdapter{client: c}, nil
		},
	}
	// dbReachable starts false (zero value) — pod is not ready until first
	// successful DB contact. No need to hack the failure counter.

	// Start the background goroutine immediately
	go e.run()

	return e, nil
}

// startCallback launches a callback in a goroutine with panic recovery and proper cleanup.
func (e *LeaderElector) startCallback(leaderCtx context.Context, idx int, cb func(context.Context)) {
	e.activeCallbacks.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				glog.Errorf("Leader callback %d panicked: %v", idx, r)
			}
		}()
		cb(leaderCtx)
		// Only warn if context wasn't cancelled
		if leaderCtx.Err() == nil {
			glog.Warningf("Leader callback %d exited early", idx)
		}
	})
}

// OnBecomeLeader registers a callback to be invoked when this instance becomes leader.
// The callback receives a context that is canceled when leadership is lost.
// The callback should block until the context is canceled for graceful shutdown.
// Multiple callbacks can be registered and will be executed concurrently.
//
// If this instance is already the leader when this method is called, the callback
// will be invoked immediately with the current leadership context.
func (e *LeaderElector) OnBecomeLeader(callback func(context.Context)) {
	e.mu.Lock()
	e.onBecomeLeader = append(e.onBecomeLeader, callback)

	// If already leader, start callback immediately
	if e.isLeader {
		leaderCtx := e.leaderCtx
		idx := len(e.onBecomeLeader) - 1
		e.mu.Unlock()
		e.startCallback(leaderCtx, idx, callback)
		return
	}
	e.mu.Unlock()
}

// Wait blocks until the background goroutine exits and returns any error.
// This replaces the old Run() method in the new API.
func (e *LeaderElector) Wait() error {
	<-e.done
	return e.err
}

// Healthy reports whether this elector can reach the database.
// Returns false before first DB contact (cold start) and when consecutive
// infrastructure failures exceed the threshold.
func (e *LeaderElector) Healthy() bool {
	return e.dbReachable.Load() && e.consecutiveFailures.Load() < e.unhealthyThreshold
}

// run starts the leader election process with automatic retry.
// This runs in a background goroutine started by NewLeaderElector.
//
// Continuously attempts to acquire leadership until the context cancels.
// Handles retry logic internally with exponential backoff.
//
// Behavior:
//  1. Acquires the distributed lock
//  2. On success, invokes all registered callbacks concurrently with a leadership context
//  3. Renews the lock at heartbeatFreq intervals
//  4. On loss or error, retries with exponential backoff
//  5. Returns only when context cancels (graceful shutdown)
func (e *LeaderElector) run() {
	defer close(e.done)

	ctx := e.ctx
	backoff := e.retryBackoff
	if backoff == nil {
		backoff = newBackoff()
	}

	for {
		if ctx.Err() != nil {
			e.err = ctx.Err()
			return
		}

		glog.Info("Attempting to acquire leadership...")
		err := e.runOnce(ctx)

		if errors.Is(err, context.Canceled) {
			glog.Info("Leader election canceled, shutting down")
			e.err = err
			return
		}

		if err != nil {
			// pglock returns ErrNotAcquired (not context.Canceled) when the
			// context is cancelled during acquisition — check context first.
			if ctx.Err() != nil {
				glog.Info("Leader election canceled, shutting down")
				e.err = ctx.Err()
				return
			}

			var delay time.Duration
			if errors.Is(err, pglock.ErrNotAcquired) {
				// Lock held by another pod — DB is reachable, pod is healthy.
				// Keep retry interval short so we acquire quickly when the
				// lease is released (e.g., during rolling updates).
				e.dbReachable.Store(true)
				e.consecutiveFailures.Store(0)
				backoff.reset()
				delay = backoff.next()
				glog.Infof("Lock held by another instance, retrying in %v", delay)
			} else if isFatalError(err) {
				if e.resetFunc == nil {
					// No recovery path — exit so the pod restarts and recreates the schema.
					glog.Errorf("Fatal leader election error (schema lost): %v — exiting for pod restart", err)
					e.err = fmt.Errorf("fatal leader election error: %w", err)
					return
				}
				glog.Warningf("Schema lost (pglock table missing): %v — attempting in-process recovery", err)
				e.consecutiveFailures.Add(1)
				newLocker, resetErr := e.resetFunc()
				if resetErr != nil {
					glog.Errorf("Unable to recreate pglock schema: %v — exiting for pod restart", resetErr)
					e.err = fmt.Errorf("fatal leader election error: %w (recovery failed: %v)", err, resetErr)
					return
				}
				e.locker = newLocker
				e.consecutiveFailures.Store(0)
				glog.Info("pglock schema recreated successfully, resuming leader election")
				backoff.reset()
				delay = backoff.next()
			} else {
				failures := e.consecutiveFailures.Add(1)
				delay = backoff.next()
				glog.Errorf("Leader election error: %v (consecutive failures: %d, retrying in %v)", err, failures, delay)
			}

			select {
			case <-time.After(delay):
				continue
			case <-ctx.Done():
				e.err = ctx.Err()
				return
			}
		}

		glog.Info("Leadership ended gracefully, attempting to reacquire")
		backoff.reset()
	}
}

// runOnce attempts to acquire leadership once and run the leader callbacks.
// Returns when context is canceled or lock is lost.
// Per the plan, callbacks exiting early no longer causes lock release.
func (e *LeaderElector) runOnce(ctx context.Context) error {
	handle, err := e.locker.acquireContext(ctx, e.lockName)
	if err != nil {
		return fmt.Errorf("failed to acquire lock %q: %w", e.lockName, err)
	}

	glog.Infof("Successfully acquired leadership lock: %s", e.lockName)
	e.dbReachable.Store(true)
	e.consecutiveFailures.Store(0)

	// Create a context that will be canceled when we lose leadership
	leaderCtx, cancelLeader := context.WithCancel(e.ctx)
	defer cancelLeader()

	// Set leadership state and store leaderCtx before starting callbacks
	e.mu.Lock()
	e.isLeader = true
	e.leaderCtx = leaderCtx
	e.cancelLeader = cancelLeader

	// Get snapshot of callbacks
	callbacks := make([]func(context.Context), len(e.onBecomeLeader))
	copy(callbacks, e.onBecomeLeader)
	e.mu.Unlock()

	// Start all callbacks concurrently using startCallback
	for i, callback := range callbacks {
		e.startCallback(leaderCtx, i, callback)
	}

	ticker := time.NewTicker(e.heartbeatFreq)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Parent context canceled - graceful shutdown
			glog.Infof("Context canceled, releasing leadership lock: %s", e.lockName)
			cancelLeader()           // Signal ALL callbacks to stop
			e.activeCallbacks.Wait() // Wait for ALL callbacks to finish

			// Clear leadership state
			e.mu.Lock()
			e.isLeader = false
			e.leaderCtx = nil
			e.cancelLeader = nil
			e.mu.Unlock()

			if err := handle.release(); err != nil {
				glog.Errorf("Error releasing lock: %v", err)
			}
			return ctx.Err()

		case <-ticker.C:
			// Verify we still own the lock
			if err := handle.sendHeartbeat(ctx); err != nil {
				glog.Errorf("Lost leadership lock: %v", err)
				cancelLeader()           // Signal ALL callbacks to stop
				e.activeCallbacks.Wait() // Wait for ALL callbacks to finish

				// Clear leadership state
				e.mu.Lock()
				e.isLeader = false
				e.leaderCtx = nil
				e.cancelLeader = nil
				e.mu.Unlock()

				return fmt.Errorf("lost leadership: %w", err)
			}
		}
	}
}
