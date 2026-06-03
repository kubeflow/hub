package service

import (
	"strings"

	"github.com/kubeflow/hub/internal/platform/db/filter"
)

type catalogModelEntityMappings struct{}

func NewCatalogModelEntityMappings() filter.EntityMappingFunctions {
	return &catalogModelEntityMappings{}
}

func (m *catalogModelEntityMappings) GetMLMDEntityType(_ filter.RestEntityType) filter.EntityType {
	return filter.EntityTypeContext
}

func (m *catalogModelEntityMappings) GetPropertyDefinitionForRestEntity(_ filter.RestEntityType, propertyName string) filter.PropertyDefinition {
	if def, ok := catalogModelProperties[propertyName]; ok {
		return def
	}

	if artifactPropertyPath, found := strings.CutPrefix(propertyName, "artifacts."); found {
		return filter.PropertyDefinition{
			Location:          filter.RelatedEntity,
			ValueType:         "",
			Column:            artifactPropertyPath,
			RelatedEntityType: filter.RelatedEntityArtifact,
			RelatedProperty:   artifactPropertyPath,
			JoinTable:         "Attribution",
		}
	}

	return filter.PropertyDefinition{
		Location:  filter.Custom,
		ValueType: filter.StringValueType,
		Column:    propertyName,
	}
}

func (m *catalogModelEntityMappings) IsChildEntity(_ filter.RestEntityType) bool {
	return false
}

func (m *catalogModelEntityMappings) GetEqualityExpansion(_ filter.RestEntityType, propertyName string, value any) (likeArg any, useExpansion bool) {
	strVal, ok := value.(string)
	if !ok || strVal == "" {
		return nil, false
	}
	switch propertyName {
	case "externalId":
		return "%:" + escapeLike(strVal), true
	case "name":
		if strings.Contains(strVal, ":") {
			return nil, false
		}
		return "%:" + escapeLike(strVal), true
	default:
		return nil, false
	}
}

var catalogModelProperties = map[string]filter.PropertyDefinition{
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
