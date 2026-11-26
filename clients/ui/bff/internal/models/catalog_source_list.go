package models

type CatalogSource struct {
	// Unique catalog source identifier.
	Id string `json:"id" example:"community"`
	// Human-readable source name.
	Name string `json:"name" example:"Community Catalog"`
	// Whether this source is enabled for discovery.
	Enabled *bool `json:"enabled,omitempty" example:"true"`
	// Source labels usable with the sourceLabel query parameter.
	Labels []string `json:"labels" example:"community,validated"`
	// Loading or availability status reported by the catalog provider.
	Status *string `json:"status,omitempty"`
	// Provider error details, when loading or authentication failed.
	Error *string `json:"error,omitempty"`
	// Whether an API key is available to the provider.
	HasApiKey *bool `json:"hasApiKey,omitempty" example:"true"`
	// Authentication status reported by the provider, when available.
	Authenticated *bool `json:"authenticated,omitempty" example:"true"`
	// Hugging Face username associated with the provider's credentials.
	HfUsername string `json:"hfUsername,omitempty" example:"catalog-user"`
}

type CatalogSourceList struct {
	// Opaque token for retrieving the next page; empty on the final page.
	NextPageToken string `json:"nextPageToken"`
	// Maximum number of sources requested per page.
	PageSize int32 `json:"pageSize" example:"10"`
	// Number of sources returned in this page.
	Size int32 `json:"size" example:"1"`
	// Catalog sources returned in this page.
	Items []CatalogSource `json:"items,omitempty"`
}
