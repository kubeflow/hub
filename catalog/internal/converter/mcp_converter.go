package converter

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog/models"
	"github.com/kubeflow/hub/catalog/pkg/openapi"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
)

// ==============================================================================
// MCP Server Conversions
// ==============================================================================

// ConvertOpenapiMCPServerToDb converts an OpenAPI MCPServer to a database MCPServer model.
// This sets all properties from the OpenAPI struct into the database model's Properties field.
func ConvertOpenapiMCPServerToDb(openapiServer *openapi.MCPServer) models.MCPServer {
	dbServer := &models.MCPServerImpl{
		Attributes: &models.MCPServerAttributes{
			Name:                     &openapiServer.Name,
			ExternalID:               openapiServer.ExternalId,
			CreateTimeSinceEpoch:     parseEpochString(openapiServer.CreateTimeSinceEpoch),
			LastUpdateTimeSinceEpoch: parseEpochString(openapiServer.LastUpdateTimeSinceEpoch),
		},
	}

	// Build properties array from OpenAPI fields
	properties := []dbmodels.Properties{}

	// Simple string fields
	addStringProperty(&properties, "source_id", openapiServer.SourceId)
	addStringProperty(&properties, "displayName", openapiServer.DisplayName)
	addStringProperty(&properties, "provider", openapiServer.Provider)
	addStringProperty(&properties, "logo", openapiServer.Logo)
	addStringProperty(&properties, "version", openapiServer.Version)
	addStringProperty(&properties, "license", openapiServer.License)
	addStringProperty(&properties, "license_link", openapiServer.LicenseLink)
	addStringProperty(&properties, "readme", openapiServer.Readme)
	addStringProperty(&properties, "deploymentMode", openapiServer.DeploymentMode)
	addStringProperty(&properties, "documentationUrl", openapiServer.DocumentationUrl)
	addStringProperty(&properties, "repositoryUrl", openapiServer.RepositoryUrl)
	addStringProperty(&properties, "sourceCode", openapiServer.SourceCode)
	addStringProperty(&properties, "description", openapiServer.Description)

	// Array fields (stored as JSON)
	addArrayProperty(&properties, "tags", openapiServer.Tags)
	addArrayProperty(&properties, "transports", openapiServer.Transports)

	// Time fields (convert time.Time to ISO 8601 string for content timestamps)
	if publishedDate, ok := openapiServer.GetPublishedDateOk(); ok && publishedDate != nil {
		timeStr := formatTimeToISO8601(publishedDate)
		addStringProperty(&properties, "publishedDate", &timeStr)
	}
	if lastUpdated, ok := openapiServer.GetLastUpdatedOk(); ok && lastUpdated != nil {
		timeStr := formatTimeToISO8601(lastUpdated)
		addStringProperty(&properties, "lastUpdated", &timeStr)
	}

	// Security indicators (store as individual boolean properties)
	if securityIndicators, ok := openapiServer.GetSecurityIndicatorsOk(); ok && securityIndicators != nil {
		addBoolProperty(&properties, "verifiedSource", securityIndicators.VerifiedSource)
		addBoolProperty(&properties, "secureEndpoint", securityIndicators.SecureEndpoint)
		addBoolProperty(&properties, "sast", securityIndicators.Sast)
		addBoolProperty(&properties, "readOnlyTools", securityIndicators.ReadOnlyTools)
	}

	// Complex objects (store as JSON)
	if endpoints, ok := openapiServer.GetEndpointsOk(); ok && endpoints != nil {
		addJSONProperty(&properties, "endpoints", endpoints)
	}
	if len(openapiServer.Artifacts) > 0 {
		addJSONProperty(&properties, "artifacts", openapiServer.Artifacts)
	}
	if runtimeMetadata, ok := openapiServer.GetRuntimeMetadataOk(); ok && runtimeMetadata != nil {
		addJSONProperty(&properties, "runtimeMetadata", runtimeMetadata)
	}

	// Set properties on the server
	dbServer.Properties = &properties

	// Handle custom properties
	if len(openapiServer.CustomProperties) > 0 {
		customProps := []dbmodels.Properties{}
		for name, value := range openapiServer.CustomProperties {
			customProps = append(customProps, convertMetadataValueToProperty(name, value))
		}
		dbServer.CustomProperties = &customProps
	}

	return dbServer
}

