package modelcatalog

import (
	"context"
	"fmt"
	"math"
	"testing"

	"github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	sharedmodels "github.com/kubeflow/hub/catalog/internal/db/models"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockPerfArtifactRepo is a testify mock implementation of CatalogArtifactRepository for performance tests
type mockPerfArtifactRepo struct {
	mock.Mock
}

// mockPerfModelRepo is a testify mock implementation of CatalogModelRepository for performance tests
type mockPerfModelRepo struct {
	mock.Mock
	TypeID int32 // Set to a non-zero value when testing GetFilterOptions scoping
}

func (m *mockPerfModelRepo) GetByName(name string) (models.CatalogModel, error) {
	args := m.Called(name)
	return args.Get(0).(models.CatalogModel), args.Error(1)
}

func (m *mockPerfModelRepo) GetByID(id int32) (models.CatalogModel, error) {
	return nil, nil
}

func (m *mockPerfModelRepo) List(opts models.CatalogModelListOptions) (*dbmodels.ListWrapper[models.CatalogModel], error) {
	return nil, nil
}

func (m *mockPerfModelRepo) Save(model models.CatalogModel) (models.CatalogModel, error) {
	return nil, nil
}

func (m *mockPerfModelRepo) DeleteBySource(sourceID string) error {
	return nil
}

func (m *mockPerfModelRepo) DeleteByID(id int32) error {
	return nil
}

func (m *mockPerfModelRepo) GetDistinctSourceIDs() ([]string, error) {
	return nil, nil
}

func (m *mockPerfModelRepo) GetTypeID() int32 {
	return m.TypeID
}

func (m *mockPerfArtifactRepo) GetByID(id int32) (sharedmodels.CatalogArtifact, error) {
	args := m.Called(id)
	return args.Get(0).(sharedmodels.CatalogArtifact), args.Error(1)
}

func (m *mockPerfArtifactRepo) List(listOptions sharedmodels.CatalogArtifactListOptions) (*dbmodels.ListWrapper[sharedmodels.CatalogArtifact], error) {
	args := m.Called(listOptions)
	return args.Get(0).(*dbmodels.ListWrapper[sharedmodels.CatalogArtifact]), args.Error(1)
}

func (m *mockPerfArtifactRepo) DeleteByParentID(artifactType string, parentResourceID int32) error {
	args := m.Called(artifactType, parentResourceID)
	return args.Error(0)
}

func (m *mockPerfArtifactRepo) CountByParentIDs(parentIDs []int32) (map[int32]map[string]int32, error) {
	return make(map[int32]map[string]int32), nil
}

func TestPerformanceArtifactService_GetArtifacts(t *testing.T) {
	// Mock repository for testing
	mockArtifactRepo := &mockPerfArtifactRepo{}
	service := NewPerformanceArtifactService(mockArtifactRepo, nil)

	// Setup mock to return test artifacts with performance metrics
	id := int32(1)
	testArtifacts := []sharedmodels.CatalogArtifact{
		{
			CatalogMetricsArtifact: &dbmodels.BaseEntity[models.CatalogMetricsArtifactAttributes]{
				Attributes: &models.CatalogMetricsArtifactAttributes{
					Name:        new("test-perf-artifact"),
					MetricsType: models.MetricsTypePerformance,
				},
				Properties: &[]dbmodels.Properties{},
				CustomProperties: &[]dbmodels.Properties{
					{Name: "requests_per_second", DoubleValue: new(200.0)},
				},
			},
		},
	}
	// Set ID for the artifact
	testArtifacts[0].CatalogMetricsArtifact.SetID(id)

	mockArtifactRepo.On("List", mock.AnythingOfType("models.CatalogArtifactListOptions")).
		Return(&dbmodels.ListWrapper[sharedmodels.CatalogArtifact]{
			Items: testArtifacts,
		}, nil)

	params := PerformanceArtifactParams{
		ModelID:         123,
		TargetRPS:       300,
		Recommendations: false,
		PageSize:        10,
	}

	result, err := service.GetArtifacts(params)

	require.NoError(t, err)
	require.Len(t, result.Items, 1)

	// Verify repository was called with correct performance filtering
	mockArtifactRepo.AssertCalled(t, "List", mock.MatchedBy(func(opts sharedmodels.CatalogArtifactListOptions) bool {
		return opts.ParentResourceID != nil && *opts.ParentResourceID == 123
	}))
}

func TestPerformanceArtifactService_ProcessWithTargetRPSAndRecommendataion(t *testing.T) {
	mockArtifactRepo := &mockPerfArtifactRepo{}
	service := NewPerformanceArtifactService(mockArtifactRepo, nil)

	// Mock repository to return test artifacts with performance data
	id := int32(1)
	testDBMetrics := &dbmodels.BaseEntity[models.CatalogMetricsArtifactAttributes]{
		Attributes: &models.CatalogMetricsArtifactAttributes{
			Name:        new("test-perf-artifact"),
			MetricsType: models.MetricsTypePerformance,
		},
		Properties: &[]dbmodels.Properties{},
		CustomProperties: &[]dbmodels.Properties{
			{Name: "requests_per_second", DoubleValue: new(200.0)},
			{Name: "ttft_p90", DoubleValue: new(50.0)},
			{Name: "estimated_cost_per_hour", DoubleValue: new(80.0)},
			{Name: "hardware_type", StringValue: new("gpu-a100")},
		},
	}
	testDBMetrics.SetID(id)

	testDBResult := &dbmodels.ListWrapper[sharedmodels.CatalogArtifact]{
		Items: []sharedmodels.CatalogArtifact{
			{CatalogMetricsArtifact: testDBMetrics},
		},
	}

	mockArtifactRepo.On("List", mock.Anything).Return(testDBResult, nil)

	params := PerformanceArtifactParams{
		ModelID:         123,
		TargetRPS:       600, // Should result in 3 replicas (600/200)
		Recommendations: true,
		PageSize:        10,
	}

	result, err := service.GetArtifacts(params)

	require.NoError(t, err)
	require.Len(t, result.Items, 1)

	artifact := result.Items[0]

	// Check targetRPS calculations were applied
	customProps := artifact.GetCustomProperties()
	require.NotNil(t, customProps)

	// Helper function to find property by name
	findProperty := func(name string) *dbmodels.Properties {
		for _, prop := range *customProps {
			if prop.Name == name {
				return &prop
			}
		}
		return nil
	}

	replicasProp := findProperty("replicas")
	require.NotNil(t, replicasProp)
	require.NotNil(t, replicasProp.IntValue)
	require.Equal(t, int32(3), *replicasProp.IntValue)

	totalRPSProp := findProperty("total_requests_per_second")
	require.NotNil(t, totalRPSProp)
	require.NotNil(t, totalRPSProp.DoubleValue)
	require.Equal(t, 600.0, *totalRPSProp.DoubleValue)
}

func TestPerformanceArtifactParamsStructure(t *testing.T) {
	params := PerformanceArtifactParams{
		ModelID:               1,
		TargetRPS:             100,
		Recommendations:       true,
		RPSProperty:           "throughput",
		LatencyProperty:       "p90_latency",
		HardwareCountProperty: "nodes",
		HardwareTypeProperty:  "instance_type",
	}

	require.Equal(t, "throughput", params.RPSProperty)
	require.Equal(t, "p90_latency", params.LatencyProperty)
	require.Equal(t, "nodes", params.HardwareCountProperty)
	require.Equal(t, "instance_type", params.HardwareTypeProperty)
}

func TestPerformanceArtifactService_Recommendataions(t *testing.T) {
	service := NewPerformanceArtifactService(nil, nil) // No repo needed for this test

	id1 := int32(1)
	id2 := int32(2)
	artifacts := []sharedmodels.CatalogMetricsArtifact{
		&dbmodels.BaseEntity[models.CatalogMetricsArtifactAttributes]{
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name: new("fast-expensive"),
			},
			CustomProperties: &[]dbmodels.Properties{
				dbmodels.NewStringProperty("hardware_type", "gpu-a100", true),
				dbmodels.NewDoubleProperty("ttft_p90", 50.0, true),
				dbmodels.NewIntProperty("hardware_count", 2, true),
				dbmodels.NewIntProperty("replicas", 3, true),
			},
		},
		&dbmodels.BaseEntity[models.CatalogMetricsArtifactAttributes]{
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name: new("slow-cheap"),
			},
			CustomProperties: &[]dbmodels.Properties{
				dbmodels.NewStringProperty("hardware_type", "gpu-a100", true),
				dbmodels.NewDoubleProperty("ttft_p90", 150.0, true),
				dbmodels.NewIntProperty("hardware_count", 1, true),
				dbmodels.NewIntProperty("replicas", 2, true),
			},
		},
	}

	// Set IDs for the artifacts
	artifacts[0].SetID(id1)
	artifacts[1].SetID(id2)

	result := service.generateRecommended(artifacts, "ttft_p90", "hardware_count", "hardware_type")

	// Should apply the two-pass filtering with epsilon thresholds
	require.True(t, len(result) <= len(artifacts))
}

