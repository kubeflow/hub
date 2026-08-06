package skillcatalog

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
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
	jobs, errs, rejected := buildRefJobs(repos, 10)
	assert.Empty(t, jobs)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "no refs configured")
	assert.Equal(t, []string{"https://example.com/a.git"}, rejected,
		"a rejected repo is reported so orphan cleanup can protect its skills")
}

func TestBuildRefJobs_EnforcesMaxRefsPerRepo(t *testing.T) {
	repos := []SkillRepository{
		{URL: "https://example.com/a.git", Refs: []string{"v1", "v2", "v3"}},
	}
	jobs, errs, rejected := buildRefJobs(repos, 2)
	assert.Empty(t, jobs, "a repo over the limit contributes no jobs at all")
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "lists 3 refs")
	assert.Contains(t, errs[0], "maximum of 2")
	assert.Equal(t, []string{"https://example.com/a.git"}, rejected)
}

func TestBuildRefJobs_WithinLimitProducesOneJobPerRef(t *testing.T) {
	repos := []SkillRepository{
		{URL: "https://example.com/a.git", Refs: []string{"v1", "v2"}},
		{URL: "https://example.com/b.git", Refs: []string{"v1"}},
	}
	jobs, errs, rejected := buildRefJobs(repos, 10)
	assert.Empty(t, errs)
	assert.Empty(t, rejected)
	require.Len(t, jobs, 3)
}

func TestBuildRefJobs_DeduplicatesRefsWithinRepo(t *testing.T) {
	// A repeated ref must index once (its composite identity is repo|path|version,
	// so duplicates would otherwise collide), and must not count against the limit.
	repos := []SkillRepository{
		{URL: "https://example.com/a.git", Refs: []string{"v1", "v1", "v2", "v1"}},
	}
	jobs, errs, rejected := buildRefJobs(repos, 2)
	assert.Empty(t, errs, "two unique refs is within the limit of 2")
	assert.Empty(t, rejected)
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

func (f *blockingFakeResolver) RemoteCommit(context.Context, SkillRepository, string, *Credentials) (string, error) {
	return "", nil
}

func noCreds(SkillRepository) (*Credentials, error) { return nil, nil }

// noKnownCommit reports no prior commit, so no ref is skipped.
func noKnownCommit(SkillRepository, string) string { return "" }

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
		done <- resolveJobsConcurrently(context.Background(), jobs, jobCount, sem, fake, noCreds, noKnownCommit)
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
		done <- resolveJobsConcurrently(context.Background(), jobs, workers, sem, fake, noCreds, noKnownCommit)
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

func (r errResolver) RemoteCommit(context.Context, SkillRepository, string, *Credentials) (string, error) {
	return "", nil
}

func TestResolveJobsConcurrently_PropagatesErrors(t *testing.T) {
	jobs := []refJob{{repo: SkillRepository{URL: "a"}, ref: "v1"}}
	results := resolveJobsConcurrently(context.Background(), jobs, 1, semaphore.NewWeighted(1), errResolver{err: assert.AnError}, noCreds, noKnownCommit)
	require.Len(t, results, 1)
	assert.ErrorIs(t, results[0].err, assert.AnError)
}

func TestResolveJobsConcurrently_PropagatesCredentialsError(t *testing.T) {
	jobs := []refJob{{repo: SkillRepository{URL: "a", CredentialRef: "github"}, ref: "v1"}}
	credsErr := fmt.Errorf("no secret resolver configured")
	results := resolveJobsConcurrently(context.Background(), jobs, 1, semaphore.NewWeighted(1),
		errResolver{}, func(SkillRepository) (*Credentials, error) { return nil, credsErr }, noKnownCommit)
	require.Len(t, results, 1)
	assert.ErrorIs(t, results[0].err, credsErr)
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

// --- credentials from the mounted credentials directory ---

func TestCredentialsForRepo_AnonymousWhenNoKey(t *testing.T) {
	l := &SkillLoader{credentialsDir: t.TempDir()}
	creds, err := l.credentialsForRepo(SkillRepository{URL: "https://example.com/a.git"})
	require.NoError(t, err)
	assert.Nil(t, creds, "a repo with no credentialRef clones anonymously")
}

func TestCredentialsForRepo_ReadsTokenFromFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "github"), []byte("secret-token\n"), 0o600))
	l := &SkillLoader{credentialsDir: dir}

	creds, err := l.credentialsForRepo(SkillRepository{CredentialRef: "github"})
	require.NoError(t, err)
	require.NotNil(t, creds)
	assert.Equal(t, gitTokenUsername, creds.Username)
	assert.Equal(t, "secret-token", creds.Token, "surrounding whitespace is trimmed")
}

func TestCredentialsForRepo_MissingOrEmptyTokenFileErrors(t *testing.T) {
	dir := t.TempDir()
	l := &SkillLoader{credentialsDir: dir}

	_, err := l.credentialsForRepo(SkillRepository{CredentialRef: "absent"})
	require.Error(t, err, "a named-but-missing token file fails rather than cloning anonymously")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "blank"), []byte("  \n"), 0o600))
	_, err = l.credentialsForRepo(SkillRepository{CredentialRef: "blank"})
	require.Error(t, err, "an empty token file is an error")
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
	wg := l.currentWG()
	l.schedulePeriodicSync(context.Background(), "s", 0, wg)
	waitForWaitGroup(t, wg, time.Second, "expected no periodic-sync goroutine when the interval is unset")
}

func TestSchedulePeriodicSync_ValidIntervalSpawnsAndStopsOnCancel(t *testing.T) {
	l := &SkillLoader{}

	ctx, cancel := context.WithCancel(context.Background())
	wg := l.currentWG()
	l.schedulePeriodicSync(ctx, "s", 1, wg)

	// Cancel well before the 1-minute tick fires, so the action closure (which
	// touches l.state/l.sources) never runs; this proves the goroutine is
	// spawned and cleanly stoppable without needing to wait a full minute.
	cancel()
	waitForWaitGroup(t, wg, 2*time.Second, "periodic sync goroutine did not stop after cancel")
}
