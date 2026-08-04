package skillcatalog

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/golang/glog"
	"golang.org/x/sync/semaphore"
	"golang.org/x/sync/singleflight"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
)

// Defaults for SyncLimits.
const (
	defaultMaxInFlightClones = 50
	defaultMaxRefsPerRepo    = 10
	defaultMaxResolveWorkers = 10
)

// SyncLimits bound the loader's sync fan-out so a large or misconfigured source
// list cannot saturate the pod.
type SyncLimits struct {
	// MaxInFlightClones caps concurrent Resolve (clone) calls across all
	// concurrently-syncing sources — a global backstop shared by every sync.
	// 0 -> defaultMaxInFlightClones.
	MaxInFlightClones int64
	// MaxRefsPerRepo rejects a repository entry that lists more refs than this.
	// 0 -> defaultMaxRefsPerRepo.
	MaxRefsPerRepo int
	// MaxResolveWorkers caps how many refs a single source sync resolves at once
	// via a fixed FIFO worker pool. Each worker takes one job and processes it to
	// completion (credential lookup, then clone), so this bounds per-sync
	// goroutines and Secret lookups, not just clones. 0 -> defaultMaxResolveWorkers.
	MaxResolveWorkers int
}

// refResolver resolves one repository at one ref. Satisfied by *RepoResolver;
// tests inject a fake to control concurrency and error behavior without git.
type refResolver interface {
	Resolve(ctx context.Context, repo SkillRepository, ref string, creds *Credentials) ([]ResolvedSkill, error)
}

// SecretResolver fetches git credentials referenced by a repository entry's
// authSecretName.
type SecretResolver interface {
	GetCredentials(ctx context.Context, secretName string) (*Credentials, error)
}

// LoaderOption configures a SkillLoader.
type LoaderOption func(*SkillLoader)

// WithSyncLimits overrides the default sync fan-out limits.
func WithSyncLimits(limits SyncLimits) LoaderOption {
	return func(l *SkillLoader) {
		if limits.MaxInFlightClones > 0 {
			l.limits.MaxInFlightClones = limits.MaxInFlightClones
		}
		if limits.MaxRefsPerRepo > 0 {
			l.limits.MaxRefsPerRepo = limits.MaxRefsPerRepo
		}
		if limits.MaxResolveWorkers > 0 {
			l.limits.MaxResolveWorkers = limits.MaxResolveWorkers
		}
		l.sem = semaphore.NewWeighted(l.limits.MaxInFlightClones)
	}
}

// WithSecretResolver configures how the loader resolves authSecretName into
// git credentials for private repositories. Without one, a repository entry
// that sets authSecretName fails to sync with a clear error.
func WithSecretResolver(r SecretResolver) LoaderOption {
	return func(l *SkillLoader) { l.secretResolver = r }
}

// SkillLoader syncs skills from git repositories into the datastore.
type SkillLoader struct {
	state basecatalog.LoaderState

	sources        *SkillSourceCollection
	services       Services
	resolver       refResolver
	limits         SyncLimits
	sem            *semaphore.Weighted
	secretResolver SecretResolver

	closerMu sync.Mutex
	closer   func()

	// tickerMu guards tickerWG so PerformLeaderOperations can swap it per term
	// while schedulePeriodicSync reads it concurrently.
	tickerMu sync.Mutex
	tickerWG *sync.WaitGroup

	// syncLocks serializes syncs of the same source so a periodic tick cannot
	// interleave its DeleteBySource+reindex with a concurrent sync (another tick,
	// or a reload-triggered PerformLeaderOperations) and drop or duplicate rows.
	syncLocksMu sync.Mutex
	syncLocks   map[string]*sync.Mutex
}

// currentWG returns the WaitGroup for the current leader term, creating it on
// first use. Callers must not store the returned pointer across swapWG calls.
func (l *SkillLoader) currentWG() *sync.WaitGroup {
	l.tickerMu.Lock()
	defer l.tickerMu.Unlock()
	if l.tickerWG == nil {
		l.tickerWG = &sync.WaitGroup{}
	}
	return l.tickerWG
}

