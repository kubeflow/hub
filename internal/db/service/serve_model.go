package service

import (
	"errors"
	"fmt"

	"github.com/kubeflow/hub/internal/db/models"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"github.com/kubeflow/hub/internal/platform/db/utils"
	"gorm.io/gorm"
)

var ErrServeModelNotFound = errors.New("serve model by id not found")

type ServeModelRepositoryImpl struct {
	*GenericRepository[models.ServeModel, schema.Execution, schema.ExecutionProperty, *models.ServeModelListOptions]
}

func NewServeModelRepository(db *gorm.DB, typeID int32) models.ServeModelRepository {
	config := GenericRepositoryConfig[models.ServeModel, schema.Execution, schema.ExecutionProperty, *models.ServeModelListOptions]{
		DB:                  db,
		TypeID:              typeID,
		EntityToSchema:      mapServeModelToExecution,
		SchemaToEntity:      mapDataLayerToServeModel,
		EntityToProperties:  mapServeModelToExecutionProperties,
		NotFoundError:       ErrServeModelNotFound,
		EntityName:          "serve model",
		PropertyFieldName:   "execution_id",
		ApplyListFilters:    applyServeModelListFilters,
		IsNewEntity:         func(entity models.ServeModel) bool { return entity.GetID() == nil },
		HasCustomProperties: func(entity models.ServeModel) bool { return entity.GetCustomProperties() != nil },
	}

	return &ServeModelRepositoryImpl{
		GenericRepository: NewGenericRepository(config),
	}
}

func (r *ServeModelRepositoryImpl) Save(serveModel models.ServeModel, inferenceServiceID *int32) (models.ServeModel, error) {
	return r.GenericRepository.Save(serveModel, inferenceServiceID)
}

func (r *ServeModelRepositoryImpl) List(listOptions models.ServeModelListOptions) (*models.ListWrapper[models.ServeModel], error) {
	return r.GenericRepository.List(&listOptions)
}

func applyServeModelListFilters(query *gorm.DB, listOptions *models.ServeModelListOptions) *gorm.DB {
	if listOptions.Name != nil {
		query = query.Where(utils.GetTableName(query, &schema.Execution{})+".name LIKE ?", fmt.Sprintf("%%:%s", escapeLike(*listOptions.Name)))
	} else if listOptions.ExternalID != nil {
		query = query.Where(utils.GetTableName(query, &schema.Execution{})+".external_id = ?", listOptions.ExternalID)
	}

	if listOptions.InferenceServiceID != nil {
		// Proper GORM JOIN: Use helper that respects naming strategy
		query = query.Joins(utils.BuildAssociationJoin(query)).
			Where(utils.GetColumnRef(query, &schema.Association{}, "context_id")+" = ?", listOptions.InferenceServiceID)
	}

	return query
}

func mapServeModelToExecution(serveModel models.ServeModel) schema.Execution {
	attrs := serveModel.GetAttributes()
	execution := schema.Execution{
		TypeID: *serveModel.GetTypeID(),
	}

	// Only set ID if it's not nil (for existing entities)
	if serveModel.GetID() != nil {
		execution.ID = *serveModel.GetID()
	}

	if attrs != nil {
		execution.Name = attrs.Name
		execution.ExternalID = attrs.ExternalID
		// Handle LastKnownState conversion - ServeModel uses string, schema.Execution uses int32
		if attrs.LastKnownState != nil {
			stateValue := models.Execution_State_value[*attrs.LastKnownState]
			execution.LastKnownState = &stateValue
		}
		if attrs.CreateTimeSinceEpoch != nil {
			execution.CreateTimeSinceEpoch = *attrs.CreateTimeSinceEpoch
		}
		if attrs.LastUpdateTimeSinceEpoch != nil {
			execution.LastUpdateTimeSinceEpoch = *attrs.LastUpdateTimeSinceEpoch
		}
	}

	return execution
}

func mapServeModelToExecutionProperties(serveModel models.ServeModel, executionID int32) []schema.ExecutionProperty {
	var properties []schema.ExecutionProperty

	if serveModel.GetProperties() != nil {
		for _, prop := range *serveModel.GetProperties() {
			properties = append(properties, MapPropertiesToExecutionProperty(prop, executionID, false))
		}
	}

	if serveModel.GetCustomProperties() != nil {
		for _, prop := range *serveModel.GetCustomProperties() {
			properties = append(properties, MapPropertiesToExecutionProperty(prop, executionID, true))
		}
	}

	return properties
}

func mapDataLayerToServeModel(serveModel schema.Execution, properties []schema.ExecutionProperty) models.ServeModel {
	var lastKnownState *string
	if serveModel.LastKnownState != nil {
		stateStr := models.Execution_State_name[*serveModel.LastKnownState]
		lastKnownState = &stateStr
	}

	serveModelModel := &models.BaseEntity[models.ServeModelAttributes]{
		ID:     &serveModel.ID,
		TypeID: &serveModel.TypeID,
		Attributes: &models.ServeModelAttributes{
			Name:                     serveModel.Name,
			ExternalID:               serveModel.ExternalID,
			LastKnownState:           lastKnownState,
			CreateTimeSinceEpoch:     &serveModel.CreateTimeSinceEpoch,
			LastUpdateTimeSinceEpoch: &serveModel.LastUpdateTimeSinceEpoch,
		},
	}

	modelProperties := []models.Properties{}
	customProperties := []models.Properties{}

	for _, prop := range properties {
		mappedProperty := MapExecutionPropertyToProperties(prop)

		if prop.IsCustomProperty {
			customProperties = append(customProperties, mappedProperty)
		} else {
			modelProperties = append(modelProperties, mappedProperty)
		}
	}

	// Always set Properties and CustomProperties, even if empty
	serveModelModel.Properties = &modelProperties
	serveModelModel.CustomProperties = &customProperties

	return serveModelModel
}
