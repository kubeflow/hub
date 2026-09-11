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

func TestUpsertInferenceService(t *testing.T) {
	_service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	t.Run("successful create", func(t *testing.T) {
		// Create prerequisites: registered model and serving environment
		registeredModel := &openapi.RegisteredModel{
			Name: "test-registered-model",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		servingEnv := &openapi.ServingEnvironment{
			Name: "test-serving-env",
		}
		createdEnv, err := _service.UpsertServingEnvironment(servingEnv)
		require.NoError(t, err)

		// Create inference service
		input := &openapi.InferenceService{
			Name:                 new("test-inference-service"),
			Description:          new("Test inference service description"),
			ExternalId:           new("inference-ext-123"),
			ServingEnvironmentId: *createdEnv.Id,
			RegisteredModelId:    *createdModel.Id,
			Runtime:              new("tensorflow"),
		}

		result, err := _service.UpsertInferenceService(input)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotNil(t, result.Id)
		assert.Equal(t, "test-inference-service", *result.Name)
		assert.Equal(t, "inference-ext-123", *result.ExternalId)
		assert.Equal(t, "Test inference service description", *result.Description)
		assert.Equal(t, *createdEnv.Id, result.ServingEnvironmentId)
		assert.Equal(t, *createdModel.Id, result.RegisteredModelId)
		assert.Equal(t, "tensorflow", *result.Runtime)
		assert.NotNil(t, result.CreateTimeSinceEpoch)
		assert.NotNil(t, result.LastUpdateTimeSinceEpoch)
	})

	t.Run("successful update", func(t *testing.T) {
		// Create prerequisites
		registeredModel := &openapi.RegisteredModel{
			Name: "update-test-registered-model",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		servingEnv := &openapi.ServingEnvironment{
			Name: "update-test-serving-env",
		}
		createdEnv, err := _service.UpsertServingEnvironment(servingEnv)
		require.NoError(t, err)

		// Create first
		input := &openapi.InferenceService{
			Name:                 new("update-test-inference-service"),
			Description:          new("Original description"),
			ServingEnvironmentId: *createdEnv.Id,
			RegisteredModelId:    *createdModel.Id,
		}

		created, err := _service.UpsertInferenceService(input)
		require.NoError(t, err)
		require.NotNil(t, created.Id)

		// Update
		update := &openapi.InferenceService{
			Id:                   created.Id,
			Name:                 new("update-test-inference-service"), // Name should remain the same
			Description:          new("Updated description"),
			ExternalId:           new("updated-ext-456"),
			ServingEnvironmentId: *createdEnv.Id,
			RegisteredModelId:    *createdModel.Id,
			Runtime:              new("pytorch"),
		}

		updated, err := _service.UpsertInferenceService(update)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, *created.Id, *updated.Id)
		assert.Equal(t, "update-test-inference-service", *updated.Name)
		assert.Equal(t, "Updated description", *updated.Description)
		assert.Equal(t, "updated-ext-456", *updated.ExternalId)
		assert.Equal(t, "pytorch", *updated.Runtime)
	})

	t.Run("create with custom properties", func(t *testing.T) {
		// Create prerequisites
		registeredModel := &openapi.RegisteredModel{
			Name: "custom-props-registered-model",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		servingEnv := &openapi.ServingEnvironment{
			Name: "custom-props-serving-env",
		}
		createdEnv, err := _service.UpsertServingEnvironment(servingEnv)
		require.NoError(t, err)

		customProps := map[string]openapi.MetadataValue{
			"model_uri": {
				MetadataStringValue: &openapi.MetadataStringValue{
					StringValue: "s3://bucket/model",
				},
			},
			"batch_size": {
				MetadataIntValue: &openapi.MetadataIntValue{
					IntValue: "32",
				},
			},
			"enable_logging": {
				MetadataBoolValue: &openapi.MetadataBoolValue{
					BoolValue: true,
				},
			},
			"confidence_threshold": {
				MetadataDoubleValue: &openapi.MetadataDoubleValue{
					DoubleValue: 0.85,
				},
			},
		}

		input := &openapi.InferenceService{
			Name:                 new("custom-props-inference-service"),
			ServingEnvironmentId: *createdEnv.Id,
			RegisteredModelId:    *createdModel.Id,
			CustomProperties:     customProps,
		}

		result, err := _service.UpsertInferenceService(input)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "custom-props-inference-service", *result.Name)
		assert.NotNil(t, result.CustomProperties)

		resultProps := result.CustomProperties
		assert.Contains(t, resultProps, "model_uri")
		assert.Contains(t, resultProps, "batch_size")
		assert.Contains(t, resultProps, "enable_logging")
		assert.Contains(t, resultProps, "confidence_threshold")

		assert.Equal(t, "s3://bucket/model", resultProps["model_uri"].MetadataStringValue.StringValue)
		assert.Equal(t, "32", resultProps["batch_size"].MetadataIntValue.IntValue)
		assert.Equal(t, true, resultProps["enable_logging"].MetadataBoolValue.BoolValue)
		assert.Equal(t, 0.85, resultProps["confidence_threshold"].MetadataDoubleValue.DoubleValue)
	})

	t.Run("minimal inference service", func(t *testing.T) {
		// Create prerequisites
		registeredModel := &openapi.RegisteredModel{
			Name: "minimal-registered-model",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		servingEnv := &openapi.ServingEnvironment{
			Name: "minimal-serving-env",
		}
		createdEnv, err := _service.UpsertServingEnvironment(servingEnv)
		require.NoError(t, err)

		input := &openapi.InferenceService{
			Name:                 new("minimal-inference-service"),
			ServingEnvironmentId: *createdEnv.Id,
			RegisteredModelId:    *createdModel.Id,
		}

		result, err := _service.UpsertInferenceService(input)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "minimal-inference-service", *result.Name)
		assert.NotNil(t, result.Id)
		assert.Equal(t, *createdEnv.Id, result.ServingEnvironmentId)
		assert.Equal(t, *createdModel.Id, result.RegisteredModelId)
	})

	t.Run("nil inference service error", func(t *testing.T) {
		result, err := _service.UpsertInferenceService(nil)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid inference service pointer")
	})

	t.Run("nil desired state preserved", func(t *testing.T) {
		// Create prerequisites
		registeredModel := &openapi.RegisteredModel{
			Name: "nil-state-registered-model",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		servingEnv := &openapi.ServingEnvironment{
			Name: "nil-state-serving-env",
		}
		createdEnv, err := _service.UpsertServingEnvironment(servingEnv)
		require.NoError(t, err)

		// Create inference service with nil DesiredState and other optional fields
		input := &openapi.InferenceService{
			Name:                 new("nil-state-inference-service"),
			Description:          new("Test inference service with nil desired state"),
			ExternalId:           new("nil-state-ext-123"),
			ServingEnvironmentId: *createdEnv.Id,
			RegisteredModelId:    *createdModel.Id,
			Runtime:              new("tensorflow"),
			DesiredState:         nil, // Explicitly set to nil
		}

		result, err := _service.UpsertInferenceService(input)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotNil(t, result.Id)
		assert.Equal(t, "nil-state-inference-service", *result.Name)
		assert.Equal(t, "nil-state-ext-123", *result.ExternalId)
		assert.Equal(t, "Test inference service with nil desired state", *result.Description)
		assert.Equal(t, *createdEnv.Id, result.ServingEnvironmentId)
		assert.Equal(t, *createdModel.Id, result.RegisteredModelId)
		assert.Equal(t, "tensorflow", *result.Runtime)
		assert.Nil(t, result.DesiredState) // Verify desired state remains nil
		assert.NotNil(t, result.CreateTimeSinceEpoch)
		assert.NotNil(t, result.LastUpdateTimeSinceEpoch)
	})

	t.Run("unicode characters in name", func(t *testing.T) {
		// Create prerequisites
		registeredModel := &openapi.RegisteredModel{
			Name: "unicode-test-registered-model",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		servingEnv := &openapi.ServingEnvironment{
			Name: "unicode-test-serving-env",
		}
		createdEnv, err := _service.UpsertServingEnvironment(servingEnv)
		require.NoError(t, err)

		// Test with unicode characters: Chinese, Russian, Japanese, and emoji
		unicodeName := "推理服务-тест-推論サービス-🚀"
		input := &openapi.InferenceService{
			Name:                 new(unicodeName),
			Description:          new("Test inference service with unicode characters"),
			ServingEnvironmentId: *createdEnv.Id,
			RegisteredModelId:    *createdModel.Id,
			Runtime:              new("tensorflow"),
		}

		result, err := _service.UpsertInferenceService(input)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, unicodeName, *result.Name)
		assert.Equal(t, "Test inference service with unicode characters", *result.Description)
		assert.NotNil(t, result.Id)
		assert.NotNil(t, result.CreateTimeSinceEpoch)
		assert.NotNil(t, result.LastUpdateTimeSinceEpoch)

		// Verify we can retrieve it by ID
		retrieved, err := _service.GetInferenceServiceById(*result.Id)
		require.NoError(t, err)
		assert.Equal(t, unicodeName, *retrieved.Name)
	})

	t.Run("special characters in name", func(t *testing.T) {
		// Create prerequisites
		registeredModel := &openapi.RegisteredModel{
			Name: "special-chars-test-registered-model",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		servingEnv := &openapi.ServingEnvironment{
			Name: "special-chars-test-serving-env",
		}
		createdEnv, err := _service.UpsertServingEnvironment(servingEnv)
		require.NoError(t, err)

		// Test with various special characters
		specialName := "!@#$%^&*()_+-=[]{}|;':\",./<>?"
		input := &openapi.InferenceService{
			Name:                 new(specialName),
			Description:          new("Test inference service with special characters"),
			ServingEnvironmentId: *createdEnv.Id,
			RegisteredModelId:    *createdModel.Id,
			Runtime:              new("pytorch"),
		}

		result, err := _service.UpsertInferenceService(input)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, specialName, *result.Name)
		assert.Equal(t, "Test inference service with special characters", *result.Description)
		assert.NotNil(t, result.Id)

		// Verify we can retrieve it by ID
		retrieved, err := _service.GetInferenceServiceById(*result.Id)
		require.NoError(t, err)
		assert.Equal(t, specialName, *retrieved.Name)
	})

	t.Run("mixed unicode and special characters", func(t *testing.T) {
		// Create prerequisites
		registeredModel := &openapi.RegisteredModel{
			Name: "mixed-chars-test-registered-model",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		servingEnv := &openapi.ServingEnvironment{
			Name: "mixed-chars-test-serving-env",
		}
		createdEnv, err := _service.UpsertServingEnvironment(servingEnv)
		require.NoError(t, err)

		// Test with mixed unicode and special characters
		mixedName := "推理@#$%服务-тест!@#-推論()サービス-🚀[]"
		input := &openapi.InferenceService{
			Name:                 new(mixedName),
			Description:          new("Test inference service with mixed unicode and special characters"),
			ServingEnvironmentId: *createdEnv.Id,
			RegisteredModelId:    *createdModel.Id,
			Runtime:              new("onnx"),
		}

		result, err := _service.UpsertInferenceService(input)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, mixedName, *result.Name)
		assert.Equal(t, "Test inference service with mixed unicode and special characters", *result.Description)
		assert.NotNil(t, result.Id)

		// Verify we can retrieve it by ID
		retrieved, err := _service.GetInferenceServiceById(*result.Id)
		require.NoError(t, err)
		assert.Equal(t, mixedName, *retrieved.Name)
	})

	t.Run("pagination with 10+ inference services", func(t *testing.T) {
		// Create prerequisites
		registeredModel := &openapi.RegisteredModel{
			Name: "paging-test-registered-model",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		servingEnv := &openapi.ServingEnvironment{
			Name: "paging-test-serving-env",
		}
		createdEnv, err := _service.UpsertServingEnvironment(servingEnv)
		require.NoError(t, err)

		// Create 15 inference services for pagination testing
		var createdServices []string
		for i := range 15 {
			serviceName := "paging-test-inference-service-" + fmt.Sprintf("%02d", i)
			input := &openapi.InferenceService{
				Name:                 new(serviceName),
				Description:          new("Pagination test inference service " + fmt.Sprintf("%02d", i)),
				ServingEnvironmentId: *createdEnv.Id,
				RegisteredModelId:    *createdModel.Id,
				Runtime:              new("tensorflow"),
			}

			result, err := _service.UpsertInferenceService(input)
			require.NoError(t, err)
			createdServices = append(createdServices, *result.Id)
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
		firstPage, err := _service.GetInferenceServices(listOptions, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, firstPage)
		assert.LessOrEqual(t, len(firstPage.Items), 5, "First page should have at most 5 items")
		assert.Equal(t, int32(5), firstPage.PageSize)

		// Filter to only our test inference services in first page
		var firstPageTestServices []openapi.InferenceService
		firstPageIds := make(map[string]bool)
		for _, item := range firstPage.Items {
			// Only include our test services (those with the specific prefix)
			if strings.HasPrefix(*item.Name, "paging-test-inference-service-") {
				assert.False(t, firstPageIds[*item.Id], "Should not have duplicate IDs in first page")
				firstPageIds[*item.Id] = true
				firstPageTestServices = append(firstPageTestServices, item)
			}
		}

		// Only proceed with second page test if we have a next page token and found test services
		if firstPage.NextPageToken != "" && len(firstPageTestServices) > 0 {
			// Get second page using next page token
			listOptions.NextPageToken = &firstPage.NextPageToken
			secondPage, err := _service.GetInferenceServices(listOptions, nil, nil)
			require.NoError(t, err)
			require.NotNil(t, secondPage)
			assert.LessOrEqual(t, len(secondPage.Items), 5, "Second page should have at most 5 items")

			// Verify no duplicates between pages (only check our test services)
			for _, item := range secondPage.Items {
				if strings.HasPrefix(*item.Name, "paging-test-inference-service-") {
					assert.False(t, firstPageIds[*item.Id], "Should not have duplicate IDs between pages")
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

		allItems, err := _service.GetInferenceServices(listOptions, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, allItems)
		assert.GreaterOrEqual(t, len(allItems.Items), 15, "Should have at least our 15 test inference services")

		// Count our test services in the results
		foundCount := 0
		for _, item := range allItems.Items {
			if slices.Contains(createdServices, *item.Id) {
				foundCount++
			}
		}
		assert.Equal(t, 15, foundCount, "Should find all 15 created inference services")

		// Test descending order
		descOrder := "DESC"
		listOptions = api.ListOptions{
			PageSize:  &pageSize,
			OrderBy:   &orderBy,
			SortOrder: &descOrder,
		}

		descPage, err := _service.GetInferenceServices(listOptions, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, descPage)
		assert.LessOrEqual(t, len(descPage.Items), 5, "Desc page should have at most 5 items")

		// Verify ordering (names should be in descending order)
		if len(descPage.Items) > 1 {
			for i := 1; i < len(descPage.Items); i++ {
				assert.GreaterOrEqual(t, *descPage.Items[i-1].Name, *descPage.Items[i].Name,
					"Items should be in descending order by name")
			}
		}
	})
}

func TestGetInferenceServiceById(t *testing.T) {
	_service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	t.Run("successful get", func(t *testing.T) {
		// Create prerequisites
		registeredModel := &openapi.RegisteredModel{
			Name: "get-test-registered-model",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		servingEnv := &openapi.ServingEnvironment{
			Name: "get-test-serving-env",
		}
		createdEnv, err := _service.UpsertServingEnvironment(servingEnv)
		require.NoError(t, err)

		// First create an inference service to retrieve
		input := &openapi.InferenceService{
			Name:                 new("get-test-inference-service"),
			Description:          new("Test description"),
			ExternalId:           new("get-ext-123"),
			ServingEnvironmentId: *createdEnv.Id,
			RegisteredModelId:    *createdModel.Id,
			Runtime:              new("tensorflow"),
		}

		created, err := _service.UpsertInferenceService(input)
		require.NoError(t, err)
		require.NotNil(t, created.Id)

		// Get the inference service by ID
		result, err := _service.GetInferenceServiceById(*created.Id)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, *created.Id, *result.Id)
		assert.Equal(t, "get-test-inference-service", *result.Name)
		assert.Equal(t, "get-ext-123", *result.ExternalId)
		assert.Equal(t, "Test description", *result.Description)
		assert.Equal(t, "tensorflow", *result.Runtime)
	})

	t.Run("invalid id", func(t *testing.T) {
		result, err := _service.GetInferenceServiceById("invalid")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid")
	})

	t.Run("non-existent id", func(t *testing.T) {
		result, err := _service.GetInferenceServiceById("99999")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "no InferenceService found")
	})
}

func TestGetInferenceServiceByParams(t *testing.T) {
	_service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	t.Run("successful get by name and parent resource id", func(t *testing.T) {
		// Create prerequisites
		registeredModel := &openapi.RegisteredModel{
			Name: "params-test-registered-model",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		servingEnv := &openapi.ServingEnvironment{
			Name: "params-test-serving-env",
		}
		createdEnv, err := _service.UpsertServingEnvironment(servingEnv)
		require.NoError(t, err)

		input := &openapi.InferenceService{
			Name:                 new("params-test-inference-service"),
			ExternalId:           new("params-ext-123"),
			ServingEnvironmentId: *createdEnv.Id,
			RegisteredModelId:    *createdModel.Id,
		}
		created, err := _service.UpsertInferenceService(input)
		require.NoError(t, err)

		// Get by name and parent resource ID
		serviceName := "params-test-inference-service"
		result, err := _service.GetInferenceServiceByParams(&serviceName, createdEnv.Id, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, *created.Id, *result.Id)
		assert.Equal(t, "params-test-inference-service", *result.Name)
	})

	t.Run("successful get by external id", func(t *testing.T) {
		// Create prerequisites
		registeredModel := &openapi.RegisteredModel{
			Name: "params-ext-test-registered-model",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		servingEnv := &openapi.ServingEnvironment{
			Name: "params-ext-test-serving-env",
		}
		createdEnv, err := _service.UpsertServingEnvironment(servingEnv)
		require.NoError(t, err)

		input := &openapi.InferenceService{
			Name:                 new("params-ext-test-inference-service"),
			ExternalId:           new("params-unique-ext-456"),
			ServingEnvironmentId: *createdEnv.Id,
			RegisteredModelId:    *createdModel.Id,
		}
		created, err := _service.UpsertInferenceService(input)
		require.NoError(t, err)

		// Get by external ID
		externalId := "params-unique-ext-456"
		result, err := _service.GetInferenceServiceByParams(nil, nil, &externalId)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, *created.Id, *result.Id)
		assert.Equal(t, "params-unique-ext-456", *result.ExternalId)
	})

	t.Run("invalid parameters", func(t *testing.T) {
		result, err := _service.GetInferenceServiceByParams(nil, nil, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid parameters call")
	})

	t.Run("same inference service name across different serving environments", func(t *testing.T) {
		// This test catches the bug where ParentResourceID was not being used to filter inference services

		// Create first serving environment
		servingEnv1 := &openapi.ServingEnvironment{
			Name: "serving-env-with-shared-service-1",
		}
		createdEnv1, err := _service.UpsertServingEnvironment(servingEnv1)
		require.NoError(t, err)

		// Create second serving environment
		servingEnv2 := &openapi.ServingEnvironment{
			Name: "serving-env-with-shared-service-2",
		}
		createdEnv2, err := _service.UpsertServingEnvironment(servingEnv2)
		require.NoError(t, err)

		// Create registered model for the services
		registeredModel := &openapi.RegisteredModel{
			Name: "model-for-shared-services",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		// Create inference service "shared-service-name-test" for the first environment
		service1 := &openapi.InferenceService{
			Name:                 new("shared-service-name-test"),
			ServingEnvironmentId: *createdEnv1.Id,
			RegisteredModelId:    *createdModel.Id,
			Description:          new("Service for environment 1"),
		}
		createdService1, err := _service.UpsertInferenceService(service1)
		require.NoError(t, err)

		// Create inference service "shared-service-name-test" for the second environment
		service2 := &openapi.InferenceService{
			Name:                 new("shared-service-name-test"),
			ServingEnvironmentId: *createdEnv2.Id,
			RegisteredModelId:    *createdModel.Id,
			Description:          new("Service for environment 2"),
		}
		createdService2, err := _service.UpsertInferenceService(service2)
		require.NoError(t, err)

		// Query for service "shared-service-name-test" of the first environment
		serviceName := "shared-service-name-test"
		result1, err := _service.GetInferenceServiceByParams(&serviceName, createdEnv1.Id, nil)
		require.NoError(t, err)
		require.NotNil(t, result1)
		assert.Equal(t, *createdService1.Id, *result1.Id)
		assert.Equal(t, *createdEnv1.Id, result1.ServingEnvironmentId)
		assert.Equal(t, "Service for environment 1", *result1.Description)

		// Query for service "shared-service-name-test" of the second environment
		result2, err := _service.GetInferenceServiceByParams(&serviceName, createdEnv2.Id, nil)
		require.NoError(t, err)
		require.NotNil(t, result2)
		assert.Equal(t, *createdService2.Id, *result2.Id)
		assert.Equal(t, *createdEnv2.Id, result2.ServingEnvironmentId)
		assert.Equal(t, "Service for environment 2", *result2.Description)

		// Ensure we got different services
		assert.NotEqual(t, *result1.Id, *result2.Id)
	})

	t.Run("no inference service found", func(t *testing.T) {
		serviceName := "nonexistent-inference-service"
		parentResourceId := "999"
		result, err := _service.GetInferenceServiceByParams(&serviceName, &parentResourceId, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "no inference service found")
	})
}

func TestGetInferenceServices(t *testing.T) {
	_service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	t.Run("successful list", func(t *testing.T) {
		// Create prerequisites
		registeredModel := &openapi.RegisteredModel{
			Name: "list-test-registered-model",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		servingEnv := &openapi.ServingEnvironment{
			Name: "list-test-serving-env",
		}
		createdEnv, err := _service.UpsertServingEnvironment(servingEnv)
		require.NoError(t, err)

		// Create multiple inference services for listing
		testInferenceServices := []*openapi.InferenceService{
			{
				Name:                 new("list-inference-service-1"),
				ExternalId:           new("list-ext-1"),
				ServingEnvironmentId: *createdEnv.Id,
				RegisteredModelId:    *createdModel.Id,
				Runtime:              new("tensorflow"),
			},
			{
				Name:                 new("list-inference-service-2"),
				ExternalId:           new("list-ext-2"),
				ServingEnvironmentId: *createdEnv.Id,
				RegisteredModelId:    *createdModel.Id,
				Runtime:              new("pytorch"),
			},
			{
				Name:                 new("list-inference-service-3"),
				ExternalId:           new("list-ext-3"),
				ServingEnvironmentId: *createdEnv.Id,
				RegisteredModelId:    *createdModel.Id,
				Runtime:              new("onnx"),
			},
		}

		var createdIds []string
		for _, infSvc := range testInferenceServices {
			created, err := _service.UpsertInferenceService(infSvc)
			require.NoError(t, err)
			createdIds = append(createdIds, *created.Id)
		}

		// List inference services with basic pagination
		pageSize := int32(10)
		listOptions := api.ListOptions{
			PageSize: &pageSize,
		}

		result, err := _service.GetInferenceServices(listOptions, nil, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.GreaterOrEqual(t, len(result.Items), 3) // Should have at least our 3 test inference services
		assert.Equal(t, int32(10), result.PageSize)

		// Verify our inference services are in the result
		foundServices := 0
		for _, item := range result.Items {
			if slices.Contains(createdIds, *item.Id) {
				foundServices++
			}
		}
		assert.Equal(t, 3, foundServices, "All created inference services should be found in the list")
	})

	t.Run("list with serving environment filter", func(t *testing.T) {
		// Create prerequisites
		registeredModel := &openapi.RegisteredModel{
			Name: "filter-test-registered-model",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		servingEnv1 := &openapi.ServingEnvironment{
			Name: "filter-test-serving-env-1",
		}
		createdEnv1, err := _service.UpsertServingEnvironment(servingEnv1)
		require.NoError(t, err)

		servingEnv2 := &openapi.ServingEnvironment{
			Name: "filter-test-serving-env-2",
		}
		createdEnv2, err := _service.UpsertServingEnvironment(servingEnv2)
		require.NoError(t, err)

		// Create inference services in different serving environments
		infSvc1 := &openapi.InferenceService{
			Name:                 new("filter-inference-service-1"),
			ServingEnvironmentId: *createdEnv1.Id,
			RegisteredModelId:    *createdModel.Id,
		}
		created1, err := _service.UpsertInferenceService(infSvc1)
		require.NoError(t, err)

		infSvc2 := &openapi.InferenceService{
			Name:                 new("filter-inference-service-2"),
			ServingEnvironmentId: *createdEnv2.Id,
			RegisteredModelId:    *createdModel.Id,
		}
		_, err = _service.UpsertInferenceService(infSvc2)
		require.NoError(t, err)

		// List inference services filtered by serving environment
		pageSize := int32(10)
		listOptions := api.ListOptions{
			PageSize: &pageSize,
		}

		result, err := _service.GetInferenceServices(listOptions, createdEnv1.Id, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.GreaterOrEqual(t, len(result.Items), 1) // Should have at least our 1 service in env1

		// Verify that only services from the specified environment are returned
		found := false
		for _, item := range result.Items {
			if *item.Id == *created1.Id {
				found = true
				assert.Equal(t, *createdEnv1.Id, item.ServingEnvironmentId)
			}
		}
		assert.True(t, found, "Should find the inference service in the specified serving environment")
	})

	t.Run("list with runtime filter", func(t *testing.T) {
		// Create prerequisites
		registeredModel := &openapi.RegisteredModel{
			Name: "runtime-filter-registered-model",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		servingEnv := &openapi.ServingEnvironment{
			Name: "runtime-filter-serving-env",
		}
		createdEnv, err := _service.UpsertServingEnvironment(servingEnv)
		require.NoError(t, err)

		// Create inference services with different runtimes
		infSvcTensorflow := &openapi.InferenceService{
			Name:                 new("runtime-tensorflow-service"),
			ServingEnvironmentId: *createdEnv.Id,
			RegisteredModelId:    *createdModel.Id,
			Runtime:              new("tensorflow"),
		}
		createdTensorflow, err := _service.UpsertInferenceService(infSvcTensorflow)
		require.NoError(t, err)

		infSvcPytorch := &openapi.InferenceService{
			Name:                 new("runtime-pytorch-service"),
			ServingEnvironmentId: *createdEnv.Id,
			RegisteredModelId:    *createdModel.Id,
			Runtime:              new("pytorch"),
		}
		_, err = _service.UpsertInferenceService(infSvcPytorch)
		require.NoError(t, err)

		// List inference services filtered by runtime
		pageSize := int32(10)
		listOptions := api.ListOptions{
			PageSize: &pageSize,
		}
		runtime := "tensorflow"

		result, err := _service.GetInferenceServices(listOptions, nil, &runtime)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.GreaterOrEqual(t, len(result.Items), 1) // Should have at least our 1 tensorflow service

		// Verify that only services with the specified runtime are returned
		found := false
		for _, item := range result.Items {
			if *item.Id == *createdTensorflow.Id {
				found = true
				assert.Equal(t, "tensorflow", *item.Runtime)
			}
		}
		assert.True(t, found, "Should find the inference service with the specified runtime")
	})

	t.Run("pagination and ordering", func(t *testing.T) {
		// Create prerequisites
		registeredModel := &openapi.RegisteredModel{
			Name: "pagination-test-registered-model",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		servingEnv := &openapi.ServingEnvironment{
			Name: "pagination-test-serving-env",
		}
		createdEnv, err := _service.UpsertServingEnvironment(servingEnv)
		require.NoError(t, err)

		// Create several inference services for pagination testing
		for i := range 5 {
			infSvc := &openapi.InferenceService{
				Name:                 new("pagination-inference-service-" + string(rune('A'+i))),
				ExternalId:           new("pagination-ext-" + string(rune('A'+i))),
				ServingEnvironmentId: *createdEnv.Id,
				RegisteredModelId:    *createdModel.Id,
			}
			_, err := _service.UpsertInferenceService(infSvc)
			require.NoError(t, err)
		}

		// Test with small page size and ordering
		pageSize := int32(2)
		orderBy := "name"
		sortOrder := "asc"
		listOptions := api.ListOptions{
			PageSize:  &pageSize,
			OrderBy:   &orderBy,
			SortOrder: &sortOrder,
		}

		result, err := _service.GetInferenceServices(listOptions, nil, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.GreaterOrEqual(t, len(result.Items), 2) // Should have at least 2 items
		assert.Equal(t, int32(2), result.PageSize)
	})

	t.Run("invalid serving environment id", func(t *testing.T) {
		invalidId := "invalid"
		listOptions := api.ListOptions{}

		result, err := _service.GetInferenceServices(listOptions, &invalidId, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid syntax: bad request")
	})
}

func TestInferenceServiceRoundTrip(t *testing.T) {
	_service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	t.Run("complete roundtrip", func(t *testing.T) {
		// Create prerequisites
		registeredModel := &openapi.RegisteredModel{
			Name:        "roundtrip-registered-model",
			Description: new("Roundtrip test registered model"),
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		servingEnv := &openapi.ServingEnvironment{
			Name:        "roundtrip-serving-env",
			Description: new("Roundtrip test serving environment"),
		}
		createdEnv, err := _service.UpsertServingEnvironment(servingEnv)
		require.NoError(t, err)

		// Create an inference service with all fields
		original := &openapi.InferenceService{
			Name:                 new("roundtrip-inference-service"),
			Description:          new("Roundtrip test description"),
			ExternalId:           new("roundtrip-ext-123"),
			ServingEnvironmentId: *createdEnv.Id,
			RegisteredModelId:    *createdModel.Id,
			Runtime:              new("tensorflow"),
		}

		// Create
		created, err := _service.UpsertInferenceService(original)
		require.NoError(t, err)
		require.NotNil(t, created.Id)

		// Get by ID
		retrieved, err := _service.GetInferenceServiceById(*created.Id)
		require.NoError(t, err)

		// Verify all fields match
		assert.Equal(t, *created.Id, *retrieved.Id)
		assert.Equal(t, *original.Name, *retrieved.Name)
		assert.Equal(t, *original.Description, *retrieved.Description)
		assert.Equal(t, *original.ExternalId, *retrieved.ExternalId)
		assert.Equal(t, original.ServingEnvironmentId, retrieved.ServingEnvironmentId)
		assert.Equal(t, original.RegisteredModelId, retrieved.RegisteredModelId)
		assert.Equal(t, *original.Runtime, *retrieved.Runtime)

		// Update
		retrieved.Description = new("Updated description")
		retrieved.Runtime = new("pytorch")

		updated, err := _service.UpsertInferenceService(retrieved)
		require.NoError(t, err)

		// Verify update
		assert.Equal(t, *created.Id, *updated.Id)
		assert.Equal(t, "Updated description", *updated.Description)
		assert.Equal(t, "pytorch", *updated.Runtime)

		// Get again to verify persistence
		final, err := _service.GetInferenceServiceById(*created.Id)
		require.NoError(t, err)
		assert.Equal(t, "Updated description", *final.Description)
		assert.Equal(t, "pytorch", *final.Runtime)
	})

	t.Run("roundtrip with custom properties", func(t *testing.T) {
		// Create prerequisites
		registeredModel := &openapi.RegisteredModel{
			Name: "roundtrip-custom-props-registered-model",
		}
		createdModel, err := _service.UpsertRegisteredModel(registeredModel)
		require.NoError(t, err)

		servingEnv := &openapi.ServingEnvironment{
			Name: "roundtrip-custom-props-serving-env",
		}
		createdEnv, err := _service.UpsertServingEnvironment(servingEnv)
		require.NoError(t, err)

		customProps := map[string]openapi.MetadataValue{
			"deployment_type": {
				MetadataStringValue: &openapi.MetadataStringValue{
					StringValue: "canary",
				},
			},
			"replicas": {
				MetadataIntValue: &openapi.MetadataIntValue{
					IntValue: "3",
				},
			},
		}

		original := &openapi.InferenceService{
			Name:                 new("roundtrip-custom-props-inference-service"),
			ServingEnvironmentId: *createdEnv.Id,
			RegisteredModelId:    *createdModel.Id,
			CustomProperties:     customProps,
		}

		// Create
		created, err := _service.UpsertInferenceService(original)
		require.NoError(t, err)
		require.NotNil(t, created.Id)

		// Get by ID
		retrieved, err := _service.GetInferenceServiceById(*created.Id)
		require.NoError(t, err)

		// Verify custom properties
		assert.NotNil(t, retrieved.CustomProperties)
		retrievedProps := retrieved.CustomProperties
		assert.Contains(t, retrievedProps, "deployment_type")
		assert.Contains(t, retrievedProps, "replicas")
		assert.Equal(t, "canary", retrievedProps["deployment_type"].MetadataStringValue.StringValue)
		assert.Equal(t, "3", retrievedProps["replicas"].MetadataIntValue.IntValue)

		// Update custom properties
		updatedProps := map[string]openapi.MetadataValue{
			"deployment_type": {
				MetadataStringValue: &openapi.MetadataStringValue{
					StringValue: "blue-green",
				},
			},
			"replicas": {
				MetadataIntValue: &openapi.MetadataIntValue{
					IntValue: "5",
				},
			},
			"new_prop": {
				MetadataStringValue: &openapi.MetadataStringValue{
					StringValue: "new_value",
				},
			},
		}
		retrieved.CustomProperties = updatedProps

		updated, err := _service.UpsertInferenceService(retrieved)
		require.NoError(t, err)

		// Verify updated custom properties
		assert.NotNil(t, updated.CustomProperties)
		finalProps := updated.CustomProperties
		assert.Equal(t, "blue-green", finalProps["deployment_type"].MetadataStringValue.StringValue)
		assert.Equal(t, "5", finalProps["replicas"].MetadataIntValue.IntValue)
		assert.Equal(t, "new_value", finalProps["new_prop"].MetadataStringValue.StringValue)
	})
}

// TestGetInferenceServicesWithFilterQuery verifies that the filterQuery parameter
// is correctly propagated and applied when listing InferenceServices.
//
// Regression test for: filterQuery silently ignored on GET /inference_services
// Root cause was missing FilterQuery field propagation in GetInferenceServices
// and missing GetRestEntityType() on InferenceServiceListOptions.
func TestGetInferenceServicesWithFilterQuery(t *testing.T) {
	_service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	// Create shared prerequisites
	rm, err := _service.UpsertRegisteredModel(&openapi.RegisteredModel{Name: "filter-test-rm"})
	require.NoError(t, err)

	env, err := _service.UpsertServingEnvironment(&openapi.ServingEnvironment{Name: "filter-test-env"})
	require.NoError(t, err)

	// Create inference services with distinct properties for filtering
	type svcDef struct {
		name    string
		runtime string
		extID   string
	}
	svcDefs := []svcDef{
		{"fraud-detector", "tensorflow", "ext-fraud-001"},
		{"image-classifier", "pytorch", "ext-image-002"},
		{"nlp-pipeline", "tensorflow", "ext-nlp-003"},
		{"recommendation-engine", "sklearn", "ext-rec-004"},
	}
	for _, svc := range svcDefs {
		name := svc.name
		rt := svc.runtime
		eid := svc.extID
		_, err := _service.UpsertInferenceService(&openapi.InferenceService{
			Name:                 &name,
			ExternalId:           &eid,
			Runtime:              &rt,
			ServingEnvironmentId: *env.Id,
			RegisteredModelId:    *rm.Id,
		})
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
			filterQuery:   "name = 'fraud-detector'",
			expectedCount: 1,
			expectedNames: []string{"fraud-detector"},
		},
		{
			name:          "Filter by name pattern",
			filterQuery:   "name LIKE '%-detector'",
			expectedCount: 1,
			expectedNames: []string{"fraud-detector"},
		},
		{
			name:          "Filter by externalId",
			filterQuery:   "externalId = 'ext-image-002'",
			expectedCount: 1,
			expectedNames: []string{"image-classifier"},
		},
		{
			name:          "Filter by runtime - tensorflow",
			filterQuery:   "runtime = 'tensorflow'",
			expectedCount: 2,
			expectedNames: []string{"fraud-detector", "nlp-pipeline"},
		},
		{
			name:          "Filter by runtime - pytorch",
			filterQuery:   "runtime = 'pytorch'",
			expectedCount: 1,
			expectedNames: []string{"image-classifier"},
		},
		{
			name:          "Complex filter with AND",
			filterQuery:   "runtime = 'tensorflow' AND name = 'fraud-detector'",
			expectedCount: 1,
			expectedNames: []string{"fraud-detector"},
		},
		{
			name:          "Complex filter with OR",
			filterQuery:   "runtime = 'pytorch' OR runtime = 'sklearn'",
			expectedCount: 2,
			expectedNames: []string{"image-classifier", "recommendation-engine"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pageSize := int32(20)
			fq := tc.filterQuery
			result, err := _service.GetInferenceServices(api.ListOptions{
				PageSize:    &pageSize,
				FilterQuery: &fq,
			}, nil, nil)

			require.NoError(t, err)
			require.NotNil(t, result)

			var matchedNames []string
			for _, item := range result.Items {
				if slices.Contains(tc.expectedNames, *item.Name) {
					matchedNames = append(matchedNames, *item.Name)
				}
			}

			assert.Equal(t, tc.expectedCount, len(matchedNames),
				"filterQuery %q: expected %d items, got %d (filter may be silently ignored)",
				tc.filterQuery, tc.expectedCount, len(matchedNames))
			assert.ElementsMatch(t, tc.expectedNames, matchedNames,
				"filterQuery %q: unexpected items returned", tc.filterQuery)
		})
	}

	t.Run("Invalid filter syntax returns error", func(t *testing.T) {
		invalidFilter := "invalid <<<syntax"
		result, err := _service.GetInferenceServices(api.ListOptions{
			FilterQuery: &invalidFilter,
		}, nil, nil)

		assert.Error(t, err, "invalid filter syntax should return an error, not silently ignore the filter")
		assert.Nil(t, result)
	})

	t.Run("Filter with no matches returns empty list", func(t *testing.T) {
		fq := "runtime = 'nonexistent-runtime'"
		result, err := _service.GetInferenceServices(api.ListOptions{
			FilterQuery: &fq,
		}, nil, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 0, len(result.Items), "filter with no matches should return empty items")
		assert.Equal(t, int32(0), result.Size)
	})

	t.Run("Filter combined with pagination", func(t *testing.T) {
		fq := "runtime = 'tensorflow'"
		pageSize := int32(1)

		firstPage, err := _service.GetInferenceServices(api.ListOptions{
			PageSize:    &pageSize,
			FilterQuery: &fq,
		}, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 1, len(firstPage.Items))
		assert.NotEmpty(t, firstPage.NextPageToken,
			"should have a next page token for 2 tensorflow services with pageSize=1")

		secondPage, err := _service.GetInferenceServices(api.ListOptions{
			PageSize:      &pageSize,
			FilterQuery:   &fq,
			NextPageToken: &firstPage.NextPageToken,
		}, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 1, len(secondPage.Items))
		assert.NotEqual(t, firstPage.Items[0].Id, secondPage.Items[0].Id,
			"each page should return a different item")
	})

	t.Run("Filter scoped to serving environment", func(t *testing.T) {
		fq := "runtime = 'tensorflow'"
		pageSize := int32(20)

		result, err := _service.GetInferenceServices(api.ListOptions{
			PageSize:    &pageSize,
			FilterQuery: &fq,
		}, env.Id, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, len(result.Items),
			"environment-scoped filter should return 2 tensorflow services")
		for _, item := range result.Items {
			assert.Equal(t, "tensorflow", *item.Runtime,
				"all returned items should match the runtime filter")
		}
	})
}
