package service

import (
	"testing"

	"github.com/kubeflow/hub/internal/platform/db/filter"
	"github.com/stretchr/testify/assert"
)

// TODO: Add entity-specific properties from the OpenAPI spec to this map.
// See catalog/internal/catalog/modelcatalog/service/entity_mappings_test.go for reference.
var expectedSkillProperties = map[string]filter.PropertyDefinition{
	"id":                       {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "id"},
	"name":                     {Location: filter.EntityTable, ValueType: filter.StringValueType, Column: "name"},
	"externalId":               {Location: filter.EntityTable, ValueType: filter.StringValueType, Column: "external_id"},
	"createTimeSinceEpoch":     {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "create_time_since_epoch"},
	"lastUpdateTimeSinceEpoch": {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "last_update_time_since_epoch"},
}

func TestSkillEntityMappings(t *testing.T) {
	mappings := newSkillEntityMappings()
	assert.Equal(t, filter.EntityTypeContext, mappings.GetMLMDEntityType(""))

	for prop, expected := range expectedSkillProperties {
		t.Run(prop, func(t *testing.T) {
			got := mappings.GetPropertyDefinitionForRestEntity("", prop)
			assert.Equal(t, expected, got)
		})
	}

	got := mappings.GetPropertyDefinitionForRestEntity("", "unknownProp")
	assert.Equal(t, filter.Custom, got.Location)

	assert.False(t, mappings.IsChildEntity(""))
}
