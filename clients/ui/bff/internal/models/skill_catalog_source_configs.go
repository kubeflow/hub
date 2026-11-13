package models

// SkillRepository represents an inline git repository entry in a git-skills-plugin source.
//
// Field names mirror the catalog service's SkillRepository
// (catalog/internal/catalog/skillcatalog/source_schema.go). That struct is decoded
// with DisallowUnknownFields, so any key written here that it does not declare
// fails the entire source at load time.
type SkillRepository struct {
	URL          string   `json:"url"`
	CanonicalURL *string  `json:"canonicalUrl,omitempty"`
	Refs         []string `json:"refs,omitempty"`
	ScanPaths    []string `json:"scanPaths,omitempty"`
	// CredentialRef names a key in the git-credentials Secret mounted into the
	// catalog pod — not a Secret name. It must be a plain filename. The BFF sets
	// this to the source id when a token is supplied; it is never entered by hand.
	CredentialRef *string `json:"credentialRef,omitempty"`
	// AuthToken is a write-only git token from the admin form. The BFF stores it in
	// the shared git-credentials Secret and clears it before serialisation, so it is
	// never written to the ConfigMap and never returned on reads.
	AuthToken *string `json:"authToken,omitempty"`
	// Labels are stamped onto every skill this repository yields, so they show on
	// skill cards and in filter_options (catalog service buildSkillEntity).
	//
	// Not to be confused with SkillCatalogSourceConfig.Labels: those sit on the
	// source itself and only drive the sourceLabel grouping of sources.
	Labels         []string `json:"labels,omitempty"`
	IncludedSkills []string `json:"includedSkills,omitempty"`
	ExcludedSkills []string `json:"excludedSkills,omitempty"`
	// SkillOverrides are not editable in the settings UI; they are preserved verbatim so
	// a UI save does not delete per-skill metadata a GitOps author set.
	SkillOverrides []SkillOverride `json:"skillOverrides,omitempty"`
}

// SkillOverride is a per-skill exception to a repository's custom metadata, matching
// the catalog service's SkillOverride (skillcatalog/source_schema.go). The repository's
// own category/labels apply to every skill it yields; an override replaces them for the
// one skill it names (buildSkillEntity precedence: repo entry -> skillOverride).
//
// The settings UI has no control for these — it edits repository-level metadata only —
// but they are carried through reads and writes so that a source authored by GitOps
// keeps them. The write path rebuilds properties.repositories from this model, so a
// field missing here is not merely uneditable, it is erased on the next save.
type SkillOverride struct {
	Name     string   `json:"name"`
	Category *string  `json:"category,omitempty"`
	Labels   []string `json:"labels,omitempty"`
}

// SkillCatalogSourceConfig is a catalog source entry for a git-skills-plugin source.
type SkillCatalogSourceConfig struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Enabled *bool  `json:"enabled,omitempty"`
	// Labels group the source itself for the sourceLabel selector. A nil slice on an
	// update means "leave as-is"; an empty (non-nil) slice means "clear them", which
	// is how the settings UI removes the last label. For per-skill labels see
	// SkillRepository.Labels.
	Labels    []string `json:"labels,omitempty"`
	Provider  *string  `json:"provider,omitempty"`
	Category  *string  `json:"category,omitempty"`
	TrustTier *string  `json:"trustTier,omitempty"`
	// Repositories holds exactly one entry for any source this API writes, matching the
	// catalog service's inline form. Provider/Category/TrustTier above are stamped onto
	// that entry on write and read back from it, so a second entry could carry neither.
	// It stays a slice because that is the shape the ConfigMap and the catalog service
	// use, and a read of a yamlCatalogPath-backed source can return several.
	Repositories []SkillRepository `json:"repositories,omitempty"`
	IsDefault    *bool             `json:"isDefault,omitempty"`
}

type SkillCatalogSourceConfigPayload = SkillCatalogSourceConfig

type SkillCatalogSourceConfigList struct {
	Catalogs []SkillCatalogSourceConfig `json:"catalogs,omitempty"`
}
