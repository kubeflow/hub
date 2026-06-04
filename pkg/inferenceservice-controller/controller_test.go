package inferenceservicecontroller_test

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	kservev1beta1 "github.com/kserve/kserve/pkg/apis/serving/v1beta1"
	"github.com/kubeflow/hub/pkg/openapi"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"knative.dev/pkg/apis"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("InferenceService Controller", func() {
	When("Creating a new InferenceService with Model Registry labels", func() {
		It("If a label with inference service id is missing, it should add it after creating the required resources on model registry", func() {
			const CorrectInferenceServicePath = "./testdata/inferenceservices/inference-service-correct.yaml"
			const ModelRegistrySVCPath = "./testdata/deploy/model-registry-svc.yaml"
			const namespace = "correct"

			ns := &corev1.Namespace{}

			ns.SetName(namespace)

			if err := cli.Create(ctx, ns); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			mrSvc := &corev1.Service{}
			Expect(ConvertFileToStructuredResource(ModelRegistrySVCPath, mrSvc)).To(Succeed())

			mrSvc.SetNamespace(namespace)

			if err := cli.Create(ctx, mrSvc); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			inferenceService := &kservev1beta1.InferenceService{}
			Expect(ConvertFileToStructuredResource(CorrectInferenceServicePath, inferenceService)).To(Succeed())

			inferenceService.SetNamespace(namespace)

			inferenceService.Labels[namespaceLabel] = namespace

			if err := cli.Create(ctx, inferenceService); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			Eventually(func() error {
				isvc := &kservev1beta1.InferenceService{}
				err := cli.Get(ctx, types.NamespacedName{
					Name:      inferenceService.Name,
					Namespace: inferenceService.Namespace,
				}, isvc)
				if err != nil {
					return err
				}

				if isvc.Labels[inferenceServiceIDLabel] != "1" {
					return fmt.Errorf("Label for InferenceServiceID is not set, got %s", isvc.Labels[inferenceServiceIDLabel])
				}

				return nil
			}, 10*time.Second, 1*time.Second).Should(Succeed())
		})

		It("Should successfully set the DesiredState to UNDEPLOYED if the InferenceService is deleted", func() {
			const CorrectInferenceServicePath = "./testdata/inferenceservices/inference-service-correct.yaml"
			const ModelRegistrySVCPath = "./testdata/deploy/model-registry-svc.yaml"
			const namespace = "correct-delete"
			const mrUrl = "http://model-registry.svc.cluster.local:8080"

			restIsvc := &openapi.InferenceService{}

			ns := &corev1.Namespace{}

			ns.SetName(namespace)

			if err := cli.Create(ctx, ns); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			mrSvc := &corev1.Service{}
			Expect(ConvertFileToStructuredResource(ModelRegistrySVCPath, mrSvc)).To(Succeed())

			mrSvc.SetNamespace(namespace)

			if err := cli.Create(ctx, mrSvc); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			inferenceService := &kservev1beta1.InferenceService{}
			Expect(ConvertFileToStructuredResource(CorrectInferenceServicePath, inferenceService)).To(Succeed())

			inferenceService.SetNamespace(namespace)

			inferenceService.Labels[namespaceLabel] = namespace

			if err := cli.Create(ctx, inferenceService); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			Eventually(func() error {
				isvc := &kservev1beta1.InferenceService{}
				err := cli.Get(ctx, types.NamespacedName{
					Name:      inferenceService.Name,
					Namespace: inferenceService.Namespace,
				}, isvc)
				if err != nil {
					return err
				}

				if isvc.Labels[inferenceServiceIDLabel] != "1" {
					return fmt.Errorf("Label for InferenceServiceID is not set, got %s", isvc.Labels[inferenceServiceIDLabel])
				}

				return nil
			}, 10*time.Second, 1*time.Second).Should(Succeed())

			err := cli.Delete(ctx, inferenceService)
			Expect(err).To(BeNil())

			Eventually(func() error {
				resp, err := mrMockServer.Client().Get(mrUrl + "/api/model_registry/v1alpha3/inference_services/1")
				Expect(err).To(BeNil())

				//nolint:errcheck
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				Expect(err).To(BeNil())

				err = json.Unmarshal(body, &restIsvc)
				Expect(err).To(BeNil())

				if restIsvc.DesiredState == nil {
					return fmt.Errorf("DesiredState is not set")
				}

				if *restIsvc.DesiredState != openapi.INFERENCESERVICESTATE_UNDEPLOYED {
					return fmt.Errorf("DesiredState is not set to UNDEPLOYED, got %s", *restIsvc.DesiredState)
				}

				return nil
			}, 10*time.Second, 1*time.Second).Should(Succeed())
		})
	})

	When("Creating a new InferenceService without a Model Registry name", func() {
		It("Should successfully create the InferenceService if there's just one model registry in the namespace", func() {
			const InferenceServiceMissingNamePath = "./testdata/inferenceservices/inference-service-missing-name.yaml"
			const ModelRegistrySVCPath = "./testdata/deploy/model-registry-svc.yaml"
			const namespace = "correct-no-name"

			ns := &corev1.Namespace{}

			ns.SetName(namespace)

			if err := cli.Create(ctx, ns); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			mrSvc := &corev1.Service{}
			Expect(ConvertFileToStructuredResource(ModelRegistrySVCPath, mrSvc)).To(Succeed())

			mrSvc.SetNamespace(namespace)

			if err := cli.Create(ctx, mrSvc); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			inferenceService := &kservev1beta1.InferenceService{}
			Expect(ConvertFileToStructuredResource(InferenceServiceMissingNamePath, inferenceService)).To(Succeed())

			inferenceService.SetNamespace(namespace)

			inferenceService.Labels[namespaceLabel] = namespace

			if err := cli.Create(ctx, inferenceService); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			Eventually(func() error {
				isvc := &kservev1beta1.InferenceService{}
				err := cli.Get(ctx, types.NamespacedName{
					Name:      inferenceService.Name,
					Namespace: inferenceService.Namespace,
				}, isvc)
				if err != nil {
					return err
				}

				if isvc.Labels[inferenceServiceIDLabel] != "1" {
					return fmt.Errorf("Label for InferenceServiceID is not set, got %s", isvc.Labels[inferenceServiceIDLabel])
				}

				return nil
			}, 10*time.Second, 1*time.Second).Should(Succeed())
		})

		It("Should fail to create the InferenceService if there are multiple model registries in the namespace", func() {
			const InferenceServiceMissingNamePath = "./testdata/inferenceservices/inference-service-missing-name.yaml"
			const ModelRegistrySVCPath = "./testdata/deploy/model-registry-svc.yaml"
			const namespace = "fail-no-name"

			ns := &corev1.Namespace{}

			ns.SetName(namespace)

			if err := cli.Create(ctx, ns); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			mrSvc := &corev1.Service{}
			Expect(ConvertFileToStructuredResource(ModelRegistrySVCPath, mrSvc)).To(Succeed())

			mrSvc.SetNamespace(namespace)

			if err := cli.Create(ctx, mrSvc); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			mrSvc2 := &corev1.Service{}
			Expect(ConvertFileToStructuredResource(ModelRegistrySVCPath, mrSvc2)).To(Succeed())

			mrSvc2.SetNamespace(namespace)
			mrSvc2.SetName("model-registry-2")

			if err := cli.Create(ctx, mrSvc2); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			inferenceService := &kservev1beta1.InferenceService{}
			Expect(ConvertFileToStructuredResource(InferenceServiceMissingNamePath, inferenceService)).To(Succeed())

			inferenceService.SetNamespace(namespace)

			inferenceService.Labels[namespaceLabel] = namespace

			if err := cli.Create(ctx, inferenceService); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			Consistently(func() error {
				isvc := &kservev1beta1.InferenceService{}
				err := cli.Get(ctx, types.NamespacedName{
					Name:      inferenceService.Name,
					Namespace: inferenceService.Namespace,
				}, isvc)
				if err != nil {
					return err
				}

				if isvc.Labels[inferenceServiceIDLabel] != "1" {
					return fmt.Errorf("Label for InferenceServiceID is not set, got %s", isvc.Labels[inferenceServiceIDLabel])
				}

				return nil
			}, 5*time.Second, 1*time.Second).Should(Not(Succeed()))
		})
	})

	When("Creating a new InferenceService with a Model Registry service specifies an annotation URL", func() {
		It("Should successfully create the InferenceService with the correct URL", func() {
			const CorrectInferenceServicePath = "./testdata/inferenceservices/inference-service-correct.yaml"
			const ModelRegistrySVCPath = "./testdata/deploy/model-registry-svc.yaml"
			const namespace = "correct-annotation-url"

			ns := &corev1.Namespace{}

			ns.SetName(namespace)

			if err := cli.Create(ctx, ns); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			mrSvc := &corev1.Service{}
			Expect(ConvertFileToStructuredResource(ModelRegistrySVCPath, mrSvc)).To(Succeed())

			mrSvc.SetNamespace(namespace)

			mrSvc.Annotations = map[string]string{
				urlAnnotation: "model-registry.svc.cluster.local:8080",
			}

			if err := cli.Create(ctx, mrSvc); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			inferenceService := &kservev1beta1.InferenceService{}
			Expect(ConvertFileToStructuredResource(CorrectInferenceServicePath, inferenceService)).To(Succeed())

			inferenceService.SetNamespace(namespace)

			inferenceService.Labels[namespaceLabel] = namespace

			if err := cli.Create(ctx, inferenceService); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			Eventually(func() error {
				isvc := &kservev1beta1.InferenceService{}
				err := cli.Get(ctx, types.NamespacedName{
					Name:      inferenceService.Name,
					Namespace: inferenceService.Namespace,
				}, isvc)
				if err != nil {
					return err
				}

				if isvc.Labels[inferenceServiceIDLabel] != "1" {
					return fmt.Errorf("Label for InferenceServiceID is not set, got %s", isvc.Labels[inferenceServiceIDLabel])
				}

				return nil
			}, 10*time.Second, 1*time.Second).Should(Succeed())
		})
	})

	When("An InferenceService have a status.url defined", func() {
		It("Should get reconciled in the model registry InferenceService resource", func() {
			const InferenceServiceMissingNamePath = "./testdata/inferenceservices/inference-service-missing-name.yaml"
			const ModelRegistrySVCPath = "./testdata/deploy/model-registry-svc.yaml"
			const namespace = "url-reconcile"
			const mrUrl = "http://model-registry.svc.cluster.local:8080"

			ns := &corev1.Namespace{}

			ns.SetName(namespace)

			if err := cli.Create(ctx, ns); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			mrSvc := &corev1.Service{}
			Expect(ConvertFileToStructuredResource(ModelRegistrySVCPath, mrSvc)).To(Succeed())

			mrSvc.SetNamespace(namespace)

			if err := cli.Create(ctx, mrSvc); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			inferenceService := &kservev1beta1.InferenceService{}
			Expect(ConvertFileToStructuredResource(InferenceServiceMissingNamePath, inferenceService)).To(Succeed())

			inferenceService.SetNamespace(namespace)

			inferenceService.Labels[namespaceLabel] = namespace

			if err := cli.Create(ctx, inferenceService); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			Eventually(func() error {
				isvc := &kservev1beta1.InferenceService{}
				err := cli.Get(ctx, types.NamespacedName{
					Name:      inferenceService.Name,
					Namespace: inferenceService.Namespace,
				}, isvc)
				if err != nil {
					return err
				}

				if isvc.Labels[inferenceServiceIDLabel] != "1" {
					return fmt.Errorf("Label for InferenceServiceID is not set, got %s", isvc.Labels[inferenceServiceIDLabel])
				}

				return nil
			}, 10*time.Second, 1*time.Second).Should(Succeed())

			restIsvc := &openapi.InferenceService{}

			resp, err := mrMockServer.Client().Get(mrUrl + "/api/model_registry/v1alpha3/inference_services/1")
			Expect(err).To(BeNil())

			//nolint:errcheck
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			Expect(err).To(BeNil())

			err = json.Unmarshal(body, &restIsvc)
			Expect(err).To(BeNil())
			Expect(restIsvc.CustomProperties).To(BeNil())

			url, err := apis.ParseURL("https://example.com")
			Expect(err).To(BeNil())

			err = cli.Get(ctx, types.NamespacedName{Name: inferenceService.Name, Namespace: inferenceService.Namespace}, inferenceService)
			Expect(err).To(BeNil())

			inferenceService.Status.URL = url

			if err := cli.Status().Update(ctx, inferenceService); err != nil {
				Fail(err.Error())
			}

			Eventually(func() error {
				resp, err := mrMockServer.Client().Get(mrUrl + "/api/model_registry/v1alpha3/inference_services/1")
				Expect(err).To(BeNil())

				//nolint:errcheck
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				Expect(err).To(BeNil())

				err = json.Unmarshal(body, &restIsvc)
				Expect(err).To(BeNil())

				if restIsvc.CustomProperties == nil {
					return fmt.Errorf("InferenceService URL is not set")
				}

				if restIsvc.CustomProperties["url"].MetadataStringValue.GetStringValue() != url.String() {
					return fmt.Errorf("InferenceService URL is not set correctly, got %s, want %s", restIsvc.CustomProperties["url"].MetadataStringValue.GetStringValue(), url.String())
				}

				return nil
			}, 20*time.Second, 1*time.Second).Should(Succeed())
		})

		It("Should not set the model registry InferenceService url if the status.url is nil", func() {
			const InferenceServiceMissingNamePath = "./testdata/inferenceservices/inference-service-missing-name.yaml"
			const ModelRegistrySVCPath = "./testdata/deploy/model-registry-svc.yaml"
			const namespace = "url-reconcile-empty-status-url"
			const mrUrl = "http://model-registry.svc.cluster.local:8080"

			ns := &corev1.Namespace{}

			ns.SetName(namespace)

			if err := cli.Create(ctx, ns); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			mrSvc := &corev1.Service{}
			Expect(ConvertFileToStructuredResource(ModelRegistrySVCPath, mrSvc)).To(Succeed())

			mrSvc.SetNamespace(namespace)

			if err := cli.Create(ctx, mrSvc); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			inferenceService := &kservev1beta1.InferenceService{}
			Expect(ConvertFileToStructuredResource(InferenceServiceMissingNamePath, inferenceService)).To(Succeed())

			inferenceService.SetNamespace(namespace)

			inferenceService.Labels[namespaceLabel] = namespace

			if err := cli.Create(ctx, inferenceService); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			Eventually(func() error {
				isvc := &kservev1beta1.InferenceService{}
				err := cli.Get(ctx, types.NamespacedName{
					Name:      inferenceService.Name,
					Namespace: inferenceService.Namespace,
				}, isvc)
				if err != nil {
					return err
				}

				if isvc.Labels[inferenceServiceIDLabel] != "1" {
					return fmt.Errorf("Label for InferenceServiceID is not set, got %s", isvc.Labels[inferenceServiceIDLabel])
				}

				return nil
			}, 10*time.Second, 1*time.Second).Should(Succeed())

			restIsvc := &openapi.InferenceService{}

			err := cli.Get(ctx, types.NamespacedName{Name: inferenceService.Name, Namespace: inferenceService.Namespace}, inferenceService)
			Expect(err).To(BeNil())

			inferenceService.Status.URL = nil

			if err := cli.Status().Update(ctx, inferenceService); err != nil {
				Fail(err.Error())
			}

			Eventually(func() error {
				resp, err := mrMockServer.Client().Get(mrUrl + "/api/model_registry/v1alpha3/inference_services/1")
				Expect(err).To(BeNil())

				//nolint:errcheck
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				Expect(err).To(BeNil())

				err = json.Unmarshal(body, &restIsvc)
				Expect(err).To(BeNil())

				if restIsvc.CustomProperties != nil {
					return fmt.Errorf("InferenceService URL should not be set")
				}

				return nil
			}, 20*time.Second, 1*time.Second).Should(Succeed())
		})

		It("Should update the model registry InferenceService url if the status.url is updated", func() {
			const InferenceServiceMissingNamePath = "./testdata/inferenceservices/inference-service-missing-name.yaml"
			const ModelRegistrySVCPath = "./testdata/deploy/model-registry-svc.yaml"
			const namespace = "url-reconcile-update-status-url"
			const mrUrl = "http://model-registry.svc.cluster.local:8080"

			ns := &corev1.Namespace{}

			ns.SetName(namespace)

			if err := cli.Create(ctx, ns); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			mrSvc := &corev1.Service{}
			Expect(ConvertFileToStructuredResource(ModelRegistrySVCPath, mrSvc)).To(Succeed())

			mrSvc.SetNamespace(namespace)

			if err := cli.Create(ctx, mrSvc); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			inferenceService := &kservev1beta1.InferenceService{}
			Expect(ConvertFileToStructuredResource(InferenceServiceMissingNamePath, inferenceService)).To(Succeed())

			inferenceService.SetNamespace(namespace)

			inferenceService.Labels[namespaceLabel] = namespace

			if err := cli.Create(ctx, inferenceService); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			Eventually(func() error {
				isvc := &kservev1beta1.InferenceService{}
				err := cli.Get(ctx, types.NamespacedName{
					Name:      inferenceService.Name,
					Namespace: inferenceService.Namespace,
				}, isvc)
				if err != nil {
					return err
				}

				if isvc.Labels[inferenceServiceIDLabel] != "1" {
					return fmt.Errorf("Label for InferenceServiceID is not set, got %s", isvc.Labels[inferenceServiceIDLabel])
				}

				return nil
			}, 10*time.Second, 1*time.Second).Should(Succeed())

			restIsvc := &openapi.InferenceService{}

			resp, err := mrMockServer.Client().Get(mrUrl + "/api/model_registry/v1alpha3/inference_services/1")
			Expect(err).To(BeNil())

			//nolint:errcheck
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			Expect(err).To(BeNil())

			err = json.Unmarshal(body, &restIsvc)
			Expect(err).To(BeNil())
			Expect(restIsvc.CustomProperties).To(BeNil())

			url, err := apis.ParseURL("https://example.com")
			Expect(err).To(BeNil())

			err = cli.Get(ctx, types.NamespacedName{Name: inferenceService.Name, Namespace: inferenceService.Namespace}, inferenceService)
			Expect(err).To(BeNil())

			inferenceService.Status.URL = url

			if err := cli.Status().Update(ctx, inferenceService); err != nil {
				Fail(err.Error())
			}

			Eventually(func() error {
				resp, err := mrMockServer.Client().Get(mrUrl + "/api/model_registry/v1alpha3/inference_services/1")
				Expect(err).To(BeNil())

				//nolint:errcheck
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				Expect(err).To(BeNil())

				err = json.Unmarshal(body, &restIsvc)
				Expect(err).To(BeNil())

				if restIsvc.CustomProperties == nil {
					return fmt.Errorf("InferenceService URL is not set")
				}

				if restIsvc.CustomProperties["url"].MetadataStringValue.GetStringValue() != url.String() {
					return fmt.Errorf("InferenceService URL is not set correctly, got %s, want %s", restIsvc.CustomProperties["url"].MetadataStringValue.GetStringValue(), url.String())
				}

				return nil
			}, 20*time.Second, 1*time.Second).Should(Succeed())

			url, err = apis.ParseURL("https://example-updated.com")
			Expect(err).To(BeNil())

			err = cli.Get(ctx, types.NamespacedName{Name: inferenceService.Name, Namespace: inferenceService.Namespace}, inferenceService)
			Expect(err).To(BeNil())

			inferenceService.Status.URL = url

			if err := cli.Status().Update(ctx, inferenceService); err != nil {
				Fail(err.Error())
			}

			Eventually(func() error {
				resp, err := mrMockServer.Client().Get(mrUrl + "/api/model_registry/v1alpha3/inference_services/1")
				Expect(err).To(BeNil())

				//nolint:errcheck
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				Expect(err).To(BeNil())

				err = json.Unmarshal(body, &restIsvc)
				Expect(err).To(BeNil())

				if restIsvc.CustomProperties == nil {
					return fmt.Errorf("InferenceService URL is not set")
				}

				if restIsvc.CustomProperties["url"].MetadataStringValue.GetStringValue() != url.String() {
					return fmt.Errorf("InferenceService URL is not set correctly, got %s, want %s", restIsvc.CustomProperties["url"].MetadataStringValue.GetStringValue(), url.String())
				}

				return nil
			}, 20*time.Second, 1*time.Second).Should(Succeed())
		})
	})

	When("Deleting an InferenceService after the ModelRegistry service has been removed", func() {
		It("Should remove the finalizer and allow deletion even if the ModelRegistry service no longer exists", func() {
			const CorrectInferenceServicePath = "./testdata/inferenceservices/inference-service-correct.yaml"
			const ModelRegistrySVCPath = "./testdata/deploy/model-registry-svc.yaml"
			const namespace = "mr-deleted-before-isvc"

			ns := &corev1.Namespace{}

			ns.SetName(namespace)

			if err := cli.Create(ctx, ns); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			mrSvc := &corev1.Service{}
			Expect(ConvertFileToStructuredResource(ModelRegistrySVCPath, mrSvc)).To(Succeed())

			mrSvc.SetNamespace(namespace)

			if err := cli.Create(ctx, mrSvc); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			inferenceService := &kservev1beta1.InferenceService{}
			Expect(ConvertFileToStructuredResource(CorrectInferenceServicePath, inferenceService)).To(Succeed())

			inferenceService.SetNamespace(namespace)

			inferenceService.Labels[namespaceLabel] = namespace

			if err := cli.Create(ctx, inferenceService); err != nil && !errors.IsAlreadyExists(err) {
				Fail(err.Error())
			}

			// Wait for the controller to reconcile: IS ID label should be set and finalizer added
			Eventually(func() error {
				isvc := &kservev1beta1.InferenceService{}
				err := cli.Get(ctx, types.NamespacedName{
					Name:      inferenceService.Name,
					Namespace: inferenceService.Namespace,
				}, isvc)
				if err != nil {
					return err
				}

				if isvc.Labels[inferenceServiceIDLabel] == "" {
					return fmt.Errorf("Label for InferenceServiceID is not set")
				}

				hasFinalizer := false
				for _, f := range isvc.Finalizers {
					if f == finalizerLabel {
						hasFinalizer = true
						break
					}
				}
				if !hasFinalizer {
					return fmt.Errorf("Finalizer %s not added to InferenceService", finalizerLabel)
				}

				return nil
			}, 10*time.Second, 1*time.Second).Should(Succeed())

			// Delete the ModelRegistry Service BEFORE deleting the InferenceService
			Expect(cli.Delete(ctx, mrSvc)).To(Succeed())

			// Verify the MR Service is actually gone
			Eventually(func() bool {
				svc := &corev1.Service{}
				err := cli.Get(ctx, types.NamespacedName{
					Name:      mrSvc.Name,
					Namespace: mrSvc.Namespace,
				}, svc)
				return errors.IsNotFound(err)
			}, 5*time.Second, 500*time.Millisecond).Should(BeTrue())

			// Now delete the InferenceService
			err := cli.Get(ctx, types.NamespacedName{
				Name:      inferenceService.Name,
				Namespace: inferenceService.Namespace,
			}, inferenceService)
			Expect(err).To(BeNil())

			Expect(cli.Delete(ctx, inferenceService)).To(Succeed())

			// The InferenceService should be fully deleted (finalizer removed)
			// despite the ModelRegistry service being gone
			Eventually(func() bool {
				isvc := &kservev1beta1.InferenceService{}
				err := cli.Get(ctx, types.NamespacedName{
					Name:      inferenceService.Name,
					Namespace: inferenceService.Namespace,
				}, isvc)
				return errors.IsNotFound(err)
			}, 15*time.Second, 1*time.Second).Should(BeTrue())
		})
	})
})
