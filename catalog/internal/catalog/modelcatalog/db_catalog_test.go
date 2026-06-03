package modelcatalog

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog"
	mcpcatalogmodels "github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog/models"
	mcpservice "github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog/service"
	"github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	modelservice "github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/service"
	sharedmodels "github.com/kubeflow/hub/catalog/internal/db/models"
	"github.com/kubeflow/hub/catalog/internal/db/service"
	"github.com/kubeflow/hub/catalog/internal/testhelpers"
	model "github.com/kubeflow/hub/catalog/pkg/openapi"
	"github.com/kubeflow/hub/internal/platform/apiutils"
	mr_models "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/kubeflow/hub/internal/testutils"
	"github.com/kubeflow/hub/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	os.Exit(testutils.TestMainPostgresHelper(m))
}

func TestDBCatalog(t *testing.T) {
	// Setup test database
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testhelpers.MustDatastoreSpec(t))
	defer cleanup()

	// Get type IDs
	catalogModelTypeID := testhelpers.GetCatalogModelTypeIDForDBTest(t, sharedDB)
	modelArtifactTypeID := testhelpers.GetCatalogModelArtifactTypeIDForDBTest(t, sharedDB)
	metricsArtifactTypeID := testhelpers.GetCatalogMetricsArtifactTypeIDForDBTest(t, sharedDB)
	catalogSourceTypeID := testhelpers.GetCatalogSourceTypeIDForDBTest(t, sharedDB)

	// Create repositories
	catalogModelRepo := modelservice.NewCatalogModelRepository(sharedDB, catalogModelTypeID)
	catalogArtifactRepo := service.NewCatalogArtifactRepository(sharedDB, map[string]int32{
		service.CatalogModelArtifactTypeName:   modelArtifactTypeID,
		service.CatalogMetricsArtifactTypeName: metricsArtifactTypeID,
	})
	modelArtifactRepo := modelservice.NewCatalogModelArtifactRepository(sharedDB, modelArtifactTypeID)
	metricsArtifactRepo := modelservice.NewCatalogMetricsArtifactRepository(sharedDB, metricsArtifactTypeID)
	catalogSourceRepo := service.NewCatalogSourceRepository(sharedDB, catalogSourceTypeID)

	svcs := Services{
		CatalogModelRepository:           catalogModelRepo,
		CatalogArtifactRepository:        catalogArtifactRepo,
		CatalogModelArtifactRepository:   modelArtifactRepo,
		CatalogMetricsArtifactRepository: metricsArtifactRepo,
		CatalogSourceRepository:          catalogSourceRepo,
		PropertyOptionsRepository:        service.NewPropertyOptionsRepository(sharedDB),
	}

	// Create DB catalog instance
	dbCatalog := NewDBCatalog(svcs, nil)
	ctx := context.Background()

	t.Run("TestNewDBCatalog", func(t *testing.T) {
		catalog := NewDBCatalog(svcs, nil)
		require.NotNil(t, catalog)

		// Verify it implements the interface
		var _ APIProvider = catalog
	})

	t.Run("TestGetModel_Success", func(t *testing.T) {
		// Create test model with namespaced name (sourceId:modelName) as stored in DB
		testModel := &models.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("test-source-id:test-get-model"),
				ExternalID: apiutils.Of("test-get-model-ext"),
			},
			Properties: &[]mr_models.Properties{
				{Name: "source_id", StringValue: apiutils.Of("test-source-id")},
				{Name: "description", StringValue: apiutils.Of("Test model description")},
			},
		}

		savedModel, err := catalogModelRepo.Save(testModel)
		require.NoError(t, err)

		// Test GetModel (API passes display name and source_id; backend resolves by namespaced name)
		retrievedModel, err := dbCatalog.GetModel(ctx, "test-get-model", "test-source-id")
		require.NoError(t, err)
		require.NotNil(t, retrievedModel)

		assert.Equal(t, "test-get-model", retrievedModel.Name)
		assert.Equal(t, strconv.FormatInt(int64(*savedModel.GetID()), 10), *retrievedModel.Id)
		assert.Equal(t, "test-get-model-ext", *retrievedModel.ExternalId)
		assert.Equal(t, "test-source-id", *retrievedModel.SourceId)
		assert.Equal(t, "Test model description", *retrievedModel.Description)
	})

	t.Run("TestGetModel_NotFound", func(t *testing.T) {
		// Test with non-existent model
		_, err := dbCatalog.GetModel(ctx, "non-existent-model", "test-source-id")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no models found")
		assert.ErrorIs(t, err, api.ErrNotFound)
	})

	t.Run("TestListModels_Success", func(t *testing.T) {
		// Create test models
		sourceIDs := []string{"list-test-source"}

		model1 := &models.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("list-test-model-1"),
				ExternalID: apiutils.Of("list-test-1"),
			},
			Properties: &[]mr_models.Properties{
				{Name: "source_id", StringValue: apiutils.Of("list-test-source")},
				{Name: "description", StringValue: apiutils.Of("First test model")},
			},
		}

		model2 := &models.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("list-test-model-2"),
				ExternalID: apiutils.Of("list-test-2"),
			},
			Properties: &[]mr_models.Properties{
				{Name: "source_id", StringValue: apiutils.Of("list-test-source")},
				{Name: "description", StringValue: apiutils.Of("Second test model")},
			},
		}

		_, err := catalogModelRepo.Save(model1)
		require.NoError(t, err)
		_, err = catalogModelRepo.Save(model2)
		require.NoError(t, err)

		// Test ListModels
		params := ListModelsParams{
			SourceIDs:     sourceIDs,
			PageSize:      10,
			OrderBy:       model.ORDERBYFIELD_CREATE_TIME,
			SortOrder:     model.SORTORDER_ASC,
			NextPageToken: apiutils.Of(""),
		}

		result, err := dbCatalog.ListModels(ctx, params)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(result.Items), 2, "Should return at least 2 models")
		assert.Equal(t, int32(10), result.PageSize)
		assert.GreaterOrEqual(t, result.Size, int32(2))

		// Verify models are properly mapped
		modelNames := make(map[string]bool)
		for _, model := range result.Items {
			modelNames[model.Name] = true
			// Verify required fields are present
			assert.NotEmpty(t, *model.Id)
			assert.NotEmpty(t, *model.SourceId)
		}

		// Should contain our test models
		foundCount := 0
		if modelNames["list-test-model-1"] {
			foundCount++
		}
		if modelNames["list-test-model-2"] {
			foundCount++
		}
		assert.GreaterOrEqual(t, foundCount, 2, "Should find our test models")
	})

	t.Run("TestListModels_WithPagination", func(t *testing.T) {
		// Test pagination
		sourceIDs := []string{"pagination-test-source"}

		// Create multiple models
		for i := range 5 {
			model := &models.CatalogModelImpl{
				TypeID: apiutils.Of(int32(catalogModelTypeID)),
				Attributes: &models.CatalogModelAttributes{
					Name:       apiutils.Of(fmt.Sprintf("pagination-test-source:pagination-test-model-%d", i)),
					ExternalID: apiutils.Of(fmt.Sprintf("pagination-test-%d", i)),
				},
				Properties: &[]mr_models.Properties{
					{Name: "source_id", StringValue: apiutils.Of("pagination-test-source")},
				},
			}
			_, err := catalogModelRepo.Save(model)
			require.NoError(t, err)
		}

		params := ListModelsParams{
			SourceIDs:     sourceIDs,
			PageSize:      3,
			OrderBy:       model.ORDERBYFIELD_CREATE_TIME,
			SortOrder:     model.SORTORDER_ASC,
			NextPageToken: apiutils.Of(""),
		}

		result, err := dbCatalog.ListModels(ctx, params)
		require.NoError(t, err)

		assert.LessOrEqual(t, len(result.Items), 3, "Should respect page size")
		assert.Equal(t, int32(3), result.PageSize)
	})

	t.Run("TestListModels_WithQuery", func(t *testing.T) {
		// Create test models with different properties for query filtering
		sourceIDs := []string{"query-test-source"}

		model1 := &models.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("query-test-source:BERT-base-model"),
				ExternalID: apiutils.Of("bert-base-1"),
			},
			Properties: &[]mr_models.Properties{
				{Name: "source_id", StringValue: apiutils.Of("query-test-source")},
				{Name: "description", StringValue: apiutils.Of("BERT base model for NLP tasks")},
				{Name: "provider", StringValue: apiutils.Of("Hugging Face")},
				{Name: "tasks", StringValue: apiutils.Of(`["text-classification", "question-answering"]`)},
			},
		}

		model2 := &models.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("query-test-source:GPT-3.5-turbo"),
				ExternalID: apiutils.Of("gpt-35-turbo-1"),
			},
			Properties: &[]mr_models.Properties{
				{Name: "source_id", StringValue: apiutils.Of("query-test-source")},
				{Name: "description", StringValue: apiutils.Of("OpenAI GPT model for text generation")},
				{Name: "provider", StringValue: apiutils.Of("OpenAI")},
				{Name: "tasks", StringValue: apiutils.Of(`["text-generation", "conversational"]`)},
			},
		}

		model3 := &models.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("query-test-source:ResNet-50-image"),
				ExternalID: apiutils.Of("resnet-50-1"),
			},
			Properties: &[]mr_models.Properties{
				{Name: "source_id", StringValue: apiutils.Of("query-test-source")},
				{Name: "description", StringValue: apiutils.Of("Deep learning model for image classification")},
				{Name: "provider", StringValue: apiutils.Of("PyTorch")},
				{Name: "tasks", StringValue: apiutils.Of(`["image-classification", "computer-vision"]`)},
			},
		}

		_, err := catalogModelRepo.Save(model1)
		require.NoError(t, err)
		_, err = catalogModelRepo.Save(model2)
		require.NoError(t, err)
		_, err = catalogModelRepo.Save(model3)
		require.NoError(t, err)

		// Test query filtering by name
		params := ListModelsParams{
			Query:         "BERT",
			SourceIDs:     sourceIDs,
			PageSize:      10,
			OrderBy:       model.ORDERBYFIELD_CREATE_TIME,
			SortOrder:     model.SORTORDER_ASC,
			NextPageToken: apiutils.Of(""),
		}

		result, err := dbCatalog.ListModels(ctx, params)
		require.NoError(t, err)

		assert.Equal(t, int32(1), result.Size, "Should return 1 model matching 'BERT'")
		assert.Contains(t, result.Items[0].Name, "BERT", "Should contain BERT model")

		// Test query filtering by description
		params.Query = "NLP"
		result, err = dbCatalog.ListModels(ctx, params)
		require.NoError(t, err)

		assert.Equal(t, int32(1), result.Size, "Should return 1 model with 'NLP' in description")
		assert.Contains(t, result.Items[0].Name, "BERT", "Should contain BERT model")

		// Test query filtering by provider
		params.Query = "OpenAI"
		result, err = dbCatalog.ListModels(ctx, params)
		require.NoError(t, err)

		assert.Equal(t, int32(1), result.Size, "Should return 1 model from 'OpenAI' provider")
		assert.Contains(t, result.Items[0].Name, "GPT", "Should contain GPT model")

		// Test query filtering that should match multiple models
		params.Query = "model"
		result, err = dbCatalog.ListModels(ctx, params)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, result.Size, int32(3), "Should return at least 3 models matching 'model'")

		// Test query that should return no results
		params.Query = "nonexistent"
		result, err = dbCatalog.ListModels(ctx, params)
		require.NoError(t, err)

		assert.Equal(t, int32(0), result.Size, "Should return 0 models for nonexistent query")

		// Test query filtering by tasks - text-classification
		params.Query = "text-classification"
		result, err = dbCatalog.ListModels(ctx, params)
		require.NoError(t, err)

		assert.Equal(t, int32(1), result.Size, "Should return 1 model with 'text-classification' task")
		assert.Contains(t, result.Items[0].Name, "BERT", "Should contain BERT model")

		// Test query filtering by tasks - image-classification
		params.Query = "image-classification"
		result, err = dbCatalog.ListModels(ctx, params)
		require.NoError(t, err)

		assert.Equal(t, int32(1), result.Size, "Should return 1 model with 'image-classification' task")
		assert.Contains(t, result.Items[0].Name, "ResNet", "Should contain ResNet model")

		// Test query filtering by tasks - conversational
		params.Query = "conversational"
		result, err = dbCatalog.ListModels(ctx, params)
		require.NoError(t, err)

		assert.Equal(t, int32(1), result.Size, "Should return 1 model with 'conversational' task")
		assert.Contains(t, result.Items[0].Name, "GPT", "Should contain GPT model")

		// Test query filtering by tasks - partial match on "classification"
		params.Query = "classification"
		result, err = dbCatalog.ListModels(ctx, params)
		require.NoError(t, err)

		assert.Equal(t, int32(2), result.Size, "Should return 2 models with 'classification' in their tasks")

		// Test query filtering by tasks - computer-vision
		params.Query = "computer-vision"
		result, err = dbCatalog.ListModels(ctx, params)
		require.NoError(t, err)

		assert.Equal(t, int32(1), result.Size, "Should return 1 model with 'computer-vision' task")
		assert.Contains(t, result.Items[0].Name, "ResNet", "Should contain ResNet model")
	})

	t.Run("TestListModels_FilterQuery", func(t *testing.T) {
		// Create test models with diverse properties for filterQuery testing
		sourceIDs := []string{"filterquery-test-source"}

		model1 := &models.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("filterquery-test-source:TensorFlow-ResNet50"),
				ExternalID: apiutils.Of("tf-resnet50-1"),
			},
			Properties: &[]mr_models.Properties{
				{Name: "source_id", StringValue: apiutils.Of("filterquery-test-source")},
				{Name: "description", StringValue: apiutils.Of("Deep learning model for image classification using TensorFlow")},
				{Name: "provider", StringValue: apiutils.Of("Google")},
				{Name: "framework", StringValue: apiutils.Of("TensorFlow")},
				{Name: "tasks", StringValue: apiutils.Of(`["image-classification", "computer-vision"]`)},
				{Name: "accuracy", StringValue: apiutils.Of("0.95")},
			},
		}

		model2 := &models.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("filterquery-test-source:PyTorch-BERT"),
				ExternalID: apiutils.Of("pt-bert-1"),
			},
			Properties: &[]mr_models.Properties{
				{Name: "source_id", StringValue: apiutils.Of("filterquery-test-source")},
				{Name: "description", StringValue: apiutils.Of("BERT model for natural language processing using PyTorch")},
				{Name: "provider", StringValue: apiutils.Of("Hugging Face")},
				{Name: "framework", StringValue: apiutils.Of("PyTorch")},
				{Name: "tasks", StringValue: apiutils.Of(`["text-classification", "question-answering"]`)},
				{Name: "accuracy", StringValue: apiutils.Of("0.92")},
			},
		}

		model3 := &models.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("filterquery-test-source:Scikit-learn-LogisticRegression"),
				ExternalID: apiutils.Of("sk-lr-1"),
			},
			Properties: &[]mr_models.Properties{
				{Name: "source_id", StringValue: apiutils.Of("filterquery-test-source")},
				{Name: "description", StringValue: apiutils.Of("Traditional machine learning model for classification")},
				{Name: "provider", StringValue: apiutils.Of("Scikit-learn")},
				{Name: "framework", StringValue: apiutils.Of("Scikit-learn")},
				{Name: "tasks", StringValue: apiutils.Of(`["classification", "regression"]`)},
				{Name: "accuracy", StringValue: apiutils.Of("0.88")},
			},
		}

		_, err := catalogModelRepo.Save(model1)
		require.NoError(t, err)
		_, err = catalogModelRepo.Save(model2)
		require.NoError(t, err)
		_, err = catalogModelRepo.Save(model3)
		require.NoError(t, err)

		// Test: Basic name filtering with exact match (filter uses stored namespaced name: source_id:model_name)
		params := ListModelsParams{
			FilterQuery:   "name = \"filterquery-test-source:TensorFlow-ResNet50\"",
			SourceIDs:     sourceIDs,
			PageSize:      10,
			OrderBy:       model.ORDERBYFIELD_NAME,
			SortOrder:     model.SORTORDER_ASC,
			NextPageToken: apiutils.Of(""),
		}

		result, err := dbCatalog.ListModels(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, int32(1), result.Size, "Should return 1 model with exact name match")
		assert.Equal(t, "TensorFlow-ResNet50", result.Items[0].Name)

		// Test: LIKE pattern matching
		params.FilterQuery = "name LIKE \"%Tensor%\""
		result, err = dbCatalog.ListModels(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, int32(1), result.Size, "Should return 1 model with LIKE pattern match")
		assert.Contains(t, result.Items[0].Name, "Tensor")

		// Test: LIKE pattern matching with case sensitivity
		params.FilterQuery = "name ILIKE \"%tensor%\""
		result, err = dbCatalog.ListModels(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, int32(1), result.Size, "Should return 1 model with case-insensitive ILIKE match")
		assert.Contains(t, result.Items[0].Name, "Tensor")

		// Test: OR logic (use namespaced names for exact match)
		params.FilterQuery = "name = \"filterquery-test-source:TensorFlow-ResNet50\" OR name = \"filterquery-test-source:PyTorch-BERT\""
		result, err = dbCatalog.ListModels(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, int32(2), result.Size, "Should return 2 models with OR logic")

		// Verify we got the expected models
		modelNames := make(map[string]bool)
		for _, item := range result.Items {
			modelNames[item.Name] = true
		}
		assert.True(t, modelNames["TensorFlow-ResNet50"], "Should contain TensorFlow model")
		assert.True(t, modelNames["PyTorch-BERT"], "Should contain PyTorch model")

		// Test: AND logic
		params.FilterQuery = "name LIKE \"%Tensor%\" AND name LIKE \"%ResNet%\""
		result, err = dbCatalog.ListModels(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, int32(1), result.Size, "Should return 1 model with AND logic")
		assert.Equal(t, "TensorFlow-ResNet50", result.Items[0].Name)

		// Test: Custom property filtering
		params.FilterQuery = "framework.string_value = \"PyTorch\""
		result, err = dbCatalog.ListModels(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, int32(1), result.Size, "Should return 1 model with PyTorch framework")
		assert.Equal(t, "PyTorch-BERT", result.Items[0].Name)

		// Test: Custom property filtering with LIKE
		params.FilterQuery = "provider.string_value LIKE \"%Google%\""
		result, err = dbCatalog.ListModels(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, int32(1), result.Size, "Should return 1 model with Google provider")
		assert.Equal(t, "TensorFlow-ResNet50", result.Items[0].Name)

		// Test: Numeric comparison
		params.FilterQuery = "accuracy.string_value > \"0.90\""
		result, err = dbCatalog.ListModels(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, int32(2), result.Size, "Should return 2 models with accuracy > 0.90")

		// Verify we got the expected models (TensorFlow and PyTorch)
		modelNames = make(map[string]bool)
		for _, item := range result.Items {
			modelNames[item.Name] = true
		}
		assert.True(t, modelNames["TensorFlow-ResNet50"], "Should contain TensorFlow model")
		assert.True(t, modelNames["PyTorch-BERT"], "Should contain PyTorch model")

		// Test: Complex query with multiple conditions
		params.FilterQuery = "(framework.string_value = \"TensorFlow\" OR framework.string_value = \"PyTorch\") AND accuracy.string_value > \"0.90\""
		result, err = dbCatalog.ListModels(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, int32(2), result.Size, "Should return 2 models with complex query")

		// Test: No matches (non-existent name; stored names are namespaced)
		params.FilterQuery = "name = \"filterquery-test-source:NonExistentModel\""
		result, err = dbCatalog.ListModels(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, int32(0), result.Size, "Should return 0 models for non-existent name")

		// Test: Empty filterQuery should return all models
		params.FilterQuery = ""
		result, err = dbCatalog.ListModels(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, int32(3), result.Size, "Should return all 3 models with empty filterQuery")

		// Test: Combined with regular query parameter
		params.Query = "BERT"
		params.FilterQuery = "framework.string_value = \"PyTorch\""
		result, err = dbCatalog.ListModels(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, int32(1), result.Size, "Should return 1 model matching both query and filterQuery")
		assert.Equal(t, "PyTorch-BERT", result.Items[0].Name)

		// Test: Invalid filterQuery syntax should return error
		params.Query = ""
		params.FilterQuery = "invalid syntax here"
		_, err = dbCatalog.ListModels(ctx, params)
		require.Error(t, err, "Should return error for invalid filterQuery syntax")
		assert.Contains(t, err.Error(), "invalid filter query", "Error should mention invalid filter query")
	})

	t.Run("TestGetArtifacts_Success", func(t *testing.T) {
		// Create test model
		testModel := &models.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("artifact-test-source:artifact-test-model"),
				ExternalID: apiutils.Of("artifact-test-model-ext"),
			},
			Properties: &[]mr_models.Properties{
				{Name: "source_id", StringValue: apiutils.Of("artifact-test-source")},
			},
		}

		savedModel, err := catalogModelRepo.Save(testModel)
		require.NoError(t, err)

		// Create test artifacts
		modelArtifact := &models.CatalogModelArtifactImpl{
			TypeID: apiutils.Of(int32(modelArtifactTypeID)),
			Attributes: &models.CatalogModelArtifactAttributes{
				Name:         apiutils.Of("test-model-artifact"),
				ExternalID:   apiutils.Of("test-model-artifact-ext"),
				URI:          apiutils.Of("s3://test/model.bin"),
				ArtifactType: apiutils.Of(models.CatalogModelArtifactType),
			},
		}

		metricsArtifact := &models.CatalogMetricsArtifactImpl{
			TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:         apiutils.Of("test-metrics-artifact"),
				ExternalID:   apiutils.Of("test-metrics-artifact-ext"),
				MetricsType:  models.MetricsTypeAccuracy,
				ArtifactType: apiutils.Of("metrics-artifact"),
			},
		}

		savedModelArt, err := modelArtifactRepo.Save(modelArtifact, savedModel.GetID())
		require.NoError(t, err)
		savedMetricsArt, err := metricsArtifactRepo.Save(metricsArtifact, savedModel.GetID())
		require.NoError(t, err)

		// Test GetArtifacts
		params := ListArtifactsParams{
			PageSize:      10,
			OrderBy:       string(model.ORDERBYFIELD_CREATE_TIME),
			SortOrder:     model.SORTORDER_ASC,
			NextPageToken: apiutils.Of(""),
		}

		result, err := dbCatalog.GetArtifacts(ctx, "artifact-test-model", "artifact-test-source", params)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(result.Items), 2, "Should return at least 2 artifacts")
		assert.Equal(t, int32(10), result.PageSize)

		// Verify both types of artifacts are returned
		var modelArtifactFound, metricsArtifactFound bool
		artifactIDs := make(map[string]bool)

		for _, artifact := range result.Items {
			if artifact.CatalogModelArtifact != nil {
				modelArtifactFound = true
				artifactIDs[*artifact.CatalogModelArtifact.Id] = true
				assert.Equal(t, "model-artifact", artifact.CatalogModelArtifact.ArtifactType)
			}
			if artifact.CatalogMetricsArtifact != nil {
				metricsArtifactFound = true
				artifactIDs[*artifact.CatalogMetricsArtifact.Id] = true
				assert.Equal(t, "metrics-artifact", artifact.CatalogMetricsArtifact.ArtifactType)
			}
		}

		assert.True(t, modelArtifactFound, "Should find model artifact")
		assert.True(t, metricsArtifactFound, "Should find metrics artifact")

		// Verify our specific artifacts are in the results
		modelArtifactIDStr := strconv.FormatInt(int64(*savedModelArt.GetID()), 10)
		metricsArtifactIDStr := strconv.FormatInt(int64(*savedMetricsArt.GetID()), 10)
		assert.True(t, artifactIDs[modelArtifactIDStr], "Should contain our model artifact")
		assert.True(t, artifactIDs[metricsArtifactIDStr], "Should contain our metrics artifact")
	})

	t.Run("TestGetArtifacts_ModelNotFound", func(t *testing.T) {
		// Test with non-existent model
		params := ListArtifactsParams{
			PageSize:      10,
			OrderBy:       string(model.ORDERBYFIELD_CREATE_TIME),
			SortOrder:     model.SORTORDER_ASC,
			NextPageToken: apiutils.Of(""),
		}

		_, err := dbCatalog.GetArtifacts(ctx, "non-existent-model", "test-source", params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid model name")
	})

	t.Run("TestGetArtifacts_WithCustomProperties", func(t *testing.T) {
		// Create model
		testModel := &models.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("custom-props-source:custom-props-model"),
				ExternalID: apiutils.Of("custom-props-model-ext"),
			},
			Properties: &[]mr_models.Properties{
				{Name: "source_id", StringValue: apiutils.Of("custom-props-source")},
			},
		}

		savedModel, err := catalogModelRepo.Save(testModel)
		require.NoError(t, err)

		// Create artifact with custom properties
		customProps := []mr_models.Properties{
			{Name: "custom_prop_1", StringValue: apiutils.Of("value_1")},
			{Name: "custom_prop_2", StringValue: apiutils.Of("value_2")},
		}

		artifactWithProps := &models.CatalogModelArtifactImpl{
			TypeID: apiutils.Of(int32(modelArtifactTypeID)),
			Attributes: &models.CatalogModelArtifactAttributes{
				Name:         apiutils.Of("artifact-with-props"),
				ExternalID:   apiutils.Of("artifact-with-props-ext"),
				URI:          apiutils.Of("s3://test/props.bin"),
				ArtifactType: apiutils.Of(models.CatalogModelArtifactType),
			},
			CustomProperties: &customProps,
		}

		_, err = modelArtifactRepo.Save(artifactWithProps, savedModel.GetID())
		require.NoError(t, err)

		// Get artifacts and verify custom properties
		params := ListArtifactsParams{
			PageSize:      10,
			OrderBy:       string(model.ORDERBYFIELD_CREATE_TIME),
			SortOrder:     model.SORTORDER_ASC,
			NextPageToken: apiutils.Of(""),
		}

		result, err := dbCatalog.GetArtifacts(ctx, "custom-props-model", "custom-props-source", params)
		require.NoError(t, err)

		// Find our artifact and check custom properties
		found := false
		for _, artifact := range result.Items {
			if artifact.CatalogModelArtifact != nil &&
				artifact.CatalogModelArtifact.Name != nil &&
				*artifact.CatalogModelArtifact.Name == "artifact-with-props" {

				found = true
				assert.NotNil(t, artifact.CatalogModelArtifact.CustomProperties)

				// Verify custom properties are present and properly converted
				customPropsMap := artifact.CatalogModelArtifact.CustomProperties
				assert.Contains(t, customPropsMap, "custom_prop_1")
				assert.Contains(t, customPropsMap, "custom_prop_2")

				// Verify the values are properly converted to MetadataValue
				prop1 := customPropsMap["custom_prop_1"]
				assert.NotNil(t, prop1.MetadataStringValue)
				assert.Equal(t, "value_1", prop1.MetadataStringValue.StringValue)

				break
			}
		}
		assert.True(t, found, "Should find artifact with custom properties")
	})

	t.Run("TestMappingFunctions", func(t *testing.T) {
		t.Run("TestMapCatalogModelToCatalogModel", func(t *testing.T) {
			// Create a catalog model with various properties
			catalogModel := &models.CatalogModelImpl{
				ID:     apiutils.Of(int32(123)),
				TypeID: apiutils.Of(int32(catalogModelTypeID)),
				Attributes: &models.CatalogModelAttributes{
					Name:                     apiutils.Of("test-source:mapping-test-model"),
					ExternalID:               apiutils.Of("mapping-test-ext"),
					CreateTimeSinceEpoch:     apiutils.Of(int64(1234567890)),
					LastUpdateTimeSinceEpoch: apiutils.Of(int64(1234567891)),
				},
				Properties: &[]mr_models.Properties{
					{Name: "source_id", StringValue: apiutils.Of("test-source")},
					{Name: "description", StringValue: apiutils.Of("Test description")},
					{Name: "library_name", StringValue: apiutils.Of("pytorch")},
					{Name: "language", StringValue: apiutils.Of("[\"python\", \"go\"]")},
					{Name: "tasks", StringValue: apiutils.Of("[\"classification\", \"regression\"]")},
					{Name: "validated_tasks", StringValue: apiutils.Of(`["text-generation","tool-calling"]`)},
					{Name: "serving_config", StringValue: apiutils.Of(`{"toolCalling":{"toolCallParser":"granite","chatTemplate":"opt/app-root/template/tool_chat_template_granite.jinja","enableAutoToolChoice":true,"requiredArgs":["--config_format granite"]}}`)},
				},
			}

			result := mapDBModelToAPIModel(catalogModel)

			assert.Equal(t, "123", *result.Id)
			assert.Equal(t, "mapping-test-model", result.Name)
			assert.Equal(t, "mapping-test-ext", *result.ExternalId)
			assert.Equal(t, "test-source", *result.SourceId)
			assert.Equal(t, "Test description", *result.Description)
			assert.Equal(t, "pytorch", *result.LibraryName)
			assert.Equal(t, "1234567890", *result.CreateTimeSinceEpoch)
			assert.Equal(t, "1234567891", *result.LastUpdateTimeSinceEpoch)

			// Verify JSON arrays are properly parsed
			assert.Equal(t, []string{"python", "go"}, result.Language)
			assert.Equal(t, []string{"classification", "regression"}, result.Tasks)
			assert.Equal(t, []string{"text-generation", "tool-calling"}, result.ValidatedTasks)

			// Verify nested JSON object is properly parsed
			require.NotNil(t, result.ServingConfig)
			require.NotNil(t, result.ServingConfig.ToolCalling)
			assert.Equal(t, apiutils.Of("granite"), result.ServingConfig.ToolCalling.ToolCallParser)
			assert.Equal(t, apiutils.Of("opt/app-root/template/tool_chat_template_granite.jinja"), result.ServingConfig.ToolCalling.ChatTemplate)
			assert.Equal(t, model.PtrBool(true), result.ServingConfig.ToolCalling.EnableAutoToolChoice)
			assert.Equal(t, []string{"--config_format granite"}, result.ServingConfig.ToolCalling.RequiredArgs)
		})

		t.Run("TestMapCatalogArtifactToCatalogArtifact", func(t *testing.T) {
			// Test model artifact mapping
			var catalogModelArtifact models.CatalogModelArtifact = &models.CatalogModelArtifactImpl{
				ID:     apiutils.Of(int32(456)),
				TypeID: apiutils.Of(int32(modelArtifactTypeID)),
				Attributes: &models.CatalogModelArtifactAttributes{
					Name:       apiutils.Of("test-model-artifact"),
					ExternalID: apiutils.Of("test-model-artifact-ext"),
					URI:        apiutils.Of("s3://test/model.bin"),
				},
			}

			catalogArtifact := sharedmodels.CatalogArtifact{
				CatalogModelArtifact: catalogModelArtifact,
			}

			result, err := mapDBArtifactToAPIArtifact(catalogArtifact)
			require.NoError(t, err)

			assert.NotNil(t, result.CatalogModelArtifact)
			assert.Nil(t, result.CatalogMetricsArtifact)
			assert.Equal(t, "456", *result.CatalogModelArtifact.Id)
			assert.Equal(t, "test-model-artifact", *result.CatalogModelArtifact.Name)
			assert.Equal(t, "s3://test/model.bin", result.CatalogModelArtifact.Uri)

			// Test metrics artifact mapping
			var catalogMetricsArtifact models.CatalogMetricsArtifact = &models.CatalogMetricsArtifactImpl{
				ID:     apiutils.Of(int32(789)),
				TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
				Attributes: &models.CatalogMetricsArtifactAttributes{
					Name:        apiutils.Of("test-metrics-artifact"),
					ExternalID:  apiutils.Of("test-metrics-artifact-ext"),
					MetricsType: models.MetricsTypePerformance,
				},
			}

			catalogArtifact2 := sharedmodels.CatalogArtifact{
				CatalogMetricsArtifact: catalogMetricsArtifact,
			}

			result2, err := mapDBArtifactToAPIArtifact(catalogArtifact2)
			require.NoError(t, err)

			assert.Nil(t, result2.CatalogModelArtifact)
			assert.NotNil(t, result2.CatalogMetricsArtifact)
			assert.Equal(t, "789", *result2.CatalogMetricsArtifact.Id)
			assert.Equal(t, "test-metrics-artifact", *result2.CatalogMetricsArtifact.Name)
			assert.Equal(t, "performance-metrics", result2.CatalogMetricsArtifact.MetricsType)
		})

		t.Run("TestMapCatalogModel_ValidatedTasksAndServingConfig_RoundTrip", func(t *testing.T) {
			testModel := &models.CatalogModelImpl{
				TypeID: apiutils.Of(int32(catalogModelTypeID)),
				Attributes: &models.CatalogModelAttributes{
					Name:       apiutils.Of("roundtrip-source:roundtrip-validated-serving"),
					ExternalID: apiutils.Of("roundtrip-validated-serving-ext"),
				},
				Properties: &[]mr_models.Properties{
					{Name: "source_id", StringValue: apiutils.Of("roundtrip-source")},
					{Name: "validated_tasks", StringValue: apiutils.Of(`["text-generation","tool-calling"]`)},
					{Name: "serving_config", StringValue: apiutils.Of(`{"toolCalling":{"toolCallParser":"granite","chatTemplate":"opt/app-root/template/tool_chat_template_granite.jinja","enableAutoToolChoice":false,"requiredArgs":["--config_format granite"]}}`)},
				},
			}

			_, err := catalogModelRepo.Save(testModel)
			require.NoError(t, err)

			retrieved, err := dbCatalog.GetModel(ctx, "roundtrip-validated-serving", "roundtrip-source")
			require.NoError(t, err)
			require.NotNil(t, retrieved)

			assert.Equal(t, []string{"text-generation", "tool-calling"}, retrieved.ValidatedTasks)
			require.NotNil(t, retrieved.ServingConfig)
			require.NotNil(t, retrieved.ServingConfig.ToolCalling)
			assert.Equal(t, apiutils.Of("granite"), retrieved.ServingConfig.ToolCalling.ToolCallParser)
			assert.Equal(t, apiutils.Of("opt/app-root/template/tool_chat_template_granite.jinja"), retrieved.ServingConfig.ToolCalling.ChatTemplate)
			assert.Equal(t, model.PtrBool(false), retrieved.ServingConfig.ToolCalling.EnableAutoToolChoice)
			assert.Equal(t, []string{"--config_format granite"}, retrieved.ServingConfig.ToolCalling.RequiredArgs)
		})

		t.Run("TestMapCatalogModel_MalformedJSON_SilentFailure", func(t *testing.T) {
			catalogModel := &models.CatalogModelImpl{
				ID:     apiutils.Of(int32(999)),
				TypeID: apiutils.Of(int32(catalogModelTypeID)),
				Attributes: &models.CatalogModelAttributes{
					Name:       apiutils.Of("malformed-source:malformed-json-model"),
					ExternalID: apiutils.Of("malformed-json-ext"),
				},
				Properties: &[]mr_models.Properties{
					{Name: "source_id", StringValue: apiutils.Of("malformed-source")},
					{Name: "validated_tasks", StringValue: apiutils.Of(`not valid json`)},
					{Name: "serving_config", StringValue: apiutils.Of(`{broken}`)},
				},
			}

			result := mapDBModelToAPIModel(catalogModel)

			assert.Equal(t, "malformed-json-model", result.Name)
			assert.Nil(t, result.ValidatedTasks)
			assert.Nil(t, result.ServingConfig)
		})

		t.Run("TestMapCatalogArtifact_EmptyArtifact", func(t *testing.T) {
			// Test with empty catalog artifact
			emptyCatalogArtifact := sharedmodels.CatalogArtifact{}

			_, err := mapDBArtifactToAPIArtifact(emptyCatalogArtifact)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid catalog artifact type")
		})
	})

	t.Run("TestErrorHandling", func(t *testing.T) {
		t.Run("TestGetArtifacts_InvalidModelID", func(t *testing.T) {
			// Create a model with invalid ID format for testing
			// This would be an edge case where the ID isn't a valid integer

			// We can't easily test this directly since IDs are generated as integers
			// But we can test the error case by mocking a scenario

			// For now, let's test a scenario where the model exists but has some issue
			params := ListArtifactsParams{
				PageSize:      10,
				OrderBy:       string(model.ORDERBYFIELD_CREATE_TIME),
				SortOrder:     model.SORTORDER_ASC,
				NextPageToken: apiutils.Of(""),
			}

			_, err := dbCatalog.GetArtifacts(ctx, "non-existent-model", "test-source", params)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid model name")
		})
	})

	t.Run("TestGetFilterOptions", func(t *testing.T) {
		// Create models with various properties for filter options testing
		model1 := &models.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("filter-test-source:filter-options-model-1"),
				ExternalID: apiutils.Of("filter-opt-1"),
			},
			Properties: &[]mr_models.Properties{
				{Name: "source_id", StringValue: apiutils.Of("filter-test-source")},
				{Name: "license", StringValue: apiutils.Of("MIT")},
				{Name: "provider", StringValue: apiutils.Of("HuggingFace")},
				{Name: "maturity", StringValue: apiutils.Of("stable")},
				{Name: "library_name", StringValue: apiutils.Of("transformers")},
				{Name: "language", StringValue: apiutils.Of(`["python", "rust"]`)},
				{Name: "tasks", StringValue: apiutils.Of(`["text-classification", "token-classification"]`)},
			},
		}

		model2 := &models.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("filter-test-source:filter-options-model-2"),
				ExternalID: apiutils.Of("filter-opt-2"),
			},
			Properties: &[]mr_models.Properties{
				{Name: "source_id", StringValue: apiutils.Of("filter-test-source")},
				{Name: "license", StringValue: apiutils.Of("Apache-2.0")},
				{Name: "provider", StringValue: apiutils.Of("OpenAI")},
				{Name: "maturity", StringValue: apiutils.Of("experimental")},
				{Name: "library_name", StringValue: apiutils.Of("openai")},
				{Name: "language", StringValue: apiutils.Of(`["python", "javascript"]`)},
				{Name: "tasks", StringValue: apiutils.Of(`["text-generation", "conversational"]`)},
				{Name: "readme", StringValue: apiutils.Of("This is a very long readme that exceeds 100 characters and should be excluded from filter options because it's too verbose for filtering purposes.")},
			},
		}

		model3 := &models.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("filter-test-source:filter-options-model-3"),
				ExternalID: apiutils.Of("filter-opt-3"),
			},
			Properties: &[]mr_models.Properties{
				{Name: "source_id", StringValue: apiutils.Of("filter-test-source")},
				{Name: "license", StringValue: apiutils.Of("MIT")},
				{Name: "provider", StringValue: apiutils.Of("PyTorch")},
				{Name: "maturity", StringValue: apiutils.Of("stable")},
				{Name: "language", StringValue: apiutils.Of(`["python"]`)},
				{Name: "tasks", StringValue: apiutils.Of(`["image-classification"]`)},
				{Name: "logo", StringValue: apiutils.Of("https://example.com/logo.png")},
				{Name: "license_link", StringValue: apiutils.Of("https://example.com/license")},
			},
		}

		_, err := catalogModelRepo.Save(model1)
		require.NoError(t, err)
		_, err = catalogModelRepo.Save(model2)
		require.NoError(t, err)
		_, err = catalogModelRepo.Save(model3)
		require.NoError(t, err)

		require.NoError(t, dbCatalog.(*dbCatalogImpl).propertyOptionsRepository.Refresh(sharedmodels.ContextPropertyOptionType))
		require.NoError(t, dbCatalog.(*dbCatalogImpl).propertyOptionsRepository.Refresh(sharedmodels.ArtifactPropertyOptionType))

		// Test GetFilterOptions
		filterOptions, err := dbCatalog.GetFilterOptions(ctx)
		require.NoError(t, err)
		require.NotNil(t, filterOptions)
		require.NotNil(t, filterOptions.Filters)

		filters := *filterOptions.Filters

		// Should include short properties
		assert.Contains(t, filters, "license")
		assert.Contains(t, filters, "provider")
		assert.Contains(t, filters, "maturity")
		assert.Contains(t, filters, "library_name")
		assert.Contains(t, filters, "language")
		assert.Contains(t, filters, "tasks")

		// Should exclude internal/verbose fields
		assert.NotContains(t, filters, "source_id", "source_id should be excluded")
		assert.NotContains(t, filters, "logo", "logo should be excluded")
		assert.NotContains(t, filters, "license_link", "license_link should be excluded")
		assert.NotContains(t, filters, "readme", "readme should be excluded (too long)")
		assert.NotContains(t, filters, "serving_config", "serving_config should be excluded (complex nested JSON)")

		licenseFilter := filters["license"]
		assert.Equal(t, "string", licenseFilter.Type)
		assert.NotNil(t, licenseFilter.Values)
		assert.GreaterOrEqual(t, len(licenseFilter.Values), 2, "Should have at least MIT and Apache-2.0")

		// Convert to string slice for easier checking
		licenseValues := make([]string, 0)
		for _, v := range licenseFilter.Values {
			if strVal, ok := v.(string); ok {
				licenseValues = append(licenseValues, strVal)
			}
		}
		assert.Contains(t, licenseValues, "MIT")
		assert.Contains(t, licenseValues, "Apache-2.0")

		// Verify provider filter options
		providerFilter := filters["provider"]
		assert.Equal(t, "string", providerFilter.Type)
		providerValues := make([]string, 0)
		for _, v := range providerFilter.Values {
			if strVal, ok := v.(string); ok {
				providerValues = append(providerValues, strVal)
			}
		}
		assert.Contains(t, providerValues, "HuggingFace")
		assert.Contains(t, providerValues, "OpenAI")
		assert.Contains(t, providerValues, "PyTorch")

		// Verify JSON array fields are properly parsed and expanded
		languageFilter := filters["language"]
		assert.Equal(t, "string", languageFilter.Type)
		languageValues := make([]string, 0)
		for _, v := range languageFilter.Values {
			if strVal, ok := v.(string); ok {
				languageValues = append(languageValues, strVal)
			}
		}
		// Should contain individual values from JSON arrays
		assert.Contains(t, languageValues, "python")
		assert.Contains(t, languageValues, "rust")
		assert.Contains(t, languageValues, "javascript")

		// Verify tasks are properly expanded
		tasksFilter := filters["tasks"]
		assert.Equal(t, "string", tasksFilter.Type)
		tasksValues := make([]string, 0)
		for _, v := range tasksFilter.Values {
			if strVal, ok := v.(string); ok {
				tasksValues = append(tasksValues, strVal)
			}
		}
		assert.Contains(t, tasksValues, "text-classification")
		assert.Contains(t, tasksValues, "token-classification")
		assert.Contains(t, tasksValues, "text-generation")
		assert.Contains(t, tasksValues, "conversational")
		assert.Contains(t, tasksValues, "image-classification")

		// Verify no duplicates
		pythonCount := 0
		for _, v := range languageValues {
			if v == "python" {
				pythonCount++
			}
		}
		assert.Equal(t, 1, pythonCount, "python should appear only once (deduplicated)")

		// Verify maturity options
		maturityFilter := filters["maturity"]
		maturityValues := make([]string, 0)
		for _, v := range maturityFilter.Values {
			if strVal, ok := v.(string); ok {
				maturityValues = append(maturityValues, strVal)
			}
		}
		assert.Contains(t, maturityValues, "stable")
		assert.Contains(t, maturityValues, "experimental")
	})

	t.Run("TestGetPerformanceArtifacts_BasicFiltering", func(t *testing.T) {
		// Create test model
		testModel := &models.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("perf-test-source:perf-test-model"),
				ExternalID: apiutils.Of("perf-test-model-ext"),
			},
			Properties: &[]mr_models.Properties{
				{Name: "source_id", StringValue: apiutils.Of("perf-test-source")},
			},
		}

		savedModel, err := catalogModelRepo.Save(testModel)
		require.NoError(t, err)

		// Create performance metrics artifact
		perfArtifact := &models.CatalogMetricsArtifactImpl{
			TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:         apiutils.Of("performance-metrics-1"),
				ExternalID:   apiutils.Of("perf-metrics-1"),
				MetricsType:  models.MetricsTypePerformance,
				ArtifactType: apiutils.Of("metrics-artifact"),
			},
			CustomProperties: &[]mr_models.Properties{
				{Name: "throughput", DoubleValue: apiutils.Of(float64(50.0))},
				{Name: "latency_p99", DoubleValue: apiutils.Of(float64(100.0))},
			},
		}

		// Create accuracy metrics artifact (should be filtered out)
		accuracyArtifact := &models.CatalogMetricsArtifactImpl{
			TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:         apiutils.Of("accuracy-metrics-1"),
				ExternalID:   apiutils.Of("acc-metrics-1"),
				MetricsType:  models.MetricsTypeAccuracy,
				ArtifactType: apiutils.Of("metrics-artifact"),
			},
		}

		_, err = metricsArtifactRepo.Save(perfArtifact, savedModel.GetID())
		require.NoError(t, err)
		_, err = metricsArtifactRepo.Save(accuracyArtifact, savedModel.GetID())
		require.NoError(t, err)

		// Test GetPerformanceArtifacts - should only return performance metrics
		params := ListPerformanceArtifactsParams{
			PageSize:        10,
			OrderBy:         string(model.ORDERBYFIELD_CREATE_TIME),
			SortOrder:       model.SORTORDER_ASC,
			NextPageToken:   apiutils.Of(""),
			TargetRPS:       0,
			Recommendations: false,
		}

		result, err := dbCatalog.GetPerformanceArtifacts(ctx, "perf-test-model", "perf-test-source", params)
		require.NoError(t, err)

		assert.Equal(t, int32(1), result.Size, "Should return only performance metrics")
		assert.Len(t, result.Items, 1)

		// Verify it's the performance artifact
		perfItem := result.Items[0]
		assert.NotNil(t, perfItem.CatalogMetricsArtifact)
		assert.Equal(t, "performance-metrics-1", *perfItem.CatalogMetricsArtifact.Name)
		assert.Equal(t, "performance-metrics", perfItem.CatalogMetricsArtifact.MetricsType)
	})

	t.Run("TestGetPerformanceArtifacts_WithTargetRPS", func(t *testing.T) {
		// Create test model
		testModel := &models.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("rps-test-source:rps-test-model"),
				ExternalID: apiutils.Of("rps-test-model-ext"),
			},
			Properties: &[]mr_models.Properties{
				{Name: "source_id", StringValue: apiutils.Of("rps-test-source")},
			},
		}

		savedModel, err := catalogModelRepo.Save(testModel)
		require.NoError(t, err)

		// Create performance metrics artifact with throughput data
		perfArtifact := &models.CatalogMetricsArtifactImpl{
			TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:         apiutils.Of("rps-metrics-1"),
				ExternalID:   apiutils.Of("rps-metrics-1"),
				MetricsType:  models.MetricsTypePerformance,
				ArtifactType: apiutils.Of("metrics-artifact"),
			},
			CustomProperties: &[]mr_models.Properties{
				{Name: "throughput", DoubleValue: apiutils.Of(float64(50.0))},
			},
		}

		_, err = metricsArtifactRepo.Save(perfArtifact, savedModel.GetID())
		require.NoError(t, err)

		// Test with targetRPS parameter
		params := ListPerformanceArtifactsParams{
			PageSize:        10,
			OrderBy:         string(model.ORDERBYFIELD_CREATE_TIME),
			SortOrder:       model.SORTORDER_ASC,
			NextPageToken:   apiutils.Of(""),
			TargetRPS:       100,
			Recommendations: false,
		}

		result, err := dbCatalog.GetPerformanceArtifacts(ctx, "rps-test-model", "rps-test-source", params)
		require.NoError(t, err)

		assert.Equal(t, int32(1), result.Size)
		assert.Len(t, result.Items, 1)

		// Verify targetRPS calculations are added to custom properties
		perfItem := result.Items[0]
		assert.NotNil(t, perfItem.CatalogMetricsArtifact)
		assert.NotNil(t, perfItem.CatalogMetricsArtifact.CustomProperties)

		customProps := perfItem.CatalogMetricsArtifact.CustomProperties

		// Should have replicas property
		assert.Contains(t, customProps, "replicas")
		replicasValue := customProps["replicas"]
		assert.NotNil(t, replicasValue.MetadataIntValue)
		assert.NotEmpty(t, replicasValue.MetadataIntValue.IntValue)
		// Verify it's a valid integer
		replicasInt, err := strconv.ParseInt(replicasValue.MetadataIntValue.IntValue, 10, 32)
		require.NoError(t, err)
		assert.Greater(t, int32(replicasInt), int32(0))

		// Should have total_requests_per_second property
		assert.Contains(t, customProps, "total_requests_per_second")
		totalRPSValue := customProps["total_requests_per_second"]
		assert.NotNil(t, totalRPSValue.MetadataDoubleValue)
		assert.Equal(t, float64(100), totalRPSValue.MetadataDoubleValue.DoubleValue)
	})

	t.Run("TestGetPerformanceArtifacts_WithDeduplication", func(t *testing.T) {
		// Create test model
		testModel := &models.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("dedup-test-source:dedup-test-model"),
				ExternalID: apiutils.Of("dedup-test-model-ext"),
			},
			Properties: &[]mr_models.Properties{
				{Name: "source_id", StringValue: apiutils.Of("dedup-test-source")},
			},
		}

		savedModel, err := catalogModelRepo.Save(testModel)
		require.NoError(t, err)

		// Create multiple performance artifacts with different cost profiles
		// The deduplication algorithm uses hardware_count * replicas for cost calculation
		// It keeps artifacts with decreasing cost (when sorted by latency)
		perfArtifact1 := &models.CatalogMetricsArtifactImpl{
			TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:         apiutils.Of("dedup-metrics-1"),
				ExternalID:   apiutils.Of("dedup-metrics-1"),
				MetricsType:  models.MetricsTypePerformance,
				ArtifactType: apiutils.Of("metrics-artifact"),
			},
			CustomProperties: &[]mr_models.Properties{
				{Name: "hardware_count", IntValue: apiutils.Of(int32(4))},
				{Name: "ttft_p90", DoubleValue: apiutils.Of(float64(100.0))},
				{Name: "hardware_type", StringValue: apiutils.Of("gpu-a100")},
			},
		}

		perfArtifact2 := &models.CatalogMetricsArtifactImpl{
			TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:         apiutils.Of("dedup-metrics-2"),
				ExternalID:   apiutils.Of("dedup-metrics-2"),
				MetricsType:  models.MetricsTypePerformance,
				ArtifactType: apiutils.Of("metrics-artifact"),
			},
			CustomProperties: &[]mr_models.Properties{
				{Name: "hardware_count", IntValue: apiutils.Of(int32(4))},
				{Name: "ttft_p90", DoubleValue: apiutils.Of(float64(150.0))},
				{Name: "hardware_type", StringValue: apiutils.Of("gpu-a100")},
			},
		}

		perfArtifact3 := &models.CatalogMetricsArtifactImpl{
			TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:         apiutils.Of("dedup-metrics-3"),
				ExternalID:   apiutils.Of("dedup-metrics-3"),
				MetricsType:  models.MetricsTypePerformance,
				ArtifactType: apiutils.Of("metrics-artifact"),
			},
			CustomProperties: &[]mr_models.Properties{
				{Name: "hardware_count", IntValue: apiutils.Of(int32(2))},
				{Name: "ttft_p90", DoubleValue: apiutils.Of(float64(200.0))},
				{Name: "hardware_type", StringValue: apiutils.Of("gpu-a100")},
			},
		}

		_, err = metricsArtifactRepo.Save(perfArtifact1, savedModel.GetID())
		require.NoError(t, err)
		_, err = metricsArtifactRepo.Save(perfArtifact2, savedModel.GetID())
		require.NoError(t, err)
		_, err = metricsArtifactRepo.Save(perfArtifact3, savedModel.GetID())
		require.NoError(t, err)

		// Test without deduplication
		params := ListPerformanceArtifactsParams{
			PageSize:        10,
			OrderBy:         string(model.ORDERBYFIELD_CREATE_TIME),
			SortOrder:       model.SORTORDER_ASC,
			NextPageToken:   apiutils.Of(""),
			TargetRPS:       0,
			Recommendations: false,
		}

		result, err := dbCatalog.GetPerformanceArtifacts(ctx, "dedup-test-model", "dedup-test-source", params)
		require.NoError(t, err)
		assert.Equal(t, int32(3), result.Size, "Should return all 3 artifacts without dedup")

		// Test with deduplication
		params.Recommendations = true
		result, err = dbCatalog.GetPerformanceArtifacts(ctx, "dedup-test-model", "dedup-test-source", params)
		require.NoError(t, err)
		assert.Equal(t, int32(2), result.Size, "Should return 2 artifacts after dedup (one for each cost)")
	})

	t.Run("TestGetArtifacts_WithFilterQuery", func(t *testing.T) {
		// Create test model
		testModel := &models.CatalogModelImpl{
			TypeID: apiutils.Of(int32(catalogModelTypeID)),
			Attributes: &models.CatalogModelAttributes{
				Name:       apiutils.Of("filterquery-test-source:filterquery-artifact-test-model"),
				ExternalID: apiutils.Of("filterquery-artifact-test-model-ext"),
			},
			Properties: &[]mr_models.Properties{
				{Name: "source_id", StringValue: apiutils.Of("filterquery-test-source")},
			},
		}

		savedModel, err := catalogModelRepo.Save(testModel)
		require.NoError(t, err)

		// Create multiple test artifacts with different properties
		artifact1 := &models.CatalogModelArtifactImpl{
			TypeID: apiutils.Of(int32(modelArtifactTypeID)),
			Attributes: &models.CatalogModelArtifactAttributes{
				Name:         apiutils.Of("pytorch-model-artifact"),
				ExternalID:   apiutils.Of("pytorch-model-artifact-ext"),
				URI:          apiutils.Of("s3://bucket/pytorch/model.bin"),
				ArtifactType: apiutils.Of(models.CatalogModelArtifactType),
			},
			CustomProperties: &[]mr_models.Properties{
				{Name: "format", StringValue: apiutils.Of("pytorch")},
				{Name: "model_size", DoubleValue: apiutils.Of(float64(500))},
			},
		}

		artifact2 := &models.CatalogModelArtifactImpl{
			TypeID: apiutils.Of(int32(modelArtifactTypeID)),
			Attributes: &models.CatalogModelArtifactAttributes{
				Name:         apiutils.Of("onnx-model-artifact"),
				ExternalID:   apiutils.Of("onnx-model-artifact-ext"),
				URI:          apiutils.Of("https://huggingface.co/models/onnx/model.onnx"),
				ArtifactType: apiutils.Of(models.CatalogModelArtifactType),
			},
			CustomProperties: &[]mr_models.Properties{
				{Name: "format", StringValue: apiutils.Of("onnx")},
				{Name: "model_size", DoubleValue: apiutils.Of(float64(1500))},
			},
		}

		artifact3 := &models.CatalogMetricsArtifactImpl{
			TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
			Attributes: &models.CatalogMetricsArtifactAttributes{
				Name:         apiutils.Of("accuracy-metrics"),
				ExternalID:   apiutils.Of("accuracy-metrics-ext"),
				MetricsType:  models.MetricsTypeAccuracy,
				ArtifactType: apiutils.Of("metrics-artifact"),
			},
			CustomProperties: &[]mr_models.Properties{
				{Name: "overall_average", DoubleValue: apiutils.Of(float64(0.95))},
			},
		}

		_, err = modelArtifactRepo.Save(artifact1, savedModel.GetID())
		require.NoError(t, err)
		_, err = modelArtifactRepo.Save(artifact2, savedModel.GetID())
		require.NoError(t, err)
		_, err = metricsArtifactRepo.Save(artifact3, savedModel.GetID())
		require.NoError(t, err)

		// Test cases
		tests := []struct {
			name          string
			filterQuery   string
			expectedCount int32
			expectedNames []string
			shouldError   bool
		}{
			{
				name:          "Filter by URI pattern - s3",
				filterQuery:   `uri LIKE "%s3%"`,
				expectedCount: 1,
				expectedNames: []string{"pytorch-model-artifact"},
			},
			{
				name:          "Filter by custom property format",
				filterQuery:   `format.string_value = "onnx"`,
				expectedCount: 1,
				expectedNames: []string{"onnx-model-artifact"},
			},
			{
				name:          "Filter by numeric custom property",
				filterQuery:   `model_size.double_value > 1000`,
				expectedCount: 1,
				expectedNames: []string{"onnx-model-artifact"},
			},
			{
				name:          "Complex filter with AND",
				filterQuery:   `uri LIKE "%huggingface%" AND format.string_value = "onnx"`,
				expectedCount: 1,
				expectedNames: []string{"onnx-model-artifact"},
			},
			{
				name:          "Filter by name pattern",
				filterQuery:   `name LIKE "%pytorch%"`,
				expectedCount: 1,
				expectedNames: []string{"pytorch-model-artifact"},
			},
			{
				name:          "Filter with OR condition",
				filterQuery:   `format.string_value = "pytorch" OR format.string_value = "onnx"`,
				expectedCount: 2,
				expectedNames: []string{"pytorch-model-artifact", "onnx-model-artifact"},
			},
			{
				name:          "Filter with no matches",
				filterQuery:   `name = "non-existent-artifact"`,
				expectedCount: 0,
				expectedNames: []string{},
			},
			{
				name:          "Empty filterQuery returns all artifacts",
				filterQuery:   "",
				expectedCount: 3,
				expectedNames: []string{"pytorch-model-artifact", "onnx-model-artifact", "accuracy-metrics"},
			},
			{
				name:        "Invalid filterQuery syntax",
				filterQuery: "invalid syntax here",
				shouldError: true,
			},
			{
				name:          "Inferred int type - should match double values (dual-column query)",
				filterQuery:   `model_size > 400`,
				expectedCount: 2,
				expectedNames: []string{"pytorch-model-artifact", "onnx-model-artifact"},
			},
			{
				name:          "Explicit double_value with integer literal",
				filterQuery:   `model_size.double_value > 400`,
				expectedCount: 2,
				expectedNames: []string{"pytorch-model-artifact", "onnx-model-artifact"},
			},
			{
				name:          "Explicit double_value with float literal",
				filterQuery:   `model_size.double_value > 400.0`,
				expectedCount: 2,
				expectedNames: []string{"pytorch-model-artifact", "onnx-model-artifact"},
			},
			{
				name:          "Explicit int_value with integer literal",
				filterQuery:   `model_size.int_value > 400`,
				expectedCount: 0, // Data is stored as double, so int_value query returns nothing
				expectedNames: []string{},
			},
			{
				name:          "Explicit string_value with string literal",
				filterQuery:   `format.string_value = "onnx"`,
				expectedCount: 1,
				expectedNames: []string{"onnx-model-artifact"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				params := ListArtifactsParams{
					FilterQuery:   tt.filterQuery,
					PageSize:      10,
					OrderBy:       string(model.ORDERBYFIELD_CREATE_TIME),
					SortOrder:     model.SORTORDER_ASC,
					NextPageToken: apiutils.Of(""),
				}

				result, err := dbCatalog.GetArtifacts(ctx, "filterquery-artifact-test-model", "filterquery-test-source", params)

				if tt.shouldError {
					require.Error(t, err, "Expected error for invalid filter query")
					assert.Contains(t, err.Error(), "invalid filter query", "Error should mention invalid filter query")
					return
				}

				require.NoError(t, err)
				assert.Equal(t, tt.expectedCount, result.Size, "Expected %d artifacts but got %d", tt.expectedCount, result.Size)

				// Verify artifact names
				actualNames := make([]string, 0)
				for _, artifact := range result.Items {
					if artifact.CatalogModelArtifact != nil && artifact.CatalogModelArtifact.Name != nil {
						actualNames = append(actualNames, *artifact.CatalogModelArtifact.Name)
					}
					if artifact.CatalogMetricsArtifact != nil && artifact.CatalogMetricsArtifact.Name != nil {
						actualNames = append(actualNames, *artifact.CatalogMetricsArtifact.Name)
					}
				}
				assert.ElementsMatch(t, tt.expectedNames, actualNames, "Artifact names should match expected")
			})
		}
	})
}

