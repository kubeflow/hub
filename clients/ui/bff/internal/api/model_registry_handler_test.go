package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/kubeflow/hub/ui/bff/internal/constants"
	"github.com/kubeflow/hub/ui/bff/internal/mocks"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	"github.com/kubeflow/hub/ui/bff/internal/repositories"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TestModelRegistryHandler", func() {
	Context("fetching model registries", Ordered, func() {
		It("should retrieve the model registries successfully", func() {

			By("creating the test app")
			testApp := App{
				kubernetesClientFactory: kubernetesMockedStaticClientFactory,
				repositories:            repositories.NewRepositories(mockMRClient, mockModelCatalogClient),
				logger:                  logger,
			}

			By("creating the http test infrastructure")
			requestPath := fmt.Sprintf(" %s?namespace=kubeflow", ModelRegistryListPath)
			req, err := http.NewRequest(http.MethodGet, requestPath, nil)

			ctx := mocks.NewMockSessionContext(req.Context())
			ctx = context.WithValue(ctx, constants.NamespaceHeaderParameterKey, "kubeflow")
			req = req.WithContext(ctx)

			Expect(err).NotTo(HaveOccurred())
			rr := httptest.NewRecorder()

			By("creating the http request for the handler")
			testApp.GetAllModelRegistriesHandler(rr, req, nil)
			rs := rr.Result()
			defer rs.Body.Close()
			body, err := io.ReadAll(rs.Body)
			Expect(err).NotTo(HaveOccurred())

			By("unmarshalling the model registries")
			var actual ModelRegistryListEnvelope
			err = json.Unmarshal(body, &actual)
			Expect(err).NotTo(HaveOccurred())
			Expect(rr.Code).To(Equal(http.StatusOK))

			By("should match the expected model registries")
			var expected = []models.ModelRegistryModel{
				{Name: "model-registry", Description: "Model Registry Description", DisplayName: "Model Registry", ServerAddress: "http://127.0.0.1:8080/api/model_registry/v1", IsAvailable: true},
				{Name: "model-registry-one", Description: "Model Registry One description", DisplayName: "Model Registry One", ServerAddress: "http://127.0.0.1:8080/api/model_registry/v1", IsAvailable: false},
			}
			Expect(actual.Data).To(ConsistOf(expected))
		})

	})
})
