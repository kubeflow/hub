package service_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/kubeflow/hub/internal/db/models"
	"github.com/kubeflow/hub/internal/db/service"
	"github.com/kubeflow/hub/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelVersionRepository(t *testing.T) {
	sharedDB, cleanup := testutils.SetupMySQLWithMigrations(t, service.DatastoreSpec())
	defer cleanup()

	// Get the actual ModelVersion type ID from the database
	typeID := getModelVersionTypeID(t, sharedDB)
	repo := service.NewModelVersionRepository(sharedDB, typeID)

	// Also get RegisteredModel type ID for creating parent models
	registeredModelTypeID := getRegisteredModelTypeID(t, sharedDB)
	registeredModelRepo := service.NewRegisteredModelRepository(sharedDB, registeredModelTypeID)

	t.Run("TestSave", func(t *testing.T) {
		// First create a parent registered model
		parentModel := &models.RegisteredModelImpl{
			TypeID: new(int32(registeredModelTypeID)),
			Attributes: &models.RegisteredModelAttributes{
				Name: new("parent-model-for-version"),
			},
		}
		savedParent, err := registeredModelRepo.Save(parentModel)
		require.NoError(t, err)

		// Test creating a new model version
		modelVersion := &models.ModelVersionImpl{
			TypeID: new(int32(typeID)),
			Attributes: &models.ModelVersionAttributes{
				Name:       new(fmt.Sprintf("%d:test-version", *savedParent.GetID())),
				ExternalID: new("version-ext-123"),
			},
			Properties: &[]models.Properties{
				{
					Name:        "description",
					StringValue: new("Test version description"),
				},
				{
					Name:     "registered_model_id",
					IntValue: savedParent.GetID(),
				},
			},
			CustomProperties: &[]models.Properties{
				{
					Name:        "custom-version-prop",
					StringValue: new("custom-version-value"),
				},
			},
		}

		saved, err := repo.Save(modelVersion)
		require.NoError(t, err)
		require.NotNil(t, saved)
		require.NotNil(t, saved.GetID())
		assert.Equal(t, fmt.Sprintf("%d:test-version", *savedParent.GetID()), *saved.GetAttributes().Name)
		assert.Equal(t, "version-ext-123", *saved.GetAttributes().ExternalID)

		// Test updating the same model version
		modelVersion.ID = saved.GetID()
		modelVersion.GetAttributes().Name = new(fmt.Sprintf("%d:updated-version", *savedParent.GetID()))
		// Preserve CreateTimeSinceEpoch from the saved entity (simulating what OpenAPI converter would do)
		modelVersion.GetAttributes().CreateTimeSinceEpoch = saved.GetAttributes().CreateTimeSinceEpoch

		updated, err := repo.Save(modelVersion)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, *saved.GetID(), *updated.GetID())
		assert.Equal(t, fmt.Sprintf("%d:updated-version", *savedParent.GetID()), *updated.GetAttributes().Name)
	})

	t.Run("TestGetByID", func(t *testing.T) {
		// First create a parent registered model
		parentModel := &models.RegisteredModelImpl{
			TypeID: new(int32(registeredModelTypeID)),
			Attributes: &models.RegisteredModelAttributes{
				Name: new("parent-model-for-getbyid"),
			},
		}
		savedParent, err := registeredModelRepo.Save(parentModel)
		require.NoError(t, err)

		// First create a model version to retrieve
		modelVersion := &models.ModelVersionImpl{
			TypeID: new(int32(typeID)),
			Attributes: &models.ModelVersionAttributes{
				Name:       new(fmt.Sprintf("%d:get-test-version", *savedParent.GetID())),
				ExternalID: new("get-version-ext-123"),
			},
			Properties: &[]models.Properties{
				{
					Name:     "registered_model_id",
					IntValue: savedParent.GetID(),
				},
			},
		}

		saved, err := repo.Save(modelVersion)
		require.NoError(t, err)
		require.NotNil(t, saved.GetID())

		// Test retrieving by ID
		retrieved, err := repo.GetByID(*saved.GetID())
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, *saved.GetID(), *retrieved.GetID())
		assert.Equal(t, fmt.Sprintf("%d:get-test-version", *savedParent.GetID()), *retrieved.GetAttributes().Name)
		assert.Equal(t, "get-version-ext-123", *retrieved.GetAttributes().ExternalID)

		// Test retrieving non-existent ID
		_, err = repo.GetByID(99999)
		assert.Error(t, err)
	})

	t.Run("TestList", func(t *testing.T) {
		// Create a parent registered model for the versions
		parentModel := &models.RegisteredModelImpl{
			TypeID: new(int32(registeredModelTypeID)),
			Attributes: &models.RegisteredModelAttributes{
				Name: new("parent-model-for-list"),
			},
		}
		savedParent, err := registeredModelRepo.Save(parentModel)
		require.NoError(t, err)

		// Create multiple model versions for listing
		testVersions := []*models.ModelVersionImpl{
			{
				TypeID: new(int32(typeID)),
				Attributes: &models.ModelVersionAttributes{
					Name:       new(fmt.Sprintf("%d:list-version-1", *savedParent.GetID())),
					ExternalID: new("list-version-ext-1"),
				},
				Properties: &[]models.Properties{
					{
						Name:     "registered_model_id",
						IntValue: savedParent.GetID(),
					},
				},
			},
			{
				TypeID: new(int32(typeID)),
				Attributes: &models.ModelVersionAttributes{
					Name:       new(fmt.Sprintf("%d:list-version-2", *savedParent.GetID())),
					ExternalID: new("list-version-ext-2"),
				},
				Properties: &[]models.Properties{
					{
						Name:     "registered_model_id",
						IntValue: savedParent.GetID(),
					},
				},
			},
			{
				TypeID: new(int32(typeID)),
				Attributes: &models.ModelVersionAttributes{
					Name:       new(fmt.Sprintf("%d:list-version-3", *savedParent.GetID())),
					ExternalID: new("list-version-ext-3"),
				},
				Properties: &[]models.Properties{
					{
						Name:     "registered_model_id",
						IntValue: savedParent.GetID(),
					},
				},
			},
		}

		for _, version := range testVersions {
			_, err := repo.Save(version)
			require.NoError(t, err)
		}

		// Test listing all versions with basic pagination
		pageSize := int32(10)
		listOptions := models.ModelVersionListOptions{}
		listOptions.PageSize = &pageSize

		result, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.GreaterOrEqual(t, len(result.Items), 3) // At least our 3 test versions

		// Test listing by name
		listOptions = models.ModelVersionListOptions{
			Name: new("list-version-1"),
		}
		listOptions.PageSize = &pageSize

		result, err = repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		if len(result.Items) > 0 {
			assert.Equal(t, 1, len(result.Items))
			assert.Equal(t, fmt.Sprintf("%d:list-version-1", *savedParent.GetID()), *result.Items[0].GetAttributes().Name)
		}

		// Test listing by external ID
		listOptions = models.ModelVersionListOptions{
			ExternalID: new("list-version-ext-2"),
		}
		listOptions.PageSize = &pageSize

		result, err = repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		if len(result.Items) > 0 {
			assert.Equal(t, 1, len(result.Items))
			assert.Equal(t, "list-version-ext-2", *result.Items[0].GetAttributes().ExternalID)
		}

		// Test listing by parent resource ID
		listOptions = models.ModelVersionListOptions{
			ParentResourceID: savedParent.GetID(),
		}
		listOptions.PageSize = &pageSize

		result, err = repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Test ordering by ID (deterministic)
		listOptions = models.ModelVersionListOptions{
			Pagination: models.Pagination{
				OrderBy: new("ID"),
			},
		}
		listOptions.PageSize = &pageSize

		result, err = repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		// Verify we get results back and they are ordered by ID
		assert.GreaterOrEqual(t, len(result.Items), 1)
		if len(result.Items) > 1 {
			// Verify ascending ID order
			firstID := *result.Items[0].GetID()
			secondID := *result.Items[1].GetID()
			assert.Less(t, firstID, secondID, "Results should be ordered by ID ascending")
		}
	})

	t.Run("TestListOrdering", func(t *testing.T) {
		// First create a parent registered model
		parentModel := &models.RegisteredModelImpl{
			TypeID: new(int32(registeredModelTypeID)),
			Attributes: &models.RegisteredModelAttributes{
				Name: new("parent-model-for-ordering"),
			},
		}
		savedParent, err := registeredModelRepo.Save(parentModel)
		require.NoError(t, err)

		// Create versions sequentially with time delays to ensure deterministic ordering
		version1 := &models.ModelVersionImpl{
			TypeID: new(int32(typeID)),
			Attributes: &models.ModelVersionAttributes{
				Name: new(fmt.Sprintf("%d:time-test-version-1", *savedParent.GetID())),
			},
			Properties: &[]models.Properties{
				{
					Name:     "registered_model_id",
					IntValue: savedParent.GetID(),
				},
			},
		}
		saved1, err := repo.Save(version1)
		require.NoError(t, err)

		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)

		version2 := &models.ModelVersionImpl{
			TypeID: new(int32(typeID)),
			Attributes: &models.ModelVersionAttributes{
				Name: new(fmt.Sprintf("%d:time-test-version-2", *savedParent.GetID())),
			},
			Properties: &[]models.Properties{
				{
					Name:     "registered_model_id",
					IntValue: savedParent.GetID(),
				},
			},
		}
		saved2, err := repo.Save(version2)
		require.NoError(t, err)

		// Test ordering by CREATE_TIME
		pageSize := int32(100) // Increased page size to ensure all test entities are included
		listOptions := models.ModelVersionListOptions{
			Pagination: models.Pagination{
				OrderBy: new("CREATE_TIME"),
			},
		}
		listOptions.PageSize = &pageSize

		result, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Find our test versions in the results
		var foundVersion1, foundVersion2 models.ModelVersion
		var index1, index2 = -1, -1

		for i, item := range result.Items {
			if *item.GetID() == *saved1.GetID() {
				foundVersion1 = item
				index1 = i
			}
			if *item.GetID() == *saved2.GetID() {
				foundVersion2 = item
				index2 = i
			}
		}

		// Verify both versions were found and version1 comes before version2 (ascending order)
		require.NotEqual(t, -1, index1, "Version 1 should be found in results")
		require.NotEqual(t, -1, index2, "Version 2 should be found in results")
		assert.Less(t, index1, index2, "Version 1 should come before Version 2 when ordered by CREATE_TIME")
		assert.Less(t, *foundVersion1.GetAttributes().CreateTimeSinceEpoch, *foundVersion2.GetAttributes().CreateTimeSinceEpoch, "Version 1 should have earlier create time")
	})

	t.Run("TestSaveWithProperties", func(t *testing.T) {
		// First create a parent registered model
		parentModel := &models.RegisteredModelImpl{
			TypeID: new(int32(registeredModelTypeID)),
			Attributes: &models.RegisteredModelAttributes{
				Name: new("parent-model-for-props"),
			},
		}
		savedParent, err := registeredModelRepo.Save(parentModel)
		require.NoError(t, err)

		modelVersion := &models.ModelVersionImpl{
			TypeID: new(int32(typeID)),
			Attributes: &models.ModelVersionAttributes{
				Name: new(fmt.Sprintf("%d:props-test-version", *savedParent.GetID())),
			},
			Properties: &[]models.Properties{
				{
					Name:        "description",
					StringValue: new("Version with properties"),
				},
				{
					Name:     "version_number",
					IntValue: new(int32(1)),
				},
				{
					Name:     "registered_model_id",
					IntValue: savedParent.GetID(),
				},
			},
			CustomProperties: &[]models.Properties{
				{
					Name:        "team",
					StringValue: new("ml-team"),
				},
				{
					Name:     "priority",
					IntValue: new(int32(5)),
				},
			},
		}

		saved, err := repo.Save(modelVersion)
		require.NoError(t, err)
		require.NotNil(t, saved)

		// Verify properties were saved
		retrieved, err := repo.GetByID(*saved.GetID())
		require.NoError(t, err)

		assert.NotNil(t, retrieved.GetProperties())
		assert.Len(t, *retrieved.GetProperties(), 3) // description, version_number, registered_model_id

		assert.NotNil(t, retrieved.GetCustomProperties())
		assert.Len(t, *retrieved.GetCustomProperties(), 2)
	})
}

