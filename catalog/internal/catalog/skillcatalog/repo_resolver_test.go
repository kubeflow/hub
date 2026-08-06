package skillcatalog

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- git fixture helpers ---

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(),
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v: %s", args, out)
	return string(out)
}

func initFixtureRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q", "-b", "main")
	return dir
}

func writeSkill(t *testing.T, repo, dir, name, description string) {
	t.Helper()
	body := "---\nname: " + name + "\ndescription: " + description + "\n---\n# " + name + "\n\nBody.\n"
	writeRepoFile(t, repo, filepath.Join(dir, "SKILL.md"), body)
}

func writeRepoFile(t *testing.T, repo, rel, content string) {
	t.Helper()
	full := filepath.Join(repo, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o600))
}

func commitAll(t *testing.T, repo, msg string) string {
	t.Helper()
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", msg)
	return strings.TrimSpace(runGit(t, repo, "rev-parse", "HEAD"))
}

// testResolver allows the local `file`/path transport used by fixture repos.
func testResolver() *RepoResolver {
	return NewRepoResolver(ResolveLimits{CloneTimeout: 30 * time.Second}, WithAllowedProtocols("file"))
}

func repoEntry(url string, scanPaths, included, excluded []string) SkillRepository {
	return SkillRepository{URL: url, ScanPaths: scanPaths, IncludedSkills: included, ExcludedSkills: excluded}
}

// --- tests ---

func TestResolve_AtCommitSHA(t *testing.T) {
	repo := initFixtureRepo(t)
	writeSkill(t, repo, "skills/deploy", "deploy", "Deploy the app.")
	sha := commitAll(t, repo, "init")

	skills, err := testResolver().Resolve(context.Background(), repoEntry(repo, nil, nil, nil), sha, nil)
	require.NoError(t, err)
	require.Len(t, skills, 1)

	s := skills[0]
	assert.Equal(t, repo, s.Repository)
	assert.Equal(t, "skills/deploy", s.Path)
	assert.Equal(t, sha, s.Version, "a commit SHA is the version verbatim")
	assert.Equal(t, sha, s.ResolvedCommit)
	require.NotNil(t, s.Skill)
	assert.Equal(t, "deploy", s.Skill.Name)
	assert.Equal(t, "Deploy the app.", s.Skill.Description)
}

func TestResolve_BranchRefRejected(t *testing.T) {
	repo := initFixtureRepo(t)
	writeSkill(t, repo, "s", "s", "S.")
	commitAll(t, repo, "init")

	_, err := testResolver().Resolve(context.Background(), repoEntry(repo, nil, nil, nil), "main", nil)
	require.Error(t, err, "a branch ref must be rejected; skills require a tag or commit SHA")

	_, err = testResolver().Resolve(context.Background(), repoEntry(repo, nil, nil, nil), "HEAD", nil)
	require.Error(t, err, "HEAD must be rejected")
}

func TestResolve_TagKeepsVersion(t *testing.T) {
	repo := initFixtureRepo(t)
	writeSkill(t, repo, "deploy", "deploy", "Deploy.")
	sha := commitAll(t, repo, "init")
	runGit(t, repo, "tag", "v1.0")

	skills, err := testResolver().Resolve(context.Background(), repoEntry(repo, nil, nil, nil), "v1.0", nil)
	require.NoError(t, err)
	require.Len(t, skills, 1)
	assert.Equal(t, "v1.0", skills[0].Version, "a tag ref is kept as-is")
	assert.Equal(t, sha, skills[0].ResolvedCommit)
}

func TestResolve_ScanPathsRestrictScan(t *testing.T) {
	repo := initFixtureRepo(t)
	writeSkill(t, repo, "skills/a", "a", "A.")
	writeSkill(t, repo, "other/b", "b", "B.")
	sha := commitAll(t, repo, "init")

	skills, err := testResolver().Resolve(context.Background(), repoEntry(repo, []string{"skills"}, nil, nil), sha, nil)
	require.NoError(t, err)
	require.Len(t, skills, 1)
	assert.Equal(t, "skills/a", skills[0].Path)
}

func TestResolve_IncludeExcludeFilters(t *testing.T) {
	repo := initFixtureRepo(t)
	writeSkill(t, repo, "skills/deploy", "deploy", "D.")
	writeSkill(t, repo, "skills/deploy-draft", "deploy-draft", "Draft.")
	sha := commitAll(t, repo, "init")

	skills, err := testResolver().Resolve(context.Background(),
		repoEntry(repo, nil, []string{"*"}, []string{"*-draft"}), sha, nil)
	require.NoError(t, err)
	require.Len(t, skills, 1)
	assert.Equal(t, "deploy", skills[0].Skill.Name)
}

