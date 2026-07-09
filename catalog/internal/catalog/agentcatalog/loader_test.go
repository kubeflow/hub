package agentcatalog

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/kubeflow/hub/catalog/internal/catalog/agentcatalog/models"
	agentservice "github.com/kubeflow/hub/catalog/internal/catalog/agentcatalog/service"
	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	"github.com/kubeflow/hub/catalog/internal/db/service"
	"github.com/kubeflow/hub/catalog/internal/testhelpers"
	"github.com/kubeflow/hub/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	os.Exit(testutils.TestMainPostgresHelper(m))
}

func setupAgentLoaderTest(t *testing.T) (*gorm.DB, Services, func()) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testhelpers.MustDatastoreSpec(t))

	catalogSourceTypeID := testhelpers.GetCatalogSourceTypeIDForDBTest(t, sharedDB)
	agentTypeID := testhelpers.GetAgentTypeIDForDBTest(t, sharedDB)
	agentTemplateArtifactTypeID := testhelpers.GetAgentTemplateArtifactTypeIDForDBTest(t, sharedDB)

	services := Services{
		AgentRepository:                 agentservice.NewAgentRepository(sharedDB, agentTypeID),
		AgentTemplateArtifactRepository: agentservice.NewAgentTemplateArtifactRepository(sharedDB, agentTemplateArtifactTypeID),
		CatalogSourceRepository:         service.NewCatalogSourceRepository(sharedDB, catalogSourceTypeID),
		PropertyOptionsRepository:       service.NewPropertyOptionsRepository(sharedDB),
	}

	return sharedDB, services, cleanup
}

func runAgentLeaderOperations(t *testing.T, baseLoader *basecatalog.BaseLoader, loader *AgentLoader) {
	t.Helper()

	require.NoError(t, loader.ParseAllConfigs())

	baseLoader.SetLeader(true)

	leaderDone := make(chan error, 1)
	go func() {
		leaderDone <- loader.PerformLeaderOperations(context.Background(), mapset.NewSet[string]())
	}()

	select {
	case err := <-leaderDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for leader operations")
	}

	baseLoader.WaitForInflightWrites(5 * time.Second)
}

func writeAgentSourceFiles(t *testing.T, dir string, agentsYAML string) (agentsFile, sourcesFile string) {
	t.Helper()

	agentsFile = filepath.Join(dir, "agents.yaml")
	require.NoError(t, os.WriteFile(agentsFile, []byte(agentsYAML), 0644))

	sourcesFile = filepath.Join(dir, "sources.yaml")
	require.NoError(t, os.WriteFile(sourcesFile, []byte(`agent_catalogs:
  - name: "Test Agent Catalog"
    id: test_agent_catalog
    type: yaml
    enabled: true
    properties:
      yamlCatalogPath: `+agentsFile+`
`), 0644))

	return agentsFile, sourcesFile
}

func listTemplatesForParent(t *testing.T, services Services, parentID int32) []models.AgentTemplateArtifact {
	t.Helper()
	result, err := services.AgentTemplateArtifactRepository.List(models.AgentTemplateArtifactListOptions{ParentResourceID: &parentID})
	require.NoError(t, err)
	return result.Items
}

// TestAgentLoaderTemplateSaveFlow_InitialLoad verifies that on first load the
// loader saves the parent agent and persists all of its template artifacts,
// qualified with source/agent/name.
func TestAgentLoaderTemplateSaveFlow_InitialLoad(t *testing.T) {
	_, services, cleanup := setupAgentLoaderTest(t)
	defer cleanup()

	tmpDir := t.TempDir()
	_, sourcesFile := writeAgentSourceFiles(t, tmpDir, `agents:
  - name: "test-agent"
    description: "Test agent"
    templates:
      - name: "template-a.yaml"
        content: "content-a"
      - name: "template-b.yaml"
        content: "content-b"
`)

	baseLoader := basecatalog.NewBaseLoader([]string{sourcesFile})
	loader := NewAgentLoader(services, baseLoader)
	runAgentLeaderOperations(t, baseLoader, loader)

	agent, err := services.AgentRepository.GetByName("test_agent_catalog:test-agent")
	require.NoError(t, err)
	require.NotNil(t, agent.GetID())

	templates := listTemplatesForParent(t, services, *agent.GetID())
	require.Len(t, templates, 2)

	byName := map[string]string{}
	for _, tmpl := range templates {
		attrs := tmpl.GetAttributes()
		require.NotNil(t, attrs.Name)
		require.NotNil(t, attrs.Content)
		byName[*attrs.Name] = *attrs.Content
	}
	assert.Equal(t, "content-a", byName["test_agent_catalog:test-agent:template-a.yaml"])
	assert.Equal(t, "content-b", byName["test_agent_catalog:test-agent:template-b.yaml"])
}

