package mcpcatalog

import (
	"context"
	"fmt"
	"strings"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog/models"
	"github.com/kubeflow/hub/catalog/internal/converter"
	sharedmodels "github.com/kubeflow/hub/catalog/internal/db/models"
	openapi "github.com/kubeflow/hub/catalog/pkg/openapi"
	"github.com/kubeflow/hub/internal/platform/apiutils"
	"github.com/kubeflow/hub/pkg/api"
)

// NamedQueryResolver resolves a named query name to its field filters.
// Returns (filters, true) if found, or (nil, false) if unknown.
type NamedQueryResolver func(name string) (map[string]basecatalog.FieldFilter, bool)

type dbMCPCatalogImpl struct {
	mcpServerRepo             models.MCPServerRepository
	mcpServerToolRepo         models.MCPServerToolRepository
	resolveNamedQuery         NamedQueryResolver
	propertyOptionsRepository sharedmodels.PropertyOptionsRepository
	mcpSources                *MCPSourceCollection
}

func NewDBMCPCatalog(services Services, mcpSources *MCPSourceCollection, resolver NamedQueryResolver) MCPCatalogProvider {
	return &dbMCPCatalogImpl{
		mcpServerRepo:             services.MCPServerRepository,
		mcpServerToolRepo:         services.MCPServerToolRepository,
		resolveNamedQuery:         resolver,
		propertyOptionsRepository: services.PropertyOptionsRepository,
		mcpSources:                mcpSources,
	}
}

func (d *dbMCPCatalogImpl) GetFilterOptions(ctx context.Context) (*openapi.FilterOptionsList, error) {
	mcpServerTypeID := d.mcpServerRepo.GetTypeID()

	contextProperties, err := d.propertyOptionsRepository.List(sharedmodels.ContextPropertyOptionType, mcpServerTypeID)
	if err != nil {
		return nil, err
	}

	options := make(map[string]openapi.FilterOption, len(contextProperties))

	for _, prop := range contextProperties {
		switch prop.Name {
		case "artifacts", "base_name", "documentationUrl", "logo", "repositoryUrl", "sourceCode", "source_id", "tools", "version":
			continue
		}

		option := basecatalog.DbPropToAPIOption(prop)
		if option != nil {
			options[prop.FullName("")] = *option
		}
	}

	var namedQueriesPtr *map[string]map[string]openapi.FieldFilter
	if d.mcpSources != nil {
		namedQueriesPtr = basecatalog.ConvertNamedQueries(d.mcpSources.GetNamedQueries(), options)
	}

	return &openapi.FilterOptionsList{
		Filters:      &options,
		NamedQueries: namedQueriesPtr,
	}, nil
}