// TestPropertyValidation tests the validateCustomProperties method
func TestPropertyValidation(t *testing.T) {
	service := NewPerformanceArtifactService(nil, nil)

	// Test case: empty artifacts should pass validation (not an error condition)
	err := service.validateCustomProperties([]sharedmodels.CatalogMetricsArtifact{}, "rps", "latency", "hw_count", "hw_type")
	require.NoError(t, err, "Empty artifacts should not cause validation error")

	// Test case: property exists in some artifacts
	id1 := int32(1)
	id2 := int32(2)
	artifacts := []sharedmodels.CatalogMetricsArtifact{
		// artifact with custom property
		&dbmodels.BaseEntity[models.CatalogMetricsArtifactAttributes]{
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name: new("artifact-with-prop"),
			},
			CustomProperties: &[]dbmodels.Properties{
				dbmodels.NewDoubleProperty("custom_rps", 100.0, true),
				dbmodels.NewDoubleProperty("custom_latency", 50.0, true),
				dbmodels.NewIntProperty("custom_hw_count", 2, true),
				dbmodels.NewStringProperty("custom_hw_type", "gpu", true),
			},
		},
		// artifact without custom property
		&dbmodels.BaseEntity[models.CatalogMetricsArtifactAttributes]{
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name: new("artifact-without-prop"),
			},
			CustomProperties: &[]dbmodels.Properties{
				dbmodels.NewDoubleProperty("other_prop", 200.0, true),
			},
		},
	}
	artifacts[0].SetID(id1)
	artifacts[1].SetID(id2)

	// Should not error when property exists in at least one artifact
	err = service.validateCustomProperties(artifacts, "custom_rps", "custom_latency", "custom_hw_count", "custom_hw_type")
	require.NoError(t, err)

	// Test case: property doesn't exist in any artifact
	err = service.validateCustomProperties(artifacts, "nonexistent_prop", "", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "nonexistent_prop")
	require.Contains(t, err.Error(), "not found")

	// Test case: multiple properties missing
	err = service.validateCustomProperties(artifacts, "missing1", "missing2", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")

	// Test case: empty property names should be ignored
	err = service.validateCustomProperties(artifacts, "", "", "", "")
	require.NoError(t, err)
}

