package core_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/kubeflow/hub/pkg/api"
	"github.com/kubeflow/hub/pkg/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpsertArtifact(t *testing.T) {
	_service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	t.Run("successful create model artifact", func(t *testing.T) {
		modelArtifact := &openapi.ModelArtifact{
			Name:               new("test-model-artifact"),
			Description:        new("Test model artifact description"),
			ExternalId:         new("model-ext-123"),
			Uri:                new("s3://bucket/model.pkl"),
			State:              new(openapi.ARTIFACTSTATE_LIVE),
			ModelFormatName:    new("pickle"),
			ModelFormatVersion: new("1.0"),
			StorageKey:         new("model-storage-key"),
			StoragePath:        new("/models/test"),
			ServiceAccountName: new("model-sa"),
		}

		artifact := &openapi.Artifact{
			ModelArtifact: modelArtifact,
		}

		result, err := _service.UpsertArtifact(artifact)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.ModelArtifact)
		assert.NotNil(t, result.ModelArtifact.Id)
		assert.Equal(t, "test-model-artifact", *result.ModelArtifact.Name)
		assert.Equal(t, "model-ext-123", *result.ModelArtifact.ExternalId)
		assert.Equal(t, "s3://bucket/model.pkl", *result.ModelArtifact.Uri)
		assert.Equal(t, openapi.ARTIFACTSTATE_LIVE, *result.ModelArtifact.State)
		assert.Equal(t, "pickle", *result.ModelArtifact.ModelFormatName)
		assert.Equal(t, "1.0", *result.ModelArtifact.ModelFormatVersion)
		assert.Equal(t, "model-storage-key", *result.ModelArtifact.StorageKey)
		assert.Equal(t, "/models/test", *result.ModelArtifact.StoragePath)
		assert.Equal(t, "model-sa", *result.ModelArtifact.ServiceAccountName)
		assert.NotNil(t, result.ModelArtifact.CreateTimeSinceEpoch)
		assert.NotNil(t, result.ModelArtifact.LastUpdateTimeSinceEpoch)
	})

	t.Run("successful create doc artifact", func(t *testing.T) {
		docArtifact := &openapi.DocArtifact{
			Name:        new("test-doc-artifact"),
			Description: new("Test doc artifact description"),
			ExternalId:  new("doc-ext-123"),
			Uri:         new("s3://bucket/doc.pdf"),
			State:       new(openapi.ARTIFACTSTATE_LIVE),
		}

		artifact := &openapi.Artifact{
			DocArtifact: docArtifact,
		}

		result, err := _service.UpsertArtifact(artifact)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.DocArtifact)
		assert.NotNil(t, result.DocArtifact.Id)
		assert.Equal(t, "test-doc-artifact", *result.DocArtifact.Name)
		assert.Equal(t, "doc-ext-123", *result.DocArtifact.ExternalId)
		assert.Equal(t, "s3://bucket/doc.pdf", *result.DocArtifact.Uri)
		assert.Equal(t, openapi.ARTIFACTSTATE_LIVE, *result.DocArtifact.State)
		assert.NotNil(t, result.DocArtifact.CreateTimeSinceEpoch)
		assert.NotNil(t, result.DocArtifact.LastUpdateTimeSinceEpoch)
	})

	t.Run("successful update model artifact", func(t *testing.T) {
		// Create first
		modelArtifact := &openapi.ModelArtifact{
			Name: new("update-model-artifact"),
			Uri:  new("s3://bucket/original.pkl"),
		}

		created, err := _service.UpsertModelArtifact(modelArtifact)
		require.NoError(t, err)
		require.NotNil(t, created.Id)

		// Update by modifying the created artifact
		created.Uri = new("s3://bucket/updated.pkl")
		created.Description = new("Updated description")

		updated, err := _service.UpsertModelArtifact(created)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, *created.Id, *updated.Id)
		assert.Equal(t, "s3://bucket/updated.pkl", *updated.Uri)
		assert.Equal(t, "Updated description", *updated.Description)
	})

	t.Run("create with custom properties", func(t *testing.T) {
		customProps := map[string]openapi.MetadataValue{
			"accuracy": {
				MetadataDoubleValue: &openapi.MetadataDoubleValue{
					DoubleValue: 0.95,
				},
			},
			"framework": {
				MetadataStringValue: &openapi.MetadataStringValue{
					StringValue: "tensorflow",
				},
			},
		}

		modelArtifact := &openapi.ModelArtifact{
			Name:             new("custom-props-artifact"),
			CustomProperties: customProps,
		}

		artifact := &openapi.Artifact{
			ModelArtifact: modelArtifact,
		}

		result, err := _service.UpsertArtifact(artifact)

		require.NoError(t, err)
		require.NotNil(t, result.ModelArtifact)
		assert.NotNil(t, result.ModelArtifact.CustomProperties)

		resultProps := result.ModelArtifact.CustomProperties
		assert.Contains(t, resultProps, "accuracy")
		assert.Contains(t, resultProps, "framework")
		assert.Equal(t, 0.95, resultProps["accuracy"].MetadataDoubleValue.DoubleValue)
		assert.Equal(t, "tensorflow", resultProps["framework"].MetadataStringValue.StringValue)
	})

	t.Run("nil artifact error", func(t *testing.T) {
		result, err := _service.UpsertArtifact(nil)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid artifact pointer")
	})

	t.Run("invalid artifact type", func(t *testing.T) {
		artifact := &openapi.Artifact{}

		result, err := _service.UpsertArtifact(artifact)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid artifact type, must be either ModelArtifact, DocArtifact, DataSet, Metric, or Parameter")
	})

	t.Run("metric without value error", func(t *testing.T) {
		artifact := &openapi.Artifact{
			Metric: &openapi.Metric{
				Name: new("test-metric-no-value"),
				// Value is intentionally omitted
			},
		}

		result, err := _service.UpsertArtifact(artifact)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "metric value is required")
	})

	// Tests for null name handling - should generate UUID for all artifact types
	t.Run("create model artifact with null name generates UUID", func(t *testing.T) {
		modelArtifact := &openapi.ModelArtifact{
			// Name is intentionally nil/not set
			Uri: new("s3://bucket/model-no-name.pkl"),
		}

		artifact := &openapi.Artifact{
			ModelArtifact: modelArtifact,
		}

		result, err := _service.UpsertArtifact(artifact)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.ModelArtifact)
		assert.NotNil(t, result.ModelArtifact.Name, "Name should be auto-generated")
		assert.NotEmpty(t, *result.ModelArtifact.Name, "Generated name should not be empty")
		// Check if it looks like a UUID (basic check for format)
		assert.Len(t, *result.ModelArtifact.Name, 36, "Generated name should be UUID length")
		assert.Contains(t, *result.ModelArtifact.Name, "-", "Generated name should have UUID format")
	})

	t.Run("create doc artifact with null name generates UUID", func(t *testing.T) {
		docArtifact := &openapi.DocArtifact{
			// Name is intentionally nil/not set
			Uri: new("s3://bucket/doc-no-name.pdf"),
		}

		artifact := &openapi.Artifact{
			DocArtifact: docArtifact,
		}

		result, err := _service.UpsertArtifact(artifact)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.DocArtifact)
		assert.NotNil(t, result.DocArtifact.Name, "Name should be auto-generated")
		assert.NotEmpty(t, *result.DocArtifact.Name, "Generated name should not be empty")
		assert.Len(t, *result.DocArtifact.Name, 36, "Generated name should be UUID length")
		assert.Contains(t, *result.DocArtifact.Name, "-", "Generated name should have UUID format")
	})

	t.Run("create dataset with null name generates UUID", func(t *testing.T) {
		dataSet := &openapi.DataSet{
			// Name is intentionally nil/not set
			Uri: new("s3://bucket/dataset-no-name.csv"),
		}

		artifact := &openapi.Artifact{
			DataSet: dataSet,
		}

		result, err := _service.UpsertArtifact(artifact)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.DataSet)
		assert.NotNil(t, result.DataSet.Name, "Name should be auto-generated")
		assert.NotEmpty(t, *result.DataSet.Name, "Generated name should not be empty")
		assert.Len(t, *result.DataSet.Name, 36, "Generated name should be UUID length")
		assert.Contains(t, *result.DataSet.Name, "-", "Generated name should have UUID format")
	})

	t.Run("create metric with null name generates UUID", func(t *testing.T) {
		metric := &openapi.Metric{
			// Name is intentionally nil/not set
			Value: new(0.99),
		}

		artifact := &openapi.Artifact{
			Metric: metric,
		}

		result, err := _service.UpsertArtifact(artifact)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Metric)
		assert.NotNil(t, result.Metric.Name, "Name should be auto-generated")
		assert.NotEmpty(t, *result.Metric.Name, "Generated name should not be empty")
		assert.Len(t, *result.Metric.Name, 36, "Generated name should be UUID length")
		assert.Contains(t, *result.Metric.Name, "-", "Generated name should have UUID format")
	})

	t.Run("create parameter with null name generates UUID", func(t *testing.T) {
		parameter := &openapi.Parameter{
			// Name is intentionally nil/not set
			Value: new("param-value"),
		}

		artifact := &openapi.Artifact{
			Parameter: parameter,
		}

		result, err := _service.UpsertArtifact(artifact)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Parameter)
		assert.NotNil(t, result.Parameter.Name, "Name should be auto-generated")
		assert.NotEmpty(t, *result.Parameter.Name, "Generated name should not be empty")
		assert.Len(t, *result.Parameter.Name, 36, "Generated name should be UUID length")
		assert.Contains(t, *result.Parameter.Name, "-", "Generated name should have UUID format")
	})

	t.Run("update artifact with null name preserves existing name", func(t *testing.T) {
		// First create an artifact with a specific name
		originalName := "original-artifact-name"
		modelArtifact := &openapi.ModelArtifact{
			Name: new(originalName),
			Uri:  new("s3://bucket/original.pkl"),
		}

		created, err := _service.UpsertModelArtifact(modelArtifact)
		require.NoError(t, err)
		require.NotNil(t, created.Id)

		// Update with nil name - should preserve existing name
		updateArtifact := &openapi.ModelArtifact{
			Id: created.Id,
			// Name is intentionally nil
			Uri: new("s3://bucket/updated.pkl"),
		}

		updated, err := _service.UpsertModelArtifact(updateArtifact)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, originalName, *updated.Name, "Name should be preserved during update")
		assert.Equal(t, "s3://bucket/updated.pkl", *updated.Uri, "Uri should be updated")
	})

	t.Run("unicode characters in model artifact name", func(t *testing.T) {
		// Test with unicode characters: Chinese, Russian, Japanese, and emoji
		unicodeName := "模型工件-тест-モデルアーティファクト-🚀"
		modelArtifact := &openapi.ModelArtifact{
			Name:        new(unicodeName),
			Description: new("Test model artifact with unicode characters"),
			Uri:         new("s3://bucket/unicode-model.pkl"),
		}

		artifact := &openapi.Artifact{
			ModelArtifact: modelArtifact,
		}

		result, err := _service.UpsertArtifact(artifact)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.ModelArtifact)
		assert.Equal(t, unicodeName, *result.ModelArtifact.Name)
		assert.Equal(t, "Test model artifact with unicode characters", *result.ModelArtifact.Description)
		assert.Equal(t, "s3://bucket/unicode-model.pkl", *result.ModelArtifact.Uri)
		assert.NotNil(t, result.ModelArtifact.Id)

		// Verify we can retrieve it by ID
		retrieved, err := _service.GetArtifactById(*result.ModelArtifact.Id)
		require.NoError(t, err)
		require.NotNil(t, retrieved.ModelArtifact)
		assert.Equal(t, unicodeName, *retrieved.ModelArtifact.Name)
	})

	t.Run("special characters in model artifact name", func(t *testing.T) {
		// Test with various special characters
		specialName := "!@#$%^&*()_+-=[]{}|;':\",./<>?"
		modelArtifact := &openapi.ModelArtifact{
			Name:        new(specialName),
			Description: new("Test model artifact with special characters"),
			Uri:         new("s3://bucket/special-model.pkl"),
		}

		artifact := &openapi.Artifact{
			ModelArtifact: modelArtifact,
		}

		result, err := _service.UpsertArtifact(artifact)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.ModelArtifact)
		assert.Equal(t, specialName, *result.ModelArtifact.Name)
		assert.Equal(t, "Test model artifact with special characters", *result.ModelArtifact.Description)
		assert.NotNil(t, result.ModelArtifact.Id)

		// Verify we can retrieve it by ID
		retrieved, err := _service.GetArtifactById(*result.ModelArtifact.Id)
		require.NoError(t, err)
		require.NotNil(t, retrieved.ModelArtifact)
		assert.Equal(t, specialName, *retrieved.ModelArtifact.Name)
	})

	t.Run("mixed unicode and special characters in doc artifact", func(t *testing.T) {
		// Test with mixed unicode and special characters
		mixedName := "文档@#$%工件-тест!@#-ドキュメント()アーティファクト-🚀[]"
		docArtifact := &openapi.DocArtifact{
			Name:        new(mixedName),
			Description: new("Test doc artifact with mixed unicode and special characters"),
			Uri:         new("s3://bucket/mixed-doc.pdf"),
		}

		artifact := &openapi.Artifact{
			DocArtifact: docArtifact,
		}

		result, err := _service.UpsertArtifact(artifact)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.DocArtifact)
		assert.Equal(t, mixedName, *result.DocArtifact.Name)
		assert.Equal(t, "Test doc artifact with mixed unicode and special characters", *result.DocArtifact.Description)
		assert.NotNil(t, result.DocArtifact.Id)

		// Verify we can retrieve it by ID
		retrieved, err := _service.GetArtifactById(*result.DocArtifact.Id)
		require.NoError(t, err)
		require.NotNil(t, retrieved.DocArtifact)
		assert.Equal(t, mixedName, *retrieved.DocArtifact.Name)
	})

	t.Run("pagination with 10+ artifacts", func(t *testing.T) {
		// Create 15 artifacts for pagination testing
		var createdArtifacts []string
		for i := range 15 {
			artifactName := "paging-test-artifact-" + fmt.Sprintf("%02d", i)
			modelArtifact := &openapi.ModelArtifact{
				Name:        new(artifactName),
				Description: new("Pagination test artifact " + fmt.Sprintf("%02d", i)),
				Uri:         new("s3://bucket/paging-test-" + fmt.Sprintf("%02d", i) + ".pkl"),
			}

			artifact := &openapi.Artifact{
				ModelArtifact: modelArtifact,
			}

			result, err := _service.UpsertArtifact(artifact)
			require.NoError(t, err)
			createdArtifacts = append(createdArtifacts, *result.ModelArtifact.Id)
		}

		// Test pagination with page size 5
		pageSize := int32(5)
		orderBy := "name"
		sortOrder := "ASC"
		listOptions := api.ListOptions{
			PageSize:  &pageSize,
			OrderBy:   &orderBy,
			SortOrder: &sortOrder,
		}

		// Get first page
		firstPage, err := _service.GetArtifacts(openapi.ARTIFACTTYPEQUERYPARAM_MODEL_ARTIFACT, listOptions, nil)
		require.NoError(t, err)
		require.NotNil(t, firstPage)
		assert.LessOrEqual(t, len(firstPage.Items), 5, "First page should have at most 5 items")
		assert.Equal(t, int32(5), firstPage.PageSize)

		// Filter to only our test artifacts in first page
		var firstPageTestArtifacts []openapi.Artifact
		firstPageIds := make(map[string]bool)
		for _, item := range firstPage.Items {
			// Only include our test artifacts (those with the specific prefix)
			var artifactName string
			if item.ModelArtifact != nil {
				artifactName = *item.ModelArtifact.Name
			} else if item.DocArtifact != nil {
				artifactName = *item.DocArtifact.Name
			}

			if strings.HasPrefix(artifactName, "paging-test-artifact-") {
				var artifactId string
				if item.ModelArtifact != nil {
					artifactId = *item.ModelArtifact.Id
				} else if item.DocArtifact != nil {
					artifactId = *item.DocArtifact.Id
				}
				assert.False(t, firstPageIds[artifactId], "Should not have duplicate IDs in first page")
				firstPageIds[artifactId] = true
				firstPageTestArtifacts = append(firstPageTestArtifacts, item)
			}
		}

		// Only proceed with second page test if we have a next page token and found test artifacts
		if firstPage.NextPageToken != "" && len(firstPageTestArtifacts) > 0 {
			// Get second page using next page token
			listOptions.NextPageToken = &firstPage.NextPageToken
			secondPage, err := _service.GetArtifacts(openapi.ARTIFACTTYPEQUERYPARAM_MODEL_ARTIFACT, listOptions, nil)
			require.NoError(t, err)
			require.NotNil(t, secondPage)
			assert.LessOrEqual(t, len(secondPage.Items), 5, "Second page should have at most 5 items")

			// Verify no duplicates between pages (only check our test artifacts)
			for _, item := range secondPage.Items {
				var artifactName, artifactId string
				if item.ModelArtifact != nil {
					artifactName = *item.ModelArtifact.Name
					artifactId = *item.ModelArtifact.Id
				} else if item.DocArtifact != nil {
					artifactName = *item.DocArtifact.Name
					artifactId = *item.DocArtifact.Id
				}

				if strings.HasPrefix(artifactName, "paging-test-artifact-") {
					assert.False(t, firstPageIds[artifactId], "Should not have duplicate IDs between pages")
				}
			}
		}

		// Test with larger page size
		largePage := int32(100)
		listOptions = api.ListOptions{
			PageSize:  &largePage,
			OrderBy:   &orderBy,
			SortOrder: &sortOrder,
		}

		allItems, err := _service.GetArtifacts(openapi.ARTIFACTTYPEQUERYPARAM_MODEL_ARTIFACT, listOptions, nil)
		require.NoError(t, err)
		require.NotNil(t, allItems)
		assert.GreaterOrEqual(t, len(allItems.Items), 15, "Should have at least our 15 test artifacts")

		// Count our test artifacts in the results
		foundCount := 0
		for _, item := range allItems.Items {
			var artifactId string
			if item.ModelArtifact != nil {
				artifactId = *item.ModelArtifact.Id
			} else if item.DocArtifact != nil {
				artifactId = *item.DocArtifact.Id
			}

			if slices.Contains(createdArtifacts, artifactId) {
				foundCount++
			}
		}
		assert.Equal(t, 15, foundCount, "Should find all 15 created artifacts")

		// Test descending order
		descOrder := "DESC"
		listOptions = api.ListOptions{
			PageSize:  &pageSize,
			OrderBy:   &orderBy,
			SortOrder: &descOrder,
		}

		descPage, err := _service.GetArtifacts(openapi.ARTIFACTTYPEQUERYPARAM_MODEL_ARTIFACT, listOptions, nil)
		require.NoError(t, err)
		require.NotNil(t, descPage)
		assert.LessOrEqual(t, len(descPage.Items), 5, "Desc page should have at most 5 items")

		// Verify ordering (names should be in descending order)
		if len(descPage.Items) > 1 {
			for i := 1; i < len(descPage.Items); i++ {
				var prevName, currName string
				if descPage.Items[i-1].ModelArtifact != nil {
					prevName = *descPage.Items[i-1].ModelArtifact.Name
				} else if descPage.Items[i-1].DocArtifact != nil {
					prevName = *descPage.Items[i-1].DocArtifact.Name
				}
				if descPage.Items[i].ModelArtifact != nil {
					currName = *descPage.Items[i].ModelArtifact.Name
				} else if descPage.Items[i].DocArtifact != nil {
					currName = *descPage.Items[i].DocArtifact.Name
				}
				assert.GreaterOrEqual(t, prevName, currName,
					"Items should be in descending order by name")
			}
		}
	})
}

