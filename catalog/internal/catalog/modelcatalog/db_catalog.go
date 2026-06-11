package modelcatalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	sharedmodels "github.com/kubeflow/hub/catalog/internal/db/models"
	apimodels "github.com/kubeflow/hub/catalog/pkg/openapi"
	"github.com/kubeflow/hub/internal/converter"
	mrmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/kubeflow/hub/pkg/api"
	"github.com/kubeflow/hub/pkg/openapi"
)

type dbCatalogImpl struct {
	catalogModelRepository    models.CatalogModelRepository
	catalogArtifactRepository sharedmodels.CatalogArtifactRepository
	propertyOptionsRepository sharedmodels.PropertyOptionsRepository
	performanceService        *PerformanceArtifactService
	sources                   *SourceCollection
}

func NewDBCatalog(services Services, sources *SourceCollection) APIProvider {
	return &dbCatalogImpl{
		catalogArtifactRepository: services.CatalogArtifactRepository,
		catalogModelRepository:    services.CatalogModelRepository,
		propertyOptionsRepository: services.PropertyOptionsRepository,
		performanceService:        NewPerformanceArtifactService(services.CatalogArtifactRepository, services.CatalogModelRepository),
		sources:                   sources,
	}
}

func (d *dbCatalogImpl) GetModel(ctx context.Context, modelName string, sourceID string) (*apimodels.CatalogModel, error) {
	// Resolve by namespaced identifier: sourceId:modelName
	namespacedName := sourceID + ":" + modelName
	modelsList, err := d.catalogModelRepository.List(models.CatalogModelListOptions{
		Name:      &namespacedName,
		SourceIDs: &[]string{sourceID},
	})
	if err != nil {
		return nil, err
	}

	if len(modelsList.Items) == 0 {
		return nil, fmt.Errorf("no models found for name=%v: %w", modelName, api.ErrNotFound)
	}

	if len(modelsList.Items) > 1 {
		return nil, fmt.Errorf("multiple models found for name=%v: %w", modelName, api.ErrNotFound)
	}

	model, err := mapDBModelToAPIModel(modelsList.Items[0])
	if err != nil {
		return nil, fmt.Errorf("error mapping model %s: %w", modelName, err)
	}

	return &model, nil
}

func (d *dbCatalogImpl) ListModels(ctx context.Context, params ListModelsParams) (apimodels.CatalogModelList, error) {
	pageSize := int32(params.PageSize)
	orderBy := string(params.OrderBy)
	sortOrder := string(params.SortOrder)

	// Use consistent defaults to match pagination logic
	if orderBy == "" {
		orderBy = mrmodels.DefaultOrderBy
	} else if orderBy == "ACCURACY" {
		orderBy = "artifacts.overall_average.double_value"
	}

	if sortOrder == "" {
		sortOrder = mrmodels.DefaultSortOrder
	}

	nextPageToken := params.NextPageToken

	var queryPtr *string
	if params.Query != "" {
		queryPtr = &params.Query
	}

	sourceIDs := params.SourceIDs

	modelsList, err := d.catalogModelRepository.List(models.CatalogModelListOptions{
		SourceIDs: &sourceIDs,
		Query:     queryPtr,
		Pagination: mrmodels.Pagination{
			FilterQuery:   &params.FilterQuery,
			PageSize:      &pageSize,
			OrderBy:       &orderBy,
			SortOrder:     &sortOrder,
			NextPageToken: nextPageToken,
		},
	})
	if err != nil {
		return apimodels.CatalogModelList{}, err
	}

	modelList := &apimodels.CatalogModelList{
		Items: make([]apimodels.CatalogModel, 0, len(modelsList.Items)),
	}

	for _, model := range modelsList.Items {
		apiModel, err := mapDBModelToAPIModel(model)
		if err != nil {
			glog.Warningf("error mapping model, skipping: %v", err)
			continue
		}
		modelList.Items = append(modelList.Items, apiModel)
	}

	modelList.NextPageToken = modelsList.NextPageToken
	modelList.PageSize = pageSize
	modelList.Size = int32(len(modelsList.Items))

	return *modelList, nil
}

