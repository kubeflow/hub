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

var ErrParameterNotFound = errors.New("parameter by id not found")

type ParameterRepositoryImpl struct {
	*GenericRepository[models.Parameter, schema.Artifact, schema.ArtifactProperty, *models.ParameterListOptions]
}

func NewParameterRepository(db *gorm.DB, typeID int32) models.ParameterRepository {
	config := GenericRepositoryConfig[models.Parameter, schema.Artifact, schema.ArtifactProperty, *models.ParameterListOptions]{
		DB:                  db,
		TypeID:              typeID,
		EntityToSchema:      mapParameterToArtifact,
		SchemaToEntity:      mapDataLayerToParameter,
		EntityToProperties:  mapParameterToArtifactProperties,
		NotFoundError:       ErrParameterNotFound,
		EntityName:          "parameter",
		PropertyFieldName:   "artifact_id",
		ApplyListFilters:    applyParameterListFilters,
		IsNewEntity:         func(entity models.Parameter) bool { return entity.GetID() == nil },
		HasCustomProperties: func(entity models.Parameter) bool { return entity.GetCustomProperties() != nil },
	}

	return &ParameterRepositoryImpl{
		GenericRepository: NewGenericRepository(config),
	}
}

// List adapts the generic repository List method to match the interface contract
func (r *ParameterRepositoryImpl) List(listOptions models.ParameterListOptions) (*models.ListWrapper[models.Parameter], error) {
	return r.GenericRepository.List(&listOptions)
}

func applyParameterListFilters(query *gorm.DB, listOptions *models.ParameterListOptions) *gorm.DB {
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

func mapParameterToArtifact(parameter models.Parameter) schema.Artifact {
	if parameter == nil {
		return schema.Artifact{}
	}

	artifact := schema.Artifact{
		ID:     apiutils.ZeroIfNil(parameter.GetID()),
		TypeID: apiutils.ZeroIfNil(parameter.GetTypeID()),
	}

	if parameter.GetAttributes() != nil {
		artifact.Name = parameter.GetAttributes().Name
		artifact.URI = parameter.GetAttributes().URI
		artifact.ExternalID = parameter.GetAttributes().ExternalID
		if parameter.GetAttributes().State != nil {
			stateValue := models.Artifact_State_value[*parameter.GetAttributes().State]
			artifact.State = &stateValue
		}
		artifact.CreateTimeSinceEpoch = apiutils.ZeroIfNil(parameter.GetAttributes().CreateTimeSinceEpoch)
		artifact.LastUpdateTimeSinceEpoch = apiutils.ZeroIfNil(parameter.GetAttributes().LastUpdateTimeSinceEpoch)
	}

	return artifact
}

func mapParameterToArtifactProperties(parameter models.Parameter, artifactID int32) []schema.ArtifactProperty {
	if parameter == nil {
		return []schema.ArtifactProperty{}
	}

	properties := []schema.ArtifactProperty{}

	if parameter.GetProperties() != nil {
		for _, prop := range *parameter.GetProperties() {
			properties = append(properties, MapPropertiesToArtifactProperty(prop, artifactID, false))
		}
	}

	if parameter.GetCustomProperties() != nil {
		for _, prop := range *parameter.GetCustomProperties() {
			properties = append(properties, MapPropertiesToArtifactProperty(prop, artifactID, true))
		}
	}

	return properties
}

func mapDataLayerToParameter(parameter schema.Artifact, artProperties []schema.ArtifactProperty) models.Parameter {
	var state *string

	if parameter.State != nil {
		parameterState := models.Artifact_State_name[*parameter.State]
		state = &parameterState
	}

	parameterArt := models.ParameterImpl{
		ID:     &parameter.ID,
		TypeID: &parameter.TypeID,
		Attributes: &models.ParameterAttributes{
			Name:                     parameter.Name,
			URI:                      parameter.URI,
			State:                    state,
			ArtifactType:             apiutils.StrPtr(models.ParameterType),
			ExternalID:               parameter.ExternalID,
			CreateTimeSinceEpoch:     &parameter.CreateTimeSinceEpoch,
			LastUpdateTimeSinceEpoch: &parameter.LastUpdateTimeSinceEpoch,
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

	parameterArt.CustomProperties = &customProperties
	parameterArt.Properties = &properties

	return &parameterArt
}
