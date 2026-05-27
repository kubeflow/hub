package mcpcatalog

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	mcpmodels "github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog/models"
	mcpcatalogservice "github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog/service"
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

func setupMCPLoaderTest(t *testing.T) (*gorm.DB, Services, func()) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, service.DatastoreSpec())

	catalogSourceTypeID := testhelpers.GetCatalogSourceTypeIDForDBTest(t, sharedDB)
	mcpServerTypeID := testhelpers.GetMCPServerTypeIDForDBTest(t, sharedDB)
	mcpServerToolTypeID := testhelpers.GetMCPServerToolTypeIDForDBTest(t, sharedDB)

	services := Services{
		MCPServerRepository:       mcpcatalogservice.NewMCPServerRepository(sharedDB, mcpServerTypeID),
		MCPServerToolRepository:   mcpcatalogservice.NewMCPServerToolRepository(sharedDB, mcpServerToolTypeID),
		CatalogSourceRepository:   service.NewCatalogSourceRepository(sharedDB, catalogSourceTypeID),
		PropertyOptionsRepository: service.NewPropertyOptionsRepository(sharedDB),
	}

	return sharedDB, services, cleanup
}

func TestMCPLoaderBasicLoad(t *testing.T) {
	_, services, cleanup := setupMCPLoaderTest(t)
	defer cleanup()

	// Create temporary MCP catalog files
	tmpDir := t.TempDir()

	serversFile := filepath.Join(tmpDir, "servers.yaml")
	err := os.WriteFile(serversFile, []byte(`mcp_servers:
  - name: "test-server"
    description: "Test MCP server"
    provider: "Test Provider"
    version: "1.0.0"
    tools:
      - name: "test-tool"
        description: "Test tool"
`), 0644)
	require.NoError(t, err)

	sourcesFile := filepath.Join(tmpDir, "sources.yaml")
	err = os.WriteFile(sourcesFile, []byte(`mcp_catalogs:
  - name: "Test MCP Catalog"
    id: test_mcp_catalog
    type: yaml
    enabled: true
    properties:
      yamlCatalogPath: `+serversFile+`
`), 0644)
	require.NoError(t, err)

	// Create and start the loader
	baseLoader := basecatalog.NewBaseLoader([]string{sourcesFile})
	loader := NewMCPLoaderWithState(services, baseLoader)
	ctx := context.Background()

	// Parse configs
	err = loader.ParseAllConfigs()
	require.NoError(t, err)

	// Set leader mode and perform leader operations
	baseLoader.SetLeader(true)

	// Perform leader operations in background
	leaderDone := make(chan error, 1)
	go func() {
		leaderDone <- loader.PerformLeaderOperations(ctx, mapset.NewSet[string]())
	}()

	// Wait for leader operations to complete
	select {
	case err := <-leaderDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for leader operations")
	}

	// Wait for all database writes to complete
	baseLoader.WaitForInflightWrites(5 * time.Second)

	// Verify server was loaded
	server, err := services.MCPServerRepository.GetByNameAndVersion("test-server", "1.0.0")
	require.NoError(t, err)
	assert.NotNil(t, server)

	attrs := server.GetAttributes()
	require.NotNil(t, attrs)
	assert.Equal(t, "test-server", *attrs.Name)

	// Verify properties
	props := server.GetProperties()
	require.NotNil(t, props)

	foundDescription := false
	foundSourceID := false
	for _, prop := range *props {
		if prop.Name == "description" && prop.StringValue != nil {
			assert.Equal(t, "Test MCP server", *prop.StringValue)
			foundDescription = true
		}
		if prop.Name == "source_id" && prop.StringValue != nil {
			assert.Equal(t, "test_mcp_catalog", *prop.StringValue)
			foundSourceID = true
		}
	}
	assert.True(t, foundDescription, "description property should be set")
	assert.True(t, foundSourceID, "source_id property should be set")

	// Verify tools were persisted to the database
	require.NotNil(t, server.GetID())
	toolsList, err := services.MCPServerToolRepository.List(mcpmodels.MCPServerToolListOptions{ParentID: *server.GetID()})
	require.NoError(t, err)
	require.Len(t, toolsList.Items, 1)
	toolAttrs := toolsList.Items[0].GetAttributes()
	require.NotNil(t, toolAttrs)
	assert.Equal(t, "test-server@1.0.0:test-tool", *toolAttrs.Name)
}