// swapWG replaces the current term's WaitGroup with a fresh one and returns
// the old one so the caller can wait for previous-term goroutines to drain.
func (l *SkillLoader) swapWG() *sync.WaitGroup {
	l.tickerMu.Lock()
	defer l.tickerMu.Unlock()
	prev := l.tickerWG
	if prev == nil {
		prev = &sync.WaitGroup{}
	}
	l.tickerWG = &sync.WaitGroup{}
	return prev
}

// AllSources returns a snapshot of the currently loaded sources. Use this
// instead of accessing the internal sources field directly from outside the
// package.
func (l *SkillLoader) AllSources() map[string]basecatalog.PluginSource {
	return l.sources.AllSources()
}

func (l *SkillLoader) setCloser(closer func()) {
	l.closerMu.Lock()
	defer l.closerMu.Unlock()
	if l.closer != nil {
		l.closer()
	}
	l.closer = closer
}

func NewSkillLoader(services Services, state basecatalog.LoaderState, opts ...LoaderOption) *SkillLoader {
	paths := state.Paths()
	l := &SkillLoader{
		state:    state,
		sources:  NewSkillSourceCollection(paths...),
		services: services,
		resolver: NewRepoResolver(ResolveLimits{}),
		limits: SyncLimits{
			MaxInFlightClones: defaultMaxInFlightClones,
			MaxRefsPerRepo:    defaultMaxRefsPerRepo,
			MaxResolveWorkers: defaultMaxResolveWorkers,
		},
		tickerWG: &sync.WaitGroup{},
	}
	l.sem = semaphore.NewWeighted(l.limits.MaxInFlightClones)
	for _, opt := range opts {
		opt(l)
	}
	return l
}

func (l *SkillLoader) ParseAllConfigs() error {
	glog.Info("Initializing skill loader - parsing configs")
	for _, path := range l.state.Paths() {
		if err := l.parseAndMerge(path); err != nil {
			return fmt.Errorf("failed to parse skill config %s: %w", path, err)
		}
	}
	glog.Info("skill loader config parsing complete")
	return nil
}

func (l *SkillLoader) PerformLeaderOperations(ctx context.Context, allKnownSourceIDs mapset.Set[string]) error {
	glog.Info("skill loader performing leader operations")

	ctx, cancel := context.WithCancel(ctx)

	// Swap in a fresh WaitGroup for this term and cancel the previous context so
	// old ticker goroutines stop. prevWG.Wait() then drains them before we begin
	// writing, preventing a stale goroutine from racing with the new sync.
	prevWG := l.swapWG()
	l.setCloser(cancel)

	// Drain in-flight DB writes from the previous term first, then wait for the
	// ticker goroutines themselves (which exit promptly after context cancellation).
	l.state.WaitForInflightWrites(30 * time.Second)
	prevWG.Wait()

	if err := l.removeSkillsFromMissingSources(allKnownSourceIDs); err != nil {
		glog.Errorf("error removing skills from missing sources: %v", err)
	}

	for id, source := range l.sources.AllSources() {
		if !source.IsEnabled() {
			basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, id, basecatalog.SourceStatusDisabled, "")
			continue
		}
		if source.Type != SourceTypeGitSkillsPlugin {
			glog.Warningf("unknown skill provider type: %s", source.Type)
			basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, id, basecatalog.SourceStatusError, "unknown provider type: "+source.Type)
			continue
		}
		if !l.state.ShouldWriteDatabase() {
			glog.Info("No longer leader, stopping skill database writes")
			return nil
		}
		spec, err := ParseSkillSource(source)
		if err != nil {
			basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, id, basecatalog.SourceStatusError, err.Error())
			continue
		}
		l.syncSource(ctx, id, spec)
		l.schedulePeriodicSync(ctx, id, spec.SyncIntervalMinutes)
	}

	glog.Info("skill loader leader operations complete")
	return nil
}

