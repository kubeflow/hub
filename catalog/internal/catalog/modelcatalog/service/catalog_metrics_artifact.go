package service

import (
	"errors"
	"fmt"
	"math"

	"github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	sharedmodels "github.com/kubeflow/hub/catalog/internal/db/models"
	"github.com/kubeflow/hub/internal/platform/apiutils"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	service "github.com/kubeflow/hub/internal/platform/db/repository"
	"github.com/kubeflow/hub/internal/platform/db/utils"
	"gorm.io/gorm"
)

// Register the mapping function for CatalogMetricsArtifact
func init() {
	sharedmodels.RegisterArtifactMapper("kf.CatalogMetricsArtifact", func(artifact schema.Artifact, properties []schema.ArtifactProperty) interface{} {
		return mapDataLayerToCatalogMetricsArtifact(artifact, properties)
	})
}

var ErrCatalogMetricsArtifactNotFound = errors.New("catalog metrics artifact by id not found")

type CatalogMetricsArtifactRepositoryImpl struct {
	*service.GenericRepository[models.CatalogMetricsArtifact, schema.Artifact, schema.ArtifactProperty, *models.CatalogMetricsArtifactListOptions]
}

func NewCatalogMetricsArtifactRepository(db *gorm.DB, typeID int32) models.CatalogMetricsArtifactRepository {
	config := service.GenericRepositoryConfig[models.CatalogMetricsArtifact, schema.Artifact, schema.ArtifactProperty, *models.CatalogMetricsArtifactListOptions]{
		DB:                      db,
		TypeID:                  typeID,
		EntityToSchema:          mapCatalogMetricsArtifactToArtifact,
		SchemaToEntity:          mapDataLayerToCatalogMetricsArtifact,
		EntityToProperties:      mapCatalogMetricsArtifactToArtifactProperties,
		NotFoundError:           ErrCatalogMetricsArtifactNotFound,
		EntityName:              "catalog metrics artifact",
		PropertyFieldName:       "artifact_id",
		ApplyListFilters:        applyCatalogMetricsArtifactListFilters,
		IsNewEntity:             func(entity models.CatalogMetricsArtifact) bool { return entity.GetID() == nil },
		HasCustomProperties:     func(entity models.CatalogMetricsArtifact) bool { return entity.GetCustomProperties() != nil },
		PreserveHistoricalTimes: true, // Catalog preserves timestamps from YAML source data
	}

	return &CatalogMetricsArtifactRepositoryImpl{
		GenericRepository: service.NewGenericRepository(config),
	}
}

func (r *CatalogMetricsArtifactRepositoryImpl) List(listOptions models.CatalogMetricsArtifactListOptions) (*dbmodels.ListWrapper[models.CatalogMetricsArtifact], error) {
	return r.GenericRepository.List(&listOptions)
}

func (r *CatalogMetricsArtifactRepositoryImpl) Save(ma models.CatalogMetricsArtifact, parentResourceID *int32) (models.CatalogMetricsArtifact, error) {
	config := r.GetConfig()
	if ma.GetTypeID() == nil {
		if config.TypeID > 0 {
			ma.SetTypeID(config.TypeID)
		}
	}

	attr := ma.GetAttributes()
	if attr == nil {
		return ma, fmt.Errorf("invalid artifact: nil attributes")
	}

	if ma.GetID() == nil && attr.Name != nil {
		existing, err := r.lookupMetricsArtifactByName(*attr.Name)
		if err != nil {
			if !errors.Is(err, ErrCatalogMetricsArtifactNotFound) {
				return ma, fmt.Errorf("error finding existing metrics artifact named %s: %w", *attr.Name, err)
			}
		} else {
			ma.SetID(existing.ID)
		}
	}

	switch attr.MetricsType {
	case models.MetricsTypeAccuracy, models.MetricsTypePerformance, models.MetricsTypeSecurityMetrics:
		// OK
	default:
		return ma, fmt.Errorf("invalid artifact: unknown metrics type: %s", attr.MetricsType)
	}

	return r.GenericRepository.Save(ma, parentResourceID)
}