// TestGetArtifactsWithValidation tests that GetArtifacts validates custom properties
func TestGetArtifactsWithValidation(t *testing.T) {
	mockArtifactRepo := &mockPerfArtifactRepo{}
	service := NewPerformanceArtifactService(mockArtifactRepo, nil)

	id1 := int32(1)
	testArtifacts := []sharedmodels.CatalogArtifact{
		{
			CatalogMetricsArtifact: &dbmodels.BaseEntity[models.CatalogMetricsArtifactAttributes]{
				Attributes: &models.CatalogMetricsArtifactAttributes{
					Name:        new("test-artifact"),
					MetricsType: models.MetricsTypePerformance,
				},
				CustomProperties: &[]dbmodels.Properties{
					dbmodels.NewDoubleProperty("custom_rps", 100.0, true),
					dbmodels.NewDoubleProperty("custom_latency", 50.0, true),
				},
			},
		},
	}
	testArtifacts[0].CatalogMetricsArtifact.SetID(id1)

	mockArtifactRepo.On("List", mock.AnythingOfType("models.CatalogArtifactListOptions")).
		Return(&dbmodels.ListWrapper[sharedmodels.CatalogArtifact]{Items: testArtifacts}, nil)

	// Test with valid custom properties
	params := PerformanceArtifactParams{
		ModelID:         123,
		TargetRPS:       100,
		Recommendations: false,
		PageSize:        10,
		RPSProperty:     "custom_rps",
		LatencyProperty: "custom_latency",
	}

	result, err := service.GetArtifacts(params)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Test with missing custom property
	params.RPSProperty = "nonexistent_property"
	result, err = service.GetArtifacts(params)
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "invalid custom properties")
	require.Contains(t, err.Error(), "nonexistent_property")
}