func (d *dbCatalogImpl) GetArtifacts(ctx context.Context, modelName string, sourceID string, params ListArtifactsParams) (apimodels.CatalogArtifactList, error) {
	pageSize := params.PageSize

	// Use consistent defaults to match pagination logic
	orderBy := string(params.OrderBy)
	if orderBy == "" {
		orderBy = mrmodels.DefaultOrderBy
	}

	sortOrder := string(params.SortOrder)
	if sortOrder == "" {
		sortOrder = mrmodels.DefaultSortOrder
	}

	nextPageToken := params.NextPageToken

	m, err := d.GetModel(ctx, modelName, sourceID)
	if err != nil {
		if errors.Is(err, api.ErrNotFound) {
			return apimodels.CatalogArtifactList{}, fmt.Errorf("invalid model name '%s' for source '%s': %w", modelName, sourceID, api.ErrBadRequest)
		}
		return apimodels.CatalogArtifactList{}, err
	}

	parentResourceID, err := strconv.ParseInt(*m.Id, 10, 32)
	if err != nil {
		return apimodels.CatalogArtifactList{}, err
	}

	parentResourceID32 := int32(parentResourceID)

	var filterQueryPtr *string
	if params.FilterQuery != "" {
		filterQueryPtr = &params.FilterQuery
	}

	artifactsList, err := d.catalogArtifactRepository.List(sharedmodels.CatalogArtifactListOptions{
		ParentResourceID:    &parentResourceID32,
		ArtifactTypesFilter: params.ArtifactTypesFilter,
		Pagination: mrmodels.Pagination{
			FilterQuery:   filterQueryPtr,
			PageSize:      &pageSize,
			OrderBy:       &orderBy,
			SortOrder:     &sortOrder,
			NextPageToken: nextPageToken,
		},
	})
	if err != nil {
		return apimodels.CatalogArtifactList{}, err
	}

	artifactList := &apimodels.CatalogArtifactList{
		Items: make([]apimodels.CatalogArtifact, 0),
	}

	for _, artifact := range artifactsList.Items {
		mappedArtifact, err := mapDBArtifactToAPIArtifact(artifact)
		if err != nil {
			return apimodels.CatalogArtifactList{}, err
		}
		artifactList.Items = append(artifactList.Items, mappedArtifact)
	}

	artifactList.NextPageToken = artifactsList.NextPageToken
	artifactList.PageSize = pageSize
	artifactList.Size = int32(len(artifactList.Items))

	return *artifactList, nil
}

func (d *dbCatalogImpl) GetFilterOptions(ctx context.Context) (*apimodels.FilterOptionsList, error) {
	catalogModelTypeID := d.catalogModelRepository.GetTypeID()

	contextProperties, err := d.propertyOptionsRepository.List(sharedmodels.ContextPropertyOptionType, catalogModelTypeID)
	if err != nil {
		return nil, err
	}
	// Artifact type filtering uses 0 (all types) intentionally: models have multiple
	// artifact types (kf.CatalogModelArtifact, kf.CatalogMetricsArtifact) so a single
	// type ID cannot scope them. MCP servers have no artifact types, so no cross-contamination.
	artifactProperties, err := d.propertyOptionsRepository.List(sharedmodels.ArtifactPropertyOptionType, 0)
	if err != nil {
		return nil, err
	}

	// Build FilterOptionsList
	options := make(map[string]apimodels.FilterOption, len(contextProperties)+len(artifactProperties))

	for _, prop := range contextProperties {
		// Skip internal/technical fields that shouldn't be exposed as filters
		switch prop.Name {
		case "source_id", "logo", "license_link", "serving_config":
			continue
		}

		option := basecatalog.DbPropToAPIOption(prop)
		if option != nil {
			options[prop.FullName("")] = *option
		}
	}

	for _, prop := range artifactProperties {
		// Skip internal/technical fields that shouldn't be exposed as filters
		switch prop.Name {
		case "metricsType", "model_id":
			continue
		}
		option := basecatalog.DbPropToAPIOption(prop)
		if option != nil {
			options[prop.FullName("artifacts")] = *option
		}
	}

	// Get named queries from sources configuration
	var namedQueriesPtr *map[string]map[string]apimodels.FieldFilter
	if d.sources != nil {
		namedQueriesPtr = basecatalog.ConvertNamedQueries(d.sources.GetNamedQueries(), options)
	}

	return &apimodels.FilterOptionsList{
		Filters:      &options,
		NamedQueries: namedQueriesPtr,
	}, nil
}

