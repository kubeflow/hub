package openapi

type Sortable interface {
	// SortValue returns the value of a requested field converted to a string.
	SortValue(field OrderByField) string
}

func (s CatalogSource) SortValue(field OrderByField) string {
	switch field {
	case ORDERBYFIELD_ID:
		return s.Id
	case ORDERBYFIELD_NAME:
		return s.Name
	}
	return ""
}

func (m CatalogModel) SortValue(field OrderByField) string {
	switch field {
	case ORDERBYFIELD_ID:
		return m.Name // Name is ID for models
	case ORDERBYFIELD_NAME:
		return m.Name
	case ORDERBYFIELD_LAST_UPDATE_TIME:
		return unrefString(m.LastUpdateTimeSinceEpoch)
	case ORDERBYFIELD_CREATE_TIME:
		return unrefString(m.CreateTimeSinceEpoch)
	}
	return ""
}

func unrefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func (m ModelPreviewResult) SortValue(field OrderByField) string {
	switch field {
	case ORDERBYFIELD_ID, ORDERBYFIELD_NAME:
		return m.Name
	}
	return ""
}

func (a AssetPreviewResult) SortValue(field OrderByField) string {
	switch field {
	case ORDERBYFIELD_ID, ORDERBYFIELD_NAME:
		return a.Name
	}
	return ""
}
