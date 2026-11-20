package service

import (
	"errors"

	"github.com/kubeflow/hub/internal/db/models"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"gorm.io/gorm"
)

var ErrRegisteredModelNotFound = errors.New("registered model by id not found")

type RegisteredModelRepositoryImpl struct {
	*GenericRepository[models.RegisteredModel, schema.Context, schema.ContextProperty, *models.RegisteredModelListOptions]
}

func NewRegisteredModelRepository(db *gorm.DB, typeID int32) models.RegisteredModelRepository {
	config := GenericRepositoryConfig[models.RegisteredModel, schema.Context, schema.ContextProperty, *models.RegisteredModelListOptions]{
		DB:                  db,
		TypeID:              typeID,
		EntityToSchema:      mapRegisteredModelToContext,
		SchemaToEntity:      mapDataLayerToRegisteredModel,
		EntityToProperties:  mapRegisteredModelToContextProperties,
		NotFoundError:       ErrRegisteredModelNotFound,
		EntityName:          "registered model",
		PropertyFieldName:   "context_id",
		ApplyListFilters:    applyRegisteredModelListFilters,
		IsNewEntity:         func(entity models.RegisteredModel) bool { return entity.GetID() == nil },
		HasCustomProperties: func(entity models.RegisteredModel) bool { return entity.GetCustomProperties() != nil },
	}

	return &RegisteredModelRepositoryImpl{
		GenericRepository: NewGenericRepository(config),
	}
}

func (r *RegisteredModelRepositoryImpl) Save(model models.RegisteredModel) (models.RegisteredModel, error) {
	return r.GenericRepository.Save(model, nil)
}

func (r *RegisteredModelRepositoryImpl) List(listOptions models.RegisteredModelListOptions) (*models.ListWrapper[models.RegisteredModel], error) {
	return r.GenericRepository.List(&listOptions)
}

func applyRegisteredModelListFilters(query *gorm.DB, listOptions *models.RegisteredModelListOptions) *gorm.DB {
	if listOptions.Name != nil {
		query = query.Where("name LIKE ?", escapeLike(*listOptions.Name))
	} else if listOptions.ExternalID != nil {
		query = query.Where("external_id = ?", listOptions.ExternalID)
	}
	return query
}

func mapRegisteredModelToContext(model models.RegisteredModel) schema.Context {
	attrs := model.GetAttributes()
	context := schema.Context{
		TypeID: *model.GetTypeID(),
	}

	// Only set ID if it's not nil (for existing entities)
	if model.GetID() != nil {
		context.ID = *model.GetID()
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

func mapRegisteredModelToContextProperties(model models.RegisteredModel, contextID int32) []schema.ContextProperty {
	var properties []schema.ContextProperty

	if model.GetProperties() != nil {
		for _, prop := range *model.GetProperties() {
			properties = append(properties, MapPropertiesToContextProperty(prop, contextID, false))
		}
	}

	if model.GetCustomProperties() != nil {
		for _, prop := range *model.GetCustomProperties() {
			properties = append(properties, MapPropertiesToContextProperty(prop, contextID, true))
		}
	}

	return properties
}

func mapDataLayerToRegisteredModel(modelCtx schema.Context, propertiesCtx []schema.ContextProperty) models.RegisteredModel {
	registeredModelModel := &models.BaseEntity[models.RegisteredModelAttributes]{
		ID:     &modelCtx.ID,
		TypeID: &modelCtx.TypeID,
		Attributes: &models.RegisteredModelAttributes{
			Name:                     &modelCtx.Name,
			ExternalID:               modelCtx.ExternalID,
			CreateTimeSinceEpoch:     &modelCtx.CreateTimeSinceEpoch,
			LastUpdateTimeSinceEpoch: &modelCtx.LastUpdateTimeSinceEpoch,
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
	registeredModelModel.Properties = &properties
	registeredModelModel.CustomProperties = &customProperties

	return registeredModelModel
}