func (d *dbCatalogImpl) GetPerformanceArtifacts(ctx context.Context, modelName string, sourceID string, params ListPerformanceArtifactsParams) (apimodels.CatalogArtifactList, error) {
	// Resolve by namespaced identifier: sourceId:modelName
	namespacedName := sourceID + ":" + modelName
	// Get the model to validate it exists and get its ID
	modelsList, err := d.catalogModelRepository.List(models.CatalogModelListOptions{
		Name:      &namespacedName,
		SourceIDs: &[]string{sourceID},
	})
	if err != nil {
		return apimodels.CatalogArtifactList{}, err
	}

	if len(modelsList.Items) == 0 {
		return apimodels.CatalogArtifactList{}, fmt.Errorf("no models found for name=%v: %w", modelName, api.ErrNotFound)
	}

	if len(modelsList.Items) > 1 {
		return apimodels.CatalogArtifactList{}, fmt.Errorf("multiple models found for name=%v: %w", modelName, api.ErrNotFound)
	}

	model := modelsList.Items[0]

	serviceParams := PerformanceArtifactParams{
		ModelID:               *model.GetID(),
		TargetRPS:             params.TargetRPS,
		Recommendations:       params.Recommendations,
		FilterQuery:           params.FilterQuery,
		PageSize:              params.PageSize,
		OrderBy:               params.OrderBy,
		SortOrder:             string(params.SortOrder),
		NextPageToken:         params.NextPageToken,
		RPSProperty:           params.RPSProperty,
		LatencyProperty:       params.LatencyProperty,
		HardwareCountProperty: params.HardwareCountProperty,
		HardwareTypeProperty:  params.HardwareTypeProperty,
	}

	artifactsList, err := d.performanceService.GetArtifacts(serviceParams)
	if err != nil {
		return apimodels.CatalogArtifactList{}, fmt.Errorf("failed to get performance artifacts: %w", err)
	}

	artifactList := &apimodels.CatalogArtifactList{
		Items: make([]apimodels.CatalogArtifact, 0, len(artifactsList.Items)),
	}

	for _, artifact := range artifactsList.Items {
		mappedArtifact, err := mapDBArtifactToAPIArtifact(sharedmodels.CatalogArtifact{
			CatalogMetricsArtifact: artifact,
		})
		if err != nil {
			return apimodels.CatalogArtifactList{}, err
		}
		artifactList.Items = append(artifactList.Items, mappedArtifact)
	}

	artifactList.NextPageToken = artifactsList.NextPageToken
	artifactList.PageSize = params.PageSize
	artifactList.Size = int32(len(artifactList.Items))

	return *artifactList, nil
}

func mapDBModelToAPIModel(m models.CatalogModel) (apimodels.CatalogModel, error) {
	res := apimodels.CatalogModel{}

	id := strconv.FormatInt(int64(*m.GetID()), 10)
	res.Id = &id

	if m.GetAttributes() != nil {
		storedName := *m.GetAttributes().Name
		// Return display name: strip "sourceId:" prefix so API shows model name only.
		// Source is already exposed via source_id.
		res.Name = DisplayNameFromStoredName(storedName)
		res.ExternalId = m.GetAttributes().ExternalID

		if m.GetAttributes().CreateTimeSinceEpoch != nil {
			createTimeSinceEpoch := strconv.FormatInt(*m.GetAttributes().CreateTimeSinceEpoch, 10)
			res.CreateTimeSinceEpoch = &createTimeSinceEpoch
		}
		if m.GetAttributes().LastUpdateTimeSinceEpoch != nil {
			lastUpdateTimeSinceEpoch := strconv.FormatInt(*m.GetAttributes().LastUpdateTimeSinceEpoch, 10)
			res.LastUpdateTimeSinceEpoch = &lastUpdateTimeSinceEpoch
		}
	}

	if m.GetProperties() != nil {
		for _, prop := range *m.GetProperties() {
			switch prop.Name {
			case "source_id":
				if prop.StringValue != nil {
					res.SourceId = prop.StringValue
				}
			case "description":
				if prop.StringValue != nil {
					res.Description = prop.StringValue
				}
			case "library_name":
				if prop.StringValue != nil {
					res.LibraryName = prop.StringValue
				}
			case "license_link":
				if prop.StringValue != nil {
					res.LicenseLink = prop.StringValue
				}
			case "license":
				if prop.StringValue != nil {
					res.License = prop.StringValue
				}
			case "logo":
				if prop.StringValue != nil {
					res.Logo = prop.StringValue
				}
			case "maturity":
				if prop.StringValue != nil {
					res.Maturity = prop.StringValue
				}
			case "provider":
				if prop.StringValue != nil {
					res.Provider = prop.StringValue
				}
			case "readme":
				if prop.StringValue != nil {
					res.Readme = prop.StringValue
				}
			case "language":
				if prop.StringValue != nil {
					var languages []string
					if err := json.Unmarshal([]byte(*prop.StringValue), &languages); err == nil {
						res.Language = languages
					}
				}
			case "tasks":
				if prop.StringValue != nil {
					var tasks []string
					if err := json.Unmarshal([]byte(*prop.StringValue), &tasks); err == nil {
						res.Tasks = tasks
					}
				}
			case "validated_tasks":
				if prop.StringValue != nil {
					var validatedTasks []string
					if err := json.Unmarshal([]byte(*prop.StringValue), &validatedTasks); err != nil {
						glog.Warningf("failed to unmarshal validated_tasks for model %s: %v", res.Name, err)
					} else {
						res.ValidatedTasks = validatedTasks
					}
				}
			case "serving_config":
				if prop.StringValue != nil {
					var servingConfig apimodels.ServingConfig
					if err := json.Unmarshal([]byte(*prop.StringValue), &servingConfig); err != nil {
						glog.Warningf("failed to unmarshal serving_config for model %s: %v", res.Name, err)
					} else {
						res.ServingConfig = &servingConfig
					}
				}
			}
		}
	}

	// Map custom properties
	if m.GetCustomProperties() != nil && len(*m.GetCustomProperties()) > 0 {
		customPropsMap, err := converter.MapEmbedMDCustomProperties(*m.GetCustomProperties())
		if err != nil {
			return apimodels.CatalogModel{}, fmt.Errorf("error mapping custom properties: %w", err)
		}
		res.CustomProperties = convertMetadataValueMap(customPropsMap)
	}

	return res, nil
}

