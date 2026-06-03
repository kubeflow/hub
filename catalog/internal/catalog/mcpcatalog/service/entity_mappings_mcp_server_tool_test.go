package service

import (
	"testing"

	"github.com/kubeflow/hub/internal/platform/db/filter"
	"github.com/stretchr/testify/assert"
)

var expectedMCPServerToolProperties = map[string]filter.PropertyDefinition{
	"id":                       {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "id"},
	"name":                     {Location: filter.EntityTable, ValueType: filter.StringValueType, Column: "name"},
	"createTimeSinceEpoch":     {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "create_time_since_epoch"},
	"lastUpdateTimeSinceEpoch": {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "last_update_time_since_epoch"},
	"description":              {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "description"},
	"accessType":               {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "accessType"},
}

func TestMCPServerToolEntityMappings(t *testing.T) {
	mappings := NewMCPServerToolEntityMappings()

	assert.Equal(t, filter.EntityTypeExecution, mappings.GetMLMDEntityType(""))

	for prop, expected := range expectedMCPServerToolProperties {
		t.Run(prop, func(t *testing.T) {
			got := mappings.GetPropertyDefinitionForRestEntity("", prop)
			assert.Equal(t, expected, got)
		})
	}

	got := mappings.GetPropertyDefinitionForRestEntity("", "unknownProp")
	assert.Equal(t, filter.Custom, got.Location)

	assert.False(t, mappings.IsChildEntity(""))
}
