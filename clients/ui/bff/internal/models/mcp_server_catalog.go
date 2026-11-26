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
	// Streamable HTTP endpoint of the remote MCP server.
	HTTP *string `json:"http,omitempty" format:"uri" example:"https://mcp.example.com/mcp"`
	// Server-sent events endpoint of the remote MCP server.
	SSE *string `json:"sse,omitempty" format:"uri" example:"https://mcp.example.com/sse"`
}

type McpArtifact struct {
	// URI of the server deployment artifact.
	URI string `json:"uri" example:"oci://quay.io/example/mcp-server:latest"`
	// Creation time in milliseconds since the Unix epoch, encoded as a string.
	CreateTimeSinceEpoch *string `json:"createTimeSinceEpoch,omitempty" format:"int64" example:"1700000000000"`
	// Last update time in milliseconds since the Unix epoch, encoded as a string.
	LastUpdateTimeSinceEpoch *string `json:"lastUpdateTimeSinceEpoch,omitempty" format:"int64" example:"1700000000000"`
}

type McpSecurityIndicator struct {
	// Whether the catalog identifies the server's source as verified.
	VerifiedSource *bool `json:"verifiedSource,omitempty" example:"true"`
	// Whether the catalog identifies the server endpoint as secure.
	SecureEndpoint *bool `json:"secureEndpoint,omitempty" example:"true"`
	// Whether the server has a positive static application security testing indicator.
	SAST *bool `json:"sast,omitempty" example:"true"`
	// Whether the server exposes only read-only tools.
	ReadOnlyTools *bool `json:"readOnlyTools,omitempty" example:"false"`
}

type McpEnvVarMetadata struct {
	// Environment variable name.
	Name string `json:"name" example:"LOG_LEVEL"`
	// Purpose and expected contents of the variable.
	Description string `json:"description" example:"Logging verbosity for the server."`
	// Whether a deployment must provide this variable.
	Required *bool `json:"required,omitempty" example:"false"`
	// Value used when the variable is not provided.
	DefaultValue *string `json:"defaultValue,omitempty" example:"info"`
	// Expected value type.
	Type *string `json:"type,omitempty" example:"string"`
	// Illustrative, non-secret value for the variable.
	Example *string `json:"example,omitempty" example:"debug"`
}

type McpResourceTier struct {
	// CPU quantity expressed in Kubernetes resource units.
	CPU *string `json:"cpu,omitempty" example:"500m"`
	// Memory quantity expressed in Kubernetes resource units.
	Memory *string `json:"memory,omitempty" example:"512Mi"`
}

type McpResourceRecommendation struct {
	// Minimum suggested resources for running the server.
	Minimal *McpResourceTier `json:"minimal,omitempty"`
	// Resources recommended for typical workloads.
	Recommended *McpResourceTier `json:"recommended,omitempty"`
	// Resources suggested for higher-demand workloads.
	High *McpResourceTier `json:"high,omitempty"`
}

type McpRuntimeMetadataHealthEndpoints struct {
	// HTTP path used to check whether the server is alive.
	Liveness *string `json:"liveness,omitempty" example:"/healthz"`
	// HTTP path used to check whether the server can accept requests.
	Readiness *string `json:"readiness,omitempty" example:"/readyz"`
}

type McpRuntimeMetadataCapabilities struct {
	// Whether the server needs network access.
	RequiresNetwork *bool `json:"requiresNetwork,omitempty" example:"true"`
	// Whether the server needs filesystem access.
	RequiresFileSystem *bool `json:"requiresFileSystem,omitempty" example:"false"`
	// Whether the server needs a GPU.
	RequiresGPU *bool `json:"requiresGPU,omitempty" example:"false"`
}

type McpServiceAccountRequirement struct {
	// Whether a Kubernetes service account is required.
	Required *bool `json:"required,omitempty" example:"true"`
	// Guidance for configuring the service account and its permissions.
	Hint *string `json:"hint,omitempty" example:"Grant read access to the required namespace."`
	// Suggested name of the service account.
	SuggestedName *string `json:"suggestedName,omitempty" example:"mcp-server"`
}

