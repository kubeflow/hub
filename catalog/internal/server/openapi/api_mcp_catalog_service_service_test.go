package openapi

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/kubeflow/hub/catalog/internal/catalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog"
	model "github.com/kubeflow/hub/catalog/pkg/openapi"
	"github.com/kubeflow/hub/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMCPProvider implements catalog.MCPProvider for testing
type mockMCPProvider struct {
	servers map[string]*model.MCPServer
	tools   map[string]map[string]*model.MCPTool // server_id -> tool_name -> tool
	// error simulation
	shouldErrorOnListServers      bool
	shouldErrorOnGetServer        bool
	shouldErrorOnListTools        bool
	shouldErrorOnGetTool          bool
	shouldErrorOnGetFilterOptions bool
	// namedQueryErr simulates an error returned by the provider for a specific named query.
	// This is used to verify the service correctly propagates provider errors.
	namedQueryErr map[string]error
	// filterOptions is returned by GetFilterOptions.
	filterOptions *model.FilterOptionsList
	// capturedParams captures the last params passed to ListMCPServers.
	capturedParams *catalog.ListMCPServersParams
}

func newMockMCPProvider() *mockMCPProvider {
	return &mockMCPProvider{
		servers: make(map[string]*model.MCPServer),
		tools:   make(map[string]map[string]*model.MCPTool),
	}
}

func (m *mockMCPProvider) addServer(id string, server *model.MCPServer) {
	server.Id = &id
	m.servers[id] = server
	if m.tools[id] == nil {
		m.tools[id] = make(map[string]*model.MCPTool)
	}
}

func (m *mockMCPProvider) addTool(serverID string, toolName string, tool *model.MCPTool) {
	if m.tools[serverID] == nil {
		m.tools[serverID] = make(map[string]*model.MCPTool)
	}
	tool.Name = toolName
	m.tools[serverID][toolName] = tool
}

func (m *mockMCPProvider) ListMCPServers(ctx context.Context, params catalog.ListMCPServersParams) (model.MCPServerList, error) {
	m.capturedParams = &params
	if m.shouldErrorOnListServers {
		return model.MCPServerList{}, fmt.Errorf("mock error in ListMCPServers")
	}

	if params.NamedQuery != "" {
		if err, ok := m.namedQueryErr[params.NamedQuery]; ok {
			return model.MCPServerList{}, err
		}
	}

	var items []model.MCPServer
	for _, server := range m.servers {
		// Simple filtering for tests
		if params.FilterQuery != "" {
			if params.FilterQuery == "provider = 'OpenAI'" && *server.Provider != "OpenAI" {
				continue
			}
			if params.FilterQuery == "name = 'nonexistent'" {
				continue // skip all for this test case
			}
		}

		// Include tools if requested
		if params.IncludeTools {
			serverID := *server.Id
			tools := make([]model.MCPTool, 0)
			for _, tool := range m.tools[serverID] {
				tools = append(tools, *tool)
			}
			serverCopy := *server
			serverCopy.Tools = tools
			serverCopy.ToolCount = int32(len(tools))
			items = append(items, serverCopy)
		} else {
			items = append(items, *server)
		}
	}

	// Simple pagination for tests
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	return model.MCPServerList{
		Items:         items,
		Size:          int32(len(items)),
		PageSize:      pageSize,
		NextPageToken: "",
	}, nil
}

func (m *mockMCPProvider) GetMCPServer(ctx context.Context, serverID string, includeTools bool, toolLimit int32) (*model.MCPServer, error) {
	if m.shouldErrorOnGetServer {
		return nil, fmt.Errorf("mock error in GetMCPServer")
	}

	server, exists := m.servers[serverID]
	if !exists {
		return nil, fmt.Errorf("server not found with ID %s: %w", serverID, api.ErrNotFound)
	}

	serverCopy := *server
	if includeTools {
		tools := make([]model.MCPTool, 0)
		for _, tool := range m.tools[serverID] {
			tools = append(tools, *tool)
		}
		serverCopy.Tools = tools
		serverCopy.ToolCount = int32(len(tools))
	}

	return &serverCopy, nil
}

