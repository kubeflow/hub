package modelcatalog

import (
	"context"
	"fmt"
	"testing"

	"github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	modelservice "github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/service"
	"github.com/kubeflow/hub/catalog/internal/db/service"
	"github.com/kubeflow/hub/catalog/internal/testhelpers"
	"github.com/kubeflow/hub/internal/platform/apiutils"
	mr_models "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"github.com/kubeflow/hub/internal/testutils"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupIntegrationTestProvider(t *testing.T, ctx context.Context, sharedDB *gorm.DB) *dbCatalogImpl {
	// Get type IDs
	catalogModelTypeID := testhelpers.GetCatalogModelTypeIDForDBTest(t, sharedDB)
	modelArtifactTypeID := testhelpers.GetCatalogModelArtifactTypeIDForDBTest(t, sharedDB)
	metricsArtifactTypeID := testhelpers.GetCatalogMetricsArtifactTypeIDForDBTest(t, sharedDB)
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

	svcs := Services{
		CatalogModelRepository:           catalogModelRepo,
		CatalogArtifactRepository:        catalogArtifactRepo,
		CatalogModelArtifactRepository:   modelArtifactRepo,
		CatalogMetricsArtifactRepository: metricsArtifactRepo,
		CatalogSourceRepository:          catalogSourceRepo,
		PropertyOptionsRepository:        service.NewPropertyOptionsRepository(sharedDB),
	}

	// Insert test data:
	//   - Models: "fast-model", "medium-model", "slow-model", "no-perf-model"
	//   - Performance artifacts with varying ttft_p90, hardware_count, hardware_type
	//   - Ensure Pareto filtering will produce predictable order
	insertTestData(t, ctx, svcs, catalogModelTypeID, metricsArtifactTypeID)

	// Return configured provider
	return NewDBCatalog(svcs, nil).(*dbCatalogImpl)
}

func insertTestData(t *testing.T, ctx context.Context, svcs Services, catalogModelTypeID, metricsArtifactTypeID int32) {
	// Create test models
	testModels := []struct {
		name        string
		sourceID    string
		description string
	}{
		{"fast-model", "test-source", "Fast model with low latency"},
		{"medium-model", "test-source", "Medium speed model"},
		{"slow-model", "test-source", "Slow model with high latency"},
		{"no-perf-model", "test-source", "Model without performance data"},
	}

	var modelIDs []int32
	for _, modelData := range testModels {
		model := &models.CatalogModelImpl{
			TypeID: apiutils.Of(catalogModelTypeID),
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of(modelData.name),
				ExternalID: apiutils.Of(modelData.name + "-ext"),
			},
			Properties: &[]mr_models.Properties{
				{Name: "source_id", StringValue: apiutils.Of(modelData.sourceID)},
				{Name: "description", StringValue: apiutils.Of(modelData.description)},
			},
		}

		savedModel, err := svcs.CatalogModelRepository.Save(model)
		require.NoError(t, err)
		modelIDs = append(modelIDs, *savedModel.GetID())
	}

	// Create performance artifacts with predictable latency order
	performanceArtifacts := []struct {
		modelIdx              int
		name                  string
		ttft_p90              float64
		custom_latency_metric float64
		requests_per_second   float64
		hardware_count        int32
		hardware_type         string
	}{
		// Fast model - should be first in sorted results
		{0, "fast-model-perf-1", 50.0, 45.0, 200.0, 1, "gpu-a100"},
		{0, "fast-model-perf-2", 55.0, 48.0, 180.0, 2, "gpu-a100"},

		// Medium model - should be second
		{1, "medium-model-perf-1", 100.0, 95.0, 150.0, 1, "gpu-v100"},
		{1, "medium-model-perf-2", 110.0, 105.0, 120.0, 2, "gpu-v100"},

		// Slow model - should be last among models with perf data
		{2, "slow-model-perf-1", 200.0, 195.0, 100.0, 1, "gpu-t4"},
		{2, "slow-model-perf-2", 220.0, 210.0, 80.0, 2, "gpu-t4"},

		// No performance artifacts for no-perf-model (index 3)
	}

	for _, perfData := range performanceArtifacts {
		artifact := &models.CatalogMetricsArtifactImpl{
			TypeID: apiutils.Of(metricsArtifactTypeID),
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:        apiutils.Of(perfData.name),
				MetricsType: models.MetricsTypePerformance,
			},
			Properties: &[]mr_models.Properties{},
			CustomProperties: &[]mr_models.Properties{
				{Name: "ttft_p90", DoubleValue: apiutils.Of(perfData.ttft_p90)},
				{Name: "custom_latency_metric", DoubleValue: apiutils.Of(perfData.custom_latency_metric)},
				{Name: "requests_per_second", DoubleValue: apiutils.Of(perfData.requests_per_second)},
				{Name: "hardware_count", IntValue: apiutils.Of(perfData.hardware_count)},
				{Name: "hardware_type", StringValue: apiutils.Of(perfData.hardware_type)},
			},
		}

		// Save with parent model relationship
		parentResourceID := modelIDs[perfData.modelIdx]
		_, err := svcs.CatalogMetricsArtifactRepository.Save(artifact, &parentResourceID)
		require.NoError(t, err)
	}
}

