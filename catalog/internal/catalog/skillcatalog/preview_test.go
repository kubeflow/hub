package skillcatalog

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// previewFakeResolver returns canned skills (or errors) per (repository, ref),
// letting preview tests exercise the full parse/resolve/filter path without git.
type previewFakeResolver struct {
	skills map[string][]ResolvedSkill
	errs   map[string]error
}

func (f *previewFakeResolver) Resolve(_ context.Context, repo SkillRepository, ref string, _ *Credentials) ([]ResolvedSkill, error) {
	k := repo.URL + "@" + ref
	if err := f.errs[k]; err != nil {
		return nil, err
	}
	return f.skills[k], nil
}

func (f *previewFakeResolver) RemoteCommit(_ context.Context, _ SkillRepository, _ string, _ *Credentials) (string, error) {
	return "", nil
}

func newTestPreviewer(resolver refResolver) *SkillPreviewer {
	return &SkillPreviewer{
		resolver:       resolver,
		credentialsDir: "/nonexistent",
		limits:         SyncLimits{MaxInFlightClones: 10, MaxRefsPerRepo: 10, MaxResolveWorkers: 4},
	}
}

func resolved(path, name string) ResolvedSkill {
	return ResolvedSkill{Path: path, Version: "v1.0", ResolvedCommit: "abc", Skill: &ParsedSkill{Name: name}}
}

func TestPreviewSkillSource_IncludesAndExcludes(t *testing.T) {
	fake := &previewFakeResolver{skills: map[string][]ResolvedSkill{
		"https://example.com/a.git@v1.0": {
			resolved("skills/deploy", "deploy"),
			resolved("skills/rollout-draft", "rollout-draft"),
		},
	}}
	p := newTestPreviewer(fake)

	cfg := `
type: git-skills-plugin
properties:
  repositories:
    - url: https://example.com/a.git
      refs: ["v1.0"]
      includedSkills: ["*"]
      excludedSkills: ["*-draft"]
`
	results, err := p.PreviewSkillSource(context.Background(), []byte(cfg))
	require.NoError(t, err)
	require.Len(t, results, 2)

	byName := map[string]bool{}
	for _, r := range results {
		byName[r.Name] = r.Included
	}
	assert.True(t, byName["deploy"], "deploy should be included")
	assert.False(t, byName["rollout-draft"], "rollout-draft should be excluded but still listed")
}

func TestPreviewSkillSource_AcceptsJSONConfig(t *testing.T) {
	// The source-management UI collects form fields; the BFF serializes them into
	// the config payload (typically JSON) and posts it inline — there is no config
	// file. Confirm the parser accepts a JSON body (not just YAML), with the single
	// repository wrapped under properties.repositories exactly as the save path writes it.
	fake := &previewFakeResolver{skills: map[string][]ResolvedSkill{
		"https://example.com/a.git@v1.0": {resolved("skills/deploy", "deploy")},
	}}
	p := newTestPreviewer(fake)

	cfg := `{"assetType":"skills","type":"git-skills-plugin","properties":{"repositories":[{"url":"https://example.com/a.git","refs":["v1.0"],"includedSkills":["*"]}]}}`
	results, err := p.PreviewSkillSource(context.Background(), []byte(cfg))
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "deploy", results[0].Name)
	assert.True(t, results[0].Included)
}

func TestPreviewSkillSource_RootLevelSkillUsesFrontmatterName(t *testing.T) {
	fake := &previewFakeResolver{skills: map[string][]ResolvedSkill{
		"https://example.com/a.git@v1.0": {resolved(".", "whole-repo-skill")},
	}}
	p := newTestPreviewer(fake)

	cfg := `
type: git-skills-plugin
properties:
  repositories:
    - url: https://example.com/a.git
      refs: ["v1.0"]
      excludedSkills: ["whole-repo-skill"]
`
	results, err := p.PreviewSkillSource(context.Background(), []byte(cfg))
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "whole-repo-skill", results[0].Name)
	assert.False(t, results[0].Included, "root-level skill filtered by frontmatter name")
}

func TestPreviewSkillSource_InvalidConfig(t *testing.T) {
	p := newTestPreviewer(&previewFakeResolver{})

	// No repositories configured is a config error, surfaced before any clone.
	_, err := p.PreviewSkillSource(context.Background(), []byte("type: git-skills-plugin\nproperties: {}\n"))
	require.Error(t, err)
}