func TestMCPLoaderToolAccessTypeAndParameters(t *testing.T) {
	_, services, cleanup := setupMCPLoaderTest(t)
	defer cleanup()

	tmpDir := t.TempDir()
	serversFile := filepath.Join(tmpDir, "servers.yaml")
	err := os.WriteFile(serversFile, []byte(`mcp_servers:
  - name: "jira-mcp"
    version: "1.0.0"
    tools:
      - name: "search_issues"
        description: "Search issues using JQL"
        accessType: read_only
        parameters:
          - name: jql
            type: string
            description: "JQL query"
            required: true
      - name: "create_issue"
        description: "Create a new issue"
        accessType: read_write
        parameters:
          - name: project
            type: string
            required: true
          - name: summary
            type: string
            required: true
`), 0644)
	require.NoError(t, err)

	sourcesFile := filepath.Join(tmpDir, "sources.yaml")
	err = os.WriteFile(sourcesFile, []byte(`mcp_catalogs:
  - name: "Test MCP Catalog"
    id: test_mcp_catalog
    type: yaml
    enabled: true
    properties:
      yamlCatalogPath: `+serversFile+`
`), 0644)
	require.NoError(t, err)

	baseLoader := basecatalog.NewBaseLoader([]string{sourcesFile})
	loader := NewMCPLoaderWithState(services, baseLoader)
	ctx := context.Background()

	err = loader.ParseAllConfigs()
	require.NoError(t, err)

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

	server, err := services.MCPServerRepository.GetByNameAndVersion("jira-mcp", "1.0.0")
	require.NoError(t, err)
	require.NotNil(t, server.GetID())

	toolsList, err := services.MCPServerToolRepository.List(mcpmodels.MCPServerToolListOptions{ParentID: *server.GetID()})
	require.NoError(t, err)
	require.Len(t, toolsList.Items, 2)

	// Build a map for easy lookup
	toolByName := make(map[string]mcpmodels.MCPServerTool)
	for _, tool := range toolsList.Items {
		attrs := tool.GetAttributes()
		require.NotNil(t, attrs)
		toolByName[*attrs.Name] = tool
	}

	// search_issues should have accessType=read_only and 1 parameter
	searchTool := toolByName["jira-mcp@1.0.0:search_issues"]
	require.NotNil(t, searchTool, "search_issues tool should exist")
	searchProps := searchTool.GetProperties()
	require.NotNil(t, searchProps)
	foundAccessType := false
	foundParameters := false
	for _, prop := range *searchProps {
		if prop.Name == "accessType" {
			assert.Equal(t, "read_only", *prop.StringValue)
			foundAccessType = true
		}
		if prop.Name == "parameters" {
			assert.Contains(t, *prop.StringValue, "jql")
			foundParameters = true
		}
	}
	assert.True(t, foundAccessType, "search_issues should have accessType property")
	assert.True(t, foundParameters, "search_issues should have parameters property")

	// create_issue should have accessType=read_write and 2 parameters
	createTool := toolByName["jira-mcp@1.0.0:create_issue"]
	require.NotNil(t, createTool, "create_issue tool should exist")
	createProps := createTool.GetProperties()
	require.NotNil(t, createProps)
	foundAccessType = false
	foundParameters = false
	for _, prop := range *createProps {
		if prop.Name == "accessType" {
			assert.Equal(t, "read_write", *prop.StringValue)
			foundAccessType = true
		}
		if prop.Name == "parameters" {
			assert.Contains(t, *prop.StringValue, "project")
			assert.Contains(t, *prop.StringValue, "summary")
			foundParameters = true
		}
	}
	assert.True(t, foundAccessType, "create_issue should have accessType property")
	assert.True(t, foundParameters, "create_issue should have parameters property")
}

