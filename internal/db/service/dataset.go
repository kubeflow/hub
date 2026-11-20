package service

import (
	"errors"
	"fmt"

	"github.com/kubeflow/hub/internal/platform/apiutils"
	"github.com/kubeflow/hub/internal/db/models"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"github.com/kubeflow/hub/internal/platform/db/utils"
	"gorm.io/gorm"
)

var ErrDataSetNotFound = errors.New("dataset by id not found")

type DataSetRepositoryImpl struct {
	*GenericRepository[models.DataSet, schema.Artifact, schema.ArtifactProperty, *models.DataSetListOptions]
}

func NewDataSetRepository(db *gorm.DB, typeID int32) models.DataSetRepository {
	config := GenericRepositoryConfig[models.DataSet, schema.Artifact, schema.ArtifactProperty, *models.DataSetListOptions]{
		DB:                  db,
		TypeID:              typeID,
		EntityToSchema:      mapDataSetToArtifact,
		SchemaToEntity:      mapDataLayerToDataSet,
		EntityToProperties:  mapDataSetToArtifactProperties,
		NotFoundError:       ErrDataSetNotFound,
		EntityName:          "dataset",
		PropertyFieldName:   "artifact_id",
		ApplyListFilters:    applyDataSetListFilters,
		IsNewEntity:         func(entity models.DataSet) bool { return entity.GetID() == nil },
		HasCustomProperties: func(entity models.DataSet) bool { return entity.GetCustomProperties() != nil },
	}

	return &DataSetRepositoryImpl{
		GenericRepository: NewGenericRepository(config),
	}
}

// List adapts the generic repository List method to match the interface contract
func (r *DataSetRepositoryImpl) List(listOptions models.DataSetListOptions) (*models.ListWrapper[models.DataSet], error) {
	return r.GenericRepository.List(&listOptions)
}

func applyDataSetListFilters(query *gorm.DB, listOptions *models.DataSetListOptions) *gorm.DB {
	if listOptions.Name != nil {
		query = query.Where("name LIKE ?", fmt.Sprintf("%%:%s", escapeLike(*listOptions.Name)))
	} else if listOptions.ExternalID != nil {
		query = query.Where("external_id = ?", listOptions.ExternalID)
	}

	if listOptions.ParentResourceID != nil {
		// Proper GORM JOIN: Use helper that respects naming strategy
		query = query.Joins(utils.BuildAttributionJoin(query)).
			Where(utils.GetColumnRef(query, &schema.Attribution{}, "context_id")+" = ?", listOptions.ParentResourceID)
	}

	return query
}

func mapDataSetToArtifact(dataSet models.DataSet) schema.Artifact {
	if dataSet == nil {
		return schema.Artifact{}
	}

	artifact := schema.Artifact{
		ID:     apiutils.ZeroIfNil(dataSet.GetID()),
		TypeID: apiutils.ZeroIfNil(dataSet.GetTypeID()),
	}

	if dataSet.GetAttributes() != nil {
		artifact.Name = dataSet.GetAttributes().Name
		artifact.URI = dataSet.GetAttributes().URI
		artifact.ExternalID = dataSet.GetAttributes().ExternalID
		if dataSet.GetAttributes().State != nil {
			stateValue := models.Artifact_State_value[*dataSet.GetAttributes().State]
			artifact.State = &stateValue
		}
		artifact.CreateTimeSinceEpoch = apiutils.ZeroIfNil(dataSet.GetAttributes().CreateTimeSinceEpoch)
		artifact.LastUpdateTimeSinceEpoch = apiutils.ZeroIfNil(dataSet.GetAttributes().LastUpdateTimeSinceEpoch)
	}

	return artifact
}

func mapDataSetToArtifactProperties(dataSet models.DataSet, artifactID int32) []schema.ArtifactProperty {
	if dataSet == nil {
		return []schema.ArtifactProperty{}
	}

	properties := []schema.ArtifactProperty{}

	if dataSet.GetProperties() != nil {
		for _, prop := range *dataSet.GetProperties() {
			properties = append(properties, MapPropertiesToArtifactProperty(prop, artifactID, false))
		}
	}

	if dataSet.GetCustomProperties() != nil {
		for _, prop := range *dataSet.GetCustomProperties() {
			properties = append(properties, MapPropertiesToArtifactProperty(prop, artifactID, true))
		}
	}

	return properties
}

func mapDataLayerToDataSet(dataSet schema.Artifact, artProperties []schema.ArtifactProperty) models.DataSet {
	var state *string
	dataSetType := models.DataSetType

	if dataSet.State != nil {
		dsState := models.Artifact_State_name[*dataSet.State]
		state = &dsState
	}

	dataSetArt := models.DataSetImpl{
		ID:     &dataSet.ID,
		TypeID: &dataSet.TypeID,
		Attributes: &models.DataSetAttributes{
			Name:                     dataSet.Name,
			URI:                      dataSet.URI,
			State:                    state,
			ArtifactType:             &dataSetType,
			ExternalID:               dataSet.ExternalID,
			CreateTimeSinceEpoch:     &dataSet.CreateTimeSinceEpoch,
			LastUpdateTimeSinceEpoch: &dataSet.LastUpdateTimeSinceEpoch,
		},
	}

	customProperties := []models.Properties{}
	properties := []models.Properties{}

	for _, prop := range artProperties {
		if prop.IsCustomProperty {
			customProperties = append(customProperties, MapArtifactPropertyToProperties(prop))
		} else {
			properties = append(properties, MapArtifactPropertyToProperties(prop))
		}
	}

	dataSetArt.CustomProperties = &customProperties
	dataSetArt.Properties = &properties

	return &dataSetArt
}
