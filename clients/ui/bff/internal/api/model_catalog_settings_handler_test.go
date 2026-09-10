package api

import (
	"net/http"

	"github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TestModelCatalogSettings", func() {
	var requestIdentity kubernetes.RequestIdentity

	BeforeEach(func() {
		requestIdentity = kubernetes.RequestIdentity{
			UserID: "user@example.com",
		}
	})
	Context("fetching catalog source config", func() {
		It("GET ALL returns 200", func() {
			_, rs, err := setupApiTest[ModelCatalogSettingsSourceConfigListEnvelope](
				http.MethodGet,
				"/api/v1/settings/model_catalog/source_configs?namespace=kubeflow",
				nil,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
		})

		It("GET SINGLE returns 200", func() {
			_, rs, err := setupApiTest[ModelCatalogSettingsSourceConfigEnvelope](
				http.MethodGet,
				"/api/v1/settings/model_catalog/source_configs/dora_ai_models?namespace=kubeflow",
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
				"/api/v1/settings/model_catalog/source_configs/does_not_exist?namespace=kubeflow",
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
				"/api/v1/settings/model_catalog/source_configs",
				nil,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"", // empty namespace
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
		})
	})

	Context("creating source config", func() {
		It("POST returns 201 on success", func() {
			payload := ModelCatalogSourcePayloadEnvelope{
				Data: &models.CatalogSourceConfigPayload{
					Id:      "minimal_handler_test",
					Name:    "Minimal Handler Test",
					Type:    "yaml",
					Enabled: boolPtr(true),
					Yaml:    stringPtr("models: []"),
				},
			}
			_, rs, err := setupApiTest[ModelCatalogSettingsSourceConfigEnvelope](
				http.MethodPost,
				"/api/v1/settings/model_catalog/source_configs?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusCreated))
		})

		It("POST returns 400 for validation error (missing required field)", func() {
			payload := ModelCatalogSourcePayloadEnvelope{
				Data: &models.CatalogSourceConfigPayload{
					Name:    "Test", // missing id
					Type:    "yaml",
					Enabled: boolPtr(true),
					Yaml:    stringPtr("models: []"),
				},
			}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPost,
				"/api/v1/settings/model_catalog/source_configs?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("POST returns 400 for duplicate source", func() {
			payload := ModelCatalogSourcePayloadEnvelope{
				Data: &models.CatalogSourceConfigPayload{
					Id:      "dora_ai_models", // existing default
					Name:    "Duplicate",
					Type:    "yaml",
					Enabled: boolPtr(true),
					Yaml:    stringPtr("models: []"),
				},
			}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPost,
				"/api/v1/settings/model_catalog/source_configs?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("POST returns 400 when type is missing", func() {
			payload := ModelCatalogSourcePayloadEnvelope{
				Data: &models.CatalogSourceConfigPayload{
					Id:      "test_missing_type",
					Name:    "Test Missing Type",
					Enabled: boolPtr(true),
					Yaml:    stringPtr("models: []"),
					//type is missing
				},
			}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPost,
				"/api/v1/settings/model_catalog/source_configs?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("POST returns 400 when yaml is missing for yaml type", func() {
			payload := ModelCatalogSourcePayloadEnvelope{
				Data: &models.CatalogSourceConfigPayload{
					Id:      "test_yaml_no_content",
					Name:    "Test YAML No Content",
					Type:    "yaml",
					Enabled: boolPtr(true),
					// yaml field missing
				},
			}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPost,
				"/api/v1/settings/model_catalog/source_configs?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("POST returns 400 for unsupported catalog type", func() {
			payload := ModelCatalogSourcePayloadEnvelope{
				Data: &models.CatalogSourceConfigPayload{
					Id:      "test_invalid_type",
					Name:    "Test Invalid Type",
					Type:    "invalid_type",
					Enabled: boolPtr(true),
					// invalid type
				},
			}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPost,
				"/api/v1/settings/model_catalog/source_configs?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
		})
	})

	Context("patching a source config", func() {
		It("PATCH returns 200 on success", func() {
			payload := ModelCatalogSourcePayloadEnvelope{
				Data: &models.CatalogSourceConfigPayload{Enabled: boolPtr(false)},
			}
			_, rs, err := setupApiTest[ModelCatalogSettingsSourceConfigEnvelope](
				http.MethodPatch,
				"/api/v1/settings/model_catalog/source_configs/dora_ai_models?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
		})

		It("PATCH returns 404 for non-existent source", func() {
			payload := ModelCatalogSourcePayloadEnvelope{
				Data: &models.CatalogSourceConfigPayload{Enabled: boolPtr(false)},
			}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPatch,
				"/api/v1/settings/model_catalog/source_configs/does_not_exist?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("PATCH returns 403 when changing type", func() {
			payload := ModelCatalogSourcePayloadEnvelope{
				Data: &models.CatalogSourceConfigPayload{Type: "hf"},
			}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPatch,
				"/api/v1/settings/model_catalog/source_configs/dora_ai_models?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusForbidden))
		})

		It("PATCH returns 403 when updating forbidden field on default", func() {
			payload := ModelCatalogSourcePayloadEnvelope{
				Data: &models.CatalogSourceConfigPayload{Name: "Changed Name"},
			}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPatch,
				"/api/v1/settings/model_catalog/source_configs/dora_ai_models?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusForbidden))
		})

	})

	Context("deleting a source config", func() {
		It("DELETE returns 200 on success", func() {
			// create a source config, before deleting that
			createPayload := ModelCatalogSourcePayloadEnvelope{
				Data: &models.CatalogSourceConfigPayload{
					Id:      "delete_handler_test",
					Name:    "Delete Test",
					Type:    "yaml",
					Enabled: boolPtr(true),
					Yaml:    stringPtr("models: []"),
				},
			}
			_, _, err := setupApiTest[ModelCatalogSettingsSourceConfigEnvelope](
				http.MethodPost,
				"/api/v1/settings/model_catalog/source_configs?namespace=kubeflow",
				createPayload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())

			_, rs, err := setupApiTest[ModelCatalogSettingsSourceConfigEnvelope](
				http.MethodDelete,
				"/api/v1/settings/model_catalog/source_configs/delete_handler_test?namespace=kubeflow",
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
				"/api/v1/settings/model_catalog/source_configs/does_not_exist?namespace=kubeflow",
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
				"/api/v1/settings/model_catalog/source_configs/dora_ai_models?namespace=kubeflow",
				nil,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusForbidden))
		})

		It("DELETE credentials returns 200 with empty JSON body", func() {
			rs, body, err := serveApiTest(
				http.MethodDelete,
				"/api/v1/settings/model_catalog/source_configs/hugging_face_source/credentials?namespace=kubeflow",
				nil,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal("{}\n"))
		})
	})

})

func boolPtr(b bool) *bool {
	return &b
}

func stringPtr(s string) *string {
	return &s
}
