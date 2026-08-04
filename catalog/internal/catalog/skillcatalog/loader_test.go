package skillcatalog

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
)

func TestSkillSourceStatus(t *testing.T) {
	tests := []struct {
		name        string
		indexed     int
		warningMsgs []string
		errs        []string
		wantStatus  string
		wantMsgPart string // non-empty → assert msg contains this substring
	}{
		{"all good", 3, nil, nil, basecatalog.SourceStatusAvailable, ""},
		{"warnings only", 3, []string{"skills/a: name too long"}, nil, basecatalog.SourceStatusPartiallyAvailable, "skills/a"},
		{"some errors but some indexed", 2, nil, []string{"repo@main: boom"}, basecatalog.SourceStatusPartiallyAvailable, "boom"},
		{"all failed", 0, nil, []string{"repo@main: boom"}, basecatalog.SourceStatusError, "boom"},
		{"nothing indexed, no errors", 0, nil, nil, basecatalog.SourceStatusAvailable, ""},
		{"warnings and errors", 1, []string{"skills/b: warn"}, []string{"repo@v1: err"}, basecatalog.SourceStatusPartiallyAvailable, "err"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, msg := skillSourceStatus(tt.indexed, tt.warningMsgs, tt.errs)
			assert.Equal(t, tt.wantStatus, status)
			if tt.wantMsgPart != "" {
				assert.Contains(t, msg, tt.wantMsgPart)
			}
		})
	}
}

// --- ref job construction (fix 4: configurable max refs per repo) ---

func TestBuildRefJobs_NoRefsIsError(t *testing.T) {
	repos := []SkillRepository{{URL: "https://example.com/a.git"}}
	jobs, errs := buildRefJobs(repos, 10)
	assert.Empty(t, jobs)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "no refs configured")
}

func TestBuildRefJobs_EnforcesMaxRefsPerRepo(t *testing.T) {
	repos := []SkillRepository{
		{URL: "https://example.com/a.git", Refs: []string{"v1", "v2", "v3"}},
	}
	jobs, errs := buildRefJobs(repos, 2)
	assert.Empty(t, jobs, "a repo over the limit contributes no jobs at all")
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "lists 3 refs")
	assert.Contains(t, errs[0], "maximum of 2")
}

func TestBuildRefJobs_WithinLimitProducesOneJobPerRef(t *testing.T) {
	repos := []SkillRepository{
		{URL: "https://example.com/a.git", Refs: []string{"v1", "v2"}},
		{URL: "https://example.com/b.git", Refs: []string{"v1"}},
	}
	jobs, errs := buildRefJobs(repos, 10)
	assert.Empty(t, errs)
	require.Len(t, jobs, 3)
}

func TestBuildRefJobs_DeduplicatesRefsWithinRepo(t *testing.T) {
	// A repeated ref must index once (its composite identity is repo|path|version,
	// so duplicates would otherwise collide), and must not count against the limit.
	repos := []SkillRepository{
		{URL: "https://example.com/a.git", Refs: []string{"v1", "v1", "v2", "v1"}},
	}
	jobs, errs := buildRefJobs(repos, 2)
	assert.Empty(t, errs, "two unique refs is within the limit of 2")
	require.Len(t, jobs, 2)
	assert.Equal(t, "v1", jobs[0].ref, "first-seen order is preserved")
	assert.Equal(t, "v2", jobs[1].ref)
}

// --- bounded concurrent resolution (fix 3: configurable global in-flight-clone cap) ---

// blockingFakeResolver tracks the maximum number of concurrent Resolve calls it
// ever observes, so tests can prove a semaphore actually bounds concurrency.
type blockingFakeResolver struct {
	mu       sync.Mutex
	current  int
	maxSeen  int
	capacity int
	atCap    chan struct{} // closed when current first reaches capacity
	release  chan struct{} // closed to let all in-flight calls proceed
}

func newBlockingFakeResolver(capacity int) *blockingFakeResolver {
	return &blockingFakeResolver{
		capacity: capacity,
		atCap:    make(chan struct{}),
		release:  make(chan struct{}),
	}
}

