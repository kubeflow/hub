package service

import (
	"errors"
	"fmt"

	"github.com/kubeflow/hub/catalog/internal/catalog/agentcatalog/models"
	sharedmodels "github.com/kubeflow/hub/catalog/internal/db/models"
	"github.com/kubeflow/hub/internal/platform/apiutils"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	service "github.com/kubeflow/hub/internal/platform/db/repository"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"github.com/kubeflow/hub/internal/platform/db/utils"
	"gorm.io/gorm"
)

const AgentTemplateArtifactTypeName = "kf.AgentTemplateArtifact"

func init() {
	sharedmodels.RegisterArtifactMapper(AgentTemplateArtifactTypeName, func(artifact schema.Artifact, properties []schema.ArtifactProperty) any {
		return mapSchemaToAgentTemplateArtifact(artifact, properties)
	})
}

var ErrAgentTemplateArtifactNotFound = errors.New("agent template artifact not found")

type AgentTemplateArtifactRepositoryImpl struct {
	*service.GenericRepository[models.AgentTemplateArtifact, schema.Artifact, schema.ArtifactProperty, *models.AgentTemplateArtifactListOptions]
}

func NewAgentTemplateArtifactRepository(db *gorm.DB, typeID int32) models.AgentTemplateArtifactRepository {
	config := service.GenericRepositoryConfig[models.AgentTemplateArtifact, schema.Artifact, schema.ArtifactProperty, *models.AgentTemplateArtifactListOptions]{
		DB:                      db,
		TypeID:                  typeID,
		EntityToSchema:          mapAgentTemplateArtifactToSchema,
		SchemaToEntity:          mapSchemaToAgentTemplateArtifact,
		EntityToProperties:      mapAgentTemplateArtifactToProperties,
		NotFoundError:           ErrAgentTemplateArtifactNotFound,
		EntityName:              "agent template artifact",
		PropertyFieldName:       "artifact_id",
		ApplyListFilters:        applyAgentTemplateArtifactListFilters,
		IsNewEntity:             func(entity models.AgentTemplateArtifact) bool { return entity.GetID() == nil },
		HasCustomProperties:     func(entity models.AgentTemplateArtifact) bool { return entity.GetCustomProperties() != nil },
		PreserveHistoricalTimes: true,
	}

	return &AgentTemplateArtifactRepositoryImpl{
		GenericRepository: service.NewGenericRepository(config),
	}
}

func (r *AgentTemplateArtifactRepositoryImpl) Save(artifact models.AgentTemplateArtifact, parentResourceID *int32) (models.AgentTemplateArtifact, error) {
	config := r.GetConfig()
	if artifact.GetTypeID() == nil && config.TypeID > 0 {
		artifact.SetTypeID(config.TypeID)
	}

	attr := artifact.GetAttributes()
	if artifact.GetID() == nil && attr != nil && attr.Name != nil {
		existing, err := r.lookupByName(*attr.Name)
		if err != nil {
			if !errors.Is(err, ErrAgentTemplateArtifactNotFound) {
				return nil, fmt.Errorf("error finding existing agent template artifact named %s: %w", *attr.Name, err)
			}
		} else {
			artifact.SetID(existing.ID)
		}
	}

	return r.GenericRepository.Save(artifact, parentResourceID)
}

func (r *AgentTemplateArtifactRepositoryImpl) List(listOptions models.AgentTemplateArtifactListOptions) (*dbmodels.ListWrapper[models.AgentTemplateArtifact], error) {
	return r.GenericRepository.List(&listOptions)
}

func (r *AgentTemplateArtifactRepositoryImpl) DeleteByParentID(parentID int32) error {
	config := r.GetConfig()
	return config.DB.Exec(
		`DELETE FROM "Artifact" WHERE id IN (SELECT artifact_id FROM "Attribution" INNER JOIN "Artifact" ON "Artifact".id = artifact_id WHERE context_id = ? AND type_id = ?)`,
		parentID, config.TypeID,
	).Error
}

