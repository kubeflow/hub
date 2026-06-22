package modelcatalog

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	sharedmodels "github.com/kubeflow/hub/catalog/internal/db/models"
	apimodels "github.com/kubeflow/hub/catalog/pkg/openapi"
	mrmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/stretchr/testify/assert"
)

func TestLoadCatalogSources(t *testing.T) {
	type args struct {
		catalogsPath string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "test-catalog-sources",
			args: args{catalogsPath: "testdata/test-catalog-sources.yaml"},
			want: []string{"catalog1", "catalog2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock services
			services := Services{
				CatalogModelRepository:           &MockCatalogModelRepository{},
				CatalogArtifactRepository:        &MockCatalogArtifactRepository{},
				CatalogModelArtifactRepository:   &MockCatalogModelArtifactRepository{},
				CatalogMetricsArtifactRepository: &MockCatalogMetricsArtifactRepository{},
				CatalogSourceRepository:          &MockCatalogSourceRepository{},
				PropertyOptionsRepository:        &MockPropertyOptionsRepository{},
			}
			loader := NewModelLoader(services, basecatalog.NewBaseLoader([]string{tt.args.catalogsPath}))
			// Parse config and populate Sources/Labels
			err := loader.ParseAllConfigs()
			if (err != nil) != tt.wantErr {
				t.Errorf("StartReadOnly() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			gotKeys := make([]string, 0, len(loader.Sources.All()))
			for k := range loader.Sources.All() {
				gotKeys = append(gotKeys, k)
			}
			sort.Strings(gotKeys)
			if !reflect.DeepEqual(gotKeys, tt.want) {
				t.Errorf("NewLoader().Start() got = %v, want %v", gotKeys, tt.want)
			}
		})
	}
}

func TestLoadCatalogSourcesEnabledDisabled(t *testing.T) {
	trueValue := true
	falseValue := false
	type args struct {
		catalogsPath string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]apimodels.CatalogSource
		wantErr bool
	}{
		{
			name: "test-catalog-sources-enabled-and-disabled",
			args: args{catalogsPath: "testdata/test-catalog-sources.yaml"},
			want: map[string]apimodels.CatalogSource{
				"catalog1": {
					Id:      "catalog1",
					Name:    "Catalog 1",
					Enabled: &trueValue,
					Labels:  []string{},
				},
				"catalog2": {
					Id:      "catalog2",
					Name:    "Catalog 2",
					Enabled: &falseValue,
					Labels:  []string{},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock services
			services := Services{
				CatalogModelRepository:           &MockCatalogModelRepository{},
				CatalogArtifactRepository:        &MockCatalogArtifactRepository{},
				CatalogModelArtifactRepository:   &MockCatalogModelArtifactRepository{},
				CatalogMetricsArtifactRepository: &MockCatalogMetricsArtifactRepository{},
				CatalogSourceRepository:          &MockCatalogSourceRepository{},
				PropertyOptionsRepository:        &MockPropertyOptionsRepository{},
			}
			loader := NewModelLoader(services, basecatalog.NewBaseLoader([]string{tt.args.catalogsPath}))
			err := loader.ParseAllConfigs()
			if (err != nil) != tt.wantErr {
				t.Errorf("StartReadOnly() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if !reflect.DeepEqual(loader.Sources.All(), tt.want) {
				t.Errorf("StartReadOnly() got metadata = %#v, want %#v", loader.Sources.All(), tt.want)
			}
		})
	}
}

func TestLabelsValidation(t *testing.T) {
	// Create mock services
	services := Services{
		CatalogModelRepository:           &MockCatalogModelRepository{},
		CatalogArtifactRepository:        &MockCatalogArtifactRepository{},
		CatalogModelArtifactRepository:   &MockCatalogModelArtifactRepository{},
		CatalogMetricsArtifactRepository: &MockCatalogMetricsArtifactRepository{},
		CatalogSourceRepository:          &MockCatalogSourceRepository{},
		PropertyOptionsRepository:        &MockPropertyOptionsRepository{},
	}

	tests := []struct {
		name    string
		config  *basecatalog.SourceConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid labels with name field",
			config: &basecatalog.SourceConfig{
				ModelCatalogs: []basecatalog.ModelSource{},
				Labels: []map[string]any{
					{"name": "labelNameOne", "displayName": "Label Name One"},
					{"name": "labelNameTwo", "displayName": "Label Name Two"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid label missing name field",
			config: &basecatalog.SourceConfig{
				ModelCatalogs: []basecatalog.ModelSource{},
				Labels: []map[string]any{
					{"name": "labelNameOne", "displayName": "Label Name One"},
					{"displayName": "Label Name Two"}, // Missing "name"
				},
			},
			wantErr: true,
			errMsg:  "invalid label at index 1: missing required 'name' field",
		},
		{
			name: "invalid label with empty name",
			config: &basecatalog.SourceConfig{
				ModelCatalogs: []basecatalog.ModelSource{},
				Labels: []map[string]any{
					{"name": "", "displayName": "Empty Name"},
				},
			},
			wantErr: true,
			errMsg:  "invalid label at index 0: missing required 'name' field",
		},
		{
			name: "duplicate label names within same origin",
			config: &basecatalog.SourceConfig{
				ModelCatalogs: []basecatalog.ModelSource{},
				Labels: []map[string]any{
					{"name": "labelNameOne", "displayName": "Label Name One 1"},
					{"name": "labelNameTwo", "displayName": "Label Name Two"},
					{"name": "labelNameOne", "displayName": "Label Name One 2"},
				},
			},
			wantErr: true,
			errMsg:  "duplicate label name 'labelNameOne' within the same origin",
		},
		{
			name: "nil labels should not error",
			config: &basecatalog.SourceConfig{
				ModelCatalogs: []basecatalog.ModelSource{},
				Labels:        nil,
			},
			wantErr: false,
		},
		{
			name: "empty labels array should not error",
			config: &basecatalog.SourceConfig{
				ModelCatalogs: []basecatalog.ModelSource{},
				Labels:        []map[string]any{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewModelLoader(services, basecatalog.NewBaseLoader([]string{}))
			err := loader.updateLabels("test-path", tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("updateLabels() expected error but got none")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("updateLabels() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("updateLabels() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestCatalogSourceLabelsDefaultToEmptySlice(t *testing.T) {
	type args struct {
		catalogsPath string
	}
	tests := []struct {
		name string
		args args
		want func(sources map[string]apimodels.CatalogSource) bool
	}{
		{
			name: "labels-default-to-empty-slice",
			args: args{catalogsPath: "testdata/test-catalog-sources.yaml"},
			want: func(sources map[string]apimodels.CatalogSource) bool {
				// Verify that all loaded catalog sources have labels defaulting to empty slice
				for _, source := range sources {
					if source.Labels == nil {
						return false // Labels should not be nil
					}
					if len(source.Labels) != 0 {
						return false // Labels should be empty slice, not nil and not containing elements
					}
				}
				return len(sources) > 0 // Ensure we actually loaded some sources to test
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock services
			services := Services{
				CatalogModelRepository:           &MockCatalogModelRepository{},
				CatalogArtifactRepository:        &MockCatalogArtifactRepository{},
				CatalogModelArtifactRepository:   &MockCatalogModelArtifactRepository{},
				CatalogMetricsArtifactRepository: &MockCatalogMetricsArtifactRepository{},
				CatalogSourceRepository:          &MockCatalogSourceRepository{},
				PropertyOptionsRepository:        &MockPropertyOptionsRepository{},
			}
			loader := NewModelLoader(services, basecatalog.NewBaseLoader([]string{tt.args.catalogsPath}))
			err := loader.ParseAllConfigs()
			if err != nil {
				t.Errorf("StartReadOnly() error = %v", err)
				return
			}

			sources := loader.Sources.All()
			if !tt.want(sources) {
				t.Errorf("Labels validation failed for sources: %#v", sources)
			}

			// Explicitly verify each source has empty labels slice
			for id, source := range sources {
				if source.Labels == nil {
					t.Errorf("Source %s has nil Labels, expected empty slice", id)
				} else if len(source.Labels) != 0 {
					t.Errorf("Source %s has non-empty Labels %v, expected empty slice", id, source.Labels)
				}
			}
		})
	}
}

func TestLoadCatalogSourcesWithMockRepositories(t *testing.T) {
	// Create mock repositories with tracking capabilities
	mockModelRepo := &MockCatalogModelRepository{}
	mockArtifactRepo := &MockCatalogArtifactRepository{}
	mockModelArtifactRepo := &MockCatalogModelArtifactRepository{}
	mockMetricsArtifactRepo := &MockCatalogMetricsArtifactRepository{}

	services := Services{
		CatalogModelRepository:           mockModelRepo,
		CatalogArtifactRepository:        mockArtifactRepo,
		CatalogModelArtifactRepository:   mockModelArtifactRepo,
		CatalogMetricsArtifactRepository: mockMetricsArtifactRepo,
		CatalogSourceRepository:          &MockCatalogSourceRepository{},
		PropertyOptionsRepository:        &MockPropertyOptionsRepository{},
	}

	// Register a test provider that will create some test data
	testProviderName := "test-provider"
	RegisterModelProvider(testProviderName, func(ctx context.Context, source *basecatalog.ModelSource, reldir string) (<-chan ModelProviderRecord, error) {
		ch := make(chan ModelProviderRecord, 1)

		// Create a test model
		modelName := "test-model"
		model := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name: &modelName,
			},
		}

		// Create test artifacts
		modelArtifactName := "model-artifact"
		metricsArtifactName := "metrics-artifact"

		artifacts := []sharedmodels.CatalogArtifact{
			{
				CatalogModelArtifact: &models.CatalogModelArtifactImpl{
					Attributes: &models.CatalogModelArtifactAttributes{
						Name: &modelArtifactName,
					},
				},
			},
			{
				CatalogMetricsArtifact: &models.CatalogMetricsArtifactImpl{
					Attributes: &models.CatalogMetricsArtifactAttributes{
						Name: &metricsArtifactName,
					},
				},
			},
		}

		ch <- ModelProviderRecord{
			Model:     model,
			Artifacts: artifacts,
		}
		close(ch)

		return ch, nil
	})

	// Create test config content (use in-memory instead of file)
	testConfig := &basecatalog.SourceConfig{
		ModelCatalogs: []basecatalog.ModelSource{
			{
				CatalogSource: apimodels.CatalogSource{
					Id:      "test-catalog",
					Name:    "Test Catalog",
					Enabled: new(true),
				},
				Type: testProviderName,
				Properties: map[string]any{
					"test": "property",
				},
			},
		},
	}

	// Create a loader and test the database update
	baseLoader := basecatalog.NewBaseLoader([]string{})
	l := NewModelLoader(services, baseLoader)
	ctx := context.Background()

	// First call updateSources to populate the SourceCollection
	err := l.updateSources("test-path", testConfig)
	if err != nil {
		t.Fatalf("updateSources() error = %v", err)
	}

	// Parse configs
	err = l.ParseAllConfigs()
	if err != nil {
		t.Fatalf("ParseAllConfigs() error = %v", err)
	}

	// Set leader mode and perform leader operations
	baseLoader.SetLeader(true)

	// Perform leader operations to perform database writes
	go func() {
		if err := l.PerformLeaderOperations(ctx, mapset.NewSet[string]()); err != nil {
			t.Logf("PerformLeaderOperations error: %v", err)
		}
	}()

	// Wait for leader mode to activate and process
	time.Sleep(300 * time.Millisecond)

	// Verify that the model was saved
	savedModels := mockModelRepo.GetSavedModels()
	if len(savedModels) != 1 {
		t.Errorf("Expected 1 model to be saved, got %d", len(savedModels))
	}

	if len(savedModels) > 0 {
		savedModel := savedModels[0]
		if savedModel.GetAttributes() == nil || savedModel.GetAttributes().Name == nil {
			t.Error("Saved model should have attributes with name")
		} else if DisplayName(savedModel) != "test-model" {
			t.Errorf("Expected model name 'test-model', got '%s'", DisplayName(savedModel))
		}
	}

	mockModelArtifactRepo.mu.RLock()
	defer mockModelArtifactRepo.mu.RUnlock()

	// Verify that artifacts were saved
	if len(mockModelArtifactRepo.SavedArtifacts) != 1 {
		t.Errorf("Expected 1 model artifact to be saved, got %d", len(mockModelArtifactRepo.SavedArtifacts))
	}

	mockMetricsArtifactRepo.mu.RLock()
	defer mockMetricsArtifactRepo.mu.RUnlock()

	if len(mockMetricsArtifactRepo.SavedMetrics) != 1 {
		t.Errorf("Expected 1 metrics artifact to be saved, got %d", len(mockMetricsArtifactRepo.SavedMetrics))
	}
}

func TestLoadCatalogSourcesWithRepositoryErrors(t *testing.T) {
	// Create a mock repository that fails on save
	mockModelRepo := &MockCatalogModelRepositoryWithErrors{
		shouldFailSave: true,
	}
	mockArtifactRepo := &MockCatalogArtifactRepository{}
	mockModelArtifactRepo := &MockCatalogModelArtifactRepository{}
	mockMetricsArtifactRepo := &MockCatalogMetricsArtifactRepository{}

	services := Services{
		CatalogModelRepository:           mockModelRepo,
		CatalogArtifactRepository:        mockArtifactRepo,
		CatalogModelArtifactRepository:   mockModelArtifactRepo,
		CatalogMetricsArtifactRepository: mockMetricsArtifactRepo,
		CatalogSourceRepository:          &MockCatalogSourceRepository{},
		PropertyOptionsRepository:        &MockPropertyOptionsRepository{},
	}

	// Register a test provider
	testProviderName := "test-error-provider"
	RegisterModelProvider(testProviderName, func(ctx context.Context, source *basecatalog.ModelSource, reldir string) (<-chan ModelProviderRecord, error) {
		ch := make(chan ModelProviderRecord, 1)

		modelName := "test-model"
		model := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name: &modelName,
			},
		}

		ch <- ModelProviderRecord{
			Model:     model,
			Artifacts: []sharedmodels.CatalogArtifact{},
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
					Enabled: new(true),
				},
				Type: testProviderName,
			},
		},
	}

	l := NewModelLoader(services, basecatalog.NewBaseLoader([]string{}))
	ctx := context.Background()

	// First call updateSources to populate the SourceCollection
	err := l.updateSources("test-path", testConfig)
	if err != nil {
		t.Fatalf("updateSources() error = %v", err)
	}

	// This should not return an error even if repository operations fail
	// (errors are logged but don't stop the loading process)
	err = l.updateDatabase(ctx)
	if err != nil {
		t.Fatalf("updateDatabase() should not fail even with repository errors, got error = %v", err)
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Verify that no models were saved due to the error
	savedModels := mockModelRepo.GetSavedModels()
	if len(savedModels) != 0 {
		t.Errorf("Expected 0 models to be saved due to error, got %d", len(savedModels))
	}
}

func TestLoadCatalogSourcesWithNilEnabled(t *testing.T) {
	// Test that nil Enabled field is treated as enabled (per OpenAPI spec default: true)
	mockModelRepo := &MockCatalogModelRepository{}
	mockArtifactRepo := &MockCatalogArtifactRepository{}
	mockModelArtifactRepo := &MockCatalogModelArtifactRepository{}
	mockMetricsArtifactRepo := &MockCatalogMetricsArtifactRepository{}

	services := Services{
		CatalogModelRepository:           mockModelRepo,
		CatalogArtifactRepository:        mockArtifactRepo,
		CatalogModelArtifactRepository:   mockModelArtifactRepo,
		CatalogMetricsArtifactRepository: mockMetricsArtifactRepo,
		CatalogSourceRepository:          &MockCatalogSourceRepository{},
		PropertyOptionsRepository:        &MockPropertyOptionsRepository{},
	}

	// Register a test provider
	testProviderName := "test-nil-enabled-provider"
	RegisterModelProvider(testProviderName, func(ctx context.Context, source *basecatalog.ModelSource, reldir string) (<-chan ModelProviderRecord, error) {
		ch := make(chan ModelProviderRecord, 1)

		modelName := "test-model-nil-enabled"
		model := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name: &modelName,
			},
		}

		ch <- ModelProviderRecord{
			Model:     model,
			Artifacts: []sharedmodels.CatalogArtifact{},
		}
		close(ch)

		return ch, nil
	})

	testConfig := &basecatalog.SourceConfig{
		ModelCatalogs: []basecatalog.ModelSource{
			{
				CatalogSource: apimodels.CatalogSource{
					Id:      "test-catalog-nil-enabled",
					Name:    "Test Catalog Nil Enabled",
					Enabled: nil, // Nil should be treated as enabled
				},
				Type: testProviderName,
			},
		},
	}

	baseLoader := basecatalog.NewBaseLoader([]string{})
	l := NewModelLoader(services, baseLoader)
	ctx := context.Background()

	// First call updateSources to populate the SourceCollection
	err := l.updateSources("test-path", testConfig)
	if err != nil {
		t.Fatalf("updateSources() error = %v", err)
	}

	// Parse configs
	err = l.ParseAllConfigs()
	if err != nil {
		t.Fatalf("ParseAllConfigs() error = %v", err)
	}

	// Set leader mode and perform leader operations
	baseLoader.SetLeader(true)

	// Perform leader operations to perform database writes
	go func() {
		if err := l.PerformLeaderOperations(ctx, mapset.NewSet[string]()); err != nil {
			t.Logf("PerformLeaderOperations error: %v", err)
		}
	}()

	// Wait for leader mode to activate and process
	time.Sleep(300 * time.Millisecond)

	// Verify that the model WAS saved (because nil Enabled is treated as enabled)
	savedModels := mockModelRepo.GetSavedModels()
	if len(savedModels) != 1 {
		t.Errorf("Expected 1 model to be saved (nil Enabled should be treated as enabled), got %d", len(savedModels))
	}

	if len(savedModels) > 0 {
		savedModel := savedModels[0]
		if savedModel.GetAttributes() == nil || savedModel.GetAttributes().Name == nil {
			t.Error("Saved model should have attributes with name")
		} else if DisplayName(savedModel) != "test-model-nil-enabled" {
			t.Errorf("Expected model name 'test-model-nil-enabled', got '%s'", DisplayName(savedModel))
		}
	}
}

func TestMockRepositoryBehavior(t *testing.T) {
	mockRepo := &MockCatalogModelRepository{}

	// Test Save operation
	modelName := "test-model"
	model := &models.CatalogModelImpl{
		Attributes: &models.CatalogModelAttributes{
			Name: &modelName,
		},
	}

	savedModel, err := mockRepo.Save(model)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if savedModel.GetID() == nil {
		t.Error("Saved model should have an ID")
	}

	if *savedModel.GetID() != 1 {
		t.Errorf("Expected ID 1, got %d", *savedModel.GetID())
	}

	// Test GetByID operation
	retrievedModel, err := mockRepo.GetByID(1)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrievedModel.GetAttributes().Name == nil || *retrievedModel.GetAttributes().Name != modelName {
		t.Errorf("Retrieved model name mismatch, expected %s", modelName)
	}

	// Test GetByName operation
	retrievedModel, err = mockRepo.GetByName(modelName)
	if err != nil {
		t.Fatalf("GetByName() error = %v", err)
	}

	if retrievedModel.GetID() == nil || *retrievedModel.GetID() != 1 {
		t.Error("Retrieved model should have ID 1")
	}

	// Test List operation
	listWrapper, err := mockRepo.List(models.CatalogModelListOptions{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(listWrapper.Items) != 1 {
		t.Errorf("Expected 1 item in list, got %d", len(listWrapper.Items))
	}

	// Test not found scenarios
	_, err = mockRepo.GetByID(999)
	if err == nil {
		t.Error("GetByID() should return error for non-existent ID")
	}

	_, err = mockRepo.GetByName("non-existent")
	if err == nil {
		t.Error("GetByName() should return error for non-existent name")
	}
}

// MockCatalogModelRepositoryWithErrors is a mock that can simulate errors
type MockCatalogModelRepositoryWithErrors struct {
	MockCatalogModelRepository
	shouldFailSave bool
}

func (m *MockCatalogModelRepositoryWithErrors) Save(model models.CatalogModel) (models.CatalogModel, error) {
	if m.shouldFailSave {
		return nil, fmt.Errorf("simulated save error")
	}
	return m.MockCatalogModelRepository.Save(model)
}

// MockCatalogModelRepository mocks the CatalogModelRepository interface.
type MockCatalogModelRepository struct {
	mu          sync.RWMutex
	SavedModels []models.CatalogModel
	NextID      int32
	TypeID      int32 // Set to a non-zero value when testing GetFilterOptions scoping
}

func (m *MockCatalogModelRepository) GetByID(id int32) (models.CatalogModel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, model := range m.SavedModels {
		if model.GetID() != nil && *model.GetID() == id {
			return model, nil
		}
	}
	return nil, &MockNotFoundError{Entity: "CatalogModel", ID: id}
}

func (m *MockCatalogModelRepository) List(listOptions models.CatalogModelListOptions) (*mrmodels.ListWrapper[models.CatalogModel], error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return &mrmodels.ListWrapper[models.CatalogModel]{
		Items:    m.SavedModels,
		PageSize: int32(len(m.SavedModels)),
		Size:     int32(len(m.SavedModels)),
	}, nil
}

func (m *MockCatalogModelRepository) GetByName(name string) (models.CatalogModel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, model := range m.SavedModels {
		if model.GetAttributes() != nil && model.GetAttributes().Name != nil && *model.GetAttributes().Name == name {
			return model, nil
		}
	}
	return nil, &MockNotFoundError{Entity: "CatalogModel", ID: 0}
}

func (m *MockCatalogModelRepository) Save(model models.CatalogModel) (models.CatalogModel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.NextID++
	id := m.NextID

	// Create a new model with assigned ID
	savedModel := &models.CatalogModelImpl{
		ID:               &id,
		TypeID:           model.GetTypeID(),
		Attributes:       model.GetAttributes(),
		Properties:       model.GetProperties(),
		CustomProperties: model.GetCustomProperties(),
	}

	m.SavedModels = append(m.SavedModels, savedModel)
	return savedModel, nil
}

func (m *MockCatalogModelRepository) DeleteBySource(sourceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Mock implementation - no-op for testing
	return nil
}

func (m *MockCatalogModelRepository) DeleteByID(id int32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Mock implementation - no-op for testing
	return nil
}

func (m *MockCatalogModelRepository) GetDistinctSourceIDs() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Mock implementation - return empty list by default
	return []string{}, nil
}

func (m *MockCatalogModelRepository) GetTypeID() int32 {
	return m.TypeID
}

// GetSavedModels returns a copy of the saved models slice in a thread-safe manner.
// This should be used by tests instead of directly accessing SavedModels field.
func (m *MockCatalogModelRepository) GetSavedModels() []models.CatalogModel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Return a copy to prevent external modifications
	result := make([]models.CatalogModel, len(m.SavedModels))
	copy(result, m.SavedModels)
	return result
}

// Reset clears all saved models and resets the NextID counter in a thread-safe manner.
func (m *MockCatalogModelRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SavedModels = []models.CatalogModel{}
	m.NextID = 0
}

// MockCatalogModelArtifactRepository mocks the CatalogModelArtifactRepository interface.
type MockCatalogModelArtifactRepository struct {
	mu             sync.RWMutex
	SavedArtifacts []models.CatalogModelArtifact
	NextID         int32
}

func (m *MockCatalogModelArtifactRepository) GetByID(id int32) (models.CatalogModelArtifact, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, artifact := range m.SavedArtifacts {
		if artifact.GetID() != nil && *artifact.GetID() == id {
			return artifact, nil
		}
	}
	return nil, &MockNotFoundError{Entity: "CatalogModelArtifact", ID: id}
}

func (m *MockCatalogModelArtifactRepository) List(listOptions models.CatalogModelArtifactListOptions) (*mrmodels.ListWrapper[models.CatalogModelArtifact], error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return &mrmodels.ListWrapper[models.CatalogModelArtifact]{
		Items:    m.SavedArtifacts,
		PageSize: int32(len(m.SavedArtifacts)),
		Size:     int32(len(m.SavedArtifacts)),
	}, nil
}

func (m *MockCatalogModelArtifactRepository) Save(modelArtifact models.CatalogModelArtifact, parentResourceID *int32) (models.CatalogModelArtifact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.NextID++
	id := m.NextID

	// Create a new artifact with assigned ID
	savedArtifact := &models.CatalogModelArtifactImpl{
		ID:               &id,
		TypeID:           modelArtifact.GetTypeID(),
		Attributes:       modelArtifact.GetAttributes(),
		Properties:       modelArtifact.GetProperties(),
		CustomProperties: modelArtifact.GetCustomProperties(),
	}

	m.SavedArtifacts = append(m.SavedArtifacts, savedArtifact)
	return savedArtifact, nil
}

// GetSavedArtifacts returns a copy of the saved artifacts slice in a thread-safe manner.
func (m *MockCatalogModelArtifactRepository) GetSavedArtifacts() []models.CatalogModelArtifact {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]models.CatalogModelArtifact, len(m.SavedArtifacts))
	copy(result, m.SavedArtifacts)
	return result
}

// Reset clears all saved artifacts and resets the NextID counter in a thread-safe manner.
func (m *MockCatalogModelArtifactRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SavedArtifacts = []models.CatalogModelArtifact{}
	m.NextID = 0
}

// MockCatalogMetricsArtifactRepository mocks the CatalogMetricsArtifactRepository interface.
type MockCatalogMetricsArtifactRepository struct {
	mu           sync.RWMutex
	SavedMetrics []models.CatalogMetricsArtifact
	NextID       int32
}

func (m *MockCatalogMetricsArtifactRepository) GetByID(id int32) (models.CatalogMetricsArtifact, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, metrics := range m.SavedMetrics {
		if metrics.GetID() != nil && *metrics.GetID() == id {
			return metrics, nil
		}
	}
	return nil, &MockNotFoundError{Entity: "CatalogMetricsArtifact", ID: id}
}

func (m *MockCatalogMetricsArtifactRepository) List(listOptions models.CatalogMetricsArtifactListOptions) (*mrmodels.ListWrapper[models.CatalogMetricsArtifact], error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &mrmodels.ListWrapper[models.CatalogMetricsArtifact]{
		Items:    m.SavedMetrics,
		PageSize: int32(len(m.SavedMetrics)),
		Size:     int32(len(m.SavedMetrics)),
	}, nil
}

func (m *MockCatalogMetricsArtifactRepository) Save(metricsArtifact models.CatalogMetricsArtifact, parentResourceID *int32) (models.CatalogMetricsArtifact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.NextID++
	id := m.NextID

	// Create a new metrics artifact with assigned ID
	savedMetrics := &models.CatalogMetricsArtifactImpl{
		ID:               &id,
		TypeID:           metricsArtifact.GetTypeID(),
		Attributes:       metricsArtifact.GetAttributes(),
		Properties:       metricsArtifact.GetProperties(),
		CustomProperties: metricsArtifact.GetCustomProperties(),
	}

	m.SavedMetrics = append(m.SavedMetrics, savedMetrics)
	return savedMetrics, nil
}

func (m *MockCatalogMetricsArtifactRepository) BatchSave(metricsArtifacts []models.CatalogMetricsArtifact, parentResourceID *int32) ([]models.CatalogMetricsArtifact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	savedArtifacts := make([]models.CatalogMetricsArtifact, len(metricsArtifacts))

	for i, metricsArtifact := range metricsArtifacts {
		m.NextID++
		id := m.NextID

		// Create a new metrics artifact with assigned ID
		savedMetrics := &models.CatalogMetricsArtifactImpl{
			ID:               &id,
			TypeID:           metricsArtifact.GetTypeID(),
			Attributes:       metricsArtifact.GetAttributes(),
			Properties:       metricsArtifact.GetProperties(),
			CustomProperties: metricsArtifact.GetCustomProperties(),
		}

		m.SavedMetrics = append(m.SavedMetrics, savedMetrics)
		savedArtifacts[i] = savedMetrics
	}

	return savedArtifacts, nil
}

// GetSavedMetrics returns a copy of the saved metrics slice in a thread-safe manner.
func (m *MockCatalogMetricsArtifactRepository) GetSavedMetrics() []models.CatalogMetricsArtifact {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]models.CatalogMetricsArtifact, len(m.SavedMetrics))
	copy(result, m.SavedMetrics)
	return result
}

// Reset clears all saved metrics and resets the NextID counter in a thread-safe manner.
func (m *MockCatalogMetricsArtifactRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SavedMetrics = []models.CatalogMetricsArtifact{}
	m.NextID = 0
}

// MockCatalogArtifactRepository mocks the CatalogArtifactRepository interface.
type MockCatalogArtifactRepository struct {
	mu             sync.RWMutex
	SavedArtifacts []sharedmodels.CatalogArtifact
	NextID         int32
}

func (m *MockCatalogArtifactRepository) GetByID(id int32) (sharedmodels.CatalogArtifact, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, artifact := range m.SavedArtifacts {
		// Check both model and metrics artifacts for the ID
		if artifact.CatalogModelArtifact != nil && artifact.CatalogModelArtifact.GetID() != nil && *artifact.CatalogModelArtifact.GetID() == id {
			return artifact, nil
		}
		if artifact.CatalogMetricsArtifact != nil && artifact.CatalogMetricsArtifact.GetID() != nil && *artifact.CatalogMetricsArtifact.GetID() == id {
			return artifact, nil
		}
	}
	return sharedmodels.CatalogArtifact{}, &MockNotFoundError{Entity: "CatalogArtifact", ID: id}
}

func (m *MockCatalogArtifactRepository) List(listOptions sharedmodels.CatalogArtifactListOptions) (*mrmodels.ListWrapper[sharedmodels.CatalogArtifact], error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return &mrmodels.ListWrapper[sharedmodels.CatalogArtifact]{
		Items:    m.SavedArtifacts,
		PageSize: int32(len(m.SavedArtifacts)),
		Size:     int32(len(m.SavedArtifacts)),
	}, nil
}

func (m *MockCatalogArtifactRepository) DeleteByParentID(artifactType string, parentResourceID int32) error {
	// Simple mock implementation - could be enhanced to actually filter and delete
	return nil
}

func (m *MockCatalogArtifactRepository) CountByParentIDs(parentIDs []int32) (map[int32]map[string]int32, error) {
	return make(map[int32]map[string]int32), nil
}

// MockNotFoundError represents an error when an entity is not found.
type MockNotFoundError struct {
	Entity string
	ID     int32
}

func (e *MockNotFoundError) Error() string {
	return fmt.Sprintf("%s with ID %d not found", e.Entity, e.ID)
}

// MockPropertyOptionsRepository mocks the PropertyOptionsRepository interface.
type MockPropertyOptionsRepository struct {
	RefreshCalls []sharedmodels.PropertyOptionType
	ListCalls    []struct {
		Type   sharedmodels.PropertyOptionType
		TypeID int32
	}
	MockOptions map[sharedmodels.PropertyOptionType]map[int32][]sharedmodels.PropertyOption
}

func NewMockPropertyOptionsRepository() *MockPropertyOptionsRepository {
	return &MockPropertyOptionsRepository{
		RefreshCalls: make([]sharedmodels.PropertyOptionType, 0),
		ListCalls: make([]struct {
			Type   sharedmodels.PropertyOptionType
			TypeID int32
		}, 0),
		MockOptions: make(map[sharedmodels.PropertyOptionType]map[int32][]sharedmodels.PropertyOption),
	}
}

func (m *MockPropertyOptionsRepository) Refresh(t sharedmodels.PropertyOptionType) error {
	m.RefreshCalls = append(m.RefreshCalls, t)
	return nil
}

func (m *MockPropertyOptionsRepository) List(t sharedmodels.PropertyOptionType, typeID int32) ([]sharedmodels.PropertyOption, error) {
	m.ListCalls = append(m.ListCalls, struct {
		Type   sharedmodels.PropertyOptionType
		TypeID int32
	}{Type: t, TypeID: typeID})

	if typeMap, exists := m.MockOptions[t]; exists {
		if options, exists := typeMap[typeID]; exists {
			return options, nil
		}
	}

	// Return empty slice by default
	return []sharedmodels.PropertyOption{}, nil
}

// SetMockOptions allows tests to set up mock data for specific types and typeIDs.
func (m *MockPropertyOptionsRepository) SetMockOptions(t sharedmodels.PropertyOptionType, typeID int32, options []sharedmodels.PropertyOption) {
	if m.MockOptions[t] == nil {
		m.MockOptions[t] = make(map[int32][]sharedmodels.PropertyOption)
	}
	m.MockOptions[t][typeID] = options
}

// MockCatalogSourceRepository mocks the CatalogSourceRepository interface.
type MockCatalogSourceRepository struct {
	mu      sync.RWMutex
	Sources []sharedmodels.CatalogSource
}

func (m *MockCatalogSourceRepository) GetBySourceID(sourceID string) (sharedmodels.CatalogSource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, s := range m.Sources {
		if attrs := s.GetAttributes(); attrs != nil && attrs.Name != nil && *attrs.Name == sourceID {
			return s, nil
		}
	}
	return nil, nil
}

func (m *MockCatalogSourceRepository) Save(source sharedmodels.CatalogSource) (sharedmodels.CatalogSource, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Sources = append(m.Sources, source)
	return source, nil
}

func (m *MockCatalogSourceRepository) Delete(sourceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Mock implementation - no-op for testing
	return nil
}

func (m *MockCatalogSourceRepository) GetAll() ([]sharedmodels.CatalogSource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Return a copy to prevent external modifications
	result := make([]sharedmodels.CatalogSource, len(m.Sources))
	copy(result, m.Sources)
	return result, nil
}

func (m *MockCatalogSourceRepository) GetAllStatuses() (map[string]sharedmodels.SourceStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]sharedmodels.SourceStatus)
	for _, source := range m.Sources {
		if attrs := source.GetAttributes(); attrs != nil && attrs.Name != nil {
			status := sharedmodels.SourceStatus{}
			if props := source.GetProperties(); props != nil {
				for _, prop := range *props {
					switch prop.Name {
					case "status":
						if prop.StringValue != nil {
							status.Status = *prop.StringValue
						}
					case "error":
						if prop.StringValue != nil {
							status.Error = *prop.StringValue
						}
					}
				}
			}
			result[*attrs.Name] = status
		}
	}
	return result, nil
}

