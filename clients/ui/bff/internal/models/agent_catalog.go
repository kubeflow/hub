package models

import "github.com/kubeflow/hub/pkg/openapi"

type AgentImageArtifact struct {
	// Artifact kind identifying an agent deployment image.
	ArtifactType string `json:"artifactType" example:"image-artifact"`
	// URI of the OCI image used to deploy the agent.
	URI string `json:"uri" example:"oci://quay.io/example/research-agent:latest"`
	// Unique artifact identifier.
	ID *string `json:"id,omitempty" example:"1"`
	// Human-readable artifact name.
	Name *string `json:"name,omitempty" example:"Research agent image"`
	// Creation time in milliseconds since the Unix epoch, encoded as a string.
	CreateTimeSinceEpoch *string `json:"createTimeSinceEpoch,omitempty" format:"int64" example:"1700000000000"`
	// Last update time in milliseconds since the Unix epoch, encoded as a string.
	LastUpdateTimeSinceEpoch *string `json:"lastUpdateTimeSinceEpoch,omitempty" format:"int64" example:"1700000000000"`
}

type AgentTemplateArtifact struct {
	// Artifact kind identifying an agent deployment template.
	ArtifactType string `json:"artifactType" example:"template-artifact"`
	// Full agent.yaml specification serialized as a JSON string.
	Content string `json:"content" example:"{\"name\":\"research-agent\"}"`
	// Unique artifact identifier.
	ID *string `json:"id,omitempty" example:"1"`
	// Human-readable artifact name.
	Name *string `json:"name,omitempty" example:"Research agent template"`
	// Creation time in milliseconds since the Unix epoch, encoded as a string.
	CreateTimeSinceEpoch *string `json:"createTimeSinceEpoch,omitempty" format:"int64" example:"1700000000000"`
	// Last update time in milliseconds since the Unix epoch, encoded as a string.
	LastUpdateTimeSinceEpoch *string `json:"lastUpdateTimeSinceEpoch,omitempty" format:"int64" example:"1700000000000"`
}

type AgentArtifact struct {
	// Artifact kind selecting the deployment content or image URI.
	ArtifactType string `json:"artifactType" example:"template-artifact"`
	// OCI image URI, when this is an image artifact.
	URI *string `json:"uri,omitempty" example:"oci://quay.io/example/research-agent:latest"`
	// Full agent.yaml specification serialized as a JSON string, for template artifacts.
	Content *string `json:"content,omitempty" example:"{\"name\":\"research-agent\"}"`
	// Unique artifact identifier.
	ID *string `json:"id,omitempty" example:"1"`
	// Human-readable artifact name.
	Name *string `json:"name,omitempty" example:"Research agent template"`
	// Creation time in milliseconds since the Unix epoch, encoded as a string.
	CreateTimeSinceEpoch *string `json:"createTimeSinceEpoch,omitempty" format:"int64" example:"1700000000000"`
	// Last update time in milliseconds since the Unix epoch, encoded as a string.
	LastUpdateTimeSinceEpoch *string `json:"lastUpdateTimeSinceEpoch,omitempty" format:"int64" example:"1700000000000"`
}

type AgentArtifactList struct {
	// Opaque token for retrieving the next page; empty on the final page.
	NextPageToken string `json:"nextPageToken"`
	// Maximum number of artifacts requested per page.
	PageSize int32 `json:"pageSize" example:"10"`
	// Number of artifacts returned in this page.
	Size int32 `json:"size" example:"1"`
	// Deployment artifacts associated with the agent.
	Items []AgentArtifact `json:"items"`
}

type AgentEnvVar struct {
	// Environment variable name.
	Name string `json:"name" example:"MODEL_ENDPOINT"`
	// Whether this variable must be supplied for deployment.
	Required bool `json:"required" example:"true"`
	// Purpose of the environment variable and guidance for setting it.
	Description *string `json:"description,omitempty" example:"URL of the model inference endpoint."`
}

// Agent describes a deployable agent discovered from a catalog source.
// @Description An agent with framework, Markdown documentation, environment variables, and OCI images.
type Agent struct {
	// Unique agent identifier.
	ID string `json:"id" example:"1"`
	// Agent name, unique within its catalog source.
	Name string `json:"name" example:"research-assistant"`
	// Catalog source providing this agent.
	SourceID *string `json:"source_id,omitempty" example:"community"`
	// Human-readable display name.
	DisplayName *string `json:"displayName,omitempty" example:"Research Assistant"`
	// Short description of the agent's capabilities.
	Description *string `json:"description,omitempty" example:"Researches topics and summarizes sources."`
	// Full agent documentation in Markdown.
	Readme *string `json:"readme,omitempty" example:"# Research assistant"`
	// Agent framework, such as langgraph, crewai, or autogen.
	Framework *string `json:"framework,omitempty" example:"langgraph"`
	// Labels used for categorization and filtering.
	Labels []string `json:"labels,omitempty" example:"research,community"`
	// URL or path of the agent's logo image.
	Logo *string `json:"logo,omitempty" example:"https://example.com/agent-logo.png"`
	// URL of the agent's source code repository.
	RepositoryURL *string `json:"repositoryUrl,omitempty" format:"uri" example:"https://github.com/example/research-agent"`
	// Environment variables required or supported by the deployment.
	Env []AgentEnvVar `json:"env,omitempty"`
	// OCI image artifacts used to deploy the agent.
	Artifacts []AgentImageArtifact `json:"artifacts,omitempty"`
	// Custom properties keyed by name; each value is selected by metadataType.
	CustomProperties *map[string]openapi.MetadataValue `json:"customProperties,omitempty"`
	// Creation time in milliseconds since the Unix epoch, encoded as a string.
	CreateTimeSinceEpoch *string `json:"createTimeSinceEpoch,omitempty" format:"int64" example:"1700000000000"`
	// Last update time in milliseconds since the Unix epoch, encoded as a string.
	LastUpdateTimeSinceEpoch *string `json:"lastUpdateTimeSinceEpoch,omitempty" format:"int64" example:"1700000000000"`
}

type AgentList struct {
	// Opaque token for retrieving the next page; empty on the final page.
	NextPageToken string `json:"nextPageToken"`
	// Maximum number of agents requested per page.
	PageSize int32 `json:"pageSize" example:"10"`
	// Number of agents returned in this page.
	Size int32 `json:"size" example:"1"`
	// Agents returned in this page.
	Items []Agent `json:"items"`
}
