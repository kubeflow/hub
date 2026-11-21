package service

import (
	"errors"
	"fmt"

	"github.com/kubeflow/hub/internal/db/models"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"github.com/kubeflow/hub/internal/platform/db/utils"
	"gorm.io/gorm"
)

var ErrExperimentRunNotFound = errors.New("experiment run by id not found")

type ExperimentRunRepositoryImpl struct {
	*GenericRepository[models.ExperimentRun, schema.Context, schema.ContextProperty, *models.ExperimentRunListOptions]
}

func NewExperimentRunRepository(db *gorm.DB, typeID int32) models.ExperimentRunRepository {
	config := GenericRepositoryConfig[models.ExperimentRun, schema.Context, schema.ContextProperty, *models.ExperimentRunListOptions]{
		DB:                  db,
		TypeID:              typeID,
		EntityToSchema:      mapExperimentRunToContext,
		SchemaToEntity:      mapDataLayerToExperimentRun,
		EntityToProperties:  mapExperimentRunToContextProperties,
		NotFoundError:       ErrExperimentRunNotFound,
		EntityName:          "experiment run",
		PropertyFieldName:   "context_id",
		ApplyListFilters:    applyExperimentRunListFilters,
		IsNewEntity:         func(entity models.ExperimentRun) bool { return entity.GetID() == nil },
		HasCustomProperties: func(entity models.ExperimentRun) bool { return entity.GetCustomProperties() != nil },
	}

	return &ExperimentRunRepositoryImpl{
		GenericRepository: NewGenericRepository(config),
	}
}

func (r *ExperimentRunRepositoryImpl) Save(experimentRun models.ExperimentRun, experimentID *int32) (models.ExperimentRun, error) {
	return r.GenericRepository.Save(experimentRun, experimentID)
}

func (r *ExperimentRunRepositoryImpl) List(listOptions models.ExperimentRunListOptions) (*models.ListWrapper[models.ExperimentRun], error) {
	return r.GenericRepository.List(&listOptions)
}

func applyExperimentRunListFilters(query *gorm.DB, listOptions *models.ExperimentRunListOptions) *gorm.DB {
	if listOptions.Name != nil {
		if listOptions.ExperimentID != nil {
			query = query.Where("name LIKE ?", fmt.Sprintf("%d:%s", *listOptions.ExperimentID, escapeLike(*listOptions.Name)))
		} else {
			query = query.Where("name LIKE ?", fmt.Sprintf("%%:%s", escapeLike(*listOptions.Name)))
		}
	} else if listOptions.ExternalID != nil {
		query = query.Where("external_id = ?", listOptions.ExternalID)
	}

	if listOptions.ExperimentID != nil {
		// Proper GORM JOIN: Use helper that respects naming strategy
		query = query.Joins(utils.BuildParentContextJoin(query)).
			Where(utils.GetColumnRef(query, &schema.ParentContext{}, "parent_context_id")+" = ?", listOptions.ExperimentID)
	}

	return query
}

func mapExperimentRunToContext(experimentRun models.ExperimentRun) schema.Context {
	attrs := experimentRun.GetAttributes()
	context := schema.Context{
		TypeID: *experimentRun.GetTypeID(),
	}

	// Only set ID if it's not nil (for existing entities)
	if experimentRun.GetID() != nil {
		context.ID = *experimentRun.GetID()
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

func mapExperimentRunToContextProperties(experimentRun models.ExperimentRun, contextID int32) []schema.ContextProperty {
	var properties []schema.ContextProperty

	if experimentRun.GetProperties() != nil {
		for _, prop := range *experimentRun.GetProperties() {
			properties = append(properties, MapPropertiesToContextProperty(prop, contextID, false))
		}
	}

	if experimentRun.GetCustomProperties() != nil {
		for _, prop := range *experimentRun.GetCustomProperties() {
			properties = append(properties, MapPropertiesToContextProperty(prop, contextID, true))
		}
	}

	return properties
}

func mapDataLayerToExperimentRun(expRunCtx schema.Context, propertiesCtx []schema.ContextProperty) models.ExperimentRun {
	experimentRunModel := &models.BaseEntity[models.ExperimentRunAttributes]{
		ID:     &expRunCtx.ID,
		TypeID: &expRunCtx.TypeID,
		Attributes: &models.ExperimentRunAttributes{
			Name:                     &expRunCtx.Name,
			ExternalID:               expRunCtx.ExternalID,
			CreateTimeSinceEpoch:     &expRunCtx.CreateTimeSinceEpoch,
			LastUpdateTimeSinceEpoch: &expRunCtx.LastUpdateTimeSinceEpoch,
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
	experimentRunModel.Properties = &properties
	experimentRunModel.CustomProperties = &customProperties

	return experimentRunModel
}
