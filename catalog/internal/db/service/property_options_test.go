package service_test

import (
	"testing"

	modelcatalogmodels "github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	modelcatalogservice "github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/service"
	"github.com/kubeflow/hub/catalog/internal/db/models"
	"github.com/kubeflow/hub/catalog/internal/db/service"
	"github.com/kubeflow/hub/catalog/internal/testhelpers"
	"github.com/kubeflow/hub/internal/platform/apiutils"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/kubeflow/hub/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPropertyOptionsRepository_RefreshOnEmptyDatabase verifies that the materialized
// views can be refreshed and queried even when no models have been loaded.
// This is a regression test for the startup refresh fix - previously, querying
// unpopulated materialized views would fail with "has not been populated" error.
func TestPropertyOptionsRepository_RefreshOnEmptyDatabase(t *testing.T) {
	// Create a database with migrations
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testhelpers.MustDatastoreSpec(t))
	defer cleanup()

	// Clean up all test data to ensure empty database
	testutils.CleanupPostgresTestData(t, sharedDB)

	repo := service.NewPropertyOptionsRepository(sharedDB)
	catalogModelTypeID := getCatalogModelTypeID(t, sharedDB)
	modelArtifactTypeID := getCatalogModelArtifactTypeID(t, sharedDB)

	t.Run("RefreshAndListWithNoModels_ContextPropertyOptions", func(t *testing.T) {
		// Refresh should succeed even with no data
		err := repo.Refresh(models.ContextPropertyOptionType)
		require.NoError(t, err, "Refresh should succeed on empty database")

		// List should succeed after refresh (returning empty results)
		options, err := repo.List(models.ContextPropertyOptionType, catalogModelTypeID)
		require.NoError(t, err, "List should succeed after refresh even with no models")
		assert.NotNil(t, options)
		assert.Empty(t, options, "Should return empty list when no models exist")
	})

	t.Run("RefreshAndListWithNoModels_ArtifactPropertyOptions", func(t *testing.T) {
		// Refresh should succeed even with no data
		err := repo.Refresh(models.ArtifactPropertyOptionType)
		require.NoError(t, err, "Refresh should succeed on empty database")

		// List should succeed after refresh (returning empty results)
		options, err := repo.List(models.ArtifactPropertyOptionType, modelArtifactTypeID)
		require.NoError(t, err, "List should succeed after refresh even with no models")
		assert.NotNil(t, options)
		assert.Empty(t, options, "Should return empty list when no artifacts exist")
	})

	t.Run("ListAllTypesWithNoModels", func(t *testing.T) {
		// First refresh both views
		require.NoError(t, repo.Refresh(models.ContextPropertyOptionType))
		require.NoError(t, repo.Refresh(models.ArtifactPropertyOptionType))

		// List with typeID=0 should return all options (empty in this case)
		contextOptions, err := repo.List(models.ContextPropertyOptionType, 0)
		require.NoError(t, err)
		assert.NotNil(t, contextOptions)
		assert.Empty(t, contextOptions, "Should return empty list when no models exist")

		artifactOptions, err := repo.List(models.ArtifactPropertyOptionType, 0)
		require.NoError(t, err)
		assert.NotNil(t, artifactOptions)
		assert.Empty(t, artifactOptions, "Should return empty list when no artifacts exist")
	})
}

