package modelcatalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMetadataJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		want     metadataJSON
		wantErr  bool
	}{
		{
			name: "complete metadata with all core fields",
			jsonData: `{
				"id": "test-model-123",
				"description": "A test model for unit testing",
				"readme": "# Test Model\nThis is a test model.",
				"maturity": "stable",
				"languages": ["python", "go"],
				"tasks": ["classification", "regression"],
				"provider_name": "test-provider",
				"logo": "https://example.com/logo.png",
				"license": "MIT",
				"license_link": "https://opensource.org/licenses/MIT",
				"library_name": "test-library",
				"created_at": 1609459200,
				"updated_at": 1609545600
			}`,
			want: metadataJSON{
				ID: "test-model-123",
			},
			wantErr: false,
		},
		{
			name: "minimal metadata with only required fields",
			jsonData: `{
				"id": "minimal-model"
			}`,
			want: metadataJSON{
				ID: "minimal-model",
			},
			wantErr: false,
		},
		{
			name: "metadata with custom properties",
			jsonData: `{
				"id": "custom-model",
				"description": "Model with custom properties",
				"custom_field_string": "custom value",
				"custom_field_number": 42,
				"custom_field_float": 3.14,
				"custom_field_bool": true,
				"custom_field_array": ["item1", "item2"],
				"custom_field_object": {"nested": "value"}
			}`,
			want: metadataJSON{
				ID: "custom-model",
			},
			wantErr: false,
		},
		{
			name: "metadata with mixed core and custom fields",
			jsonData: `{
				"id": "mixed-model",
				"description": "Mixed fields model",
				"languages": ["python"],
				"custom_version": "1.0.0",
				"custom_tags": ["ml", "ai"],
				"custom_config": {
					"batch_size": 32,
					"learning_rate": 0.001
				}
			}`,
			want: metadataJSON{
				ID: "mixed-model",
			},
			wantErr: false,
		},
		{
			name: "empty arrays and objects",
			jsonData: `{
				"id": "empty-arrays-model",
				"languages": [],
				"tasks": [],
				"custom_empty_array": [],
				"custom_empty_object": {}
			}`,
			want: metadataJSON{
				ID: "empty-arrays-model",
			},
			wantErr: false,
		},
		{
			name: "zero timestamps",
			jsonData: `{
				"id": "zero-timestamps-model",
				"created_at": 0,
				"updated_at": 0
			}`,
			want: metadataJSON{
				ID: "zero-timestamps-model",
			},
			wantErr: false,
		},
		{
			name: "null values in custom properties",
			jsonData: `{
				"id": "null-values-model",
				"custom_null_field": null,
				"custom_string": "not null"
			}`,
			want: metadataJSON{
				ID: "null-values-model",
			},
			wantErr: false,
		},
		{
			name:     "invalid JSON",
			jsonData: `{"id": "invalid-json", "description":}`,
			want:     metadataJSON{},
			wantErr:  true,
		},
		{
			name:     "empty JSON object",
			jsonData: `{}`,
			want:     metadataJSON{},
			wantErr:  true, // Should error because ID is required
		},
		{
			name:     "missing ID field",
			jsonData: `{"description": "has description but no id"}`,
			want:     metadataJSON{},
			wantErr:  true, // Should error because ID is required
		},
		{
			name: "JSON with type mismatches should fail",
			jsonData: `{
				"id": 123,
				"languages": "not-an-array",
				"created_at": "not-a-number"
			}`,
			want:    metadataJSON{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMetadataJSON([]byte(tt.jsonData))

			if (err != nil) != tt.wantErr {
				t.Errorf("parseMetadataJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return // If we expected an error and got one, we're done
			}

			// Compare all fields
			if got.ID != tt.want.ID {
				t.Errorf("parseMetadataJSON() ID = %v, want %v", got.ID, tt.want.ID)
			}
		})
	}
}

