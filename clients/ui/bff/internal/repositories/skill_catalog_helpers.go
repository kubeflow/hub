package repositories

import (
	"fmt"
	"regexp"

	"github.com/kubeflow/hub/ui/bff/internal/models"
	"gopkg.in/yaml.v3"
)

const SkillCatalogTypeGit = "git-skills-plugin"

// validCredentialRef mirrors the catalog service's credentialRef check
// (catalog/internal/catalog/skillcatalog/source_schema.go): a plain filename with
// no path separators, so it cannot escape the mounted git-credentials directory.
var validCredentialRef = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9._-]*[A-Za-z0-9])?$`)

// validSkillTrustTiers is the closed set defined in catalog-v1.yaml / SkillTrustTier enum.
var validSkillTrustTiers = map[string]struct{}{
	"platformProvided":     {},
	"partnerVerified":      {},
	"organizationApproved": {},
	"communityContributed": {},
}

// validSkillCatalogIdRegex is deliberately more permissive than the shared
// validateCatalogId (model/mcp catalogs): it allows hyphens so a hand-written or
// GitOps-managed skill source (kebab-case is the common convention there) doesn't
// need to be renamed to satisfy validation. This must stay skill-catalog-specific —
// widening the shared validCatalogIdRegex instead would loosen ID validation for
// the model and MCP catalogs too.
var validSkillCatalogIdRegex = regexp.MustCompile(`^[a-z0-9_-]+$`)

func validateSkillCatalogId(id string) error {
	if id == "" {
		return ErrCatalogSourceIdRequired
	}

	if !validSkillCatalogIdRegex.MatchString(id) {
		return fmt.Errorf("invalid catalog ID: must contain only lowercase letters, numbers, hyphens, and underscores")
	}

	if len(id) > 238 {
		return fmt.Errorf("%w: '%s' (max 238 characters)", ErrCatalogIDTooLong, id)
	}

	return nil
}

func ParseSkillCatalogYaml(raw string, isDefault bool) ([]models.SkillCatalogSourceConfig, error) {
	var parsed struct {
		Catalogs []struct {
			Name       string         `yaml:"name"`
			Id         string         `yaml:"id"`
			Type       string         `yaml:"type"`
			Enabled    *bool          `yaml:"enabled"`
			Labels     []string       `yaml:"labels"`
			Properties map[string]any `yaml:"properties"`
		} `yaml:"skill_catalogs"`
	}

	if err := yaml.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse skill catalogs yaml: %w", err)
	}

	catalogs := make([]models.SkillCatalogSourceConfig, 0, len(parsed.Catalogs))
	for _, c := range parsed.Catalogs {
		props := c.Properties
		if props == nil {
			props = map[string]any{}
		}

		type rawSkillOverride struct {
			Name     string   `yaml:"name"`
			Category string   `yaml:"category"`
			Labels   []string `yaml:"labels"`
		}

		type rawRepo struct {
			URL            string   `yaml:"url"`
			CanonicalURL   string   `yaml:"canonicalUrl"`
			Refs           []string `yaml:"refs"`
			ScanPaths      []string `yaml:"scanPaths"`
			CredentialRef  string   `yaml:"credentialRef"`
			Labels         []string `yaml:"labels"`
			IncludedSkills []string `yaml:"includedSkills"`
			ExcludedSkills []string `yaml:"excludedSkills"`
			Provider       string   `yaml:"provider"`
			Category       string   `yaml:"category"`
			TrustTier      string   `yaml:"trustTier"`
			// Read so the write path can put them back; the UI never edits them.
			SkillOverrides []rawSkillOverride `yaml:"skillOverrides"`
		}

		var repos []models.SkillRepository
		var rawRepos []rawRepo
		if repoList, ok := props["repositories"]; ok {
			repoBytes, err := yaml.Marshal(repoList)
			if err != nil {
				return nil, fmt.Errorf("failed to re-marshal repositories for skill catalog source %q: %w", c.Id, err)
			}
			if err := yaml.Unmarshal(repoBytes, &rawRepos); err != nil {
				return nil, fmt.Errorf("failed to parse repositories for skill catalog source %q: %w", c.Id, err)
			}
			for _, r := range rawRepos {
				repo := models.SkillRepository{
					URL:            r.URL,
					Refs:           r.Refs,
					ScanPaths:      r.ScanPaths,
					Labels:         r.Labels,
					IncludedSkills: r.IncludedSkills,
					ExcludedSkills: r.ExcludedSkills,
				}
				if r.CanonicalURL != "" {
					repo.CanonicalURL = &r.CanonicalURL
				}
				if r.CredentialRef != "" {
					repo.CredentialRef = &r.CredentialRef
				}
				for _, ov := range r.SkillOverrides {
					entry := models.SkillOverride{Name: ov.Name, Labels: ov.Labels}
					if ov.Category != "" {
						category := ov.Category
						entry.Category = &category
					}
					repo.SkillOverrides = append(repo.SkillOverrides, entry)
				}
				repos = append(repos, repo)
			}
		}

		entry := models.SkillCatalogSourceConfig{
			Id:           c.Id,
			Name:         c.Name,
			Type:         c.Type,
			Enabled:      c.Enabled,
			Labels:       c.Labels,
			Repositories: repos,
			IsDefault:    &isDefault,
		}
		// Read category/provider/trustTier from the first repository (canonical location per
		// the backend schema). A source written through this API has exactly one repository,
		// so the first entry is the only entry; a hand-written or yamlCatalogPath-backed
		// source may list more, and only the first one's provenance is surfaced. Fall back to
		// properties level for backwards compat with manually-created ConfigMaps that placed
		// them at the source/properties level.
		if len(rawRepos) > 0 {
			if rawRepos[0].Provider != "" {
				entry.Provider = &rawRepos[0].Provider
			}
			if rawRepos[0].Category != "" {
				entry.Category = &rawRepos[0].Category
			}
			if rawRepos[0].TrustTier != "" {
				entry.TrustTier = &rawRepos[0].TrustTier
			}
		}
		if entry.Provider == nil {
			if v, ok := props["provider"].(string); ok && v != "" {
				entry.Provider = &v
			}
		}
		if entry.Category == nil {
			if v, ok := props["category"].(string); ok && v != "" {
				entry.Category = &v
			}
		}
		if entry.TrustTier == nil {
			if v, ok := props["trustTier"].(string); ok && v != "" {
				entry.TrustTier = &v
			}
		}
		catalogs = append(catalogs, entry)
	}

	return catalogs, nil
}

func FindSkillCatalogSourceById(sourceYAML string, catalogId string, isDefault bool) *models.SkillCatalogSourceConfig {
	if sourceYAML == "" {
		return nil
	}

	configs, err := ParseSkillCatalogYaml(sourceYAML, isDefault)
	if err != nil {
		return nil
	}

	for i := range configs {
		if configs[i].Id == catalogId {
			return &configs[i]
		}
	}

	return nil
}

func AppendSkillCatalogSourceToYaml(existingConfigMapEntry string, newEntry map[string]interface{}) (string, error) {
	var parsed struct {
		Catalogs []map[string]interface{} `yaml:"skill_catalogs"`
	}

	if existingConfigMapEntry != "" {
		if err := yaml.Unmarshal([]byte(existingConfigMapEntry), &parsed); err != nil {
			return "", fmt.Errorf("failed to parse existing skill sources.yaml: %w", err)
		}
	} else {
		parsed.Catalogs = []map[string]interface{}{}
	}
	parsed.Catalogs = append(parsed.Catalogs, newEntry)

	updatedBytes, err := yaml.Marshal(parsed)
	if err != nil {
		return "", fmt.Errorf("failed to marshal updated skill sources.yaml: %w", err)
	}

	return string(updatedBytes), nil
}

func RemoveSkillCatalogSourceFromYAML(existingYAML string, sourceId string) (string, error) {
	var parsed struct {
		Catalogs []map[string]interface{} `yaml:"skill_catalogs"`
	}

	if err := yaml.Unmarshal([]byte(existingYAML), &parsed); err != nil {
		return "", fmt.Errorf("failed to parse skill sources.yaml: %w", err)
	}

	filtered := make([]map[string]interface{}, 0)
	for _, src := range parsed.Catalogs {
		if id, ok := src["id"].(string); ok && id != sourceId {
			filtered = append(filtered, src)
		}
	}

	parsed.Catalogs = filtered
	updatedBytes, err := yaml.Marshal(parsed)
	if err != nil {
		return "", fmt.Errorf("failed to marshal updated skill sources.yaml: %w", err)
	}

	return string(updatedBytes), nil
}

func ConvertSkillSourceConfigToYamlEntry(payload models.SkillCatalogSourceConfigPayload) map[string]interface{} {
	entry := map[string]interface{}{
		"id":      payload.Id,
		"name":    payload.Name,
		"type":    payload.Type,
		"enabled": payload.Enabled,
	}

	if len(payload.Labels) > 0 {
		entry["labels"] = payload.Labels
	}

	props := map[string]interface{}{}
	if len(payload.Repositories) > 0 {
		props["repositories"] = buildSkillRepoYamlEntries(payload)
	}
	entry["properties"] = props

	return entry
}

// buildSkillRepoYamlEntries serialises payload.Repositories into the
// properties.repositories[] form the catalog service expects.
//
// The catalog service decodes a skill source's properties with
// DisallowUnknownFields (see catalog/internal/catalog/skillcatalog/source_schema.go),
// so an unrecognised key here is not ignored — it fails the whole source.
// Only keys present on the backend's SkillRepository struct may be emitted, and
// category/provider/trustTier belong on each repository rather than on properties.
//
// This is the single writer for repository entries; all update paths must go
// through it so the two representations cannot drift apart again.
//
// Entries are rebuilt from the payload rather than merged into what is stored, so
// provider/category/trustTier are replace-not-merge: an update that carries
// repositories but omits those three erases them from the stored entry. That is safe
// only because the settings form lifts all three into every save
// (SkillManageSourceForm's optionalFields). Nothing enforces that coupling, so a
// future change that moves them to be genuinely per-repository — where the catalog
// schema wants them — must keep sending them here, or start merging them from the
// existing entry, otherwise every edit silently drops a source's trust tier and its
// provider/category filter values.
func buildSkillRepoYamlEntries(payload models.SkillCatalogSourceConfigPayload) []map[string]interface{} {
	repos := make([]map[string]interface{}, 0, len(payload.Repositories))
	for _, r := range payload.Repositories {
		repo := map[string]interface{}{
			"url": r.URL,
		}
		if r.CanonicalURL != nil && *r.CanonicalURL != "" {
			repo["canonicalUrl"] = *r.CanonicalURL
		}
		if len(r.Refs) > 0 {
			repo["refs"] = r.Refs
		}
		if len(r.ScanPaths) > 0 {
			repo["scanPaths"] = r.ScanPaths
		}
		if r.CredentialRef != nil && *r.CredentialRef != "" {
			repo["credentialRef"] = *r.CredentialRef
		}
		if len(r.IncludedSkills) > 0 {
			repo["includedSkills"] = r.IncludedSkills
		}
		if len(r.ExcludedSkills) > 0 {
			repo["excludedSkills"] = r.ExcludedSkills
		}
		// Per-skill labels. Omitting them when empty is how they get cleared: an
		// update replaces the whole repositories list, so a key left out is gone.
		if len(r.Labels) > 0 {
			repo["labels"] = r.Labels
		}
		// Round-tripped, never authored here: the form has no per-skill control, so these
		// arrive only from a previous read of a GitOps-authored source. Omitting them
		// would delete them, since this rebuilds the entry from scratch.
		if len(r.SkillOverrides) > 0 {
			overrides := make([]map[string]interface{}, 0, len(r.SkillOverrides))
			for _, ov := range r.SkillOverrides {
				entry := map[string]interface{}{"name": ov.Name}
				if ov.Category != nil && *ov.Category != "" {
					entry["category"] = *ov.Category
				}
				if len(ov.Labels) > 0 {
					entry["labels"] = ov.Labels
				}
				overrides = append(overrides, entry)
			}
			repo["skillOverrides"] = overrides
		}
		if payload.Provider != nil && *payload.Provider != "" {
			repo["provider"] = *payload.Provider
		}
		if payload.Category != nil && *payload.Category != "" {
			repo["category"] = *payload.Category
		}
		if payload.TrustTier != nil && *payload.TrustTier != "" {
			repo["trustTier"] = *payload.TrustTier
		}
		repos = append(repos, repo)
	}
	return repos
}

func UpdateSkillCatalogSourceInYAML(
	existingYAML string,
	catalogId string,
	payload models.SkillCatalogSourceConfigPayload,
) (string, error) {
	var parsed struct {
		Catalogs []map[string]interface{} `yaml:"skill_catalogs"`
	}

	if existingYAML == "" {
		return "", fmt.Errorf("no existing yaml to update")
	}

	if err := yaml.Unmarshal([]byte(existingYAML), &parsed); err != nil {
		return "", fmt.Errorf("failed to parse skill sources.yaml: %w", err)
	}

	found := false
	for i, src := range parsed.Catalogs {
		if id, ok := src["id"].(string); ok && id == catalogId {
			found = true

			if payload.Name != "" {
				src["name"] = payload.Name
			}
			// nil means the caller did not send labels at all (e.g. a toggle-only
			// update), so leave them alone. An empty but non-nil slice is an explicit
			// "remove them all" — writing that back is what lets the settings UI clear
			// the last label instead of silently keeping it.
			if payload.Labels != nil {
				if len(payload.Labels) > 0 {
					src["labels"] = payload.Labels
				} else {
					delete(src, "labels")
				}
			}
			if payload.Enabled != nil {
				src["enabled"] = *payload.Enabled
			}
			existingProps, _ := src["properties"].(map[string]interface{})
			if existingProps == nil {
				existingProps = map[string]interface{}{}
			}
			if len(payload.Repositories) > 0 {
				// Drop stale source-level metadata written by earlier BFF versions.
				// Only done when replacing repos, so a toggle-only update does not
				// erase fields it was never given a replacement for.
				delete(existingProps, "provider")
				delete(existingProps, "category")
				delete(existingProps, "trustTier")
				existingProps["repositories"] = buildSkillRepoYamlEntries(payload)
			}
			// Only write properties back when there is something in them. An empty map
			// serialises as `properties: {}`, which is not the same as omitting the key:
			// MergeCommonSourceFields replaces the base source's Properties whenever the
			// override's is non-nil, so a default source overridden with `{}` loses its
			// repositories and then fails to load ("no repositories configured"). This is
			// reachable by toggling a shipped default off and back on, since the first
			// toggle writes an override entry that carries no properties at all.
			if len(existingProps) > 0 {
				src["properties"] = existingProps
			}

			parsed.Catalogs[i] = src
			break
		}
	}

	if !found {
		return "", fmt.Errorf("skill catalog '%s' not found in yaml", catalogId)
	}

	updatedBytes, err := yaml.Marshal(parsed)
	if err != nil {
		return "", fmt.Errorf("failed to marshal updated skill sources.yaml: %w", err)
	}

	return string(updatedBytes), nil
}

func mergeSkillCatalogSourceConfigs(defaultCatalog, userCatalog models.SkillCatalogSourceConfig) models.SkillCatalogSourceConfig {
	merged := defaultCatalog

	if userCatalog.Name != "" {
		merged.Name = userCatalog.Name
	}
	if userCatalog.Type != "" {
		merged.Type = userCatalog.Type
	}
	if userCatalog.Enabled != nil {
		merged.Enabled = userCatalog.Enabled
	}
	// Same nil-vs-empty rule as the update path: a user override that clears the
	// labels of a default source must win, not fall back to the default's labels.
	if userCatalog.Labels != nil {
		merged.Labels = userCatalog.Labels
	}
	if userCatalog.Provider != nil {
		merged.Provider = userCatalog.Provider
	}
	if userCatalog.Category != nil {
		merged.Category = userCatalog.Category
	}
	if userCatalog.TrustTier != nil {
		merged.TrustTier = userCatalog.TrustTier
	}
	if len(userCatalog.Repositories) > 0 {
		merged.Repositories = userCatalog.Repositories
	}

	return merged
}

func validateSkillCatalogSourceConfigPayload(payload models.SkillCatalogSourceConfigPayload) error {
	if payload.Id == "" {
		return fmt.Errorf("%w", ErrSkillCatalogSourceIdRequired)
	}

	if err := validateSkillCatalogId(payload.Id); err != nil {
		return fmt.Errorf("%w: %v", ErrSkillCatalogValidationFailed, err)
	}

	if payload.Name == "" {
		return fmt.Errorf("%w: name is required", ErrSkillCatalogValidationFailed)
	}

	if payload.Type == "" {
		return fmt.Errorf("%w: type is required", ErrSkillCatalogValidationFailed)
	}

	if payload.Type != SkillCatalogTypeGit {
		return fmt.Errorf("%w: unsupported skill catalog type: %s (supported: %s)", ErrSkillCatalogValidationFailed, payload.Type, SkillCatalogTypeGit)
	}

	if len(payload.Repositories) == 0 {
		return fmt.Errorf("%w: at least one repository is required for git-skills-plugin sources", ErrSkillCatalogValidationFailed)
	}

	if err := validateSkillRepositories(payload.Repositories); err != nil {
		return err
	}

	if err := validateSourceMetadataFields(payload.Category, payload.Provider, payload.TrustTier); err != nil {
		return fmt.Errorf("%w: %v", ErrSkillCatalogValidationFailed, err)
	}

	return nil
}

// validateSkillUpdatePayloadForDefaultOverride mirrors the model and MCP catalogs
// (validateMcpUpdatePayloadForDefaultOverride): a shipped default source is read-only
// apart from being enabled or disabled. Allowing repositories through here would let an
// admin repoint a platform-provided source at an arbitrary git repo while its skills keep
// the source's trust tier.
func validateSkillUpdatePayloadForDefaultOverride(payload models.SkillCatalogSourceConfigPayload) error {
	if payload.Name != "" {
		return fmt.Errorf("%w: cannot change 'name'", ErrSkillCatalogCannotChangeDefault)
	}

	if payload.Labels != nil {
		return fmt.Errorf("%w: cannot change 'labels'", ErrSkillCatalogCannotChangeDefault)
	}

	if len(payload.Repositories) > 0 {
		return fmt.Errorf("%w: cannot change 'repositories'", ErrSkillCatalogCannotChangeDefault)
	}

	if payload.Provider != nil || payload.Category != nil || payload.TrustTier != nil {
		return fmt.Errorf("%w: cannot change source metadata", ErrSkillCatalogCannotChangeDefault)
	}

	return nil
}

func validateSkillCatalogUpdatePayload(payload models.SkillCatalogSourceConfigPayload) error {
	if err := validateSourceMetadataFields(payload.Category, payload.Provider, payload.TrustTier); err != nil {
		return fmt.Errorf("%w: %v", ErrSkillCatalogValidationFailed, err)
	}
	if err := validateSkillRepositories(payload.Repositories); err != nil {
		return err
	}
	return nil
}

// validateSkillRepositories applies the per-repository checks the catalog service
// performs at load time, so a bad value is reported to the admin here rather than
// silently breaking the source on the next sync.
//
// A source written through this API carries exactly one repository, matching the
// catalog service's inline form (resolveRepositories rejects a longer inline list).
// The write path assumes it: provider/category/trustTier arrive at the source level
// and are stamped onto every repository entry, and reads recover them from the first
// entry only — so a second repository could neither carry its own provenance nor be
// read back. Multi-repository sources are configured through yamlCatalogPath, which
// this API does not write.
func validateSkillRepositories(repos []models.SkillRepository) error {
	if len(repos) > 1 {
		return fmt.Errorf("%w: a source accepts a single repository (got %d); configure several through yamlCatalogPath, or register one source per repository",
			ErrSkillCatalogValidationFailed, len(repos))
	}
	for i, repo := range repos {
		if repo.URL == "" {
			return fmt.Errorf("%w: repository[%d].url is required", ErrSkillCatalogValidationFailed, i)
		}
		if repo.CredentialRef != nil && *repo.CredentialRef != "" && !validCredentialRef.MatchString(*repo.CredentialRef) {
			return fmt.Errorf("%w: repository[%d].credentialRef %q must be a plain filename (letters, digits, '.', '_', '-')",
				ErrSkillCatalogValidationFailed, i, *repo.CredentialRef)
		}
	}
	return nil
}

// validateSourceMetadataFields validates category, provider, and trustTier together.
// All three fields are optional; when provided they must be non-empty, and trustTier
// must be one of the values defined in the SkillTrustTier enum.
func validateSourceMetadataFields(category, provider, trustTier *string) error {
	if category != nil && *category == "" {
		return fmt.Errorf("category must not be an empty string when provided")
	}
	if provider != nil && *provider == "" {
		return fmt.Errorf("provider must not be an empty string when provided")
	}
	return validateTrustTier(trustTier)
}

func validateTrustTier(trustTier *string) error {
	if trustTier == nil || *trustTier == "" {
		return nil
	}
	if _, ok := validSkillTrustTiers[*trustTier]; !ok {
		valid := make([]string, 0, len(validSkillTrustTiers))
		for k := range validSkillTrustTiers {
			valid = append(valid, k)
		}
		return fmt.Errorf("invalid trustTier %q: must be one of %v", *trustTier, valid)
	}
	return nil
}
