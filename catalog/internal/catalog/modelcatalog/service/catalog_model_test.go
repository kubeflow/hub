package service

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"github.com/kubeflow/hub/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogModelRepository(t *testing.T) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testDatastoreSpec())
	defer cleanup()

	// Create or get the CatalogModel type ID
	typeID := getCatalogModelTypeID(t, sharedDB)
	repo := NewCatalogModelRepository(sharedDB, typeID)

	t.Run("TestSave", func(t *testing.T) {
		// Test creating a new catalog model
		catalogModel := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       new("test-catalog-model"),
				ExternalID: new("catalog-ext-123"),
			},
			Properties: &[]dbmodels.Properties{
				{
					Name:        "description",
					StringValue: new("Test catalog model description"),
				},
			},
			CustomProperties: &[]dbmodels.Properties{
				{
					Name:        "custom-prop",
					StringValue: new("custom-value"),
				},
			},
		}

		saved, err := repo.Save(catalogModel)
		require.NoError(t, err)
		require.NotNil(t, saved)
		require.NotNil(t, saved.GetID())
		assert.Equal(t, "test-catalog-model", *saved.GetAttributes().Name)
		assert.Equal(t, "catalog-ext-123", *saved.GetAttributes().ExternalID)

		// Test updating the same model
		catalogModel.ID = saved.GetID()
		catalogModel.GetAttributes().Name = new("updated-catalog-model")
		// Preserve CreateTimeSinceEpoch from the saved entity
		catalogModel.GetAttributes().CreateTimeSinceEpoch = saved.GetAttributes().CreateTimeSinceEpoch

		updated, err := repo.Save(catalogModel)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, *saved.GetID(), *updated.GetID())
		assert.Equal(t, "updated-catalog-model", *updated.GetAttributes().Name)
	})

	t.Run("TestGetByID", func(t *testing.T) {
		// First create a model to retrieve
		catalogModel := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       new("get-test-catalog-model"),
				ExternalID: new("get-catalog-ext-123"),
			},
		}

		saved, err := repo.Save(catalogModel)
		require.NoError(t, err)
		require.NotNil(t, saved.GetID())

		// Test retrieving by ID
		retrieved, err := repo.GetByID(*saved.GetID())
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, *saved.GetID(), *retrieved.GetID())
		assert.Equal(t, "get-test-catalog-model", *retrieved.GetAttributes().Name)
		assert.Equal(t, "get-catalog-ext-123", *retrieved.GetAttributes().ExternalID)

		// Test retrieving non-existent ID
		_, err = repo.GetByID(99999)
		assert.ErrorIs(t, err, ErrCatalogModelNotFound)
	})

	t.Run("TestList", func(t *testing.T) {
		// Create multiple models for listing
		testModels := []*models.CatalogModelImpl{
			{
				Attributes: &models.CatalogModelAttributes{
					Name:       new("list-catalog-model-1"),
					ExternalID: new("list-catalog-ext-1"),
				},
			},
			{
				Attributes: &models.CatalogModelAttributes{
					Name:       new("list-catalog-model-2"),
					ExternalID: new("list-catalog-ext-2"),
				},
			},
		}

		// Save all test models
		var savedModels []models.CatalogModel
		for _, model := range testModels {
			saved, err := repo.Save(model)
			require.NoError(t, err)
			savedModels = append(savedModels, saved)
		}

		// Test listing all models
		listOptions := models.CatalogModelListOptions{}
		result, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.GreaterOrEqual(t, len(result.Items), 2) // At least our 2 test models

		// Test filtering by name
		nameFilter := "list-catalog-model-1"
		listOptions = models.CatalogModelListOptions{
			Name: &nameFilter,
		}
		result, err = repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, len(result.Items))
		assert.Equal(t, "list-catalog-model-1", *result.Items[0].GetAttributes().Name)

		// Test filtering by external ID
		externalIDFilter := "list-catalog-ext-2"
		listOptions = models.CatalogModelListOptions{
			ExternalID: &externalIDFilter,
		}
		result, err = repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, len(result.Items))
		assert.Equal(t, "list-catalog-ext-2", *result.Items[0].GetAttributes().ExternalID)
	})

	t.Run("TestGetByName", func(t *testing.T) {
		// First create a model to retrieve by name
		catalogModel := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       new("get-by-name-test-model"),
				ExternalID: new("get-by-name-ext-123"),
			},
		}

		saved, err := repo.Save(catalogModel)
		require.NoError(t, err)
		require.NotNil(t, saved.GetID())

		// Test retrieving by name
		retrieved, err := repo.GetByName("get-by-name-test-model")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, *saved.GetID(), *retrieved.GetID())
		assert.Equal(t, "get-by-name-test-model", *retrieved.GetAttributes().Name)
		assert.Equal(t, "get-by-name-ext-123", *retrieved.GetAttributes().ExternalID)

		// Test retrieving non-existent name
		_, err = repo.GetByName("non-existent-model")
		assert.ErrorIs(t, err, ErrCatalogModelNotFound)
	})

	t.Run("TestSameNameDifferentSources", func(t *testing.T) {
		// Allow multiple models with the same display name as long as they have different sources.
		sharedName := "shared-model-name"
		modelA := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       new(sharedName),
				ExternalID: new("ext-a"),
			},
			Properties: &[]dbmodels.Properties{
				{Name: "source_id", StringValue: new("source-a")},
			},
		}
		modelB := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       new(sharedName),
				ExternalID: new("ext-b"),
			},
			Properties: &[]dbmodels.Properties{
				{Name: "source_id", StringValue: new("source-b")},
			},
		}

		savedA, err := repo.Save(modelA)
		require.NoError(t, err)
		require.NotNil(t, savedA.GetID())

		savedB, err := repo.Save(modelB)
		require.NoError(t, err)
		require.NotNil(t, savedB.GetID())

		assert.NotEqual(t, *savedA.GetID(), *savedB.GetID(), "same display name from different sources must be distinct rows")

		// Lookup by namespaced name returns the correct model
		retrievedA, err := repo.GetByName("source-a:" + sharedName)
		require.NoError(t, err)
		assert.Equal(t, *savedA.GetID(), *retrievedA.GetID())
		assert.Equal(t, "source-a:"+sharedName, *retrievedA.GetAttributes().Name)

		retrievedB, err := repo.GetByName("source-b:" + sharedName)
		require.NoError(t, err)
		assert.Equal(t, *savedB.GetID(), *retrievedB.GetID())
		assert.Equal(t, "source-b:"+sharedName, *retrievedB.GetAttributes().Name)
	})

	t.Run("TestSameModelAndExternalIDAcrossSources", func(t *testing.T) {
		// Same display name and same external_id (e.g. same HF model in two sources) can both be saved.
		// external_id is namespaced per source in the DB to satisfy UNIQUE(external_id).
		sharedName := "Qwen/Qwen3.5-9B"
		sharedExternalID := "Qwen/Qwen3.5-9B"
		modelSource1 := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       new(sharedName),
				ExternalID: new(sharedExternalID),
			},
			Properties: &[]dbmodels.Properties{
				{Name: "source_id", StringValue: new("hugging_face_models")},
			},
		}
		modelSource2 := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       new(sharedName),
				ExternalID: new(sharedExternalID),
			},
			Properties: &[]dbmodels.Properties{
				{Name: "source_id", StringValue: new("hugging_face_models_2")},
			},
		}

		saved1, err := repo.Save(modelSource1)
		require.NoError(t, err)
		require.NotNil(t, saved1.GetID())

		saved2, err := repo.Save(modelSource2)
		require.NoError(t, err)
		require.NotNil(t, saved2.GetID())

		assert.NotEqual(t, *saved1.GetID(), *saved2.GetID(), "same model from different sources must be distinct rows")

		// Both stored with namespaced name
		assert.Equal(t, "hugging_face_models:"+sharedName, *saved1.GetAttributes().Name)
		assert.Equal(t, "hugging_face_models_2:"+sharedName, *saved2.GetAttributes().Name)
		// ExternalID returned in display form (no source prefix)
		require.NotNil(t, saved1.GetAttributes().ExternalID)
		assert.Equal(t, sharedExternalID, *saved1.GetAttributes().ExternalID)
		require.NotNil(t, saved2.GetAttributes().ExternalID)
		assert.Equal(t, sharedExternalID, *saved2.GetAttributes().ExternalID)

		// GetByName with namespaced name returns the correct model
		retrieved1, err := repo.GetByName("hugging_face_models:" + sharedName)
		require.NoError(t, err)
		assert.Equal(t, *saved1.GetID(), *retrieved1.GetID())
		assert.Equal(t, sharedExternalID, *retrieved1.GetAttributes().ExternalID)

		retrieved2, err := repo.GetByName("hugging_face_models_2:" + sharedName)
		require.NoError(t, err)
		assert.Equal(t, *saved2.GetID(), *retrieved2.GetID())
		assert.Equal(t, sharedExternalID, *retrieved2.GetAttributes().ExternalID)
	})

	t.Run("TestListByExternalIDReturnsModelsFromAllSources", func(t *testing.T) {
		// When external_id is namespaced in DB (sourceId:externalId), listing by raw external_id
		// should return all models that match across any source (backward compatibility).
		sharedExtID := "shared-ext-list-test"
		model1 := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       new("list-ext-model-a"),
				ExternalID: new(sharedExtID),
			},
			Properties: &[]dbmodels.Properties{
				{Name: "source_id", StringValue: new("src-list-1")},
			},
		}
		model2 := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       new("list-ext-model-b"),
				ExternalID: new(sharedExtID),
			},
			Properties: &[]dbmodels.Properties{
				{Name: "source_id", StringValue: new("src-list-2")},
			},
		}
		_, err := repo.Save(model1)
		require.NoError(t, err)
		_, err = repo.Save(model2)
		require.NoError(t, err)

		listOptions := models.CatalogModelListOptions{
			ExternalID: &sharedExtID,
		}
		result, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Items, 2, "filtering by raw external_id should return all models from all sources")
		extIDs := make([]string, len(result.Items))
		for i, m := range result.Items {
			require.NotNil(t, m.GetAttributes().ExternalID)
			extIDs[i] = *m.GetAttributes().ExternalID
		}
		assert.ElementsMatch(t, extIDs, []string{sharedExtID, sharedExtID})
	})

	t.Run("TestFilterQueryExternalIDReturnsModelsFromAllSources", func(t *testing.T) {
		// filterQuery externalId = "value" should match both exact and namespaced stored values.
		sharedExtID := "filterquery-ext-shared"
		model1 := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       new("fq-ext-model-a"),
				ExternalID: new(sharedExtID),
			},
			Properties: &[]dbmodels.Properties{
				{Name: "source_id", StringValue: new("src-fq-1")},
			},
		}
		model2 := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       new("fq-ext-model-b"),
				ExternalID: new(sharedExtID),
			},
			Properties: &[]dbmodels.Properties{
				{Name: "source_id", StringValue: new("src-fq-2")},
			},
		}
		_, err := repo.Save(model1)
		require.NoError(t, err)
		_, err = repo.Save(model2)
		require.NoError(t, err)

		filterQuery := `externalId = "` + sharedExtID + `"`
		listOptions := models.CatalogModelListOptions{
			Pagination: dbmodels.Pagination{
				FilterQuery: &filterQuery,
			},
		}
		result, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Items, 2, "filterQuery externalId = value should return all models from all sources")
		for _, m := range result.Items {
			require.NotNil(t, m.GetAttributes().ExternalID)
			assert.Equal(t, sharedExtID, *m.GetAttributes().ExternalID)
		}
	})

	t.Run("TestUpdateWithID", func(t *testing.T) {
		// First create a model
		catalogModel := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       new("update-test-model"),
				ExternalID: new("update-ext-123"),
			},
			Properties: &[]dbmodels.Properties{
				{
					Name:        "version",
					StringValue: new("1.0.0"),
				},
			},
			CustomProperties: &[]dbmodels.Properties{
				{
					Name:        "environment",
					StringValue: new("dev"),
				},
			},
		}

		saved, err := repo.Save(catalogModel)
		require.NoError(t, err)
		require.NotNil(t, saved.GetID())

		// Update the model with ID specified
		updateModel := &models.CatalogModelImpl{
			ID: saved.GetID(), // Specify the ID for update
			Attributes: &models.CatalogModelAttributes{
				Name:                 new("updated-test-model"),
				ExternalID:           new("updated-ext-456"),
				CreateTimeSinceEpoch: saved.GetAttributes().CreateTimeSinceEpoch, // Preserve create time
			},
			Properties: &[]dbmodels.Properties{
				{
					Name:        "version",
					StringValue: new("2.0.0"), // Updated version
				},
				{
					Name:        "description",
					StringValue: new("Updated description"), // New property
				},
			},
			CustomProperties: &[]dbmodels.Properties{
				{
					Name:        "environment",
					StringValue: new("prod"), // Updated environment
				},
			},
		}

		updated, err := repo.Save(updateModel)
		require.NoError(t, err)
		require.NotNil(t, updated)

		// Verify the update
		assert.Equal(t, *saved.GetID(), *updated.GetID()) // Same ID
		assert.Equal(t, "updated-test-model", *updated.GetAttributes().Name)
		assert.Equal(t, "updated-ext-456", *updated.GetAttributes().ExternalID)

		// Verify properties were updated
		require.NotNil(t, updated.GetProperties())
		assert.Len(t, *updated.GetProperties(), 2)

		// Verify custom properties were updated
		require.NotNil(t, updated.GetCustomProperties())
		assert.Len(t, *updated.GetCustomProperties(), 1)
	})

	t.Run("TestUpdateWithName", func(t *testing.T) {
		// First create a model
		catalogModel := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       new("update-by-name-model"),
				ExternalID: new("update-by-name-ext-123"),
			},
			Properties: &[]dbmodels.Properties{
				{
					Name:        "status",
					StringValue: new("draft"),
				},
			},
		}

		saved, err := repo.Save(catalogModel)
		require.NoError(t, err)
		require.NotNil(t, saved.GetID())

		// Update the model without specifying ID (should lookup by name)
		updateModel := &models.CatalogModelImpl{
			// No ID specified - should trigger name lookup in Save method
			Attributes: &models.CatalogModelAttributes{
				Name:       new("update-by-name-model"), // Same name to trigger lookup
				ExternalID: new("updated-by-name-ext-789"),
			},
			Properties: &[]dbmodels.Properties{
				{
					Name:        "status",
					StringValue: new("published"), // Updated status
				},
				{
					Name:        "category",
					StringValue: new("ml-model"), // New property
				},
			},
		}

		updated, err := repo.Save(updateModel)
		require.NoError(t, err)
		require.NotNil(t, updated)

		// Verify the update happened (same ID, updated fields)
		assert.Equal(t, *saved.GetID(), *updated.GetID()) // Should have same ID from lookup
		assert.Equal(t, "update-by-name-model", *updated.GetAttributes().Name)
		assert.Equal(t, "updated-by-name-ext-789", *updated.GetAttributes().ExternalID)

		// Verify properties were updated
		require.NotNil(t, updated.GetProperties())
		assert.Len(t, *updated.GetProperties(), 2)
	})

	t.Run("TestListWithPropertiesAndCustomProperties", func(t *testing.T) {
		// Create a model with both properties and custom properties
		catalogModel := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       new("props-test-catalog-model"),
				ExternalID: new("props-catalog-ext-123"),
			},
			Properties: &[]dbmodels.Properties{
				{
					Name:        "version",
					StringValue: new("1.0.0"),
				},
				{
					Name:     "priority",
					IntValue: new(int32(5)),
				},
			},
			CustomProperties: &[]dbmodels.Properties{
				{
					Name:        "team",
					StringValue: new("ml-team"),
				},
				{
					Name:      "active",
					BoolValue: new(true),
				},
			},
		}

		saved, err := repo.Save(catalogModel)
		require.NoError(t, err)
		require.NotNil(t, saved)

		// Retrieve and verify properties
		retrieved, err := repo.GetByID(*saved.GetID())
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		// Check regular properties
		require.NotNil(t, retrieved.GetProperties())
		assert.Len(t, *retrieved.GetProperties(), 2)

		// Check custom properties
		require.NotNil(t, retrieved.GetCustomProperties())
		assert.Len(t, *retrieved.GetCustomProperties(), 2)
	})

	t.Run("TestAccuracySorting", func(t *testing.T) {
		// Get the CatalogMetricsArtifact type ID for creating accuracy metrics
		metricsTypeID := getCatalogMetricsArtifactTypeID(t, sharedDB)
		metricsRepo := NewCatalogMetricsArtifactRepository(sharedDB, metricsTypeID)

		// Create test models with different accuracy scores
		testModels := []struct {
			name     string
			accuracy *float64 // nil means no accuracy score
		}{
			{"high-accuracy-model", new(95.5)},
			{"medium-accuracy-model", new(75.0)},
			{"low-accuracy-model", new(45.2)},
			{"no-accuracy-model", nil},
			{"zero-accuracy-model", new(0.0)},
		}

		var savedModels []models.CatalogModel
		for _, testModel := range testModels {
			// Create the model
			catalogModel := &models.CatalogModelImpl{
				Attributes: &models.CatalogModelAttributes{
					Name:       new(testModel.name),
					ExternalID: new(testModel.name + "-ext"),
				},
			}

			savedModel, err := repo.Save(catalogModel)
			require.NoError(t, err)
			savedModels = append(savedModels, savedModel)

			// Create accuracy metrics artifact if accuracy score is provided
			if testModel.accuracy != nil {
				metricsArtifact := &models.CatalogMetricsArtifactImpl{
					Attributes: &models.CatalogMetricsArtifactAttributes{
						Name:        new(fmt.Sprintf("accuracy-metrics-%s", testModel.name)),
						ExternalID:  new(fmt.Sprintf("accuracy-metrics-%s", testModel.name)),
						MetricsType: models.MetricsTypeAccuracy,
					},
					CustomProperties: &[]dbmodels.Properties{
						{
							Name:        "overall_average",
							DoubleValue: testModel.accuracy,
						},
						{
							Name:        "benchmark1",
							DoubleValue: new(*testModel.accuracy + 1.0), // Individual benchmark score
						},
						{
							Name:        "benchmark2",
							DoubleValue: new(*testModel.accuracy - 1.0), // Individual benchmark score
						},
					},
				}

				_, err := metricsRepo.Save(metricsArtifact, savedModel.GetID())
				require.NoError(t, err)
			}
		}

		// Test ACCURACY sorting DESC (default)
		listOptions := models.CatalogModelListOptions{
			Pagination: dbmodels.Pagination{
				OrderBy:   new("artifacts.overall_average.double_value"),
				SortOrder: new("DESC"),
			},
		}
		result, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify DESC order: high accuracy first, then medium, low, zero, then models without accuracy scores
		var accuracyModelsFound []string
		for _, model := range result.Items {
			name := *model.GetAttributes().Name
			if name == "high-accuracy-model" || name == "medium-accuracy-model" ||
				name == "low-accuracy-model" || name == "zero-accuracy-model" || name == "no-accuracy-model" {
				accuracyModelsFound = append(accuracyModelsFound, name)
			}
		}

		// We should have found all our test models
		require.GreaterOrEqual(t, len(accuracyModelsFound), 5)

		// Check that high accuracy comes before medium accuracy
		highIdx := findIndex(accuracyModelsFound, "high-accuracy-model")
		mediumIdx := findIndex(accuracyModelsFound, "medium-accuracy-model")
		lowIdx := findIndex(accuracyModelsFound, "low-accuracy-model")
		zeroIdx := findIndex(accuracyModelsFound, "zero-accuracy-model")
		noAccIdx := findIndex(accuracyModelsFound, "no-accuracy-model")

		require.NotEqual(t, -1, highIdx, "high-accuracy-model not found in results")
		require.NotEqual(t, -1, mediumIdx, "medium-accuracy-model not found in results")
		require.NotEqual(t, -1, lowIdx, "low-accuracy-model not found in results")
		require.NotEqual(t, -1, zeroIdx, "zero-accuracy-model not found in results")
		require.NotEqual(t, -1, noAccIdx, "no-accuracy-model not found in results")

		// Verify DESC ordering: high > medium > low > zero > no-accuracy
		assert.Less(t, highIdx, mediumIdx, "high accuracy model should come before medium accuracy")
		assert.Less(t, mediumIdx, lowIdx, "medium accuracy model should come before low accuracy")
		assert.Less(t, lowIdx, zeroIdx, "low accuracy model should come before zero accuracy")
		assert.Less(t, zeroIdx, noAccIdx, "zero accuracy model should come before no accuracy")

		// Test ACCURACY sorting ASC
		listOptions = models.CatalogModelListOptions{
			Pagination: dbmodels.Pagination{
				OrderBy:   new("artifacts.overall_average.double_value"),
				SortOrder: new("ASC"),
			},
		}
		result, err = repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Find our test models in ASC results
		accuracyModelsFound = []string{}
		for _, model := range result.Items {
			name := *model.GetAttributes().Name
			if name == "high-accuracy-model" || name == "medium-accuracy-model" ||
				name == "low-accuracy-model" || name == "zero-accuracy-model" || name == "no-accuracy-model" {
				accuracyModelsFound = append(accuracyModelsFound, name)
			}
		}

		// Get indices for ASC order
		highIdxAsc := findIndex(accuracyModelsFound, "high-accuracy-model")
		mediumIdxAsc := findIndex(accuracyModelsFound, "medium-accuracy-model")
		lowIdxAsc := findIndex(accuracyModelsFound, "low-accuracy-model")
		zeroIdxAsc := findIndex(accuracyModelsFound, "zero-accuracy-model")
		noAccIdxAsc := findIndex(accuracyModelsFound, "no-accuracy-model")

		// Verify ASC ordering: zero < low < medium < high, with no-accuracy still last
		assert.Less(t, zeroIdxAsc, lowIdxAsc, "zero accuracy model should come before low accuracy in ASC")
		assert.Less(t, lowIdxAsc, mediumIdxAsc, "low accuracy model should come before medium accuracy in ASC")
		assert.Less(t, mediumIdxAsc, highIdxAsc, "medium accuracy model should come before high accuracy in ASC")
		assert.Less(t, highIdxAsc, noAccIdxAsc, "models with accuracy should come before models without accuracy in ASC")

		// Test fallback to standard sorting for non-ACCURACY orderBy
		listOptions = models.CatalogModelListOptions{
			Pagination: dbmodels.Pagination{
				OrderBy:   new("ID"),
				SortOrder: new("ASC"),
			},
		}
		result, err = repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		// Should not error and should return results (detailed verification not needed since we're testing fallback)
		assert.Greater(t, len(result.Items), 0)
	})

	t.Run("TestAccuracySortingPagination", func(t *testing.T) {
		// Get the CatalogMetricsArtifact type ID for creating accuracy metrics
		metricsTypeID := getCatalogMetricsArtifactTypeID(t, sharedDB)
		metricsRepo := NewCatalogMetricsArtifactRepository(sharedDB, metricsTypeID)

		// Create 5 test models with accuracy scores for pagination testing
		// Use unique names to avoid interference with other tests
		testModels := []struct {
			name     string
			accuracy float64
		}{
			{"pagination-test-model-a", 95.0}, // Should be first in DESC order
			{"pagination-test-model-b", 85.0},
			{"pagination-test-model-c", 75.0},
			{"pagination-test-model-d", 65.0},
			{"pagination-test-model-e", 55.0}, // Should be last in DESC order
		}

		var savedModels []models.CatalogModel
		for _, testModel := range testModels {
			// Create the model
			catalogModel := &models.CatalogModelImpl{
				Attributes: &models.CatalogModelAttributes{
					Name:       new(testModel.name),
					ExternalID: new(testModel.name + "-ext"),
				},
			}

			savedModel, err := repo.Save(catalogModel)
			require.NoError(t, err)
			savedModels = append(savedModels, savedModel)

			// Create accuracy metrics artifact
			metricsArtifact := &models.CatalogMetricsArtifactImpl{
				Attributes: &models.CatalogMetricsArtifactAttributes{
					Name:        new(fmt.Sprintf("accuracy-metrics-%s", testModel.name)),
					ExternalID:  new(fmt.Sprintf("accuracy-metrics-%s", testModel.name)),
					MetricsType: models.MetricsTypeAccuracy,
				},
				CustomProperties: &[]dbmodels.Properties{
					{
						Name:        "overall_average",
						DoubleValue: &testModel.accuracy,
					},
				},
			}

			_, err = metricsRepo.Save(metricsArtifact, savedModel.GetID())
			require.NoError(t, err)
		}

		// Test pagination by collecting all pages
		// This approach is more robust and less sensitive to test interference
		listOptions := models.CatalogModelListOptions{
			Pagination: dbmodels.Pagination{
				OrderBy:   new("artifacts.overall_average.double_value"),
				SortOrder: new("DESC"),
				PageSize:  new(int32(2)),
			},
		}

		// Collect all our test models across pages
		var allPaginatedModels []models.CatalogModel
		var pageCount int
		currentToken := (*string)(nil)

		for {
			pageCount++
			if currentToken != nil {
				listOptions.Pagination.NextPageToken = currentToken
			}

			page, err := repo.List(listOptions)
			require.NoError(t, err)
			require.NotNil(t, page)
			assert.Equal(t, int32(2), page.PageSize)

			// Filter to only include our test models
			for _, model := range page.Items {
				if strings.HasPrefix(*model.GetAttributes().Name, "pagination-test-model-") {
					allPaginatedModels = append(allPaginatedModels, model)
				}
			}

			// Stop if no more pages or we've collected all our test models
			if page.NextPageToken == "" || len(allPaginatedModels) >= 5 {
				if page.NextPageToken == "" {
					t.Logf("Pagination completed in %d pages", pageCount)
				}
				break
			}
			currentToken = &page.NextPageToken

			// Safety check to prevent infinite loop
			if pageCount > 10 {
				t.Fatal("Too many pages, might be an infinite loop")
			}
		}

		// Verify we collected all our test models
		assert.GreaterOrEqual(t, len(allPaginatedModels), 5, "Should have found all pagination test models")

		// Extract names and verify ordering (DESC by accuracy)
		var modelNames []string
		for _, model := range allPaginatedModels {
			if strings.HasPrefix(*model.GetAttributes().Name, "pagination-test-model-") {
				modelNames = append(modelNames, *model.GetAttributes().Name)
			}
		}

		// Check that pagination preserved the correct ordering
		// In DESC order: a(95.0) -> b(85.0) -> c(75.0) -> d(65.0) -> e(55.0)
		expectedOrder := []string{
			"pagination-test-model-a", // 95.0 (highest)
			"pagination-test-model-b", // 85.0
			"pagination-test-model-c", // 75.0
			"pagination-test-model-d", // 65.0
			"pagination-test-model-e", // 55.0 (lowest)
		}

		// Verify our test models appear in correct order (allowing for other models in between)
		lastIndex := -1
		for _, expectedModel := range expectedOrder {
			foundIndex := -1
			for i, actualModel := range modelNames {
				if actualModel == expectedModel {
					foundIndex = i
					break
				}
			}
			assert.NotEqual(t, -1, foundIndex, "Should find model %s", expectedModel)
			if foundIndex != -1 {
				assert.Greater(t, foundIndex, lastIndex, "Model %s should appear after previous models in DESC order", expectedModel)
				lastIndex = foundIndex
			}
		}

		// Test ASC pagination briefly to verify token generation works in both directions
		listOptions = models.CatalogModelListOptions{
			Pagination: dbmodels.Pagination{
				OrderBy:   new("artifacts.overall_average.double_value"),
				SortOrder: new("ASC"),
				PageSize:  new(int32(3)),
			},
		}

		pageAsc, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, pageAsc)

		// Just verify that ASC pagination works and generates tokens when there are more results
		if len(pageAsc.Items) == 3 {
			assert.NotEmpty(t, pageAsc.NextPageToken, "Should have next page token in ASC order when page is full")
		}
	})

	t.Run("TestNameOrdering", func(t *testing.T) {
		// Create models from two different sources so stored names have source prefixes.
		// This validates that sorting is by display name (after the colon) not the raw stored name.
		// Without the fix, "source-z:alpha-model" would sort after "source-a:zebra-model".
		type modelSpec struct {
			displayName string
			sourceID    string
		}
		testModels := []modelSpec{
			// source-z prefix ensures raw-name sort would put alpha/beta last
			{displayName: "alpha-model", sourceID: "source-z"},
			{displayName: "beta-model", sourceID: "source-z"},
			// source-a prefix ensures raw-name sort would put gamma/zebra first
			{displayName: "gamma-model", sourceID: "source-a"},
			{displayName: "zebra-model", sourceID: "source-a"},
			{displayName: "delta-model", sourceID: "source-m"},
			// colon in display name: stored as "source-a:epsilon:v2", display name is "epsilon:v2"
			{displayName: "epsilon:v2", sourceID: "source-a"},
		}

		for _, spec := range testModels {
			catalogModel := &models.CatalogModelImpl{
				Attributes: &models.CatalogModelAttributes{
					Name:       new(spec.displayName),
					ExternalID: new(spec.displayName + "-ord-ext"),
				},
				Properties: &[]dbmodels.Properties{
					{Name: "source_id", StringValue: new(spec.sourceID)},
				},
			}

			_, err := repo.Save(catalogModel)
			require.NoError(t, err)
		}

		// Test NAME ordering ASC
		listOptions := models.CatalogModelListOptions{
			Pagination: dbmodels.Pagination{
				OrderBy:   new("NAME"),
				SortOrder: new("ASC"),
			},
		}
		result, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Extract our test model display names from results (repo returns stored namespaced names)
		displayNames := map[string]string{
			"source-z:alpha-model": "alpha-model",
			"source-z:beta-model":  "beta-model",
			"source-a:gamma-model": "gamma-model",
			"source-a:zebra-model": "zebra-model",
			"source-m:delta-model": "delta-model",
			// display name contains a colon: stored name is "source-a:epsilon:v2"
			"source-a:epsilon:v2": "epsilon:v2",
		}
		var foundNames []string
		for _, model := range result.Items {
			storedName := *model.GetAttributes().Name
			if displayName, ok := displayNames[storedName]; ok {
				foundNames = append(foundNames, displayName)
			}
		}

		// Verify we found all our test models
		require.GreaterOrEqual(t, len(foundNames), 6, "Should find all test models")

		// Verify ASC ordering by display name: alpha < beta < delta < epsilon:v2 < gamma < zebra
		// Without the fix, source-z models (alpha, beta) would appear after source-a models (gamma, zebra).
		// Also verifies multi-colon display names ("epsilon:v2") sort correctly.
		alphaIdx := findIndex(foundNames, "alpha-model")
		betaIdx := findIndex(foundNames, "beta-model")
		deltaIdx := findIndex(foundNames, "delta-model")
		epsilonIdx := findIndex(foundNames, "epsilon:v2")
		gammaIdx := findIndex(foundNames, "gamma-model")
		zebraIdx := findIndex(foundNames, "zebra-model")

		require.NotEqual(t, -1, alphaIdx, "alpha-model not found")
		require.NotEqual(t, -1, betaIdx, "beta-model not found")
		require.NotEqual(t, -1, deltaIdx, "delta-model not found")
		require.NotEqual(t, -1, epsilonIdx, "epsilon:v2 not found")
		require.NotEqual(t, -1, gammaIdx, "gamma-model not found")
		require.NotEqual(t, -1, zebraIdx, "zebra-model not found")

		assert.Less(t, alphaIdx, betaIdx, "alpha should come before beta in ASC")
		assert.Less(t, betaIdx, deltaIdx, "beta should come before delta in ASC")
		assert.Less(t, deltaIdx, epsilonIdx, "delta should come before epsilon:v2 in ASC")
		assert.Less(t, epsilonIdx, gammaIdx, "epsilon:v2 should come before gamma in ASC")
		assert.Less(t, gammaIdx, zebraIdx, "gamma should come before zebra in ASC")

		// Test NAME ordering DESC
		listOptions = models.CatalogModelListOptions{
			Pagination: dbmodels.Pagination{
				OrderBy:   new("NAME"),
				SortOrder: new("DESC"),
			},
		}
		result, err = repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Extract our test model names from DESC results
		foundNames = []string{}
		for _, model := range result.Items {
			storedName := *model.GetAttributes().Name
			if displayName, ok := displayNames[storedName]; ok {
				foundNames = append(foundNames, displayName)
			}
		}

		// Verify DESC ordering: zebra > gamma > epsilon:v2 > delta > beta > alpha
		alphaIdxDesc := findIndex(foundNames, "alpha-model")
		betaIdxDesc := findIndex(foundNames, "beta-model")
		deltaIdxDesc := findIndex(foundNames, "delta-model")
		epsilonIdxDesc := findIndex(foundNames, "epsilon:v2")
		gammaIdxDesc := findIndex(foundNames, "gamma-model")
		zebraIdxDesc := findIndex(foundNames, "zebra-model")

		assert.Less(t, zebraIdxDesc, gammaIdxDesc, "zebra should come before gamma in DESC")
		assert.Less(t, gammaIdxDesc, epsilonIdxDesc, "gamma should come before epsilon:v2 in DESC")
		assert.Less(t, epsilonIdxDesc, deltaIdxDesc, "epsilon:v2 should come before delta in DESC")
		assert.Less(t, deltaIdxDesc, betaIdxDesc, "delta should come before beta in DESC")
		assert.Less(t, betaIdxDesc, alphaIdxDesc, "beta should come before alpha in DESC")
	})

	t.Run("TestNameOrderingPagination", func(t *testing.T) {
		// Create models with sequential names for pagination testing
		testModels := []string{
			"page-test-model-01",
			"page-test-model-02",
			"page-test-model-03",
			"page-test-model-04",
			"page-test-model-05",
		}

		for _, name := range testModels {
			catalogModel := &models.CatalogModelImpl{
				Attributes: &models.CatalogModelAttributes{
					Name:       new(name),
					ExternalID: new(name + "-ext"),
				},
			}

			_, err := repo.Save(catalogModel)
			require.NoError(t, err)
		}

		// Test pagination with NAME ordering
		listOptions := models.CatalogModelListOptions{
			Pagination: dbmodels.Pagination{
				OrderBy:   new("NAME"),
				SortOrder: new("ASC"),
				PageSize:  new(int32(2)),
			},
		}

		// Collect all our test models across pages
		var allPaginatedModels []string
		var pageCount int
		currentToken := (*string)(nil)

		for {
			pageCount++
			if currentToken != nil {
				listOptions.Pagination.NextPageToken = currentToken
			}

			page, err := repo.List(listOptions)
			require.NoError(t, err)
			require.NotNil(t, page)
			assert.Equal(t, int32(2), page.PageSize)

			// Filter to only include our test models
			for _, model := range page.Items {
				name := *model.GetAttributes().Name
				if strings.HasPrefix(name, "page-test-model-") {
					allPaginatedModels = append(allPaginatedModels, name)
				}
			}

			// Stop if no more pages or we've collected all our test models
			if page.NextPageToken == "" || len(allPaginatedModels) >= 5 {
				if page.NextPageToken == "" {
					t.Logf("NAME pagination completed in %d pages", pageCount)
				}
				break
			}
			currentToken = &page.NextPageToken

			// Safety check to prevent infinite loop
			if pageCount > 10 {
				t.Fatal("Too many pages, might be an infinite loop")
			}
		}

		// Verify we collected all our test models
		assert.GreaterOrEqual(t, len(allPaginatedModels), 5, "Should have found all page-test models")

		// Verify ordering is maintained across pages
		expectedOrder := []string{
			"page-test-model-01",
			"page-test-model-02",
			"page-test-model-03",
			"page-test-model-04",
			"page-test-model-05",
		}

		// Verify our test models appear in correct order
		lastIndex := -1
		for _, expectedModel := range expectedOrder {
			foundIndex := findIndex(allPaginatedModels, expectedModel)
			assert.NotEqual(t, -1, foundIndex, "Should find model %s", expectedModel)
			if foundIndex != -1 {
				assert.Greater(t, foundIndex, lastIndex, "Model %s should appear after previous models", expectedModel)
				lastIndex = foundIndex
			}
		}
	})

	t.Run("TestNameOrderingMultiColonPagination", func(t *testing.T) {
		// Regression test: display names containing a colon (e.g. "llama:7b" stored as "source:llama:7b")
		// caused cursor-based pagination to break because SUBSTRING(name FROM STRPOS(name, ':') + 1)
		// returns everything after the first colon (e.g. "llama:7b"), while strings.Cut does the same,
		// keeping cursor values consistent with ORDER BY values.
		multiColonModels := []struct {
			displayName string
			sourceID    string
		}{
			{"mc-model:aa", "source-x"},
			{"mc-model:bb", "source-x"},
			{"mc-model:cc", "source-x"},
			{"mc-model:dd", "source-x"},
		}

		for _, spec := range multiColonModels {
			catalogModel := &models.CatalogModelImpl{
				Attributes: &models.CatalogModelAttributes{
					Name:       new(spec.displayName),
					ExternalID: new(spec.displayName + "-mc-ext"),
				},
				Properties: &[]dbmodels.Properties{
					{Name: "source_id", StringValue: new(spec.sourceID)},
				},
			}
			_, err := repo.Save(catalogModel)
			require.NoError(t, err)
		}

		// Paginate through all 4 models 2 at a time; verify ordering is stable across pages.
		listOptions := models.CatalogModelListOptions{
			Pagination: dbmodels.Pagination{
				OrderBy:   new("NAME"),
				SortOrder: new("ASC"),
				PageSize:  new(int32(2)),
			},
		}

		var allFound []string
		currentToken := (*string)(nil)
		for range 10 {
			if currentToken != nil {
				listOptions.Pagination.NextPageToken = currentToken
			}
			page, err := repo.List(listOptions)
			require.NoError(t, err)

			for _, model := range page.Items {
				name := *model.GetAttributes().Name
				if strings.HasPrefix(name, "source-x:mc-model:") {
					allFound = append(allFound, name)
				}
			}

			if page.NextPageToken == "" || len(allFound) >= 4 {
				break
			}
			currentToken = &page.NextPageToken
		}

		require.GreaterOrEqual(t, len(allFound), 4, "Should find all multi-colon models")

		// Verify ASC ordering is preserved across pages
		expectedOrder := []string{
			"source-x:mc-model:aa",
			"source-x:mc-model:bb",
			"source-x:mc-model:cc",
			"source-x:mc-model:dd",
		}
		lastIndex := -1
		for _, expected := range expectedOrder {
			idx := findIndex(allFound, expected)
			assert.NotEqual(t, -1, idx, "Should find %s", expected)
			if idx != -1 {
				assert.Greater(t, idx, lastIndex, "%s should appear after previous models in ASC order", expected)
				lastIndex = idx
			}
		}
	})

	t.Run("TestDeleteBySource", func(t *testing.T) {
		// Setup: Create models with different source IDs
		sourceID1 := "test_source_1"
		sourceID2 := "test_source_2"

		model1 := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name: new("model-source-1"),
			},
			Properties: &[]dbmodels.Properties{
				{
					Name:        "source_id",
					StringValue: &sourceID1,
				},
			},
		}

		model2 := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name: new("model-source-1-second"),
			},
			Properties: &[]dbmodels.Properties{
				{
					Name:        "source_id",
					StringValue: &sourceID1,
				},
			},
		}

		model3 := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name: new("model-source-2"),
			},
			Properties: &[]dbmodels.Properties{
				{
					Name:        "source_id",
					StringValue: &sourceID2,
				},
			},
		}

		// Save all models
		saved1, err := repo.Save(model1)
		require.NoError(t, err)
		saved2, err := repo.Save(model2)
		require.NoError(t, err)
		saved3, err := repo.Save(model3)
		require.NoError(t, err)

		// Delete by source_id
		err = repo.DeleteBySource(sourceID1)
		require.NoError(t, err)

		// Verify models from source1 are deleted
		_, err = repo.GetByID(*saved1.GetID())
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrCatalogModelNotFound))

		_, err = repo.GetByID(*saved2.GetID())
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrCatalogModelNotFound))

		// Verify model from source2 still exists
		retrieved, err := repo.GetByID(*saved3.GetID())
		require.NoError(t, err)
		// Repository returns stored name (namespaced: sourceId:modelName)
		assert.Equal(t, sourceID2+":model-source-2", *retrieved.GetAttributes().Name)
	})

	t.Run("TestDeleteByID", func(t *testing.T) {
		// Setup: Create a model
		model := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name: new("model-to-delete"),
			},
			Properties: &[]dbmodels.Properties{
				{
					Name:        "description",
					StringValue: new("Model that will be deleted"),
				},
			},
		}

		saved, err := repo.Save(model)
		require.NoError(t, err)

		// Delete by ID
		err = repo.DeleteByID(*saved.GetID())
		require.NoError(t, err)

		// Verify model is deleted
		_, err = repo.GetByID(*saved.GetID())
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrCatalogModelNotFound))
	})

	t.Run("TestDeleteBySourceNonExistent", func(t *testing.T) {
		// Test deleting by non-existent source - should not error
		err := repo.DeleteBySource("non-existent-source")
		require.NoError(t, err)
	})

	t.Run("TestDeleteByIDNonExistent", func(t *testing.T) {
		// Test deleting non-existent ID - should return NotFoundError
		err := repo.DeleteByID(999999)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrCatalogModelNotFound))
	})

	t.Run("TestGetDistinctSourceIDs", func(t *testing.T) {
		// Get initial count of source IDs
		initialSourceIDs, err := repo.GetDistinctSourceIDs()
		assert.NoError(t, err)
		initialCount := len(initialSourceIDs)

		// Create test data with different source_ids (use unique prefixes to avoid collision with other tests)
		testSourceID1 := "test-distinct-source-1"
		testSourceID2 := "test-distinct-source-2"

		model1 := createTestCatalogModelWithSourceID(t, testSourceID1)
		model2 := createTestCatalogModelWithSourceID(t, testSourceID2)
		model3 := createTestCatalogModelWithSourceID(t, testSourceID1) // duplicate

		_, err = repo.Save(model1)
		assert.NoError(t, err)
		_, err = repo.Save(model2)
		assert.NoError(t, err)
		_, err = repo.Save(model3)
		assert.NoError(t, err)

		// Test distinct source_ids - should have 2 new source IDs added
		sourceIDs, err := repo.GetDistinctSourceIDs()
		assert.NoError(t, err)
		assert.Len(t, sourceIDs, initialCount+2, "Should have exactly 2 new distinct source IDs")
		assert.Contains(t, sourceIDs, testSourceID1)
		assert.Contains(t, sourceIDs, testSourceID2)
	})

	t.Run("TestDeleteBySourceWithArtifactCleanup", func(t *testing.T) {
		sourceID := "test-source-with-artifacts"

		// Create a test model with source_id
		model := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name: new("model-with-artifacts"),
			},
			Properties: &[]dbmodels.Properties{
				{
					Name:        "source_id",
					StringValue: &sourceID,
				},
			},
		}

		savedModel, err := repo.Save(model)
		require.NoError(t, err)
		modelID := *savedModel.GetID()

		// Create artifacts and link them to the model
		artifactTypeID := getCatalogModelArtifactTypeID(t, sharedDB)
		executionTypeID := getOrCreateExecutionTypeID(t, sharedDB)

		// Create artifact 1 (will be orphaned - should be deleted)
		orphanedArtifact := createTestArtifact(t, sharedDB, artifactTypeID, "orphaned-artifact")
		createAttribution(t, sharedDB, modelID, orphanedArtifact.ID)

		// Create artifact 2 (will have Event - should be preserved)
		preservedArtifact := createTestArtifact(t, sharedDB, artifactTypeID, "preserved-artifact")
		createAttribution(t, sharedDB, modelID, preservedArtifact.ID)

		// Create an execution and event for artifact 2
		execution := createTestExecution(t, sharedDB, executionTypeID, "test-execution")
		createTestEvent(t, sharedDB, preservedArtifact.ID, execution.ID)

		// Verify artifacts exist before deletion
		var artifactCount int64
		err = sharedDB.Model(&schema.Artifact{}).Where("id IN (?)", []int32{orphanedArtifact.ID, preservedArtifact.ID}).Count(&artifactCount).Error
		require.NoError(t, err)
		assert.Equal(t, int64(2), artifactCount, "Both artifacts should exist before deletion")

		// Delete by source
		err = repo.DeleteBySource(sourceID)
		require.NoError(t, err)

		// Verify model is deleted
		_, err = repo.GetByID(modelID)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrCatalogModelNotFound))

		// Verify orphaned artifact is deleted
		var orphanedArtifactExists int64
		err = sharedDB.Model(&schema.Artifact{}).Where("id = ?", orphanedArtifact.ID).Count(&orphanedArtifactExists).Error
		require.NoError(t, err)
		assert.Equal(t, int64(0), orphanedArtifactExists, "Orphaned artifact should be deleted")

		// Verify artifact with Event is preserved
		var preservedArtifactExists int64
		err = sharedDB.Model(&schema.Artifact{}).Where("id = ?", preservedArtifact.ID).Count(&preservedArtifactExists).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), preservedArtifactExists, "Artifact with Event should be preserved")
	})

	t.Run("TestDeleteByIDWithArtifactCleanup", func(t *testing.T) {
		// Create a test model
		model := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name: new("model-for-id-deletion"),
			},
		}

		savedModel, err := repo.Save(model)
		require.NoError(t, err)
		modelID := *savedModel.GetID()

		// Create artifacts and link them to the model
		artifactTypeID := getCatalogModelArtifactTypeID(t, sharedDB)
		executionTypeID := getOrCreateExecutionTypeID(t, sharedDB)

		// Create artifact 1 (will be orphaned - should be deleted)
		orphanedArtifact := createTestArtifact(t, sharedDB, artifactTypeID, "orphaned-artifact-id")
		createAttribution(t, sharedDB, modelID, orphanedArtifact.ID)

		// Create artifact 2 (will have Event - should be preserved)
		preservedArtifact := createTestArtifact(t, sharedDB, artifactTypeID, "preserved-artifact-id")
		createAttribution(t, sharedDB, modelID, preservedArtifact.ID)

		// Create an execution and event for artifact 2
		execution := createTestExecution(t, sharedDB, executionTypeID, "test-execution-id")
		createTestEvent(t, sharedDB, preservedArtifact.ID, execution.ID)

		// Verify artifacts exist before deletion
		var artifactCount int64
		err = sharedDB.Model(&schema.Artifact{}).Where("id IN (?)", []int32{orphanedArtifact.ID, preservedArtifact.ID}).Count(&artifactCount).Error
		require.NoError(t, err)
		assert.Equal(t, int64(2), artifactCount, "Both artifacts should exist before deletion")

		// Delete by ID
		err = repo.DeleteByID(modelID)
		require.NoError(t, err)

		// Verify model is deleted
		_, err = repo.GetByID(modelID)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrCatalogModelNotFound))

		// Verify orphaned artifact is deleted
		var orphanedArtifactExists int64
		err = sharedDB.Model(&schema.Artifact{}).Where("id = ?", orphanedArtifact.ID).Count(&orphanedArtifactExists).Error
		require.NoError(t, err)
		assert.Equal(t, int64(0), orphanedArtifactExists, "Orphaned artifact should be deleted")

		// Verify artifact with Event is preserved
		var preservedArtifactExists int64
		err = sharedDB.Model(&schema.Artifact{}).Where("id = ?", preservedArtifact.ID).Count(&preservedArtifactExists).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), preservedArtifactExists, "Artifact with Event should be preserved")
	})

	t.Run("TestDeleteBySourceMultipleModelsWithArtifacts", func(t *testing.T) {
		sourceID := "test-source-multiple-models"

		// Create two models with the same source_id
		model1 := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name: new("model-1-multi-source"),
			},
			Properties: &[]dbmodels.Properties{
				{
					Name:        "source_id",
					StringValue: &sourceID,
				},
			},
		}

		model2 := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name: new("model-2-multi-source"),
			},
			Properties: &[]dbmodels.Properties{
				{
					Name:        "source_id",
					StringValue: &sourceID,
				},
			},
		}

		savedModel1, err := repo.Save(model1)
		require.NoError(t, err)
		savedModel2, err := repo.Save(model2)
		require.NoError(t, err)

		artifactTypeID := getCatalogModelArtifactTypeID(t, sharedDB)
		executionTypeID := getOrCreateExecutionTypeID(t, sharedDB)

		// Create artifacts for model 1
		orphanedArtifact1 := createTestArtifact(t, sharedDB, artifactTypeID, "orphaned-artifact-model1")
		createAttribution(t, sharedDB, *savedModel1.GetID(), orphanedArtifact1.ID)

		// Create artifacts for model 2 with Event (should be preserved)
		preservedArtifact2 := createTestArtifact(t, sharedDB, artifactTypeID, "preserved-artifact-model2")
		createAttribution(t, sharedDB, *savedModel2.GetID(), preservedArtifact2.ID)
		execution := createTestExecution(t, sharedDB, executionTypeID, "test-execution-multi")
		createTestEvent(t, sharedDB, preservedArtifact2.ID, execution.ID)

		// Delete by source - should delete both models
		err = repo.DeleteBySource(sourceID)
		require.NoError(t, err)

		// Verify both models are deleted
		_, err = repo.GetByID(*savedModel1.GetID())
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrCatalogModelNotFound))

		_, err = repo.GetByID(*savedModel2.GetID())
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrCatalogModelNotFound))

		// Verify orphaned artifact is deleted
		var orphanedExists int64
		err = sharedDB.Model(&schema.Artifact{}).Where("id = ?", orphanedArtifact1.ID).Count(&orphanedExists).Error
		require.NoError(t, err)
		assert.Equal(t, int64(0), orphanedExists, "Orphaned artifact should be deleted")

		// Verify artifact with Event is preserved
		var preservedExists int64
		err = sharedDB.Model(&schema.Artifact{}).Where("id = ?", preservedArtifact2.ID).Count(&preservedExists).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), preservedExists, "Artifact with Event should be preserved")
	})

	t.Run("TestDeleteByIDNonExistentWithArtifacts", func(t *testing.T) {
		// Test deleting non-existent model ID - should return NotFoundError and not affect any artifacts
		err := repo.DeleteByID(999999)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrCatalogModelNotFound))
	})

	t.Run("TestQuerySearchMatchesValidatedTasks", func(t *testing.T) {
		uniqueTerm := "xyzspecialvalidated"

		modelWithValidatedTask := &models.CatalogModelImpl{
			TypeID: new(typeID),
			Attributes: &models.CatalogModelAttributes{
				Name:       new("query-vtask-model"),
				ExternalID: new("query-vtask-ext"),
			},
			Properties: &[]dbmodels.Properties{
				{Name: "source_id", StringValue: new("query-vtask-source")},
				{Name: "tasks", StringValue: new(`["text-generation"]`)},
				{Name: "validated_tasks", StringValue: new(fmt.Sprintf(`["%s"]`, uniqueTerm))},
			},
		}

		_, err := repo.Save(modelWithValidatedTask)
		require.NoError(t, err)

		query := uniqueTerm
		listOptions := models.CatalogModelListOptions{
			Query: &query,
		}
		result, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, len(result.Items))
		assert.Equal(t, "query-vtask-source:query-vtask-model", *result.Items[0].GetAttributes().Name)
	})
}