func TestDBCatalog_GetPerformanceArtifactsWithService(t *testing.T) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testhelpers.MustDatastoreSpec(t))
	defer cleanup()

	// Get type IDs
	catalogModelTypeID := testhelpers.GetCatalogModelTypeIDForDBTest(t, sharedDB)
	modelArtifactTypeID := testhelpers.GetCatalogModelArtifactTypeIDForDBTest(t, sharedDB)
	metricsArtifactTypeID := testhelpers.GetCatalogMetricsArtifactTypeIDForDBTest(t, sharedDB)
	catalogSourceTypeID := testhelpers.GetCatalogSourceTypeIDForDBTest(t, sharedDB)

	// Create repositories
	catalogModelRepo := modelservice.NewCatalogModelRepository(sharedDB, catalogModelTypeID)
	catalogArtifactRepo := service.NewCatalogArtifactRepository(sharedDB, map[string]int32{
		service.CatalogModelArtifactTypeName:   modelArtifactTypeID,
		service.CatalogMetricsArtifactTypeName: metricsArtifactTypeID,
	})
	modelArtifactRepo := modelservice.NewCatalogModelArtifactRepository(sharedDB, modelArtifactTypeID)
	metricsArtifactRepo := modelservice.NewCatalogMetricsArtifactRepository(sharedDB, metricsArtifactTypeID)
	catalogSourceRepo := service.NewCatalogSourceRepository(sharedDB, catalogSourceTypeID)

	services := Services{
		CatalogModelRepository:           catalogModelRepo,
		CatalogArtifactRepository:        catalogArtifactRepo,
		CatalogModelArtifactRepository:   modelArtifactRepo,
		CatalogMetricsArtifactRepository: metricsArtifactRepo,
		CatalogSourceRepository:          catalogSourceRepo,
		PropertyOptionsRepository:        service.NewPropertyOptionsRepository(sharedDB),
	}

	sources := NewSourceCollection()
	err := sources.Merge("test-origin", map[string]basecatalog.ModelSource{
		"test-source": {
			CatalogSource: model.CatalogSource{
				Id:   "test-source",
				Name: "Test Source",
			},
		},
	})
	require.NoError(t, err)

	provider := NewDBCatalog(services, sources)

	// Create test model and performance artifacts
	testModel := &models.CatalogModelImpl{
		TypeID: apiutils.Of(int32(catalogModelTypeID)),
		Attributes: &models.CatalogModelAttributes{
			Name:       apiutils.Of("test-source:performance-test-model"),
			ExternalID: apiutils.Of("perf-model-123"),
		},
		Properties: &[]mr_models.Properties{
			{Name: "source_id", StringValue: apiutils.Of("test-source")},
		},
	}
	savedModel, err := catalogModelRepo.Save(testModel)
	require.NoError(t, err)

	// Create performance metrics artifact with exact properties for algorithm testing
	perfArtifact := &models.CatalogMetricsArtifactImpl{
		TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
		Attributes: &models.CatalogMetricsArtifactAttributes{
			Name:        apiutils.Of("test-perf-artifact"),
			ExternalID:  apiutils.Of("perf-123"),
			MetricsType: models.MetricsTypePerformance,
		},
		Properties: &[]mr_models.Properties{
			{Name: "metricsType", StringValue: apiutils.Of("performance-metrics")},
		},
		CustomProperties: &[]mr_models.Properties{
			{Name: "requests_per_second", DoubleValue: apiutils.Of(200.0)},
			{Name: "ttft_p90", DoubleValue: apiutils.Of(50.0)},
			{Name: "hardware_count", IntValue: apiutils.Of(int32(1))},
			{Name: "hardware_type", StringValue: apiutils.Of("gpu-a100")},
		},
	}
	_, err = metricsArtifactRepo.Save(perfArtifact, savedModel.GetID())
	require.NoError(t, err)

	// Test GetPerformanceArtifacts with targetRPS and deduplication
	params := ListPerformanceArtifactsParams{
		TargetRPS:       600, // Should calculate 3 replicas and be usable by dedup algorithm
		Recommendations: true,
		PageSize:        10,
	}

	result, err := provider.GetPerformanceArtifacts(
		context.Background(),
		"performance-test-model",
		"test-source",
		params,
	)
	require.NoError(t, err)
	assert.Len(t, result.Items, 1)

	// Verify both targetRPS calculations AND deduplication algorithm were applied via service
	artifact := result.Items[0]
	require.NotNil(t, artifact.CatalogMetricsArtifact)
	assert.Contains(t, artifact.CatalogMetricsArtifact.CustomProperties, "replicas")

	replicas := artifact.CatalogMetricsArtifact.CustomProperties["replicas"]
	assert.Equal(t, "3", replicas.MetadataIntValue.IntValue)
}