func TestMCPLoaderDisabledSource(t *testing.T) {
	_, services, cleanup := setupMCPLoaderTest(t)
	defer cleanup()

	// Create temporary MCP catalog files
	tmpDir := t.TempDir()

	serversFile := filepath.Join(tmpDir, "servers.yaml")
	err := os.WriteFile(serversFile, []byte(`mcp_servers:
  - name: "disabled-server"
    description: "This server should not be loaded"
`), 0644)
	require.NoError(t, err)

	sourcesFile := filepath.Join(tmpDir, "sources.yaml")
	err = os.WriteFile(sourcesFile, []byte(`mcp_catalogs:
  - name: "Disabled MCP Catalog"
    id: disabled_mcp_catalog
    type: yaml
    enabled: false
    properties:
      yamlCatalogPath: `+serversFile+`
`), 0644)
	require.NoError(t, err)

	// Create and start the loader
	baseLoader := basecatalog.NewBaseLoader([]string{sourcesFile})
	loader := NewMCPLoaderWithState(services, baseLoader)
	ctx := context.Background()

	// Parse configs
	err = loader.ParseAllConfigs()
	require.NoError(t, err)

	// Set leader mode and perform leader operations
	baseLoader.SetLeader(true)

	// Perform leader operations in background
	leaderDone := make(chan error, 1)
	go func() {
		leaderDone <- loader.PerformLeaderOperations(ctx, mapset.NewSet[string]())
	}()

	// Wait for leader operations to complete
	select {
	case err := <-leaderDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for leader operations")
	}

	// Wait for all database writes to complete
	baseLoader.WaitForInflightWrites(5 * time.Second)

	// Verify server was NOT loaded
	_, err = services.MCPServerRepository.GetByNameAndVersion("disabled-server", "")
	assert.Error(t, err)

}

func TestMCPLoaderEventHandlers(t *testing.T) {
	_, services, cleanup := setupMCPLoaderTest(t)
	defer cleanup()

	// Create temporary MCP catalog files
	tmpDir := t.TempDir()

	serversFile := filepath.Join(tmpDir, "servers.yaml")
	err := os.WriteFile(serversFile, []byte(`mcp_servers:
  - name: "event-test-server"
    description: "Server for testing event handlers"
    tools:
      - name: "handler-tool"
`), 0644)
	require.NoError(t, err)

	sourcesFile := filepath.Join(tmpDir, "sources.yaml")
	err = os.WriteFile(sourcesFile, []byte(`mcp_catalogs:
  - name: "Event Test Catalog"
    id: event_test_catalog
    type: yaml
    enabled: true
    properties:
      yamlCatalogPath: `+serversFile+`
`), 0644)
	require.NoError(t, err)

	// Create loader and register event handler
	baseLoader := basecatalog.NewBaseLoader([]string{sourcesFile})
	loader := NewMCPLoaderWithState(services, baseLoader)

	handlerCalled := false
	var handlerRecord MCPServerProviderRecord

	loader.RegisterEventHandler(func(ctx context.Context, record MCPServerProviderRecord) error {
		handlerCalled = true
		handlerRecord = record
		return nil
	})

	ctx := context.Background()

	// Parse configs
	err = loader.ParseAllConfigs()
	require.NoError(t, err)

	// Set leader mode and perform leader operations
	baseLoader.SetLeader(true)

	// Perform leader operations in background
	leaderDone := make(chan error, 1)
	go func() {
		leaderDone <- loader.PerformLeaderOperations(ctx, mapset.NewSet[string]())
	}()

	// Wait for leader operations to complete
	select {
	case err := <-leaderDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for leader operations")
	}

	// Wait for all database writes to complete
	baseLoader.WaitForInflightWrites(5 * time.Second)

	// Verify handler was called
	assert.True(t, handlerCalled)
	assert.NotNil(t, handlerRecord.Server)
	require.Equal(t, 1, len(handlerRecord.Tools))
	assert.Equal(t, "handler-tool", handlerRecord.Tools[0].Name)

}

