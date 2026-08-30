package api

import (
	"net/http"

	"github.com/kubeflow/hub/pkg/openapi-v1"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TestGetModelVersionHandler", func() {
	Context("testing Model Version Handler", Ordered, func() {

		It("should retrieve all model versions", func() {
			By("fetching all model versions")
			data := mocks.GetModelVersionListMock()
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			expected := ModelVersionListEnvelope{Data: &data}
			actual, rs, err := setupApiTest[ModelVersionListEnvelope](http.MethodGet, "/api/v1/model_registry/model-registry/model_versions?namespace=kubeflow", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())
			By("should match the expected model versions")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data.Size).To(Equal(expected.Data.Size))
			Expect(actual.Data.Items).To(Equal(expected.Data.Items))
		})

		It("should retrieve a model version", func() {
			By("fetching a model version")
			data := mocks.GetModelVersionMocks()[0]
			expected := ModelVersionEnvelope{Data: &data}
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}
			actual, rs, err := setupApiTest[ModelVersionEnvelope](http.MethodGet, "/api/v1/model_registry/model-registry/model_versions/1?namespace=kubeflow", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())
			By("should match the expected model version")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data.Name).To(Equal(expected.Data.Name))
		})

		It("should create a model version", func() {
			By("creating a model version")
			data := mocks.GetModelVersionMocks()[0]
			expected := ModelVersionEnvelope{Data: &data}
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}
			body := ModelVersionEnvelope{Data: openapi.NewModelVersion("Model One", "1")}
			actual, rs, err := setupApiTest[ModelVersionEnvelope](http.MethodPost, "/api/v1/model_registry/model-registry/model_versions?namespace=kubeflow", body, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should match the expected model version created")
			Expect(rs.StatusCode).To(Equal(http.StatusCreated))
			Expect(actual.Data.Name).To(Equal(expected.Data.Name))
			Expect(rs.Header.Get("Location")).To(Equal("/api/v1/model_registry/model-registry/model_versions/1?namespace=kubeflow"))
		})

		It("should updated a model version", func() {
			By("updating a model version")
			data := mocks.GetModelVersionMocks()[0]
			expected := ModelVersionEnvelope{Data: &data}

			reqData := openapi.ModelVersionUpdate{
				Description: openapi.PtrString("New description"),
			}
			body := ModelVersionUpdateEnvelope{Data: &reqData}
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			actual, rs, err := setupApiTest[ModelVersionEnvelope](http.MethodPatch, "/api/v1/model_registry/model-registry/model_versions/1?namespace=kubeflow", body, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should match the expected model version updated")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data.Name).To(Equal(expected.Data.Name))
		})

		It("get all model artifacts by a model version", func() {
			By("getting a model artifacts by model version")
			data := mocks.GetModelArtifactListMock()
			expected := ModelArtifactListEnvelope{Data: &data}
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}
			actual, rs, err := setupApiTest[ModelArtifactListEnvelope](http.MethodGet, "/api/v1/model_registry/model-registry/model_versions/1/artifacts?namespace=kubeflow", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should get all expected model version artifacts")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data.Size).To(Equal(expected.Data.Size))
			Expect(actual.Data.PageSize).To(Equal(expected.Data.PageSize))
			Expect(actual.Data.NextPageToken).To(Equal(expected.Data.NextPageToken))
			Expect(len(actual.Data.Items)).To(Equal(len(expected.Data.Items)))
		})

		It("create Model Artifact By Model Version", func() {
			By("creating a model version")
			data := mocks.GetModelArtifactMocks()[0]
			expected := ModelArtifactEnvelope{Data: &data}
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}
			artifact := openapi.ModelArtifact{
				Name:         openapi.PtrString("Artifact One"),
				ArtifactType: openapi.PtrString("ARTIFACT_TYPE_ONE"),
			}
			body := ModelArtifactEnvelope{Data: &artifact}
			actual, rs, err := setupApiTest[ModelArtifactEnvelope](http.MethodPost, "/api/v1/model_registry/model-registry/model_versions/1/artifacts?namespace=kubeflow", body, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should get all expected model artifacts")
			Expect(rs.StatusCode).To(Equal(http.StatusCreated))
			Expect(actual.Data.GetArtifactType()).To(Equal(expected.Data.GetArtifactType()))
			Expect(rs.Header.Get("Location")).To(Equal("/api/v1/model_registry/model-registry/model_artifacts/1"))

		})

		It("should return 403 when not using the wrong KubeflowUserIDHeaderValue", func() {
			By("making a request with an incorrect username")
			wrongRequestIdentity := kubernetes.RequestIdentity{
				UserID: "bella@dora.com", // Incorrect username header value
			}

			// Test: GET /model_versions/1
			_, rs, err := setupApiTest[ModelVersionEnvelope](http.MethodGet, "/api/v1/model_registry/model-registry/model_versions/1?namespace=kubeflow", nil, kubernetesMockedStaticClientFactory, wrongRequestIdentity, "kubeflow")

			Expect(err).NotTo(HaveOccurred())
			By("should return a 403 Forbidden response")
			Expect(rs.StatusCode).To(Equal(http.StatusForbidden))

			// Test: POST /model_versions/1/artifacts
			artifact := openapi.ModelArtifact{
				Name:         openapi.PtrString("Artifact One"),
				ArtifactType: openapi.PtrString("ARTIFACT_TYPE_ONE"),
			}
			body := ModelArtifactEnvelope{Data: &artifact}
			_, rs, err = setupApiTest[ModelArtifactEnvelope](http.MethodPost, "/api/v1/model_registry/model-registry/model_versions/1/artifacts?namespace=kubeflow", body, kubernetesMockedStaticClientFactory, wrongRequestIdentity, "kubeflow")

			Expect(err).NotTo(HaveOccurred())
			By("should return a 403 Forbidden response")
			Expect(rs.StatusCode).To(Equal(http.StatusForbidden))

			// Test: GET /model_versions/1/artifacts
			_, rs, err = setupApiTest[ModelArtifactListEnvelope](http.MethodGet, "/api/v1/model_registry/model-registry/model_versions/1/artifacts?namespace=kubeflow", nil, kubernetesMockedStaticClientFactory, wrongRequestIdentity, "kubeflow")

			Expect(err).NotTo(HaveOccurred())
			By("should return a 403 Forbidden response")
			Expect(rs.StatusCode).To(Equal(http.StatusForbidden))

			// Test: PATCH /model_versions/1
			reqData := openapi.ModelVersionUpdate{
				Description: openapi.PtrString("New description"),
			}
			body1 := ModelVersionUpdateEnvelope{Data: &reqData}
			_, rs, err = setupApiTest[ModelVersionEnvelope](http.MethodPatch, "/api/v1/model_registry/model-registry/model_versions/1?namespace=kubeflow", body1, kubernetesMockedStaticClientFactory, wrongRequestIdentity, "kubeflow")

			Expect(err).NotTo(HaveOccurred())
			By("should return a 403 Forbidden response")
			Expect(rs.StatusCode).To(Equal(http.StatusForbidden))

			// Test: POST /model_versions
			body2 := ModelVersionEnvelope{Data: openapi.NewModelVersion("Model One", "1")}
			_, rs, err = setupApiTest[ModelVersionEnvelope](http.MethodPost, "/api/v1/model_registry/model-registry/model_versions?namespace=kubeflow", body2, kubernetesMockedStaticClientFactory, wrongRequestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())
			By("should return a 403 Forbidden response")
			Expect(rs.StatusCode).To(Equal(http.StatusForbidden))
		})
	})
})