func TestUpsertModelVersionArtifact(t *testing.T) {
	_service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	t.Run("successful create with model version", func(t *testing.T) {
		// First create a registered model and model version
		registeredModel := &openapi.RegisteredModel{
			Name: "test-model-for-artifact",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		modelVersion := &openapi.ModelVersion{
			Name:        "v1.0",
			Description: new("Version 1.0"),
		}
		createdVersion, err := _service.UpsertModelVersion(modelVersion, createdModel.Id)
		require.NoError(t, err)

		// Create artifact associated with model version
		modelArtifact := &openapi.ModelArtifact{
			Name: new("version-artifact"),
			Uri:  new("s3://bucket/version-model.pkl"),
		}

		artifact := &openapi.Artifact{
			ModelArtifact: modelArtifact,
		}

		result, err := _service.UpsertModelVersionArtifact(artifact, *createdVersion.Id)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.ModelArtifact)
		assert.NotNil(t, result.ModelArtifact.Id)
		// Name should be prefixed with model version ID
		assert.Contains(t, *result.ModelArtifact.Name, "version-artifact")
		assert.Equal(t, "s3://bucket/version-model.pkl", *result.ModelArtifact.Uri)
	})

	t.Run("invalid model version id", func(t *testing.T) {
		modelArtifact := &openapi.ModelArtifact{
			Name: new("test-artifact"),
		}

		artifact := &openapi.Artifact{
			ModelArtifact: modelArtifact,
		}

		result, err := _service.UpsertModelVersionArtifact(artifact, "invalid")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid syntax: bad request")
	})

	t.Run("unicode characters in model version artifact name", func(t *testing.T) {
		// First create a registered model and model version
		registeredModel := &openapi.RegisteredModel{
			Name: "unicode-test-model-for-artifact",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		modelVersion := &openapi.ModelVersion{
			Name: "v1.0-unicode",
		}
		createdVersion, err := _service.UpsertModelVersion(modelVersion, createdModel.Id)
		require.NoError(t, err)

		// Test with unicode characters: Chinese, Russian, Japanese, and emoji
		unicodeName := "版本工件-тест-バージョンアーティファクト-🚀"
		modelArtifact := &openapi.ModelArtifact{
			Name:        new(unicodeName),
			Description: new("Test model version artifact with unicode characters"),
			Uri:         new("s3://bucket/unicode-version-model.pkl"),
		}

		artifact := &openapi.Artifact{
			ModelArtifact: modelArtifact,
		}

		result, err := _service.UpsertModelVersionArtifact(artifact, *createdVersion.Id)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.ModelArtifact)
		assert.Contains(t, *result.ModelArtifact.Name, unicodeName)
		assert.Equal(t, "Test model version artifact with unicode characters", *result.ModelArtifact.Description)
		assert.Equal(t, "s3://bucket/unicode-version-model.pkl", *result.ModelArtifact.Uri)
		assert.NotNil(t, result.ModelArtifact.Id)

		// Verify we can retrieve it by ID
		retrieved, err := _service.GetArtifactById(*result.ModelArtifact.Id)
		require.NoError(t, err)
		require.NotNil(t, retrieved.ModelArtifact)
		assert.Contains(t, *retrieved.ModelArtifact.Name, unicodeName)
	})

	t.Run("special characters in model version artifact name", func(t *testing.T) {
		// First create a registered model and model version
		registeredModel := &openapi.RegisteredModel{
			Name: "special-chars-test-model-for-artifact",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		modelVersion := &openapi.ModelVersion{
			Name: "v1.0-special",
		}
		createdVersion, err := _service.UpsertModelVersion(modelVersion, createdModel.Id)
		require.NoError(t, err)

		// Test with various special characters
		specialName := "!@#$%^&*()_+-=[]{}|;':\",./<>?"
		modelArtifact := &openapi.ModelArtifact{
			Name:        new(specialName),
			Description: new("Test model version artifact with special characters"),
			Uri:         new("s3://bucket/special-version-model.pkl"),
		}

		artifact := &openapi.Artifact{
			ModelArtifact: modelArtifact,
		}

		result, err := _service.UpsertModelVersionArtifact(artifact, *createdVersion.Id)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.ModelArtifact)
		assert.Contains(t, *result.ModelArtifact.Name, specialName)
		assert.Equal(t, "Test model version artifact with special characters", *result.ModelArtifact.Description)
		assert.NotNil(t, result.ModelArtifact.Id)

		// Verify we can retrieve it by ID
		retrieved, err := _service.GetArtifactById(*result.ModelArtifact.Id)
		require.NoError(t, err)
		require.NotNil(t, retrieved.ModelArtifact)
		assert.Contains(t, *retrieved.ModelArtifact.Name, specialName)
	})

	t.Run("mixed unicode and special characters in model version artifact", func(t *testing.T) {
		// First create a registered model and model version
		registeredModel := &openapi.RegisteredModel{
			Name: "mixed-chars-test-model-for-artifact",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		modelVersion := &openapi.ModelVersion{
			Name: "v1.0-mixed",
		}
		createdVersion, err := _service.UpsertModelVersion(modelVersion, createdModel.Id)
		require.NoError(t, err)

		// Test with mixed unicode and special characters
		mixedName := "版本@#$%工件-тест!@#-バージョン()アーティファクト-🚀[]"
		modelArtifact := &openapi.ModelArtifact{
			Name:        new(mixedName),
			Description: new("Test model version artifact with mixed unicode and special characters"),
			Uri:         new("s3://bucket/mixed-version-model.pkl"),
		}

		artifact := &openapi.Artifact{
			ModelArtifact: modelArtifact,
		}

		result, err := _service.UpsertModelVersionArtifact(artifact, *createdVersion.Id)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.ModelArtifact)
		assert.Contains(t, *result.ModelArtifact.Name, mixedName)
		assert.Equal(t, "Test model version artifact with mixed unicode and special characters", *result.ModelArtifact.Description)
		assert.NotNil(t, result.ModelArtifact.Id)

		// Verify we can retrieve it by ID
		retrieved, err := _service.GetArtifactById(*result.ModelArtifact.Id)
		require.NoError(t, err)
		require.NotNil(t, retrieved.ModelArtifact)
		assert.Contains(t, *retrieved.ModelArtifact.Name, mixedName)
	})

	t.Run("pagination with 10+ model version artifacts", func(t *testing.T) {
		// First create a registered model and model version
		registeredModel := &openapi.RegisteredModel{
			Name: "paging-test-model-for-artifacts",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		modelVersion := &openapi.ModelVersion{
			Name: "v1.0-paging",
		}
		createdVersion, err := _service.UpsertModelVersion(modelVersion, createdModel.Id)
		require.NoError(t, err)

		// Create 15 model version artifacts for pagination testing
		var createdArtifacts []string
		for i := range 15 {
			artifactName := "paging-test-version-artifact-" + fmt.Sprintf("%02d", i)
			modelArtifact := &openapi.ModelArtifact{
				Name:        new(artifactName),
				Description: new("Pagination test model version artifact " + fmt.Sprintf("%02d", i)),
				Uri:         new("s3://bucket/paging-version-test-" + fmt.Sprintf("%02d", i) + ".pkl"),
			}

			artifact := &openapi.Artifact{
				ModelArtifact: modelArtifact,
			}

			result, err := _service.UpsertModelVersionArtifact(artifact, *createdVersion.Id)
			require.NoError(t, err)
			createdArtifacts = append(createdArtifacts, *result.ModelArtifact.Id)
		}

		// Test pagination with page size 5
		pageSize := int32(5)
		orderBy := "name"
		sortOrder := "ASC"
		listOptions := api.ListOptions{
			PageSize:  &pageSize,
			OrderBy:   &orderBy,
			SortOrder: &sortOrder,
		}

		// Get first page
		firstPage, err := _service.GetArtifacts(openapi.ARTIFACTTYPEQUERYPARAM_MODEL_ARTIFACT, listOptions, createdVersion.Id)
		require.NoError(t, err)
		require.NotNil(t, firstPage)
		assert.LessOrEqual(t, len(firstPage.Items), 5, "First page should have at most 5 items")
		assert.Equal(t, int32(5), firstPage.PageSize)

		// Filter to only our test artifacts in first page
		var firstPageTestArtifacts []openapi.Artifact
		firstPageIds := make(map[string]bool)
		for _, item := range firstPage.Items {
			// Only include our test artifacts (those with the specific prefix)
			var artifactName string
			if item.ModelArtifact != nil {
				artifactName = *item.ModelArtifact.Name
			}

			if strings.Contains(artifactName, "paging-test-version-artifact-") {
				artifactId := *item.ModelArtifact.Id
				assert.False(t, firstPageIds[artifactId], "Should not have duplicate IDs in first page")
				firstPageIds[artifactId] = true
				firstPageTestArtifacts = append(firstPageTestArtifacts, item)
			}
		}

		// Only proceed with second page test if we have a next page token and found test artifacts
		if firstPage.NextPageToken != "" && len(firstPageTestArtifacts) > 0 {
			// Get second page using next page token
			listOptions.NextPageToken = &firstPage.NextPageToken
			secondPage, err := _service.GetArtifacts(openapi.ARTIFACTTYPEQUERYPARAM_MODEL_ARTIFACT, listOptions, createdVersion.Id)
			require.NoError(t, err)
			require.NotNil(t, secondPage)
			assert.LessOrEqual(t, len(secondPage.Items), 5, "Second page should have at most 5 items")

			// Verify no duplicates between pages (only check our test artifacts)
			for _, item := range secondPage.Items {
				if item.ModelArtifact != nil && strings.Contains(*item.ModelArtifact.Name, "paging-test-version-artifact-") {
					assert.False(t, firstPageIds[*item.ModelArtifact.Id], "Should not have duplicate IDs between pages")
				}
			}
		}

		// Test with larger page size
		largePage := int32(100)
		listOptions = api.ListOptions{
			PageSize:  &largePage,
			OrderBy:   &orderBy,
			SortOrder: &sortOrder,
		}

		allItems, err := _service.GetArtifacts(openapi.ARTIFACTTYPEQUERYPARAM_MODEL_ARTIFACT, listOptions, createdVersion.Id)
		require.NoError(t, err)
		require.NotNil(t, allItems)
		assert.GreaterOrEqual(t, len(allItems.Items), 15, "Should have at least our 15 test artifacts")

		// Count our test artifacts in the results
		foundCount := 0
		for _, item := range allItems.Items {
			if item.ModelArtifact != nil {
				if slices.Contains(createdArtifacts, *item.ModelArtifact.Id) {
					foundCount++
				}
			}
		}
		assert.Equal(t, 15, foundCount, "Should find all 15 created model version artifacts")

		// Test descending order
		descOrder := "DESC"
		listOptions = api.ListOptions{
			PageSize:  &pageSize,
			OrderBy:   &orderBy,
			SortOrder: &descOrder,
		}

		descPage, err := _service.GetArtifacts(openapi.ARTIFACTTYPEQUERYPARAM_MODEL_ARTIFACT, listOptions, createdVersion.Id)
		require.NoError(t, err)
		require.NotNil(t, descPage)
		assert.LessOrEqual(t, len(descPage.Items), 5, "Desc page should have at most 5 items")

		// Verify ordering (names should be in descending order)
		if len(descPage.Items) > 1 {
			for i := 1; i < len(descPage.Items); i++ {
				var prevName, currName string
				if descPage.Items[i-1].ModelArtifact != nil {
					prevName = *descPage.Items[i-1].ModelArtifact.Name
				}
				if descPage.Items[i].ModelArtifact != nil {
					currName = *descPage.Items[i].ModelArtifact.Name
				}
				if prevName != "" && currName != "" {
					assert.GreaterOrEqual(t, prevName, currName,
						"Items should be in descending order by name")
				}
			}
		}
	})
}

func TestGetArtifactById(t *testing.T) {
	_service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	t.Run("successful get model artifact", func(t *testing.T) {
		// Create a model artifact first
		modelArtifact := &openapi.ModelArtifact{
			Name:        new("get-test-model-artifact"),
			Description: new("Test description"),
			Uri:         new("s3://bucket/test.pkl"),
		}

		artifact := &openapi.Artifact{
			ModelArtifact: modelArtifact,
		}

		created, err := _service.UpsertArtifact(artifact)
		require.NoError(t, err)
		require.NotNil(t, created.ModelArtifact.Id)

		// Get the artifact by ID
		result, err := _service.GetArtifactById(*created.ModelArtifact.Id)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.ModelArtifact)
		assert.Equal(t, *created.ModelArtifact.Id, *result.ModelArtifact.Id)
		assert.Equal(t, "get-test-model-artifact", *result.ModelArtifact.Name)
		assert.Equal(t, "Test description", *result.ModelArtifact.Description)
		assert.Equal(t, "s3://bucket/test.pkl", *result.ModelArtifact.Uri)
	})

	t.Run("successful get doc artifact", func(t *testing.T) {
		// Create a doc artifact first
		docArtifact := &openapi.DocArtifact{
			Name:        new("get-test-doc-artifact"),
			Description: new("Test doc description"),
			Uri:         new("s3://bucket/test.pdf"),
		}

		artifact := &openapi.Artifact{
			DocArtifact: docArtifact,
		}

		created, err := _service.UpsertArtifact(artifact)
		require.NoError(t, err)
		require.NotNil(t, created.DocArtifact.Id)

		// Get the artifact by ID
		result, err := _service.GetArtifactById(*created.DocArtifact.Id)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.DocArtifact)
		assert.Equal(t, *created.DocArtifact.Id, *result.DocArtifact.Id)
		assert.Equal(t, "get-test-doc-artifact", *result.DocArtifact.Name)
		assert.Equal(t, "Test doc description", *result.DocArtifact.Description)
		assert.Equal(t, "s3://bucket/test.pdf", *result.DocArtifact.Uri)
	})

	t.Run("invalid id", func(t *testing.T) {
		result, err := _service.GetArtifactById("invalid")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid")
	})

	t.Run("non-existent id", func(t *testing.T) {
		result, err := _service.GetArtifactById("99999")

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestGetArtifactByParams(t *testing.T) {
	_service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	t.Run("successful get by name and model version", func(t *testing.T) {
		// Create registered model and model version
		registeredModel := &openapi.RegisteredModel{
			Name: "test-model-for-params",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		modelVersion := &openapi.ModelVersion{
			Name: "v1.0",
		}
		createdVersion, err := _service.UpsertModelVersion(modelVersion, createdModel.Id)
		require.NoError(t, err)

		// Create artifact with model version
		modelArtifact := &openapi.ModelArtifact{
			Name: new("params-test-artifact"),
			Uri:  new("s3://bucket/params-test.pkl"),
		}

		artifact := &openapi.Artifact{
			ModelArtifact: modelArtifact,
		}

		created, err := _service.UpsertModelVersionArtifact(artifact, *createdVersion.Id)
		require.NoError(t, err)

		// Get by name and model version ID
		result, err := _service.GetArtifactByParams(new("params-test-artifact"), createdVersion.Id, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.ModelArtifact)
		assert.Equal(t, *created.ModelArtifact.Id, *result.ModelArtifact.Id)
	})

	t.Run("successful get by external id", func(t *testing.T) {
		modelArtifact := &openapi.ModelArtifact{
			Name:       new("external-id-artifact"),
			ExternalId: new("ext-params-123"),
			Uri:        new("s3://bucket/external.pkl"),
		}

		artifact := &openapi.Artifact{
			ModelArtifact: modelArtifact,
		}

		created, err := _service.UpsertArtifact(artifact)
		require.NoError(t, err)

		// Get by external ID
		result, err := _service.GetArtifactByParams(nil, nil, new("ext-params-123"))

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.ModelArtifact)
		assert.Equal(t, *created.ModelArtifact.Id, *result.ModelArtifact.Id)
		assert.Equal(t, "ext-params-123", *result.ModelArtifact.ExternalId)
	})

	t.Run("invalid parameters", func(t *testing.T) {
		result, err := _service.GetArtifactByParams(nil, nil, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid parameters")
	})

	t.Run("artifact not found", func(t *testing.T) {
		result, err := _service.GetArtifactByParams(nil, nil, new("non-existent"))

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "no artifacts found")
	})

	t.Run("same artifact name across different model versions - all artifact types", func(t *testing.T) {
		// This test verifies that parentResourceId filtering works correctly for all artifact types

		// Create a registered model
		registeredModel := &openapi.RegisteredModel{
			Name: "model-for-all-artifact-types",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		// Create two model versions
		version1 := &openapi.ModelVersion{
			Name: "version-with-all-artifacts-1",
		}
		createdVersion1, err := _service.UpsertModelVersion(version1, createdModel.Id)
		require.NoError(t, err)

		version2 := &openapi.ModelVersion{
			Name: "version-with-all-artifacts-2",
		}
		createdVersion2, err := _service.UpsertModelVersion(version2, createdModel.Id)
		require.NoError(t, err)

		// Test cases for each artifact type
		artifactTypes := []struct {
			name            string
			artifactName    string
			createArtifact1 *openapi.Artifact
			createArtifact2 *openapi.Artifact
			checkField      func(*openapi.Artifact) any
			getDescription  func(*openapi.Artifact) string
		}{
			{
				name:         "ModelArtifact",
				artifactName: "shared-model-artifact-name",
				createArtifact1: &openapi.Artifact{
					ModelArtifact: &openapi.ModelArtifact{
						Name:        new("shared-model-artifact-name"),
						Uri:         new("s3://bucket/model-v1.pkl"),
						Description: new("Model artifact for version 1"),
					},
				},
				createArtifact2: &openapi.Artifact{
					ModelArtifact: &openapi.ModelArtifact{
						Name:        new("shared-model-artifact-name"),
						Uri:         new("s3://bucket/model-v2.pkl"),
						Description: new("Model artifact for version 2"),
					},
				},
				checkField: func(a *openapi.Artifact) any { return a.ModelArtifact },
				getDescription: func(a *openapi.Artifact) string {
					if a.ModelArtifact != nil && a.ModelArtifact.Description != nil {
						return *a.ModelArtifact.Description
					}
					return ""
				},
			},
			{
				name:         "DocArtifact",
				artifactName: "shared-doc-artifact-name",
				createArtifact1: &openapi.Artifact{
					DocArtifact: &openapi.DocArtifact{
						Name:        new("shared-doc-artifact-name"),
						Uri:         new("s3://bucket/doc-v1.pdf"),
						Description: new("Doc artifact for version 1"),
					},
				},
				createArtifact2: &openapi.Artifact{
					DocArtifact: &openapi.DocArtifact{
						Name:        new("shared-doc-artifact-name"),
						Uri:         new("s3://bucket/doc-v2.pdf"),
						Description: new("Doc artifact for version 2"),
					},
				},
				checkField: func(a *openapi.Artifact) any { return a.DocArtifact },
				getDescription: func(a *openapi.Artifact) string {
					if a.DocArtifact != nil && a.DocArtifact.Description != nil {
						return *a.DocArtifact.Description
					}
					return ""
				},
			},
			{
				name:         "DataSet",
				artifactName: "shared-dataset-artifact-name",
				createArtifact1: &openapi.Artifact{
					DataSet: &openapi.DataSet{
						Name:        new("shared-dataset-artifact-name"),
						Uri:         new("s3://bucket/dataset-v1.csv"),
						Description: new("Dataset for version 1"),
					},
				},
				createArtifact2: &openapi.Artifact{
					DataSet: &openapi.DataSet{
						Name:        new("shared-dataset-artifact-name"),
						Uri:         new("s3://bucket/dataset-v2.csv"),
						Description: new("Dataset for version 2"),
					},
				},
				checkField: func(a *openapi.Artifact) any { return a.DataSet },
				getDescription: func(a *openapi.Artifact) string {
					if a.DataSet != nil && a.DataSet.Description != nil {
						return *a.DataSet.Description
					}
					return ""
				},
			},
			{
				name:         "Metric",
				artifactName: "shared-metric-artifact-name",
				createArtifact1: &openapi.Artifact{
					Metric: &openapi.Metric{
						Name:        new("shared-metric-artifact-name"),
						Value:       new(0.95),
						Description: new("Metric for version 1"),
					},
				},
				createArtifact2: &openapi.Artifact{
					Metric: &openapi.Metric{
						Name:        new("shared-metric-artifact-name"),
						Value:       new(0.97),
						Description: new("Metric for version 2"),
					},
				},
				checkField: func(a *openapi.Artifact) any { return a.Metric },
				getDescription: func(a *openapi.Artifact) string {
					if a.Metric != nil && a.Metric.Description != nil {
						return *a.Metric.Description
					}
					return ""
				},
			},
			{
				name:         "Parameter",
				artifactName: "shared-parameter-artifact-name",
				createArtifact1: &openapi.Artifact{
					Parameter: &openapi.Parameter{
						Name:        new("shared-parameter-artifact-name"),
						Value:       new("0.001"),
						Description: new("Parameter for version 1"),
					},
				},
				createArtifact2: &openapi.Artifact{
					Parameter: &openapi.Parameter{
						Name:        new("shared-parameter-artifact-name"),
						Value:       new("0.002"),
						Description: new("Parameter for version 2"),
					},
				},
				checkField: func(a *openapi.Artifact) any { return a.Parameter },
				getDescription: func(a *openapi.Artifact) string {
					if a.Parameter != nil && a.Parameter.Description != nil {
						return *a.Parameter.Description
					}
					return ""
				},
			},
		}

		for _, tc := range artifactTypes {
			t.Run(tc.name, func(t *testing.T) {
				// Create artifact with same name for version 1
				created1, err := _service.UpsertModelVersionArtifact(tc.createArtifact1, *createdVersion1.Id)
				require.NoError(t, err)
				require.NotNil(t, tc.checkField(created1))

				// Create artifact with same name for version 2
				created2, err := _service.UpsertModelVersionArtifact(tc.createArtifact2, *createdVersion2.Id)
				require.NoError(t, err)
				require.NotNil(t, tc.checkField(created2))

				// Query for artifact by name and version 1
				result1, err := _service.GetArtifactByParams(&tc.artifactName, createdVersion1.Id, nil)
				require.NoError(t, err)
				require.NotNil(t, result1)
				require.NotNil(t, tc.checkField(result1))
				assert.Contains(t, tc.getDescription(result1), "version 1")

				// Query for artifact by name and version 2
				result2, err := _service.GetArtifactByParams(&tc.artifactName, createdVersion2.Id, nil)
				require.NoError(t, err)
				require.NotNil(t, result2)
				require.NotNil(t, tc.checkField(result2))
				assert.Contains(t, tc.getDescription(result2), "version 2")

				// Ensure we got different artifacts
				assert.NotEqual(t, tc.getDescription(result1), tc.getDescription(result2))
			})
		}
	})

	t.Run("same artifact name across different experiment runs - all artifact types", func(t *testing.T) {
		// This test verifies that parentResourceId filtering works correctly for all artifact types in experiment runs

		// Create an experiment
		experiment := &openapi.Experiment{
			Name: "experiment-for-all-artifact-types",
		}
		createdExperiment, err := _service.UpsertExperiment(experiment)
		require.NoError(t, err)

		// Create two experiment runs
		run1 := &openapi.ExperimentRun{
			Name: new("run-with-all-artifacts-1"),
		}
		createdRun1, err := _service.UpsertExperimentRun(run1, createdExperiment.Id)
		require.NoError(t, err)

		run2 := &openapi.ExperimentRun{
			Name: new("run-with-all-artifacts-2"),
		}
		createdRun2, err := _service.UpsertExperimentRun(run2, createdExperiment.Id)
		require.NoError(t, err)

		// Test cases for each artifact type
		artifactTypes := []struct {
			name            string
			artifactName    string
			createArtifact1 *openapi.Artifact
			createArtifact2 *openapi.Artifact
			checkField      func(*openapi.Artifact) any
			getDescription  func(*openapi.Artifact) string
		}{
			{
				name:         "ModelArtifact",
				artifactName: "shared-run-model-artifact-name",
				createArtifact1: &openapi.Artifact{
					ModelArtifact: &openapi.ModelArtifact{
						Name:        new("shared-run-model-artifact-name"),
						Uri:         new("s3://bucket/run1-model.pkl"),
						Description: new("Model artifact for run 1"),
					},
				},
				createArtifact2: &openapi.Artifact{
					ModelArtifact: &openapi.ModelArtifact{
						Name:        new("shared-run-model-artifact-name"),
						Uri:         new("s3://bucket/run2-model.pkl"),
						Description: new("Model artifact for run 2"),
					},
				},
				checkField: func(a *openapi.Artifact) any { return a.ModelArtifact },
				getDescription: func(a *openapi.Artifact) string {
					if a.ModelArtifact != nil && a.ModelArtifact.Description != nil {
						return *a.ModelArtifact.Description
					}
					return ""
				},
			},
			{
				name:         "DocArtifact",
				artifactName: "shared-run-doc-artifact-name",
				createArtifact1: &openapi.Artifact{
					DocArtifact: &openapi.DocArtifact{
						Name:        new("shared-run-doc-artifact-name"),
						Uri:         new("s3://bucket/run1-doc.pdf"),
						Description: new("Doc artifact for run 1"),
					},
				},
				createArtifact2: &openapi.Artifact{
					DocArtifact: &openapi.DocArtifact{
						Name:        new("shared-run-doc-artifact-name"),
						Uri:         new("s3://bucket/run2-doc.pdf"),
						Description: new("Doc artifact for run 2"),
					},
				},
				checkField: func(a *openapi.Artifact) any { return a.DocArtifact },
				getDescription: func(a *openapi.Artifact) string {
					if a.DocArtifact != nil && a.DocArtifact.Description != nil {
						return *a.DocArtifact.Description
					}
					return ""
				},
			},
			{
				name:         "DataSet",
				artifactName: "shared-run-dataset-artifact-name",
				createArtifact1: &openapi.Artifact{
					DataSet: &openapi.DataSet{
						Name:        new("shared-run-dataset-artifact-name"),
						Uri:         new("s3://bucket/run1-dataset.csv"),
						Description: new("Dataset for run 1"),
					},
				},
				createArtifact2: &openapi.Artifact{
					DataSet: &openapi.DataSet{
						Name:        new("shared-run-dataset-artifact-name"),
						Uri:         new("s3://bucket/run2-dataset.csv"),
						Description: new("Dataset for run 2"),
					},
				},
				checkField: func(a *openapi.Artifact) any { return a.DataSet },
				getDescription: func(a *openapi.Artifact) string {
					if a.DataSet != nil && a.DataSet.Description != nil {
						return *a.DataSet.Description
					}
					return ""
				},
			},
			{
				name:         "Metric",
				artifactName: "shared-run-metric-artifact-name",
				createArtifact1: &openapi.Artifact{
					Metric: &openapi.Metric{
						Name:        new("shared-run-metric-artifact-name"),
						Value:       new(0.91),
						Description: new("Metric for run 1"),
					},
				},
				createArtifact2: &openapi.Artifact{
					Metric: &openapi.Metric{
						Name:        new("shared-run-metric-artifact-name"),
						Value:       new(0.93),
						Description: new("Metric for run 2"),
					},
				},
				checkField: func(a *openapi.Artifact) any { return a.Metric },
				getDescription: func(a *openapi.Artifact) string {
					if a.Metric != nil && a.Metric.Description != nil {
						return *a.Metric.Description
					}
					return ""
				},
			},
			{
				name:         "Parameter",
				artifactName: "shared-run-parameter-artifact-name",
				createArtifact1: &openapi.Artifact{
					Parameter: &openapi.Parameter{
						Name:        new("shared-run-parameter-artifact-name"),
						Value:       new("0.01"),
						Description: new("Parameter for run 1"),
					},
				},
				createArtifact2: &openapi.Artifact{
					Parameter: &openapi.Parameter{
						Name:        new("shared-run-parameter-artifact-name"),
						Value:       new("0.02"),
						Description: new("Parameter for run 2"),
					},
				},
				checkField: func(a *openapi.Artifact) any { return a.Parameter },
				getDescription: func(a *openapi.Artifact) string {
					if a.Parameter != nil && a.Parameter.Description != nil {
						return *a.Parameter.Description
					}
					return ""
				},
			},
		}

		for _, tc := range artifactTypes {
			t.Run(tc.name, func(t *testing.T) {
				// Create artifact with same name for run 1
				created1, err := _service.UpsertExperimentRunArtifact(tc.createArtifact1, *createdRun1.Id)
				require.NoError(t, err)
				require.NotNil(t, tc.checkField(created1))

				// Create artifact with same name for run 2
				created2, err := _service.UpsertExperimentRunArtifact(tc.createArtifact2, *createdRun2.Id)
				require.NoError(t, err)
				require.NotNil(t, tc.checkField(created2))

				// Query for artifact by name and run 1
				result1, err := _service.GetArtifactByParams(&tc.artifactName, createdRun1.Id, nil)
				require.NoError(t, err)
				require.NotNil(t, result1)
				require.NotNil(t, tc.checkField(result1))
				assert.Contains(t, tc.getDescription(result1), "run 1")

				// Query for artifact by name and run 2
				result2, err := _service.GetArtifactByParams(&tc.artifactName, createdRun2.Id, nil)
				require.NoError(t, err)
				require.NotNil(t, result2)
				require.NotNil(t, tc.checkField(result2))
				assert.Contains(t, tc.getDescription(result2), "run 2")

				// Ensure we got different artifacts
				assert.NotEqual(t, tc.getDescription(result1), tc.getDescription(result2))
			})
		}
	})
}

func TestGetArtifacts(t *testing.T) {
	_service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	t.Run("successful list all artifacts", func(t *testing.T) {
		// Create multiple artifacts
		artifacts := []*openapi.Artifact{
			{
				ModelArtifact: &openapi.ModelArtifact{
					Name: new("list-artifact-1"),
					Uri:  new("s3://bucket/artifact1.pkl"),
				},
			},
			{
				ModelArtifact: &openapi.ModelArtifact{
					Name: new("list-artifact-2"),
					Uri:  new("s3://bucket/artifact2.pkl"),
				},
			},
			{
				DocArtifact: &openapi.DocArtifact{
					Name: new("list-doc-artifact"),
					Uri:  new("s3://bucket/doc.pdf"),
				},
			},
		}

		for _, artifact := range artifacts {
			_, err := _service.UpsertArtifact(artifact)
			require.NoError(t, err)
		}

		// List all artifacts
		listOptions := api.ListOptions{
			PageSize: new(int32(10)),
		}

		result, err := _service.GetArtifacts(openapi.ARTIFACTTYPEQUERYPARAM_MODEL_ARTIFACT, listOptions, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.GreaterOrEqual(t, len(result.Items), 2)
		assert.NotNil(t, result.Size)
		assert.Equal(t, int32(10), result.PageSize)
	})

	t.Run("successful list artifacts by model version", func(t *testing.T) {
		// Create registered model and model version
		registeredModel := &openapi.RegisteredModel{
			Name: "test-model-for-list",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		modelVersion := &openapi.ModelVersion{
			Name: "v1.0",
		}
		createdVersion, err := _service.UpsertModelVersion(modelVersion, createdModel.Id)
		require.NoError(t, err)

		// Create artifacts for this model version
		for i := range 3 {
			artifact := &openapi.Artifact{
				ModelArtifact: &openapi.ModelArtifact{
					Name: new("version-artifact-" + string(rune('1'+i))),
					Uri:  new("s3://bucket/version" + string(rune('1'+i)) + ".pkl"),
				},
			}
			_, err := _service.UpsertModelVersionArtifact(artifact, *createdVersion.Id)
			require.NoError(t, err)
		}

		// List artifacts for this model version
		listOptions := api.ListOptions{
			PageSize: new(int32(10)),
		}

		result, err := _service.GetArtifacts(openapi.ARTIFACTTYPEQUERYPARAM_MODEL_ARTIFACT, listOptions, createdVersion.Id)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 3, len(result.Items))
	})

	t.Run("invalid model version id", func(t *testing.T) {
		listOptions := api.ListOptions{}

		result, err := _service.GetArtifacts(openapi.ARTIFACTTYPEQUERYPARAM_MODEL_ARTIFACT, listOptions, new("invalid"))

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid syntax: bad request")
	})
}

func TestUpsertModelArtifact(t *testing.T) {
	_service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	t.Run("successful create", func(t *testing.T) {
		modelArtifact := &openapi.ModelArtifact{
			Name:               new("direct-model-artifact"),
			Description:        new("Direct model artifact"),
			Uri:                new("s3://bucket/direct.pkl"),
			ModelFormatName:    new("tensorflow"),
			ModelFormatVersion: new("2.8"),
		}

		result, err := _service.UpsertModelArtifact(modelArtifact)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotNil(t, result.Id)
		assert.Equal(t, "direct-model-artifact", *result.Name)
		assert.Equal(t, "Direct model artifact", *result.Description)
		assert.Equal(t, "s3://bucket/direct.pkl", *result.Uri)
		assert.Equal(t, "tensorflow", *result.ModelFormatName)
		assert.Equal(t, "2.8", *result.ModelFormatVersion)
	})

	t.Run("nil model artifact error", func(t *testing.T) {
		result, err := _service.UpsertModelArtifact(nil)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid model artifact pointer")
	})

	t.Run("unicode characters in model artifact name", func(t *testing.T) {
		// Test with unicode characters: Chinese, Russian, Japanese, and emoji
		unicodeName := "直接模型工件-тест-ダイレクトモデルアーティファクト-🚀"
		modelArtifact := &openapi.ModelArtifact{
			Name:               new(unicodeName),
			Description:        new("Direct model artifact with unicode characters"),
			Uri:                new("s3://bucket/unicode-direct.pkl"),
			ModelFormatName:    new("tensorflow-unicode"),
			ModelFormatVersion: new("2.8-测试"),
		}

		result, err := _service.UpsertModelArtifact(modelArtifact)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, unicodeName, *result.Name)
		assert.Equal(t, "Direct model artifact with unicode characters", *result.Description)
		assert.Equal(t, "s3://bucket/unicode-direct.pkl", *result.Uri)
		assert.Equal(t, "tensorflow-unicode", *result.ModelFormatName)
		assert.Equal(t, "2.8-测试", *result.ModelFormatVersion)
		assert.NotNil(t, result.Id)

		// Verify we can retrieve it by ID
		retrieved, err := _service.GetModelArtifactById(*result.Id)
		require.NoError(t, err)
		assert.Equal(t, unicodeName, *retrieved.Name)
		assert.Equal(t, "2.8-测试", *retrieved.ModelFormatVersion)
	})

	t.Run("special characters in model artifact name", func(t *testing.T) {
		// Test with various special characters
		specialName := "!@#$%^&*()_+-=[]{}|;':\",./<>?"
		modelArtifact := &openapi.ModelArtifact{
			Name:               new(specialName),
			Description:        new("Direct model artifact with special characters"),
			Uri:                new("s3://bucket/special-direct.pkl"),
			ModelFormatName:    new("format@#$%"),
			ModelFormatVersion: new("1.0!@#"),
		}

		result, err := _service.UpsertModelArtifact(modelArtifact)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, specialName, *result.Name)
		assert.Equal(t, "Direct model artifact with special characters", *result.Description)
		assert.Equal(t, "s3://bucket/special-direct.pkl", *result.Uri)
		assert.Equal(t, "format@#$%", *result.ModelFormatName)
		assert.Equal(t, "1.0!@#", *result.ModelFormatVersion)
		assert.NotNil(t, result.Id)

		// Verify we can retrieve it by ID
		retrieved, err := _service.GetModelArtifactById(*result.Id)
		require.NoError(t, err)
		assert.Equal(t, specialName, *retrieved.Name)
		assert.Equal(t, "format@#$%", *retrieved.ModelFormatName)
	})

	t.Run("mixed unicode and special characters in model artifact", func(t *testing.T) {
		// Test with mixed unicode and special characters
		mixedName := "直接@#$%模型-тест!@#-ダイレクト()モデル-🚀[]"
		modelArtifact := &openapi.ModelArtifact{
			Name:               new(mixedName),
			Description:        new("Direct model artifact with mixed unicode and special characters"),
			Uri:                new("s3://bucket/mixed-direct.pkl"),
			ModelFormatName:    new("tensorflow@#$%-测试"),
			ModelFormatVersion: new("2.8!@#-тест"),
		}

		result, err := _service.UpsertModelArtifact(modelArtifact)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, mixedName, *result.Name)
		assert.Equal(t, "Direct model artifact with mixed unicode and special characters", *result.Description)
		assert.Equal(t, "s3://bucket/mixed-direct.pkl", *result.Uri)
		assert.Equal(t, "tensorflow@#$%-测试", *result.ModelFormatName)
		assert.Equal(t, "2.8!@#-тест", *result.ModelFormatVersion)
		assert.NotNil(t, result.Id)

		// Verify we can retrieve it by ID
		retrieved, err := _service.GetModelArtifactById(*result.Id)
		require.NoError(t, err)
		assert.Equal(t, mixedName, *retrieved.Name)
		assert.Equal(t, "tensorflow@#$%-测试", *retrieved.ModelFormatName)
	})

	t.Run("create with null name generates UUID", func(t *testing.T) {
		modelArtifact := &openapi.ModelArtifact{
			// Name is intentionally nil
			Uri:             new("s3://bucket/direct-no-name.pkl"),
			ModelFormatName: new("tensorflow"),
		}

		result, err := _service.UpsertModelArtifact(modelArtifact)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotNil(t, result.Name, "Name should be auto-generated")
		assert.NotEmpty(t, *result.Name, "Generated name should not be empty")
		assert.Len(t, *result.Name, 36, "Generated name should be UUID length")
		assert.Contains(t, *result.Name, "-", "Generated name should have UUID format")
		assert.Equal(t, "s3://bucket/direct-no-name.pkl", *result.Uri)
		assert.Equal(t, "tensorflow", *result.ModelFormatName)
	})

	t.Run("pagination test", func(t *testing.T) {
		// Create multiple model artifacts for pagination testing
		for i := range 15 {
			modelArtifact := &openapi.ModelArtifact{
				Name: new(fmt.Sprintf("paging-test-direct-model-artifact-%d", i+1)),
				Uri:  new(fmt.Sprintf("s3://bucket/paging-direct-model-%d.pkl", i+1)),
			}

			result, err := _service.UpsertModelArtifact(modelArtifact)
			require.NoError(t, err)
			require.NotNil(t, result.Id)
		}

		// Test pagination with page size 5
		listOptions := api.ListOptions{
			PageSize: new(int32(5)),
		}

		// Get first page
		firstPage, err := _service.GetModelArtifacts(listOptions, nil)
		require.NoError(t, err)
		require.NotNil(t, firstPage)
		assert.Equal(t, 5, len(firstPage.Items))
		assert.NotNil(t, firstPage.NextPageToken)

		// Get second page
		listOptions.NextPageToken = new(firstPage.NextPageToken)
		secondPage, err := _service.GetModelArtifacts(listOptions, nil)
		require.NoError(t, err)
		require.NotNil(t, secondPage)
		assert.GreaterOrEqual(t, len(secondPage.Items), 5)

		// Verify no duplicate IDs between pages
		firstPageIds := make(map[string]bool)
		for _, item := range firstPage.Items {
			firstPageIds[*item.Id] = true
		}

		for _, item := range secondPage.Items {
			if firstPageIds[*item.Id] {
				t.Errorf("Found duplicate ID %s between pages", *item.Id)
			}
		}
	})
}

func TestGetModelArtifactById(t *testing.T) {
	_service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	t.Run("successful get", func(t *testing.T) {
		// Create a model artifact
		modelArtifact := &openapi.ModelArtifact{
			Name: new("get-model-artifact"),
			Uri:  new("s3://bucket/get-model.pkl"),
		}

		created, err := _service.UpsertModelArtifact(modelArtifact)
		require.NoError(t, err)
		require.NotNil(t, created.Id)

		// Get by ID
		result, err := _service.GetModelArtifactById(*created.Id)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, *created.Id, *result.Id)
		assert.Equal(t, "get-model-artifact", *result.Name)
		assert.Equal(t, "s3://bucket/get-model.pkl", *result.Uri)
	})

	t.Run("artifact is not model artifact", func(t *testing.T) {
		// Create a doc artifact
		docArtifact := &openapi.DocArtifact{
			Name: new("doc-not-model"),
			Uri:  new("s3://bucket/doc.pdf"),
		}

		artifact := &openapi.Artifact{
			DocArtifact: docArtifact,
		}

		created, err := _service.UpsertArtifact(artifact)
		require.NoError(t, err)
		require.NotNil(t, created.DocArtifact.Id)

		// Try to get as model artifact
		result, err := _service.GetModelArtifactById(*created.DocArtifact.Id)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "is not a model artifact")
	})

	t.Run("non-existent id", func(t *testing.T) {
		result, err := _service.GetModelArtifactById("99999")

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestGetModelArtifactByInferenceService(t *testing.T) {
	_service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	t.Run("successful get", func(t *testing.T) {
		// Create the full chain: RegisteredModel -> ModelVersion -> InferenceService -> ModelArtifact
		registeredModel := &openapi.RegisteredModel{
			Name: "inference-artifact-model",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		servingEnv := &openapi.ServingEnvironment{
			Name: "inference-artifact-env",
		}
		createdEnv, err := _service.UpsertServingEnvironment(servingEnv)
		require.NoError(t, err)

		modelVersion := &openapi.ModelVersion{
			Name: "v1.0",
		}
		createdVersion, err := _service.UpsertModelVersion(modelVersion, createdModel.Id)
		require.NoError(t, err)

		inferenceService := &openapi.InferenceService{
			Name:                 new("inference-artifact-service"),
			RegisteredModelId:    *createdModel.Id,
			ServingEnvironmentId: *createdEnv.Id,
			ModelVersionId:       createdVersion.Id,
		}
		createdInference, err := _service.UpsertInferenceService(inferenceService)
		require.NoError(t, err)

		// Create model artifact for the model version
		modelArtifact := &openapi.ModelArtifact{
			Name: new("inference-model-artifact"),
			Uri:  new("s3://bucket/inference-model.pkl"),
		}

		artifact := &openapi.Artifact{
			ModelArtifact: modelArtifact,
		}

		_, err = _service.UpsertModelVersionArtifact(artifact, *createdVersion.Id)
		require.NoError(t, err)

		// Get model artifact by inference service
		result, err := _service.GetModelArtifactByInferenceService(*createdInference.Id)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotNil(t, result.Id)
		assert.Contains(t, *result.Name, "inference-model-artifact")
		assert.Equal(t, "s3://bucket/inference-model.pkl", *result.Uri)
	})

	t.Run("no artifacts found", func(t *testing.T) {
		// Create inference service without artifacts
		registeredModel := &openapi.RegisteredModel{
			Name: "no-artifact-model",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		servingEnv := &openapi.ServingEnvironment{
			Name: "no-artifact-env",
		}
		createdEnv, err := _service.UpsertServingEnvironment(servingEnv)
		require.NoError(t, err)

		modelVersion := &openapi.ModelVersion{
			Name: "v1.0",
		}
		createdVersion, err := _service.UpsertModelVersion(modelVersion, createdModel.Id)
		require.NoError(t, err)

		inferenceService := &openapi.InferenceService{
			Name:                 new("no-artifact-service"),
			RegisteredModelId:    *createdModel.Id,
			ServingEnvironmentId: *createdEnv.Id,
			ModelVersionId:       createdVersion.Id,
		}
		createdInference, err := _service.UpsertInferenceService(inferenceService)
		require.NoError(t, err)

		// Try to get model artifact
		result, err := _service.GetModelArtifactByInferenceService(*createdInference.Id)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "no artifacts found")
	})
}

func TestGetModelArtifactByParams(t *testing.T) {
	_service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	t.Run("successful get by external id", func(t *testing.T) {
		modelArtifact := &openapi.ModelArtifact{
			Name:       new("params-model-artifact"),
			ExternalId: new("model-params-ext-123"),
			Uri:        new("s3://bucket/params-model.pkl"),
		}

		created, err := _service.UpsertModelArtifact(modelArtifact)
		require.NoError(t, err)

		// Get by external ID
		result, err := _service.GetModelArtifactByParams(nil, nil, new("model-params-ext-123"))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, *created.Id, *result.Id)
		assert.Equal(t, "model-params-ext-123", *result.ExternalId)
	})

	t.Run("artifact is not model artifact", func(t *testing.T) {
		// Create a doc artifact
		docArtifact := &openapi.DocArtifact{
			Name:       new("doc-params-artifact"),
			ExternalId: new("doc-params-ext-123"),
			Uri:        new("s3://bucket/doc-params.pdf"),
		}

		artifact := &openapi.Artifact{
			DocArtifact: docArtifact,
		}

		_, err := _service.UpsertArtifact(artifact)
		require.NoError(t, err)

		// Try to get as model artifact
		result, err := _service.GetModelArtifactByParams(nil, nil, new("doc-params-ext-123"))

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "is not a model artifact")
	})

	t.Run("same model artifact name across different model versions", func(t *testing.T) {
		// This test catches the bug where ParentResourceID was not being used to filter artifacts

		// Create a registered model
		registeredModel := &openapi.RegisteredModel{
			Name: "model-with-shared-artifacts",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		// Create first model version
		version1 := &openapi.ModelVersion{
			Name:              "version-with-shared-artifact-1",
			RegisteredModelId: *createdModel.Id,
		}
		createdVersion1, err := _service.UpsertModelVersion(version1, createdModel.Id)
		require.NoError(t, err)

		// Create second model version
		version2 := &openapi.ModelVersion{
			Name:              "version-with-shared-artifact-2",
			RegisteredModelId: *createdModel.Id,
		}
		createdVersion2, err := _service.UpsertModelVersion(version2, createdModel.Id)
		require.NoError(t, err)

		// Create model artifact "shared-artifact-name-test" for the first version
		artifact1 := &openapi.ModelArtifact{
			Name:            new("shared-artifact-name-test"),
			Uri:             new("s3://bucket/artifact-v1.pkl"),
			Description:     new("Artifact for version 1"),
			ModelFormatName: new("pickle"),
		}
		artifactWrapper1 := &openapi.Artifact{
			ModelArtifact: artifact1,
		}
		createdArtifact1, err := _service.UpsertModelVersionArtifact(artifactWrapper1, *createdVersion1.Id)
		require.NoError(t, err)

		// Create model artifact "shared-artifact-name-test" for the second version
		artifact2 := &openapi.ModelArtifact{
			Name:            new("shared-artifact-name-test"),
			Uri:             new("s3://bucket/artifact-v2.pkl"),
			Description:     new("Artifact for version 2"),
			ModelFormatName: new("pickle"),
		}
		artifactWrapper2 := &openapi.Artifact{
			ModelArtifact: artifact2,
		}
		createdArtifact2, err := _service.UpsertModelVersionArtifact(artifactWrapper2, *createdVersion2.Id)
		require.NoError(t, err)

		// Query for artifact "shared-artifact-name-test" of the first version
		artifactName := "shared-artifact-name-test"
		result1, err := _service.GetModelArtifactByParams(&artifactName, createdVersion1.Id, nil)
		require.NoError(t, err)
		require.NotNil(t, result1)
		assert.Equal(t, *createdArtifact1.ModelArtifact.Id, *result1.Id)
		assert.Equal(t, "Artifact for version 1", *result1.Description)
		assert.Equal(t, "s3://bucket/artifact-v1.pkl", *result1.Uri)

		// Query for artifact "shared-artifact-name-test" of the second version
		result2, err := _service.GetModelArtifactByParams(&artifactName, createdVersion2.Id, nil)
		require.NoError(t, err)
		require.NotNil(t, result2)
		assert.Equal(t, *createdArtifact2.ModelArtifact.Id, *result2.Id)
		assert.Equal(t, "Artifact for version 2", *result2.Description)
		assert.Equal(t, "s3://bucket/artifact-v2.pkl", *result2.Uri)

		// Ensure we got different artifacts
		assert.NotEqual(t, *result1.Id, *result2.Id)
	})
}

func TestGetModelArtifacts(t *testing.T) {
	_service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	t.Run("successful list all model artifacts", func(t *testing.T) {
		// Create multiple model artifacts
		for i := range 3 {
			modelArtifact := &openapi.ModelArtifact{
				Name: new("list-model-artifact-" + string(rune('1'+i))),
				Uri:  new("s3://bucket/model" + string(rune('1'+i)) + ".pkl"),
			}
			_, err := _service.UpsertModelArtifact(modelArtifact)
			require.NoError(t, err)
		}

		// List all model artifacts
		listOptions := api.ListOptions{
			PageSize: new(int32(10)),
		}

		result, err := _service.GetModelArtifacts(listOptions, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.GreaterOrEqual(t, len(result.Items), 3)
		assert.NotNil(t, result.Size)
		assert.Equal(t, int32(10), result.PageSize)
	})

	t.Run("successful list model artifacts by model version", func(t *testing.T) {
		// Create registered model and model version
		registeredModel := &openapi.RegisteredModel{
			Name: "test-model-for-model-artifacts",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		modelVersion := &openapi.ModelVersion{
			Name: "v1.0",
		}
		createdVersion, err := _service.UpsertModelVersion(modelVersion, createdModel.Id)
		require.NoError(t, err)

		// Create model artifacts for this model version
		for i := range 2 {
			modelArtifact := &openapi.ModelArtifact{
				Name: new("version-model-artifact-" + string(rune('1'+i))),
				Uri:  new("s3://bucket/version-model" + string(rune('1'+i)) + ".pkl"),
			}

			artifact := &openapi.Artifact{
				ModelArtifact: modelArtifact,
			}

			_, err := _service.UpsertModelVersionArtifact(artifact, *createdVersion.Id)
			require.NoError(t, err)
		}

		// List model artifacts for this model version
		listOptions := api.ListOptions{
			PageSize: new(int32(10)),
		}

		result, err := _service.GetModelArtifacts(listOptions, createdVersion.Id)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, len(result.Items))
	})

	t.Run("invalid model version id", func(t *testing.T) {
		listOptions := api.ListOptions{}

		result, err := _service.GetModelArtifacts(listOptions, new("invalid"))

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid syntax: bad request")
	})
}

func TestArtifactRoundTrip(t *testing.T) {
	_service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	t.Run("complete roundtrip", func(t *testing.T) {
		// Create registered model and model version
		registeredModel := &openapi.RegisteredModel{
			Name:        "roundtrip-model",
			Description: new("Model for roundtrip test"),
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		modelVersion := &openapi.ModelVersion{
			Name:        "v1.0",
			Description: new("Version 1.0"),
		}
		createdVersion, err := _service.UpsertModelVersion(modelVersion, createdModel.Id)
		require.NoError(t, err)

		// Create model artifact
		modelArtifact := &openapi.ModelArtifact{
			Name:               new("roundtrip-artifact"),
			Description:        new("Roundtrip test artifact"),
			Uri:                new("s3://bucket/roundtrip.pkl"),
			ModelFormatName:    new("sklearn"),
			ModelFormatVersion: new("1.0"),
			StorageKey:         new("roundtrip-key"),
			StoragePath:        new("/models/roundtrip"),
			ServiceAccountName: new("roundtrip-sa"),
		}

		artifact := &openapi.Artifact{
			ModelArtifact: modelArtifact,
		}

		// Create
		created, err := _service.UpsertModelVersionArtifact(artifact, *createdVersion.Id)
		require.NoError(t, err)
		require.NotNil(t, created.ModelArtifact.Id)

		// Get by ID
		retrieved, err := _service.GetArtifactById(*created.ModelArtifact.Id)
		require.NoError(t, err)
		require.NotNil(t, retrieved.ModelArtifact)
		assert.Equal(t, *created.ModelArtifact.Id, *retrieved.ModelArtifact.Id)
		assert.Contains(t, *retrieved.ModelArtifact.Name, "roundtrip-artifact")
		assert.Equal(t, "Roundtrip test artifact", *retrieved.ModelArtifact.Description)
		assert.Equal(t, "s3://bucket/roundtrip.pkl", *retrieved.ModelArtifact.Uri)
		assert.Equal(t, "sklearn", *retrieved.ModelArtifact.ModelFormatName)
		assert.Equal(t, "1.0", *retrieved.ModelArtifact.ModelFormatVersion)
		assert.Equal(t, "roundtrip-key", *retrieved.ModelArtifact.StorageKey)
		assert.Equal(t, "/models/roundtrip", *retrieved.ModelArtifact.StoragePath)
		assert.Equal(t, "roundtrip-sa", *retrieved.ModelArtifact.ServiceAccountName)

		// Update
		retrieved.ModelArtifact.Description = new("Updated description")
		retrieved.ModelArtifact.Uri = new("s3://bucket/updated-roundtrip.pkl")
		retrieved.ModelArtifact.State = new(openapi.ARTIFACTSTATE_DELETED)

		updated, err := _service.UpsertArtifact(retrieved)
		require.NoError(t, err)
		require.NotNil(t, updated.ModelArtifact)
		assert.Equal(t, *created.ModelArtifact.Id, *updated.ModelArtifact.Id)
		assert.Equal(t, "Updated description", *updated.ModelArtifact.Description)
		assert.Equal(t, "s3://bucket/updated-roundtrip.pkl", *updated.ModelArtifact.Uri)
		assert.Equal(t, openapi.ARTIFACTSTATE_DELETED, *updated.ModelArtifact.State)

		// List artifacts for model version
		listOptions := api.ListOptions{
			PageSize: new(int32(10)),
		}

		artifacts, err := _service.GetArtifacts(openapi.ARTIFACTTYPEQUERYPARAM_MODEL_ARTIFACT, listOptions, createdVersion.Id)
		require.NoError(t, err)
		require.NotNil(t, artifacts)
		assert.Equal(t, 1, len(artifacts.Items))
		assert.Equal(t, *updated.ModelArtifact.Id, *artifacts.Items[0].ModelArtifact.Id)
	})

	t.Run("roundtrip with custom properties", func(t *testing.T) {
		customProps := map[string]openapi.MetadataValue{
			"accuracy": {
				MetadataDoubleValue: &openapi.MetadataDoubleValue{
					DoubleValue: 0.95,
				},
			},
			"framework": {
				MetadataStringValue: &openapi.MetadataStringValue{
					StringValue: "tensorflow",
				},
			},
			"epochs": {
				MetadataIntValue: &openapi.MetadataIntValue{
					IntValue: "100",
				},
			},
			"is_production": {
				MetadataBoolValue: &openapi.MetadataBoolValue{
					BoolValue: true,
				},
			},
		}

		modelArtifact := &openapi.ModelArtifact{
			Name:             new("custom-props-roundtrip"),
			Uri:              new("s3://bucket/custom-props.pkl"),
			CustomProperties: customProps,
		}

		// Create
		created, err := _service.UpsertModelArtifact(modelArtifact)
		require.NoError(t, err)
		require.NotNil(t, created.Id)

		// Verify custom properties
		retrieved, err := _service.GetModelArtifactById(*created.Id)
		require.NoError(t, err)
		require.NotNil(t, retrieved.CustomProperties)

		resultProps := retrieved.CustomProperties
		assert.Contains(t, resultProps, "accuracy")
		assert.Contains(t, resultProps, "framework")
		assert.Contains(t, resultProps, "epochs")
		assert.Contains(t, resultProps, "is_production")

		assert.Equal(t, 0.95, resultProps["accuracy"].MetadataDoubleValue.DoubleValue)
		assert.Equal(t, "tensorflow", resultProps["framework"].MetadataStringValue.StringValue)
		assert.Equal(t, "100", resultProps["epochs"].MetadataIntValue.IntValue)
		assert.Equal(t, true, resultProps["is_production"].MetadataBoolValue.BoolValue)

		// Update custom properties
		newProps := map[string]openapi.MetadataValue{
			"accuracy": {
				MetadataDoubleValue: &openapi.MetadataDoubleValue{
					DoubleValue: 0.97,
				},
			},
			"new_prop": {
				MetadataStringValue: &openapi.MetadataStringValue{
					StringValue: "new_value",
				},
			},
		}

		retrieved.CustomProperties = newProps

		updated, err := _service.UpsertModelArtifact(retrieved)
		require.NoError(t, err)
		require.NotNil(t, updated.CustomProperties)

		updatedProps := updated.CustomProperties
		assert.Contains(t, updatedProps, "accuracy")
		assert.Contains(t, updatedProps, "new_prop")
		assert.Equal(t, 0.97, updatedProps["accuracy"].MetadataDoubleValue.DoubleValue)
		assert.Equal(t, "new_value", updatedProps["new_prop"].MetadataStringValue.StringValue)
	})
}

func TestModelArtifactNilFieldsPreservation(t *testing.T) {
	_service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	t.Run("nil fields preserved during model artifact upsert", func(t *testing.T) {
		// Create model artifact with only required fields, leaving optional fields as nil
		modelArtifact := &openapi.ModelArtifact{
			Name: new("nil-fields-test"),
			Uri:  new("s3://bucket/test.pkl"),
			// Explicitly leaving these fields as nil:
			// Description: nil,
			// ExternalId: nil,
			// ModelFormatName: nil,
			// ModelFormatVersion: nil,
			// StorageKey: nil,
			// StoragePath: nil,
			// ServiceAccountName: nil,
			// ModelSourceKind: nil,
			// ModelSourceClass: nil,
			// ModelSourceGroup: nil,
			// ModelSourceId: nil,
			// ModelSourceName: nil,
			// State: nil (will get default),
		}

		// Create the artifact
		created, err := _service.UpsertModelArtifact(modelArtifact)
		require.NoError(t, err)
		require.NotNil(t, created.Id)

		// Verify nil fields are preserved (not set to default values)
		assert.Nil(t, created.Description)
		assert.Nil(t, created.ExternalId)
		assert.Nil(t, created.ModelFormatName)
		assert.Nil(t, created.ModelFormatVersion)
		assert.Nil(t, created.StorageKey)
		assert.Nil(t, created.StoragePath)
		assert.Nil(t, created.ServiceAccountName)
		assert.Nil(t, created.ModelSourceKind)
		assert.Nil(t, created.ModelSourceClass)
		assert.Nil(t, created.ModelSourceGroup)
		assert.Nil(t, created.ModelSourceId)
		assert.Nil(t, created.ModelSourceName)

		// Update the artifact while keeping nil fields as nil
		created.Uri = new("s3://bucket/updated.pkl")
		// Keep all other optional fields as nil

		updated, err := _service.UpsertModelArtifact(created)
		require.NoError(t, err)

		// Verify nil fields are still preserved after update
		assert.Equal(t, "s3://bucket/updated.pkl", *updated.Uri)
		assert.Nil(t, updated.Description)
		assert.Nil(t, updated.ExternalId)
		assert.Nil(t, updated.ModelFormatName)
		assert.Nil(t, updated.ModelFormatVersion)
		assert.Nil(t, updated.StorageKey)
		assert.Nil(t, updated.StoragePath)
		assert.Nil(t, updated.ServiceAccountName)
		assert.Nil(t, updated.ModelSourceKind)
		assert.Nil(t, updated.ModelSourceClass)
		assert.Nil(t, updated.ModelSourceGroup)
		assert.Nil(t, updated.ModelSourceId)
		assert.Nil(t, updated.ModelSourceName)
	})
}

func TestDocArtifactNilFieldsPreservation(t *testing.T) {
	_service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	t.Run("nil fields preserved during doc artifact upsert", func(t *testing.T) {
		// Create doc artifact with only required fields, leaving optional fields as nil
		docArtifact := &openapi.DocArtifact{
			Name: new("nil-fields-doc-test"),
			Uri:  new("s3://bucket/doc.pdf"),
			// Explicitly leaving these fields as nil:
			// Description: nil,
			// ExternalId: nil,
			// State: nil (will get default),
		}

		artifact := &openapi.Artifact{
			DocArtifact: docArtifact,
		}

		// Create the artifact
		created, err := _service.UpsertArtifact(artifact)
		require.NoError(t, err)
		require.NotNil(t, created.DocArtifact.Id)

		// Verify nil fields are preserved (not set to default values)
		assert.Nil(t, created.DocArtifact.Description)
		assert.Nil(t, created.DocArtifact.ExternalId)

		// Update the artifact while keeping nil fields as nil
		created.DocArtifact.Uri = new("s3://bucket/updated-doc.pdf")
		// Keep all other optional fields as nil
		updated, err := _service.UpsertArtifact(created)
		require.NoError(t, err)

		// Verify nil fields are still preserved after update
		assert.Equal(t, "s3://bucket/updated-doc.pdf", *updated.DocArtifact.Uri)
		assert.Nil(t, updated.DocArtifact.Description)
		assert.Nil(t, updated.DocArtifact.ExternalId)
	})
}

func TestArtifactTypeFiltering(t *testing.T) {
	service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	// Setup: Create a registered model, model version, and experiment + experiment run
	registeredModel := &openapi.RegisteredModel{
		Name: "artifact-type-test-model",
	}
	createdModel, err := service.UpsertRegisteredModel(registeredModel)
	require.NoError(t, err)

	modelVersion := &openapi.ModelVersion{
		Name: "v1.0",
	}
	createdVersion, err := service.UpsertModelVersion(modelVersion, createdModel.Id)
	require.NoError(t, err)

	experiment := &openapi.Experiment{
		Name: "artifact-type-test-experiment",
	}
	createdExperiment, err := service.UpsertExperiment(experiment)
	require.NoError(t, err)

	experimentRun := &openapi.ExperimentRun{
		Name: new("artifact-type-test-run"),
	}
	createdExperimentRun, err := service.UpsertExperimentRun(experimentRun, createdExperiment.Id)
	require.NoError(t, err)

	// Create one artifact of each type for general testing
	t.Run("setup artifacts", func(t *testing.T) {
		// Create ModelArtifact
		modelArtifact := &openapi.Artifact{
			ModelArtifact: &openapi.ModelArtifact{
				Name: new("test-model-artifact"),
				Uri:  new("s3://bucket/model.pkl"),
			},
		}
		_, err := service.UpsertArtifact(modelArtifact)
		require.NoError(t, err)

		// Create DocArtifact
		docArtifact := &openapi.Artifact{
			DocArtifact: &openapi.DocArtifact{
				Name: new("test-doc-artifact"),
				Uri:  new("s3://bucket/doc.pdf"),
			},
		}
		_, err = service.UpsertArtifact(docArtifact)
		require.NoError(t, err)

		// Create DataSet
		dataSet := &openapi.Artifact{
			DataSet: &openapi.DataSet{
				Name: new("test-dataset-artifact"),
				Uri:  new("s3://bucket/dataset.csv"),
			},
		}
		_, err = service.UpsertArtifact(dataSet)
		require.NoError(t, err)

		// Create Metric
		metric := &openapi.Artifact{
			Metric: &openapi.Metric{
				Name:  new("test-metric-artifact"),
				Value: new(0.95),
			},
		}
		_, err = service.UpsertArtifact(metric)
		require.NoError(t, err)

		// Create Parameter
		parameter := &openapi.Artifact{
			Parameter: &openapi.Parameter{
				Name:  new("test-parameter-artifact"),
				Value: new("param-value"),
			},
		}
		_, err = service.UpsertArtifact(parameter)
		require.NoError(t, err)
	})

	// Test all artifact types for GetArtifacts (general endpoint)
	t.Run("GetArtifacts endpoint filtering", func(t *testing.T) {
		testCases := []struct {
			name         string
			artifactType openapi.ArtifactTypeQueryParam
			expectField  string
		}{
			{"model-artifact filter", openapi.ARTIFACTTYPEQUERYPARAM_MODEL_ARTIFACT, "ModelArtifact"},
			{"doc-artifact filter", openapi.ARTIFACTTYPEQUERYPARAM_DOC_ARTIFACT, "DocArtifact"},
			{"dataset-artifact filter", openapi.ARTIFACTTYPEQUERYPARAM_DATASET_ARTIFACT, "DataSet"},
			{"metric filter", openapi.ARTIFACTTYPEQUERYPARAM_METRIC, "Metric"},
			{"parameter filter", openapi.ARTIFACTTYPEQUERYPARAM_PARAMETER, "Parameter"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				listOptions := api.ListOptions{
					PageSize: new(int32(100)),
				}

				result, err := service.GetArtifacts(tc.artifactType, listOptions, nil)
				require.NoError(t, err)
				require.NotNil(t, result)

				// Should have at least one artifact of the specified type
				assert.GreaterOrEqual(t, len(result.Items), 1, "Should find at least one artifact of type %s", tc.artifactType)

				// Verify all returned artifacts are of the correct type
				for i, artifact := range result.Items {
					switch tc.expectField {
					case "ModelArtifact":
						assert.NotNil(t, artifact.ModelArtifact, "Artifact %d should be ModelArtifact", i)
						assert.Equal(t, string(openapi.ARTIFACTTYPEQUERYPARAM_MODEL_ARTIFACT), *artifact.ModelArtifact.ArtifactType)
						assert.Nil(t, artifact.DocArtifact, "Artifact %d should not be DocArtifact", i)
						assert.Nil(t, artifact.DataSet, "Artifact %d should not be DataSet", i)
						assert.Nil(t, artifact.Metric, "Artifact %d should not be Metric", i)
						assert.Nil(t, artifact.Parameter, "Artifact %d should not be Parameter", i)
					case "DocArtifact":
						assert.Nil(t, artifact.ModelArtifact, "Artifact %d should not be ModelArtifact", i)
						assert.NotNil(t, artifact.DocArtifact, "Artifact %d should be DocArtifact", i)
						assert.Equal(t, string(openapi.ARTIFACTTYPEQUERYPARAM_DOC_ARTIFACT), *artifact.DocArtifact.ArtifactType)
						assert.Nil(t, artifact.DataSet, "Artifact %d should not be DataSet", i)
						assert.Nil(t, artifact.Metric, "Artifact %d should not be Metric", i)
						assert.Nil(t, artifact.Parameter, "Artifact %d should not be Parameter", i)
					case "DataSet":
						assert.Nil(t, artifact.ModelArtifact, "Artifact %d should not be ModelArtifact", i)
						assert.Nil(t, artifact.DocArtifact, "Artifact %d should not be DocArtifact", i)
						assert.NotNil(t, artifact.DataSet, "Artifact %d should be DataSet", i)
						assert.Equal(t, string(openapi.ARTIFACTTYPEQUERYPARAM_DATASET_ARTIFACT), *artifact.DataSet.ArtifactType)
						assert.Nil(t, artifact.Metric, "Artifact %d should not be Metric", i)
						assert.Nil(t, artifact.Parameter, "Artifact %d should not be Parameter", i)
					case "Metric":
						assert.Nil(t, artifact.ModelArtifact, "Artifact %d should not be ModelArtifact", i)
						assert.Nil(t, artifact.DocArtifact, "Artifact %d should not be DocArtifact", i)
						assert.Nil(t, artifact.DataSet, "Artifact %d should not be DataSet", i)
						assert.NotNil(t, artifact.Metric, "Artifact %d should be Metric", i)
						assert.Equal(t, string(openapi.ARTIFACTTYPEQUERYPARAM_METRIC), *artifact.Metric.ArtifactType)
						assert.Nil(t, artifact.Parameter, "Artifact %d should not be Parameter", i)
					case "Parameter":
						assert.Nil(t, artifact.ModelArtifact, "Artifact %d should not be ModelArtifact", i)
						assert.Nil(t, artifact.DocArtifact, "Artifact %d should not be DocArtifact", i)
						assert.Nil(t, artifact.DataSet, "Artifact %d should not be DataSet", i)
						assert.Nil(t, artifact.Metric, "Artifact %d should not be Metric", i)
						assert.NotNil(t, artifact.Parameter, "Artifact %d should be Parameter", i)
						assert.Equal(t, string(openapi.ARTIFACTTYPEQUERYPARAM_PARAMETER), *artifact.Parameter.ArtifactType)
					}
				}
			})
		}

		// Test empty filter returns all types
		t.Run("no filter returns all types", func(t *testing.T) {
			listOptions := api.ListOptions{
				PageSize: new(int32(100)),
			}

			result, err := service.GetArtifacts("", listOptions, nil)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Should have at least 5 artifacts (ModelArtifact, DocArtifact, DataSet, Metric, Parameter)
			assert.GreaterOrEqual(t, len(result.Items), 5, "Should find artifacts of all types when no filter is applied")
		})
	})

	// Create artifacts specifically associated with model version
	t.Run("setup model version artifacts", func(t *testing.T) {
		// Create different types of artifacts for the model version
		artifacts := []*openapi.Artifact{
			{
				ModelArtifact: &openapi.ModelArtifact{
					Name: new("mv-model-artifact"),
					Uri:  new("s3://bucket/mv-model.pkl"),
				},
			},
			{
				DocArtifact: &openapi.DocArtifact{
					Name: new("mv-doc-artifact"),
					Uri:  new("s3://bucket/mv-doc.pdf"),
				},
			},
			{
				DataSet: &openapi.DataSet{
					Name: new("mv-dataset-artifact"),
					Uri:  new("s3://bucket/mv-dataset.csv"),
				},
			},
			{
				Metric: &openapi.Metric{
					Name:  new("mv-metric-artifact"),
					Value: new(0.95),
				},
			},
			{
				Parameter: &openapi.Parameter{
					Name:  new("mv-parameter-artifact"),
					Value: new("mv-param-value"),
				},
			},
		}

		for _, artifact := range artifacts {
			_, err := service.UpsertModelVersionArtifact(artifact, *createdVersion.Id)
			require.NoError(t, err)
		}
	})

	// Test all artifact types for GetArtifacts with model version (scoped endpoint)
	t.Run("GetArtifacts with model version filtering", func(t *testing.T) {
		testCases := []struct {
			name         string
			artifactType openapi.ArtifactTypeQueryParam
			expectField  string
			expectCount  int
		}{
			{"model-artifact filter", openapi.ARTIFACTTYPEQUERYPARAM_MODEL_ARTIFACT, "ModelArtifact", 1},
			{"doc-artifact filter", openapi.ARTIFACTTYPEQUERYPARAM_DOC_ARTIFACT, "DocArtifact", 1},
			{"dataset-artifact filter", openapi.ARTIFACTTYPEQUERYPARAM_DATASET_ARTIFACT, "DataSet", 1},
			{"metric filter", openapi.ARTIFACTTYPEQUERYPARAM_METRIC, "Metric", 1},
			{"parameter filter", openapi.ARTIFACTTYPEQUERYPARAM_PARAMETER, "Parameter", 1},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				listOptions := api.ListOptions{
					PageSize: new(int32(100)),
				}

				result, err := service.GetArtifacts(tc.artifactType, listOptions, createdVersion.Id)
				require.NoError(t, err)
				require.NotNil(t, result)

				assert.Equal(t, tc.expectCount, len(result.Items), "Should find exactly %d artifacts of type %s for this model version", tc.expectCount, tc.artifactType)

				// Verify all returned artifacts are of the correct type (if any)
				for i, artifact := range result.Items {
					switch tc.expectField {
					case "ModelArtifact":
						assert.NotNil(t, artifact.ModelArtifact, "Artifact %d should be ModelArtifact", i)
						assert.Nil(t, artifact.DocArtifact, "Artifact %d should not be DocArtifact", i)
						assert.Nil(t, artifact.DataSet, "Artifact %d should not be DataSet", i)
						assert.Nil(t, artifact.Metric, "Artifact %d should not be Metric", i)
						assert.Nil(t, artifact.Parameter, "Artifact %d should not be Parameter", i)
					case "DocArtifact":
						assert.Nil(t, artifact.ModelArtifact, "Artifact %d should not be ModelArtifact", i)
						assert.NotNil(t, artifact.DocArtifact, "Artifact %d should be DocArtifact", i)
						assert.Nil(t, artifact.DataSet, "Artifact %d should not be DataSet", i)
						assert.Nil(t, artifact.Metric, "Artifact %d should not be Metric", i)
						assert.Nil(t, artifact.Parameter, "Artifact %d should not be Parameter", i)
					case "DataSet":
						assert.Nil(t, artifact.ModelArtifact, "Artifact %d should not be ModelArtifact", i)
						assert.Nil(t, artifact.DocArtifact, "Artifact %d should not be DocArtifact", i)
						assert.NotNil(t, artifact.DataSet, "Artifact %d should be DataSet", i)
						assert.Nil(t, artifact.Metric, "Artifact %d should not be Metric", i)
						assert.Nil(t, artifact.Parameter, "Artifact %d should not be Parameter", i)
					case "Metric":
						assert.Nil(t, artifact.ModelArtifact, "Artifact %d should not be ModelArtifact", i)
						assert.Nil(t, artifact.DocArtifact, "Artifact %d should not be DocArtifact", i)
						assert.Nil(t, artifact.DataSet, "Artifact %d should not be DataSet", i)
						assert.NotNil(t, artifact.Metric, "Artifact %d should be Metric", i)
						assert.Nil(t, artifact.Parameter, "Artifact %d should not be Parameter", i)
					case "Parameter":
						assert.Nil(t, artifact.ModelArtifact, "Artifact %d should not be ModelArtifact", i)
						assert.Nil(t, artifact.DocArtifact, "Artifact %d should not be DocArtifact", i)
						assert.Nil(t, artifact.DataSet, "Artifact %d should not be DataSet", i)
						assert.Nil(t, artifact.Metric, "Artifact %d should not be Metric", i)
						assert.NotNil(t, artifact.Parameter, "Artifact %d should be Parameter", i)
					}
				}
			})
		}
	})

	// Create artifacts specifically associated with experiment run
	t.Run("setup experiment run artifacts", func(t *testing.T) {
		// Create different types of artifacts for the experiment run
		artifacts := []*openapi.Artifact{
			{
				ModelArtifact: &openapi.ModelArtifact{
					Name: new("er-model-artifact"),
					Uri:  new("s3://bucket/er-model.pkl"),
				},
			},
			{
				DocArtifact: &openapi.DocArtifact{
					Name: new("er-doc-artifact"),
					Uri:  new("s3://bucket/er-doc.pdf"),
				},
			},
			{
				DataSet: &openapi.DataSet{
					Name: new("er-dataset-artifact"),
					Uri:  new("s3://bucket/er-dataset.csv"),
				},
			},
			{
				Metric: &openapi.Metric{
					Name:  new("er-metric-artifact"),
					Value: new(0.85),
				},
			},
			{
				Parameter: &openapi.Parameter{
					Name:  new("er-parameter-artifact"),
					Value: new("er-param-value"),
				},
			},
		}

		for _, artifact := range artifacts {
			_, err := service.UpsertExperimentRunArtifact(artifact, *createdExperimentRun.Id)
			require.NoError(t, err)
		}

		// Create multiple metric values to generate metric history records
		metricName := "er-accuracy-history"
		values := []float64{0.1, 0.5, 0.8, 0.95}
		for i, value := range values {
			metricArtifact := &openapi.Artifact{
				Metric: &openapi.Metric{
					Name:        new(metricName),
					Value:       new(value),
					Description: new(fmt.Sprintf("Accuracy step %d", i+1)),
				},
			}
			_, err := service.UpsertExperimentRunArtifact(metricArtifact, *createdExperimentRun.Id)
			require.NoError(t, err)
		}
	})

	// Test all artifact types for GetExperimentRunArtifacts (scoped endpoint)
	t.Run("GetExperimentRunArtifacts filtering", func(t *testing.T) {
		testCases := []struct {
			name         string
			artifactType openapi.ArtifactTypeQueryParam
			expectField  string
			expectCount  int
		}{
			{"model-artifact filter", openapi.ARTIFACTTYPEQUERYPARAM_MODEL_ARTIFACT, "ModelArtifact", 1},
			{"doc-artifact filter", openapi.ARTIFACTTYPEQUERYPARAM_DOC_ARTIFACT, "DocArtifact", 1},
			{"dataset-artifact filter", openapi.ARTIFACTTYPEQUERYPARAM_DATASET_ARTIFACT, "DataSet", 1},
			{"metric filter", openapi.ARTIFACTTYPEQUERYPARAM_METRIC, "Metric", 2},
			{"parameter filter", openapi.ARTIFACTTYPEQUERYPARAM_PARAMETER, "Parameter", 1},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				listOptions := api.ListOptions{
					PageSize: new(int32(100)),
				}

				result, err := service.GetExperimentRunArtifacts(tc.artifactType, listOptions, createdExperimentRun.Id)
				require.NoError(t, err)
				require.NotNil(t, result)

				assert.Equal(t, tc.expectCount, len(result.Items), "Should find exactly %d artifacts of type %s for this experiment run", tc.expectCount, tc.artifactType)

				// Verify all returned artifacts are of the correct type (if any)
				for i, artifact := range result.Items {
					switch tc.expectField {
					case "ModelArtifact":
						assert.NotNil(t, artifact.ModelArtifact, "Artifact %d should be ModelArtifact", i)
						assert.Nil(t, artifact.DocArtifact, "Artifact %d should not be DocArtifact", i)
						assert.Nil(t, artifact.DataSet, "Artifact %d should not be DataSet", i)
						assert.Nil(t, artifact.Metric, "Artifact %d should not be Metric", i)
						assert.Nil(t, artifact.Parameter, "Artifact %d should not be Parameter", i)
					case "DocArtifact":
						assert.Nil(t, artifact.ModelArtifact, "Artifact %d should not be ModelArtifact", i)
						assert.NotNil(t, artifact.DocArtifact, "Artifact %d should be DocArtifact", i)
						assert.Nil(t, artifact.DataSet, "Artifact %d should not be DataSet", i)
						assert.Nil(t, artifact.Metric, "Artifact %d should not be Metric", i)
						assert.Nil(t, artifact.Parameter, "Artifact %d should not be Parameter", i)
					case "DataSet":
						assert.Nil(t, artifact.ModelArtifact, "Artifact %d should not be ModelArtifact", i)
						assert.Nil(t, artifact.DocArtifact, "Artifact %d should not be DocArtifact", i)
						assert.NotNil(t, artifact.DataSet, "Artifact %d should be DataSet", i)
						assert.Nil(t, artifact.Metric, "Artifact %d should not be Metric", i)
						assert.Nil(t, artifact.Parameter, "Artifact %d should not be Parameter", i)
					case "Metric":
						assert.Nil(t, artifact.ModelArtifact, "Artifact %d should not be ModelArtifact", i)
						assert.Nil(t, artifact.DocArtifact, "Artifact %d should not be DocArtifact", i)
						assert.Nil(t, artifact.DataSet, "Artifact %d should not be DataSet", i)
						assert.NotNil(t, artifact.Metric, "Artifact %d should be Metric", i)
						assert.Nil(t, artifact.Parameter, "Artifact %d should not be Parameter", i)
					case "Parameter":
						assert.Nil(t, artifact.ModelArtifact, "Artifact %d should not be ModelArtifact", i)
						assert.Nil(t, artifact.DocArtifact, "Artifact %d should not be DocArtifact", i)
						assert.Nil(t, artifact.DataSet, "Artifact %d should not be DataSet", i)
						assert.Nil(t, artifact.Metric, "Artifact %d should not be Metric", i)
						assert.NotNil(t, artifact.Parameter, "Artifact %d should be Parameter", i)
					}
				}
			})
		}
	})

	// Test edge cases
	t.Run("edge cases", func(t *testing.T) {
		t.Run("invalid artifact type", func(t *testing.T) {
			listOptions := api.ListOptions{
				PageSize: new(int32(100)),
			}

			result, err := service.GetArtifacts("invalid-artifact-type", listOptions, nil)
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "invalid artifact type")
		})

		t.Run("empty result with valid filter", func(t *testing.T) {
			// Create a new model version with no artifacts
			emptyModel := &openapi.RegisteredModel{
				Name: "empty-test-model",
			}
			createdEmptyModel, err := service.UpsertRegisteredModel(emptyModel)
			require.NoError(t, err)

			emptyModelVersion := &openapi.ModelVersion{
				Name: "v1.0",
			}
			createdEmptyVersion, err := service.UpsertModelVersion(emptyModelVersion, createdEmptyModel.Id)
			require.NoError(t, err)

			listOptions := api.ListOptions{
				PageSize: new(int32(100)),
			}

			result, err := service.GetArtifacts(openapi.ARTIFACTTYPEQUERYPARAM_MODEL_ARTIFACT, listOptions, createdEmptyVersion.Id)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, 0, len(result.Items), "Should find no artifacts for empty model version")
		})
	})

	// Test that metric history records are NOT returned as artifacts
	t.Run("metric history filtering", func(t *testing.T) {
		// Verify that GetExperimentRunArtifacts does NOT return metric history records
		listOptions := api.ListOptions{
			PageSize: new(int32(100)),
		}

		result, err := service.GetExperimentRunArtifacts("", listOptions, createdExperimentRun.Id)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Count artifacts by type - should have exactly 6 artifacts:
		// 1 ModelArtifact, 1 DocArtifact, 1 DataSet, 2 Metrics (er-metric-artifact + er-accuracy-history), 1 Parameter
		// NOTE: Should NOT have 4 additional metric history records
		var modelCount, docCount, datasetCount, metricCount, parameterCount int
		metricNames := make([]string, 0)

		for _, artifact := range result.Items {
			switch {
			case artifact.ModelArtifact != nil:
				modelCount++
			case artifact.DocArtifact != nil:
				docCount++
			case artifact.DataSet != nil:
				datasetCount++
			case artifact.Metric != nil:
				metricCount++
				metricNames = append(metricNames, *artifact.Metric.Name)
			case artifact.Parameter != nil:
				parameterCount++
			}
		}

		assert.Equal(t, 1, modelCount, "Should have exactly 1 ModelArtifact")
		assert.Equal(t, 1, docCount, "Should have exactly 1 DocArtifact")
		assert.Equal(t, 1, datasetCount, "Should have exactly 1 DataSet")
		assert.Equal(t, 2, metricCount, "Should have exactly 2 Metrics (not 6 with history records)")
		assert.Equal(t, 1, parameterCount, "Should have exactly 1 Parameter")

		// Verify the metric names are the expected ones (current metrics, not history)
		expectedMetricNames := []string{"er-metric-artifact", "er-accuracy-history"}
		assert.ElementsMatch(t, expectedMetricNames, metricNames, "Should only have current metric artifacts, not history records")

		// Total should be 6 artifacts, not 10 (6 + 4 history records)
		assert.Equal(t, 6, len(result.Items), "Should have exactly 6 artifacts total (no metric history records)")

		// Verify metric history is still accessible via dedicated endpoint
		metricName := "er-accuracy-history"
		metricHistory, err := service.GetExperimentRunMetricHistory(&metricName, nil, api.ListOptions{}, createdExperimentRun.Id)
		require.NoError(t, err)
		require.NotNil(t, metricHistory)

		// Should have all 4 history values
		assert.Equal(t, 4, len(metricHistory.Items), "Metric history endpoint should return all 4 history records")

		// Verify values are correct
		expectedValues := []float64{0.1, 0.5, 0.8, 0.95}
		for i, historyItem := range metricHistory.Items {
			assert.Equal(t, expectedValues[i], *historyItem.Value,
				fmt.Sprintf("History item %d should have value %f", i, expectedValues[i]))
		}
	})
}

