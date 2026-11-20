package core_test

import (
	"fmt"
	"testing"

	"github.com/kubeflow/hub/pkg/api"
	"github.com/kubeflow/hub/pkg/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetExperimentRunMetricHistory(t *testing.T) {
	service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	// Create experiment
	experiment := &openapi.Experiment{
		Name:        "test-experiment",
		Description: new("Test experiment for metric history"),
	}
	savedExperiment, err := service.UpsertExperiment(experiment)
	require.NoError(t, err)

	// Create experiment run
	experimentRun := &openapi.ExperimentRun{
		Name:        new("test-experiment-run"),
		Description: new("Test experiment run for metric history"),
	}
	savedExperimentRun, err := service.UpsertExperimentRun(experimentRun, savedExperiment.Id)
	require.NoError(t, err)

	// Create metrics via UpsertExperimentRunArtifact (this should populate experiment fields and create metric history)
	metricArtifact1 := &openapi.Artifact{
		Metric: &openapi.Metric{
			Name:        new("accuracy"),
			Value:       new(0.95),
			Timestamp:   new("1234567890"),
			Step:        new(int64(1)),
			Description: new("Test accuracy metric"),
		},
	}

	metricArtifact2 := &openapi.Artifact{
		Metric: &openapi.Metric{
			Name:        new("loss"),
			Value:       new(0.05),
			Timestamp:   new("1234567891"),
			Step:        new(int64(2)),
			Description: new("Test loss metric"),
		},
	}

	// Upsert metric artifacts through experiment run (should trigger metric history creation with experiment fields)
	_, err = service.UpsertExperimentRunArtifact(metricArtifact1, *savedExperimentRun.Id)
	require.NoError(t, err, "error upserting metric artifact 1")

	_, err = service.UpsertExperimentRunArtifact(metricArtifact2, *savedExperimentRun.Id)
	require.NoError(t, err, "error upserting metric artifact 2")

	// Test getting all metric history
	result, err := service.GetExperimentRunMetricHistory(nil, nil, api.ListOptions{}, savedExperimentRun.Id)
	require.NoError(t, err, "error getting metric history")
	assert.Equal(t, int32(2), result.Size, "should return 2 metric history records")
	assert.Equal(t, 2, len(result.Items), "should have 2 items in the result")

	// Verify the metrics are returned correctly
	foundAccuracy := false
	foundLoss := false
	for _, item := range result.Items {
		// Verify experiment fields are populated
		assert.Equal(t, savedExperiment.Id, item.ExperimentId, "experimentId should be populated")
		assert.Equal(t, savedExperimentRun.Id, item.ExperimentRunId, "experimentRunId should be populated")
		assert.NotNil(t, item.CustomProperties, "customProperties should not be nil")

		// Verify that experiment properties are filtered out of custom properties
		assert.NotContains(t, item.CustomProperties, "experiment_id", "experiment_id should be filtered out of custom properties")
		assert.NotContains(t, item.CustomProperties, "experiment_run_id", "experiment_run_id should be filtered out of custom properties")

		switch *item.Name {
		case "accuracy":
			foundAccuracy = true
			assert.Equal(t, 0.95, *item.Value)
			assert.Equal(t, "1234567890", *item.Timestamp)
			assert.Equal(t, int64(1), *item.Step)
		case "loss":
			foundLoss = true
			assert.Equal(t, 0.05, *item.Value)
			assert.Equal(t, "1234567891", *item.Timestamp)
			assert.Equal(t, int64(2), *item.Step)
		}
	}
	assert.True(t, foundAccuracy, "should find accuracy metric")
	assert.True(t, foundLoss, "should find loss metric")
}

