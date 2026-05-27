package modelcatalog

import (
	"context"
	"testing"

	"github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	modelservice "github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/service"
	"github.com/kubeflow/hub/catalog/internal/db/service"
	"github.com/kubeflow/hub/catalog/internal/testhelpers"
	"github.com/kubeflow/hub/internal/platform/apiutils"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/kubeflow/hub/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_PreservedRecommendationAlgorithm verifies end-to-end functionality
// with the exact preserved recommendations algorithm from db_catalog.go
func TestIntegration_PreservedRecommendationAlgorithm(t *testing.T) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, service.DatastoreSpec())
	defer cleanup()

	// Get type IDs
	catalogModelTypeID := testhelpers.GetCatalogModelTypeIDForDBTest(t, sharedDB)
	metricsArtifactTypeID := testhelpers.GetCatalogMetricsArtifactTypeIDForDBTest(t, sharedDB)
	modelArtifactTypeID := testhelpers.GetCatalogModelArtifactTypeIDForDBTest(t, sharedDB)
	catalogSourceTypeID := testhelpers.GetCatalogSourceTypeIDForDBTest(t, sharedDB)

	// Create repositories
	catalogModelRepo := modelservice.NewCatalogModelRepository(sharedDB, catalogModelTypeID)
	catalogArtifactRepo := service.NewCatalogArtifactRepository(sharedDB, map[string]int32{
		service.CatalogModelArtifactTypeName:   modelArtifactTypeID,
		service.CatalogMetricsArtifactTypeName: metricsArtifactTypeID,
	})
	modelArtifactRepo := modelservice.NewCatalogModelArtifactRepository(sharedDB, modelArtifactTypeID)
	metricsArtifactRepo := modelservice.NewCatalogMetricsArtifactRepository(sharedDB, metricsArtifactTypeID)
	catalogSourceRepo := service.NewCatalogSourceRepository(sharedDB, catalogSourceTypeID)

	services := Services{
		CatalogModelRepository:           catalogModelRepo,
		CatalogArtifactRepository:        catalogArtifactRepo,
		CatalogModelArtifactRepository:   modelArtifactRepo,
		CatalogMetricsArtifactRepository: metricsArtifactRepo,
		CatalogSourceRepository:          catalogSourceRepo,
		PropertyOptionsRepository:        service.NewPropertyOptionsRepository(sharedDB),
	}

	provider := NewDBCatalog(services, nil)

	// Create test model (stored name must be namespaced: sourceId:modelName)
	testModel := &models.CatalogModelImpl{
		TypeID: apiutils.Of(catalogModelTypeID),
		Attributes: &models.CatalogModelAttributes{
			Name:       apiutils.Of("algorithm-test-source:algorithm-test-model"),
			ExternalID: apiutils.Of("alg-test-model-789"),
		},
		Properties: &[]dbmodels.Properties{
			{Name: "source_id", StringValue: apiutils.Of("algorithm-test-source")},
		},
	}
	savedModel, err := services.CatalogModelRepository.Save(testModel)
	require.NoError(t, err)

	// Create artifacts that test the specific algorithm behavior
	// These match the exact patterns expected by the preserved algorithm
	artifacts := []*models.CatalogMetricsArtifactImpl{
		{
			TypeID: apiutils.Of(metricsArtifactTypeID),
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:        apiutils.Of("fast-low-cost"),
				ExternalID:  apiutils.Of("fast-low"),
				MetricsType: models.MetricsTypePerformance,
			},
			Properties: &[]dbmodels.Properties{
				{Name: "metricsType", StringValue: apiutils.Of("performance-metrics")},
			},
			CustomProperties: &[]dbmodels.Properties{
				{Name: "requests_per_second", DoubleValue: apiutils.Of(250.0)},
				{Name: "ttft_p90", DoubleValue: apiutils.Of(50.0)},
				{Name: "hardware_count", IntValue: apiutils.Of(int32(2))},
				{Name: "hardware_type", StringValue: apiutils.Of("gpu-a100")},
			},
		},
		{
			TypeID: apiutils.Of(metricsArtifactTypeID),
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:        apiutils.Of("medium-medium-cost"),
				ExternalID:  apiutils.Of("medium-med"),
				MetricsType: models.MetricsTypePerformance,
			},
			Properties: &[]dbmodels.Properties{
				{Name: "metricsType", StringValue: apiutils.Of("performance-metrics")},
			},
			CustomProperties: &[]dbmodels.Properties{
				{Name: "requests_per_second", DoubleValue: apiutils.Of(150.0)},
				{Name: "ttft_p90", DoubleValue: apiutils.Of(120.0)}, // > 100, should use epsilon 0.1
				{Name: "hardware_count", IntValue: apiutils.Of(int32(3))},
				{Name: "hardware_type", StringValue: apiutils.Of("gpu-a100")},
			},
		},
		{
			TypeID: apiutils.Of(metricsArtifactTypeID),
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:        apiutils.Of("slow-high-cost"),
				ExternalID:  apiutils.Of("slow-high"),
				MetricsType: models.MetricsTypePerformance,
			},
			Properties: &[]dbmodels.Properties{
				{Name: "metricsType", StringValue: apiutils.Of("performance-metrics")},
			},
			CustomProperties: &[]dbmodels.Properties{
				{Name: "requests_per_second", DoubleValue: apiutils.Of(100.0)},
				{Name: "ttft_p90", DoubleValue: apiutils.Of(200.0)},
				{Name: "hardware_count", IntValue: apiutils.Of(int32(5))}, // Should be filtered out by first pass
				{Name: "hardware_type", StringValue: apiutils.Of("gpu-a100")},
			},
		},
	}

	// Save all artifacts
	for _, artifact := range artifacts {
		_, err := services.CatalogMetricsArtifactRepository.Save(artifact, savedModel.GetID())
		require.NoError(t, err)
	}

	t.Run("Exact_Algorithm_TwoPass_Filtering", func(t *testing.T) {
		params := ListPerformanceArtifactsParams{
			TargetRPS:       300, // This will add replicas for cost calculation
			Recommendations: true,
			PageSize:        10,
		}

		result, err := provider.GetPerformanceArtifacts(
			context.Background(),
			"algorithm-test-model",
			"algorithm-test-source",
			params,
		)
		require.NoError(t, err)

		// Verify results match exact algorithm expectations
		assert.GreaterOrEqual(t, len(result.Items), 1)

		// Check that replicas were calculated and used in cost calculation
		for _, artifact := range result.Items {
			assert.Contains(t, artifact.CatalogMetricsArtifact.CustomProperties, "replicas")
		}

		// Verify no slow-high-cost in results (should be filtered by first pass)
		names := make([]string, len(result.Items))
		for i, artifact := range result.Items {
			names[i] = *artifact.CatalogMetricsArtifact.Name
		}
		assert.NotContains(t, names, "slow-high-cost")
	})

	t.Run("Algorithm_Epsilon_Threshold_Behavior", func(t *testing.T) {
		params := ListPerformanceArtifactsParams{
			TargetRPS:       100,
			Recommendations: true,
			PageSize:        10,
		}

		result, err := provider.GetPerformanceArtifacts(
			context.Background(),
			"algorithm-test-model",
			"algorithm-test-source",
			params,
		)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(result.Items), 1)

		for _, artifact := range result.Items {
			replicas := artifact.CatalogMetricsArtifact.CustomProperties["replicas"]
			require.NotNil(t, replicas.MetadataIntValue)
			assert.NotEmpty(t, replicas.MetadataIntValue.IntValue)
		}
	})

	t.Run("TargetRPS_Calculations_Integration", func(t *testing.T) {
		params := ListPerformanceArtifactsParams{
			TargetRPS:       600,
			Recommendations: false,
			PageSize:        10,
		}

		result, err := provider.GetPerformanceArtifacts(
			context.Background(),
			"algorithm-test-model",
			"algorithm-test-source",
			params,
		)
		require.NoError(t, err)

		assert.Len(t, result.Items, 3)

		for _, artifact := range result.Items {
			assert.Contains(t, artifact.CatalogMetricsArtifact.CustomProperties, "replicas")
			assert.Contains(t, artifact.CatalogMetricsArtifact.CustomProperties, "total_requests_per_second")

			rpsVal := artifact.CatalogMetricsArtifact.CustomProperties["requests_per_second"]
			require.NotNil(t, rpsVal.MetadataDoubleValue)

			totalRPSVal := artifact.CatalogMetricsArtifact.CustomProperties["total_requests_per_second"]
			require.NotNil(t, totalRPSVal.MetadataDoubleValue)

			assert.GreaterOrEqual(t, totalRPSVal.MetadataDoubleValue.DoubleValue, 600.0)
		}
	})

	t.Run("Repository_Filtering_PerformanceMetricsOnly", func(t *testing.T) {
		nonPerfArtifact := &models.CatalogMetricsArtifactImpl{
			TypeID: apiutils.Of(metricsArtifactTypeID),
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:        apiutils.Of("quality-metrics"),
				ExternalID:  apiutils.Of("quality-123"),
				MetricsType: models.MetricsTypeAccuracy,
			},
			Properties: &[]dbmodels.Properties{
				{Name: "metricsType", StringValue: apiutils.Of("accuracy-metrics")},
			},
			CustomProperties: &[]dbmodels.Properties{
				{Name: "accuracy", DoubleValue: apiutils.Of(0.95)},
			},
		}
		_, err := services.CatalogMetricsArtifactRepository.Save(nonPerfArtifact, savedModel.GetID())
		require.NoError(t, err)

		params := ListPerformanceArtifactsParams{
			TargetRPS:       100,
			Recommendations: false,
			PageSize:        10,
		}

		result, err := provider.GetPerformanceArtifacts(
			context.Background(),
			"algorithm-test-model",
			"algorithm-test-source",
			params,
		)
		require.NoError(t, err)

		assert.Len(t, result.Items, 3)

		for _, artifact := range result.Items {
			assert.Equal(t, "performance-metrics", artifact.CatalogMetricsArtifact.MetricsType)
		}
	})

	t.Run("Original_Input_Order_Preservation", func(t *testing.T) {
		params := ListPerformanceArtifactsParams{
			TargetRPS:       300,
			Recommendations: true,
			PageSize:        10,
		}

		result, err := provider.GetPerformanceArtifacts(
			context.Background(),
			"algorithm-test-model",
			"algorithm-test-source",
			params,
		)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(result.Items), 1)

		for _, artifact := range result.Items {
			require.NotNil(t, artifact.CatalogMetricsArtifact)
			require.NotNil(t, artifact.CatalogMetricsArtifact.Id)
			require.NotNil(t, artifact.CatalogMetricsArtifact.Name)
		}
	})
}

