package openapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kubeflow/hub/catalog/internal/catalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog"
	"github.com/kubeflow/hub/catalog/internal/server/openapi"
	model "github.com/kubeflow/hub/catalog/pkg/openapi"
	mrmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPerformanceArtifactsEndToEnd tests the full integration from HTTP endpoint to database
// This test verifies the complete request flow including routing, parameter parsing, and response generation
func TestPerformanceArtifactsEndToEnd(t *testing.T) {
	// Test basic endpoint functionality
	t.Run("basic performance artifacts retrieval", func(t *testing.T) {
		// Setup
		artifact1Name := "perf-artifact-1"
		artifact1 := model.CatalogArtifact{
			CatalogMetricsArtifact: &model.CatalogMetricsArtifact{
				Name:             &artifact1Name,
				ArtifactType:     "metrics-artifact",
				MetricsType:      "performance-metrics",
				CustomProperties: map[string]model.MetadataValue{},
			},
		}

		provider := &mockPerformanceProvider{
			models: map[string]*model.CatalogModel{
				"test-model": {Name: "test-model"},
			},
			artifacts: map[string][]model.CatalogArtifact{
				"test-model": {artifact1},
			},
		}

		router, _ := setupTestServer(t, provider)

		// Make HTTP request
		req := httptest.NewRequest("GET", "/api/model_catalog/v1alpha1/sources/test-source/models/test-model/artifacts/performance", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		// Assert
		assert.Equal(t, http.StatusOK, resp.Code)

		var result model.CatalogArtifactList
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)

		assert.Equal(t, int32(1), result.Size)
		assert.Len(t, result.Items, 1)
		assert.Equal(t, artifact1Name, *result.Items[0].CatalogMetricsArtifact.Name)
	})

	t.Run("targetRPS parameter handling", func(t *testing.T) {
		// Setup
		artifact1Name := "perf-artifact-1"
		artifact1 := model.CatalogArtifact{
			CatalogMetricsArtifact: &model.CatalogMetricsArtifact{
				Name:             &artifact1Name,
				ArtifactType:     "metrics-artifact",
				MetricsType:      "performance-metrics",
				CustomProperties: map[string]model.MetadataValue{},
			},
		}

		provider := &mockPerformanceProvider{
			models: map[string]*model.CatalogModel{
				"test-model": {Name: "test-model"},
			},
			artifacts: map[string][]model.CatalogArtifact{
				"test-model": {artifact1},
			},
			captureParams: true,
		}

		router, _ := setupTestServer(t, provider)

		// Make HTTP request with targetRPS parameter
		req := httptest.NewRequest("GET", "/api/model_catalog/v1alpha1/sources/test-source/models/test-model/artifacts/performance?targetRPS=100", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		// Assert
		assert.Equal(t, http.StatusOK, resp.Code)

		var result model.CatalogArtifactList
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)

		// Verify targetRPS was passed to provider
		assert.Equal(t, int32(100), provider.lastParams.TargetRPS)

		// Verify targetRPS calculations are in response
		assert.Greater(t, len(result.Items), 0)
		artifact := result.Items[0]
		require.NotNil(t, artifact.CatalogMetricsArtifact)
		require.NotNil(t, artifact.CatalogMetricsArtifact.CustomProperties)

		// Check for calculated properties
		_, foundReplicas := artifact.CatalogMetricsArtifact.CustomProperties["replicas"]
		_, foundTotalRPS := artifact.CatalogMetricsArtifact.CustomProperties["total_requests_per_second"]

		assert.True(t, foundReplicas, "Should have replicas custom property")
		assert.True(t, foundTotalRPS, "Should have total_requests_per_second custom property")
	})

	t.Run("recommendations parameter handling", func(t *testing.T) {
		// Setup
		artifact1Name := "perf-artifact-1"
		artifact1 := model.CatalogArtifact{
			CatalogMetricsArtifact: &model.CatalogMetricsArtifact{
				Name:             &artifact1Name,
				ArtifactType:     "metrics-artifact",
				MetricsType:      "performance-metrics",
				CustomProperties: map[string]model.MetadataValue{},
			},
		}

		provider := &mockPerformanceProvider{
			models: map[string]*model.CatalogModel{
				"test-model": {Name: "test-model"},
			},
			artifacts: map[string][]model.CatalogArtifact{
				"test-model": {artifact1},
			},
			captureParams: true,
		}

		router, _ := setupTestServer(t, provider)

		// Make HTTP request with recommendations parameter
		req := httptest.NewRequest("GET", "/api/model_catalog/v1alpha1/sources/test-source/models/test-model/artifacts/performance?recommendations=true", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		// Assert
		assert.Equal(t, http.StatusOK, resp.Code)

		var result model.CatalogArtifactList
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)

		// Verify recommendations was passed to provider
		assert.True(t, provider.lastParams.Recommendations)
	})

	t.Run("combined targetRPS and recommendations parameters", func(t *testing.T) {
		// Setup
		artifact1Name := "perf-artifact-1"
		artifact1 := model.CatalogArtifact{
			CatalogMetricsArtifact: &model.CatalogMetricsArtifact{
				Name:             &artifact1Name,
				ArtifactType:     "metrics-artifact",
				MetricsType:      "performance-metrics",
				CustomProperties: map[string]model.MetadataValue{},
			},
		}

		provider := &mockPerformanceProvider{
			models: map[string]*model.CatalogModel{
				"test-model": {Name: "test-model"},
			},
			artifacts: map[string][]model.CatalogArtifact{
				"test-model": {artifact1},
			},
			captureParams: true,
		}

		router, _ := setupTestServer(t, provider)

		// Make HTTP request with both parameters
		req := httptest.NewRequest("GET", "/api/model_catalog/v1alpha1/sources/test-source/models/test-model/artifacts/performance?targetRPS=200&recommendations=true", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		// Assert
		assert.Equal(t, http.StatusOK, resp.Code)

		var result model.CatalogArtifactList
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)

		// Verify both parameters were passed correctly
		assert.Equal(t, int32(200), provider.lastParams.TargetRPS)
		assert.True(t, provider.lastParams.Recommendations)

		// Verify response has calculated properties
		assert.Greater(t, len(result.Items), 0)
		artifact := result.Items[0]
		_, foundReplicas := artifact.CatalogMetricsArtifact.CustomProperties["replicas"]
		_, foundTotalRPS := artifact.CatalogMetricsArtifact.CustomProperties["total_requests_per_second"]

		assert.True(t, foundReplicas)
		assert.True(t, foundTotalRPS)
	})

	t.Run("pageSize parameter handling", func(t *testing.T) {
		// Setup - create multiple artifacts
		artifacts := make([]model.CatalogArtifact, 5)
		for i := range 5 {
			name := "perf-artifact-" + string(rune('1'+i))
			artifacts[i] = model.CatalogArtifact{
				CatalogMetricsArtifact: &model.CatalogMetricsArtifact{
					Name:             &name,
					ArtifactType:     "metrics-artifact",
					MetricsType:      "performance-metrics",
					CustomProperties: map[string]model.MetadataValue{},
				},
			}
		}

		provider := &mockPerformanceProvider{
			models: map[string]*model.CatalogModel{
				"test-model": {Name: "test-model"},
			},
			artifacts: map[string][]model.CatalogArtifact{
				"test-model": artifacts,
			},
			captureParams: true,
		}

		router, _ := setupTestServer(t, provider)

		// Make HTTP request with pageSize parameter
		req := httptest.NewRequest("GET", "/api/model_catalog/v1alpha1/sources/test-source/models/test-model/artifacts/performance?pageSize=3", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		// Assert
		assert.Equal(t, http.StatusOK, resp.Code)

		var result model.CatalogArtifactList
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)

		// Verify pageSize was passed correctly
		assert.Equal(t, int32(3), provider.lastParams.PageSize)
	})

	t.Run("filterQuery parameter handling", func(t *testing.T) {
		// Setup
		artifact1Name := "perf-artifact-1"
		artifact1 := model.CatalogArtifact{
			CatalogMetricsArtifact: &model.CatalogMetricsArtifact{
				Name:             &artifact1Name,
				ArtifactType:     "metrics-artifact",
				MetricsType:      "performance-metrics",
				CustomProperties: map[string]model.MetadataValue{},
			},
		}

		provider := &mockPerformanceProvider{
			models: map[string]*model.CatalogModel{
				"test-model": {Name: "test-model"},
			},
			artifacts: map[string][]model.CatalogArtifact{
				"test-model": {artifact1},
			},
			captureParams: true,
		}

		router, _ := setupTestServer(t, provider)

		// Make HTTP request with filterQuery parameter
		req := httptest.NewRequest("GET", "/api/model_catalog/v1alpha1/sources/test-source/models/test-model/artifacts/performance?filterQuery=customProp%3D%27value%27", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		// Assert
		assert.Equal(t, http.StatusOK, resp.Code)

		// Verify filterQuery was passed correctly
		assert.Equal(t, "customProp='value'", provider.lastParams.FilterQuery)
	})

	t.Run("orderBy and sortOrder parameters", func(t *testing.T) {
		// Setup
		artifact1Name := "perf-artifact-1"
		artifact1 := model.CatalogArtifact{
			CatalogMetricsArtifact: &model.CatalogMetricsArtifact{
				Name:             &artifact1Name,
				ArtifactType:     "metrics-artifact",
				MetricsType:      "performance-metrics",
				CustomProperties: map[string]model.MetadataValue{},
			},
		}

		provider := &mockPerformanceProvider{
			models: map[string]*model.CatalogModel{
				"test-model": {Name: "test-model"},
			},
			artifacts: map[string][]model.CatalogArtifact{
				"test-model": {artifact1},
			},
			captureParams: true,
		}

		router, _ := setupTestServer(t, provider)

		// Make HTTP request with orderBy and sortOrder parameters
		req := httptest.NewRequest("GET", "/api/model_catalog/v1alpha1/sources/test-source/models/test-model/artifacts/performance?orderBy=name&sortOrder=DESC", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		// Assert
		assert.Equal(t, http.StatusOK, resp.Code)

		// Verify orderBy and sortOrder were passed correctly
		assert.Equal(t, "name", provider.lastParams.OrderBy)
		assert.Equal(t, model.SORTORDER_DESC, provider.lastParams.SortOrder)
	})

	t.Run("model not found", func(t *testing.T) {
		// Setup empty provider
		provider := &mockPerformanceProvider{
			models:    map[string]*model.CatalogModel{},
			artifacts: map[string][]model.CatalogArtifact{},
		}

		router, _ := setupTestServer(t, provider)

		// Make HTTP request for non-existent model
		req := httptest.NewRequest("GET", "/api/model_catalog/v1alpha1/sources/test-source/models/nonexistent-model/artifacts/performance", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		// Assert - should return OK with empty list
		assert.Equal(t, http.StatusOK, resp.Code)

		var result model.CatalogArtifactList
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)

		assert.Equal(t, int32(0), result.Size)
		assert.Empty(t, result.Items)
	})

	t.Run("invalid targetRPS parameter", func(t *testing.T) {
		// Setup
		provider := &mockPerformanceProvider{
			models:    map[string]*model.CatalogModel{},
			artifacts: map[string][]model.CatalogArtifact{},
		}

		router, _ := setupTestServer(t, provider)

		// Make HTTP request with invalid targetRPS
		req := httptest.NewRequest("GET", "/api/model_catalog/v1alpha1/sources/test-source/models/test-model/artifacts/performance?targetRPS=invalid", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		// Assert - should return bad request
		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("URL encoded model name", func(t *testing.T) {
		// Setup
		artifact1Name := "perf-artifact-1"
		artifact1 := model.CatalogArtifact{
			CatalogMetricsArtifact: &model.CatalogMetricsArtifact{
				Name:             &artifact1Name,
				ArtifactType:     "metrics-artifact",
				MetricsType:      "performance-metrics",
				CustomProperties: map[string]model.MetadataValue{},
			},
		}

		provider := &mockPerformanceProvider{
			models: map[string]*model.CatalogModel{
				"test model with spaces": {Name: "test model with spaces"},
			},
			artifacts: map[string][]model.CatalogArtifact{
				"test model with spaces": {artifact1},
			},
		}

		router, _ := setupTestServer(t, provider)

		// Make HTTP request with URL encoded model name
		req := httptest.NewRequest("GET", "/api/model_catalog/v1alpha1/sources/test-source/models/test%20model%20with%20spaces/artifacts/performance", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		// Assert
		assert.Equal(t, http.StatusOK, resp.Code)

		var result model.CatalogArtifactList
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)

		assert.Equal(t, int32(1), result.Size)
	})
}

// setupTestServer creates a test server with the full routing and HTTP handling stack
func setupTestServer(t *testing.T, provider catalog.APIProvider) (chi.Router, openapi.ModelCatalogServiceAPIServicer) {
	// Create source collection
	sources := catalog.NewSourceCollection()
	sources.Merge("", map[string]catalog.ModelSource{
		"test-source": {
			CatalogSource: model.CatalogSource{Id: "test-source", Name: "Test Source"},
		},
	})
	sourceLabels := catalog.NewLabelCollection()

	// Create service and controller
	service := openapi.NewModelCatalogServiceAPIService(provider, sources, nil, sourceLabels, nil)
	controller := openapi.NewModelCatalogServiceAPIController(service)

	// Create router with proper routing
	router := openapi.NewRouter(controller)

	return router, service
}

// mockPerformanceProvider is a mock implementation of catalog.APIProvider for testing
type mockPerformanceProvider struct {
	models        map[string]*model.CatalogModel
	artifacts     map[string][]model.CatalogArtifact
	captureParams bool
	lastParams    catalog.ListPerformanceArtifactsParams
}

func (m *mockPerformanceProvider) GetModel(ctx context.Context, name string, sourceID string) (*model.CatalogModel, error) {
	mdl, exists := m.models[name]
	if !exists {
		return nil, nil
	}
	return mdl, nil
}

func (m *mockPerformanceProvider) ListModels(ctx context.Context, params catalog.ListModelsParams) (model.CatalogModelList, error) {
	return model.CatalogModelList{}, nil
}

func (m *mockPerformanceProvider) GetArtifacts(ctx context.Context, modelName string, sourceID string, params catalog.ListArtifactsParams) (model.CatalogArtifactList, error) {
	return model.CatalogArtifactList{}, nil
}

func (m *mockPerformanceProvider) GetPerformanceArtifacts(ctx context.Context, modelName string, sourceID string, params catalog.ListPerformanceArtifactsParams) (model.CatalogArtifactList, error) {
	// Capture parameters if requested
	if m.captureParams {
		m.lastParams = params
	}

	artifacts, exists := m.artifacts[modelName]
	if !exists {
		return model.CatalogArtifactList{
			Items:         []model.CatalogArtifact{},
			Size:          0,
			PageSize:      params.PageSize,
			NextPageToken: "",
		}, nil
	}

	// Apply targetRPS calculations if specified
	processedArtifacts := make([]model.CatalogArtifact, len(artifacts))
	copy(processedArtifacts, artifacts)

	if params.TargetRPS > 0 {
		for i := range processedArtifacts {
			artifact := &processedArtifacts[i]
			if artifact.CatalogMetricsArtifact != nil {
				if artifact.CatalogMetricsArtifact.CustomProperties == nil {
					artifact.CatalogMetricsArtifact.CustomProperties = make(map[string]model.MetadataValue)
				}

				// Add calculated replicas
				replicas := max(
					// Simple calculation for testing
					params.TargetRPS/50, 1)
				replicasStr := strconv.FormatInt(int64(replicas), 10)
				artifact.CatalogMetricsArtifact.CustomProperties["replicas"] = model.MetadataValue{
					MetadataIntValue: &model.MetadataIntValue{
						IntValue:     replicasStr,
						MetadataType: "MetadataIntValue",
					},
				}

				// Add calculated total RPS
				totalRPS := float64(params.TargetRPS)
				artifact.CatalogMetricsArtifact.CustomProperties["total_requests_per_second"] = model.MetadataValue{
					MetadataDoubleValue: &model.MetadataDoubleValue{
						DoubleValue:  totalRPS,
						MetadataType: "MetadataDoubleValue",
					},
				}
			}
		}
	}

	// Apply recommendations if requested (simplified for testing)
	if params.Recommendations {
		// In a real implementation, this would deduplicate by cost
		// For testing, we just keep all artifacts
	}

	pageSize := params.PageSize
	if pageSize == 0 {
		pageSize = 10
	}

	// Apply pagination
	endIndex := min(int(pageSize), len(processedArtifacts))
	pagedArtifacts := processedArtifacts[:endIndex]

	nextPageToken := ""
	if len(processedArtifacts) > int(pageSize) {
		nextPageToken = "next-page-token"
	}

	return model.CatalogArtifactList{
		Items:         pagedArtifacts,
		Size:          int32(len(pagedArtifacts)),
		PageSize:      pageSize,
		NextPageToken: nextPageToken,
	}, nil
}

func (m *mockPerformanceProvider) GetFilterOptions(ctx context.Context) (*model.FilterOptionsList, error) {
	return &model.FilterOptionsList{}, nil
}

func (m *mockPerformanceProvider) FindModelsWithRecommendedLatency(ctx context.Context, pagination mrmodels.Pagination, paretoParams modelcatalog.ParetoFilteringParams, sourceIDs []string, query string, sortOrder string) (*model.CatalogModelList, error) {
	// Basic mock implementation - just return models sorted by name
	var allModels []*model.CatalogModel
	for _, mdl := range m.models {
		allModels = append(allModels, mdl)
	}

	// Sort by name for consistent results
	sort.SliceStable(allModels, func(i, j int) bool {
		return allModels[i].Name < allModels[j].Name
	})

	items := make([]model.CatalogModel, len(allModels))
	for i, mdl := range allModels {
		items[i] = *mdl
	}

	return &model.CatalogModelList{
		Items:         items,
		Size:          int32(len(items)),
		PageSize:      10,
		NextPageToken: "",
	}, nil
}
