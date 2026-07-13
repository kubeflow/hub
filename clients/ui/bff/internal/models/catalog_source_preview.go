package models

type CatalogSourcePreviewRequest struct {
	Type            string                 `json:"type"`
	IncludedModels  []string               `json:"includedModels,omitempty"`
	ExcludedModels  []string               `json:"excludedModels,omitempty"`
	IncludedServers []string               `json:"includedServers,omitempty"`
	ExcludedServers []string               `json:"excludedServers,omitempty"`
	Properties      map[string]interface{} `json:"properties,omitempty"`
}

type CatalogSourcePreviewModel struct {
	Name     string `json:"name"`
	Included bool   `json:"included"`
}

type CatalogSourcePreviewSummary struct {
	TotalModels    int32 `json:"totalModels,omitempty"`
	IncludedModels int32 `json:"includedModels,omitempty"`
	ExcludedModels int32 `json:"excludedModels,omitempty"`
	TotalAssets    int32 `json:"totalAssets,omitempty"`
	IncludedAssets int32 `json:"includedAssets,omitempty"`
	ExcludedAssets int32 `json:"excludedAssets,omitempty"`
}

type CatalogSourcePreviewResult struct {
	Items         []CatalogSourcePreviewModel `json:"items"`
	Summary       CatalogSourcePreviewSummary `json:"summary"`
	NextPageToken string                      `json:"nextPageToken"`
	PageSize      int32                       `json:"pageSize"`
	Size          int32                       `json:"size"`
}
