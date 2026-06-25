package openapi

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/kubeflow/hub/catalog/internal/catalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog"
	"github.com/kubeflow/hub/catalog/internal/db/models"
	model "github.com/kubeflow/hub/catalog/pkg/openapi"
	mrmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/kubeflow/hub/pkg/api"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// ModelCatalogServiceAPIService is a service that implements the logic for the ModelCatalogServiceAPIServicer
// This service should implement the business logic for every endpoint for the ModelCatalogServiceAPI s.coreApi.
// Include any external packages or services that will be required by this service.
type ModelCatalogServiceAPIService struct {
	provider         catalog.APIProvider
	sources          *catalog.SourceCollection
	mcpSources       *catalog.MCPSourceCollection
	labels           *catalog.LabelCollection
	sourceRepository models.CatalogSourceRepository
}

// GetAllModelArtifacts retrieves all model artifacts for a given model from the specified source.
func (m *ModelCatalogServiceAPIService) GetAllModelArtifacts(ctx context.Context, sourceID string, modelName string, artifactType []model.ArtifactTypeQueryParam, artifactType2 []model.ArtifactTypeQueryParam, filterQuery string, pageSize string, orderBy string, sortOrder model.SortOrder, nextPageToken string) (ImplResponse, error) {
	// Handle multiple artifact_type parameters (snake case - deprecated, will be removed in future)
	for _, v := range artifactType2 {
		if v != "" {
			artifactType = append(artifactType, v)
		}
	}

	if newName, err := url.PathUnescape(modelName); err == nil {
		modelName = newName
	}

	pageSizeInt, err := parsePaginationParams(pageSize, nextPageToken)
	if err != nil {
		return ErrorResponse(http.StatusBadRequest, err), err
	}

	// Handle multiple artifact types
	var artifactTypesFilter []string

	if len(artifactType) > 0 {
		// Convert slice of ArtifactTypeQueryParam to slice of strings
		artifactTypesFilter = make([]string, len(artifactType))
		for i, at := range artifactType {
			artifactTypesFilter[i] = string(at)
		}
	}

	artifacts, err := m.provider.GetArtifacts(ctx, modelName, sourceID, catalog.ListArtifactsParams{
		FilterQuery:         filterQuery,
		ArtifactTypesFilter: artifactTypesFilter,
		PageSize:            pageSizeInt,
		OrderBy:             orderBy,
		SortOrder:           sortOrder,
		NextPageToken:       &nextPageToken,
	})
	if err != nil {
		statusCode := api.ErrToStatus(err)
		var errorMsg string
		if errors.Is(err, api.ErrBadRequest) {
			// Use the original error message which should be more specific
			errorMsg = err.Error()
		} else if errors.Is(err, api.ErrNotFound) {
			errorMsg = fmt.Sprintf("No model found '%s' in source '%s'", modelName, sourceID)
		} else {
			errorMsg = err.Error()
		}
		return ErrorResponse(statusCode, errors.New(errorMsg)), err
	}

	return Response(http.StatusOK, artifacts), nil
}

