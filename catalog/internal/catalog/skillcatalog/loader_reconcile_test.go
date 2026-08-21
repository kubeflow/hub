package skillcatalog

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	sharedmodels "github.com/kubeflow/hub/catalog/internal/db/models"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"

	skillmodels "github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog/models"
)

// --- fakes ---

// fakeLoaderState is a minimal LoaderState that always permits writes and tracks
// in-flight writes with an atomic counter, so tests can wait for launched source
// syncs to finish (mirroring the base's WaitForInflightWrites contract).
type fakeLoaderState struct {
	paths    []string
	inflight int64
}

func (f *fakeLoaderState) IsLeader() bool            { return true }
func (f *fakeLoaderState) ShouldWriteDatabase() bool { return true }
func (f *fakeLoaderState) TrackWrite()               { atomic.AddInt64(&f.inflight, 1) }
func (f *fakeLoaderState) WriteComplete()            { atomic.AddInt64(&f.inflight, -1) }
func (f *fakeLoaderState) WaitForInflightWrites(timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for atomic.LoadInt64(&f.inflight) > 0 {
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(time.Millisecond)
	}
}
func (f *fakeLoaderState) SetCloser(func()) {}
func (f *fakeLoaderState) Paths() []string  { return f.paths }

// fakeSkillRepo is an in-memory SkillRepository that upserts by composite name
// (like the real Save) and records deletions, so tests can assert the loader
// reconciles rather than wipes.
type fakeSkillRepo struct {
	mu             sync.Mutex
	byName         map[string]skillmodels.Skill
	nextID         int32
	deletedIDs     []int32
	deleteBySource int

	// distinctSourceIDs is what GetDistinctSourceIDs reports as already indexed.
	distinctSourceIDs []string

	// onDistinctSourceIDs, when set, is invoked from GetDistinctSourceIDs.
	onDistinctSourceIDs func()
}

func newFakeSkillRepo() *fakeSkillRepo {
	return &fakeSkillRepo{byName: map[string]skillmodels.Skill{}}
}

// seed inserts a pre-existing row attributed to the default test source and repo.
func (f *fakeSkillRepo) seed(name string, id int32) {
	f.seedFor(name, id, "src", "https://example.com/a.git")
}

// seedFor inserts a pre-existing row at the default ref.
func (f *fakeSkillRepo) seedFor(name string, id int32, sourceID, repository string) {
	f.seedForRef(name, id, sourceID, repository, "v1")
}

// seedForRef inserts a pre-existing row carrying the source_id, repository, and
// version properties the real loader writes, so source scoping and both tiers of
// orphan protection (per-repository and per-ref) are exercised rather than
// bypassed by a fixture that omits the fields they key on.
func (f *fakeSkillRepo) seedForRef(name string, id int32, sourceID, repository, version string) {
	src, repoURL, ref := sourceID, repository, version
	e := &skillmodels.SkillImpl{
		ID:         &id,
		Attributes: &skillmodels.SkillAttributes{Name: &name},
		Properties: &[]dbmodels.Properties{
			{Name: propSourceID, StringValue: &src},
			{Name: propRepository, StringValue: &repoURL},
			{Name: propSkillVersion, StringValue: &ref},
		},
	}
	f.byName[name] = e
	if id >= f.nextID {
		f.nextID = id
	}
}

func (f *fakeSkillRepo) Save(entity skillmodels.Skill) (skillmodels.Skill, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	name := *entity.GetAttributes().Name
	if existing, ok := f.byName[name]; ok {
		entity.SetID(*existing.GetID()) // upsert: keep the existing row's ID
	} else {
		f.nextID++
		entity.SetID(f.nextID)
	}
	f.byName[name] = entity
	return entity, nil
}

func (f *fakeSkillRepo) List(*skillmodels.SkillListOptions) (*dbmodels.ListWrapper[skillmodels.Skill], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	items := make([]skillmodels.Skill, 0, len(f.byName))
	for _, s := range f.byName {
		items = append(items, s)
	}
	return &dbmodels.ListWrapper[skillmodels.Skill]{Items: items, Size: int32(len(items))}, nil
}

