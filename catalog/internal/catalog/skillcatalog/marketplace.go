package skillcatalog

import (
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	model "github.com/kubeflow/hub/catalog/pkg/openapi"
)

// This file renders indexed skills as a Claude Code plugin marketplace document,
// per https://code.claude.com/docs/en/plugin-marketplaces. Every indexed ref of a
// skill becomes its own plugin entry whose source is a git-subdir pinned to the
// exact resolved commit, so an install is reproducible.

// gitSubdirSourceType is the Claude Code plugin source discriminator for a
// subdirectory within a git repository.
const gitSubdirSourceType = "git-subdir"

// Marketplace is the top-level Claude Code marketplace document.
type Marketplace struct {
	Name     string               `json:"name"`
	Owner    MarketplaceOwner     `json:"owner"`
	Metadata *MarketplaceMetadata `json:"metadata,omitempty"`
	Plugins  []MarketplacePlugin  `json:"plugins"`
}

// MarketplaceOwner identifies the marketplace maintainer.
type MarketplaceOwner struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

// MarketplaceMetadata carries optional descriptive fields.
type MarketplaceMetadata struct {
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
}

// MarketplacePlugin is one installable entry (one skill at one ref).
type MarketplacePlugin struct {
	Name        string             `json:"name"`
	Source      GitSubdirSource    `json:"source"`
	Description string             `json:"description,omitempty"`
	Version     string             `json:"version,omitempty"`
	Author      *MarketplaceAuthor `json:"author,omitempty"`
	License     string             `json:"license,omitempty"`
	Keywords    []string           `json:"keywords,omitempty"`
	Category    string             `json:"category,omitempty"`
}

// MarketplaceAuthor is the plugin author block (name required by the schema).
type MarketplaceAuthor struct {
	Name string `json:"name"`
}

// GitSubdirSource pins a skill to an exact commit within a git-repository
// subdirectory. `sha` is the reproducible pin; `ref` records the human-readable
// tag/commit the entry was indexed at.
type GitSubdirSource struct {
	Source string `json:"source"`
	URL    string `json:"url"`
	Path   string `json:"path"`
	Ref    string `json:"ref,omitempty"`
	SHA    string `json:"sha,omitempty"`
}

// MarketplaceConfig holds the deployment-level marketplace settings resolved from
// the environment (see MarketplaceConfigFromEnv).
type MarketplaceConfig struct {
	Name           string
	Owner          string
	ExternalMirror string
	InternalMirror string
}

// MarketplaceOptions are the per-request rendering inputs.
type MarketplaceOptions struct {
	Name  string
	Owner MarketplaceOwner
	// URLRewrite, when non-nil, maps a skill's canonical repository URL to the URL
	// clients should clone from (e.g. an internal or external mirror base).
	URLRewrite func(repoURL string) string
}

// Options resolves the rendering options for one request. Clone URLs default to
// the external mirror when configured; a request with audience "cluster" uses the
// internal mirror instead, so in-cluster consumers pull from the internal Service
// while laptops pull from the external Route. With no mirror configured, canonical
// upstream URLs are served unchanged.
func (c MarketplaceConfig) Options(audience string) MarketplaceOptions {
	base := c.ExternalMirror
	if audience == "cluster" && c.InternalMirror != "" {
		base = c.InternalMirror
	}
	var rewrite func(string) string
	if base != "" {
		rewrite = mirrorRewriter(base)
	}
	return MarketplaceOptions{
		Name:       c.Name,
		Owner:      MarketplaceOwner{Name: c.Owner},
		URLRewrite: rewrite,
	}
}

// mirrorRewriter maps a canonical repo URL onto a mirror base by taking its
// {org}/{repo} path and joining it to the base as {base}/{org}/{repo}.git. A URL
// that cannot be parsed is returned unchanged so a single odd entry does not break
// the whole document.
func mirrorRewriter(base string) func(string) string {
	base = strings.TrimRight(base, "/")
	return func(repoURL string) string {
		u, err := url.Parse(repoURL)
		if err != nil || u.Path == "" {
			return repoURL
		}
		p := strings.Trim(u.Path, "/")
		if p == "" {
			return repoURL
		}
		if !strings.HasSuffix(p, ".git") {
			p += ".git"
		}
		return base + "/" + p
	}
}