func (m *ModelCatalogServiceAPIService) GetAllModelPerformanceArtifacts(ctx context.Context, sourceID string, modelName string, targetRPS int32, recommendations bool, rpsProperty string, latencyProperty string, hardwareCountProperty string, hardwareTypeProperty string, filterQuery string, pageSize string, orderBy string, sortOrder model.SortOrder, nextPageToken string) (ImplResponse, error) {
	if newName, err := url.PathUnescape(modelName); err == nil {
		modelName = newName
	}

	pageSizeInt, err := parsePaginationParams(pageSize, nextPageToken)
	if err != nil {
		return ErrorResponse(http.StatusBadRequest, err), err
	}

	// Call the provider's GetPerformanceArtifacts method
	artifacts, err := m.provider.GetPerformanceArtifacts(ctx, modelName, sourceID, catalog.ListPerformanceArtifactsParams{
		FilterQuery:           filterQuery,
		PageSize:              pageSizeInt,
		OrderBy:               orderBy,
		SortOrder:             sortOrder,
		NextPageToken:         &nextPageToken,
		TargetRPS:             targetRPS,
		Recommendations:       recommendations,
		RPSProperty:           rpsProperty,
		LatencyProperty:       latencyProperty,
		HardwareCountProperty: hardwareCountProperty,
		HardwareTypeProperty:  hardwareTypeProperty,
	})
	if err != nil {
		statusCode := api.ErrToStatus(err)
		var errorMsg string
		if errors.Is(err, api.ErrBadRequest) {
			errorMsg = err.Error()
		} else if errors.Is(err, api.ErrNotFound) {
			errorMsg = fmt.Sprintf("No model found '%s' in source '%s'", modelName, sourceID)
		} else {
			errorMsg = err.Error()
		}
		return ErrorResponse(statusCode, errors.New(errorMsg)), err
	}

	return Response(http.StatusOK, artifacts), nil
}

func (m *ModelCatalogServiceAPIService) FindLabels(ctx context.Context, assetType model.CatalogAssetType, pageSize string, orderBy string, sortOrder model.SortOrder, nextPageToken string) (ImplResponse, error) {
	if assetType == "" {
		assetType = model.CATALOGASSETTYPE_MODELS
	} else if !assetType.IsValid() {
		err := fmt.Errorf("invalid value '%s' for assetType: valid values are %v", assetType, model.AllowedCatalogAssetTypeEnumValues)
		return ErrorResponse(http.StatusBadRequest, err), err
	}

	labels := m.labels.All()
	if len(labels) > math.MaxInt32 {
		err := errors.New("too many registered labels")
		return ErrorResponse(http.StatusInternalServerError, err), err
	}

	// Filter labels by assetType
	filtered := make([]map[string]any, 0, len(labels))
	for _, label := range labels {
		labelAssetType := model.CATALOGASSETTYPE_MODELS // default when not specified
		if at, ok := label["assetType"]; ok {
			if atStr, ok := at.(string); ok {
				labelAssetType = model.CatalogAssetType(atStr)
			}
		}
		if labelAssetType != assetType {
			continue
		}
		filtered = append(filtered, label)
	}

	// Wrap labels to make them sortable
	sortableLabels := make([]sortableLabel, len(filtered))
	for i, label := range filtered {
		sortableLabels[i] = sortableLabel{
			data:  label,
			index: i, // Keep original index for stable sort
			id:    generateLabelID(i),
		}
	}

	// Create paginator - use empty OrderByField since we don't use it for labels
	paginator, err := newPaginator[sortableLabel](pageSize, model.OrderByField(""), sortOrder, nextPageToken)
	if err != nil {
		return ErrorResponse(http.StatusBadRequest, err), err
	}

	// Create comparison function for labels using the string key
	cmpFunc := genLabelCmpFunc(orderBy, sortOrder)
	slices.SortStableFunc(sortableLabels, cmpFunc)

	// Paginate the sorted labels
	pagedSortableLabels, next := paginator.Paginate(sortableLabels)

	// Convert map[string]string to model.CatalogLabel
	pagedLabels := make([]model.CatalogLabel, len(pagedSortableLabels))
	for i, sl := range pagedSortableLabels {
		// Extract the "name" field (required)
		name, ok := sl.data["name"]
		if !ok || name == "" {
			err := fmt.Errorf("internal error: label at index %d missing required name field", i)
			return ErrorResponse(http.StatusInternalServerError, err), err
		}

		// Create CatalogLabel with name (which may be null)
		var label *model.CatalogLabel
		if nameStr, ok := name.(string); ok {
			label = model.NewCatalogLabel(*model.NewNullableString(&nameStr))
		} else {
			label = model.NewCatalogLabel(*model.NewNullableString(nil))
		}

		// Add all other properties to AdditionalProperties
		label.AdditionalProperties = make(map[string]any)
		for key, value := range sl.data {
			if key != "name" {
				label.AdditionalProperties[key] = value
			}
		}

		pagedLabels[i] = *label
	}

	res := model.CatalogLabelList{
		PageSize:      paginator.PageSize,
		Items:         pagedLabels,
		Size:          int32(len(pagedLabels)), // Number of items in current page, not total
		NextPageToken: next.Token(),
	}
	return Response(http.StatusOK, res), nil
}

