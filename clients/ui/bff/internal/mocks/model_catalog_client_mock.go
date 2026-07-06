package mocks

import (
	"fmt"
	"log/slog"
	"math"
	"net/url"
	"strconv"
	"strings"

	"github.com/kubeflow/hub/ui/bff/internal/models"

	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"
	"github.com/stretchr/testify/mock"
)

type ModelCatalogClientMock struct {
	mock.Mock
}

func NewModelCatalogClientMock(logger *slog.Logger) (*ModelCatalogClientMock, error) {
	return &ModelCatalogClientMock{}, nil
}

func (m *ModelCatalogClientMock) GetAllCatalogModelsAcrossSources(client httpclient.HTTPClientInterface, pageValues url.Values) (*models.CatalogModelList, error) {
	allModels := GetCatalogModelMocks()
	var filteredModels []models.CatalogModel

	sourceId := pageValues.Get("source")
	sourceLabel := pageValues.Get("sourceLabel")
	query := pageValues.Get("q")

	if sourceId != "" {
		for _, model := range allModels {
			if model.SourceId != nil && *model.SourceId == sourceId {
				filteredModels = append(filteredModels, model)
			}
		}
	} else if sourceLabel != "" {
		allSources := GetCatalogSourceMocks()
		var matchingSourceIds []string

		if sourceLabel == "null" {
			for _, source := range allSources {
				if source.Enabled != nil && !*source.Enabled {
					continue
				}
				if len(source.Labels) == 0 {
					matchingSourceIds = append(matchingSourceIds, source.Id)
				}
			}
		} else {
			for _, source := range allSources {
				for _, label := range source.Labels {
					if label == sourceLabel {
						matchingSourceIds = append(matchingSourceIds, source.Id)
						break
					}
				}
			}
		}

		for _, model := range allModels {
			if model.SourceId != nil {
				for _, sid := range matchingSourceIds {
					if *model.SourceId == sid {
						filteredModels = append(filteredModels, model)
						break
					}
				}
			}
		}
	} else {
		filteredModels = allModels
	}

	if query != "" {
		var queryFilteredModels []models.CatalogModel
		queryLower := strings.ToLower(query)

		for _, model := range filteredModels {
			matchFound := false

			// Check name
			if strings.Contains(strings.ToLower(model.Name), queryLower) {
				matchFound = true
			}

			// Check description
			if !matchFound && model.Description != nil && strings.Contains(strings.ToLower(*model.Description), queryLower) {
				matchFound = true
			}

			// Check provider
			if !matchFound && model.Provider != nil && strings.Contains(strings.ToLower(*model.Provider), queryLower) {
				matchFound = true
			}

			if matchFound {
				queryFilteredModels = append(queryFilteredModels, model)
			}
		}

		filteredModels = queryFilteredModels
	}

	pageSizeStr := pageValues.Get("pageSize")
	pageSize := 10 // default
	if pageSizeStr != "" {
		if parsed, err := strconv.Atoi(pageSizeStr); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}

	pageTokenStr := pageValues.Get("nextPageToken")
	startIndex := 0
	if pageTokenStr != "" {
		if parsed, err := strconv.Atoi(pageTokenStr); err == nil && parsed > 0 {
			startIndex = parsed
		}
	}

	totalSize := len(filteredModels)
	endIndex := startIndex + pageSize
	if endIndex > totalSize {
		endIndex = totalSize
	}

	var pagedModels []models.CatalogModel
	if startIndex < totalSize {
		pagedModels = filteredModels[startIndex:endIndex]
	} else {
		pagedModels = []models.CatalogModel{}
	}

	var nextPageToken string
	if endIndex < totalSize {
		nextPageToken = strconv.Itoa(endIndex)
	}

	size := len(pagedModels)
	if size > math.MaxInt32 {
		size = math.MaxInt32
	}
	ps := pageSize
	if ps > math.MaxInt32 {
		ps = math.MaxInt32
	}

	catalogModelList := models.CatalogModelList{
		Items:         pagedModels,
		Size:          int32(size),
		PageSize:      int32(ps),
		NextPageToken: nextPageToken,
	}

	return &catalogModelList, nil

}

