package mcpcatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"

	"github.com/golang/glog"
	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog/models"
	apimodels "github.com/kubeflow/hub/catalog/pkg/openapi"
	mrmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	yamlMCPCatalogPathKey = "yamlCatalogPath"
	maxMCPDisplayNameLen  = 255
)

// yamlMCPParameter represents a single input parameter for an MCP tool.
type yamlMCPParameter struct {
	Name        string  `yaml:"name" json:"name"`
	Type        string  `yaml:"type" json:"type"`
	Description *string `yaml:"description,omitempty" json:"description,omitempty"`
	Required    bool    `yaml:"required" json:"required"`
}

// MCPToolRecord carries the data for a single MCP tool from a provider.
type MCPToolRecord struct {
	Name        string
	Description *string
	Schema      *string
	AccessType  *string
	Parameters  []yamlMCPParameter
}

// MCPServerProviderRecord represents a single MCP server from a provider along with its tools
type MCPServerProviderRecord struct {
	Server *models.MCPServerImpl
	Tools  []MCPToolRecord
	Error  error
}

// MCPServerProviderFunc is a function that provides MCP servers from a source
type MCPServerProviderFunc func(basecatalog.MCPSource) (MCPProvider, error)

// MCPProvider is an interface for providers of MCP servers
type MCPProvider interface {
	Servers(ctx context.Context) <-chan MCPServerProviderRecord
}

// Provider registration

var (
	registeredMCPProviders   = make(map[string]MCPServerProviderFunc)
	registeredMCPProvidersMu sync.RWMutex
)

// RegisterMCPProvider registers an MCP provider function by type name
func RegisterMCPProvider(typeName string, providerFunc MCPServerProviderFunc) error {
	registeredMCPProvidersMu.Lock()
	defer registeredMCPProvidersMu.Unlock()

	if _, exists := registeredMCPProviders[typeName]; exists {
		return fmt.Errorf("MCP provider %q is already registered", typeName)
	}

	registeredMCPProviders[typeName] = providerFunc
	return nil
}

// unregisterMCPProvider removes a registered provider by type name.
// This is only intended for use in tests to clean up global state.
func unregisterMCPProvider(typeName string) {
	registeredMCPProvidersMu.Lock()
	defer registeredMCPProvidersMu.Unlock()

	delete(registeredMCPProviders, typeName)
}

// GetMCPProvider retrieves a registered MCP provider by type name
func GetMCPProvider(typeName string) (MCPServerProviderFunc, bool) {
	registeredMCPProvidersMu.RLock()
	defer registeredMCPProvidersMu.RUnlock()

	providerFunc, exists := registeredMCPProviders[typeName]
	return providerFunc, exists
}

// yamlMCPSecurityIndicator represents the security indicators for an MCP server
type yamlMCPSecurityIndicator struct {
	VerifiedSource *bool `yaml:"verifiedSource,omitempty"`
	SecureEndpoint *bool `yaml:"secureEndpoint,omitempty"`
	Sast           *bool `yaml:"sast,omitempty"`
	ReadOnlyTools  *bool `yaml:"readOnlyTools,omitempty"`
}

// yamlMCPServer represents an MCP server definition in YAML
type yamlMCPServer struct {
	Name                     string                              `yaml:"name"`
	DisplayName              *string                             `yaml:"displayName,omitempty"`
	ExternalID               *string                             `yaml:"externalId,omitempty"`
	Description              *string                             `yaml:"description,omitempty"`
	Provider                 *string                             `yaml:"provider,omitempty"`
	Version                  *string                             `yaml:"version,omitempty"`
	Logo                     *string                             `yaml:"logo,omitempty"`
	License                  *string                             `yaml:"license,omitempty"`
	LicenseLink              *string                             `yaml:"license_link,omitempty"`
	DocumentationUrl         *string                             `yaml:"documentationUrl,omitempty"`
	RepositoryUrl            *string                             `yaml:"repositoryUrl,omitempty"`
	SourceCode               *string                             `yaml:"sourceCode,omitempty"`
	Readme                   *string                             `yaml:"readme,omitempty"`
	PublishedDate            *string                             `yaml:"publishedDate,omitempty"`
	Transports               []string                            `yaml:"transports,omitempty"`
	Tools                    []*yamlMCPTool                      `yaml:"tools,omitempty"`
	Artifacts                []apimodels.MCPArtifact             `yaml:"artifacts,omitempty"`
	DeploymentMode           *string                             `yaml:"deploymentMode,omitempty"`
	Endpoints                *yamlMCPEndpoints                   `yaml:"endpoints,omitempty"`
	RuntimeMetadata          *apimodels.MCPRuntimeMetadata       `yaml:"runtimeMetadata,omitempty"`
	Tags                     []string                            `yaml:"tags,omitempty"`
	SecurityIndicators       *yamlMCPSecurityIndicator           `yaml:"securityIndicators,omitempty"`
	CustomProperties         *map[string]apimodels.MetadataValue `yaml:"customProperties,omitempty"`
	CreateTimeSinceEpoch     *string                             `yaml:"createTimeSinceEpoch,omitempty"`
	LastUpdateTimeSinceEpoch *string                             `yaml:"lastUpdateTimeSinceEpoch,omitempty"`
}

