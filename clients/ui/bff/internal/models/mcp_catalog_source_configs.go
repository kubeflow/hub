package models

// McpCatalogSourceConfig configures an MCP catalog provider.
// @Description MCP source configuration, YAML catalog content, and server inclusion rules.
type McpCatalogSourceConfig struct {
	// Unique source identifier used by catalog queries and settings routes.
	Id string `json:"id" example:"community_mcp"`
	// Human-readable name of the catalog source.
	Name string `json:"name" example:"Community MCP Servers"`
	// Provider type used to load this source.
	Type string `json:"type" example:"yaml"`
	// Whether this source is enabled for catalog discovery.
	Enabled *bool `json:"enabled,omitempty" example:"true"`
	// Labels used to group sources and filter queries with sourceLabel.
	Labels []string `json:"labels" example:"community,validated"`
	// Server name patterns to include from this source.
	IncludedServers []string `json:"includedServers,omitempty" example:"search-*"`
	// Server name patterns to exclude from this source.
	ExcludedServers []string `json:"excludedServers,omitempty" example:"experimental-*"`
	// Whether this is a built-in source with restricted configuration changes.
	IsDefault *bool `json:"isDefault,omitempty" example:"false"`
	// Complete inline YAML catalog content.
	Yaml *string `json:"yaml,omitempty" example:"mcp_servers: []"`
	// Server-side path to a YAML catalog file, when file-backed configuration is used.
	YamlCatalogPath *string `json:"yamlCatalogPath,omitempty" example:"/catalog/mcp-servers.yaml"`
}

type McpCatalogSourceConfigPayload = McpCatalogSourceConfig

type McpCatalogSourceConfigList struct {
	// Default and user-defined MCP source configurations.
	Catalogs []McpCatalogSourceConfig `json:"catalogs,omitempty"`
}