func TestAPIProviderGetPerformanceArtifacts(t *testing.T) {
	// This test verifies that the APIProvider interface has GetPerformanceArtifacts method
	// The actual implementation is tested in db_catalog_test.go

	// Create a mock provider to verify interface compliance
	services := Services{
		CatalogModelRepository:           &MockCatalogModelRepository{},
		CatalogArtifactRepository:        &MockCatalogArtifactRepository{},
		CatalogModelArtifactRepository:   &MockCatalogModelArtifactRepository{},
		CatalogMetricsArtifactRepository: &MockCatalogMetricsArtifactRepository{},
		CatalogSourceRepository:          &MockCatalogSourceRepository{},
		PropertyOptionsRepository:        &MockPropertyOptionsRepository{},
	}
	provider := NewDBCatalog(services, nil)

	// Verify provider implements APIProvider interface with GetPerformanceArtifacts
	var _ APIProvider = provider

	// Basic test - should return error for non-existent model
	ctx := context.Background()
	_, err := provider.GetPerformanceArtifacts(ctx, "non-existent-model", "source-1", ListPerformanceArtifactsParams{
		TargetRPS:       100,
		Recommendations: true,
		PageSize:        10,
	})

	// Should get an error since the model doesn't exist
	assert.Error(t, err)
}

// TestAPIProviderInterface verifies that the APIProvider interface supports
// all required fields in ListPerformanceArtifactsParams
func TestAPIProviderInterface(t *testing.T) {
	services := Services{
		CatalogModelRepository:           &MockCatalogModelRepository{},
		CatalogArtifactRepository:        &MockCatalogArtifactRepository{},
		CatalogModelArtifactRepository:   &MockCatalogModelArtifactRepository{},
		CatalogMetricsArtifactRepository: &MockCatalogMetricsArtifactRepository{},
		CatalogSourceRepository:          &MockCatalogSourceRepository{},
		PropertyOptionsRepository:        &MockPropertyOptionsRepository{},
	}
	var provider APIProvider = NewDBCatalog(services, nil)

	params := ListPerformanceArtifactsParams{
		TargetRPS:             100,
		Recommendations:       true,
		RPSProperty:           "custom_rps",
		LatencyProperty:       "custom_latency",
		HardwareCountProperty: "custom_hw_count",
		HardwareTypeProperty:  "custom_hw_type",
	}

	// Should compile without errors and be callable
	ctx := context.Background()
	_, err := provider.GetPerformanceArtifacts(ctx, "test-model", "source-1", params)
	assert.Error(t, err) // Expected error since model doesn't exist
}