func (f *fakeSkillRepo) DeleteByID(id int32) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deletedIDs = append(f.deletedIDs, id)
	for name, s := range f.byName {
		if s.GetID() != nil && *s.GetID() == id {
			delete(f.byName, name)
		}
	}
	return nil
}

func (f *fakeSkillRepo) DeleteBySource(string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleteBySource++
	return nil
}

func (f *fakeSkillRepo) names() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.byName))
	for n := range f.byName {
		out = append(out, n)
	}
	return out
}

func (f *fakeSkillRepo) GetByID(int32) (skillmodels.Skill, error) {
	return nil, fmt.Errorf("not implemented")
}
func (f *fakeSkillRepo) GetByName(string) (skillmodels.Skill, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetDistinctSourceIDs runs synchronously inside PerformLeaderOperations, so
// onDistinctSourceIDs is a convenient hook for observing term overlap.
func (f *fakeSkillRepo) GetDistinctSourceIDs() ([]string, error) {
	if f.onDistinctSourceIDs != nil {
		f.onDistinctSourceIDs()
	}
	return f.distinctSourceIDs, nil
}
func (f *fakeSkillRepo) GetTypeID() int32 { return 0 }

// deleteBySourceCount reads the counter under the lock, for tests that observe it
// while another goroutine may be calling DeleteBySource.
func (f *fakeSkillRepo) deleteBySourceCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.deleteBySource
}

// noopSourceRepo satisfies CatalogSourceRepository for status writes.
type noopSourceRepo struct{}

func (noopSourceRepo) GetBySourceID(string) (sharedmodels.CatalogSource, error) { return nil, nil }
func (noopSourceRepo) Save(s sharedmodels.CatalogSource) (sharedmodels.CatalogSource, error) {
	return s, nil
}
func (noopSourceRepo) Delete(string) error                           { return nil }
func (noopSourceRepo) GetAll() ([]sharedmodels.CatalogSource, error) { return nil, nil }
func (noopSourceRepo) GetAllStatuses() (map[string]sharedmodels.SourceStatus, error) {
	return nil, nil
}

// scriptedResolver returns a fixed set of skills (or an error) for every job.
type scriptedResolver struct {
	skills []ResolvedSkill
	err    error
}

func (r scriptedResolver) Resolve(context.Context, SkillRepository, string, *Credentials) ([]ResolvedSkill, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.skills, nil
}

func (r scriptedResolver) RemoteCommit(context.Context, SkillRepository, string, *Credentials) (string, error) {
	return "", nil
}

// perRepoResolver resolves per repository URL, so a test can make one repo
// succeed while another fails.
type perRepoResolver struct {
	byURL map[string][]ResolvedSkill
	errBy map[string]error
}

func (r perRepoResolver) Resolve(_ context.Context, repo SkillRepository, _ string, _ *Credentials) ([]ResolvedSkill, error) {
	if err, ok := r.errBy[repo.URL]; ok {
		return nil, err
	}
	return r.byURL[repo.URL], nil
}

func (r perRepoResolver) RemoteCommit(context.Context, SkillRepository, string, *Credentials) (string, error) {
	return "", nil
}

func newReconcileLoader(repo *fakeSkillRepo, resolver refResolver) *SkillLoader {
	return &SkillLoader{
		state: &fakeLoaderState{},
		services: Services{
			SkillRepository:         repo,
			CatalogSourceRepository: noopSourceRepo{},
		},
		resolver: resolver,
		limits:   SyncLimits{MaxInFlightClones: 4, MaxRefsPerRepo: 10, MaxResolveWorkers: 4},
		sem:      semaphore.NewWeighted(4),
	}
}

func resolvedAt(path string) ResolvedSkill {
	return ResolvedSkill{
		Repository:     "https://example.com/a.git",
		Path:           path,
		Version:        "v1",
		ResolvedCommit: "abc",
		Skill:          &ParsedSkill{Name: "n", Description: "d"},
	}
}

