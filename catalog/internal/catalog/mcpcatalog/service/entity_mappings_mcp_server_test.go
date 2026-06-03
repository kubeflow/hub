package service

import (
	"testing"

	"github.com/kubeflow/hub/internal/platform/db/filter"
	"github.com/stretchr/testify/assert"
)

var expectedMCPServerProperties = map[string]filter.PropertyDefinition{
	"id":                       {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "id"},
	"name":                     {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "base_name"},
	"externalId":               {Location: filter.EntityTable, ValueType: filter.StringValueType, Column: "external_id"},
	"createTimeSinceEpoch":     {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "create_time_since_epoch"},
	"lastUpdateTimeSinceEpoch": {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "last_update_time_since_epoch"},
	"source_id":                {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "source_id"},
	"base_name":                {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "base_name"},
	"description":              {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "description"},
	"provider":                 {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "provider"},
	"license":                  {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "license"},
	"license_link":             {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "license_link"},
	"logo":                     {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "logo"},
	"readme":                   {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "readme"},
	"version":                  {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "version"},
	"tags":                     {Location: filter.PropertyTable, ValueType: filter.ArrayValueType, Column: "tags"},
	"transports":               {Location: filter.PropertyTable, ValueType: filter.ArrayValueType, Column: "transports"},
	"deploymentMode":           {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "deploymentMode"},
	"documentationUrl":         {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "documentationUrl"},
	"repositoryUrl":            {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "repositoryUrl"},
	"sourceCode":               {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "sourceCode"},
	"publishedDate":            {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "publishedDate"},
	"lastUpdated":              {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "lastUpdated"},
	"verifiedSource":           {Location: filter.PropertyTable, ValueType: filter.BoolValueType, Column: "verifiedSource"},
	"secureEndpoint":           {Location: filter.PropertyTable, ValueType: filter.BoolValueType, Column: "secureEndpoint"},
	"sast":                     {Location: filter.PropertyTable, ValueType: filter.BoolValueType, Column: "sast"},
	"readOnlyTools":            {Location: filter.PropertyTable, ValueType: filter.BoolValueType, Column: "readOnlyTools"},
	"endpoints":                {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "endpoints"},
	"artifacts":                {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "artifacts"},
	"runtimeMetadata":          {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "runtimeMetadata"},
}

func TestMCPServerEntityMappings(t *testing.T) {
	mappings := NewMCPServerEntityMappings()

	assert.Equal(t, filter.EntityTypeContext, mappings.GetMLMDEntityType(""))

	for prop, expected := range expectedMCPServerProperties {
		t.Run(prop, func(t *testing.T) {
			got := mappings.GetPropertyDefinitionForRestEntity("", prop)
			assert.Equal(t, expected, got)
		})
	}

	got := mappings.GetPropertyDefinitionForRestEntity("", "unknownProp")
	assert.Equal(t, filter.Custom, got.Location)

	assert.False(t, mappings.IsChildEntity(""))
}