func createTestCatalogModelWithSourceID(t *testing.T, sourceID string) models.CatalogModel {
	model := &models.CatalogModelImpl{
		Attributes: &models.CatalogModelAttributes{
			Name: new(fmt.Sprintf("test-model-%s", sourceID)),
		},
	}

	// Add source_id as a property
	properties := []dbmodels.Properties{
		{
			Name:        "source_id",
			StringValue: &sourceID,
		},
	}
	model.Properties = &properties

	return model
}

func TestCatalogModelRepository_TimestampPreservation(t *testing.T) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testDatastoreSpec())
	defer cleanup()

	typeID := getCatalogModelTypeID(t, sharedDB)
	repo := NewCatalogModelRepository(sharedDB, typeID)

	t.Run("Preserve historical timestamps from YAML catalog", func(t *testing.T) {
		// Simulate loading a model from YAML with historical timestamps
		historicalCreateTime := int64(1739776988000) // From YAML example
		historicalUpdateTime := int64(1746720264000) // From YAML example

		catalogModel := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:                     new("yaml-loaded-model"),
				ExternalID:               new("yaml-model-123"),
				CreateTimeSinceEpoch:     &historicalCreateTime,
				LastUpdateTimeSinceEpoch: &historicalUpdateTime,
			},
			Properties: &[]dbmodels.Properties{
				{
					Name:        "description",
					StringValue: new("Model loaded from YAML"),
				},
			},
		}

		// Save the model - timestamps should be preserved
		saved, err := repo.Save(catalogModel)
		require.NoError(t, err)
		require.NotNil(t, saved)
		require.NotNil(t, saved.GetID())

		// Verify historical timestamps were preserved
		savedAttrs := saved.GetAttributes()
		require.NotNil(t, savedAttrs.CreateTimeSinceEpoch)
		require.NotNil(t, savedAttrs.LastUpdateTimeSinceEpoch)
		assert.Equal(t, historicalCreateTime, *savedAttrs.CreateTimeSinceEpoch,
			"CreateTimeSinceEpoch should be preserved from YAML")
		assert.Equal(t, historicalUpdateTime, *savedAttrs.LastUpdateTimeSinceEpoch,
			"LastUpdateTimeSinceEpoch should be preserved from YAML")

		// Reload from database to verify persistence
		retrieved, err := repo.GetByID(*saved.GetID())
		require.NoError(t, err)
		retrievedAttrs := retrieved.GetAttributes()
		assert.Equal(t, historicalCreateTime, *retrievedAttrs.CreateTimeSinceEpoch,
			"CreateTimeSinceEpoch should persist in database")
		assert.Equal(t, historicalUpdateTime, *retrievedAttrs.LastUpdateTimeSinceEpoch,
			"LastUpdateTimeSinceEpoch should persist in database")
	})

	t.Run("Auto-generate timestamps for API-created models", func(t *testing.T) {
		// Model created via API without timestamps
		catalogModel := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       new("api-created-model"),
				ExternalID: new("api-model-456"),
				// No timestamps set - should be auto-generated
			},
			Properties: &[]dbmodels.Properties{
				{
					Name:        "description",
					StringValue: new("Model created via API"),
				},
			},
		}

		// Save the model - timestamps should be auto-generated
		saved, err := repo.Save(catalogModel)
		require.NoError(t, err)
		require.NotNil(t, saved)

		// Verify timestamps were auto-generated (non-zero)
		savedAttrs := saved.GetAttributes()
		require.NotNil(t, savedAttrs.CreateTimeSinceEpoch)
		require.NotNil(t, savedAttrs.LastUpdateTimeSinceEpoch)
		assert.Greater(t, *savedAttrs.CreateTimeSinceEpoch, int64(0),
			"CreateTimeSinceEpoch should be auto-generated")
		assert.Greater(t, *savedAttrs.LastUpdateTimeSinceEpoch, int64(0),
			"LastUpdateTimeSinceEpoch should be auto-generated")
	})

	t.Run("Update existing model from YAML preserves CreateTime", func(t *testing.T) {
		// First save: Create model with historical timestamps
		historicalCreateTime := int64(1739776988000)
		firstUpdateTime := int64(1746720264000)

		catalogModel := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:                     new("yaml-updated-model"),
				ExternalID:               new("yaml-updated-123"),
				CreateTimeSinceEpoch:     &historicalCreateTime,
				LastUpdateTimeSinceEpoch: &firstUpdateTime,
			},
		}

		saved, err := repo.Save(catalogModel)
		require.NoError(t, err)
		savedID := saved.GetID()

		// Second save: Update the model with new LastUpdateTime (simulating catalog reload)
		newerUpdateTime := int64(1750000000000)
		catalogModel.ID = savedID
		catalogModel.GetAttributes().LastUpdateTimeSinceEpoch = &newerUpdateTime

		updated, err := repo.Save(catalogModel)
		require.NoError(t, err)

		// Verify CreateTime is preserved but LastUpdateTime is updated
		updatedAttrs := updated.GetAttributes()
		assert.Equal(t, historicalCreateTime, *updatedAttrs.CreateTimeSinceEpoch,
			"CreateTimeSinceEpoch should be preserved on update")
		assert.Equal(t, newerUpdateTime, *updatedAttrs.LastUpdateTimeSinceEpoch,
			"LastUpdateTimeSinceEpoch should be updated")
	})
}