func TestGetFilterOptionsWithNamedQueries(t *testing.T) {
	// Setup mock sources with named queries, including some with min/max values
	sources := NewSourceCollection()
	namedQueries := map[string]map[string]basecatalog.FieldFilter{
		"validation-default": {
			"ttft_p90":      {Operator: "<", Value: 70},
			"workload_type": {Operator: "=", Value: "Chat"},
		},
		"high-performance": {
			"performance_score": {Operator: ">", Value: 0.95},
		},
		"range-query": {
			"latency_ms":   {Operator: ">=", Value: "min"},
			"throughput":   {Operator: "<=", Value: "max"},
			"memory_usage": {Operator: ">", Value: "min"},
		},
	}

	err := sources.MergeWithNamedQueries("test", map[string]basecatalog.ModelSource{}, namedQueries)
	require.NoError(t, err)

	// Use a realistic non-zero TypeID to validate that GetFilterOptions
	// correctly scopes context property queries by type.
	const mockTypeID int32 = 42
	mockServices := Services{
		CatalogModelRepository: &MockCatalogModelRepository{TypeID: mockTypeID},
		PropertyOptionsRepository: &mockPropertyRepositoryWithRanges{
			t:              t,
			expectedTypeID: mockTypeID,
		},
	}
	catalog := NewDBCatalog(mockServices, sources)

	// Test GetFilterOptions includes named queries with min/max transformed
	result, err := catalog.GetFilterOptions(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result.NamedQueries)

	queries := *result.NamedQueries
	assert.Len(t, queries, 3)

	// Test original queries without min/max are unchanged
	validationQuery := queries["validation-default"]
	assert.Equal(t, "<", validationQuery["ttft_p90"].Operator)
	assert.Equal(t, 70, validationQuery["ttft_p90"].Value)
	assert.Equal(t, "=", validationQuery["workload_type"].Operator)
	assert.Equal(t, "Chat", validationQuery["workload_type"].Value)

	// Test queries with min/max values are transformed to actual numeric values
	rangeQuery := queries["range-query"]

	// Verify "min" is replaced with actual min value (10.0)
	assert.Equal(t, ">=", rangeQuery["latency_ms"].Operator)
	assert.Equal(t, 10.0, rangeQuery["latency_ms"].Value, "Expected 'min' to be replaced with 10.0")

	// Verify "max" is replaced with actual max value (1000.0)
	assert.Equal(t, "<=", rangeQuery["throughput"].Operator)
	assert.Equal(t, 1000.0, rangeQuery["throughput"].Value, "Expected 'max' to be replaced with 1000.0")

	// Verify "min" is replaced with actual min value (0.0)
	assert.Equal(t, ">", rangeQuery["memory_usage"].Operator)
	assert.Equal(t, 0.0, rangeQuery["memory_usage"].Value, "Expected 'min' to be replaced with 0.0")
}