func (m *mockMCPProvider) ListMCPServerTools(ctx context.Context, serverID string, params catalog.ListMCPServerToolsParams) (model.MCPToolsList, error) {
	if m.shouldErrorOnListTools {
		return model.MCPToolsList{}, fmt.Errorf("mock error in ListMCPServerTools")
	}

	if _, exists := m.servers[serverID]; !exists {
		return model.MCPToolsList{}, fmt.Errorf("server not found with ID %s: %w", serverID, api.ErrNotFound)
	}

	var items []model.MCPTool
	for _, tool := range m.tools[serverID] {
		// Simple filtering for tests
		if params.FilterQuery != "" {
			if params.FilterQuery == "accessType = 'read_only'" && tool.AccessType != "read_only" {
				continue
			}
		}
		items = append(items, *tool)
	}

	return model.MCPToolsList{
		Items:         items,
		Size:          int32(len(items)),
		PageSize:      params.PageSize,
		NextPageToken: "",
	}, nil
}

func (m *mockMCPProvider) GetFilterOptions(ctx context.Context) (*model.FilterOptionsList, error) {
	if m.shouldErrorOnGetFilterOptions {
		return nil, fmt.Errorf("mock error in GetFilterOptions")
	}
	if m.filterOptions != nil {
		return m.filterOptions, nil
	}
	return &model.FilterOptionsList{}, nil
}

func (m *mockMCPProvider) GetMCPServerTool(ctx context.Context, serverID string, toolName string) (*model.MCPTool, error) {
	if m.shouldErrorOnGetTool {
		return nil, fmt.Errorf("mock error in GetMCPServerTool")
	}

	if _, exists := m.servers[serverID]; !exists {
		return nil, fmt.Errorf("server not found with ID %s: %w", serverID, api.ErrNotFound)
	}

	tool, exists := m.tools[serverID][toolName]
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found in server %s: %w", toolName, serverID, api.ErrNotFound)
	}

	return tool, nil
}

func timeToStringPointer(t time.Time) *string {
	s := strconv.FormatInt(t.UnixMilli(), 10)
	return &s
}