func mapDBArtifactToAPIArtifact(a sharedmodels.CatalogArtifact) (apimodels.CatalogArtifact, error) {
	if a.CatalogModelArtifact != nil {
		modelArtifact, ok := a.CatalogModelArtifact.(models.CatalogModelArtifact)
		if !ok {
			return apimodels.CatalogArtifact{}, fmt.Errorf("invalid catalog model artifact type: %T", a.CatalogModelArtifact)
		}
		return mapToModelArtifact(modelArtifact)
	} else if a.CatalogMetricsArtifact != nil {
		metricsArtifact, ok := a.CatalogMetricsArtifact.(models.CatalogMetricsArtifact)
		if !ok {
			return apimodels.CatalogArtifact{}, fmt.Errorf("invalid catalog metrics artifact type: %T", a.CatalogMetricsArtifact)
		}
		metricsTypeValue := string(metricsArtifact.GetAttributes().MetricsType)
		return mapToMetricsArtifact(metricsArtifact, metricsTypeValue)
	}

	return apimodels.CatalogArtifact{}, fmt.Errorf("invalid catalog artifact type: %v", a)
}

func mapToModelArtifact(a models.CatalogModelArtifact) (apimodels.CatalogArtifact, error) {
	catalogModelArtifact := &apimodels.CatalogModelArtifact{
		ArtifactType: models.CatalogModelArtifactType,
	}

	if a.GetID() != nil {
		id := strconv.FormatInt(int64(*a.GetID()), 10)
		catalogModelArtifact.Id = &id
	}

	if a.GetAttributes() != nil {
		attrs := a.GetAttributes()

		catalogModelArtifact.Name = attrs.Name
		catalogModelArtifact.ExternalId = attrs.ExternalID

		if attrs.URI != nil {
			catalogModelArtifact.Uri = *attrs.URI
		}

		if attrs.CreateTimeSinceEpoch != nil {
			createTime := strconv.FormatInt(*attrs.CreateTimeSinceEpoch, 10)
			catalogModelArtifact.CreateTimeSinceEpoch = &createTime
		}

		if attrs.LastUpdateTimeSinceEpoch != nil {
			updateTime := strconv.FormatInt(*attrs.LastUpdateTimeSinceEpoch, 10)
			catalogModelArtifact.LastUpdateTimeSinceEpoch = &updateTime
		}
	}

	if a.GetProperties() != nil {
		for _, prop := range *a.GetProperties() {
			switch prop.Name {
			case "description":
				if prop.StringValue != nil {
					catalogModelArtifact.Description = prop.StringValue
				}
			case "artifactType":
				if prop.StringValue != nil {
					catalogModelArtifact.ArtifactType = *prop.StringValue
				}
			}
		}
	}

	// Map custom properties
	if a.GetCustomProperties() != nil && len(*a.GetCustomProperties()) > 0 {
		customPropsMap, err := converter.MapEmbedMDCustomProperties(*a.GetCustomProperties())
		if err != nil {
			return apimodels.CatalogArtifact{}, fmt.Errorf("error mapping custom properties: %w", err)
		}

		catalogCustomProps := convertMetadataValueMap(customPropsMap)
		catalogModelArtifact.CustomProperties = catalogCustomProps
	}

	return apimodels.CatalogArtifact{
		CatalogModelArtifact: catalogModelArtifact,
	}, nil
}