func TestModelVersionRepository_FilterQuery(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Get the actual type IDs from the database
	modelVersionTypeID := getModelVersionTypeID(t, db)
	registeredModelTypeID := getRegisteredModelTypeID(t, db)

	modelVersionRepo := service.NewModelVersionRepository(db, modelVersionTypeID)
	registeredModelRepo := service.NewRegisteredModelRepository(db, registeredModelTypeID)

	// Create a parent registered model
	registeredModel := &models.RegisteredModelImpl{
		TypeID: new(int32(registeredModelTypeID)),
		Attributes: &models.RegisteredModelAttributes{
			Name: new("test-parent-model"),
		},
	}
	savedRegisteredModel, err := registeredModelRepo.Save(registeredModel)
	require.NoError(t, err)

	// Create multiple model versions with different properties for filtering
	modelVersion1 := &models.ModelVersionImpl{
		TypeID: new(int32(modelVersionTypeID)),
		Attributes: &models.ModelVersionAttributes{
			Name: new(fmt.Sprintf("%d:pytorch-model-version", *savedRegisteredModel.GetID())),
		},
		Properties: &[]models.Properties{
			{
				Name:     "registered_model_id",
				IntValue: savedRegisteredModel.GetID(),
			},
			{
				Name:        "state",
				StringValue: new("LIVE"),
			},
			{
				Name:        "author",
				StringValue: new("data-scientist-1"),
			},
		},
		CustomProperties: &[]models.Properties{
			{
				Name:             "framework",
				StringValue:      new("pytorch"),
				IsCustomProperty: true,
			},
			{
				Name:             "accuracy",
				DoubleValue:      new(0.95),
				IsCustomProperty: true,
			},
			{
				Name:             "epoch_count",
				IntValue:         new(int32(100)),
				IsCustomProperty: true,
			},
		},
	}
	_, err = modelVersionRepo.Save(modelVersion1)
	require.NoError(t, err)

	modelVersion2 := &models.ModelVersionImpl{
		TypeID: new(int32(modelVersionTypeID)),
		Attributes: &models.ModelVersionAttributes{
			Name: new(fmt.Sprintf("%d:tensorflow-model-version", *savedRegisteredModel.GetID())),
		},
		Properties: &[]models.Properties{
			{
				Name:     "registered_model_id",
				IntValue: savedRegisteredModel.GetID(),
			},
			{
				Name:        "state",
				StringValue: new("ARCHIVED"),
			},
			{
				Name:        "author",
				StringValue: new("data-scientist-2"),
			},
		},
		CustomProperties: &[]models.Properties{
			{
				Name:             "framework",
				StringValue:      new("tensorflow"),
				IsCustomProperty: true,
			},
			{
				Name:             "accuracy",
				DoubleValue:      new(0.89),
				IsCustomProperty: true,
			},
			{
				Name:             "epoch_count",
				IntValue:         new(int32(50)),
				IsCustomProperty: true,
			},
		},
	}
	_, err = modelVersionRepo.Save(modelVersion2)
	require.NoError(t, err)

	modelVersion3 := &models.ModelVersionImpl{
		TypeID: new(int32(modelVersionTypeID)),
		Attributes: &models.ModelVersionAttributes{
			Name: new(fmt.Sprintf("%d:sklearn-model-version", *savedRegisteredModel.GetID())),
		},
		Properties: &[]models.Properties{
			{
				Name:     "registered_model_id",
				IntValue: savedRegisteredModel.GetID(),
			},
			{
				Name:        "state",
				StringValue: new("LIVE"),
			},
			{
				Name:        "author",
				StringValue: new("data-scientist-1"),
			},
		},
		CustomProperties: &[]models.Properties{
			{
				Name:             "framework",
				StringValue:      new("sklearn"),
				IsCustomProperty: true,
			},
			{
				Name:             "accuracy",
				DoubleValue:      new(0.92),
				IsCustomProperty: true,
			},
			{
				Name:             "epoch_count",
				IntValue:         new(int32(25)),
				IsCustomProperty: true,
			},
		},
	}
	_, err = modelVersionRepo.Save(modelVersion3)
	require.NoError(t, err)

	// Test core property filtering
	t.Run("CorePropertyFilter", func(t *testing.T) {
		filterQuery := `name = "pytorch-model-version"`
		pageSize := int32(10)
		listOptions := models.ModelVersionListOptions{
			Pagination: models.Pagination{
				PageSize:    &pageSize,
				FilterQuery: &filterQuery,
			},
		}

		result, err := modelVersionRepo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, len(result.Items))
		assert.Equal(t, fmt.Sprintf("%d:pytorch-model-version", *savedRegisteredModel.GetID()), *result.Items[0].GetAttributes().Name)
	})

	// Test custom property filtering
	t.Run("CustomPropertyFilter", func(t *testing.T) {
		filterQuery := `framework = "tensorflow"`
		pageSize := int32(10)
		listOptions := models.ModelVersionListOptions{
			Pagination: models.Pagination{
				PageSize:    &pageSize,
				FilterQuery: &filterQuery,
			},
		}

		result, err := modelVersionRepo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, len(result.Items))
		assert.Equal(t, fmt.Sprintf("%d:tensorflow-model-version", *savedRegisteredModel.GetID()), *result.Items[0].GetAttributes().Name)
	})

	// Test numeric custom property filtering
	t.Run("NumericCustomPropertyFilter", func(t *testing.T) {
		filterQuery := `epoch_count >= 50`
		pageSize := int32(10)
		listOptions := models.ModelVersionListOptions{
			Pagination: models.Pagination{
				PageSize:    &pageSize,
				FilterQuery: &filterQuery,
			},
		}

		result, err := modelVersionRepo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, len(result.Items)) // pytorch (100) and tensorflow (50)
	})

	// Test double custom property filtering
	t.Run("DoubleCustomPropertyFilter", func(t *testing.T) {
		filterQuery := `accuracy > 0.9`
		pageSize := int32(10)
		listOptions := models.ModelVersionListOptions{
			Pagination: models.Pagination{
				PageSize:    &pageSize,
				FilterQuery: &filterQuery,
			},
		}

		result, err := modelVersionRepo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, len(result.Items)) // pytorch (0.95) and sklearn (0.92)
	})

	// Test standard property filtering
	t.Run("StandardPropertyFilter", func(t *testing.T) {
		filterQuery := `state = "LIVE"`
		pageSize := int32(10)
		listOptions := models.ModelVersionListOptions{
			Pagination: models.Pagination{
				PageSize:    &pageSize,
				FilterQuery: &filterQuery,
			},
		}

		result, err := modelVersionRepo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, len(result.Items)) // pytorch and sklearn are LIVE
	})

	// Test complex AND filter
	t.Run("ComplexANDFilter", func(t *testing.T) {
		filterQuery := `framework = "pytorch" AND accuracy > 0.9`
		pageSize := int32(10)
		listOptions := models.ModelVersionListOptions{
			Pagination: models.Pagination{
				PageSize:    &pageSize,
				FilterQuery: &filterQuery,
			},
		}

		result, err := modelVersionRepo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, len(result.Items))
		assert.Equal(t, fmt.Sprintf("%d:pytorch-model-version", *savedRegisteredModel.GetID()), *result.Items[0].GetAttributes().Name)
	})

	// Test complex OR filter
	t.Run("ComplexORFilter", func(t *testing.T) {
		filterQuery := `framework = "pytorch" OR framework = "sklearn"`
		pageSize := int32(10)
		listOptions := models.ModelVersionListOptions{
			Pagination: models.Pagination{
				PageSize:    &pageSize,
				FilterQuery: &filterQuery,
			},
		}

		result, err := modelVersionRepo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, len(result.Items)) // pytorch and sklearn
	})

	// Test ILIKE operator
	t.Run("ILIKEFilter", func(t *testing.T) {
		filterQuery := `name ILIKE "%model%"`
		pageSize := int32(10)
		listOptions := models.ModelVersionListOptions{
			Pagination: models.Pagination{
				PageSize:    &pageSize,
				FilterQuery: &filterQuery,
			},
		}

		result, err := modelVersionRepo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 3, len(result.Items)) // All model versions contain "model"
	})

	// Test mixed core and custom property filter
	t.Run("MixedCoreAndCustomFilter", func(t *testing.T) {
		filterQuery := `name ILIKE "%pytorch%" AND accuracy > 0.9`
		pageSize := int32(10)
		listOptions := models.ModelVersionListOptions{
			Pagination: models.Pagination{
				PageSize:    &pageSize,
				FilterQuery: &filterQuery,
			},
		}

		result, err := modelVersionRepo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, len(result.Items))
		assert.Equal(t, fmt.Sprintf("%d:pytorch-model-version", *savedRegisteredModel.GetID()), *result.Items[0].GetAttributes().Name)
	})

	// Test author property filtering
	t.Run("AuthorPropertyFilter", func(t *testing.T) {
		filterQuery := `author = "data-scientist-1"`
		pageSize := int32(10)
		listOptions := models.ModelVersionListOptions{
			Pagination: models.Pagination{
				PageSize:    &pageSize,
				FilterQuery: &filterQuery,
			},
		}

		result, err := modelVersionRepo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, len(result.Items)) // pytorch and sklearn have data-scientist-1
	})

	// Test invalid filter query
	t.Run("InvalidFilterQuery", func(t *testing.T) {
		filterQuery := `invalid syntax =`
		pageSize := int32(10)
		listOptions := models.ModelVersionListOptions{
			Pagination: models.Pagination{
				PageSize:    &pageSize,
				FilterQuery: &filterQuery,
			},
		}

		result, err := modelVersionRepo.List(listOptions)
		require.Error(t, err)
		require.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid filter query")
	})

	// Test with parentheses grouping
	t.Run("ParenthesesGrouping", func(t *testing.T) {
		filterQuery := `(framework = "pytorch" OR framework = "tensorflow") AND epoch_count > 40`
		pageSize := int32(10)
		listOptions := models.ModelVersionListOptions{
			Pagination: models.Pagination{
				PageSize:    &pageSize,
				FilterQuery: &filterQuery,
			},
		}

		result, err := modelVersionRepo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, len(result.Items)) // pytorch (100 epochs) and tensorflow (50 epochs)
	})
}