// TestSortingFilteringInconsistency reproduces the bug where sorting and filtering
// by artifact properties use different artifact sets, leading to inconsistent results.
//
// Bug: When both sorting and filtering by artifact properties, the sorting logic
// considers ALL artifacts while filtering only considers artifacts passing the filter.
//
// Test scenario:
// - Model A: artifacts with accuracy=[0.95 deprecated, 0.65 active]
// - Model B: artifacts with accuracy=[0.75 active]
// - Filter: artifacts.status.string_value='active'
// - Sort: artifacts.accuracy.double_value DESC
//
// Expected: Model B (0.75) before Model A (0.65) - considering only active artifacts
// Actual: Model A before Model B - sorting uses 0.95 value from deprecated artifact
func TestSortingFilteringInconsistency(t *testing.T) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testDatastoreSpec())
	defer cleanup()
	defer testutils.CleanupPostgresTestData(t, sharedDB)

	// Clean up any existing test data to ensure test isolation
	testutils.CleanupPostgresTestData(t, sharedDB)

	// Setup repositories
	catalogModelTypeID := getCatalogModelTypeID(t, sharedDB)
	modelRepo := NewCatalogModelRepository(sharedDB, catalogModelTypeID)

	modelArtifactTypeID := getCatalogModelArtifactTypeID(t, sharedDB)
	modelArtifactRepo := NewCatalogModelArtifactRepository(sharedDB, modelArtifactTypeID)

	// Test case: Create Model A with high accuracy deprecated artifact + low accuracy active artifact
	t.Run("TestBugReproduction", func(t *testing.T) {
		// Create Model A
		modelA := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       new("model-a-bug-test"),
				ExternalID: new("model-a-ext-123"),
			},
		}
		savedModelA, err := modelRepo.Save(modelA)
		require.NoError(t, err)

		// Create Model B
		modelB := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       new("model-b-bug-test"),
				ExternalID: new("model-b-ext-456"),
			},
		}
		savedModelB, err := modelRepo.Save(modelB)
		require.NoError(t, err)

		// Create artifacts for Model A:
		// 1. High accuracy artifact (0.95) with status="deprecated"
		artifactA1 := &models.CatalogModelArtifactImpl{
			Attributes: &models.CatalogModelArtifactAttributes{
				Name:       new("model-a-deprecated-artifact"),
				ExternalID: new("model-a-deprecated-ext"),
			},
			CustomProperties: &[]dbmodels.Properties{
				{
					Name:        "accuracy",
					DoubleValue: new(0.95),
				},
				{
					Name:        "status",
					StringValue: new("deprecated"),
				},
			},
		}
		_, err = modelArtifactRepo.Save(artifactA1, savedModelA.GetID())
		require.NoError(t, err)

		// 2. Low accuracy artifact (0.65) with status="active"
		artifactA2 := &models.CatalogModelArtifactImpl{
			Attributes: &models.CatalogModelArtifactAttributes{
				Name:       new("model-a-active-artifact"),
				ExternalID: new("model-a-active-ext"),
			},
			CustomProperties: &[]dbmodels.Properties{
				{
					Name:        "accuracy",
					DoubleValue: new(0.65),
				},
				{
					Name:        "status",
					StringValue: new("active"),
				},
			},
		}
		_, err = modelArtifactRepo.Save(artifactA2, savedModelA.GetID())
		require.NoError(t, err)

		// Create artifacts for Model B:
		// 1. Medium accuracy artifact (0.75) with status="active"
		artifactB1 := &models.CatalogModelArtifactImpl{
			Attributes: &models.CatalogModelArtifactAttributes{
				Name:       new("model-b-active-artifact"),
				ExternalID: new("model-b-active-ext"),
			},
			CustomProperties: &[]dbmodels.Properties{
				{
					Name:        "accuracy",
					DoubleValue: new(0.75),
				},
				{
					Name:        "status",
					StringValue: new("active"),
				},
			},
		}
		_, err = modelArtifactRepo.Save(artifactB1, savedModelB.GetID())
		require.NoError(t, err)

		// Test the bug: Filter by active status + sort by accuracy DESC
		listOptions := models.CatalogModelListOptions{
			Pagination: dbmodels.Pagination{
				FilterQuery: new("artifacts.status.string_value='active'"),
				OrderBy:     new("artifacts.accuracy.double_value"),
				SortOrder:   new("DESC"),
			},
		}

		result, err := modelRepo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.GreaterOrEqual(t, len(result.Items), 2, "Should find both models")

		// Extract our test models from results
		var foundModels []models.CatalogModel
		for _, model := range result.Items {
			name := *model.GetAttributes().Name
			if name == "model-a-bug-test" || name == "model-b-bug-test" {
				foundModels = append(foundModels, model)
			}
		}
		require.Len(t, foundModels, 2, "Should find both test models")

		// Find positions of Model A and Model B in results
		modelAIndex := -1
		modelBIndex := -1
		for i, model := range foundModels {
			name := *model.GetAttributes().Name
			if name == "model-a-bug-test" {
				modelAIndex = i
			} else if name == "model-b-bug-test" {
				modelBIndex = i
			}
		}

		require.NotEqual(t, -1, modelAIndex, "Should find Model A in filtered results")
		require.NotEqual(t, -1, modelBIndex, "Should find Model B in filtered results")

		// **EXPECT CORRECT BEHAVIOR**:
		// Model B (accuracy 0.75 active) should come before Model A (accuracy 0.65 active)
		// When filtering by status='active', sorting should only consider active artifacts
		//
		// This test will FAIL until the bug is fixed, then PASS once correct behavior is implemented
		t.Logf("Model A index: %d, Model B index: %d", modelAIndex, modelBIndex)

		// Assert the CORRECT expected behavior - this will fail until bug is fixed
		assert.Less(t, modelBIndex, modelAIndex,
			"Model B (active artifact accuracy=0.75) should be sorted before Model A (active artifact accuracy=0.65). "+
				"Current bug: sorting uses Model A's deprecated artifact (accuracy=0.95) instead of only considering active artifacts that pass the filter. "+
				"Fix: sorting and filtering must operate on the same artifact dataset.")
	})
}