// BuildMarketplace renders the given skills into a Claude Code marketplace
// document. Skills are ordered deterministically by (repository, path, version) so
// the document — and the disambiguated plugin names it assigns — are stable across
// renders of identical content. A skill missing the identity needed to install it
// (repository or name) is skipped.
func BuildMarketplace(skills []model.Skill, opts MarketplaceOptions) Marketplace {
	sorted := append([]model.Skill(nil), skills...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if a, b := deref(sorted[i].Repository), deref(sorted[j].Repository); a != b {
			return a < b
		}
		if a, b := deref(sorted[i].Path), deref(sorted[j].Path); a != b {
			return a < b
		}
		return deref(sorted[i].Version) < deref(sorted[j].Version)
	})

	used := make(map[string]int, len(sorted))
	plugins := make([]MarketplacePlugin, 0, len(sorted))
	for i := range sorted {
		s := sorted[i]
		repo := deref(s.Repository)
		if repo == "" || s.Name == "" {
			continue
		}
		cloneURL := repo
		if opts.URLRewrite != nil {
			cloneURL = opts.URLRewrite(repo)
		}
		version := deref(s.Version)
		commit := deref(s.ResolvedCommit)

		plugins = append(plugins, MarketplacePlugin{
			Name: uniquePluginName(kebabName(s.Name), version, commit, used),
			Source: GitSubdirSource{
				Source: gitSubdirSourceType,
				URL:    cloneURL,
				Path:   marketplacePath(deref(s.Path)),
				Ref:    version,
				SHA:    commit,
			},
			Description: deref(s.Description),
			Version:     marketplaceVersion(version, commit),
			Author:      marketplaceAuthor(s),
			License:     deref(s.License),
			Keywords:    s.Labels,
			Category:    deref(s.Category),
		})
	}

	return Marketplace{
		Name:    opts.Name,
		Owner:   opts.Owner,
		Plugins: plugins,
	}
}

// nonKebab matches any run of characters that are not lowercase alphanumerics, used
// to normalize a skill name into the kebab-case a plugin name requires.
var nonKebab = regexp.MustCompile(`[^a-z0-9]+`)

// kebabName normalizes a skill name to the kebab-case a Claude Code plugin name
// requires (lowercase alphanumerics and single hyphens). Agent Skills names are
// already kebab-case; this guards against anything that is not.
func kebabName(name string) string {
	n := nonKebab.ReplaceAllString(strings.ToLower(name), "-")
	n = strings.Trim(n, "-")
	if n == "" {
		return "skill"
	}
	return n
}

// uniquePluginName returns a marketplace-unique plugin name for a skill. Plugin
// names must be unique within a marketplace, but a skill can be indexed at several
// refs and different repositories can define skills of the same name; base collides
// on those. It prefers the bare name, then appends the version, then a short commit
// prefix, and finally a numeric suffix — deterministic because callers render in a
// stable order.
func uniquePluginName(base, version, commit string, used map[string]int) string {
	candidate := base
	if _, taken := used[candidate]; taken && version != "" {
		candidate = base + "-" + kebabName(version)
	}
	if _, taken := used[candidate]; taken && commit != "" {
		short := commit
		if len(short) > 12 {
			short = short[:12]
		}
		candidate = base + "-" + kebabName(short)
	}
	for {
		if n, taken := used[candidate]; taken {
			used[candidate] = n + 1
			candidate = base + "-" + strconv.Itoa(n+1)
			continue
		}
		used[candidate] = 1
		return candidate
	}
}

// marketplacePath returns the git-subdir path for a skill directory. A repo-root
// skill (path ".") maps to "." so the whole repository is checked out.
func marketplacePath(p string) string {
	if p == "" {
		return "."
	}
	return p
}

// marketplaceVersion is the plugin's display/update version: the ref it was indexed
// at, falling back to the resolved commit. The reproducible pin is always the
// source's sha; this field only drives Claude Code's update tracking and display.
func marketplaceVersion(version, commit string) string {
	if version != "" {
		return version
	}
	return commit
}

// marketplaceAuthor derives the plugin author from the skill's author, then its
// provider; nil when neither is set (the field is optional).
func marketplaceAuthor(s model.Skill) *MarketplaceAuthor {
	if a := deref(s.Author); a != "" {
		return &MarketplaceAuthor{Name: a}
	}
	if p := deref(s.Provider); p != "" {
		return &MarketplaceAuthor{Name: p}
	}
	return nil
}