func (m *ModelCatalogServiceAPIService) FindModels(ctx context.Context, recommended bool, targetRPS int32, latencyProperty string, rpsProperty string, hardwareCountProperty string, hardwareTypeProperty string, sourceIDs []string, q string, sourceLabels []string, filterQuery string, pageSize string, orderBy model.OrderByField, sortOrder model.SortOrder, nextPageToken string) (ImplResponse, error) {
	// Validate pageSize and nextPageToken up-front. The recommended path uses numeric
	// offset tokens; the non-recommended path uses base64-encoded DB cursors
	// validated inside parsePaginationParams.
	var pageSizeInt int32
	var err error
	if recommended || orderBy == model.ORDERBYFIELD_RECOMMENDED {
		pageSizeInt, err = parsePageSize(pageSize)
		if err != nil {
			return ErrorResponse(http.StatusBadRequest, err), err
		}
		if tokenErr := validateRecommendedNextPageToken(nextPageToken); tokenErr != nil {
			return ErrorResponse(http.StatusBadRequest, tokenErr), tokenErr
		}
	} else {
		pageSizeInt, err = parsePaginationParams(pageSize, nextPageToken)
		if err != nil {
			return ErrorResponse(http.StatusBadRequest, err), err
		}
	}

	if len(sourceIDs) == 1 && sourceIDs[0] == "" {
		sourceIDs = nil
	}
	if len(sourceLabels) == 1 && sourceLabels[0] == "" {
		sourceLabels = nil
	}

	if len(sourceIDs) > 0 && len(sourceLabels) > 0 {
		err := fmt.Errorf("source and sourceLabel cannot be used together")
		return ErrorResponse(http.StatusBadRequest, err), err
	}

	// Convert sourceLabels to sourceIDs
	if len(sourceIDs) == 0 && len(sourceLabels) > 0 {
		sources := m.sources.ByLabel(sourceLabels)
		if len(sources) == 0 {
			return Response(http.StatusOK, model.CatalogModelList{
				Items:    []model.CatalogModel{},
				PageSize: pageSizeInt,
			}), nil
		}
		sourceIDs = make([]string, len(sources))
		for i, source := range sources {
			sourceIDs[i] = source.Id
		}
	}

	// Handle recommended latency sorting (triggered by recommendations=true or orderBy=RECOMMENDED)
	if recommended || orderBy == model.ORDERBYFIELD_RECOMMENDED {
		// Build Pareto filtering parameters with defaults
		var targetRPSPtr *int32
		if targetRPS != 0 {
			targetRPSPtr = &targetRPS
		}

		latencyProp := latencyProperty
		if latencyProp == "" {
			latencyProp = "ttft_p90"
		}

		rpsProp := rpsProperty
		if rpsProp == "" {
			rpsProp = "requests_per_second"
		}

		hardwareCountProp := hardwareCountProperty
		if hardwareCountProp == "" {
			hardwareCountProp = "hardware_count"
		}

		hardwareTypeProp := hardwareTypeProperty
		if hardwareTypeProp == "" {
			hardwareTypeProp = "hardware_type"
		}

		paretoParams := modelcatalog.ParetoFilteringParams{
			TargetRPS:             targetRPSPtr,
			LatencyProperty:       latencyProp,
			RpsProperty:           rpsProp,
			HardwareCountProperty: hardwareCountProp,
			HardwareTypeProperty:  hardwareTypeProp,
		}

		pagination := mrmodels.Pagination{
			PageSize:      &pageSizeInt,
			NextPageToken: &nextPageToken,
			FilterQuery:   &filterQuery,
		}

		// Use recommended latency sorting
		models, err := m.provider.FindModelsWithRecommendedLatency(ctx, pagination, paretoParams, sourceIDs, q, string(sortOrder))
		if err != nil {
			return ErrorResponse(api.ErrToStatus(err), fmt.Errorf("failed to find models with recommended latency: %w", err)), err
		}

		return Response(http.StatusOK, *models), nil
	}

	// Existing logic for non-recommended sorting
	if orderBy == "" {
		orderBy = model.ORDERBYFIELD_NAME
	}

	listModelsParams := catalog.ListModelsParams{
		Query:         q,
		FilterQuery:   filterQuery,
		SourceIDs:     sourceIDs,
		PageSize:      pageSizeInt,
		OrderBy:       orderBy,
		SortOrder:     sortOrder,
		NextPageToken: &nextPageToken,
	}

	models, err := m.provider.ListModels(ctx, listModelsParams)
	if err != nil {
		return ErrorResponse(api.ErrToStatus(err), err), err
	}

	return Response(http.StatusOK, models), nil
}

