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

var ErrMetricNotFound = errors.New("metric by id not found")

type MetricRepositoryImpl struct {
	*GenericRepository[models.Metric, schema.Artifact, schema.ArtifactProperty, *models.MetricListOptions]
}

func NewMetricRepository(db *gorm.DB, typeID int32) models.MetricRepository {
	config := GenericRepositoryConfig[models.Metric, schema.Artifact, schema.ArtifactProperty, *models.MetricListOptions]{
		DB:                  db,
		TypeID:              typeID,
		EntityToSchema:      mapMetricToArtifact,
		SchemaToEntity:      mapDataLayerToMetric,
		EntityToProperties:  mapMetricToArtifactProperties,
		NotFoundError:       ErrMetricNotFound,
		EntityName:          "metric",
		PropertyFieldName:   "artifact_id",
		ApplyListFilters:    applyMetricListFilters,
		IsNewEntity:         func(entity models.Metric) bool { return entity.GetID() == nil },
		HasCustomProperties: func(entity models.Metric) bool { return entity.GetCustomProperties() != nil },
	}

	return &MetricRepositoryImpl{
		GenericRepository: NewGenericRepository(config),
	}
}

// List adapts the generic repository List method to match the interface contract
func (r *MetricRepositoryImpl) List(listOptions models.MetricListOptions) (*models.ListWrapper[models.Metric], error) {
	return r.GenericRepository.List(&listOptions)
}

func applyMetricListFilters(query *gorm.DB, listOptions *models.MetricListOptions) *gorm.DB {
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

func mapMetricToArtifact(metric models.Metric) schema.Artifact {
	if metric == nil {
		return schema.Artifact{}
	}

	artifact := schema.Artifact{
		ID:     apiutils.ZeroIfNil(metric.GetID()),
		TypeID: apiutils.ZeroIfNil(metric.GetTypeID()),
	}

	if metric.GetAttributes() != nil {
		artifact.Name = metric.GetAttributes().Name
		artifact.URI = metric.GetAttributes().URI
		artifact.ExternalID = metric.GetAttributes().ExternalID
		if metric.GetAttributes().State != nil {
			stateValue := models.Artifact_State_value[*metric.GetAttributes().State]
			artifact.State = &stateValue
		}
		artifact.CreateTimeSinceEpoch = apiutils.ZeroIfNil(metric.GetAttributes().CreateTimeSinceEpoch)
		artifact.LastUpdateTimeSinceEpoch = apiutils.ZeroIfNil(metric.GetAttributes().LastUpdateTimeSinceEpoch)
	}

	return artifact
}

func mapMetricToArtifactProperties(metric models.Metric, artifactID int32) []schema.ArtifactProperty {
	if metric == nil {
		return []schema.ArtifactProperty{}
	}

	properties := []schema.ArtifactProperty{}

	if metric.GetProperties() != nil {
		for _, prop := range *metric.GetProperties() {
			properties = append(properties, MapPropertiesToArtifactProperty(prop, artifactID, false))
		}
	}

	if metric.GetCustomProperties() != nil {
		for _, prop := range *metric.GetCustomProperties() {
			properties = append(properties, MapPropertiesToArtifactProperty(prop, artifactID, true))
		}
	}

	return properties
}

func mapDataLayerToMetric(metric schema.Artifact, artProperties []schema.ArtifactProperty) models.Metric {
	var state *string

	if metric.State != nil {
		metricState := models.Artifact_State_name[*metric.State]
		state = &metricState
	}

	metricArt := models.MetricImpl{
		ID:     &metric.ID,
		TypeID: &metric.TypeID,
		Attributes: &models.MetricAttributes{
			Name:                     metric.Name,
			URI:                      metric.URI,
			State:                    state,
			ArtifactType:             apiutils.StrPtr(models.MetricType),
			ExternalID:               metric.ExternalID,
			CreateTimeSinceEpoch:     &metric.CreateTimeSinceEpoch,
			LastUpdateTimeSinceEpoch: &metric.LastUpdateTimeSinceEpoch,
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

	metricArt.CustomProperties = &customProperties
	metricArt.Properties = &properties

	return &metricArt
}