// ConvertDbMCPServerToOpenapi converts a database MCPServer model to an OpenAPI MCPServer.
// This extracts all properties from the database model and populates the OpenAPI struct.
// It does NOT populate ToolCount or the Tools array — callers MUST set ToolCount
// (e.g. via CountByParentIDs) to avoid returning an incorrect zero value.
func ConvertDbMCPServerToOpenapi(dbServer models.MCPServer) *openapi.MCPServer {
	return convertDbMCPServerToOpenapiInternal(dbServer, nil)
}

// ConvertDbMCPServerWithToolsToOpenapi converts a database MCPServer to OpenAPI representation
// with accurate tool count and tools array populated from the provided tools.
//
// Parameters:
//   - dbServer: The database server entity to convert
//   - tools: Associated tools (can be nil or empty for toolCount=0)
//
// This is the preferred method when tools have been loaded via repository queries.
func ConvertDbMCPServerWithToolsToOpenapi(dbServer models.MCPServer, tools []models.MCPServerTool) *openapi.MCPServer {
	openapiTools := make([]openapi.MCPTool, 0, len(tools))
	for _, tool := range tools {
		if converted := ConvertDbMCPToolToOpenapi(tool); converted != nil {
			openapiTools = append(openapiTools, *converted)
		}
	}
	return convertDbMCPServerToOpenapiInternal(dbServer, openapiTools)
}

// convertDbMCPServerToOpenapiInternal is the shared implementation for server conversion.
// ToolCount is derived from len(tools) here as a preliminary value; callers MUST
// override it with the accurate total from CountByParentIDs when a ToolLimit is applied.
func convertDbMCPServerToOpenapiInternal(dbServer models.MCPServer, tools []openapi.MCPTool) *openapi.MCPServer {
	attrs := dbServer.GetAttributes()
	props := dbServer.GetProperties()

	// Create property accessor for O(1) lookups
	pa := NewPropertyAccessor(props)

	// Extract base name and version
	baseName := ""
	if attrs != nil && attrs.Name != nil {
		baseName = *attrs.Name
	}
	version := pa.GetString("version")

	// Compute tool count from provided tools
	toolCount := int32(0)
	if tools != nil {
		toolCount = int32(len(tools))
	}

	openapiServer := &openapi.MCPServer{
		Name:      baseName,
		ToolCount: toolCount,
	}

	// Set tools if provided
	if len(tools) > 0 {
		openapiServer.Tools = tools
	}

	// Set core attributes
	if attrs != nil {
		openapiServer.ExternalId = attrs.ExternalID
		openapiServer.CreateTimeSinceEpoch = formatEpochToString(attrs.CreateTimeSinceEpoch)
		openapiServer.LastUpdateTimeSinceEpoch = formatEpochToString(attrs.LastUpdateTimeSinceEpoch)
	}

	// Set ID
	if dbServer.GetID() != nil {
		idStr := fmt.Sprintf("%d", *dbServer.GetID())
		openapiServer.Id = &idStr
	}

	// Extract simple string properties
	openapiServer.SourceId = pa.GetStringPtr("source_id")
	openapiServer.DisplayName = pa.GetStringPtr("displayName")
	openapiServer.Provider = pa.GetStringPtr("provider")
	openapiServer.Logo = pa.GetStringPtr("logo")
	openapiServer.Version = &version
	openapiServer.License = pa.GetStringPtr("license")
	openapiServer.LicenseLink = pa.GetStringPtr("license_link")
	openapiServer.Readme = pa.GetStringPtr("readme")
	openapiServer.DeploymentMode = pa.GetStringPtr("deploymentMode")
	openapiServer.DocumentationUrl = pa.GetStringPtr("documentationUrl")
	openapiServer.RepositoryUrl = pa.GetStringPtr("repositoryUrl")
	openapiServer.SourceCode = pa.GetStringPtr("sourceCode")
	openapiServer.Description = pa.GetStringPtr("description")

	// Extract array properties
	openapiServer.Tags = pa.GetStringArray("tags")
	openapiServer.Transports = pa.GetStringArray("transports")

	// Extract time properties (parse ISO 8601 strings for content timestamps)
	openapiServer.PublishedDate = parseISO8601ToTime(pa.GetString("publishedDate"))
	openapiServer.LastUpdated = parseISO8601ToTime(pa.GetString("lastUpdated"))

	// Extract security indicators
	if pa.HasAny("verifiedSource", "secureEndpoint", "sast", "readOnlyTools") {
		openapiServer.SecurityIndicators = &openapi.MCPSecurityIndicator{
			VerifiedSource: pa.GetBoolPtr("verifiedSource"),
			SecureEndpoint: pa.GetBoolPtr("secureEndpoint"),
			Sast:           pa.GetBoolPtr("sast"),
			ReadOnlyTools:  pa.GetBoolPtr("readOnlyTools"),
		}
	}

	// Extract complex objects from JSON
	if endpointsJSON := pa.GetString("endpoints"); endpointsJSON != "" {
		var endpoints openapi.MCPEndpoints
		if err := json.Unmarshal([]byte(endpointsJSON), &endpoints); err != nil {
			glog.Warningf("Failed to unmarshal endpoints JSON: %v", err)
		} else {
			openapiServer.Endpoints = &endpoints
		}
	}
	if artifactsJSON := pa.GetString("artifacts"); artifactsJSON != "" {
		var artifacts []openapi.MCPArtifact
		if err := json.Unmarshal([]byte(artifactsJSON), &artifacts); err != nil {
			glog.Warningf("Failed to unmarshal artifacts JSON: %v", err)
		} else {
			openapiServer.Artifacts = artifacts
		}
	}
	if runtimeJSON := pa.GetString("runtimeMetadata"); runtimeJSON != "" {
		var runtimeMetadata openapi.MCPRuntimeMetadata
		if err := json.Unmarshal([]byte(runtimeJSON), &runtimeMetadata); err == nil {
			openapiServer.RuntimeMetadata = &runtimeMetadata
		}
	}

	// Convert custom properties
	if dbServer.GetCustomProperties() != nil {
		customProps := make(map[string]openapi.MetadataValue)
		for _, prop := range *dbServer.GetCustomProperties() {
			customProps[prop.Name] = convertPropertyToMetadataValue(prop)
		}
		openapiServer.CustomProperties = customProps
	}

	return openapiServer
}

