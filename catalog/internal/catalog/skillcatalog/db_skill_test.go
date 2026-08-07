package skillcatalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	skillmodels "github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog/models"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
)

func i32Ptr(n int32) *int32 { return new(n) }

func TestMapDBSkillToAPI_KnownFieldsAndCustomProperties(t *testing.T) {
	entity := &skillmodels.SkillImpl{
		ID: i32Ptr(7),
		Properties: &[]dbmodels.Properties{
			{Name: propSkillName, StringValue: new("deploy")},
			{Name: propRepository, StringValue: new("https://github.com/acme/skills.git")},
			{Name: propAuthor, StringValue: new("Jane Doe")},
			{Name: propSkillVersion, StringValue: new("v1.0")},
			{Name: propBodyLineCount, IntValue: i32Ptr(42)},
			{Name: propLabels, StringValue: new(`["a","b"]`)},
			// A non-custom property that is NOT a pre-configured field: must not be dropped.
			{Name: "futureField", StringValue: new("keep-me")},
		},
		CustomProperties: &[]dbmodels.Properties{
			{Name: "author", IsCustomProperty: true, StringValue: new("Jane")},
			{Name: "featured", IsCustomProperty: true, BoolValue: new(true)},
			{Name: "rank", IsCustomProperty: true, IntValue: i32Ptr(3)},
		},
	}

	res := mapDBSkillToAPI(entity)

	require.NotNil(t, res.Id)
	assert.Equal(t, "7", *res.Id)
	assert.Equal(t, "deploy", res.Name)
	require.NotNil(t, res.Repository)
	assert.Equal(t, "https://github.com/acme/skills.git", *res.Repository)
	require.NotNil(t, res.Author)
	assert.Equal(t, "Jane Doe", *res.Author)
	require.NotNil(t, res.BodyLineCount)
	assert.Equal(t, int32(42), *res.BodyLineCount)
	assert.Equal(t, []string{"a", "b"}, res.Labels)

	// Frontmatter metadata (custom) is surfaced.
	require.NotNil(t, res.CustomProperties)
	require.Contains(t, res.CustomProperties, "author")
	assert.Equal(t, "Jane", res.CustomProperties["author"].MetadataStringValue.StringValue)
	require.Contains(t, res.CustomProperties, "featured")
	assert.True(t, res.CustomProperties["featured"].MetadataBoolValue.BoolValue)
	require.Contains(t, res.CustomProperties, "rank")
	assert.Equal(t, "3", res.CustomProperties["rank"].MetadataIntValue.IntValue)

	// An unknown, non-custom property is not dropped — it becomes a custom property.
	require.Contains(t, res.CustomProperties, "futureField")
	assert.Equal(t, "keep-me", res.CustomProperties["futureField"].MetadataStringValue.StringValue)
}
