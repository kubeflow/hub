package skillcatalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	skillmodels "github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog/models"
	"github.com/kubeflow/hub/internal/platform/datastore"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
)

// TestSkillFieldsCoverage guards the shared skillFields table against the failure
// mode the writer/reader/datastore-spec split invites: a field registered in one
// place but silently forgotten in another. It asserts (a) the loader's writer emits
// a property for every field, and (b) the DB→API reader consumes every field into a
// typed slot rather than letting it fall through to customProperties.
func TestSkillFieldsCoverage(t *testing.T) {
	// (a) writer: a fully-populated skill must emit every defined field.
	repo := SkillRepository{Provider: "Example Org", Category: "DevOps", Labels: []string{"community"}}
	spec := &SkillSourceSpec{TrustTier: "communityContributed"}
	entity := buildSkillEntity(sampleResolved(), repo, spec, "community-skills")

	written := map[string]bool{}
	require.NotNil(t, entity.GetProperties())
	for _, p := range *entity.GetProperties() {
		written[p.Name] = true
	}
	for _, f := range skillFields {
		assert.Truef(t, written[f.name], "field %q is in skillFields but buildSkillEntity never writes it", f.name)
	}

	// (b) reader: each defined field must map into a typed API slot, never leak into
	// customProperties.
	for _, f := range skillFields {
		prop := dbmodels.Properties{Name: f.name}
		switch f.kind {
		case fieldInt:
			prop.IntValue = i32Ptr(1)
		case fieldStringSlice:
			prop.StringValue = strPtr(`["x"]`)
		default:
			prop.StringValue = strPtr("v")
		}
		res := mapDBSkillToAPI(&skillmodels.SkillImpl{Properties: &[]dbmodels.Properties{prop}})
		assert.NotContainsf(t, res.CustomProperties, f.name,
			"field %q is in skillFields but mapDBSkillToAPI leaks it into customProperties", f.name)
	}
}

func TestAddSkillProperties_RegistersEveryField(t *testing.T) {
	spec := AddSkillProperties(datastore.NewSpecType(nil))
	require.Len(t, spec.Properties, len(skillFields), "every skillFields entry is registered on the datastore spec")
	for _, f := range skillFields {
		_, ok := spec.Properties[f.name]
		assert.Truef(t, ok, "field %q is not registered on the datastore spec", f.name)
	}
}
