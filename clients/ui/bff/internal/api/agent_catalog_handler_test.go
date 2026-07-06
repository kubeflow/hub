package api

import (
	"net/http"

	"github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TestAgentCatalogHandler", func() {
	Context("testing Agent Catalog Handler", Ordered, func() {

		It("should retrieve all agents with pagination", func() {
			By("fetching all agents with default page size")
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			actual, rs, err := setupApiTest[AgentListEnvelope](http.MethodGet, "/api/v1/agent_catalog/agents?namespace=kubeflow", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should return first page of agents")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data.Size).To(Equal(int32(10)))
			Expect(actual.Data.PageSize).To(Equal(int32(10)))
			Expect(len(actual.Data.Items)).To(Equal(10))
			Expect(actual.Data.NextPageToken).NotTo(BeEmpty())
		})

		It("should retrieve second page of agents", func() {
			By("fetching agents with nextPageToken")
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			actual, rs, err := setupApiTest[AgentListEnvelope](http.MethodGet, "/api/v1/agent_catalog/agents?namespace=kubeflow&nextPageToken=10", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should return remaining agents")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data.Size).To(Equal(int32(2)))
			Expect(len(actual.Data.Items)).To(Equal(2))
			Expect(actual.Data.NextPageToken).To(BeEmpty())
		})

		It("should retrieve agent filter options", func() {
			By("fetching agent filter options")
			data := mocks.GetAgentFilterOptionsListMock()
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			expected := AgentFilterOptionsListEnvelope{Data: &data}
			actual, rs, err := setupApiTest[AgentFilterOptionsListEnvelope](http.MethodGet, "/api/v1/agent_catalog/agents_filter_options?namespace=kubeflow", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should match the expected filter options")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data).NotTo(BeNil())
			Expect(actual.Data).To(Equal(expected.Data))
		})

		It("should retrieve a single agent by id", func() {
			By("fetching agent by agent_id")
			data := mocks.GetAgentMocks()[0]
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			actual, rs, err := setupApiTest[AgentEnvelope](http.MethodGet, "/api/v1/agent_catalog/agents/1?namespace=kubeflow", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should match the expected agent")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data).NotTo(BeNil())
			Expect(actual.Data.Name).To(Equal(data.Name))
			Expect(actual.Data.ID).To(Equal(data.ID))
		})

		It("should return 404 when agent id does not exist", func() {
			By("fetching a non-existent agent")
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			_, rs, err := setupApiTest[ErrorEnvelope](http.MethodGet, "/api/v1/agent_catalog/agents/non-existent-id?namespace=kubeflow", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should return not found")
			Expect(rs.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("should return 400 when namespace is missing for agent list", func() {
			By("fetching agents without namespace")
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			_, rs, err := setupApiTest[ErrorEnvelope](http.MethodGet, "/api/v1/agent_catalog/agents", nil, kubernetesMockedStaticClientFactory, requestIdentity, "")
			Expect(err).NotTo(HaveOccurred())

			By("should return bad request")
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("should return 400 when namespace is missing for filter options", func() {
			By("fetching filter options without namespace")
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			_, rs, err := setupApiTest[ErrorEnvelope](http.MethodGet, "/api/v1/agent_catalog/agents_filter_options", nil, kubernetesMockedStaticClientFactory, requestIdentity, "")
			Expect(err).NotTo(HaveOccurred())

			By("should return bad request")
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("should return 400 when namespace is missing for single agent", func() {
			By("fetching single agent without namespace")
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			_, rs, err := setupApiTest[ErrorEnvelope](http.MethodGet, "/api/v1/agent_catalog/agents/1", nil, kubernetesMockedStaticClientFactory, requestIdentity, "")
			Expect(err).NotTo(HaveOccurred())

			By("should return bad request")
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
		})
	})
})