// sourceLock returns the per-source mutex, creating it on first use.
func (l *SkillLoader) sourceLock(sourceID string) *sync.Mutex {
	l.syncLocksMu.Lock()
	defer l.syncLocksMu.Unlock()
	if l.syncLocks == nil {
		l.syncLocks = make(map[string]*sync.Mutex)
	}
	m, ok := l.syncLocks[sourceID]
	if !ok {
		m = &sync.Mutex{}
		l.syncLocks[sourceID] = m
	}
	return m
}

// forgetSourceLock drops the per-source lock for a source that no longer exists,
// so the map does not grow without bound as sources come and go.
func (l *SkillLoader) forgetSourceLock(sourceID string) {
	l.syncLocksMu.Lock()
	defer l.syncLocksMu.Unlock()
	delete(l.syncLocks, sourceID)
}

// runSyncExclusive runs fn holding sourceID's lock. When block is true it waits
// for any in-progress sync (used by the authoritative leader/reload sync); when
// false it returns immediately with false if a sync is already running (used by
// the periodic ticker, which should skip rather than pile up).
func (l *SkillLoader) runSyncExclusive(sourceID string, block bool, fn func()) bool {
	mu := l.sourceLock(sourceID)
	if block {
		mu.Lock()
	} else if !mu.TryLock() {
		return false
	}
	defer mu.Unlock()
	fn()
	return true
}

// syncSource rebuilds the index for one source, waiting for any in-progress sync
// of the same source to finish first. spec is the already-parsed configuration.
func (l *SkillLoader) syncSource(ctx context.Context, sourceID string, spec *SkillSourceSpec) {
	l.runSyncExclusive(sourceID, true, func() {
		l.syncSourceLocked(ctx, sourceID, spec)
	})
}

// syncSourceLocked performs the actual drop-and-reindex from an already-parsed
// spec. The caller must hold the source's lock. It resolves every repository at
// every ref (bounded per sync by the worker pool, and globally by the in-flight
// clone cap) and re-indexes what it finds; the ephemeral, rebuild-each-sync model
// guarantees no orphans.
func (l *SkillLoader) syncSourceLocked(ctx context.Context, sourceID string, spec *SkillSourceSpec) {
	l.state.TrackWrite()
	delErr := l.services.SkillRepository.DeleteBySource(sourceID)
	l.state.WriteComplete()
	if delErr != nil {
		basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, sourceID, basecatalog.SourceStatusError, delErr.Error())
		return
	}

	jobs, errs := buildRefJobs(spec.Repositories, l.limits.MaxRefsPerRepo)

	if !l.state.ShouldWriteDatabase() || ctx.Err() != nil {
		return
	}
	// Cache credentials for this source's sync so a repo with several refs (or
	// several repos sharing a Secret) triggers at most one Secret lookup per name.
	credentials := newCredentialCache(func(repo SkillRepository) (*Credentials, error) {
		return l.credentialsFor(ctx, repo)
	})
	results := resolveJobsConcurrently(ctx, jobs, l.limits.MaxResolveWorkers, l.sem, l.resolver, credentials)

	var warningMsgs []string
	indexed := 0
	for _, res := range results {
		if res.err != nil {
			errs = append(errs, fmt.Sprintf("%s@%s: %v", res.repo.URL, res.ref, res.err))
			continue
		}
		for j := range res.skills {
			for _, w := range res.skills[j].Skill.Warnings {
				warningMsgs = append(warningMsgs, fmt.Sprintf("%s: %s", res.skills[j].Path, w))
			}
			entity := buildSkillEntity(res.skills[j], res.repo, spec, sourceID)
			l.state.TrackWrite()
			_, serr := l.services.SkillRepository.Save(entity)
			l.state.WriteComplete()
			if serr != nil {
				errs = append(errs, fmt.Sprintf("saving %s: %v", res.skills[j].Path, serr))
				continue
			}
			indexed++
		}
	}

	status, msg := skillSourceStatus(indexed, warningMsgs, errs)
	basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, sourceID, status, msg)
	glog.Infof("skill source %s: indexed %d skills, %d warnings, %d errors", sourceID, indexed, len(warningMsgs), len(errs))
}