func TestParseMetadataJSON_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		wantErr  bool
	}{
		{
			name:     "null JSON",
			jsonData: `null`,
			wantErr:  true, // Should error because ID will be empty
		},
		{
			name:     "array instead of object",
			jsonData: `["not", "an", "object"]`,
			wantErr:  true,
		},
		{
			name:     "string instead of object",
			jsonData: `"not an object"`,
			wantErr:  true,
		},
		{
			name:     "number instead of object",
			jsonData: `42`,
			wantErr:  true,
		},
		{
			name:     "boolean instead of object",
			jsonData: `true`,
			wantErr:  true,
		},
		{
			name:     "empty string",
			jsonData: ``,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseMetadataJSON([]byte(tt.jsonData))

			if (err != nil) != tt.wantErr {
				t.Errorf("parseMetadataJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseMetadataJSON_OnlyIDMatters(t *testing.T) {
	// Test that only the ID field is extracted, other fields are ignored
	jsonData := `{
		"id": "test-id",
		"custom_field": "ignored"
	}`

	metadata, err := parseMetadataJSON([]byte(jsonData))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify that only the ID field is populated
	if metadata.ID != "test-id" {
		t.Errorf("ID = %v, want %v", metadata.ID, "test-id")
	}
}

func TestOverallAccuracyToOverallAverage(t *testing.T) {
	t.Run("parse overall_accuracy from metadata", func(t *testing.T) {
		tests := []struct {
			name      string
			jsonData  string
			wantNil   bool
			wantValue float64
		}{
			{
				name:      "overall_accuracy present",
				jsonData:  `{"id": "model-1", "overall_accuracy": 85.5}`,
				wantNil:   false,
				wantValue: 85.5,
			},
			{
				name:      "overall_accuracy is zero",
				jsonData:  `{"id": "model-2", "overall_accuracy": 0}`,
				wantNil:   false,
				wantValue: 0.0,
			},
			{
				name:     "overall_accuracy is null",
				jsonData: `{"id": "model-3", "overall_accuracy": null}`,
				wantNil:  true,
			},
			{
				name:     "overall_accuracy missing",
				jsonData: `{"id": "model-4"}`,
				wantNil:  true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				metadata, err := parseMetadataJSON([]byte(tt.jsonData))
				if err != nil {
					t.Fatalf("parseMetadataJSON() error = %v", err)
				}

				if tt.wantNil {
					if metadata.OverallAccuracy != nil {
						t.Errorf("OverallAccuracy = %v, want nil", *metadata.OverallAccuracy)
					}
				} else {
					if metadata.OverallAccuracy == nil {
						t.Errorf("OverallAccuracy = nil, want %v", tt.wantValue)
					} else if *metadata.OverallAccuracy != tt.wantValue {
						t.Errorf("OverallAccuracy = %v, want %v", *metadata.OverallAccuracy, tt.wantValue)
					}
				}
			})
		}
	})

	t.Run("artifact has overall_average when overall_accuracy provided", func(t *testing.T) {
		overallAccuracy := 87.5
		evalRecords := []evaluationRecord{
			{Benchmark: "mmlu", CustomProperties: map[string]any{"score": 90.0}},
		}

		artifact := createAccuracyMetricsArtifact(evalRecords, 1, 100, &overallAccuracy, nil, nil)

		found := false
		for _, prop := range *artifact.CustomProperties {
			if prop.Name == "overall_average" && prop.DoubleValue != nil {
				if *prop.DoubleValue != overallAccuracy {
					t.Errorf("overall_average = %v, want %v", *prop.DoubleValue, overallAccuracy)
				}
				found = true
				break
			}
		}
		if !found {
			t.Error("overall_average custom property not found in artifact")
		}
	})

	t.Run("artifact has no overall_average when overall_accuracy is nil", func(t *testing.T) {
		evalRecords := []evaluationRecord{
			{Benchmark: "mmlu", CustomProperties: map[string]any{"score": 90.0}},
		}

		artifact := createAccuracyMetricsArtifact(evalRecords, 1, 100, nil, nil, nil)

		for _, prop := range *artifact.CustomProperties {
			if prop.Name == "overall_average" {
				t.Error("overall_average should not exist when overall_accuracy is nil")
			}
		}
	})
}

func TestCreateAccuracyMetricsArtifact_DuplicateBenchmarks(t *testing.T) {
	t.Run("result field is accepted as the score for the proposed schema", func(t *testing.T) {
		result := 0.72
		evalRecords := []evaluationRecord{
			{Benchmark: "mmlu", Result: &result, CustomProperties: map[string]any{"result": result}},
		}

		artifact := createAccuracyMetricsArtifact(evalRecords, 1, 100, nil, nil, nil)

		require.Len(t, *artifact.CustomProperties, 1)
		require.NotNil(t, (*artifact.CustomProperties)[0].DoubleValue)
		assert.Equal(t, "mmlu", (*artifact.CustomProperties)[0].Name)
		assert.Equal(t, result, *(*artifact.CustomProperties)[0].DoubleValue)
	})

	t.Run("duplicate benchmarks are deduplicated using last score", func(t *testing.T) {
		evalRecords := []evaluationRecord{
			{Benchmark: "mmlu", CustomProperties: map[string]any{"score": 80.0}},
			{Benchmark: "aime24", CustomProperties: map[string]any{"score": 63.3}},
			{Benchmark: "mmlu", CustomProperties: map[string]any{"score": 85.0}},
		}

		artifact := createAccuracyMetricsArtifact(evalRecords, 1, 100, nil, nil, nil)

		// Count occurrences of each benchmark name
		benchmarkCounts := map[string]int{}
		benchmarkScores := map[string]float64{}
		for _, prop := range *artifact.CustomProperties {
			benchmarkCounts[prop.Name]++
			if prop.DoubleValue != nil {
				benchmarkScores[prop.Name] = *prop.DoubleValue
			}
		}

		// "mmlu" should appear exactly once (deduplicated)
		if benchmarkCounts["mmlu"] != 1 {
			t.Errorf("expected mmlu to appear once, got %d", benchmarkCounts["mmlu"])
		}

		// The last score (85.0) should win
		if benchmarkScores["mmlu"] != 85.0 {
			t.Errorf("expected mmlu score 85.0, got %v", benchmarkScores["mmlu"])
		}

		// "aime24" should still be present
		if benchmarkCounts["aime24"] != 1 {
			t.Errorf("expected aime24 to appear once, got %d", benchmarkCounts["aime24"])
		}
		if benchmarkScores["aime24"] != 63.3 {
			t.Errorf("expected aime24 score 63.3, got %v", benchmarkScores["aime24"])
		}
	})

	t.Run("no duplicates produces all benchmarks", func(t *testing.T) {
		evalRecords := []evaluationRecord{
			{Benchmark: "mmlu", CustomProperties: map[string]any{"score": 90.0}},
			{Benchmark: "aime24", CustomProperties: map[string]any{"score": 63.3}},
			{Benchmark: "gpqa", CustomProperties: map[string]any{"score": 72.5}},
		}

		artifact := createAccuracyMetricsArtifact(evalRecords, 1, 100, nil, nil, nil)

		benchmarkNames := map[string]bool{}
		for _, prop := range *artifact.CustomProperties {
			benchmarkNames[prop.Name] = true
		}

		for _, expected := range []string{"mmlu", "aime24", "gpqa"} {
			if !benchmarkNames[expected] {
				t.Errorf("expected benchmark %q not found in custom properties", expected)
			}
		}
	})

	t.Run("all records with same benchmark produce single property", func(t *testing.T) {
		evalRecords := []evaluationRecord{
			{Benchmark: "mmlu", CustomProperties: map[string]any{"score": 80.0}},
			{Benchmark: "mmlu", CustomProperties: map[string]any{"score": 82.0}},
			{Benchmark: "mmlu", CustomProperties: map[string]any{"score": 85.0}},
		}

		artifact := createAccuracyMetricsArtifact(evalRecords, 1, 100, nil, nil, nil)

		count := 0
		for _, prop := range *artifact.CustomProperties {
			if prop.Name == "mmlu" {
				count++
			}
		}

		if count != 1 {
			t.Errorf("expected exactly 1 mmlu property, got %d", count)
		}
	})
}

func TestEvaluationRecordUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name             string
		jsonData         string
		wantModelID      string
		wantBenchmark    string
		wantCustomProps  map[string]any
		wantErr          bool
		checkCustomProps bool
	}{
		{
			name: "complete evaluation record",
			jsonData: `{
				"model_id": "test-model-123",
				"benchmark": "aime24",
				"score": 63.3333,
				"created_at": 1609459200,
				"updated_at": 1609545600
			}`,
			wantModelID:   "test-model-123",
			wantBenchmark: "aime24",
			wantCustomProps: map[string]any{
				"model_id":   "test-model-123",
				"benchmark":  "aime24",
				"score":      63.3333,
				"created_at": float64(1609459200),
				"updated_at": float64(1609545600),
			},
			wantErr:          false,
			checkCustomProps: true,
		},
		{
			name: "minimal evaluation record with only core fields",
			jsonData: `{
				"model_id": "minimal-model",
				"benchmark": "test-benchmark"
			}`,
			wantModelID:   "minimal-model",
			wantBenchmark: "test-benchmark",
			wantCustomProps: map[string]any{
				"model_id":  "minimal-model",
				"benchmark": "test-benchmark",
			},
			wantErr:          false,
			checkCustomProps: true,
		},
		{
			name: "evaluation record with custom properties",
			jsonData: `{
				"model_id": "custom-model",
				"benchmark": "custom-bench",
				"score": 95.5,
				"custom_field_string": "custom value",
				"custom_field_number": 42,
				"custom_field_float": 3.14,
				"custom_field_bool": true
			}`,
			wantModelID:   "custom-model",
			wantBenchmark: "custom-bench",
			wantCustomProps: map[string]any{
				"model_id":            "custom-model",
				"benchmark":           "custom-bench",
				"score":               95.5,
				"custom_field_string": "custom value",
				"custom_field_number": float64(42),
				"custom_field_float":  3.14,
				"custom_field_bool":   true,
			},
			wantErr:          false,
			checkCustomProps: true,
		},
		{
			name: "evaluation record with nested objects",
			jsonData: `{
				"model_id": "nested-model",
				"benchmark": "nested-bench",
				"custom_object": {
					"nested_key": "nested_value",
					"nested_number": 123
				},
				"custom_array": ["item1", "item2", "item3"]
			}`,
			wantModelID:      "nested-model",
			wantBenchmark:    "nested-bench",
			wantErr:          false,
			checkCustomProps: false, // Don't check deep equality for complex nested structures
		},
		{
			name: "evaluation record with null values",
			jsonData: `{
				"model_id": "null-model",
				"benchmark": "null-bench",
				"null_field": null,
				"score": 50.0
			}`,
			wantModelID:   "null-model",
			wantBenchmark: "null-bench",
			wantCustomProps: map[string]any{
				"model_id":   "null-model",
				"benchmark":  "null-bench",
				"null_field": nil,
				"score":      50.0,
			},
			wantErr:          false,
			checkCustomProps: true,
		},
		{
			name: "evaluation record missing core fields",
			jsonData: `{
				"score": 75.5,
				"created_at": 1609459200
			}`,
			wantModelID:      "",
			wantBenchmark:    "",
			wantErr:          false,
			checkCustomProps: false,
		},
		{
			name: "evaluation record with wrong type for core fields",
			jsonData: `{
				"model_id": 123,
				"benchmark": 456,
				"score": 85.0
			}`,
			wantModelID:      "",
			wantBenchmark:    "",
			wantErr:          false,
			checkCustomProps: false,
		},
		{
			name:             "empty JSON object",
			jsonData:         `{}`,
			wantModelID:      "",
			wantBenchmark:    "",
			wantErr:          false,
			checkCustomProps: false,
		},
		{
			name:             "invalid JSON",
			jsonData:         `{"model_id": "invalid", "benchmark":}`,
			wantErr:          true,
			checkCustomProps: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var er evaluationRecord
			err := er.UnmarshalJSON([]byte(tt.jsonData))

			if (err != nil) != tt.wantErr {
				t.Errorf("evaluationRecord.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return // If we expected an error and got one, we're done
			}

			// Check core fields
			if er.ModelID != tt.wantModelID {
				t.Errorf("ModelID = %v, want %v", er.ModelID, tt.wantModelID)
			}
			if er.Benchmark != tt.wantBenchmark {
				t.Errorf("Benchmark = %v, want %v", er.Benchmark, tt.wantBenchmark)
			}

			// Check CustomProperties
			if er.CustomProperties == nil {
				t.Error("CustomProperties should not be nil")
			}

			// Optionally check custom properties in detail
			if tt.checkCustomProps {
				if len(er.CustomProperties) != len(tt.wantCustomProps) {
					t.Errorf("CustomProperties length = %v, want %v", len(er.CustomProperties), len(tt.wantCustomProps))
				}
				for key, wantValue := range tt.wantCustomProps {
					gotValue, exists := er.CustomProperties[key]
					if !exists {
						t.Errorf("CustomProperties missing key %v", key)
						continue
					}
					if gotValue != wantValue {
						t.Errorf("CustomProperties[%v] = %v (type %T), want %v (type %T)",
							key, gotValue, gotValue, wantValue, wantValue)
					}
				}
			}
		})
	}
}

func TestPerformanceRecordUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name             string
		jsonData         string
		wantID           string
		wantModelID      string
		wantCustomProps  map[string]any
		wantErr          bool
		checkCustomProps bool
	}{
		{
			name: "complete performance record",
			jsonData: `{
				"id": "perf-123",
				"model_id": "test-model-456",
				"throughput": 1000.5,
				"latency_p50": 10.5,
				"latency_p95": 25.3,
				"latency_p99": 50.1,
				"created_at": 1609459200,
				"updated_at": 1609545600
			}`,
			wantID:      "perf-123",
			wantModelID: "test-model-456",
			wantCustomProps: map[string]any{
				"id":          "perf-123",
				"model_id":    "test-model-456",
				"throughput":  1000.5,
				"latency_p50": 10.5,
				"latency_p95": 25.3,
				"latency_p99": 50.1,
				"created_at":  float64(1609459200),
				"updated_at":  float64(1609545600),
			},
			wantErr:          false,
			checkCustomProps: true,
		},
		{
			name: "minimal performance record with only core fields",
			jsonData: `{
				"id": "minimal-perf",
				"model_id": "minimal-model"
			}`,
			wantID:      "minimal-perf",
			wantModelID: "minimal-model",
			wantCustomProps: map[string]any{
				"id":       "minimal-perf",
				"model_id": "minimal-model",
			},
			wantErr:          false,
			checkCustomProps: true,
		},
		{
			name: "performance record with custom properties",
			jsonData: `{
				"id": "custom-perf",
				"model_id": "custom-model",
				"throughput": 500.0,
				"custom_field_string": "custom value",
				"custom_field_number": 42,
				"custom_field_float": 3.14,
				"custom_field_bool": true
			}`,
			wantID:      "custom-perf",
			wantModelID: "custom-model",
			wantCustomProps: map[string]any{
				"id":                  "custom-perf",
				"model_id":            "custom-model",
				"throughput":          500.0,
				"custom_field_string": "custom value",
				"custom_field_number": float64(42),
				"custom_field_float":  3.14,
				"custom_field_bool":   true,
			},
			wantErr:          false,
			checkCustomProps: true,
		},
		{
			name: "performance record with nested objects and arrays",
			jsonData: `{
				"id": "nested-perf",
				"model_id": "nested-model",
				"custom_object": {
					"nested_key": "nested_value",
					"nested_number": 123
				},
				"custom_array": ["item1", "item2", "item3"]
			}`,
			wantID:           "nested-perf",
			wantModelID:      "nested-model",
			wantErr:          false,
			checkCustomProps: false, // Don't check deep equality for complex nested structures
		},
		{
			name: "performance record with null values",
			jsonData: `{
				"id": "null-perf",
				"model_id": "null-model",
				"null_field": null,
				"throughput": 250.0
			}`,
			wantID:      "null-perf",
			wantModelID: "null-model",
			wantCustomProps: map[string]any{
				"id":         "null-perf",
				"model_id":   "null-model",
				"null_field": nil,
				"throughput": 250.0,
			},
			wantErr:          false,
			checkCustomProps: true,
		},
		{
			name: "performance record missing core fields",
			jsonData: `{
				"throughput": 100.0,
				"latency_p50": 5.0
			}`,
			wantID:           "",
			wantModelID:      "",
			wantErr:          false,
			checkCustomProps: false,
		},
		{
			name: "performance record with wrong type for core fields",
			jsonData: `{
				"id": 123,
				"model_id": 456,
				"throughput": 500.0
			}`,
			wantID:           "",
			wantModelID:      "",
			wantErr:          false,
			checkCustomProps: false,
		},
		{
			name: "performance record with zero values",
			jsonData: `{
				"id": "zero-perf",
				"model_id": "zero-model",
				"throughput": 0,
				"latency_p50": 0.0,
				"created_at": 0
			}`,
			wantID:      "zero-perf",
			wantModelID: "zero-model",
			wantCustomProps: map[string]any{
				"id":          "zero-perf",
				"model_id":    "zero-model",
				"throughput":  float64(0),
				"latency_p50": 0.0,
				"created_at":  float64(0),
			},
			wantErr:          false,
			checkCustomProps: true,
		},
		{
			name: "performance record with cold_start and runtime_command",
			jsonData: `{
				"id": "perf-coldstart-001",
				"model_id": "certified/production-llm-8b",
				"requests_per_second": 4.0,
				"ttft_p90": 45.8,
				"hardware_type": "H100",
				"hardware_count": 2,
				"cold_start_time_to_load_seconds": 38.1,
				"runtime_command": "python3 -m vllm.entrypoints.openai.api_server --model certified/production-llm-8b --tensor-parallel-size 2"
			}`,
			wantID:      "perf-coldstart-001",
			wantModelID: "certified/production-llm-8b",
			wantCustomProps: map[string]any{
				"id":                              "perf-coldstart-001",
				"model_id":                        "certified/production-llm-8b",
				"requests_per_second":             4.0,
				"ttft_p90":                        45.8,
				"hardware_type":                   "H100",
				"hardware_count":                  float64(2),
				"cold_start_time_to_load_seconds": 38.1,
				"runtime_command":                 "python3 -m vllm.entrypoints.openai.api_server --model certified/production-llm-8b --tensor-parallel-size 2",
			},
			wantErr:          false,
			checkCustomProps: true,
		},
		{
			name: "performance record with cold_start zero value",
			jsonData: `{
				"id": "perf-coldstart-zero",
				"model_id": "test/model",
				"requests_per_second": 2.0,
				"hardware_type": "A100-80",
				"hardware_count": 1,
				"cold_start_time_to_load_seconds": 0,
				"runtime_command": ""
			}`,
			wantID:      "perf-coldstart-zero",
			wantModelID: "test/model",
			wantCustomProps: map[string]any{
				"id":                              "perf-coldstart-zero",
				"model_id":                        "test/model",
				"requests_per_second":             2.0,
				"hardware_type":                   "A100-80",
				"hardware_count":                  float64(1),
				"cold_start_time_to_load_seconds": float64(0),
				"runtime_command":                 "",
			},
			wantErr:          false,
			checkCustomProps: true,
		},
		{
			name:             "empty JSON object",
			jsonData:         `{}`,
			wantID:           "",
			wantModelID:      "",
			wantErr:          false,
			checkCustomProps: false,
		},
		{
			name:             "invalid JSON",
			jsonData:         `{"id": "invalid", "model_id":}`,
			wantErr:          true,
			checkCustomProps: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pr performanceRecord
			err := pr.UnmarshalJSON([]byte(tt.jsonData))

			if (err != nil) != tt.wantErr {
				t.Errorf("performanceRecord.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return // If we expected an error and got one, we're done
			}

			// Check core fields
			if pr.ID != tt.wantID {
				t.Errorf("ID = %v, want %v", pr.ID, tt.wantID)
			}
			if pr.ModelID != tt.wantModelID {
				t.Errorf("ModelID = %v, want %v", pr.ModelID, tt.wantModelID)
			}

			// Check CustomProperties
			if pr.CustomProperties == nil {
				t.Error("CustomProperties should not be nil")
			}

			// Optionally check custom properties in detail
			if tt.checkCustomProps {
				if len(pr.CustomProperties) != len(tt.wantCustomProps) {
					t.Errorf("CustomProperties length = %v, want %v", len(pr.CustomProperties), len(tt.wantCustomProps))
				}
				for key, wantValue := range tt.wantCustomProps {
					gotValue, exists := pr.CustomProperties[key]
					if !exists {
						t.Errorf("CustomProperties missing key %v", key)
						continue
					}

					// Translate json.Number values
					if jsonNumber, ok := gotValue.(json.Number); ok {
						var newValue any
						switch wantValue.(type) {
						case float64:
							newValue, err = jsonNumber.Float64()
						case int, int32, int64:
							newValue, err = jsonNumber.Int64()
						}
						if err == nil {
							gotValue = newValue
						}
					}

					if gotValue != wantValue {
						t.Errorf("CustomProperties[%v] = %v (type %T), want %v (type %T)",
							key, gotValue, gotValue, wantValue, wantValue)
					}
				}
			}
		})
	}
}

func TestEvaluationRecordUnmarshalJSON_CoreFieldsInCustomProperties(t *testing.T) {
	// Test that core fields are included in CustomProperties
	jsonData := `{
		"model_id": "test-model",
		"benchmark": "test-benchmark",
		"score": 90.5
	}`

	var er evaluationRecord
	err := er.UnmarshalJSON([]byte(jsonData))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify core fields are in CustomProperties
	if er.CustomProperties["model_id"] != "test-model" {
		t.Errorf("CustomProperties[model_id] = %v, want %v", er.CustomProperties["model_id"], "test-model")
	}
	if er.CustomProperties["benchmark"] != "test-benchmark" {
		t.Errorf("CustomProperties[benchmark] = %v, want %v", er.CustomProperties["benchmark"], "test-benchmark")
	}
	if er.CustomProperties["score"] != 90.5 {
		t.Errorf("CustomProperties[score] = %v, want %v", er.CustomProperties["score"], 90.5)
	}
}

