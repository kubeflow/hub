// Package skillcatalog implements the git-skills-plugin catalog source. This file
// covers parsing and validating skill source configuration; repository resolution,
// SKILL.md indexing, and the API service are added by later changes.
//
// Skill sources are configured as generic basecatalog.PluginSource entries under
// the SourceConfig.SkillCatalogs (`skill_catalogs`) section. Because PluginSource
// carries only a free-form `properties` map and configs are parsed with strict
// unmarshalling, all skill-specific configuration lives under `properties`:
//
//	skill_catalogs:
//	  - name: Community Skills
//	    id: community-skills
//	    type: git-skills-plugin
//	    enabled: true
//	    labels: [community]
//	    properties:
//	      trustTier: communityContributed       # optional provenance label
//	      syncIntervalMinutes: 60               # optional
//	      repositories:                         # inline form (UI-added sources)
//	        - url: https://github.com/example/skills.git
//	          refs: [main, v1.0]
//	          scanPaths: [skills/]
//	          authSecretName: git-creds
//	          provider: Example Org
//	          category: DevOps
//	          labels: [community]
//	          includedSkills: ["*"]
//	          excludedSkills: ["*-draft"]
//	          skillOverrides:
//	            - name: deploy
//	              category: SRE
//
// A shipped default instead points `properties.yamlCatalogPath` at a bundled file
// holding the same `repositories:` list. Both forms are equivalent downstream.
package skillcatalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	apimodels "github.com/kubeflow/hub/catalog/pkg/openapi"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// SourceTypeGitSkillsPlugin is the source `type` value for git-hosted skill catalogs.
const SourceTypeGitSkillsPlugin = "git-skills-plugin"

// Property keys understood under a skill source's `properties` map.
const (
	propYAMLCatalogPath = "yamlCatalogPath"
	propRepositories    = "repositories"
)

// SkillOverride carries per-skill custom-metadata overrides for a repository entry.
type SkillOverride struct {
	Name     string   `json:"name"`
	Category string   `json:"category,omitempty"`
	Labels   []string `json:"labels,omitempty"`
}

// SkillRepository is a single git repository listed by a skill source, together
// with its custom metadata and include/exclude filters.
type SkillRepository struct {
	// URL is the git repository to resolve. Required.
	URL string `json:"url"`
	// CanonicalURL is the upstream identity when URL points at a mirror.
	CanonicalURL string `json:"canonicalUrl,omitempty"`
	// Refs are the tags, releases, branches, or commit SHAs to index. When empty
	// the repository's default branch is used.
	Refs []string `json:"refs,omitempty"`
	// ScanPaths limits the SKILL.md scan to these subdirectories (default: whole repo).
	ScanPaths []string `json:"scanPaths,omitempty"`
	// AuthSecretName references a Secret providing credentials for private repos.
	AuthSecretName string `json:"authSecretName,omitempty"`
	// Provider, Category, Labels are custom metadata stamped onto the repo's skills.
	Provider string   `json:"provider,omitempty"`
	Category string   `json:"category,omitempty"`
	Labels   []string `json:"labels,omitempty"`
	// IncludedSkills / ExcludedSkills are wildcard filters over skill names.
	IncludedSkills []string `json:"includedSkills,omitempty"`
	ExcludedSkills []string `json:"excludedSkills,omitempty"`
	// SkillOverrides carry per-skill custom-metadata overrides.
	SkillOverrides []SkillOverride `json:"skillOverrides,omitempty"`
}

// SkillSourceSpec is the validated skill-specific configuration extracted from a
// PluginSource. Repositories are resolved to a single list regardless of whether
// they were provided inline or via a file.
type SkillSourceSpec struct {
	TrustTier           string
	SyncIntervalMinutes int
	Repositories        []SkillRepository
}

// skillSourceProperties is the strict schema of a skill source's `properties` map.
type skillSourceProperties struct {
	TrustTier           string            `json:"trustTier,omitempty"`
	SyncIntervalMinutes int               `json:"syncIntervalMinutes,omitempty"`
	YAMLCatalogPath     string            `json:"yamlCatalogPath,omitempty"`
	Repositories        []SkillRepository `json:"repositories,omitempty"`
}

// skillRepositoriesFile is the schema of the file referenced by yamlCatalogPath.
type skillRepositoriesFile struct {
	Repositories []SkillRepository `json:"repositories"`
}