func specWithOneRepo() *SkillSourceSpec {
	return &SkillSourceSpec{
		Repositories: []SkillRepository{{URL: "https://example.com/a.git", Refs: []string{"v1"}}},
	}
}

// --- tests ---

func TestSyncSourceLocked_UpsertsCurrentAndRemovesOrphansWithoutWiping(t *testing.T) {
	repo := newFakeSkillRepo()
	// A skill left over from a previous sync that this sync will not produce.
	orphan := skillEntityName("src", "https://example.com/a.git", "skills/old", "v1")
	repo.seed(orphan, 99)

	resolver := scriptedResolver{skills: []ResolvedSkill{resolvedAt("skills/a"), resolvedAt("skills/b")}}
	l := newReconcileLoader(repo, resolver)

	l.syncSourceLocked(context.Background(), "src", specWithOneRepo())

	nameA := skillEntityName("src", "https://example.com/a.git", "skills/a", "v1")
	nameB := skillEntityName("src", "https://example.com/a.git", "skills/b", "v1")
	assert.ElementsMatch(t, []string{nameA, nameB}, repo.names(), "current skills present, orphan gone")
	assert.Equal(t, []int32{99}, repo.deletedIDs, "only the orphan is deleted, by ID")
	assert.Zero(t, repo.deleteBySource, "reconcile must never wipe the whole source")
}

func TestSyncSourceLocked_ReSyncKeepsIDsStable(t *testing.T) {
	repo := newFakeSkillRepo()
	resolver := scriptedResolver{skills: []ResolvedSkill{resolvedAt("skills/a")}}
	l := newReconcileLoader(repo, resolver)

	l.syncSourceLocked(context.Background(), "src", specWithOneRepo())
	nameA := skillEntityName("src", "https://example.com/a.git", "skills/a", "v1")
	firstID := *repo.byName[nameA].GetID()

	l.syncSourceLocked(context.Background(), "src", specWithOneRepo())
	assert.Equal(t, firstID, *repo.byName[nameA].GetID(), "an unchanged skill keeps its row (upsert, not delete+insert)")
	assert.Empty(t, repo.deletedIDs, "nothing is deleted when the same skills are re-synced")
}

func TestSyncSourceLocked_KeepsStaleDataOnCompleteFailure(t *testing.T) {
	repo := newFakeSkillRepo()
	stale := skillEntityName("src", "https://example.com/a.git", "skills/a", "v1")
	repo.seed(stale, 7)

	// Every resolve fails: nothing indexed, so the last good index must be preserved.
	resolver := scriptedResolver{err: fmt.Errorf("clone failed")}
	l := newReconcileLoader(repo, resolver)

	l.syncSourceLocked(context.Background(), "src", specWithOneRepo())

	assert.Equal(t, []string{stale}, repo.names(), "stale-but-usable skill is kept on complete failure")
	assert.Empty(t, repo.deletedIDs)
}

func TestRemoveOrphans_DeletesOnlyNonCurrent(t *testing.T) {
	repo := newFakeSkillRepo()
	repo.seed("a", 1)
	repo.seed("b", 2)
	repo.seed("c", 3)
	l := newReconcileLoader(repo, scriptedResolver{})

	existing, err := l.listSkillsForSource("src")
	require.NoError(t, err)
	current := mapset.NewSet("a", "c")
	require.NoError(t, l.removeOrphans(existing, current, newUnsafeScope(nil)))

	assert.Equal(t, []int32{2}, repo.deletedIDs, "only the non-current skill b is removed")
	assert.ElementsMatch(t, []string{"a", "c"}, repo.names())
}

func TestSyncSourceLocked_DoesNotDeleteOtherSourcesSkills(t *testing.T) {
	repo := newFakeSkillRepo()
	// A skill owned by a different source. The datastore query layer does not yet
	// filter by source, so List hands it to this sync too; because composite names
	// are namespaced by source ID it can never appear in currentNames.
	foreign := skillEntityName("other-src", "https://example.com/z.git", "skills/z", "v1")
	repo.seedFor(foreign, 42, "other-src", "https://example.com/z.git")

	resolver := scriptedResolver{skills: []ResolvedSkill{resolvedAt("skills/a")}}
	l := newReconcileLoader(repo, resolver)

	l.syncSourceLocked(context.Background(), "src", specWithOneRepo())

	assert.Empty(t, repo.deletedIDs, "another source's skills are never touched")
	assert.Contains(t, repo.names(), foreign)
}