// TestSortingFilteringEdgeCases tests edge cases for the sorting/filtering consistency fix
func TestSortingFilteringEdgeCases(t *testing.T) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testDatastoreSpec())
	defer cleanup()
	defer testutils.CleanupPostgresTestData(t, sharedDB)

	// Clean up any existing test data to ensure test isolation
	testutils.CleanupPostgresTestData(t, sharedDB)

	// Setup repositories
	catalogModelTypeID := getCatalogModelTypeID(t, sharedDB)
	modelRepo := NewCatalogModelRepository(sharedDB, catalogModelTypeID)

	modelArtifactTypeID := getCatalogModelArtifactTypeID(t, sharedDB)
	modelArtifactRepo := NewCatalogModelArtifactRepository(sharedDB, modelArtifactTypeID)

	// Create test models with various artifact scenarios
	modelA := &models.CatalogModelImpl{
		Attributes: &models.CatalogModelAttributes{
			Name: new("Edge Test Model A"),
		},
		Properties: &[]dbmodels.Properties{
			{Name: "provider", StringValue: new("edge-provider-a")},
		},
	}

	modelB := &models.CatalogModelImpl{
		Attributes: &models.CatalogModelAttributes{
			Name: new("Edge Test Model B"),
		},
		Properties: &[]dbmodels.Properties{
			{Name: "provider", StringValue: new("edge-provider-b")},
		},
	}

	// Save models
	savedModelA, err := modelRepo.Save(modelA)
	require.NoError(t, err)
	savedModelB, err := modelRepo.Save(modelB)
	require.NoError(t, err)

	// Create artifacts for Model A
	artifactA1 := &models.CatalogModelArtifactImpl{
		Attributes: &models.CatalogModelArtifactAttributes{
			Name: new("edge-artifact-a1"),
		},
		CustomProperties: &[]dbmodels.Properties{
			{Name: "accuracy", DoubleValue: new(0.95)},
			{Name: "status", StringValue: new("deprecated")},
		},
	}

	artifactA2 := &models.CatalogModelArtifactImpl{
		Attributes: &models.CatalogModelArtifactAttributes{
			Name: new("edge-artifact-a2"),
		},
		CustomProperties: &[]dbmodels.Properties{
			{Name: "accuracy", DoubleValue: new(0.65)},
			{Name: "status", StringValue: new("active")},
		},
	}

	// Create artifacts for Model B
	artifactB1 := &models.CatalogModelArtifactImpl{
		Attributes: &models.CatalogModelArtifactAttributes{
			Name: new("edge-artifact-b1"),
		},
		CustomProperties: &[]dbmodels.Properties{
			{Name: "accuracy", DoubleValue: new(0.75)},
			{Name: "status", StringValue: new("active")},
		},
	}

	// Save artifacts with parent relationships
	_, err = modelArtifactRepo.Save(artifactA1, savedModelA.GetID())
	require.NoError(t, err)
	_, err = modelArtifactRepo.Save(artifactA2, savedModelA.GetID())
	require.NoError(t, err)
	_, err = modelArtifactRepo.Save(artifactB1, savedModelB.GetID())
	require.NoError(t, err)

	t.Run("EdgeCase_NoArtifactsMatchFilter", func(t *testing.T) {
		// Test filtering for artifacts that don't exist
		listOptions := models.CatalogModelListOptions{
			Pagination: dbmodels.Pagination{
				FilterQuery: new("artifacts.nonexistent.string_value='missing'"),
				OrderBy:     new("artifacts.accuracy.double_value"),
				SortOrder:   new("DESC"),
			},
		}

		result, err := modelRepo.List(listOptions)
		assert.NoError(t, err, "Should handle filters with no matches gracefully")
		assert.Equal(t, int32(0), result.Size, "Should return empty results when no artifacts match filter")
	})

	t.Run("EdgeCase_MultipleFilterConditions", func(t *testing.T) {
		// Test with multiple AND filter conditions
		listOptions := models.CatalogModelListOptions{
			Pagination: dbmodels.Pagination{
				FilterQuery: new("artifacts.status.string_value='active' AND artifacts.accuracy.double_value>0.6"),
				OrderBy:     new("artifacts.accuracy.double_value"),
				SortOrder:   new("DESC"),
			},
		}

		result, err := modelRepo.List(listOptions)
		assert.NoError(t, err, "Should handle multiple filter conditions")

		// Should find Model B (0.75 active > 0.6) and Model A (0.65 active > 0.6)
		// Model B (0.75) should come first in DESC order
		assert.Equal(t, int32(2), result.Size)

		// Find model positions
		modelAIndex, modelBIndex := -1, -1
		for i, model := range result.Items {
			if *model.GetID() == *savedModelA.GetID() {
				modelAIndex = i
			} else if *model.GetID() == *savedModelB.GetID() {
				modelBIndex = i
			}
		}

		assert.NotEqual(t, -1, modelAIndex, "Should find Model A")
		assert.NotEqual(t, -1, modelBIndex, "Should find Model B")
		assert.Less(t, modelBIndex, modelAIndex, "Model B (0.75) should be sorted before Model A (0.65)")
	})

	t.Run("EdgeCase_FilterWithNoResults", func(t *testing.T) {
		// Test filter that matches no artifacts
		listOptions := models.CatalogModelListOptions{
			Pagination: dbmodels.Pagination{
				FilterQuery: new("artifacts.accuracy.double_value>1.0"), // No accuracy > 1.0
				OrderBy:     new("artifacts.accuracy.double_value"),
				SortOrder:   new("DESC"),
			},
		}

		result, err := modelRepo.List(listOptions)
		assert.NoError(t, err, "Should handle filters with no results")
		assert.Equal(t, int32(0), result.Size, "Should return empty results")
	})

	t.Run("EdgeCase_OnlyOneModelMatches", func(t *testing.T) {
		// Test filter that matches only one model
		listOptions := models.CatalogModelListOptions{
			Pagination: dbmodels.Pagination{
				FilterQuery: new("artifacts.status.string_value='deprecated'"),
				OrderBy:     new("artifacts.accuracy.double_value"),
				SortOrder:   new("DESC"),
			},
		}

		result, err := modelRepo.List(listOptions)
		assert.NoError(t, err, "Should handle single result")

		// Should find only Model A's deprecated artifact (0.95)
		assert.Equal(t, int32(1), result.Size)
		assert.Equal(t, savedModelA.GetID(), result.Items[0].GetID())
	})

	t.Run("EdgeCase_StringPropertySorting", func(t *testing.T) {
		// Test string property sorting with numeric filter
		listOptions := models.CatalogModelListOptions{
			Pagination: dbmodels.Pagination{
				FilterQuery: new("artifacts.accuracy.double_value>0.6"),
				OrderBy:     new("artifacts.status.string_value"),
				SortOrder:   new("ASC"), // Should sort: active, active, deprecated
			},
		}

		result, err := modelRepo.List(listOptions)
		assert.NoError(t, err, "Should handle string property sorting with numeric filter")

		// Should find Model A (accuracy 0.95 deprecated, 0.65 active) and Model B (0.75 active)
		// Both models have artifacts with accuracy > 0.6, so both should be included
		assert.Equal(t, int32(2), result.Size)
	})
}

