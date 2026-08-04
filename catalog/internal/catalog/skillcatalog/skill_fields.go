package skillcatalog

import (
	"github.com/kubeflow/hub/internal/platform/datastore"
)

// Datastore property names for skills. These align with the OpenAPI Skill field
// names (camelCase), except source_id which is the cross-plugin convention.
const (
	propSkillName       = "name"
	propDescription     = "description"
	propRepository      = "repository"
	propPath            = "path"
	propSkillVersion    = "version"
	propResolvedCommit  = "resolvedCommit"
	propSourceID        = "source_id"
	propTrustTier       = "trustTier"
	propProvider        = "provider"
	propCategory        = "category"
	propLicense         = "license"
	propAuthor          = "author"
	propCompatibility   = "compatibility"
	propReadme          = "readme"
	propLabels          = "labels"
	propAllowedTools    = "allowedTools"
	propSupportingFiles = "supportingFiles"
	propBodyLineCount   = "bodyLineCount"
)

// skillFieldKind is the datastore storage kind of a defined skill property.
type skillFieldKind int

const (
	// fieldString is a plain string property. fieldStringSlice is a []string
	// stored JSON-encoded in a string property; both register as datastore strings.
	fieldString skillFieldKind = iota
	fieldStringSlice
	fieldInt
)

// skillFields is the single source of truth for the skill entity's defined
// (non-custom) properties: their datastore names and storage kinds. The datastore
// spec (AddSkillProperties), the loader's writer (buildSkillEntity), and the DB→API
// reader (mapDBSkillToAPI) all key off these names, so a field is added in one place
// and TestSkillFieldsCoverage guards that the writer and reader both handle it.
var skillFields = []skillField{
	{propSourceID, fieldString},
	{propSkillName, fieldString},
	{propDescription, fieldString},
	{propRepository, fieldString},
	{propPath, fieldString},
	{propSkillVersion, fieldString},
	{propResolvedCommit, fieldString},
	{propTrustTier, fieldString},
	{propProvider, fieldString},
	{propCategory, fieldString},
	{propLicense, fieldString},
	{propAuthor, fieldString},
	{propCompatibility, fieldString},
	{propReadme, fieldString},
	{propLabels, fieldStringSlice},
	{propAllowedTools, fieldStringSlice},
	{propSupportingFiles, fieldStringSlice},
	{propBodyLineCount, fieldInt},
}

// skillField pairs a defined property's datastore name with its storage kind.
type skillField struct {
	name string
	kind skillFieldKind
}

// AddSkillProperties registers the skill entity's defined properties on the given
// datastore spec. Plugins call this instead of listing property names inline, so the
// datastore schema cannot silently drift from the names the loader and API mapper use.
func AddSkillProperties(spec *datastore.SpecType) *datastore.SpecType {
	for _, f := range skillFields {
		if f.kind == fieldInt {
			spec = spec.AddInt(f.name)
			continue
		}
		spec = spec.AddString(f.name)
	}
	return spec
}