// schedulePeriodicSync starts a background resync loop for a source when its
// syncIntervalMinutes property is set, on top of the existing debounced
// hot-reload and manual-sync triggers. The loop stops when ctx is cancelled —
// leadership lost, or a later PerformLeaderOperations call supersedes it via
// setCloser (which cancels this ctx before deriving and scheduling a fresh one).
func (l *SkillLoader) schedulePeriodicSync(ctx context.Context, sourceID string, intervalMinutes int) {
	if intervalMinutes <= 0 {
		return
	}
	interval := time.Duration(intervalMinutes) * time.Minute

	runPeriodicSync(ctx, interval, l.currentWG(), func() {
		if !l.state.ShouldWriteDatabase() {
			return
		}
		current, ok := l.sources.AllSources()[sourceID]
		if !ok || !current.IsEnabled() || current.Type != SourceTypeGitSkillsPlugin {
			return
		}
		// Re-parse the current snapshot each tick so a hot-reloaded config is
		// picked up; a config that has since become invalid marks the source in
		// error rather than syncing stale data.
		spec, err := ParseSkillSource(current)
		if err != nil {
			basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, sourceID, basecatalog.SourceStatusError, err.Error())
			return
		}
		// Skip this tick if a sync is already running for this source, rather
		// than piling a second drop-and-reindex on top of it.
		ran := l.runSyncExclusive(sourceID, false, func() {
			glog.Infof("skill source %s: periodic sync triggered (every %s)", sourceID, interval)
			l.syncSourceLocked(ctx, sourceID, spec)
		})
		if !ran {
			glog.Infof("skill source %s: previous sync still running, skipping periodic tick", sourceID)
		}
	})
}

// runPeriodicSync runs action every interval until ctx is cancelled, tracking
// the goroutine on wg. It is a general-purpose ticker loop, kept independent of
// minutes/spec semantics so it can be tested with arbitrarily short intervals.
func runPeriodicSync(ctx context.Context, interval time.Duration, wg *sync.WaitGroup, action func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				action()
			}
		}
	}()
}

// credentialsFor resolves a repository entry's authSecretName into git
// credentials. A repository with no authSecretName resolves anonymously
// (nil, nil). A repository that requests a secret with no resolver configured
// fails clearly rather than silently attempting an anonymous clone.
func (l *SkillLoader) credentialsFor(ctx context.Context, repo SkillRepository) (*Credentials, error) {
	if repo.AuthSecretName == "" {
		return nil, nil
	}
	if l.secretResolver == nil {
		return nil, fmt.Errorf("repository requires secret %q but no secret resolver is configured", repo.AuthSecretName)
	}
	return l.secretResolver.GetCredentials(ctx, repo.AuthSecretName)
}

// newCredentialCache wraps fetch so each distinct authSecretName is resolved at
// most once for the lifetime of the returned function: concurrent lookups for the
// same secret are collapsed (singleflight) and the result is memoized, so a repo
// with N refs (or several repos sharing a Secret) causes one Secret fetch, not N.
// Anonymous repos (no authSecretName) are never cached — nothing to fetch.
func newCredentialCache(fetch func(SkillRepository) (*Credentials, error)) func(SkillRepository) (*Credentials, error) {
	type entry struct {
		creds *Credentials
		err   error
	}
	var (
		mu    sync.Mutex
		cache = map[string]entry{}
		group singleflight.Group
	)
	return func(repo SkillRepository) (*Credentials, error) {
		key := repo.AuthSecretName
		if key == "" {
			return fetch(repo)
		}

		mu.Lock()
		if e, ok := cache[key]; ok {
			mu.Unlock()
			return e.creds, e.err
		}
		mu.Unlock()

		v, err, _ := group.Do(key, func() (any, error) {
			creds, ferr := fetch(repo)
			mu.Lock()
			cache[key] = entry{creds: creds, err: ferr}
			mu.Unlock()
			return creds, ferr
		})
		creds, _ := v.(*Credentials)
		return creds, err
	}
}