// yamlMCPTool represents an MCP tool definition
type yamlMCPTool struct {
	Name        string             `yaml:"name"`
	Description *string            `yaml:"description,omitempty"`
	Schema      *string            `yaml:"schema,omitempty"`
	AccessType  *string            `yaml:"accessType,omitempty"`
	Parameters  []yamlMCPParameter `yaml:"parameters,omitempty"`
}

// yamlMCPEndpoints represents MCP server endpoints
type yamlMCPEndpoints struct {
	HTTP      *string `yaml:"http,omitempty" json:"http,omitempty"`
	SSE       *string `yaml:"sse,omitempty" json:"sse,omitempty"`
	WebSocket *string `yaml:"websocket,omitempty" json:"websocket,omitempty"`
}

// validate checks that all non-nil endpoint URLs have valid schemes.
// HTTP and SSE endpoints must use http/https; WebSocket endpoints also accept ws/wss.
func (e *yamlMCPEndpoints) validate() error {
	type endpointField struct {
		name    string
		url     *string
		schemes []string
	}
	fields := []endpointField{
		{"http", e.HTTP, []string{"http", "https"}},
		{"sse", e.SSE, []string{"http", "https"}},
		{"websocket", e.WebSocket, []string{"http", "https", "ws", "wss"}},
	}
	for _, f := range fields {
		if f.url == nil {
			continue
		}
		u, err := url.Parse(*f.url)
		if err != nil || !slices.Contains(f.schemes, u.Scheme) {
			return fmt.Errorf("%s endpoint %q must be a valid URL with one of the following schemes: %v", f.name, *f.url, f.schemes)
		}
	}
	return nil
}

// yamlMCPCatalog represents a complete MCP catalog YAML file
type yamlMCPCatalog struct {
	MCPServers []*yamlMCPServer `yaml:"mcp_servers" json:"mcp_servers"`
}

// yamlMCPProvider implements MCPProvider for YAML files
type yamlMCPProvider struct {
	paths []string
}

// NewYamlMCPProvider creates a new YAML MCP provider
func NewYamlMCPProvider(source basecatalog.MCPSource) (MCPProvider, error) {
	yamlPath, ok := source.Properties[yamlMCPCatalogPathKey].(string)
	if !ok {
		return nil, fmt.Errorf("yamlCatalogPath property is required for YAML MCP provider")
	}

	paths := []string{}
	if filepath.IsAbs(yamlPath) {
		paths = append(paths, yamlPath)
	} else {
		// Resolve relative paths relative to the source config file's directory.
		// This matches the model loader behavior and allows sources from different
		// config files (e.g., mounted from different configmaps) to use relative
		// paths correctly.
		sourceDir := filepath.Dir(source.Origin)
		absPath := filepath.Join(sourceDir, yamlPath)
		paths = append(paths, absPath)
	}

	return &yamlMCPProvider{
		paths: paths,
	}, nil
}

// Servers implements MCPProvider
func (yp *yamlMCPProvider) Servers(ctx context.Context) <-chan MCPServerProviderRecord {
	recordChan := make(chan MCPServerProviderRecord)

	go func() {
		defer close(recordChan)

		for _, path := range yp.paths {
			select {
			case <-ctx.Done():
				return
			default:
				yp.emit(ctx, path, recordChan)
			}
		}
	}()

	return recordChan
}

// read reads and parses a YAML catalog file
func (yp *yamlMCPProvider) read(path string) (*yamlMCPCatalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading YAML file %s: %w", path, err)
	}

	var catalog yamlMCPCatalog
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("error parsing YAML from %s: %w", path, err)
	}

	return &catalog, nil
}

// emit reads a YAML file and emits MCP server records
func (yp *yamlMCPProvider) emit(ctx context.Context, path string, recordChan chan<- MCPServerProviderRecord) {
	catalog, err := yp.read(path)
	if err != nil {
		glog.Errorf("Error reading MCP catalog from %s: %v", path, err)
		recordChan <- MCPServerProviderRecord{Error: err}
		return
	}

	for _, yamlServer := range catalog.MCPServers {
		select {
		case <-ctx.Done():
			return
		default:
			record := yamlServer.ToMCPServerProviderRecord()
			recordChan <- record
		}
	}
}