func TestResolve_CollectsSupportingFiles(t *testing.T) {
	repo := initFixtureRepo(t)
	writeSkill(t, repo, "skills/deploy", "deploy", "D.")
	writeRepoFile(t, repo, "skills/deploy/scripts/run.sh", "echo hi\n")
	writeRepoFile(t, repo, "skills/deploy/reference.md", "ref\n")
	sha := commitAll(t, repo, "init")

	skills, err := testResolver().Resolve(context.Background(), repoEntry(repo, nil, nil, nil), sha, nil)
	require.NoError(t, err)
	require.Len(t, skills, 1)
	assert.ElementsMatch(t,
		[]string{"skills/deploy/scripts/run.sh", "skills/deploy/reference.md"},
		skills[0].SupportingFiles,
		"SKILL.md itself is not a supporting file")
}

func TestResolve_SupportingFilesAreSorted(t *testing.T) {
	// Supporting files are accumulated by ranging over maps, whose iteration order
	// Go randomizes. Without an explicit sort the same repository content would
	// yield a different order on each resolve, churning the persisted JSON array
	// and reshuffling the API response. Files are spread across several
	// directories so an unsorted result is overwhelmingly likely to be caught.
	repo := initFixtureRepo(t)
	writeSkill(t, repo, "skills/deploy", "deploy", "D.")
	writeRepoFile(t, repo, "skills/deploy/z.md", "z\n")
	writeRepoFile(t, repo, "skills/deploy/a.md", "a\n")
	writeRepoFile(t, repo, "skills/deploy/scripts/run.sh", "echo hi\n")
	writeRepoFile(t, repo, "skills/deploy/scripts/helper.sh", "echo helper\n")
	writeRepoFile(t, repo, "skills/deploy/docs/guide.md", "guide\n")
	sha := commitAll(t, repo, "init")

	want := []string{
		"skills/deploy/a.md",
		"skills/deploy/docs/guide.md",
		"skills/deploy/scripts/helper.sh",
		"skills/deploy/scripts/run.sh",
		"skills/deploy/z.md",
	}

	// Repeat: map ordering is re-randomized per iteration, so a single pass could
	// coincidentally come out sorted.
	for i := range 5 {
		skills, err := testResolver().Resolve(context.Background(), repoEntry(repo, nil, nil, nil), sha, nil)
		require.NoError(t, err)
		require.Len(t, skills, 1)
		assert.Equal(t, want, skills[0].SupportingFiles, "resolve %d returned files out of order", i)
	}
}

func TestResolve_NestedSkillFilesAttributedToNearestSkill(t *testing.T) {
	// A skill nested inside another skill's tree must own its own files; the outer
	// skill must not absorb the inner skill's supporting files.
	repo := initFixtureRepo(t)
	writeSkill(t, repo, "skills/outer", "outer", "Outer.")
	writeRepoFile(t, repo, "skills/outer/outer.txt", "outer\n")
	writeSkill(t, repo, "skills/outer/inner", "inner", "Inner.")
	writeRepoFile(t, repo, "skills/outer/inner/inner.txt", "inner\n")
	sha := commitAll(t, repo, "init")

	skills, err := testResolver().Resolve(context.Background(), repoEntry(repo, []string{"skills"}, nil, nil), sha, nil)
	require.NoError(t, err)

	byPath := map[string][]string{}
	for _, s := range skills {
		byPath[s.Path] = s.SupportingFiles
	}
	assert.ElementsMatch(t, []string{"skills/outer/outer.txt"}, byPath["skills/outer"],
		"outer skill owns only its own file, not the nested skill's")
	assert.ElementsMatch(t, []string{"skills/outer/inner/inner.txt"}, byPath["skills/outer/inner"],
		"inner skill owns its own file")
}

func TestResolve_SkipsMalformedSkillButSucceeds(t *testing.T) {
	repo := initFixtureRepo(t)
	writeSkill(t, repo, "skills/good", "good", "Good.")
	// A SKILL.md with no description is skipped by the parser.
	writeRepoFile(t, repo, "skills/bad/SKILL.md", "---\nname: bad\n---\nbody\n")
	sha := commitAll(t, repo, "init")

	skills, err := testResolver().Resolve(context.Background(), repoEntry(repo, nil, nil, nil), sha, nil)
	require.NoError(t, err)
	require.Len(t, skills, 1)
	assert.Equal(t, "good", skills[0].Skill.Name)
}