type McpSecretKey struct {
	// Key within the Kubernetes Secret.
	Key string `json:"key" example:"api-token"`
	// Purpose of the secret value; no credentials are included here.
	Description string `json:"description" example:"Token for the upstream service."`
	// Environment variable receiving this secret value.
	EnvVarName *string `json:"envVarName,omitempty" example:"UPSTREAM_TOKEN"`
	// Whether the secret must contain this key.
	Required *bool `json:"required,omitempty" example:"true"`
}

type McpSecretRequirement struct {
	// Name of the Kubernetes Secret needed by the server.
	Name string `json:"name" example:"mcp-credentials"`
	// Purpose of the Secret.
	Description string `json:"description" example:"Credentials for upstream services."`
	// Secret keys consumed by the server.
	Keys []McpSecretKey `json:"keys,omitempty"`
	// Whether the Secret should be mounted as files.
	MountAsFile *bool `json:"mountAsFile,omitempty" example:"false"`
	// Container path at which to mount Secret files.
	MountPath *string `json:"mountPath,omitempty" example:"/var/run/secrets/mcp"`
}

type McpConfigMapKey struct {
	// Key within the Kubernetes ConfigMap.
	Key string `json:"key" example:"config.yaml"`
	// Purpose of the configuration entry.
	Description string `json:"description" example:"Server configuration file."`
	// Suggested default content of the entry.
	DefaultContent *string `json:"defaultContent,omitempty" example:"logLevel: info"`
	// Environment variable receiving this configuration value.
	EnvVarName *string `json:"envVarName,omitempty" example:"SERVER_CONFIG"`
	// Whether the ConfigMap must contain this key.
	Required *bool `json:"required,omitempty" example:"true"`
}

type McpConfigMapRequirement struct {
	// Name of the Kubernetes ConfigMap needed by the server.
	Name string `json:"name" example:"mcp-config"`
	// Purpose of the ConfigMap.
	Description string `json:"description" example:"Runtime settings for the MCP server."`
	// Configuration entries consumed by the server.
	Keys []McpConfigMapKey `json:"keys,omitempty"`
	// Whether the ConfigMap should be mounted as files.
	MountAsFile *bool `json:"mountAsFile,omitempty" example:"true"`
	// Container path at which to mount configuration files.
	MountPath *string `json:"mountPath,omitempty" example:"/etc/mcp"`
}

type McpPrerequisites struct {
	// Kubernetes service account requirements.
	ServiceAccount *McpServiceAccountRequirement `json:"serviceAccount,omitempty"`
	// Kubernetes Secrets needed before deployment.
	Secrets []McpSecretRequirement `json:"secrets,omitempty"`
	// Kubernetes ConfigMaps needed before deployment.
	ConfigMaps []McpConfigMapRequirement `json:"configMaps,omitempty"`
	// Environment variables that must be configured for deployment.
	EnvironmentVariables []McpEnvVarMetadata `json:"environmentVariables,omitempty"`
	// Custom resource definitions or resources needed by the server.
	CustomResources []string `json:"customResources,omitempty" example:"widgets.example.com"`
}

type McpRuntimeMetadata struct {
	// Suggested port exposed by the server process.
	DefaultPort *int32 `json:"defaultPort,omitempty" example:"8080"`
	// Suggested command-line arguments for the server process.
	DefaultArgs []string `json:"defaultArgs,omitempty" example:"--transport,streamable-http"`
	// Environment variables that the server requires.
	RequiredEnvironmentVariables []McpEnvVarMetadata `json:"requiredEnvironmentVariables,omitempty"`
	// Optional environment variables supported by the server.
	OptionalEnvironmentVariables []McpEnvVarMetadata `json:"optionalEnvironmentVariables,omitempty"`
	// Suggested CPU and memory allocations for different workloads.
	RecommendedResources *McpResourceRecommendation `json:"recommendedResources,omitempty"`
	// Liveness and readiness probe paths.
	HealthEndpoints *McpRuntimeMetadataHealthEndpoints `json:"healthEndpoints,omitempty"`
	// Network, filesystem, and accelerator requirements.
	Capabilities *McpRuntimeMetadataCapabilities `json:"capabilities,omitempty"`
	// HTTP path at which the server exposes MCP.
	McpPath *string `json:"mcpPath,omitempty" example:"/mcp"`
	// Kubernetes resources and configuration needed before deployment.
	Prerequisites *McpPrerequisites `json:"prerequisites,omitempty"`
	// Volume mounts needed by the server.
	Storage []McpStorageMount `json:"storage,omitempty"`
}

