package models

import (
	catalogmodels "github.com/kubeflow/hub/catalog/internal/db/models"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/kubeflow/hub/internal/platform/db/filter"
)

type MetricsType string

const (
	MetricsTypePerformance     MetricsType = "performance-metrics"
	MetricsTypeAccuracy        MetricsType = "accuracy-metrics"
	MetricsTypeSecurityMetrics MetricsType = "security-metrics"
	CatalogMetricsArtifactType             = "metrics-artifact"
)

type CatalogMetricsArtifactListOptions struct {
	dbmodels.Pagination
	Name             *string
	ExternalID       *string
	ParentResourceID *int32
}

// GetRestEntityType implements the FilterApplier interface
func (c *CatalogMetricsArtifactListOptions) GetRestEntityType() filter.RestEntityType {
	return filter.RestEntityType(catalogmodels.RestEntityCatalogArtifact)
}

type CatalogMetricsArtifactAttributes struct {
	Name                     *string
	ArtifactType             *string
	MetricsType              MetricsType
	ExternalID               *string
	CreateTimeSinceEpoch     *int64
	LastUpdateTimeSinceEpoch *int64
}

type CatalogMetricsArtifact interface {
	dbmodels.Entity[CatalogMetricsArtifactAttributes]
}

type CatalogMetricsArtifactImpl = dbmodels.BaseEntity[CatalogMetricsArtifactAttributes]

type CatalogMetricsArtifactRepository interface {
	GetByID(id int32) (CatalogMetricsArtifact, error)
	List(listOptions CatalogMetricsArtifactListOptions) (*dbmodels.ListWrapper[CatalogMetricsArtifact], error)
	Save(metricsArtifact CatalogMetricsArtifact, parentResourceID *int32) (CatalogMetricsArtifact, error)
	// BatchSave inserts multiple metrics artifacts in a single batch operation
	BatchSave(metricsArtifacts []CatalogMetricsArtifact, parentResourceID *int32) ([]CatalogMetricsArtifact, error)
}