func (m *ModelCatalogClientMock) GetCatalogSourceModel(client httpclient.HTTPClientInterface, sourceId string, modelName string) (*models.CatalogModel, error) {
	allModels := GetCatalogModelMocks()

	decodedModelName, err := url.QueryUnescape(modelName)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modelName: %w", err)
	}

	decodedModelName = strings.TrimPrefix(decodedModelName, "/")

	for _, model := range allModels {
		if model.SourceId != nil && *model.SourceId == sourceId && model.Name == decodedModelName {
			return &model, nil
		}
	}

	return nil, fmt.Errorf("catalog model not found for sourceId: %s, modelName: %s", sourceId, decodedModelName)
}

func (m *ModelCatalogClientMock) GetAllCatalogSources(client httpclient.HTTPClientInterface, pageValues url.Values) (*models.CatalogSourceList, error) {
	var allMockSources models.CatalogSourceList
	assetType := pageValues.Get("assetType")

	switch assetType {
	case "mcp_servers":
		allMockSources = GetMcpServerCatalogSourceListMock()
	case "agents":
		allMockSources = GetAgentCatalogSourceListMock()
	default:
		allMockSources = GetCatalogSourceListMock()
	}
	var filteredMockSources []models.CatalogSource

	name := pageValues.Get("name")

	if name != "" {
		nameFilterLower := strings.ToLower(name)
		for _, source := range allMockSources.Items {
			if strings.ToLower(source.Id) == nameFilterLower || strings.ToLower(source.Name) == nameFilterLower {
				filteredMockSources = append(filteredMockSources, source)
			}
		}
	} else {
		filteredMockSources = allMockSources.Items
	}
	catalogSourceList := models.CatalogSourceList{
		Items:         filteredMockSources,
		PageSize:      int32(10),
		NextPageToken: "",
		Size:          int32(len(filteredMockSources)),
	}

	return &catalogSourceList, nil
}

func (m *ModelCatalogClientMock) GetCatalogSourceModelArtifacts(client httpclient.HTTPClientInterface, sourceId string, modelName string, pageValues url.Values) (*models.CatalogModelArtifactList, error) {
	var allMockModelArtifacts models.CatalogModelArtifactList

	if sourceId == "sample-source" && (modelName == "repo1%2Fgranite-8b-code-instruct" || modelName == "repo1%2Fgranite-8b-code-instruct-quantized.w4a16") {
		performanceArtifacts := GetCatalogPerformanceMetricsArtifactListMock(4)
		accuracyArtifacts := GetCatalogAccuracyMetricsArtifactListMock()
		securityArtifacts := GetCatalogSecurityMetricsArtifactListMock()
		modelArtifacts := GetCatalogModelArtifactListMock()
		combinedItems := append(performanceArtifacts.Items, accuracyArtifacts.Items...)
		combinedItems = append(combinedItems, securityArtifacts.Items...)
		combinedItems = append(combinedItems, modelArtifacts.Items...)
		allMockModelArtifacts = models.CatalogModelArtifactList{
			Items:         combinedItems,
			Size:          int32(len(combinedItems)),
			PageSize:      performanceArtifacts.PageSize,
			NextPageToken: "",
		}
	} else if sourceId == "sample-source" && modelName == "repo1%2Fgranite-7b-instruct" {
		accuracyArtifacts := GetCatalogAccuracyMetricsArtifactListMock()
		securityArtifacts := GetCatalogSecurityMetricsArtifactListMock()
		modelArtifacts := GetCatalogModelArtifactListMock()
		combinedItems := append(accuracyArtifacts.Items, securityArtifacts.Items...)
		combinedItems = append(combinedItems, modelArtifacts.Items...)
		allMockModelArtifacts = models.CatalogModelArtifactList{
			Items:         combinedItems,
			Size:          int32(len(combinedItems)),
			PageSize:      accuracyArtifacts.PageSize,
			NextPageToken: "",
		}
	} else if sourceId == "sample-source" && (modelName == "repo1%2Fgranite-3b-code-base") {
		allMockModelArtifacts = GetCatalogModelArtifactListMock()
	} else {
		allMockModelArtifacts = GetCatalogModelArtifactListMock()
	}

	if filterQuery := pageValues.Get("filterQuery"); filterQuery != "" {
		allMockModelArtifacts = filterArtifactsByQuery(allMockModelArtifacts, filterQuery)
	}

	return &allMockModelArtifacts, nil
}