func TestMCPLoaderRemoveOrphans(t *testing.T) {
	_, services, cleanup := setupMCPLoaderTest(t)
	defer cleanup()

	// Create temporary MCP catalog files
	tmpDir := t.TempDir()

	serversFile := filepath.Join(tmpDir, "servers.yaml")

	// First load with two servers
	err := os.WriteFile(serversFile, []byte(`mcp_servers:
  - name: "server-1"
    description: "First server"
  - name: "server-2"
    description: "Second server"
`), 0644)
	require.NoError(t, err)

	sourcesFile := filepath.Join(tmpDir, "sources.yaml")
	err = os.WriteFile(sourcesFile, []byte(`mcp_catalogs:
  - name: "Orphan Test Catalog"
    id: orphan_test_catalog
    type: yaml
    enabled: true
    properties:
      yamlCatalogPath: `+serversFile+`
`), 0644)
	require.NoError(t, err)

	baseLoader := basecatalog.NewBaseLoader([]string{sourcesFile})
	loader := NewMCPLoaderWithState(services, baseLoader)

	ctx := context.Background()

	// Parse configs
	err = loader.ParseAllConfigs()
	require.NoError(t, err)

	// Set leader mode and perform leader operations
	baseLoader.SetLeader(true)

	// Perform leader operations in background
	leaderDone := make(chan error, 1)
	go func() {
		leaderDone <- loader.PerformLeaderOperations(ctx, mapset.NewSet[string]())
	}()

	// Wait for leader operations to complete
	select {
	case err := <-leaderDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for leader operations")
	}

	// Wait for all database writes to complete
	baseLoader.WaitForInflightWrites(5 * time.Second)

	// Verify both servers are loaded
	_, err = services.MCPServerRepository.GetByNameAndVersion("server-1", "")
	require.NoError(t, err)
	_, err = services.MCPServerRepository.GetByNameAndVersion("server-2", "")
	require.NoError(t, err)

	// Update the servers file to remove server-2
	err = os.WriteFile(serversFile, []byte(`mcp_servers:
  - name: "server-1"
    description: "First server (updated)"
`), 0644)
	require.NoError(t, err)

	// Reload
	baseLoader2 := basecatalog.NewBaseLoader([]string{sourcesFile})
	loader2 := NewMCPLoaderWithState(services, baseLoader2)

	// Parse configs
	err = loader2.ParseAllConfigs()
	require.NoError(t, err)

	// Set leader mode and perform leader operations
	baseLoader2.SetLeader(true)

	// Perform leader operations in background
	leaderDone2 := make(chan error, 1)
	go func() {
		leaderDone2 <- loader2.PerformLeaderOperations(ctx, mapset.NewSet[string]())
	}()

	// Wait for leader operations to complete
	select {
	case err := <-leaderDone2:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for second leader operations")
	}

	// Wait for all database writes to complete
	baseLoader2.WaitForInflightWrites(5 * time.Second)

	// Verify server-1 still exists
	_, err = services.MCPServerRepository.GetByNameAndVersion("server-1", "")
	require.NoError(t, err)

	// Verify server-2 was removed (orphaned)
	_, err = services.MCPServerRepository.GetByNameAndVersion("server-2", "")
	assert.Error(t, err)

}

