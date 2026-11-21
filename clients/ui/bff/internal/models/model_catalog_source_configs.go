package models

// CatalogSourceConfig configures a model catalog provider.
// @Description Model catalog source configuration, inclusion rules, and provider-specific settings.
type CatalogSourceConfig struct {
	// Unique source identifier used by catalog queries and settings routes.
	Id string `json:"id" example:"community_models"`
	// Human-readable name of the catalog source.
	Name string `json:"name" example:"Community Models"`
	// Provider type: yaml or hf.
	Type string `json:"type" example:"yaml"`
	// Whether this source is enabled for catalog discovery.
	Enabled *bool `json:"enabled,omitempty" example:"true"`
	// Labels used to group sources and filter queries with sourceLabel.
	Labels []string `json:"labels" example:"community,validated"`
	// Hugging Face API key supplied when creating or updating a source; treated as a credential.
	ApiKey *string `json:"apiKey,omitempty"`
	// Whether an effective API-key configuration exists; this does not verify the Secret or authentication.
	HasConfiguredApiKey *bool `json:"hasConfiguredApiKey,omitempty" example:"true"`
	// Hugging Face organization whose models may be discovered; required when creating an hf source.
	AllowedOrganization *string `json:"allowedOrganization,omitempty" example:"ibm-granite"`
	// Model name patterns to include from this source.
	IncludedModels []string `json:"includedModels,omitempty" example:"ibm-granite/*"`
	// Model name patterns to exclude from this source.
	ExcludedModels []string `json:"excludedModels,omitempty" example:"*/experimental-*"`
	// Whether this is a built-in source with restricted configuration changes.
	IsDefault *bool `json:"isDefault,omitempty" example:"false"`
	// Complete inline YAML catalog content for a YAML provider.
	Yaml *string `json:"yaml,omitempty" example:"models: []"`
	// Server-side path to a YAML catalog file, when file-backed configuration is used.
	YamlCatalogPath *string `json:"yamlCatalogPath,omitempty" example:"/catalog/models.yaml"`
}

type CatalogSourceConfigPayload = CatalogSourceConfig

type CatalogSourceConfigList struct {
	// Default and user-defined model source configurations.
	Catalogs []CatalogSourceConfig `json:"catalogs,omitempty"`
}
