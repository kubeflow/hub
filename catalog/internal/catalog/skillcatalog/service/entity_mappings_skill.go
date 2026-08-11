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

// skillProperties maps skill property names to their filterQuery locations.
// "name" points at the property table's skill-name column (the SKILL.md
// name), matching what the dedicated `name` list parameter filters — not the
// Context entity table's "name" column, which holds the internal
// (source, repository, path, version) composite identity key and has no
// filterQuery-facing meaning. "source_id" likewise mirrors the dedicated
// `source` list parameter, matching the mcpcatalog/modelcatalog convention.
//
// configDigest is deliberately absent: it's an internal sync-cache digest
// with no filterQuery-facing meaning, same as the entity table's "name".
// Any property not listed here falls through to filter.Custom in
// GetPropertyDefinitionForRestEntity, which queries the wrong table for a
// skill property and should be treated as "not filterable" rather than used.
var skillProperties = map[string]filter.PropertyDefinition{
	"id":                       {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "id"},
	"externalId":               {Location: filter.EntityTable, ValueType: filter.StringValueType, Column: "external_id"},
	"createTimeSinceEpoch":     {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "create_time_since_epoch"},
	"lastUpdateTimeSinceEpoch": {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "last_update_time_since_epoch"},

	"name":            {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "name"},
	"source_id":       {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "source_id"},
	"description":     {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "description"},
	"repository":      {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "repository"},
	"path":            {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "path"},
	"version":         {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "version"},
	"resolvedCommit":  {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "resolvedCommit"},
	"trustTier":       {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "trustTier"},
	"provider":        {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "provider"},
	"category":        {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "category"},
	"license":         {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "license"},
	"author":          {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "author"},
	"compatibility":   {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "compatibility"},
	"readme":          {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "readme"},
	"bodyLineCount":   {Location: filter.PropertyTable, ValueType: filter.IntValueType, Column: "bodyLineCount"},
	"labels":          {Location: filter.PropertyTable, ValueType: filter.ArrayValueType, Column: "labels"},
	"allowedTools":    {Location: filter.PropertyTable, ValueType: filter.ArrayValueType, Column: "allowedTools"},
	"supportingFiles": {Location: filter.PropertyTable, ValueType: filter.ArrayValueType, Column: "supportingFiles"},
}