func TestMCPLoaderRespectsContextCancellation(t *testing.T) {
	_, services, cleanup := setupMCPLoaderTest(t)
	defer cleanup()

	// Create temporary MCP catalog files with 3 servers
	tmpDir := t.TempDir()

	serversFile := filepath.Join(tmpDir, "servers.yaml")
	err := os.WriteFile(serversFile, []byte(`mcp_servers:
  - name: "cancel-server-1"
    description: "First server"
  - name: "cancel-server-2"
    description: "Second server"
  - name: "cancel-server-3"
    description: "Third server"
`), 0644)
	require.NoError(t, err)

	sourcesFile := filepath.Join(tmpDir, "sources.yaml")
	err = os.WriteFile(sourcesFile, []byte(`mcp_catalogs:
  - name: "Cancel Test Catalog"
    id: cancel_test_catalog
    type: yaml
    enabled: true
    properties:
      yamlCatalogPath: `+serversFile+`
`), 0644)
	require.NoError(t, err)

	// Create loader with a cancellable context
	baseLoader := basecatalog.NewBaseLoader([]string{sourcesFile})
	loader := NewMCPLoaderWithState(services, baseLoader)

	ctx, cancel := context.WithCancel(context.Background())

	// Track how many servers were processed via event handler
	serverCount := 0
	loader.RegisterEventHandler(func(ctx context.Context, record MCPServerProviderRecord) error {
		serverCount++
		// Cancel context after the first server is processed
		if serverCount == 1 {
			cancel()
		}
		return nil
	})

	// Parse configs
	err = loader.ParseAllConfigs()
	require.NoError(t, err)

	// Set leader mode and perform leader operations
	baseLoader.SetLeader(true)

	// Perform leader operations in background
	leaderDone := make(chan error, 1)
	go func() {
		leaderDone <- loader.PerformLeaderOperations(ctx, mapset.NewSet[string]())
	}()

	// Wait for leader operations to complete
	select {
	case <-leaderDone:
		// Error may or may not propagate depending on how loadAllServers handles it
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for leader operations")
	}

	// Wait for all database writes to complete
	baseLoader.WaitForInflightWrites(5 * time.Second)

	// Verify that not all 3 servers were processed
	assert.LessOrEqual(t, serverCount, 2, "context cancellation should stop processing before all servers are handled")
}

func TestMCPLoaderSavesSourceStatus(t *testing.T) {
	_, services, cleanup := setupMCPLoaderTest(t)
	defer cleanup()

	tmpDir := t.TempDir()

	serversFile := filepath.Join(tmpDir, "servers.yaml")
	err := os.WriteFile(serversFile, []byte(`mcp_servers:
  - name: "status-test-server"
    description: "Server for status test"
`), 0644)
	require.NoError(t, err)

	sourcesFile := filepath.Join(tmpDir, "sources.yaml")
	err = os.WriteFile(sourcesFile, []byte(`mcp_catalogs:
  - name: "Status Test Catalog"
    id: status_test_catalog
    type: yaml
    enabled: true
    properties:
      yamlCatalogPath: `+serversFile+`
`), 0644)
	require.NoError(t, err)

	baseLoader := basecatalog.NewBaseLoader([]string{sourcesFile})
	loader := NewMCPLoaderWithState(services, baseLoader)
	ctx := context.Background()

	err = loader.ParseAllConfigs()
	require.NoError(t, err)

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

	// Verify source status was saved as "available"
	source, err := services.CatalogSourceRepository.GetBySourceID("status_test_catalog")
	require.NoError(t, err)
	require.NotNil(t, source)

	props := source.GetProperties()
	require.NotNil(t, props)

	foundStatus := false
	for _, prop := range *props {
		if prop.Name == "status" && prop.StringValue != nil {
			assert.Equal(t, "available", *prop.StringValue)
			foundStatus = true
		}
	}
	assert.True(t, foundStatus, "status property should be set to 'available'")
}