func TestGetExperimentRunMetricHistoryWithNameFilter(t *testing.T) {
	service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	// Create experiment
	experiment := &openapi.Experiment{
		Name:        "test-experiment-filter",
		Description: new("Test experiment for metric history name filter"),
	}
	savedExperiment, err := service.UpsertExperiment(experiment)
	require.NoError(t, err)

	// Create experiment run
	experimentRun := &openapi.ExperimentRun{
		Name:        new("test-experiment-run-filter"),
		Description: new("Test experiment run for metric history name filter"),
	}
	savedExperimentRun, err := service.UpsertExperimentRun(experimentRun, savedExperiment.Id)
	require.NoError(t, err)

	// Create metrics via UpsertExperimentRunArtifact (this should populate experiment fields and create metric history)
	metricArtifact1 := &openapi.Artifact{
		Metric: &openapi.Metric{
			Name:      new("accuracy"),
			Value:     new(0.95),
			Timestamp: new("1234567890"),
			Step:      new(int64(1)),
		},
	}

	metricArtifact2 := &openapi.Artifact{
		Metric: &openapi.Metric{
			Name:      new("loss"),
			Value:     new(0.05),
			Timestamp: new("1234567891"),
			Step:      new(int64(2)),
		},
	}

	// Upsert metric artifacts through experiment run (should trigger metric history creation with experiment fields)
	_, err = service.UpsertExperimentRunArtifact(metricArtifact1, *savedExperimentRun.Id)
	require.NoError(t, err, "error upserting metric artifact 1")

	_, err = service.UpsertExperimentRunArtifact(metricArtifact2, *savedExperimentRun.Id)
	require.NoError(t, err, "error upserting metric artifact 2")

	// Test filtering by name
	accuracyName := "accuracy"
	result, err := service.GetExperimentRunMetricHistory(&accuracyName, nil, api.ListOptions{}, savedExperimentRun.Id)
	require.NoError(t, err, "error getting metric history with name filter")
	assert.Equal(t, int32(1), result.Size, "should return 1 metric history record for accuracy")
	assert.Equal(t, 1, len(result.Items), "should have 1 item in the result")
	assert.Equal(t, "accuracy", *result.Items[0].Name)
	assert.Equal(t, 0.95, *result.Items[0].Value)

	// Verify experiment fields are populated
	assert.Equal(t, savedExperiment.Id, result.Items[0].ExperimentId, "experimentId should be populated")
	assert.Equal(t, savedExperimentRun.Id, result.Items[0].ExperimentRunId, "experimentRunId should be populated")
	assert.NotNil(t, result.Items[0].CustomProperties, "customProperties should not be nil")

	// Verify that experiment properties are filtered out of custom properties
	assert.NotContains(t, result.Items[0].CustomProperties, "experiment_id", "experiment_id should be filtered out of custom properties")
	assert.NotContains(t, result.Items[0].CustomProperties, "experiment_run_id", "experiment_run_id should be filtered out of custom properties")
}

func TestGetExperimentRunMetricHistoryNameFilterMatchesWholeName(t *testing.T) {
	service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	savedExperiment, err := service.UpsertExperiment(&openapi.Experiment{Name: "metric-name-match-experiment"})
	require.NoError(t, err)
	savedExperimentRun, err := service.UpsertExperimentRun(&openapi.ExperimentRun{Name: new("metric-name-match-run")}, savedExperiment.Id)
	require.NoError(t, err)

	// "loss" is a substring of the other names, and "_" is a LIKE wildcard.
	for i, name := range []string{"loss", "loss_", "val_loss", "loss_scale", "loss__ema", "lossXscale"} {
		_, err := service.UpsertExperimentRunArtifact(&openapi.Artifact{
			Metric: &openapi.Metric{
				Name:  new(name),
				Value: new(float64(i)),
				Step:  new(int64(1)),
			},
		}, *savedExperimentRun.Id)
		require.NoError(t, err, "error upserting metric %s", name)
	}

	for _, name := range []string{"loss", "loss_", "loss__ema", "loss_scale"} {
		result, err := service.GetExperimentRunMetricHistory(new(name), nil, api.ListOptions{}, savedExperimentRun.Id)
		require.NoError(t, err)
		require.Len(t, result.Items, 1, "metric history for %q", name)
		assert.Equal(t, name, *result.Items[0].Name)

		result, err = service.GetExperimentRunMetricHistory(new(name), nil, api.ListOptions{}, nil)
		require.NoError(t, err)
		require.Len(t, result.Items, 1, "bulk metric history for %q", name)
		assert.Equal(t, name, *result.Items[0].Name)
	}
}

