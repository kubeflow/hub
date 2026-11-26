package models

import (
	"github.com/kubeflow/hub/pkg/openapi"
)

type CatalogModelArtifact struct {
	// Creation time in milliseconds since the Unix epoch, encoded as a string.
	CreateTimeSinceEpoch *string `json:"createTimeSinceEpoch,omitempty" format:"int64" example:"1700000000000"`
	// Last update time in milliseconds since the Unix epoch, encoded as a string.
	LastUpdateTimeSinceEpoch *string `json:"lastUpdateTimeSinceEpoch,omitempty" format:"int64" example:"1700000000000"`
	// Location of the model's downloadable artifact.
	Uri string `json:"uri" example:"s3://example-models/model/"`
	// Custom properties keyed by name; each value is selected by metadataType.
	CustomProperties *map[string]openapi.MetadataValue `json:"customProperties,omitempty"`
	// Artifact kind identifying model storage metadata.
	ArtifactType string `json:"artifactType" example:"model-artifact"`
}

type CatalogMetricsArtifact struct {
	// Artifact kind identifying catalog evaluation metrics.
	ArtifactType string `json:"artifactType" example:"metrics-artifact"`
	// Category of evaluation metrics, such as performance or security metrics.
	MetricsType *string `json:"metricsType" example:"security-metrics"`
	// Creation time in milliseconds since the Unix epoch, encoded as a string.
	CreateTimeSinceEpoch *string `json:"createTimeSinceEpoch,omitempty" format:"int64" example:"1700000000000"`
	// Evaluation values and configuration keyed by custom-property name.
	CustomProperties *map[string]openapi.MetadataValue `json:"customProperties,omitempty"`
	// Last update time in milliseconds since the Unix epoch, encoded as a string.
	LastUpdateTimeSinceEpoch *string `json:"lastUpdateTimeSinceEpoch,omitempty" format:"int64" example:"1700000000000"`
}

type CatalogArtifact struct {
	// Unique catalog artifact identifier.
	ID *string `json:"id,omitempty" example:"artifact-1"`
	// Human-readable artifact name.
	Name *string `json:"name,omitempty" example:"Model weights"`
	// Identifier assigned by an external system, when supplied.
	ExternalID *string `json:"externalId,omitempty" example:"upstream-artifact-1"`
	// Artifact category selecting model storage or evaluation metadata.
	ArtifactType string `json:"artifactType" example:"model-artifact"`
	// Evaluation category, when this artifact contains metrics.
	MetricsType *string `json:"metricsType,omitempty" example:"security-metrics"`
	// Physical location of the model artifact, when present.
	Uri *string `json:"uri,omitempty" example:"s3://example-models/model/"`
	// Creation time in milliseconds since the Unix epoch, encoded as a string.
	CreateTimeSinceEpoch *string `json:"createTimeSinceEpoch,omitempty" format:"int64" example:"1700000000000"`
	// Artifact metadata and evaluation values keyed by custom-property name.
	CustomProperties *map[string]openapi.MetadataValue `json:"customProperties,omitempty"`
	// Last update time in milliseconds since the Unix epoch, encoded as a string.
	LastUpdateTimeSinceEpoch *string `json:"lastUpdateTimeSinceEpoch,omitempty" format:"int64" example:"1700000000000"`
}

type CatalogModelArtifactList struct {
	// Opaque token for retrieving the next page; empty on the final page.
	NextPageToken string `json:"nextPageToken"`
	// Maximum number of artifacts requested per page.
	PageSize int32 `json:"pageSize" example:"10"`
	// Number of artifacts returned in this page.
	Size int32 `json:"size" example:"1"`
	// Model and evaluation artifacts returned in this page.
	Items []CatalogArtifact `json:"items"`
}
