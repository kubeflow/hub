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

func runAgentLeaderOperations(ctx context.Context, t *testing.T, baseLoader *basecatalog.BaseLoader, loader *AgentLoader) {
	t.Helper()

	require.NoError(t, loader.ParseAllConfigs())

	baseLoader.SetLeader(true)

	leaderDone := make(chan error, 1)
	go func() {
		leaderDone <- loader.PerformLeaderOperations(ctx, mapset.NewSet[string]())
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
	runAgentLeaderOperations(t.Context(), t, baseLoader, loader)

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
	ctx1, cancel1 := context.WithCancel(t.Context())
	defer cancel1()
	runAgentLeaderOperations(ctx1, t, baseLoader, loader)

	agentBefore, err := services.AgentRepository.GetByName("test_agent_catalog:test-agent")
	require.NoError(t, err)
	agentIDBefore := *agentBefore.GetID()

	// Cancel loader1's watcher goroutines before modifying the file to avoid
	// concurrent writes from the old loader during the reload.
	cancel1()

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
	runAgentLeaderOperations(t.Context(), t, baseLoader2, loader2)

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

// TestAgentLoaderFileWatchReload verifies that modifying a YAML data file while
// the loader is running triggers a live reload: the new agent appears in the DB
// and the stale agent from the previous load is removed by orphan cleanup.
func TestAgentLoaderFileWatchReload(t *testing.T) {
	_, services, cleanup := setupAgentLoaderTest(t)
	defer cleanup()

	tmpDir := t.TempDir()

	agentsFile := filepath.Join(tmpDir, "agents.yaml")
	require.NoError(t, os.WriteFile(agentsFile, []byte(`agents:
  - name: "watch-agent-old"
    description: "Initial agent"
`), 0644))

	sourcesFile := filepath.Join(tmpDir, "sources.yaml")
	require.NoError(t, os.WriteFile(sourcesFile, []byte(`agent_catalogs:
  - name: "File Watch Agent Catalog"
    id: file_watch_agent_catalog
    type: yaml
    enabled: true
    properties:
      yamlCatalogPath: `+agentsFile+`
`), 0644))

	baseLoader := basecatalog.NewBaseLoader([]string{sourcesFile})
	loader := NewAgentLoader(services, baseLoader)

	// Keep ctx alive so the file-watch goroutines continue running during the test.
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	runAgentLeaderOperations(ctx, t, baseLoader, loader)

	// Verify initial state: the original agent is in the DB.
	agent, err := services.AgentRepository.GetByName("file_watch_agent_catalog:watch-agent-old")
	require.NoError(t, err)
	require.NotNil(t, agent.GetID())

	// Update the YAML file with a different agent name — the file-watch path
	// should trigger a live reload that adds the new agent and (via orphan cleanup
	// in removeOrphanedAgentsFromSource) removes the old one.
	require.NoError(t, os.WriteFile(agentsFile, []byte(`agents:
  - name: "watch-agent-new"
    description: "Updated agent"
`), 0644))

	// Wait for the file-watcher to detect the change and commit the reload to DB.
	// The monitor pauses 1 second after an fsnotify event before dispatching, so
	// we allow a generous timeout.
	assert.Eventually(t, func() bool {
		a, err := services.AgentRepository.GetByName("file_watch_agent_catalog:watch-agent-new")
		return err == nil && a != nil
	}, 15*time.Second, 100*time.Millisecond, "file-watch reload should add the new agent to the catalog")

	// removeOrphanedAgentsFromSource runs after the sentinel — which arrives after
	// the new agent is already committed, so poll until cleanup finishes.
	assert.Eventually(t, func() bool {
		result, err := services.AgentRepository.List(&models.AgentListOptions{
			SourceIDs: &[]string{"file_watch_agent_catalog"},
		})
		if err != nil {
			return false
		}
		for _, a := range result.Items {
			if attrs := a.GetAttributes(); attrs != nil && attrs.Name != nil {
				if *attrs.Name == "file_watch_agent_catalog:watch-agent-old" {
					return false
				}
			}
		}
		return true
	}, 15*time.Second, 100*time.Millisecond, "orphaned agent from previous load should be removed")
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
	ctx1, cancel1 := context.WithCancel(t.Context())
	defer cancel1()
	runAgentLeaderOperations(ctx1, t, baseLoader, loader)

	agent, err := services.AgentRepository.GetByName("test_agent_catalog:test-agent")
	require.NoError(t, err)
	agentID := *agent.GetID()

	templatesBefore := listTemplatesForParent(t, services, agentID)
	require.Len(t, templatesBefore, 1)

	// Cancel loader1's watcher goroutines before modifying the file to avoid
	// concurrent writes from the old loader during the reload.
	cancel1()

	// Reload the same agent with its templates list removed entirely.
	require.NoError(t, os.WriteFile(agentsFile, []byte(`agents:
  - name: "test-agent"
    description: "Test agent"
`), 0644))

	baseLoader2 := basecatalog.NewBaseLoader([]string{sourcesFile})
	loader2 := NewAgentLoader(services, baseLoader2)
	runAgentLeaderOperations(t.Context(), t, baseLoader2, loader2)

	templatesAfter := listTemplatesForParent(t, services, agentID)
	assert.Empty(t, templatesAfter, "stale template artifacts must be cleared when the reloaded agent declares no templates")
}