func TestFindMCPServers(t *testing.T) {
	time1 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	time2 := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

	openAIServer := &model.MCPServer{
		Name:                     "openai-assistant",
		DisplayName:              new("OpenAI Assistant"),
		Description:              new("OpenAI Assistant MCP server"),
		Provider:                 new("OpenAI"),
		Version:                  new("1.2.0"),
		CreateTimeSinceEpoch:     timeToStringPointer(time1),
		LastUpdateTimeSinceEpoch: timeToStringPointer(time1),
		ToolCount:                0,
	}

	githubServer := &model.MCPServer{
		Name:                     "github-integration",
		DisplayName:              new("GitHub Integration"),
		Description:              new("GitHub MCP server"),
		Provider:                 new("GitHub Inc"),
		Version:                  new("2.1.3"),
		CreateTimeSinceEpoch:     timeToStringPointer(time2),
		LastUpdateTimeSinceEpoch: timeToStringPointer(time2),
		ToolCount:                0,
	}

	testCases := []struct {
		name               string
		mockSetup          func(*mockMCPProvider)
		sourceLabel        []string
		filterQuery        string
		namedQuery         string
		includeTools       bool
		toolLimit          int32
		pageSize           string
		orderBy            model.OrderByField
		sortOrder          model.SortOrder
		nextPageToken      string
		expectedStatus     int
		expectedServerList *model.MCPServerList
		expectError        bool
	}{
		{
			name: "Successful query with no filters",
			mockSetup: func(provider *mockMCPProvider) {
				provider.addServer("1", openAIServer)
				provider.addServer("2", githubServer)
			},
			filterQuery:    "",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_NAME,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedServerList: &model.MCPServerList{
				Items: []model.MCPServer{
					func() model.MCPServer { s := *openAIServer; s.Id = new("1"); return s }(),
					func() model.MCPServer { s := *githubServer; s.Id = new("2"); return s }(),
				},
				Size:          2,
				PageSize:      10,
				NextPageToken: "",
			},
		},
		{
			name: "Filter by provider",
			mockSetup: func(provider *mockMCPProvider) {
				provider.addServer("1", openAIServer)
				provider.addServer("2", githubServer)
			},
			filterQuery:    "provider = 'OpenAI'",
			pageSize:       "10",
			expectedStatus: http.StatusOK,
			expectedServerList: &model.MCPServerList{
				Items: []model.MCPServer{
					func() model.MCPServer { s := *openAIServer; s.Id = new("1"); return s }(),
				},
				Size:          1,
				PageSize:      10,
				NextPageToken: "",
			},
		},
		{
			name: "Include tools",
			mockSetup: func(provider *mockMCPProvider) {
				provider.addServer("1", openAIServer)
				provider.addTool("1", "chat_completion", &model.MCPTool{
					AccessType:  "read_only",
					Description: new("Generate chat completions"),
				})
			},
			includeTools:   true,
			pageSize:       "10",
			expectedStatus: http.StatusOK,
			expectedServerList: &model.MCPServerList{
				Items: []model.MCPServer{
					func() model.MCPServer {
						s := *openAIServer
						s.Id = new("1")
						s.Tools = []model.MCPTool{{
							Name:        "chat_completion",
							AccessType:  "read_only",
							Description: new("Generate chat completions"),
						}}
						s.ToolCount = 1
						return s
					}(),
				},
				Size:          1,
				PageSize:      10,
				NextPageToken: "",
			},
		},
		{
			name: "No results found",
			mockSetup: func(provider *mockMCPProvider) {
				provider.addServer("1", openAIServer)
			},
			filterQuery:    "name = 'nonexistent'",
			pageSize:       "10",
			expectedStatus: http.StatusOK,
			expectedServerList: &model.MCPServerList{
				Items:         []model.MCPServer{},
				Size:          0,
				PageSize:      10,
				NextPageToken: "",
			},
		},
		{
			name: "Invalid pageSize",
			mockSetup: func(provider *mockMCPProvider) {
				provider.addServer("1", openAIServer)
			},
			pageSize:       "invalid",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "Provider error",
			mockSetup: func(provider *mockMCPProvider) {
				provider.shouldErrorOnListServers = true
			},
			pageSize:       "10",
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name: "Unknown named query returns bad request",
			mockSetup: func(provider *mockMCPProvider) {
				provider.namedQueryErr = map[string]error{
					"unknown_query": fmt.Errorf("unknown named query %q: %w", "unknown_query", api.ErrBadRequest),
				}
			},
			namedQuery:     "unknown_query",
			pageSize:       "10",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "Known named query is forwarded to provider",
			mockSetup: func(provider *mockMCPProvider) {
				provider.addServer("1", openAIServer)
				// production_ready is a known query — no error entry means success
				provider.namedQueryErr = map[string]error{}
			},
			namedQuery:     "production_ready",
			pageSize:       "10",
			expectedStatus: http.StatusOK,
			expectedServerList: &model.MCPServerList{
				Items: []model.MCPServer{
					func() model.MCPServer { s := *openAIServer; s.Id = new("1"); return s }(),
				},
				Size:          1,
				PageSize:      10,
				NextPageToken: "",
			},
		},
		{
			name: "Named query combined with filterQuery both forwarded",
			mockSetup: func(provider *mockMCPProvider) {
				provider.addServer("1", openAIServer)
				provider.addServer("2", githubServer)
				provider.namedQueryErr = map[string]error{}
			},
			namedQuery:     "production_ready",
			filterQuery:    "provider = 'OpenAI'",
			pageSize:       "10",
			expectedStatus: http.StatusOK,
			expectedServerList: &model.MCPServerList{
				Items: []model.MCPServer{
					func() model.MCPServer { s := *openAIServer; s.Id = new("1"); return s }(),
				},
				Size:          1,
				PageSize:      10,
				NextPageToken: "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockProvider := newMockMCPProvider()
			tc.mockSetup(mockProvider)

			service := NewMCPCatalogServiceAPIService(mockProvider, nil)
			ctx := context.Background()

			result, err := service.FindMCPServers(
				ctx, "", "", tc.sourceLabel, tc.filterQuery, tc.namedQuery,
				tc.includeTools, tc.toolLimit, tc.pageSize, tc.orderBy, tc.sortOrder, tc.nextPageToken,
			)

			assert.Equal(t, tc.expectedStatus, result.Code)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.expectedServerList != nil {
					serverList, ok := result.Body.(model.MCPServerList)
					require.True(t, ok)
					assert.Equal(t, tc.expectedServerList.Size, serverList.Size)
					assert.Equal(t, tc.expectedServerList.PageSize, serverList.PageSize)
					assert.Equal(t, tc.expectedServerList.NextPageToken, serverList.NextPageToken)
					assert.Len(t, serverList.Items, int(tc.expectedServerList.Size))
				}
			}
		})
	}
}

