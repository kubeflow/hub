package skillcatalog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
)

// mustProps parses a YAML snippet into a properties map the way source loading
// produces it (YAML -> JSON-compatible map[string]any).
func mustProps(t *testing.T, y string) map[string]any {
	t.Helper()
	props := map[string]any{}
	require.NoError(t, yaml.Unmarshal([]byte(y), &props), "failed to parse test properties")
	return props
}

func skillSource(props map[string]any) basecatalog.PluginSource {
	return basecatalog.PluginSource{
		Name:       "Community Skills",
		ID:         "community-skills",
		Type:       SourceTypeGitSkillsPlugin,
		Properties: props,
	}
}

func TestParseSkillSource_InlineRepositories(t *testing.T) {
	src := skillSource(mustProps(t, `
trustTier: communityContributed
syncIntervalMinutes: 60
repositories:
  - url: https://github.com/example/skills.git
    refs: [main, v1.0]
    scanPaths: [skills/]
    authSecretName: git-creds
    provider: Example Org
    category: DevOps
    labels: [community]
    includedSkills: ["*"]
    excludedSkills: ["*-draft"]
    skillOverrides:
      - name: deploy
        category: SRE
        labels: [ops]
`))

	spec, err := ParseSkillSource(src)
	require.NoError(t, err)
	assert.Equal(t, "communityContributed", spec.TrustTier)
	assert.Equal(t, 60, spec.SyncIntervalMinutes)
	require.Len(t, spec.Repositories, 1)

	r := spec.Repositories[0]
	assert.Equal(t, "https://github.com/example/skills.git", r.URL)
	assert.Equal(t, []string{"main", "v1.0"}, r.Refs)
	assert.Equal(t, []string{"skills/"}, r.ScanPaths)
	assert.Equal(t, "git-creds", r.AuthSecretName)
	assert.Equal(t, []string{"*"}, r.IncludedSkills)
	assert.Equal(t, []string{"*-draft"}, r.ExcludedSkills)
	require.Len(t, r.SkillOverrides, 1)
	assert.Equal(t, SkillOverride{Name: "deploy", Category: "SRE", Labels: []string{"ops"}}, r.SkillOverrides[0])
}

func TestParseSkillSource_FileFormEquivalentToInline(t *testing.T) {
	repoYAML := `
repositories:
  - url: https://github.com/example/skills.git
    refs: [main, v1.0]
    provider: Example Org
    category: DevOps
    labels: [community]
`
	dir := t.TempDir()
	repoFile := filepath.Join(dir, "example-skills.yaml")
	require.NoError(t, os.WriteFile(repoFile, []byte(repoYAML), 0o600))

	fileSrc := skillSource(mustProps(t, `
trustTier: communityContributed
yamlCatalogPath: example-skills.yaml
`))
	fileSrc.Origin = filepath.Join(dir, "catalog-sources.yaml")

	inlineSrc := skillSource(mustProps(t, `
trustTier: communityContributed
repositories:
  - url: https://github.com/example/skills.git
    refs: [main, v1.0]
    provider: Example Org
    category: DevOps
    labels: [community]
`))

	fileSpec, err := ParseSkillSource(fileSrc)
	require.NoError(t, err, "file form")
	inlineSpec, err := ParseSkillSource(inlineSrc)
	require.NoError(t, err, "inline form")

	assert.Equal(t, inlineSpec.Repositories, fileSpec.Repositories,
		"file and inline forms should yield equivalent repositories")
}

func TestParseSkillSource_FileFormAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	repoFile := filepath.Join(dir, "abs-skills.yaml")
	require.NoError(t, os.WriteFile(repoFile,
		[]byte("repositories:\n  - url: https://github.com/example/skills.git\n"), 0o600))

	src := skillSource(map[string]any{"yamlCatalogPath": repoFile})
	spec, err := ParseSkillSource(src)
	require.NoError(t, err)
	require.Len(t, spec.Repositories, 1)
}