// TestConfigurablePropertyUsage tests that custom property names are used in calculations
func TestConfigurablePropertyUsage(t *testing.T) {
	service := NewPerformanceArtifactService(nil, nil) // No repo needed for this test

	// Create test artifact with custom property names
	id := int32(1)
	artifact := &dbmodels.BaseEntity[models.CatalogMetricsArtifactAttributes]{
		Attributes: &models.CatalogMetricsArtifactAttributes{
			Name: new("test-artifact"),
		},
		CustomProperties: &[]dbmodels.Properties{
			dbmodels.NewDoubleProperty("throughput", 50.0, true),
			dbmodels.NewDoubleProperty("p90_latency", 120.0, true),
			dbmodels.NewIntProperty("nodes", 2, true),
			dbmodels.NewStringProperty("instance_type", "gpu-large", true),
		},
	}
	artifact.SetID(id)

	artifacts := []sharedmodels.CatalogMetricsArtifact{artifact}

	params := PerformanceArtifactParams{
		TargetRPS:             100,
		Recommendations:       true,
		RPSProperty:           "throughput",
		LatencyProperty:       "p90_latency",
		HardwareCountProperty: "nodes",
		HardwareTypeProperty:  "instance_type",
	}

	result := service.processArtifacts(artifacts, params)

	// Should calculate replicas based on custom throughput property
	// 100 targetRPS / 50 throughput = 2 replicas
	replicas := service.extractCustomPropertiesIntValue(result[0].GetCustomProperties(), "replicas", 0)
	require.Equal(t, int32(2), replicas)

	// Should calculate total RPS correctly
	totalRPS := service.extractCustomPropertiesDoubleValue(result[0].GetCustomProperties(), "total_requests_per_second", 0)
	require.Equal(t, 100.0, totalRPS)
}

// TestConfigurableRecommendataion tests deduplication with configurable property names
func TestConfigurableRecommendataion(t *testing.T) {
	service := NewPerformanceArtifactService(nil, nil)

	id1 := int32(1)
	id2 := int32(2)
	id3 := int32(3)
	artifacts := []sharedmodels.CatalogMetricsArtifact{
		// Dominated artifact: slower AND more expensive
		&dbmodels.BaseEntity[models.CatalogMetricsArtifactAttributes]{
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name: new("artifact-slow-expensive"),
			},
			CustomProperties: &[]dbmodels.Properties{
				dbmodels.NewStringProperty("instance_type", "gpu-small", true),
				dbmodels.NewDoubleProperty("p90_latency", 150.0, true),
				dbmodels.NewIntProperty("nodes", 3, true),
				dbmodels.NewIntProperty("replicas", 1, true),
			},
		},
		// Fast but expensive
		&dbmodels.BaseEntity[models.CatalogMetricsArtifactAttributes]{
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name: new("artifact-fast-expensive"),
			},
			CustomProperties: &[]dbmodels.Properties{
				dbmodels.NewStringProperty("instance_type", "gpu-small", true),
				dbmodels.NewDoubleProperty("p90_latency", 100.0, true),
				dbmodels.NewIntProperty("nodes", 2, true),
				dbmodels.NewIntProperty("replicas", 1, true),
			},
		},
		// Cheapest option (but slower than artifact-fast-expensive)
		&dbmodels.BaseEntity[models.CatalogMetricsArtifactAttributes]{
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name: new("artifact-cheap"),
			},
			CustomProperties: &[]dbmodels.Properties{
				dbmodels.NewStringProperty("instance_type", "gpu-small", true),
				dbmodels.NewDoubleProperty("p90_latency", 120.0, true),
				dbmodels.NewIntProperty("nodes", 1, true),
				dbmodels.NewIntProperty("replicas", 1, true),
			},
		},
	}

	artifacts[0].SetID(id1)
	artifacts[1].SetID(id2)
	artifacts[2].SetID(id3)

	result := service.generateRecommended(artifacts, "p90_latency", "nodes", "instance_type")

	// The dominated artifact (slow and expensive) should be filtered out
	require.Less(t, len(result), len(artifacts))

	// Verify that the function uses configurable property names by checking the results
	// The algorithm should filter based on the custom property names provided
	for _, artifact := range result {
		latency := service.extractCustomPropertiesDoubleValue(artifact.GetCustomProperties(), "p90_latency", 0)
		nodes := service.extractCustomPropertiesIntValue(artifact.GetCustomProperties(), "nodes", 0)
		instanceType := service.extractCustomPropertiesStringValue(artifact.GetCustomProperties(), "instance_type")

		// Verify we can extract values using custom property names
		require.Greater(t, latency, 0.0)
		require.Greater(t, nodes, int32(0))
		require.Equal(t, "gpu-small", instanceType)
	}
}