func TestGetMCPServer(t *testing.T) {
	time1 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	openAIServer := &model.MCPServer{
		Name:                     "openai-assistant",
		Description:              new("OpenAI Assistant MCP server"),
		Provider:                 new("OpenAI"),
		Version:                  new("1.2.0"),
		CreateTimeSinceEpoch:     timeToStringPointer(time1),
		LastUpdateTimeSinceEpoch: timeToStringPointer(time1),
		ToolCount:                0,
	}

	testCases := []struct {
		name           string
		mockSetup      func(*mockMCPProvider)
		serverID       string
		includeTools   bool
		expectedStatus int
		expectedServer *model.MCPServer
		expectError    bool
	}{
		{
			name: "Get server without tools",
			mockSetup: func(provider *mockMCPProvider) {
				provider.addServer("1", openAIServer)
			},
			serverID:       "1",
			includeTools:   false,
			expectedStatus: http.StatusOK,
			expectedServer: func() *model.MCPServer {
				s := *openAIServer
				s.Id = new("1")
				return &s
			}(),
		},
		{
			name: "Get server with tools",
			mockSetup: func(provider *mockMCPProvider) {
				provider.addServer("1", openAIServer)
				provider.addTool("1", "chat_completion", &model.MCPTool{
					AccessType:  "read_only",
					Description: new("Generate chat completions"),
				})
			},
			serverID:       "1",
			includeTools:   true,
			expectedStatus: http.StatusOK,
			expectedServer: func() *model.MCPServer {
				s := *openAIServer
				s.Id = new("1")
				s.Tools = []model.MCPTool{{
					Name:        "chat_completion",
					AccessType:  "read_only",
					Description: new("Generate chat completions"),
				}}
				s.ToolCount = 1
				return &s
			}(),
		},
		{
			name: "Server not found",
			mockSetup: func(provider *mockMCPProvider) {
				provider.addServer("1", openAIServer)
			},
			serverID:       "999",
			includeTools:   false,
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name: "Provider error",
			mockSetup: func(provider *mockMCPProvider) {
				provider.shouldErrorOnGetServer = true
			},
			serverID:       "1",
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockProvider := newMockMCPProvider()
			tc.mockSetup(mockProvider)

			service := NewMCPCatalogServiceAPIService(mockProvider, nil)
			ctx := context.Background()

			result, err := service.GetMCPServer(ctx, tc.serverID, tc.includeTools, 0)

			assert.Equal(t, tc.expectedStatus, result.Code)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.expectedServer != nil {
					server, ok := result.Body.(*model.MCPServer)
					require.True(t, ok)
					assert.Equal(t, tc.expectedServer.Name, server.Name)
					assert.Equal(t, tc.expectedServer.ToolCount, server.ToolCount)
					if tc.includeTools {
						assert.Len(t, server.Tools, int(tc.expectedServer.ToolCount))
					}
				}
			}
		})
	}
}