func TestPreviewSkillSource_RejectsYamlCatalogPath(t *testing.T) {
	p := newTestPreviewer(&previewFakeResolver{})

	// yamlCatalogPath is a server-side file listing repos; preview must read from
	// git (inline repository) instead, so this is rejected before any file access.
	cfg := `
type: git-skills-plugin
properties:
  yamlCatalogPath: /etc/skill-catalog/repos.yaml
`
	_, err := p.PreviewSkillSource(context.Background(), []byte(cfg))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git repository")
}

func TestPreviewSkillSource_RepoWithoutRefs(t *testing.T) {
	p := newTestPreviewer(&previewFakeResolver{})

	cfg := `
type: git-skills-plugin
properties:
  repositories:
    - url: https://example.com/a.git
`
	_, err := p.PreviewSkillSource(context.Background(), []byte(cfg))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no refs configured")
}

func TestPreviewSkillSource_AllReposFailIsError(t *testing.T) {
	fake := &previewFakeResolver{errs: map[string]error{
		"https://example.com/a.git@v1.0": errors.New("boom"),
	}}
	p := newTestPreviewer(fake)

	cfg := `
type: git-skills-plugin
properties:
  repositories:
    - url: https://example.com/a.git
      refs: ["v1.0"]
`
	_, err := p.PreviewSkillSource(context.Background(), []byte(cfg))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve any repository")
}

func TestPreviewSkillSource_RejectsMultipleRepositories(t *testing.T) {
	p := newTestPreviewer(&previewFakeResolver{})

	// Preview is scoped to a single repository; a config listing more than one is
	// a caller error, surfaced before any clone.
	cfg := `
type: git-skills-plugin
properties:
  repositories:
    - url: https://example.com/a.git
      refs: ["v1.0"]
    - url: https://example.com/b.git
      refs: ["v1.0"]
`
	_, err := p.PreviewSkillSource(context.Background(), []byte(cfg))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "single repository")
}

func TestPreviewSkillSource_PartialRefFailureReturnsResolved(t *testing.T) {
	// One repository, two refs: a ref that fails to resolve does not sink the refs
	// that succeed.
	fake := &previewFakeResolver{
		skills: map[string][]ResolvedSkill{
			"https://example.com/a.git@v1.0": {resolved("skills/deploy", "deploy")},
		},
		errs: map[string]error{
			"https://example.com/a.git@v2.0": errors.New("ref not found"),
		},
	}
	p := newTestPreviewer(fake)

	cfg := `
type: git-skills-plugin
properties:
  repositories:
    - url: https://example.com/a.git
      refs: ["v1.0", "v2.0"]
`
	results, err := p.PreviewSkillSource(context.Background(), []byte(cfg))
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "deploy", results[0].Name)
}

// TestPreviewSkillSource_RejectsDisallowedTransport exercises the real resolver to
// confirm the transport allowlist (which blocks file:// and ext:: — SSRF/command
// execution vectors) is enforced on the preview path, not just the sync path. The
// protocol is rejected before any clone, so no network or filesystem access occurs.
func TestPreviewSkillSource_RejectsDisallowedTransport(t *testing.T) {
	p := NewSkillPreviewer(ResolveLimits{}, SyncLimits{}, t.TempDir())

	cfg := `
type: git-skills-plugin
properties:
  repositories:
    - url: file:///tmp/whatever.git
      refs: ["v1.0"]
`
	_, err := p.PreviewSkillSource(context.Background(), []byte(cfg))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disallowed transport")
}

func TestPreviewSkillSource_TooManyJobs(t *testing.T) {
	p := newTestPreviewer(&previewFakeResolver{})
	// Raise the per-repo ref cap so buildRefJobs accepts the repo; the total-job
	// cap (maxPreviewJobs) is what should reject it.
	p.limits.MaxRefsPerRepo = maxPreviewJobs + 100

	refs := make([]string, 0, maxPreviewJobs+1)
	for i := 0; i <= maxPreviewJobs; i++ {
		refs = append(refs, fmt.Sprintf(`"v%d"`, i))
	}
	cfg := fmt.Sprintf(`
type: git-skills-plugin
properties:
  repositories:
    - url: https://example.com/a.git
      refs: [%s]
`, strings.Join(refs, ", "))

	_, err := p.PreviewSkillSource(context.Background(), []byte(cfg))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeding the limit")
}