func TestSyncSourceLocked_PartialFailureCleansOnlyHealthyRepos(t *testing.T) {
	repo := newFakeSkillRepo()
	const healthyURL = "https://example.com/a.git"
	const brokenURL = "https://example.com/b.git"

	// One stale skill in each repo; neither is produced by this sync.
	healthyOrphan := skillEntityName("src", healthyURL, "skills/old", "v1")
	brokenStale := skillEntityName("src", brokenURL, "skills/keep", "v1")
	repo.seedFor(healthyOrphan, 1, "src", healthyURL)
	repo.seedFor(brokenStale, 2, "src", brokenURL)

	// The healthy repo resolves; the broken one always errors.
	resolver := perRepoResolver{
		byURL: map[string][]ResolvedSkill{healthyURL: {resolvedAt("skills/a")}},
		errBy: map[string]error{brokenURL: fmt.Errorf("clone failed")},
	}
	l := newReconcileLoader(repo, resolver)

	l.syncSourceLocked(context.Background(), "src", &SkillSourceSpec{
		Repositories: []SkillRepository{
			{URL: healthyURL, Refs: []string{"v1"}},
			{URL: brokenURL, Refs: []string{"v1"}},
		},
	})

	assert.Equal(t, []int32{1}, repo.deletedIDs,
		"the healthy repo is reconciled even though another repo failed")
	assert.Contains(t, repo.names(), brokenStale,
		"the failed repo keeps its stale-but-valid skills")
}

// perRefResolver fails specific (repository, ref) pairs, so a test can break one
// ref of a repository while its siblings resolve.
type perRefResolver struct {
	errBy map[string]error // keyed by refKey(repoURL, ref)
}

func (r perRefResolver) Resolve(_ context.Context, repo SkillRepository, ref string, _ *Credentials) ([]ResolvedSkill, error) {
	if err, ok := r.errBy[refKey(repo.URL, ref)]; ok {
		return nil, err
	}
	return []ResolvedSkill{{
		Repository: repo.URL, Path: "skills/a", Version: ref, ResolvedCommit: "abc",
		Skill: &ParsedSkill{Name: "a", Description: "d"},
	}}, nil
}

func (r perRefResolver) RemoteCommit(context.Context, SkillRepository, string, *Credentials) (string, error) {
	return "", nil
}

// TestSyncSourceLocked_FailedRefDoesNotBlockSiblingRefCleanup pins the granularity
// of orphan protection. One repository is pinned to several refs; an operator drops
// v2 from the config while v3 is temporarily unreachable. Protecting at repository
// granularity would keep v2's now-orphaned skills for as long as v3 kept failing —
// indefinitely, since nothing else would ever re-enumerate them.
func TestSyncSourceLocked_FailedRefDoesNotBlockSiblingRefCleanup(t *testing.T) {
	repo := newFakeSkillRepo()
	const url = "https://example.com/a.git"

	// Already indexed: one skill per ref, including v2 which config no longer lists.
	v1Skill := skillEntityName("src", url, "skills/a", "v1")
	v2Orphan := skillEntityName("src", url, "skills/a", "v2")
	v3Stale := skillEntityName("src", url, "skills/a", "v3")
	repo.seedForRef(v1Skill, 1, "src", url, "v1")
	repo.seedForRef(v2Orphan, 2, "src", url, "v2")
	repo.seedForRef(v3Stale, 3, "src", url, "v3")

	// v3 is unreachable this sync; v1 resolves cleanly. v2 is gone from the config,
	// so no job is built for it at all.
	resolver := perRefResolver{errBy: map[string]error{refKey(url, "v3"): fmt.Errorf("clone failed")}}
	l := newReconcileLoader(repo, resolver)

	l.syncSourceLocked(context.Background(), "src", &SkillSourceSpec{
		Repositories: []SkillRepository{{URL: url, Refs: []string{"v1", "v3"}}},
	})

	assert.Equal(t, []int32{2}, repo.deletedIDs,
		"the removed ref's orphan is cleaned even though a sibling ref of the same repository failed")
	assert.Contains(t, repo.names(), v3Stale,
		"the failed ref keeps its stale-but-valid skills")
	assert.Contains(t, repo.names(), v1Skill,
		"the healthy ref's skill is retained")
}