func (m *ModelCatalogServiceAPIService) FindModelsFilterOptions(ctx context.Context) (ImplResponse, error) {
	filterOptions, err := m.provider.GetFilterOptions(ctx)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, err), err
	}

	return Response(http.StatusOK, filterOptions), nil
}

func (m *ModelCatalogServiceAPIService) GetModel(ctx context.Context, sourceID, modelName string) (ImplResponse, error) {
	if newName, err := url.PathUnescape(modelName); err == nil {
		modelName = newName
	}

	model, err := m.provider.GetModel(ctx, modelName, sourceID)
	if err != nil {
		statusCode := api.ErrToStatus(err)
		var errorMsg string
		if errors.Is(err, api.ErrNotFound) {
			errorMsg = fmt.Sprintf("No model found '%s' in source '%s'", modelName, sourceID)
		} else {
			errorMsg = err.Error()
		}
		return ErrorResponse(statusCode, errors.New(errorMsg)), err
	}

	if model == nil {
		return notFound("Unknown model or version"), nil
	}

	return Response(http.StatusOK, model), nil
}

func (m *ModelCatalogServiceAPIService) FindSources(ctx context.Context, name string, assetType model.CatalogAssetType, strPageSize string, orderBy model.OrderByField, sortOrder model.SortOrder, nextPageToken string) (ImplResponse, error) {
	// Collect all sources (model + MCP) as CatalogSource objects
	sources := m.sources.All()

	if m.mcpSources != nil {
		for id, mcpSrc := range m.mcpSources.AllSources() {
			sources[id] = mcpSourceToCatalogSource(mcpSrc)
		}
	}

	if len(sources) > math.MaxInt32 {
		err := errors.New("too many registered sources")
		return ErrorResponse(http.StatusInternalServerError, err), err
	}

	// Fetch status from database
	var statuses map[string]models.SourceStatus
	if m.sourceRepository != nil {
		var err error
		statuses, err = m.sourceRepository.GetAllStatuses()
		if err != nil {
			// Log error but continue - status is optional
			statuses = nil
		}
	}

	paginator, err := newPaginator[model.CatalogSource](strPageSize, orderBy, sortOrder, nextPageToken)
	if err != nil {
		return ErrorResponse(http.StatusBadRequest, err), err
	}

	if assetType == "" {
		assetType = model.CATALOGASSETTYPE_MODELS
	}

	items := make([]model.CatalogSource, 0, len(sources))

	name = strings.ToLower(name)

	for _, v := range sources {
		if !strings.Contains(strings.ToLower(v.Name), name) {
			continue
		}

		sourceAssetType := v.GetAssetType()
		if !v.HasAssetType() {
			sourceAssetType = model.CATALOGASSETTYPE_MODELS
		}
		if sourceAssetType != assetType {
			continue
		}

		// Merge status from database if available
		if statuses != nil {
			if status, ok := statuses[v.Id]; ok {
				if status.Status != "" {
					statusEnum := model.CatalogSourceStatus(status.Status)
					v.Status = &statusEnum
				}
				if status.Error != "" {
					v.Error = *model.NewNullableString(&status.Error)
				} else {
					v.Error = *model.NewNullableString(nil)
				}
			}
		}

		items = append(items, v)
	}

	cmpFunc, err := genCatalogCmpFunc(orderBy, sortOrder)
	if err != nil {
		return ErrorResponse(http.StatusBadRequest, err), err
	}
	slices.SortStableFunc(items, cmpFunc)

	pagedItems, next := paginator.Paginate(items)

	res := model.CatalogSourceList{
		PageSize:      paginator.PageSize,
		Items:         pagedItems,
		Size:          int32(len(pagedItems)), // Number of items in current page, not total
		NextPageToken: next.Token(),
	}
	return Response(http.StatusOK, res), nil
}

