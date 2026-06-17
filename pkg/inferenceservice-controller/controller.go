package inferenceservicecontroller

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	kservev1beta1 "github.com/kserve/kserve/pkg/apis/serving/v1beta1"
	"github.com/kubeflow/hub/pkg/openapi"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type InferenceServiceController struct {
	client                        client.Client
	httpClient                    *http.Client
	log                           logr.Logger
	bearerToken                   string
	inferenceServiceIDLabel       string
	registeredModelIDLabel        string
	modelVersionIDLabel           string
	modelRegistryNamespaceLabel   string
	modelRegistryNameLabel        string
	modelRegistryURLAnnotation    string
	modelRegistryFinalizer        string
	serviceURLAnnotation          string
	defaultModelRegistryNamespace string
}

func NewInferenceServiceController(
	client client.Client,
	log logr.Logger,
	skipTLSVerify bool,
	bearerToken,
	isIDLabel,
	regModelIDLabel,
	modelVerIDLabel,
	mrNamespaceLabel,
	mrNameLabel,
	mrURLAnnotation,
	mrFinalizer,
	serviceURLAnnotation,
	defaultMRNamespace string,
) *InferenceServiceController {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if skipTLSVerify {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
			NextProtos:         []string{"h2", "http/1.1"},
		}
	}
	httpClient := &http.Client{Transport: transport}

	return &InferenceServiceController{
		client:                        client,
		httpClient:                    httpClient,
		log:                           log,
		bearerToken:                   bearerToken,
		inferenceServiceIDLabel:       isIDLabel,
		registeredModelIDLabel:        regModelIDLabel,
		modelVersionIDLabel:           modelVerIDLabel,
		modelRegistryNamespaceLabel:   mrNamespaceLabel,
		modelRegistryNameLabel:        mrNameLabel,
		modelRegistryURLAnnotation:    mrURLAnnotation,
		modelRegistryFinalizer:        mrFinalizer,
		serviceURLAnnotation:          serviceURLAnnotation,
		defaultModelRegistryNamespace: defaultMRNamespace,
	}
}

func (r *InferenceServiceController) OverrideHTTPClient(client *http.Client) {
	r.httpClient = client
}

