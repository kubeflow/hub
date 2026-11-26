package models

type CatalogSourcePreviewRequest struct {
	// Existing source identifier, used when resolving stored source credentials.
	Id string `json:"id,omitempty" example:"community"`
	// Catalog provider type to preview.
	Type string `json:"type" validate:"required" example:"yaml"`
	// Model name patterns to include when previewing model assets.
	IncludedModels []string `json:"includedModels,omitempty" example:"ibm-granite/*"`
	// Model name patterns to exclude when previewing model assets.
	ExcludedModels []string `json:"excludedModels,omitempty" example:"*/experimental-*"`
	// Server name patterns to include when previewing MCP server assets.
	IncludedServers []string `json:"includedServers,omitempty" example:"search-*"`
	// Server name patterns to exclude when previewing MCP server assets.
	ExcludedServers []string `json:"excludedServers,omitempty" example:"experimental-*"`
	// Provider settings, such as yaml, yamlCatalogPath, apiKey, and allowedOrganization.
	Properties map[string]interface{} `json:"properties,omitempty"`
}

type CatalogSourcePreviewModel struct {
	// Model or MCP server name evaluated by the preview.
	Name string `json:"name" example:"ibm-granite/granite-3.1-8b-base"`
	// Whether the asset matches the source's inclusion and exclusion rules.
	Included bool `json:"included" example:"true"`
	// Hugging Face access classification, when provided by that provider.
	HfAccessType *string `json:"hfAccessType,omitempty"`
	// Whether the configured credentials grant access to a gated Hugging Face model.
	HfGatedAccessGranted *bool `json:"hfGatedAccessGranted,omitempty" example:"true"`
}

type CatalogSourcePreviewSummary struct {
	// Total number of model assets evaluated across all preview pages.
	TotalModels int32 `json:"totalModels,omitempty"`
	// Number of models matching the source's selection rules across all pages.
	IncludedModels int32 `json:"includedModels,omitempty"`
	// Number of models excluded by the source's selection rules across all pages.
	ExcludedModels int32 `json:"excludedModels,omitempty"`
	// Whether any gated Hugging Face model is inaccessible, across all pages.
	HasGatedAccessDeniedModels bool `json:"hasGatedAccessDeniedModels"`
	// Total number of evaluated assets, for asset-oriented catalog previews.
	TotalAssets int32 `json:"totalAssets,omitempty"`
	// Number of included assets across all preview pages.
	IncludedAssets int32 `json:"includedAssets,omitempty"`
	// Number of excluded assets across all preview pages.
	ExcludedAssets int32 `json:"excludedAssets,omitempty"`
}

type CatalogSourcePreviewResult struct {
	// Evaluated assets in the current preview page.
	Items []CatalogSourcePreviewModel `json:"items"`
	// Source-wide counts and access status across all preview pages.
	Summary CatalogSourcePreviewSummary `json:"summary"`
	// Opaque token for retrieving the next preview page; empty on the final page.
	NextPageToken string `json:"nextPageToken"`
	// Maximum number of assets requested per page.
	PageSize int32 `json:"pageSize" example:"10"`
	// Number of assets returned in this page.
	Size int32 `json:"size" example:"1"`
}