func TestSyncSourceLocked_RejectedRepoKeepsItsSkills(t *testing.T) {
	repo := newFakeSkillRepo()
	const healthyURL = "https://example.com/a.git"
	const norefsURL = "https://example.com/norefs.git"

	healthyOrphan := skillEntityName("src", healthyURL, "skills/old", "v1")
	rejectedStale := skillEntityName("src", norefsURL, "skills/keep", "v1")
	repo.seedFor(healthyOrphan, 1, "src", healthyURL)
	repo.seedFor(rejectedStale, 2, "src", norefsURL)

	resolver := perRepoResolver{byURL: map[string][]ResolvedSkill{healthyURL: {resolvedAt("skills/a")}}}
	l := newReconcileLoader(repo, resolver)

	// The second repo lists no refs, so buildRefJobs rejects it before any clone.
	l.syncSourceLocked(context.Background(), "src", &SkillSourceSpec{
		Repositories: []SkillRepository{
			{URL: healthyURL, Refs: []string{"v1"}},
			{URL: norefsURL},
		},
	})

	assert.Equal(t, []int32{1}, repo.deletedIDs)
	assert.Contains(t, repo.names(), rejectedStale,
		"a repo rejected before cloning keeps its previously-indexed skills")
}

// commitResolver reports a fixed remote commit and records which refs got a full
// clone, so tests can prove unchanged refs are skipped.
type commitResolver struct {
	mu           sync.Mutex
	remoteCommit string
	resolveRefs  []string
}

func (r *commitResolver) RemoteCommit(context.Context, SkillRepository, string, *Credentials) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.remoteCommit, nil
}

func (r *commitResolver) Resolve(_ context.Context, repo SkillRepository, ref string, _ *Credentials) ([]ResolvedSkill, error) {
	r.mu.Lock()
	r.resolveRefs = append(r.resolveRefs, ref)
	commit := r.remoteCommit
	r.mu.Unlock()
	return []ResolvedSkill{{
		Repository: repo.URL, Path: "skills/a", Version: ref, ResolvedCommit: commit,
		Skill: &ParsedSkill{Name: "a", Description: "d"},
	}}, nil
}

func (r *commitResolver) refs() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.resolveRefs...)
}

func TestSyncSourceLocked_SkipsUnchangedRef(t *testing.T) {
	repo := newFakeSkillRepo()
	res := &commitResolver{remoteCommit: "abc123"}
	l := newReconcileLoader(repo, res)

	// First sync: nothing indexed yet, so the ref is fully resolved and saved.
	l.syncSourceLocked(context.Background(), "src", specWithOneRepo())
	require.Equal(t, []string{"v1"}, res.refs())
	nameA := skillEntityName("src", "https://example.com/a.git", "skills/a", "v1")
	require.Contains(t, repo.names(), nameA)

	// Second sync: the remote commit is unchanged, so the ref must be skipped (no
	// second Resolve) and its skill retained (not orphaned).
	l.syncSourceLocked(context.Background(), "src", specWithOneRepo())
	assert.Equal(t, []string{"v1"}, res.refs(), "unchanged ref is not re-cloned")
	assert.Contains(t, repo.names(), nameA, "skipped ref's skill is retained")
	assert.Empty(t, repo.deletedIDs, "nothing deleted for a skipped ref")
}

