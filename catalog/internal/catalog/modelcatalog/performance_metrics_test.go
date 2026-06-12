package modelcatalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
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
	}
}

func TestParseMetadataJSON_NewFields(t *testing.T) {
	tests := []struct {
		name                       string
		jsonData                   string
		wantID                     string
		wantSize                   *string
		wantTensorType             *string
		wantVariantID              *string
		wantMinVRAMGB              *float64
		wantModelcarImageSize      *float64
		wantModelcarImageSizeBytes *int64
		wantColdStartMatrix        []coldStartEntry
		wantErr                    bool
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
			name: "metadata with vRAM and cold-start matrix fields",
			jsonData: `{
				"id": "sample-model/test-405b-instruct",
				"size": "405B params",
				"tensor_type": "FP8",
				"variant_group_id": "vwx234mn-5678-901v-wx23-456789abcdef",
				"min_vram_gb": 265.0,
				"cold_start_matrix": [
					{
						"gpu_type": "A100-80",
						"gpu_count": 4,
						"cold_start_time_to_load_seconds": 587.3
					},
					{
						"gpu_type": "B200",
						"gpu_count": 2,
						"cold_start_time_to_load_seconds": 559.9
					}
				]
			}`,
			wantID:         "sample-model/test-405b-instruct",
			wantSize:       &[]string{"405B params"}[0],
			wantTensorType: &[]string{"FP8"}[0],
			wantVariantID:  &[]string{"vwx234mn-5678-901v-wx23-456789abcdef"}[0],
			wantMinVRAMGB:  &[]float64{265.0}[0],
			wantColdStartMatrix: []coldStartEntry{
				{GPUType: "A100-80", GPUCount: 4, ColdStartTimeToLoadSeconds: 587.3},
				{GPUType: "B200", GPUCount: 2, ColdStartTimeToLoadSeconds: 559.9},
			},
			wantErr: false,
		},
		{
			name: "metadata with modelcar image size fields",
			jsonData: `{
				"id": "sample-model/test-405b-instruct",
				"size": "405B params",
				"tensor_type": "FP8",
				"min_vram_gb": 265.0,
				"modelcar_image_size": 405.19,
				"modelcar_image_size_bytes": 405186009411
			}`,
			wantID:                     "sample-model/test-405b-instruct",
			wantSize:                   &[]string{"405B params"}[0],
			wantTensorType:             &[]string{"FP8"}[0],
			wantMinVRAMGB:              &[]float64{265.0}[0],
			wantModelcarImageSize:      &[]float64{405.19}[0],
			wantModelcarImageSizeBytes: &[]int64{405186009411}[0],
			wantErr:                    false,
		},
		{
			name: "metadata with runtime_command in cold-start matrix",
			jsonData: `{
				"id": "RedHatAI/MiniMax-M2.5",
				"size": "229B",
				"tensor_type": "FP8",
				"min_vram_gb": 265.0,
				"cold_start_matrix": [
					{
						"gpu_type": "A100-80",
						"gpu_count": 4,
						"cold_start_time_to_load_seconds": 587.3,
						"runtime_command": "python3 -m vllm.entrypoints.openai.api_server --model RedHatAI/MiniMax-M2.5 --tensor-parallel-size 4"
					},
					{
						"gpu_type": "H200",
						"gpu_count": 4,
						"cold_start_time_to_load_seconds": 806.7,
						"runtime_command": "python3 -m vllm.entrypoints.openai.api_server --model RedHatAI/MiniMax-M2.5 --tensor-parallel-size 4"
					}
				]
			}`,
			wantID:         "RedHatAI/MiniMax-M2.5",
			wantSize:       &[]string{"229B"}[0],
			wantTensorType: &[]string{"FP8"}[0],
			wantMinVRAMGB:  &[]float64{265.0}[0],
			wantColdStartMatrix: []coldStartEntry{
				{GPUType: "A100-80", GPUCount: 4, ColdStartTimeToLoadSeconds: 587.3, RuntimeCommand: "python3 -m vllm.entrypoints.openai.api_server --model RedHatAI/MiniMax-M2.5 --tensor-parallel-size 4"},
				{GPUType: "H200", GPUCount: 4, ColdStartTimeToLoadSeconds: 806.7, RuntimeCommand: "python3 -m vllm.entrypoints.openai.api_server --model RedHatAI/MiniMax-M2.5 --tensor-parallel-size 4"},
			},
			wantErr: false,
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

			// Test ModelcarImageSizeBytes field
			if (got.ModelcarImageSizeBytes == nil) != (tt.wantModelcarImageSizeBytes == nil) {
				t.Errorf("parseMetadataJSON() ModelcarImageSizeBytes nil mismatch: got %v, want %v", got.ModelcarImageSizeBytes, tt.wantModelcarImageSizeBytes)
			} else if got.ModelcarImageSizeBytes != nil && *got.ModelcarImageSizeBytes != *tt.wantModelcarImageSizeBytes {
				t.Errorf("parseMetadataJSON() ModelcarImageSizeBytes = %v, want %v", *got.ModelcarImageSizeBytes, *tt.wantModelcarImageSizeBytes)
			}

			// Test ColdStartMatrix field
			if len(got.ColdStartMatrix) != len(tt.wantColdStartMatrix) {
				t.Errorf("parseMetadataJSON() ColdStartMatrix length = %d, want %d", len(got.ColdStartMatrix), len(tt.wantColdStartMatrix))
			} else {
				for i, entry := range got.ColdStartMatrix {
					want := tt.wantColdStartMatrix[i]
					if entry.GPUType != want.GPUType || entry.GPUCount != want.GPUCount || entry.ColdStartTimeToLoadSeconds != want.ColdStartTimeToLoadSeconds || entry.RuntimeCommand != want.RuntimeCommand {
						t.Errorf("parseMetadataJSON() ColdStartMatrix[%d] = %+v, want %+v", i, entry, want)
					}
				}
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
		ColdStartMatrix: []coldStartEntry{
			{GPUType: "A100", GPUCount: 2, ColdStartTimeToLoadSeconds: 127.3},
			{GPUType: "H100", GPUCount: 1, ColdStartTimeToLoadSeconds: 68.9},
		},
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

	// Verify cold_start_matrix is set as a JSON string custom property
	stringPropMap := make(map[string]string)
	for _, p := range *props {
		if p.StringValue != nil {
			stringPropMap[p.Name] = *p.StringValue
		}
	}
	csJSON, ok := stringPropMap["cold_start_matrix"]
	if !ok {
		t.Fatal("expected custom property 'cold_start_matrix' to be set")
	}
	var csMatrix []coldStartEntry
	if err := json.Unmarshal([]byte(csJSON), &csMatrix); err != nil {
		t.Fatalf("cold_start_matrix is not valid JSON: %v", err)
	}
	if len(csMatrix) != 2 {
		t.Fatalf("cold_start_matrix length = %d, want 2", len(csMatrix))
	}
	if csMatrix[0].GPUType != "A100" || csMatrix[0].GPUCount != 2 || csMatrix[0].ColdStartTimeToLoadSeconds != 127.3 {
		t.Errorf("cold_start_matrix[0] = %+v, want {A100, 2, 127.3}", csMatrix[0])
	}
	if csMatrix[1].GPUType != "H100" || csMatrix[1].GPUCount != 1 || csMatrix[1].ColdStartTimeToLoadSeconds != 68.9 {
		t.Errorf("cold_start_matrix[1] = %+v, want {H100, 1, 68.9}", csMatrix[1])
	}
}

func TestEnrichCatalogModelFromMetadata_EmptyColdStartMatrix(t *testing.T) {
	modelName := "test-vendor/test-model"
	modelID := int32(2)

	existingModel := &models.CatalogModelImpl{
		ID: &modelID,
		Attributes: &models.CatalogModelAttributes{
			Name: &modelName,
		},
	}

	size := "8B params"
	metadata := metadataJSON{
		ID:              modelName,
		Size:            &size,
		ColdStartMatrix: []coldStartEntry{},
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

	for _, p := range *props {
		if p.Name == "cold_start_matrix" {
			t.Error("cold_start_matrix should not be set when matrix is empty")
		}
	}
}

func TestCreateColdStartArtifact(t *testing.T) {
	modelID := int32(42)
	typeID := int32(7)

	tests := []struct {
		name           string
		entry          coldStartEntry
		wantGPUType    string
		wantGPUCount   int
		wantSeconds    *float64
		wantExtID      string
		wantArtName    string
		wantRuntimeCmd string
	}{
		{
			name: "valid entry with float seconds",
			entry: coldStartEntry{
				GPUType:                    "A100-80",
				GPUCount:                   4,
				ColdStartTimeToLoadSeconds: 587.3,
				RuntimeCommand:             "python3 -m vllm.entrypoints.openai.api_server --model RedHatAI/MiniMax-M2.5 --max-model-len -1 --tensor-parallel-size 4 --trust-remote-code",
			},
			wantGPUType:    "A100-80",
			wantGPUCount:   4,
			wantSeconds:    &[]float64{587.3}[0],
			wantExtID:      "cold-start-model-42-A100-80-4",
			wantArtName:    "cold-start-model-42-A100-80-4",
			wantRuntimeCmd: "python3 -m vllm.entrypoints.openai.api_server --model RedHatAI/MiniMax-M2.5 --max-model-len -1 --tensor-parallel-size 4 --trust-remote-code",
		},
		{
			name: "valid entry with integer-like seconds",
			entry: coldStartEntry{
				GPUType:                    "H100",
				GPUCount:                   1,
				ColdStartTimeToLoadSeconds: 68.0,
				RuntimeCommand:             "python3 -m vllm.entrypoints.openai.api_server --model RedHatAI/MiniMax-M2.5 --max-model-len -1 --tensor-parallel-size 1 --trust-remote-code",
			},
			wantGPUType:    "H100",
			wantGPUCount:   1,
			wantSeconds:    &[]float64{68.0}[0],
			wantExtID:      "cold-start-model-42-H100-1",
			wantArtName:    "cold-start-model-42-H100-1",
			wantRuntimeCmd: "python3 -m vllm.entrypoints.openai.api_server --model RedHatAI/MiniMax-M2.5 --max-model-len -1 --tensor-parallel-size 1 --trust-remote-code",
		},
		{
			name: "zero seconds value omits seconds property",
			entry: coldStartEntry{
				GPUType:                    "B200",
				GPUCount:                   2,
				ColdStartTimeToLoadSeconds: 0,
				RuntimeCommand:             "python3 -m vllm.entrypoints.openai.api_server --model RedHatAI/MiniMax-M2.5 --max-model-len -1 --tensor-parallel-size 2 --trust-remote-code",
			},
			wantGPUType:    "B200",
			wantGPUCount:   2,
			wantSeconds:    nil,
			wantExtID:      "cold-start-model-42-B200-2",
			wantArtName:    "cold-start-model-42-B200-2",
			wantRuntimeCmd: "python3 -m vllm.entrypoints.openai.api_server --model RedHatAI/MiniMax-M2.5 --max-model-len -1 --tensor-parallel-size 2 --trust-remote-code",
		},
		{
			name: "empty runtime_command omits property",
			entry: coldStartEntry{
				GPUType:                    "H200",
				GPUCount:                   4,
				ColdStartTimeToLoadSeconds: 806.7,
			},
			wantGPUType:    "H200",
			wantGPUCount:   4,
			wantSeconds:    &[]float64{806.7}[0],
			wantExtID:      "cold-start-model-42-H200-4",
			wantArtName:    "cold-start-model-42-H200-4",
			wantRuntimeCmd: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			externalID := coldStartExternalID(modelID, tt.entry)
			artifact := createColdStartArtifact(tt.entry, externalID, modelID, typeID)

			if artifact == nil {
				t.Fatal("expected non-nil artifact")
			}

			// Verify type and metrics type
			if *artifact.TypeID != typeID {
				t.Errorf("TypeID = %d, want %d", *artifact.TypeID, typeID)
			}
			attrs := artifact.GetAttributes()
			if attrs.MetricsType != models.MetricsTypeColdStart {
				t.Errorf("MetricsType = %q, want %q", attrs.MetricsType, models.MetricsTypeColdStart)
			}

			// Verify naming
			if *attrs.ExternalID != tt.wantExtID {
				t.Errorf("ExternalID = %q, want %q", *attrs.ExternalID, tt.wantExtID)
			}
			if *attrs.Name != tt.wantArtName {
				t.Errorf("Name = %q, want %q", *attrs.Name, tt.wantArtName)
			}

			// Verify timestamps are set
			if attrs.CreateTimeSinceEpoch == nil || *attrs.CreateTimeSinceEpoch <= 0 {
				t.Error("expected CreateTimeSinceEpoch to be set")
			}
			if attrs.LastUpdateTimeSinceEpoch == nil || *attrs.LastUpdateTimeSinceEpoch <= 0 {
				t.Error("expected LastUpdateTimeSinceEpoch to be set")
			}

			// Verify custom properties
			customProps := artifact.GetCustomProperties()
			if customProps == nil {
				t.Fatal("expected custom properties to be set")
			}
			stringPropMap := make(map[string]string)
			doublePropMap := make(map[string]float64)
			intPropMap := make(map[string]int32)
			for _, p := range *customProps {
				if p.StringValue != nil {
					stringPropMap[p.Name] = *p.StringValue
				}
				if p.DoubleValue != nil {
					doublePropMap[p.Name] = *p.DoubleValue
				}
				if p.IntValue != nil {
					intPropMap[p.Name] = *p.IntValue
				}
			}

			if stringPropMap["gpu_type"] != tt.wantGPUType {
				t.Errorf("gpu_type = %v, want %q", stringPropMap["gpu_type"], tt.wantGPUType)
			}
			if intPropMap["gpu_count"] != int32(tt.wantGPUCount) {
				t.Errorf("gpu_count = %v, want %d", intPropMap["gpu_count"], tt.wantGPUCount)
			}
			if tt.wantRuntimeCmd != "" {
				if stringPropMap["runtime_command"] != tt.wantRuntimeCmd {
					t.Errorf("runtime_command = %v, want %q", stringPropMap["runtime_command"], tt.wantRuntimeCmd)
				}
			} else {
				if _, exists := stringPropMap["runtime_command"]; exists {
					t.Error("expected runtime_command to be absent when not provided")
				}
			}
			if tt.wantSeconds != nil {
				if doublePropMap["cold_start_time_to_load_seconds"] != *tt.wantSeconds {
					t.Errorf("cold_start_time_to_load_seconds = %v, want %v", doublePropMap["cold_start_time_to_load_seconds"], *tt.wantSeconds)
				}
			} else {
				if _, exists := doublePropMap["cold_start_time_to_load_seconds"]; exists {
					t.Error("expected cold_start_time_to_load_seconds to be absent for zero value")
				}
			}
		})
	}
}

func TestProcessModelArtifactsBatch_ColdStartOnly(t *testing.T) {
	tmpDir := t.TempDir()

	coldStartMatrix := []coldStartEntry{
		{GPUType: "A100", GPUCount: 2, ColdStartTimeToLoadSeconds: 127.3},
		{GPUType: "H100", GPUCount: 1, ColdStartTimeToLoadSeconds: 68.9},
	}

	modelID := int32(1)
	typeID := int32(7)

	var savedArtifacts []models.CatalogMetricsArtifact
	mockRepo := &mockMetricsArtifactRepo{
		listResult: &dbmodels.ListWrapper[models.CatalogMetricsArtifact]{},
		batchSaveFunc: func(artifacts []models.CatalogMetricsArtifact, parentID *int32) ([]models.CatalogMetricsArtifact, error) {
			savedArtifacts = artifacts
			return artifacts, nil
		},
	}

	count, err := processModelArtifactsBatch(tmpDir, modelID, "test-model", nil, coldStartMatrix, mockRepo, typeID)
	if err != nil {
		t.Fatalf("processModelArtifactsBatch() error = %v", err)
	}

	if count != 2 {
		t.Errorf("processModelArtifactsBatch() returned count = %d, want 2", count)
	}

	if len(savedArtifacts) != 2 {
		t.Fatalf("expected 2 saved artifacts, got %d", len(savedArtifacts))
	}

	// Verify both artifacts have cold-start metrics type
	for _, a := range savedArtifacts {
		if a.GetAttributes().MetricsType != models.MetricsTypeColdStart {
			t.Errorf("expected MetricsType %q, got %q", models.MetricsTypeColdStart, a.GetAttributes().MetricsType)
		}
	}
}

func TestProcessModelArtifactsBatch_ColdStartDedup(t *testing.T) {
	tmpDir := t.TempDir()

	coldStartMatrix := []coldStartEntry{
		{GPUType: "A100", GPUCount: 2, ColdStartTimeToLoadSeconds: 127.3},
	}

	modelID := int32(1)
	typeID := int32(7)
	existingExternalID := "cold-start-model-1-A100-2"

	mockRepo := &mockMetricsArtifactRepo{
		listResult: &dbmodels.ListWrapper[models.CatalogMetricsArtifact]{
			Items: []models.CatalogMetricsArtifact{
				&models.CatalogMetricsArtifactImpl{
					Attributes: &models.CatalogMetricsArtifactAttributes{
						ExternalID: &existingExternalID,
					},
				},
			},
			Size: 1,
		},
	}

	count, err := processModelArtifactsBatch(tmpDir, modelID, "test-model", nil, coldStartMatrix, mockRepo, typeID)
	if err != nil {
		t.Fatalf("processModelArtifactsBatch() error = %v", err)
	}

	if count != 0 {
		t.Errorf("processModelArtifactsBatch() returned count = %d, want 0 (all duplicates)", count)
	}
}

func TestProcessModelArtifactsBatch_ColdStartInvalidGPUCount(t *testing.T) {
	tmpDir := t.TempDir()

	coldStartMatrix := []coldStartEntry{
		{GPUType: "A100", GPUCount: -1, ColdStartTimeToLoadSeconds: 100.0},
		{GPUType: "H100", GPUCount: 0, ColdStartTimeToLoadSeconds: 50.0},
		{GPUType: "", GPUCount: 4, ColdStartTimeToLoadSeconds: 200.0},
		{GPUType: "H200", GPUCount: 2, ColdStartTimeToLoadSeconds: 75.5},
	}

	modelID := int32(1)
	typeID := int32(7)

	var savedArtifacts []models.CatalogMetricsArtifact
	mockRepo := &mockMetricsArtifactRepo{
		listResult: &dbmodels.ListWrapper[models.CatalogMetricsArtifact]{},
		batchSaveFunc: func(artifacts []models.CatalogMetricsArtifact, parentID *int32) ([]models.CatalogMetricsArtifact, error) {
			savedArtifacts = artifacts
			return artifacts, nil
		},
	}

	count, err := processModelArtifactsBatch(tmpDir, modelID, "test-model", nil, coldStartMatrix, mockRepo, typeID)
	if err != nil {
		t.Fatalf("processModelArtifactsBatch() error = %v", err)
	}

	if count != 1 {
		t.Errorf("processModelArtifactsBatch() returned count = %d, want 1 (only H200 entry is valid)", count)
	}

	if len(savedArtifacts) != 1 {
		t.Fatalf("expected 1 saved artifact, got %d", len(savedArtifacts))
	}

	attrs := savedArtifacts[0].GetAttributes()
	wantExtID := "cold-start-model-1-H200-2"
	if *attrs.ExternalID != wantExtID {
		t.Errorf("expected saved artifact ExternalID = %q, got %q", wantExtID, *attrs.ExternalID)
	}
}

func TestProcessModelArtifactsBatch_ColdStartNamesUniqueAcrossModels(t *testing.T) {
	tmpDir := t.TempDir()

	coldStartMatrix := []coldStartEntry{
		{GPUType: "H100", GPUCount: 4, ColdStartTimeToLoadSeconds: 105.7},
		{GPUType: "H200", GPUCount: 2, ColdStartTimeToLoadSeconds: 98.5},
	}

	typeID := int32(7)

	// Collect all artifact names across both models to check uniqueness
	allNames := map[string]int32{}

	for _, modelID := range []int32{1, 2} {
		mockRepo := &mockMetricsArtifactRepo{
			listResult: &dbmodels.ListWrapper[models.CatalogMetricsArtifact]{},
			batchSaveFunc: func(artifacts []models.CatalogMetricsArtifact, parentID *int32) ([]models.CatalogMetricsArtifact, error) {
				for _, a := range artifacts {
					name := *a.GetAttributes().Name
					if prev, exists := allNames[name]; exists {
						t.Errorf("artifact name %q from model %d collides with model %d — would violate UNIQUE(type_id, name) constraint", name, modelID, prev)
					}
					allNames[name] = modelID
				}
				return artifacts, nil
			},
		}

		_, err := processModelArtifactsBatch(tmpDir, modelID, fmt.Sprintf("test-model-%d", modelID), nil, coldStartMatrix, mockRepo, typeID)
		if err != nil {
			t.Fatalf("processModelArtifactsBatch() model %d error = %v", modelID, err)
		}
	}

	if len(allNames) != 4 {
		t.Errorf("expected 4 unique artifact names (2 GPU configs x 2 models), got %d", len(allNames))
	}
}

// mockMetricsArtifactRepo is a minimal mock for CatalogMetricsArtifactRepository used in batch tests
type mockMetricsArtifactRepo struct {
	listResult    *dbmodels.ListWrapper[models.CatalogMetricsArtifact]
	batchSaveFunc func([]models.CatalogMetricsArtifact, *int32) ([]models.CatalogMetricsArtifact, error)
}

func (m *mockMetricsArtifactRepo) GetByID(id int32) (models.CatalogMetricsArtifact, error) {
	return nil, nil
}

func (m *mockMetricsArtifactRepo) List(opts models.CatalogMetricsArtifactListOptions) (*dbmodels.ListWrapper[models.CatalogMetricsArtifact], error) {
	return m.listResult, nil
}

func (m *mockMetricsArtifactRepo) Save(artifact models.CatalogMetricsArtifact, parentID *int32) (models.CatalogMetricsArtifact, error) {
	return artifact, nil
}

func (m *mockMetricsArtifactRepo) BatchSave(artifacts []models.CatalogMetricsArtifact, parentID *int32) ([]models.CatalogMetricsArtifact, error) {
	if m.batchSaveFunc != nil {
		return m.batchSaveFunc(artifacts, parentID)
	}
	return artifacts, nil
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