func mapToMetricsArtifact(a models.CatalogMetricsArtifact, metricsType string) (apimodels.CatalogArtifact, error) {
	catalogMetricsArtifact := &apimodels.CatalogMetricsArtifact{
		ArtifactType: models.CatalogMetricsArtifactType,
		MetricsType:  metricsType,
	}

	if a.GetID() != nil {
		id := strconv.FormatInt(int64(*a.GetID()), 10)
		catalogMetricsArtifact.Id = &id
	}

	if a.GetAttributes() != nil {
		attrs := a.GetAttributes()

		catalogMetricsArtifact.Name = attrs.Name
		catalogMetricsArtifact.ExternalId = attrs.ExternalID

		if attrs.CreateTimeSinceEpoch != nil {
			createTime := strconv.FormatInt(*attrs.CreateTimeSinceEpoch, 10)
			catalogMetricsArtifact.CreateTimeSinceEpoch = &createTime
		}

		if attrs.LastUpdateTimeSinceEpoch != nil {
			updateTime := strconv.FormatInt(*attrs.LastUpdateTimeSinceEpoch, 10)
			catalogMetricsArtifact.LastUpdateTimeSinceEpoch = &updateTime
		}
	}

	if a.GetProperties() != nil {
		for _, prop := range *a.GetProperties() {
			switch prop.Name {
			case "description":
				if prop.StringValue != nil {
					catalogMetricsArtifact.Description = prop.StringValue
				}
			}
		}
	}

	// Map custom properties
	if a.GetCustomProperties() != nil && len(*a.GetCustomProperties()) > 0 {
		customPropsMap, err := converter.MapEmbedMDCustomProperties(*a.GetCustomProperties())
		if err != nil {
			return apimodels.CatalogArtifact{}, fmt.Errorf("error mapping custom properties: %w", err)
		}

		catalogCustomProps := convertMetadataValueMap(customPropsMap)
		catalogMetricsArtifact.CustomProperties = catalogCustomProps

	}

	return apimodels.CatalogArtifact{
		CatalogMetricsArtifact: catalogMetricsArtifact,
	}, nil
}

// convertMetadataValueMap converts from pkg/openapi.MetadataValue to catalog/pkg/openapi.MetadataValue
func convertMetadataValueMap(source map[string]openapi.MetadataValue) map[string]apimodels.MetadataValue {
	result := make(map[string]apimodels.MetadataValue)

	for key, value := range source {
		catalogValue := apimodels.MetadataValue{}

		if value.MetadataStringValue != nil {
			catalogValue.MetadataStringValue = &apimodels.MetadataStringValue{
				StringValue:  value.MetadataStringValue.StringValue,
				MetadataType: value.MetadataStringValue.MetadataType,
			}
		} else if value.MetadataIntValue != nil {
			catalogValue.MetadataIntValue = &apimodels.MetadataIntValue{
				IntValue:     value.MetadataIntValue.IntValue,
				MetadataType: value.MetadataIntValue.MetadataType,
			}
		} else if value.MetadataDoubleValue != nil {
			catalogValue.MetadataDoubleValue = &apimodels.MetadataDoubleValue{
				DoubleValue:  value.MetadataDoubleValue.DoubleValue,
				MetadataType: value.MetadataDoubleValue.MetadataType,
			}
		} else if value.MetadataBoolValue != nil {
			catalogValue.MetadataBoolValue = &apimodels.MetadataBoolValue{
				BoolValue:    value.MetadataBoolValue.BoolValue,
				MetadataType: value.MetadataBoolValue.MetadataType,
			}
		} else if value.MetadataStructValue != nil {
			catalogValue.MetadataStructValue = &apimodels.MetadataStructValue{
				StructValue:  value.MetadataStructValue.StructValue,
				MetadataType: value.MetadataStructValue.MetadataType,
			}
		}

		result[key] = catalogValue
	}

	return result
}

