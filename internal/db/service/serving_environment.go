package service

import (
	"errors"

	"github.com/kubeflow/hub/internal/db/models"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"gorm.io/gorm"
)

var ErrServingEnvironmentNotFound = errors.New("serving environment by id not found")

type ServingEnvironmentRepositoryImpl struct {
	*GenericRepository[models.ServingEnvironment, schema.Context, schema.ContextProperty, *models.ServingEnvironmentListOptions]
}

func NewServingEnvironmentRepository(db *gorm.DB, typeID int32) models.ServingEnvironmentRepository {
	config := GenericRepositoryConfig[models.ServingEnvironment, schema.Context, schema.ContextProperty, *models.ServingEnvironmentListOptions]{
		DB:                  db,
		TypeID:              typeID,
		EntityToSchema:      mapServingEnvironmentToContext,
		SchemaToEntity:      mapDataLayerToServingEnvironment,
		EntityToProperties:  mapServingEnvironmentToContextProperties,
		NotFoundError:       ErrServingEnvironmentNotFound,
		EntityName:          "serving environment",
		PropertyFieldName:   "context_id",
		ApplyListFilters:    applyServingEnvironmentListFilters,
		IsNewEntity:         func(entity models.ServingEnvironment) bool { return entity.GetID() == nil },
		HasCustomProperties: func(entity models.ServingEnvironment) bool { return entity.GetCustomProperties() != nil },
	}

	return &ServingEnvironmentRepositoryImpl{
		GenericRepository: NewGenericRepository(config),
	}
}

func (r *ServingEnvironmentRepositoryImpl) Save(servEnv models.ServingEnvironment) (models.ServingEnvironment, error) {
	return r.GenericRepository.Save(servEnv, nil)
}

func (r *ServingEnvironmentRepositoryImpl) List(listOptions models.ServingEnvironmentListOptions) (*models.ListWrapper[models.ServingEnvironment], error) {
	return r.GenericRepository.List(&listOptions)
}

func applyServingEnvironmentListFilters(query *gorm.DB, listOptions *models.ServingEnvironmentListOptions) *gorm.DB {
	if listOptions.Name != nil {
		query = query.Where("name LIKE ?", escapeLike(*listOptions.Name))
	} else if listOptions.ExternalID != nil {
		query = query.Where("external_id = ?", listOptions.ExternalID)
	}
	return query
}

func mapServingEnvironmentToContext(servEnv models.ServingEnvironment) schema.Context {
	attrs := servEnv.GetAttributes()
	context := schema.Context{
		TypeID: *servEnv.GetTypeID(),
	}

	// Only set ID if it's not nil (for existing entities)
	if servEnv.GetID() != nil {
		context.ID = *servEnv.GetID()
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

func mapServingEnvironmentToContextProperties(servEnv models.ServingEnvironment, contextID int32) []schema.ContextProperty {
	var properties []schema.ContextProperty

	if servEnv.GetProperties() != nil {
		for _, prop := range *servEnv.GetProperties() {
			properties = append(properties, MapPropertiesToContextProperty(prop, contextID, false))
		}
	}

	if servEnv.GetCustomProperties() != nil {
		for _, prop := range *servEnv.GetCustomProperties() {
			properties = append(properties, MapPropertiesToContextProperty(prop, contextID, true))
		}
	}

	return properties
}

func mapDataLayerToServingEnvironment(modelCtx schema.Context, propertiesCtx []schema.ContextProperty) models.ServingEnvironment {
	servingEnvironmentModel := &models.BaseEntity[models.ServingEnvironmentAttributes]{
		ID:     &modelCtx.ID,
		TypeID: &modelCtx.TypeID,
		Attributes: &models.ServingEnvironmentAttributes{
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
	servingEnvironmentModel.Properties = &properties
	servingEnvironmentModel.CustomProperties = &customProperties

	return servingEnvironmentModel
}
