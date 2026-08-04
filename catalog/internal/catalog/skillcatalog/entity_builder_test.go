package skillcatalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	skillmodels "github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog/models"
)

func propString(t *testing.T, e skillmodels.Skill, name string) string {
	t.Helper()
	if e.GetProperties() == nil {
		return ""
	}
	for _, p := range *e.GetProperties() {
		if p.Name == name && p.StringValue != nil {
			return *p.StringValue
		}
	}
	return ""
}

func propInt(t *testing.T, e skillmodels.Skill, name string) (int32, bool) {
	t.Helper()
	if e.GetProperties() == nil {
		return 0, false
	}
	for _, p := range *e.GetProperties() {
		if p.Name == name && p.IntValue != nil {
			return *p.IntValue, true
		}
	}
	return 0, false
}

func sampleResolved() ResolvedSkill {
	return ResolvedSkill{
		Repository:      "https://github.com/example/skills.git",
		Path:            "skills/deploy",
		Version:         "v1.0",
		ResolvedCommit:  "abc123",
		SupportingFiles: []string{"skills/deploy/run.sh"},
		Skill: &ParsedSkill{
			Name:          "deploy",
			Description:   "Deploy the app.",
			License:       "apache-2.0",
			Author:        "Jane Doe",
			Compatibility: "claude-code",
			AllowedTools:  []string{"Bash", "Read"},
			Metadata:      map[string]any{"team": "platform", "featured": true},
			Body:          "# Deploy\nbody",
			BodyLineCount: 2,
		},
	}
}

func TestBuildSkillEntity_FieldsAndIdentity(t *testing.T) {
	repo := SkillRepository{Provider: "Example Org", Category: "DevOps", Labels: []string{"community"}}
	spec := &SkillSourceSpec{TrustTier: "communityContributed"}

	e := buildSkillEntity(sampleResolved(), repo, spec, "community-skills")

	require.NotNil(t, e.GetAttributes())
	require.NotNil(t, e.GetAttributes().Name)
	assert.Equal(t, "community-skills|https://github.com/example/skills.git|skills/deploy|v1.0", *e.GetAttributes().Name,
		"identity keys on (sourceID, repository, path, version)")

	assert.Equal(t, "deploy", propString(t, e, propSkillName))
	assert.Equal(t, "https://github.com/example/skills.git", propString(t, e, propRepository))
	assert.Equal(t, "skills/deploy", propString(t, e, propPath))
	assert.Equal(t, "v1.0", propString(t, e, propSkillVersion))
	assert.Equal(t, "abc123", propString(t, e, propResolvedCommit))
	assert.Equal(t, "community-skills", propString(t, e, propSourceID))
	assert.Equal(t, "communityContributed", propString(t, e, propTrustTier))
	assert.Equal(t, "Example Org", propString(t, e, propProvider))
	assert.Equal(t, "DevOps", propString(t, e, propCategory))
	assert.Equal(t, "apache-2.0", propString(t, e, propLicense))
	assert.Equal(t, "Jane Doe", propString(t, e, propAuthor))
	assert.Equal(t, `["Bash","Read"]`, propString(t, e, propAllowedTools))
	assert.Equal(t, `["community"]`, propString(t, e, propLabels))
	assert.Equal(t, `["skills/deploy/run.sh"]`, propString(t, e, propSupportingFiles))

	n, ok := propInt(t, e, propBodyLineCount)
	assert.True(t, ok)
	assert.Equal(t, int32(2), n)
}

func TestBuildSkillEntity_SkillOverrideWins(t *testing.T) {
	repo := SkillRepository{
		Category: "DevOps",
		Labels:   []string{"community"},
		SkillOverrides: []SkillOverride{
			{Name: "deploy", Category: "SRE", Labels: []string{"ops", "prod"}},
		},
	}
	e := buildSkillEntity(sampleResolved(), repo, &SkillSourceSpec{}, "s")

	assert.Equal(t, "SRE", propString(t, e, propCategory), "override category wins over repo entry")
	assert.Equal(t, `["ops","prod"]`, propString(t, e, propLabels), "override labels win over repo entry")
}

func TestBuildSkillEntity_SameSkillFromTwoSourcesDoesNotCollide(t *testing.T) {
	// The database index is ephemeral (rebuilt from git at every sync), so
	// identity must never depend on a database-assigned ID. Two sources listing
	// the exact same (repository, path, version) must get distinct entity names,
	// since Context has a UNIQUE(type_id, name) constraint.
	resolved := sampleResolved()
	a := buildSkillEntity(resolved, SkillRepository{}, &SkillSourceSpec{}, "source-a")
	b := buildSkillEntity(resolved, SkillRepository{}, &SkillSourceSpec{}, "source-b")

	require.NotNil(t, a.GetAttributes().Name)
	require.NotNil(t, b.GetAttributes().Name)
	assert.NotEqual(t, *a.GetAttributes().Name, *b.GetAttributes().Name)
	assert.Equal(t, "source-a", propString(t, a, propSourceID))
	assert.Equal(t, "source-b", propString(t, b, propSourceID))
}

func TestBuildSkillEntity_CustomPropertiesFromMetadata(t *testing.T) {
	e := buildSkillEntity(sampleResolved(), SkillRepository{}, &SkillSourceSpec{}, "s")

	require.NotNil(t, e.GetCustomProperties())
	custom := map[string]any{}
	for _, p := range *e.GetCustomProperties() {
		assert.True(t, p.IsCustomProperty)
		switch {
		case p.StringValue != nil:
			custom[p.Name] = *p.StringValue
		case p.BoolValue != nil:
			custom[p.Name] = *p.BoolValue
		}
	}
	assert.Equal(t, "platform", custom["team"])
	assert.Equal(t, true, custom["featured"])
}