// ==============================================================================
// MCP Tool Conversions
// ==============================================================================

// ConvertOpenapiMCPToolToDb converts an OpenAPI MCPTool to database MCPServerTool.
func ConvertOpenapiMCPToolToDb(openapiTool *openapi.MCPTool) models.MCPServerTool {
	dbTool := &models.MCPServerToolImpl{
		Attributes: &models.MCPServerToolAttributes{
			Name:                     &openapiTool.Name,
			CreateTimeSinceEpoch:     parseEpochString(openapiTool.CreateTimeSinceEpoch),
			LastUpdateTimeSinceEpoch: parseEpochString(openapiTool.LastUpdateTimeSinceEpoch),
		},
	}

	// Build properties
	properties := []dbmodels.Properties{}

	// Required field: accessType
	accessType := openapiTool.AccessType
	addStringProperty(&properties, "accessType", &accessType)

	// Optional field: description
	addStringProperty(&properties, "description", openapiTool.Description)

	// Optional field: parameters (JSON array)
	if len(openapiTool.Parameters) > 0 {
		addJSONProperty(&properties, "parameters", openapiTool.Parameters)
	}

	// Handle ExternalID if present (store as property since MCPServerToolAttributes doesn't have this field)
	if openapiTool.ExternalId != nil {
		addStringProperty(&properties, "externalId", openapiTool.ExternalId)
	}

	// Handle custom properties
	if customProps := openapiTool.GetCustomProperties(); len(customProps) > 0 {
		customProperties := []dbmodels.Properties{}
		for key, value := range customProps {
			customProperties = append(customProperties, convertMetadataValueToProperty(key, value))
		}
		dbTool.CustomProperties = &customProperties
	}

	dbTool.Properties = &properties
	return dbTool
}

// UnqualifyToolName strips the "serverName@version:" prefix from a qualified
// tool name, returning just the tool name portion. If the name contains no
// colon, it is returned unchanged.
func UnqualifyToolName(name string) string {
	if idx := strings.LastIndex(name, ":"); idx != -1 {
		return name[idx+1:]
	}
	return name
}

