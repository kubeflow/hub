package service

import "github.com/kubeflow/hub/internal/platform/db/filter"

type skillEntityMappings struct{}

// Unexported: only used by the repository constructor in this package.
// Export (New...) if tests in a parent package need to call it directly.
func newSkillEntityMappings() filter.EntityMappingFunctions {
	return &skillEntityMappings{}
}

func (m *skillEntityMappings) GetMLMDEntityType(_ filter.RestEntityType) filter.EntityType {
	return filter.EntityTypeContext
}

func (m *skillEntityMappings) GetPropertyDefinitionForRestEntity(_ filter.RestEntityType, propertyName string) filter.PropertyDefinition {
	if def, ok := skillProperties[propertyName]; ok {
		return def
	}
	return filter.PropertyDefinition{
		Location:  filter.Custom,
		ValueType: filter.StringValueType,
		Column:    propertyName,
	}
}

func (m *skillEntityMappings) IsChildEntity(_ filter.RestEntityType) bool {
	return false
}

var skillProperties = map[string]filter.PropertyDefinition{
	"id":                       {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "id"},
	"name":                     {Location: filter.EntityTable, ValueType: filter.StringValueType, Column: "name"},
	"externalId":               {Location: filter.EntityTable, ValueType: filter.StringValueType, Column: "external_id"},
	"createTimeSinceEpoch":     {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "create_time_since_epoch"},
	"lastUpdateTimeSinceEpoch": {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "last_update_time_since_epoch"},
}