func TestFindMCPServerTools(t *testing.T) {
	openAIServer := &model.MCPServer{Name: "openai-assistant"}
	chatTool := &model.MCPTool{
		AccessType:  "read_only",
		Description: new("Generate chat completions"),
	}
	embedTool := &model.MCPTool{
		AccessType:  "read_write",
		Description: new("Generate embeddings"),
	}

	testCases := []struct {
		name             string
		mockSetup        func(*mockMCPProvider)
		serverID         string
		filterQuery      string
		pageSize         string
		expectedStatus   int
		expectedToolList *model.MCPToolsList
		expectError      bool
	}{
		{
			name: "List all tools",
			mockSetup: func(provider *mockMCPProvider) {
				provider.addServer("1", openAIServer)
				provider.addTool("1", "chat_completion", chatTool)
				provider.addTool("1", "embeddings", embedTool)
			},
			serverID:       "1",
			pageSize:       "10",
			expectedStatus: http.StatusOK,
			expectedToolList: &model.MCPToolsList{
				Items: []model.MCPTool{
					{Name: "chat_completion", AccessType: "read_only", Description: new("Generate chat completions")},
					{Name: "embeddings", AccessType: "read_write", Description: new("Generate embeddings")},
				},
				Size:          2,
				PageSize:      10,
				NextPageToken: "",
			},
		},
		{
			name: "Filter by access type",
			mockSetup: func(provider *mockMCPProvider) {
				provider.addServer("1", openAIServer)
				provider.addTool("1", "chat_completion", chatTool)
				provider.addTool("1", "embeddings", embedTool)
			},
			serverID:       "1",
			filterQuery:    "accessType = 'read_only'",
			pageSize:       "10",
			expectedStatus: http.StatusOK,
			expectedToolList: &model.MCPToolsList{
				Items: []model.MCPTool{
					{Name: "chat_completion", AccessType: "read_only", Description: new("Generate chat completions")},
				},
				Size:          1,
				PageSize:      10,
				NextPageToken: "",
			},
		},
		{
			name: "Server not found",
			mockSetup: func(provider *mockMCPProvider) {
				provider.addServer("1", openAIServer)
			},
			serverID:       "999",
			pageSize:       "10",
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name: "Invalid pageSize",
			mockSetup: func(provider *mockMCPProvider) {
				provider.addServer("1", openAIServer)
			},
			serverID:       "1",
			pageSize:       "invalid",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "Provider error",
			mockSetup: func(provider *mockMCPProvider) {
				provider.shouldErrorOnListTools = true
			},
			serverID:       "1",
			pageSize:       "10",
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockProvider := newMockMCPProvider()
			tc.mockSetup(mockProvider)

			service := NewMCPCatalogServiceAPIService(mockProvider, nil)
			ctx := context.Background()

			result, err := service.FindMCPServerTools(
				ctx, tc.serverID, tc.filterQuery, tc.pageSize,
				model.ORDERBYFIELD_NAME, model.SORTORDER_ASC, "",
			)

			assert.Equal(t, tc.expectedStatus, result.Code)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.expectedToolList != nil {
					toolList, ok := result.Body.(model.MCPToolsList)
					require.True(t, ok)
					assert.Equal(t, tc.expectedToolList.Size, toolList.Size)
					assert.Equal(t, tc.expectedToolList.PageSize, toolList.PageSize)
					assert.Len(t, toolList.Items, int(tc.expectedToolList.Size))
				}
			}
		})
	}
}

