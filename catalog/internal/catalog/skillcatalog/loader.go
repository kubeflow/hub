package skillcatalog

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/golang/glog"
	"golang.org/x/sync/semaphore"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog/models"
)

// Defaults for SyncLimits.
const (
	defaultMaxInFlightClones = 50
	defaultMaxRefsPerRepo    = 10
	defaultMaxResolveWorkers = 10
)

// defaultGitCredentialsDir is where a git-credentials Secret is expected to be
// mounted. Each file in it is a token, named by a repository's credentialRef.
const defaultGitCredentialsDir = "/etc/skill-catalog/git-credentials"

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
	// RemoteCommit returns the commit a ref currently points to, without cloning,
	// so an unchanged ref can be skipped.
	RemoteCommit(ctx context.Context, repo SkillRepository, ref string, creds *Credentials) (string, error)
}

// gitTokenUsername is the username paired with a token when authenticating to a
// git host. Hosts that accept a PAT as the password (GitHub, GitLab, …) ignore or
// accept this placeholder, so a per-source username is not needed.
const gitTokenUsername = "x-access-token"

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

// WithResolveLimits overrides the per-clone resolver limits (clone timeout and
// maximum checked-out repository size). Any non-default allowed protocols on the
// current resolver are preserved so this option does not silently discard a
// transport allowlist that was set by a prior option.
func WithResolveLimits(limits ResolveLimits) LoaderOption {
	return func(l *SkillLoader) {
		var opts []Option
		if r, ok := l.resolver.(*RepoResolver); ok && len(r.allowedProtocols) > 0 {
			protocols := make([]string, 0, len(r.allowedProtocols))
			for p := range r.allowedProtocols {
				protocols = append(protocols, p)
			}
			opts = append(opts, WithAllowedProtocols(protocols...))
		}
		l.resolver = NewRepoResolver(limits, opts...)
	}
}

// WithCredentialsDir overrides the directory holding per-repository git tokens
// (the mounted git-credentials Secret). An empty dir keeps the default.
func WithCredentialsDir(dir string) LoaderOption {
	return func(l *SkillLoader) {
		if dir != "" {
			l.credentialsDir = dir
		}
	}
}