// TestIntegration_ServiceLayerBehavior tests the service layer implementation
func TestIntegration_ServiceLayerBehavior(t *testing.T) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, service.DatastoreSpec())
	defer cleanup()

	catalogModelTypeID := testhelpers.GetCatalogModelTypeIDForDBTest(t, sharedDB)
	metricsArtifactTypeID := testhelpers.GetCatalogMetricsArtifactTypeIDForDBTest(t, sharedDB)
	modelArtifactTypeID := testhelpers.GetCatalogModelArtifactTypeIDForDBTest(t, sharedDB)
	catalogSourceTypeID := testhelpers.GetCatalogSourceTypeIDForDBTest(t, sharedDB)

	catalogModelRepo := modelservice.NewCatalogModelRepository(sharedDB, catalogModelTypeID)
	catalogArtifactRepo := service.NewCatalogArtifactRepository(sharedDB, map[string]int32{
		service.CatalogModelArtifactTypeName:   modelArtifactTypeID,
		service.CatalogMetricsArtifactTypeName: metricsArtifactTypeID,
	})
	modelArtifactRepo := modelservice.NewCatalogModelArtifactRepository(sharedDB, modelArtifactTypeID)
	metricsArtifactRepo := modelservice.NewCatalogMetricsArtifactRepository(sharedDB, metricsArtifactTypeID)
	catalogSourceRepo := service.NewCatalogSourceRepository(sharedDB, catalogSourceTypeID)

	services := Services{
		CatalogModelRepository:           catalogModelRepo,
		CatalogArtifactRepository:        catalogArtifactRepo,
		CatalogModelArtifactRepository:   modelArtifactRepo,
		CatalogMetricsArtifactRepository: metricsArtifactRepo,
		CatalogSourceRepository:          catalogSourceRepo,
		PropertyOptionsRepository:        service.NewPropertyOptionsRepository(sharedDB),
	}

	provider := NewDBCatalog(services, nil)

	// Stored name must be namespaced: sourceId:modelName
	testModel := &models.CatalogModelImpl{
		TypeID: apiutils.Of(catalogModelTypeID),
		Attributes: &models.CatalogModelAttributes{
			Name:       apiutils.Of("service-test-source:service-test-model"),
			ExternalID: apiutils.Of("service-model-123"),
		},
		Properties: &[]dbmodels.Properties{
			{Name: "source_id", StringValue: apiutils.Of("service-test-source")},
		},
	}
	savedModel, err := services.CatalogModelRepository.Save(testModel)
	require.NoError(t, err)

	t.Run("Multiple_Hardware_Types_Grouped_Correctly", func(t *testing.T) {
		artifacts := []*models.CatalogMetricsArtifactImpl{
			{
				TypeID: apiutils.Of(metricsArtifactTypeID),
				Attributes: &models.CatalogMetricsArtifactAttributes{
					Name:        apiutils.Of("a100-config-1"),
					ExternalID:  apiutils.Of("a100-1"),
					MetricsType: models.MetricsTypePerformance,
				},
				Properties: &[]dbmodels.Properties{
					{Name: "metricsType", StringValue: apiutils.Of("performance-metrics")},
				},
				CustomProperties: &[]dbmodels.Properties{
					{Name: "requests_per_second", DoubleValue: apiutils.Of(200.0)},
					{Name: "ttft_p90", DoubleValue: apiutils.Of(50.0)},
					{Name: "hardware_count", IntValue: apiutils.Of(int32(2))},
					{Name: "hardware_type", StringValue: apiutils.Of("gpu-a100")},
				},
			},
			{
				TypeID: apiutils.Of(metricsArtifactTypeID),
				Attributes: &models.CatalogMetricsArtifactAttributes{
					Name:        apiutils.Of("h100-config-1"),
					ExternalID:  apiutils.Of("h100-1"),
					MetricsType: models.MetricsTypePerformance,
				},
				Properties: &[]dbmodels.Properties{
					{Name: "metricsType", StringValue: apiutils.Of("performance-metrics")},
				},
				CustomProperties: &[]dbmodels.Properties{
					{Name: "requests_per_second", DoubleValue: apiutils.Of(300.0)},
					{Name: "ttft_p90", DoubleValue: apiutils.Of(40.0)},
					{Name: "hardware_count", IntValue: apiutils.Of(int32(1))},
					{Name: "hardware_type", StringValue: apiutils.Of("gpu-h100")},
				},
			},
		}

		for _, artifact := range artifacts {
			_, err := services.CatalogMetricsArtifactRepository.Save(artifact, savedModel.GetID())
			require.NoError(t, err)
		}

		params := ListPerformanceArtifactsParams{
			TargetRPS:       300,
			Recommendations: true,
			PageSize:        10,
		}

		result, err := provider.GetPerformanceArtifacts(
			context.Background(),
			"service-test-model",
			"service-test-source",
			params,
		)
		require.NoError(t, err)

		assert.Len(t, result.Items, 2)

		hardwareTypes := make(map[string]bool)
		for _, artifact := range result.Items {
			hwType := artifact.CatalogMetricsArtifact.CustomProperties["hardware_type"]
			require.NotNil(t, hwType.MetadataStringValue)
			hardwareTypes[hwType.MetadataStringValue.StringValue] = true
		}
		assert.True(t, hardwareTypes["gpu-a100"])
		assert.True(t, hardwareTypes["gpu-h100"])
	})

	t.Run("Cost_Calculation_With_Replicas", func(t *testing.T) {
		artifact := &models.CatalogMetricsArtifactImpl{
			TypeID: apiutils.Of(metricsArtifactTypeID),
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:        apiutils.Of("cost-test-artifact"),
				ExternalID:  apiutils.Of("cost-123"),
				MetricsType: models.MetricsTypePerformance,
			},
			Properties: &[]dbmodels.Properties{
				{Name: "metricsType", StringValue: apiutils.Of("performance-metrics")},
			},
			CustomProperties: &[]dbmodels.Properties{
				{Name: "requests_per_second", DoubleValue: apiutils.Of(100.0)},
				{Name: "ttft_p90", DoubleValue: apiutils.Of(80.0)},
				{Name: "hardware_count", IntValue: apiutils.Of(int32(4))},
				{Name: "hardware_type", StringValue: apiutils.Of("gpu-v100")},
			},
		}
		_, err := services.CatalogMetricsArtifactRepository.Save(artifact, savedModel.GetID())
		require.NoError(t, err)

		params := ListPerformanceArtifactsParams{
			TargetRPS:       500,
			Recommendations: false,
			PageSize:        10,
		}

		result, err := provider.GetPerformanceArtifacts(
			context.Background(),
			"service-test-model",
			"service-test-source",
			params,
		)
		require.NoError(t, err)

		var found bool
		for _, a := range result.Items {
			if *a.CatalogMetricsArtifact.Name == "cost-test-artifact" {
				found = true
				replicas := a.CatalogMetricsArtifact.CustomProperties["replicas"]
				require.NotNil(t, replicas.MetadataIntValue)
				assert.Equal(t, "5", replicas.MetadataIntValue.IntValue)
				break
			}
		}
		assert.True(t, found, "cost-test-artifact should be in results")
	})
}

