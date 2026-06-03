package service

import "github.com/kubeflow/hub/internal/platform/db/filter"

type mcpServerToolEntityMappings struct{}

func NewMCPServerToolEntityMappings() filter.EntityMappingFunctions {
	return &mcpServerToolEntityMappings{}
}

func (m *mcpServerToolEntityMappings) GetMLMDEntityType(_ filter.RestEntityType) filter.EntityType {
	return filter.EntityTypeExecution
}

func (m *mcpServerToolEntityMappings) GetPropertyDefinitionForRestEntity(_ filter.RestEntityType, propertyName string) filter.PropertyDefinition {
	if def, ok := mcpServerToolProperties[propertyName]; ok {
		return def
	}
	return filter.PropertyDefinition{
		Location:  filter.Custom,
		ValueType: filter.StringValueType,
		Column:    propertyName,
	}
}

func (m *mcpServerToolEntityMappings) IsChildEntity(_ filter.RestEntityType) bool {
	return false
}

var mcpServerToolProperties = map[string]filter.PropertyDefinition{
	"id":                       {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "id"},
	"name":                     {Location: filter.EntityTable, ValueType: filter.StringValueType, Column: "name"},
	"createTimeSinceEpoch":     {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "create_time_since_epoch"},
	"lastUpdateTimeSinceEpoch": {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "last_update_time_since_epoch"},
	"description":              {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "description"},
	"accessType":               {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "accessType"},
}
