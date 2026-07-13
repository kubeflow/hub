package repositories

import (
	"context"

	k8s "github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/mocks"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

var _ = Describe("McpCatalogSettingsRepository", func() {
	var (
		repo      *McpCatalogSettingsRepository
		k8sClient k8s.KubernetesClientInterface
		ctx       context.Context
	)

	BeforeEach(func() {
		repo = NewMcpCatalogSettingsRepository()
		var err error
		k8sClient, err = kubernetesMockedStaticClientFactory.GetClient(mocks.NewMockSessionContextNoParent())
		Expect(err).NotTo(HaveOccurred())
		ctx = mocks.NewMockSessionContextNoParent()
	})

	Describe("GetAllMcpCatalogSourceConfigs", func() {
		It("should return merged source config from both default and user managed configMap", func() {
			catalogs, err := repo.GetAllMcpCatalogSourceConfigs(ctx, k8sClient, "kubeflow")
			Expect(err).NotTo(HaveOccurred())
			Expect(catalogs.Catalogs).NotTo(BeEmpty())
		})

		It("default catalog should have isDefault=true", func() {
			catalogs, err := repo.GetAllMcpCatalogSourceConfigs(ctx, k8sClient, "kubeflow")
			Expect(err).NotTo(HaveOccurred())
			var defaultCatalog *models.McpCatalogSourceConfig
			for _, c := range catalogs.Catalogs {
				if c.Id == "community_mcp_servers" {
					defaultCatalog = &c
					break
				}
			}
			Expect(defaultCatalog).NotTo(BeNil())
			Expect(*defaultCatalog.IsDefault).To(BeTrue())
		})

		It("should merge, if user overrides the default source", func() {
			payload := models.McpCatalogSourceConfigPayload{
				Enabled: mcpBoolPtr(false),
			}
			_, err := repo.UpdateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", "community_mcp_servers", payload)
			Expect(err).NotTo(HaveOccurred())

			catalogs, err := repo.GetAllMcpCatalogSourceConfigs(ctx, k8sClient, "kubeflow")
			Expect(err).NotTo(HaveOccurred())
			var mergedCatalog *models.McpCatalogSourceConfig
			for _, c := range catalogs.Catalogs {
				if c.Id == "community_mcp_servers" {
					mergedCatalog = &c
					break
				}
			}
			Expect(mergedCatalog).NotTo(BeNil())
			Expect(*mergedCatalog.Enabled).To(BeFalse())
			Expect(mergedCatalog.Name).To(Equal("Community MCP Servers"))
		})
	})

	Describe("GetMcpCatalogSourceConfig", func() {
		It("should return Yaml content and YamlCatalogPath for yaml type source config", func() {
			catalog, err := repo.GetMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", "community_mcp_servers")
			Expect(err).NotTo(HaveOccurred())
			Expect(catalog.Type).To(Equal("yaml"))
			Expect(catalog.Yaml).NotTo(BeNil())
			Expect(*catalog.Yaml).NotTo(BeEmpty())
			Expect(catalog.YamlCatalogPath).NotTo(BeNil())
			Expect(*catalog.YamlCatalogPath).To(Equal("community_mcp_servers.yaml"))
		})

		It("should return error if the source config doesn't exist", func() {
			_, err := repo.GetMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", "does_not_exist")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("should return isDefault=false for user managed source", func() {
			catalog, err := repo.GetMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", "custom_mcp_servers")
			Expect(err).NotTo(HaveOccurred())
			Expect(*catalog.IsDefault).To(BeFalse())
		})

		It("should return merged data for user override default source config", func() {
			payload := models.McpCatalogSourceConfigPayload{
				Enabled: mcpBoolPtr(false),
			}
			_, err := repo.UpdateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", "community_mcp_servers", payload)
			Expect(err).NotTo(HaveOccurred())

			catalog, err := repo.GetMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", "community_mcp_servers")
			Expect(err).NotTo(HaveOccurred())
			Expect(catalog.Name).To(Equal("Community MCP Servers"))
			Expect(*catalog.IsDefault).To(BeTrue())
			Expect(*catalog.Enabled).To(BeFalse())
		})
	})

	Describe("CreateMcpCatalogSourceConfig", func() {
		It("should fail when id is missing", func() {
			payload := models.McpCatalogSourceConfigPayload{
				Name:    "Test",
				Type:    "yaml",
				Enabled: mcpBoolPtr(true),
				Yaml:    mcpStringPtr("servers: []"),
			}
			_, err := repo.CreateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mcp catalog source ID is required"))
		})

		It("should fail when name is missing", func() {
			payload := models.McpCatalogSourceConfigPayload{
				Id:      "test",
				Type:    "yaml",
				Enabled: mcpBoolPtr(true),
				Yaml:    mcpStringPtr("servers: []"),
			}
			_, err := repo.CreateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("name is required"))
		})

		It("should fail when type is missing", func() {
			payload := models.McpCatalogSourceConfigPayload{
				Id:      "test_id",
				Name:    "Test",
				Enabled: mcpBoolPtr(true),
				Yaml:    mcpStringPtr("servers: []"),
			}
			_, err := repo.CreateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("type is required"))
		})

		It("should fail when yaml is missing for yaml type source config", func() {
			payload := models.McpCatalogSourceConfigPayload{
				Id:      "test_id",
				Name:    "Test",
				Type:    "yaml",
				Enabled: mcpBoolPtr(true),
			}
			_, err := repo.CreateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("yaml field is required"))
		})

		It("should fail for unsupported type", func() {
			payload := models.McpCatalogSourceConfigPayload{
				Id:      "test_id",
				Name:    "Test",
				Type:    "invalid-type",
				Enabled: mcpBoolPtr(true),
				Yaml:    mcpStringPtr("servers: []"),
			}
			_, err := repo.CreateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported MCP catalog type"))
		})

		It("should fail when id already exists in user sources", func() {
			payload := models.McpCatalogSourceConfigPayload{
				Id:      "custom_mcp_servers",
				Name:    "Duplicate",
				Type:    "yaml",
				Enabled: mcpBoolPtr(true),
				Yaml:    mcpStringPtr("servers: []"),
			}
			_, err := repo.CreateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists"))
		})

		It("should fail when id matches a default source", func() {
			payload := models.McpCatalogSourceConfigPayload{
				Id:      "community_mcp_servers",
				Name:    "Duplicate Default",
				Type:    "yaml",
				Enabled: mcpBoolPtr(true),
				Yaml:    mcpStringPtr("servers: []"),
			}
			_, err := repo.CreateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists in default"))
		})

		It("should create yaml type source successfully", func() {
			payload := models.McpCatalogSourceConfigPayload{
				Id:      "test_mcp_yaml_create",
				Name:    "Test MCP YAML Create",
				Type:    "yaml",
				Enabled: mcpBoolPtr(true),
				Yaml:    mcpStringPtr("servers:\n  - name: test"),
				Labels:  []string{"test"},
			}
			result, err := repo.CreateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Id).To(Equal("test_mcp_yaml_create"))
			Expect(*result.IsDefault).To(BeFalse())
		})

		It("should create source with includedServers and excludedServers", func() {
			payload := models.McpCatalogSourceConfigPayload{
				Id:              "test_mcp_with_filters",
				Name:            "Test MCP With Filters",
				Type:            "yaml",
				Enabled:         mcpBoolPtr(true),
				Yaml:            mcpStringPtr("servers: []"),
				IncludedServers: []string{"server-*"},
				ExcludedServers: []string{"test-*"},
			}
			result, err := repo.CreateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IncludedServers).To(ConsistOf("server-*"))
			Expect(result.ExcludedServers).To(ConsistOf("test-*"))
		})

		It("should reject catalog ID with special characters", func() {
			payload := models.McpCatalogSourceConfigPayload{
				Id:   "test@#$%",
				Name: "Special",
				Type: "yaml",
				Yaml: mcpStringPtr("servers: []"),
			}
			_, err := repo.CreateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid catalog ID"))
		})

		It("should reject catalog ID with path traversal attempt", func() {
			payload := models.McpCatalogSourceConfigPayload{
				Id:   "../../../etc/passwd",
				Name: "Malicious",
				Type: "yaml",
				Yaml: mcpStringPtr("servers: []"),
			}
			_, err := repo.CreateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid catalog ID"))
		})

		It("should reject catalog ID with forward slash", func() {
			payload := models.McpCatalogSourceConfigPayload{
				Id:   "test/malicious",
				Name: "Malicious",
				Type: "yaml",
				Yaml: mcpStringPtr("servers: []"),
			}
			_, err := repo.CreateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid catalog ID"))
		})
	})

	Describe("UpdateMcpCatalogSourceConfig", func() {
		It("should fail if source does not exist", func() {
			payload := models.McpCatalogSourceConfigPayload{
				Enabled: mcpBoolPtr(false),
			}
			_, err := repo.UpdateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", "does_not_exist", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("should fail when trying to update the source type", func() {
			payload := models.McpCatalogSourceConfigPayload{
				Type: "hf",
			}
			_, err := repo.UpdateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", "custom_mcp_servers", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot change"))
		})

		It("should fail when updating the name of the default source config", func() {
			payload := models.McpCatalogSourceConfigPayload{
				Name: "Changed Name",
			}
			_, err := repo.UpdateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", "community_mcp_servers", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot change"))
		})

		It("should fail when updating the labels of default source config", func() {
			payload := models.McpCatalogSourceConfigPayload{
				Labels: []string{"new-label"},
			}
			_, err := repo.UpdateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", "community_mcp_servers", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot change"))
		})

		It("should fail when updating the yaml of the default source config", func() {
			payload := models.McpCatalogSourceConfigPayload{
				Yaml: mcpStringPtr("new content"),
			}
			_, err := repo.UpdateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", "community_mcp_servers", payload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot change"))
		})

		It("should update the user managed source config successfully", func() {
			payload := models.McpCatalogSourceConfigPayload{
				Name:    "Updated Name",
				Enabled: mcpBoolPtr(false),
			}
			result, err := repo.UpdateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", "custom_mcp_servers", payload)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Name).To(Equal("Updated Name"))
		})

		It("should override the enabled property of default source config in user managed configMap", func() {
			payload := models.McpCatalogSourceConfigPayload{
				Enabled: mcpBoolPtr(false),
			}
			result, err := repo.UpdateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", "community_mcp_servers", payload)
			Expect(err).NotTo(HaveOccurred())
			Expect(*result.Enabled).To(BeFalse())
		})

		It("should allow updating includedServers on default catalog", func() {
			payload := models.McpCatalogSourceConfigPayload{
				IncludedServers: []string{"server-a", "server-b"},
			}
			result, err := repo.UpdateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", "community_mcp_servers", payload)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IncludedServers).To(ConsistOf("server-a", "server-b"))
		})

		It("should allow updating excludedServers on default catalog", func() {
			payload := models.McpCatalogSourceConfigPayload{
				ExcludedServers: []string{"blocked-*"},
			}
			result, err := repo.UpdateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", "community_mcp_servers", payload)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ExcludedServers).To(ConsistOf("blocked-*"))
		})

		It("should allow updating default twice", func() {
			payload1 := models.McpCatalogSourceConfigPayload{
				Enabled: mcpBoolPtr(false),
			}
			_, err := repo.UpdateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", "verified_mcp_servers", payload1)
			Expect(err).NotTo(HaveOccurred())

			payload2 := models.McpCatalogSourceConfigPayload{
				Enabled:         mcpBoolPtr(true),
				IncludedServers: []string{"updated-server"},
			}
			result, err := repo.UpdateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", "verified_mcp_servers", payload2)
			Expect(err).NotTo(HaveOccurred())
			Expect(*result.Enabled).To(BeTrue())
			Expect(result.IncludedServers).To(ConsistOf("updated-server"))
		})

		It("should clear includedServers when empty array is provided", func() {
			initialPayload := models.McpCatalogSourceConfigPayload{
				Id:              "test_clear_mcp_servers",
				Name:            "Test Clear MCP Servers",
				Type:            "yaml",
				IncludedServers: []string{"server-*"},
				ExcludedServers: []string{"old-*"},
				Yaml:            mcpStringPtr("servers: []"),
			}
			_, err := repo.CreateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", initialPayload)
			Expect(err).NotTo(HaveOccurred())

			emptyServers := []string{}
			updatePayload := models.McpCatalogSourceConfigPayload{
				IncludedServers: emptyServers,
				ExcludedServers: []string{"*"},
			}

			result, err := repo.UpdateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", "test_clear_mcp_servers", updatePayload)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IncludedServers).To(BeEmpty())
			Expect(result.ExcludedServers).To(Equal([]string{"*"}))
		})

		It("should clear excludedServers when empty array is provided", func() {
			initialPayload := models.McpCatalogSourceConfigPayload{
				Id:              "test_clear_excluded_mcp_servers",
				Name:            "Test Clear Excluded MCP Servers",
				Type:            "yaml",
				IncludedServers: []string{"server-*"},
				ExcludedServers: []string{"old-*"},
				Yaml:            mcpStringPtr("servers: []"),
			}
			_, err := repo.CreateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", initialPayload)
			Expect(err).NotTo(HaveOccurred())

			emptyServers := []string{}
			updatePayload := models.McpCatalogSourceConfigPayload{
				IncludedServers: []string{"*"},
				ExcludedServers: emptyServers,
			}

			result, err := repo.UpdateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", "test_clear_excluded_mcp_servers", updatePayload)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ExcludedServers).To(BeEmpty())
			Expect(result.IncludedServers).To(Equal([]string{"*"}))
		})

		It("should return conflict error when concurrent updates occur with stale resourceVersion", func() {
			createPayload := models.McpCatalogSourceConfigPayload{
				Id:      "mcp_concurrency_test",
				Name:    "MCP Concurrency Test",
				Type:    "yaml",
				Enabled: mcpBoolPtr(true),
				Yaml:    mcpStringPtr("servers: []"),
			}
			_, err := repo.CreateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", createPayload)
			Expect(err).NotTo(HaveOccurred())

			_, cmA, err := k8sClient.GetAllMcpCatalogSourceConfigs(ctx, "kubeflow")
			Expect(err).NotTo(HaveOccurred())
			resourceVersionA := cmA.ResourceVersion
			Expect(resourceVersionA).NotTo(BeEmpty())

			_, cmB, err := k8sClient.GetAllMcpCatalogSourceConfigs(ctx, "kubeflow")
			Expect(err).NotTo(HaveOccurred())
			Expect(cmB.ResourceVersion).To(Equal(resourceVersionA))

			payloadA := models.McpCatalogSourceConfigPayload{
				Name: "Updated by Request A",
			}
			_, err = repo.UpdateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", "mcp_concurrency_test", payloadA)
			Expect(err).NotTo(HaveOccurred())

			_, cmAfterA, err := k8sClient.GetAllMcpCatalogSourceConfigs(ctx, "kubeflow")
			Expect(err).NotTo(HaveOccurred())
			Expect(cmAfterA.ResourceVersion).NotTo(Equal(resourceVersionA))

			cmB.Data[k8s.McpCatalogSourceKey] = "mcp_catalogs:\n  - id: mcp_concurrency_test\n    name: Updated by Request B\n    type: yaml\n    enabled: true"
			err = k8sClient.UpdateMcpCatalogSourceConfig(ctx, "kubeflow", &cmB)
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsConflict(err)).To(BeTrue(), "Expected conflict error due to stale resourceVersion, got: %v", err)
		})
	})

	Describe("DeleteMcpCatalogSourceConfig", func() {
		It("should fail when deleting default catalog", func() {
			_, err := repo.DeleteMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", "community_mcp_servers")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot delete the default MCP source: 'community_mcp_servers' is a default source"))
		})

		It("should fail when the source config does not exist in catalog", func() {
			_, err := repo.DeleteMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", "totally_fake_id")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("should delete user yaml source successfully", func() {
			createPayload := models.McpCatalogSourceConfigPayload{
				Id:      "delete_test_mcp",
				Name:    "Delete MCP Test",
				Type:    "yaml",
				Enabled: mcpBoolPtr(true),
				Yaml:    mcpStringPtr("servers: []"),
			}
			_, err := repo.CreateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", createPayload)
			Expect(err).NotTo(HaveOccurred())

			deleted, err := repo.DeleteMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", "delete_test_mcp")
			Expect(err).NotTo(HaveOccurred())
			Expect(deleted.Id).To(Equal("delete_test_mcp"))
		})
	})
})

func mcpBoolPtr(b bool) *bool {
	return &b
}

func mcpStringPtr(s string) *string {
	return &s
}