func (d *dbMCPCatalogImpl) ListMCPServers(ctx context.Context, params ListMCPServersParams) (openapi.MCPServerList, error) {
	filterQuery := params.FilterQuery

	// Resolve named query server-side and merge with any user-provided filterQuery.
	if params.NamedQuery != "" {
		if d.resolveNamedQuery == nil {
			return openapi.MCPServerList{}, fmt.Errorf("named query %q is not supported: no resolver configured: %w", params.NamedQuery, api.ErrBadRequest)
		}
		filters, found := d.resolveNamedQuery(params.NamedQuery)
		if !found {
			return openapi.MCPServerList{}, fmt.Errorf("unknown named query %q: %w", params.NamedQuery, api.ErrBadRequest)
		}
		resolved, err := basecatalog.FieldFiltersToFilterQuery(filters)
		if err != nil {
			return openapi.MCPServerList{}, fmt.Errorf("named query %q has invalid filter: %w", params.NamedQuery, err)
		}
		filterQuery = mergeFilterQueries(filterQuery, resolved)
	}

	// Build MCPServerListOptions from params
	listOptions := models.MCPServerListOptions{
		Query:       &params.Query,
		FilterQuery: &filterQuery,
	}

	if params.Name != "" {
		listOptions.Name = &params.Name
	}

	if len(params.SourceIDs) > 0 {
		listOptions.SourceIDs = &params.SourceIDs
	}

	// Set pagination
	orderBy := strings.ToUpper(string(params.OrderBy))
	sortOrder := strings.ToUpper(string(params.SortOrder))
	listOptions.Pagination.PageSize = &params.PageSize
	if orderBy != "" {
		listOptions.Pagination.OrderBy = &orderBy
	}
	if sortOrder != "" {
		listOptions.Pagination.SortOrder = &sortOrder
	}
	if params.NextPageToken != nil {
		listOptions.Pagination.NextPageToken = params.NextPageToken
	}

	// Call repository
	serversList, err := d.mcpServerRepo.List(listOptions)
	if err != nil {
		return openapi.MCPServerList{}, err
	}

	// Collect all server IDs and get tool counts in a single batch query
	serverIDs := make([]int32, len(serversList.Items))
	for i, dbServer := range serversList.Items {
		serverIDs[i] = *dbServer.GetID()
	}
	toolCounts, err := d.mcpServerToolRepo.CountByParentIDs(serverIDs)
	if err != nil {
		return openapi.MCPServerList{}, fmt.Errorf("error counting tools: %w", err)
	}

	// Convert to OpenAPI models
	apiServers := make([]openapi.MCPServer, 0) // Initialize as empty slice, not nil
	for _, dbServer := range serversList.Items {
		var apiServer *openapi.MCPServer

		if params.IncludeTools {
			// Load tools and convert with full tool data
			toolOptions := models.MCPServerToolListOptions{
				ParentID: *dbServer.GetID(),
			}
			if params.ToolLimit > 0 {
				toolOptions.Pagination.PageSize = &params.ToolLimit
			}

			tools, err := d.mcpServerToolRepo.List(toolOptions)
			if err != nil {
				return openapi.MCPServerList{}, fmt.Errorf("error loading tools for server %d: %w", *dbServer.GetID(), err)
			}

			apiServer = converter.ConvertDbMCPServerWithToolsToOpenapi(dbServer, tools.Items)
		} else {
			apiServer = converter.ConvertDbMCPServerToOpenapi(dbServer)
		}
		apiServer.ToolCount = toolCounts[*dbServer.GetID()]

		apiServers = append(apiServers, *apiServer)
	}

	return openapi.MCPServerList{
		Items:         apiServers,
		Size:          int32(len(apiServers)),
		PageSize:      params.PageSize,
		NextPageToken: serversList.NextPageToken,
	}, nil
}

// mergeFilterQueries combines two filterQuery strings with AND.
// Empty strings are ignored. If both are non-empty, each is wrapped in parentheses
// to preserve operator precedence.
func mergeFilterQueries(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	default:
		return fmt.Sprintf("(%s) AND (%s)", a, b)
	}
}

func (d *dbMCPCatalogImpl) GetMCPServer(ctx context.Context, serverID string, includeTools bool, toolLimit int32) (*openapi.MCPServer, error) {
	id, err := apiutils.ValidateIDAsInt32(serverID, "server")
	if err != nil {
		return nil, fmt.Errorf("invalid server ID '%s': %w", serverID, api.ErrBadRequest)
	}

	// Get server by ID
	dbServer, err := d.mcpServerRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("server not found with ID %s: %w", serverID, api.ErrNotFound)
	}

	// Get the accurate total tool count
	toolCounts, err := d.mcpServerToolRepo.CountByParentIDs([]int32{*dbServer.GetID()})
	if err != nil {
		return nil, fmt.Errorf("error counting tools for server %s: %w", serverID, err)
	}

	var apiServer *openapi.MCPServer
	if includeTools {
		toolOptions := models.MCPServerToolListOptions{
			ParentID: *dbServer.GetID(),
		}
		if toolLimit > 0 {
			toolOptions.Pagination.PageSize = &toolLimit
		}
		tools, err := d.mcpServerToolRepo.List(toolOptions)
		if err != nil {
			return nil, fmt.Errorf("error loading tools for server %s: %w", serverID, err)
		}
		apiServer = converter.ConvertDbMCPServerWithToolsToOpenapi(dbServer, tools.Items)
	} else {
		apiServer = converter.ConvertDbMCPServerToOpenapi(dbServer)
	}
	apiServer.ToolCount = toolCounts[*dbServer.GetID()]
	return apiServer, nil
}