// refJob is one (repository, ref) pair to resolve.
type refJob struct {
	repo SkillRepository
	ref  string
}

// buildRefJobs expands each repository's refs into individual jobs, rejecting
// (with an error, not a hang or a silent truncation) any repository with no
// refs configured or with more refs than maxRefsPerRepo. Duplicate refs within a
// repository are collapsed so a repeated ref indexes once rather than producing
// colliding entries.
func buildRefJobs(repos []SkillRepository, maxRefsPerRepo int) ([]refJob, []string) {
	var jobs []refJob
	var errs []string
	for _, repo := range repos {
		refs := dedupeStrings(repo.Refs)
		switch {
		case len(refs) == 0:
			errs = append(errs, fmt.Sprintf("%s: no refs configured; skills must be pinned to a tag or commit SHA", repo.URL))
		case len(refs) > maxRefsPerRepo:
			errs = append(errs, fmt.Sprintf("%s: lists %d refs, exceeding the maximum of %d", repo.URL, len(refs), maxRefsPerRepo))
		default:
			for _, ref := range refs {
				jobs = append(jobs, refJob{repo: repo, ref: ref})
			}
		}
	}
	return jobs, errs
}

// dedupeStrings returns values with duplicates removed, preserving first-seen order.
func dedupeStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

// resolveResult is the outcome of resolving one refJob.
type resolveResult struct {
	repo   SkillRepository
	ref    string
	skills []ResolvedSkill
	err    error
}

// resolveJobsConcurrently resolves every job through a fixed FIFO worker pool of
// at most `workers` goroutines (0 -> defaultMaxResolveWorkers). Jobs are fed to
// the workers in order; each worker takes one job and processes it to completion,
// so the worker count — not the job count — bounds how many goroutines run and how
// many credential lookups (e.g. Kubernetes Secret fetches) happen at once. Clones
// are additionally bounded across all concurrent syncs by sem. Every job produces
// a result, indexed to match the input order.
func resolveJobsConcurrently(ctx context.Context, jobs []refJob, workers int, sem *semaphore.Weighted, resolver refResolver, credentials func(SkillRepository) (*Credentials, error)) []resolveResult {
	results := make([]resolveResult, len(jobs))
	if len(jobs) == 0 {
		return results
	}
	if workers <= 0 {
		workers = defaultMaxResolveWorkers
	}
	workers = min(workers, len(jobs))

	type indexedJob struct {
		idx int
		job refJob
	}
	queue := make(chan indexedJob)

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ij := range queue {
				results[ij.idx] = resolveOne(ctx, ij.job, sem, resolver, credentials)
			}
		}()
	}

	// Feed jobs in order; workers return promptly on a cancelled context, so this
	// drains without blocking even mid-cancellation.
	for i, job := range jobs {
		queue <- indexedJob{idx: i, job: job}
	}
	close(queue)

	wg.Wait()
	return results
}

// resolveOne resolves a single job to completion: credential lookup first, then a
// clone gated by the global in-flight-clone semaphore. The credential lookup runs
// before the clone slot is acquired so a slow lookup never holds one; it is still
// bounded because resolveOne runs on a worker-pool goroutine.
func resolveOne(ctx context.Context, job refJob, sem *semaphore.Weighted, resolver refResolver, credentials func(SkillRepository) (*Credentials, error)) resolveResult {
	if ctx.Err() != nil {
		return resolveResult{repo: job.repo, ref: job.ref, err: ctx.Err()}
	}
	creds, err := credentials(job.repo)
	if err != nil {
		return resolveResult{repo: job.repo, ref: job.ref, err: err}
	}
	if err := sem.Acquire(ctx, 1); err != nil {
		return resolveResult{repo: job.repo, ref: job.ref, err: err}
	}
	defer sem.Release(1)

	skills, err := resolver.Resolve(ctx, job.repo, job.ref, creds)
	return resolveResult{repo: job.repo, ref: job.ref, skills: skills, err: err}
}

