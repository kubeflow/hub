package service

import (
	"errors"
	"fmt"

	"github.com/kubeflow/hub/internal/db/models"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"github.com/kubeflow/hub/internal/platform/db/utils"
	"gorm.io/gorm"
)

var ErrInferenceServiceNotFound = errors.New("inference service by id not found")

type InferenceServiceRepositoryImpl struct {
	*GenericRepository[models.InferenceService, schema.Context, schema.ContextProperty, *models.InferenceServiceListOptions]
}

func NewInferenceServiceRepository(db *gorm.DB, typeID int32) models.InferenceServiceRepository {
	config := GenericRepositoryConfig[models.InferenceService, schema.Context, schema.ContextProperty, *models.InferenceServiceListOptions]{
		DB:                  db,
		TypeID:              typeID,
		EntityToSchema:      mapInferenceServiceToContext,
		SchemaToEntity:      mapDataLayerToInferenceService,
		EntityToProperties:  mapInferenceServiceToContextProperties,
		NotFoundError:       ErrInferenceServiceNotFound,
		EntityName:          "inference service",
		PropertyFieldName:   "context_id",
		ApplyListFilters:    applyInferenceServiceListFilters,
		IsNewEntity:         func(entity models.InferenceService) bool { return entity.GetID() == nil },
		HasCustomProperties: func(entity models.InferenceService) bool { return entity.GetCustomProperties() != nil },
	}

	return &InferenceServiceRepositoryImpl{
		GenericRepository: NewGenericRepository(config),
	}
}

func (r *InferenceServiceRepositoryImpl) Save(inferenceService models.InferenceService) (models.InferenceService, error) {
	// Extract serving_environment_id from properties for parent relationship
	var servingEnvironmentID *int32
	if inferenceService.GetProperties() != nil {
		for _, prop := range *inferenceService.GetProperties() {
			if prop.Name == "serving_environment_id" && prop.IntValue != nil {
				servingEnvironmentID = prop.IntValue
				break
			}
		}
	}
	return r.GenericRepository.Save(inferenceService, servingEnvironmentID)
}

func (r *InferenceServiceRepositoryImpl) List(listOptions models.InferenceServiceListOptions) (*models.ListWrapper[models.InferenceService], error) {
	return r.GenericRepository.List(&listOptions)
}

func applyInferenceServiceListFilters(query *gorm.DB, listOptions *models.InferenceServiceListOptions) *gorm.DB {
	if listOptions.Name != nil {
		if listOptions.ParentResourceID != nil {
			query = query.Where("name LIKE ?", fmt.Sprintf("%d:%s", *listOptions.ParentResourceID, escapeLike(*listOptions.Name)))
		} else {
			query = query.Where("name LIKE ?", fmt.Sprintf("%%:%s", escapeLike(*listOptions.Name)))
		}
	} else if listOptions.ExternalID != nil {
		query = query.Where("external_id = ?", listOptions.ExternalID)
	}

	if listOptions.ParentResourceID != nil {
		query = query.Joins(utils.BuildParentContextJoin(query)).
			Where(utils.GetColumnRef(query, &schema.ParentContext{}, "parent_context_id")+" = ?", listOptions.ParentResourceID)
	}

	if listOptions.Runtime != nil {
		// Proper GORM JOIN: Use helper that respects naming strategy
		query = query.Joins(utils.BuildContextPropertyJoin(query, "runtime")).
			Where(utils.GetColumnRef(query, &schema.ContextProperty{}, "string_value")+" = ?", listOptions.Runtime)
	}

	return query
}

func mapInferenceServiceToContext(inferenceService models.InferenceService) schema.Context {
	attrs := inferenceService.GetAttributes()
	context := schema.Context{
		TypeID: *inferenceService.GetTypeID(),
	}

	// Only set ID if it's not nil (for existing entities)
	if inferenceService.GetID() != nil {
		context.ID = *inferenceService.GetID()
	}

	if attrs != nil {
		if attrs.Name != nil {
			context.Name = *attrs.Name
		}
		context.ExternalID = attrs.ExternalID
		if attrs.CreateTimeSinceEpoch != nil {
			context.CreateTimeSinceEpoch = *attrs.CreateTimeSinceEpoch
		}
		if attrs.LastUpdateTimeSinceEpoch != nil {
			context.LastUpdateTimeSinceEpoch = *attrs.LastUpdateTimeSinceEpoch
		}
	}

	return context
}

func mapInferenceServiceToContextProperties(inferenceService models.InferenceService, contextID int32) []schema.ContextProperty {
	var properties []schema.ContextProperty

	if inferenceService.GetProperties() != nil {
		for _, prop := range *inferenceService.GetProperties() {
			properties = append(properties, MapPropertiesToContextProperty(prop, contextID, false))
		}
	}

	if inferenceService.GetCustomProperties() != nil {
		for _, prop := range *inferenceService.GetCustomProperties() {
			properties = append(properties, MapPropertiesToContextProperty(prop, contextID, true))
		}
	}

	return properties
}

func mapDataLayerToInferenceService(infSvcCtx schema.Context, propertiesCtx []schema.ContextProperty) models.InferenceService {
	inferenceServiceModel := &models.BaseEntity[models.InferenceServiceAttributes]{
		ID:     &infSvcCtx.ID,
		TypeID: &infSvcCtx.TypeID,
		Attributes: &models.InferenceServiceAttributes{
			Name:                     &infSvcCtx.Name,
			ExternalID:               infSvcCtx.ExternalID,
			CreateTimeSinceEpoch:     &infSvcCtx.CreateTimeSinceEpoch,
			LastUpdateTimeSinceEpoch: &infSvcCtx.LastUpdateTimeSinceEpoch,
		},
	}

	properties := []models.Properties{}
	customProperties := []models.Properties{}

	for _, prop := range propertiesCtx {
		mappedProperty := MapContextPropertyToProperties(prop)

		if prop.IsCustomProperty {
			customProperties = append(customProperties, mappedProperty)
		} else {
			properties = append(properties, mappedProperty)
		}
	}

	// Always set Properties and CustomProperties, even if empty
	inferenceServiceModel.Properties = &properties
	inferenceServiceModel.CustomProperties = &customProperties

	return inferenceServiceModel
}