func TestInsertMetricHistory(t *testing.T) {
	service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	// Create experiment
	experiment := &openapi.Experiment{
		Name:        "test-experiment-insert",
		Description: new("Test experiment for metric history insert"),
	}
	savedExperiment, err := service.UpsertExperiment(experiment)
	require.NoError(t, err)

	// Create experiment run
	experimentRun := &openapi.ExperimentRun{
		Name:        new("test-experiment-run-insert"),
		Description: new("Test experiment run for metric history insert"),
	}
	savedExperimentRun, err := service.UpsertExperimentRun(experimentRun, savedExperiment.Id)
	require.NoError(t, err)

	// Test 1: Basic metric history insertion
	metric := &openapi.Metric{
		Name:            new("test_metric"),
		Value:           new(42.5),
		Timestamp:       new("1234567890"),
		Step:            new(int64(1)),
		Description:     new("Test metric description"),
		ExperimentId:    savedExperiment.Id,
		ExperimentRunId: savedExperimentRun.Id,
		CustomProperties: map[string]openapi.MetadataValue{
			"custom_prop": {
				MetadataStringValue: &openapi.MetadataStringValue{
					StringValue:  "custom_value",
					MetadataType: "MetadataStringValue",
				},
			},
			"experiment_id": {
				MetadataIntValue: &openapi.MetadataIntValue{
					IntValue:     *savedExperiment.Id,
					MetadataType: "MetadataIntValue",
				},
			},
			"experiment_run_id": {
				MetadataIntValue: &openapi.MetadataIntValue{
					IntValue:     *savedExperimentRun.Id,
					MetadataType: "MetadataIntValue",
				},
			},
		},
	}

	// Insert metric history
	err = service.InsertMetricHistory(metric, *savedExperimentRun.Id)
	require.NoError(t, err, "error inserting metric history")

	// Verify the metric history was created
	result, err := service.GetExperimentRunMetricHistory(nil, nil, api.ListOptions{}, savedExperimentRun.Id)
	require.NoError(t, err, "error getting metric history after insertion")
	assert.Equal(t, int32(1), result.Size, "should have 1 metric history record")
	assert.Equal(t, 1, len(result.Items), "should have 1 item in the result")

	// Verify the inserted metric properties
	insertedMetric := &result.Items[0]
	assert.Equal(t, "test_metric", *insertedMetric.Name)
	assert.Equal(t, 42.5, *insertedMetric.Value)
	assert.Equal(t, "1234567890", *insertedMetric.Timestamp)
	assert.Equal(t, int64(1), *insertedMetric.Step)
	assert.Equal(t, "Test metric description", *insertedMetric.Description)

	// Verify experiment fields are populated
	assert.Equal(t, savedExperiment.Id, insertedMetric.ExperimentId, "experimentId should be populated")
	assert.Equal(t, savedExperimentRun.Id, insertedMetric.ExperimentRunId, "experimentRunId should be populated")
	assert.NotNil(t, insertedMetric.CustomProperties, "customProperties should not be nil")

	// Verify that experiment properties are filtered out but other custom properties remain
	assert.NotContains(t, insertedMetric.CustomProperties, "experiment_id", "experiment_id should be filtered out of custom properties")
	assert.NotContains(t, insertedMetric.CustomProperties, "experiment_run_id", "experiment_run_id should be filtered out of custom properties")
	assert.Contains(t, insertedMetric.CustomProperties, "custom_prop", "other custom properties should remain")

	// Test 2: Error handling - nil metric
	err = service.InsertMetricHistory(nil, *savedExperimentRun.Id)
	assert.Error(t, err, "should return error for nil metric")
	assert.Contains(t, err.Error(), "metric cannot be nil")

	// Test 3: Error handling - empty experiment run ID
	err = service.InsertMetricHistory(metric, "")
	assert.Error(t, err, "should return error for empty experiment run ID")
	assert.Contains(t, err.Error(), "experiment run ID is required")

	// Test 4: Error handling - non-existent experiment run ID
	err = service.InsertMetricHistory(metric, "999999")
	assert.Error(t, err, "should return error for non-existent experiment run ID")
	assert.Contains(t, err.Error(), "experiment run not found")
}

func TestUpsertExperimentRunArtifactTriggersMetricHistory(t *testing.T) {
	service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	// Create experiment
	experiment := &openapi.Experiment{
		Name:        "test-experiment-upsert",
		Description: new("Test experiment for metric history upsert"),
	}
	savedExperiment, err := service.UpsertExperiment(experiment)
	require.NoError(t, err)

	// Create experiment run
	experimentRun := &openapi.ExperimentRun{
		Name:        new("test-experiment-run-upsert"),
		Description: new("Test experiment run for metric history upsert"),
	}
	savedExperimentRun, err := service.UpsertExperimentRun(experimentRun, savedExperiment.Id)
	require.NoError(t, err)

	// Create a metric artifact
	metricArtifact := &openapi.Artifact{
		Metric: &openapi.Metric{
			Name:      new("test_metric"),
			Value:     new(42.5),
			Timestamp: new("1234567890"),
			Step:      new(int64(1)),
		},
	}

	// Upsert the metric artifact (this should trigger InsertMetricHistory)
	createdArtifact, err := service.UpsertExperimentRunArtifact(metricArtifact, *savedExperimentRun.Id)
	require.NoError(t, err, "error upserting metric artifact")
	assert.NotNil(t, createdArtifact.Metric, "should have created metric artifact")

	// Verify that metric history was also created
	result, err := service.GetExperimentRunMetricHistory(nil, nil, api.ListOptions{}, savedExperimentRun.Id)
	require.NoError(t, err, "error getting metric history after upsert")
	assert.Equal(t, int32(1), result.Size, "should have 1 metric history record created automatically")

	// Verify the metric history has the correct properties
	insertedMetric := &result.Items[0]
	assert.Equal(t, "test_metric", *insertedMetric.Name)
	assert.Equal(t, 42.5, *insertedMetric.Value)
	assert.Equal(t, "1234567890", *insertedMetric.Timestamp)
	assert.Equal(t, int64(1), *insertedMetric.Step)

	// Verify experiment fields are populated from UpsertExperimentRunArtifact
	assert.Equal(t, savedExperiment.Id, insertedMetric.ExperimentId, "experimentId should be populated from UpsertExperimentRunArtifact")
	assert.Equal(t, savedExperimentRun.Id, insertedMetric.ExperimentRunId, "experimentRunId should be populated from UpsertExperimentRunArtifact")
	assert.NotNil(t, insertedMetric.CustomProperties, "customProperties should not be nil")
	assert.Empty(t, insertedMetric.CustomProperties, "customProperties should be empty (no duplicate experiment fields)")
}

