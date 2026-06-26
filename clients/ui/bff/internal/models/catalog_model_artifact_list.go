package models

import (
	"github.com/kubeflow/hub/pkg/openapi"
)

type CatalogModelArtifact struct {
	CreateTimeSinceEpoch     *string                           `json:"createTimeSinceEpoch,omitempty"`
	LastUpdateTimeSinceEpoch *string                           `json:"lastUpdateTimeSinceEpoch,omitempty"`
	Uri                      string                            `json:"uri"`
	CustomProperties         *map[string]openapi.MetadataValue `json:"customProperties,omitempty"`
	ArtifactType             string                            `json:"artifactType"`
}

type CatalogMetricsArtifact struct {
	ArtifactType             string                            `json:"artifactType"`
	MetricsType              *string                           `json:"metricsType"`
	CreateTimeSinceEpoch     *string                           `json:"createTimeSinceEpoch,omitempty"`
	CustomProperties         *map[string]openapi.MetadataValue `json:"customProperties,omitempty"`
	LastUpdateTimeSinceEpoch *string                           `json:"lastUpdateTimeSinceEpoch,omitempty"`
}

type CatalogArtifact struct {
	ID                       *string                           `json:"id,omitempty"`
	Name                     *string                           `json:"name,omitempty"`
	ExternalID               *string                           `json:"externalId,omitempty"`
	ArtifactType             string                            `json:"artifactType"`
	MetricsType              *string                           `json:"metricsType,omitempty"`
	Uri                      *string                           `json:"uri,omitempty"`
	CreateTimeSinceEpoch     *string                           `json:"createTimeSinceEpoch,omitempty"`
	CustomProperties         *map[string]openapi.MetadataValue `json:"customProperties,omitempty"`
	LastUpdateTimeSinceEpoch *string                           `json:"lastUpdateTimeSinceEpoch,omitempty"`
}

type CatalogModelArtifactList struct {
	NextPageToken string            `json:"nextPageToken"`
	PageSize      int32             `json:"pageSize"`
	Size          int32             `json:"size"`
	Items         []CatalogArtifact `json:"items"`
}
