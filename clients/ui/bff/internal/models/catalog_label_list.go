package models

// CatalogLabel represents a label used to categorize catalog sources.
// Labels can have optional display names and descriptions for UI presentation.
type CatalogLabel struct {
	// Label identifier usable with the sourceLabel query parameter.
	Name *string `json:"name" example:"validated"`
	// Human-readable label name.
	DisplayName *string `json:"displayName,omitempty" example:"Validated Sources"`
	// Explanation of what the label signifies.
	Description *string `json:"description,omitempty" example:"Sources validated for supported deployments."`
}

// CatalogLabelList represents a paginated list of catalog labels.
type CatalogLabelList struct {
	// Opaque token for retrieving the next page; empty on the final page.
	NextPageToken string `json:"nextPageToken"`
	// Maximum number of labels requested per page.
	PageSize int32 `json:"pageSize" example:"10"`
	// Number of labels returned in this page.
	Size int32 `json:"size" example:"1"`
	// Labels returned in this page.
	Items []CatalogLabel `json:"items"`
}
