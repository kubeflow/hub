package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	"github.com/kubeflow/hub/internal/platform/apiutils"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/kubeflow/hub/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogMetricsArtifactRepository(t *testing.T) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testDatastoreSpec())
	defer cleanup()

	// Get the CatalogMetricsArtifact type ID
	typeID := getCatalogMetricsArtifactTypeID(t, sharedDB)
	repo := NewCatalogMetricsArtifactRepository(sharedDB, typeID)

	// Also get CatalogModel type ID for creating parent entities
	catalogModelTypeID := getCatalogModelTypeID(t, sharedDB)
	catalogModelRepo := NewCatalogModelRepository(sharedDB, catalogModelTypeID)

	t.Run("TestSave", func(t *testing.T) {
		// First create a catalog model for attribution
		catalogModel := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("test-catalog-model-for-metrics"),
				ExternalID: apiutils.Of("catalog-model-metrics-ext-123"),
			},
		}
		savedCatalogModel, err := catalogModelRepo.Save(catalogModel)
		require.NoError(t, err)

		// Test creating a new catalog metrics artifact
		catalogMetricsArtifact := &models.CatalogMetricsArtifactImpl{
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:        apiutils.Of("test-catalog-metrics-artifact"),
				ExternalID:  apiutils.Of("catalog-metrics-ext-123"),
				MetricsType: models.MetricsTypeAccuracy,
			},
			Properties: &[]dbmodels.Properties{
				{
					Name:        "description",
					StringValue: apiutils.Of("Test catalog metrics artifact description"),
				},
			},
			CustomProperties: &[]dbmodels.Properties{
				{
					Name:        "custom-metrics-prop",
					StringValue: apiutils.Of("custom-metrics-value"),
				},
			},
		}

		saved, err := repo.Save(catalogMetricsArtifact, savedCatalogModel.GetID())
		require.NoError(t, err)
		require.NotNil(t, saved)
		require.NotNil(t, saved.GetID())
		assert.Equal(t, "test-catalog-metrics-artifact", *saved.GetAttributes().Name)
		assert.Equal(t, "catalog-metrics-ext-123", *saved.GetAttributes().ExternalID)
		assert.Equal(t, models.MetricsTypeAccuracy, saved.GetAttributes().MetricsType)

		// Test updating the same catalog metrics artifact
		catalogMetricsArtifact.ID = saved.GetID()
		catalogMetricsArtifact.GetAttributes().Name = apiutils.Of("updated-catalog-metrics-artifact")
		catalogMetricsArtifact.GetAttributes().MetricsType = models.MetricsTypePerformance
		// Preserve CreateTimeSinceEpoch from the saved entity
		catalogMetricsArtifact.GetAttributes().CreateTimeSinceEpoch = saved.GetAttributes().CreateTimeSinceEpoch

		updated, err := repo.Save(catalogMetricsArtifact, savedCatalogModel.GetID())
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, *saved.GetID(), *updated.GetID())
		assert.Equal(t, "updated-catalog-metrics-artifact", *updated.GetAttributes().Name)
		assert.Equal(t, models.MetricsTypePerformance, updated.GetAttributes().MetricsType)
	})

	t.Run("TestGetByID", func(t *testing.T) {
		// First create a catalog model
		catalogModel := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("test-catalog-model-for-getbyid-metrics"),
				ExternalID: apiutils.Of("catalog-model-getbyid-metrics-ext"),
			},
		}
		savedCatalogModel, err := catalogModelRepo.Save(catalogModel)
		require.NoError(t, err)

		// Create a catalog metrics artifact to retrieve
		catalogMetricsArtifact := &models.CatalogMetricsArtifactImpl{
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:        apiutils.Of("get-test-catalog-metrics-artifact"),
				ExternalID:  apiutils.Of("get-catalog-metrics-ext-123"),
				MetricsType: models.MetricsTypeAccuracy,
			},
		}

		saved, err := repo.Save(catalogMetricsArtifact, savedCatalogModel.GetID())
		require.NoError(t, err)
		require.NotNil(t, saved.GetID())

		// Test retrieving by ID
		retrieved, err := repo.GetByID(*saved.GetID())
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, *saved.GetID(), *retrieved.GetID())
		assert.Equal(t, "get-test-catalog-metrics-artifact", *retrieved.GetAttributes().Name)
		assert.Equal(t, "get-catalog-metrics-ext-123", *retrieved.GetAttributes().ExternalID)
		assert.Equal(t, models.MetricsTypeAccuracy, retrieved.GetAttributes().MetricsType)

		// Test retrieving non-existent ID
		_, err = repo.GetByID(99999)
		assert.ErrorIs(t, err, ErrCatalogMetricsArtifactNotFound)
	})

	t.Run("TestList", func(t *testing.T) {
		// Create a catalog model for the artifacts
		catalogModel := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("test-catalog-model-for-list-metrics"),
				ExternalID: apiutils.Of("catalog-model-list-metrics-ext"),
			},
		}
		savedCatalogModel, err := catalogModelRepo.Save(catalogModel)
		require.NoError(t, err)

		// Create multiple catalog metrics artifacts for listing
		testArtifacts := []*models.CatalogMetricsArtifactImpl{
			{
				Attributes: &models.CatalogMetricsArtifactAttributes{
					Name:        apiutils.Of("list-catalog-metrics-artifact-1"),
					ExternalID:  apiutils.Of("list-catalog-metrics-ext-1"),
					MetricsType: models.MetricsTypeAccuracy,
				},
			},
			{
				Attributes: &models.CatalogMetricsArtifactAttributes{
					Name:        apiutils.Of("list-catalog-metrics-artifact-2"),
					ExternalID:  apiutils.Of("list-catalog-metrics-ext-2"),
					MetricsType: models.MetricsTypePerformance,
				},
			},
			{
				Attributes: &models.CatalogMetricsArtifactAttributes{
					Name:        apiutils.Of("list-catalog-metrics-artifact-3"),
					ExternalID:  apiutils.Of("list-catalog-metrics-ext-3"),
					MetricsType: models.MetricsTypePerformance,
				},
			},
		}

		// Save all test artifacts
		var savedArtifacts []models.CatalogMetricsArtifact
		for _, artifact := range testArtifacts {
			saved, err := repo.Save(artifact, savedCatalogModel.GetID())
			require.NoError(t, err)
			savedArtifacts = append(savedArtifacts, saved)
		}

		// Test listing all artifacts
		listOptions := models.CatalogMetricsArtifactListOptions{}
		result, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.GreaterOrEqual(t, len(result.Items), 3) // At least our 3 test artifacts

		// Test filtering by name
		nameFilter := "list-catalog-metrics-artifact-1"
		listOptions = models.CatalogMetricsArtifactListOptions{
			Name: &nameFilter,
		}
		result, err = repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		if len(result.Items) > 0 {
			assert.Equal(t, 1, len(result.Items))
			assert.Equal(t, "list-catalog-metrics-artifact-1", *result.Items[0].GetAttributes().Name)
		}

		// Test filtering by external ID
		externalIDFilter := "list-catalog-metrics-ext-2"
		listOptions = models.CatalogMetricsArtifactListOptions{
			ExternalID: &externalIDFilter,
		}
		result, err = repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		if len(result.Items) > 0 {
			assert.Equal(t, 1, len(result.Items))
			assert.Equal(t, "list-catalog-metrics-ext-2", *result.Items[0].GetAttributes().ExternalID)
		}

		// Test filtering by parent resource ID (catalog model)
		listOptions = models.CatalogMetricsArtifactListOptions{
			ParentResourceID: savedCatalogModel.GetID(),
		}
		result, err = repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.GreaterOrEqual(t, len(result.Items), 3) // Should find our 3 test artifacts
	})

	t.Run("TestListWithPropertiesAndCustomProperties", func(t *testing.T) {
		// Create a catalog model
		catalogModel := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("test-catalog-model-for-props-metrics"),
				ExternalID: apiutils.Of("catalog-model-props-metrics-ext"),
			},
		}
		savedCatalogModel, err := catalogModelRepo.Save(catalogModel)
		require.NoError(t, err)

		// Create a catalog metrics artifact with both properties and custom properties
		catalogMetricsArtifact := &models.CatalogMetricsArtifactImpl{
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:        apiutils.Of("props-test-catalog-metrics-artifact"),
				ExternalID:  apiutils.Of("props-catalog-metrics-ext-123"),
				MetricsType: models.MetricsTypeAccuracy,
			},
			Properties: &[]dbmodels.Properties{
				{
					Name:        "version",
					StringValue: apiutils.Of("1.0.0"),
				},
				{
					Name:        "value",
					DoubleValue: apiutils.Of(0.95),
				},
			},
			CustomProperties: &[]dbmodels.Properties{
				{
					Name:        "team",
					StringValue: apiutils.Of("catalog-metrics-team"),
				},
				{
					Name:      "is_validated",
					BoolValue: apiutils.Of(true),
				},
			},
		}

		saved, err := repo.Save(catalogMetricsArtifact, savedCatalogModel.GetID())
		require.NoError(t, err)
		require.NotNil(t, saved)

		// Retrieve and verify properties
		retrieved, err := repo.GetByID(*saved.GetID())
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		// Check that metricsType is properly set
		assert.Equal(t, models.MetricsTypeAccuracy, retrieved.GetAttributes().MetricsType)

		// Check regular properties
		require.NotNil(t, retrieved.GetProperties())
		assert.Len(t, *retrieved.GetProperties(), 2)

		// Check custom properties
		require.NotNil(t, retrieved.GetCustomProperties())
		assert.Len(t, *retrieved.GetCustomProperties(), 2)

		// Verify specific properties exist
		properties := *retrieved.GetProperties()
		var foundVersion, foundValue bool
		for _, prop := range properties {
			switch prop.Name {
			case "version":
				foundVersion = true
				assert.Equal(t, "1.0.0", *prop.StringValue)
			case "value":
				foundValue = true
				assert.Equal(t, 0.95, *prop.DoubleValue)
			}
		}
		assert.True(t, foundVersion, "Should find version property")
		assert.True(t, foundValue, "Should find value property")

		// Verify custom properties
		customProperties := *retrieved.GetCustomProperties()
		var foundTeam, foundIsValidated bool
		for _, prop := range customProperties {
			switch prop.Name {
			case "team":
				foundTeam = true
				assert.Equal(t, "catalog-metrics-team", *prop.StringValue)
			case "is_validated":
				foundIsValidated = true
				assert.Equal(t, true, *prop.BoolValue)
			}
		}
		assert.True(t, foundTeam, "Should find team custom property")
		assert.True(t, foundIsValidated, "Should find is_validated custom property")
	})

	t.Run("TestSaveWithoutParentResource", func(t *testing.T) {
		// Test creating a catalog metrics artifact without parent resource attribution
		catalogMetricsArtifact := &models.CatalogMetricsArtifactImpl{
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:        apiutils.Of("standalone-catalog-metrics-artifact"),
				ExternalID:  apiutils.Of("standalone-catalog-metrics-ext"),
				MetricsType: models.MetricsTypeAccuracy,
			},
			Properties: &[]dbmodels.Properties{
				{
					Name:        "description",
					StringValue: apiutils.Of("Standalone catalog metrics artifact without parent"),
				},
			},
		}

		saved, err := repo.Save(catalogMetricsArtifact, nil)
		require.NoError(t, err)
		require.NotNil(t, saved)
		require.NotNil(t, saved.GetID())
		assert.Equal(t, "standalone-catalog-metrics-artifact", *saved.GetAttributes().Name)
		assert.Equal(t, models.MetricsTypeAccuracy, saved.GetAttributes().MetricsType)

		// Verify it can be retrieved
		retrieved, err := repo.GetByID(*saved.GetID())
		require.NoError(t, err)
		assert.Equal(t, "standalone-catalog-metrics-artifact", *retrieved.GetAttributes().Name)
		assert.Equal(t, models.MetricsTypeAccuracy, retrieved.GetAttributes().MetricsType)
	})

	t.Run("TestListOrdering", func(t *testing.T) {
		// Create a catalog model
		catalogModel := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("test-catalog-model-for-ordering-metrics"),
				ExternalID: apiutils.Of("catalog-model-ordering-metrics-ext"),
			},
		}
		savedCatalogModel, err := catalogModelRepo.Save(catalogModel)
		require.NoError(t, err)

		// Create artifacts sequentially with time delays to ensure deterministic ordering
		artifact1 := &models.CatalogMetricsArtifactImpl{
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:        apiutils.Of("time-test-catalog-metrics-artifact-1"),
				ExternalID:  apiutils.Of("time-catalog-metrics-ext-1"),
				MetricsType: models.MetricsTypeAccuracy,
			},
		}
		saved1, err := repo.Save(artifact1, savedCatalogModel.GetID())
		require.NoError(t, err)

		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)

		artifact2 := &models.CatalogMetricsArtifactImpl{
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:        apiutils.Of("time-test-catalog-metrics-artifact-2"),
				ExternalID:  apiutils.Of("time-catalog-metrics-ext-2"),
				MetricsType: models.MetricsTypePerformance,
			},
		}
		saved2, err := repo.Save(artifact2, savedCatalogModel.GetID())
		require.NoError(t, err)

		// Test ordering by CREATE_TIME
		listOptions := models.CatalogMetricsArtifactListOptions{
			Pagination: dbmodels.Pagination{
				OrderBy: apiutils.Of("CREATE_TIME"),
			},
		}

		result, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Find our test artifacts in the results
		var foundArtifact1, foundArtifact2 models.CatalogMetricsArtifact
		var index1, index2 = -1, -1

		for i, item := range result.Items {
			if *item.GetID() == *saved1.GetID() {
				foundArtifact1 = item
				index1 = i
			}
			if *item.GetID() == *saved2.GetID() {
				foundArtifact2 = item
				index2 = i
			}
		}

		// Verify both artifacts were found and artifact1 comes before artifact2 (ascending order)
		require.NotEqual(t, -1, index1, "Artifact 1 should be found in results")
		require.NotEqual(t, -1, index2, "Artifact 2 should be found in results")
		assert.Less(t, index1, index2, "Artifact 1 should come before Artifact 2 when ordered by CREATE_TIME")
		assert.Less(t, *foundArtifact1.GetAttributes().CreateTimeSinceEpoch, *foundArtifact2.GetAttributes().CreateTimeSinceEpoch, "Artifact 1 should have earlier create time")
	})

	t.Run("TestMetricsTypeField", func(t *testing.T) {
		// Test various metrics types
		metricsTypes := []models.MetricsType{models.MetricsTypeAccuracy, models.MetricsTypePerformance, models.MetricsTypeSecurityMetrics}

		catalogModel := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("test-catalog-model-for-metrics-types"),
				ExternalID: apiutils.Of("catalog-model-metrics-types-ext"),
			},
		}
		savedCatalogModel, err := catalogModelRepo.Save(catalogModel)
		require.NoError(t, err)

		for i, metricsType := range metricsTypes {
			artifact := &models.CatalogMetricsArtifactImpl{
				Attributes: &models.CatalogMetricsArtifactAttributes{
					Name:        apiutils.Of(fmt.Sprintf("metrics-type-test-%d", i)),
					ExternalID:  apiutils.Of(fmt.Sprintf("metrics-type-ext-%d", i)),
					MetricsType: metricsType,
				},
			}

			saved, err := repo.Save(artifact, savedCatalogModel.GetID())
			require.NoError(t, err)
			assert.Equal(t, metricsType, saved.GetAttributes().MetricsType)

			// Verify retrieval preserves metricsType
			retrieved, err := repo.GetByID(*saved.GetID())
			require.NoError(t, err)
			assert.Equal(t, metricsType, retrieved.GetAttributes().MetricsType)
		}
	})

	t.Run("TestSaveWithTypeIDSetting", func(t *testing.T) {
		// Create a catalog model
		catalogModel := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("test-catalog-model-for-typeid-metrics"),
				ExternalID: apiutils.Of("catalog-model-typeid-metrics-ext"),
			},
		}
		savedCatalogModel, err := catalogModelRepo.Save(catalogModel)
		require.NoError(t, err)

		// Test creating artifact without explicit type_id (should be set automatically)
		catalogMetricsArtifact := &models.CatalogMetricsArtifactImpl{
			// Intentionally not setting TypeID to test auto-setting
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:        apiutils.Of("typeid-test-metrics-artifact"),
				ExternalID:  apiutils.Of("typeid-metrics-ext-123"),
				MetricsType: models.MetricsTypeAccuracy,
			},
		}

		saved, err := repo.Save(catalogMetricsArtifact, savedCatalogModel.GetID())
		require.NoError(t, err)
		require.NotNil(t, saved)
		require.NotNil(t, saved.GetTypeID())
		assert.Equal(t, int32(typeID), *saved.GetTypeID())
		assert.Equal(t, "typeid-test-metrics-artifact", *saved.GetAttributes().Name)

		// Test with explicitly set type_id (should not be overridden)
		explicitTypeID := int32(typeID)
		catalogMetricsArtifact2 := &models.CatalogMetricsArtifactImpl{
			TypeID: &explicitTypeID,
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:        apiutils.Of("explicit-typeid-metrics-artifact"),
				ExternalID:  apiutils.Of("explicit-typeid-metrics-ext-123"),
				MetricsType: models.MetricsTypePerformance,
			},
		}

		saved2, err := repo.Save(catalogMetricsArtifact2, savedCatalogModel.GetID())
		require.NoError(t, err)
		require.NotNil(t, saved2)
		require.NotNil(t, saved2.GetTypeID())
		assert.Equal(t, explicitTypeID, *saved2.GetTypeID())
	})

	t.Run("TestSaveWithNameMatching", func(t *testing.T) {
		// Create a catalog model
		catalogModel := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("test-catalog-model-for-name-matching-metrics"),
				ExternalID: apiutils.Of("catalog-model-name-match-metrics-ext"),
			},
		}
		savedCatalogModel, err := catalogModelRepo.Save(catalogModel)
		require.NoError(t, err)

		// Create initial metrics artifact
		artifactName := "name-matching-metrics-artifact"
		catalogMetricsArtifact1 := &models.CatalogMetricsArtifactImpl{
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:        apiutils.Of(artifactName),
				ExternalID:  apiutils.Of("name-match-metrics-ext-123"),
				MetricsType: models.MetricsTypeAccuracy,
			},
			Properties: &[]dbmodels.Properties{
				{
					Name:        "accuracy",
					StringValue: apiutils.Of("0.95"),
				},
			},
		}

		saved1, err := repo.Save(catalogMetricsArtifact1, savedCatalogModel.GetID())
		require.NoError(t, err)
		require.NotNil(t, saved1)
		originalID := *saved1.GetID()
		assert.Equal(t, artifactName, *saved1.GetAttributes().Name)
		assert.Equal(t, models.MetricsTypeAccuracy, saved1.GetAttributes().MetricsType)

		// Create second artifact with same name (should update existing)
		catalogMetricsArtifact2 := &models.CatalogMetricsArtifactImpl{
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:        apiutils.Of(artifactName), // Same name
				ExternalID:  apiutils.Of("name-match-metrics-ext-456"),
				MetricsType: models.MetricsTypePerformance, // Different metrics type
			},
			Properties: &[]dbmodels.Properties{
				{
					Name:        "latency",
					StringValue: apiutils.Of("50ms"),
				},
			},
		}

		saved2, err := repo.Save(catalogMetricsArtifact2, savedCatalogModel.GetID())
		require.NoError(t, err)
		require.NotNil(t, saved2)

		// Should have same ID (updated existing)
		assert.Equal(t, originalID, *saved2.GetID())
		assert.Equal(t, artifactName, *saved2.GetAttributes().Name)
		assert.Equal(t, models.MetricsTypePerformance, saved2.GetAttributes().MetricsType)
		assert.Equal(t, "name-match-metrics-ext-456", *saved2.GetAttributes().ExternalID)

		// Verify by retrieving from database
		retrieved, err := repo.GetByID(originalID)
		require.NoError(t, err)
		assert.Equal(t, models.MetricsTypePerformance, retrieved.GetAttributes().MetricsType)
		assert.Equal(t, "name-match-metrics-ext-456", *retrieved.GetAttributes().ExternalID)

		// Verify properties were updated
		require.NotNil(t, retrieved.GetProperties())
		properties := *retrieved.GetProperties()
		var foundLatency bool
		for _, prop := range properties {
			if prop.Name == "latency" {
				foundLatency = true
				assert.Equal(t, "50ms", *prop.StringValue)
				break
			}
		}
		assert.True(t, foundLatency, "Should find updated latency property")

		// Test that artifact with different name creates new entity
		catalogMetricsArtifact3 := &models.CatalogMetricsArtifactImpl{
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:        apiutils.Of("different-name-metrics-artifact"),
				ExternalID:  apiutils.Of("different-name-metrics-ext-789"),
				MetricsType: models.MetricsTypeAccuracy,
			},
		}

		saved3, err := repo.Save(catalogMetricsArtifact3, savedCatalogModel.GetID())
		require.NoError(t, err)
		require.NotNil(t, saved3)

		// Should have different ID (new entity)
		assert.NotEqual(t, originalID, *saved3.GetID())
		assert.Equal(t, "different-name-metrics-artifact", *saved3.GetAttributes().Name)
	})

	t.Run("TestSaveWithNameMatchingNoExistingName", func(t *testing.T) {
		// Create a catalog model
		catalogModel := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("test-catalog-model-for-no-match-metrics"),
				ExternalID: apiutils.Of("catalog-model-no-match-metrics-ext"),
			},
		}
		savedCatalogModel, err := catalogModelRepo.Save(catalogModel)
		require.NoError(t, err)

		// Test saving artifact when no existing artifact with same name exists
		catalogMetricsArtifact := &models.CatalogMetricsArtifactImpl{
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:        apiutils.Of("unique-metrics-artifact-name"),
				ExternalID:  apiutils.Of("unique-metrics-ext-123"),
				MetricsType: models.MetricsTypeAccuracy,
			},
		}

		saved, err := repo.Save(catalogMetricsArtifact, savedCatalogModel.GetID())
		require.NoError(t, err)
		require.NotNil(t, saved)
		require.NotNil(t, saved.GetID())
		assert.Equal(t, "unique-metrics-artifact-name", *saved.GetAttributes().Name)
		assert.Equal(t, models.MetricsTypeAccuracy, saved.GetAttributes().MetricsType)
	})

	t.Run("TestSaveWithInvalidMetricsType", func(t *testing.T) {
		// Create a catalog model
		catalogModel := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("test-catalog-model-for-invalid-metrics"),
				ExternalID: apiutils.Of("catalog-model-invalid-metrics-ext"),
			},
		}
		savedCatalogModel, err := catalogModelRepo.Save(catalogModel)
		require.NoError(t, err)

		// Test saving artifact with invalid metrics type (should fail)
		catalogMetricsArtifact := &models.CatalogMetricsArtifactImpl{
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:        apiutils.Of("invalid-metrics-type-artifact"),
				ExternalID:  apiutils.Of("invalid-metrics-ext-123"),
				MetricsType: models.MetricsType("invalid-type"),
			},
		}

		_, err = repo.Save(catalogMetricsArtifact, savedCatalogModel.GetID())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown metrics type")
	})

	t.Run("TestBatchSaveWithSecurityMetrics", func(t *testing.T) {
		catalogModel := &models.CatalogModelImpl{
			Attributes: &models.CatalogModelAttributes{
				Name:       new("test-catalog-model-for-batch-security"),
				ExternalID: new("catalog-model-batch-security-ext"),
			},
		}
		savedCatalogModel, err := catalogModelRepo.Save(catalogModel)
		require.NoError(t, err)

		artifacts := []models.CatalogMetricsArtifact{
			&models.CatalogMetricsArtifactImpl{
				Attributes: &models.CatalogMetricsArtifactAttributes{
					Name:        new("batch-security-artifact-1"),
					ExternalID:  new("batch-security-ext-1"),
					MetricsType: models.MetricsTypeSecurityMetrics,
				},
			},
			&models.CatalogMetricsArtifactImpl{
				Attributes: &models.CatalogMetricsArtifactAttributes{
					Name:        new("batch-security-artifact-2"),
					ExternalID:  new("batch-security-ext-2"),
					MetricsType: models.MetricsTypeSecurityMetrics,
				},
				CustomProperties: &[]dbmodels.Properties{
					{Name: "attack_success_rate", DoubleValue: new(0.15)},
					{Name: "pass", BoolValue: new(true)},
				},
			},
		}

		saved, err := repo.BatchSave(artifacts, savedCatalogModel.GetID())
		require.NoError(t, err)
		require.Len(t, saved, 2)
		for _, a := range saved {
			assert.NotNil(t, a.GetID())
			assert.Equal(t, models.MetricsTypeSecurityMetrics, a.GetAttributes().MetricsType)
		}

		// Verify retrieval
		retrieved1, err := repo.GetByID(*saved[0].GetID())
		require.NoError(t, err)
		assert.Equal(t, models.MetricsTypeSecurityMetrics, retrieved1.GetAttributes().MetricsType)

		retrieved2, err := repo.GetByID(*saved[1].GetID())
		require.NoError(t, err)
		assert.Equal(t, models.MetricsTypeSecurityMetrics, retrieved2.GetAttributes().MetricsType)
		require.NotNil(t, retrieved2.GetCustomProperties())
		props := *retrieved2.GetCustomProperties()
		var foundRate bool
		for _, p := range props {
			if p.Name == "attack_success_rate" {
				foundRate = true
				assert.Equal(t, 0.15, *p.DoubleValue)
			}
		}
		assert.True(t, foundRate, "attack_success_rate custom property should be present")
	})
}
