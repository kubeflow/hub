package modelcatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	catalogmodels "github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	model "github.com/kubeflow/hub/catalog/pkg/openapi"
	models "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYamlModelToModelProviderRecord(t *testing.T) {
	tests := []struct {
		name         string
		yamlModel    yamlModel
		expectError  bool
		validateFunc func(t *testing.T, record ModelProviderRecord)
	}{
		{
			name: "complete model with all properties",
			yamlModel: yamlModel{
				CatalogModel: model.CatalogModel{
					Name:           "test-model",
					Description:    new("Test model description"),
					Readme:         new("# Test Model\nThis is a test model."),
					Maturity:       new("Generally Available"),
					Language:       []string{"en", "fr"},
					Tasks:          []string{"text-generation", "nlp"},
					ValidatedTasks: []string{"text-generation", "tool-calling"},
					Provider:       new("IBM"),
					Logo:           new("https://example.com/logo.png"),
					License:        new("apache-2.0"),
					LicenseLink:    new("https://www.apache.org/licenses/LICENSE-2.0"),
					LibraryName:    new("transformers"),
					ServingConfig: &model.ServingConfig{
						ToolCalling: &model.ToolCallingConfig{
							ToolCallParser:       new("granite"),
							ChatTemplate:         new("opt/app-root/template/tool_chat_template_granite.jinja"),
							EnableAutoToolChoice: new(true),
							RequiredArgs:         []string{"--config_format granite"},
						},
					},
					SourceId:                 new("test-source"),
					CreateTimeSinceEpoch:     new("1678886400000"),
					LastUpdateTimeSinceEpoch: new("1681564800000"),
					CustomProperties: map[string]model.MetadataValue{
						"custom_key": {
							MetadataStringValue: &model.MetadataStringValue{
								StringValue:  "custom_value",
								MetadataType: "MetadataStringValue",
							},
						},
					},
				},
				Artifacts: []*yamlArtifact{
					{
						CatalogArtifact: model.CatalogArtifact{
							CatalogModelArtifact: &model.CatalogModelArtifact{
								ArtifactType:             "model-artifact",
								Uri:                      "https://example.com/model.tar.gz",
								CreateTimeSinceEpoch:     new("1678886400000"),
								LastUpdateTimeSinceEpoch: new("1681564800000"),
								CustomProperties: map[string]model.MetadataValue{
									"model_size": {
										MetadataStringValue: &model.MetadataStringValue{
											StringValue:  "2GB",
											MetadataType: "MetadataStringValue",
										},
									},
									"accuracy": {
										MetadataDoubleValue: &model.MetadataDoubleValue{
											DoubleValue:  0.95,
											MetadataType: "MetadataDoubleValue",
										},
									},
								},
							},
						},
					},
					{
						CatalogArtifact: model.CatalogArtifact{
							CatalogMetricsArtifact: &model.CatalogMetricsArtifact{
								ArtifactType:             "metrics-artifact",
								MetricsType:              "evaluation-metrics",
								CreateTimeSinceEpoch:     new("1678886400000"),
								LastUpdateTimeSinceEpoch: new("1681564800000"),
								CustomProperties: map[string]model.MetadataValue{
									"framework": {
										MetadataStringValue: &model.MetadataStringValue{
											StringValue:  "scikit-learn",
											MetadataType: "MetadataStringValue",
										},
									},
								},
							},
						},
					},
				},
			},
			validateFunc: func(t *testing.T, record ModelProviderRecord) {
				require.NotNil(t, record.Model)

				attrs := record.Model.GetAttributes()
				require.NotNil(t, attrs)
				assert.Equal(t, int64(1678886400000), *attrs.CreateTimeSinceEpoch)
				assert.Equal(t, int64(1681564800000), *attrs.LastUpdateTimeSinceEpoch)

				// Check regular properties (spec-defined properties)
				regularProps := record.Model.GetProperties()
				require.NotNil(t, regularProps)

				regularPropMap := make(map[string]models.Properties)
				for _, prop := range *regularProps {
					regularPropMap[prop.Name] = prop
				}

				// Validate spec-defined properties are regular properties
				assert.Contains(t, regularPropMap, "description")
				assert.Equal(t, "Test model description", *regularPropMap["description"].StringValue)
				assert.False(t, regularPropMap["description"].IsCustomProperty)

				assert.Contains(t, regularPropMap, "readme")
				assert.Equal(t, "# Test Model\nThis is a test model.", *regularPropMap["readme"].StringValue)
				assert.False(t, regularPropMap["readme"].IsCustomProperty)

				assert.Contains(t, regularPropMap, "maturity")
				assert.Equal(t, "Generally Available", *regularPropMap["maturity"].StringValue)
				assert.False(t, regularPropMap["maturity"].IsCustomProperty)

				assert.Contains(t, regularPropMap, "provider")
				assert.Equal(t, "IBM", *regularPropMap["provider"].StringValue)
				assert.False(t, regularPropMap["provider"].IsCustomProperty)

				assert.Contains(t, regularPropMap, "logo")
				assert.Equal(t, "https://example.com/logo.png", *regularPropMap["logo"].StringValue)
				assert.False(t, regularPropMap["logo"].IsCustomProperty)

				assert.Contains(t, regularPropMap, "license")
				assert.Equal(t, "Apache 2.0", *regularPropMap["license"].StringValue)
				assert.False(t, regularPropMap["license"].IsCustomProperty)

				assert.Contains(t, regularPropMap, "license_link")
				assert.Equal(t, "https://www.apache.org/licenses/LICENSE-2.0", *regularPropMap["license_link"].StringValue)
				assert.False(t, regularPropMap["license_link"].IsCustomProperty)

				assert.Contains(t, regularPropMap, "library_name")
				assert.Equal(t, "transformers", *regularPropMap["library_name"].StringValue)
				assert.False(t, regularPropMap["library_name"].IsCustomProperty)

				assert.Contains(t, regularPropMap, "source_id")
				assert.Equal(t, "test-source", *regularPropMap["source_id"].StringValue)
				assert.False(t, regularPropMap["source_id"].IsCustomProperty)

				// Validate array properties are JSON encoded as regular properties
				assert.Contains(t, regularPropMap, "language")
				var languages []string
				err := json.Unmarshal([]byte(*regularPropMap["language"].StringValue), &languages)
				require.NoError(t, err)
				assert.Equal(t, []string{"en", "fr"}, languages)
				assert.False(t, regularPropMap["language"].IsCustomProperty)

				assert.Contains(t, regularPropMap, "tasks")
				var tasks []string
				err = json.Unmarshal([]byte(*regularPropMap["tasks"].StringValue), &tasks)
				require.NoError(t, err)
				assert.Equal(t, []string{"text-generation", "nlp"}, tasks)
				assert.False(t, regularPropMap["tasks"].IsCustomProperty)

				assert.Contains(t, regularPropMap, "validated_tasks")
				var validatedTasks []string
				err = json.Unmarshal([]byte(*regularPropMap["validated_tasks"].StringValue), &validatedTasks)
				require.NoError(t, err)
				assert.Equal(t, []string{"text-generation", "tool-calling"}, validatedTasks)
				assert.False(t, regularPropMap["validated_tasks"].IsCustomProperty)

				assert.Contains(t, regularPropMap, "serving_config")
				var servingConfig model.ServingConfig
				err = json.Unmarshal([]byte(*regularPropMap["serving_config"].StringValue), &servingConfig)
				require.NoError(t, err)
				require.NotNil(t, servingConfig.ToolCalling)
				assert.Equal(t, new("granite"), servingConfig.ToolCalling.ToolCallParser)
				assert.Equal(t, new("opt/app-root/template/tool_chat_template_granite.jinja"), servingConfig.ToolCalling.ChatTemplate)
				assert.Equal(t, new(true), servingConfig.ToolCalling.EnableAutoToolChoice)
				assert.Equal(t, []string{"--config_format granite"}, servingConfig.ToolCalling.RequiredArgs)
				assert.False(t, regularPropMap["serving_config"].IsCustomProperty)

				// Check custom properties
				customProps := record.Model.GetCustomProperties()
				require.NotNil(t, customProps)

				customPropMap := make(map[string]models.Properties)
				for _, prop := range *customProps {
					customPropMap[prop.Name] = prop
				}

				// Validate truly custom properties
				assert.Contains(t, customPropMap, "custom_key")
				assert.Equal(t, "custom_value", *customPropMap["custom_key"].StringValue)
				assert.True(t, customPropMap["custom_key"].IsCustomProperty)

				// Validate artifacts
				assert.Len(t, record.Artifacts, 2)

				// Validate ModelArtifact
				modelArtifact := record.Artifacts[0]
				require.NotNil(t, modelArtifact.CatalogModelArtifact)
				assert.Nil(t, modelArtifact.CatalogMetricsArtifact)

				// Check CatalogModelArtifact attributes
				modelAttrs := modelArtifact.CatalogModelArtifact.(catalogmodels.CatalogModelArtifact).GetAttributes()
				assert.Equal(t, "https://example.com/model.tar.gz", *modelAttrs.URI)
				assert.Equal(t, int64(1678886400000), *modelAttrs.CreateTimeSinceEpoch)
				assert.Equal(t, int64(1681564800000), *modelAttrs.LastUpdateTimeSinceEpoch)

				// Check CatalogModelArtifact regular properties
				modelArtifactProps := modelArtifact.CatalogModelArtifact.GetProperties()
				require.NotNil(t, modelArtifactProps)
				modelArtifactPropMap := make(map[string]models.Properties)
				for _, prop := range *modelArtifactProps {
					modelArtifactPropMap[prop.Name] = prop
				}
				assert.Contains(t, modelArtifactPropMap, "uri")
				assert.Equal(t, "https://example.com/model.tar.gz", *modelArtifactPropMap["uri"].StringValue)
				assert.False(t, modelArtifactPropMap["uri"].IsCustomProperty)

				// Check CatalogModelArtifact custom properties
				modelArtifactCustomProps := modelArtifact.CatalogModelArtifact.GetCustomProperties()
				require.NotNil(t, modelArtifactCustomProps)
				modelArtifactCustomPropMap := make(map[string]models.Properties)
				for _, prop := range *modelArtifactCustomProps {
					modelArtifactCustomPropMap[prop.Name] = prop
				}
				assert.Contains(t, modelArtifactCustomPropMap, "model_size")
				assert.Equal(t, "2GB", *modelArtifactCustomPropMap["model_size"].StringValue)
				assert.True(t, modelArtifactCustomPropMap["model_size"].IsCustomProperty)
				assert.Contains(t, modelArtifactCustomPropMap, "accuracy")
				assert.Equal(t, 0.95, *modelArtifactCustomPropMap["accuracy"].DoubleValue)
				assert.True(t, modelArtifactCustomPropMap["accuracy"].IsCustomProperty)

				// Validate CatalogMetricsArtifact
				metricsArtifact := record.Artifacts[1]
				require.NotNil(t, metricsArtifact.CatalogMetricsArtifact)
				assert.Nil(t, metricsArtifact.CatalogModelArtifact)

				// Check CatalogMetricsArtifact attributes
				metricsAttrs := metricsArtifact.CatalogMetricsArtifact.(catalogmodels.CatalogMetricsArtifact).GetAttributes()
				assert.Equal(t, "evaluation-metrics", string(metricsAttrs.MetricsType))
				assert.Equal(t, int64(1678886400000), *metricsAttrs.CreateTimeSinceEpoch)
				assert.Equal(t, int64(1681564800000), *metricsAttrs.LastUpdateTimeSinceEpoch)

				// Check CatalogMetricsArtifact regular properties
				metricsArtifactProps := metricsArtifact.CatalogMetricsArtifact.GetProperties()
				require.NotNil(t, metricsArtifactProps)
				metricsArtifactPropMap := make(map[string]models.Properties)
				for _, prop := range *metricsArtifactProps {
					metricsArtifactPropMap[prop.Name] = prop
				}
				assert.Contains(t, metricsArtifactPropMap, "metricsType")
				assert.Equal(t, "evaluation-metrics", *metricsArtifactPropMap["metricsType"].StringValue)
				assert.False(t, metricsArtifactPropMap["metricsType"].IsCustomProperty)

				// Check CatalogMetricsArtifact custom properties
				metricsArtifactCustomProps := metricsArtifact.CatalogMetricsArtifact.GetCustomProperties()
				require.NotNil(t, metricsArtifactCustomProps)
				metricsArtifactCustomPropMap := make(map[string]models.Properties)
				for _, prop := range *metricsArtifactCustomProps {
					metricsArtifactCustomPropMap[prop.Name] = prop
				}
				assert.Contains(t, metricsArtifactCustomPropMap, "framework")
				assert.Equal(t, "scikit-learn", *metricsArtifactCustomPropMap["framework"].StringValue)
				assert.True(t, metricsArtifactCustomPropMap["framework"].IsCustomProperty)
			},
		},
		{
			name: "minimal model with only required fields",
			yamlModel: yamlModel{
				CatalogModel: model.CatalogModel{
					Name: "minimal-model",
				},
			},
			validateFunc: func(t *testing.T, record ModelProviderRecord) {
				require.NotNil(t, record.Model)

				attrs := record.Model.GetAttributes()
				require.NotNil(t, attrs)
				assert.Nil(t, attrs.CreateTimeSinceEpoch)
				assert.Nil(t, attrs.LastUpdateTimeSinceEpoch)

				// Should have no regular properties for minimal model
				regularProps := record.Model.GetProperties()
				if regularProps != nil {
					*regularProps = slices.DeleteFunc(*regularProps, func(p models.Properties) bool {
						switch p.Name {
						case "language", "tasks", "validated_tasks":
							return true
						}
						return false
					})
					assert.Empty(t, *regularProps)
				}

				// Should have no custom properties for minimal model
				customProps := record.Model.GetCustomProperties()
				if customProps != nil {
					assert.Empty(t, *customProps)
				}

				// Should have no artifacts for minimal model
				assert.Empty(t, record.Artifacts)
			},
		},
		{
			name: "model with only ModelArtifact",
			yamlModel: yamlModel{
				CatalogModel: model.CatalogModel{
					Name: "model-with-artifact",
				},
				Artifacts: []*yamlArtifact{
					{
						CatalogArtifact: model.CatalogArtifact{
							CatalogModelArtifact: &model.CatalogModelArtifact{
								ArtifactType: "model-artifact",
								Uri:          "s3://bucket/model.bin",
							},
						},
					},
				},
			},
			validateFunc: func(t *testing.T, record ModelProviderRecord) {
				require.NotNil(t, record.Model)
				assert.Len(t, record.Artifacts, 1)

				artifact := record.Artifacts[0]
				require.NotNil(t, artifact.CatalogModelArtifact)
				assert.Nil(t, artifact.CatalogMetricsArtifact)

				attrs := artifact.CatalogModelArtifact.(catalogmodels.CatalogModelArtifact).GetAttributes()
				assert.Equal(t, "s3://bucket/model.bin", *attrs.URI)
				assert.Nil(t, attrs.CreateTimeSinceEpoch)
				assert.Nil(t, attrs.LastUpdateTimeSinceEpoch)

				// Check regular properties
				props := artifact.CatalogModelArtifact.GetProperties()
				require.NotNil(t, props)
				assert.Len(t, *props, 1)
				assert.Equal(t, "uri", (*props)[0].Name)
				assert.Equal(t, "s3://bucket/model.bin", *(*props)[0].StringValue)
				assert.False(t, (*props)[0].IsCustomProperty)

				// Should have no custom properties
				customProps := artifact.CatalogModelArtifact.GetCustomProperties()
				if customProps != nil {
					assert.Empty(t, *customProps)
				}
			},
		},
		{
			name: "model with only MetricsArtifact",
			yamlModel: yamlModel{
				CatalogModel: model.CatalogModel{
					Name: "model-with-metrics",
				},
				Artifacts: []*yamlArtifact{
					{
						CatalogArtifact: model.CatalogArtifact{
							CatalogMetricsArtifact: &model.CatalogMetricsArtifact{
								ArtifactType: "metrics-artifact",
								MetricsType:  "performance-metrics",
							},
						},
					},
				},
			},
			validateFunc: func(t *testing.T, record ModelProviderRecord) {
				require.NotNil(t, record.Model)
				assert.Len(t, record.Artifacts, 1)

				artifact := record.Artifacts[0]
				assert.Nil(t, artifact.CatalogModelArtifact)
				require.NotNil(t, artifact.CatalogMetricsArtifact)

				attrs := artifact.CatalogMetricsArtifact.(catalogmodels.CatalogMetricsArtifact).GetAttributes()
				assert.Equal(t, "performance-metrics", string(attrs.MetricsType))
				assert.Nil(t, attrs.CreateTimeSinceEpoch)
				assert.Nil(t, attrs.LastUpdateTimeSinceEpoch)

				// Check regular properties
				props := artifact.CatalogMetricsArtifact.GetProperties()
				require.NotNil(t, props)
				assert.Len(t, *props, 1)
				assert.Equal(t, "metricsType", (*props)[0].Name)
				assert.Equal(t, "performance-metrics", *(*props)[0].StringValue)
				assert.False(t, (*props)[0].IsCustomProperty)

				// Should have no custom properties
				customProps := artifact.CatalogMetricsArtifact.GetCustomProperties()
				if customProps != nil {
					assert.Empty(t, *customProps)
				}
			},
		},
		{
			name: "model with validatedTasks only",
			yamlModel: yamlModel{
				CatalogModel: model.CatalogModel{
					Name:           "model-validated-tasks",
					ValidatedTasks: []string{"text-generation", "tool-calling"},
				},
			},
			validateFunc: func(t *testing.T, record ModelProviderRecord) {
				require.NotNil(t, record.Model)

				regularProps := record.Model.GetProperties()
				require.NotNil(t, regularProps)

				regularPropMap := make(map[string]models.Properties)
				for _, prop := range *regularProps {
					regularPropMap[prop.Name] = prop
				}

				assert.Contains(t, regularPropMap, "validated_tasks")
				var validatedTasks []string
				err := json.Unmarshal([]byte(*regularPropMap["validated_tasks"].StringValue), &validatedTasks)
				require.NoError(t, err)
				assert.Equal(t, []string{"text-generation", "tool-calling"}, validatedTasks)

				assert.NotContains(t, regularPropMap, "serving_config")
			},
		},
		{
			name: "model with servingConfig only",
			yamlModel: yamlModel{
				CatalogModel: model.CatalogModel{
					Name: "model-serving-config",
					ServingConfig: &model.ServingConfig{
						ToolCalling: &model.ToolCallingConfig{
							ToolCallParser:       new("granite"),
							ChatTemplate:         new("opt/app-root/template/tool_chat_template_granite.jinja"),
							EnableAutoToolChoice: new(true),
							RequiredArgs:         []string{"--config_format granite"},
						},
					},
				},
			},
			validateFunc: func(t *testing.T, record ModelProviderRecord) {
				require.NotNil(t, record.Model)

				regularProps := record.Model.GetProperties()
				require.NotNil(t, regularProps)

				regularPropMap := make(map[string]models.Properties)
				for _, prop := range *regularProps {
					regularPropMap[prop.Name] = prop
				}

				assert.Contains(t, regularPropMap, "serving_config")
				var servingConfig model.ServingConfig
				err := json.Unmarshal([]byte(*regularPropMap["serving_config"].StringValue), &servingConfig)
				require.NoError(t, err)
				require.NotNil(t, servingConfig.ToolCalling)
				assert.Equal(t, new("granite"), servingConfig.ToolCalling.ToolCallParser)
				assert.Equal(t, new("opt/app-root/template/tool_chat_template_granite.jinja"), servingConfig.ToolCalling.ChatTemplate)
				assert.Equal(t, new(true), servingConfig.ToolCalling.EnableAutoToolChoice)
				assert.Equal(t, []string{"--config_format granite"}, servingConfig.ToolCalling.RequiredArgs)
			},
		},
		{
			name: "model with empty servingConfig",
			yamlModel: yamlModel{
				CatalogModel: model.CatalogModel{
					Name:          "model-empty-serving-config",
					ServingConfig: &model.ServingConfig{},
				},
			},
			validateFunc: func(t *testing.T, record ModelProviderRecord) {
				require.NotNil(t, record.Model)

				regularProps := record.Model.GetProperties()
				require.NotNil(t, regularProps)

				regularPropMap := make(map[string]models.Properties)
				for _, prop := range *regularProps {
					regularPropMap[prop.Name] = prop
				}

				assert.Contains(t, regularPropMap, "serving_config")
				var servingConfig model.ServingConfig
				err := json.Unmarshal([]byte(*regularPropMap["serving_config"].StringValue), &servingConfig)
				require.NoError(t, err)
				assert.Nil(t, servingConfig.ToolCalling)
			},
		},
		{
			name: "model with servingConfig partial toolCalling fields",
			yamlModel: yamlModel{
				CatalogModel: model.CatalogModel{
					Name: "model-partial-serving-config",
					ServingConfig: &model.ServingConfig{
						ToolCalling: &model.ToolCallingConfig{
							EnableAutoToolChoice: new(false),
						},
					},
				},
			},
			validateFunc: func(t *testing.T, record ModelProviderRecord) {
				require.NotNil(t, record.Model)

				regularProps := record.Model.GetProperties()
				require.NotNil(t, regularProps)

				regularPropMap := make(map[string]models.Properties)
				for _, prop := range *regularProps {
					regularPropMap[prop.Name] = prop
				}

				assert.Contains(t, regularPropMap, "serving_config")
				var servingConfig model.ServingConfig
				err := json.Unmarshal([]byte(*regularPropMap["serving_config"].StringValue), &servingConfig)
				require.NoError(t, err)
				require.NotNil(t, servingConfig.ToolCalling)
				assert.Equal(t, new(false), servingConfig.ToolCalling.EnableAutoToolChoice)
				assert.Nil(t, servingConfig.ToolCalling.ToolCallParser)
				assert.Nil(t, servingConfig.ToolCalling.ChatTemplate)
				assert.Nil(t, servingConfig.ToolCalling.RequiredArgs)
			},
		},
		{
			name: "artifacts with invalid timestamps",
			yamlModel: yamlModel{
				CatalogModel: model.CatalogModel{
					Name: "model-with-invalid-artifact-timestamps",
				},
				Artifacts: []*yamlArtifact{
					{
						CatalogArtifact: model.CatalogArtifact{
							CatalogModelArtifact: &model.CatalogModelArtifact{
								ArtifactType:             "model-artifact",
								Uri:                      "https://example.com/model.bin",
								CreateTimeSinceEpoch:     new("invalid-timestamp"),
								LastUpdateTimeSinceEpoch: new("also-invalid"),
							},
						},
					},
				},
			},
			validateFunc: func(t *testing.T, record ModelProviderRecord) {
				require.NotNil(t, record.Model)
				assert.Len(t, record.Artifacts, 1)

				artifact := record.Artifacts[0]
				require.NotNil(t, artifact.CatalogModelArtifact)

				attrs := artifact.CatalogModelArtifact.(catalogmodels.CatalogModelArtifact).GetAttributes()
				assert.Equal(t, "https://example.com/model.bin", *attrs.URI)
				// Invalid timestamps should be ignored (not set)
				assert.Nil(t, attrs.CreateTimeSinceEpoch)
				assert.Nil(t, attrs.LastUpdateTimeSinceEpoch)
			},
		},
		{
			name: "model with invalid timestamps",
			yamlModel: yamlModel{
				CatalogModel: model.CatalogModel{
					Name:                     "invalid-timestamp-model",
					CreateTimeSinceEpoch:     new("invalid-timestamp"),
					LastUpdateTimeSinceEpoch: new("also-invalid"),
				},
			},
			validateFunc: func(t *testing.T, record ModelProviderRecord) {
				require.NotNil(t, record.Model)

				attrs := record.Model.GetAttributes()
				require.NotNil(t, attrs)
				assert.Equal(t, "invalid-timestamp-model", *attrs.Name)
				// Invalid timestamps should be ignored (not set)
				assert.Nil(t, attrs.CreateTimeSinceEpoch)
				assert.Nil(t, attrs.LastUpdateTimeSinceEpoch)
			},
		},
		{
			name: "artifact with invalid URI returns error",
			yamlModel: yamlModel{
				CatalogModel: model.CatalogModel{
					Name: "model-with-invalid-uri",
				},
				Artifacts: []*yamlArtifact{
					{
						CatalogArtifact: model.CatalogArtifact{
							CatalogModelArtifact: &model.CatalogModelArtifact{
								ArtifactType: "model-artifact",
								Uri:          "not-a-valid-uri",
							},
						},
					},
				},
			},
			expectError: true,
			validateFunc: func(t *testing.T, record ModelProviderRecord) {
				require.Error(t, record.Error)
				assert.Contains(t, record.Error.Error(), "must have a scheme")
				assert.Contains(t, record.Error.Error(), "model-with-invalid-uri")
			},
		},
		{
			name: "artifact with unsupported URI scheme returns error",
			yamlModel: yamlModel{
				CatalogModel: model.CatalogModel{
					Name: "model-with-ftp-uri",
				},
				Artifacts: []*yamlArtifact{
					{
						CatalogArtifact: model.CatalogArtifact{
							CatalogModelArtifact: &model.CatalogModelArtifact{
								ArtifactType: "model-artifact",
								Uri:          "ftp://example.com/model.bin",
							},
						},
					},
				},
			},
			expectError: true,
			validateFunc: func(t *testing.T, record ModelProviderRecord) {
				require.Error(t, record.Error)
				assert.Contains(t, record.Error.Error(), "unsupported scheme")
				assert.Contains(t, record.Error.Error(), "ftp")
			},
		},
		{
			name: "artifact with empty URI returns error",
			yamlModel: yamlModel{
				CatalogModel: model.CatalogModel{
					Name: "model-with-empty-uri",
				},
				Artifacts: []*yamlArtifact{
					{
						CatalogArtifact: model.CatalogArtifact{
							CatalogModelArtifact: &model.CatalogModelArtifact{
								ArtifactType: "model-artifact",
								Uri:          "",
							},
						},
					},
				},
			},
			expectError: true,
			validateFunc: func(t *testing.T, record ModelProviderRecord) {
				require.Error(t, record.Error)
				assert.Contains(t, record.Error.Error(), "cannot be empty")
			},
		},
		{
			name: "valid artifact followed by invalid artifact returns error",
			yamlModel: yamlModel{
				CatalogModel: model.CatalogModel{
					Name: "model-with-mixed-artifacts",
				},
				Artifacts: []*yamlArtifact{
					{
						CatalogArtifact: model.CatalogArtifact{
							CatalogModelArtifact: &model.CatalogModelArtifact{
								ArtifactType: "model-artifact",
								Uri:          "s3://valid-bucket/model.bin",
							},
						},
					},
					{
						CatalogArtifact: model.CatalogArtifact{
							CatalogModelArtifact: &model.CatalogModelArtifact{
								ArtifactType: "model-artifact",
								Uri:          "invalid-uri-without-scheme",
							},
						},
					},
				},
			},
			expectError: true,
			validateFunc: func(t *testing.T, record ModelProviderRecord) {
				require.Error(t, record.Error)
				assert.Contains(t, record.Error.Error(), "artifact 1")
				assert.Contains(t, record.Error.Error(), "must have a scheme")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := tt.yamlModel.ToModelProviderRecord()
			tt.validateFunc(t, record)
		})
	}
}

