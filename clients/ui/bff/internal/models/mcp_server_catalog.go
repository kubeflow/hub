package models

import "github.com/kubeflow/hub/pkg/openapi"

type McpDeploymentMode string

const (
	McpDeploymentModeLocal  McpDeploymentMode = "local"
	McpDeploymentModeRemote McpDeploymentMode = "remote"
)

type McpTransportType string

const (
	McpTransportTypeStdio McpTransportType = "stdio"
	McpTransportTypeSSE   McpTransportType = "sse"
	McpTransportTypeHTTP  McpTransportType = "http"
)

type McpToolAccessType string

const (
	McpToolAccessTypeReadOnly  McpToolAccessType = "read_only"
	McpToolAccessTypeReadWrite McpToolAccessType = "read_write"
	McpToolAccessTypeExecute   McpToolAccessType = "execute"
)

type McpEndpoints struct {
	HTTP *string `json:"http,omitempty"`
	SSE  *string `json:"sse,omitempty"`
}

type McpArtifact struct {
	URI                      string  `json:"uri"`
	CreateTimeSinceEpoch     *string `json:"createTimeSinceEpoch,omitempty"`
	LastUpdateTimeSinceEpoch *string `json:"lastUpdateTimeSinceEpoch,omitempty"`
}

type McpSecurityIndicator struct {
	VerifiedSource *bool `json:"verifiedSource,omitempty"`
	SecureEndpoint *bool `json:"secureEndpoint,omitempty"`
	SAST           *bool `json:"sast,omitempty"`
	ReadOnlyTools  *bool `json:"readOnlyTools,omitempty"`
}

type McpEnvVarMetadata struct {
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Required     *bool   `json:"required,omitempty"`
	DefaultValue *string `json:"defaultValue,omitempty"`
	Type         *string `json:"type,omitempty"`
	Example      *string `json:"example,omitempty"`
}

type McpResourceTier struct {
	CPU    *string `json:"cpu,omitempty"`
	Memory *string `json:"memory,omitempty"`
}

type McpResourceRecommendation struct {
	Minimal     *McpResourceTier `json:"minimal,omitempty"`
	Recommended *McpResourceTier `json:"recommended,omitempty"`
	High        *McpResourceTier `json:"high,omitempty"`
}

type McpRuntimeMetadataHealthEndpoints struct {
	Liveness  *string `json:"liveness,omitempty"`
	Readiness *string `json:"readiness,omitempty"`
}

type McpRuntimeMetadataCapabilities struct {
	RequiresNetwork    *bool `json:"requiresNetwork,omitempty"`
	RequiresFileSystem *bool `json:"requiresFileSystem,omitempty"`
	RequiresGPU        *bool `json:"requiresGPU,omitempty"`
}

type McpServiceAccountRequirement struct {
	Required      *bool   `json:"required,omitempty"`
	Hint          *string `json:"hint,omitempty"`
	SuggestedName *string `json:"suggestedName,omitempty"`
}

type McpSecretKey struct {
	Key         string  `json:"key"`
	Description string  `json:"description"`
	EnvVarName  *string `json:"envVarName,omitempty"`
	Required    *bool   `json:"required,omitempty"`
}

type McpSecretRequirement struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Keys        []McpSecretKey `json:"keys,omitempty"`
	MountAsFile *bool          `json:"mountAsFile,omitempty"`
	MountPath   *string        `json:"mountPath,omitempty"`
}

type McpConfigMapKey struct {
	Key            string  `json:"key"`
	Description    string  `json:"description"`
	DefaultContent *string `json:"defaultContent,omitempty"`
	EnvVarName     *string `json:"envVarName,omitempty"`
	Required       *bool   `json:"required,omitempty"`
}

type McpConfigMapRequirement struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Keys        []McpConfigMapKey `json:"keys,omitempty"`
	MountAsFile *bool             `json:"mountAsFile,omitempty"`
	MountPath   *string           `json:"mountPath,omitempty"`
}

type McpPrerequisites struct {
	ServiceAccount       *McpServiceAccountRequirement `json:"serviceAccount,omitempty"`
	Secrets              []McpSecretRequirement        `json:"secrets,omitempty"`
	ConfigMaps           []McpConfigMapRequirement     `json:"configMaps,omitempty"`
	EnvironmentVariables []McpEnvVarMetadata           `json:"environmentVariables,omitempty"`
	CustomResources      []string                      `json:"customResources,omitempty"`
}