func TestPerformanceArtifactService_RecommendationData(t *testing.T) {
	tests := []struct {
		name         string
		targetRPS    int32
		inputRows    []testArtifactRow
		expectedKept []int // indices of rows that should be kept after filtering
	}{
		{
			name:      "known example",
			targetRPS: 10,
			inputRows: []testArtifactRow{
				{hardwareType: "A100-80", hardwareCount: 4, requestsPerSecond: 10.0 / 5.0, latency: 75.0},
				{hardwareType: "A100-80", hardwareCount: 4, requestsPerSecond: 12.0 / 4.0, latency: 76.0},
				{hardwareType: "A100-80", hardwareCount: 4, requestsPerSecond: 12.0 / 3.0, latency: 77.0},
				{hardwareType: "A100-80", hardwareCount: 4, requestsPerSecond: 10.0 / 2.0, latency: 80.0}, // Keep
				{hardwareType: "A100-80", hardwareCount: 4, requestsPerSecond: 12.0 / 2.0, latency: 86.0},
				{hardwareType: "A100-80", hardwareCount: 2, requestsPerSecond: 10.0 / 10.0, latency: 112.0},
				{hardwareType: "A100-80", hardwareCount: 2, requestsPerSecond: 10.0 / 5.0, latency: 113.0},
				{hardwareType: "A100-80", hardwareCount: 2, requestsPerSecond: 12.0 / 4.0, latency: 117.0},
				{hardwareType: "A100-80", hardwareCount: 2, requestsPerSecond: 12.0 / 3.0, latency: 128.0},
				{hardwareType: "A100-80", hardwareCount: 2, requestsPerSecond: 10.0 / 2.0, latency: 129.0}, // Keep
				{hardwareType: "A100-80", hardwareCount: 2, requestsPerSecond: 12.0 / 2.0, latency: 141.0},
				{hardwareType: "A100-80", hardwareCount: 2, requestsPerSecond: 14.0 / 2.0, latency: 176.0},
				{hardwareType: "A100-80", hardwareCount: 1, requestsPerSecond: 10.0 / 10.0, latency: 187.0},
				{hardwareType: "A100-80", hardwareCount: 1, requestsPerSecond: 10.0 / 5.0, latency: 198.0},
				{hardwareType: "A100-80", hardwareCount: 1, requestsPerSecond: 12.0 / 4.0, latency: 205.0},
				{hardwareType: "A100-80", hardwareCount: 2, requestsPerSecond: 10.0 / 10.0, latency: 748.0},
			},
			expectedKept: []int{9, 3},
		},
		{
			name:      "same latency, same cost, different hardware count",
			targetRPS: 8,
			inputRows: []testArtifactRow{
				{hardwareType: "A100-80", hardwareCount: 4, latency: 80.0, requestsPerSecond: 4}, // 2 replicas, cost is 8
				{hardwareType: "A100-80", hardwareCount: 2, latency: 80.0, requestsPerSecond: 2}, // 4 replicas, cost is 8
			},
			expectedKept: []int{1}, // Only second row should be kept (lower cost, same latency)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewPerformanceArtifactService(nil, nil)

			// Convert test rows to full artifacts
			artifacts := createArtifactsFromRows(tt.inputRows, tt.targetRPS)

			// Apply Pareto filtering
			result := service.generateRecommended(artifacts, "ttft_p90", "hardware_count", "hardware_type")

			// Verify expected artifacts remain
			verifyExpectedArtifacts(t, artifacts, result, tt.expectedKept)
		})
	}
}

type testArtifactRow struct {
	hardwareType      string
	hardwareCount     int32
	latency           float64
	requestsPerSecond float64
}

// createArtifactsFromRows converts simple test data into full CatalogMetricsArtifact objects
func createArtifactsFromRows(rows []testArtifactRow, targetRPS int32) []sharedmodels.CatalogMetricsArtifact {
	artifacts := make([]sharedmodels.CatalogMetricsArtifact, len(rows))

	for i, row := range rows {
		id := int32(i + 1)

		if row.requestsPerSecond < 1 {
			row.requestsPerSecond = 1
		}
		replicas := int32(math.Ceil(float64(targetRPS) / float64(row.requestsPerSecond)))

		artifact := &dbmodels.BaseEntity[models.CatalogMetricsArtifactAttributes]{
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name: new(fmt.Sprintf("artifact-%d", i)),
			},
			CustomProperties: &[]dbmodels.Properties{
				dbmodels.NewStringProperty("hardware_type", row.hardwareType, true),
				dbmodels.NewIntProperty("hardware_count", row.hardwareCount, true),
				dbmodels.NewDoubleProperty("ttft_p90", row.latency, true),
				dbmodels.NewDoubleProperty("requests_per_second", row.requestsPerSecond, true),
				dbmodels.NewIntProperty("replicas", replicas, true),
			},
		}
		artifact.SetID(id)
		artifacts[i] = artifact
	}

	return artifacts
}