// ParseSkillSource extracts and validates the skill-specific schema from a generic
// PluginSource. Repositories come either inline (properties.repositories) or from a
// file (properties.yamlCatalogPath); both forms yield an equivalent repository list.
func ParseSkillSource(source basecatalog.PluginSource) (*SkillSourceSpec, error) {
	if source.Type != SourceTypeGitSkillsPlugin {
		return nil, fmt.Errorf("skill source %q has unexpected type %q (want %q)", source.GetId(), source.Type, SourceTypeGitSkillsPlugin)
	}

	props, err := decodeProperties(source.Properties)
	if err != nil {
		return nil, fmt.Errorf("skill source %q: %w", source.GetId(), err)
	}

	if err := validateTrustTier(props.TrustTier); err != nil {
		return nil, fmt.Errorf("skill source %q: %w", source.GetId(), err)
	}
	if props.SyncIntervalMinutes < 0 {
		return nil, fmt.Errorf("skill source %q: syncIntervalMinutes must not be negative", source.GetId())
	}

	repos, err := resolveRepositories(source, props)
	if err != nil {
		return nil, fmt.Errorf("skill source %q: %w", source.GetId(), err)
	}
	if err := validateRepositories(repos); err != nil {
		return nil, fmt.Errorf("skill source %q: %w", source.GetId(), err)
	}

	return &SkillSourceSpec{
		TrustTier:           props.TrustTier,
		SyncIntervalMinutes: props.SyncIntervalMinutes,
		Repositories:        repos,
	}, nil
}

// decodeProperties strictly decodes the free-form properties map into the typed
// skill schema, rejecting unknown keys so config typos surface as errors.
func decodeProperties(props map[string]any) (*skillSourceProperties, error) {
	raw, err := json.Marshal(props)
	if err != nil {
		return nil, fmt.Errorf("invalid properties: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var out skillSourceProperties
	if err := dec.Decode(&out); err != nil {
		return nil, fmt.Errorf("invalid properties: %w", err)
	}
	return &out, nil
}

// resolveRepositories returns the repository list from exactly one of the inline
// or file forms.
func resolveRepositories(source basecatalog.PluginSource, props *skillSourceProperties) ([]SkillRepository, error) {
	hasInline := len(props.Repositories) > 0
	hasFile := props.YAMLCatalogPath != ""

	switch {
	case hasInline && hasFile:
		return nil, fmt.Errorf("specify either %q (inline) or %q (file), not both", propRepositories, propYAMLCatalogPath)
	case hasInline:
		return props.Repositories, nil
	case hasFile:
		return loadRepositoriesFile(source, props.YAMLCatalogPath)
	default:
		return nil, fmt.Errorf("no repositories configured: set %q inline or %q", propRepositories, propYAMLCatalogPath)
	}
}

// loadRepositoriesFile reads the repositories list from the file referenced by
// yamlCatalogPath, resolving relative paths against the source's config file.
func loadRepositoriesFile(source basecatalog.PluginSource, path string) ([]SkillRepository, error) {
	resolved := path
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(filepath.Dir(source.Origin), resolved)
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, fmt.Errorf("reading repositories file %q: %w", resolved, err)
	}
	var file skillRepositoriesFile
	if err := yaml.UnmarshalStrict(data, &file); err != nil {
		return nil, fmt.Errorf("parsing repositories file %q: %w", resolved, err)
	}
	return file.Repositories, nil
}

// validateTrustTier accepts an empty tier (no label) or one of the vendor-neutral
// SkillTrustTier enum values; anything else is rejected with a clear error.
func validateTrustTier(tier string) error {
	if tier == "" {
		return nil
	}
	if !apimodels.SkillTrustTier(tier).IsValid() {
		return fmt.Errorf("invalid trustTier %q: valid values are %v", tier, apimodels.AllowedSkillTrustTierEnumValues)
	}
	return nil
}

// validateRepositories checks the resolved repository list is non-empty, every
// entry has a URL, and URLs are unique within the source.
func validateRepositories(repos []SkillRepository) error {
	if len(repos) == 0 {
		return fmt.Errorf("at least one repository is required")
	}
	seen := make(map[string]bool, len(repos))
	for i, r := range repos {
		if r.URL == "" {
			return fmt.Errorf("repository[%d]: url is required", i)
		}
		if seen[r.URL] {
			return fmt.Errorf("duplicate repository url %q", r.URL)
		}
		seen[r.URL] = true
	}
	return nil
}