// mcpSourceToCatalogSource converts an internal MCPSource to the API CatalogSource type.
func mcpSourceToCatalogSource(src basecatalog.MCPSource) model.CatalogSource {
	cs := model.CatalogSource{
		Id:      src.ID,
		Name:    src.Name,
		Enabled: src.Enabled,
		Labels:  src.Labels,
	}
	if src.AssetType != nil {
		cs.AssetType = src.AssetType
	}
	return cs
}

func (m *ModelCatalogServiceAPIService) PreviewCatalogSource(ctx context.Context, configParam *os.File, pageSizeParam string, nextPageTokenParam string, filterStatusParam string, catalogDataParam *os.File) (ImplResponse, error) {
	if _, err := parsePageSize(pageSizeParam); err != nil {
		return ErrorResponse(http.StatusBadRequest, err), err
	}

	// Parse filterStatus (default: "all")
	filterStatus := "all"
	if filterStatusParam != "" {
		filterStatus = strings.ToLower(filterStatusParam)
		if filterStatus != "all" && filterStatus != "included" && filterStatus != "excluded" {
			err := fmt.Errorf("invalid filterStatus: must be 'all', 'included', or 'excluded'")
			return ErrorResponse(http.StatusBadRequest, err), err
		}
	}

	// Read and parse the uploaded config file
	if configParam == nil {
		err := errors.New("config file is required")
		return ErrorResponse(http.StatusBadRequest, err), err
	}
	defer configParam.Close()

	configBytes, err := os.ReadFile(configParam.Name())
	if err != nil {
		return ErrorResponse(http.StatusBadRequest, fmt.Errorf("failed to read config file: %w", err)), err
	}

	// Read catalog data if provided (stateless mode)
	var catalogDataBytes []byte
	if catalogDataParam != nil {
		defer catalogDataParam.Close()
		catalogDataBytes, err = os.ReadFile(catalogDataParam.Name())
		if err != nil {
			return ErrorResponse(http.StatusBadRequest, fmt.Errorf("failed to read catalogData file: %w", err)), err
		}
	}

	// Detect assetType from config to determine the preview path
	var assetTypeProbe struct {
		AssetType string `json:"assetType" yaml:"assetType"`
	}
	// Ignore errors — missing assetType defaults to models
	_ = yaml.Unmarshal(configBytes, &assetTypeProbe)

	if assetTypeProbe.AssetType == string(model.CATALOGASSETTYPE_MCP_SERVERS) {
		return m.previewMCPSource(ctx, configBytes, catalogDataBytes, pageSizeParam, nextPageTokenParam, filterStatus)
	}

	return m.previewModelSource(ctx, configBytes, catalogDataBytes, pageSizeParam, nextPageTokenParam, filterStatus)
}

type previewPage[T model.Sortable] struct {
	items         []T
	total         int32
	includedCount int32
	excludedCount int32
	nextPageToken string
	pageSize      int32
}

