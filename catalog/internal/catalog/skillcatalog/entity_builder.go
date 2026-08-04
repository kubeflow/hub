package skillcatalog

import (
	"encoding/json"

	"github.com/golang/glog"

	skillmodels "github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog/models"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
)

// skillEntityName is the datastore's unique key for a skill entry (Context.Name,
// which carries a UNIQUE(type_id, name) constraint). It is namespaced by
// sourceID — never by a database-assigned ID, since the index is ephemeral and
// rebuilt from git at every sync — so the same (repository, path, version)
// listed by two sources is indexed once per source rather than colliding.
// Multi-version support keys on (repository, path, version) within a source, so
// distinct refs of the same skill are distinct entries.
func skillEntityName(sourceID, repository, path, version string) string {
	return sourceID + "|" + repository + "|" + path + "|" + version
}

// buildSkillEntity converts a resolved skill into a datastore entity, stamping
// custom metadata from the source configuration. Precedence for category/labels
// is repo entry → skillOverride (per-skill); provider and trustTier come from the
// repo entry and source respectively; the SKILL.md frontmatter metadata becomes
// customProperties.
func buildSkillEntity(resolved ResolvedSkill, repo SkillRepository, spec *SkillSourceSpec, sourceID string) skillmodels.Skill {
	parsed := resolved.Skill

	category := repo.Category
	labels := repo.Labels
	if ov := findSkillOverride(repo.SkillOverrides, parsed.Name); ov != nil {
		if ov.Category != "" {
			category = ov.Category
		}
		if len(ov.Labels) > 0 {
			labels = ov.Labels
		}
	}

	props := []dbmodels.Properties{}
	addString(&props, propSkillName, parsed.Name)
	addString(&props, propDescription, parsed.Description)
	addString(&props, propRepository, resolved.Repository)
	addString(&props, propPath, resolved.Path)
	addString(&props, propSkillVersion, resolved.Version)
	addString(&props, propResolvedCommit, resolved.ResolvedCommit)
	addString(&props, propSourceID, sourceID)
	if spec != nil {
		addString(&props, propTrustTier, spec.TrustTier)
	}
	addString(&props, propProvider, repo.Provider)
	addString(&props, propCategory, category)
	addString(&props, propLicense, parsed.License)
	addString(&props, propAuthor, parsed.Author)
	addString(&props, propCompatibility, parsed.Compatibility)
	addString(&props, propReadme, parsed.Body)
	addStringSlice(&props, propLabels, labels)
	addStringSlice(&props, propAllowedTools, parsed.AllowedTools)
	addStringSlice(&props, propSupportingFiles, resolved.SupportingFiles)
	addInt(&props, propBodyLineCount, int32(parsed.BodyLineCount))

	custom := customPropertiesFromMetadata(parsed.Metadata)

	name := skillEntityName(sourceID, resolved.Repository, resolved.Path, resolved.Version)
	return &skillmodels.SkillImpl{
		Attributes: &skillmodels.SkillAttributes{
			Name:       &name,
			ExternalID: nil,
		},
		Properties:       &props,
		CustomProperties: &custom,
	}
}

func findSkillOverride(overrides []SkillOverride, name string) *SkillOverride {
	for i := range overrides {
		if overrides[i].Name == name {
			return &overrides[i]
		}
	}
	return nil
}

func addString(props *[]dbmodels.Properties, name, value string) {
	if value == "" {
		return
	}
	v := value
	*props = append(*props, dbmodels.Properties{Name: name, StringValue: &v})
}

// addInt always persists the value, including zero. Zero is meaningful for
// bodyLineCount (an empty body has zero lines), unlike addString which skips "".
func addInt(props *[]dbmodels.Properties, name string, value int32) {
	v := value
	*props = append(*props, dbmodels.Properties{Name: name, IntValue: &v})
}

// addStringSlice stores a []string as a JSON array in a string property, matching
// how the DB→API mapping decodes array fields.
func addStringSlice(props *[]dbmodels.Properties, name string, values []string) {
	if len(values) == 0 {
		return
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		// Unreachable for []string; skip the field rather than fail the whole skill.
		glog.Errorf("skill: marshaling property %q failed, dropping value: %v", name, err)
		return
	}
	s := string(encoded)
	*props = append(*props, dbmodels.Properties{Name: name, StringValue: &s})
}

// customPropertiesFromMetadata converts SKILL.md frontmatter metadata into custom
// properties. String values map to string properties; anything else is JSON-encoded.
func customPropertiesFromMetadata(metadata map[string]any) []dbmodels.Properties {
	custom := []dbmodels.Properties{}
	for k, v := range metadata {
		prop := dbmodels.Properties{Name: k, IsCustomProperty: true}
		switch t := v.(type) {
		case string:
			s := t
			prop.StringValue = &s
		case bool:
			b := t
			prop.BoolValue = &b
		case float64:
			d := t
			prop.DoubleValue = &d
		default:
			if encoded, err := json.Marshal(v); err == nil {
				s := string(encoded)
				prop.StringValue = &s
			} else {
				glog.Errorf("skill customProperty %q: marshaling failed, dropping value: %v", k, err)
			}
		}
		custom = append(custom, prop)
	}
	return custom
}