func (f *blockingFakeResolver) Resolve(ctx context.Context, repo SkillRepository, ref string, _ *Credentials) ([]ResolvedSkill, error) {
	f.mu.Lock()
	f.current++
	if f.current > f.maxSeen {
		f.maxSeen = f.current
	}
	if f.current == f.capacity {
		select {
		case <-f.atCap: // already closed
		default:
			close(f.atCap)
		}
	}
	f.mu.Unlock()

	select {
	case <-f.release:
	case <-ctx.Done():
	}

	f.mu.Lock()
	f.current--
	f.mu.Unlock()

	return []ResolvedSkill{{Repository: repo.URL, Version: ref, Skill: &ParsedSkill{}}}, nil
}

func noCreds(SkillRepository) (*Credentials, error) { return nil, nil }

func TestResolveJobsConcurrently_BoundedBySemaphore(t *testing.T) {
	const capacity = 2
	const jobCount = 6

	var jobs []refJob
	for i := 0; i < jobCount; i++ {
		jobs = append(jobs, refJob{repo: SkillRepository{URL: fmt.Sprintf("repo-%d", i)}, ref: "v1"})
	}

	fake := newBlockingFakeResolver(capacity)
	sem := semaphore.NewWeighted(capacity)

	done := make(chan []resolveResult, 1)
	go func() {
		// Workers exceed the semaphore cap, so the clone semaphore is the binding
		// constraint here.
		done <- resolveJobsConcurrently(context.Background(), jobs, jobCount, sem, fake, noCreds)
	}()

	// Wait deterministically until exactly `capacity` goroutines are in-flight
	// before releasing, avoiding the flaky sleep-based approach.
	select {
	case <-fake.atCap:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for semaphore to reach capacity")
	}
	close(fake.release)

	select {
	case results := <-done:
		require.Len(t, results, jobCount)
		for _, r := range results {
			assert.NoError(t, r.err)
			assert.Len(t, r.skills, 1)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for resolveJobsConcurrently")
	}

	fake.mu.Lock()
	defer fake.mu.Unlock()
	assert.LessOrEqualf(t, fake.maxSeen, capacity, "never more than %d concurrent resolves", capacity)
	assert.Equal(t, capacity, fake.maxSeen, "should actually reach the cap with more jobs than capacity")
}

func TestResolveJobsConcurrently_BoundedByWorkerPool(t *testing.T) {
	const workers = 2
	const jobCount = 6

	var jobs []refJob
	for i := 0; i < jobCount; i++ {
		jobs = append(jobs, refJob{repo: SkillRepository{URL: fmt.Sprintf("repo-%d", i)}, ref: "v1"})
	}

	// The worker pool, not the clone semaphore, is the binding constraint: a large
	// semaphore lets every job clone, so concurrency is capped by the worker count.
	fake := newBlockingFakeResolver(workers)
	sem := semaphore.NewWeighted(100)

	done := make(chan []resolveResult, 1)
	go func() {
		done <- resolveJobsConcurrently(context.Background(), jobs, workers, sem, fake, noCreds)
	}()

	select {
	case <-fake.atCap:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for worker pool to reach capacity")
	}
	close(fake.release)

	select {
	case results := <-done:
		require.Len(t, results, jobCount)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for resolveJobsConcurrently")
	}

	fake.mu.Lock()
	defer fake.mu.Unlock()
	assert.Equal(t, workers, fake.maxSeen, "concurrency is capped by the worker pool size, not the job count")
}

// errResolver always fails, to verify error propagation through resolveJobsConcurrently.
type errResolver struct{ err error }

func (r errResolver) Resolve(context.Context, SkillRepository, string, *Credentials) ([]ResolvedSkill, error) {
	return nil, r.err
}

func TestResolveJobsConcurrently_PropagatesErrors(t *testing.T) {
	jobs := []refJob{{repo: SkillRepository{URL: "a"}, ref: "v1"}}
	results := resolveJobsConcurrently(context.Background(), jobs, 1, semaphore.NewWeighted(1), errResolver{err: assert.AnError}, noCreds)
	require.Len(t, results, 1)
	assert.ErrorIs(t, results[0].err, assert.AnError)
}

func TestResolveJobsConcurrently_PropagatesCredentialsError(t *testing.T) {
	jobs := []refJob{{repo: SkillRepository{URL: "a", AuthSecretName: "creds"}, ref: "v1"}}
	credsErr := fmt.Errorf("no secret resolver configured")
	results := resolveJobsConcurrently(context.Background(), jobs, 1, semaphore.NewWeighted(1),
		errResolver{}, func(SkillRepository) (*Credentials, error) { return nil, credsErr })
	require.Len(t, results, 1)
	assert.ErrorIs(t, results[0].err, credsErr)
}

// --- credentialsFor (fix 5: secret-based auth wiring) ---

type fakeSecretResolver struct {
	creds map[string]*Credentials
	err   error
}

func (f *fakeSecretResolver) GetCredentials(_ context.Context, name string) (*Credentials, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.creds[name], nil
}

func TestSkillLoader_CredentialsFor_NoSecretConfigured(t *testing.T) {
	l := &SkillLoader{}
	creds, err := l.credentialsFor(context.Background(), SkillRepository{})
	require.NoError(t, err)
	assert.Nil(t, creds)
}

func TestSkillLoader_CredentialsFor_MissingResolverErrors(t *testing.T) {
	l := &SkillLoader{}
	_, err := l.credentialsFor(context.Background(), SkillRepository{AuthSecretName: "git-creds"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no secret resolver")
}

func TestSkillLoader_CredentialsFor_DelegatesToResolver(t *testing.T) {
	want := &Credentials{Username: "x-access-token", Token: "abc"}
	l := &SkillLoader{secretResolver: &fakeSecretResolver{creds: map[string]*Credentials{"git-creds": want}}}
	got, err := l.credentialsFor(context.Background(), SkillRepository{AuthSecretName: "git-creds"})
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// --- per-source concurrency guard (fix C) ---

func TestRunSyncExclusive_NonBlockingSkipsWhenSourceBusy(t *testing.T) {
	l := &SkillLoader{}
	started := make(chan struct{})
	release := make(chan struct{})

	go l.runSyncExclusive("s", true, func() {
		close(started)
		<-release
	})
	<-started // the first (blocking) sync now holds source "s"

	ran := l.runSyncExclusive("s", false, func() {
		t.Error("non-blocking sync must not run while the source is busy")
	})
	assert.False(t, ran)

	close(release)
}

func TestRunSyncExclusive_DifferentSourcesDoNotBlockEachOther(t *testing.T) {
	l := &SkillLoader{}
	started := make(chan struct{})
	release := make(chan struct{})

	go l.runSyncExclusive("a", true, func() {
		close(started)
		<-release
	})
	<-started // source "a" is busy

	ran := l.runSyncExclusive("b", false, func() {})
	assert.True(t, ran, "a different source must not be blocked by source a")

	close(release)
}

func TestRunSyncExclusive_SerializesBlockingCallsOnSameSource(t *testing.T) {
	l := &SkillLoader{}
	var mu sync.Mutex
	var maxConcurrent, current int

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l.runSyncExclusive("s", true, func() {
				mu.Lock()
				current++
				if current > maxConcurrent {
					maxConcurrent = current
				}
				mu.Unlock()
				time.Sleep(10 * time.Millisecond)
				mu.Lock()
				current--
				mu.Unlock()
			})
		}()
	}
	wg.Wait()
	assert.Equal(t, 1, maxConcurrent, "blocking syncs of one source never overlap")
}

// --- credential caching ---

func TestNewCredentialCache_FetchesEachSecretOnce(t *testing.T) {
	var mu sync.Mutex
	calls := map[string]int{}
	base := func(repo SkillRepository) (*Credentials, error) {
		mu.Lock()
		calls[repo.AuthSecretName]++
		mu.Unlock()
		return &Credentials{Token: "t-" + repo.AuthSecretName}, nil
	}
	cached := newCredentialCache(base)

	// Two secrets, requested repeatedly (as N refs of a repo would).
	for i := 0; i < 5; i++ {
		c, err := cached(SkillRepository{URL: "a", AuthSecretName: "sec-a"})
		require.NoError(t, err)
		assert.Equal(t, "t-sec-a", c.Token)
		_, err = cached(SkillRepository{URL: "b", AuthSecretName: "sec-b"})
		require.NoError(t, err)
	}

	assert.Equal(t, 1, calls["sec-a"], "secret fetched once despite repeated lookups")
	assert.Equal(t, 1, calls["sec-b"])
}

func TestNewCredentialCache_AnonymousNotCachedAndErrorsPropagate(t *testing.T) {
	var calls int
	sentinel := fmt.Errorf("boom")
	base := func(repo SkillRepository) (*Credentials, error) {
		calls++
		if repo.AuthSecretName == "" {
			return nil, nil
		}
		return nil, sentinel
	}
	cached := newCredentialCache(base)

	// Anonymous repos always call through (nothing to cache).
	_, _ = cached(SkillRepository{})
	_, _ = cached(SkillRepository{})
	assert.Equal(t, 2, calls, "anonymous lookups are not memoized")

	// Errors are cached and returned like successes.
	_, err1 := cached(SkillRepository{AuthSecretName: "sec"})
	_, err2 := cached(SkillRepository{AuthSecretName: "sec"})
	assert.ErrorIs(t, err1, sentinel)
	assert.ErrorIs(t, err2, sentinel)
	assert.Equal(t, 3, calls, "a failed secret fetch is cached, not retried per ref")
}

func TestNewCredentialCache_ConcurrentLookupsCollapse(t *testing.T) {
	var calls int32
	base := func(repo SkillRepository) (*Credentials, error) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(30 * time.Millisecond) // widen the window for concurrent callers
		return &Credentials{Token: "t"}, nil
	}
	cached := newCredentialCache(base)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = cached(SkillRepository{AuthSecretName: "shared"})
		}()
	}
	wg.Wait()
	assert.LessOrEqual(t, int(atomic.LoadInt32(&calls)), 1, "concurrent lookups for one secret collapse to a single fetch")
}