func TestMCPLoaderSavesDisabledSourceStatus(t *testing.T) {
	_, services, cleanup := setupMCPLoaderTest(t)
	defer cleanup()

	tmpDir := t.TempDir()

	serversFile := filepath.Join(tmpDir, "servers.yaml")
	err := os.WriteFile(serversFile, []byte(`mcp_servers:
  - name: "disabled-status-server"
    description: "This server should not be loaded"
`), 0644)
	require.NoError(t, err)

	sourcesFile := filepath.Join(tmpDir, "sources.yaml")
	err = os.WriteFile(sourcesFile, []byte(`mcp_catalogs:
  - name: "Disabled Status Catalog"
    id: disabled_status_catalog
    type: yaml
    enabled: false
    properties:
      yamlCatalogPath: `+serversFile+`
`), 0644)
	require.NoError(t, err)

	baseLoader := basecatalog.NewBaseLoader([]string{sourcesFile})
	loader := NewMCPLoaderWithState(services, baseLoader)
	ctx := context.Background()

	err = loader.ParseAllConfigs()
	require.NoError(t, err)

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

	// Verify source status was saved as "disabled"
	source, err := services.CatalogSourceRepository.GetBySourceID("disabled_status_catalog")
	require.NoError(t, err)
	require.NotNil(t, source)

	props := source.GetProperties()
	require.NotNil(t, props)

	foundStatus := false
	for _, prop := range *props {
		if prop.Name == "status" && prop.StringValue != nil {
			assert.Equal(t, "disabled", *prop.StringValue)
			foundStatus = true
		}
	}
	assert.True(t, foundStatus, "status property should be set to 'disabled'")
}

func TestMCPLoaderSavesErrorSourceStatus(t *testing.T) {
	_, services, cleanup := setupMCPLoaderTest(t)
	defer cleanup()

	tmpDir := t.TempDir()

	sourcesFile := filepath.Join(tmpDir, "sources.yaml")
	err := os.WriteFile(sourcesFile, []byte(`mcp_catalogs:
  - name: "Error Status Catalog"
    id: error_status_catalog
    type: unknown_provider_type
    enabled: true
    properties:
      someProp: someValue
`), 0644)
	require.NoError(t, err)

	baseLoader := basecatalog.NewBaseLoader([]string{sourcesFile})
	loader := NewMCPLoaderWithState(services, baseLoader)
	ctx := context.Background()

	err = loader.ParseAllConfigs()
	require.NoError(t, err)

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

	// Verify source status was saved as "error"
	source, err := services.CatalogSourceRepository.GetBySourceID("error_status_catalog")
	require.NoError(t, err)
	require.NotNil(t, source)

	props := source.GetProperties()
	require.NotNil(t, props)

	foundStatus := false
	foundError := false
	for _, prop := range *props {
		if prop.Name == "status" && prop.StringValue != nil {
			assert.Equal(t, "error", *prop.StringValue)
			foundStatus = true
		}
		if prop.Name == "error" && prop.StringValue != nil {
			assert.Contains(t, *prop.StringValue, "unknown MCP provider type")
			foundError = true
		}
	}
	assert.True(t, foundStatus, "status property should be set to 'error'")
	assert.True(t, foundError, "error property should be set")
}

func TestMCPLoader_NamedQueriesFromConfig(t *testing.T) {
	_, services, cleanup := setupMCPLoaderTest(t)
	defer cleanup()

	tmpDir := t.TempDir()

	// A sources file that includes named queries scoped to mcp_servers
	sourcesFile := filepath.Join(tmpDir, "sources.yaml")
	err := os.WriteFile(sourcesFile, []byte(`mcp_catalogs: []
namedQueries:
  production_ready:
    assetType: mcp_servers
    filters:
      verifiedSource:
        operator: "="
        value: true
      version:
        operator: ">="
        value: "2"
`), 0644)
	require.NoError(t, err)

	baseLoader := basecatalog.NewBaseLoader([]string{sourcesFile})
	loader := NewMCPLoaderWithState(services, baseLoader)

	err = loader.ParseAllConfigs()
	require.NoError(t, err)

	queries := loader.Sources.GetNamedQueries()
	require.Len(t, queries, 1, "should have one named query")

	pq, ok := queries["production_ready"]
	require.True(t, ok, "production_ready named query should exist")
	assert.Equal(t, "=", pq["verifiedSource"].Operator)
	assert.Equal(t, true, pq["verifiedSource"].Value)
	assert.Equal(t, ">=", pq["version"].Operator)
	assert.Equal(t, "2", pq["version"].Value)
}