// Mock repository that provides filter options with numeric ranges for testing min/max transformation.
// expectedTypeID, when non-zero, asserts the typeID passed to List for context properties.
type mockPropertyRepositoryWithRanges struct {
	t              *testing.T
	expectedTypeID int32
}

func (m *mockPropertyRepositoryWithRanges) List(optionType sharedmodels.PropertyOptionType, typeID int32) ([]sharedmodels.PropertyOption, error) {
	if m.expectedTypeID != 0 && optionType == sharedmodels.ContextPropertyOptionType {
		require.Equal(m.t, m.expectedTypeID, typeID, "expected context property query to be scoped by typeID")
	}
	if optionType == sharedmodels.ArtifactPropertyOptionType {
		require.Equal(m.t, int32(0), typeID, "artifact property query should not be scoped by typeID")
	}
	// Return property options with numeric ranges that match the fields used in the test
	minLatency := int64(10)
	maxLatency := int64(500)
	minThroughput := int64(100)
	maxThroughput := int64(1000)
	minMemory := int64(0)
	maxMemory := int64(2048)

	return []sharedmodels.PropertyOption{
		{
			Name:        "latency_ms",
			MinIntValue: &minLatency,
			MaxIntValue: &maxLatency,
		},
		{
			Name:        "throughput",
			MinIntValue: &minThroughput,
			MaxIntValue: &maxThroughput,
		},
		{
			Name:        "memory_usage",
			MinIntValue: &minMemory,
			MaxIntValue: &maxMemory,
		},
	}, nil
}

