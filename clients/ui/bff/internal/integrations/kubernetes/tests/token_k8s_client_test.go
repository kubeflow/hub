package tests

import (
	"context"
	"errors"

	"github.com/kubeflow/hub/ui/bff/internal/constants"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	k8mocks "github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes/k8mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

var _ = Describe("KubernetesClient SSAR Test", func() {
	Context("CanListServicesInNamespace", func() {

		It("should allow allowed user to access services", func() {
			By("should allow allowed user to access services")

			identity := &kubernetes.RequestIdentity{
				Token: k8mocks.DefaultTestUsers[0].Token,
			}
			ctx := context.WithValue(context.Background(), constants.RequestIdentityKey, identity)

			kubernetesMockedTokenClientFactory, err := k8mocks.NewTokenClientFactory(clientset, restConfig, logger)
			Expect(err).NotTo(HaveOccurred())
			tokenK8client, err := kubernetesMockedTokenClientFactory.GetClient(ctx)
			Expect(err).NotTo(HaveOccurred())

			allowed, err := tokenK8client.CanListServicesInNamespace(ctx, identity, "kubeflow")
			Expect(err).NotTo(HaveOccurred(), "Failed to perform SSAR for Kubeflow User ID\"")
			Expect(allowed).To(BeTrue(), "Expected Kubeflow User ID to have access")
		})

		It("should check dora's access to namespaces", func() {
			By("performing SAR for dora user")
			identity := &kubernetes.RequestIdentity{
				Token: k8mocks.DefaultTestUsers[1].Token,
			}
			ctx := context.WithValue(context.Background(), constants.RequestIdentityKey, identity)

			kubernetesMockedTokenClientFactory, err := k8mocks.NewTokenClientFactory(clientset, restConfig, logger)
			Expect(err).NotTo(HaveOccurred())
			tokenK8client, err := kubernetesMockedTokenClientFactory.GetClient(ctx)
			Expect(err).NotTo(HaveOccurred())

			allowed, err := tokenK8client.CanListServicesInNamespace(ctx, identity, "kubeflow")
			Expect(err).NotTo(HaveOccurred(), "Failed to perform SAR for unauthorized-dora@example.com")
			Expect(allowed).To(BeFalse(), "Expected doraNonAdmin@example.com to be denied access")

			allowed, err = tokenK8client.CanListServicesInNamespace(ctx, identity, "dora-namespace")
			Expect(err).NotTo(HaveOccurred(), "Failed to perform SSAR for unauthorized-dora@example.com")
			Expect(allowed).To(BeTrue(), "Expected doraNonAdmin@example.com ID to have access")
		})

		It("should deny access for another user", func() {
			By("performing SAR for another user")
			identity := &kubernetes.RequestIdentity{
				Token: k8mocks.DefaultTestUsers[2].Token,
			}
			ctx := context.WithValue(context.Background(), constants.RequestIdentityKey, identity)

			kubernetesMockedTokenClientFactory, err := k8mocks.NewTokenClientFactory(clientset, restConfig, logger)
			Expect(err).NotTo(HaveOccurred())
			tokenK8client, err := kubernetesMockedTokenClientFactory.GetClient(ctx)
			Expect(err).NotTo(HaveOccurred())

			allowed, err := tokenK8client.CanListServicesInNamespace(ctx, identity, "kubeflow")
			Expect(err).NotTo(HaveOccurred(), "Failed to perform SSAR for unauthorized-dora@example.com")
			Expect(allowed).To(BeFalse(), "Expected unauthorized-dora@example.com to be denied access")
		})

	})

	Context("CanUpdateMcpCatalogSources", func() {
		It("should allow a user with MCP source update permission", func() {
			identity := &kubernetes.RequestIdentity{
				Token: k8mocks.DefaultTestUsers[0].Token,
			}
			ctx := context.WithValue(context.Background(), constants.RequestIdentityKey, identity)

			kubernetesMockedTokenClientFactory, err := k8mocks.NewTokenClientFactory(clientset, restConfig, logger)
			Expect(err).NotTo(HaveOccurred())
			tokenK8client, err := kubernetesMockedTokenClientFactory.GetClient(ctx)
			Expect(err).NotTo(HaveOccurred())

			allowed, err := tokenK8client.CanUpdateMcpCatalogSources(ctx, identity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())
			Expect(allowed).To(BeTrue())
		})

		It("should deny a user without MCP source update permission", func() {
			identity := &kubernetes.RequestIdentity{
				Token: k8mocks.DefaultTestUsers[1].Token,
			}
			ctx := context.WithValue(context.Background(), constants.RequestIdentityKey, identity)

			kubernetesMockedTokenClientFactory, err := k8mocks.NewTokenClientFactory(clientset, restConfig, logger)
			Expect(err).NotTo(HaveOccurred())
			tokenK8client, err := kubernetesMockedTokenClientFactory.GetClient(ctx)
			Expect(err).NotTo(HaveOccurred())

			allowed, err := tokenK8client.CanUpdateMcpCatalogSources(ctx, identity, "dora-namespace")
			Expect(err).NotTo(HaveOccurred())
			Expect(allowed).To(BeFalse())
		})

		It("should propagate an access-review API failure", func() {
			expectedErr := errors.New("Kubernetes API unavailable")
			fakeClientset := fake.NewClientset()
			fakeClientset.PrependReactor("create", "selfsubjectaccessreviews", func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, expectedErr
			})
			tokenK8client := &kubernetes.TokenKubernetesClient{
				SharedClientLogic: kubernetes.SharedClientLogic{
					Client: fakeClientset,
				},
			}

			allowed, err := tokenK8client.CanUpdateMcpCatalogSources(context.Background(), nil, "kubeflow")
			Expect(allowed).To(BeFalse())
			Expect(err).To(MatchError(ContainSubstring("checking MCP source update permission")))
			Expect(errors.Is(err, expectedErr)).To(BeTrue())
		})
	})

	Context("CanAccessServiceInNamespace", func() {

		It("should allow allowed user to access service", func() {
			By("should allow allowed user to access service")

			identity := &kubernetes.RequestIdentity{
				Token: k8mocks.DefaultTestUsers[0].Token,
			}
			ctx := context.WithValue(context.Background(), constants.RequestIdentityKey, identity)

			kubernetesMockedTokenClientFactory, err := k8mocks.NewTokenClientFactory(clientset, restConfig, logger)
			Expect(err).NotTo(HaveOccurred())
			tokenK8client, err := kubernetesMockedTokenClientFactory.GetClient(ctx)
			Expect(err).NotTo(HaveOccurred())

			allowed, err := tokenK8client.CanAccessServiceInNamespace(ctx, identity, "kubeflow", "model-registry")
			Expect(err).NotTo(HaveOccurred(), "Failed to perform SSAR for Kubeflow User ID\"")
			Expect(allowed).To(BeTrue(), "Expected Kubeflow User ID to have access")
		})

		It("should check dora's access to services", func() {
			By("performing SAR for dora user")
			identity := &kubernetes.RequestIdentity{
				Token: k8mocks.DefaultTestUsers[1].Token,
			}
			ctx := context.WithValue(context.Background(), constants.RequestIdentityKey, identity)

			kubernetesMockedTokenClientFactory, err := k8mocks.NewTokenClientFactory(clientset, restConfig, logger)
			Expect(err).NotTo(HaveOccurred())
			tokenK8client, err := kubernetesMockedTokenClientFactory.GetClient(ctx)
			Expect(err).NotTo(HaveOccurred())

			allowed, err := tokenK8client.CanAccessServiceInNamespace(ctx, identity, "dora-namespace", "model-registry-dora")
			Expect(err).NotTo(HaveOccurred(), "Failed to perform SSAR")
			Expect(allowed).To(BeTrue())

			allowed, err = tokenK8client.CanAccessServiceInNamespace(ctx, identity, "bella-namespace", "model-registry-bella")
			Expect(err).NotTo(HaveOccurred())
			Expect(allowed).To(BeFalse())
		})

	})

	Context("CanNamespaceAccessRegistry", func() {
		It("should return true when namespace default SA has access to the registry", func() {
			identity := &kubernetes.RequestIdentity{
				Token: k8mocks.DefaultTestUsers[0].Token,
			}
			ctx := context.WithValue(context.Background(), constants.RequestIdentityKey, identity)

			kubernetesMockedTokenClientFactory, err := k8mocks.NewTokenClientFactory(clientset, restConfig, logger)
			Expect(err).NotTo(HaveOccurred())
			tokenK8client, err := kubernetesMockedTokenClientFactory.GetClient(ctx)
			Expect(err).NotTo(HaveOccurred())

			allowed, err := tokenK8client.CanNamespaceAccessRegistry(ctx, identity, "dora-namespace", "model-registry-dora", "dora-namespace")
			Expect(err).NotTo(HaveOccurred())
			Expect(allowed).To(BeTrue())
		})

		It("should return false when namespace default SA has no access to the registry", func() {
			identity := &kubernetes.RequestIdentity{
				Token: k8mocks.DefaultTestUsers[0].Token,
			}
			ctx := context.WithValue(context.Background(), constants.RequestIdentityKey, identity)

			kubernetesMockedTokenClientFactory, err := k8mocks.NewTokenClientFactory(clientset, restConfig, logger)
			Expect(err).NotTo(HaveOccurred())
			tokenK8client, err := kubernetesMockedTokenClientFactory.GetClient(ctx)
			Expect(err).NotTo(HaveOccurred())

			allowed, err := tokenK8client.CanNamespaceAccessRegistry(ctx, identity, "bella-namespace", "model-registry-dora", "dora-namespace")
			Expect(err).NotTo(HaveOccurred())
			Expect(allowed).To(BeFalse())
		})
	})

	Context("GetNamespaces", func() {

		It("should allow allowed user to get namespaces", func() {
			By("cluster admin should be allowed to get namespaces")

			identity := &kubernetes.RequestIdentity{
				Token: k8mocks.DefaultTestUsers[0].Token,
			}
			ctx := context.WithValue(context.Background(), constants.RequestIdentityKey, identity)

			kubernetesMockedTokenClientFactory, err := k8mocks.NewTokenClientFactory(clientset, restConfig, logger)
			Expect(err).NotTo(HaveOccurred())
			tokenK8client, err := kubernetesMockedTokenClientFactory.GetClient(ctx)
			Expect(err).NotTo(HaveOccurred())

			ns, err := tokenK8client.GetNamespaces(ctx, identity)
			Expect(err).NotTo(HaveOccurred(), "Failed to perform SSAR for Kubeflow User ID\"")
			Expect(ns).NotTo(BeEmpty(), "Expected Kubeflow User ID to have access")
		})

		It("should allow allowed user to get namespaces", func() {
			By("other users should not be allowed to get namespaces")

			identity := &kubernetes.RequestIdentity{
				Token: k8mocks.DefaultTestUsers[1].Token,
			}
			ctx := context.WithValue(context.Background(), constants.RequestIdentityKey, identity)

			kubernetesMockedTokenClientFactory, err := k8mocks.NewTokenClientFactory(clientset, restConfig, logger)
			Expect(err).NotTo(HaveOccurred())
			tokenK8client, err := kubernetesMockedTokenClientFactory.GetClient(ctx)
			Expect(err).NotTo(HaveOccurred())

			ns, err := tokenK8client.GetNamespaces(ctx, identity)
			Expect(ns).To(BeEmpty())
			Expect(err).To(HaveOccurred())
		})
	})
})