func filterAndPaginate[T model.Sortable](results []T, isIncluded func(T) bool, filterStatus, pageSizeParam, nextPageTokenParam string) (*previewPage[T], error) {
	var filtered []T
	var includedCount, excludedCount int32
	for _, r := range results {
		if isIncluded(r) {
			includedCount++
		} else {
			excludedCount++
		}
		switch filterStatus {
		case "included":
			if isIncluded(r) {
				filtered = append(filtered, r)
			}
		case "excluded":
			if !isIncluded(r) {
				filtered = append(filtered, r)
			}
		default:
			filtered = append(filtered, r)
		}
	}

	p, err := newPaginator[T](pageSizeParam, model.OrderByField(""), model.SortOrder(""), nextPageTokenParam)
	if err != nil {
		return nil, err
	}

	paged, next := p.Paginate(filtered)

	pageSize, _ := parsePageSize(pageSizeParam)

	return &previewPage[T]{
		items:         paged,
		total:         int32(len(results)),
		includedCount: includedCount,
		excludedCount: excludedCount,
		nextPageToken: next.Token(),
		pageSize:      pageSize,
	}, nil
}

func (m *ModelCatalogServiceAPIService) previewModelSource(ctx context.Context, configBytes, catalogDataBytes []byte, pageSizeParam, nextPageTokenParam, filterStatus string) (ImplResponse, error) {
	previewRequest, err := catalog.ParsePreviewConfig(configBytes)
	if err != nil {
		return ErrorResponse(http.StatusUnprocessableEntity, fmt.Errorf("invalid config: %w", err)), err
	}

	previewResults, err := catalog.PreviewSourceModels(ctx, previewRequest, catalogDataBytes)
	if err != nil {
		return ErrorResponse(http.StatusUnprocessableEntity, fmt.Errorf("failed to load models: %w", err)), err
	}

	page, err := filterAndPaginate(previewResults, func(r model.ModelPreviewResult) bool { return r.Included }, filterStatus, pageSizeParam, nextPageTokenParam)
	if err != nil {
		return ErrorResponse(http.StatusBadRequest, err), err
	}

	return Response(http.StatusOK, model.CatalogSourcePreviewResponse{
		AssetType:     model.CATALOGASSETTYPE_MODELS,
		PageSize:      page.pageSize,
		Size:          int32(len(page.items)),
		NextPageToken: page.nextPageToken,
		Items:         page.items,
		Summary: model.CatalogSourcePreviewResponseAllOfSummary{
			TotalModels:    page.total,
			IncludedModels: page.includedCount,
			ExcludedModels: page.excludedCount,
		},
	}), nil
}

func (m *ModelCatalogServiceAPIService) previewMCPSource(ctx context.Context, configBytes, catalogDataBytes []byte, pageSizeParam, nextPageTokenParam, filterStatus string) (ImplResponse, error) {
	previewRequest, err := mcpcatalog.ParseMCPPreviewConfig(configBytes)
	if err != nil {
		return ErrorResponse(http.StatusUnprocessableEntity, fmt.Errorf("invalid config: %w", err)), err
	}

	previewResults, err := mcpcatalog.PreviewSourceServers(ctx, previewRequest, catalogDataBytes)
	if err != nil {
		return ErrorResponse(http.StatusUnprocessableEntity, fmt.Errorf("failed to load servers: %w", err)), err
	}

	page, err := filterAndPaginate(previewResults, func(r model.AssetPreviewResult) bool { return r.Included }, filterStatus, pageSizeParam, nextPageTokenParam)
	if err != nil {
		return ErrorResponse(http.StatusBadRequest, err), err
	}

	return Response(http.StatusOK, model.AssetSourcePreviewResponse{
		AssetType:     model.CATALOGASSETTYPE_MCP_SERVERS,
		PageSize:      page.pageSize,
		Size:          int32(len(page.items)),
		NextPageToken: page.nextPageToken,
		Items:         page.items,
		Summary: model.AssetSourcePreviewResponseAllOfSummary{
			TotalAssets:    page.total,
			IncludedAssets: page.includedCount,
			ExcludedAssets: page.excludedCount,
		},
	}), nil
}

