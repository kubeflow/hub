package models

import (
	models "github.com/kubeflow/hub/internal/platform/db/entity"
)

// CatalogSourceAttributes holds the attributes for a catalog source record.
type CatalogSourceAttributes struct {
	// Name is the source ID (used as the context name)
	Name *string
	// CreateTimeSinceEpoch is when the record was created
	CreateTimeSinceEpoch *int64
	// LastUpdateTimeSinceEpoch is when the record was last updated
	LastUpdateTimeSinceEpoch *int64
}

// CatalogSource represents a catalog source stored in the database.
type CatalogSource interface {
	models.Entity[CatalogSourceAttributes]
}

// CatalogSourceImpl is the concrete implementation of CatalogSource.
type CatalogSourceImpl = models.BaseEntity[CatalogSourceAttributes]

// SourceStatus holds the operational status and error for a source.
type SourceStatus struct {
	Status string
	Error  string
}

// CatalogSourceRepository defines the interface for catalog source persistence.
type CatalogSourceRepository interface {
	// GetBySourceID retrieves a catalog source by its source ID.
	GetBySourceID(sourceID string) (CatalogSource, error)

	// Save creates or updates a catalog source.
	// The source ID is used as the unique identifier (context name).
	Save(source CatalogSource) (CatalogSource, error)

	// Delete removes a catalog source by its source ID.
	Delete(sourceID string) error

	// GetAll retrieves all catalog sources.
	GetAll() ([]CatalogSource, error)

	// GetAllStatuses returns a map of source ID to status/error for all sources.
	GetAllStatuses() (map[string]SourceStatus, error)

	// GetStatus returns the status/error persisted for a single source. It returns an empty
	// SourceStatus (not an error) if the source does not exist.
	GetStatus(sourceID string) (SourceStatus, error)
}