func TestGetMCPServerTool(t *testing.T) {
	openAIServer := &model.MCPServer{Name: "openai-assistant"}
	chatTool := &model.MCPTool{
		AccessType:  "read_only",
		Description: new("Generate chat completions"),
	}

	testCases := []struct {
		name           string
		mockSetup      func(*mockMCPProvider)
		serverID       string
		toolName       string
		expectedStatus int
		expectedTool   *model.MCPTool
		expectError    bool
	}{
		{
			name: "Get existing tool",
			mockSetup: func(provider *mockMCPProvider) {
				provider.addServer("1", openAIServer)
				provider.addTool("1", "chat_completion", chatTool)
			},
			serverID:       "1",
			toolName:       "chat_completion",
			expectedStatus: http.StatusOK,
			expectedTool: &model.MCPTool{
				Name:        "chat_completion",
				AccessType:  "read_only",
				Description: new("Generate chat completions"),
			},
		},
		{
			name: "Tool not found",
			mockSetup: func(provider *mockMCPProvider) {
				provider.addServer("1", openAIServer)
			},
			serverID:       "1",
			toolName:       "nonexistent_tool",
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name: "Server not found",
			mockSetup: func(provider *mockMCPProvider) {
				provider.addServer("1", openAIServer)
			},
			serverID:       "999",
			toolName:       "chat_completion",
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name: "Provider error",
			mockSetup: func(provider *mockMCPProvider) {
				provider.shouldErrorOnGetTool = true
			},
			serverID:       "1",
			toolName:       "chat_completion",
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockProvider := newMockMCPProvider()
			tc.mockSetup(mockProvider)

			service := NewMCPCatalogServiceAPIService(mockProvider, nil)
			ctx := context.Background()

			result, err := service.GetMCPServerTool(ctx, tc.serverID, tc.toolName)

			assert.Equal(t, tc.expectedStatus, result.Code)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.expectedTool != nil {
					tool, ok := result.Body.(*model.MCPTool)
					require.True(t, ok)
					assert.Equal(t, tc.expectedTool.Name, tool.Name)
					assert.Equal(t, tc.expectedTool.AccessType, tool.AccessType)
					assert.Equal(t, tc.expectedTool.Description, tool.Description)
				}
			}
		})
	}
}

func TestFindMCPServersFilterOptions(t *testing.T) {
	ctx := context.Background()

	t.Run("provider returns empty FilterOptionsList", func(t *testing.T) {
		provider := newMockMCPProvider()
		service := NewMCPCatalogServiceAPIService(provider, nil)
		result, err := service.FindMCPServersFilterOptions(ctx)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, result.Code)
		body, ok := result.Body.(model.FilterOptionsList)
		require.True(t, ok)
		assert.Nil(t, body.NamedQueries)
		assert.Nil(t, body.Filters)
	})

	t.Run("provider returns named queries and filters", func(t *testing.T) {
		namedQueries := map[string]map[string]model.FieldFilter{
			"production_ready": {
				"maturity": {Operator: "=", Value: "production"},
			},
		}
		filters := map[string]model.FilterOption{
			"provider": {Type: "string", Values: []any{"OpenAI", "GitHub"}},
		}
		provider := newMockMCPProvider()
		provider.filterOptions = &model.FilterOptionsList{
			NamedQueries: &namedQueries,
			Filters:      &filters,
		}

		service := NewMCPCatalogServiceAPIService(provider, nil)
		result, err := service.FindMCPServersFilterOptions(ctx)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, result.Code)
		body, ok := result.Body.(model.FilterOptionsList)
		require.True(t, ok)
		require.NotNil(t, body.NamedQueries)
		nq := *body.NamedQueries
		require.Contains(t, nq, "production_ready")
		assert.Equal(t, model.FieldFilter{Operator: "=", Value: "production"}, nq["production_ready"]["maturity"])
		require.NotNil(t, body.Filters)
		assert.Contains(t, *body.Filters, "provider")
	})

	t.Run("provider error returns internal server error", func(t *testing.T) {
		provider := newMockMCPProvider()
		provider.shouldErrorOnGetFilterOptions = true

		service := NewMCPCatalogServiceAPIService(provider, nil)
		result, err := service.FindMCPServersFilterOptions(ctx)

		assert.Error(t, err)
		assert.Equal(t, http.StatusInternalServerError, result.Code)
	})
}

