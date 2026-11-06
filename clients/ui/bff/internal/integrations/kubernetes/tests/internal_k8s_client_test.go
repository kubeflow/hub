package tests

import (
	"context"
	"errors"

	"github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	mocks2 "github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes/k8mocks"
	"github.com/kubeflow/hub/ui/bff/internal/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

var _ = Describe("Kubernetes Internal Client Test", func() {
	Context("with existing services", Ordered, func() {

		It("should retrieve the get all service successfully", func() {

			By("getting service details")
			serviceAccountMockedK8client, err := kubernetesMockedStaticClientFactory.GetClient(mocks.NewMockSessionContextNoParent())
			Expect(err).NotTo(HaveOccurred())

			services, err := serviceAccountMockedK8client.GetServiceDetails(mocks.NewMockSessionContextNoParent(), "kubeflow")
			Expect(err).NotTo(HaveOccurred(), "Failed to create HTTP request")

			By("checking that all services have the modified ClusterIP and HTTPPort")
			for _, service := range services {
				Expect(service.ClusterIP).To(Equal("127.0.0.1"), "ClusterIP should be set to 127.0.0.1")
				Expect(service.HTTPPort).To(Equal(int32(8080)), "HTTPPort should be set to 8080")

			}

			By("checking that that a specific service exists")
			foundService := false
			for _, service := range services {
				if service.Name == "model-registry" {
					foundService = true
					Expect(service.DisplayName).To(Equal("Model Registry"))
					Expect(service.Description).To(Equal("Model Registry Description"))
					break
				}
			}
			Expect(foundService).To(Equal(true), "Expected to find service 'model-registry'")
		})

		It("should retrieve the service details by name", func() {

			By("getting service by name")
			serviceAccountMockedK8client, err := kubernetesMockedStaticClientFactory.GetClient(mocks.NewMockSessionContextNoParent())
			Expect(err).NotTo(HaveOccurred())

			service, err := serviceAccountMockedK8client.GetServiceDetailsByName(mocks.NewMockSessionContextNoParent(), "dora-namespace", "model-registry-dora", kubernetes.ComponentLabelValue)
			Expect(err).NotTo(HaveOccurred(), "Failed to create k8s request")

			By("checking that service details are correct")
			Expect(service.Name).To(Equal("model-registry-dora"))
			Expect(service.Description).To(Equal("Model Registry Dora description"))
			Expect(service.DisplayName).To(Equal("Model Registry Dora"))
		})

		It("should retrieve the services names", func() {

			By("getting service by name")
			serviceAccountMockedK8client, err := kubernetesMockedStaticClientFactory.GetClient(mocks.NewMockSessionContextNoParent())
			Expect(err).NotTo(HaveOccurred())

			services, err := serviceAccountMockedK8client.GetServiceNames(mocks.NewMockSessionContextNoParent(), "kubeflow")
			Expect(err).NotTo(HaveOccurred(), "Failed to create HTTP request")

			By("checking that service details are correct")
			Expect(services).To(ConsistOf("model-registry", "model-registry-one"))

		})
	})

})

var _ = Describe("KubernetesClient Internal SAR Test", func() {
	Context("CanUpdateMcpCatalogSources", func() {
		It("should allow a user with MCP source update permission", func() {
			identity := &kubernetes.RequestIdentity{
				UserID: mocks2.DefaultTestUsers[0].UserName,
				Groups: []string{},
			}
			serviceAccountMockedK8client, err := kubernetesMockedStaticClientFactory.GetClient(mocks.NewMockSessionContextNoParent())
			Expect(err).NotTo(HaveOccurred())

			allowed, err := serviceAccountMockedK8client.CanUpdateMcpCatalogSources(ctx, identity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())
			Expect(allowed).To(BeTrue())
		})

		It("should deny a user without MCP source update permission", func() {
			identity := &kubernetes.RequestIdentity{
				UserID: mocks2.DefaultTestUsers[1].UserName,
				Groups: mocks2.DefaultTestUsers[1].Groups,
			}
			serviceAccountMockedK8client, err := kubernetesMockedStaticClientFactory.GetClient(mocks.NewMockSessionContextNoParent())
			Expect(err).NotTo(HaveOccurred())

			allowed, err := serviceAccountMockedK8client.CanUpdateMcpCatalogSources(ctx, identity, "dora-namespace")
			Expect(err).NotTo(HaveOccurred())
			Expect(allowed).To(BeFalse())
		})

		It("should propagate an access-review API failure", func() {
			expectedErr := errors.New("Kubernetes API unavailable")
			fakeClientset := fake.NewClientset()
			fakeClientset.PrependReactor("create", "subjectaccessreviews", func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, expectedErr
			})
			internalK8client := &kubernetes.InternalKubernetesClient{
				SharedClientLogic: kubernetes.SharedClientLogic{
					Client: fakeClientset,
				},
			}
			identity := &kubernetes.RequestIdentity{UserID: mocks2.DefaultTestUsers[0].UserName}

			allowed, err := internalK8client.CanUpdateMcpCatalogSources(context.Background(), identity, "kubeflow")
			Expect(allowed).To(BeFalse())
			Expect(err).To(MatchError(ContainSubstring("checking MCP source update permission")))
			Expect(errors.Is(err, expectedErr)).To(BeTrue())
		})
	})

	Context("CanListServicesInNamespace", func() {

		It("should allow allowed user to access services", func() {
			By("should allow allowed user to access services")

			identity := &kubernetes.RequestIdentity{
				UserID: mocks2.DefaultTestUsers[0].UserName,
				Groups: []string{},
			}
			serviceAccountMockedK8client, err := kubernetesMockedStaticClientFactory.GetClient(mocks.NewMockSessionContextNoParent())
			Expect(err).NotTo(HaveOccurred())

			allowed, err := serviceAccountMockedK8client.CanListServicesInNamespace(ctx, identity, "kubeflow")
			Expect(err).NotTo(HaveOccurred(), "Failed to perform SAR for Kubeflow User ID\"")
			Expect(allowed).To(BeTrue(), "Expected Kubeflow User ID to have access")
		})

		It("should check dora's access to namespaces", func() {
			By("performing SAR for dora user")
			identity := &kubernetes.RequestIdentity{
				UserID: mocks2.DefaultTestUsers[1].UserName,
				Groups: []string{},
			}
			serviceAccountMockedK8client, err := kubernetesMockedStaticClientFactory.GetClient(mocks.NewMockSessionContextNoParent())
			Expect(err).NotTo(HaveOccurred())

			allowed, err := serviceAccountMockedK8client.CanListServicesInNamespace(ctx, identity, "kubeflow")
			Expect(err).NotTo(HaveOccurred(), "Failed to perform SAR for unauthorized-dora@example.com")
			Expect(allowed).To(BeFalse(), "Expected doraNonAdmin@example.com to be denied access")

			allowed, err = serviceAccountMockedK8client.CanListServicesInNamespace(ctx, identity, "dora-namespace")
			Expect(err).NotTo(HaveOccurred(), "Failed to perform SAR for unauthorized-dora@example.com")
			Expect(allowed).To(BeTrue(), "Expected doraNonAdmin@example.com ID to have access")
		})

		It("should deny access for another user", func() {
			By("performing SAR for another user")
			identity := &kubernetes.RequestIdentity{
				UserID: "unauthorized-dora@example.com",
				Groups: []string{},
			}
			serviceAccountMockedK8client, err := kubernetesMockedStaticClientFactory.GetClient(mocks.NewMockSessionContextNoParent())
			Expect(err).NotTo(HaveOccurred())

			allowed, err := serviceAccountMockedK8client.CanListServicesInNamespace(ctx, identity, "kubeflow")
			Expect(err).NotTo(HaveOccurred(), "Failed to perform SAR for unauthorized-dora@example.com")
			Expect(allowed).To(BeFalse(), "Expected unauthorized-dora@example.com to be denied access")
		})

	})
})