// Reconcile performs the reconciliation of the model registry based on Kubeflow InferenceService CRs
func (r *InferenceServiceController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	mrNamespace := r.defaultModelRegistryNamespace
	mrIs := &openapi.InferenceService{}
	mrApiCtx := context.Background()

	if r.bearerToken != "" {
		mrApiCtx = context.WithValue(context.Background(), openapi.ContextAccessToken, r.bearerToken)
	}

	// Initialize logger format
	log := r.log.WithValues("InferenceService", req.Name, "namespace", req.Namespace)

	isvc := &kservev1beta1.InferenceService{}
	err := r.client.Get(ctx, req.NamespacedName, isvc)
	if err != nil {
		if apierrs.IsNotFound(err) {
			log.V(1).Info("Stop ModelRegistry InferenceService reconciliation, ISVC not found.")
			return ctrl.Result{}, nil
		}

		log.Error(err, "Unable to fetch the InferenceService")
		return ctrl.Result{}, err
	}

	mrIsvcId, okMrIsvcId := isvc.Labels[r.inferenceServiceIDLabel]
	registeredModelId, okRegisteredModelId := isvc.Labels[r.registeredModelIDLabel]
	modelVersionId := isvc.Labels[r.modelVersionIDLabel]
	mrName, okMrName := isvc.Labels[r.modelRegistryNameLabel]
	mrUrl, okMrUrl := isvc.Annotations[r.modelRegistryURLAnnotation]

	if !okMrIsvcId && !okRegisteredModelId {
		// Early check: no model registry specific labels set in the ISVC, ignore the CR
		log.Info(fmt.Sprintf("missing model registry specific label, unable to link ISVC to Model Registry, skipping InferenceService: %s", isvc.Name))

		return ctrl.Result{}, nil
	}

	if !okMrName && !okMrUrl {
		// Early check: it's optional to have the model registry name or url set in the ISVC, but if not set, it will fail if there's more than one model registry in the namespace
		log.Info(fmt.Sprintf("missing %s or %s, will try to connect to the Model Registry service in the namespace %s", r.modelRegistryNameLabel, r.modelRegistryURLAnnotation, mrNamespace))
	}

	if mrNSFromISVC, ok := isvc.Labels[r.modelRegistryNamespaceLabel]; ok {
		mrNamespace = mrNSFromISVC
	}

	// Check if the InferenceService instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isMarkedToBeDeleted := isvc.GetDeletionTimestamp() != nil

	if isMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(isvc, r.modelRegistryFinalizer) {
			// Best-effort: try to update MR InferenceService state before removing the finalizer.
			// If the MR service is unavailable (e.g., deleted before the ISVC), we still
			// must remove the finalizer to avoid blocking deletion indefinitely.
			mrApi, err := r.initModelRegistryService(ctx, log, mrName, mrNamespace, mrUrl)
			if err != nil {
				log.Info("Model Registry service unavailable during deletion, skipping state update", "error", err)
			} else if mrIsvcId, ok := isvc.Labels[r.inferenceServiceIDLabel]; ok {
				mrIs, _, getErr := mrApi.ModelRegistryServiceAPI.GetInferenceService(mrApiCtx, mrIsvcId).Execute()
				if getErr != nil {
					log.Info("Unable to fetch InferenceService from Model Registry during deletion", "mrIsvcId", mrIsvcId, "error", getErr)
				} else if err := r.onDeletion(mrApiCtx, mrApi, log, mrIs); err != nil {
					log.Error(err, "Failed to update DesiredState to UNDEPLOYED, proceeding with finalizer removal")
				}
			}

			log.Info("Removing Finalizer for modelRegistry")
			if ok := controllerutil.RemoveFinalizer(isvc, r.modelRegistryFinalizer); !ok {
				return ctrl.Result{}, fmt.Errorf("failed to remove modelRegistry finalizer for InferenceService")
			}

			if err := r.client.Update(ctx, isvc); IgnoreDeletingErrors(err) != nil {
				return ctrl.Result{}, fmt.Errorf("failed to remove modelRegistry finalizer for InferenceService: %w", err)
			}
		}
		return ctrl.Result{}, nil
	}

	log.Info("Creating model registry service..")
	mrApi, err := r.initModelRegistryService(ctx, log, mrName, mrNamespace, mrUrl)
	if err != nil {
		log.Error(err, "Unable to initialize Model Registry service")
		return ctrl.Result{}, err
	}

	// Let's add a finalizer. Then, we can define some operations which should
	// occur before the custom resource is deleted.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers
	if !controllerutil.ContainsFinalizer(isvc, r.modelRegistryFinalizer) {
		log.Info("Adding Finalizer for ModelRegistry")

		if ok := controllerutil.AddFinalizer(isvc, r.modelRegistryFinalizer); !ok {
			log.Error(err, "Failed to add finalizer into the InferenceService custom resource")

			return ctrl.Result{Requeue: true}, nil
		}

		if err = r.client.Update(ctx, isvc); err != nil {
			log.Error(err, "Failed to update InferenceService custom resource to add finalizer")

			return ctrl.Result{}, err
		}
	}

	// Retrieve or create the ServingEnvironment associated to the current namespace
	servingEnvironment, err := r.getOrCreateServingEnvironment(mrApiCtx, log, mrApi, req.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	if okMrIsvcId {
		// Retrieve the IS from model registry using the id
		log.Info("Retrieving model registry InferenceService by id", "mrIsvcId", mrIsvcId)
		mrIs, _, err = mrApi.ModelRegistryServiceAPI.GetInferenceService(mrApiCtx, mrIsvcId).Execute()
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to find InferenceService with id %s in model registry: %w", mrIsvcId, err)
		}

		mrCurrentIvcUrl := mrIs.CustomProperties["url"].MetadataStringValue.GetStringValue()

		urlAreDiff := r.checkURLDiff(isvc, mrCurrentIvcUrl)
		if urlAreDiff {
			err := r.updateMRInferenceService(
				mrApiCtx,
				log,
				mrApi,
				isvc,
				mrIs,
			)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

	} else if okRegisteredModelId {
		// No corresponding InferenceService in model registry, create new one
		mrIs, err = r.createMRInferenceService(mrApiCtx, log, mrApi, isvc, *servingEnvironment.Id, registeredModelId, modelVersionId)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if mrIs == nil {
		// This should NOT happen
		return ctrl.Result{}, fmt.Errorf("unexpected nil model registry InferenceService")
	}

	// No need to update the ISVC, the IS id is already set
	if isvc.Labels[r.inferenceServiceIDLabel] == *mrIs.Id {
		return ctrl.Result{}, nil
	}

	// Update the ISVC label, set the newly created IS id if not present yet
	desired := isvc.DeepCopy()

	desired.Labels[r.inferenceServiceIDLabel] = *mrIs.Id

	if err := r.client.Update(ctx, desired); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *InferenceServiceController) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).
		For(&kservev1beta1.InferenceService{})

	return builder.Complete(r)
}

