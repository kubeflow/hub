package service

import (
	"errors"
	"fmt"

	"github.com/kubeflow/hub/internal/db/models"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"github.com/kubeflow/hub/internal/platform/db/utils"
	"gorm.io/gorm"
)

var ErrModelVersionNotFound = errors.New("model version by id not found")

type ModelVersionRepositoryImpl struct {
	*GenericRepository[models.ModelVersion, schema.Context, schema.ContextProperty, *models.ModelVersionListOptions]
}

func NewModelVersionRepository(db *gorm.DB, typeID int32) models.ModelVersionRepository {
	config := GenericRepositoryConfig[models.ModelVersion, schema.Context, schema.ContextProperty, *models.ModelVersionListOptions]{
		DB:                  db,
		TypeID:              typeID,
		EntityToSchema:      mapModelVersionToContext,
		SchemaToEntity:      mapDataLayerToModelVersion,
		EntityToProperties:  mapModelVersionToContextProperties,
		NotFoundError:       ErrModelVersionNotFound,
		EntityName:          "model version",
		PropertyFieldName:   "context_id",
		ApplyListFilters:    applyModelVersionListFilters,
		IsNewEntity:         func(entity models.ModelVersion) bool { return entity.GetID() == nil },
		HasCustomProperties: func(entity models.ModelVersion) bool { return entity.GetCustomProperties() != nil },
	}

	return &ModelVersionRepositoryImpl{
		GenericRepository: NewGenericRepository(config),
	}
}

func (r *ModelVersionRepositoryImpl) Save(modelVersion models.ModelVersion) (models.ModelVersion, error) {
	// Extract registered_model_id from properties for parent relationship
	var registeredModelID *int32
	if modelVersion.GetProperties() != nil {
		for _, prop := range *modelVersion.GetProperties() {
			if prop.Name == "registered_model_id" && prop.IntValue != nil {
				registeredModelID = prop.IntValue
				break
			}
		}
	}
	return r.GenericRepository.Save(modelVersion, registeredModelID)
}

func (r *ModelVersionRepositoryImpl) List(listOptions models.ModelVersionListOptions) (*models.ListWrapper[models.ModelVersion], error) {
	return r.GenericRepository.List(&listOptions)
}

func applyModelVersionListFilters(query *gorm.DB, listOptions *models.ModelVersionListOptions) *gorm.DB {
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
		// Proper GORM JOIN: Use helper that respects naming strategy
		query = query.Joins(utils.BuildParentContextJoin(query)).
			Where(utils.GetColumnRef(query, &schema.ParentContext{}, "parent_context_id")+" = ?", listOptions.ParentResourceID)
	}

	return query
}

func mapModelVersionToContext(modelVersion models.ModelVersion) schema.Context {
	attrs := modelVersion.GetAttributes()
	context := schema.Context{
		TypeID: *modelVersion.GetTypeID(),
	}

	// Only set ID if it's not nil (for existing entities)
	if modelVersion.GetID() != nil {
		context.ID = *modelVersion.GetID()
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

func mapModelVersionToContextProperties(modelVersion models.ModelVersion, contextID int32) []schema.ContextProperty {
	var properties []schema.ContextProperty

	if modelVersion.GetProperties() != nil {
		for _, prop := range *modelVersion.GetProperties() {
			properties = append(properties, MapPropertiesToContextProperty(prop, contextID, false))
		}
	}

	if modelVersion.GetCustomProperties() != nil {
		for _, prop := range *modelVersion.GetCustomProperties() {
			properties = append(properties, MapPropertiesToContextProperty(prop, contextID, true))
		}
	}

	return properties
}

func mapDataLayerToModelVersion(modelVersionCtx schema.Context, propertiesCtx []schema.ContextProperty) models.ModelVersion {
	modelVersionModel := &models.BaseEntity[models.ModelVersionAttributes]{
		ID:     &modelVersionCtx.ID,
		TypeID: &modelVersionCtx.TypeID,
		Attributes: &models.ModelVersionAttributes{
			Name:                     &modelVersionCtx.Name,
			ExternalID:               modelVersionCtx.ExternalID,
			CreateTimeSinceEpoch:     &modelVersionCtx.CreateTimeSinceEpoch,
			LastUpdateTimeSinceEpoch: &modelVersionCtx.LastUpdateTimeSinceEpoch,
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
	modelVersionModel.Properties = &properties
	modelVersionModel.CustomProperties = &customProperties

	return modelVersionModel
}
