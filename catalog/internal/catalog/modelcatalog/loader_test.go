package modelcatalog

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	sharedmodels "github.com/kubeflow/hub/catalog/internal/db/models"
	apimodels "github.com/kubeflow/hub/catalog/pkg/openapi"
	"github.com/kubeflow/hub/internal/platform/apiutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestRemoveModelsFromMissingSources(t *testing.T) {
	// This test verifies the current behavior of removeModelsFromMissingSources.
	// The method removes models from sources that are either:
	// 1. Not present in the current configuration
	// 2. Explicitly disabled (Enabled = false)
	// Note: Sources with nil Enabled are filtered out by AllSources() - see bug comment below

	tests := []struct {
		name                   string
		enabledSources         map[string]*bool // source ID -> enabled status (nil means default true)
		existingSourceIDs      []string         // source IDs currently in database
		expectedDeletedSources []string         // source IDs that should be deleted
		repositoryError        string           // if set, repository returns this error
		expectError            bool
	}{
		{
			name: "removes models from sources not in config",
			enabledSources: map[string]*bool{
				"source1": apiutils.Of(true),
				"source2": apiutils.Of(true),
			},
			existingSourceIDs:      []string{"source1", "source2", "source3", "source4"},
			expectedDeletedSources: []string{"source3", "source4"}, // source3 and source4 not in config
		},
		{
			name: "no deletion when all database sources are in config",
			enabledSources: map[string]*bool{
				"source1": apiutils.Of(true),
				"source2": apiutils.Of(true),
				"source3": apiutils.Of(true),
			},
			existingSourceIDs:      []string{"source1", "source2"},
			expectedDeletedSources: []string{}, // no deletions - all database sources are in config
		},
		{
			name: "handles empty existing sources",
			enabledSources: map[string]*bool{
				"source1": apiutils.Of(true),
			},
			existingSourceIDs:      []string{},
			expectedDeletedSources: []string{}, // no deletions needed - no existing sources
		},
		{
			name:                   "handles empty config sources",
			enabledSources:         map[string]*bool{}, // no sources in config
			existingSourceIDs:      []string{"source1", "source2"},
			expectedDeletedSources: []string{"source1", "source2"}, // all existing sources deleted
		},
		{
			name: "correctly handles default enabled sources",
			enabledSources: map[string]*bool{
				"source1": nil,               // default enabled - converted to true by applyDefaults
				"source2": apiutils.Of(true), // explicitly enabled
			},
			existingSourceIDs:      []string{"source1", "source2", "source3"},
			expectedDeletedSources: []string{"source3"}, // only source3 (not in config) gets deleted
		},
		{
			name: "handles repository error on GetDistinctSourceIDs",
			enabledSources: map[string]*bool{
				"source1": apiutils.Of(true),
			},
			existingSourceIDs: []string{"source1"},
			repositoryError:   "get_distinct_source_ids_error",
			expectError:       true,
		},
		{
			name: "handles repository error on DeleteBySource",
			enabledSources: map[string]*bool{
				"source1": apiutils.Of(true),
			},
			existingSourceIDs:      []string{"source1", "source2"},
			repositoryError:        "delete_by_source_error",
			expectedDeletedSources: []string{"source2"}, // source2 should be attempted for deletion
			expectError:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock repository with test data
			mockModelRepo := &MockCatalogModelRepositoryWithSourceTracking{
				ExistingSourceIDs: tt.existingSourceIDs,
				DeletedSources:    []string{},
				ErrorType:         tt.repositoryError,
			}

			services := Services{
				CatalogModelRepository:           mockModelRepo,
				CatalogArtifactRepository:        &MockCatalogArtifactRepository{},
				CatalogModelArtifactRepository:   &MockCatalogModelArtifactRepository{},
				CatalogMetricsArtifactRepository: &MockCatalogMetricsArtifactRepository{},
				CatalogSourceRepository:          &MockCatalogSourceRepository{},
				PropertyOptionsRepository:        &MockPropertyOptionsRepository{},
			}

			// Create loader and populate sources
			loader := NewModelLoader(services, basecatalog.NewBaseLoader([]string{}))

			// Add all test sources to the loader's source collection in a single Merge call
			sourcesMap := make(map[string]basecatalog.ModelSource)
			for sourceID, enabled := range tt.enabledSources {
				source := apimodels.CatalogSource{
					Id:      sourceID,
					Name:    "Test " + sourceID,
					Enabled: enabled,
				}

				sourcesMap[sourceID] = basecatalog.ModelSource{
					CatalogSource: source,
					Type:          "test",
				}
			}

			if len(sourcesMap) > 0 {
				err := loader.Sources.Merge("test-path", sourcesMap)
				if err != nil {
					t.Fatalf("Failed to add sources: %v", err)
				}
			}

			// Call the method under test (empty combined IDs: model-only test)
			err := loader.removeModelsFromMissingSources(mapset.NewSet[string]())

			// Verify error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify which sources were deleted
			expectedSet := mapset.NewSet(tt.expectedDeletedSources...)
			actualSet := mapset.NewSet(mockModelRepo.DeletedSources...)

			if !expectedSet.Equal(actualSet) {
				t.Errorf("Expected deleted sources %v, got %v",
					tt.expectedDeletedSources, mockModelRepo.DeletedSources)
			}
		})
	}
}