// verifyExpectedArtifacts checks that only the expected artifacts remain after filtering
func verifyExpectedArtifacts(t *testing.T, originalArtifacts []sharedmodels.CatalogMetricsArtifact, filteredResult []sharedmodels.CatalogMetricsArtifact, expectedIndices []int) {
	require.Len(t, filteredResult, len(expectedIndices), "Unexpected number of artifacts after filtering")

	// Create a set of expected IDs
	expectedIDs := make(map[int32]bool)
	for _, idx := range expectedIndices {
		id := originalArtifacts[idx].GetID()
		require.NotNil(t, id, "Original artifact should have ID")
		expectedIDs[*id] = true
	}

	// Verify all filtered results are in the expected set
	for _, artifact := range filteredResult {
		id := artifact.GetID()
		require.NotNil(t, id, "Filtered artifact should have ID")
		require.True(t, expectedIDs[*id], "Unexpected artifact ID %d in filtered results", *id)
	}
}

func TestGetMinimumRecommendedLatency(t *testing.T) {
	// Mock repositories for testing
	mockArtifactRepo := &mockPerfArtifactRepo{}
	mockModelRepo := &mockPerfModelRepo{}

	// Setup mock model repository to return a model with ID
	testModelID := int32(42)
	mockModelRepo.On("GetByName", "test-model").Return(&dbmodels.BaseEntity[models.CatalogModelAttributes]{
		ID: new(testModelID),
	}, nil)

	service := NewPerformanceArtifactService(mockArtifactRepo, mockModelRepo)

	// Setup mock to return test artifacts with performance data
	id := int32(1)
	testArtifacts := []sharedmodels.CatalogArtifact{
		{
			CatalogMetricsArtifact: &dbmodels.BaseEntity[models.CatalogMetricsArtifactAttributes]{
				Attributes: &models.CatalogMetricsArtifactAttributes{
					Name:        new("test-perf-artifact"),
					MetricsType: models.MetricsTypePerformance,
				},
				Properties: &[]dbmodels.Properties{},
				CustomProperties: &[]dbmodels.Properties{
					{Name: "ttft_p90", DoubleValue: new(150.0)},
					{Name: "requests_per_second", DoubleValue: new(200.0)},
					{Name: "hardware_count", IntValue: new(int32(2))},
					{Name: "hardware_type", StringValue: new("gpu-a100")},
				},
			},
		},
	}
	// Set ID for the artifact
	testArtifacts[0].CatalogMetricsArtifact.SetID(id)

	mockArtifactRepo.On("List", mock.AnythingOfType("models.CatalogArtifactListOptions")).
		Return(&dbmodels.ListWrapper[sharedmodels.CatalogArtifact]{
			Items: testArtifacts,
		}, nil)

	// Test with valid performance data
	minLatency, err := service.GetMinimumRecommendedLatency(
		context.Background(),
		"test-model",
		"test-source",
		ParetoFilteringParams{
			LatencyProperty:       "ttft_p90",
			RpsProperty:           "requests_per_second",
			HardwareCountProperty: "hardware_count",
			HardwareTypeProperty:  "hardware_type",
		},
		"",
	)

	require.NoError(t, err)
	require.NotNil(t, minLatency)
	assert.Equal(t, 150.0, *minLatency) // Expected minimum from test data
}

func TestGetMinimumRecommendedLatency_NoArtifacts(t *testing.T) {
	// Mock repositories for testing
	mockArtifactRepo := &mockPerfArtifactRepo{}
	mockModelRepo := &mockPerfModelRepo{}

	// Setup mock model repository to return a model with ID
	testModelID := int32(42)
	mockModelRepo.On("GetByName", "nonexistent-model").Return(&dbmodels.BaseEntity[models.CatalogModelAttributes]{
		ID: new(testModelID),
	}, nil)

	service := NewPerformanceArtifactService(mockArtifactRepo, mockModelRepo)

	// Setup mock to return empty result
	mockArtifactRepo.On("List", mock.AnythingOfType("models.CatalogArtifactListOptions")).
		Return(&dbmodels.ListWrapper[sharedmodels.CatalogArtifact]{
			Items: []sharedmodels.CatalogArtifact{},
		}, nil)

	// Test with model that has no performance artifacts
	minLatency, err := service.GetMinimumRecommendedLatency(
		context.Background(),
		"nonexistent-model",
		"test-source",
		ParetoFilteringParams{},
		"",
	)

	require.NoError(t, err)
	require.Nil(t, minLatency) // Should return nil for models without data
}