// --- periodic sync (fix 2: syncIntervalMinutes trigger) ---

func TestRunPeriodicSync_FiresRepeatedlyUntilCancelled(t *testing.T) {
	ticks := make(chan struct{}, 20)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	runPeriodicSync(ctx, 20*time.Millisecond, &wg, func() { ticks <- struct{}{} })

	// Wait for at least 3 ticks deterministically, with a per-tick timeout.
	for i := 0; i < 3; i++ {
		select {
		case <-ticks:
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for tick %d", i+1)
		}
	}
	cancel()
	waitForWaitGroup(t, &wg, 2*time.Second, "periodic sync goroutine did not stop after cancel")
}

func waitForWaitGroup(t *testing.T, wg *sync.WaitGroup, timeout time.Duration, msg string) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatal(msg)
	}
}

func TestSchedulePeriodicSync_NoIntervalSpawnsNoGoroutine(t *testing.T) {
	l := &SkillLoader{}
	// intervalMinutes <= 0 (unset, or a source that PerformLeaderOperations parsed
	// with no syncIntervalMinutes) starts no ticker.
	l.schedulePeriodicSync(context.Background(), "s", 0)
	waitForWaitGroup(t, l.currentWG(), time.Second, "expected no periodic-sync goroutine when the interval is unset")
}

func TestSchedulePeriodicSync_ValidIntervalSpawnsAndStopsOnCancel(t *testing.T) {
	l := &SkillLoader{}

	ctx, cancel := context.WithCancel(context.Background())
	l.schedulePeriodicSync(ctx, "s", 1)

	// Cancel well before the 1-minute tick fires, so the action closure (which
	// touches l.state/l.sources) never runs; this proves the goroutine is
	// spawned and cleanly stoppable without needing to wait a full minute.
	cancel()
	waitForWaitGroup(t, l.currentWG(), 2*time.Second, "periodic sync goroutine did not stop after cancel")
}