func TestResolve_SymlinkedSkillMdNotFollowed(t *testing.T) {
	// A SKILL.md that is a symlink (potentially to a host file) must be skipped,
	// not read and indexed.
	repo := initFixtureRepo(t)
	writeSkill(t, repo, "skills/good", "good", "Good.")
	// A secret file outside the skill dir, and a skill dir whose SKILL.md links to it.
	writeRepoFile(t, repo, "secret.txt", "---\nname: evil\ndescription: leaked\n---\nsecret body\n")
	require.NoError(t, os.MkdirAll(filepath.Join(repo, "skills/evil"), 0o755))
	require.NoError(t, os.Symlink(filepath.Join(repo, "secret.txt"), filepath.Join(repo, "skills/evil/SKILL.md")))
	sha := commitAll(t, repo, "init")

	skills, err := testResolver().Resolve(context.Background(), repoEntry(repo, []string{"skills"}, nil, nil), sha, nil)
	require.NoError(t, err)
	require.Len(t, skills, 1, "only the real skill is indexed; the symlinked SKILL.md is skipped")
	assert.Equal(t, "good", skills[0].Skill.Name)
}

func TestRemoteCommit_MatchesResolvedCommit(t *testing.T) {
	repo := initFixtureRepo(t)
	writeSkill(t, repo, "s", "s", "S.")
	sha := commitAll(t, repo, "init")
	runGit(t, repo, "tag", "v1.0")                          // lightweight tag
	runGit(t, repo, "tag", "-a", "v2.0", "-m", "annotated") // annotated tag

	r := testResolver()
	ctx := context.Background()
	entry := repoEntry(repo, nil, nil, nil)

	lightweight, err := r.RemoteCommit(ctx, entry, "v1.0", nil)
	require.NoError(t, err)
	assert.Equal(t, sha, lightweight, "a lightweight tag resolves to its commit")

	annotated, err := r.RemoteCommit(ctx, entry, "v2.0", nil)
	require.NoError(t, err)
	assert.Equal(t, sha, annotated, "an annotated tag is peeled to its commit, not the tag object")

	fromSHA, err := r.RemoteCommit(ctx, entry, sha, nil)
	require.NoError(t, err)
	assert.Equal(t, sha, fromSHA, "a commit SHA resolves to itself")

	// The value must equal what Resolve records, or the skip would be inaccurate.
	skills, err := r.Resolve(ctx, entry, "v2.0", nil)
	require.NoError(t, err)
	require.Len(t, skills, 1)
	assert.Equal(t, annotated, skills[0].ResolvedCommit)
}

func TestResolve_UnknownRefErrors(t *testing.T) {
	repo := initFixtureRepo(t)
	writeSkill(t, repo, "s", "s", "S.")
	commitAll(t, repo, "init")

	_, err := testResolver().Resolve(context.Background(), repoEntry(repo, nil, nil, nil), "does-not-exist", nil)
	require.Error(t, err)
}

func TestCheckImmutableRef(t *testing.T) {
	repo := initFixtureRepo(t)
	writeSkill(t, repo, "s", "s", "S.")
	sha := commitAll(t, repo, "init")
	runGit(t, repo, "tag", "v1.0")

	r := testResolver()
	_, env, err := r.gitEnv(nil)
	require.NoError(t, err)
	ctx := context.Background()
	dir := t.TempDir()

	assert.NoError(t, r.checkImmutableRef(ctx, dir, repo, "v1.0", env), "a tag is immutable")
	assert.NoError(t, r.checkImmutableRef(ctx, dir, repo, sha, env), "a commit SHA is immutable")
	assert.Error(t, r.checkImmutableRef(ctx, dir, repo, "main", env), "a branch is refused")
	assert.Error(t, r.checkImmutableRef(ctx, dir, repo, "HEAD", env), "HEAD is refused")

	// Fail closed: when the branch probe itself fails (unreadable repo), the ref
	// is rejected rather than assumed non-branch.
	err = r.checkImmutableRef(ctx, dir, "/nonexistent/repo.git", "v1.0", env)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to verify")
}

func TestResolve_RejectsLeadingDashArgs(t *testing.T) {
	// A url or ref beginning with '-' could be misparsed by git as an option.
	_, err := testResolver().Resolve(context.Background(), repoEntry("--upload-pack=touch /tmp/x", nil, nil, nil), "v1.0", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "url")

	_, err = testResolver().Resolve(context.Background(), repoEntry("https://example.com/a.git", nil, nil, nil), "--output=/tmp/x", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ref")
}

func TestResolve_RejectsDisallowedProtocol(t *testing.T) {
	// The default resolver only allows https/http/git; a file path/URL is refused,
	// which also blocks dangerous transports like ext::.
	r := NewRepoResolver(ResolveLimits{})
	_, err := r.Resolve(context.Background(), repoEntry("ext::sh -c touch /tmp/pwned", nil, nil, nil), "main", nil)
	require.Error(t, err)
}