func (d *dbCatalogImpl) FindModelsWithRecommendedLatency(
	ctx context.Context,
	pagination mrmodels.Pagination,
	paretoParams ParetoFilteringParams,
	sourceIDs []string,
	query string,
	sortOrder string,
) (*apimodels.CatalogModelList, error) {
	// Get all models first (without pagination)
	var sourceIDsPtr *[]string
	if len(sourceIDs) > 0 {
		sourceIDsPtr = &sourceIDs
	}

	var queryPtr *string
	if query != "" {
		queryPtr = &query
	}

	allModels, err := d.catalogModelRepository.List(models.CatalogModelListOptions{
		SourceIDs: sourceIDsPtr,
		Query:     queryPtr,
		Pagination: mrmodels.Pagination{
			FilterQuery: pagination.FilterQuery,
			PageSize:    new(int32(0)), // Get all models
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}

	type modelWithLatency struct {
		Model   apimodels.CatalogModel
		Latency *float64
	}

	var modelsWithLatency []modelWithLatency

	// Get recommended latency for each model
	for _, model := range allModels.Items {
		apiModel, err := mapDBModelToAPIModel(model)
		if err != nil {
			glog.Warningf("error mapping model: %v, skipping", err)
			continue
		}

		// Extract source ID from model properties
		sourceID := ""
		if apiModel.SourceId != nil {
			sourceID = *apiModel.SourceId
		}

		// Transform filter query from models format to performance artifacts format
		// Models API: artifacts.use_case.string_value='chatbot'
		// Performance API: use_case.string_value='chatbot'
		filterQuery := ""
		if pagination.FilterQuery != nil {
			filterQuery = *pagination.FilterQuery
			// Extract only artifact-related filters and remove "artifacts." prefix
			filterQuery = strings.ReplaceAll(filterQuery, "artifacts.", "")
		}

		// Use stored (namespaced) name for lookup: sourceId:modelName
		namespacedName := ""
		if model.GetAttributes() != nil && model.GetAttributes().Name != nil {
			namespacedName = *model.GetAttributes().Name
		}
		latency, err := d.performanceService.GetMinimumRecommendedLatency(
			ctx,
			namespacedName,
			sourceID,
			paretoParams,
			filterQuery,
		)
		if err != nil {
			glog.Warningf("Warning: failed to get latency for model %s: %v", apiModel.Name, err)
			latency = nil // Treat as no latency data
		}

		modelsWithLatency = append(modelsWithLatency, modelWithLatency{
			Model:   apiModel,
			Latency: latency,
		})
	}

	// Sort: models with latency first (ascending), then models without latency.
	// ASC (default) = lowest latency first (most recommended); DESC = highest latency first.
	descending := strings.EqualFold(sortOrder, "DESC")
	sort.Slice(modelsWithLatency, func(i, j int) bool {
		latencyI, latencyJ := modelsWithLatency[i].Latency, modelsWithLatency[j].Latency

		if latencyI == nil && latencyJ == nil {
			return false
		}
		if latencyI == nil {
			return false // Models without latency always last
		}
		if latencyJ == nil {
			return true // Models with latency always first
		}
		if descending {
			return *latencyI > *latencyJ
		}
		return *latencyI < *latencyJ
	})

	// Apply pagination to sorted results
	pageSize := int32(10) // default
	if pagination.PageSize != nil {
		pageSize = *pagination.PageSize
	}

	start := 0
	if pagination.NextPageToken != nil {
		if parsed, err := strconv.Atoi(*pagination.NextPageToken); err == nil {
			start = parsed
		}
	}
	if start > len(modelsWithLatency) {
		start = len(modelsWithLatency)
	}

	end := min(start+int(pageSize), len(modelsWithLatency))

	paginatedItems := make([]apimodels.CatalogModel, 0, end-start)
	for i := start; i < end; i++ {
		paginatedItems = append(paginatedItems, modelsWithLatency[i].Model)
	}

	// Calculate next page token
	var nextPageToken string
	if end < len(modelsWithLatency) {
		nextPageToken = fmt.Sprintf("%d", end)
	}

	return &apimodels.CatalogModelList{
		Items:         paginatedItems,
		NextPageToken: nextPageToken,
		PageSize:      pageSize,
		Size:          int32(len(paginatedItems)),
	}, nil
}