func (d *dbMCPCatalogImpl) ListMCPServerTools(ctx context.Context, serverID string, params ListMCPServerToolsParams) (openapi.MCPToolsList, error) {
	id, err := apiutils.ValidateIDAsInt32(serverID, "server")
	if err != nil {
		return openapi.MCPToolsList{}, fmt.Errorf("invalid server ID '%s': %w", serverID, api.ErrBadRequest)
	}

	// Verify server exists
	_, err = d.mcpServerRepo.GetByID(id)
	if err != nil {
		return openapi.MCPToolsList{}, fmt.Errorf("server not found with ID %s: %w", serverID, api.ErrNotFound)
	}

	// Build MCPServerToolListOptions from params
	listOptions := models.MCPServerToolListOptions{
		ParentID: id,
	}

	// Set pagination
	toolsOrderBy := strings.ToUpper(string(params.OrderBy))
	toolsSortOrder := strings.ToUpper(string(params.SortOrder))
	listOptions.Pagination.PageSize = &params.PageSize
	listOptions.Pagination.OrderBy = &toolsOrderBy
	listOptions.Pagination.SortOrder = &toolsSortOrder
	if params.NextPageToken != nil {
		listOptions.Pagination.NextPageToken = params.NextPageToken
	}

	// Set filterQuery if provided
	if params.FilterQuery != "" {
		listOptions.Pagination.FilterQuery = &params.FilterQuery
	}

	// Call repository
	tools, err := d.mcpServerToolRepo.List(listOptions)
	if err != nil {
		return openapi.MCPToolsList{}, err
	}

	// Convert to OpenAPI models
	apiTools := make([]openapi.MCPTool, 0) // Initialize as empty slice, not nil
	for _, dbTool := range tools.Items {
		apiTool := converter.ConvertDbMCPToolToOpenapi(dbTool)
		if apiTool != nil {
			apiTools = append(apiTools, *apiTool)
		}
	}

	return openapi.MCPToolsList{
		Items:         apiTools,
		Size:          int32(len(apiTools)),
		PageSize:      params.PageSize,
		NextPageToken: tools.NextPageToken,
	}, nil
}

func (d *dbMCPCatalogImpl) GetMCPServerTool(ctx context.Context, serverID string, toolName string) (*openapi.MCPTool, error) {
	id, err := apiutils.ValidateIDAsInt32(serverID, "server")
	if err != nil {
		return nil, fmt.Errorf("invalid server ID '%s': %w", serverID, api.ErrBadRequest)
	}

	// Verify server exists
	_, err = d.mcpServerRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("server not found with ID %s: %w", serverID, api.ErrNotFound)
	}

	// Filter by name at the DB level. The DB stores qualified names (serverName@version:toolName)
	// but the API exposes only the unqualified name, so we match by suffix using LIKE.
	toolOptions := models.MCPServerToolListOptions{
		ParentID: id,
		ToolName: &toolName,
	}
	tools, err := d.mcpServerToolRepo.List(toolOptions)
	if err != nil {
		return nil, fmt.Errorf("error loading tools for server %s: %w", serverID, err)
	}

	if len(tools.Items) == 0 {
		return nil, fmt.Errorf("tool '%s' not found in server %s: %w", toolName, serverID, api.ErrNotFound)
	}

	apiTool := converter.ConvertDbMCPToolToOpenapi(tools.Items[0])
	return apiTool, nil
}
