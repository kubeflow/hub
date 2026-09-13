package v1

import (
	"context"
	"errors"
	"net/http"

	"github.com/kubeflow/hub/catalog/internal/catalog"
	model "github.com/kubeflow/hub/catalog/pkg/openapi"
	"github.com/kubeflow/hub/pkg/api"
)

// MCPCatalogServiceAPIService is a service that implements the logic for the MCPCatalogServiceAPIServicer
type MCPCatalogServiceAPIService struct {
	mcpProvider catalog.MCPProvider
	mcpSources  *catalog.MCPSourceCollection
}

var _ MCPCatalogServiceAPIServicer = &MCPCatalogServiceAPIService{}

// NewMCPCatalogServiceAPIService creates a default api service
func NewMCPCatalogServiceAPIService(mcpProvider catalog.MCPProvider, mcpSources *catalog.MCPSourceCollection) MCPCatalogServiceAPIServicer {
	return &MCPCatalogServiceAPIService{
		mcpProvider: mcpProvider,
		mcpSources:  mcpSources,
	}
}

// FindMCPServers - List MCP servers.
func (m *MCPCatalogServiceAPIService) FindMCPServers(ctx context.Context, name string, q string, sourceLabel []string, filterQuery string, namedQuery string, includeTools bool, toolLimit int32, pageSize string, orderBy model.OrderByField, sortOrder model.SortOrder, nextPageToken string) (ImplResponse, error) {
	if err := enumOrderBy.validate(string(orderBy)); err != nil {
		return ErrorResponse(http.StatusBadRequest, err), err
	}

	pageSizeInt, err := parsePaginationParams(pageSize, nextPageToken)
	if err != nil {
		return ErrorResponse(http.StatusBadRequest, err), err
	}

	// Clean up empty sourceLabel values
	if len(sourceLabel) == 1 && sourceLabel[0] == "" {
		sourceLabel = nil
	}

	// Convert sourceLabels to sourceIDs
	var sourceIDs []string
	if len(sourceLabel) > 0 && m.mcpSources != nil {
		sources := m.mcpSources.ByLabel(sourceLabel)
		if len(sources) == 0 {
			return Response(http.StatusOK, model.MCPServerList{
				Items:    []model.MCPServer{},
				PageSize: pageSizeInt,
			}), nil
		}
		sourceIDs = make([]string, len(sources))
		for i, source := range sources {
			sourceIDs[i] = source.ID
		}
	}

	// Convert parameters to internal format
	params := catalog.ListMCPServersParams{
		Name:          name,
		Query:         q,
		SourceIDs:     sourceIDs,
		FilterQuery:   filterQuery,
		NamedQuery:    namedQuery,
		IncludeTools:  includeTools,
		ToolLimit:     toolLimit,
		PageSize:      pageSizeInt,
		OrderBy:       orderBy,
		SortOrder:     sortOrder,
		NextPageToken: &nextPageToken,
	}

	servers, err := m.mcpProvider.ListMCPServers(ctx, params)
	if err != nil {
		return ErrorResponse(api.ErrToStatus(err), err), err
	}

	return Response(http.StatusOK, servers), nil
}

// FindMCPServersFilterOptions - Lists fields, values, and named queries that can be used in `filterQuery` on the list MCP servers endpoint.
func (m *MCPCatalogServiceAPIService) FindMCPServersFilterOptions(ctx context.Context) (ImplResponse, error) {
	filterOptions, err := m.mcpProvider.GetFilterOptions(ctx)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, err), err
	}
	return Response(http.StatusOK, *filterOptions), nil
}

// GetMCPServer - Get an `MCPServer`.
func (m *MCPCatalogServiceAPIService) GetMCPServer(ctx context.Context, serverID string, includeTools bool, toolLimit int32) (ImplResponse, error) {
	server, err := m.mcpProvider.GetMCPServer(ctx, serverID, includeTools, toolLimit)
	if err != nil {
		return ErrorResponse(api.ErrToStatus(err), err), err
	}

	if server == nil {
		return ErrorResponse(http.StatusNotFound, errors.New("server not found")), nil
	}

	return Response(http.StatusOK, server), nil
}

// FindMCPServerTools - List MCP server tools.
func (m *MCPCatalogServiceAPIService) FindMCPServerTools(ctx context.Context, serverID string, filterQuery string, pageSize string, orderBy model.OrderByField, sortOrder model.SortOrder, nextPageToken string) (ImplResponse, error) {
	if err := enumOrderBy.validate(string(orderBy)); err != nil {
		return ErrorResponse(http.StatusBadRequest, err), err
	}

	pageSizeInt, err := parsePaginationParams(pageSize, nextPageToken)
	if err != nil {
		return ErrorResponse(http.StatusBadRequest, err), err
	}

	// Convert parameters to internal format
	params := catalog.ListMCPServerToolsParams{
		FilterQuery:   filterQuery,
		PageSize:      pageSizeInt,
		OrderBy:       orderBy,
		SortOrder:     sortOrder,
		NextPageToken: &nextPageToken,
	}

	tools, err := m.mcpProvider.ListMCPServerTools(ctx, serverID, params)
	if err != nil {
		return ErrorResponse(api.ErrToStatus(err), err), err
	}

	return Response(http.StatusOK, tools), nil
}

// GetMCPServerTool - Get an `MCPTool` from an `MCPServer`.
func (m *MCPCatalogServiceAPIService) GetMCPServerTool(ctx context.Context, serverID string, toolName string) (ImplResponse, error) {
	tool, err := m.mcpProvider.GetMCPServerTool(ctx, serverID, toolName)
	if err != nil {
		return ErrorResponse(api.ErrToStatus(err), err), err
	}

	if tool == nil {
		return ErrorResponse(http.StatusNotFound, errors.New("tool not found")), nil
	}

	return Response(http.StatusOK, tool), nil
}
