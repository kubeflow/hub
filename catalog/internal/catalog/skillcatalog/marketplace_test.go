package skillcatalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	model "github.com/kubeflow/hub/catalog/pkg/openapi"
)

func ptr(s string) *string { return &s }

func skill(name, repo, path, version, commit string) model.Skill {
	return model.Skill{
		Name:           name,
		Repository:     ptr(repo),
		Path:           ptr(path),
		Version:        ptr(version),
		ResolvedCommit: ptr(commit),
	}
}

func pluginByName(m Marketplace, name string) (MarketplacePlugin, bool) {
	for _, p := range m.Plugins {
		if p.Name == name {
			return p, true
		}
	}
	return MarketplacePlugin{}, false
}

func TestBuildMarketplace_GitSubdirPinnedToCommit(t *testing.T) {
	skills := []model.Skill{skill("deploy", "https://github.com/org/repo.git", "skills/deploy", "v1.0", "deadbeef")}

	m := BuildMarketplace(skills, MarketplaceOptions{Name: "cat", Owner: MarketplaceOwner{Name: "Kubeflow"}})

	require.Len(t, m.Plugins, 1)
	p := m.Plugins[0]
	assert.Equal(t, "deploy", p.Name)
	assert.Equal(t, gitSubdirSourceType, p.Source.Source)
	assert.Equal(t, "https://github.com/org/repo.git", p.Source.URL)
	assert.Equal(t, "skills/deploy", p.Source.Path)
	assert.Equal(t, "v1.0", p.Source.Ref)
	assert.Equal(t, "deadbeef", p.Source.SHA, "install must be pinned to the resolved commit")
}

func TestBuildMarketplace_UniqueNamesAcrossRefsAndRepos(t *testing.T) {
	skills := []model.Skill{
		skill("deploy", "https://github.com/org/repo.git", "skills/deploy", "v2.0", "c2"),
		skill("deploy", "https://github.com/org/repo.git", "skills/deploy", "v1.0", "c1"),
		skill("deploy", "https://github.com/other/repo.git", "deploy", "v1.0", "c3"),
	}

	m := BuildMarketplace(skills, MarketplaceOptions{Name: "cat", Owner: MarketplaceOwner{Name: "Kubeflow"}})
	require.Len(t, m.Plugins, 3)

	names := map[string]int{}
	for _, p := range m.Plugins {
		names[p.Name]++
	}
	for name, count := range names {
		assert.Equalf(t, 1, count, "plugin name %q must be unique within the marketplace", name)
	}

	// The first entry in deterministic (repo, path, version) order keeps the bare name.
	_, ok := pluginByName(m, "deploy")
	assert.True(t, ok, "one entry should keep the bare skill name")
}

func TestBuildMarketplace_DeterministicOrder(t *testing.T) {
	skills := []model.Skill{
		skill("b", "https://github.com/org/repo.git", "skills/b", "v1.0", "c2"),
		skill("a", "https://github.com/org/repo.git", "skills/a", "v1.0", "c1"),
	}
	first := BuildMarketplace(skills, MarketplaceOptions{Name: "cat"})
	second := BuildMarketplace(skills, MarketplaceOptions{Name: "cat"})
	assert.Equal(t, first, second)
	// Sorted by path: skills/a before skills/b.
	assert.Equal(t, "a", first.Plugins[0].Name)
}

func TestBuildMarketplace_SkipsIncompleteIdentity(t *testing.T) {
	skills := []model.Skill{
		{Name: "no-repo"},                                              // missing repository
		{Repository: ptr("https://github.com/org/repo.git")},          // missing name
		skill("ok", "https://github.com/org/repo.git", "s/ok", "v1", "c"),
	}
	m := BuildMarketplace(skills, MarketplaceOptions{Name: "cat"})
	require.Len(t, m.Plugins, 1)
	assert.Equal(t, "ok", m.Plugins[0].Name)
}

func TestBuildMarketplace_AuthorFallsBackToProvider(t *testing.T) {
	s := skill("deploy", "https://github.com/org/repo.git", "skills/deploy", "v1.0", "c")
	s.Provider = ptr("Example Org")
	m := BuildMarketplace([]model.Skill{s}, MarketplaceOptions{Name: "cat"})
	require.Len(t, m.Plugins, 1)
	require.NotNil(t, m.Plugins[0].Author)
	assert.Equal(t, "Example Org", m.Plugins[0].Author.Name)
}

func TestBuildMarketplace_RootPathMapsToDot(t *testing.T) {
	m := BuildMarketplace([]model.Skill{skill("whole", "https://github.com/org/repo.git", ".", "v1.0", "c")}, MarketplaceOptions{Name: "cat"})
	require.Len(t, m.Plugins, 1)
	assert.Equal(t, ".", m.Plugins[0].Source.Path)
}

func TestMarketplaceConfig_MirrorRewrite(t *testing.T) {
	cfg := MarketplaceConfig{
		Name:           "cat",
		Owner:          "Kubeflow",
		ExternalMirror: "https://mirror.example.com/git",
		InternalMirror: "http://skills-git.hub.svc:8080",
	}
	skills := []model.Skill{skill("deploy", "https://github.com/org/repo.git", "skills/deploy", "v1.0", "c")}

	// Default audience -> external mirror.
	ext := BuildMarketplace(skills, cfg.Options(""))
	require.Len(t, ext.Plugins, 1)
	assert.Equal(t, "https://mirror.example.com/git/org/repo.git", ext.Plugins[0].Source.URL)

	// audience=cluster -> internal mirror.
	in := BuildMarketplace(skills, cfg.Options("cluster"))
	require.Len(t, in.Plugins, 1)
	assert.Equal(t, "http://skills-git.hub.svc:8080/org/repo.git", in.Plugins[0].Source.URL)
}

func TestMarketplaceConfig_NoMirrorKeepsCanonical(t *testing.T) {
	cfg := MarketplaceConfig{Name: "cat", Owner: "Kubeflow"}
	skills := []model.Skill{skill("deploy", "https://github.com/org/repo.git", "skills/deploy", "v1.0", "c")}
	m := BuildMarketplace(skills, cfg.Options("cluster"))
	require.Len(t, m.Plugins, 1)
	assert.Equal(t, "https://github.com/org/repo.git", m.Plugins[0].Source.URL)
	assert.Equal(t, "cat", m.Name)
	assert.Equal(t, "Kubeflow", m.Owner.Name)
}