func TestEmbedMDMetricDuplicateHandling(t *testing.T) {
	service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	// Create experiment
	experiment := &openapi.Experiment{
		Name:        "test-experiment-duplicate-metrics",
		Description: new("Test experiment for duplicate metric handling"),
	}
	savedExperiment, err := service.UpsertExperiment(experiment)
	require.NoError(t, err)

	// Create experiment run
	experimentRun := &openapi.ExperimentRun{
		Name:        new("test-experiment-run-duplicate-metrics"),
		Description: new("Test experiment run for duplicate metric handling"),
	}
	savedExperimentRun, err := service.UpsertExperimentRun(experimentRun, savedExperiment.Id)
	require.NoError(t, err)

	// Create first metric
	firstMetric := &openapi.Artifact{
		Metric: &openapi.Metric{
			Name:        new("accuracy"),
			Value:       new(0.85),
			Timestamp:   new("1234567890"),
			Step:        new(int64(1)),
			Description: new("First accuracy measurement"),
		},
	}

	// Upsert the first metric
	firstResult, err := service.UpsertExperimentRunArtifact(firstMetric, *savedExperimentRun.Id)
	require.NoError(t, err, "error creating first metric")
	require.NotNil(t, firstResult.Metric)
	firstMetricId := firstResult.Metric.Id

	// Create second metric with same name but different value
	secondMetric := &openapi.Artifact{
		Metric: &openapi.Metric{
			Name:        new("accuracy"), // Same name as first metric
			Value:       new(0.92),       // Different value
			Timestamp:   new("1234567900"),
			Step:        new(int64(2)),
			Description: new("Updated accuracy measurement"),
		},
	}

	// Upsert the second metric - should update the existing one
	secondResult, err := service.UpsertExperimentRunArtifact(secondMetric, *savedExperimentRun.Id)
	require.NoError(t, err, "error creating/updating second metric")
	require.NotNil(t, secondResult.Metric)

	// Verify that it's the same metric ID (updated, not created new)
	assert.Equal(t, firstMetricId, secondResult.Metric.Id, "should update existing metric, not create new one")

	// Verify the value was updated
	assert.Equal(t, 0.92, *secondResult.Metric.Value, "metric value should be updated")
	assert.Equal(t, "Updated accuracy measurement", *secondResult.Metric.Description, "metric description should be updated")

	// Verify only one metric exists for this experiment run
	artifacts, err := service.GetExperimentRunArtifacts(openapi.ARTIFACTTYPEQUERYPARAM_METRIC, api.ListOptions{}, savedExperimentRun.Id)
	require.NoError(t, err)
	assert.Equal(t, int32(1), artifacts.Size, "should have only one metric artifact")
	assert.Equal(t, 1, len(artifacts.Items), "should have only one metric in results")

	// Verify it's the updated metric
	retrievedMetric := artifacts.Items[0].Metric
	assert.Equal(t, "accuracy", *retrievedMetric.Name)
	assert.Equal(t, 0.92, *retrievedMetric.Value)
}