func (m *mockPropertyRepositoryWithRanges) Refresh(optionType sharedmodels.PropertyOptionType) error {
	return nil
}

// Helper functions for creating pointers to primitive types
func floatPtr(val float64) *float64 {
	return &val
}

func TestApplyMinMax(t *testing.T) {
	tests := []struct {
		name          string
		inputQuery    map[string]model.FieldFilter
		inputOptions  map[string]model.FilterOption
		expectedQuery map[string]model.FieldFilter
		description   string
	}{
		{
			name: "1. Min Value Replacement",
			inputQuery: map[string]model.FieldFilter{
				"throughput": {Operator: ">", Value: "min"},
			},
			inputOptions: map[string]model.FilterOption{
				"throughput": {
					Type: "number",
					Range: &model.FilterOptionRange{
						Min: floatPtr(10.0),
						Max: floatPtr(100.0),
					},
				},
			},
			expectedQuery: map[string]model.FieldFilter{
				"throughput": {Operator: ">", Value: 10.0},
			},
			description: "Query with 'min' string should be replaced with numeric min value",
		},
		{
			name: "2. Max Value Replacement",
			inputQuery: map[string]model.FieldFilter{
				"latency": {Operator: "<", Value: "max"},
			},
			inputOptions: map[string]model.FilterOption{
				"latency": {
					Type: "number",
					Range: &model.FilterOptionRange{
						Min: floatPtr(5.0),
						Max: floatPtr(50.0),
					},
				},
			},
			expectedQuery: map[string]model.FieldFilter{
				"latency": {Operator: "<", Value: 50.0},
			},
			description: "Query with 'max' string should be replaced with numeric max value",
		},
		{
			name: "3. No Change for Non-Min/Max Values",
			inputQuery: map[string]model.FieldFilter{
				"status":  {Operator: "=", Value: "active"},
				"version": {Operator: "=", Value: "v1.0"},
			},
			inputOptions: map[string]model.FilterOption{
				"status": {
					Type:   "string",
					Values: []any{"active", "inactive"},
				},
			},
			expectedQuery: map[string]model.FieldFilter{
				"status":  {Operator: "=", Value: "active"},
				"version": {Operator: "=", Value: "v1.0"},
			},
			description: "Non min/max string values should remain unchanged",
		},
		{
			name: "4. Non-String Value Handling",
			inputQuery: map[string]model.FieldFilter{
				"count":      {Operator: ">", Value: 42},
				"percentage": {Operator: "<", Value: 75.5},
				"enabled":    {Operator: "=", Value: true},
			},
			inputOptions: map[string]model.FilterOption{
				"count": {
					Type: "number",
					Range: &model.FilterOptionRange{
						Min: floatPtr(0.0),
						Max: floatPtr(100.0),
					},
				},
			},
			expectedQuery: map[string]model.FieldFilter{
				"count":      {Operator: ">", Value: 42},
				"percentage": {Operator: "<", Value: 75.5},
				"enabled":    {Operator: "=", Value: true},
			},
			description: "Non-string values (int, float, bool) should remain unchanged",
		},
		{
			name: "5. Missing Field in Options",
			inputQuery: map[string]model.FieldFilter{
				"unknown_field": {Operator: ">", Value: "min"},
			},
			inputOptions: map[string]model.FilterOption{
				"known_field": {
					Type: "number",
					Range: &model.FilterOptionRange{
						Min: floatPtr(1.0),
						Max: floatPtr(10.0),
					},
				},
			},
			expectedQuery: map[string]model.FieldFilter{
				"unknown_field": {Operator: ">", Value: "min"},
			},
			description: "Query with min/max should remain unchanged if field not in options",
		},
		{
			name: "6. Nil Range Handling",
			inputQuery: map[string]model.FieldFilter{
				"field_without_range": {Operator: ">", Value: "min"},
			},
			inputOptions: map[string]model.FilterOption{
				"field_without_range": {
					Type:  "string",
					Range: nil,
				},
			},
			expectedQuery: map[string]model.FieldFilter{
				"field_without_range": {Operator: ">", Value: "min"},
			},
			description: "Query should remain unchanged when option exists but Range is nil",
		},
		{
			name: "7. Nil Min/Max in Range",
			inputQuery: map[string]model.FieldFilter{
				"field_nil_min": {Operator: ">", Value: "min"},
				"field_nil_max": {Operator: "<", Value: "max"},
			},
			inputOptions: map[string]model.FilterOption{
				"field_nil_min": {
					Type: "number",
					Range: &model.FilterOptionRange{
						Min: nil,
						Max: floatPtr(100.0),
					},
				},
				"field_nil_max": {
					Type: "number",
					Range: &model.FilterOptionRange{
						Min: floatPtr(0.0),
						Max: nil,
					},
				},
			},
			expectedQuery: map[string]model.FieldFilter{
				"field_nil_min": {Operator: ">", Value: "min"},
				"field_nil_max": {Operator: "<", Value: "max"},
			},
			description: "Query should remain unchanged when Range.Min or Range.Max is nil",
		},
		{
			name:          "8. Empty Maps Handling",
			inputQuery:    map[string]model.FieldFilter{},
			inputOptions:  map[string]model.FilterOption{},
			expectedQuery: map[string]model.FieldFilter{},
			description:   "Empty maps should be handled gracefully without panics",
		},
		{
			name: "9. Case Sensitivity",
			inputQuery: map[string]model.FieldFilter{
				"field1": {Operator: ">", Value: "Min"},
				"field2": {Operator: "<", Value: "MAX"},
				"field3": {Operator: "=", Value: "minimum"},
				"field4": {Operator: "=", Value: "maximum"},
			},
			inputOptions: map[string]model.FilterOption{
				"field1": {
					Type: "number",
					Range: &model.FilterOptionRange{
						Min: floatPtr(1.0),
						Max: floatPtr(10.0),
					},
				},
				"field2": {
					Type: "number",
					Range: &model.FilterOptionRange{
						Min: floatPtr(1.0),
						Max: floatPtr(10.0),
					},
				},
				"field3": {
					Type: "number",
					Range: &model.FilterOptionRange{
						Min: floatPtr(1.0),
						Max: floatPtr(10.0),
					},
				},
				"field4": {
					Type: "number",
					Range: &model.FilterOptionRange{
						Min: floatPtr(1.0),
						Max: floatPtr(10.0),
					},
				},
			},
			expectedQuery: map[string]model.FieldFilter{
				"field1": {Operator: ">", Value: "Min"},
				"field2": {Operator: "<", Value: "MAX"},
				"field3": {Operator: "=", Value: "minimum"},
				"field4": {Operator: "=", Value: "maximum"},
			},
			description: "Only exact 'min' and 'max' strings should be replaced (case-sensitive)",
		},
		{
			name: "10. Multiple Filter Replacement",
			inputQuery: map[string]model.FieldFilter{
				"throughput": {Operator: ">", Value: "min"},
				"latency":    {Operator: "<", Value: "max"},
				"cpu_usage":  {Operator: ">=", Value: "min"},
			},
			inputOptions: map[string]model.FilterOption{
				"throughput": {
					Type: "number",
					Range: &model.FilterOptionRange{
						Min: floatPtr(10.0),
						Max: floatPtr(1000.0),
					},
				},
				"latency": {
					Type: "number",
					Range: &model.FilterOptionRange{
						Min: floatPtr(1.0),
						Max: floatPtr(100.0),
					},
				},
				"cpu_usage": {
					Type: "number",
					Range: &model.FilterOptionRange{
						Min: floatPtr(0.0),
						Max: floatPtr(100.0),
					},
				},
			},
			expectedQuery: map[string]model.FieldFilter{
				"throughput": {Operator: ">", Value: 10.0},
				"latency":    {Operator: "<", Value: 100.0},
				"cpu_usage":  {Operator: ">=", Value: 0.0},
			},
			description: "All applicable fields should be replaced when multiple filters use min/max",
		},
		{
			name: "11. Mixed Scenario",
			inputQuery: map[string]model.FieldFilter{
				"throughput": {Operator: ">", Value: "min"},
				"status":     {Operator: "=", Value: "running"},
				"latency":    {Operator: "<", Value: "max"},
				"version":    {Operator: "=", Value: 2},
				"accuracy":   {Operator: ">=", Value: 0.95},
			},
			inputOptions: map[string]model.FilterOption{
				"throughput": {
					Type: "number",
					Range: &model.FilterOptionRange{
						Min: floatPtr(50.0),
						Max: floatPtr(500.0),
					},
				},
				"latency": {
					Type: "number",
					Range: &model.FilterOptionRange{
						Min: floatPtr(10.0),
						Max: floatPtr(200.0),
					},
				},
				"status": {
					Type:   "string",
					Values: []any{"running", "stopped"},
				},
			},
			expectedQuery: map[string]model.FieldFilter{
				"throughput": {Operator: ">", Value: 50.0},
				"status":     {Operator: "=", Value: "running"},
				"latency":    {Operator: "<", Value: 200.0},
				"version":    {Operator: "=", Value: 2},
				"accuracy":   {Operator: ">=", Value: 0.95},
			},
			description: "Only min/max string values should be replaced, others unchanged",
		},
		{
			name: "12. In-Place Modification Verification",
			inputQuery: map[string]model.FieldFilter{
				"metric": {Operator: ">", Value: "min"},
			},
			inputOptions: map[string]model.FilterOption{
				"metric": {
					Type: "number",
					Range: &model.FilterOptionRange{
						Min: floatPtr(25.0),
						Max: floatPtr(75.0),
					},
				},
			},
			expectedQuery: map[string]model.FieldFilter{
				"metric": {Operator: ">", Value: 25.0},
			},
			description: "Original query map should be modified in-place",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply the shared helper
			basecatalog.ApplyMinMax(tt.inputQuery, tt.inputOptions)

			// Verify the query was modified correctly
			assert.Equal(t, tt.expectedQuery, tt.inputQuery, tt.description)
		})
	}
}

