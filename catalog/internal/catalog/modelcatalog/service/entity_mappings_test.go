package service

import (
	"testing"

	"github.com/kubeflow/hub/internal/platform/db/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var expectedCatalogModelProperties = map[string]filter.PropertyDefinition{
	"id":                       {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "id"},
	"name":                     {Location: filter.EntityTable, ValueType: filter.StringValueType, Column: "name"},
	"externalId":               {Location: filter.EntityTable, ValueType: filter.StringValueType, Column: "external_id"},
	"createTimeSinceEpoch":     {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "create_time_since_epoch"},
	"lastUpdateTimeSinceEpoch": {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "last_update_time_since_epoch"},
	"source_id":                {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "source_id"},
	"description":              {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "description"},
	"owner":                    {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "owner"},
	"state":                    {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "state"},
	"language":                 {Location: filter.PropertyTable, ValueType: filter.ArrayValueType, Column: "language"},
	"library_name":             {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "library_name"},
	"license_link":             {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "license_link"},
	"license":                  {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "license"},
	"logo":                     {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "logo"},
	"maturity":                 {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "maturity"},
	"provider":                 {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "provider"},
	"readme":                   {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "readme"},
	"tasks":                    {Location: filter.PropertyTable, ValueType: filter.ArrayValueType, Column: "tasks"},
	"tags":                     {Location: filter.PropertyTable, ValueType: filter.ArrayValueType, Column: "tags"},
	"verifiedSource":           {Location: filter.PropertyTable, ValueType: filter.BoolValueType, Column: "verifiedSource"},
	"validated_tasks":          {Location: filter.PropertyTable, ValueType: filter.ArrayValueType, Column: "validated_tasks"},
	"serving_config":           {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "serving_config"},
}

func TestCatalogModelEntityMappings(t *testing.T) {
	mappings := NewCatalogModelEntityMappings()

	assert.Equal(t, filter.EntityTypeContext, mappings.GetMLMDEntityType(""))

	for prop, expected := range expectedCatalogModelProperties {
		t.Run(prop, func(t *testing.T) {
			got := mappings.GetPropertyDefinitionForRestEntity("", prop)
			assert.Equal(t, expected, got)
		})
	}

	got := mappings.GetPropertyDefinitionForRestEntity("", "nonExistentProp")
	assert.Equal(t, filter.Custom, got.Location)
	assert.Equal(t, filter.StringValueType, got.ValueType)
	assert.Equal(t, "nonExistentProp", got.Column)

	assert.False(t, mappings.IsChildEntity(""))
}

func TestCatalogModelArtifactPrefix(t *testing.T) {
	mappings := NewCatalogModelEntityMappings()

	tests := []struct {
		name            string
		propertyPath    string
		expectedColumn  string
		expectedRelProp string
	}{
		{
			name:            "simple artifact property",
			propertyPath:    "artifacts.modelFormatName",
			expectedColumn:  "modelFormatName",
			expectedRelProp: "modelFormatName",
		},
		{
			name:            "nested artifact custom property",
			propertyPath:    "artifacts.customProperties.myProp",
			expectedColumn:  "customProperties.myProp",
			expectedRelProp: "customProperties.myProp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mappings.GetPropertyDefinitionForRestEntity("", tt.propertyPath)
			require.Equal(t, filter.RelatedEntity, got.Location)
			assert.Empty(t, got.ValueType, "ValueType should be empty for runtime inference")
			assert.Equal(t, tt.expectedColumn, got.Column)
			assert.Equal(t, filter.RelatedEntityArtifact, got.RelatedEntityType)
			assert.Equal(t, tt.expectedRelProp, got.RelatedProperty)
			assert.Equal(t, "Attribution", got.JoinTable)
		})
	}
}

func TestCatalogModelEqualityExpansion(t *testing.T) {
	mappings := NewCatalogModelEntityMappings()
	expander, ok := mappings.(filter.EqualityExpander)
	require.True(t, ok, "catalog model mappings must implement EqualityExpander")

	likeArg, use := expander.GetEqualityExpansion("", "externalId", "my-ext-id")
	assert.True(t, use)
	assert.Equal(t, "%:my-ext-id", likeArg)

	likeArg, use = expander.GetEqualityExpansion("", "name", "Qwen/Qwen3.5-9B")
	assert.True(t, use)
	assert.Equal(t, "%:Qwen/Qwen3.5-9B", likeArg)

	_, use = expander.GetEqualityExpansion("", "name", "hugging_face_models:Qwen/Qwen3.5-9B")
	assert.False(t, use)

	_, use = expander.GetEqualityExpansion("", "description", "foo")
	assert.False(t, use)
}