func filterArtifactsByQuery(list models.CatalogModelArtifactList, filterQuery string) models.CatalogModelArtifactList {
	metricsTypeFilter := extractMetricsTypeFilter(filterQuery)
	if metricsTypeFilter == "" {
		return list
	}

	filtered := []models.CatalogArtifact{}
	for _, item := range list.Items {
		if item.MetricsType != nil && *item.MetricsType == metricsTypeFilter {
			filtered = append(filtered, item)
		}
	}

	return models.CatalogModelArtifactList{
		Items:         filtered,
		Size:          int32(len(filtered)),
		PageSize:      list.PageSize,
		NextPageToken: "",
	}
}

func extractMetricsTypeFilter(filterQuery string) string {
	prefix := "metricsType.string_value="
	idx := strings.Index(filterQuery, prefix)
	if idx == -1 {
		return ""
	}
	value := filterQuery[idx+len(prefix):]
	value = strings.Trim(value, "\"")
	if sepIdx := strings.IndexAny(value, "&, "); sepIdx != -1 {
		value = value[:sepIdx]
	}
	return value
}

func (m *ModelCatalogClientMock) GetCatalogModelPerformanceArtifacts(client httpclient.HTTPClientInterface, sourceId string, modelName string, pageValues url.Values) (*models.CatalogModelArtifactList, error) {
	allMockModelPerformanceArtifacts := GetCatalogPerformanceMetricsArtifactListMock(4)
	return &allMockModelPerformanceArtifacts, nil

}

func (m *ModelCatalogClientMock) GetCatalogModelSecurityArtifacts(client httpclient.HTTPClientInterface, sourceId string, modelName string, pageValues url.Values) (*models.CatalogModelArtifactList, error) {
	allMockModelSecurityArtifacts := GetCatalogSecurityMetricsArtifactListMock()
	return &allMockModelSecurityArtifacts, nil
}

func (m *ModelCatalogClientMock) GetCatalogFilterOptions(client httpclient.HTTPClientInterface) (*models.FilterOptionsList, error) {
	filterOptions := GetFilterOptionsListMock()

	return &filterOptions, nil
}

func (m *ModelCatalogClientMock) GetCatalogLabels(client httpclient.HTTPClientInterface, pageValues url.Values) (*models.CatalogLabelList, error) {
	assetType := pageValues.Get("assetType")

	var labels models.CatalogLabelList
	switch assetType {
	case "mcp_servers":
		labels = GetMcpServerCatalogLabelListMock()
	case "agents":
		labels = GetAgentCatalogLabelListMock()
	default:
		labels = GetCatalogLabelListMock()
	}
	return &labels, nil
}