// ConvertDbMCPToolToOpenapi converts a database MCPServerTool to OpenAPI MCPTool.
func ConvertDbMCPToolToOpenapi(dbTool models.MCPServerTool) *openapi.MCPTool {
	attr := dbTool.GetAttributes()
	if attr == nil || attr.Name == nil {
		return nil
	}

	// Create property accessor for O(1) lookups
	props := dbTool.GetProperties()
	pa := NewPropertyAccessor(props)

	// Get required accessType (required field in OpenAPI spec)
	accessType := pa.GetString("accessType")
	if accessType == "" {
		glog.Warningf("Required field 'accessType' missing for tool '%s', defaulting to 'read_only'", *attr.Name)
		accessType = "read_only" // default fallback to prevent API contract violation
	}

	toolName := UnqualifyToolName(*attr.Name)

	// Create OpenAPI tool with required fields
	openapiTool := openapi.NewMCPTool(toolName, accessType)

	// Set ID if available
	if id := dbTool.GetID(); id != nil {
		idStr := fmt.Sprintf("%d", *id)
		openapiTool.Id = &idStr
	}

	// Set timestamps (system timestamps converted to numeric strings)
	openapiTool.CreateTimeSinceEpoch = formatEpochToString(attr.CreateTimeSinceEpoch)
	openapiTool.LastUpdateTimeSinceEpoch = formatEpochToString(attr.LastUpdateTimeSinceEpoch)

	// Extract optional description
	openapiTool.Description = pa.GetStringPtr("description")

	// Extract externalId if present
	openapiTool.ExternalId = pa.GetStringPtr("externalId")

	// Extract parameters (JSON array)
	if paramsJSON := pa.GetString("parameters"); paramsJSON != "" {
		var parameters []openapi.MCPToolParameter
		if err := json.Unmarshal([]byte(paramsJSON), &parameters); err != nil {
			glog.Warningf("Failed to unmarshal tool parameters JSON: %v", err)
		} else {
			openapiTool.Parameters = parameters
		}
	}

	// Extract custom properties
	if customProps := dbTool.GetCustomProperties(); customProps != nil {
		customProperties := make(map[string]openapi.MetadataValue)
		for _, prop := range *customProps {
			customProperties[prop.Name] = convertPropertyToMetadataValue(prop)
		}
		openapiTool.CustomProperties = customProperties
	}

	return openapiTool
}

// ==============================================================================
// Property Helper Functions
// ==============================================================================
//
// NOTE: Two timestamp patterns are used in this codebase:
//
// 1. SYSTEM TIMESTAMPS (CreateTimeSinceEpoch, LastUpdateTimeSinceEpoch):
//    - Stored as int64 milliseconds in entity attributes (Context/Artifact tables)
//    - Converted to numeric strings for OpenAPI (e.g., "1704067200000")
//    - Managed automatically by the system
//
// 2. CONTENT TIMESTAMPS (PublishedDate, LastUpdated):
//    - Stored as ISO 8601 strings in properties (ContextProperty/ArtifactProperty tables)
//    - Parsed to/from time.Time for Go API layer
//    - Serialized as RFC3339 in JSON API
//    - Sourced from external data (catalogs, user input)
//
// ==============================================================================

// addStringProperty adds a string property if the value is not nil.
func addStringProperty(props *[]dbmodels.Properties, name string, value *string) {
	if value != nil && *value != "" {
		*props = append(*props, dbmodels.Properties{
			Name:        name,
			StringValue: value,
		})
	}
}

// addBoolProperty adds a boolean property if the value is not nil.
func addBoolProperty(props *[]dbmodels.Properties, name string, value *bool) {
	if value != nil {
		*props = append(*props, dbmodels.Properties{
			Name:      name,
			BoolValue: value,
		})
	}
}

// addArrayProperty adds a string array property as JSON.
func addArrayProperty(props *[]dbmodels.Properties, name string, value []string) {
	if len(value) > 0 {
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			glog.Warningf("Failed to marshal array property '%s': %v", name, err)
			return
		}
		jsonStr := string(jsonBytes)
		*props = append(*props, dbmodels.Properties{
			Name:        name,
			StringValue: &jsonStr,
		})
	}
}

// addJSONProperty adds a complex object property as JSON.
func addJSONProperty(props *[]dbmodels.Properties, name string, value any) {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		glog.Warningf("Failed to marshal JSON property '%s': %v", name, err)
		return
	}
	jsonStr := string(jsonBytes)
	*props = append(*props, dbmodels.Properties{
		Name:        name,
		StringValue: &jsonStr,
	})
}