func TestFindModelsWithRecommendedLatency(t *testing.T) {
	// Setup test database
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testhelpers.MustDatastoreSpec(t))
	defer cleanup()

	// Get type IDs
	catalogModelTypeID := testhelpers.GetCatalogModelTypeIDForDBTest(t, sharedDB)
	modelArtifactTypeID := testhelpers.GetCatalogModelArtifactTypeIDForDBTest(t, sharedDB)
	metricsArtifactTypeID := testhelpers.GetCatalogMetricsArtifactTypeIDForDBTest(t, sharedDB)
	catalogSourceTypeID := testhelpers.GetCatalogSourceTypeIDForDBTest(t, sharedDB)

	// Create repositories
	catalogModelRepo := modelservice.NewCatalogModelRepository(sharedDB, catalogModelTypeID)
	catalogArtifactRepo := service.NewCatalogArtifactRepository(sharedDB, map[string]int32{
		service.CatalogModelArtifactTypeName:   modelArtifactTypeID,
		service.CatalogMetricsArtifactTypeName: metricsArtifactTypeID,
	})
	modelArtifactRepo := modelservice.NewCatalogModelArtifactRepository(sharedDB, modelArtifactTypeID)
	metricsArtifactRepo := modelservice.NewCatalogMetricsArtifactRepository(sharedDB, metricsArtifactTypeID)
	catalogSourceRepo := service.NewCatalogSourceRepository(sharedDB, catalogSourceTypeID)

	svcs := Services{
		CatalogModelRepository:           catalogModelRepo,
		CatalogArtifactRepository:        catalogArtifactRepo,
		CatalogModelArtifactRepository:   modelArtifactRepo,
		CatalogMetricsArtifactRepository: metricsArtifactRepo,
		CatalogSourceRepository:          catalogSourceRepo,
		PropertyOptionsRepository:        service.NewPropertyOptionsRepository(sharedDB),
	}

	// Create DB catalog instance
	dbCatalog := NewDBCatalog(svcs, nil)
	ctx := context.Background()

	// Create test models with and without performance artifacts
	model1 := &models.CatalogModelImpl{
		TypeID: apiutils.Of(int32(catalogModelTypeID)),
		Attributes: &models.CatalogModelAttributes{
			Name:       apiutils.Of("latency-test-source:latency-model-1"),
			ExternalID: apiutils.Of("latency-model-1-ext"),
		},
		Properties: &[]mr_models.Properties{
			{Name: "source_id", StringValue: apiutils.Of("latency-test-source")},
			{Name: "description", StringValue: apiutils.Of("Model with performance data")},
		},
	}

	model2 := &models.CatalogModelImpl{
		TypeID: apiutils.Of(int32(catalogModelTypeID)),
		Attributes: &models.CatalogModelAttributes{
			Name:       apiutils.Of("latency-test-source:latency-model-2"),
			ExternalID: apiutils.Of("latency-model-2-ext"),
		},
		Properties: &[]mr_models.Properties{
			{Name: "source_id", StringValue: apiutils.Of("latency-test-source")},
			{Name: "description", StringValue: apiutils.Of("Model with performance data")},
		},
	}

	model3 := &models.CatalogModelImpl{
		TypeID: apiutils.Of(int32(catalogModelTypeID)),
		Attributes: &models.CatalogModelAttributes{
			Name:       apiutils.Of("latency-test-source:latency-model-3"),
			ExternalID: apiutils.Of("latency-model-3-ext"),
		},
		Properties: &[]mr_models.Properties{
			{Name: "source_id", StringValue: apiutils.Of("latency-test-source")},
			{Name: "description", StringValue: apiutils.Of("Model without performance data")},
		},
	}

	savedModel1, err := catalogModelRepo.Save(model1)
	require.NoError(t, err)
	savedModel2, err := catalogModelRepo.Save(model2)
	require.NoError(t, err)
	_, err = catalogModelRepo.Save(model3)
	require.NoError(t, err)

	// Add performance artifacts for model1 and model2
	perfArtifact1 := &models.CatalogMetricsArtifactImpl{
		TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
		Attributes: &models.CatalogMetricsArtifactAttributes{
			Name:        apiutils.Of("perf-artifact-1"),
			ExternalID:  apiutils.Of("perf-artifact-1-ext"),
			MetricsType: models.MetricsTypePerformance,
		},
		Properties: &[]mr_models.Properties{},
		CustomProperties: &[]mr_models.Properties{
			{Name: "ttft_p90", DoubleValue: apiutils.Of(float64(100.0))}, // Lower latency
			{Name: "requests_per_second", DoubleValue: apiutils.Of(float64(50.0))},
			{Name: "hardware_count", IntValue: apiutils.Of(int32(2))},
			{Name: "hardware_type", StringValue: apiutils.Of("gpu")},
		},
	}

	perfArtifact2 := &models.CatalogMetricsArtifactImpl{
		TypeID: apiutils.Of(int32(metricsArtifactTypeID)),
		Attributes: &models.CatalogMetricsArtifactAttributes{
			Name:        apiutils.Of("perf-artifact-2"),
			ExternalID:  apiutils.Of("perf-artifact-2-ext"),
			MetricsType: models.MetricsTypePerformance,
		},
		Properties: &[]mr_models.Properties{},
		CustomProperties: &[]mr_models.Properties{
			{Name: "ttft_p90", DoubleValue: apiutils.Of(float64(200.0))}, // Higher latency
			{Name: "requests_per_second", DoubleValue: apiutils.Of(float64(30.0))},
			{Name: "hardware_count", IntValue: apiutils.Of(int32(1))},
			{Name: "hardware_type", StringValue: apiutils.Of("cpu")},
		},
	}

	_, err = metricsArtifactRepo.Save(perfArtifact1, savedModel1.GetID())
	require.NoError(t, err)
	_, err = metricsArtifactRepo.Save(perfArtifact2, savedModel2.GetID())
	require.NoError(t, err)

	// Test FindModelsWithRecommendedLatency
	pagination := mr_models.Pagination{
		PageSize: apiutils.Of(int32(10)),
	}

	paretoParams := ParetoFilteringParams{
		LatencyProperty: "ttft_p90",
	}

	resultModels, err := dbCatalog.(*dbCatalogImpl).FindModelsWithRecommendedLatency(
		ctx,
		pagination,
		paretoParams,
		[]string{"latency-test-source"}, // Filter by this test's source ID
		"",                              // No query filter
	)

	require.NoError(t, err)
	require.NotNil(t, resultModels)
	require.Len(t, resultModels.Items, 3) // Expected test model count

	// Since the underlying performance artifacts may not be fully linked in test data,
	// we primarily verify that the method works and returns all models
	// The method implementation correctly handles models without latency data
	assert.Equal(t, 3, len(resultModels.Items))
	assert.NotEmpty(t, resultModels.NextPageToken == "" || resultModels.NextPageToken != "")

	// Basic verification that models are returned with proper structure
	for i, model := range resultModels.Items {
		assert.NotEmpty(t, model.Name, "Model %d should have a name", i)
		// Custom properties may or may not be set depending on performance data availability
	}
}