func makeMCPSources(sources map[string]basecatalog.MCPSource) *catalog.MCPSourceCollection {
	coll := mcpcatalog.NewMCPSourceCollection()
	_ = coll.Merge("test", sources)
	return coll
}

func TestFindMCPServers_SourceLabel(t *testing.T) {
	openAIServer := &model.MCPServer{
		Name:     "openai-assistant",
		Provider: new("OpenAI"),
	}

	t.Run("matching label passes source ID to provider", func(t *testing.T) {
		mcpSources := makeMCPSources(map[string]basecatalog.MCPSource{
			"src-openai": {ID: "src-openai", Labels: []string{"openai"}, Enabled: new(true)},
		})

		mockProvider := newMockMCPProvider()
		mockProvider.addServer("src-openai", openAIServer)

		service := NewMCPCatalogServiceAPIService(mockProvider, mcpSources)
		result, err := service.FindMCPServers(context.Background(), "", "", []string{"openai"}, "", "", false, 0, "10", model.ORDERBYFIELD_NAME, model.SORTORDER_ASC, "")

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, result.Code)
		require.NotNil(t, mockProvider.capturedParams)
		assert.Equal(t, []string{"src-openai"}, mockProvider.capturedParams.SourceIDs)
	})

	t.Run("non-matching label returns empty list without calling provider", func(t *testing.T) {
		mcpSources := makeMCPSources(map[string]basecatalog.MCPSource{
			"src-openai": {ID: "src-openai", Labels: []string{"openai"}, Enabled: new(true)},
		})

		mockProvider := newMockMCPProvider()
		mockProvider.addServer("src-openai", openAIServer)

		service := NewMCPCatalogServiceAPIService(mockProvider, mcpSources)
		result, err := service.FindMCPServers(context.Background(), "", "", []string{"no-match"}, "", "", false, 0, "10", model.ORDERBYFIELD_NAME, model.SORTORDER_ASC, "")

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, result.Code)
		serverList, ok := result.Body.(model.MCPServerList)
		require.True(t, ok)
		assert.Empty(t, serverList.Items)
		assert.Nil(t, mockProvider.capturedParams)
	})

	t.Run("empty sourceLabel string is treated as no filter", func(t *testing.T) {
		mcpSources := makeMCPSources(map[string]basecatalog.MCPSource{
			"src-openai": {ID: "src-openai", Labels: []string{"openai"}, Enabled: new(true)},
		})

		mockProvider := newMockMCPProvider()
		mockProvider.addServer("1", openAIServer)

		service := NewMCPCatalogServiceAPIService(mockProvider, mcpSources)
		result, err := service.FindMCPServers(context.Background(), "", "", []string{""}, "", "", false, 0, "10", model.ORDERBYFIELD_NAME, model.SORTORDER_ASC, "")

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, result.Code)
		serverList, ok := result.Body.(model.MCPServerList)
		require.True(t, ok)
		// No filter applied, provider returns all servers
		assert.Equal(t, int32(1), serverList.Size)
	})

	t.Run("nil mcpSources ignores sourceLabel", func(t *testing.T) {
		mockProvider := newMockMCPProvider()
		mockProvider.addServer("1", openAIServer)

		service := NewMCPCatalogServiceAPIService(mockProvider, nil)
		result, err := service.FindMCPServers(context.Background(), "", "", []string{"openai"}, "", "", false, 0, "10", model.ORDERBYFIELD_NAME, model.SORTORDER_ASC, "")

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, result.Code)
	})
}