func (r *AgentTemplateArtifactRepositoryImpl) lookupByName(name string) (*schema.Artifact, error) {
	config := r.GetConfig()
	var entity schema.Artifact
	if err := config.DB.Where("name = ? AND type_id = ?", name, config.TypeID).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: name=%s", ErrAgentTemplateArtifactNotFound, name)
		}
		return nil, fmt.Errorf("error querying agent template artifact by name: %w", err)
	}
	return &entity, nil
}

func applyAgentTemplateArtifactListFilters(query *gorm.DB, listOptions *models.AgentTemplateArtifactListOptions) *gorm.DB {
	if listOptions.Name != nil {
		query = query.Where("name LIKE ?", fmt.Sprintf("%%:%s", *listOptions.Name))
	}

	if listOptions.ParentResourceID != nil {
		query = query.Joins(utils.BuildAttributionJoin(query)).
			Where(utils.GetColumnRef(query, &schema.Attribution{}, "context_id")+" = ?", listOptions.ParentResourceID)
	}

	return query
}

func mapAgentTemplateArtifactToSchema(artifact models.AgentTemplateArtifact) schema.Artifact {
	if artifact == nil {
		return schema.Artifact{}
	}

	s := schema.Artifact{
		ID:     apiutils.ZeroIfNil(artifact.GetID()),
		TypeID: apiutils.ZeroIfNil(artifact.GetTypeID()),
	}

	if artifact.GetAttributes() != nil {
		s.Name = artifact.GetAttributes().Name
		s.ExternalID = artifact.GetAttributes().ExternalID
		s.CreateTimeSinceEpoch = apiutils.ZeroIfNil(artifact.GetAttributes().CreateTimeSinceEpoch)
		s.LastUpdateTimeSinceEpoch = apiutils.ZeroIfNil(artifact.GetAttributes().LastUpdateTimeSinceEpoch)
	}

	return s
}

func mapAgentTemplateArtifactToProperties(artifact models.AgentTemplateArtifact, artifactID int32) []schema.ArtifactProperty {
	if artifact == nil {
		return []schema.ArtifactProperty{}
	}

	var properties []schema.ArtifactProperty
	if artifact.GetProperties() != nil {
		for _, prop := range *artifact.GetProperties() {
			properties = append(properties, service.MapPropertiesToArtifactProperty(prop, artifactID, false))
		}
	}
	if artifact.GetCustomProperties() != nil {
		for _, prop := range *artifact.GetCustomProperties() {
			properties = append(properties, service.MapPropertiesToArtifactProperty(prop, artifactID, true))
		}
	}
	return properties
}

func mapSchemaToAgentTemplateArtifact(artifact schema.Artifact, artProperties []schema.ArtifactProperty) models.AgentTemplateArtifact {
	artifactType := models.AgentTemplateArtifactType
	entity := models.AgentTemplateArtifactImpl{
		ID:     &artifact.ID,
		TypeID: &artifact.TypeID,
		Attributes: &models.AgentTemplateArtifactAttributes{
			Name:                     artifact.Name,
			ArtifactType:             &artifactType,
			ExternalID:               artifact.ExternalID,
			CreateTimeSinceEpoch:     &artifact.CreateTimeSinceEpoch,
			LastUpdateTimeSinceEpoch: &artifact.LastUpdateTimeSinceEpoch,
		},
	}

	var properties []dbmodels.Properties
	var customProperties []dbmodels.Properties
	for _, prop := range artProperties {
		mapped := service.MapArtifactPropertyToProperties(prop)
		if prop.IsCustomProperty {
			customProperties = append(customProperties, mapped)
		} else {
			if prop.Name == "content" && prop.StringValue != nil {
				entity.Attributes.Content = prop.StringValue
			}
			properties = append(properties, mapped)
		}
	}

	entity.Properties = &properties
	entity.CustomProperties = &customProperties
	return &entity
}