func TestMCPLoader_ModelNamedQueriesExcluded(t *testing.T) {
	_, services, cleanup := setupMCPLoaderTest(t)
	defer cleanup()

	tmpDir := t.TempDir()

	// A sources file with both model-scoped and MCP-scoped named queries
	sourcesFile := filepath.Join(tmpDir, "sources.yaml")
	err := os.WriteFile(sourcesFile, []byte(`mcp_catalogs: []
namedQueries:
  model-only-query:
    assetType: models
    filters:
      artifacts.ttft_p90.double_value:
        operator: "<="
        value: 100
  mcp-query:
    assetType: mcp_servers
    filters:
      verifiedSource:
        operator: "="
        value: true
  default-query:
    field:
      operator: "="
      value: "x"
`), 0644)
	require.NoError(t, err)

	baseLoader := basecatalog.NewBaseLoader([]string{sourcesFile})
	loader := NewMCPLoaderWithState(services, baseLoader)

	err = loader.ParseAllConfigs()
	require.NoError(t, err)

	queries := loader.Sources.GetNamedQueries()
	assert.Len(t, queries, 1, "MCP loader should only pick up mcp_servers-scoped named queries")
	assert.Contains(t, queries, "mcp-query")
	assert.NotContains(t, queries, "model-only-query")
	assert.NotContains(t, queries, "default-query") // defaults to models
}

func TestMCPLoader_InvalidNamedQueriesRejected(t *testing.T) {
	_, services, cleanup := setupMCPLoaderTest(t)
	defer cleanup()

	tmpDir := t.TempDir()

	sourcesFile := filepath.Join(tmpDir, "sources.yaml")
	err := os.WriteFile(sourcesFile, []byte(`mcp_catalogs: []
namedQueries:
  bad_query:
    assetType: mcp_servers
    filters:
      name:
        operator: "INVALID_OP"
        value: "something"
`), 0644)
	require.NoError(t, err)

	baseLoader := basecatalog.NewBaseLoader([]string{sourcesFile})
	loader := NewMCPLoaderWithState(services, baseLoader)

	err = loader.ParseAllConfigs()
	require.Error(t, err, "invalid named query operator should be rejected")
	assert.Contains(t, err.Error(), "INVALID_OP")
}

func TestMCPLoaderExcludedServersNotSaved(t *testing.T) {
	_, services, cleanup := setupMCPLoaderTest(t)
	defer cleanup()

	tmpDir := t.TempDir()

	serversFile := filepath.Join(tmpDir, "servers.yaml")
	err := os.WriteFile(serversFile, []byte(`mcp_servers:
  - name: "allowed-server"
    description: "This server should be loaded"
  - name: "excluded-server"
    description: "This server should be skipped"
`), 0644)
	require.NoError(t, err)

	sourcesFile := filepath.Join(tmpDir, "sources.yaml")
	err = os.WriteFile(sourcesFile, []byte(`mcp_catalogs:
  - name: "Filter Test Catalog"
    id: filter_test_catalog
    type: yaml
    enabled: true
    excludedServers:
      - "excluded-server"
    properties:
      yamlCatalogPath: `+serversFile+`
`), 0644)
	require.NoError(t, err)

	baseLoader := basecatalog.NewBaseLoader([]string{sourcesFile})
	loader := NewMCPLoaderWithState(services, baseLoader)
	ctx := context.Background()

	err = loader.ParseAllConfigs()
	require.NoError(t, err)

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

	// allowed-server should be present
	server, err := services.MCPServerRepository.GetByNameAndVersion("allowed-server", "")
	require.NoError(t, err)
	assert.NotNil(t, server)

	// excluded-server must not be present
	_, err = services.MCPServerRepository.GetByNameAndVersion("excluded-server", "")
	assert.Error(t, err, "excluded-server should not be saved to the database")
}