// MockCatalogModelRepositoryWithSourceTracking extends the existing mock to add source tracking
type MockCatalogModelRepositoryWithSourceTracking struct {
	MockCatalogModelRepository
	ExistingSourceIDs []string
	DeletedSources    []string
	ErrorType         string // "get_distinct_source_ids_error" or "delete_by_source_error"
}

func (m *MockCatalogModelRepositoryWithSourceTracking) GetDistinctSourceIDs() ([]string, error) {
	if m.ErrorType == "get_distinct_source_ids_error" {
		return nil, NewMockError("failed to get distinct source IDs")
	}
	return m.ExistingSourceIDs, nil
}

func (m *MockCatalogModelRepositoryWithSourceTracking) DeleteBySource(sourceID string) error {
	if m.ErrorType == "delete_by_source_error" {
		return NewMockError("failed to delete models from source: " + sourceID)
	}
	m.DeletedSources = append(m.DeletedSources, sourceID)
	return nil
}

// Helper function to create mock errors
func NewMockError(message string) error {
	return &MockRepositoryError{Message: message}
}

type MockRepositoryError struct {
	Message string
}

func (e *MockRepositoryError) Error() string {
	return e.Message

}

func TestSourceConfigNamedQueries(t *testing.T) {
	t.Run("legacy flat format without assetType", func(t *testing.T) {
		yamlContent := `
catalogs: []
namedQueries:
  validation-default:
    ttft_p90:
      operator: '<'
      value: 70
    workload_type:
      operator: '='
      value: "Chat"
  high-performance:
    performance_score:
      operator: '>'
      value: 0.95
`
		var config basecatalog.SourceConfig
		err := yaml.UnmarshalStrict([]byte(yamlContent), &config)
		assert.NoError(t, err)
		assert.NotNil(t, config.NamedQueries)
		assert.Len(t, config.NamedQueries, 2)

		validationQuery := config.NamedQueries["validation-default"]
		assert.Equal(t, "", validationQuery.AssetType) // defaults to models when filtering
		assert.Equal(t, "<", validationQuery.Filters["ttft_p90"].Operator)
		assert.Equal(t, float64(70), validationQuery.Filters["ttft_p90"].Value)
		assert.Equal(t, "=", validationQuery.Filters["workload_type"].Operator)
		assert.Equal(t, "Chat", validationQuery.Filters["workload_type"].Value)
	})

	t.Run("new format with explicit assetType", func(t *testing.T) {
		yamlContent := `
catalogs: []
namedQueries:
  model-query:
    assetType: models
    filters:
      ttft_p90:
        operator: '<'
        value: 70
  mcp-query:
    assetType: mcp_servers
    filters:
      verified:
        operator: '='
        value: true
`
		var config basecatalog.SourceConfig
		err := yaml.UnmarshalStrict([]byte(yamlContent), &config)
		assert.NoError(t, err)
		assert.Len(t, config.NamedQueries, 2)

		modelQuery := config.NamedQueries["model-query"]
		assert.Equal(t, basecatalog.AssetTypeModels, modelQuery.AssetType)
		assert.Equal(t, "<", modelQuery.Filters["ttft_p90"].Operator)

		mcpQuery := config.NamedQueries["mcp-query"]
		assert.Equal(t, basecatalog.AssetTypeMCPServers, mcpQuery.AssetType)
		assert.Equal(t, "=", mcpQuery.Filters["verified"].Operator)
	})

	t.Run("model loader only picks up models-scoped queries", func(t *testing.T) {
		queries := map[string]basecatalog.NamedQuery{
			"model-query": {
				AssetType: basecatalog.AssetTypeModels,
				Filters:   map[string]basecatalog.FieldFilter{"field": {Operator: "=", Value: "v"}},
			},
			"mcp-query": {
				AssetType: basecatalog.AssetTypeMCPServers,
				Filters:   map[string]basecatalog.FieldFilter{"field": {Operator: "=", Value: "v"}},
			},
			"default-query": {
				// No AssetType — defaults to models
				Filters: map[string]basecatalog.FieldFilter{"field": {Operator: "=", Value: "v"}},
			},
		}
		filtered := basecatalog.FilterNamedQueriesByAssetType(queries, basecatalog.AssetTypeModels)
		assert.Len(t, filtered, 2)
		assert.Contains(t, filtered, "model-query")
		assert.Contains(t, filtered, "default-query")
		assert.NotContains(t, filtered, "mcp-query")
	})
}