func TestModelVersionRepository_FilterQueryNameIsLiteral(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	modelVersionTypeID := getModelVersionTypeID(t, db)
	registeredModelTypeID := getRegisteredModelTypeID(t, db)

	modelVersionRepo := service.NewModelVersionRepository(db, modelVersionTypeID)
	registeredModelRepo := service.NewRegisteredModelRepository(db, registeredModelTypeID)

	savedRegisteredModel, err := registeredModelRepo.Save(&models.RegisteredModelImpl{
		TypeID: new(int32(registeredModelTypeID)),
		Attributes: &models.RegisteredModelAttributes{
			Name: new("literal-name-model"),
		},
	})
	require.NoError(t, err)

	for _, name := range []string{"v_1", "vX1", `v\1`} {
		_, err = modelVersionRepo.Save(&models.ModelVersionImpl{
			TypeID: new(int32(modelVersionTypeID)),
			Attributes: &models.ModelVersionAttributes{
				Name: new(fmt.Sprintf("%d:%s", *savedRegisteredModel.GetID(), name)),
			},
			Properties: &[]models.Properties{
				{
					Name:     "registered_model_id",
					IntValue: savedRegisteredModel.GetID(),
				},
			},
		})
		require.NoError(t, err)
	}

	// "_" and "%" in an equality filter are part of the name, not LIKE wildcards.
	tests := []struct {
		filterQuery string
		want        []string
	}{
		{filterQuery: `name = "v_1"`, want: []string{"v_1"}},
		{filterQuery: `name = "v%"`, want: nil},
		{filterQuery: `name = "vX1"`, want: []string{"vX1"}},
		{filterQuery: `name = "v\\1"`, want: []string{`v\1`}},
		{filterQuery: `name = "v_1" OR name = "vX1"`, want: []string{"v_1", "vX1"}},
	}

	for _, tt := range tests {
		t.Run(tt.filterQuery, func(t *testing.T) {
			filterQuery := tt.filterQuery
			pageSize := int32(10)
			result, err := modelVersionRepo.List(models.ModelVersionListOptions{
				Pagination: models.Pagination{
					PageSize:    &pageSize,
					FilterQuery: &filterQuery,
				},
			})
			require.NoError(t, err)

			var got []string
			for _, item := range result.Items {
				got = append(got, *item.GetAttributes().Name)
			}
			var want []string
			for _, name := range tt.want {
				want = append(want, fmt.Sprintf("%d:%s", *savedRegisteredModel.GetID(), name))
			}
			assert.ElementsMatch(t, want, got)
		})
	}
}
