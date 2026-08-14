package skillcatalog

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/sync/semaphore"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	model "github.com/kubeflow/hub/catalog/pkg/openapi"
)

// maxPreviewJobs caps how many ref clones a single preview request may trigger.
// Preview resolves one repository, but that repository can list several refs and
// each ref is a synchronous clone in the request path. MaxRefsPerRepo already
// bounds this in the normal case; maxPreviewJobs is a hard backstop should that
// limit be raised.
const maxPreviewJobs = 100

// SkillPreviewer previews a git-skills-plugin source by resolving its repositories
// at each ref and reporting which discovered skills the source's include/exclude
// filters would keep. It clones repositories (parse-only, discarded) exactly like
// the loader — reusing the same resolver, credential handling, and clone limits —
// so a preview behaves like the sync that would follow it.
type SkillPreviewer struct {
	resolver       refResolver
	credentialsDir string
	limits         SyncLimits
}

// NewSkillPreviewer builds a previewer with the loader's limits and credentials
// mount. Zero-valued limits fall back to the compiled-in defaults.
func NewSkillPreviewer(resolveLimits ResolveLimits, syncLimits SyncLimits, credentialsDir string) *SkillPreviewer {
	if syncLimits.MaxInFlightClones <= 0 {
		syncLimits.MaxInFlightClones = defaultMaxInFlightClones
	}
	if syncLimits.MaxRefsPerRepo <= 0 {
		syncLimits.MaxRefsPerRepo = defaultMaxRefsPerRepo
	}
	if syncLimits.MaxResolveWorkers <= 0 {
		syncLimits.MaxResolveWorkers = defaultMaxResolveWorkers
	}
	if credentialsDir == "" {
		credentialsDir = defaultGitCredentialsDir
	}
	return &SkillPreviewer{
		resolver:       NewRepoResolver(resolveLimits),
		credentialsDir: credentialsDir,
		limits:         syncLimits,
	}
}

// PreviewSkillSource previews a pasted git-skills-plugin source config. Scoped to
// one repository per request because each is a network clone; configs with more
// than one repository are rejected. Returns one result per discovered skill with
// Included reflecting the config's include/exclude filters.
func (p *SkillPreviewer) PreviewSkillSource(ctx context.Context, configBytes []byte) ([]model.AssetPreviewResult, error) {
	spec, err := parsePreviewSource(configBytes)
	if err != nil {
		return nil, err
	}
	switch len(spec.Repositories) {
	case 0:
		return nil, fmt.Errorf("no repositories configured; add exactly one repository under properties.repositories to preview")
	case 1:
		// expected
	default:
		return nil, fmt.Errorf("preview operates on a single repository at a time, but the configuration lists %d; remove all but one repository before previewing", len(spec.Repositories))
	}

	// Compute each repository's filter up front, then resolve with the filters
	// cleared so every skill in the repo is discovered and can be reported as
	// included or excluded. Resolving with the filters left in place would drop
	// excluded skills before they reach the preview, hiding exactly what the user
	// is trying to see.
	filtersByURL := make(map[string]*basecatalog.NameFilter, len(spec.Repositories))
	cleared := make([]SkillRepository, len(spec.Repositories))
	for i, r := range spec.Repositories {
		nf, ferr := basecatalog.NewNameFilter("includedSkills", r.IncludedSkills, "excludedSkills", r.ExcludedSkills)
		if ferr != nil {
			return nil, fmt.Errorf("repository %q: invalid include/exclude patterns: %w", r.URL, ferr)
		}
		filtersByURL[r.URL] = nf
		c := r
		c.IncludedSkills = nil
		c.ExcludedSkills = nil
		cleared[i] = c
	}

	jobs, jobErrs, _ := buildRefJobs(cleared, p.limits.MaxRefsPerRepo) // rejected URLs unused: preview has no index to clean up
	if len(jobErrs) > 0 {
		return nil, fmt.Errorf("invalid repository configuration: %s", strings.Join(jobErrs, "; "))
	}
	if len(jobs) == 0 {
		return []model.AssetPreviewResult{}, nil
	}
	if len(jobs) > maxPreviewJobs {
		return nil, fmt.Errorf("preview would resolve %d repository refs, exceeding the limit of %d; reduce the number of repositories or refs and try again", len(jobs), maxPreviewJobs)
	}

	sem := semaphore.NewWeighted(p.limits.MaxInFlightClones)
	creds := func(repo SkillRepository) (*Credentials, error) { return loadRepoCredentials(p.credentialsDir, repo) }
	// Preview always resolves fresh: there is no previously-indexed commit to
	// compare against, so nothing is skipped.
	noSkip := func(SkillRepository, string) string { return "" }
	results := resolveJobsConcurrently(ctx, jobs, p.limits.MaxResolveWorkers, sem, p.resolver, creds, noSkip)

	var previews []model.AssetPreviewResult
	var errs []string
	resolved := 0
	for _, res := range results {
		if res.err != nil {
			errs = append(errs, fmt.Sprintf("%s@%s: %v", res.repo.URL, res.ref, res.err))
			continue
		}
		resolved++
		filter := filtersByURL[res.repo.URL]
		for i := range res.skills {
			rs := res.skills[i]
			previews = append(previews, model.AssetPreviewResult{
				Name:     previewSkillName(rs),
				Included: filter.Allows(filterMatchName(rs)),
			})
		}
	}

	// A preview that resolved nothing while hitting errors is a failed preview (bad
	// URL, auth failure, or an unreachable host): surface it rather than returning
	// an empty list that reads as "no skills found". A partial failure (some refs
	// resolved) still returns what it found.
	if resolved == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("failed to resolve any repository: %s", strings.Join(errs, "; "))
	}
	return previews, nil
}

// parsePreviewSource decodes a single pasted skill source entry into a validated
// SkillSourceSpec. The assetType discriminator already selected this path, so an
// omitted `type` is tolerated and defaulted to the skill source type.
//
// Preview always resolves skills from a git repository, never from a catalog file:
// the file-based repository list (yamlCatalogPath) is rejected so a pasted config
// cannot point the server at a local file, and so preview cannot be mistaken for a
// static-catalog read like the model/MCP previews. The repository must be inline.
func parsePreviewSource(configBytes []byte) (*SkillSourceSpec, error) {
	var source basecatalog.PluginSource
	if err := yaml.Unmarshal(configBytes, &source); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	if source.Type == "" {
		source.Type = SourceTypeGitSkillsPlugin
	}
	if raw, ok := source.Properties[propYAMLCatalogPath]; ok {
		if s, _ := raw.(string); s != "" {
			return nil, fmt.Errorf("preview reads skills from a git repository: set the repository inline under %q, not %q (a server-side catalog file)", propRepositories, propYAMLCatalogPath)
		}
	}
	return ParseSkillSource(source)
}

// filterMatchName is the name a skill's include/exclude filter matches against. It
// defers to skillFilterName — the resolver's own rule — so preview and scan cannot
// disagree about which skills a filter would keep.
func filterMatchName(rs ResolvedSkill) string {
	frontmatter := ""
	if rs.Skill != nil {
		frontmatter = rs.Skill.Name
	}
	return skillFilterName(rs.Path, frontmatter)
}

// previewSkillName is the human-facing name shown for a previewed skill: its
// frontmatter name, falling back to the directory name.
func previewSkillName(rs ResolvedSkill) string {
	if rs.Skill != nil && rs.Skill.Name != "" {
		return rs.Skill.Name
	}
	return filterMatchName(rs)
}