func (r *CatalogMetricsArtifactRepositoryImpl) BatchSave(artifacts []models.CatalogMetricsArtifact, parentResourceID *int32) ([]models.CatalogMetricsArtifact, error) {
	numArtifacts := len(artifacts)
	if numArtifacts == 0 {
		return artifacts, nil
	}

	config := r.GetConfig()

	// Pre-allocate schema artifacts slice
	schemaArtifacts := make([]schema.Artifact, numArtifacts)

	// Validate, prepare, and convert all artifacts in one pass
	for i, ma := range artifacts {
		if ma.GetTypeID() == nil {
			if config.TypeID > 0 && config.TypeID < math.MaxInt32 {
				ma.SetTypeID(int32(config.TypeID))
			}
		}

		attr := ma.GetAttributes()
		if attr == nil {
			return nil, fmt.Errorf("invalid artifact at index %d: nil attributes", i)
		}

		switch attr.MetricsType {
		case models.MetricsTypeAccuracy, models.MetricsTypePerformance, models.MetricsTypeSecurityMetrics:
			// OK
		default:
			return nil, fmt.Errorf("invalid artifact at index %d: unknown metrics type: %s", i, attr.MetricsType)
		}

		schemaArtifacts[i] = mapCatalogMetricsArtifactToArtifact(ma)
		artifacts[i] = ma
	}

	// Execute all batch operations in a single transaction
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		// Batch insert artifacts (batch size of 100)
		if err := tx.CreateInBatches(&schemaArtifacts, 100).Error; err != nil {
			return fmt.Errorf("failed to batch insert artifacts: %w", err)
		}

		// Pre-allocate slices for properties and attributions
		// Estimate ~10 properties per artifact on average
		allProperties := []schema.ArtifactProperty{}
		var allAttributions []schema.Attribution
		if parentResourceID != nil {
			allAttributions = make([]schema.Attribution, 0, numArtifacts)
		}

		// Collect all properties and attributions
		for i, schemaArtifact := range schemaArtifacts {
			artifactID := schemaArtifact.ID
			artifacts[i].SetID(artifactID)

			// Collect properties
			properties := mapCatalogMetricsArtifactToArtifactProperties(artifacts[i], artifactID)
			allProperties = append(allProperties, properties...)

			// Collect attribution if parentResourceID is provided
			if parentResourceID != nil {
				allAttributions = append(allAttributions, schema.Attribution{
					ContextID:  *parentResourceID,
					ArtifactID: artifactID,
				})
			}
		}

		// Batch insert all properties
		if len(allProperties) > 0 {
			if err := tx.CreateInBatches(&allProperties, 100).Error; err != nil {
				return fmt.Errorf("failed to batch insert properties: %w", err)
			}
		}

		// Batch insert all attributions
		if len(allAttributions) > 0 {
			if err := tx.CreateInBatches(&allAttributions, 100).Error; err != nil {
				return fmt.Errorf("failed to batch insert attributions: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return artifacts, nil
}

func (r *CatalogMetricsArtifactRepositoryImpl) lookupMetricsArtifactByName(name string) (*schema.Artifact, error) {
	var entity schema.Artifact

	config := r.GetConfig()

	if err := config.DB.Where("name = ? AND type_id = ?", name, config.TypeID).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %v", config.NotFoundError, err)
		}
		return nil, fmt.Errorf("error getting %s by name: %w", config.EntityName, err)
	}

	return &entity, nil
}

func applyCatalogMetricsArtifactListFilters(query *gorm.DB, listOptions *models.CatalogMetricsArtifactListOptions) *gorm.DB {
	if listOptions.Name != nil {
		query = query.Where("name LIKE ?", fmt.Sprintf("%%:%s", *listOptions.Name))
	} else if listOptions.ExternalID != nil {
		query = query.Where("external_id = ?", listOptions.ExternalID)
	}

	if listOptions.ParentResourceID != nil {
		query = query.Joins(utils.BuildAttributionJoin(query)).
			Where(utils.GetColumnRef(query, &schema.Attribution{}, "context_id")+" = ?", listOptions.ParentResourceID)
	}

	return query
}

func mapCatalogMetricsArtifactToArtifact(catalogMetricsArtifact models.CatalogMetricsArtifact) schema.Artifact {
	if catalogMetricsArtifact == nil {
		return schema.Artifact{}
	}

	artifact := schema.Artifact{
		ID:     apiutils.ZeroIfNil(catalogMetricsArtifact.GetID()),
		TypeID: apiutils.ZeroIfNil(catalogMetricsArtifact.GetTypeID()),
	}

	if catalogMetricsArtifact.GetAttributes() != nil {
		artifact.Name = catalogMetricsArtifact.GetAttributes().Name
		artifact.ExternalID = catalogMetricsArtifact.GetAttributes().ExternalID
		artifact.CreateTimeSinceEpoch = apiutils.ZeroIfNil(catalogMetricsArtifact.GetAttributes().CreateTimeSinceEpoch)
		artifact.LastUpdateTimeSinceEpoch = apiutils.ZeroIfNil(catalogMetricsArtifact.GetAttributes().LastUpdateTimeSinceEpoch)
	}

	return artifact
}

func mapCatalogMetricsArtifactToArtifactProperties(catalogMetricsArtifact models.CatalogMetricsArtifact, artifactID int32) []schema.ArtifactProperty {
	if catalogMetricsArtifact == nil {
		return []schema.ArtifactProperty{}
	}

	properties := []schema.ArtifactProperty{}

	// Add the metricsType as a property
	if catalogMetricsArtifact.GetAttributes() != nil {
		metricsTypeProp := dbmodels.Properties{
			Name:        "metricsType",
			StringValue: apiutils.Of(string(catalogMetricsArtifact.GetAttributes().MetricsType)),
		}
		properties = append(properties, service.MapPropertiesToArtifactProperty(metricsTypeProp, artifactID, false))
	}

	if catalogMetricsArtifact.GetProperties() != nil {
		for _, prop := range *catalogMetricsArtifact.GetProperties() {
			properties = append(properties, service.MapPropertiesToArtifactProperty(prop, artifactID, false))
		}
	}

	if catalogMetricsArtifact.GetCustomProperties() != nil {
		for _, prop := range *catalogMetricsArtifact.GetCustomProperties() {
			properties = append(properties, service.MapPropertiesToArtifactProperty(prop, artifactID, true))
		}
	}

	return properties
}

func mapDataLayerToCatalogMetricsArtifact(artifact schema.Artifact, artProperties []schema.ArtifactProperty) models.CatalogMetricsArtifact {
	catalogMetricsArtifact := models.CatalogMetricsArtifactImpl{
		ID:     &artifact.ID,
		TypeID: &artifact.TypeID,
		Attributes: &models.CatalogMetricsArtifactAttributes{
			Name:                     artifact.Name,
			ArtifactType:             apiutils.Of(models.CatalogMetricsArtifactType),
			ExternalID:               artifact.ExternalID,
			CreateTimeSinceEpoch:     &artifact.CreateTimeSinceEpoch,
			LastUpdateTimeSinceEpoch: &artifact.LastUpdateTimeSinceEpoch,
		},
	}

	customProperties := []dbmodels.Properties{}
	properties := []dbmodels.Properties{}

	for _, prop := range artProperties {
		mappedProperty := service.MapArtifactPropertyToProperties(prop)

		// Extract metricsType from properties and set it as an attribute
		if mappedProperty.Name == "metricsType" && !prop.IsCustomProperty {
			if mappedProperty.StringValue != nil {
				catalogMetricsArtifact.Attributes.MetricsType = models.MetricsType(*mappedProperty.StringValue)
			}
		} else if prop.IsCustomProperty {
			customProperties = append(customProperties, mappedProperty)
		} else {
			properties = append(properties, mappedProperty)
		}
	}

	catalogMetricsArtifact.CustomProperties = &customProperties
	catalogMetricsArtifact.Properties = &properties

	return &catalogMetricsArtifact
}