// TestSyncSourceLocked_ReIndexesOnceAfterConfigChange covers the case the commit
// check alone cannot see: an operator edits stamped metadata (or a filter) without
// the ref moving. The ref must be re-indexed the next sync, and — because the new
// rows carry the new digest — only that once.
func TestSyncSourceLocked_ReIndexesOnceAfterConfigChange(t *testing.T) {
	repo := newFakeSkillRepo()
	res := &commitResolver{remoteCommit: "abc123"}
	l := newReconcileLoader(repo, res)

	specWith := func(category string) *SkillSourceSpec {
		return &SkillSourceSpec{Repositories: []SkillRepository{
			{URL: "https://example.com/a.git", Refs: []string{"v1"}, Category: category},
		}}
	}

	l.syncSourceLocked(context.Background(), "src", specWith("DevOps"))
	require.Equal(t, []string{"v1"}, res.refs())

	// The ref has not moved, only the config. It must still be re-resolved.
	l.syncSourceLocked(context.Background(), "src", specWith("SRE"))
	require.Equal(t, []string{"v1", "v1"}, res.refs(), "a config change re-indexes an unmoved ref")

	nameA := skillEntityName("src", "https://example.com/a.git", "skills/a", "v1")
	require.Contains(t, repo.byName, nameA)
	assert.Equal(t, "SRE", skillProperty(repo.byName[nameA], propCategory), "the new category is stamped")

	// The refreshed rows carry the new digest, so the change costs exactly one
	// re-index rather than re-cloning on every sync from here on.
	l.syncSourceLocked(context.Background(), "src", specWith("SRE"))
	assert.Equal(t, []string{"v1", "v1"}, res.refs(), "the unchanged config skips again")
	assert.Empty(t, repo.deletedIDs, "nothing is orphaned by a re-index")
}

func TestSyncSourceLocked_ReResolvesChangedRef(t *testing.T) {
	repo := newFakeSkillRepo()
	res := &commitResolver{remoteCommit: "abc123"}
	l := newReconcileLoader(repo, res)

	l.syncSourceLocked(context.Background(), "src", specWithOneRepo())
	res.mu.Lock()
	res.remoteCommit = "def456" // the ref moved to a new commit
	res.mu.Unlock()

	l.syncSourceLocked(context.Background(), "src", specWithOneRepo())
	assert.Equal(t, []string{"v1", "v1"}, res.refs(), "a moved ref is re-cloned")
}

func twoGitSources() *SkillSourceCollection {
	mk := func(id string) basecatalog.PluginSource {
		return basecatalog.PluginSource{
			ID:   id,
			Type: SourceTypeGitSkillsPlugin,
			Properties: map[string]any{
				"repositories": []any{
					map[string]any{"url": "https://example.com/" + id + ".git", "refs": []any{"v1"}},
				},
			},
			Origin: "/cfg",
		}
	}
	sc := NewSkillSourceCollection("/cfg")
	_ = sc.Merge("/cfg", map[string]basecatalog.PluginSource{"a": mk("a"), "b": mk("b")})
	return sc
}

func TestPerformLeaderOperations_SyncsSourcesConcurrently(t *testing.T) {
	repo := newFakeSkillRepo()
	state := &fakeLoaderState{}
	// The resolver blocks until two resolves are in flight at once — reached only if
	// the two sources sync concurrently. A serial loop would stall at one and the
	// test would time out.
	fake := newBlockingFakeResolver(2)

	l := &SkillLoader{
		state:    state,
		services: Services{SkillRepository: repo, CatalogSourceRepository: noopSourceRepo{}},
		resolver: fake,
		limits:   SyncLimits{MaxInFlightClones: 4, MaxRefsPerRepo: 10, MaxResolveWorkers: 4},
		sem:      semaphore.NewWeighted(4),
		sources:  twoGitSources(),
		tickerWG: &sync.WaitGroup{},
	}

	go func() { _ = l.PerformLeaderOperations(context.Background(), mapset.NewSet[string]()) }()

	select {
	case <-fake.atCap:
	case <-time.After(5 * time.Second):
		t.Fatal("sources did not sync concurrently: resolver never reached two in-flight")
	}
	close(fake.release)

	state.WaitForInflightWrites(5 * time.Second)
	assert.Zero(t, atomic.LoadInt64(&state.inflight), "all launched source syncs finished")
}

