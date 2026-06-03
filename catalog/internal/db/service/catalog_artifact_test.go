package service_test

import (
	"fmt"
	"testing"

	"github.com/kubeflow/hub/catalog/internal/testhelpers"
	modelcatalogmodels "github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	modelcatalogservice "github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/service"
	"github.com/kubeflow/hub/catalog/internal/db/models"
	"github.com/kubeflow/hub/catalog/internal/db/service"
	"github.com/kubeflow/hub/internal/platform/apiutils"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/kubeflow/hub/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test constants for custom property names and value types
const (
	testPropertyAccuracy  = "accuracy"
	testPropertyTimestamp = "timestamp"
	testPropertyVersion   = "version"
	testPropertyScore     = "score"

	testValueTypeDouble = "double_value"
	testValueTypeString = "string_value"
	testValueTypeInt    = "int_value"

	testSortOrderASC  = "ASC"
	testSortOrderDESC = "DESC"
)

func TestCatalogArtifactRepository(t *testing.T) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testhelpers.MustDatastoreSpec(t))
	defer cleanup()

	// Get the catalog artifact type IDs
	modelArtifactTypeID := getCatalogModelArtifactTypeID(t, sharedDB)
	metricsArtifactTypeID := getCatalogMetricsArtifactTypeID(t, sharedDB)

	// Create unified artifact repository with both types
	artifactTypeMap := map[string]int32{
		service.CatalogModelArtifactTypeName:   modelArtifactTypeID,
		service.CatalogMetricsArtifactTypeName: metricsArtifactTypeID,
	}
	repo := service.NewCatalogArtifactRepository(sharedDB, artifactTypeMap)

	// Also get CatalogModel type ID for creating parent entities
	catalogModelTypeID := getCatalogModelTypeID(t, sharedDB)
	catalogModelRepo := modelcatalogservice.NewCatalogModelRepository(sharedDB, catalogModelTypeID)
	modelArtifactRepo := modelcatalogservice.NewCatalogModelArtifactRepository(sharedDB, modelArtifactTypeID)
	metricsArtifactRepo := modelcatalogservice.NewCatalogMetricsArtifactRepository(sharedDB, metricsArtifactTypeID)

	// Create shared test data
	catalogModel := &modelcatalogmodels.CatalogModelImpl{
		TypeID: apiutils.Of(int32(catalogModelTypeID)),
		Attributes: &modelcatalogmodels.CatalogModelAttributes{
			Name:       apiutils.Of("test-catalog-model-for-artifacts"),
			ExternalID: apiutils.Of("catalog-model-artifacts-ext-123"),
		},
	}
	savedCatalogModel, err := catalogModelRepo.Save(catalogModel)
	require.NoError(t, err)

	t.Run("GetByID_ModelArtifact", func(t *testing.T) {
		// Create a model artifact using the specific repository
		modelArtifact := &modelcatalogmodels.CatalogModelArtifactImpl{
			TypeID: apiutils.Of(int32(modelArtifactTypeID)),
			Attributes: &modelcatalogmodels.CatalogModelArtifactAttributes{
				Name:         apiutils.Of("test-model-artifact-getbyid"),
				ExternalID:   apiutils.Of("model-art-getbyid-ext-123"),
				URI:          apiutils.Of("s3://test-bucket/model.bin"),
				ArtifactType: apiutils.Of(modelcatalogmodels.CatalogModelArtifactType),
			},
		}
		savedModelArtifact, err := modelArtifactRepo.Save(modelArtifact, savedCatalogModel.GetID())
		require.NoError(t, err)

		// Retrieve using unified repository
		retrieved, err := repo.GetByID(*savedModelArtifact.GetID())
		require.NoError(t, err)

		// Verify it's a model artifact
		assert.NotNil(t, retrieved.CatalogModelArtifact)
		assert.Nil(t, retrieved.CatalogMetricsArtifact)
		assert.Equal(t, "test-model-artifact-getbyid", *retrieved.CatalogModelArtifact.(modelcatalogmodels.CatalogModelArtifact).GetAttributes().Name)
		assert.Equal(t, "model-art-getbyid-ext-123", *retrieved.CatalogModelArtifact.(modelcatalogmodels.CatalogModelArtifact).GetAttributes().ExternalID)
		assert.Equal(t, "s3://test-bucket/model.bin", *retrieved.CatalogModelArtifact.(modelcatalogmodels.CatalogModelArtifact).GetAttributes().URI)
	})

	t.Run("GetByID_MetricsArtifact", func(t *testing.T) {
		// Create a metrics artifact using the specific repository
		metricsArtifact := &modelcatalogmodels.CatalogMetricsArtifactImpl{
			TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
			Attributes: &modelcatalogmodels.CatalogMetricsArtifactAttributes{
				Name:         apiutils.Of("test-metrics-artifact-getbyid"),
				ExternalID:   apiutils.Of("metrics-art-getbyid-ext-123"),
				MetricsType:  modelcatalogmodels.MetricsTypeAccuracy,
				ArtifactType: apiutils.Of("metrics-artifact"),
			},
		}
		savedMetricsArtifact, err := metricsArtifactRepo.Save(metricsArtifact, savedCatalogModel.GetID())
		require.NoError(t, err)

		// Retrieve using unified repository
		retrieved, err := repo.GetByID(*savedMetricsArtifact.GetID())
		require.NoError(t, err)

		// Verify it's a metrics artifact
		assert.Nil(t, retrieved.CatalogModelArtifact)
		assert.NotNil(t, retrieved.CatalogMetricsArtifact)
		assert.Equal(t, "test-metrics-artifact-getbyid", *retrieved.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().Name)
		assert.Equal(t, "metrics-art-getbyid-ext-123", *retrieved.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().ExternalID)
		assert.Equal(t, modelcatalogmodels.MetricsTypeAccuracy, retrieved.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().MetricsType)
	})

	t.Run("GetByID_NotFound", func(t *testing.T) {
		nonExistentID := int32(99999)
		_, err := repo.GetByID(nonExistentID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "catalog artifact by id not found")
	})

	t.Run("List_AllArtifacts", func(t *testing.T) {
		// Create test artifacts of both types
		modelArtifact1 := &modelcatalogmodels.CatalogModelArtifactImpl{
			TypeID: apiutils.Of(int32(modelArtifactTypeID)),
			Attributes: &modelcatalogmodels.CatalogModelArtifactAttributes{
				Name:         apiutils.Of("test-model-artifact-list-1"),
				ExternalID:   apiutils.Of("model-list-1-ext"),
				URI:          apiutils.Of("s3://test/model1.bin"),
				ArtifactType: apiutils.Of(modelcatalogmodels.CatalogModelArtifactType),
			},
		}

		modelArtifact2 := &modelcatalogmodels.CatalogModelArtifactImpl{
			TypeID: apiutils.Of(int32(modelArtifactTypeID)),
			Attributes: &modelcatalogmodels.CatalogModelArtifactAttributes{
				Name:         apiutils.Of("test-model-artifact-list-2"),
				ExternalID:   apiutils.Of("model-list-2-ext"),
				URI:          apiutils.Of("s3://test/model2.bin"),
				ArtifactType: apiutils.Of(modelcatalogmodels.CatalogModelArtifactType),
			},
		}

		metricsArtifact1 := &modelcatalogmodels.CatalogMetricsArtifactImpl{
			TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
			Attributes: &modelcatalogmodels.CatalogMetricsArtifactAttributes{
				Name:         apiutils.Of("test-metrics-artifact-list-1"),
				ExternalID:   apiutils.Of("metrics-list-1-ext"),
				MetricsType:  modelcatalogmodels.MetricsTypeAccuracy,
				ArtifactType: apiutils.Of("metrics-artifact"),
			},
		}

		// Save artifacts
		savedModelArt1, err := modelArtifactRepo.Save(modelArtifact1, savedCatalogModel.GetID())
		require.NoError(t, err)
		savedModelArt2, err := modelArtifactRepo.Save(modelArtifact2, savedCatalogModel.GetID())
		require.NoError(t, err)
		savedMetricsArt1, err := metricsArtifactRepo.Save(metricsArtifact1, savedCatalogModel.GetID())
		require.NoError(t, err)

		// List all artifacts for the parent resource
		listOptions := models.CatalogArtifactListOptions{
			ParentResourceID: savedCatalogModel.GetID(),
		}

		result, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should return all 3 artifacts (2 model + 1 metrics)
		assert.GreaterOrEqual(t, len(result.Items), 3, "Should return at least the 3 artifacts we created")

		// Verify we got both types
		var modelArtifactCount, metricsArtifactCount int
		artifactIDs := make(map[int32]bool)

		for _, artifact := range result.Items {
			if artifact.CatalogModelArtifact != nil {
				modelArtifactCount++
				artifactIDs[*artifact.CatalogModelArtifact.GetID()] = true
			} else if artifact.CatalogMetricsArtifact != nil {
				metricsArtifactCount++
				artifactIDs[*artifact.CatalogMetricsArtifact.GetID()] = true
			}
		}

		assert.GreaterOrEqual(t, modelArtifactCount, 2, "Should have at least 2 model artifacts")
		assert.GreaterOrEqual(t, metricsArtifactCount, 1, "Should have at least 1 metrics artifact")

		// Verify our specific artifacts are in the results
		assert.True(t, artifactIDs[*savedModelArt1.GetID()], "Should contain first model artifact")
		assert.True(t, artifactIDs[*savedModelArt2.GetID()], "Should contain second model artifact")
		assert.True(t, artifactIDs[*savedMetricsArt1.GetID()], "Should contain metrics artifact")
	})

	t.Run("List_FilterByArtifactType_ModelArtifact", func(t *testing.T) {
		// Filter by model artifact type only
		artifactType := "model-artifact"
		listOptions := models.CatalogArtifactListOptions{
			ParentResourceID: savedCatalogModel.GetID(),
			ArtifactType:     &artifactType,
		}

		result, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)

		// All results should be model artifacts
		for _, artifact := range result.Items {
			assert.NotNil(t, artifact.CatalogModelArtifact, "Should only return model artifacts")
			assert.Nil(t, artifact.CatalogMetricsArtifact, "Should not return metrics artifacts")
		}
	})

	t.Run("List_FilterByArtifactType_MetricsArtifact", func(t *testing.T) {
		// Filter by metrics artifact type only
		artifactType := "metrics-artifact"
		listOptions := models.CatalogArtifactListOptions{
			ParentResourceID: savedCatalogModel.GetID(),
			ArtifactType:     &artifactType,
		}

		result, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)

		// All results should be metrics artifacts
		for _, artifact := range result.Items {
			assert.Nil(t, artifact.CatalogModelArtifact, "Should not return model artifacts")
			assert.NotNil(t, artifact.CatalogMetricsArtifact, "Should only return metrics artifacts")
		}
	})

	t.Run("List_FilterByExternalID", func(t *testing.T) {
		// Create artifact with specific external ID for filtering
		testArtifact := &modelcatalogmodels.CatalogMetricsArtifactImpl{
			TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
			Attributes: &modelcatalogmodels.CatalogMetricsArtifactAttributes{
				Name:         apiutils.Of("external-id-filter-test"),
				ExternalID:   apiutils.Of("unique-external-id-123"),
				MetricsType:  modelcatalogmodels.MetricsTypePerformance,
				ArtifactType: apiutils.Of("metrics-artifact"),
			},
		}
		savedArtifact, err := metricsArtifactRepo.Save(testArtifact, savedCatalogModel.GetID())
		require.NoError(t, err)

		// Filter by external ID
		externalID := "unique-external-id-123"
		listOptions := models.CatalogArtifactListOptions{
			ExternalID: &externalID,
		}

		result, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Items, 1, "Should find exactly one artifact with the external ID")

		// Verify it's the correct artifact
		artifact := result.Items[0]
		assert.NotNil(t, artifact.CatalogMetricsArtifact)
		assert.Equal(t, *savedArtifact.GetID(), *artifact.CatalogMetricsArtifact.GetID())
		assert.Equal(t, "unique-external-id-123", *artifact.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().ExternalID)
	})

	t.Run("List_WithPagination", func(t *testing.T) {
		// Create multiple artifacts for pagination testing
		for i := range 5 {
			artifact := &modelcatalogmodels.CatalogModelArtifactImpl{
				TypeID: apiutils.Of(int32(modelArtifactTypeID)),
				Attributes: &modelcatalogmodels.CatalogModelArtifactAttributes{
					Name:         apiutils.Of(fmt.Sprintf("pagination-test-%d", i)),
					ExternalID:   apiutils.Of(fmt.Sprintf("pagination-ext-%d", i)),
					URI:          apiutils.Of(fmt.Sprintf("s3://test/pagination-%d.bin", i)),
					ArtifactType: apiutils.Of(modelcatalogmodels.CatalogModelArtifactType),
				},
			}
			_, err := modelArtifactRepo.Save(artifact, savedCatalogModel.GetID())
			require.NoError(t, err)
		}

		// Test pagination
		pageSize := int32(3)
		listOptions := models.CatalogArtifactListOptions{
			ParentResourceID: savedCatalogModel.GetID(),
			Pagination: dbmodels.Pagination{
				PageSize: &pageSize,
				OrderBy:  apiutils.Of("ID"),
			},
		}

		result, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.LessOrEqual(t, len(result.Items), 3, "Should respect page size limit")
		assert.GreaterOrEqual(t, len(result.Items), 1, "Should return at least one item")
	})

	t.Run("List_InvalidArtifactType", func(t *testing.T) {
		// Test with invalid artifact type
		invalidType := "invalid-artifact-type"
		listOptions := models.CatalogArtifactListOptions{
			ParentResourceID: savedCatalogModel.GetID(),
			ArtifactType:     &invalidType,
		}

		_, err := repo.List(listOptions)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid catalog artifact type")
		assert.Contains(t, err.Error(), "invalid-artifact-type")
	})

	t.Run("List_WithCustomProperties", func(t *testing.T) {
		// Create artifacts with custom properties
		customProps := []dbmodels.Properties{
			{
				Name:        "custom_prop_1",
				StringValue: apiutils.Of("custom_value_1"),
			},
			{
				Name:        "custom_prop_2",
				StringValue: apiutils.Of("custom_value_2"),
			},
		}

		artifactWithCustomProps := &modelcatalogmodels.CatalogModelArtifactImpl{
			TypeID: apiutils.Of(int32(modelArtifactTypeID)),
			Attributes: &modelcatalogmodels.CatalogModelArtifactAttributes{
				Name:         apiutils.Of("artifact-with-custom-props"),
				ExternalID:   apiutils.Of("custom-props-ext"),
				URI:          apiutils.Of("s3://test/custom-props.bin"),
				ArtifactType: apiutils.Of(modelcatalogmodels.CatalogModelArtifactType),
			},
			CustomProperties: &customProps,
		}

		savedArtifact, err := modelArtifactRepo.Save(artifactWithCustomProps, savedCatalogModel.GetID())
		require.NoError(t, err)

		// Retrieve using unified repository
		retrieved, err := repo.GetByID(*savedArtifact.GetID())
		require.NoError(t, err)

		// Verify custom properties are preserved
		require.NotNil(t, retrieved.CatalogModelArtifact)
		assert.NotNil(t, retrieved.CatalogModelArtifact.GetCustomProperties())

		customPropsMap := make(map[string]string)
		for _, prop := range *retrieved.CatalogModelArtifact.GetCustomProperties() {
			if prop.StringValue != nil {
				customPropsMap[prop.Name] = *prop.StringValue
			}
		}

		assert.Equal(t, "custom_value_1", customPropsMap["custom_prop_1"])
		assert.Equal(t, "custom_value_2", customPropsMap["custom_prop_2"])
	})

	t.Run("MappingErrors", func(t *testing.T) {
		// Test error handling for invalid type mapping
		// This would typically happen if there's data inconsistency in the database

		// We can't easily test this without directly manipulating the database
		// but we can test the GetByID with an artifact that has an unknown type
		// by temporarily modifying the repository's type mapping

		// Create a repository with incomplete type mapping
		incompleteTypeMap := map[string]int32{
			service.CatalogModelArtifactTypeName: modelArtifactTypeID,
			// Missing CatalogMetricsArtifactTypeName intentionally
		}
		incompleteRepo := service.NewCatalogArtifactRepository(sharedDB, incompleteTypeMap)

		// Create a metrics artifact first using the complete repo
		metricsArtifact := &modelcatalogmodels.CatalogMetricsArtifactImpl{
			TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
			Attributes: &modelcatalogmodels.CatalogMetricsArtifactAttributes{
				Name:         apiutils.Of("test-mapping-error"),
				ExternalID:   apiutils.Of("mapping-error-ext"),
				MetricsType:  modelcatalogmodels.MetricsTypeAccuracy,
				ArtifactType: apiutils.Of("metrics-artifact"),
			},
		}
		savedMetricsArtifact, err := metricsArtifactRepo.Save(metricsArtifact, savedCatalogModel.GetID())
		require.NoError(t, err)

		// Try to retrieve using incomplete repo - should get mapping error
		_, err = incompleteRepo.GetByID(*savedMetricsArtifact.GetID())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no mapper registered for artifact type")
	})

	t.Run("TestNameOrdering", func(t *testing.T) {
		// Create a new model for this test
		testModel := &modelcatalogmodels.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &modelcatalogmodels.CatalogModelAttributes{
				Name:       apiutils.Of("test-model-for-name-ordering"),
				ExternalID: apiutils.Of("test-model-name-ordering-ext"),
			},
		}
		savedTestModel, err := catalogModelRepo.Save(testModel)
		require.NoError(t, err)

		// Create artifacts with various names (including null)
		testArtifacts := []struct {
			name *string
			desc string
		}{
			{apiutils.Of("zebra-artifact"), "zebra"},
			{apiutils.Of("alpha-artifact"), "alpha"},
			{apiutils.Of("beta-artifact"), "beta"},
			{nil, "null-name"}, // Artifact with no name (like real model artifacts)
			{apiutils.Of("gamma-artifact"), "gamma"},
		}

		for _, artifact := range testArtifacts {
			metricsArtifact := &modelcatalogmodels.CatalogMetricsArtifactImpl{
				TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
				Attributes: &modelcatalogmodels.CatalogMetricsArtifactAttributes{
					Name:         artifact.name,
					ExternalID:   apiutils.Of(fmt.Sprintf("name-test-%s", artifact.desc)),
					MetricsType:  modelcatalogmodels.MetricsTypePerformance,
					ArtifactType: apiutils.Of("metrics-artifact"),
				},
			}
			_, err := metricsArtifactRepo.Save(metricsArtifact, savedTestModel.GetID())
			require.NoError(t, err)
		}

		// Test NAME ordering ASC
		listOptions := models.CatalogArtifactListOptions{
			ParentResourceID: savedTestModel.GetID(),
			Pagination: dbmodels.Pagination{
				OrderBy:   apiutils.Of("NAME"),
				SortOrder: apiutils.Of("ASC"),
			},
		}
		result, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Extract artifact names (including nulls)
		var foundArtifacts []struct {
			name *string
			desc string
		}
		for _, artifact := range result.Items {
			if artifact.CatalogMetricsArtifact != nil {
				name := artifact.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().Name
				foundArtifacts = append(foundArtifacts, struct {
					name *string
					desc string
				}{name, fmt.Sprintf("%v", name)})
			}
		}

		require.GreaterOrEqual(t, len(foundArtifacts), 5, "Should find all test artifacts")

		// Find positions of named artifacts
		var alphaIdx, betaIdx, gammaIdx, zebraIdx, nullIdx int = -1, -1, -1, -1, -1
		for i, artifact := range foundArtifacts {
			if artifact.name != nil {
				switch *artifact.name {
				case "alpha-artifact":
					alphaIdx = i
				case "beta-artifact":
					betaIdx = i
				case "gamma-artifact":
					gammaIdx = i
				case "zebra-artifact":
					zebraIdx = i
				}
			} else {
				nullIdx = i
			}
		}

		// Verify ASC ordering: alpha < beta < gamma < zebra, and null at the end
		require.NotEqual(t, -1, alphaIdx, "alpha-artifact not found")
		require.NotEqual(t, -1, betaIdx, "beta-artifact not found")
		require.NotEqual(t, -1, gammaIdx, "gamma-artifact not found")
		require.NotEqual(t, -1, zebraIdx, "zebra-artifact not found")
		require.NotEqual(t, -1, nullIdx, "null-name artifact not found")

		assert.Less(t, alphaIdx, betaIdx, "alpha should come before beta in ASC")
		assert.Less(t, betaIdx, gammaIdx, "beta should come before gamma in ASC")
		assert.Less(t, gammaIdx, zebraIdx, "gamma should come before zebra in ASC")
		assert.Less(t, zebraIdx, nullIdx, "named artifacts should come before null in ASC")

		// Test NAME ordering DESC
		listOptions = models.CatalogArtifactListOptions{
			ParentResourceID: savedTestModel.GetID(),
			Pagination: dbmodels.Pagination{
				OrderBy:   apiutils.Of("NAME"),
				SortOrder: apiutils.Of("DESC"),
			},
		}
		result, err = repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Extract artifact names from DESC results
		foundArtifacts = []struct {
			name *string
			desc string
		}{}
		for _, artifact := range result.Items {
			if artifact.CatalogMetricsArtifact != nil {
				name := artifact.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().Name
				foundArtifacts = append(foundArtifacts, struct {
					name *string
					desc string
				}{name, fmt.Sprintf("%v", name)})
			}
		}

		// Find positions in DESC order
		alphaIdx, betaIdx, gammaIdx, zebraIdx, nullIdx = -1, -1, -1, -1, -1
		for i, artifact := range foundArtifacts {
			if artifact.name != nil {
				switch *artifact.name {
				case "alpha-artifact":
					alphaIdx = i
				case "beta-artifact":
					betaIdx = i
				case "gamma-artifact":
					gammaIdx = i
				case "zebra-artifact":
					zebraIdx = i
				}
			} else {
				nullIdx = i
			}
		}

		// Verify DESC ordering: In SQL DESC, NULL comes first, then zebra > gamma > beta > alpha
		assert.Less(t, nullIdx, zebraIdx, "null should come first in DESC (SQL default behavior)")
		assert.Less(t, zebraIdx, gammaIdx, "zebra should come before gamma in DESC")
		assert.Less(t, gammaIdx, betaIdx, "gamma should come before beta in DESC")
		assert.Less(t, betaIdx, alphaIdx, "beta should come before alpha in DESC")
	})

	t.Run("TestNameOrderingPagination", func(t *testing.T) {
		// Create a new model for this test
		testModel := &modelcatalogmodels.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &modelcatalogmodels.CatalogModelAttributes{
				Name:       apiutils.Of("test-model-for-name-pagination"),
				ExternalID: apiutils.Of("test-model-name-pagination-ext"),
			},
		}
		savedTestModel, err := catalogModelRepo.Save(testModel)
		require.NoError(t, err)

		// Create artifacts with sequential names for pagination testing
		artifactNames := []string{
			"artifact-alpha",
			"artifact-beta",
			"artifact-gamma",
			"artifact-delta",
			"artifact-epsilon",
		}

		for i, name := range artifactNames {
			metricsArtifact := &modelcatalogmodels.CatalogMetricsArtifactImpl{
				TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
				Attributes: &modelcatalogmodels.CatalogMetricsArtifactAttributes{
					Name:         apiutils.Of(name),
					ExternalID:   apiutils.Of(fmt.Sprintf("pagination-test-%d", i)),
					MetricsType:  modelcatalogmodels.MetricsTypePerformance,
					ArtifactType: apiutils.Of("metrics-artifact"),
				},
			}
			_, err := metricsArtifactRepo.Save(metricsArtifact, savedTestModel.GetID())
			require.NoError(t, err)
		}

		// Test pagination with NAME ordering (ASC)
		pageSize := int32(2)
		listOptions := models.CatalogArtifactListOptions{
			ParentResourceID: savedTestModel.GetID(),
			Pagination: dbmodels.Pagination{
				OrderBy:   apiutils.Of("NAME"),
				SortOrder: apiutils.Of("ASC"),
				PageSize:  &pageSize,
			},
		}

		// First page
		result, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Filter to only our test artifacts
		var page1Artifacts []string
		for _, artifact := range result.Items {
			if artifact.CatalogMetricsArtifact != nil && artifact.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().Name != nil {
				name := *artifact.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().Name
				if name == "artifact-alpha" || name == "artifact-beta" || name == "artifact-gamma" || name == "artifact-delta" || name == "artifact-epsilon" {
					page1Artifacts = append(page1Artifacts, name)
				}
			}
		}

		require.LessOrEqual(t, len(page1Artifacts), 2, "First page should have at most 2 artifacts")
		require.GreaterOrEqual(t, len(page1Artifacts), 1, "First page should have at least 1 artifact")
		assert.NotNil(t, result.NextPageToken, "Should have next page token")

		// Verify first page ordering
		if len(page1Artifacts) >= 2 {
			assert.Less(t, page1Artifacts[0], page1Artifacts[1], "First page should be ordered")
		}

		// Second page
		listOptions.Pagination.NextPageToken = &result.NextPageToken
		result2, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result2)

		var page2Artifacts []string
		for _, artifact := range result2.Items {
			if artifact.CatalogMetricsArtifact != nil && artifact.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().Name != nil {
				name := *artifact.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().Name
				if name == "artifact-alpha" || name == "artifact-beta" || name == "artifact-gamma" || name == "artifact-delta" || name == "artifact-epsilon" {
					page2Artifacts = append(page2Artifacts, name)
				}
			}
		}

		require.GreaterOrEqual(t, len(page2Artifacts), 1, "Second page should have at least 1 artifact")

		// Verify second page ordering
		if len(page2Artifacts) >= 2 {
			assert.Less(t, page2Artifacts[0], page2Artifacts[1], "Second page should be ordered")
		}

		// Verify no overlap between pages
		for _, name1 := range page1Artifacts {
			for _, name2 := range page2Artifacts {
				assert.NotEqual(t, name1, name2, "Pages should not have overlapping artifacts")
			}
		}

		// Verify page 2 comes after page 1
		if len(page1Artifacts) > 0 && len(page2Artifacts) > 0 {
			assert.Less(t, page1Artifacts[len(page1Artifacts)-1], page2Artifacts[0], "Page 2 should continue where page 1 ended")
		}

		// Test DESC pagination
		listOptions = models.CatalogArtifactListOptions{
			ParentResourceID: savedTestModel.GetID(),
			Pagination: dbmodels.Pagination{
				OrderBy:   apiutils.Of("NAME"),
				SortOrder: apiutils.Of("DESC"),
				PageSize:  &pageSize,
			},
		}

		resultDesc, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, resultDesc)

		var pageDescArtifacts []string
		expectedNames := map[string]bool{
			"artifact-alpha":   true,
			"artifact-beta":    true,
			"artifact-gamma":   true,
			"artifact-delta":   true,
			"artifact-epsilon": true,
		}
		for _, artifact := range resultDesc.Items {
			if artifact.CatalogMetricsArtifact != nil &&
				artifact.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().Name != nil {
				name := *artifact.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().Name
				if expectedNames[name] {
					pageDescArtifacts = append(pageDescArtifacts, name)
				}
			}
		}

		require.GreaterOrEqual(t, len(pageDescArtifacts), 1, "DESC first page should have at least 1 artifact")

		// Verify DESC ordering
		if len(pageDescArtifacts) >= 2 {
			assert.Greater(t, pageDescArtifacts[0], pageDescArtifacts[1], "DESC page should be reverse ordered")
		}
	})

	t.Run("TestCustomPropertyOrdering_DoubleValue", func(t *testing.T) {
		// Create a new model for this test
		testModel := &modelcatalogmodels.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &modelcatalogmodels.CatalogModelAttributes{
				Name:       apiutils.Of("test-model-custom-property-ordering"),
				ExternalID: apiutils.Of("test-model-custom-property-ext"),
			},
		}
		savedTestModel, err := catalogModelRepo.Save(testModel)
		require.NoError(t, err)

		// Create artifacts with custom properties (accuracy as double_value)
		testArtifacts := []struct {
			name     string
			accuracy float64
		}{
			{"artifact-high-accuracy", 0.95},
			{"artifact-low-accuracy", 0.75},
			{"artifact-medium-accuracy", 0.85},
			{"artifact-perfect-accuracy", 0.99},
			{"artifact-poor-accuracy", 0.60},
		}

		for _, tc := range testArtifacts {
			customProps := []dbmodels.Properties{
				{
					Name:        testPropertyAccuracy,
					DoubleValue: apiutils.Of(tc.accuracy),
				},
			}

			metricsArtifact := &modelcatalogmodels.CatalogMetricsArtifactImpl{
				TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
				Attributes: &modelcatalogmodels.CatalogMetricsArtifactAttributes{
					Name:         apiutils.Of(tc.name),
					ExternalID:   apiutils.Of(fmt.Sprintf("custom-prop-test-%s", tc.name)),
					MetricsType:  modelcatalogmodels.MetricsTypeAccuracy,
					ArtifactType: apiutils.Of("metrics-artifact"),
				},
				CustomProperties: &customProps,
			}
			_, err := metricsArtifactRepo.Save(metricsArtifact, savedTestModel.GetID())
			require.NoError(t, err)
		}

		// Test ordering by accuracy.double_value ASC
		listOptions := models.CatalogArtifactListOptions{
			ParentResourceID: savedTestModel.GetID(),
			Pagination: dbmodels.Pagination{
				OrderBy: apiutils.Of(
					fmt.Sprintf("%s.%s", testPropertyAccuracy, testValueTypeDouble),
				),
				SortOrder: apiutils.Of(testSortOrderASC),
			},
		}
		result, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Extract artifacts with accuracy property
		var foundArtifacts []struct {
			name     string
			accuracy float64
		}
		expectedArtifactNames := map[string]bool{
			"artifact-high-accuracy":    true,
			"artifact-low-accuracy":     true,
			"artifact-medium-accuracy":  true,
			"artifact-perfect-accuracy": true,
			"artifact-poor-accuracy":    true,
		}
		for _, artifact := range result.Items {
			if artifact.CatalogMetricsArtifact != nil {
				name := artifact.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().Name
				if name != nil && expectedArtifactNames[*name] {
					// Get accuracy from custom properties
					if artifact.CatalogMetricsArtifact.GetCustomProperties() != nil {
						for _, prop := range *artifact.CatalogMetricsArtifact.GetCustomProperties() {
							if prop.Name == testPropertyAccuracy && prop.DoubleValue != nil {
								foundArtifacts = append(foundArtifacts, struct {
									name     string
									accuracy float64
								}{*name, *prop.DoubleValue})
							}
						}
					}
				}
			}
		}

		require.GreaterOrEqual(t, len(foundArtifacts), 5, "Should find all test artifacts")

		// Verify ASC ordering by accuracy
		for i := 0; i < len(foundArtifacts)-1; i++ {
			assert.LessOrEqual(t, foundArtifacts[i].accuracy, foundArtifacts[i+1].accuracy,
				fmt.Sprintf("Artifact %s (%.2f) should come before or equal to %s (%.2f) in ASC order",
					foundArtifacts[i].name, foundArtifacts[i].accuracy,
					foundArtifacts[i+1].name, foundArtifacts[i+1].accuracy))
		}

		// Test ordering by accuracy.double_value DESC
		listOptions = models.CatalogArtifactListOptions{
			ParentResourceID: savedTestModel.GetID(),
			Pagination: dbmodels.Pagination{
				OrderBy: apiutils.Of(
					fmt.Sprintf("%s.%s", testPropertyAccuracy, testValueTypeDouble),
				),
				SortOrder: apiutils.Of(testSortOrderDESC),
			},
		}
		result, err = repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Extract artifacts again
		foundArtifacts = []struct {
			name     string
			accuracy float64
		}{}
		for _, artifact := range result.Items {
			if artifact.CatalogMetricsArtifact != nil {
				name := artifact.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().Name
				if name != nil && expectedArtifactNames[*name] {
					// Get accuracy from custom properties
					if artifact.CatalogMetricsArtifact.GetCustomProperties() != nil {
						for _, prop := range *artifact.CatalogMetricsArtifact.GetCustomProperties() {
							if prop.Name == testPropertyAccuracy && prop.DoubleValue != nil {
								foundArtifacts = append(foundArtifacts, struct {
									name     string
									accuracy float64
								}{*name, *prop.DoubleValue})
							}
						}
					}
				}
			}
		}

		require.GreaterOrEqual(t, len(foundArtifacts), 5, "Should find all test artifacts")

		// Verify DESC ordering by accuracy
		for i := 0; i < len(foundArtifacts)-1; i++ {
			assert.GreaterOrEqual(t, foundArtifacts[i].accuracy, foundArtifacts[i+1].accuracy,
				fmt.Sprintf("Artifact %s (%.2f) should come after or equal to %s (%.2f) in DESC order",
					foundArtifacts[i].name, foundArtifacts[i].accuracy,
					foundArtifacts[i+1].name, foundArtifacts[i+1].accuracy))
		}
	})

	t.Run("TestCustomPropertyOrdering_StringValue", func(t *testing.T) {
		// Create a new model for this test
		testModel := &modelcatalogmodels.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &modelcatalogmodels.CatalogModelAttributes{
				Name:       apiutils.Of("test-model-string-property-ordering"),
				ExternalID: apiutils.Of("test-model-string-property-ext"),
			},
		}
		savedTestModel, err := catalogModelRepo.Save(testModel)
		require.NoError(t, err)

		// Create artifacts with custom properties (timestamp as string_value)
		testArtifacts := []struct {
			name      string
			timestamp string
		}{
			{"artifact-2024-01-15", "2024-01-15"},
			{"artifact-2024-01-10", "2024-01-10"},
			{"artifact-2024-01-20", "2024-01-20"},
			{"artifact-2024-01-05", "2024-01-05"},
			{"artifact-2024-01-25", "2024-01-25"},
		}

		for _, tc := range testArtifacts {
			customProps := []dbmodels.Properties{
				{
					Name:        testPropertyTimestamp,
					StringValue: apiutils.Of(tc.timestamp),
				},
			}

			metricsArtifact := &modelcatalogmodels.CatalogMetricsArtifactImpl{
				TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
				Attributes: &modelcatalogmodels.CatalogMetricsArtifactAttributes{
					Name:         apiutils.Of(tc.name),
					ExternalID:   apiutils.Of(fmt.Sprintf("string-prop-test-%s", tc.name)),
					MetricsType:  modelcatalogmodels.MetricsTypePerformance,
					ArtifactType: apiutils.Of("metrics-artifact"),
				},
				CustomProperties: &customProps,
			}
			_, err := metricsArtifactRepo.Save(metricsArtifact, savedTestModel.GetID())
			require.NoError(t, err)
		}

		// Test ordering by timestamp.string_value ASC
		listOptions := models.CatalogArtifactListOptions{
			ParentResourceID: savedTestModel.GetID(),
			Pagination: dbmodels.Pagination{
				OrderBy: apiutils.Of(
					fmt.Sprintf("%s.%s", testPropertyTimestamp, testValueTypeString),
				),
				SortOrder: apiutils.Of(testSortOrderASC),
			},
		}
		result, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Extract artifacts with timestamp property
		var foundArtifacts []struct {
			name      string
			timestamp string
		}
		for _, artifact := range result.Items {
			if artifact.CatalogMetricsArtifact != nil {
				name := artifact.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().Name
				if name != nil && (*name == "artifact-2024-01-15" || *name == "artifact-2024-01-10" || *name == "artifact-2024-01-20" || *name == "artifact-2024-01-05" || *name == "artifact-2024-01-25") {
					// Get timestamp from custom properties
					if artifact.CatalogMetricsArtifact.GetCustomProperties() != nil {
						for _, prop := range *artifact.CatalogMetricsArtifact.GetCustomProperties() {
							if prop.Name == "timestamp" && prop.StringValue != nil {
								foundArtifacts = append(foundArtifacts, struct {
									name      string
									timestamp string
								}{*name, *prop.StringValue})
							}
						}
					}
				}
			}
		}

		require.GreaterOrEqual(t, len(foundArtifacts), 5, "Should find all test artifacts")

		// Verify ASC ordering by timestamp
		for i := 0; i < len(foundArtifacts)-1; i++ {
			assert.LessOrEqual(t, foundArtifacts[i].timestamp, foundArtifacts[i+1].timestamp,
				fmt.Sprintf("Artifact %s (%s) should come before or equal to %s (%s) in ASC order",
					foundArtifacts[i].name, foundArtifacts[i].timestamp,
					foundArtifacts[i+1].name, foundArtifacts[i+1].timestamp))
		}

		// Test ordering by timestamp.string_value DESC
		listOptions = models.CatalogArtifactListOptions{
			ParentResourceID: savedTestModel.GetID(),
			Pagination: dbmodels.Pagination{
				OrderBy:   apiutils.Of("timestamp.string_value"),
				SortOrder: apiutils.Of("DESC"),
			},
		}
		result, err = repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Extract artifacts again
		foundArtifacts = []struct {
			name      string
			timestamp string
		}{}
		for _, artifact := range result.Items {
			if artifact.CatalogMetricsArtifact != nil {
				name := artifact.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().Name
				if name != nil && (*name == "artifact-2024-01-15" || *name == "artifact-2024-01-10" || *name == "artifact-2024-01-20" || *name == "artifact-2024-01-05" || *name == "artifact-2024-01-25") {
					// Get timestamp from custom properties
					if artifact.CatalogMetricsArtifact.GetCustomProperties() != nil {
						for _, prop := range *artifact.CatalogMetricsArtifact.GetCustomProperties() {
							if prop.Name == "timestamp" && prop.StringValue != nil {
								foundArtifacts = append(foundArtifacts, struct {
									name      string
									timestamp string
								}{*name, *prop.StringValue})
							}
						}
					}
				}
			}
		}

		require.GreaterOrEqual(t, len(foundArtifacts), 5, "Should find all test artifacts")

		// Verify DESC ordering by timestamp
		for i := 0; i < len(foundArtifacts)-1; i++ {
			assert.GreaterOrEqual(t, foundArtifacts[i].timestamp, foundArtifacts[i+1].timestamp,
				fmt.Sprintf("Artifact %s (%s) should come after or equal to %s (%s) in DESC order",
					foundArtifacts[i].name, foundArtifacts[i].timestamp,
					foundArtifacts[i+1].name, foundArtifacts[i+1].timestamp))
		}
	})

	t.Run("TestCustomPropertyOrdering_IntValue", func(t *testing.T) {
		// Create a new model for this test
		testModel := &modelcatalogmodels.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &modelcatalogmodels.CatalogModelAttributes{
				Name:       apiutils.Of("test-model-int-property-ordering"),
				ExternalID: apiutils.Of("test-model-int-property-ext"),
			},
		}
		savedTestModel, err := catalogModelRepo.Save(testModel)
		require.NoError(t, err)

		// Create artifacts with custom properties (version as int_value)
		testArtifacts := []struct {
			name    string
			version int32
		}{
			{"artifact-v3", 3},
			{"artifact-v1", 1},
			{"artifact-v5", 5},
			{"artifact-v2", 2},
			{"artifact-v4", 4},
		}

		for _, tc := range testArtifacts {
			customProps := []dbmodels.Properties{
				{
					Name:     "version",
					IntValue: apiutils.Of(tc.version),
				},
			}

			metricsArtifact := &modelcatalogmodels.CatalogMetricsArtifactImpl{
				TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
				Attributes: &modelcatalogmodels.CatalogMetricsArtifactAttributes{
					Name:         apiutils.Of(tc.name),
					ExternalID:   apiutils.Of(fmt.Sprintf("int-prop-test-%s", tc.name)),
					MetricsType:  modelcatalogmodels.MetricsTypePerformance,
					ArtifactType: apiutils.Of("metrics-artifact"),
				},
				CustomProperties: &customProps,
			}
			_, err := metricsArtifactRepo.Save(metricsArtifact, savedTestModel.GetID())
			require.NoError(t, err)
		}

		// Test ordering by version.int_value ASC
		listOptions := models.CatalogArtifactListOptions{
			ParentResourceID: savedTestModel.GetID(),
			Pagination: dbmodels.Pagination{
				OrderBy:   apiutils.Of("version.int_value"),
				SortOrder: apiutils.Of("ASC"),
			},
		}
		result, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Extract artifacts with version property
		var foundArtifacts []struct {
			name    string
			version int32
		}
		for _, artifact := range result.Items {
			if artifact.CatalogMetricsArtifact != nil {
				name := artifact.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().Name
				if name != nil && (*name == "artifact-v3" || *name == "artifact-v1" || *name == "artifact-v5" || *name == "artifact-v2" || *name == "artifact-v4") {
					// Get version from custom properties
					if artifact.CatalogMetricsArtifact.GetCustomProperties() != nil {
						for _, prop := range *artifact.CatalogMetricsArtifact.GetCustomProperties() {
							if prop.Name == "version" && prop.IntValue != nil {
								foundArtifacts = append(foundArtifacts, struct {
									name    string
									version int32
								}{*name, *prop.IntValue})
							}
						}
					}
				}
			}
		}

		require.GreaterOrEqual(t, len(foundArtifacts), 5, "Should find all test artifacts")

		// Verify ASC ordering by version
		for i := 0; i < len(foundArtifacts)-1; i++ {
			assert.LessOrEqual(t, foundArtifacts[i].version, foundArtifacts[i+1].version,
				fmt.Sprintf("Artifact %s (%d) should come before or equal to %s (%d) in ASC order",
					foundArtifacts[i].name, foundArtifacts[i].version,
					foundArtifacts[i+1].name, foundArtifacts[i+1].version))
		}

		// Test ordering by version.int_value DESC
		listOptions = models.CatalogArtifactListOptions{
			ParentResourceID: savedTestModel.GetID(),
			Pagination: dbmodels.Pagination{
				OrderBy:   apiutils.Of("version.int_value"),
				SortOrder: apiutils.Of("DESC"),
			},
		}
		result, err = repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Extract artifacts again
		foundArtifacts = []struct {
			name    string
			version int32
		}{}
		for _, artifact := range result.Items {
			if artifact.CatalogMetricsArtifact != nil {
				name := artifact.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().Name
				if name != nil && (*name == "artifact-v3" || *name == "artifact-v1" || *name == "artifact-v5" || *name == "artifact-v2" || *name == "artifact-v4") {
					// Get version from custom properties
					if artifact.CatalogMetricsArtifact.GetCustomProperties() != nil {
						for _, prop := range *artifact.CatalogMetricsArtifact.GetCustomProperties() {
							if prop.Name == "version" && prop.IntValue != nil {
								foundArtifacts = append(foundArtifacts, struct {
									name    string
									version int32
								}{*name, *prop.IntValue})
							}
						}
					}
				}
			}
		}

		require.GreaterOrEqual(t, len(foundArtifacts), 5, "Should find all test artifacts")

		// Verify DESC ordering by version
		for i := 0; i < len(foundArtifacts)-1; i++ {
			assert.GreaterOrEqual(t, foundArtifacts[i].version, foundArtifacts[i+1].version,
				fmt.Sprintf("Artifact %s (%d) should come after or equal to %s (%d) in DESC order",
					foundArtifacts[i].name, foundArtifacts[i].version,
					foundArtifacts[i+1].name, foundArtifacts[i+1].version))
		}
	})

	t.Run("TestCustomPropertyOrderingWithPagination", func(t *testing.T) {
		// Create a new model for this test
		testModel := &modelcatalogmodels.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &modelcatalogmodels.CatalogModelAttributes{
				Name:       apiutils.Of("test-model-custom-pagination-unique"),
				ExternalID: apiutils.Of("test-model-custom-pagination-unique-ext"),
			},
		}
		savedTestModel, err := catalogModelRepo.Save(testModel)
		require.NoError(t, err)

		// Create artifacts with custom properties for pagination testing
		for i := 1; i <= 10; i++ {
			customProps := []dbmodels.Properties{
				{
					Name:        "score",
					DoubleValue: apiutils.Of(float64(i) * 10.0),
				},
			}

			metricsArtifact := &modelcatalogmodels.CatalogMetricsArtifactImpl{
				TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
				Attributes: &modelcatalogmodels.CatalogMetricsArtifactAttributes{
					Name:         apiutils.Of(fmt.Sprintf("pagination-artifact-unique-%d", i)),
					ExternalID:   apiutils.Of(fmt.Sprintf("pagination-unique-ext-%d", i)),
					MetricsType:  modelcatalogmodels.MetricsTypeAccuracy,
					ArtifactType: apiutils.Of("metrics-artifact"),
				},
				CustomProperties: &customProps,
			}
			_, err := metricsArtifactRepo.Save(metricsArtifact, savedTestModel.GetID())
			require.NoError(t, err)
		}

		// Test pagination with custom property ordering
		pageSize := int32(3)
		listOptions := models.CatalogArtifactListOptions{
			ParentResourceID: savedTestModel.GetID(),
			Pagination: dbmodels.Pagination{
				OrderBy:   apiutils.Of("score.double_value"),
				SortOrder: apiutils.Of("ASC"),
				PageSize:  &pageSize,
			},
		}

		// First page
		result, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Filter to only our test artifacts
		var page1Scores []float64
		for _, artifact := range result.Items {
			if artifact.CatalogMetricsArtifact != nil && artifact.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().Name != nil {
				name := *artifact.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().Name
				if len(name) > 27 && name[:27] == "pagination-artifact-unique-" {
					if artifact.CatalogMetricsArtifact.GetCustomProperties() != nil {
						for _, prop := range *artifact.CatalogMetricsArtifact.GetCustomProperties() {
							if prop.Name == "score" && prop.DoubleValue != nil {
								page1Scores = append(page1Scores, *prop.DoubleValue)
							}
						}
					}
				}
			}
		}

		require.LessOrEqual(t, len(page1Scores), 3, "First page should have at most 3 artifacts")
		require.GreaterOrEqual(t, len(page1Scores), 1, "First page should have at least 1 artifact")

		// Verify first page is ordered
		for i := 0; i < len(page1Scores)-1; i++ {
			assert.LessOrEqual(t, page1Scores[i], page1Scores[i+1], "First page should be ordered")
		}

		if result.NextPageToken != "" {
			// Second page
			listOptions.Pagination.NextPageToken = &result.NextPageToken
			result2, err := repo.List(listOptions)
			require.NoError(t, err)
			require.NotNil(t, result2)

			var page2Scores []float64
			for _, artifact := range result2.Items {
				if artifact.CatalogMetricsArtifact != nil && artifact.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().Name != nil {
					name := *artifact.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().Name
					if len(name) > 27 && name[:27] == "pagination-artifact-unique-" {
						if artifact.CatalogMetricsArtifact.GetCustomProperties() != nil {
							for _, prop := range *artifact.CatalogMetricsArtifact.GetCustomProperties() {
								if prop.Name == "score" && prop.DoubleValue != nil {
									page2Scores = append(page2Scores, *prop.DoubleValue)
								}
							}
						}
					}
				}
			}

			require.GreaterOrEqual(t, len(page2Scores), 1, "Second page should have at least 1 artifact")

			// Verify second page is ordered
			for i := 0; i < len(page2Scores)-1; i++ {
				assert.LessOrEqual(t, page2Scores[i], page2Scores[i+1], "Second page should be ordered")
			}

			// Verify page 2 comes after page 1
			if len(page1Scores) > 0 && len(page2Scores) > 0 {
				assert.Less(t, page1Scores[len(page1Scores)-1], page2Scores[0], "Page 2 should continue where page 1 ended")
			}

			// Verify no overlap between pages
			for _, score1 := range page1Scores {
				for _, score2 := range page2Scores {
					assert.NotEqual(t, score1, score2, "Pages should not have overlapping scores")
				}
			}
		}
	})

	t.Run("TestEmptyPropertyName_Error", func(t *testing.T) {
		// Create a new model for this test
		testModel := &modelcatalogmodels.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &modelcatalogmodels.CatalogModelAttributes{
				Name:       apiutils.Of("test-model-invalid-property-name"),
				ExternalID: apiutils.Of("test-model-invalid-property-name-ext"),
			},
		}
		savedTestModel, err := catalogModelRepo.Save(testModel)
		require.NoError(t, err)

		// Create an artifact
		metricsArtifact := &modelcatalogmodels.CatalogMetricsArtifactImpl{
			TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
			Attributes: &modelcatalogmodels.CatalogMetricsArtifactAttributes{
				Name:         apiutils.Of("test-artifact"),
				ExternalID:   apiutils.Of("test-artifact-ext"),
				MetricsType:  modelcatalogmodels.MetricsTypeAccuracy,
				ArtifactType: apiutils.Of("metrics-artifact"),
			},
		}
		_, err = metricsArtifactRepo.Save(metricsArtifact, savedTestModel.GetID())
		require.NoError(t, err)

		// Test with empty property name - should return error
		testCases := []struct {
			name        string
			orderBy     string
			expectedErr string
		}{
			{
				name:        "Empty property name",
				orderBy:     fmt.Sprintf(".%s", testValueTypeDouble),
				expectedErr: "invalid custom property name",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				listOptions := models.CatalogArtifactListOptions{
					ParentResourceID: savedTestModel.GetID(),
					Pagination: dbmodels.Pagination{
						OrderBy:   apiutils.Of(tc.orderBy),
						SortOrder: apiutils.Of("ASC"),
					},
				}
				_, err := repo.List(listOptions)
				require.Error(t, err, "Should return error for invalid property name")
				assert.Contains(t, err.Error(), tc.expectedErr, "Error message should mention invalid property name")
			})
		}

		// Test that various property names work (they fallback to ID if non-existent)
		// Property names can contain any characters - they're user-defined metadata
		validTestCases := []string{
			fmt.Sprintf("%s.%s", testPropertyAccuracy, testValueTypeDouble),
			fmt.Sprintf("model_%s.%s", testPropertyAccuracy, testValueTypeDouble),
			fmt.Sprintf("model-%s.%s", testPropertyAccuracy, testValueTypeDouble),
			fmt.Sprintf("v1.%s.%s", testPropertyAccuracy, testValueTypeDouble),
			fmt.Sprintf("%s_v2.%s", testPropertyAccuracy, testValueTypeDouble),
			fmt.Sprintf("my %s.%s", testPropertyAccuracy, testValueTypeDouble),      // spaces are allowed
			fmt.Sprintf("%s@special.%s", testPropertyAccuracy, testValueTypeDouble), // special chars allowed
		}

		for _, validOrderBy := range validTestCases {
			t.Run("PropertyName: "+validOrderBy, func(t *testing.T) {
				listOptions := models.CatalogArtifactListOptions{
					ParentResourceID: savedTestModel.GetID(),
					Pagination: dbmodels.Pagination{
						OrderBy:   apiutils.Of(validOrderBy),
						SortOrder: apiutils.Of("ASC"),
					},
				}
				_, err := repo.List(listOptions)
				// These should not error - non-existent properties just fallback to ID ordering
				require.NoError(t, err, "Should not error for property name: "+validOrderBy)
			})
		}
	})

	t.Run("TestInvalidCustomPropertyValueType_Error", func(t *testing.T) {
		// Create a new model for this test
		testModel := &modelcatalogmodels.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &modelcatalogmodels.CatalogModelAttributes{
				Name:       apiutils.Of("test-model-invalid-value-type"),
				ExternalID: apiutils.Of("test-model-invalid-value-type-ext"),
			},
		}
		savedTestModel, err := catalogModelRepo.Save(testModel)
		require.NoError(t, err)

		// Create an artifact
		metricsArtifact := &modelcatalogmodels.CatalogMetricsArtifactImpl{
			TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
			Attributes: &modelcatalogmodels.CatalogMetricsArtifactAttributes{
				Name:         apiutils.Of("test-artifact"),
				ExternalID:   apiutils.Of("test-artifact-ext"),
				MetricsType:  modelcatalogmodels.MetricsTypeAccuracy,
				ArtifactType: apiutils.Of("metrics-artifact"),
			},
		}
		_, err = metricsArtifactRepo.Save(metricsArtifact, savedTestModel.GetID())
		require.NoError(t, err)

		// Test with invalid value type - should return error
		testCases := []struct {
			name        string
			orderBy     string
			expectedErr string
		}{
			{
				name: "Invalid value type: int_valueeee",
				orderBy: fmt.Sprintf("%s.int_valueeee",
					testPropertyAccuracy),
				expectedErr: "invalid custom property value type 'int_valueeee'",
			},
			{
				name: "Invalid value type: double_val",
				orderBy: fmt.Sprintf("%s.double_val",
					testPropertyScore),
				expectedErr: "invalid custom property value type 'double_val'",
			},
			{
				name: "Invalid value type: str_value",
				orderBy: fmt.Sprintf("%s.str_value",
					testPropertyTimestamp),
				expectedErr: "invalid custom property value type 'str_value'",
			},
			{
				name:        "Invalid value type: random",
				orderBy:     "property.random",
				expectedErr: "invalid custom property value type 'random'",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				listOptions := models.CatalogArtifactListOptions{
					ParentResourceID: savedTestModel.GetID(),
					Pagination: dbmodels.Pagination{
						OrderBy:   apiutils.Of(tc.orderBy),
						SortOrder: apiutils.Of("ASC"),
					},
				}
				_, err := repo.List(listOptions)
				require.Error(t, err, "Should return error for invalid value type")
				assert.Contains(t, err.Error(), tc.expectedErr, "Error message should mention the invalid value type")
			})
		}
	})

	t.Run("TestInvalidCustomPropertyFormat_FallbackToID", func(t *testing.T) {
		// Create a new model for this test
		testModel := &modelcatalogmodels.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &modelcatalogmodels.CatalogModelAttributes{
				Name:       apiutils.Of("test-model-invalid-property-format"),
				ExternalID: apiutils.Of("test-model-invalid-property-ext"),
			},
		}
		savedTestModel, err := catalogModelRepo.Save(testModel)
		require.NoError(t, err)

		// Create some artifacts
		for i := 1; i <= 3; i++ {
			customProps := []dbmodels.Properties{
				{
					Name:        "accuracy",
					DoubleValue: apiutils.Of(float64(i) * 0.1),
				},
			}

			metricsArtifact := &modelcatalogmodels.CatalogMetricsArtifactImpl{
				TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
				Attributes: &modelcatalogmodels.CatalogMetricsArtifactAttributes{
					Name:         apiutils.Of(fmt.Sprintf("invalid-format-artifact-%d", i)),
					ExternalID:   apiutils.Of(fmt.Sprintf("invalid-format-ext-%d", i)),
					MetricsType:  modelcatalogmodels.MetricsTypeAccuracy,
					ArtifactType: apiutils.Of("metrics-artifact"),
				},
				CustomProperties: &customProps,
			}
			_, err := metricsArtifactRepo.Save(metricsArtifact, savedTestModel.GetID())
			require.NoError(t, err)
		}

		// Test with invalid format (no .double_value suffix) - should fallback to ID ordering
		listOptions := models.CatalogArtifactListOptions{
			ParentResourceID: savedTestModel.GetID(),
			Pagination: dbmodels.Pagination{
				OrderBy:   apiutils.Of("accuracy"), // Invalid: missing .double_value
				SortOrder: apiutils.Of("ASC"),
			},
		}
		result, err := repo.List(listOptions)
		require.NoError(t, err, "Should not error on invalid custom property format")
		require.NotNil(t, result)

		// Extract our test artifacts
		var foundIDs []int32
		for _, artifact := range result.Items {
			if artifact.CatalogMetricsArtifact != nil {
				name := artifact.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().Name
				if name != nil && len(*name) > 24 && (*name)[:24] == "invalid-format-artifact-" {
					foundIDs = append(foundIDs, *artifact.CatalogMetricsArtifact.GetID())
				}
			}
		}

		require.GreaterOrEqual(t, len(foundIDs), 3, "Should find all test artifacts")

		// Verify it's ordered by ID (ascending) since it fell back to default
		for i := 0; i < len(foundIDs)-1; i++ {
			assert.Less(t, foundIDs[i], foundIDs[i+1],
				"Should be ordered by ID (fallback) when custom property format is invalid")
		}

		// Test with another invalid format (random string) - should also fallback to ID ordering
		listOptions = models.CatalogArtifactListOptions{
			ParentResourceID: savedTestModel.GetID(),
			Pagination: dbmodels.Pagination{
				OrderBy:   apiutils.Of("nonexistent_property"),
				SortOrder: apiutils.Of("ASC"),
			},
		}
		result, err = repo.List(listOptions)
		require.NoError(t, err, "Should not error on nonexistent property")
		require.NotNil(t, result)

		// Should still return results, just ordered by ID
		foundIDs = []int32{}
		for _, artifact := range result.Items {
			if artifact.CatalogMetricsArtifact != nil {
				name := artifact.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().Name
				if name != nil && len(*name) > 24 && (*name)[:24] == "invalid-format-artifact-" {
					foundIDs = append(foundIDs, *artifact.CatalogMetricsArtifact.GetID())
				}
			}
		}

		require.GreaterOrEqual(t, len(foundIDs), 3, "Should still find all test artifacts")

		// Verify it's ordered by ID
		for i := 0; i < len(foundIDs)-1; i++ {
			assert.Less(t, foundIDs[i], foundIDs[i+1],
				"Should be ordered by ID (fallback) when property doesn't exist")
		}
	})

	t.Run("TestCustomPropertyOrdering_WithAndWithoutProperty", func(t *testing.T) {
		// Create a new model for this test
		testModel := &modelcatalogmodels.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &modelcatalogmodels.CatalogModelAttributes{
				Name:       apiutils.Of("test-model-mixed-properties"),
				ExternalID: apiutils.Of("test-model-mixed-properties-ext"),
			},
		}
		savedTestModel, err := catalogModelRepo.Save(testModel)
		require.NoError(t, err)

		// Create artifacts: some WITH accuracy property, some WITHOUT
		testArtifacts := []struct {
			name     string
			accuracy *float64 // nil means no property
		}{
			{"artifact-with-high-accuracy", apiutils.Of(0.95)},
			{"artifact-without-property-1", nil}, // No accuracy property
			{"artifact-with-low-accuracy", apiutils.Of(0.60)},
			{"artifact-without-property-2", nil}, // No accuracy property
			{"artifact-with-medium-accuracy", apiutils.Of(0.80)},
			{"artifact-without-property-3", nil}, // No accuracy property
		}

		artifactIDMap := make(map[string]int32)
		for _, tc := range testArtifacts {
			var customProps *[]dbmodels.Properties
			if tc.accuracy != nil {
				customProps = &[]dbmodels.Properties{
					{
						Name:        "accuracy",
						DoubleValue: tc.accuracy,
					},
				}
			}

			metricsArtifact := &modelcatalogmodels.CatalogMetricsArtifactImpl{
				TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
				Attributes: &modelcatalogmodels.CatalogMetricsArtifactAttributes{
					Name:         apiutils.Of(tc.name),
					ExternalID:   apiutils.Of(fmt.Sprintf("mixed-prop-test-%s", tc.name)),
					MetricsType:  modelcatalogmodels.MetricsTypeAccuracy,
					ArtifactType: apiutils.Of("metrics-artifact"),
				},
				CustomProperties: customProps,
			}
			saved, err := metricsArtifactRepo.Save(metricsArtifact, savedTestModel.GetID())
			require.NoError(t, err)
			artifactIDMap[tc.name] = *saved.GetID()
		}

		// Test ordering by accuracy.double_value ASC
		// Expected order:
		// 1. artifact-with-low-accuracy (0.60)
		// 2. artifact-with-medium-accuracy (0.80)
		// 3. artifact-with-high-accuracy (0.95)
		// 4. artifact-without-property-1 (ordered by ID)
		// 5. artifact-without-property-2 (ordered by ID)
		// 6. artifact-without-property-3 (ordered by ID)
		listOptions := models.CatalogArtifactListOptions{
			ParentResourceID: savedTestModel.GetID(),
			Pagination: dbmodels.Pagination{
				OrderBy:   apiutils.Of("accuracy.double_value"),
				SortOrder: apiutils.Of("ASC"),
			},
		}
		result, err := repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Extract our test artifacts in order
		var orderedArtifacts []struct {
			name     string
			accuracy *float64
			id       int32
		}
		for _, artifact := range result.Items {
			if artifact.CatalogMetricsArtifact != nil {
				name := artifact.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().Name
				if name != nil {
					// Check if it's one of our test artifacts
					if expectedID, exists := artifactIDMap[*name]; exists {
						var accuracy *float64
						if artifact.CatalogMetricsArtifact.GetCustomProperties() != nil {
							for _, prop := range *artifact.CatalogMetricsArtifact.GetCustomProperties() {
								if prop.Name == "accuracy" && prop.DoubleValue != nil {
									accuracy = prop.DoubleValue
									break
								}
							}
						}
						orderedArtifacts = append(orderedArtifacts, struct {
							name     string
							accuracy *float64
							id       int32
						}{*name, accuracy, expectedID})
					}
				}
			}
		}

		require.Equal(t, 6, len(orderedArtifacts), "Should find all 6 test artifacts")

		// Verify ordering:
		// 1. First should be artifacts WITH accuracy property, ordered by accuracy ASC
		withPropertyCount := 0
		for i := 0; i < len(orderedArtifacts); i++ {
			if orderedArtifacts[i].accuracy != nil {
				withPropertyCount++
				// Verify ascending order among artifacts with property
				if i > 0 && orderedArtifacts[i-1].accuracy != nil {
					assert.LessOrEqual(t, *orderedArtifacts[i-1].accuracy, *orderedArtifacts[i].accuracy,
						"Artifacts with accuracy should be ordered by accuracy ASC")
				}
			} else {
				// Once we hit artifacts without property, all remaining should be without property
				for j := i; j < len(orderedArtifacts); j++ {
					assert.Nil(t, orderedArtifacts[j].accuracy,
						"Artifacts without property should come after artifacts with property")
				}
				break
			}
		}

		assert.Equal(t, 3, withPropertyCount, "Should have 3 artifacts with accuracy property first")

		// 2. Verify artifacts WITHOUT property are ordered by ID
		withoutPropertyArtifacts := orderedArtifacts[withPropertyCount:]
		for i := 0; i < len(withoutPropertyArtifacts)-1; i++ {
			assert.Less(t, withoutPropertyArtifacts[i].id, withoutPropertyArtifacts[i+1].id,
				"Artifacts without property should be ordered by ID")
		}

		// Test with DESC order
		listOptions = models.CatalogArtifactListOptions{
			ParentResourceID: savedTestModel.GetID(),
			Pagination: dbmodels.Pagination{
				OrderBy:   apiutils.Of("accuracy.double_value"),
				SortOrder: apiutils.Of("DESC"),
			},
		}
		result, err = repo.List(listOptions)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Extract artifacts again
		orderedArtifacts = []struct {
			name     string
			accuracy *float64
			id       int32
		}{}
		for _, artifact := range result.Items {
			if artifact.CatalogMetricsArtifact != nil {
				name := artifact.CatalogMetricsArtifact.(modelcatalogmodels.CatalogMetricsArtifact).GetAttributes().Name
				if name != nil {
					if expectedID, exists := artifactIDMap[*name]; exists {
						var accuracy *float64
						if artifact.CatalogMetricsArtifact.GetCustomProperties() != nil {
							for _, prop := range *artifact.CatalogMetricsArtifact.GetCustomProperties() {
								if prop.Name == "accuracy" && prop.DoubleValue != nil {
									accuracy = prop.DoubleValue
									break
								}
							}
						}
						orderedArtifacts = append(orderedArtifacts, struct {
							name     string
							accuracy *float64
							id       int32
						}{*name, accuracy, expectedID})
					}
				}
			}
		}

		require.Equal(t, 6, len(orderedArtifacts), "Should find all 6 test artifacts")

		// Verify DESC ordering:
		// Expected order:
		// 1. artifact-with-high-accuracy (0.95)
		// 2. artifact-with-medium-accuracy (0.80)
		// 3. artifact-with-low-accuracy (0.60)
		// 4. artifact-without-property-1 (ordered by ID)
		// 5. artifact-without-property-2 (ordered by ID)
		// 6. artifact-without-property-3 (ordered by ID)
		withPropertyCount = 0
		for i := 0; i < len(orderedArtifacts); i++ {
			if orderedArtifacts[i].accuracy != nil {
				withPropertyCount++
				// Verify descending order among artifacts with property
				if i > 0 && orderedArtifacts[i-1].accuracy != nil {
					assert.GreaterOrEqual(t, *orderedArtifacts[i-1].accuracy, *orderedArtifacts[i].accuracy,
						"Artifacts with accuracy should be ordered by accuracy DESC")
				}
			} else {
				// Once we hit artifacts without property, all remaining should be without property
				for j := i; j < len(orderedArtifacts); j++ {
					assert.Nil(t, orderedArtifacts[j].accuracy,
						"Artifacts without property should come after artifacts with property (DESC)")
				}
				break
			}
		}

		assert.Equal(t, 3, withPropertyCount, "Should have 3 artifacts with accuracy property first (DESC)")

		// Verify artifacts WITHOUT property are ordered by ID in DESC too
		withoutPropertyArtifacts = orderedArtifacts[withPropertyCount:]
		for i := 0; i < len(withoutPropertyArtifacts)-1; i++ {
			assert.Less(t, withoutPropertyArtifacts[i].id, withoutPropertyArtifacts[i+1].id,
				"Artifacts without property should be ordered by ID (even in DESC mode)")
		}
	})
}
