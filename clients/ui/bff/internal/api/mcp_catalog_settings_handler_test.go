package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/hub/ui/bff/internal/constants"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/mocks"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	"github.com/kubeflow/hub/ui/bff/internal/repositories"
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

		It("PATCH clears status once when YAML changes", func() {
			mcpHandlerUpdateAndAssertStatus(
				"mcp_handler_yaml_changed",
				true,
				models.McpCatalogSourceConfigPayload{Yaml: mcpHandlerStringPtr("servers:\n  - name: changed")},
				1,
				http.StatusOK,
			)
		})

		It("PATCH does not clear status when YAML is unchanged", func() {
			mcpHandlerUpdateAndAssertStatus(
				"mcp_handler_yaml_unchanged",
				true,
				models.McpCatalogSourceConfigPayload{Yaml: mcpHandlerStringPtr("servers: []")},
				0,
				http.StatusOK,
			)
		})

		It("PATCH clears status when enabling a disabled source", func() {
			mcpHandlerUpdateAndAssertStatus(
				"mcp_handler_enabled_false_to_true",
				false,
				models.McpCatalogSourceConfigPayload{Enabled: mcpHandlerBoolPtr(true)},
				1,
				http.StatusOK,
			)
		})

		It("PATCH does not clear status when disabling an enabled source", func() {
			mcpHandlerUpdateAndAssertStatus(
				"mcp_handler_enabled_true_to_false",
				true,
				models.McpCatalogSourceConfigPayload{Enabled: mcpHandlerBoolPtr(false)},
				0,
				http.StatusOK,
			)
		})

		It("PATCH does not clear status when the update fails", func() {
			mcpHandlerUpdateAndAssertStatus(
				"mcp_handler_update_failure",
				true,
				models.McpCatalogSourceConfigPayload{Type: "unsupported"},
				0,
				http.StatusForbidden,
			)
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

func mcpHandlerUpdateAndAssertStatus(
	sourceID string,
	initialEnabled bool,
	update models.McpCatalogSourceConfigPayload,
	expectedDeleteCount int,
	expectedStatus int,
) {
	ctx := context.Background()
	serverCalls := 0
	var unexpectedRequest string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/sources/" + sourceID + "/status"
		if expectedDeleteCount == 0 || r.Method != http.MethodDelete || r.URL.Path != expectedPath {
			unexpectedRequest = r.Method + " " + r.URL.Path
			http.Error(w, "unexpected downstream request", http.StatusInternalServerError)
			return
		}

		serverCalls++
		if serverCalls > 1 {
			http.Error(w, "duplicate downstream request", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	modelRegistryClient, err := mocks.NewModelRegistryClient(nil)
	Expect(err).NotTo(HaveOccurred())
	modelCatalogClient, err := mocks.NewModelCatalogClientMock(nil)
	Expect(err).NotTo(HaveOccurred())
	app := App{
		repositories:            repositories.NewRepositories(modelRegistryClient, modelCatalogClient),
		kubernetesClientFactory: kubernetesMockedStaticClientFactory,
		logger:                  slog.Default(),
	}
	k8sClient, err := kubernetesMockedStaticClientFactory.GetClient(ctx)
	Expect(err).NotTo(HaveOccurred())
	_, err = app.repositories.McpCatalogSettingsRepository.CreateMcpCatalogSourceConfig(ctx, k8sClient, "kubeflow", models.McpCatalogSourceConfigPayload{
		Id:      sourceID,
		Name:    "Display name for " + sourceID,
		Type:    "yaml",
		Enabled: mcpHandlerBoolPtr(initialEnabled),
		Yaml:    mcpHandlerStringPtr("servers: []"),
	})
	Expect(err).NotTo(HaveOccurred())

	statusClient, err := httpclient.NewHTTPClient(slog.Default(), server.URL, nil, false, nil)
	Expect(err).NotTo(HaveOccurred())
	requestBody, err := json.Marshal(McpCatalogSourcePayloadEnvelope{Data: &update})
	Expect(err).NotTo(HaveOccurred())
	request := httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader(requestBody))
	request = request.WithContext(context.WithValue(context.WithValue(ctx, constants.NamespaceHeaderParameterKey, "kubeflow"), constants.ModelCatalogStatusHttpClientKey, statusClient))
	response := httptest.NewRecorder()

	app.UpdateMcpCatalogSourceConfigHandler(response, request, httprouter.Params{{Key: CatalogSourceId, Value: sourceID}})

	Expect(response.Code).To(Equal(expectedStatus))
	Expect(unexpectedRequest).To(BeEmpty())
	Expect(serverCalls).To(Equal(expectedDeleteCount))
}
