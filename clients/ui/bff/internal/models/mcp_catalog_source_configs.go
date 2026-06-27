package models

type McpCatalogSourceConfig struct {
	Id              string   `json:"id"`
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	Enabled         *bool    `json:"enabled,omitempty"`
	Labels          []string `json:"labels"`
	IncludedServers []string `json:"includedServers,omitempty"`
	ExcludedServers []string `json:"excludedServers,omitempty"`
	IsDefault       *bool    `json:"isDefault,omitempty"`
	Yaml            *string  `json:"yaml,omitempty"`
	YamlCatalogPath *string  `json:"yamlCatalogPath,omitempty"`
}

type McpCatalogSourceConfigPayload = McpCatalogSourceConfig

type McpCatalogSourceConfigList struct {
	Catalogs []McpCatalogSourceConfig `json:"catalogs,omitempty"`
}