func (m *ModelCatalogClientMock) CreateCatalogSourcePreview(client httpclient.HTTPClientInterface, sourcePreviewPayload models.CatalogSourcePreviewRequest, pageValues url.Values) (*models.CatalogSourcePreviewResult, error) {
	filterStatus := pageValues.Get("filterStatus")
	if filterStatus == "" {
		filterStatus = "all"
	}

	pageSize := 20
	if ps := pageValues.Get("pageSize"); ps != "" {
		_, _ = fmt.Sscanf(ps, "%d", &pageSize)
	}

	nextPageToken := pageValues.Get("nextPageToken")
	assetType := pageValues.Get("assetType")

	var catalogSourcePreview models.CatalogSourcePreviewResult
	if assetType == "mcp_servers" {
		catalogSourcePreview = CreateMcpCatalogSourcePreviewMockWithFilter(filterStatus, pageSize, nextPageToken)
	} else {
		catalogSourcePreview = CreateCatalogSourcePreviewMockWithFilter(filterStatus, pageSize, nextPageToken)
	}

	return &catalogSourcePreview, nil
}

const mcpSourceLabelOther = "null"

func (m *ModelCatalogClientMock) GetAllMcpServers(client httpclient.HTTPClientInterface, pageValues url.Values) (*models.McpServerList, error) {
	full := GetMcpServerListMock()
	sourceLabel := pageValues.Get("sourceLabel")

	var items []models.McpServer
	if sourceLabel != "" {
		sources := GetMcpServerCatalogSourceMocks()
		var matchingSourceIDs []string
		if sourceLabel == mcpSourceLabelOther {
			for _, source := range sources {
				if source.Enabled != nil && !*source.Enabled {
					continue
				}
				if len(source.Labels) == 0 {
					matchingSourceIDs = append(matchingSourceIDs, source.Id)
				}
			}
		} else {
			for _, source := range sources {
				for _, label := range source.Labels {
					if label == sourceLabel {
						matchingSourceIDs = append(matchingSourceIDs, source.Id)
						break
					}
				}
			}
		}
		for _, s := range full.Items {
			if s.SourceID == nil {
				if sourceLabel == mcpSourceLabelOther {
					items = append(items, s)
				}
				continue
			}
			for _, sid := range matchingSourceIDs {
				if *s.SourceID == sid {
					items = append(items, s)
					break
				}
			}
		}
	} else {
		items = full.Items
	}

	pageSizeStr := pageValues.Get("pageSize")
	pageSize := 10
	if pageSizeStr != "" {
		if parsed, err := strconv.Atoi(pageSizeStr); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}

	pageTokenStr := pageValues.Get("nextPageToken")
	startIndex := 0
	if pageTokenStr != "" {
		if parsed, err := strconv.Atoi(pageTokenStr); err == nil && parsed > 0 {
			startIndex = parsed
		}
	}

	totalSize := len(items)
	endIndex := startIndex + pageSize
	if endIndex > totalSize {
		endIndex = totalSize
	}

	var pagedItems []models.McpServer
	if startIndex < totalSize {
		pagedItems = items[startIndex:endIndex]
	} else {
		pagedItems = []models.McpServer{}
	}

	var nextPageToken string
	if endIndex < totalSize {
		nextPageToken = strconv.Itoa(endIndex)
	}

	size := len(pagedItems)
	if size > math.MaxInt32 {
		size = math.MaxInt32
	}
	ps := pageSize
	if ps > math.MaxInt32 {
		ps = math.MaxInt32
	}

	return &models.McpServerList{
		Items:         pagedItems,
		Size:          int32(size),
		PageSize:      int32(ps),
		NextPageToken: nextPageToken,
	}, nil
}

func (m *ModelCatalogClientMock) GetMcpServersFilter(client httpclient.HTTPClientInterface) (*models.FilterOptionsList, error) {
	mcpFilterOptions := GetMcpFilterOptionsListMock()

	return &mcpFilterOptions, nil
}

func (m *ModelCatalogClientMock) GetMcpServer(client httpclient.HTTPClientInterface, serverId string, pageValues url.Values) (*models.McpServer, error) {
	allMocks := GetMcpServerMocks()
	for i := range allMocks {
		if allMocks[i].ID == serverId {
			return &allMocks[i], nil
		}
	}
	return nil, fmt.Errorf("server id doesn't exist: %s", serverId)
}