type McpRuntimeMetadata struct {
	DefaultPort                  *int32                             `json:"defaultPort,omitempty"`
	DefaultArgs                  []string                           `json:"defaultArgs,omitempty"`
	RequiredEnvironmentVariables []McpEnvVarMetadata                `json:"requiredEnvironmentVariables,omitempty"`
	OptionalEnvironmentVariables []McpEnvVarMetadata                `json:"optionalEnvironmentVariables,omitempty"`
	RecommendedResources         *McpResourceRecommendation         `json:"recommendedResources,omitempty"`
	HealthEndpoints              *McpRuntimeMetadataHealthEndpoints `json:"healthEndpoints,omitempty"`
	Capabilities                 *McpRuntimeMetadataCapabilities    `json:"capabilities,omitempty"`
	McpPath                      *string                            `json:"mcpPath,omitempty"`
	Prerequisites                *McpPrerequisites                  `json:"prerequisites,omitempty"`
}

type McpToolParameter struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Description *string `json:"description,omitempty"`
	Required    bool    `json:"required"`
}

type McpTool struct {
	Name             string                            `json:"name"`
	Description      *string                           `json:"description,omitempty"`
	AccessType       McpToolAccessType                 `json:"accessType"`
	Parameters       []McpToolParameter                `json:"parameters,omitempty"`
	Revoked          *bool                             `json:"revoked,omitempty"`
	RevokedReason    *string                           `json:"revokedReason,omitempty"`
	CustomProperties *map[string]openapi.MetadataValue `json:"customProperties,omitempty"`
}

type McpServer struct {
	ID                 string                            `json:"id"`
	Name               string                            `json:"name"`
	DisplayName        *string                           `json:"displayName,omitempty"`
	SourceID           *string                           `json:"source_id,omitempty"`
	Description        *string                           `json:"description,omitempty"`
	Logo               *string                           `json:"logo,omitempty"`
	License            *string                           `json:"license,omitempty"`
	LicenseLink        *string                           `json:"licenseLink,omitempty"`
	Provider           *string                           `json:"provider,omitempty"`
	Version            *string                           `json:"version,omitempty"`
	Tags               []string                          `json:"tags,omitempty"`
	ToolCount          int                               `json:"toolCount"`
	Tools              []McpTool                         `json:"tools,omitempty"`
	SecurityIndicators *McpSecurityIndicator             `json:"securityIndicators,omitempty"`
	DocumentationURL   *string                           `json:"documentationUrl,omitempty"`
	RepositoryURL      *string                           `json:"repositoryUrl,omitempty"`
	SourceCode         *string                           `json:"sourceCode,omitempty"`
	LastUpdated        *string                           `json:"lastUpdated,omitempty"`
	PublishedDate      *string                           `json:"publishedDate,omitempty"`
	Artifacts          []McpArtifact                     `json:"artifacts,omitempty"`
	Transports         []McpTransportType                `json:"transports,omitempty"`
	Readme             *string                           `json:"readme,omitempty"`
	DeploymentMode     *McpDeploymentMode                `json:"deploymentMode,omitempty"`
	Endpoints          *McpEndpoints                     `json:"endpoints,omitempty"`
	RuntimeMetadata    *McpRuntimeMetadata               `json:"runtimeMetadata,omitempty"`
	CustomProperties   *map[string]openapi.MetadataValue `json:"customProperties,omitempty"`
	ServerJson         map[string]interface{}            `json:"serverJson,omitempty"`
}

type McpServerList struct {
	NextPageToken string      `json:"nextPageToken"`
	PageSize      int32       `json:"pageSize"`
	Size          int32       `json:"size"`
	Items         []McpServer `json:"items"`
}

type McpToolWithServer struct {
	ServerID string  `json:"serverId"`
	Tool     McpTool `json:"tool"`
}

type McpToolList struct {
	NextPageToken string              `json:"nextPageToken"`
	PageSize      int32               `json:"pageSize"`
	Size          int32               `json:"size"`
	Items         []McpToolWithServer `json:"items"`
}