// gitSourceWithInterval builds a single git source that also schedules a periodic
// resync, so PerformLeaderOperations registers a ticker on the term WaitGroup.
func gitSourceWithInterval(id string, intervalMinutes int) *SkillSourceCollection {
	sc := NewSkillSourceCollection("/cfg")
	_ = sc.Merge("/cfg", map[string]basecatalog.PluginSource{id: {
		ID:   id,
		Type: SourceTypeGitSkillsPlugin,
		Properties: map[string]any{
			"repositories": []any{
				map[string]any{"url": "https://example.com/" + id + ".git", "refs": []any{"v1"}},
			},
			"syncIntervalMinutes": intervalMinutes,
		},
		Origin: "/cfg",
	}})
	return sc
}

// impatientState mimics a term whose source syncs outlive the 30s inflight-write
// budget: WaitForInflightWrites gives up immediately, so the next term proceeds
// while the previous term's goroutines are still running. This is the condition
// under which a deferred WaitGroup.Add races the next term's Wait.
type impatientState struct{ *fakeLoaderState }

func (impatientState) WaitForInflightWrites(time.Duration) {}

func newLeaderOpsLoader(sources *SkillSourceCollection, resolver refResolver, repo *fakeSkillRepo) *SkillLoader {
	return &SkillLoader{
		state:    impatientState{&fakeLoaderState{}},
		services: Services{SkillRepository: repo, CatalogSourceRepository: noopSourceRepo{}},
		resolver: resolver,
		limits:   SyncLimits{MaxInFlightClones: 4, MaxRefsPerRepo: 10, MaxResolveWorkers: 4},
		sem:      semaphore.NewWeighted(4),
		sources:  sources,
		tickerWG: &sync.WaitGroup{},
	}
}

func TestPerformLeaderOperations_SerializesTerms(t *testing.T) {
	repo := newFakeSkillRepo()
	var inTerm, maxConcurrent int64
	repo.onDistinctSourceIDs = func() {
		n := atomic.AddInt64(&inTerm, 1)
		for {
			prev := atomic.LoadInt64(&maxConcurrent)
			if n <= prev || atomic.CompareAndSwapInt64(&maxConcurrent, prev, n) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond) // widen the window for an overlap to show
		atomic.AddInt64(&inTerm, -1)
	}

	l := newLeaderOpsLoader(gitSourceWithInterval("a", 60), scriptedResolver{}, repo)

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = l.PerformLeaderOperations(context.Background(), mapset.NewSet[string]())
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(1), atomic.LoadInt64(&maxConcurrent),
		"leader terms must not overlap: the swapWG/Wait handoff is only sound one term at a time")
}

// gateResolver blocks every resolve until its gate is closed, so a test can hold
// a source sync in flight while it inspects the term that launched it.
type gateResolver struct{ gate chan struct{} }

func (r gateResolver) Resolve(context.Context, SkillRepository, string, *Credentials) ([]ResolvedSkill, error) {
	<-r.gate
	return nil, nil
}

func (r gateResolver) RemoteCommit(context.Context, SkillRepository, string, *Credentials) (string, error) {
	return "", nil
}

// TestPerformLeaderOperations_RegistersTickerBeforeReturning pins the ordering
// that makes the swapWG/Wait handoff sound: a term's resync tickers must be on
// the term WaitGroup by the time the term returns. Registering them from the
// async sync goroutine instead lets a slow sync outlive the inflight-write drain
// and call Add on a WaitGroup the next term is already waiting on, which panics
// with "WaitGroup misuse: Add called concurrently with Wait".
func TestPerformLeaderOperations_RegistersTickerBeforeReturning(t *testing.T) {
	repo := newFakeSkillRepo()
	gate := make(chan struct{})
	l := newLeaderOpsLoader(gitSourceWithInterval("a", 60), gateResolver{gate: gate}, repo)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, l.PerformLeaderOperations(ctx, mapset.NewSet[string]()))

	// The source sync is still blocked inside Resolve, so this is exactly the
	// window in which the next term would run. The ticker must already be counted.
	termWG := l.currentWG()
	waited := make(chan struct{})
	go func() { termWG.Wait(); close(waited) }()

	select {
	case <-waited:
		t.Fatal("term returned before its resync ticker was registered on the WaitGroup")
	case <-time.After(250 * time.Millisecond):
	}

	cancel()
	close(gate)
	termWG.Wait()
}

