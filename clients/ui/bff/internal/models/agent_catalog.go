package models

import "github.com/kubeflow/hub/pkg/openapi"

type AgentArtifact struct {
	URI                      string  `json:"uri"`
	CreateTimeSinceEpoch     *string `json:"createTimeSinceEpoch,omitempty"`
	LastUpdateTimeSinceEpoch *string `json:"lastUpdateTimeSinceEpoch,omitempty"`
}

type AgentEnvVar struct {
	Name        string  `json:"name"`
	Required    bool    `json:"required"`
	Description *string `json:"description,omitempty"`
}

type Agent struct {
	ID                       string                            `json:"id"`
	Name                     string                            `json:"name"`
	SourceID                 *string                           `json:"source_id,omitempty"`
	DisplayName              *string                           `json:"displayName,omitempty"`
	Description              *string                           `json:"description,omitempty"`
	Readme                   *string                           `json:"readme,omitempty"`
	Framework                *string                           `json:"framework,omitempty"`
	Labels                   []string                          `json:"labels,omitempty"`
	Logo                     *string                           `json:"logo,omitempty"`
	RepositoryURL            *string                           `json:"repositoryUrl,omitempty"`
	Env                      []AgentEnvVar                     `json:"env,omitempty"`
	Artifacts                []AgentArtifact                   `json:"artifacts,omitempty"`
	CustomProperties         *map[string]openapi.MetadataValue `json:"customProperties,omitempty"`
	CreateTimeSinceEpoch     *string                           `json:"createTimeSinceEpoch,omitempty"`
	LastUpdateTimeSinceEpoch *string                           `json:"lastUpdateTimeSinceEpoch,omitempty"`
}

type AgentList struct {
	NextPageToken string  `json:"nextPageToken"`
	PageSize      int32   `json:"pageSize"`
	Size          int32   `json:"size"`
	Items         []Agent `json:"items"`
}