var _ = Describe("KubernetesClient Internal Permission Checks", func() {
	Context("checking access using group memberships", func() {
		const (
			namespace    = "dora-namespace"
			serviceName  = "model-registry-dora"
			existingUser = "bentoOnlyGroupAccess@example.com"
		)

		It("should deny access for a group that does not exist", func() {
			identity := &kubernetes.RequestIdentity{
				UserID: existingUser,
				Groups: []string{"non-existent-group"},
			}
			serviceAccountMockedK8client, err := kubernetesMockedStaticClientFactory.GetClient(mocks.NewMockSessionContextNoParent())
			Expect(err).NotTo(HaveOccurred())

			allowed, err := serviceAccountMockedK8client.CanAccessServiceInNamespace(ctx, identity, namespace, serviceName)
			Expect(err).NotTo(HaveOccurred())
			Expect(allowed).To(BeFalse(), "Access should be denied for a non-existent group")
		})

		It("should allow service access for the DoraServiceGroup", func() {
			identity := &kubernetes.RequestIdentity{
				UserID: existingUser,
				Groups: []string{mocks2.DefaultTestUsers[1].Groups[1]},
			}
			serviceAccountMockedK8client, err := kubernetesMockedStaticClientFactory.GetClient(mocks.NewMockSessionContextNoParent())
			Expect(err).NotTo(HaveOccurred())

			allowed, err := serviceAccountMockedK8client.CanAccessServiceInNamespace(ctx, identity, namespace, serviceName)
			Expect(err).NotTo(HaveOccurred())
			Expect(allowed).To(BeTrue(), "Access should be allowed for the DoraServiceGroup group")
		})

		It("should allow access when one group exists and the other does not", func() {
			identity := &kubernetes.RequestIdentity{
				UserID: existingUser,
				Groups: []string{mocks2.DefaultTestUsers[1].Groups[1], "non-existent-group"},
			}
			serviceAccountMockedK8client, err := kubernetesMockedStaticClientFactory.GetClient(mocks.NewMockSessionContextNoParent())
			Expect(err).NotTo(HaveOccurred())

			allowed, err := serviceAccountMockedK8client.CanAccessServiceInNamespace(ctx, identity, namespace, serviceName)
			Expect(err).NotTo(HaveOccurred())
			Expect(allowed).To(BeTrue(), "Access should be allowed if any group in the list has access")
		})

		It("should allow access only when both service and namespace groups are present\"", func() {
			serviceOnly := &kubernetes.RequestIdentity{
				UserID: existingUser,
				Groups: []string{mocks2.DefaultTestUsers[1].Groups[1]},
			}
			bothGroups := &kubernetes.RequestIdentity{
				UserID: existingUser,
				Groups: mocks2.DefaultTestUsers[1].Groups,
			}
			serviceAccountMockedK8client, err := kubernetesMockedStaticClientFactory.GetClient(mocks.NewMockSessionContextNoParent())
			Expect(err).NotTo(HaveOccurred())

			allowed, err := serviceAccountMockedK8client.CanAccessServiceInNamespace(ctx, serviceOnly, namespace, serviceName)
			Expect(err).NotTo(HaveOccurred())
			Expect(allowed).To(BeTrue(), "Access should be allowed for the DoraServiceGroup group")

			allowed, err = serviceAccountMockedK8client.CanListServicesInNamespace(ctx, serviceOnly, namespace)
			Expect(err).NotTo(HaveOccurred())
			Expect(allowed).To(BeFalse(), "Access should not be allowed for only DoraServiceGroup group")

			allowed, err = serviceAccountMockedK8client.CanListServicesInNamespace(ctx, bothGroups, namespace)
			Expect(err).NotTo(HaveOccurred())
			Expect(allowed).To(BeTrue(), "Access should be allowed for both groups")

			allowed, err = serviceAccountMockedK8client.CanAccessServiceInNamespace(ctx, bothGroups, namespace, serviceName)
			Expect(err).NotTo(HaveOccurred())
			Expect(allowed).To(BeTrue(), "Access should be allowed for for both groups")

		})
	})
})