// ToMCPServerProviderRecord converts a yamlMCPServer to an MCPServerProviderRecord
func (ys *yamlMCPServer) ToMCPServerProviderRecord() MCPServerProviderRecord {
	if ys.DisplayName != nil && len(*ys.DisplayName) > maxMCPDisplayNameLen {
		return MCPServerProviderRecord{
			Error: fmt.Errorf(
				"server %q displayName exceeds maximum length of %d characters",
				ys.Name,
				maxMCPDisplayNameLen,
			),
		}
	}

	attrs := &models.MCPServerAttributes{
		Name:       &ys.Name,
		ExternalID: ys.ExternalID,
	}

	// Convert timestamps
	if ys.CreateTimeSinceEpoch != nil {
		if createTime, err := strconv.ParseInt(*ys.CreateTimeSinceEpoch, 10, 64); err == nil {
			attrs.CreateTimeSinceEpoch = &createTime
		}
	}

	if ys.LastUpdateTimeSinceEpoch != nil {
		if updateTime, err := strconv.ParseInt(*ys.LastUpdateTimeSinceEpoch, 10, 64); err == nil {
			attrs.LastUpdateTimeSinceEpoch = &updateTime
		}
	}

	server := &models.MCPServerImpl{
		Attributes: attrs,
	}

	// Convert standard properties
	properties := []mrmodels.Properties{}

	if ys.DisplayName != nil {
		properties = append(properties, mrmodels.NewStringProperty("displayName", *ys.DisplayName, false))
	}
	if ys.Description != nil {
		properties = append(properties, mrmodels.NewStringProperty("description", *ys.Description, false))
	}
	if ys.Provider != nil {
		properties = append(properties, mrmodels.NewStringProperty("provider", *ys.Provider, false))
	}
	if ys.Version != nil {
		properties = append(properties, mrmodels.NewStringProperty("version", *ys.Version, false))
	}
	if ys.Logo != nil {
		properties = append(properties, mrmodels.NewStringProperty("logo", *ys.Logo, false))
	}
	if ys.License != nil {
		humanReadableLicense := basecatalog.TransformLicenseToHumanReadable(*ys.License)
		properties = append(properties, mrmodels.NewStringProperty("license", humanReadableLicense, false))
	}
	if ys.LicenseLink != nil {
		properties = append(properties, mrmodels.NewStringProperty("license_link", *ys.LicenseLink, false))
	}
	if ys.DocumentationUrl != nil {
		properties = append(properties, mrmodels.NewStringProperty("documentationUrl", *ys.DocumentationUrl, false))
	}
	if ys.RepositoryUrl != nil {
		properties = append(properties, mrmodels.NewStringProperty("repositoryUrl", *ys.RepositoryUrl, false))
	}
	if ys.SourceCode != nil {
		properties = append(properties, mrmodels.NewStringProperty("sourceCode", *ys.SourceCode, false))
	}
	if ys.Readme != nil {
		properties = append(properties, mrmodels.NewStringProperty("readme", *ys.Readme, false))
	}
	if ys.PublishedDate != nil {
		properties = append(properties, mrmodels.NewStringProperty("publishedDate", *ys.PublishedDate, false))
	}
	if ys.DeploymentMode != nil {
		properties = append(properties, mrmodels.NewStringProperty("deploymentMode", *ys.DeploymentMode, false))
	}

	// Convert array properties to JSON strings
	if len(ys.Transports) > 0 {
		if jsonBytes, err := json.Marshal(ys.Transports); err == nil {
			properties = append(properties, mrmodels.NewStringProperty("transports", string(jsonBytes), false))
		}
	}

	if len(ys.Tags) > 0 {
		if jsonBytes, err := json.Marshal(ys.Tags); err == nil {
			properties = append(properties, mrmodels.NewStringProperty("tags", string(jsonBytes), false))
		}
	}

	// Convert tools to individual MCPToolRecord entries (stored as separate DB entities)
	var toolRecords []MCPToolRecord
	for _, tool := range ys.Tools {
		toolRecords = append(toolRecords, MCPToolRecord{
			Name:        tool.Name,
			Description: tool.Description,
			Schema:      tool.Schema,
			AccessType:  tool.AccessType,
			Parameters:  tool.Parameters,
		})
	}

	// Validate and convert artifacts to JSON
	if len(ys.Artifacts) > 0 {
		for i, artifact := range ys.Artifacts {
			if err := basecatalog.ValidateArtifactURI(artifact.Uri); err != nil {
				return MCPServerProviderRecord{Error: fmt.Errorf("server %q artifact %d: %w", ys.Name, i, err)}
			}
		}
		if jsonBytes, err := json.Marshal(ys.Artifacts); err == nil {
			properties = append(properties, mrmodels.NewStringProperty("artifacts", string(jsonBytes), false))
		}
	}

	// Validate remote servers have at least one endpoint
	if ys.DeploymentMode != nil && *ys.DeploymentMode == "remote" {
		if ys.Endpoints == nil || (ys.Endpoints.HTTP == nil && ys.Endpoints.SSE == nil && ys.Endpoints.WebSocket == nil) {
			return MCPServerProviderRecord{Error: fmt.Errorf("server %q has deploymentMode 'remote' but no endpoints defined", ys.Name)}
		}
	}

	// Convert runtimeMetadata to JSON
	if ys.RuntimeMetadata != nil {
		if jsonBytes, err := json.Marshal(ys.RuntimeMetadata); err == nil {
			properties = append(properties, mrmodels.NewStringProperty("runtimeMetadata", string(jsonBytes), false))
		} else {
			glog.Warningf("failed to marshal runtimeMetadata for server %q: %v", ys.Name, err)
		}
	}

	// Convert endpoints to JSON
	if ys.Endpoints != nil {
		if err := ys.Endpoints.validate(); err != nil {
			return MCPServerProviderRecord{Error: fmt.Errorf("server %q has invalid endpoints: %w", ys.Name, err)}
		}
		if jsonBytes, err := json.Marshal(ys.Endpoints); err == nil {
			properties = append(properties, mrmodels.NewStringProperty("endpoints", string(jsonBytes), false))
		}
	}

	// Convert security indicators to standard bool properties
	if ys.SecurityIndicators != nil {
		si := ys.SecurityIndicators
		if si.VerifiedSource != nil {
			properties = append(properties, mrmodels.NewBoolProperty("verifiedSource", *si.VerifiedSource, false))
		}
		if si.SecureEndpoint != nil {
			properties = append(properties, mrmodels.NewBoolProperty("secureEndpoint", *si.SecureEndpoint, false))
		}
		if si.Sast != nil {
			properties = append(properties, mrmodels.NewBoolProperty("sast", *si.Sast, false))
		}
		if si.ReadOnlyTools != nil {
			properties = append(properties, mrmodels.NewBoolProperty("readOnlyTools", *si.ReadOnlyTools, false))
		}
	}

	server.Properties = &properties

	// Convert custom properties
	if ys.CustomProperties != nil {
		customProps := []mrmodels.Properties{}
		for key, value := range *ys.CustomProperties {
			customProps = append(customProps, convertMetadataValueToProperty(key, value))
		}
		server.CustomProperties = &customProps
	}

	return MCPServerProviderRecord{
		Server: server,
		Tools:  toolRecords,
		Error:  nil,
	}
}

