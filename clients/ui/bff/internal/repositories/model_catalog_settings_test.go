package repositories

import (
	"context"
	"strings"

	k8s "github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/mocks"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

var _ = Describe("ModelCatalogSettingRepository", func() {
	var (
		repo      *ModelCatalogSettingsRepository
		k8sClient k8s.KubernetesClientInterface
		ctx       context.Context
	)

	BeforeEach(func() {
		repo = NewModelCatalogSettingsRepository()
		var err error
		k8sClient, err = kubernetesMockedStaticClientFactory.GetClient(mocks.NewMockSessionContextNoParent())
		Expect(err).NotTo(HaveOccurred())
		ctx = mocks.NewMockSessionContextNoParent()
	})

	Describe("GetAllCatalogSourceConfigs", func() {
		It("should return merged source config from both default and user managed configMap", func() {
			catalogs, err := repo.GetAllCatalogSourceConfigs(ctx, k8sClient, "kubeflow")
			Expect(err).NotTo(HaveOccurred())
			Expect(catalogs.Catalogs).NotTo(BeEmpty())
		})

		It("default catalog should have isDefault=true", func() {
			catalogs, err := repo.GetAllCatalogSourceConfigs(ctx, k8sClient, "kubeflow")
			Expect(err).NotTo(HaveOccurred())
			var defaultCatalog *models.CatalogSourceConfig
			for _, c := range catalogs.Catalogs {
				if c.Id == "dora_ai_models" {
					defaultCatalog = &c
					break
				}
			}
			Expect(defaultCatalog).NotTo(BeNil())
			Expect(*defaultCatalog.IsDefault).To(BeTrue())
		})

		It("should merge, if user overrides the default source", func() {
			catalogs, err := repo.GetAllCatalogSourceConfigs(ctx, k8sClient, "kubeflow")
			Expect(err).NotTo(HaveOccurred())
			var mergedCatalog *models.CatalogSourceConfig
			for _, c := range catalogs.Catalogs {
				if c.Id == "dora_ai_models" {
					mergedCatalog = &c
					break
				}
			}

			Expect(mergedCatalog).NotTo(BeNil())
		})
	})

	Describe("GetCatalogSourceConfig", func() {
		It("should return Yaml content and YamlCatalogPath for yaml type source config", func() {
			catalog, err := repo.GetCatalogSourceConfig(ctx, k8sClient, "kubeflow", "dora_ai_models")
			Expect(err).NotTo(HaveOccurred())
			Expect(catalog.Type).To(Equal("yaml"))
			Expect(catalog.Yaml).NotTo(BeNil())
			Expect(*catalog.Yaml).NotTo(BeEmpty())
			Expect(catalog.YamlCatalogPath).NotTo(BeNil())
			Expect(*catalog.YamlCatalogPath).To(Equal("dora_ai_models.yaml"))
		})

		It("should return apiKey for the huggingFace type source config", func() {
			catalog, err := repo.GetCatalogSourceConfig(ctx, k8sClient, "kubeflow", "hugging_face_source")
			Expect(err).NotTo(HaveOccurred())
			Expect(catalog.Type).To(Equal("hf"))
			Expect(catalog.ApiKey).To(BeNil())
		})

		It("should return merged data for user override default source config", func() {
			catalog, err := repo.GetCatalogSourceConfig(ctx, k8sClient, "kubeflow", "dora_ai_models")
			Expect(err).NotTo(HaveOccurred())

			Expect(catalog.Name).To(Equal("Dora AI"))
			Expect(*catalog.IsDefault).To(BeTrue())
		})

		It("should return error if the source config doesn't exist", func() {
			_, err := repo.GetCatalogSourceConfig(ctx, k8sClient, "kubeflow", "does_not_exist")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("should return isDefault=false for user managed source", func() {
			catalog, err := repo.GetCatalogSourceConfig(ctx, k8sClient, "kubeflow", "custom_yaml_models")
			Expect(err).NotTo(HaveOccurred())
			Expect(*catalog.IsDefault).To(BeFalse())
		})

		It("should return allowedOrganization for huggingface source", func() {
			catalog, err := repo.GetCatalogSourceConfig(ctx, k8sClient, "kubeflow", "hugging_face_source")
			Expect(err).NotTo(HaveOccurred())
			Expect(catalog.AllowedOrganization).NotTo(BeNil())
		})
	})

	Describe("CreateCatalogSourceConfig", func() {
		It("should fail when id is missing", func() {
			payload := models.CatalogSourceConfigPayload{
				Name:    "Test",
				Type:    "yaml",
				Enabled: boolPtr(true),
				Yaml:    stringPtr("models: []"),
			}
			_, err := repo.CreateCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("catalog source ID is required"))
		})

		It("should fail when name is missing", func() {
			payload := models.CatalogSourceConfigPayload{
				Id:      "test",
				Type:    "yaml",
				Enabled: boolPtr(true),
				Yaml:    stringPtr("models: []"),
			}
			_, err := repo.CreateCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("name is required"))
		})

		It("should fail when type is missing", func() {
			payload := models.CatalogSourceConfigPayload{
				Id:      "test_id",
				Name:    "Test",
				Enabled: boolPtr(true),
				Yaml:    stringPtr("models: []"),
			}
			_, err := repo.CreateCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("type is required"))
		})

		It("should fail when yaml is missing for yaml type source config", func() {
			payload := models.CatalogSourceConfigPayload{
				Id:      "test_id",
				Name:    "Test",
				Type:    "yaml",
				Enabled: boolPtr(true),
			}
			_, err := repo.CreateCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("yaml field is required"))
		})

		It("should fail when allowedOrganization is missing for huggingface-type", func() {
			payload := models.CatalogSourceConfigPayload{
				Id:      "test_id",
				Name:    "Test",
				Type:    "hf",
				Enabled: boolPtr(true),
			}
			_, err := repo.CreateCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("allowedOrganization is required"))
		})

		It("should fail for unsupported type", func() {
			payload := models.CatalogSourceConfigPayload{
				Id:      "test_id",
				Name:    "Test",
				Type:    "invalid-type",
				Enabled: boolPtr(true),
			}
			_, err := repo.CreateCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported catalog type"))
		})

		It("should fail when id already exists in user sources", func() {
			payload := models.CatalogSourceConfigPayload{
				Id:      "custom_yaml_models",
				Name:    "Duplicate",
				Type:    "yaml",
				Enabled: boolPtr(true),
				Yaml:    stringPtr("models: []"),
			}
			_, err := repo.CreateCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists"))
		})

		It("should fail when id matches a default source", func() {
			payload := models.CatalogSourceConfigPayload{
				Id:      "dora_ai_models",
				Name:    "Duplicate Default",
				Type:    "yaml",
				Enabled: boolPtr(true),
				Yaml:    stringPtr("models: []"),
			}
			_, err := repo.CreateCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists in default"))
		})

		It("should fail when secret name exceeds maximum length", func() {
			longId := strings.Repeat("a", 239)

			payload := models.CatalogSourceConfigPayload{
				Id:                  longId,
				Name:                "Test Long ID",
				Type:                "hf",
				Enabled:             boolPtr(true),
				ApiKey:              stringPtr("hf_test_key"),
				AllowedOrganization: stringPtr("test-org"),
			}
			_, err := repo.CreateCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("exceeds maximum length"))
		})

		It("should succeed when catalog ID is at maximum allowed length", func() {
			maxLengthId := strings.Repeat("b", 238)

			payload := models.CatalogSourceConfigPayload{
				Id:                  maxLengthId,
				Name:                "Test Max Length ID",
				Type:                "hf",
				Enabled:             boolPtr(true),
				ApiKey:              stringPtr("hf_test_key"),
				AllowedOrganization: stringPtr("test-org"),
			}
			result, err := repo.CreateCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Id).To(Equal(maxLengthId))
		})

		It("should create yaml type source successfully", func() {
			payload := models.CatalogSourceConfigPayload{
				Id:      "test_yaml_create",
				Name:    "Test YAML Create",
				Type:    "yaml",
				Enabled: boolPtr(true),
				Yaml:    stringPtr("models:\n  - name: test"),
				Labels:  []string{"test"},
			}
			result, err := repo.CreateCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Id).To(Equal("test_yaml_create"))
			Expect(*result.IsDefault).To(BeFalse())
		})

		It("should create huggingface-type source successfully", func() {
			payload := models.CatalogSourceConfigPayload{
				Id:                  "test_hf_create",
				Name:                "Test HF Create",
				Type:                "hf",
				Enabled:             boolPtr(true),
				ApiKey:              stringPtr("hf_test_key"),
				AllowedOrganization: stringPtr("test-org"),
			}
			result, err := repo.CreateCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Id).To(Equal("test_hf_create"))
			Expect(*result.IsDefault).To(BeFalse())
		})

		It("should reject catalog ID with path traversal attempt", func() {
			payload := models.CatalogSourceConfigPayload{
				Id:   "../../../etc/passwd",
				Name: "Malicious",
				Type: "yaml",
				Yaml: stringPtr("models:\n  - name: test"),
			}
			_, err := repo.CreateCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid catalog ID"))
		})

		It("should reject catalog ID with forward slash", func() {
			payload := models.CatalogSourceConfigPayload{
				Id:   "test/malicious",
				Name: "Malicious",
				Type: "yaml",
				Yaml: stringPtr("models:\n  - name: test"),
			}
			_, err := repo.CreateCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid catalog ID"))
		})

		It("should reject catalog ID with special characters", func() {
			payload := models.CatalogSourceConfigPayload{
				Id:   "test@#$%",
				Name: "Special",
				Type: "yaml",
				Yaml: stringPtr("models:\n  - name: test"),
			}
			_, err := repo.CreateCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid catalog ID"))
		})

		It("should create source with includedModels and excludedModels", func() {
			payload := models.CatalogSourceConfigPayload{
				Id:             "test_with_filters",
				Name:           "Test With Filters",
				Type:           "yaml",
				Enabled:        boolPtr(true),
				Yaml:           stringPtr("models: []"),
				IncludedModels: []string{"model-*"},
				ExcludedModels: []string{"test-*"},
			}
			result, err := repo.CreateCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IncludedModels).To(ConsistOf("model-*"))
			Expect(result.ExcludedModels).To(ConsistOf("test-*"))
		})
	})

	Describe("UpdateCatalogSourceConfig", func() {
		It("should fail if source does not exist", func() {
			payload := models.CatalogSourceConfigPayload{
				Enabled: boolPtr(false),
			}
			_, err := repo.UpdateCatalogSourceConfig(ctx, k8sClient, "kubeflow", "does-not-exist", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("should fail, when trying to update the source type", func() {
			payload := models.CatalogSourceConfigPayload{
				Type: "hf",
			}
			_, err := repo.UpdateCatalogSourceConfig(ctx, k8sClient, "kubeflow", "custom_yaml_models", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot change"))
		})

		It("should fail when updating the name of the default source config", func() {
			payload := models.CatalogSourceConfigPayload{
				Name: "Changed Name",
			}
			_, err := repo.UpdateCatalogSourceConfig(ctx, k8sClient, "kubeflow", "dora_ai_models", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot change"))
		})

		It("should fail when updating the labels of default source config", func() {
			payload := models.CatalogSourceConfigPayload{
				Labels: []string{"new-label"},
			}
			_, err := repo.UpdateCatalogSourceConfig(ctx, k8sClient, "kubeflow", "dora_ai_models", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot change"))
		})

		It("should fail when updating the yaml of the default source config", func() {
			payload := models.CatalogSourceConfigPayload{
				Yaml: stringPtr("new content"),
			}
			_, err := repo.UpdateCatalogSourceConfig(ctx, k8sClient, "kubeflow", "dora_ai_models", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot change"))
		})

		It("should update the user managed source config successfully", func() {
			payload := models.CatalogSourceConfigPayload{
				Name:    "Updated Name",
				Enabled: boolPtr(false),
			}
			result, err := repo.UpdateCatalogSourceConfig(ctx, k8sClient, "kubeflow", "custom_yaml_models", payload)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Name).To(Equal("Updated Name"))
		})

		It("should override the enabled property of default source config in user managed configMap", func() {
			payload := models.CatalogSourceConfigPayload{
				Enabled: boolPtr(false),
			}
			result, err := repo.UpdateCatalogSourceConfig(ctx, k8sClient, "kubeflow", "dora_ai_models", payload)
			Expect(err).NotTo(HaveOccurred())
			Expect(*result.Enabled).To(BeFalse())
		})

		It("should allow updating includedModels on default catalog", func() {
			payload := models.CatalogSourceConfigPayload{
				IncludedModels: []string{"model-a", "model-b"},
			}
			result, err := repo.UpdateCatalogSourceConfig(ctx, k8sClient, "kubeflow", "dora_ai_models", payload)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IncludedModels).To(ConsistOf("model-a", "model-b"))
		})

		It("should allow updating excludedModels on default catalog", func() {
			payload := models.CatalogSourceConfigPayload{
				ExcludedModels: []string{"blocked-*"},
			}
			result, err := repo.UpdateCatalogSourceConfig(ctx, k8sClient, "kubeflow", "dora_ai_models", payload)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ExcludedModels).To(ConsistOf("blocked-*"))
		})

		It("should allow updating default twice", func() {
			// First update
			payload1 := models.CatalogSourceConfigPayload{
				Enabled: boolPtr(false),
			}
			_, err := repo.UpdateCatalogSourceConfig(ctx, k8sClient, "kubeflow", "bella_ai_validated_models", payload1)
			Expect(err).NotTo(HaveOccurred())

			// Second update
			payload2 := models.CatalogSourceConfigPayload{
				Enabled:        boolPtr(true),
				IncludedModels: []string{"updated-model"},
			}
			result, err := repo.UpdateCatalogSourceConfig(ctx, k8sClient, "kubeflow", "bella_ai_validated_models", payload2)
			Expect(err).NotTo(HaveOccurred())
			Expect(*result.Enabled).To(BeTrue())
			Expect(result.IncludedModels).To(ConsistOf("updated-model"))
		})

		It("should clear includedModels when empty array is provided", func() {
			// First create with includedModels
			initialPayload := models.CatalogSourceConfigPayload{
				Id:             "test_clear_models",
				Name:           "Test Clear Models",
				Type:           "yaml",
				IncludedModels: []string{"model-*"},
				ExcludedModels: []string{"old-*"},
				Yaml:           stringPtr("new content"),
			}
			_, err := repo.CreateCatalogSourceConfig(ctx, k8sClient, "kubeflow", initialPayload)
			Expect(err).NotTo(HaveOccurred())

			// Update with empty includedModels
			emptyModels := []string{}
			updatePayload := models.CatalogSourceConfigPayload{
				IncludedModels: emptyModels,
				ExcludedModels: []string{"*"},
			}

			result, err := repo.UpdateCatalogSourceConfig(ctx, k8sClient, "kubeflow", "test_clear_models", updatePayload)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IncludedModels).To(BeEmpty())
			Expect(result.ExcludedModels).To(Equal([]string{"*"}))
		})

		It("should clear excluded models when empty array is provided", func() {
			// First create with includedModels
			initialPayload := models.CatalogSourceConfigPayload{
				Id:             "test_clear_excluded_models",
				Name:           "Test Clear Excluded Models",
				Type:           "yaml",
				IncludedModels: []string{"model-*"},
				ExcludedModels: []string{"old-*"},
				Yaml:           stringPtr("new content"),
			}
			_, err := repo.CreateCatalogSourceConfig(ctx, k8sClient, "kubeflow", initialPayload)
			Expect(err).NotTo(HaveOccurred())

			// Update with empty excludedModels
			emptyModels := []string{}
			updatePayload := models.CatalogSourceConfigPayload{
				IncludedModels: []string{"*"},
				ExcludedModels: emptyModels,
			}

			result, err := repo.UpdateCatalogSourceConfig(ctx, k8sClient, "kubeflow", "test_clear_excluded_models", updatePayload)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ExcludedModels).To(BeEmpty())
			Expect(result.IncludedModels).To(Equal([]string{"*"}))
		})

		It("should return conflict error when concurrent updates occur with stale resourceVersion", func() {
			// First, create a source to test with
			createPayload := models.CatalogSourceConfigPayload{
				Id:      "concurrency_test_source",
				Name:    "Concurrency Test Source",
				Type:    "yaml",
				Enabled: boolPtr(true),
				Yaml:    stringPtr("models: []"),
			}
			_, err := repo.CreateCatalogSourceConfig(ctx, k8sClient, "kubeflow", createPayload)
			Expect(err).NotTo(HaveOccurred())

			// Simulate two concurrent requests by reading the configmap state twice
			// Request A reads the current state
			_, cmA, err := k8sClient.GetAllCatalogSourceConfigs(ctx, "kubeflow")
			Expect(err).NotTo(HaveOccurred())
			resourceVersionA := cmA.ResourceVersion
			Expect(resourceVersionA).NotTo(BeEmpty(), "ConfigMap should have a resourceVersion")

			// Request B reads the same state (before A updates)
			_, cmB, err := k8sClient.GetAllCatalogSourceConfigs(ctx, "kubeflow")
			Expect(err).NotTo(HaveOccurred())
			resourceVersionB := cmB.ResourceVersion
			Expect(resourceVersionB).To(Equal(resourceVersionA), "Both requests should see the same resourceVersion")

			// Request A updates successfully
			payloadA := models.CatalogSourceConfigPayload{
				Name: "Updated by Request A",
			}
			_, err = repo.UpdateCatalogSourceConfig(ctx, k8sClient, "kubeflow", "concurrency_test_source", payloadA)
			Expect(err).NotTo(HaveOccurred())

			// Verify the resourceVersion has changed after A's update
			_, cmAfterA, err := k8sClient.GetAllCatalogSourceConfigs(ctx, "kubeflow")
			Expect(err).NotTo(HaveOccurred())
			Expect(cmAfterA.ResourceVersion).NotTo(Equal(resourceVersionA), "resourceVersion should change after update")

			// Request B tries to update with stale resourceVersion
			// This simulates B modifying its local copy (cmB) and trying to update
			cmB.Data[k8s.CatalogSourceKey] = "catalogs:\n  - id: concurrency_test_source\n    name: Updated by Request B\n    type: yaml\n    enabled: true"
			err = k8sClient.UpdateCatalogSourceConfig(ctx, "kubeflow", &cmB)

			// Should fail with a conflict error because cmB has stale resourceVersion
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsConflict(err)).To(BeTrue(), "Expected conflict error due to stale resourceVersion, got: %v", err)
		})
	})

	Describe("DeleteCatalogSourceConfig", func() {
		It("should fail when deleting default catalog", func() {
			_, err := repo.DeleteCatalogSourceConfig(ctx, k8sClient, "kubeflow", "dora_ai_models")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot delete the default source: 'dora_ai_models' is a default source"))
		})

		It("should fail when the source config does not exist in catalog", func() {
			_, err := repo.DeleteCatalogSourceConfig(ctx, k8sClient, "kubeflow", "totally-fake-id")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("should delete user yaml source successfully", func() {
			// First create
			createPayload := models.CatalogSourceConfigPayload{
				Id:      "delete_test_yaml",
				Name:    "Delete Test",
				Type:    "yaml",
				Enabled: boolPtr(true),
				Yaml:    stringPtr("models: []"),
			}
			_, err := repo.CreateCatalogSourceConfig(ctx, k8sClient, "kubeflow", createPayload)
			Expect(err).NotTo(HaveOccurred())

			deleted, err := repo.DeleteCatalogSourceConfig(ctx, k8sClient, "kubeflow", "delete_test_yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(deleted.Id).To(Equal("delete_test_yaml"))
		})

		It("should delete user huggingface source and its secret", func() {
			// First create
			createPayload := models.CatalogSourceConfigPayload{
				Id:                  "delete_test_hf",
				Name:                "Delete HF Test",
				Type:                "hf",
				Enabled:             boolPtr(true),
				ApiKey:              stringPtr("hf_delete_test"),
				AllowedOrganization: stringPtr("test-org"),
			}
			_, err := repo.CreateCatalogSourceConfig(ctx, k8sClient, "kubeflow", createPayload)
			Expect(err).NotTo(HaveOccurred())

			deleted, err := repo.DeleteCatalogSourceConfig(ctx, k8sClient, "kubeflow", "delete_test_hf")
			Expect(err).NotTo(HaveOccurred())
			Expect(deleted.Id).To(Equal("delete_test_hf"))
		})
	})
})

var _ = Describe("validateCatalogId", func() {
	It("accepts an id with only lowercase letters, numbers, and underscores", func() {
		Expect(validateCatalogId("my_source_1")).NotTo(HaveOccurred())
	})

	It("rejects a hyphenated id (model/mcp catalogs never accepted hyphens)", func() {
		err := validateCatalogId("my-source")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("lowercase letters, numbers, and underscores"))
	})
})

func boolPtr(b bool) *bool {
	return &b
}

func stringPtr(s string) *string {
	return &s
}