type McpStoragePermissions string

const (
	McpStoragePermissionsReadOnly  McpStoragePermissions = "ReadOnly"
	McpStoragePermissionsReadWrite McpStoragePermissions = "ReadWrite"
)

type McpStorageSourceType string

const (
	McpStorageSourceTypeEmptyDir  McpStorageSourceType = "EmptyDir"
	McpStorageSourceTypeConfigMap McpStorageSourceType = "ConfigMap"
	McpStorageSourceTypeSecret    McpStorageSourceType = "Secret"
)

// McpStorageMount describes a catalog storage mount for an MCPServer.
type McpStorageMount struct {
	// Container path at which the volume is mounted.
	Path string `json:"path" example:"/data"`
	// Whether the mount is read-only or read-write.
	Permissions *McpStoragePermissions `json:"permissions,omitempty" example:"ReadOnly"`
	// Backing Kubernetes volume source.
	Source McpStorageSource `json:"source"`
}

// McpStorageSource preserves free-form catalog volume options. Pointers retain
// explicit empty objects (such as emptyDir: {}) while omitting absent options.
type McpStorageSource struct {
	// Volume source kind selecting emptyDir, ConfigMap, or Secret options.
	Type McpStorageSourceType `json:"type" example:"EmptyDir"`
	// Free-form Kubernetes emptyDir options; an explicit empty object is preserved.
	EmptyDir *map[string]interface{} `json:"emptyDir,omitempty"`
	// Free-form Kubernetes ConfigMap volume options.
	ConfigMap *map[string]interface{} `json:"configMap,omitempty"`
	// Free-form Kubernetes Secret volume options.
	Secret *map[string]interface{} `json:"secret,omitempty"`
}

type McpToolParameter struct {
	// Parameter name accepted by the tool.
	Name string `json:"name" example:"query"`
	// Expected parameter value type.
	Type string `json:"type" example:"string"`
	// Purpose and expected contents of the parameter.
	Description *string `json:"description,omitempty" example:"Search terms to look up."`
	// Whether the tool requires this parameter.
	Required bool `json:"required" example:"true"`
}

type McpTool struct {
	// Tool identifier exposed by the server.
	Name string `json:"name" example:"search"`
	// Human-readable description of the tool's behavior.
	Description *string `json:"description,omitempty" example:"Searches the document index."`
	// Access classification: read_only, read_write, or execute.
	AccessType McpToolAccessType `json:"accessType" example:"read_only"`
	// Parameters accepted by the tool.
	Parameters []McpToolParameter `json:"parameters,omitempty"`
	// Whether the tool has been revoked from use.
	Revoked *bool `json:"revoked,omitempty" example:"false"`
	// Explanation of why the tool was revoked, when applicable.
	RevokedReason *string `json:"revokedReason,omitempty" example:"Disabled by the provider."`
	// Custom properties keyed by name; each value is selected by metadataType.
	CustomProperties *map[string]openapi.MetadataValue `json:"customProperties,omitempty"`
}

