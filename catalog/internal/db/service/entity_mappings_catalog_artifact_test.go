package service

import (
	"testing"

	modelcatalogmodels "github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	catalogmodels "github.com/kubeflow/hub/catalog/internal/db/models"
	"github.com/kubeflow/hub/internal/platform/db/filter"
	"github.com/stretchr/testify/assert"
)

var expectedCatalogArtifactProperties = map[string]filter.PropertyDefinition{
	"id":                       {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "id"},
	"name":                     {Location: filter.EntityTable, ValueType: filter.StringValueType, Column: "name"},
	"externalId":               {Location: filter.EntityTable, ValueType: filter.StringValueType, Column: "external_id"},
	"createTimeSinceEpoch":     {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "create_time_since_epoch"},
	"lastUpdateTimeSinceEpoch": {Location: filter.EntityTable, ValueType: filter.IntValueType, Column: "last_update_time_since_epoch"},
	"uri":                      {Location: filter.EntityTable, ValueType: filter.StringValueType, Column: "uri"},
	"state":                    {Location: filter.EntityTable, ValueType: filter.StringValueType, Column: "state"},
	"artifactType":             {Location: filter.PropertyTable, ValueType: filter.StringValueType, Column: "artifactType"},
}

func TestCatalogArtifactEntityMappings(t *testing.T) {
	mappings := NewCatalogArtifactEntityMappings()

	assert.Equal(t, filter.EntityTypeArtifact, mappings.GetMLMDEntityType(""))

	for prop, expected := range expectedCatalogArtifactProperties {
		t.Run(prop, func(t *testing.T) {
			got := mappings.GetPropertyDefinitionForRestEntity("", prop)
			assert.Equal(t, expected, got)
		})
	}

	got := mappings.GetPropertyDefinitionForRestEntity("", "unknownProp")
	assert.Equal(t, filter.Custom, got.Location)

	assert.False(t, mappings.IsChildEntity(""))
}

func TestArtifactListOptionsMapToCatalogArtifactEntity(t *testing.T) {
	mappings := NewCatalogArtifactEntityMappings()

	tests := []struct {
		name       string
		entityType filter.RestEntityType
	}{
		{
			name:       "model artifact list options",
			entityType: (&modelcatalogmodels.CatalogModelArtifactListOptions{}).GetRestEntityType(),
		},
		{
			name:       "metrics artifact list options",
			entityType: (&modelcatalogmodels.CatalogMetricsArtifactListOptions{}).GetRestEntityType(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, filter.RestEntityType(catalogmodels.RestEntityCatalogArtifact), tt.entityType)
			assert.Equal(t, filter.EntityTypeArtifact, mappings.GetMLMDEntityType(tt.entityType))
		})
	}
}
