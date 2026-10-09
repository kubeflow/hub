package repositories

import (
	"context"

	"github.com/kubeflow/hub/ui/bff/internal/constants"
	k8s "github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/mocks"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TestFetchAllModelRegistry", func() {
	Context("with existing model registries", Ordered, func() {

		It("should retrieve the get all kubeflow service successfully", func() {

			By("fetching all model registries in the repository")
			modelRegistryRepository := NewModelRegistryRepository()
			serviceAccountMockedK8client, err := kubernetesMockedStaticClientFactory.GetClient(mocks.NewMockSessionContextNoParent())
			Expect(err).NotTo(HaveOccurred())

			registries, err := modelRegistryRepository.GetAllModelRegistries(mocks.NewMockSessionContextNoParent(), serviceAccountMockedK8client, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should match the expected model registries (model-registry has ready Endpoints in test env, model-registry-one has none)")
			expectedRegistries := []models.ModelRegistryModel{
				{Name: "model-registry", Description: "Model Registry Description", DisplayName: "Model Registry", ServerAddress: "http://127.0.0.1:8080/api/model_registry/v1", IsAvailable: true},
				{Name: "model-registry-one", Description: "Model Registry One description", DisplayName: "Model Registry One", ServerAddress: "http://127.0.0.1:8080/api/model_registry/v1", IsAvailable: false},
			}
			Expect(registries).To(ConsistOf(expectedRegistries))
		})

		It("should retrieve the get all dora-namespace service successfully", func() {

			By("fetching all model registries in the repository")
			modelRegistryRepository := NewModelRegistryRepository()
			serviceAccountMockedK8client, err := kubernetesMockedStaticClientFactory.GetClient(mocks.NewMockSessionContextNoParent())
			Expect(err).NotTo(HaveOccurred())

			registries, err := modelRegistryRepository.GetAllModelRegistries(mocks.NewMockSessionContextNoParent(), serviceAccountMockedK8client, "dora-namespace")
			Expect(err).NotTo(HaveOccurred())

			By("should match the expected model registries (dora-namespace has no Endpoints in test env, so we assume available per RBAC fallback)")
			expectedRegistries := []models.ModelRegistryModel{
				{Name: "model-registry-dora", Description: "Model Registry Dora description", DisplayName: "Model Registry Dora", ServerAddress: "http://127.0.0.1:8080/api/model_registry/v1", IsAvailable: true},
			}
			Expect(registries).To(ConsistOf(expectedRegistries))
		})

		It("should not retrieve namespaces", func() {

			By("fetching all model registries in the repository")
			modelRegistryRepository := NewModelRegistryRepository()
			serviceAccountMockedK8client, err := kubernetesMockedStaticClientFactory.GetClient(mocks.NewMockSessionContextNoParent())
			Expect(err).NotTo(HaveOccurred())

			registries, err := modelRegistryRepository.GetAllModelRegistries(mocks.NewMockSessionContextNoParent(), serviceAccountMockedK8client, "no-namespace")
			Expect(err).NotTo(HaveOccurred())

			By("should be empty")
			Expect(registries).To(BeEmpty())
		})
	})

	Context("with authorization context", func() {
		var modelRegistryRepository *ModelRegistryRepository
		var serviceAccountMockedK8client k8s.KubernetesClientInterface

		BeforeEach(func() {
			modelRegistryRepository = NewModelRegistryRepository()
			var err error
			serviceAccountMockedK8client, err = kubernetesMockedStaticClientFactory.GetClient(mocks.NewMockSessionContextNoParent())
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fetch all services when AllowList is true", func() {
			By("creating context with AllowList authorization")
			authCtx := &models.ServiceAuthorizationContext{
				AllowList:           true,
				AllowedServiceNames: []string{}, // Empty since AllowList=true means access to all
				Namespace:           "kubeflow",
			}
			ctx := context.WithValue(mocks.NewMockSessionContextNoParent(), constants.ServiceAuthorizationContextKey, authCtx)

			By("fetching all model registries")
			registries, err := modelRegistryRepository.GetAllModelRegistriesWithMode(ctx, serviceAccountMockedK8client, "kubeflow", false)
			Expect(err).NotTo(HaveOccurred())

			By("should return all available model registries in the namespace")
			expectedRegistries := []models.ModelRegistryModel{
				{Name: "model-registry", Description: "Model Registry Description", DisplayName: "Model Registry", ServerAddress: "http://127.0.0.1:8080/api/model_registry/v1", IsAvailable: true},
				{Name: "model-registry-one", Description: "Model Registry One description", DisplayName: "Model Registry One", ServerAddress: "http://127.0.0.1:8080/api/model_registry/v1", IsAvailable: false},
			}
			Expect(registries).To(ConsistOf(expectedRegistries))
		})

		It("should fetch only specific services when AllowList is false", func() {
			By("creating context with specific allowed services")
			authCtx := &models.ServiceAuthorizationContext{
				AllowList:           false,
				AllowedServiceNames: []string{"model-registry"}, // Only allow access to specific service
				Namespace:           "kubeflow",
			}
			ctx := context.WithValue(mocks.NewMockSessionContextNoParent(), constants.ServiceAuthorizationContextKey, authCtx)

			By("fetching model registries with restricted access")
			registries, err := modelRegistryRepository.GetAllModelRegistriesWithMode(ctx, serviceAccountMockedK8client, "kubeflow", false)
			Expect(err).NotTo(HaveOccurred())

			By("should return only the allowed service")
			expectedRegistries := []models.ModelRegistryModel{
				{Name: "model-registry", Description: "Model Registry Description", DisplayName: "Model Registry", ServerAddress: "http://127.0.0.1:8080/api/model_registry/v1", IsAvailable: true},
			}
			Expect(registries).To(ConsistOf(expectedRegistries))
		})

		It("should fetch multiple specific services when AllowList is false", func() {
			By("creating context with multiple allowed services")
			authCtx := &models.ServiceAuthorizationContext{
				AllowList:           false,
				AllowedServiceNames: []string{"model-registry", "model-registry-one"},
				Namespace:           "kubeflow",
			}
			ctx := context.WithValue(mocks.NewMockSessionContextNoParent(), constants.ServiceAuthorizationContextKey, authCtx)

			By("fetching model registries with multiple allowed services")
			registries, err := modelRegistryRepository.GetAllModelRegistriesWithMode(ctx, serviceAccountMockedK8client, "kubeflow", false)
			Expect(err).NotTo(HaveOccurred())

			By("should return all allowed services")
			expectedRegistries := []models.ModelRegistryModel{
				{Name: "model-registry", Description: "Model Registry Description", DisplayName: "Model Registry", ServerAddress: "http://127.0.0.1:8080/api/model_registry/v1", IsAvailable: true},
				{Name: "model-registry-one", Description: "Model Registry One description", DisplayName: "Model Registry One", ServerAddress: "http://127.0.0.1:8080/api/model_registry/v1", IsAvailable: false},
			}
			Expect(registries).To(ConsistOf(expectedRegistries))
		})

		It("should return empty list when AllowList is false and no services are allowed", func() {
			By("creating context with no allowed services")
			authCtx := &models.ServiceAuthorizationContext{
				AllowList:           false,
				AllowedServiceNames: []string{}, // No services allowed
				Namespace:           "kubeflow",
			}
			ctx := context.WithValue(mocks.NewMockSessionContextNoParent(), constants.ServiceAuthorizationContextKey, authCtx)

			By("fetching model registries with no allowed services")
			registries, err := modelRegistryRepository.GetAllModelRegistriesWithMode(ctx, serviceAccountMockedK8client, "kubeflow", false)
			Expect(err).NotTo(HaveOccurred())

			By("should return empty list")
			Expect(registries).To(BeEmpty())
		})

		It("should handle non-existent services gracefully when AllowList is false", func() {
			By("creating context with non-existent service names")
			authCtx := &models.ServiceAuthorizationContext{
				AllowList:           false,
				AllowedServiceNames: []string{"non-existent-service", "another-missing-service"},
				Namespace:           "kubeflow",
			}
			ctx := context.WithValue(mocks.NewMockSessionContextNoParent(), constants.ServiceAuthorizationContextKey, authCtx)

			By("fetching model registries with non-existent services")
			registries, err := modelRegistryRepository.GetAllModelRegistriesWithMode(ctx, serviceAccountMockedK8client, "kubeflow", false)
			Expect(err).NotTo(HaveOccurred())

			By("should return empty list without error")
			Expect(registries).To(BeEmpty())
		})

		It("should handle mixed existing and non-existent services when AllowList is false", func() {
			By("creating context with mix of existing and non-existent services")
			authCtx := &models.ServiceAuthorizationContext{
				AllowList:           false,
				AllowedServiceNames: []string{"model-registry", "non-existent-service"},
				Namespace:           "kubeflow",
			}
			ctx := context.WithValue(mocks.NewMockSessionContextNoParent(), constants.ServiceAuthorizationContextKey, authCtx)

			By("fetching model registries with mixed service names")
			registries, err := modelRegistryRepository.GetAllModelRegistriesWithMode(ctx, serviceAccountMockedK8client, "kubeflow", false)
			Expect(err).NotTo(HaveOccurred())

			By("should return only the existing service")
			expectedRegistries := []models.ModelRegistryModel{
				{Name: "model-registry", Description: "Model Registry Description", DisplayName: "Model Registry", ServerAddress: "http://127.0.0.1:8080/api/model_registry/v1", IsAvailable: true},
			}
			Expect(registries).To(ConsistOf(expectedRegistries))
		})

		It("should fallback to all services when no authorization context is present", func() {
			By("using context without authorization context")
			ctx := mocks.NewMockSessionContextNoParent() // No authorization context

			By("fetching model registries without authorization context")
			registries, err := modelRegistryRepository.GetAllModelRegistriesWithMode(ctx, serviceAccountMockedK8client, "kubeflow", false)
			Expect(err).NotTo(HaveOccurred())

			By("should return all available services as fallback behavior")
			expectedRegistries := []models.ModelRegistryModel{
				{Name: "model-registry", Description: "Model Registry Description", DisplayName: "Model Registry", ServerAddress: "http://127.0.0.1:8080/api/model_registry/v1", IsAvailable: true},
				{Name: "model-registry-one", Description: "Model Registry One description", DisplayName: "Model Registry One", ServerAddress: "http://127.0.0.1:8080/api/model_registry/v1", IsAvailable: false},
			}
			Expect(registries).To(ConsistOf(expectedRegistries))
		})
	})

	Context("with federated mode", func() {
		var modelRegistryRepository *ModelRegistryRepository
		var serviceAccountMockedK8client k8s.KubernetesClientInterface

		BeforeEach(func() {
			modelRegistryRepository = NewModelRegistryRepository()
			var err error
			serviceAccountMockedK8client, err = kubernetesMockedStaticClientFactory.GetClient(mocks.NewMockSessionContextNoParent())
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle federated mode with AllowList authorization", func() {
			By("creating context with AllowList authorization")
			authCtx := &models.ServiceAuthorizationContext{
				AllowList:           true,
				AllowedServiceNames: []string{},
				Namespace:           "kubeflow",
			}
			ctx := context.WithValue(mocks.NewMockSessionContextNoParent(), constants.ServiceAuthorizationContextKey, authCtx)

			By("fetching model registries in federated mode")
			registries, err := modelRegistryRepository.GetAllModelRegistriesWithMode(ctx, serviceAccountMockedK8client, "kubeflow", true)
			Expect(err).NotTo(HaveOccurred())

			By("should return registries with appropriate server addresses for federated mode")
			Expect(registries).To(HaveLen(2))
			// Note: The exact server addresses depend on the mock implementation
			// but the key point is that federated mode is properly handled with authorization
		})

		It("should handle federated mode with restricted access", func() {
			By("creating context with specific allowed services")
			authCtx := &models.ServiceAuthorizationContext{
				AllowList:           false,
				AllowedServiceNames: []string{"model-registry"},
				Namespace:           "kubeflow",
			}
			ctx := context.WithValue(mocks.NewMockSessionContextNoParent(), constants.ServiceAuthorizationContextKey, authCtx)

			By("fetching model registries in federated mode with restrictions")
			registries, err := modelRegistryRepository.GetAllModelRegistriesWithMode(ctx, serviceAccountMockedK8client, "kubeflow", true)
			Expect(err).NotTo(HaveOccurred())

			By("should return only allowed services in federated mode")
			Expect(registries).To(HaveLen(1))
			Expect(registries[0].Name).To(Equal("model-registry"))
		})
	})
})