func TestMCPLoaderIncludedServersOnly(t *testing.T) {
	_, services, cleanup := setupMCPLoaderTest(t)
	defer cleanup()

	tmpDir := t.TempDir()

	serversFile := filepath.Join(tmpDir, "servers.yaml")
	err := os.WriteFile(serversFile, []byte(`mcp_servers:
  - name: "allowed-server"
    description: "This server should be loaded"
  - name: "other-server"
    description: "This server is not in the include list"
`), 0644)
	require.NoError(t, err)

	sourcesFile := filepath.Join(tmpDir, "sources.yaml")
	err = os.WriteFile(sourcesFile, []byte(`mcp_catalogs:
  - name: "Include Filter Catalog"
    id: include_filter_catalog
    type: yaml
    enabled: true
    includedServers:
      - "allowed-server"
    properties:
      yamlCatalogPath: `+serversFile+`
`), 0644)
	require.NoError(t, err)

	baseLoader := basecatalog.NewBaseLoader([]string{sourcesFile})
	loader := NewMCPLoaderWithState(services, baseLoader)
	ctx := context.Background()

	err = loader.ParseAllConfigs()
	require.NoError(t, err)

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

	// allowed-server should be present
	server, err := services.MCPServerRepository.GetByNameAndVersion("allowed-server", "")
	require.NoError(t, err)
	assert.NotNil(t, server)

	// other-server must not be present (not in include list)
	_, err = services.MCPServerRepository.GetByNameAndVersion("other-server", "")
	assert.Error(t, err, "other-server should not be saved because it is not in includedServers")
}

func TestMCPLoaderNoFilterAllowsAll(t *testing.T) {
	_, services, cleanup := setupMCPLoaderTest(t)
	defer cleanup()

	tmpDir := t.TempDir()

	serversFile := filepath.Join(tmpDir, "servers.yaml")
	err := os.WriteFile(serversFile, []byte(`mcp_servers:
  - name: "server-a"
    description: "First server"
  - name: "server-b"
    description: "Second server"
`), 0644)
	require.NoError(t, err)

	sourcesFile := filepath.Join(tmpDir, "sources.yaml")
	err = os.WriteFile(sourcesFile, []byte(`mcp_catalogs:
  - name: "No Filter Catalog"
    id: no_filter_catalog
    type: yaml
    enabled: true
    properties:
      yamlCatalogPath: `+serversFile+`
`), 0644)
	require.NoError(t, err)

	baseLoader := basecatalog.NewBaseLoader([]string{sourcesFile})
	loader := NewMCPLoaderWithState(services, baseLoader)
	ctx := context.Background()

	err = loader.ParseAllConfigs()
	require.NoError(t, err)

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

	_, err = services.MCPServerRepository.GetByNameAndVersion("server-a", "")
	require.NoError(t, err)

	_, err = services.MCPServerRepository.GetByNameAndVersion("server-b", "")
	require.NoError(t, err)
}

func TestMCPLoaderInvalidServerFilterRejected(t *testing.T) {
	_, services, cleanup := setupMCPLoaderTest(t)
	defer cleanup()

	tmpDir := t.TempDir()

	sourcesFile := filepath.Join(tmpDir, "sources.yaml")
	err := os.WriteFile(sourcesFile, []byte(`mcp_catalogs:
  - name: "Bad Filter Catalog"
    id: bad_filter_catalog
    type: yaml
    enabled: true
    includedServers:
      - ""
    properties:
      someProp: someValue
`), 0644)
	require.NoError(t, err)

	baseLoader := basecatalog.NewBaseLoader([]string{sourcesFile})
	loader := NewMCPLoaderWithState(services, baseLoader)

	err = loader.ParseAllConfigs()
	require.Error(t, err, "empty includedServers pattern should be rejected at config load time")
	assert.Contains(t, err.Error(), "includedServers")
}