// TestAgentLoaderTemplateSaveFlow_ReloadReplacesTemplates verifies the
// loader's save-parent -> delete-children -> save-new-children flow: after a
// reload with a different set of templates, the old template artifacts must
// be gone and only the new ones remain, while the parent agent's identity
// (its ID) is preserved via upsert.
func TestAgentLoaderTemplateSaveFlow_ReloadReplacesTemplates(t *testing.T) {
	_, services, cleanup := setupAgentLoaderTest(t)
	defer cleanup()

	tmpDir := t.TempDir()
	agentsFile, sourcesFile := writeAgentSourceFiles(t, tmpDir, `agents:
  - name: "test-agent"
    description: "Test agent"
    templates:
      - name: "template-a.yaml"
        content: "content-a"
      - name: "template-b.yaml"
        content: "content-b"
`)

	baseLoader := basecatalog.NewBaseLoader([]string{sourcesFile})
	loader := NewAgentLoader(services, baseLoader)
	runAgentLeaderOperations(t, baseLoader, loader)

	agentBefore, err := services.AgentRepository.GetByName("test_agent_catalog:test-agent")
	require.NoError(t, err)
	agentIDBefore := *agentBefore.GetID()

	// Reload with a completely different template set for the same agent.
	require.NoError(t, os.WriteFile(agentsFile, []byte(`agents:
  - name: "test-agent"
    description: "Test agent"
    templates:
      - name: "template-c.yaml"
        content: "content-c"
`), 0644))

	baseLoader2 := basecatalog.NewBaseLoader([]string{sourcesFile})
	loader2 := NewAgentLoader(services, baseLoader2)
	runAgentLeaderOperations(t, baseLoader2, loader2)

	agentAfter, err := services.AgentRepository.GetByName("test_agent_catalog:test-agent")
	require.NoError(t, err)
	assert.Equal(t, agentIDBefore, *agentAfter.GetID(), "reloading should upsert the same agent, not create a new one")

	templates := listTemplatesForParent(t, services, *agentAfter.GetID())
	require.Len(t, templates, 1, "stale template artifacts from the previous load must be deleted")

	attrs := templates[0].GetAttributes()
	require.NotNil(t, attrs.Name)
	assert.Equal(t, "test_agent_catalog:test-agent:template-c.yaml", *attrs.Name)
	require.NotNil(t, attrs.Content)
	assert.Equal(t, "content-c", *attrs.Content)
}

// TestAgentLoaderTemplateSaveFlow_ReloadWithNoTemplatesRemovesAll verifies
// that dropping the "templates" key entirely on reload clears previously
// saved template artifacts instead of leaving them orphaned.
func TestAgentLoaderTemplateSaveFlow_ReloadWithNoTemplatesRemovesAll(t *testing.T) {
	_, services, cleanup := setupAgentLoaderTest(t)
	defer cleanup()

	tmpDir := t.TempDir()
	agentsFile, sourcesFile := writeAgentSourceFiles(t, tmpDir, `agents:
  - name: "test-agent"
    description: "Test agent"
    templates:
      - name: "template-a.yaml"
        content: "content-a"
`)

	baseLoader := basecatalog.NewBaseLoader([]string{sourcesFile})
	loader := NewAgentLoader(services, baseLoader)
	runAgentLeaderOperations(t, baseLoader, loader)

	agent, err := services.AgentRepository.GetByName("test_agent_catalog:test-agent")
	require.NoError(t, err)
	agentID := *agent.GetID()

	templatesBefore := listTemplatesForParent(t, services, agentID)
	require.Len(t, templatesBefore, 1)

	// Reload the same agent with its templates list removed entirely.
	require.NoError(t, os.WriteFile(agentsFile, []byte(`agents:
  - name: "test-agent"
    description: "Test agent"
`), 0644))

	baseLoader2 := basecatalog.NewBaseLoader([]string{sourcesFile})
	loader2 := NewAgentLoader(services, baseLoader2)
	runAgentLeaderOperations(t, baseLoader2, loader2)

	templatesAfter := listTemplatesForParent(t, services, agentID)
	assert.Empty(t, templatesAfter, "stale template artifacts must be cleared when the reloaded agent declares no templates")
}
