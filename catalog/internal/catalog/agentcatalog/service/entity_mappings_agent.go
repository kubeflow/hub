package service

import "github.com/kubeflow/hub/internal/platform/db/filter"

type agentEntityMappings struct{}

// Unexported: only used by the repository constructor in this package.
// Export (New...) if tests in a parent package need to call it directly.
func newAgentEntityMappings() filter.EntityMappingFunctions {
	return &agentEntityMappings{}
}

func (m *agentEntityMappings) GetMLMDEntityType(_ filter.RestEntityType) filter.EntityType {
	return filter.EntityTypeContext
}

func (m *agentEntityMappings) GetPropertyDefinitionForRestEntity(_ filter.RestEntityType, propertyName string) filter.PropertyDefinition {
	if def, ok := agentProperties[propertyName]; ok {
		return def
	}
	return filter.PropertyDefinition{
		Location:  filter.Custom,
		ValueType: filter.StringValueType,
		Column:    propertyName,
	}
}

func (m *agentEntityMappings) IsChildEntity(_ filter.RestEntityType) bool {
	return false
}

var agentProperties = map[string]filter.PropertyDefinition{
	"id":                       {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "id"},
	"name":                     {Location: filter.EntityTable, ValueType: filter.StringValueType, Column: "name"},
	"externalId":               {Location: filter.EntityTable, ValueType: filter.StringValueType, Column: "external_id"},
	"createTimeSinceEpoch":     {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "create_time_since_epoch"},
	"lastUpdateTimeSinceEpoch": {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "last_update_time_since_epoch"},
	"displayName":              {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "displayName"},
	"description":              {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "description"},
	"framework":                {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "framework"},
	"repositoryUrl":            {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "repositoryUrl"},
	"labels":                   {Location: filter.PropertyTable, ValueType: filter.ArrayValueType, Column: "labels"},
}
