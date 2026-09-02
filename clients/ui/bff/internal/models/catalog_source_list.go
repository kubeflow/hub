package models

type CatalogSource struct {
	Id            string   `json:"id"`
	Name          string   `json:"name"`
	Enabled       *bool    `json:"enabled,omitempty"`
	Labels        []string `json:"labels"`
	Status        *string  `json:"status,omitempty"`
	Error         *string  `json:"error,omitempty"`
	HasApiKey     *bool    `json:"hasApiKey,omitempty"`
	Authenticated *bool    `json:"authenticated,omitempty"`
}

type CatalogSourceList struct {
	NextPageToken string          `json:"nextPageToken"`
	PageSize      int32           `json:"pageSize"`
	Size          int32           `json:"size"`
	Items         []CatalogSource `json:"items,omitempty"`
}