func TestLoader_StartWithLeaderElection(t *testing.T) {
	// Create mock repositories with tracking capabilities
	mockModelRepo := &MockCatalogModelRepositoryWithSourceTracking{
		ExistingSourceIDs: []string{},
		DeletedSources:    []string{},
	}
	mockArtifactRepo := &MockCatalogArtifactRepository{}
	mockModelArtifactRepo := &MockCatalogModelArtifactRepository{}
	mockMetricsArtifactRepo := &MockCatalogMetricsArtifactRepository{}
	mockSourceRepo := &MockCatalogSourceRepository{}

	services := Services{
		CatalogModelRepository:           mockModelRepo,
		CatalogArtifactRepository:        mockArtifactRepo,
		CatalogModelArtifactRepository:   mockModelArtifactRepo,
		CatalogMetricsArtifactRepository: mockMetricsArtifactRepo,
		CatalogSourceRepository:          mockSourceRepo,
		PropertyOptionsRepository:        &MockPropertyOptionsRepository{},
	}

	// Register a test provider
	testProviderName := "test-leader-provider"
	RegisterModelProvider(testProviderName, func(ctx context.Context, source *basecatalog.ModelSource, reldir string) (<-chan ModelProviderRecord, error) {
		ch := make(chan ModelProviderRecord, 2)

		modelName := "test-model-1"
		model := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name: &modelName,
			},
		}

		ch <- ModelProviderRecord{
			Model:     model,
			Artifacts: []sharedmodels.CatalogArtifact{},
		}

		// Send completion marker
		ch <- ModelProviderRecord{
			Model: nil,
		}

		close(ch)
		return ch, nil
	})

	testConfig := &basecatalog.SourceConfig{
		ModelCatalogs: []basecatalog.ModelSource{
			{
				CatalogSource: apimodels.CatalogSource{
					Id:      "test-catalog",
					Name:    "Test Catalog",
					Enabled: apiutils.Of(true),
				},
				Type: testProviderName,
			},
		},
	}

	t.Run("standby mode skips database writes", func(t *testing.T) {
		baseLoader := basecatalog.NewBaseLoader([]string{})
		loader := NewModelLoader(services, baseLoader)

		// Populate sources
		err := loader.updateSources("test-path", testConfig)
		assert.NoError(t, err)

		// In standby mode (read-only), just parse configs without becoming leader
		err = loader.ParseAllConfigs()
		assert.NoError(t, err)

		// Verify we're not in leader mode
		assert.False(t, baseLoader.ShouldWriteDatabase(), "Standby mode should not write to database")

		// Wait a bit for any goroutines to process
		time.Sleep(100 * time.Millisecond)

		// Verify no database writes occurred in standby mode
		assert.Empty(t, mockModelRepo.SavedModels, "Standby mode should not write models to database")
		assert.Empty(t, mockModelArtifactRepo.SavedArtifacts, "Standby mode should not write artifacts to database")
		assert.Empty(t, mockMetricsArtifactRepo.SavedMetrics, "Standby mode should not write metrics to database")
	})

	t.Run("leader mode performs database writes", func(t *testing.T) {
		// Reset mock repositories
		mockModelRepo.SavedModels = []models.CatalogModel{}
		mockModelArtifactRepo.SavedArtifacts = []models.CatalogModelArtifact{}
		mockMetricsArtifactRepo.SavedMetrics = []models.CatalogMetricsArtifact{}

		baseLoader := basecatalog.NewBaseLoader([]string{})
		loader := NewModelLoader(services, baseLoader)
		ctx := context.Background()

		// Populate sources
		err := loader.updateSources("test-path", testConfig)
		assert.NoError(t, err)

		// Parse configs first
		err = loader.ParseAllConfigs()
		assert.NoError(t, err)

		// Set leader mode on the base loader
		baseLoader.SetLeader(true)

		// Verify we're in leader mode
		assert.True(t, baseLoader.ShouldWriteDatabase(), "Leader mode should write to database")

		// Perform leader operations (empty combined IDs: model-only test)
		err = loader.PerformLeaderOperations(ctx, mapset.NewSet[string]())
		assert.NoError(t, err)

		// Wait for goroutines to process
		time.Sleep(200 * time.Millisecond)

		// Verify database writes occurred in leader mode
		assert.NotEmpty(t, mockModelRepo.SavedModels, "Leader mode should write models to database")
	})

	// Note: Lifecycle tests (StartReadOnly/StartLeader) are now tested
	// at the integration level with the unified catalog.Loader
}