// TestRemoveSkillsFromMissingSources_WaitsForInFlightSync pins that cleanup takes
// the per-source lock. A previous term's initial sync goroutine is tracked only by
// the inflight-write counter, and that drain is best-effort — it gives up after its
// timeout. Deleting a source underneath such a straggler lets the straggler's
// remaining Save calls resurrect rows for a source that is no longer enabled, with
// nothing to clean them up until the next leader term.
func TestRemoveSkillsFromMissingSources_WaitsForInFlightSync(t *testing.T) {
	repo := newFakeSkillRepo()
	repo.distinctSourceIDs = []string{"gone"}

	l := newReconcileLoader(repo, scriptedResolver{})
	// "gone" is indexed but no longer configured, so cleanup targets it.
	l.sources = NewSkillSourceCollection("/cfg")

	// Stand in for the previous term's sync goroutine: it holds "gone"'s lock until
	// release is closed.
	holding := make(chan struct{})
	release := make(chan struct{})
	go l.runSyncExclusive("gone", true, func() {
		close(holding)
		<-release
	})
	<-holding

	done := make(chan error, 1)
	go func() { done <- l.removeSkillsFromMissingSources(mapset.NewSet[string]()) }()

	select {
	case <-done:
		t.Fatal("cleanup deleted source \"gone\" while a sync for it was still in flight")
	case <-time.After(250 * time.Millisecond):
	}
	assert.Zero(t, repo.deleteBySourceCount(), "DeleteBySource must wait for the in-flight sync")

	close(release)
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for cleanup after the in-flight sync released the lock")
	}
	assert.Equal(t, 1, repo.deleteBySourceCount(), "cleanup proceeds once the sync finishes")
}

// countingPropertyOptionsRepo records Refresh calls so a test can assert the
// filter-options view is rebuilt after a sync.
type countingPropertyOptionsRepo struct {
	refreshes int64
}

func (c *countingPropertyOptionsRepo) Refresh(sharedmodels.PropertyOptionType) error {
	atomic.AddInt64(&c.refreshes, 1)
	return nil
}

func (c *countingPropertyOptionsRepo) List(sharedmodels.PropertyOptionType, int32) ([]sharedmodels.PropertyOption, error) {
	return nil, nil
}

// Filter options are served from a materialized view that only changes when something
// refreshes it. Nothing else does so on the skill catalog's behalf, so a source that
// syncs without this leaves its provider/category out of the filter sidebar, and values
// dropped from a source linger there.
func TestPerformLeaderOperations_RefreshesPropertyOptionsAfterSync(t *testing.T) {
	po := &countingPropertyOptionsRepo{}
	fake := newBlockingFakeResolver(2)
	close(fake.release) // never block; this test is about the refresh, not concurrency
	l := &SkillLoader{
		state: &fakeLoaderState{},
		services: Services{
			SkillRepository:           newFakeSkillRepo(),
			CatalogSourceRepository:   noopSourceRepo{},
			PropertyOptionsRepository: po,
		},
		resolver: fake,
		limits:   SyncLimits{MaxInFlightClones: 4, MaxRefsPerRepo: 10, MaxResolveWorkers: 4},
		sem:      semaphore.NewWeighted(4),
		sources:  twoGitSources(),
		tickerWG: &sync.WaitGroup{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, l.PerformLeaderOperations(ctx, mapset.NewSet[string]()))
	l.state.WaitForInflightWrites(5 * time.Second)

	// The refresher debounces, so a burst of source syncs collapses into at least one
	// rebuild rather than one per source.
	require.Eventually(t, func() bool {
		return atomic.LoadInt64(&po.refreshes) > 0
	}, 5*time.Second, 50*time.Millisecond, "property options view was never refreshed after sync")
}