func genCatalogCmpFunc(orderBy model.OrderByField, sortOrder model.SortOrder) (func(model.CatalogSource, model.CatalogSource) int, error) {
	multiplier := 1
	switch model.SortOrder(strings.ToUpper(string(sortOrder))) {
	case model.SORTORDER_DESC:
		multiplier = -1
	case model.SORTORDER_ASC, "":
		multiplier = 1
	default:
		return nil, fmt.Errorf("unsupported sort order field")
	}

	switch model.OrderByField(strings.ToUpper(string(orderBy))) {
	case model.ORDERBYFIELD_ID, "":
		return func(a, b model.CatalogSource) int {
			return multiplier * strings.Compare(a.Id, b.Id)
		}, nil
	case model.ORDERBYFIELD_NAME:
		return func(a, b model.CatalogSource) int {
			return multiplier * strings.Compare(a.Name, b.Name)
		}, nil
	default:
		return nil, fmt.Errorf("unsupported order by field")
	}
}

// generateLabelID creates a stable, unique ID for a label based on its index
func generateLabelID(index int) string {
	return strconv.Itoa(index)
}

// sortableLabel wraps a label map to make it sortable
type sortableLabel struct {
	data  map[string]any
	index int    // Original position for stable sort when key is missing
	id    string // Stable ID for pagination
}

// SortValue implements the Sortable interface for labels
func (sl sortableLabel) SortValue(field model.OrderByField) string {
	// Return ID for pagination purposes
	if field == model.ORDERBYFIELD_ID {
		return sl.id
	}
	// For other fields, labels use string keys directly in genLabelCmpFunc
	return ""
}

// genLabelCmpFunc generates a comparison function for sorting labels by a string key
func genLabelCmpFunc(orderByKey string, sortOrder model.SortOrder) func(sortableLabel, sortableLabel) int {
	multiplier := 1
	switch model.SortOrder(strings.ToUpper(string(sortOrder))) {
	case model.SORTORDER_DESC:
		multiplier = -1
	case model.SORTORDER_ASC, "":
		multiplier = 1
	}

	return func(a, b sortableLabel) int {
		// If no orderBy key specified, maintain original order
		if orderByKey == "" {
			if a.index < b.index {
				return -1
			}
			if a.index > b.index {
				return 1
			}
			return 0
		}

		// Get values for the orderBy key
		aValRaw, aHasKey := a.data[orderByKey]
		bValRaw, bHasKey := b.data[orderByKey]

		var aVal string
		if aHasKey {
			aVal, aHasKey = aValRaw.(string)
		}
		var bVal string
		if bHasKey {
			bVal, bHasKey = bValRaw.(string)
		}

		// If both have the key, compare their values
		if aHasKey && bHasKey {
			return multiplier * strings.Compare(aVal, bVal)
		}

		// If only one has the key, put it first
		if aHasKey && !bHasKey {
			return -1 // a comes first
		}
		if !aHasKey && bHasKey {
			return 1 // b comes first
		}

		// If neither has the key, maintain original order
		if a.index < b.index {
			return -1
		}
		if a.index > b.index {
			return 1
		}
		return 0
	}
}

var _ ModelCatalogServiceAPIServicer = &ModelCatalogServiceAPIService{}

// NewModelCatalogServiceAPIService creates a default api service
func NewModelCatalogServiceAPIService(provider catalog.APIProvider, sources *catalog.SourceCollection, mcpSources *catalog.MCPSourceCollection, labels *catalog.LabelCollection, sourceRepository models.CatalogSourceRepository) ModelCatalogServiceAPIServicer {
	return &ModelCatalogServiceAPIService{
		provider:         provider,
		sources:          sources,
		mcpSources:       mcpSources,
		labels:           labels,
		sourceRepository: sourceRepository,
	}
}

func notFound(msg string) ImplResponse {
	if msg == "" {
		msg = "Resource not found"
	}
	return ErrorResponse(http.StatusNotFound, errors.New(msg))
}