func TestEmbedMDParameterDuplicateHandling(t *testing.T) {
	service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	// Create experiment
	experiment := &openapi.Experiment{
		Name:        "test-experiment-duplicate-parameters",
		Description: new("Test experiment for duplicate parameter handling"),
	}
	savedExperiment, err := service.UpsertExperiment(experiment)
	require.NoError(t, err)

	// Create experiment run
	experimentRun := &openapi.ExperimentRun{
		Name:        new("test-experiment-run-duplicate-parameters"),
		Description: new("Test experiment run for duplicate parameter handling"),
	}
	savedExperimentRun, err := service.UpsertExperimentRun(experimentRun, savedExperiment.Id)
	require.NoError(t, err)

	// Create first parameter
	firstParameter := &openapi.Artifact{
		Parameter: &openapi.Parameter{
			Name:        new("learning_rate"),
			Value:       new("0.01"),
			Description: new("Initial learning rate"),
		},
	}

	// Upsert the first parameter
	firstResult, err := service.UpsertExperimentRunArtifact(firstParameter, *savedExperimentRun.Id)
	require.NoError(t, err, "error creating first parameter")
	require.NotNil(t, firstResult.Parameter)
	firstParameterId := firstResult.Parameter.Id

	// Create second parameter with same name but different value
	secondParameter := &openapi.Artifact{
		Parameter: &openapi.Parameter{
			Name:        new("learning_rate"), // Same name as first parameter
			Value:       new("0.001"),         // Different value
			Description: new("Updated learning rate"),
		},
	}

	// Upsert the second parameter - should update the existing one
	secondResult, err := service.UpsertExperimentRunArtifact(secondParameter, *savedExperimentRun.Id)
	require.NoError(t, err, "error creating/updating second parameter")
	require.NotNil(t, secondResult.Parameter)

	// Verify that it's the same parameter ID (updated, not created new)
	assert.Equal(t, firstParameterId, secondResult.Parameter.Id, "should update existing parameter, not create new one")

	// Verify the value was updated
	assert.Equal(t, "0.001", *secondResult.Parameter.Value, "parameter value should be updated")
	assert.Equal(t, "Updated learning rate", *secondResult.Parameter.Description, "parameter description should be updated")

	// Verify only one parameter exists for this experiment run
	artifacts, err := service.GetExperimentRunArtifacts(openapi.ARTIFACTTYPEQUERYPARAM_PARAMETER, api.ListOptions{}, savedExperimentRun.Id)
	require.NoError(t, err)
	assert.Equal(t, int32(1), artifacts.Size, "should have only one parameter artifact")
	assert.Equal(t, 1, len(artifacts.Items), "should have only one parameter in results")

	// Verify it's the updated parameter
	retrievedParameter := artifacts.Items[0].Parameter
	assert.Equal(t, "learning_rate", *retrievedParameter.Name)
	assert.Equal(t, "0.001", *retrievedParameter.Value)
}