func BenchmarkRecommendedLatencySorting(b *testing.B) {
	// Setup test database - convert to testing.T for compatibility
	t := &testing.T{}
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testhelpers.MustDatastoreSpec(t))
	defer cleanup()

	ctx := context.Background()
	provider := setupBenchmarkProvider(b, ctx, sharedDB) // Setup with 100+ models

	paretoParams := ParetoFilteringParams{
		LatencyProperty: "ttft_p90",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := provider.FindModelsWithRecommendedLatency(ctx, mr_models.Pagination{
			PageSize: apiutils.Of(int32(20)),
		}, paretoParams, []string{"benchmark-source"}, "")

		require.NoError(b, err)
	}
}

func setupBenchmarkProvider(b *testing.B, ctx context.Context, sharedDB *gorm.DB) *dbCatalogImpl {
	// Get type IDs using a wrapper to convert *testing.B to *testing.T interface
	catalogModelTypeID := getCatalogModelTypeIDFromDB(sharedDB)
	modelArtifactTypeID := getCatalogModelArtifactTypeIDFromDB(sharedDB)
	metricsArtifactTypeID := getCatalogMetricsArtifactTypeIDFromDB(sharedDB)
	catalogSourceTypeID := getCatalogSourceTypeIDFromDB(sharedDB)

	// Create repositories
	catalogModelRepo := modelservice.NewCatalogModelRepository(sharedDB, catalogModelTypeID)
	catalogArtifactRepo := service.NewCatalogArtifactRepository(sharedDB, map[string]int32{
		service.CatalogModelArtifactTypeName:   modelArtifactTypeID,
		service.CatalogMetricsArtifactTypeName: metricsArtifactTypeID,
	})
	modelArtifactRepo := modelservice.NewCatalogModelArtifactRepository(sharedDB, modelArtifactTypeID)
	metricsArtifactRepo := modelservice.NewCatalogMetricsArtifactRepository(sharedDB, metricsArtifactTypeID)
	catalogSourceRepo := service.NewCatalogSourceRepository(sharedDB, catalogSourceTypeID)

	svcs := Services{
		CatalogModelRepository:           catalogModelRepo,
		CatalogArtifactRepository:        catalogArtifactRepo,
		CatalogModelArtifactRepository:   modelArtifactRepo,
		CatalogMetricsArtifactRepository: metricsArtifactRepo,
		CatalogSourceRepository:          catalogSourceRepo,
		PropertyOptionsRepository:        service.NewPropertyOptionsRepository(sharedDB),
	}

	// Insert 100+ models with performance data for benchmarking
	insertBenchmarkData(b, ctx, svcs, catalogModelTypeID, metricsArtifactTypeID)

	return NewDBCatalog(svcs, nil).(*dbCatalogImpl)
}