func TestGetExperimentRunMetricHistoryEmptyResult(t *testing.T) {
	service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	// Create experiment
	experiment := &openapi.Experiment{
		Name:        "test-experiment-empty",
		Description: new("Test experiment for empty metric history"),
	}
	savedExperiment, err := service.UpsertExperiment(experiment)
	require.NoError(t, err)

	// Create experiment run
	experimentRun := &openapi.ExperimentRun{
		Name:        new("test-experiment-run-empty"),
		Description: new("Test experiment run for empty metric history"),
	}
	savedExperimentRun, err := service.UpsertExperimentRun(experimentRun, savedExperiment.Id)
	require.NoError(t, err)

	// Test getting metric history for experiment run with no metrics
	result, err := service.GetExperimentRunMetricHistory(nil, nil, api.ListOptions{}, savedExperimentRun.Id)
	require.NoError(t, err, "error getting empty metric history")
	assert.Equal(t, int32(0), result.Size, "should return 0 metric history records")
	assert.Equal(t, 0, len(result.Items), "should have 0 items in the result")
}

func TestGetExperimentRunMetricHistoryWithPagination(t *testing.T) {
	service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	// Create experiment
	experiment := &openapi.Experiment{
		Name:        "test-experiment-pagination",
		Description: new("Test experiment for metric history pagination"),
	}
	savedExperiment, err := service.UpsertExperiment(experiment)
	require.NoError(t, err)

	// Create experiment run
	experimentRun := &openapi.ExperimentRun{
		Name:        new("test-experiment-run-pagination"),
		Description: new("Test experiment run for metric history pagination"),
	}
	savedExperimentRun, err := service.UpsertExperimentRun(experimentRun, savedExperiment.Id)
	require.NoError(t, err)

	// Create multiple metrics for pagination test
	for i := range 5 {
		metric := &openapi.Metric{
			Name:      new(fmt.Sprintf("metric_%d", i)),
			Value:     new(float64(i) * 0.1),
			Timestamp: new(fmt.Sprintf("123456789%d", i)),
			Step:      new(int64(i)),
		}
		err = service.InsertMetricHistory(metric, *savedExperimentRun.Id)
		require.NoError(t, err, "error inserting metric history %d", i)
	}

	// Test pagination with page size 2
	pageSize := int32(2)
	listOptions := api.ListOptions{
		PageSize: &pageSize,
	}

	result, err := service.GetExperimentRunMetricHistory(nil, nil, listOptions, savedExperimentRun.Id)
	require.NoError(t, err, "error getting paginated metric history")
	assert.Equal(t, int32(2), result.Size, "should return 2 metric history records on first page")
	assert.Equal(t, 2, len(result.Items), "should have 2 items in the result")
	assert.Equal(t, pageSize, result.PageSize, "should have correct page size")
}

func TestGetExperimentRunMetricHistoryBulkQuery(t *testing.T) {
	service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	// Test with nil experiment run ID should work (bulk query)
	result, err := service.GetExperimentRunMetricHistory(nil, nil, api.ListOptions{}, nil)
	assert.NoError(t, err, "should not return error for nil experiment run ID (bulk query)")
	assert.NotNil(t, result, "should return result for bulk query")
}

func TestGetExperimentRunMetricHistoryWithInvalidExperimentRunId(t *testing.T) {
	service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	// Test with non-existent experiment run ID
	nonExistentId := "999999"
	_, err := service.GetExperimentRunMetricHistory(nil, nil, api.ListOptions{}, &nonExistentId)
	assert.Error(t, err, "should return error for non-existent experiment run ID")
	assert.Contains(t, err.Error(), "experiment run not found")
}

func TestInsertMetricHistoryWithLastUpdateTime(t *testing.T) {
	service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	// Create experiment
	experiment := &openapi.Experiment{
		Name:        "test-experiment-last-update",
		Description: new("Test experiment for metric history with last update time"),
	}
	savedExperiment, err := service.UpsertExperiment(experiment)
	require.NoError(t, err)

	// Create experiment run
	experimentRun := &openapi.ExperimentRun{
		Name:        new("test-experiment-run-last-update"),
		Description: new("Test experiment run for metric history with last update time"),
	}
	savedExperimentRun, err := service.UpsertExperimentRun(experimentRun, savedExperiment.Id)
	require.NoError(t, err)

	// Test insertion with last update time but no timestamp
	metricWithLastUpdate := &openapi.Metric{
		Name:                     new("test_metric_last_update"),
		Value:                    new(42.5),
		LastUpdateTimeSinceEpoch: new("9876543210"),
		Step:                     new(int64(1)),
	}

	// Insert metric history with last update time
	err = service.InsertMetricHistory(metricWithLastUpdate, *savedExperimentRun.Id)
	require.NoError(t, err, "error inserting metric history with last update time")

	// Verify the metric history was created
	result, err := service.GetExperimentRunMetricHistory(nil, nil, api.ListOptions{}, savedExperimentRun.Id)
	require.NoError(t, err, "error getting metric history after insertion")
	assert.Equal(t, int32(1), result.Size, "should have 1 metric history record")

	// Verify the inserted metric has the correct name (should use last update time as timestamp)
	insertedMetric := &result.Items[0]
	assert.Equal(t, "test_metric_last_update", *insertedMetric.Name)
	assert.Equal(t, 42.5, *insertedMetric.Value)
}