// McpServer is a server discovered from the MCP catalog.
// @Description MCP server metadata, tools, transports, security indicators, and deployment requirements.
type McpServer struct {
	// Unique MCP server identifier.
	ID string `json:"id" example:"1"`
	// Server name within its catalog source.
	Name string `json:"name" example:"document-search"`
	// Human-readable display name.
	DisplayName *string `json:"displayName,omitempty" example:"Document Search"`
	// Catalog source providing this server.
	SourceID *string `json:"source_id,omitempty" example:"community"`
	// Short description of the server's capabilities.
	Description *string `json:"description,omitempty" example:"Search and retrieve documents using MCP."`
	// URL or path of the server's logo image.
	Logo *string `json:"logo,omitempty" example:"https://example.com/mcp-logo.png"`
	// Short name or identifier of the server's license.
	License *string `json:"license,omitempty" example:"Apache-2.0"`
	// URL of the full license text.
	LicenseLink *string `json:"licenseLink,omitempty" format:"uri" example:"https://www.apache.org/licenses/LICENSE-2.0"`
	// Organization or entity that provides the server.
	Provider *string `json:"provider,omitempty" example:"Example Organization"`
	// Published server version.
	Version *string `json:"version,omitempty" example:"1.0.0"`
	// Labels used to categorize and discover the server.
	Tags []string `json:"tags,omitempty" example:"search,documents"`
	// Number of tools reported by the catalog for this server.
	ToolCount int `json:"toolCount" example:"2"`
	// Tool metadata included when requested and available.
	Tools []McpTool `json:"tools,omitempty"`
	// Catalog-provided security indicators, not guarantees of security.
	SecurityIndicators *McpSecurityIndicator `json:"securityIndicators,omitempty"`
	// URL of the server's external documentation.
	DocumentationURL *string `json:"documentationUrl,omitempty" format:"uri" example:"https://example.com/mcp/docs"`
	// URL of the server's source repository.
	RepositoryURL *string `json:"repositoryUrl,omitempty" format:"uri" example:"https://github.com/example/mcp-server"`
	// Source code information supplied by the catalog.
	SourceCode *string `json:"sourceCode,omitempty" example:"https://github.com/example/mcp-server"`
	// Last update date or timestamp supplied by the catalog.
	LastUpdated *string `json:"lastUpdated,omitempty" example:"2026-01-15"`
	// Publication date or timestamp supplied by the catalog.
	PublishedDate *string `json:"publishedDate,omitempty" example:"2026-01-01"`
	// Artifacts used to deploy the server.
	Artifacts []McpArtifact `json:"artifacts,omitempty"`
	// Supported MCP transport protocols.
	Transports []McpTransportType `json:"transports,omitempty" example:"stdio,http"`
	// Full server documentation in Markdown.
	Readme *string `json:"readme,omitempty" example:"# Document Search MCP Server"`
	// Whether the server is deployed locally or accessed remotely.
	DeploymentMode *McpDeploymentMode `json:"deploymentMode,omitempty" example:"local"`
	// Remote HTTP or SSE endpoints, when applicable.
	Endpoints *McpEndpoints `json:"endpoints,omitempty"`
	// Runtime configuration, resource recommendations, and deployment prerequisites.
	RuntimeMetadata *McpRuntimeMetadata `json:"runtimeMetadata,omitempty"`
	// Custom properties keyed by name; each value is selected by metadataType.
	CustomProperties *map[string]openapi.MetadataValue `json:"customProperties,omitempty"`
	// Free-form upstream server.json metadata preserved by the catalog.
	ServerJson map[string]interface{} `json:"serverJson,omitempty"`
}

type McpServerList struct {
	// Opaque token for retrieving the next page; empty on the final page.
	NextPageToken string `json:"nextPageToken"`
	// Maximum number of servers requested per page.
	PageSize int32 `json:"pageSize" example:"10"`
	// Number of servers returned in this page.
	Size int32 `json:"size" example:"1"`
	// Servers returned in this page.
	Items []McpServer `json:"items"`
}

type McpToolWithServer struct {
	// Identifier of the server exposing this tool.
	ServerID string `json:"serverId" example:"1"`
	// Tool metadata paired with its server identifier.
	Tool McpTool `json:"tool"`
}

type McpToolList struct {
	// Opaque token for retrieving the next page; empty on the final page.
	NextPageToken string `json:"nextPageToken"`
	// Maximum number of tools requested per page.
	PageSize int32 `json:"pageSize" example:"10"`
	// Number of tools returned in this page.
	Size int32 `json:"size" example:"1"`
	// Tools paired with the identifier of their server.
	Items []McpToolWithServer `json:"items"`
}