// TestParseSkillSource_ThroughReadSourceConfig exercises the full chain: a
// skill_catalogs section parsed by basecatalog.ReadSourceConfig (strict) and then
// handed to ParseSkillSource, for both the inline and file forms.
func TestParseSkillSource_ThroughReadSourceConfig(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "redhat-skills.yaml"),
		[]byte("repositories:\n  - url: https://github.com/redhat/skills.git\n    refs: [main]\n"), 0o600))

	configPath := filepath.Join(dir, "catalog-sources.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(`
skill_catalogs:
  - name: Community Skills
    id: community-skills
    type: git-skills-plugin
    enabled: true
    labels: [community]
    properties:
      trustTier: communityContributed
      syncIntervalMinutes: 30
      repositories:
        - url: https://github.com/example/skills.git
          refs: [main, v1.0]
  - name: Red Hat Skills
    id: redhat-skills
    type: git-skills-plugin
    enabled: true
    properties:
      trustTier: platformProvided
      yamlCatalogPath: redhat-skills.yaml
`), 0o600))

	config, err := basecatalog.ReadSourceConfig(configPath)
	require.NoError(t, err)
	require.Len(t, config.SkillCatalogs, 2)

	for _, src := range config.SkillCatalogs {
		src.Origin = configPath // set by the loader in real use
		spec, err := ParseSkillSource(src)
		require.NoErrorf(t, err, "ParseSkillSource(%s)", src.GetId())
		assert.Lenf(t, spec.Repositories, 1, "source %s", src.GetId())
	}
}

func TestParseSkillSource_InvalidTrustTier(t *testing.T) {
	src := skillSource(mustProps(t, `
trustTier: superDuperTrusted
repositories:
  - url: https://github.com/example/skills.git
`))
	_, err := ParseSkillSource(src)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "trustTier")
	assert.Contains(t, err.Error(), "superDuperTrusted")
}

func TestParseSkillSource_EmptyTrustTierAllowed(t *testing.T) {
	src := skillSource(mustProps(t, `
repositories:
  - url: https://github.com/example/skills.git
`))
	spec, err := ParseSkillSource(src)
	require.NoError(t, err)
	assert.Empty(t, spec.TrustTier)
}

func TestParseSkillSource_BothFormsRejected(t *testing.T) {
	src := skillSource(mustProps(t, `
yamlCatalogPath: repos.yaml
repositories:
  - url: https://github.com/example/skills.git
`))
	_, err := ParseSkillSource(src)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not both")
}

func TestParseSkillSource_NeitherFormRejected(t *testing.T) {
	src := skillSource(mustProps(t, `trustTier: communityContributed`))
	_, err := ParseSkillSource(src)
	require.Error(t, err)
}

func TestParseSkillSource_RepoMissingURL(t *testing.T) {
	src := skillSource(mustProps(t, `
repositories:
  - provider: Example Org
    refs: [main]
`))
	_, err := ParseSkillSource(src)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "url is required")
}

func TestParseSkillSource_DuplicateRepoURL(t *testing.T) {
	src := skillSource(mustProps(t, `
repositories:
  - url: https://github.com/example/skills.git
  - url: https://github.com/example/skills.git
`))
	_, err := ParseSkillSource(src)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
}

func TestParseSkillSource_UnknownPropertyRejected(t *testing.T) {
	src := skillSource(mustProps(t, `
trustTier: communityContributed
bogusKey: nope
repositories:
  - url: https://github.com/example/skills.git
`))
	_, err := ParseSkillSource(src)
	require.Error(t, err)
}

func TestParseSkillSource_WrongSourceType(t *testing.T) {
	src := skillSource(mustProps(t, `
repositories:
  - url: https://github.com/example/skills.git
`))
	src.Type = "hf"
	_, err := ParseSkillSource(src)
	require.Error(t, err)
}

func TestParseSkillSource_NegativeSyncInterval(t *testing.T) {
	src := skillSource(mustProps(t, `
syncIntervalMinutes: -5
repositories:
  - url: https://github.com/example/skills.git
`))
	_, err := ParseSkillSource(src)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "syncIntervalMinutes")
}