func TestDetailedEvaluationRecord(t *testing.T) {
	const modelID = "RedHatAI/example-model"
	jsonData := `{
		"id": "evaluation-result-1",
		"model_id": "RedHatAI/example-model",
		"run_id": "evaluation-run-1",
		"evaluation": "General language understanding",
		"category": "general-llm",
		"benchmark": "mmlu",
		"description": "Measures multitask language understanding.",
		"result": 0,
		"result_metric": "accuracy",
		"created_at": 1747112286087,
		"updated_at": 1747548472571
	}`

	var record evaluationRecord
	if err := json.Unmarshal([]byte(jsonData), &record); err != nil {
		t.Fatalf("evaluationRecord.UnmarshalJSON() error = %v", err)
	}
	if err := record.validateDetailedEvaluation(modelID); err != nil {
		t.Fatalf("validateDetailedEvaluation() error = %v", err)
	}
	if record.Result == nil || *record.Result != 0 {
		t.Fatalf("Result = %v, want 0", record.Result)
	}
	if record.CreatedAt == nil || *record.CreatedAt != 1747112286087 {
		t.Fatalf("CreatedAt = %v, want 1747112286087", record.CreatedAt)
	}

	artifact := createEvaluationArtifact(record, 1, 100)
	if artifact.Attributes == nil {
		t.Fatal("Attributes should not be nil")
	}
	if artifact.Attributes.MetricsType != models.MetricsTypeEvaluation {
		t.Errorf("MetricsType = %q, want %q", artifact.Attributes.MetricsType, models.MetricsTypeEvaluation)
	}
	assert.Equal(t, "catalog-metrics:v1:1:evaluation-metrics:d42242c45069a85f74afa0877c78b0b201fd2223a238c38c06520586e3e5eeee", *artifact.Attributes.ExternalID)

	properties := make(map[string]any)
	for _, property := range *artifact.CustomProperties {
		switch {
		case property.StringValue != nil:
			properties[property.Name] = *property.StringValue
		case property.DoubleValue != nil:
			properties[property.Name] = *property.DoubleValue
		}
	}
	for name, expected := range map[string]any{
		"id":            "evaluation-result-1",
		"model_id":      modelID,
		"run_id":        "evaluation-run-1",
		"evaluation":    "General language understanding",
		"category":      "general-llm",
		"benchmark":     "mmlu",
		"description":   "Measures multitask language understanding.",
		"result":        float64(0),
		"result_metric": "accuracy",
	} {
		if actual := properties[name]; actual != expected {
			t.Errorf("property %q = %#v, want %#v", name, actual, expected)
		}
	}
	if _, found := properties["created_at"]; found {
		t.Error("created_at should be represented by CreateTimeSinceEpoch, not a custom property")
	}
}

func TestPerformanceArtifactConfigIDCorrelation(t *testing.T) {
	for _, idField := range []string{"", `"id":"",`, `"id":123,`} {
		t.Run(idField, func(t *testing.T) {
			var record performanceRecord
			require.NoError(t, json.Unmarshal([]byte(`{`+idField+`"config_id":"config-only","requests_per_second":10}`), &record))
			artifact := createPerformanceArtifact(record, 1, 100, nil, nil)
			properties := map[string]string{}
			for _, property := range *artifact.CustomProperties {
				if property.StringValue != nil {
					properties[property.Name] = *property.StringValue
				}
			}
			assert.Equal(t, "config-only", properties["id"])
			assert.Equal(t, "config-only", properties["config_id"])
		})
	}
}