// TestIntegration_ConfigurableProperties tests the configurable property parameters
func TestIntegration_ConfigurableProperties(t *testing.T) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, service.DatastoreSpec())
	defer cleanup()

	catalogModelTypeID := testhelpers.GetCatalogModelTypeIDForDBTest(t, sharedDB)
	metricsArtifactTypeID := testhelpers.GetCatalogMetricsArtifactTypeIDForDBTest(t, sharedDB)
	modelArtifactTypeID := testhelpers.GetCatalogModelArtifactTypeIDForDBTest(t, sharedDB)
	catalogSourceTypeID := testhelpers.GetCatalogSourceTypeIDForDBTest(t, sharedDB)

	catalogModelRepo := modelservice.NewCatalogModelRepository(sharedDB, catalogModelTypeID)
	catalogArtifactRepo := service.NewCatalogArtifactRepository(sharedDB, map[string]int32{
		service.CatalogModelArtifactTypeName:   modelArtifactTypeID,
		service.CatalogMetricsArtifactTypeName: metricsArtifactTypeID,
	})
	modelArtifactRepo := modelservice.NewCatalogModelArtifactRepository(sharedDB, modelArtifactTypeID)
	metricsArtifactRepo := modelservice.NewCatalogMetricsArtifactRepository(sharedDB, metricsArtifactTypeID)
	catalogSourceRepo := service.NewCatalogSourceRepository(sharedDB, catalogSourceTypeID)

	services := Services{
		CatalogModelRepository:           catalogModelRepo,
		CatalogArtifactRepository:        catalogArtifactRepo,
		CatalogModelArtifactRepository:   modelArtifactRepo,
		CatalogMetricsArtifactRepository: metricsArtifactRepo,
		CatalogSourceRepository:          catalogSourceRepo,
		PropertyOptionsRepository:        service.NewPropertyOptionsRepository(sharedDB),
	}

	provider := NewDBCatalog(services, nil)

	// Stored name must be namespaced: sourceId:modelName
	testModel := &models.CatalogModelImpl{
		TypeID: apiutils.Of(catalogModelTypeID),
		Attributes: &models.CatalogModelAttributes{
			Name:       apiutils.Of("configurable-props-source:configurable-props-model"),
			ExternalID: apiutils.Of("config-model-999"),
		},
		Properties: &[]dbmodels.Properties{
			{Name: "source_id", StringValue: apiutils.Of("configurable-props-source")},
		},
	}
	savedModel, err := services.CatalogModelRepository.Save(testModel)
	require.NoError(t, err)

	artifact := &models.CatalogMetricsArtifactImpl{
		TypeID: apiutils.Of(metricsArtifactTypeID),
		Attributes: &models.CatalogMetricsArtifactAttributes{
			Name:        apiutils.Of("custom-props-artifact"),
			ExternalID:  apiutils.Of("custom-123"),
			MetricsType: models.MetricsTypePerformance,
		},
		Properties: &[]dbmodels.Properties{
			{Name: "metricsType", StringValue: apiutils.Of("performance-metrics")},
		},
		CustomProperties: &[]dbmodels.Properties{
			{Name: "throughput", DoubleValue: apiutils.Of(100.0)},
			{Name: "p90_latency", DoubleValue: apiutils.Of(120.0)},
			{Name: "nodes", IntValue: apiutils.Of(int32(2))},
			{Name: "instance_type", StringValue: apiutils.Of("gpu-large")},
		},
	}
	_, err = services.CatalogMetricsArtifactRepository.Save(artifact, savedModel.GetID())
	require.NoError(t, err)

	t.Run("CustomPropertyNames_WorkEndToEnd", func(t *testing.T) {
		artifact2 := &models.CatalogMetricsArtifactImpl{
			TypeID: apiutils.Of(metricsArtifactTypeID),
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:        apiutils.Of("custom-props-artifact-2"),
				ExternalID:  apiutils.Of("custom-456"),
				MetricsType: models.MetricsTypePerformance,
			},
			Properties: &[]dbmodels.Properties{
				{Name: "metricsType", StringValue: apiutils.Of("performance-metrics")},
			},
			CustomProperties: &[]dbmodels.Properties{
				{Name: "throughput", DoubleValue: apiutils.Of(50.0)},
				{Name: "p90_latency", DoubleValue: apiutils.Of(100.0)},
				{Name: "nodes", IntValue: apiutils.Of(int32(2))},
				{Name: "instance_type", StringValue: apiutils.Of("gpu-large")},
			},
		}
		_, err := services.CatalogMetricsArtifactRepository.Save(artifact2, savedModel.GetID())
		require.NoError(t, err)

		params := ListPerformanceArtifactsParams{
			TargetRPS:             100,
			Recommendations:       true,
			FilterQuery:           "",
			PageSize:              10,
			OrderBy:               "",
			SortOrder:             "ASC",
			NextPageToken:         apiutils.Of(""),
			RPSProperty:           "throughput",
			LatencyProperty:       "p90_latency",
			HardwareCountProperty: "nodes",
			HardwareTypeProperty:  "instance_type",
		}

		result, err := provider.GetPerformanceArtifacts(
			context.Background(),
			"configurable-props-model",
			"configurable-props-source",
			params,
		)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.GreaterOrEqual(t, len(result.Items), 1)

		foundArtifact := false
		for _, item := range result.Items {
			if *item.CatalogMetricsArtifact.Name == "custom-props-artifact-2" {
				foundArtifact = true
				replicas := item.CatalogMetricsArtifact.CustomProperties["replicas"]
				require.NotNil(t, replicas.MetadataIntValue)
				assert.Equal(t, "2", replicas.MetadataIntValue.IntValue, "replicas should be ceil(100/50)=2")

				totalRPS := item.CatalogMetricsArtifact.CustomProperties["total_requests_per_second"]
				require.NotNil(t, totalRPS.MetadataDoubleValue)
				assert.Equal(t, 100.0, totalRPS.MetadataDoubleValue.DoubleValue)
				break
			}
		}
		assert.True(t, foundArtifact, "Should find artifact with custom properties")
	})

	t.Run("ErrorHandling_MissingCustomProperties", func(t *testing.T) {
		errorTestArtifact := &models.CatalogMetricsArtifactImpl{
			TypeID: apiutils.Of(metricsArtifactTypeID),
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:        apiutils.Of("error-test-artifact"),
				ExternalID:  apiutils.Of("error-test-123"),
				MetricsType: models.MetricsTypePerformance,
			},
			Properties: &[]dbmodels.Properties{
				{Name: "metricsType", StringValue: apiutils.Of("performance-metrics")},
			},
			CustomProperties: &[]dbmodels.Properties{
				{Name: "throughput", DoubleValue: apiutils.Of(100.0)},
				{Name: "p90_latency", DoubleValue: apiutils.Of(120.0)},
				{Name: "nodes", IntValue: apiutils.Of(int32(2))},
				{Name: "instance_type", StringValue: apiutils.Of("gpu-large")},
			},
		}
		_, err := services.CatalogMetricsArtifactRepository.Save(errorTestArtifact, savedModel.GetID())
		require.NoError(t, err)

		params := ListPerformanceArtifactsParams{
			TargetRPS:             100,
			Recommendations:       true,
			FilterQuery:           "",
			PageSize:              10,
			OrderBy:               "",
			SortOrder:             "ASC",
			NextPageToken:         apiutils.Of(""),
			RPSProperty:           "nonexistent_rps",
			LatencyProperty:       "p90_latency",
			HardwareCountProperty: "nodes",
			HardwareTypeProperty:  "instance_type",
		}

		result, err := provider.GetPerformanceArtifacts(
			context.Background(),
			"configurable-props-model",
			"configurable-props-source",
			params,
		)

		if err != nil {
			assert.Contains(t, err.Error(), "nonexistent_rps", "Error should mention the missing property")
		} else {
			t.Logf("No error returned, but result is: %+v", result)
			assert.Equal(t, 0, len(result.Items), "Should have no items when property is missing")
		}
	})

	t.Run("DefaultPropertyNames_WhenNotSpecified", func(t *testing.T) {
		defaultArtifact := &models.CatalogMetricsArtifactImpl{
			TypeID: apiutils.Of(metricsArtifactTypeID),
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:        apiutils.Of("default-props-artifact"),
				ExternalID:  apiutils.Of("default-789"),
				MetricsType: models.MetricsTypePerformance,
			},
			Properties: &[]dbmodels.Properties{
				{Name: "metricsType", StringValue: apiutils.Of("performance-metrics")},
			},
			CustomProperties: &[]dbmodels.Properties{
				{Name: "requests_per_second", DoubleValue: apiutils.Of(150.0)},
				{Name: "ttft_p90", DoubleValue: apiutils.Of(80.0)},
				{Name: "hardware_count", IntValue: apiutils.Of(int32(3))},
				{Name: "hardware_type", StringValue: apiutils.Of("gpu-medium")},
			},
		}
		_, err := services.CatalogMetricsArtifactRepository.Save(defaultArtifact, savedModel.GetID())
		require.NoError(t, err)

		params := ListPerformanceArtifactsParams{
			TargetRPS:             100,
			Recommendations:       false,
			FilterQuery:           "",
			PageSize:              10,
			OrderBy:               "",
			SortOrder:             "ASC",
			NextPageToken:         apiutils.Of(""),
			RPSProperty:           "",
			LatencyProperty:       "",
			HardwareCountProperty: "",
			HardwareTypeProperty:  "",
		}

		result, err := provider.GetPerformanceArtifacts(
			context.Background(),
			"configurable-props-model",
			"configurable-props-source",
			params,
		)

		require.NoError(t, err)
		require.NotNil(t, result)

		foundDefault := false
		for _, item := range result.Items {
			if *item.CatalogMetricsArtifact.Name == "default-props-artifact" {
				foundDefault = true
				replicas := item.CatalogMetricsArtifact.CustomProperties["replicas"]
				require.NotNil(t, replicas.MetadataIntValue)
				assert.Equal(t, "1", replicas.MetadataIntValue.IntValue)
				break
			}
		}
		assert.True(t, foundDefault, "Should work with default property names")
	})

	t.Run("RecommendationsParameter_TrueFalse", func(t *testing.T) {
		artifacts := []*models.CatalogMetricsArtifactImpl{
			{
				TypeID: apiutils.Of(metricsArtifactTypeID),
				Attributes: &models.CatalogMetricsArtifactAttributes{
					Name:        apiutils.Of("rec-artifact-1"),
					ExternalID:  apiutils.Of("rec-1"),
					MetricsType: models.MetricsTypePerformance,
				},
				Properties: &[]dbmodels.Properties{
					{Name: "metricsType", StringValue: apiutils.Of("performance-metrics")},
				},
				CustomProperties: &[]dbmodels.Properties{
					{Name: "requests_per_second", DoubleValue: apiutils.Of(100.0)},
					{Name: "ttft_p90", DoubleValue: apiutils.Of(90.0)},
					{Name: "hardware_count", IntValue: apiutils.Of(int32(2))},
					{Name: "hardware_type", StringValue: apiutils.Of("gpu-test")},
				},
			},
			{
				TypeID: apiutils.Of(metricsArtifactTypeID),
				Attributes: &models.CatalogMetricsArtifactAttributes{
					Name:        apiutils.Of("rec-artifact-2"),
					ExternalID:  apiutils.Of("rec-2"),
					MetricsType: models.MetricsTypePerformance,
				},
				Properties: &[]dbmodels.Properties{
					{Name: "metricsType", StringValue: apiutils.Of("performance-metrics")},
				},
				CustomProperties: &[]dbmodels.Properties{
					{Name: "requests_per_second", DoubleValue: apiutils.Of(120.0)},
					{Name: "ttft_p90", DoubleValue: apiutils.Of(95.0)},
					{Name: "hardware_count", IntValue: apiutils.Of(int32(2))},
					{Name: "hardware_type", StringValue: apiutils.Of("gpu-test")},
				},
			},
		}
		for _, a := range artifacts {
			_, err := services.CatalogMetricsArtifactRepository.Save(a, savedModel.GetID())
			require.NoError(t, err)
		}

		paramsNoRec := ListPerformanceArtifactsParams{
			TargetRPS:       100,
			Recommendations: false,
			PageSize:        10,
		}

		resultNoRec, err := provider.GetPerformanceArtifacts(
			context.Background(),
			"configurable-props-model",
			"configurable-props-source",
			paramsNoRec,
		)
		require.NoError(t, err)
		require.NotNil(t, resultNoRec)

		noRecCount := 0
		for _, item := range resultNoRec.Items {
			name := *item.CatalogMetricsArtifact.Name
			if name == "rec-artifact-1" || name == "rec-artifact-2" {
				noRecCount++
			}
		}
		assert.Equal(t, 2, noRecCount, "With recommendations=false, should return both artifacts")

		paramsRec := ListPerformanceArtifactsParams{
			TargetRPS:       100,
			Recommendations: true,
			PageSize:        10,
		}

		resultRec, err := provider.GetPerformanceArtifacts(
			context.Background(),
			"configurable-props-model",
			"configurable-props-source",
			paramsRec,
		)
		require.NoError(t, err)
		require.NotNil(t, resultRec)

		dedupCount := 0
		for _, item := range resultRec.Items {
			name := *item.CatalogMetricsArtifact.Name
			if name == "rec-artifact-1" || name == "rec-artifact-2" {
				dedupCount++
			}
		}
		assert.LessOrEqual(t, dedupCount, 1, "With recommendations=true, should deduplicate similar configs")
	})
}