func TestCatalogModelRepository_GetTypeID(t *testing.T) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testDatastoreSpec())
	defer cleanup()

	typeID := getCatalogModelTypeID(t, sharedDB)
	repo := NewCatalogModelRepository(sharedDB, typeID)

	assert.Equal(t, typeID, repo.GetTypeID())
}

// TestListDoublePropertyOrderingPagination walks every page of a listing
// ordered by a double custom property whose values need the full float64
// precision, and checks each model is returned exactly once and the walk
// ends. The cursor used to round such values, which repeated or skipped the
// row at the page boundary and, in descending order, never emptied the token.
func TestListDoublePropertyOrderingPagination(t *testing.T) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testDatastoreSpec())
	defer cleanup()
	defer testutils.CleanupPostgresTestData(t, sharedDB)

	modelRepo := NewCatalogModelRepository(sharedDB, getCatalogModelTypeID(t, sharedDB))
	artifactRepo := NewCatalogModelArtifactRepository(sharedDB, getCatalogModelArtifactTypeID(t, sharedDB))

	sharedMean := (0.7 + 0.8 + 0.9) / 3 // 0.7999999999999999

	tests := []struct {
		name      string
		values    []float64
		sortOrder string
		pageSize  int32
	}{
		{"full precision ascending", []float64{0.8523489932885906, 0.8523489932885907, 0.7345678901234567, 0.9123456789012345}, "ASC", 2},
		{"full precision descending", []float64{0.8523489932885906, 0.8523489932885907, 0.7345678901234567, 0.9123456789012345}, "DESC", 2},
		{"shared computed value ascending", []float64{sharedMean, sharedMean, sharedMean}, "ASC", 1},
		{"shared computed value descending", []float64{sharedMean, sharedMean, sharedMean}, "DESC", 1},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutils.CleanupPostgresTestData(t, sharedDB)

			seen := make(map[string]int, len(tt.values))
			for j, v := range tt.values {
				v := v
				name := fmt.Sprintf("double-order-%d-%d", i, j)
				saved, err := modelRepo.Save(&models.CatalogModelImpl{
					Attributes: &models.CatalogModelAttributes{Name: new(name), ExternalID: new(name)},
				})
				require.NoError(t, err)

				_, err = artifactRepo.Save(&models.CatalogModelArtifactImpl{
					Attributes:       &models.CatalogModelArtifactAttributes{Name: new(name + "-artifact"), ExternalID: new(name + "-artifact")},
					CustomProperties: &[]dbmodels.Properties{{Name: "accuracy", DoubleValue: &v}},
				}, saved.GetID())
				require.NoError(t, err)
				seen[name] = 0
			}

			token := ""
			for page := 0; ; page++ {
				require.Less(t, page, len(tt.values)+1, "pagination did not end after %d pages", page)

				listOptions := models.CatalogModelListOptions{
					Pagination: dbmodels.Pagination{
						PageSize:  new(tt.pageSize),
						OrderBy:   new("artifacts.accuracy.double_value"),
						SortOrder: new(tt.sortOrder),
					},
				}
				if token != "" {
					listOptions.NextPageToken = new(token)
				}

				result, err := modelRepo.List(listOptions)
				require.NoError(t, err)
				for _, item := range result.Items {
					seen[*item.GetAttributes().Name]++
				}

				if result.NextPageToken == "" {
					break
				}
				token = result.NextPageToken
			}

			for name, count := range seen {
				assert.Equal(t, 1, count, "%s should be returned exactly once", name)
			}
		})
	}
}
