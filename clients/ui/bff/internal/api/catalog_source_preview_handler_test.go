package api

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/mocks"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateCatalogSourcePreviewHandler", func() {
	Context("testing create catalog source preview handler", Ordered, func() {
		It("should create source preview for a source", func() {
			By("creating a source preview")
			data := mocks.CreateCatalogSourcePreviewMock()
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			requestBody := struct {
				Data models.CatalogSourcePreviewRequest `json:"data"`
			}{
				Data: models.CatalogSourcePreviewRequest{
					Type:           "yaml",
					IncludedModels: []string{},
					ExcludedModels: []string{},
					Properties: map[string]interface{}{
						"yaml": "models:\n  - name: test-model",
					},
				},
			}
			bodyBytes, _ := json.Marshal(requestBody)

			expected := CatalogSourcePreviewEnvelope{Data: &data}
			actual, rs, err := setupApiTest[CatalogSourcePreviewEnvelope](http.MethodPost, "/api/v1/settings/model_catalog/source_preview?namespace=kubeflow", bytes.NewReader(bodyBytes), kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should match the expected source preview")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data.Size).To(Equal(expected.Data.Size))
			Expect(actual.Data.PageSize).To(Equal(expected.Data.PageSize))
			Expect(actual.Data.NextPageToken).To(Equal(expected.Data.NextPageToken))
			Expect(actual.Data.Items).To(Equal(expected.Data.Items))

		})

		It("should return MCP server preview when assetType=mcp_servers", func() {
			By("creating a source preview with assetType=mcp_servers")
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			requestBody := struct {
				Data models.CatalogSourcePreviewRequest `json:"data"`
			}{
				Data: models.CatalogSourcePreviewRequest{
					Type:            "yaml",
					IncludedServers: []string{"*"},
					ExcludedServers: []string{},
					Properties: map[string]interface{}{
						"yaml": "mcp_servers:\n  - name: test-server",
					},
				},
			}
			bodyBytes, _ := json.Marshal(requestBody)

			actual, rs, err := setupApiTest[CatalogSourcePreviewEnvelope](http.MethodPost, "/api/v1/settings/model_catalog/source_preview?namespace=kubeflow&assetType=mcp_servers", bytes.NewReader(bodyBytes), kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should return MCP server preview data")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data.Summary.TotalAssets).To(Equal(int32(17)))
			Expect(actual.Data.Summary.IncludedAssets).To(Equal(int32(12)))
			Expect(actual.Data.Summary.ExcludedAssets).To(Equal(int32(5)))
			for _, item := range actual.Data.Items {
				Expect(item.Name).To(ContainSubstring("mcp-source/"))
			}
		})

		It("should filter by filterStatus=included", func() {
			By("creating a source preview with filterStatus=included")
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			requestBody := struct {
				Data models.CatalogSourcePreviewRequest `json:"data"`
			}{
				Data: models.CatalogSourcePreviewRequest{
					Type:           "yaml",
					IncludedModels: []string{},
					ExcludedModels: []string{},
					Properties: map[string]interface{}{
						"yaml": "models:\n  - name: test-model",
					},
				},
			}
			bodyBytes, _ := json.Marshal(requestBody)

			actual, rs, err := setupApiTest[CatalogSourcePreviewEnvelope](http.MethodPost, "/api/v1/settings/model_catalog/source_preview?namespace=kubeflow&filterStatus=included", bytes.NewReader(bodyBytes), kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should return only included models")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			for _, item := range actual.Data.Items {
				Expect(item.Included).To(BeTrue(), "All items should have Included=true when filterStatus=included")
			}
		})

		It("should filter by filterStatus=excluded", func() {
			By("creating a source preview with filterStatus=excluded")
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			requestBody := struct {
				Data models.CatalogSourcePreviewRequest `json:"data"`
			}{
				Data: models.CatalogSourcePreviewRequest{
					Type:           "yaml",
					IncludedModels: []string{},
					ExcludedModels: []string{},
					Properties: map[string]interface{}{
						"yaml": "models:\n  - name: test-model",
					},
				},
			}
			bodyBytes, _ := json.Marshal(requestBody)

			actual, rs, err := setupApiTest[CatalogSourcePreviewEnvelope](http.MethodPost, "/api/v1/settings/model_catalog/source_preview?namespace=kubeflow&filterStatus=excluded", bytes.NewReader(bodyBytes), kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should return only excluded models")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			for _, item := range actual.Data.Items {
				Expect(item.Included).To(BeFalse(), "All items should have Included=false when filterStatus=excluded")
			}
		})

		It("should paginate with pageSize and nextPageToken", func() {
			By("creating a source preview with pageSize=5")
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			requestBody := struct {
				Data models.CatalogSourcePreviewRequest `json:"data"`
			}{
				Data: models.CatalogSourcePreviewRequest{
					Type:           "yaml",
					IncludedModels: []string{},
					ExcludedModels: []string{},
					Properties: map[string]interface{}{
						"yaml": "models:\n  - name: test-model",
					},
				},
			}
			bodyBytes, _ := json.Marshal(requestBody)

			actual, rs, err := setupApiTest[CatalogSourcePreviewEnvelope](http.MethodPost, "/api/v1/settings/model_catalog/source_preview?namespace=kubeflow&pageSize=5", bytes.NewReader(bodyBytes), kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should return at most 5 items with a nextPageToken")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(len(actual.Data.Items)).To(BeNumerically("<=", 5))
			// If there are more items available, nextPageToken should be set
			if actual.Data.Size > 5 {
				Expect(actual.Data.NextPageToken).NotTo(BeEmpty(), "NextPageToken should be set when more items exist")
			}
		})

		It("should return next page when using nextPageToken", func() {
			By("creating a source preview with pageSize=5 to get first page")
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			requestBody := struct {
				Data models.CatalogSourcePreviewRequest `json:"data"`
			}{
				Data: models.CatalogSourcePreviewRequest{
					Type:           "yaml",
					IncludedModels: []string{},
					ExcludedModels: []string{},
					Properties: map[string]interface{}{
						"yaml": "models:\n  - name: test-model",
					},
				},
			}
			bodyBytes, _ := json.Marshal(requestBody)

			firstPage, rs, err := setupApiTest[CatalogSourcePreviewEnvelope](http.MethodPost, "/api/v1/settings/model_catalog/source_preview?namespace=kubeflow&pageSize=5", bytes.NewReader(bodyBytes), kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusOK))

			// If there's a next page token, fetch the next page
			if firstPage.Data.NextPageToken != "" {
				By("fetching next page using nextPageToken")
				bodyBytes2, _ := json.Marshal(requestBody)
				secondPage, rs2, err2 := setupApiTest[CatalogSourcePreviewEnvelope](http.MethodPost, "/api/v1/settings/model_catalog/source_preview?namespace=kubeflow&pageSize=5&nextPageToken="+firstPage.Data.NextPageToken, bytes.NewReader(bodyBytes2), kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
				Expect(err2).NotTo(HaveOccurred())
				Expect(rs2.StatusCode).To(Equal(http.StatusOK))

				By("should return different items on second page")
				// Verify that the items are different (not overlapping)
				if len(firstPage.Data.Items) > 0 && len(secondPage.Data.Items) > 0 {
					Expect(firstPage.Data.Items[0].Name).NotTo(Equal(secondPage.Data.Items[0].Name), "First item on second page should be different from first page")
				}
			}
		})
	})
})