func TestNewYamlModelProviderAbsolutePath(t *testing.T) {
	// Create a temporary YAML file
	tempDir, err := os.MkdirTemp("", "yaml_catalog_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	yamlContent := `
source: test
models:
  - name: test-model
    description: A test model
`

	absolutePath := filepath.Join(tempDir, "catalog.yaml")
	err = os.WriteFile(absolutePath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Test that absolute paths work correctly (bug fix)
	t.Run("absolute path works correctly", func(t *testing.T) {
		source := &basecatalog.ModelSource{
			Properties: map[string]any{
				yamlCatalogPathKey: absolutePath, // Use absolute path to existing file
			},
		}

		// Use a different reldir - this shouldn't matter for absolute paths
		reldir := "/some/other/directory"

		_, err := newYamlModelProvider(t.Context(), source, reldir)

		// This should succeed because we provided a valid absolute path
		require.NoError(t, err, "Absolute path should work without reldir being prepended")
	})

	// Test that relative paths work correctly
	t.Run("relative path works correctly", func(t *testing.T) {
		// Create the expected structure for relative path
		reldir := tempDir
		relativePath := "catalog.yaml"

		source := &basecatalog.ModelSource{
			Properties: map[string]any{
				yamlCatalogPathKey: relativePath,
			},
		}

		_, err := newYamlModelProvider(t.Context(), source, reldir)

		// This should work because filepath.Join(reldir, relativePath) points to our file
		require.NoError(t, err, "Relative paths should work correctly")
	})
}

func TestYamlModelProviderFiltersApplied(t *testing.T) {
	catalogPath := writeMiniCatalog(t, []string{"Granite/alpha", "Granite/beta-release", "DeepSeek/v1"})

	source := &basecatalog.ModelSource{
		CatalogSource: model.CatalogSource{
			Id:             "test",
			Name:           "Test source",
			Labels:         []string{},
			IncludedModels: []string{"Granite/*"},
			ExcludedModels: []string{"Granite/beta-*"},
		},
	}

	filter, err := NewModelFilterFromSource(source, nil, nil)
	require.NoError(t, err)

	names := collectNamesWithFilter(t, catalogPath, filter)
	assert.ElementsMatch(t, []string{"Granite/alpha"}, names)
}

func TestYamlModelProviderLegacyExcludesMerged(t *testing.T) {
	catalogPath := writeMiniCatalog(t, []string{"Granite/alpha", "Granite/beta-release", "DeepSeek/v1"})

	source := &basecatalog.ModelSource{
		CatalogSource: model.CatalogSource{
			Id:             "test",
			Name:           "Test source",
			Labels:         []string{},
			ExcludedModels: []string{"Granite/beta-*"},
		},
		Properties: map[string]any{
			yamlCatalogPathKey: catalogPath,
			excludedModelsKey:  []any{"DeepSeek/v1"},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	records, err := newYamlModelProvider(ctx, source, filepath.Dir(catalogPath))
	require.NoError(t, err)

	names := collectRecordsFromChannel(t, records, 1, cancel)
	assert.Equal(t, []string{"Granite/alpha"}, names)
}

func TestYamlModelProviderInvalidPattern(t *testing.T) {
	catalogPath := writeMiniCatalog(t, []string{"Granite/alpha"})

	source := &basecatalog.ModelSource{
		CatalogSource: model.CatalogSource{
			Id:             "test",
			Name:           "Test source",
			Labels:         []string{},
			IncludedModels: []string{""},
		},
		Properties: map[string]any{
			yamlCatalogPathKey: catalogPath,
		},
	}

	_, err := newYamlModelProvider(context.Background(), source, filepath.Dir(catalogPath))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pattern cannot be empty")
}

func TestYamlModelHardwareTagAsCustomProperty(t *testing.T) {
	t.Run("single hardware_tag value", func(t *testing.T) {
		ym := yamlModel{
			CatalogModel: model.CatalogModel{
				Name:  "test-vendor/single-hw",
				Tasks: []string{"text-generation"},
				CustomProperties: map[string]model.MetadataValue{
					"hardware_tag": {
						MetadataStringValue: &model.MetadataStringValue{
							StringValue:  "Hardware Tag 1",
							MetadataType: "MetadataStringValue",
						},
					},
				},
			},
			Artifacts: []*yamlArtifact{
				{CatalogArtifact: model.CatalogArtifact{
					CatalogModelArtifact: &model.CatalogModelArtifact{
						ArtifactType: "model-artifact",
						Uri:          "oci://registry.example.com/test:latest",
					},
				}},
			},
		}

		record := ym.ToModelProviderRecord()
		require.NotNil(t, record.Model)
		require.Nil(t, record.Error)

		customPropMap := customPropsToMap(t, record)
		assert.Contains(t, customPropMap, "hardware_tag")
		assert.Equal(t, "Hardware Tag 1", *customPropMap["hardware_tag"].StringValue)
		assert.True(t, customPropMap["hardware_tag"].IsCustomProperty)
	})

	t.Run("multiple hardware_tag values comma-separated", func(t *testing.T) {
		ym := yamlModel{
			CatalogModel: model.CatalogModel{
				Name:  "test-vendor/multi-hw",
				Tasks: []string{"text-generation"},
				CustomProperties: map[string]model.MetadataValue{
					"hardware_tag": {
						MetadataStringValue: &model.MetadataStringValue{
							StringValue:  "Hardware Tag 1,Hardware Tag 2",
							MetadataType: "MetadataStringValue",
						},
					},
				},
			},
			Artifacts: []*yamlArtifact{
				{CatalogArtifact: model.CatalogArtifact{
					CatalogModelArtifact: &model.CatalogModelArtifact{
						ArtifactType: "model-artifact",
						Uri:          "oci://registry.example.com/test:latest",
					},
				}},
			},
		}

		record := ym.ToModelProviderRecord()
		require.NotNil(t, record.Model)
		require.Nil(t, record.Error)

		customPropMap := customPropsToMap(t, record)
		assert.Contains(t, customPropMap, "hardware_tag")
		assert.Equal(t, "Hardware Tag 1,Hardware Tag 2", *customPropMap["hardware_tag"].StringValue)
		assert.True(t, customPropMap["hardware_tag"].IsCustomProperty)
	})

	t.Run("empty hardware_tag produces no custom property", func(t *testing.T) {
		ym := yamlModel{
			CatalogModel: model.CatalogModel{
				Name:  "test-vendor/empty-hw",
				Tasks: []string{"text-generation"},
				CustomProperties: map[string]model.MetadataValue{
					"hardware_tag": {
						MetadataStringValue: &model.MetadataStringValue{
							StringValue:  "",
							MetadataType: "MetadataStringValue",
						},
					},
					"model_type": {
						MetadataStringValue: &model.MetadataStringValue{
							StringValue:  "predictive",
							MetadataType: "MetadataStringValue",
						},
					},
				},
			},
			Artifacts: []*yamlArtifact{
				{CatalogArtifact: model.CatalogArtifact{
					CatalogModelArtifact: &model.CatalogModelArtifact{
						ArtifactType: "model-artifact",
						Uri:          "oci://registry.example.com/test:latest",
					},
				}},
			},
		}

		record := ym.ToModelProviderRecord()
		require.NotNil(t, record.Model)
		require.Nil(t, record.Error)

		customPropMap := customPropsToMap(t, record)
		assert.NotContains(t, customPropMap, "hardware_tag")
		assert.Contains(t, customPropMap, "model_type")
		assert.Equal(t, "predictive", *customPropMap["model_type"].StringValue)
	})

	t.Run("missing hardware_tag produces no custom property", func(t *testing.T) {
		ym := yamlModel{
			CatalogModel: model.CatalogModel{
				Name:  "test-vendor/no-hw",
				Tasks: []string{"text-generation"},
				CustomProperties: map[string]model.MetadataValue{
					"model_type": {
						MetadataStringValue: &model.MetadataStringValue{
							StringValue:  "generative",
							MetadataType: "MetadataStringValue",
						},
					},
				},
			},
			Artifacts: []*yamlArtifact{
				{CatalogArtifact: model.CatalogArtifact{
					CatalogModelArtifact: &model.CatalogModelArtifact{
						ArtifactType: "model-artifact",
						Uri:          "oci://registry.example.com/test:latest",
					},
				}},
			},
		}

		record := ym.ToModelProviderRecord()
		require.NotNil(t, record.Model)
		require.Nil(t, record.Error)

		customPropMap := customPropsToMap(t, record)
		assert.NotContains(t, customPropMap, "hardware_tag")
		assert.Contains(t, customPropMap, "model_type")
		assert.Equal(t, "generative", *customPropMap["model_type"].StringValue)
	})
}

func customPropsToMap(t *testing.T, record ModelProviderRecord) map[string]models.Properties {
	t.Helper()
	customProps := record.Model.GetCustomProperties()
	m := make(map[string]models.Properties)
	if customProps == nil {
		return m
	}
	for _, prop := range *customProps {
		m[prop.Name] = prop
	}
	return m
}

func writeMiniCatalog(t *testing.T, modelNames []string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.yaml")

	var b strings.Builder
	b.WriteString("source: Test\nmodels:\n")
	for _, name := range modelNames {
		b.WriteString(fmt.Sprintf("  - name: %s\n", name))
	}

	err := os.WriteFile(path, []byte(b.String()), 0o644)
	require.NoError(t, err)

	return path
}

func collectNamesWithFilter(t *testing.T, catalogPath string, filter *ModelFilter) []string {
	t.Helper()

	provider := &yamlModelProvider{
		path:   catalogPath,
		filter: filter,
	}

	catalog, err := provider.read()
	require.NoError(t, err)

	out := make(chan ModelProviderRecord, len(catalog.Models))
	provider.emit(context.Background(), catalog, out)
	close(out)

	var names []string
	for record := range out {
		if record.Model != nil {
			names = append(names, modelNameFromRecord(t, record))
		}
	}

	return names
}

func collectRecordsFromChannel(t *testing.T, records <-chan ModelProviderRecord, expected int, cancel context.CancelFunc) []string {
	t.Helper()

	names := make([]string, 0, expected)
	timeout := time.After(2 * time.Second)

	for len(names) < expected {
		select {
		case record, ok := <-records:
			if !ok {
				t.Fatalf("channel closed before receiving %d records", expected)
			}
			if record.Model != nil {
				names = append(names, modelNameFromRecord(t, record))
			}
		case <-timeout:
			t.Fatalf("timed out waiting for %d records", expected)
		}
	}

	cancel()

	select {
	case record, ok := <-records:
		if ok && record.Model != nil {
			t.Fatalf("received more than %d records", expected)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("channel did not close after cancellation")
	}

	return names
}

func modelNameFromRecord(t *testing.T, record ModelProviderRecord) string {
	t.Helper()

	require.NotNil(t, record.Model)
	attrs := record.Model.GetAttributes()
	require.NotNil(t, attrs)
	require.NotNil(t, attrs.Name)
	return *attrs.Name
}

// TestModels_InitialReadFailure verifies that when the YAML file is initially
// invalid, Models() still returns a channel (no error), and the channel emits
// an error record followed by a sentinel.
func TestModels_InitialReadFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.yaml")
	require.NoError(t, os.WriteFile(path, []byte("!!invalid: yaml: :::"), 0600))

	p := &yamlModelProvider{path: path, filter: &ModelFilter{}}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, err := p.Models(ctx)
	require.NoError(t, err, "Models() should not return an error for a bad data file")
	require.NotNil(t, ch)

	// First record: error record (Model nil, Error non-nil)
	r1 := <-ch
	assert.Nil(t, r1.Model)
	assert.Error(t, r1.Error)

	// Second record: sentinel (both nil)
	r2 := <-ch
	assert.Nil(t, r2.Model)
	assert.Nil(t, r2.Error)
}

// TestModels_RecoveryAfterInitialFailure verifies that when the initial YAML
// file is invalid but then fixed, the file watcher detects the fix and emits
// valid model records.
//
// The watcher is set up BEFORE the initial read, so by the time the test reads
// the error+sentinel and writes the fixed file, the watcher is guaranteed to
// already be watching.
func TestModels_RecoveryAfterInitialFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.yaml")
	require.NoError(t, os.WriteFile(path, []byte("!!invalid yaml"), 0600))

	p := &yamlModelProvider{path: path, filter: &ModelFilter{}}
	// Use t.Context() so the context stays alive for the full test lifetime.
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	ch, err := p.Models(ctx)
	require.NoError(t, err)

	// Drain the initial error + sentinel. Because Models() sets up the watcher
	// before the initial read, by the time we finish reading these two records
	// the watcher is already registered and watching the file.
	r1 := <-ch
	assert.Nil(t, r1.Model)
	assert.Error(t, r1.Error)
	r2 := <-ch
	assert.Nil(t, r2.Model)

	// Fix the file. The watcher is already running at this point.
	validYAML := `source: test
models:
  - name: recovered-model
    artifacts:
      - uri: "https://example.com/model.bin"
`
	require.NoError(t, os.WriteFile(path, []byte(validYAML), 0600))

	// Wait for the watcher to fire and emit the recovered model. The monitor
	// pauses 1 second after an fsnotify event before dispatching, so we allow
	// a generous timeout.
	var gotModel bool
	assert.Eventually(t, func() bool {
		select {
		case r, ok := <-ch:
			if !ok {
				return false
			}
			if r.Model != nil {
				gotModel = true
				return true
			}
		default:
		}
		return false
	}, 15*time.Second, 100*time.Millisecond, "file-watch reload should emit a valid model after the file is fixed")
	assert.True(t, gotModel)
}

// TestModels_ReloadErrorEmitsSentinel verifies that when a valid YAML file is
// replaced with an invalid one, the reload emits an error record + sentinel
// instead of silently continuing, so the consumer can update the source status.
//
// The watcher is set up BEFORE the initial read, so the file write happens
// after the watcher is already watching — no race.
func TestModels_ReloadErrorEmitsSentinel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.yaml")

	validYAML := `source: test
models:
  - name: my-model
    artifacts:
      - uri: "https://example.com/model.bin"
`
	require.NoError(t, os.WriteFile(path, []byte(validYAML), 0600))

	p := &yamlModelProvider{path: path, filter: &ModelFilter{}}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	ch, err := p.Models(ctx)
	require.NoError(t, err)

	// Drain the initial successful batch (model + sentinel). Because Models()
	// sets up the watcher before the initial read, the watcher is already
	// running once we finish reading these records.
	for {
		r := <-ch
		if r.Model == nil {
			break
		}
	}

	// Overwrite with invalid YAML. The watcher is already watching.
	require.NoError(t, os.WriteFile(path, []byte("!!!not: valid: yaml"), 0600))

	// Wait for the reload to emit an error record. The monitor pauses 1 second
	// after an fsnotify event before dispatching, so we allow a generous timeout.
	var gotErr bool
	assert.Eventually(t, func() bool {
		select {
		case r, ok := <-ch:
			if !ok {
				return false
			}
			if r.Model == nil && r.Error != nil {
				gotErr = true
				return true
			}
		default:
		}
		return false
	}, 15*time.Second, 100*time.Millisecond, "file-watch reload should emit an error record after invalid YAML is written")
	assert.True(t, gotErr)
}