func (m *ModelCatalogClientMock) GetMcpServersTools(client httpclient.HTTPClientInterface, serverId string) (*models.McpToolList, error) {
	mcpServerTools := GetMcpToolListMock()

	return &mcpServerTools, nil
}

func (m *ModelCatalogClientMock) GetAllAgents(client httpclient.HTTPClientInterface, pageValues url.Values) (*models.AgentList, error) {
	full := GetAgentListMock()
	sourceLabel := pageValues.Get("sourceLabel")

	var items []models.Agent
	if sourceLabel != "" {
		sources := GetAgentCatalogSourceMocks()
		var matchingSourceIDs []string
		if sourceLabel == "null" {
			for _, source := range sources {
				if source.Enabled != nil && !*source.Enabled {
					continue
				}
				if len(source.Labels) == 0 {
					matchingSourceIDs = append(matchingSourceIDs, source.Id)
				}
			}
		} else {
			for _, source := range sources {
				for _, label := range source.Labels {
					if label == sourceLabel {
						matchingSourceIDs = append(matchingSourceIDs, source.Id)
						break
					}
				}
			}
		}
		for _, agent := range full.Items {
			if agent.SourceID == nil {
				if sourceLabel == "null" {
					items = append(items, agent)
				}
				continue
			}
			for _, sid := range matchingSourceIDs {
				if *agent.SourceID == sid {
					items = append(items, agent)
					break
				}
			}
		}
	} else {
		items = full.Items
	}

	query := pageValues.Get("q")
	if query != "" {
		queryLower := strings.ToLower(query)
		var filtered []models.Agent
		for _, agent := range items {
			if strings.Contains(strings.ToLower(agent.Name), queryLower) {
				filtered = append(filtered, agent)
			} else if agent.Description != nil && strings.Contains(strings.ToLower(*agent.Description), queryLower) {
				filtered = append(filtered, agent)
			}
		}
		items = filtered
	}

	pageSizeStr := pageValues.Get("pageSize")
	pageSize := 10
	if pageSizeStr != "" {
		if parsed, err := strconv.Atoi(pageSizeStr); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}

	pageTokenStr := pageValues.Get("nextPageToken")
	startIndex := 0
	if pageTokenStr != "" {
		if parsed, err := strconv.Atoi(pageTokenStr); err == nil && parsed > 0 {
			startIndex = parsed
		}
	}

	totalSize := len(items)
	endIndex := startIndex + pageSize
	if endIndex > totalSize {
		endIndex = totalSize
	}

	var pagedItems []models.Agent
	if startIndex < totalSize {
		pagedItems = items[startIndex:endIndex]
	} else {
		pagedItems = []models.Agent{}
	}

	var nextPageToken string
	if endIndex < totalSize {
		nextPageToken = strconv.Itoa(endIndex)
	}

	size := len(pagedItems)
	if size > math.MaxInt32 {
		size = math.MaxInt32
	}
	ps := pageSize
	if ps > math.MaxInt32 {
		ps = math.MaxInt32
	}

	return &models.AgentList{
		Items:         pagedItems,
		Size:          int32(size),
		PageSize:      int32(ps),
		NextPageToken: nextPageToken,
	}, nil
}

func (m *ModelCatalogClientMock) GetAgentsFilter(client httpclient.HTTPClientInterface) (*models.FilterOptionsList, error) {
	agentFilterOptions := GetAgentFilterOptionsListMock()
	return &agentFilterOptions, nil
}

func (m *ModelCatalogClientMock) GetAgent(client httpclient.HTTPClientInterface, agentId string) (*models.Agent, error) {
	allMocks := GetAgentMocks()
	for i := range allMocks {
		if allMocks[i].ID == agentId {
			return &allMocks[i], nil
		}
	}
	return nil, &httpclient.HTTPError{
		StatusCode: 404,
		ErrorResponse: httpclient.ErrorResponse{
			Code:    "404",
			Message: fmt.Sprintf("agent not found: %s", agentId),
		},
	}
}