func insertBenchmarkData(b *testing.B, ctx context.Context, svcs Services, catalogModelTypeID, metricsArtifactTypeID int32) {
	const numModels = 100

	var modelIDs []int32

	// Create 100 test models
	for i := range numModels {
		model := &models.CatalogModelImpl{
			TypeID: apiutils.Of(catalogModelTypeID),
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of(fmt.Sprintf("benchmark-model-%d", i)),
				ExternalID: apiutils.Of(fmt.Sprintf("benchmark-model-%d-ext", i)),
			},
			Properties: &[]mr_models.Properties{
				{Name: "source_id", StringValue: apiutils.Of("benchmark-source")},
				{Name: "description", StringValue: apiutils.Of(fmt.Sprintf("Benchmark model %d", i))},
			},
		}

		savedModel, err := svcs.CatalogModelRepository.Save(model)
		require.NoError(b, err)
		modelIDs = append(modelIDs, *savedModel.GetID())
	}

	// Create 2-3 performance artifacts per model with varying latencies
	for i, modelID := range modelIDs {
		for j := 0; j < 2+i%2; j++ { // 2-3 artifacts per model
			latency := 50.0 + float64(i*10+j*5) // Varying latencies
			rps := 200.0 - float64(i*2)         // Varying RPS

			artifact := &models.CatalogMetricsArtifactImpl{
				TypeID: apiutils.Of(metricsArtifactTypeID),
				Attributes: &models.CatalogMetricsArtifactAttributes{
					Name:        apiutils.Of(fmt.Sprintf("benchmark-perf-%d-%d", i, j)),
					MetricsType: models.MetricsTypePerformance,
				},
				Properties: &[]mr_models.Properties{},
				CustomProperties: &[]mr_models.Properties{
					{Name: "ttft_p90", DoubleValue: apiutils.Of(latency)},
					{Name: "requests_per_second", DoubleValue: apiutils.Of(rps)},
					{Name: "hardware_count", IntValue: apiutils.Of(int32(1 + i%4))},
					{Name: "hardware_type", StringValue: apiutils.Of([]string{"gpu-a100", "gpu-v100", "gpu-t4"}[i%3])},
				},
			}

			_, err := svcs.CatalogMetricsArtifactRepository.Save(artifact, &modelID)
			require.NoError(b, err)
		}
	}
}

// Helper functions to get type IDs using existing functions from db_catalog_test.go
// These are wrapper functions for benchmark tests that don't have testing.T

func getCatalogModelTypeIDFromDB(db *gorm.DB) int32 {
	var typeRecord schema.Type
	err := db.Where("name = ?", service.CatalogModelTypeName).First(&typeRecord).Error
	if err != nil {
		panic(fmt.Sprintf("Failed to query CatalogModel type: %v", err))
	}
	return typeRecord.ID
}

func getCatalogModelArtifactTypeIDFromDB(db *gorm.DB) int32 {
	var typeRecord schema.Type
	err := db.Where("name = ?", service.CatalogModelArtifactTypeName).First(&typeRecord).Error
	if err != nil {
		panic(fmt.Sprintf("Failed to query CatalogModelArtifact type: %v", err))
	}
	return typeRecord.ID
}

func getCatalogMetricsArtifactTypeIDFromDB(db *gorm.DB) int32 {
	var typeRecord schema.Type
	err := db.Where("name = ?", service.CatalogMetricsArtifactTypeName).First(&typeRecord).Error
	if err != nil {
		panic(fmt.Sprintf("Failed to query CatalogMetricsArtifact type: %v", err))
	}
	return typeRecord.ID
}

func getCatalogSourceTypeIDFromDB(db *gorm.DB) int32 {
	var typeRecord schema.Type
	err := db.Where("name = ?", service.CatalogSourceTypeName).First(&typeRecord).Error
	if err != nil {
		panic(fmt.Sprintf("Failed to query CatalogSource type: %v", err))
	}
	return typeRecord.ID
}