func TestGetExperimentRunMetricHistoryReturnsCorrectArtifactType(t *testing.T) {
	service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	// Create experiment
	experiment := &openapi.Experiment{
		Name:        "test-experiment-artifact-type",
		Description: new("Test experiment for metric history artifact type validation"),
	}
	savedExperiment, err := service.UpsertExperiment(experiment)
	require.NoError(t, err)

	// Create experiment run
	experimentRun := &openapi.ExperimentRun{
		Name:        new("test-experiment-run-artifact-type"),
		Description: new("Test experiment run for metric history artifact type validation"),
	}
	savedExperimentRun, err := service.UpsertExperimentRun(experimentRun, savedExperiment.Id)
	require.NoError(t, err)

	// Create a metric for the experiment run
	metric := &openapi.Metric{
		Name:        new("test_metric"),
		Value:       new(0.95),
		Timestamp:   new("1234567890"),
		Step:        new(int64(1)),
		Description: new("Test metric for artifact type validation"),
	}

	// Insert metric history (this stores as 'metric-history' in database)
	err = service.InsertMetricHistory(metric, *savedExperimentRun.Id)
	require.NoError(t, err, "error inserting metric history")

	// Get metric history via the REST API endpoint
	result, err := service.GetExperimentRunMetricHistory(nil, nil, api.ListOptions{}, savedExperimentRun.Id)
	require.NoError(t, err, "error getting metric history")
	assert.Equal(t, int32(1), result.Size, "should return 1 metric history record")
	assert.Equal(t, 1, len(result.Items), "should have 1 item in the result")

	// Verify the artifact type is 'metric' (not 'metric-history') for REST API compliance
	retrievedMetric := &result.Items[0]

	// The key validation: ensure artifact type is "metric" for REST API, even though it's stored as "metric-history" internally
	assert.Equal(t, "metric", *retrievedMetric.ArtifactType, "artifact type should be 'metric' for REST API compliance")

	// Verify other properties are correctly mapped
	assert.Equal(t, "test_metric", *retrievedMetric.Name)
	assert.Equal(t, 0.95, *retrievedMetric.Value)
	assert.Equal(t, "1234567890", *retrievedMetric.Timestamp)
	assert.Equal(t, int64(1), *retrievedMetric.Step)
	assert.Equal(t, "Test metric for artifact type validation", *retrievedMetric.Description)
}

