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

		It("should retrieve all agents", func() {
			By("fetching all agents")
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			actual, rs, err := setupApiTest[AgentListEnvelope](http.MethodGet, "/api/v1/agent_catalog/agents?namespace=kubeflow", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should match the expected agent list")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data.Size).To(Equal(int32(4)))
			Expect(actual.Data.PageSize).To(Equal(int32(10)))
			Expect(len(actual.Data.Items)).To(Equal(4))
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
	})
})