func TestPartitionBySubType(t *testing.T) {
	service := NewPerformanceArtifactService(nil, nil)

	id1 := int32(1)
	id2 := int32(2)
	id3 := int32(3)

	coldStart := "cold-start"
	artifacts := []sharedmodels.CatalogMetricsArtifact{
		&dbmodels.BaseEntity[models.CatalogMetricsArtifactAttributes]{
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name: new("perf-benchmark"),
			},
			CustomProperties: &[]dbmodels.Properties{
				dbmodels.NewDoubleProperty("requests_per_second", 100.0, true),
				dbmodels.NewDoubleProperty("ttft_p90", 50.0, true),
				dbmodels.NewIntProperty("hardware_count", 2, true),
				dbmodels.NewStringProperty("hardware_type", "A100", true),
			},
		},
		&dbmodels.BaseEntity[models.CatalogMetricsArtifactAttributes]{
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name: new("cold-start-entry"),
			},
			CustomProperties: &[]dbmodels.Properties{
				{Name: "performance_sub_type", StringValue: &coldStart},
				dbmodels.NewStringProperty("gpu_type", "A100", true),
				dbmodels.NewIntProperty("gpu_count", 2, true),
			},
		},
		&dbmodels.BaseEntity[models.CatalogMetricsArtifactAttributes]{
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name: new("another-benchmark"),
			},
			CustomProperties: &[]dbmodels.Properties{
				dbmodels.NewDoubleProperty("requests_per_second", 200.0, true),
				dbmodels.NewDoubleProperty("ttft_p90", 30.0, true),
				dbmodels.NewIntProperty("hardware_count", 4, true),
				dbmodels.NewStringProperty("hardware_type", "H100", true),
			},
		},
	}
	artifacts[0].SetID(id1)
	artifacts[1].SetID(id2)
	artifacts[2].SetID(id3)

	benchmarks, coldStarts := service.partitionBySubType(artifacts)

	require.Len(t, benchmarks, 2)
	require.Len(t, coldStarts, 1)
	assert.Equal(t, id1, *benchmarks[0].GetID())
	assert.Equal(t, id3, *benchmarks[1].GetID())
	assert.Equal(t, id2, *coldStarts[0].GetID())
}

func TestPartitionBySubType_AllColdStart(t *testing.T) {
	service := NewPerformanceArtifactService(nil, nil)

	coldStart := "cold-start"
	id1 := int32(1)
	artifacts := []sharedmodels.CatalogMetricsArtifact{
		&dbmodels.BaseEntity[models.CatalogMetricsArtifactAttributes]{
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name: new("cold-start-only"),
			},
			CustomProperties: &[]dbmodels.Properties{
				{Name: "performance_sub_type", StringValue: &coldStart},
				dbmodels.NewStringProperty("gpu_type", "A100", true),
				dbmodels.NewIntProperty("gpu_count", 1, true),
			},
		},
	}
	artifacts[0].SetID(id1)

	benchmarks, coldStarts := service.partitionBySubType(artifacts)
	require.Empty(t, benchmarks)
	require.Len(t, coldStarts, 1)
}

func TestGetArtifacts_OnlyColdStartArtifacts(t *testing.T) {
	mockArtifactRepo := &mockPerfArtifactRepo{}
	service := NewPerformanceArtifactService(mockArtifactRepo, nil)

	coldStart := "cold-start"
	id1 := int32(1)
	testArtifacts := []sharedmodels.CatalogArtifact{
		{
			CatalogMetricsArtifact: &dbmodels.BaseEntity[models.CatalogMetricsArtifactAttributes]{
				Attributes: &models.CatalogMetricsArtifactAttributes{
					Name:        new("cold-start-artifact"),
					MetricsType: models.MetricsTypePerformance,
				},
				Properties: &[]dbmodels.Properties{},
				CustomProperties: &[]dbmodels.Properties{
					{Name: "performance_sub_type", StringValue: &coldStart},
					dbmodels.NewStringProperty("gpu_type", "A100", true),
					dbmodels.NewIntProperty("gpu_count", 2, true),
				},
			},
		},
	}
	testArtifacts[0].CatalogMetricsArtifact.SetID(id1)

	mockArtifactRepo.On("List", mock.AnythingOfType("models.CatalogArtifactListOptions")).
		Return(&dbmodels.ListWrapper[sharedmodels.CatalogArtifact]{
			Items: testArtifacts,
		}, nil)

	params := PerformanceArtifactParams{
		ModelID:         123,
		TargetRPS:       100,
		Recommendations: false,
		PageSize:        10,
	}

	result, err := service.GetArtifacts(params)

	require.NoError(t, err, "should not error when only cold-start artifacts are present")
	require.Len(t, result.Items, 1, "cold-start artifacts should be included in results")
	assert.Equal(t, id1, *result.Items[0].GetID())
}