func TestPropertyOptionsRepository(t *testing.T) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testhelpers.MustDatastoreSpec(t))
	defer cleanup()

	repo := service.NewPropertyOptionsRepository(sharedDB)

	// Get necessary type IDs for creating test data
	catalogModelTypeID := getCatalogModelTypeID(t, sharedDB)
	modelArtifactTypeID := getCatalogModelArtifactTypeID(t, sharedDB)

	// Create test repositories for setting up data
	catalogModelRepo := modelcatalogservice.NewCatalogModelRepository(sharedDB, catalogModelTypeID)
	artifactRepo := modelcatalogservice.NewCatalogModelArtifactRepository(sharedDB, modelArtifactTypeID)

	t.Run("Refresh_ContextPropertyOptions", func(t *testing.T) {
		// Test refreshing context property options materialized view
		err := repo.Refresh(models.ContextPropertyOptionType)
		assert.NoError(t, err)
	})

	t.Run("Refresh_ArtifactPropertyOptions", func(t *testing.T) {
		// Test refreshing artifact property options materialized view
		err := repo.Refresh(models.ArtifactPropertyOptionType)
		assert.NoError(t, err)
	})

	t.Run("Refresh_InvalidType", func(t *testing.T) {
		// Test error handling for invalid property option type
		err := repo.Refresh(models.PropertyOptionType(999))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid property option type")
	})

	t.Run("List_ContextPropertyOptions_SharedTestEnvironment", func(t *testing.T) {
		// Refresh the view first
		err := repo.Refresh(models.ContextPropertyOptionType)
		require.NoError(t, err)

		// List context property options for the test type ID
		options, err := repo.List(models.ContextPropertyOptionType, catalogModelTypeID)
		assert.NoError(t, err)
		assert.NotNil(t, options)
		// In shared test environment, other tests may have created data already
		// Just verify the function works and returns valid data if any exists
		for _, option := range options {
			assert.Equal(t, catalogModelTypeID, option.TypeID)
			assert.NotEmpty(t, option.Name)
		}
	})

	t.Run("List_ArtifactPropertyOptions_SharedTestEnvironment", func(t *testing.T) {
		// Refresh the view first
		err := repo.Refresh(models.ArtifactPropertyOptionType)
		require.NoError(t, err)

		// List artifact property options for the test type ID
		options, err := repo.List(models.ArtifactPropertyOptionType, modelArtifactTypeID)
		assert.NoError(t, err)
		assert.NotNil(t, options)
		// In shared test environment, other tests may have created data already
		// Just verify the function works and returns valid data if any exists
		for _, option := range options {
			assert.Equal(t, modelArtifactTypeID, option.TypeID)
			assert.NotEmpty(t, option.Name)
		}
	})

	t.Run("List_NonExistentTypeID", func(t *testing.T) {
		// Test with a type ID that doesn't exist - should return empty results
		nonExistentTypeID := int32(99999)

		// Test context property options
		options, err := repo.List(models.ContextPropertyOptionType, nonExistentTypeID)
		assert.NoError(t, err)
		assert.NotNil(t, options)
		assert.Len(t, options, 0)

		// Test artifact property options
		options, err = repo.List(models.ArtifactPropertyOptionType, nonExistentTypeID)
		assert.NoError(t, err)
		assert.NotNil(t, options)
		assert.Len(t, options, 0)
	})

	t.Run("List_InvalidType", func(t *testing.T) {
		// Test error handling for invalid property option type
		options, err := repo.List(models.PropertyOptionType(999), catalogModelTypeID)
		assert.Error(t, err)
		assert.Nil(t, options)
		assert.Contains(t, err.Error(), "invalid property option type")
	})

	t.Run("List_ContextPropertyOptions_WithData", func(t *testing.T) {
		// Create a catalog model with properties to populate the materialized view
		catalogModel := &modelcatalogmodels.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &modelcatalogmodels.CatalogModelAttributes{
				Name:       apiutils.Of("test-model-for-context-properties"),
				ExternalID: apiutils.Of("context-props-test-123"),
			},
			CustomProperties: &[]dbmodels.Properties{
				{
					Name:        "custom_prop_1",
					StringValue: apiutils.Of("value1"),
				},
				{
					Name:     "version_number",
					IntValue: apiutils.Of(int32(1)),
				},
				{
					Name:        "accuracy",
					DoubleValue: apiutils.Of(0.95),
				},
			},
		}

		savedModel, err := catalogModelRepo.Save(catalogModel)
		require.NoError(t, err)
		require.NotNil(t, savedModel)

		// Refresh the materialized view to include our new data
		err = repo.Refresh(models.ContextPropertyOptionType)
		require.NoError(t, err)

		// List context property options
		options, err := repo.List(models.ContextPropertyOptionType, catalogModelTypeID)
		assert.NoError(t, err)
		assert.NotNil(t, options)

		// We should have at least some property options now
		// The exact number depends on what properties were created
		if len(options) > 0 {
			// Verify the structure of returned options
			for _, option := range options {
				assert.Equal(t, catalogModelTypeID, option.TypeID)
				assert.NotEmpty(t, option.Name)
				// At least one of the value fields should be populated
				hasValue := len(option.StringValue) > 0 ||
					len(option.ArrayValue) > 0 ||
					option.MinDoubleValue != nil ||
					option.MaxDoubleValue != nil ||
					option.MinIntValue != nil ||
					option.MaxIntValue != nil
				assert.True(t, hasValue, "Option should have at least one value field populated")
			}
		}
	})

	t.Run("List_ArtifactPropertyOptions_WithData", func(t *testing.T) {
		// First create a catalog model as parent
		catalogModel := &modelcatalogmodels.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &modelcatalogmodels.CatalogModelAttributes{
				Name:       apiutils.Of("test-model-for-artifact-properties"),
				ExternalID: apiutils.Of("artifact-props-test-123"),
			},
		}
		savedModel, err := catalogModelRepo.Save(catalogModel)
		require.NoError(t, err)

		// Create an artifact with properties
		artifact := &modelcatalogmodels.CatalogModelArtifactImpl{
			TypeID: apiutils.Of(int32(modelArtifactTypeID)),
			Attributes: &modelcatalogmodels.CatalogModelArtifactAttributes{
				Name:       apiutils.Of("test-artifact-with-properties"),
				ExternalID: apiutils.Of("artifact-props-test-456"),
				URI:        apiutils.Of("s3://bucket/model.pkl"),
			},
			CustomProperties: &[]dbmodels.Properties{
				{
					Name:        "model_type",
					StringValue: apiutils.Of("classification"),
				},
				{
					Name:     "file_size",
					IntValue: apiutils.Of(int32(1024)),
				},
				{
					Name:        "validation_accuracy",
					DoubleValue: apiutils.Of(0.92),
				},
			},
		}

		savedArtifact, err := artifactRepo.Save(artifact, savedModel.GetID())
		require.NoError(t, err)
		require.NotNil(t, savedArtifact)

		// Refresh the materialized view to include our new data
		err = repo.Refresh(models.ArtifactPropertyOptionType)
		require.NoError(t, err)

		// List artifact property options
		options, err := repo.List(models.ArtifactPropertyOptionType, modelArtifactTypeID)
		assert.NoError(t, err)
		assert.NotNil(t, options)

		// We should have some property options now
		if len(options) > 0 {
			// Verify the structure of returned options
			for _, option := range options {
				assert.Equal(t, modelArtifactTypeID, option.TypeID)
				assert.NotEmpty(t, option.Name)
				// At least one of the value fields should be populated
				hasValue := len(option.StringValue) > 0 ||
					len(option.ArrayValue) > 0 ||
					option.MinDoubleValue != nil ||
					option.MaxDoubleValue != nil ||
					option.MinIntValue != nil ||
					option.MaxIntValue != nil
				assert.True(t, hasValue, "Option should have at least one value field populated")
			}
		}
	})
}
