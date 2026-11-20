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

var ErrDocArtifactNotFound = errors.New("doc artifact by id not found")

type DocArtifactRepositoryImpl struct {
	*GenericRepository[models.DocArtifact, schema.Artifact, schema.ArtifactProperty, *models.DocArtifactListOptions]
}

func NewDocArtifactRepository(db *gorm.DB, typeID int32) models.DocArtifactRepository {
	config := GenericRepositoryConfig[models.DocArtifact, schema.Artifact, schema.ArtifactProperty, *models.DocArtifactListOptions]{
		DB:                  db,
		TypeID:              typeID,
		EntityToSchema:      mapDocArtifactToArtifact,
		SchemaToEntity:      mapDataLayerToDocArtifact,
		EntityToProperties:  mapDocArtifactToArtifactProperties,
		NotFoundError:       ErrDocArtifactNotFound,
		EntityName:          "doc artifact",
		PropertyFieldName:   "artifact_id",
		ApplyListFilters:    applyDocArtifactListFilters,
		IsNewEntity:         func(entity models.DocArtifact) bool { return entity.GetID() == nil },
		HasCustomProperties: func(entity models.DocArtifact) bool { return entity.GetCustomProperties() != nil },
	}

	return &DocArtifactRepositoryImpl{
		GenericRepository: NewGenericRepository(config),
	}
}

func (r *DocArtifactRepositoryImpl) List(listOptions models.DocArtifactListOptions) (*models.ListWrapper[models.DocArtifact], error) {
	return r.GenericRepository.List(&listOptions)
}

func applyDocArtifactListFilters(query *gorm.DB, listOptions *models.DocArtifactListOptions) *gorm.DB {
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

func mapDocArtifactToArtifact(docArtifact models.DocArtifact) schema.Artifact {
	if docArtifact == nil {
		return schema.Artifact{}
	}

	artifact := schema.Artifact{
		ID:     apiutils.ZeroIfNil(docArtifact.GetID()),
		TypeID: apiutils.ZeroIfNil(docArtifact.GetTypeID()),
	}

	if docArtifact.GetAttributes() != nil {
		attrs := docArtifact.GetAttributes()
		artifact.Name = attrs.Name
		artifact.ExternalID = attrs.ExternalID
		artifact.URI = attrs.URI
		// Handle State conversion - DocArtifact uses string, schema.Artifact uses int32
		if attrs.State != nil {
			stateValue := models.Artifact_State_value[*attrs.State]
			artifact.State = &stateValue
		}
		artifact.CreateTimeSinceEpoch = apiutils.ZeroIfNil(attrs.CreateTimeSinceEpoch)
		artifact.LastUpdateTimeSinceEpoch = apiutils.ZeroIfNil(attrs.LastUpdateTimeSinceEpoch)
	}

	return artifact
}

func mapDocArtifactToArtifactProperties(docArtifact models.DocArtifact, artifactID int32) []schema.ArtifactProperty {
	var properties []schema.ArtifactProperty

	if docArtifact.GetProperties() != nil {
		for _, prop := range *docArtifact.GetProperties() {
			properties = append(properties, MapPropertiesToArtifactProperty(prop, artifactID, false))
		}
	}

	if docArtifact.GetCustomProperties() != nil {
		for _, prop := range *docArtifact.GetCustomProperties() {
			properties = append(properties, MapPropertiesToArtifactProperty(prop, artifactID, true))
		}
	}

	return properties
}

func mapDataLayerToDocArtifact(docArtifact schema.Artifact, artProperties []schema.ArtifactProperty) models.DocArtifact {
	var state *string
	if docArtifact.State != nil {
		docState := models.Artifact_State_name[*docArtifact.State]
		state = &docState
	}

	docArtifactModel := &models.BaseEntity[models.DocArtifactAttributes]{
		ID:     &docArtifact.ID,
		TypeID: &docArtifact.TypeID,
		Attributes: &models.DocArtifactAttributes{
			Name:                     docArtifact.Name,
			ExternalID:               docArtifact.ExternalID,
			URI:                      docArtifact.URI,
			State:                    state,
			CreateTimeSinceEpoch:     &docArtifact.CreateTimeSinceEpoch,
			LastUpdateTimeSinceEpoch: &docArtifact.LastUpdateTimeSinceEpoch,
		},
	}

	properties := []models.Properties{}
	customProperties := []models.Properties{}

	for _, prop := range artProperties {
		mappedProperty := MapArtifactPropertyToProperties(prop)

		if prop.IsCustomProperty {
			customProperties = append(customProperties, mappedProperty)
		} else {
			properties = append(properties, mappedProperty)
		}
	}

	// Always set Properties and CustomProperties, even if empty
	docArtifactModel.Properties = &properties
	docArtifactModel.CustomProperties = &customProperties

	return docArtifactModel
}