// convertMetadataValueToProperty converts a MetadataValue to a Properties object
func convertMetadataValueToProperty(key string, value apimodels.MetadataValue) mrmodels.Properties {
	// Handle different MetadataValue types
	if value.MetadataStringValue != nil {
		return mrmodels.NewStringProperty(key, value.MetadataStringValue.StringValue, true)
	} else if value.MetadataIntValue != nil {
		// MetadataIntValue.IntValue is a string, need to convert to int32
		if intVal, err := strconv.ParseInt(value.MetadataIntValue.IntValue, 10, 32); err == nil {
			return mrmodels.NewIntProperty(key, int32(intVal), true)
		} else {
			// If parsing fails, store as string
			return mrmodels.NewStringProperty(key, value.MetadataIntValue.IntValue, true)
		}
	} else if value.MetadataDoubleValue != nil {
		return mrmodels.NewDoubleProperty(key, value.MetadataDoubleValue.DoubleValue, true)
	} else if value.MetadataBoolValue != nil {
		return mrmodels.NewBoolProperty(key, value.MetadataBoolValue.BoolValue, true)
	} else {
		// For complex types, serialize to JSON
		if jsonBytes, err := json.Marshal(value); err == nil {
			return mrmodels.NewStringProperty(key, string(jsonBytes), true)
		}
		// Fallback to empty string if JSON marshaling fails
		return mrmodels.NewStringProperty(key, "", true)
	}
}

func init() {
	if err := RegisterMCPProvider("yaml", NewYamlMCPProvider); err != nil {
		panic(err)
	}
}