func (r *InferenceServiceController) checkURLDiff(isvc *kservev1beta1.InferenceService, mrIsvcUrl string) bool {
	if mrIsvcUrl == "" && isvc.Status.URL == nil {
		return false
	}

	if isvc.Status.URL == nil {
		return true
	}

	return mrIsvcUrl != isvc.Status.URL.String()
}

func (r *InferenceServiceController) initModelRegistryService(ctx context.Context, log logr.Logger, name, namespace, url string) (*openapi.APIClient, error) {
	var err error

	log1 := log.WithValues("mr-namespace", namespace, "mr-name", name)

	if url == "" {
		log1.Info("Retrieving url from deployed model registry service")

		url, err = r.getMRUrlFromService(ctx, name, namespace)
		if err != nil {
			log1.Error(err, "Unable to fetch the Model Registry service")
			return nil, err
		}
	}

	cfg := &openapi.Configuration{
		HTTPClient: r.httpClient,
		Servers: openapi.ServerConfigurations{
			{
				URL: url,
			},
		},
	}

	client := openapi.NewAPIClient(cfg)

	return client, nil
}

func (r *InferenceServiceController) getMRUrlFromService(ctx context.Context, name, namespace string) (string, error) {
	svc, err := r.getMRService(ctx, name, namespace)
	if err != nil {
		return "", fmt.Errorf("unable to find the Model Registry service: %w", err)
	}

	return r.buildURLFromService(svc)
}

func (r *InferenceServiceController) getMRService(ctx context.Context, name, namespace string) (*corev1.Service, error) {
	if name == "" {
		svcList := &corev1.ServiceList{}

		if err := r.client.List(ctx, svcList, client.InNamespace(namespace), client.MatchingLabels{"component": "model-registry"}); err != nil {
			return nil, fmt.Errorf("unable to list services in the namespace %s: %w", namespace, err)
		}

		if len(svcList.Items) == 0 {
			return nil, fmt.Errorf("no model registry services found in the namespace %s", namespace)
		}

		if len(svcList.Items) > 1 {
			return nil, fmt.Errorf("more than one model registry service found in the namespace %s, consider to specify the name in the label %s", namespace, r.modelRegistryNameLabel)
		}

		return &svcList.Items[0], nil
	}

	svc := &corev1.Service{}

	err := r.client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, svc)
	if err != nil {
		return nil, err
	}

	return svc, nil
}

func (r *InferenceServiceController) buildURLFromService(svc *corev1.Service) (string, error) {
	var restApiPort *int32

	if url, ok := svc.Annotations[r.serviceURLAnnotation]; ok {
		return fmt.Sprintf("https://%s", url), nil
	}

	for _, port := range svc.Spec.Ports {
		if port.Name == "http-api" {
			restApiPort = &port.Port
			break
		}
	}

	if restApiPort == nil {
		return "", fmt.Errorf("unable to find the http port in the Model Registry service")
	}

	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", svc.Name, svc.Namespace, *restApiPort), nil
}

