package models

import (
	"sync"

	models "github.com/kubeflow/hub/internal/platform/db/entity"
	dbfilter "github.com/kubeflow/hub/internal/platform/db/filter"
	"github.com/kubeflow/hub/internal/platform/db/schema"
)

type CatalogArtifactListOptions struct {
	models.Pagination
	Name                *string
	ExternalID          *string
	ParentResourceID    *int32
	ArtifactType        *string
	ArtifactTypesFilter []string
}

// GetRestEntityType implements the FilterApplier interface
// This enables advanced filtering support for catalog artifacts
func (c *CatalogArtifactListOptions) GetRestEntityType() dbfilter.RestEntityType {
	return dbfilter.RestEntityType(RestEntityCatalogArtifact)
}

// ArtifactMapperFunc defines the signature for artifact mapping functions
type ArtifactMapperFunc func(artifact schema.Artifact, properties []schema.ArtifactProperty) any

// Global registry for artifact mappers
var (
	artifactMappersMu sync.RWMutex
	artifactMappers   = make(map[string]ArtifactMapperFunc)
)

// RegisterArtifactMapper registers a mapping function for a specific artifact type
func RegisterArtifactMapper(typeName string, mapper ArtifactMapperFunc) {
	artifactMappersMu.Lock()
	defer artifactMappersMu.Unlock()
	artifactMappers[typeName] = mapper
}

// GetArtifactMapper retrieves a mapping function for a specific artifact type
func GetArtifactMapper(typeName string) (ArtifactMapperFunc, bool) {
	artifactMappersMu.RLock()
	defer artifactMappersMu.RUnlock()
	mapper, exists := artifactMappers[typeName]
	return mapper, exists
}

// CatalogArtifactEntity defines the common interface that all catalog artifacts must implement.
// This allows the shared infrastructure to work with catalog-specific types without import cycles.
// Note: GetAttributes() is intentionally excluded because concrete types return different
// typed pointers, which Go's interface matching does not allow.
type CatalogArtifactEntity interface {
	GetID() *int32
	SetID(int32)
	GetProperties() *[]models.Properties
	GetCustomProperties() *[]models.Properties
}

// CatalogModelArtifact represents the interface for model artifacts
type CatalogModelArtifact CatalogArtifactEntity

// CatalogMetricsArtifact represents the interface for metrics artifacts
type CatalogMetricsArtifact CatalogArtifactEntity

// CatalogArtifact is a discriminated union that can hold different catalog artifact types
type CatalogArtifact struct {
	CatalogModelArtifact   CatalogModelArtifact
	CatalogMetricsArtifact CatalogMetricsArtifact
}

type CatalogArtifactRepository interface {
	GetByID(id int32) (CatalogArtifact, error)
	List(listOptions CatalogArtifactListOptions) (*models.ListWrapper[CatalogArtifact], error)
	DeleteByParentID(artifactType string, parentResourceID int32) error
	// CountByParentIDs returns artifact counts grouped by category for each parent model ID.
	// The outer map key is the parent model ID; the inner map key is the artifact category
	// ("model-artifact", "performance-metrics", "accuracy-metrics", "security-metrics").
	// Categories with zero count are omitted. Parents with no artifacts have no entry.
	CountByParentIDs(parentIDs []int32) (map[int32]map[string]int32, error)
}