// TestSourceStatusPartialVsFull verifies that when some models load and
// others fail, the source is partially-available; when all models fail, the
// source is error (not partially-available).
func TestSourceStatusPartialVsFull(t *testing.T) {
	const sourceID = "status-test-source"

	tests := []struct {
		name           string
		providerFunc   func(ctx context.Context) (<-chan ModelProviderRecord, error)
		wantStatus     string
		wantErrContain string
	}{
		{
			name: "partially_available_when_one_model_succeeds_and_hf_partial_failure",
			providerFunc: func(ctx context.Context) (<-chan ModelProviderRecord, error) {
				ch := make(chan ModelProviderRecord, 4)
				go func() {
					defer close(ch)
					modelName := "ok-model"
					ch <- ModelProviderRecord{
						Model: &models.CatalogModelImpl{
							Attributes: &models.CatalogModelAttributes{Name: &modelName},
						},
						Artifacts: []sharedmodels.CatalogArtifact{},
					}
					ch <- ModelProviderRecord{
						Model: nil,
						Error: &PartiallyAvailableError{FailedModels: []string{"missing-model"}},
					}
				}()
				return ch, nil
			},
			wantStatus:     basecatalog.SourceStatusPartiallyAvailable,
			wantErrContain: "Failed models",
		},
		{
			name: "error_when_no_models_succeed_and_hf_partial_failure",
			providerFunc: func(ctx context.Context) (<-chan ModelProviderRecord, error) {
				ch := make(chan ModelProviderRecord, 2)
				go func() {
					defer close(ch)
					ch <- ModelProviderRecord{
						Model: nil,
						Error: &PartiallyAvailableError{FailedModels: []string{"only-failure"}},
					}
				}()
				return ch, nil
			},
			wantStatus:     basecatalog.SourceStatusError,
			wantErrContain: "Failed models",
		},
		{
			name: "error_when_all_models_fail_validation",
			providerFunc: func(ctx context.Context) (<-chan ModelProviderRecord, error) {
				ch := make(chan ModelProviderRecord, 4)
				go func() {
					defer close(ch)
					ch <- ModelProviderRecord{
						Model: nil,
						Error: fmt.Errorf("model %q artifact 0: URI invalid", "bad-model"),
					}
					ch <- ModelProviderRecord{}
				}()
				return ch, nil
			},
			wantStatus:     basecatalog.SourceStatusError,
			wantErrContain: "all catalog models failed to load from source",
		},
		{
			name: "partially_available_when_one_model_succeeds_and_one_validation_fails",
			providerFunc: func(ctx context.Context) (<-chan ModelProviderRecord, error) {
				ch := make(chan ModelProviderRecord, 4)
				go func() {
					defer close(ch)
					ch <- ModelProviderRecord{
						Model: nil,
						Error: fmt.Errorf("model %q artifact 0: URI invalid", "bad-model"),
					}
					goodName := "good-model"
					ch <- ModelProviderRecord{
						Model: &models.CatalogModelImpl{
							Attributes: &models.CatalogModelAttributes{Name: &goodName},
						},
						Artifacts: []sharedmodels.CatalogArtifact{},
					}
					ch <- ModelProviderRecord{}
				}()
				return ch, nil
			},
			wantStatus:     basecatalog.SourceStatusPartiallyAvailable,
			wantErrContain: "Failed to load",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providerName := "status-provider-" + strings.ReplaceAll(t.Name(), "/", "_")
			require.NoError(t, RegisterModelProvider(providerName, func(ctx context.Context, source *basecatalog.ModelSource, reldir string) (<-chan ModelProviderRecord, error) {
				return tt.providerFunc(ctx)
			}))

			mockSourceRepo := &MockCatalogSourceRepository{}
			services := Services{
				CatalogModelRepository:           &MockCatalogModelRepository{},
				CatalogArtifactRepository:        &MockCatalogArtifactRepository{},
				CatalogModelArtifactRepository:   &MockCatalogModelArtifactRepository{},
				CatalogMetricsArtifactRepository: &MockCatalogMetricsArtifactRepository{},
				CatalogSourceRepository:          mockSourceRepo,
				PropertyOptionsRepository:        &MockPropertyOptionsRepository{},
			}

			baseLoader := basecatalog.NewBaseLoader([]string{})
			baseLoader.SetLeader(true)
			loader := NewModelLoader(services, baseLoader)

			cfg := &basecatalog.SourceConfig{
				ModelCatalogs: []basecatalog.ModelSource{
					{
						CatalogSource: apimodels.CatalogSource{
							Id:      sourceID,
							Name:    "Test",
							Enabled: apiutils.Of(true),
						},
						Type: providerName,
					},
				},
			}
			require.NoError(t, loader.updateSources("test-path", cfg))

			ctx := context.Background()
			require.NoError(t, loader.PerformLeaderOperations(ctx, mapset.NewSet(sourceID)))

			var st sharedmodels.SourceStatus
			assert.Eventually(t, func() bool {
				statuses, err := mockSourceRepo.GetAllStatuses()
				if err != nil {
					return false
				}
				var ok bool
				st, ok = statuses[sourceID]
				return ok && st.Status == tt.wantStatus
			}, 3*time.Second, 10*time.Millisecond, "expected status %q for source %s", tt.wantStatus, sourceID)

			assert.Equal(t, tt.wantStatus, st.Status)
			assert.Contains(t, st.Error, tt.wantErrContain)
		})
	}
}
