package api

import (
	"net/http"

	"github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TestGetAllCatalogSourcesHandler", func() {
	Context("testing Catalog Sources Handler", Ordered, func() {

		It("should retrieve all catalog sources", func() {
			By("fetching all catalog sources")
			data := mocks.GetCatalogSourceListMock()
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			expected := CatalogSourceListEnvelope{Data: &data}
			actual, rs, err := setupApiTest[CatalogSourceListEnvelope](http.MethodGet, "/api/v1/model_catalog/sources?namespace=kubeflow", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should match the expected catalog sources")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data.Size).To(Equal(expected.Data.Size))
			Expect(actual.Data.PageSize).To(Equal(expected.Data.PageSize))
			Expect(actual.Data.NextPageToken).To(Equal(expected.Data.NextPageToken))
			Expect(len(actual.Data.Items)).To(Equal(len(expected.Data.Items)))
			Expect(actual.Data.Items).To(Equal(expected.Data.Items))
		})

		It("should retrieve all catalog sources filtered by name", func() {
			By("fetching catalog sources filtered by name")
			data := mocks.GetCatalogSourceListMock()
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			actual, rs, err := setupApiTest[CatalogSourceListEnvelope](http.MethodGet, "/api/v1/model_catalog/sources?namespace=kubeflow&name=dora", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should match the expected catalog sources")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(len(actual.Data.Items)).To(Equal(1))
			//dora is the [4]item in the mock data
			Expect(actual.Data.Items[0].Name).To(Equal(data.Items[4].Name))
		})

		It("should retrieve catalog sources with assetType query param", func() {
			By("fetching catalog sources with assetType=models")
			data := mocks.GetCatalogSourceListMock()
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			expected := CatalogSourceListEnvelope{Data: &data}
			actual, rs, err := setupApiTest[CatalogSourceListEnvelope](http.MethodGet, "/api/v1/model_catalog/sources?namespace=kubeflow&assetType=models", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should return 200 and envelope with catalog sources")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data).NotTo(BeNil())
			Expect(actual.Data.Size).To(Equal(expected.Data.Size))
			Expect(actual.Data.PageSize).To(Equal(expected.Data.PageSize))
			Expect(len(actual.Data.Items)).To(Equal(len(expected.Data.Items)))
		})

		It("should retrieve catalog sources filtered by assetType=mcp_servers", func() {
			By("fetching catalog sources with assetType=mcp_servers")
			data := mocks.GetMcpServerCatalogSourceListMock()
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			expected := CatalogSourceListEnvelope{Data: &data}
			actual, rs, err := setupApiTest[CatalogSourceListEnvelope](http.MethodGet, "/api/v1/model_catalog/sources?namespace=kubeflow&assetType=mcp_servers", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should return 200 and envelope with MCP server catalog sources")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data).NotTo(BeNil())
			Expect(actual.Data.Size).To(Equal(expected.Data.Size))
			Expect(actual.Data.PageSize).To(Equal(expected.Data.PageSize))
			Expect(len(actual.Data.Items)).To(Equal(len(expected.Data.Items)))
			Expect(actual.Data.Items).To(Equal(expected.Data.Items))
		})

		It("should retrieve catalog sources model", func() {
			By("fetching catalog sources models")
			data := mocks.GetCatalogModelMocks()[0]
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			actual, rs, err := setupApiTest[CatalogModelEnvelope](http.MethodGet, "/api/v1/model_catalog/sources/sample-source/models/repo1%2Fgranite-8b-code-instruct?namespace=kubeflow&name=dora", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should match the expected model")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data.Name).To(Equal(data.Name))
		})

		It("should retrieve all catalog model artifacts", func() {
			By("fetching all catalog model artifacts")
			data := mocks.GetCatalogModelArtifactListMock()
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			expected := catalogModelArtifactsListEnvelope{Data: &data}
			actual, rs, err := setupApiTest[catalogModelArtifactsListEnvelope](http.MethodGet, "/api/v1/model_catalog/sources/source/artifacts/model-name?namespace=kubeflow&name=dora", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should match the expected model artifacts")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data.Size).To(Equal(expected.Data.Size))
			Expect(actual.Data.PageSize).To(Equal(expected.Data.PageSize))
			Expect(actual.Data.NextPageToken).To(Equal(expected.Data.NextPageToken))
			Expect(len(actual.Data.Items)).To(Equal(len(expected.Data.Items)))
			Expect(actual.Data.Items).To(Equal(expected.Data.Items))
		})

		It("should retrieve the performance artifacts", func() {
			By("fetching the catalog performance artifacts")
			data := mocks.GetCatalogPerformanceMetricsArtifactListMock(4)
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			expected := catalogModelArtifactsListEnvelope{Data: &data}
			actual, rs, err := setupApiTest[catalogModelArtifactsListEnvelope](http.MethodGet, "/api/v1/model_catalog/sources/source/performance_artifacts/model-name?namespace=kubeflow&name=dora", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should fetch the expected performance artifacts")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data.Size).To(Equal(expected.Data.Size))
			Expect(actual.Data.PageSize).To(Equal(expected.Data.PageSize))
			Expect(actual.Data.NextPageToken).To(Equal(expected.Data.NextPageToken))
			Expect(len(actual.Data.Items)).To(Equal(len(expected.Data.Items)))
			for i := range expected.Data.Items {
				Expect(actual.Data.Items[i].ArtifactType).To(Equal(expected.Data.Items[i].ArtifactType))
				Expect(actual.Data.Items[i].MetricsType).To(Equal(expected.Data.Items[i].MetricsType))
				Expect(actual.Data.Items[i].CreateTimeSinceEpoch).To(Equal(expected.Data.Items[i].CreateTimeSinceEpoch))
			}
		})

		It("should retrieve the security artifacts", func() {
			By("fetching the catalog security artifacts")
			data := mocks.GetCatalogSecurityMetricsArtifactListMock()
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			expected := catalogModelArtifactsListEnvelope{Data: &data}
			actual, rs, err := setupApiTest[catalogModelArtifactsListEnvelope](http.MethodGet, "/api/v1/model_catalog/sources/source/security_artifacts/model-name?namespace=kubeflow", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should fetch the expected security artifacts")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data.Size).To(Equal(expected.Data.Size))
			Expect(actual.Data.PageSize).To(Equal(expected.Data.PageSize))
			Expect(actual.Data.NextPageToken).To(Equal(expected.Data.NextPageToken))
			Expect(len(actual.Data.Items)).To(Equal(len(expected.Data.Items)))
			for i := range expected.Data.Items {
				Expect(actual.Data.Items[i].ArtifactType).To(Equal(expected.Data.Items[i].ArtifactType))
				Expect(actual.Data.Items[i].MetricsType).To(Equal(expected.Data.Items[i].MetricsType))
				Expect(actual.Data.Items[i].CreateTimeSinceEpoch).To(Equal(expected.Data.Items[i].CreateTimeSinceEpoch))
			}
		})

	})
})

var _ = Describe("TestGetCatalogLabelsHandler", func() {
	Context("testing Catalog Labels Handler", Ordered, func() {

		It("should retrieve all catalog labels (default, no assetType)", func() {
			By("fetching all catalog labels without assetType")
			data := mocks.GetCatalogLabelListMock()
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			expected := CatalogLabelListEnvelope{Data: &data}
			actual, rs, err := setupApiTest[CatalogLabelListEnvelope](http.MethodGet, "/api/v1/model_catalog/labels?namespace=kubeflow", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should match the expected catalog labels")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data.Size).To(Equal(expected.Data.Size))
			Expect(actual.Data.PageSize).To(Equal(expected.Data.PageSize))
			Expect(actual.Data.NextPageToken).To(Equal(expected.Data.NextPageToken))
			Expect(len(actual.Data.Items)).To(Equal(len(expected.Data.Items)))
			Expect(actual.Data.Items).To(Equal(expected.Data.Items))
		})

		It("should retrieve catalog labels with assetType=models", func() {
			By("fetching catalog labels with assetType=models")
			data := mocks.GetCatalogLabelListMock()
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			expected := CatalogLabelListEnvelope{Data: &data}
			actual, rs, err := setupApiTest[CatalogLabelListEnvelope](http.MethodGet, "/api/v1/model_catalog/labels?namespace=kubeflow&assetType=models", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should return model catalog labels")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data).NotTo(BeNil())
			Expect(actual.Data.Size).To(Equal(expected.Data.Size))
			Expect(actual.Data.PageSize).To(Equal(expected.Data.PageSize))
			Expect(len(actual.Data.Items)).To(Equal(len(expected.Data.Items)))
			Expect(actual.Data.Items).To(Equal(expected.Data.Items))
		})

		It("should retrieve catalog labels filtered by assetType=mcp_servers", func() {
			By("fetching catalog labels with assetType=mcp_servers")
			data := mocks.GetMcpServerCatalogLabelListMock()
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			expected := CatalogLabelListEnvelope{Data: &data}
			actual, rs, err := setupApiTest[CatalogLabelListEnvelope](http.MethodGet, "/api/v1/model_catalog/labels?namespace=kubeflow&assetType=mcp_servers", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should return MCP server catalog labels")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data).NotTo(BeNil())
			Expect(actual.Data.Size).To(Equal(expected.Data.Size))
			Expect(actual.Data.PageSize).To(Equal(expected.Data.PageSize))
			Expect(len(actual.Data.Items)).To(Equal(len(expected.Data.Items)))
			Expect(actual.Data.Items).To(Equal(expected.Data.Items))
		})

	})
})
