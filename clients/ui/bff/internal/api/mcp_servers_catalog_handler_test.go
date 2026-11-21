package api

import (
	"net/http"

	"github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TestMcpServersCatalogHandler", func() {
	Context("testing MCP Servers Catalog Handler", Ordered, func() {
		It("should retrieve all MCP servers", func() {
			By("fetching all MCP servers")
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			actual, rs, err := setupApiTest[McpServerListEnvelope](http.MethodGet, "/api/v1/mcp_catalog/mcp_servers?namespace=kubeflow", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should match the expected MCP server list")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data.Size).To(Equal(int32(10)))
			Expect(actual.Data.PageSize).To(Equal(int32(10)))
			Expect(actual.Data.NextPageToken).NotTo(BeEmpty())
			Expect(len(actual.Data.Items)).To(Equal(10))
		})

		It("should retrieve MCP server filter options", func() {
			By("fetching MCP server filter options")
			data := mocks.GetMcpFilterOptionsListMock()
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			expected := McpServerFilterOptionsListEnvelope{Data: &data}
			actual, rs, err := setupApiTest[McpServerFilterOptionsListEnvelope](http.MethodGet, "/api/v1/mcp_catalog/mcp_servers_filter_options?namespace=kubeflow", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should match the expected filter options")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data).NotTo(BeNil())
			Expect(actual.Data).To(Equal(expected.Data))
		})

		It("should retrieve a single MCP server by id", func() {
			By("fetching MCP server by server_id")
			data := mocks.GetMcpServerMocks()[0]
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			actual, rs, err := setupApiTest[McpServerEnvelope](http.MethodGet, "/api/v1/mcp_catalog/mcp_servers/1?namespace=kubeflow", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should match the expected MCP server")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data).NotTo(BeNil())
			Expect(actual.Data.Name).To(Equal(data.Name))
			Expect(actual.Data.ID).To(Equal(data.ID))
		})

		It("should retrieve MCP server tools", func() {
			By("fetching MCP server tools")
			data := mocks.GetMcpToolListMock()
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			expected := McpServerToolsListEnvelope{Data: &data}
			actual, rs, err := setupApiTest[McpServerToolsListEnvelope](http.MethodGet, "/api/v1/mcp_catalog/mcp_servers/1/tools?namespace=kubeflow", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should match the expected tool list")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data).NotTo(BeNil())
			Expect(actual.Data.Size).To(Equal(expected.Data.Size))
			Expect(len(actual.Data.Items)).To(Equal(len(expected.Data.Items)))
		})

		It("should retrieve the MCP server logo image", func() {
			By("fetching the MCP server logo by server_id")
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			rs, body, err := serveApiTest(http.MethodGet, "/api/v1/mcp_catalog/mcp_servers/1/logo?namespace=kubeflow", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should return the raw logo image bytes with an image content type")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(rs.Header.Get("Content-Type")).To(HavePrefix("image/"))
			Expect(rs.Header.Get("Content-Disposition")).To(Equal("inline"))
			Expect(len(body)).To(BeNumerically(">", 0))

			By("should preserve the catalog's protective headers so re-serving under the dashboard origin does not re-open stored XSS on SVG logos")
			Expect(rs.Header.Get("X-Content-Type-Options")).To(Equal("nosniff"))
			Expect(rs.Header.Get("Content-Security-Policy")).To(Equal("default-src 'none'; style-src 'unsafe-inline'; sandbox"))
		})

		It("should translate an upstream logo error into a structured JSON error", func() {
			By("fetching a logo whose catalog response is a non-2xx status")
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			rs, body, err := serveApiTest(http.MethodGet, "/api/v1/mcp_catalog/mcp_servers/missing/logo?namespace=kubeflow", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should return the upstream status as a structured JSON error, not the raw plaintext body")
			Expect(rs.StatusCode).To(Equal(http.StatusNotFound))
			Expect(rs.Header.Get("Content-Type")).To(Equal("application/json"))
			Expect(string(body)).To(ContainSubstring(`"error"`))
			Expect(string(body)).NotTo(Equal("404 page not found"))
		})

		It("should clear status successfully", func() {
			By("clearing the MCP catalog source status")
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			actual, rs, err := setupApiTest[Envelope[any, any]](http.MethodDelete, "/api/v1/settings/mcp_catalog/source_config/community_mcp_servers/status?namespace=kubeflow", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data).To(BeNil())
		})

		It("should deny clearing status without update permission", func() {
			By("clearing status as a user without MCP source update permission")
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "bellaNonAdmin@example.com",
			}

			_, rs, err := setupApiTest[ErrorEnvelope](http.MethodDelete, "/api/v1/settings/mcp_catalog/source_config/community_mcp_servers/status?namespace=bella-namespace", nil, kubernetesMockedStaticClientFactory, requestIdentity, "bella-namespace")
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusForbidden))
		})

		It("should fail to clear status for a non-existent source", func() {
			By("clearing status for a source that does not exist")
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			_, rs, err := setupApiTest[ErrorEnvelope](http.MethodDelete, "/api/v1/settings/mcp_catalog/source_config/does_not_exist/status?namespace=kubeflow", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusNotFound))
		})
	})
})