func TestArtifactFilterQuery(t *testing.T) {
	service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	// Setup: Create experiments, experiment runs, and artifacts with different experimentId/experimentRunId values
	experiment1 := &openapi.Experiment{
		Name: "filter-test-experiment-1",
	}
	createdExperiment1, err := service.UpsertExperiment(experiment1)
	require.NoError(t, err)

	experiment2 := &openapi.Experiment{
		Name: "filter-test-experiment-2",
	}
	createdExperiment2, err := service.UpsertExperiment(experiment2)
	require.NoError(t, err)

	experimentRun1 := &openapi.ExperimentRun{
		Name: new("filter-test-run-1"),
	}
	createdExperimentRun1, err := service.UpsertExperimentRun(experimentRun1, createdExperiment1.Id)
	require.NoError(t, err)

	experimentRun2 := &openapi.ExperimentRun{
		Name: new("filter-test-run-2"),
	}
	createdExperimentRun2, err := service.UpsertExperimentRun(experimentRun2, createdExperiment1.Id)
	require.NoError(t, err)

	experimentRun3 := &openapi.ExperimentRun{
		Name: new("filter-test-run-3"),
	}
	createdExperimentRun3, err := service.UpsertExperimentRun(experimentRun3, createdExperiment2.Id)
	require.NoError(t, err)

	// Create artifacts associated with different experiments and experiment runs
	// Artifacts for experiment1/run1
	artifact1 := &openapi.Artifact{
		ModelArtifact: &openapi.ModelArtifact{
			Name: new("model-exp1-run1"),
			Uri:  new("s3://bucket/model1.pkl"),
		},
	}
	createdArtifact1, err := service.UpsertExperimentRunArtifact(artifact1, *createdExperimentRun1.Id)
	require.NoError(t, err)

	// Artifacts for experiment1/run2
	artifact2 := &openapi.Artifact{
		DocArtifact: &openapi.DocArtifact{
			Name: new("doc-exp1-run2"),
			Uri:  new("s3://bucket/doc1.pdf"),
		},
	}
	createdArtifact2, err := service.UpsertExperimentRunArtifact(artifact2, *createdExperimentRun2.Id)
	require.NoError(t, err)

	// Artifacts for experiment2/run3
	artifact3 := &openapi.Artifact{
		DataSet: &openapi.DataSet{
			Name: new("dataset-exp2-run3"),
			Uri:  new("s3://bucket/dataset1.csv"),
		},
	}
	createdArtifact3, err := service.UpsertExperimentRunArtifact(artifact3, *createdExperimentRun3.Id)
	require.NoError(t, err)

	// Create a metric for experiment1/run1
	metric1 := &openapi.Artifact{
		Metric: &openapi.Metric{
			Name:  new("accuracy-exp1-run1"),
			Value: new(0.95),
		},
	}
	createdMetric1, err := service.UpsertExperimentRunArtifact(metric1, *createdExperimentRun1.Id)
	require.NoError(t, err)

	// Create a parameter for experiment2/run3
	param1 := &openapi.Artifact{
		Parameter: &openapi.Parameter{
			Name:  new("lr-exp2-run3"),
			Value: new("0.001"),
		},
	}
	createdParam1, err := service.UpsertExperimentRunArtifact(param1, *createdExperimentRun3.Id)
	require.NoError(t, err)

	// Create artifacts that are NOT associated with any experiment or experiment run
	// These should be excluded from experiment-based filters
	standaloneArtifact1 := &openapi.Artifact{
		ModelArtifact: &openapi.ModelArtifact{
			Name: new("standalone-model-artifact"),
			Uri:  new("s3://bucket/standalone-model.pkl"),
			// No experimentId or experimentRunId
		},
	}
	createdStandaloneArtifact1, err := service.UpsertArtifact(standaloneArtifact1)
	require.NoError(t, err)

	standaloneArtifact2 := &openapi.Artifact{
		DocArtifact: &openapi.DocArtifact{
			Name: new("standalone-doc-artifact"),
			Uri:  new("s3://bucket/standalone-doc.pdf"),
			// No experimentId or experimentRunId
		},
	}
	createdStandaloneArtifact2, err := service.UpsertArtifact(standaloneArtifact2)
	require.NoError(t, err)

	standaloneArtifact3 := &openapi.Artifact{
		Metric: &openapi.Metric{
			Name:  new("standalone-metric"),
			Value: new(0.75),
			// No experimentId or experimentRunId
		},
	}
	createdStandaloneArtifact3, err := service.UpsertArtifact(standaloneArtifact3)
	require.NoError(t, err)

	// Test cases for experimentId equality filtering
	t.Run("GetArtifacts with experimentId equality filter", func(t *testing.T) {
		filterQuery := fmt.Sprintf(`experimentId = "%s"`, *createdExperiment1.Id)
		listOptions := api.ListOptions{
			PageSize:    new(int32(100)),
			FilterQuery: &filterQuery,
		}

		result, err := service.GetArtifacts("", listOptions, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should find artifacts from experiment1 (artifact1, artifact2, metric1)
		assert.Equal(t, 3, len(result.Items), "Should find 3 artifacts from experiment1")

		// Verify all artifacts belong to experiment1
		for _, artifact := range result.Items {
			if artifact.ModelArtifact != nil {
				assert.Equal(t, *createdExperiment1.Id, *artifact.ModelArtifact.ExperimentId)
			} else if artifact.DocArtifact != nil {
				assert.Equal(t, *createdExperiment1.Id, *artifact.DocArtifact.ExperimentId)
			} else if artifact.Metric != nil {
				assert.Equal(t, *createdExperiment1.Id, *artifact.Metric.ExperimentId)
			}
		}
	})

	// Test cases for experimentRunId equality filtering
	t.Run("GetArtifacts with experimentRunId equality filter", func(t *testing.T) {
		filterQuery := fmt.Sprintf(`experimentRunId = "%s"`, *createdExperimentRun1.Id)
		listOptions := api.ListOptions{
			PageSize:    new(int32(100)),
			FilterQuery: &filterQuery,
		}

		result, err := service.GetArtifacts("", listOptions, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should find artifacts from experimentRun1 (artifact1, metric1)
		assert.Equal(t, 2, len(result.Items), "Should find 2 artifacts from experimentRun1")

		// Verify all artifacts belong to experimentRun1
		for _, artifact := range result.Items {
			if artifact.ModelArtifact != nil {
				assert.Equal(t, *createdExperimentRun1.Id, *artifact.ModelArtifact.ExperimentRunId)
			} else if artifact.Metric != nil {
				assert.Equal(t, *createdExperimentRun1.Id, *artifact.Metric.ExperimentRunId)
			}
		}
	})

	// Test cases for experimentId IN operator filtering
	t.Run("GetArtifacts with experimentId IN filter", func(t *testing.T) {
		filterQuery := fmt.Sprintf(`experimentId IN ("%s", "%s")`, *createdExperiment1.Id, *createdExperiment2.Id)
		listOptions := api.ListOptions{
			PageSize:    new(int32(100)),
			FilterQuery: &filterQuery,
		}

		result, err := service.GetArtifacts("", listOptions, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should find all artifacts from both experiments (5 experiment artifacts, excluding 3 standalone)
		assert.Equal(t, 5, len(result.Items), "Should find 5 artifacts from both experiments, excluding standalone artifacts")

		// Verify all artifacts belong to either experiment1 or experiment2
		experimentIds := map[string]bool{
			*createdExperiment1.Id: true,
			*createdExperiment2.Id: true,
		}
		for _, artifact := range result.Items {
			var expId string
			if artifact.ModelArtifact != nil {
				expId = *artifact.ModelArtifact.ExperimentId
			} else if artifact.DocArtifact != nil {
				expId = *artifact.DocArtifact.ExperimentId
			} else if artifact.DataSet != nil {
				expId = *artifact.DataSet.ExperimentId
			} else if artifact.Metric != nil {
				expId = *artifact.Metric.ExperimentId
			} else if artifact.Parameter != nil {
				expId = *artifact.Parameter.ExperimentId
			}
			assert.True(t, experimentIds[expId], "Artifact should belong to one of the filtered experiments")
		}
	})

	// Test cases for experimentRunId IN operator filtering
	t.Run("GetArtifacts with experimentRunId IN filter", func(t *testing.T) {
		filterQuery := fmt.Sprintf(`experimentRunId IN ("%s", "%s")`, *createdExperimentRun1.Id, *createdExperimentRun3.Id)
		listOptions := api.ListOptions{
			PageSize:    new(int32(100)),
			FilterQuery: &filterQuery,
		}

		result, err := service.GetArtifacts("", listOptions, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should find artifacts from experimentRun1 and experimentRun3 (artifact1, metric1, artifact3, param1)
		assert.Equal(t, 4, len(result.Items), "Should find 4 artifacts from specified experiment runs")

		// Verify all artifacts belong to either experimentRun1 or experimentRun3
		experimentRunIds := map[string]bool{
			*createdExperimentRun1.Id: true,
			*createdExperimentRun3.Id: true,
		}
		for _, artifact := range result.Items {
			var runId string
			if artifact.ModelArtifact != nil {
				runId = *artifact.ModelArtifact.ExperimentRunId
			} else if artifact.DataSet != nil {
				runId = *artifact.DataSet.ExperimentRunId
			} else if artifact.Metric != nil {
				runId = *artifact.Metric.ExperimentRunId
			} else if artifact.Parameter != nil {
				runId = *artifact.Parameter.ExperimentRunId
			}
			assert.True(t, experimentRunIds[runId], "Artifact should belong to one of the filtered experiment runs")
		}
	})

	// Test combined filters
	t.Run("GetArtifacts with combined experimentId and artifact type filter", func(t *testing.T) {
		filterQuery := fmt.Sprintf(`experimentId = "%s" AND name LIKE "%%model%%"`, *createdExperiment1.Id)
		listOptions := api.ListOptions{
			PageSize:    new(int32(100)),
			FilterQuery: &filterQuery,
		}

		result, err := service.GetArtifacts("", listOptions, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should find only the model artifact from experiment1
		assert.Equal(t, 1, len(result.Items), "Should find 1 model artifact from experiment1")
		assert.NotNil(t, result.Items[0].ModelArtifact, "Should be a ModelArtifact")
		assert.Equal(t, "model-exp1-run1", *result.Items[0].ModelArtifact.Name)
	})

	// Test GetModelArtifacts endpoint with filterQuery
	t.Run("GetModelArtifacts with experimentId filter", func(t *testing.T) {
		filterQuery := fmt.Sprintf(`experimentId = "%s"`, *createdExperiment1.Id)
		listOptions := api.ListOptions{
			PageSize:    new(int32(100)),
			FilterQuery: &filterQuery,
		}

		result, err := service.GetModelArtifacts(listOptions, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		require.Len(t, result.Items, 1, "Should find only the model artifact from experiment1")
		assert.Equal(t, "model-exp1-run1", *result.Items[0].Name)
		require.NotNil(t, result.Items[0].ExperimentId)
		assert.Equal(t, *createdExperiment1.Id, *result.Items[0].ExperimentId)
	})

	// Test GetExperimentRunArtifacts endpoint with filterQuery
	t.Run("GetExperimentRunArtifacts with experimentId filter", func(t *testing.T) {
		// This should work even when filtering by experimentId within a specific experiment run
		filterQuery := fmt.Sprintf(`experimentId = "%s"`, *createdExperiment1.Id)
		listOptions := api.ListOptions{
			PageSize:    new(int32(100)),
			FilterQuery: &filterQuery,
		}

		result, err := service.GetExperimentRunArtifacts("", listOptions, createdExperimentRun1.Id)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should find artifacts from experimentRun1 that also belong to experiment1
		assert.Equal(t, 2, len(result.Items), "Should find 2 artifacts from experimentRun1 with matching experimentId")

		// Verify all artifacts belong to both experimentRun1 and experiment1
		for _, artifact := range result.Items {
			if artifact.ModelArtifact != nil {
				assert.Equal(t, *createdExperiment1.Id, *artifact.ModelArtifact.ExperimentId)
				assert.Equal(t, *createdExperimentRun1.Id, *artifact.ModelArtifact.ExperimentRunId)
			} else if artifact.Metric != nil {
				assert.Equal(t, *createdExperiment1.Id, *artifact.Metric.ExperimentId)
				assert.Equal(t, *createdExperimentRun1.Id, *artifact.Metric.ExperimentRunId)
			}
		}
	})

	// Test error cases
	t.Run("Invalid filterQuery syntax", func(t *testing.T) {
		invalidFilter := "experimentId <<< invalid syntax"
		listOptions := api.ListOptions{
			PageSize:    new(int32(100)),
			FilterQuery: &invalidFilter,
		}

		result, err := service.GetArtifacts("", listOptions, nil)
		assert.Error(t, err, "Should return error for invalid filter syntax")
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid filter query")
	})

	// Test with explicit type specification
	t.Run("GetArtifacts with explicit experimentId.int_value filter", func(t *testing.T) {
		filterQuery := fmt.Sprintf(`experimentId.int_value = "%s"`, *createdExperiment2.Id)
		listOptions := api.ListOptions{
			PageSize:    new(int32(100)),
			FilterQuery: &filterQuery,
		}

		result, err := service.GetArtifacts("", listOptions, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should find artifacts from experiment2 (artifact3, param1)
		assert.Equal(t, 2, len(result.Items), "Should find 2 artifacts from experiment2")

		// Verify all artifacts belong to experiment2
		for _, artifact := range result.Items {
			if artifact.DataSet != nil {
				assert.Equal(t, *createdExperiment2.Id, *artifact.DataSet.ExperimentId)
			} else if artifact.Parameter != nil {
				assert.Equal(t, *createdExperiment2.Id, *artifact.Parameter.ExperimentId)
			}
		}
	})

	// Test that standalone artifacts are properly excluded from experiment filters
	t.Run("Verify standalone artifacts are excluded from experiment filters", func(t *testing.T) {
		// First, get all artifacts without any filter to verify we have both experiment and standalone artifacts
		listOptionsAll := api.ListOptions{
			PageSize: new(int32(100)),
		}

		allResult, err := service.GetArtifacts("", listOptionsAll, nil)
		require.NoError(t, err)
		require.NotNil(t, allResult)

		// Should find 8 artifacts total: 5 with experiments + 3 standalone
		assert.Equal(t, 8, len(allResult.Items), "Should find 8 artifacts total (5 with experiments + 3 standalone)")

		// Count standalone artifacts in the unfiltered results
		standaloneCount := 0
		experimentCount := 0
		for _, artifact := range allResult.Items {
			hasExperiment := false
			if artifact.ModelArtifact != nil && artifact.ModelArtifact.ExperimentId != nil {
				hasExperiment = true
			} else if artifact.DocArtifact != nil && artifact.DocArtifact.ExperimentId != nil {
				hasExperiment = true
			} else if artifact.DataSet != nil && artifact.DataSet.ExperimentId != nil {
				hasExperiment = true
			} else if artifact.Metric != nil && artifact.Metric.ExperimentId != nil {
				hasExperiment = true
			} else if artifact.Parameter != nil && artifact.Parameter.ExperimentId != nil {
				hasExperiment = true
			}

			if hasExperiment {
				experimentCount++
			} else {
				standaloneCount++
			}
		}

		assert.Equal(t, 5, experimentCount, "Should have 5 artifacts with experiment associations")
		assert.Equal(t, 3, standaloneCount, "Should have 3 standalone artifacts without experiment associations")

		// Now test that experiment filters exclude standalone artifacts
		filterQuery := fmt.Sprintf(`experimentId = "%s"`, *createdExperiment1.Id)
		listOptionsFiltered := api.ListOptions{
			PageSize:    new(int32(100)),
			FilterQuery: &filterQuery,
		}

		filteredResult, err := service.GetArtifacts("", listOptionsFiltered, nil)
		require.NoError(t, err)
		require.NotNil(t, filteredResult)

		// Should find only 3 artifacts from experiment1, none of the standalone artifacts
		assert.Equal(t, 3, len(filteredResult.Items), "Should find only artifacts from experiment1, excluding standalone")

		// Verify none of the filtered results are standalone artifacts
		for _, artifact := range filteredResult.Items {
			// Each artifact should have an experimentId
			hasExperimentId := false
			if artifact.ModelArtifact != nil && artifact.ModelArtifact.ExperimentId != nil {
				hasExperimentId = true
				assert.Equal(t, *createdExperiment1.Id, *artifact.ModelArtifact.ExperimentId)
			} else if artifact.DocArtifact != nil && artifact.DocArtifact.ExperimentId != nil {
				hasExperimentId = true
				assert.Equal(t, *createdExperiment1.Id, *artifact.DocArtifact.ExperimentId)
			} else if artifact.Metric != nil && artifact.Metric.ExperimentId != nil {
				hasExperimentId = true
				assert.Equal(t, *createdExperiment1.Id, *artifact.Metric.ExperimentId)
			}
			assert.True(t, hasExperimentId, "All filtered artifacts should have experimentId")
		}
	})

	// Test that filtering by non-existent experimentId excludes all artifacts (including standalone)
	t.Run("Filter by non-existent experimentId excludes all artifacts", func(t *testing.T) {
		filterQuery := `experimentId = "non-existent-experiment-id"`
		listOptions := api.ListOptions{
			PageSize:    new(int32(100)),
			FilterQuery: &filterQuery,
		}

		result, err := service.GetArtifacts("", listOptions, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should find no artifacts (both experiment artifacts and standalone artifacts excluded)
		assert.Equal(t, 0, len(result.Items), "Should find no artifacts for non-existent experimentId")
	})

	// Note: GetArtifacts for model version artifacts with filterQuery works the same way
	// as other endpoints, but model version artifacts may not always have experimentId/experimentRunId
	// populated depending on how they were created. The filterQuery functionality itself works correctly.

	// Clean up created artifacts to avoid affecting other tests
	_ = createdArtifact1
	_ = createdArtifact2
	_ = createdArtifact3
	_ = createdMetric1
	_ = createdParam1
	_ = createdStandaloneArtifact1
	_ = createdStandaloneArtifact2
	_ = createdStandaloneArtifact3
}

func TestArtifactTypeMismatchOnUpdate(t *testing.T) {
	service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	t.Run("cannot change model-artifact to doc-artifact", func(t *testing.T) {
		modelArtifact := &openapi.ModelArtifact{
			Name:        new("test-type-mismatch-model"),
			Description: new("Original model artifact"),
			Uri:         new("s3://bucket/model.pkl"),
		}

		artifact := &openapi.Artifact{
			ModelArtifact: modelArtifact,
		}

		created, err := service.UpsertArtifact(artifact)
		require.NoError(t, err)
		require.NotNil(t, created.ModelArtifact)
		require.NotNil(t, created.ModelArtifact.Id)

		updateArtifact := &openapi.Artifact{
			DocArtifact: &openapi.DocArtifact{
				Id:          created.ModelArtifact.Id,
				Description: new("Trying to change to doc artifact"),
				Uri:         new("s3://bucket/doc.pdf"),
			},
		}

		result, err := service.UpsertArtifact(updateArtifact)
		assert.Error(t, err)
		assert.Nil(t, result)

		assert.Contains(t, err.Error(), "is not a doc artifact")
	})

	t.Run("cannot change doc-artifact to model-artifact", func(t *testing.T) {
		docArtifact := &openapi.DocArtifact{
			Name:        new("test-type-mismatch-doc"),
			Description: new("Original doc artifact"),
			Uri:         new("s3://bucket/doc.pdf"),
		}

		artifact := &openapi.Artifact{
			DocArtifact: docArtifact,
		}

		created, err := service.UpsertArtifact(artifact)
		require.NoError(t, err)
		require.NotNil(t, created.DocArtifact)
		require.NotNil(t, created.DocArtifact.Id)

		updateArtifact := &openapi.Artifact{
			ModelArtifact: &openapi.ModelArtifact{
				Id:          created.DocArtifact.Id,
				Description: new("Trying to change to model artifact"),
				Uri:         new("s3://bucket/model.pkl"),
			},
		}

		result, err := service.UpsertArtifact(updateArtifact)
		assert.Error(t, err)
		assert.Nil(t, result)

		assert.Contains(t, err.Error(), "is not a model artifact")
	})
}

// TestGetModelArtifactsWithFilterQuery verifies that the filterQuery parameter
// is correctly propagated and applied when listing ModelArtifacts.
//
// Regression test for: filterQuery silently ignored on GET /model_artifacts
// Root cause was missing FilterQuery field propagation in GetModelArtifacts.
func TestGetModelArtifactsWithFilterQuery(t *testing.T) {
	_service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	rm, err := _service.UpsertRegisteredModel(&openapi.RegisteredModel{Name: "artifact-filter-test-rm"})
	require.NoError(t, err)

	mv, err := _service.UpsertModelVersion(&openapi.ModelVersion{
		Name:              "v1",
		RegisteredModelId: *rm.Id,
	}, rm.Id)
	require.NoError(t, err)

	type artifactDef struct {
		name   string
		extID  string
		format string
	}
	artifactDefs := []artifactDef{
		{"fraud-model", "ext-fraud-001", "onnx"},
		{"image-model", "ext-image-002", "tensorflow"},
		{"nlp-model", "ext-nlp-003", "onnx"},
		{"recommendation-weights", "ext-rec-004", "pytorch"},
	}
	for _, art := range artifactDefs {
		_, err := _service.UpsertModelVersionArtifact(&openapi.Artifact{
			ModelArtifact: &openapi.ModelArtifact{
				Name:            new(art.name),
				ExternalId:      new(art.extID),
				Uri:             new("s3://bucket/" + art.name),
				ModelFormatName: new(art.format),
			},
		}, *mv.Id)
		require.NoError(t, err)
	}

	testCases := []struct {
		name          string
		filterQuery   string
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "Filter by exact name",
			filterQuery:   "name = 'fraud-model'",
			expectedCount: 1,
			expectedNames: []string{"fraud-model"},
		},
		{
			name:          "Filter by name pattern",
			filterQuery:   "name LIKE '%-model'",
			expectedCount: 3,
			expectedNames: []string{"fraud-model", "image-model", "nlp-model"},
		},
		{
			name:          "Filter by externalId",
			filterQuery:   "externalId = 'ext-image-002'",
			expectedCount: 1,
			expectedNames: []string{"image-model"},
		},
		{
			name:          "Filter by modelFormatName",
			filterQuery:   "modelFormatName = 'onnx'",
			expectedCount: 2,
			expectedNames: []string{"fraud-model", "nlp-model"},
		},
		{
			name:          "Complex filter with AND",
			filterQuery:   "modelFormatName = 'onnx' AND name = 'nlp-model'",
			expectedCount: 1,
			expectedNames: []string{"nlp-model"},
		},
		{
			name:          "Complex filter with OR",
			filterQuery:   "modelFormatName = 'tensorflow' OR modelFormatName = 'pytorch'",
			expectedCount: 2,
			expectedNames: []string{"image-model", "recommendation-weights"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pageSize := int32(20)
			fq := tc.filterQuery
			result, err := _service.GetModelArtifacts(api.ListOptions{
				PageSize:    &pageSize,
				FilterQuery: &fq,
			}, nil)

			require.NoError(t, err)
			require.NotNil(t, result)

			var returnedNames []string
			for _, item := range result.Items {
				returnedNames = append(returnedNames, *item.Name)
			}

			assert.Equal(t, tc.expectedCount, len(returnedNames),
				"filterQuery %q: expected %d items, got %d (filter may be silently ignored)",
				tc.filterQuery, tc.expectedCount, len(returnedNames))
			assert.ElementsMatch(t, tc.expectedNames, returnedNames,
				"filterQuery %q: unexpected items returned", tc.filterQuery)
		})
	}

	t.Run("Invalid filter syntax returns error", func(t *testing.T) {
		invalidFilter := "invalid <<<syntax"
		result, err := _service.GetModelArtifacts(api.ListOptions{
			FilterQuery: &invalidFilter,
		}, nil)

		assert.Error(t, err, "invalid filter syntax should return an error, not silently ignore the filter")
		assert.Nil(t, result)
	})

	t.Run("Filter with no matches returns empty list", func(t *testing.T) {
		fq := "modelFormatName = 'nonexistent-format'"
		result, err := _service.GetModelArtifacts(api.ListOptions{
			FilterQuery: &fq,
		}, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 0, len(result.Items), "filter with no matches should return empty items")
		assert.Equal(t, int32(0), result.Size)
	})

	t.Run("Filter combined with pagination", func(t *testing.T) {
		fq := "modelFormatName = 'onnx'"
		pageSize := int32(1)

		firstPage, err := _service.GetModelArtifacts(api.ListOptions{
			PageSize:    &pageSize,
			FilterQuery: &fq,
		}, nil)
		require.NoError(t, err)
		assert.Equal(t, 1, len(firstPage.Items))
		assert.NotEmpty(t, firstPage.NextPageToken,
			"should have a next page token for 2 onnx model artifacts with pageSize=1")

		secondPage, err := _service.GetModelArtifacts(api.ListOptions{
			PageSize:      &pageSize,
			FilterQuery:   &fq,
			NextPageToken: &firstPage.NextPageToken,
		}, nil)
		require.NoError(t, err)
		assert.Equal(t, 1, len(secondPage.Items))
		assert.NotEqual(t, firstPage.Items[0].Id, secondPage.Items[0].Id,
			"each page should return a different item")
	})
}
