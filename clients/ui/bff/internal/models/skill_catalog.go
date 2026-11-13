package models

import "github.com/kubeflow/hub/pkg/openapi"

type Skill struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	SourceID        *string  `json:"sourceId,omitempty"`
	Description     *string  `json:"description,omitempty"`
	Readme          *string  `json:"readme,omitempty"`
	License         *string  `json:"license,omitempty"`
	Author          *string  `json:"author,omitempty"`
	Compatibility   *string  `json:"compatibility,omitempty"`
	AllowedTools    []string `json:"allowedTools,omitempty"`
	Labels          []string `json:"labels,omitempty"`
	Provider        *string  `json:"provider,omitempty"`
	Category        *string  `json:"category,omitempty"`
	TrustTier       *string  `json:"trustTier,omitempty"`
	Repository      *string  `json:"repository,omitempty"`
	Path            *string  `json:"path,omitempty"`
	Version         *string  `json:"version,omitempty"`
	ResolvedCommit  *string  `json:"resolvedCommit,omitempty"`
	SupportingFiles []string `json:"supportingFiles,omitempty"`
	// BodyLineCount is the SKILL.md body length. The catalog service warns above its
	// maxSkillBodyLines (500) because an agent loads the whole body into its context,
	// so the UI surfaces it as a quality signal. Display only — the catalog service
	// deliberately excludes it from filter_options.
	BodyLineCount    *int32                            `json:"bodyLineCount,omitempty"`
	CustomProperties *map[string]openapi.MetadataValue `json:"customProperties,omitempty"`
}

type SkillList struct {
	NextPageToken string  `json:"nextPageToken"`
	PageSize      int32   `json:"pageSize"`
	Size          int32   `json:"size"`
	Items         []Skill `json:"items"`
}

// SkillMarketplace and its nested types are a subset pass-through of the Claude
// Code plugin marketplace document served by the catalog service's
// claude/marketplace.json endpoint (see catalog/internal/catalog/skillcatalog/marketplace.go).
// Only the fields the UI needs to build install commands are kept.
type SkillMarketplace struct {
	Name    string                   `json:"name"`
	Plugins []SkillMarketplacePlugin `json:"plugins"`
}

type SkillMarketplacePlugin struct {
	Name   string                       `json:"name"`
	Source SkillMarketplacePluginSource `json:"source"`
}

type SkillMarketplacePluginSource struct {
	URL  string `json:"url"`
	Path string `json:"path"`
	Ref  string `json:"ref,omitempty"`
	SHA  string `json:"sha,omitempty"`
}