// removeSkillsFromMissingSources deletes skills whose source is gone or disabled.
// Its DeleteBySource calls do not take the per-source lock, which is safe because
// it only targets sources that are no longer enabled — those have no active
// periodic ticker (schedulePeriodicSync's tick skips disabled sources) and no
// concurrent syncSource.
func (l *SkillLoader) removeSkillsFromMissingSources(allKnownSourceIDs mapset.Set[string]) error {
	enabledSourceIDs := mapset.NewSet[string]()
	skillSourceIDs := mapset.NewSet[string]()
	for id, source := range l.sources.AllSources() {
		skillSourceIDs.Add(id)
		if source.IsEnabled() {
			enabledSourceIDs.Add(id)
		}
	}

	existingSourceIDs, err := l.services.SkillRepository.GetDistinctSourceIDs()
	if err != nil {
		return fmt.Errorf("unable to retrieve existing skill source IDs: %w", err)
	}

	for oldSource := range mapset.NewSet(existingSourceIDs...).Difference(enabledSourceIDs).Iter() {
		glog.Infof("Removing skills from source %s", oldSource)
		l.state.TrackWrite()
		delErr := l.services.SkillRepository.DeleteBySource(oldSource)
		l.state.WriteComplete()
		if delErr != nil {
			return fmt.Errorf("unable to remove skills from source %q: %w", oldSource, delErr)
		}
		if !skillSourceIDs.Contains(oldSource) {
			if err := l.services.CatalogSourceRepository.Delete(oldSource); err != nil {
				glog.Errorf("failed to delete status for skill source %s: %v", oldSource, err)
			}
			l.forgetSourceLock(oldSource)
		}
	}

	protectedSourceIDs := skillSourceIDs.Union(allKnownSourceIDs)
	if err := basecatalog.CleanupOrphanedCatalogSources(l.services.CatalogSourceRepository, protectedSourceIDs); err != nil {
		glog.Errorf("failed to cleanup orphaned skill catalog sources: %v", err)
	}
	return nil
}

// skillSourceStatus derives the source status from a sync's outcome.
// warningMsgs carries per-skill warning details and is included in the message
// so operators can see why a source is PartiallyAvailable without inspecting logs.
func skillSourceStatus(indexed int, warningMsgs, errs []string) (status, message string) {
	switch {
	case len(errs) > 0 && indexed == 0:
		return basecatalog.SourceStatusError, strings.Join(errs, "; ")
	case len(errs) > 0 || len(warningMsgs) > 0:
		all := slices.Concat(errs, warningMsgs)
		return basecatalog.SourceStatusPartiallyAvailable, strings.Join(all, "; ")
	default:
		return basecatalog.SourceStatusAvailable, ""
	}
}

func (l *SkillLoader) ReloadParsing() error {
	var errs []error
	for _, path := range l.state.Paths() {
		if err := l.parseAndMerge(path); err != nil {
			errs = append(errs, fmt.Errorf("unable to reload skill sources from %s: %w", path, err))
		}
	}
	return errors.Join(errs...)
}

func (l *SkillLoader) parseAndMerge(path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %v", path, err)
	}

	config, err := basecatalog.ReadSourceConfig(path)
	if err != nil {
		return err
	}

	return l.updateSources(path, config)
}

func (l *SkillLoader) updateSources(path string, config *basecatalog.SourceConfig) error {
	sources := make(map[string]basecatalog.PluginSource, len(config.SkillCatalogs))

	for _, source := range config.SkillCatalogs {
		glog.Infof("reading skill catalog config type %s...", source.Type)
		if source.GetId() == "" {
			return fmt.Errorf("invalid skill source: missing id")
		}
		if _, exists := sources[source.GetId()]; exists {
			return fmt.Errorf("invalid skill source: duplicate id %s", source.GetId())
		}

		source.Origin = path
		sources[source.GetId()] = source
		glog.Infof("loaded skill source %s of type %s", source.GetId(), source.Type)
	}

	return l.sources.Merge(path, sources)
}
