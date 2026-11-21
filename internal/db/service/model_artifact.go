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

var ErrModelArtifactNotFound = errors.New("model artifact by id not found")

type ModelArtifactRepositoryImpl struct {
	*GenericRepository[models.ModelArtifact, schema.Artifact, schema.ArtifactProperty, *models.ModelArtifactListOptions]
}

func NewModelArtifactRepository(db *gorm.DB, typeID int32) models.ModelArtifactRepository {
	config := GenericRepositoryConfig[models.ModelArtifact, schema.Artifact, schema.ArtifactProperty, *models.ModelArtifactListOptions]{
		DB:                  db,
		TypeID:              typeID,
		EntityToSchema:      mapModelArtifactToArtifact,
		SchemaToEntity:      mapDataLayerToModelArtifact,
		EntityToProperties:  mapModelArtifactToArtifactProperties,
		NotFoundError:       ErrModelArtifactNotFound,
		EntityName:          "model artifact",
		PropertyFieldName:   "artifact_id",
		ApplyListFilters:    applyModelArtifactListFilters,
		IsNewEntity:         func(entity models.ModelArtifact) bool { return entity.GetID() == nil },
		HasCustomProperties: func(entity models.ModelArtifact) bool { return entity.GetCustomProperties() != nil },
	}

	return &ModelArtifactRepositoryImpl{
		GenericRepository: NewGenericRepository(config),
	}
}

// List adapts the generic repository List method to match the interface contract
func (r *ModelArtifactRepositoryImpl) List(listOptions models.ModelArtifactListOptions) (*models.ListWrapper[models.ModelArtifact], error) {
	return r.GenericRepository.List(&listOptions)
}

func applyModelArtifactListFilters(query *gorm.DB, listOptions *models.ModelArtifactListOptions) *gorm.DB {
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

func mapModelArtifactToArtifact(modelArtifact models.ModelArtifact) schema.Artifact {
	if modelArtifact == nil {
		return schema.Artifact{}
	}

	artifact := schema.Artifact{
		ID:     apiutils.ZeroIfNil(modelArtifact.GetID()),
		TypeID: apiutils.ZeroIfNil(modelArtifact.GetTypeID()),
	}

	if modelArtifact.GetAttributes() != nil {
		artifact.Name = modelArtifact.GetAttributes().Name
		artifact.URI = modelArtifact.GetAttributes().URI
		artifact.ExternalID = modelArtifact.GetAttributes().ExternalID
		if modelArtifact.GetAttributes().State != nil {
			stateValue := models.Artifact_State_value[*modelArtifact.GetAttributes().State]
			artifact.State = &stateValue
		}
		artifact.CreateTimeSinceEpoch = apiutils.ZeroIfNil(modelArtifact.GetAttributes().CreateTimeSinceEpoch)
		artifact.LastUpdateTimeSinceEpoch = apiutils.ZeroIfNil(modelArtifact.GetAttributes().LastUpdateTimeSinceEpoch)
	}

	return artifact
}

func mapModelArtifactToArtifactProperties(modelArtifact models.ModelArtifact, artifactID int32) []schema.ArtifactProperty {
	if modelArtifact == nil {
		return []schema.ArtifactProperty{}
	}

	properties := []schema.ArtifactProperty{}

	if modelArtifact.GetProperties() != nil {
		for _, prop := range *modelArtifact.GetProperties() {
			properties = append(properties, MapPropertiesToArtifactProperty(prop, artifactID, false))
		}
	}

	if modelArtifact.GetCustomProperties() != nil {
		for _, prop := range *modelArtifact.GetCustomProperties() {
			properties = append(properties, MapPropertiesToArtifactProperty(prop, artifactID, true))
		}
	}

	return properties
}

func mapDataLayerToModelArtifact(modelArtifact schema.Artifact, artProperties []schema.ArtifactProperty) models.ModelArtifact {
	var state *string
	modelArtifactType := models.ModelArtifactType

	if modelArtifact.State != nil {
		docState := models.Artifact_State_name[*modelArtifact.State]
		state = &docState
	}

	modelArtifactArt := models.ModelArtifactImpl{
		ID:     &modelArtifact.ID,
		TypeID: &modelArtifact.TypeID,
		Attributes: &models.ModelArtifactAttributes{
			Name:                     modelArtifact.Name,
			URI:                      modelArtifact.URI,
			State:                    state,
			ArtifactType:             &modelArtifactType,
			ExternalID:               modelArtifact.ExternalID,
			CreateTimeSinceEpoch:     &modelArtifact.CreateTimeSinceEpoch,
			LastUpdateTimeSinceEpoch: &modelArtifact.LastUpdateTimeSinceEpoch,
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

	modelArtifactArt.CustomProperties = &customProperties
	modelArtifactArt.Properties = &properties

	return &modelArtifactArt
}