// parseEpochString converts a string epoch to *int64.
// Used for SYSTEM TIMESTAMPS (CreateTimeSinceEpoch, LastUpdateTimeSinceEpoch).
func parseEpochString(epochStr *string) *int64 {
	if epochStr == nil {
		return nil
	}
	var epoch int64
	if _, err := fmt.Sscanf(*epochStr, "%d", &epoch); err == nil {
		return &epoch
	}
	glog.Warningf("Failed to parse epoch string '%s': invalid format", *epochStr)
	return nil
}

// formatEpochToString converts an int64 epoch to a numeric string.
// Used for SYSTEM TIMESTAMPS (CreateTimeSinceEpoch, LastUpdateTimeSinceEpoch).
func formatEpochToString(epoch *int64) *string {
	if epoch == nil {
		return nil
	}
	str := fmt.Sprintf("%d", *epoch)
	return &str
}

// parseISO8601ToTime parses an ISO 8601 string to time.Time.
// Used for CONTENT TIMESTAMPS (PublishedDate, LastUpdated).
// Returns nil if the string is empty or parsing fails.
func parseISO8601ToTime(timeStr string) *time.Time {
	if timeStr == "" {
		return nil
	}
	parsedTime, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		glog.Warningf("Failed to parse ISO 8601 timestamp '%s': %v", timeStr, err)
		return nil
	}
	return &parsedTime
}

// formatTimeToISO8601 formats a time.Time to an ISO 8601 string.
// Used for CONTENT TIMESTAMPS (PublishedDate, LastUpdated).
func formatTimeToISO8601(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

// convertMetadataValueToProperty converts an OpenAPI MetadataValue to a database Property.
func convertMetadataValueToProperty(name string, value openapi.MetadataValue) dbmodels.Properties {
	prop := dbmodels.Properties{Name: name}

	if value.MetadataStringValue != nil {
		prop.StringValue = &value.MetadataStringValue.StringValue
	} else if value.MetadataIntValue != nil {
		// MetadataIntValue stores as string, parse to int32
		var intVal int32
		if _, err := fmt.Sscanf(value.MetadataIntValue.IntValue, "%d", &intVal); err != nil {
			glog.Warningf("Failed to parse int metadata value '%s': %v", value.MetadataIntValue.IntValue, err)
			// Don't set prop.IntValue - leave it nil on parse failure
		} else {
			prop.IntValue = &intVal
		}
	} else if value.MetadataDoubleValue != nil {
		prop.DoubleValue = &value.MetadataDoubleValue.DoubleValue
	} else if value.MetadataBoolValue != nil {
		prop.BoolValue = &value.MetadataBoolValue.BoolValue
	} else if value.MetadataStructValue != nil {
		prop.StringValue = &value.MetadataStructValue.StructValue
	} else if value.MetadataProtoValue != nil {
		prop.StringValue = &value.MetadataProtoValue.ProtoValue
	}

	return prop
}

// convertPropertyToMetadataValue converts a database Property to an OpenAPI MetadataValue.
func convertPropertyToMetadataValue(prop dbmodels.Properties) openapi.MetadataValue {
	metadataValue := openapi.MetadataValue{}

	if prop.StringValue != nil {
		metadataValue.MetadataStringValue = openapi.NewMetadataStringValueWithDefaults()
		metadataValue.MetadataStringValue.StringValue = *prop.StringValue
	} else if prop.IntValue != nil {
		metadataValue.MetadataIntValue = openapi.NewMetadataIntValueWithDefaults()
		metadataValue.MetadataIntValue.IntValue = fmt.Sprintf("%d", *prop.IntValue)
	} else if prop.DoubleValue != nil {
		metadataValue.MetadataDoubleValue = openapi.NewMetadataDoubleValueWithDefaults()
		metadataValue.MetadataDoubleValue.DoubleValue = *prop.DoubleValue
	} else if prop.BoolValue != nil {
		metadataValue.MetadataBoolValue = openapi.NewMetadataBoolValueWithDefaults()
		metadataValue.MetadataBoolValue.BoolValue = *prop.BoolValue
	}

	return metadataValue
}