func TestGetArtifacts_MixedBenchmarkAndColdStart(t *testing.T) {
	mockArtifactRepo := &mockPerfArtifactRepo{}
	service := NewPerformanceArtifactService(mockArtifactRepo, nil)

	coldStart := "cold-start"
	id1 := int32(1)
	id2 := int32(2)
	id3 := int32(3)

	// Repository returns: benchmark, cold-start, benchmark (interleaved)
	testArtifacts := []sharedmodels.CatalogArtifact{
		{
			CatalogMetricsArtifact: &dbmodels.BaseEntity[models.CatalogMetricsArtifactAttributes]{
				Attributes: &models.CatalogMetricsArtifactAttributes{
					Name:        new("benchmark-1"),
					MetricsType: models.MetricsTypePerformance,
				},
				Properties: &[]dbmodels.Properties{},
				CustomProperties: &[]dbmodels.Properties{
					dbmodels.NewDoubleProperty("requests_per_second", 150.0, true),
					dbmodels.NewDoubleProperty("ttft_p90", 40.0, true),
					dbmodels.NewIntProperty("hardware_count", 2, true),
					dbmodels.NewStringProperty("hardware_type", "A100", true),
				},
			},
		},
		{
			CatalogMetricsArtifact: &dbmodels.BaseEntity[models.CatalogMetricsArtifactAttributes]{
				Attributes: &models.CatalogMetricsArtifactAttributes{
					Name:        new("cold-start-artifact"),
					MetricsType: models.MetricsTypePerformance,
				},
				Properties: &[]dbmodels.Properties{},
				CustomProperties: &[]dbmodels.Properties{
					{Name: "performance_sub_type", StringValue: &coldStart},
					dbmodels.NewStringProperty("gpu_type", "A100", true),
					dbmodels.NewIntProperty("gpu_count", 2, true),
				},
			},
		},
		{
			CatalogMetricsArtifact: &dbmodels.BaseEntity[models.CatalogMetricsArtifactAttributes]{
				Attributes: &models.CatalogMetricsArtifactAttributes{
					Name:        new("benchmark-2"),
					MetricsType: models.MetricsTypePerformance,
				},
				Properties: &[]dbmodels.Properties{},
				CustomProperties: &[]dbmodels.Properties{
					dbmodels.NewDoubleProperty("requests_per_second", 300.0, true),
					dbmodels.NewDoubleProperty("ttft_p90", 80.0, true),
					dbmodels.NewIntProperty("hardware_count", 4, true),
					dbmodels.NewStringProperty("hardware_type", "A100", true),
				},
			},
		},
	}
	testArtifacts[0].CatalogMetricsArtifact.SetID(id1)
	testArtifacts[1].CatalogMetricsArtifact.SetID(id2)
	testArtifacts[2].CatalogMetricsArtifact.SetID(id3)

	mockArtifactRepo.On("List", mock.AnythingOfType("models.CatalogArtifactListOptions")).
		Return(&dbmodels.ListWrapper[sharedmodels.CatalogArtifact]{
			Items: testArtifacts,
		}, nil)

	params := PerformanceArtifactParams{
		ModelID:         123,
		TargetRPS:       300,
		Recommendations: false,
		PageSize:        10,
	}

	result, err := service.GetArtifacts(params)

	require.NoError(t, err, "should handle mixed artifact types without error")
	require.Len(t, result.Items, 3, "all artifacts should be returned")

	// Verify order matches repository order: benchmark-1, cold-start, benchmark-2
	assert.Equal(t, id1, *result.Items[0].GetID())
	assert.Equal(t, id2, *result.Items[1].GetID())
	assert.Equal(t, id3, *result.Items[2].GetID())

	// Benchmark artifact should have targetRPS calculations added
	benchProps := result.Items[0].GetCustomProperties()
	var hasReplicas bool
	for _, prop := range *benchProps {
		if prop.Name == "replicas" {
			hasReplicas = true
			require.Equal(t, int32(2), *prop.IntValue) // ceil(300/150) = 2
		}
	}
	assert.True(t, hasReplicas, "benchmark artifact should have replicas calculated")

	// Cold-start artifact should NOT have replicas added
	coldProps := result.Items[1].GetCustomProperties()
	for _, prop := range *coldProps {
		assert.NotEqual(t, "replicas", prop.Name, "cold-start artifact should not have replicas")
	}
}