func (r *InferenceServiceController) createMRInferenceService(
	ctx context.Context,
	log logr.Logger,
	mr *openapi.APIClient,
	isvc *kservev1beta1.InferenceService,
	servingEnvironmentId string,
	registeredModelId string,
	modelVersionId string,
) (*openapi.InferenceService, error) {
	modelVersionIdPtr := &modelVersionId
	if modelVersionId == "" {
		modelVersionIdPtr = nil
	}

	isName := fmt.Sprintf("%s/%s", isvc.Name, isvc.UID)

	is, _, err := mr.ModelRegistryServiceAPI.FindInferenceService(ctx).
		Name(isName).ParentResourceId(servingEnvironmentId).Execute()
	if err != nil {
		log.Info("Creating new model registry InferenceService", "name", isName, "registeredModelId", registeredModelId, "modelVersionId", modelVersionId)

		isCreate := openapi.InferenceServiceCreate{
			DesiredState:         openapi.INFERENCESERVICESTATE_DEPLOYED.Ptr(),
			ModelVersionId:       modelVersionIdPtr,
			Name:                 &isName,
			RegisteredModelId:    registeredModelId,
			Runtime:              isvc.Spec.Predictor.Model.Runtime,
			ServingEnvironmentId: servingEnvironmentId,
		}

		if isvc.Status.URL != nil {
			isCreate.CustomProperties = map[string]openapi.MetadataValue{
				"url": openapi.MetadataValue{
					MetadataStringValue: openapi.NewMetadataStringValue(isvc.Status.URL.String(), "MetadataStringValue"),
				},
			}
		}

		is, _, err = mr.ModelRegistryServiceAPI.CreateInferenceService(ctx).InferenceServiceCreate(isCreate).Execute()
	}

	return is, err
}

func (r *InferenceServiceController) updateMRInferenceService(
	ctx context.Context,
	log logr.Logger,
	mr *openapi.APIClient,
	isvc *kservev1beta1.InferenceService,
	mrIsvc *openapi.InferenceService,
) error {
	log.Info("Updating model registry InferenceService..")

	url := ""

	if isvc.Status.URL != nil {
		url = isvc.Status.URL.String()
	}

	if mrIsvc.CustomProperties == nil {
		mrIsvc.CustomProperties = map[string]openapi.MetadataValue{}
	}

	mrIsvc.CustomProperties["url"] = openapi.MetadataValue{
		MetadataStringValue: openapi.NewMetadataStringValue(url, "MetadataStringValue"),
	}

	_, _, err := mr.ModelRegistryServiceAPI.UpdateInferenceService(ctx, *mrIsvc.Id).InferenceServiceUpdate(openapi.InferenceServiceUpdate{
		CustomProperties: mrIsvc.CustomProperties,
	}).Execute()

	return err
}

func (r *InferenceServiceController) getOrCreateServingEnvironment(ctx context.Context, log logr.Logger, mr *openapi.APIClient, namespace string) (*openapi.ServingEnvironment, error) {
	servingEnvironment, _, err := mr.ModelRegistryServiceAPI.FindServingEnvironment(ctx).Name(namespace).Execute()
	if err != nil {
		log.Info("ServingEnvironment not found, creating it..")

		servingEnvironment, _, err = mr.ModelRegistryServiceAPI.CreateServingEnvironment(ctx).ServingEnvironmentCreate(openapi.ServingEnvironmentCreate{
			Name: namespace,
		}).Execute()
		if err != nil {
			return nil, fmt.Errorf("unable to create ServingEnvironment: %w", err)
		}
	}

	return servingEnvironment, nil
}

// onDeletion mark model registry inference service to UNDEPLOYED desired state
func (r *InferenceServiceController) onDeletion(ctx context.Context, mr *openapi.APIClient, log logr.Logger, is *openapi.InferenceService) (err error) {
	log.Info("Running onDeletion logic")
	if is.DesiredState != nil && *is.DesiredState != openapi.INFERENCESERVICESTATE_UNDEPLOYED {
		log.Info("InferenceService going to be deleted from cluster, setting desired state to UNDEPLOYED in model registry")

		_, _, err = mr.ModelRegistryServiceAPI.UpdateInferenceService(ctx, *is.Id).InferenceServiceUpdate(openapi.InferenceServiceUpdate{
			DesiredState: openapi.INFERENCESERVICESTATE_UNDEPLOYED.Ptr(),
		}).Execute()
	}
	return err
}

func IgnoreDeletingErrors(err error) error {
	if err == nil {
		return nil
	}
	if apierrs.IsNotFound(err) {
		return nil
	}
	return err
}