func TestGetExperimentRunMetricHistoryWithStepIdsFilter(t *testing.T) {
	service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	// Create experiment
	experiment := &openapi.Experiment{
		Name:        "test-experiment-stepids",
		Description: new("Test experiment for metric history stepIds filter"),
	}
	savedExperiment, err := service.UpsertExperiment(experiment)
	require.NoError(t, err)

	// Create experiment run
	experimentRun := &openapi.ExperimentRun{
		Name:        new("test-experiment-run-stepids"),
		Description: new("Test experiment run for metric history stepIds filter"),
	}
	savedExperimentRun, err := service.UpsertExperimentRun(experimentRun, savedExperiment.Id)
	require.NoError(t, err)

	// Create metrics with different step values
	metrics := []*openapi.Metric{
		{
			Name:      new("accuracy"),
			Value:     new(0.85),
			Timestamp: new("1234567890"),
			Step:      new(int64(1)),
		},
		{
			Name:      new("accuracy"),
			Value:     new(0.90),
			Timestamp: new("1234567891"),
			Step:      new(int64(2)),
		},
		{
			Name:      new("accuracy"),
			Value:     new(0.95),
			Timestamp: new("1234567892"),
			Step:      new(int64(3)),
		},
		{
			Name:      new("loss"),
			Value:     new(0.15),
			Timestamp: new("1234567893"),
			Step:      new(int64(1)),
		},
		{
			Name:      new("loss"),
			Value:     new(0.10),
			Timestamp: new("1234567894"),
			Step:      new(int64(2)),
		},
	}

	// Insert all metric history records
	for _, metric := range metrics {
		err = service.InsertMetricHistory(metric, *savedExperimentRun.Id)
		require.NoError(t, err, "error inserting metric history")
	}

	// Test filtering by single step ID
	stepIds := "1"
	result, err := service.GetExperimentRunMetricHistory(nil, &stepIds, api.ListOptions{}, savedExperimentRun.Id)
	require.NoError(t, err, "error getting metric history with step filter")
	assert.Equal(t, int32(2), result.Size, "should return 2 metric history records for step 1")
	assert.Equal(t, 2, len(result.Items), "should have 2 items in the result")

	// Verify the returned metrics are from step 1
	for _, item := range result.Items {
		assert.Equal(t, int64(1), *item.Step, "all metrics should be from step 1")
	}

	// Test filtering by multiple step IDs
	stepIds = "1,3"
	result, err = service.GetExperimentRunMetricHistory(nil, &stepIds, api.ListOptions{}, savedExperimentRun.Id)
	require.NoError(t, err, "error getting metric history with multiple step filter")
	assert.Equal(t, int32(3), result.Size, "should return 3 metric history records for steps 1 and 3")
	assert.Equal(t, 3, len(result.Items), "should have 3 items in the result")

	// Test invalid step IDs are properly handled at service level
	t.Run("InvalidStepIds", func(t *testing.T) {
		// Test with invalid Unicode character
		invalidStepIds := "耪"
		_, err := service.GetExperimentRunMetricHistory(nil, &invalidStepIds, api.ListOptions{}, savedExperimentRun.Id)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid step ID '耪': must be a valid integer")

		// Test with mixed valid and invalid step IDs
		invalidStepIds = "1,invalid,3"
		_, err = service.GetExperimentRunMetricHistory(nil, &invalidStepIds, api.ListOptions{}, savedExperimentRun.Id)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid step ID 'invalid': must be a valid integer")

		// Test with non-numeric characters
		invalidStepIds = "abc"
		_, err = service.GetExperimentRunMetricHistory(nil, &invalidStepIds, api.ListOptions{}, savedExperimentRun.Id)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid step ID 'abc': must be a valid integer")
	})

	// Verify the returned metrics are from steps 1 and 3
	stepValues := make(map[int64]bool)
	for _, item := range result.Items {
		stepValues[*item.Step] = true
	}
	assert.True(t, stepValues[1], "should contain metrics from step 1")
	assert.True(t, stepValues[3], "should contain metrics from step 3")
	assert.False(t, stepValues[2], "should not contain metrics from step 2")

	// Test filtering by non-existent step ID
	stepIds = "999"
	result, err = service.GetExperimentRunMetricHistory(nil, &stepIds, api.ListOptions{}, savedExperimentRun.Id)
	require.NoError(t, err, "error getting metric history with non-existent step filter")
	assert.Equal(t, int32(0), result.Size, "should return 0 metric history records for non-existent step")
	assert.Equal(t, 0, len(result.Items), "should have 0 items in the result")

	// Test combining stepIds with name filter
	accuracyName := "accuracy"
	stepIds = "1,2"
	result, err = service.GetExperimentRunMetricHistory(&accuracyName, &stepIds, api.ListOptions{}, savedExperimentRun.Id)
	require.NoError(t, err, "error getting metric history with name and step filter")
	assert.Equal(t, int32(2), result.Size, "should return 2 accuracy metric history records for steps 1 and 2")
	assert.Equal(t, 2, len(result.Items), "should have 2 items in the result")

	// Verify all returned metrics are accuracy metrics from steps 1 and 2
	for _, item := range result.Items {
		assert.Equal(t, "accuracy", *item.Name, "all metrics should be accuracy")
		assert.True(t, *item.Step == 1 || *item.Step == 2, "all metrics should be from step 1 or 2")
	}
}

