package service

import "github.com/kubeflow/hub/internal/platform/db/filter"

type catalogArtifactEntityMappings struct{}

func NewCatalogArtifactEntityMappings() filter.EntityMappingFunctions {
	return &catalogArtifactEntityMappings{}
}

func (m *catalogArtifactEntityMappings) GetMLMDEntityType(_ filter.RestEntityType) filter.EntityType {
	return filter.EntityTypeArtifact
}

func (m *catalogArtifactEntityMappings) GetPropertyDefinitionForRestEntity(_ filter.RestEntityType, propertyName string) filter.PropertyDefinition {
	if def, ok := catalogArtifactProperties[propertyName]; ok {
		return def
	}
	return filter.PropertyDefinition{
		Location:  filter.Custom,
		ValueType: filter.StringValueType,
		Column:    propertyName,
	}
}

func (m *catalogArtifactEntityMappings) IsChildEntity(_ filter.RestEntityType) bool {
	return false
}

var catalogArtifactProperties = map[string]filter.PropertyDefinition{
	"id":                       {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "id"},
	"name":                     {Location: filter.EntityTable, ValueType: filter.StringValueType, Column: "name"},
	"externalId":               {Location: filter.EntityTable, ValueType: filter.StringValueType, Column: "external_id"},
	"createTimeSinceEpoch":     {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "create_time_since_epoch"},
	"lastUpdateTimeSinceEpoch": {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "last_update_time_since_epoch"},
	"uri":                      {Location: filter.EntityTable, ValueType: filter.StringValueType, Column: "uri"},
	"state":                    {Location: filter.EntityTable, ValueType: filter.StringValueType, Column: "state"},
	"artifactType":             {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "artifactType"},
}