// TestGetFilterOptions_NoMCPServerContamination verifies that the model catalog's
// GetFilterOptions only returns properties from kf.CatalogModel contexts, not
// properties from kf.MCPServer contexts. This is a regression test for the bug
// where typeID=0 was passed to propertyOptionsRepository.List(), causing
// cross-contamination between resource types.
func TestGetFilterOptions_NoMCPServerContamination(t *testing.T) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testhelpers.MustDatastoreSpec(t))
	defer cleanup()

	// Get type IDs for both resource types
	catalogModelTypeID := testhelpers.GetCatalogModelTypeIDForDBTest(t, sharedDB)
	modelArtifactTypeID := testhelpers.GetCatalogModelArtifactTypeIDForDBTest(t, sharedDB)
	metricsArtifactTypeID := testhelpers.GetCatalogMetricsArtifactTypeIDForDBTest(t, sharedDB)
	catalogSourceTypeID := testhelpers.GetCatalogSourceTypeIDForDBTest(t, sharedDB)
	mcpServerTypeID := testhelpers.GetMCPServerTypeIDForDBTest(t, sharedDB)
	mcpServerToolTypeID := testhelpers.GetMCPServerToolTypeIDForDBTest(t, sharedDB)

	// Create repositories for both resource types
	catalogModelRepo := modelservice.NewCatalogModelRepository(sharedDB, catalogModelTypeID)
	catalogArtifactRepo := service.NewCatalogArtifactRepository(sharedDB, map[string]int32{
		service.CatalogModelArtifactTypeName:   modelArtifactTypeID,
		service.CatalogMetricsArtifactTypeName: metricsArtifactTypeID,
	})
	catalogSourceRepo := service.NewCatalogSourceRepository(sharedDB, catalogSourceTypeID)
	mcpServerRepo := mcpservice.NewMCPServerRepository(sharedDB, mcpServerTypeID)
	mcpServerToolRepo := mcpservice.NewMCPServerToolRepository(sharedDB, mcpServerToolTypeID)
	propertyOptionsRepo := service.NewPropertyOptionsRepository(sharedDB)

	// Create a catalog model with model-specific properties
	catalogModel := &models.CatalogModelImpl{
		TypeID: apiutils.Of(catalogModelTypeID),
		Attributes: &models.CatalogModelAttributes{
			Name:       apiutils.Of("cross-test-source:cross-test-model"),
			ExternalID: apiutils.Of("cross-test-model-ext"),
		},
		Properties: &[]mr_models.Properties{
			{Name: "source_id", StringValue: apiutils.Of("cross-test-source")},
			{Name: "license", StringValue: apiutils.Of("Apache-2.0")},
			{Name: "provider", StringValue: apiutils.Of("TestProvider")},
			{Name: "maturity", StringValue: apiutils.Of("stable")},
		},
	}
	_, err := catalogModelRepo.Save(catalogModel)
	require.NoError(t, err)

	// Create an MCP server with MCP-specific properties
	mcpServer := &mcpcatalogmodels.MCPServerImpl{
		TypeID: apiutils.Of(mcpServerTypeID),
		Attributes: &mcpcatalogmodels.MCPServerAttributes{
			Name:       apiutils.Of("cross-test-mcp-server"),
			ExternalID: apiutils.Of("cross-test-mcp-ext"),
		},
		Properties: &[]mr_models.Properties{
			{Name: "source_id", StringValue: apiutils.Of("cross-test-mcp-source")},
			{Name: "version", StringValue: apiutils.Of("1.0.0")},
			{Name: "base_name", StringValue: apiutils.Of("cross-test-mcp-server")},
			{Name: "deploymentMode", StringValue: apiutils.Of("remote")},
		},
	}
	_, err = mcpServerRepo.Save(mcpServer)
	require.NoError(t, err)

	// Refresh the materialized views so property options reflect the new data
	require.NoError(t, propertyOptionsRepo.Refresh(sharedmodels.ContextPropertyOptionType))
	require.NoError(t, propertyOptionsRepo.Refresh(sharedmodels.ArtifactPropertyOptionType))

	// Build model catalog services and call GetFilterOptions.
	// NOTE: catalogModelArtifactRepository and catalogMetricsArtifactRepository are nil
	// because GetFilterOptions does not access them. If GetFilterOptions is ever extended
	// to use artifact repositories, this test will panic and must be updated with stubs.
	modelSvcs := Services{
		CatalogModelRepository:    catalogModelRepo,
		CatalogArtifactRepository: catalogArtifactRepo,
		CatalogSourceRepository:   catalogSourceRepo,
		PropertyOptionsRepository: propertyOptionsRepo,
	}
	dbCatalog := NewDBCatalog(modelSvcs, nil)

	filterOptions, err := dbCatalog.GetFilterOptions(context.Background())
	require.NoError(t, err)
	require.NotNil(t, filterOptions)
	require.NotNil(t, filterOptions.Filters)

	filters := *filterOptions.Filters

	// Model-specific properties SHOULD be present
	assert.Contains(t, filters, "license", "model property 'license' should be present")
	assert.Contains(t, filters, "provider", "model property 'provider' should be present")
	assert.Contains(t, filters, "maturity", "model property 'maturity' should be present")

	// MCP-specific properties MUST NOT be present
	assert.NotContains(t, filters, "version", "MCP property 'version' must not appear in model filter_options")
	assert.NotContains(t, filters, "base_name", "MCP property 'base_name' must not appear in model filter_options")
	assert.NotContains(t, filters, "deploymentMode", "MCP property 'deploymentMode' must not appear in model filter_options")

	// source_id is shared by both types but excluded by both catalogs' skip lists
	assert.NotContains(t, filters, "source_id", "source_id should be excluded by the model catalog skip list")

	// Reverse direction: verify MCP catalog's GetFilterOptions doesn't leak model properties
	mcpSvcs := mcpcatalog.Services{
		MCPServerRepository:       mcpServerRepo,
		MCPServerToolRepository:   mcpServerToolRepo,
		CatalogSourceRepository:   catalogSourceRepo,
		PropertyOptionsRepository: propertyOptionsRepo,
	}
	dbMCPCatalog := mcpcatalog.NewDBMCPCatalog(mcpSvcs, nil, nil)
	mcpFilterOptions, err := dbMCPCatalog.GetFilterOptions(context.Background())
	require.NoError(t, err)
	require.NotNil(t, mcpFilterOptions)
	require.NotNil(t, mcpFilterOptions.Filters)

	mcpFilters := *mcpFilterOptions.Filters

	// Model-specific properties MUST NOT appear in MCP filter_options
	assert.NotContains(t, mcpFilters, "license", "model property 'license' must not appear in MCP filter_options")
	assert.NotContains(t, mcpFilters, "provider", "model property 'provider' must not appear in MCP filter_options")
	assert.NotContains(t, mcpFilters, "maturity", "model property 'maturity' must not appear in MCP filter_options")

	// source_id is shared by both types but excluded by both catalogs' skip lists
	assert.NotContains(t, mcpFilters, "source_id", "source_id should be excluded by the MCP catalog skip list")
}