func TestGetExperimentRunsMetricHistoryBulk(t *testing.T) {
	service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	// Create experiment
	experiment := &openapi.Experiment{
		Name:        "test-experiment-bulk",
		Description: new("Test experiment for bulk metric history"),
	}
	savedExperiment, err := service.UpsertExperiment(experiment)
	require.NoError(t, err)

	// Create two experiment runs
	experimentRun1 := &openapi.ExperimentRun{
		Name:        new("test-experiment-run-1"),
		Description: new("Test experiment run 1 for bulk metric history"),
	}
	savedExperimentRun1, err := service.UpsertExperimentRun(experimentRun1, savedExperiment.Id)
	require.NoError(t, err)

	experimentRun2 := &openapi.ExperimentRun{
		Name:        new("test-experiment-run-2"),
		Description: new("Test experiment run 2 for bulk metric history"),
	}
	savedExperimentRun2, err := service.UpsertExperimentRun(experimentRun2, savedExperiment.Id)
	require.NoError(t, err)

	// Create metrics for first experiment run
	metric1Run1 := &openapi.Metric{
		Name:      new("accuracy"),
		Value:     new(0.85),
		Timestamp: new("1234567890"),
		Step:      new(int64(1)),
	}
	metric2Run1 := &openapi.Metric{
		Name:      new("loss"),
		Value:     new(0.15),
		Timestamp: new("1234567891"),
		Step:      new(int64(1)),
	}

	// Create metrics for second experiment run
	metric1Run2 := &openapi.Metric{
		Name:      new("accuracy"),
		Value:     new(0.90),
		Timestamp: new("1234567892"),
		Step:      new(int64(2)),
	}
	metric2Run2 := &openapi.Metric{
		Name:      new("loss"),
		Value:     new(0.10),
		Timestamp: new("1234567893"),
		Step:      new(int64(2)),
	}

	// Insert metric history for both runs
	err = service.InsertMetricHistory(metric1Run1, *savedExperimentRun1.Id)
	require.NoError(t, err, "error inserting metric 1 for run 1")
	err = service.InsertMetricHistory(metric2Run1, *savedExperimentRun1.Id)
	require.NoError(t, err, "error inserting metric 2 for run 1")
	err = service.InsertMetricHistory(metric1Run2, *savedExperimentRun2.Id)
	require.NoError(t, err, "error inserting metric 1 for run 2")
	err = service.InsertMetricHistory(metric2Run2, *savedExperimentRun2.Id)
	require.NoError(t, err, "error inserting metric 2 for run 2")

	// Test getting all metric history for all experiment runs (bulk endpoint)
	result, err := service.GetExperimentRunMetricHistory(nil, nil, api.ListOptions{}, nil)
	require.NoError(t, err, "error getting bulk metric history")
	assert.GreaterOrEqual(t, int(result.Size), 4, "should return at least 4 metric history records")
	assert.GreaterOrEqual(t, len(result.Items), 4, "should have at least 4 items in the result")

	// Debug: print information about returned metrics
	t.Logf("Found %d total metrics in bulk query", len(result.Items))
	for i, item := range result.Items {
		nameStr := "<nil>"
		if item.Name != nil {
			nameStr = *item.Name
		}
		expRunIdStr := "<nil>"
		if item.ExperimentRunId != nil {
			expRunIdStr = *item.ExperimentRunId
		}
		expIdStr := "<nil>"
		if item.ExperimentId != nil {
			expIdStr = *item.ExperimentId
		}
		t.Logf("Metric %d: name=%s, experimentRunId=%s, experimentId=%s",
			i, nameStr, expRunIdStr, expIdStr)
	}

	// Verify we have metrics from both experiment runs
	run1MetricsFound := 0
	run2MetricsFound := 0
	for _, item := range result.Items {
		// For metrics we inserted, experiment fields should be populated
		// But bulk query might return other metrics too, so we only check our specific ones
		if item.ExperimentRunId != nil {
			switch *item.ExperimentRunId {
			case *savedExperimentRun1.Id:
				run1MetricsFound++
				// Verify experiment fields are populated for our metrics
				assert.Equal(t, savedExperiment.Id, item.ExperimentId, "experimentId should be populated for run 1 metrics")
			case *savedExperimentRun2.Id:
				run2MetricsFound++
				// Verify experiment fields are populated for our metrics
				assert.Equal(t, savedExperiment.Id, item.ExperimentId, "experimentId should be populated for run 2 metrics")
			}
		}
	}
	assert.GreaterOrEqual(t, run1MetricsFound, 2, "should find at least 2 metrics from experiment run 1")
	assert.GreaterOrEqual(t, run2MetricsFound, 2, "should find at least 2 metrics from experiment run 2")

	// Test filtering by name in bulk mode
	accuracyName := "accuracy"
	result, err = service.GetExperimentRunMetricHistory(&accuracyName, nil, api.ListOptions{}, nil)
	require.NoError(t, err, "error getting bulk metric history with name filter")
	assert.GreaterOrEqual(t, int(result.Size), 2, "should return at least 2 accuracy metrics")
	for _, item := range result.Items {
		assert.Equal(t, "accuracy", *item.Name, "all returned metrics should be accuracy metrics")
	}

	// Test filtering by step IDs in bulk mode
	stepIds := "1"
	result, err = service.GetExperimentRunMetricHistory(nil, &stepIds, api.ListOptions{}, nil)
	require.NoError(t, err, "error getting bulk metric history with step filter")
	assert.GreaterOrEqual(t, int(result.Size), 2, "should return at least 2 metrics from step 1")
	for _, item := range result.Items {
		assert.Equal(t, int64(1), *item.Step, "all returned metrics should be from step 1")
	}
}

