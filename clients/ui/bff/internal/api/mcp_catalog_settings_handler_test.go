package api

import (
	"net/http"

	"github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TestMcpCatalogSettings", func() {
	var requestIdentity kubernetes.RequestIdentity

	BeforeEach(func() {
		requestIdentity = kubernetes.RequestIdentity{
			UserID: "user@example.com",
		}
	})

	Context("fetching MCP catalog source config", func() {
		It("GET ALL returns 200", func() {
			_, rs, err := setupApiTest[McpCatalogSettingsSourceConfigListEnvelope](
				http.MethodGet,
				"/api/v1/settings/mcp_catalog/source_configs?namespace=kubeflow",
				nil,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
		})

		It("GET SINGLE returns 200", func() {
			_, rs, err := setupApiTest[McpCatalogSettingsSourceConfigEnvelope](
				http.MethodGet,
				"/api/v1/settings/mcp_catalog/source_configs/community_mcp_servers?namespace=kubeflow",
				nil,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
		})

		It("GET returns 404 for non-existent source", func() {
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodGet,
				"/api/v1/settings/mcp_catalog/source_configs/does_not_exist?namespace=kubeflow",
				nil,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("GET returns 400 when namespace is missing", func() {
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodGet,
				"/api/v1/settings/mcp_catalog/source_configs",
				nil,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
		})
	})

	Context("creating MCP source config", func() {
		It("POST returns 201 on success", func() {
			payload := McpCatalogSourcePayloadEnvelope{
				Data: &models.McpCatalogSourceConfigPayload{
					Id:      "mcp_handler_test_create",
					Name:    "MCP Handler Test",
					Type:    "yaml",
					Enabled: mcpHandlerBoolPtr(true),
					Yaml:    mcpHandlerStringPtr("servers: []"),
				},
			}
			_, rs, err := setupApiTest[McpCatalogSettingsSourceConfigEnvelope](
				http.MethodPost,
				"/api/v1/settings/mcp_catalog/source_configs?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusCreated))
		})

		It("POST returns 400 for validation error (missing required field)", func() {
			payload := McpCatalogSourcePayloadEnvelope{
				Data: &models.McpCatalogSourceConfigPayload{
					Name:    "Test",
					Type:    "yaml",
					Enabled: mcpHandlerBoolPtr(true),
					Yaml:    mcpHandlerStringPtr("servers: []"),
				},
			}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPost,
				"/api/v1/settings/mcp_catalog/source_configs?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("POST returns 400 for duplicate source", func() {
			payload := McpCatalogSourcePayloadEnvelope{
				Data: &models.McpCatalogSourceConfigPayload{
					Id:      "community_mcp_servers",
					Name:    "Duplicate",
					Type:    "yaml",
					Enabled: mcpHandlerBoolPtr(true),
					Yaml:    mcpHandlerStringPtr("servers: []"),
				},
			}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPost,
				"/api/v1/settings/mcp_catalog/source_configs?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("POST returns 400 when type is missing", func() {
			payload := McpCatalogSourcePayloadEnvelope{
				Data: &models.McpCatalogSourceConfigPayload{
					Id:      "test_mcp_missing_type",
					Name:    "Test Missing Type",
					Enabled: mcpHandlerBoolPtr(true),
					Yaml:    mcpHandlerStringPtr("servers: []"),
				},
			}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPost,
				"/api/v1/settings/mcp_catalog/source_configs?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("POST returns 400 when yaml is missing for yaml type", func() {
			payload := McpCatalogSourcePayloadEnvelope{
				Data: &models.McpCatalogSourceConfigPayload{
					Id:      "test_mcp_yaml_no_content",
					Name:    "Test YAML No Content",
					Type:    "yaml",
					Enabled: mcpHandlerBoolPtr(true),
				},
			}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPost,
				"/api/v1/settings/mcp_catalog/source_configs?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("POST returns 400 for unsupported catalog type", func() {
			payload := McpCatalogSourcePayloadEnvelope{
				Data: &models.McpCatalogSourceConfigPayload{
					Id:      "test_mcp_invalid_type",
					Name:    "Test Invalid Type",
					Type:    "invalid_type",
					Enabled: mcpHandlerBoolPtr(true),
					Yaml:    mcpHandlerStringPtr("servers: []"),
				},
			}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPost,
				"/api/v1/settings/mcp_catalog/source_configs?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
		})
	})

	Context("patching an MCP source config", func() {
		It("PATCH returns 200 on success", func() {
			payload := McpCatalogSourcePayloadEnvelope{
				Data: &models.McpCatalogSourceConfigPayload{Enabled: mcpHandlerBoolPtr(false)},
			}
			_, rs, err := setupApiTest[McpCatalogSettingsSourceConfigEnvelope](
				http.MethodPatch,
				"/api/v1/settings/mcp_catalog/source_configs/community_mcp_servers?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
		})

		It("PATCH returns 404 for non-existent source", func() {
			payload := McpCatalogSourcePayloadEnvelope{
				Data: &models.McpCatalogSourceConfigPayload{Enabled: mcpHandlerBoolPtr(false)},
			}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPatch,
				"/api/v1/settings/mcp_catalog/source_configs/does_not_exist?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("PATCH returns 403 when changing type", func() {
			payload := McpCatalogSourcePayloadEnvelope{
				Data: &models.McpCatalogSourceConfigPayload{Type: "hf"},
			}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPatch,
				"/api/v1/settings/mcp_catalog/source_configs/community_mcp_servers?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusForbidden))
		})

		It("PATCH returns 403 when updating forbidden field on default", func() {
			payload := McpCatalogSourcePayloadEnvelope{
				Data: &models.McpCatalogSourceConfigPayload{Name: "Changed Name"},
			}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPatch,
				"/api/v1/settings/mcp_catalog/source_configs/community_mcp_servers?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusForbidden))
		})
	})

	Context("deleting an MCP source config", func() {
		It("DELETE returns 200 on success", func() {
			createPayload := McpCatalogSourcePayloadEnvelope{
				Data: &models.McpCatalogSourceConfigPayload{
					Id:      "mcp_delete_handler_test",
					Name:    "MCP Delete Test",
					Type:    "yaml",
					Enabled: mcpHandlerBoolPtr(true),
					Yaml:    mcpHandlerStringPtr("servers: []"),
				},
			}
			_, _, err := setupApiTest[McpCatalogSettingsSourceConfigEnvelope](
				http.MethodPost,
				"/api/v1/settings/mcp_catalog/source_configs?namespace=kubeflow",
				createPayload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())

			_, rs, err := setupApiTest[McpCatalogSettingsSourceConfigEnvelope](
				http.MethodDelete,
				"/api/v1/settings/mcp_catalog/source_configs/mcp_delete_handler_test?namespace=kubeflow",
				nil,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
		})

		It("DELETE returns 404 for non-existent source", func() {
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodDelete,
				"/api/v1/settings/mcp_catalog/source_configs/does_not_exist?namespace=kubeflow",
				nil,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("DELETE returns 403 for default source", func() {
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodDelete,
				"/api/v1/settings/mcp_catalog/source_configs/community_mcp_servers?namespace=kubeflow",
				nil,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusForbidden))
		})
	})
})

func mcpHandlerBoolPtr(b bool) *bool {
	return &b
}

func mcpHandlerStringPtr(s string) *string {
	return &s
}