func TestDetailedEvaluationValidation(t *testing.T) {
	createdAt := int64(1747112286087)
	result := 0.85
	valid := evaluationRecord{
		ID:           "evaluation-result-1",
		ModelID:      "RedHatAI/example-model",
		RunID:        "evaluation-run-1",
		Evaluation:   "General language understanding",
		Category:     "general-llm",
		Benchmark:    "mmlu",
		Description:  "Measures multitask language understanding.",
		Result:       &result,
		ResultMetric: "accuracy",
		CreatedAt:    &createdAt,
	}
	if err := valid.validateDetailedEvaluation(valid.ModelID); err != nil {
		t.Fatalf("valid record rejected: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*evaluationRecord)
	}{
		{name: "missing id", mutate: func(record *evaluationRecord) { record.ID = "" }},
		{name: "mismatched model", mutate: func(record *evaluationRecord) { record.ModelID = "Other/model" }},
		{name: "missing run id", mutate: func(record *evaluationRecord) { record.RunID = "" }},
		{name: "missing evaluation", mutate: func(record *evaluationRecord) { record.Evaluation = "" }},
		{name: "missing category", mutate: func(record *evaluationRecord) { record.Category = "" }},
		{name: "missing benchmark", mutate: func(record *evaluationRecord) { record.Benchmark = "" }},
		{name: "missing description", mutate: func(record *evaluationRecord) { record.Description = "" }},
		{name: "missing result", mutate: func(record *evaluationRecord) { record.Result = nil }},
		{name: "missing result metric", mutate: func(record *evaluationRecord) { record.ResultMetric = "" }},
		{name: "missing timestamp", mutate: func(record *evaluationRecord) { record.CreatedAt = nil }},
		{name: "invalid category", mutate: func(record *evaluationRecord) { record.Category = "General LLM" }},
		{name: "update before creation", mutate: func(record *evaluationRecord) {
			updatedAt := createdAt - 1
			record.UpdatedAt = &updatedAt
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := valid
			tt.mutate(&record)
			if err := record.validateDetailedEvaluation(valid.ModelID); err == nil {
				t.Fatal("validateDetailedEvaluation() error = nil, want error")
			}
		})
	}

	var fractionalTimestamp evaluationRecord
	if err := json.Unmarshal([]byte(`{"created_at": 1.5}`), &fractionalTimestamp); err != nil {
		t.Fatalf("evaluationRecord.UnmarshalJSON() error = %v", err)
	}
	if fractionalTimestamp.CreatedAt != nil {
		t.Errorf("CreatedAt = %v, want nil for fractional timestamp", fractionalTimestamp.CreatedAt)
	}

	var invalidOptionalTimestamp evaluationRecord
	if err := json.Unmarshal([]byte(`{
		"id":"result-1",
		"model_id":"RedHatAI/example-model",
		"run_id":"run-1",
		"evaluation":"General evaluation",
		"category":"general-llm",
		"benchmark":"mmlu",
		"description":"Description",
		"result":0.72,
		"result_metric":"accuracy",
		"created_at":1747112286087,
		"updated_at":"invalid"
	}`), &invalidOptionalTimestamp); err != nil {
		t.Fatalf("evaluationRecord.UnmarshalJSON() error = %v", err)
	}
	if err := invalidOptionalTimestamp.validateDetailedEvaluation(valid.ModelID); err == nil {
		t.Fatal("validateDetailedEvaluation() error = nil, want invalid updated_at error")
	}
}

func TestFilterEvaluationRecordsForModel(t *testing.T) {
	records := []evaluationRecord{
		{ID: "matching", ModelID: "RedHatAI/example-model"},
		{ID: "legacy-without-model-id"},
		{ID: "mismatched", ModelID: "Other/model"},
	}

	filtered := filterEvaluationRecordsForModel(records, "RedHatAI/example-model")
	if len(filtered) != 2 {
		t.Fatalf("len(filtered) = %d, want 2", len(filtered))
	}
	if filtered[0].ID != "matching" || filtered[1].ID != "legacy-without-model-id" {
		t.Errorf("filtered IDs = %q, %q", filtered[0].ID, filtered[1].ID)
	}
}

func TestPerformanceRecordUnmarshalJSON_CoreFieldsInCustomProperties(t *testing.T) {
	// Test that core fields are included in CustomProperties
	jsonData := `{
		"id": "perf-id",
		"model_id": "test-model",
		"throughput": 1000.0
	}`

	var pr performanceRecord
	err := pr.UnmarshalJSON([]byte(jsonData))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify core fields are in CustomProperties
	if pr.CustomProperties["id"] != "perf-id" {
		t.Errorf("CustomProperties[id] = %v, want %v", pr.CustomProperties["id"], "perf-id")
	}
	if pr.CustomProperties["model_id"] != "test-model" {
		t.Errorf("CustomProperties[model_id] = %v, want %v", pr.CustomProperties["model_id"], "test-model")
	}
	if v, _ := pr.CustomProperties["throughput"].(json.Number).Float64(); v != 1000.0 {
		t.Errorf("CustomProperties[throughput] = %v, want %v", pr.CustomProperties["throughput"], 1000.0)
	}
}

func TestPerformanceRecordUnmarshalJSON_ColdStartAndRuntimeCommand(t *testing.T) {
	jsonData := `{
		"id": "a1b2c3d4-1111-4aaa-bbbb-000000000004",
		"model_id": "certified/production-llm-8b",
		"config_id": "cfg-h100-2gpu-chatbot",
		"use_case": "chatbot",
		"ttft_p90": 45.8,
		"requests_per_second": 4.0,
		"hardware_type": "H100",
		"hardware_count": 2,
		"runtime_command": "python3 -m vllm.entrypoints.openai.api_server \\\n  --model certified/production-llm-8b \\\n  --max-model-len 8192 \\\n  --tensor-parallel-size 2 \\\n  --trust-remote-code",
		"cold_start_time_to_load_seconds": 38.1
	}`

	var pr performanceRecord
	err := pr.UnmarshalJSON([]byte(jsonData))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if pr.ID != "a1b2c3d4-1111-4aaa-bbbb-000000000004" {
		t.Errorf("ID = %v, want a1b2c3d4-1111-4aaa-bbbb-000000000004", pr.ID)
	}
	if pr.ModelID != "certified/production-llm-8b" {
		t.Errorf("ModelID = %v, want certified/production-llm-8b", pr.ModelID)
	}

	// Verify cold_start_time_to_load_seconds is captured as a custom property
	csValue, ok := pr.CustomProperties["cold_start_time_to_load_seconds"]
	if !ok {
		t.Fatal("cold_start_time_to_load_seconds not found in CustomProperties")
	}
	csFloat, err := csValue.(json.Number).Float64()
	if err != nil {
		t.Fatalf("cold_start_time_to_load_seconds is not a number: %v", err)
	}
	if csFloat != 38.1 {
		t.Errorf("cold_start_time_to_load_seconds = %v, want 38.1", csFloat)
	}

	// Verify runtime_command is captured as a custom property
	rtValue, ok := pr.CustomProperties["runtime_command"]
	if !ok {
		t.Fatal("runtime_command not found in CustomProperties")
	}
	rtStr, ok := rtValue.(string)
	if !ok {
		t.Fatalf("runtime_command is not a string: %T", rtValue)
	}
	if !strings.Contains(rtStr, "--tensor-parallel-size 2") {
		t.Errorf("runtime_command missing tensor-parallel-size: %v", rtStr)
	}
	if !strings.Contains(rtStr, "certified/production-llm-8b") {
		t.Errorf("runtime_command missing model name: %v", rtStr)
	}
}

func TestParsePerformanceFile_ColdStartFields(t *testing.T) {
	tmpDir := t.TempDir()

	ndjsonContent := `{"id": "perf-001", "model_id": "test/model-8b", "requests_per_second": 4.0, "ttft_p90": 45.8, "hardware_type": "H100", "hardware_count": 2, "cold_start_time_to_load_seconds": 38.1, "runtime_command": "vllm serve test/model-8b --tensor-parallel-size 2"}
{"id": "perf-002", "model_id": "test/model-8b", "requests_per_second": 2.0, "ttft_p90": 65.3, "hardware_type": "A100-80", "hardware_count": 1, "cold_start_time_to_load_seconds": 52.7, "runtime_command": "vllm serve test/model-8b --tensor-parallel-size 1"}
`
	filePath := filepath.Join(tmpDir, "performance.ndjson")
	if err := os.WriteFile(filePath, []byte(ndjsonContent), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	records, err := parsePerformanceFile(filePath)
	if err != nil {
		t.Fatalf("parsePerformanceFile() error = %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	// Verify first record
	cs1, err := records[0].CustomProperties["cold_start_time_to_load_seconds"].(json.Number).Float64()
	if err != nil {
		t.Fatalf("record[0] cold_start_time_to_load_seconds not a number: %v", err)
	}
	if cs1 != 38.1 {
		t.Errorf("record[0] cold_start_time_to_load_seconds = %v, want 38.1", cs1)
	}
	if records[0].CustomProperties["runtime_command"] != "vllm serve test/model-8b --tensor-parallel-size 2" {
		t.Errorf("record[0] runtime_command = %v", records[0].CustomProperties["runtime_command"])
	}

	// Verify second record
	cs2, err := records[1].CustomProperties["cold_start_time_to_load_seconds"].(json.Number).Float64()
	if err != nil {
		t.Fatalf("record[1] cold_start_time_to_load_seconds not a number: %v", err)
	}
	if cs2 != 52.7 {
		t.Errorf("record[1] cold_start_time_to_load_seconds = %v, want 52.7", cs2)
	}
	if records[1].CustomProperties["runtime_command"] != "vllm serve test/model-8b --tensor-parallel-size 1" {
		t.Errorf("record[1] runtime_command = %v", records[1].CustomProperties["runtime_command"])
	}
}

func TestUnmarshalJSON_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		wantErr  bool
	}{
		{
			name:     "null JSON for evaluationRecord",
			jsonData: `null`,
			wantErr:  false, // null JSON unmarshals to empty map, not an error
		},
		{
			name:     "array instead of object for evaluationRecord",
			jsonData: `["not", "an", "object"]`,
			wantErr:  true,
		},
		{
			name:     "string instead of object for evaluationRecord",
			jsonData: `"not an object"`,
			wantErr:  true,
		},
		{
			name:     "number instead of object",
			jsonData: `42`,
			wantErr:  true,
		},
		{
			name:     "boolean instead of object",
			jsonData: `true`,
			wantErr:  true,
		},
		{
			name:     "empty string",
			jsonData: ``,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" (evaluationRecord)", func(t *testing.T) {
			var er evaluationRecord
			err := er.UnmarshalJSON([]byte(tt.jsonData))

			if (err != nil) != tt.wantErr {
				t.Errorf("evaluationRecord.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})

		t.Run(tt.name+" (performanceRecord)", func(t *testing.T) {
			var pr performanceRecord
			err := pr.UnmarshalJSON([]byte(tt.jsonData))

			if (err != nil) != tt.wantErr {
				t.Errorf("performanceRecord.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})

		t.Run(tt.name+" (securityEvaluationRecord)", func(t *testing.T) {
			var sr securityEvaluationRecord
			err := sr.UnmarshalJSON([]byte(tt.jsonData))

			if (err != nil) != tt.wantErr {
				t.Errorf("securityEvaluationRecord.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseMetadataJSON_NewFields(t *testing.T) {
	tests := []struct {
		name                  string
		jsonData              string
		wantID                string
		wantSize              *string
		wantTensorType        *string
		wantVariantID         *string
		wantMinVRAMGB         *float64
		wantModelcarImageSize *float64
		wantErr               bool
	}{
		{
			name: "complete metadata with all new fields",
			jsonData: `{
				"id": "sample-model/test-8b-instruct",
				"size": "8B params",
				"tensor_type": "FP16",
				"variant_group_id": "abc123de-f456-789a-bcde-f0123456789a"
			}`,
			wantID:         "sample-model/test-8b-instruct",
			wantSize:       &[]string{"8B params"}[0],
			wantTensorType: &[]string{"FP16"}[0],
			wantVariantID:  &[]string{"abc123de-f456-789a-bcde-f0123456789a"}[0],
			wantErr:        false,
		},
		{
			name: "metadata with quantized model INT4",
			jsonData: `{
				"id": "sample-model/test-70b-quantized.w4a16",
				"size": "11B params",
				"tensor_type": "INT4",
				"variant_group_id": "def456ab-c789-012d-ef34-56789abcdef0"
			}`,
			wantID:         "sample-model/test-70b-quantized.w4a16",
			wantSize:       &[]string{"11B params"}[0],
			wantTensorType: &[]string{"INT4"}[0],
			wantVariantID:  &[]string{"def456ab-c789-012d-ef34-56789abcdef0"}[0],
			wantErr:        false,
		},
		{
			name: "metadata with different tensor types",
			jsonData: `{
				"id": "sample-model/test-bf16",
				"size": "13B params",
				"tensor_type": "BF16",
				"variant_group_id": "ghi789cd-e012-345g-hi67-89abcdef0123"
			}`,
			wantID:         "sample-model/test-bf16",
			wantSize:       &[]string{"13B params"}[0],
			wantTensorType: &[]string{"BF16"}[0],
			wantVariantID:  &[]string{"ghi789cd-e012-345g-hi67-89abcdef0123"}[0],
			wantErr:        false,
		},
		{
			name: "metadata with INT8 tensor type",
			jsonData: `{
				"id": "sample-model/test-int8",
				"size": "7B params",
				"tensor_type": "INT8",
				"variant_group_id": "jkl012ef-3456-789j-kl01-23456789abcd"
			}`,
			wantID:         "sample-model/test-int8",
			wantSize:       &[]string{"7B params"}[0],
			wantTensorType: &[]string{"INT8"}[0],
			wantVariantID:  &[]string{"jkl012ef-3456-789j-kl01-23456789abcd"}[0],
			wantErr:        false,
		},
		{
			name: "metadata missing all new fields",
			jsonData: `{
				"id": "sample-model/minimal-test"
			}`,
			wantID:         "sample-model/minimal-test",
			wantSize:       nil,
			wantTensorType: nil,
			wantVariantID:  nil,
			wantErr:        false,
		},
		{
			name: "metadata with null new fields",
			jsonData: `{
				"id": "sample-model/null-fields",
				"size": null,
				"tensor_type": null,
				"variant_group_id": null
			}`,
			wantID:         "sample-model/null-fields",
			wantSize:       nil,
			wantTensorType: nil,
			wantVariantID:  nil,
			wantErr:        false,
		},
		{
			name: "metadata with empty string new fields",
			jsonData: `{
				"id": "sample-model/empty-strings",
				"size": "",
				"tensor_type": "",
				"variant_group_id": ""
			}`,
			wantID:         "sample-model/empty-strings",
			wantSize:       &[]string{""}[0],
			wantTensorType: &[]string{""}[0],
			wantVariantID:  &[]string{""}[0],
			wantErr:        false,
		},
		{
			name: "metadata with partial new fields",
			jsonData: `{
				"id": "sample-model/partial-fields",
				"size": "15B params",
				"tensor_type": "FP8"
			}`,
			wantID:         "sample-model/partial-fields",
			wantSize:       &[]string{"15B params"}[0],
			wantTensorType: &[]string{"FP8"}[0],
			wantVariantID:  nil,
			wantErr:        false,
		},
		{
			name: "metadata with mixed precision format",
			jsonData: `{
				"id": "sample-model/mixed-precision",
				"size": "22B params",
				"tensor_type": "MXFP4",
				"variant_group_id": "mno345gh-6789-012m-no34-56789abcdef1"
			}`,
			wantID:         "sample-model/mixed-precision",
			wantSize:       &[]string{"22B params"}[0],
			wantTensorType: &[]string{"MXFP4"}[0],
			wantVariantID:  &[]string{"mno345gh-6789-012m-no34-56789abcdef1"}[0],
			wantErr:        false,
		},
		{
			name: "metadata with large model size",
			jsonData: `{
				"id": "sample-model/large-model",
				"size": "175B params",
				"tensor_type": "FP16",
				"variant_group_id": "pqr678ij-9abc-def0-pqr1-23456789abcd"
			}`,
			wantID:         "sample-model/large-model",
			wantSize:       &[]string{"175B params"}[0],
			wantTensorType: &[]string{"FP16"}[0],
			wantVariantID:  &[]string{"pqr678ij-9abc-def0-pqr1-23456789abcd"}[0],
			wantErr:        false,
		},
		{
			name: "metadata with decimal size",
			jsonData: `{
				"id": "sample-model/decimal-size",
				"size": "6.7B params",
				"tensor_type": "BF16",
				"variant_group_id": "stu901kl-2def-456s-tu90-123456789abc"
			}`,
			wantID:         "sample-model/decimal-size",
			wantSize:       &[]string{"6.7B params"}[0],
			wantTensorType: &[]string{"BF16"}[0],
			wantVariantID:  &[]string{"stu901kl-2def-456s-tu90-123456789abc"}[0],
			wantErr:        false,
		},
		{
			name: "metadata with vRAM field",
			jsonData: `{
				"id": "sample-model/test-405b-instruct",
				"size": "405B params",
				"tensor_type": "FP8",
				"variant_group_id": "vwx234mn-5678-901v-wx23-456789abcdef",
				"min_vram_gb": 265.0
			}`,
			wantID:         "sample-model/test-405b-instruct",
			wantSize:       &[]string{"405B params"}[0],
			wantTensorType: &[]string{"FP8"}[0],
			wantVariantID:  &[]string{"vwx234mn-5678-901v-wx23-456789abcdef"}[0],
			wantMinVRAMGB:  &[]float64{265.0}[0],
			wantErr:        false,
		},
		{
			name: "metadata with modelcar image size fields",
			jsonData: `{
				"id": "sample-model/test-405b-instruct",
				"size": "405B params",
				"tensor_type": "FP8",
				"min_vram_gb": 265.0,
				"modelcar_image_size": 405.19
			}`,
			wantID:                "sample-model/test-405b-instruct",
			wantSize:              &[]string{"405B params"}[0],
			wantTensorType:        &[]string{"FP8"}[0],
			wantMinVRAMGB:         &[]float64{265.0}[0],
			wantModelcarImageSize: &[]float64{405.19}[0],
			wantErr:               false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMetadataJSON([]byte(tt.jsonData))

			if (err != nil) != tt.wantErr {
				t.Errorf("parseMetadataJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			// Test ID field
			if got.ID != tt.wantID {
				t.Errorf("parseMetadataJSON() ID = %v, want %v", got.ID, tt.wantID)
			}

			// Test Size field
			if (got.Size == nil) != (tt.wantSize == nil) || (got.Size != nil && tt.wantSize != nil && *got.Size != *tt.wantSize) {
				t.Errorf("parseMetadataJSON() Size = %v, want %v", got.Size, tt.wantSize)
			}

			// Test TensorType field
			if (got.TensorType == nil) != (tt.wantTensorType == nil) || (got.TensorType != nil && tt.wantTensorType != nil && *got.TensorType != *tt.wantTensorType) {
				t.Errorf("parseMetadataJSON() TensorType = %v, want %v", got.TensorType, tt.wantTensorType)
			}

			// Test VariantGroupID field
			if (got.VariantGroupID == nil) != (tt.wantVariantID == nil) || (got.VariantGroupID != nil && tt.wantVariantID != nil && *got.VariantGroupID != *tt.wantVariantID) {
				t.Errorf("parseMetadataJSON() VariantGroupID = %v, want %v", got.VariantGroupID, tt.wantVariantID)
			}

			// Test MinVRAMGB field
			if (got.MinVRAMGB == nil) != (tt.wantMinVRAMGB == nil) {
				t.Errorf("parseMetadataJSON() MinVRAMGB nil mismatch: got %v, want %v", got.MinVRAMGB, tt.wantMinVRAMGB)
			} else if got.MinVRAMGB != nil && *got.MinVRAMGB != *tt.wantMinVRAMGB {
				t.Errorf("parseMetadataJSON() MinVRAMGB = %v, want %v", *got.MinVRAMGB, *tt.wantMinVRAMGB)
			}

			// Test ModelcarImageSize field
			if (got.ModelcarImageSize == nil) != (tt.wantModelcarImageSize == nil) {
				t.Errorf("parseMetadataJSON() ModelcarImageSize nil mismatch: got %v, want %v", got.ModelcarImageSize, tt.wantModelcarImageSize)
			} else if got.ModelcarImageSize != nil && *got.ModelcarImageSize != *tt.wantModelcarImageSize {
				t.Errorf("parseMetadataJSON() ModelcarImageSize = %v, want %v", *got.ModelcarImageSize, *tt.wantModelcarImageSize)
			}

		})
	}
}

func TestMetadataJSONEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		wantErr  bool
		validate func(*testing.T, metadataJSON)
	}{
		{
			name: "metadata with very long field values",
			jsonData: `{
				"id": "test-model/long-values",
				"size": "` + generateLongString(1000) + `",
				"tensor_type": "` + generateLongString(100) + `",
				"variant_group_id": "` + generateLongString(500) + `"
			}`,
			wantErr: false,
			validate: func(t *testing.T, m metadataJSON) {
				if m.Size == nil || len(*m.Size) != 1000 {
					t.Errorf("Size should be 1000 characters long, got %v", m.Size)
				}
				if m.TensorType == nil || len(*m.TensorType) != 100 {
					t.Errorf("TensorType should be 100 characters long, got %v", m.TensorType)
				}
				if m.VariantGroupID == nil || len(*m.VariantGroupID) != 500 {
					t.Errorf("VariantGroupID should be 500 characters long, got %v", m.VariantGroupID)
				}
			},
		},
		{
			name: "metadata with special characters and unicode",
			jsonData: `{
				"id": "test-model/special-chars-测试",
				"size": "8B params 🤖",
				"tensor_type": "FP16-αβγ",
				"variant_group_id": "uuid-with-special-chars_$#@"
			}`,
			wantErr: false,
			validate: func(t *testing.T, m metadataJSON) {
				expectedID := "test-model/special-chars-测试"
				if m.ID != expectedID {
					t.Errorf("ID should handle unicode, got %v", m.ID)
				}
				expectedSize := "8B params 🤖"
				if m.Size == nil || *m.Size != expectedSize {
					t.Errorf("Size should handle unicode, got %v", m.Size)
				}
				expectedType := "FP16-αβγ"
				if m.TensorType == nil || *m.TensorType != expectedType {
					t.Errorf("TensorType should handle unicode, got %v", m.TensorType)
				}
			},
		},
		{
			name: "metadata with numeric string values that could cause confusion",
			jsonData: `{
				"id": "test-model/numeric-strings",
				"size": "123",
				"tensor_type": "456.789",
				"variant_group_id": "0000-0000-0000-0000"
			}`,
			wantErr: false,
			validate: func(t *testing.T, m metadataJSON) {
				if m.Size == nil || *m.Size != "123" {
					t.Errorf("Size should be string '123', got %v", m.Size)
				}
				if m.TensorType == nil || *m.TensorType != "456.789" {
					t.Errorf("TensorType should be string '456.789', got %v", m.TensorType)
				}
				if m.VariantGroupID == nil || *m.VariantGroupID != "0000-0000-0000-0000" {
					t.Errorf("VariantGroupID should be string '0000-0000-0000-0000', got %v", m.VariantGroupID)
				}
			},
		},
		{
			name: "metadata with wrong type for new fields",
			jsonData: `{
				"id": "test-model/type-mismatch",
				"size": 123,
				"tensor_type": true,
				"variant_group_id": 456.789
			}`,
			wantErr: true,
		},
		{
			name: "metadata with nested objects in new fields (should be handled gracefully)",
			jsonData: `{
				"id": "test-model/nested-objects",
				"size": {"value": "8B params"},
				"tensor_type": ["FP16", "INT4"],
				"variant_group_id": {"id": "abc123"}
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMetadataJSON([]byte(tt.jsonData))

			if (err != nil) != tt.wantErr {
				t.Errorf("parseMetadataJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			if tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

func TestBuildModelDirCache_CaseInsensitive(t *testing.T) {
	// Create a temp directory structure with metadata.json files using mixed-case IDs
	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		metaID    string // ID written to metadata.json
		lookupKey string // key used to look up in cache (simulates DisplayNameFromStoredName output)
		wantFound bool
	}{
		{
			name:      "exact case match",
			metaID:    "sarvam-30b-FP8-Dynamic",
			lookupKey: "sarvam-30b-fp8-dynamic",
			wantFound: true,
		},
		{
			name:      "all uppercase metadata ID",
			metaID:    "MODEL-ABC-FP16",
			lookupKey: "model-abc-fp16",
			wantFound: true,
		},
		{
			name:      "already lowercase",
			metaID:    "already-lowercase",
			lookupKey: "already-lowercase",
			wantFound: true,
		},
		{
			name:      "mixed case with org prefix",
			metaID:    "RedHatAI/Granite-3B-Code",
			lookupKey: "redhatai/granite-3b-code",
			wantFound: true,
		},
		{
			name:      "no match returns not found",
			metaID:    "some-model",
			lookupKey: "different-model",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a subdirectory with a metadata.json for this test case
			modelDir := filepath.Join(tmpDir, tt.name)
			if err := os.MkdirAll(modelDir, 0o755); err != nil {
				t.Fatalf("failed to create model dir: %v", err)
			}
			metaContent := `{"id": "` + tt.metaID + `"}`
			if err := os.WriteFile(filepath.Join(modelDir, "metadata.json"), []byte(metaContent), 0o644); err != nil {
				t.Fatalf("failed to write metadata.json: %v", err)
			}

			// Build cache with only this directory
			loader := &PerformanceMetricsLoader{
				path:          []string{modelDir},
				modelDirCache: make(map[string]string),
			}
			if err := loader.buildModelDirCache(); err != nil {
				t.Fatalf("buildModelDirCache() error = %v", err)
			}

			// Lookup using the lowercased key (same as Load does)
			_, found := loader.modelDirCache[strings.ToLower(tt.lookupKey)]
			if found != tt.wantFound {
				t.Errorf("cache lookup for %q: got found=%v, want found=%v (cache keys: %v)",
					tt.lookupKey, found, tt.wantFound, cacheKeys(loader.modelDirCache))
			}
		})
	}
}

func TestBuildModelDirCache_CollisionWarning(t *testing.T) {
	// Two directories with metadata IDs that differ only by case should result
	// in the second overwriting the first (last-write-wins). The collision
	// warning is logged but we verify the cache ends up with a single entry.
	tmpDir := t.TempDir()

	dir1 := filepath.Join(tmpDir, "dir1")
	dir2 := filepath.Join(tmpDir, "dir2")
	for _, d := range []string{dir1, dir2} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir1, "metadata.json"), []byte(`{"id": "Model-FP8"}`), 0o644); err != nil {
		t.Fatalf("failed to write metadata.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir2, "metadata.json"), []byte(`{"id": "model-fp8"}`), 0o644); err != nil {
		t.Fatalf("failed to write metadata.json: %v", err)
	}

	loader := &PerformanceMetricsLoader{
		path:          []string{tmpDir},
		modelDirCache: make(map[string]string),
	}
	if err := loader.buildModelDirCache(); err != nil {
		t.Fatalf("buildModelDirCache() error = %v", err)
	}

	// Both IDs normalize to "model-fp8", so the cache should have exactly one entry
	if len(loader.modelDirCache) != 1 {
		t.Errorf("expected 1 cache entry after collision, got %d: %v", len(loader.modelDirCache), cacheKeys(loader.modelDirCache))
	}
	// filepath.Walk visits in lexicographic order, so dir2 (last-write) wins
	cachedPath, found := loader.modelDirCache["model-fp8"]
	if !found {
		t.Fatalf("expected cache key %q, got keys: %v", "model-fp8", cacheKeys(loader.modelDirCache))
	}
	if cachedPath != dir2 {
		t.Errorf("expected collision winner to be %s (last walked), got %s", dir2, cachedPath)
	}
}

func TestEnrichCatalogModelFromMetadata_NewFields(t *testing.T) {
	modelName := "test-vendor/test-405b-instruct"
	modelID := int32(1)

	existingModel := &models.CatalogModelImpl{
		ID: &modelID,
		Attributes: &models.CatalogModelAttributes{
			Name: &modelName,
		},
	}

	minVRAM := 80.0
	metadata := metadataJSON{
		ID:        modelName,
		MinVRAMGB: &minVRAM,
	}

	mockRepo := &mockPerfModelRepo{}

	err := enrichCatalogModelFromMetadata(existingModel, metadata, mockRepo)
	if err != nil {
		t.Fatalf("enrichCatalogModelFromMetadata() error = %v", err)
	}

	props := existingModel.GetCustomProperties()
	if props == nil {
		t.Fatal("expected custom properties to be set, got nil")
	}

	propMap := make(map[string]float64)
	for _, p := range *props {
		if p.DoubleValue != nil {
			propMap[p.Name] = *p.DoubleValue
		}
	}

	if v, ok := propMap["min_vram_gb"]; !ok {
		t.Error("expected custom property 'min_vram_gb' to be set")
	} else if v != 80.0 {
		t.Errorf("min_vram_gb = %v, want %v", v, 80.0)
	}

}

// cacheKeys returns all keys from a map for diagnostic output.
func cacheKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func generateLongString(length int) string {
	var result strings.Builder
	char := "a"
	for range length {
		result.WriteString(char)
	}
	return result.String()
}

func TestSecurityEvaluationRecordUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name             string
		jsonData         string
		wantID           string
		wantModelID      string
		wantCustomProps  map[string]any
		wantErr          bool
		checkCustomProps bool
	}{
		{
			name: "complete security evaluation record",
			jsonData: `{
				"id": "sec-eval-123",
				"model_id": "test-model-456",
				"cve_score": 7.5,
				"created_at": 1609459200,
				"updated_at": 1609545600
			}`,
			wantID:      "sec-eval-123",
			wantModelID: "test-model-456",
			wantCustomProps: map[string]any{
				"id":         "sec-eval-123",
				"model_id":   "test-model-456",
				"cve_score":  7.5,
				"created_at": float64(1609459200),
				"updated_at": float64(1609545600),
			},
			wantErr:          false,
			checkCustomProps: true,
		},
		{
			name: "minimal security record with only core fields",
			jsonData: `{
				"id": "minimal-sec",
				"model_id": "minimal-model"
			}`,
			wantID:      "minimal-sec",
			wantModelID: "minimal-model",
			wantCustomProps: map[string]any{
				"id":       "minimal-sec",
				"model_id": "minimal-model",
			},
			wantErr:          false,
			checkCustomProps: true,
		},
		{
			name: "security record with custom properties",
			jsonData: `{
				"id": "custom-sec",
				"model_id": "custom-model",
				"risk_level": "high",
				"exploitability": 8.5,
				"patched": true
			}`,
			wantID:      "custom-sec",
			wantModelID: "custom-model",
			wantCustomProps: map[string]any{
				"id":             "custom-sec",
				"model_id":       "custom-model",
				"risk_level":     "high",
				"exploitability": 8.5,
				"patched":        true,
			},
			wantErr:          false,
			checkCustomProps: true,
		},
		{
			name: "security record with null values",
			jsonData: `{
				"id": "null-sec",
				"model_id": "null-model",
				"null_field": null,
				"score": 5.0
			}`,
			wantID:      "null-sec",
			wantModelID: "null-model",
			wantCustomProps: map[string]any{
				"id":         "null-sec",
				"model_id":   "null-model",
				"null_field": nil,
				"score":      5.0,
			},
			wantErr:          false,
			checkCustomProps: true,
		},
		{
			name: "security record missing core fields",
			jsonData: `{
				"cve_score": 9.8,
				"severity": "critical"
			}`,
			wantID:           "",
			wantModelID:      "",
			wantErr:          false,
			checkCustomProps: false,
		},
		{
			name: "security record with wrong type for core fields",
			jsonData: `{
				"id": 123,
				"model_id": 456,
				"score": 7.0
			}`,
			wantID:           "",
			wantModelID:      "",
			wantErr:          false,
			checkCustomProps: false,
		},
		{
			name:             "empty JSON object",
			jsonData:         `{}`,
			wantID:           "",
			wantModelID:      "",
			wantErr:          false,
			checkCustomProps: false,
		},
		{
			name:             "invalid JSON",
			jsonData:         `{"id": "invalid", "model_id":}`,
			wantErr:          true,
			checkCustomProps: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sr securityEvaluationRecord
			err := sr.UnmarshalJSON([]byte(tt.jsonData))

			if (err != nil) != tt.wantErr {
				t.Errorf("securityEvaluationRecord.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			if sr.ID != tt.wantID {
				t.Errorf("ID = %v, want %v", sr.ID, tt.wantID)
			}
			if sr.ModelID != tt.wantModelID {
				t.Errorf("ModelID = %v, want %v", sr.ModelID, tt.wantModelID)
			}

			if sr.CustomProperties == nil {
				t.Error("CustomProperties should not be nil")
			}

			if tt.checkCustomProps {
				if len(sr.CustomProperties) != len(tt.wantCustomProps) {
					t.Errorf("CustomProperties length = %v, want %v", len(sr.CustomProperties), len(tt.wantCustomProps))
				}
				for key, wantValue := range tt.wantCustomProps {
					gotValue, exists := sr.CustomProperties[key]
					if !exists {
						t.Errorf("CustomProperties missing key %v", key)
						continue
					}

					if jsonNumber, ok := gotValue.(json.Number); ok {
						var newValue any
						var convErr error
						switch wantValue.(type) {
						case float64:
							newValue, convErr = jsonNumber.Float64()
						case int, int32, int64:
							newValue, convErr = jsonNumber.Int64()
						}
						if convErr == nil {
							gotValue = newValue
						}
					}

					if gotValue != wantValue {
						t.Errorf("CustomProperties[%v] = %v (type %T), want %v (type %T)",
							key, gotValue, gotValue, wantValue, wantValue)
					}
				}
			}
		})
	}
}

func TestSecurityEvaluationRecordUnmarshalJSON_CoreFieldsInCustomProperties(t *testing.T) {
	jsonData := `{
		"id": "sec-id",
		"model_id": "test-model",
		"cve_score": 8.5
	}`

	var sr securityEvaluationRecord
	err := sr.UnmarshalJSON([]byte(jsonData))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if sr.CustomProperties["id"] != "sec-id" {
		t.Errorf("CustomProperties[id] = %v, want %v", sr.CustomProperties["id"], "sec-id")
	}
	if sr.CustomProperties["model_id"] != "test-model" {
		t.Errorf("CustomProperties[model_id] = %v, want %v", sr.CustomProperties["model_id"], "test-model")
	}
	if v, _ := sr.CustomProperties["cve_score"].(json.Number).Float64(); v != 8.5 {
		t.Errorf("CustomProperties[cve_score] = %v, want 8.5", sr.CustomProperties["cve_score"])
	}
}

func TestCreateSecurityArtifact(t *testing.T) {
	t.Run("artifact name is scoped to the catalog model and metrics type", func(t *testing.T) {
		secRecord := securityEvaluationRecord{
			ID:      "sec-eval-abc123",
			ModelID: "test-model",
			CustomProperties: map[string]any{
				"id":       "sec-eval-abc123",
				"model_id": "test-model",
			},
		}

		artifact := createSecurityArtifact(secRecord, 1, 100, nil, nil)

		if artifact.Attributes == nil {
			t.Fatal("Attributes should not be nil")
		}
		assert.Equal(t, "catalog-metrics:v1:1:security-metrics:caf342983861a986457b9a8bd6188d2ac2df2a60a04b6e44098e73de898d9129", *artifact.Attributes.Name)
	})

	t.Run("external ID is scoped and producer ID is preserved", func(t *testing.T) {
		secRecord := securityEvaluationRecord{
			ID:               "sec-eval-xyz789",
			CustomProperties: map[string]any{"id": "sec-eval-xyz789"},
		}

		artifact := createSecurityArtifact(secRecord, 1, 100, nil, nil)

		assert.Equal(t, "catalog-metrics:v1:1:security-metrics:e87ae8ab4a508b9229f4a587a2797e4fd19f6b9cc4482a49a3d653a995c11ff5", *artifact.Attributes.ExternalID)
		require.Len(t, *artifact.CustomProperties, 1)
		assert.Equal(t, "id", (*artifact.CustomProperties)[0].Name)
		assert.Equal(t, "sec-eval-xyz789", *(*artifact.CustomProperties)[0].StringValue)
	})

	t.Run("metrics type is security-metrics", func(t *testing.T) {
		secRecord := securityEvaluationRecord{
			ID:               "sec-eval-001",
			CustomProperties: map[string]any{"id": "sec-eval-001"},
		}

		artifact := createSecurityArtifact(secRecord, 1, 100, nil, nil)

		if artifact.Attributes.MetricsType != "security-metrics" {
			t.Errorf("MetricsType = %v, want security-metrics", artifact.Attributes.MetricsType)
		}
	})

	t.Run("custom properties are mapped correctly", func(t *testing.T) {
		strVal := "high"
		floatVal := 9.1
		boolVal := true
		secRecord := securityEvaluationRecord{
			ID:      "sec-eval-props",
			ModelID: "test-model",
			CustomProperties: map[string]any{
				"id":           "sec-eval-props",
				"risk_level":   strVal,
				"cvss_score":   json.Number("9.1"),
				"is_exploited": boolVal,
			},
		}

		artifact := createSecurityArtifact(secRecord, 1, 100, nil, nil)

		propMap := map[string]any{}
		for _, p := range *artifact.CustomProperties {
			if p.StringValue != nil {
				propMap[p.Name] = *p.StringValue
			} else if p.DoubleValue != nil {
				propMap[p.Name] = *p.DoubleValue
			} else if p.BoolValue != nil {
				propMap[p.Name] = *p.BoolValue
			}
		}

		if propMap["risk_level"] != strVal {
			t.Errorf("risk_level = %v, want %v", propMap["risk_level"], strVal)
		}
		if propMap["cvss_score"] != floatVal {
			t.Errorf("cvss_score = %v, want %v", propMap["cvss_score"], floatVal)
		}
		if propMap["is_exploited"] != boolVal {
			t.Errorf("is_exploited = %v, want %v", propMap["is_exploited"], boolVal)
		}
	})

	t.Run("created_at and updated_at are removed from custom properties", func(t *testing.T) {
		createdAt := int64(1609459200000)
		secRecord := securityEvaluationRecord{
			ID: "sec-eval-timestamps",
			CustomProperties: map[string]any{
				"id":         "sec-eval-timestamps",
				"created_at": json.Number("1609459200000"),
				"updated_at": json.Number("1609545600000"),
				"score":      json.Number("7.5"),
			},
		}

		artifact := createSecurityArtifact(secRecord, 1, 100, nil, nil)

		for _, p := range *artifact.CustomProperties {
			if p.Name == "created_at" || p.Name == "updated_at" {
				t.Errorf("unexpected property %q in custom properties", p.Name)
			}
		}

		if artifact.Attributes.CreateTimeSinceEpoch == nil || *artifact.Attributes.CreateTimeSinceEpoch != createdAt {
			t.Errorf("CreateTimeSinceEpoch = %v, want %v", artifact.Attributes.CreateTimeSinceEpoch, createdAt)
		}
	})
}

func TestParseSecurityEvaluationFile(t *testing.T) {
	t.Run("valid multi-record NDJSON", func(t *testing.T) {
		f := writeTempNDJSON(t, []string{
			`{"id":"id-1","model_id":"m1","result":0.1,"pass":true}`,
			`{"id":"id-2","model_id":"m1","result":0.2,"pass":false}`,
		})

		records, err := parseSecurityEvaluationFile(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(records) != 2 {
			t.Fatalf("expected 2 records, got %d", len(records))
		}
		if records[0].ID != "id-1" || records[1].ID != "id-2" {
			t.Errorf("unexpected record IDs: %v, %v", records[0].ID, records[1].ID)
		}
	})

	t.Run("blank lines are skipped", func(t *testing.T) {
		f := writeTempNDJSON(t, []string{
			`{"id":"id-1","model_id":"m1"}`,
			``,
			`   `,
			`{"id":"id-2","model_id":"m1"}`,
		})

		records, err := parseSecurityEvaluationFile(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(records) != 2 {
			t.Errorf("expected 2 records after skipping blank lines, got %d", len(records))
		}
	})

	t.Run("malformed record is skipped, valid records returned", func(t *testing.T) {
		f := writeTempNDJSON(t, []string{
			`{"id":"id-1","model_id":"m1"}`,
			`{not valid json`,
			`{"id":"id-2","model_id":"m1"}`,
		})

		records, err := parseSecurityEvaluationFile(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(records) != 2 {
			t.Errorf("expected 2 records after skipping malformed line, got %d", len(records))
		}
	})

	t.Run("empty file returns zero records without error", func(t *testing.T) {
		f := writeTempNDJSON(t, []string{})

		records, err := parseSecurityEvaluationFile(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(records) != 0 {
			t.Errorf("expected 0 records, got %d", len(records))
		}
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		_, err := parseSecurityEvaluationFile("/nonexistent/path/security-evaluations.ndjson")
		if err == nil {
			t.Error("expected error for nonexistent file, got nil")
		}
	})
}

// writeTempNDJSON writes lines to a temp file and returns its path.
func writeTempNDJSON(t *testing.T, lines []string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.ndjson")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	for _, line := range lines {
		if _, err := f.WriteString(line + "\n"); err != nil {
			t.Fatalf("failed to write line: %v", err)
		}
	}
	f.Close()
	return f.Name()
}