func TestGetExperimentRunMetricHistoryExperimentFields(t *testing.T) {
	service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	// Create experiment
	experiment := &openapi.Experiment{
		Name:        "test-experiment-fields",
		Description: new("Test experiment for validating experiment fields in metric history"),
	}
	savedExperiment, err := service.UpsertExperiment(experiment)
	require.NoError(t, err)

	// Create experiment run
	experimentRun := &openapi.ExperimentRun{
		Name:        new("test-experiment-run-fields"),
		Description: new("Test experiment run for validating experiment fields in metric history"),
	}
	savedExperimentRun, err := service.UpsertExperimentRun(experimentRun, savedExperiment.Id)
	require.NoError(t, err)

	// Create metrics via UpsertExperimentRunArtifact (this should populate experiment fields)
	metricArtifact1 := &openapi.Artifact{
		Metric: &openapi.Metric{
			Name:        new("accuracy"),
			Value:       new(0.95),
			Timestamp:   new("1234567890"),
			Step:        new(int64(1)),
			Description: new("Test accuracy metric"),
		},
	}

	metricArtifact2 := &openapi.Artifact{
		Metric: &openapi.Metric{
			Name:        new("precision"),
			Value:       new(0.87),
			Timestamp:   new("1234567891"),
			Step:        new(int64(2)),
			Description: new("Test precision metric"),
		},
	}

	// Upsert metric artifacts through experiment run (should trigger metric history creation with experiment fields)
	createdArtifact1, err := service.UpsertExperimentRunArtifact(metricArtifact1, *savedExperimentRun.Id)
	require.NoError(t, err, "error upserting metric artifact 1")
	assert.NotNil(t, createdArtifact1.Metric, "should have created metric artifact 1")

	createdArtifact2, err := service.UpsertExperimentRunArtifact(metricArtifact2, *savedExperimentRun.Id)
	require.NoError(t, err, "error upserting metric artifact 2")
	assert.NotNil(t, createdArtifact2.Metric, "should have created metric artifact 2")

	// Verify the created artifacts have experiment fields
	assert.Equal(t, savedExperiment.Id, createdArtifact1.Metric.ExperimentId, "created artifact 1 should have experimentId")
	assert.Equal(t, savedExperimentRun.Id, createdArtifact1.Metric.ExperimentRunId, "created artifact 1 should have experimentRunId")
	assert.Equal(t, savedExperiment.Id, createdArtifact2.Metric.ExperimentId, "created artifact 2 should have experimentId")
	assert.Equal(t, savedExperimentRun.Id, createdArtifact2.Metric.ExperimentRunId, "created artifact 2 should have experimentRunId")

	// Get metric history and verify experiment fields are populated
	result, err := service.GetExperimentRunMetricHistory(nil, nil, api.ListOptions{}, savedExperimentRun.Id)
	require.NoError(t, err, "error getting metric history")
	assert.Equal(t, int32(2), result.Size, "should return 2 metric history records")
	assert.Equal(t, 2, len(result.Items), "should have 2 items in the result")

	// Verify ALL metrics in history have experiment fields populated
	for i, item := range result.Items {
		assert.Equal(t, savedExperiment.Id, item.ExperimentId, "metric %d should have experimentId populated", i)
		assert.Equal(t, savedExperimentRun.Id, item.ExperimentRunId, "metric %d should have experimentRunId populated", i)
		assert.NotNil(t, item.CustomProperties, "metric %d customProperties should not be nil", i)
		assert.Empty(t, item.CustomProperties, "metric %d customProperties should be empty (no duplicate experiment fields)", i)
		assert.Equal(t, "metric", *item.ArtifactType, "metric %d should have correct artifactType", i)
	}

	// Verify specific metric properties
	foundAccuracy := false
	foundPrecision := false
	for _, item := range result.Items {
		switch *item.Name {
		case "accuracy":
			foundAccuracy = true
			assert.Equal(t, 0.95, *item.Value)
			assert.Equal(t, "1234567890", *item.Timestamp)
			assert.Equal(t, int64(1), *item.Step)
		case "precision":
			foundPrecision = true
			assert.Equal(t, 0.87, *item.Value)
			assert.Equal(t, "1234567891", *item.Timestamp)
			assert.Equal(t, int64(2), *item.Step)
		}
	}
	assert.True(t, foundAccuracy, "should find accuracy metric in history")
	assert.True(t, foundPrecision, "should find precision metric in history")

	// Test filtering by name still preserves experiment fields
	accuracyName := "accuracy"
	filteredResult, err := service.GetExperimentRunMetricHistory(&accuracyName, nil, api.ListOptions{}, savedExperimentRun.Id)
	require.NoError(t, err, "error getting filtered metric history")
	assert.Equal(t, int32(1), filteredResult.Size, "should return 1 filtered metric history record")
	assert.Equal(t, 1, len(filteredResult.Items), "should have 1 item in filtered result")

	// Verify experiment fields are preserved in filtered results
	filteredMetric := &filteredResult.Items[0]
	assert.Equal(t, "accuracy", *filteredMetric.Name)
	assert.Equal(t, savedExperiment.Id, filteredMetric.ExperimentId, "filtered metric should have experimentId")
	assert.Equal(t, savedExperimentRun.Id, filteredMetric.ExperimentRunId, "filtered metric should have experimentRunId")
	assert.NotNil(t, filteredMetric.CustomProperties, "filtered metric customProperties should not be nil")
	assert.Empty(t, filteredMetric.CustomProperties, "filtered metric customProperties should be empty")
}