// SkillLoader syncs skills from git repositories into the datastore.
type SkillLoader struct {
	state basecatalog.LoaderState

	sources        *SkillSourceCollection
	services       Services
	resolver       refResolver
	limits         SyncLimits
	sem            *semaphore.Weighted
	credentialsDir string

	closerMu sync.Mutex
	closer   func()

	// termMu serializes PerformLeaderOperations. Leadership acquisition and the
	// config-reload watcher call it from separate goroutines, and the
	// swapWG/prevWG.Wait() handoff is only sound for one term at a time: two
	// concurrent terms could have one calling Add on the very WaitGroup the other
	// is already waiting on, which panics the process.
	termMu sync.Mutex

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

// SourceCollection returns the loader's underlying source collection, used to
// resolve sourceLabel query params to source IDs.
func (l *SkillLoader) SourceCollection() *SkillSourceCollection {
	return l.sources
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
		credentialsDir: defaultGitCredentialsDir,
		tickerWG:       &sync.WaitGroup{},
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

	// One term at a time: see termMu. Every WaitGroup.Add for this term happens
	// before this call returns and releases the lock, so the next term's
	// prevWG.Wait() can never observe an Add in flight.
	l.termMu.Lock()
	defer l.termMu.Unlock()

	ctx, cancel := context.WithCancel(ctx)

	// Swap in a fresh WaitGroup for this term and cancel the previous context so
	// old ticker goroutines stop. prevWG.Wait() then drains them before we begin
	// writing, preventing a stale goroutine from racing with the new sync.
	prevWG := l.swapWG()
	termWG := l.currentWG()
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
		// Sync each source concurrently rather than serially, so one large or slow
		// source does not delay the rest. The outer TrackWrite/WriteComplete spans
		// the whole source sync so the base's WaitForInflightWrites blocks until
		// every source's initial index is built before readiness is signalled;
		// clone concurrency across sources stays bounded by the global in-flight
		// clone semaphore.
		l.state.TrackWrite()
		go func(id string, spec *SkillSourceSpec) {
			defer l.state.WriteComplete()
			l.syncSource(ctx, id, spec)
		}(id, spec)

		// Register the resync ticker synchronously, not from the goroutine above:
		// its WaitGroup.Add must happen before this term returns. Deferring it until
		// after syncSource would let a slow sync outlive the 30s inflight-write
		// drain and call Add on a WaitGroup the next term is already waiting on.
		// A tick that fires while the initial sync is still running is skipped by
		// runSyncExclusive's TryLock, so starting the ticker early is harmless.
		l.schedulePeriodicSync(ctx, id, spec.SyncIntervalMinutes, termWG)
	}

	glog.Info("skill loader leader operations launched")
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

// syncSourceLocked reconciles one source from an already-parsed spec. The caller
// must hold the source's lock. It resolves every repository at every ref (bounded
// per sync by the worker pool, and globally by the in-flight clone cap), upserts
// each discovered skill, and then deletes only the skills that are no longer
// present (removed skill dirs, refs, or repos) — so a read never sees the source
// empty mid-sync, unlike a delete-then-rebuild.
func (l *SkillLoader) syncSourceLocked(ctx context.Context, sourceID string, spec *SkillSourceSpec) {
	jobs, errs, rejectedRepos := buildRefJobs(spec.Repositories, l.limits.MaxRefsPerRepo)

	if !l.state.ShouldWriteDatabase() || ctx.Err() != nil {
		return
	}

	// Load the source's currently-indexed skills once, used both to skip unchanged
	// refs (ref-level skip) and for orphan cleanup. A list error is non-fatal: the
	// sync proceeds with no known commits (every ref is resolved) and no orphan pass.
	existing, lerr := l.listSkillsForSource(sourceID)
	if lerr != nil {
		glog.Warningf("skill source %s: unable to list existing skills (%v); resolving all refs", sourceID, lerr)
	}
	knownCommit, namesByRef := indexExistingByRef(existing)

	results := resolveJobsConcurrently(ctx, jobs, l.limits.MaxResolveWorkers, l.sem, l.resolver, l.credentialsForRepo, knownCommit)

	// currentNames collects the composite name of every skill still present this
	// sync — freshly upserted or retained from a skipped (unchanged) ref. Anything
	// else still attributed to the source is an orphan.
	currentNames := mapset.NewSet[string]()
	// unsafe collects what this sync could not fully determine, so orphan cleanup
	// leaves it alone rather than turning a transient failure into permanent data
	// loss. Everything it does not cover is still reconciled, so one flaky ref or
	// repository does not freeze cleanup for the rest of the source.
	unsafe := newUnsafeScope(rejectedRepos)
	var warningMsgs []string
	indexed, skipped := 0, 0
	for _, res := range results {
		if res.err != nil {
			errs = append(errs, fmt.Sprintf("%s@%s: %v", res.repo.URL, res.ref, res.err))
			unsafe.addRef(res.repo.URL, res.ref)
			continue
		}
		if res.skipped {
			currentNames.Append(namesByRef[refKey(res.repo.URL, res.ref)]...)
			skipped++
			continue
		}
		for j := range res.skills {
			for _, w := range res.skills[j].Skill.Warnings {
				warningMsgs = append(warningMsgs, fmt.Sprintf("%s: %s", res.skills[j].Path, w))
			}
			entity := buildSkillEntity(res.skills[j], res.repo, sourceID)
			if attrs := entity.GetAttributes(); attrs != nil && attrs.Name != nil {
				currentNames.Add(*attrs.Name)
			}
			l.state.TrackWrite()
			_, serr := l.services.SkillRepository.Save(entity)
			l.state.WriteComplete()
			if serr != nil {
				errs = append(errs, fmt.Sprintf("saving %s: %v", res.skills[j].Path, serr))
				unsafe.addRef(res.repo.URL, res.ref)
				continue
			}
			indexed++
		}
	}

	// If leadership was lost or the context was cancelled mid-sync, currentNames is
	// incomplete; skip reconciliation and the status write so live skills are not
	// deleted as false orphans.
	if !l.state.ShouldWriteDatabase() || ctx.Err() != nil {
		return
	}
	// Reconcile only what this sync fully enumerated. A list error leaves `existing`
	// unknown, so cleanup is skipped entirely; otherwise each (repository, ref) is
	// cleaned or protected independently via unsafe.
	if lerr == nil {
		if err := l.removeOrphans(existing, currentNames, unsafe); err != nil {
			errs = append(errs, fmt.Sprintf("removing orphaned skills: %v", err))
		}
	}

	status, msg := skillSourceStatus(indexed+skipped, warningMsgs, errs)
	basecatalog.SaveSourceStatus(l.services.CatalogSourceRepository, sourceID, status, msg)
	glog.Infof("skill source %s: indexed %d skills, skipped %d unchanged refs, %d warnings, %d errors",
		sourceID, indexed, skipped, len(warningMsgs), len(errs))
}

// listSkillsForSource returns every skill currently attributed to sourceID. List
// with no page size returns all rows, so a single call sees every candidate.
//
// The SourceIDs list option is not yet honoured by the query layer, so the result
// is narrowed to sourceID here as well. This filter is load-bearing, not just
// defensive: composite skill names are namespaced by source ID, so another
// source's skills can never appear in this sync's currentNames and removeOrphans
// would delete every one of them.
func (l *SkillLoader) listSkillsForSource(sourceID string) ([]models.Skill, error) {
	sourceIDs := []string{sourceID}
	result, err := l.services.SkillRepository.List(&models.SkillListOptions{SourceIDs: &sourceIDs})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	owned := make([]models.Skill, 0, len(result.Items))
	for _, s := range result.Items {
		if skillProperty(s, propSourceID) == sourceID {
			owned = append(owned, s)
		}
	}
	return owned, nil
}

// skillProperty returns the named string property of a skill, or "" if absent.
func skillProperty(s models.Skill, name string) string {
	props := s.GetProperties()
	if props == nil {
		return ""
	}
	for _, p := range *props {
		if p.Name == name && p.StringValue != nil {
			return *p.StringValue
		}
	}
	return ""
}

// refKey is the map key identifying a (repository, ref) pair.
func refKey(repoURL, ref string) string { return repoURL + "\x00" + ref }

// indexExistingByRef groups already-indexed skills by (repository, ref), returning
// the commit each ref was last resolved at and the composite names indexed under
// it. All skills at one ref share a resolvedCommit and configDigest, so the last
// one wins.
//
// The returned lookup reports no known commit when the repository's configuration
// has changed since the ref was indexed, which forces a re-resolve: an unmoved ref
// still needs re-indexing when a category, label, or include filter was edited.
// Rows written before configDigest existed have an empty digest and so refresh
// once on upgrade.
func indexExistingByRef(existing []models.Skill) (knownCommitFunc, map[string][]string) {
	commitByRef := map[string]string{}
	digestByRef := map[string]string{}
	namesByRef := map[string][]string{}
	for _, s := range existing {
		name, repo, version, commit, digest := skillRefInfo(s)
		if name == "" || repo == "" || version == "" {
			continue
		}
		k := refKey(repo, version)
		namesByRef[k] = append(namesByRef[k], name)
		if commit != "" {
			commitByRef[k] = commit
		}
		digestByRef[k] = digest
	}
	known := func(repo SkillRepository, ref string) string {
		k := refKey(repo.URL, ref)
		if digestByRef[k] != configDigest(repo) {
			return ""
		}
		return commitByRef[k]
	}
	return known, namesByRef
}

// skillRefInfo extracts a skill's composite name and the repository, version,
// resolved commit, and config digest it was indexed under.
func skillRefInfo(s models.Skill) (name, repo, version, commit, digest string) {
	if a := s.GetAttributes(); a != nil && a.Name != nil {
		name = *a.Name
	}
	if props := s.GetProperties(); props != nil {
		for _, p := range *props {
			if p.StringValue == nil {
				continue
			}
			switch p.Name {
			case propRepository:
				repo = *p.StringValue
			case propSkillVersion:
				version = *p.StringValue
			case propResolvedCommit:
				commit = *p.StringValue
			case propConfigDigest:
				digest = *p.StringValue
			}
		}
	}
	return
}

// unsafeScope records what a sync could not enumerate, at the finest granularity
// the failure allows.
//
// Repository-level entries cover repositories rejected before any job was built
// (no refs configured, or more refs than the limit): nothing about them was
// enumerated, so every skill they own must be kept.
//
// Ref-level entries cover individual (repository, ref) jobs that failed to
// resolve or save. Protecting the whole repository for those would be too coarse:
// a repository pinned to several refs, with one temporarily unreachable, would
// keep orphans belonging to its other refs — including refs an operator has since
// removed from the config — for as long as the one ref stays broken. Tracking the
// failure against the ref that actually failed lets the siblings reconcile.
type unsafeScope struct {
	repos mapset.Set[string]
	refs  mapset.Set[string]
}

func newUnsafeScope(rejectedRepos []string) *unsafeScope {
	return &unsafeScope{
		repos: mapset.NewSet(rejectedRepos...),
		refs:  mapset.NewSet[string](),
	}
}

// addRef marks one (repository, ref) as not fully enumerated.
func (u *unsafeScope) addRef(repoURL, ref string) { u.refs.Add(refKey(repoURL, ref)) }

// protects reports whether a skill indexed at (repoURL, version) must be kept
// regardless of its absence from currentNames, and why (for logging).
func (u *unsafeScope) protects(repoURL, version string) (bool, string) {
	switch {
	case u.repos.Contains(repoURL):
		return true, "its repository did not fully sync"
	case u.refs.Contains(refKey(repoURL, version)):
		return true, "its ref did not fully sync"
	default:
		return false, ""
	}
}

// removeOrphans deletes, from the pre-loaded existing set, every skill whose
// composite name is not in currentNames — i.e. skills that a previous sync indexed
// but this one neither upserted nor retained via a skipped ref. Skills covered by
// unsafe are left alone: this sync did not fully enumerate them, so their absence
// from currentNames proves nothing.
func (l *SkillLoader) removeOrphans(existing []models.Skill, currentNames mapset.Set[string], unsafe *unsafeScope) error {
	var errs []error
	for _, skill := range existing {
		attrs := skill.GetAttributes()
		if attrs == nil || attrs.Name == nil || skill.GetID() == nil {
			continue
		}
		if currentNames.Contains(*attrs.Name) {
			continue
		}
		if kept, why := unsafe.protects(skillProperty(skill, propRepository), skillProperty(skill, propSkillVersion)); kept {
			glog.V(1).Infof("keeping skill %q: %s", *attrs.Name, why)
			continue
		}
		glog.Infof("removing orphaned skill %q", *attrs.Name)
		l.state.TrackWrite()
		delErr := l.services.SkillRepository.DeleteByID(*skill.GetID())
		l.state.WriteComplete()
		if delErr != nil {
			errs = append(errs, fmt.Errorf("deleting orphaned skill %q: %w", *attrs.Name, delErr))
		}
	}
	return errors.Join(errs...)
}

// schedulePeriodicSync starts a background resync loop for a source when its
// syncIntervalMinutes property is set, on top of the existing debounced
// hot-reload and manual-sync triggers. The loop stops when ctx is cancelled —
// leadership lost, or a later PerformLeaderOperations call supersedes it via
// setCloser (which cancels this ctx before deriving and scheduling a fresh one).
func (l *SkillLoader) schedulePeriodicSync(ctx context.Context, sourceID string, intervalMinutes int, wg *sync.WaitGroup) {
	if intervalMinutes <= 0 {
		return
	}
	interval := time.Duration(intervalMinutes) * time.Minute

	runPeriodicSync(ctx, interval, wg, func() {
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

// credentialsForRepo resolves a repository's git credentials by reading the token
// file named by its credentialRef from the mounted credentials directory. A repo
// with no credentialRef clones anonymously (nil, nil); one whose token file is
// missing or empty fails clearly rather than silently cloning anonymously. The key
// is validated as a plain filename at config-parse time, so it cannot escape the
// directory. The file is re-read each time, so a hot-reloaded Secret is picked up.
func (l *SkillLoader) credentialsForRepo(repo SkillRepository) (*Credentials, error) {
	if repo.CredentialRef == "" {
		return nil, nil
	}
	data, err := os.ReadFile(filepath.Join(l.credentialsDir, repo.CredentialRef))
	if err != nil {
		return nil, fmt.Errorf("reading git token for credentialRef %q: %w", repo.CredentialRef, err)
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return nil, fmt.Errorf("git token for credentialRef %q is empty", repo.CredentialRef)
	}
	return &Credentials{Username: gitTokenUsername, Token: token}, nil
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
// colliding entries. The URLs of rejected repositories are returned alongside the
// errors so orphan cleanup can leave their already-indexed skills in place.
func buildRefJobs(repos []SkillRepository, maxRefsPerRepo int) (jobs []refJob, errs []string, rejected []string) {
	for _, repo := range repos {
		refs := dedupeStrings(repo.Refs)
		switch {
		case len(refs) == 0:
			errs = append(errs, fmt.Sprintf("%s: no refs configured; skills must be pinned to a tag or commit SHA", repo.URL))
			rejected = append(rejected, repo.URL)
		case len(refs) > maxRefsPerRepo:
			errs = append(errs, fmt.Sprintf("%s: lists %d refs, exceeding the maximum of %d", repo.URL, len(refs), maxRefsPerRepo))
			rejected = append(rejected, repo.URL)
		default:
			for _, ref := range refs {
				jobs = append(jobs, refJob{repo: repo, ref: ref})
			}
		}
	}
	return jobs, errs, rejected
}

// dedupeStrings returns values with duplicates removed, preserving first-seen order.
func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// resolveResult is the outcome of resolving one refJob. skipped is true when the
// ref's remote commit was unchanged since the last sync, so it was not re-cloned
// and its existing skills should be retained.
type resolveResult struct {
	repo    SkillRepository
	ref     string
	skills  []ResolvedSkill
	err     error
	skipped bool
}

// resolveJobsConcurrently resolves every job through a fixed FIFO worker pool of
// at most `workers` goroutines (0 -> defaultMaxResolveWorkers). Jobs are fed to
// the workers in order; each worker takes one job and processes it to completion,
// so the worker count — not the job count — bounds how many goroutines run and how
// many credential lookups (e.g. Kubernetes Secret fetches) happen at once. Clones
// are additionally bounded across all concurrent syncs by sem. Every job produces
// a result, indexed to match the input order.
// knownCommit reports the commit a (repo, ref) was last indexed at, or "" if it
// was not previously indexed. It lets resolveOne skip a ref whose remote commit is
// unchanged.
type knownCommitFunc func(repo SkillRepository, ref string) string

func resolveJobsConcurrently(ctx context.Context, jobs []refJob, workers int, sem *semaphore.Weighted, resolver refResolver, credentials func(SkillRepository) (*Credentials, error), knownCommit knownCommitFunc) []resolveResult {
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
				results[ij.idx] = resolveOne(ctx, ij.job, knownCommit(ij.job.repo, ij.job.ref), sem, resolver, credentials)
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

// resolveOne resolves a single job to completion: credential lookup, an optional
// unchanged-ref skip, then a clone gated by the global in-flight-clone semaphore.
// When knownCommit is non-empty and the ref's current remote commit still matches
// it, the clone is skipped (the ref is immutable, so its content cannot have
// changed). A remote-commit lookup error is non-fatal — it falls through to a full
// resolve. Credential lookup and the ls-remote check run before the clone slot is
// acquired so neither holds one; both are still bounded by the worker pool.
func resolveOne(ctx context.Context, job refJob, knownCommit string, sem *semaphore.Weighted, resolver refResolver, credentials func(SkillRepository) (*Credentials, error)) resolveResult {
	if ctx.Err() != nil {
		return resolveResult{repo: job.repo, ref: job.ref, err: ctx.Err()}
	}
	creds, err := credentials(job.repo)
	if err != nil {
		return resolveResult{repo: job.repo, ref: job.ref, err: err}
	}
	if knownCommit != "" {
		if remote, rerr := resolver.RemoteCommit(ctx, job.repo, job.ref, creds); rerr == nil && remote == knownCommit {
			return resolveResult{repo: job.repo, ref: job.ref, skipped: true}
		}
	}
	if err := sem.Acquire(ctx, 1); err != nil {
		return resolveResult{repo: job.repo, ref: job.ref, err: err}
	}
	defer sem.Release(1)

	skills, err := resolver.Resolve(ctx, job.repo, job.ref, creds)
	return resolveResult{repo: job.repo, ref: job.ref, skills: skills, err: err}
}

// removeSkillsFromMissingSources deletes skills whose source is gone or disabled.
// Each DeleteBySource takes the per-source lock: the previous term's initial sync
// goroutine is tracked only by the inflight-write counter, and that drain is
// best-effort (WaitForInflightWrites gives up after its timeout), so a slow sync
// can still be saving skills for a source this pass is about to delete. Blocking
// is bounded — the previous term's context is already cancelled before the drain,
// and clones run under it plus CloneTimeout, so a straggler unwinds promptly.
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
		var delErr error
		l.runSyncExclusive(oldSource, true, func() {
			l.state.TrackWrite()
			delErr = l.services.SkillRepository.DeleteBySource(oldSource)
			l.state.WriteComplete()
		})
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
