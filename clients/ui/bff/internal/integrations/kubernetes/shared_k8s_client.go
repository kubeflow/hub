package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/kubeflow/hub/ui/bff/internal/constants"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

type SharedClientLogic struct {
	Client kubernetes.Interface
	Logger *slog.Logger
	Token  BearerToken
}

func (kc *SharedClientLogic) GetServiceNames(sessionCtx context.Context, namespace string) ([]string, error) {
	services, err := kc.GetServiceDetails(sessionCtx, namespace)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(services))
	for _, svc := range services {
		names = append(names, svc.Name)
	}

	return names, nil
}

func (kc *SharedClientLogic) GetServiceDetails(sessionCtx context.Context, namespace string) ([]ServiceDetails, error) {

	if namespace == "" {
		return nil, fmt.Errorf("namespace cannot be empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sessionLogger := sessionCtx.Value(constants.TraceLoggerKey).(*slog.Logger)

	labelSelector := fmt.Sprintf("component=%s", ComponentLabelValue)

	serviceList, err := kc.Client.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	var services []ServiceDetails

	for _, service := range serviceList.Items {
		serviceDetails, err := buildServiceDetails(&service, sessionLogger)
		if err != nil {
			sessionLogger.Warn("skipping service", "error", err)
			continue
		}
		services = append(services, *serviceDetails)

	}

	return services, nil
}

func buildServiceDetails(service *corev1.Service, logger *slog.Logger) (*ServiceDetails, error) {
	if service == nil {
		return nil, fmt.Errorf("service cannot be nil")
	}

	var httpPort int32
	var isHTTPS bool
	hasHTTPPort := false
	for _, port := range service.Spec.Ports {
		if port.Name == "http-api" || port.Name == "https-api" {
			httpPort = port.Port
			isHTTPS = port.Name == "https-api"
			hasHTTPPort = true
			break
		}
	}
	if !hasHTTPPort {
		logger.Error("service missing HTTP/HTTPS port", "serviceName", service.Name)
		return nil, fmt.Errorf("service %q missing required 'http-api' or 'https-api' port", service.Name)
	}

	if service.Spec.ClusterIP == "" {
		logger.Error("service missing valid ClusterIP", "serviceName", service.Name)
		return nil, fmt.Errorf("service %q missing ClusterIP", service.Name)
	}

	displayName := ""
	description := ""
	externalAddressRest := ""

	// Check for annotations including external-address-rest
	if service.Annotations != nil {
		displayName = service.Annotations["displayName"]
		description = service.Annotations["description"]

		// Look for external-address-rest annotation with any prefix
		for key, value := range service.Annotations {
			if strings.HasSuffix(key, "/external-address-rest") {
				externalAddressRest = value
				break
			}
		}
	}

	return &ServiceDetails{
		Name:                service.Name,
		DisplayName:         displayName,
		Description:         description,
		ClusterIP:           service.Spec.ClusterIP,
		HTTPPort:            httpPort,
		IsHTTPS:             isHTTPS,
		ExternalAddressRest: externalAddressRest,
	}, nil
}

func (kc *SharedClientLogic) GetServiceDetailsByName(sessionCtx context.Context, namespace string, serviceName string, serviceType string) (ServiceDetails, error) {
	if namespace == "" || serviceName == "" {
		return ServiceDetails{}, fmt.Errorf("namespace and serviceName cannot be empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sessionLogger := sessionCtx.Value(constants.TraceLoggerKey).(*slog.Logger)

	service, err := kc.Client.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return ServiceDetails{}, fmt.Errorf("failed to get service %q in namespace %q: %w", serviceName, namespace, err)
	}
	if serviceType != "" && service.Labels["component"] != serviceType {
		return ServiceDetails{}, fmt.Errorf("service %q in namespace %q is not a %s", serviceName, namespace, serviceType)
	}

	details, err := buildServiceDetails(service, sessionLogger)
	if err != nil {
		return ServiceDetails{}, err
	}
	return *details, nil
}

// GetServiceEndpoints returns the Endpoints for a service. Used to detect registry availability. Callers should treat fetch errors (e.g. Forbidden) as "assume available" to avoid new permission requirements upstream.
//
//nolint:staticcheck // intentionally using deprecated corev1.Endpoints for RBAC compatibility; see tech debt ticket for EndpointSlice migration
func (kc *SharedClientLogic) GetServiceEndpoints(ctx context.Context, namespace, serviceName string) (*corev1.Endpoints, error) {
	if namespace == "" || serviceName == "" {
		return nil, fmt.Errorf("namespace and serviceName cannot be empty")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	endpoints, err := kc.Client.CoreV1().Endpoints(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoints for service %q in namespace %q: %w", serviceName, namespace, err)
	}
	return endpoints, nil
}

// EndpointsHasReadyAddresses returns true if the Endpoints resource has at least one subset with ready addresses.
//
//nolint:staticcheck // intentionally using deprecated corev1.Endpoints for RBAC compatibility; see tech debt ticket for EndpointSlice migration
func EndpointsHasReadyAddresses(endpoints *corev1.Endpoints) bool {
	if endpoints == nil {
		return false
	}
	for _, subset := range endpoints.Subsets {
		if len(subset.Addresses) > 0 {
			return true
		}
	}
	return false
}

func (kc *SharedClientLogic) BearerToken() (string, error) {
	// Token is retained for follow-up calls; do not log it.
	return kc.Token.Raw(), nil
}

func (kc *SharedClientLogic) GetGroups(ctx context.Context) ([]string, error) {
	kc.Logger.Info("This functionality is not implement yet. This is a STUB API to unblock frontend development until we have a definition on how to create model registries")
	return []string{}, nil
}

func (kc *SharedClientLogic) GetAllCatalogSourceConfigs(
	sessionCtx context.Context,
	namespace string,
) (corev1.ConfigMap, corev1.ConfigMap, error) {

	if namespace == "" {
		return corev1.ConfigMap{}, corev1.ConfigMap{}, fmt.Errorf("namespace cannot be empty")
	}

	sessionLogger := sessionCtx.Value(constants.TraceLoggerKey).(*slog.Logger)

	// Fetch default sources
	defaultCM, err := kc.Client.CoreV1().
		ConfigMaps(namespace).
		Get(sessionCtx, CatalogSourceDefaultConfigMapName, metav1.GetOptions{})

	if err != nil {
		sessionLogger.Error("failed to fetch default catalog source configmap",
			"namespace", namespace,
			"name", CatalogSourceDefaultConfigMapName,
			"error", err,
		)
		return corev1.ConfigMap{}, corev1.ConfigMap{}, fmt.Errorf("failed to get %s: %w", CatalogSourceDefaultConfigMapName, err)
	}

	userCM, err := kc.Client.CoreV1().ConfigMaps(namespace).Get(sessionCtx, CatalogSourceUserConfigMapName, metav1.GetOptions{})

	if err != nil {
		if apierrors.IsNotFound(err) {
			return *defaultCM, corev1.ConfigMap{}, nil
		}
		sessionLogger.Error("failed to fetch catalog source configmap",
			"namespace", namespace,
			"name", CatalogSourceUserConfigMapName,
			"error", err,
		)
		return corev1.ConfigMap{}, corev1.ConfigMap{}, fmt.Errorf("failed to get %s: %w", CatalogSourceUserConfigMapName, err)
	}

	return *defaultCM, *userCM, nil
}

func (kc *SharedClientLogic) CreateSecret(
	ctx context.Context,
	namespace string,
	secret *corev1.Secret,
) (*corev1.Secret, error) {
	sessionLogger := ctx.Value(constants.TraceLoggerKey).(*slog.Logger)

	secretCreated, err := kc.Client.CoreV1().
		Secrets(namespace).
		Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		nameForLog := secret.Name
		if nameForLog == "" {
			nameForLog = secret.GenerateName + "<generated>"
		}
		sessionLogger.Error("failed to create secret",
			"namespace", namespace,
			"name", nameForLog,
			"error", err,
		)
		return nil, fmt.Errorf("failed to create secret %s: %w", nameForLog, err)
	}

	sessionLogger.Info("successfully created secret",
		"namespace", namespace,
		"name", secretCreated.Name,
	)
	return secretCreated, nil
}

func (kc *SharedClientLogic) PatchSecret(ctx context.Context, namespace string, secretName string,
	data map[string]string) error {
	sessionLogger := ctx.Value(constants.TraceLoggerKey).(*slog.Logger)

	secret, err := kc.Client.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		sessionLogger.Error("failed to get secret for patching",
			"namespace", namespace,
			"name", secretName,
			"error", err,
		)
		return err
	}

	if secret.StringData == nil {
		secret.StringData = make(map[string]string)
	}
	for k, v := range data {
		secret.StringData[k] = v
	}

	_, err = kc.Client.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		sessionLogger.Error("failed to patch secret",
			"namespace", namespace,
			"name", secretName,
			"error", err,
		)
		return fmt.Errorf("failed to patch secret %s: %w", secretName, err)
	}

	sessionLogger.Info("successfully patched secret",
		"namespace", namespace,
		"name", secretName,
	)
	return nil
}

func (kc *SharedClientLogic) DeleteSecret(
	ctx context.Context,
	namespace string,
	secretName string,
) error {
	sessionLogger := ctx.Value(constants.TraceLoggerKey).(*slog.Logger)

	err := kc.Client.CoreV1().
		Secrets(namespace).
		Delete(ctx, secretName, metav1.DeleteOptions{})

	if err != nil {
		sessionLogger.Warn("failed to delete secret (may not exist)",
			"namespace", namespace,
			"name", secretName,
			"error", err,
		)
		return fmt.Errorf("failed to delete secret %s: %w", secretName, err)
	}

	sessionLogger.Info("successfully deleted secret",
		"namespace", namespace,
		"name", secretName,
	)

	return nil
}

func (kc *SharedClientLogic) UpdateCatalogSourceConfig(
	ctx context.Context,
	namespace string,
	configMap *corev1.ConfigMap,
) error {
	sessionLogger := ctx.Value(constants.TraceLoggerKey).(*slog.Logger)

	_, err := kc.Client.CoreV1().
		ConfigMaps(namespace).
		Update(ctx, configMap, metav1.UpdateOptions{})

	if err != nil {
		sessionLogger.Error("failed to update catalog sources configmap",
			"namespace", namespace,
			"name", configMap.Name,
			"error", err,
		)
		return fmt.Errorf("failed to update configmap %s: %w", configMap.Name, err)
	}

	sessionLogger.Info("successfully updated catalog sources configmap",
		"namespace", namespace,
		"name", configMap.Name,
	)

	return nil
}

func (kc *SharedClientLogic) GetAllMcpCatalogSourceConfigs(
	sessionCtx context.Context,
	namespace string,
) (corev1.ConfigMap, corev1.ConfigMap, error) {
	if namespace == "" {
		return corev1.ConfigMap{}, corev1.ConfigMap{}, fmt.Errorf("namespace cannot be empty")
	}

	sessionLogger := sessionCtx.Value(constants.TraceLoggerKey).(*slog.Logger)

	defaultCM, err := kc.Client.CoreV1().
		ConfigMaps(namespace).
		Get(sessionCtx, McpCatalogSourceDefaultConfigMapName, metav1.GetOptions{})

	if err != nil {
		sessionLogger.Error("failed to fetch default MCP catalog source configmap",
			"namespace", namespace,
			"name", McpCatalogSourceDefaultConfigMapName,
			"error", err,
		)
		return corev1.ConfigMap{}, corev1.ConfigMap{}, fmt.Errorf("failed to get %s: %w", McpCatalogSourceDefaultConfigMapName, err)
	}

	userCM, err := kc.Client.CoreV1().ConfigMaps(namespace).Get(sessionCtx, McpCatalogSourceUserConfigMapName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return *defaultCM, corev1.ConfigMap{}, nil
		}
		sessionLogger.Error("failed to fetch MCP catalog source configmap",
			"namespace", namespace,
			"name", McpCatalogSourceUserConfigMapName,
			"error", err,
		)
		return corev1.ConfigMap{}, corev1.ConfigMap{}, fmt.Errorf("failed to get %s: %w", McpCatalogSourceUserConfigMapName, err)
	}

	return *defaultCM, *userCM, nil
}

func (kc *SharedClientLogic) UpdateMcpCatalogSourceConfig(
	ctx context.Context,
	namespace string,
	configMap *corev1.ConfigMap,
) error {
	sessionLogger := ctx.Value(constants.TraceLoggerKey).(*slog.Logger)

	_, err := kc.Client.CoreV1().
		ConfigMaps(namespace).
		Update(ctx, configMap, metav1.UpdateOptions{})

	if err != nil {
		sessionLogger.Error("failed to update MCP catalog sources configmap",
			"namespace", namespace,
			"name", configMap.Name,
			"error", err,
		)
		return fmt.Errorf("failed to update configmap %s: %w", configMap.Name, err)
	}

	sessionLogger.Info("successfully updated MCP catalog sources configmap",
		"namespace", namespace,
		"name", configMap.Name,
	)

	return nil
}

// GetAllModelTransferJobs lists model transfer jobs. If jobNamespace is non-empty,
// jobs are listed only in that namespace. Otherwise jobs are listed across all namespaces.
func (kc *SharedClientLogic) GetAllModelTransferJobs(
	ctx context.Context,
	namespace string,
	modelRegistryID string,
	jobNamespace string,
) (*batchv1.JobList, error) {
	if namespace == "" {
		return &batchv1.JobList{}, fmt.Errorf("namespace cannot be empty")
	}

	if modelRegistryID == "" {
		return &batchv1.JobList{}, fmt.Errorf("model registry name is required")
	}

	sessionLogger := ctx.Value(constants.TraceLoggerKey).(*slog.Logger)

	labelSelector := "modelregistry.kubeflow.org/job-type=async-upload,modelregistry.kubeflow.org/model-registry-name=" + modelRegistryID

	targetNamespace := metav1.NamespaceAll
	if jobNamespace != "" {
		targetNamespace = jobNamespace
	}

	modelTransferJobList, err := kc.Client.BatchV1().Jobs(targetNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})

	if err != nil {
		sessionLogger.Error("failed to fetch list of model transfer job",
			"namespace", namespace,
			"jobNamespace", jobNamespace,
			"error", err,
		)
		return &batchv1.JobList{}, fmt.Errorf("failed to list model transfer job: %w", err)
	}

	return modelTransferJobList, nil
}

func (kc *SharedClientLogic) GetModelTransferJob(ctx context.Context, namespace string, jobName string) (*batchv1.Job, error) {
	if namespace == "" {
		return &batchv1.Job{}, fmt.Errorf("namespace cannot be empty")
	}

	sessionLogger := ctx.Value(constants.TraceLoggerKey).(*slog.Logger)

	job, err := kc.Client.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		sessionLogger.Error("failed to get job", "namespace", namespace, "jobName", jobName, "error", err)
		return nil, fmt.Errorf("failed to get job %s: %w", jobName, err)
	}

	return job, nil
}

func (kc *SharedClientLogic) CreateModelTransferJob(ctx context.Context, namespace string, job *batchv1.Job) (*batchv1.Job, error) {
	sessionLogger := ctx.Value(constants.TraceLoggerKey).(*slog.Logger)

	jobCreated, err := kc.Client.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		sessionLogger.Error("failed to create job",
			"namespace", namespace,
			"name", job.Name,
			"error", err,
		)
		return nil, fmt.Errorf("failed to create job %s: %w", job.Name, err)
	}
	sessionLogger.Info("successfully created job",
		"namespace", namespace,
		"name", job.Name,
	)
	return jobCreated, nil
}

func (kc *SharedClientLogic) DeleteModelTransferJob(ctx context.Context, namespace string, jobName string) error {
	sessionLogger := ctx.Value(constants.TraceLoggerKey).(*slog.Logger)

	err := kc.Client.BatchV1().Jobs(namespace).Delete(ctx, jobName, metav1.DeleteOptions{})

	if err != nil {
		if apierrors.IsNotFound(err) {
			sessionLogger.Info("job not found, nothing to delete", "namespace", namespace, "jobName", jobName)
			return err
		}
		sessionLogger.Error("failed to delete job", "namespace", namespace, "jobName", jobName, "error", err)
		return fmt.Errorf("failed to delete job %s: %w", jobName, err)
	}

	sessionLogger.Info("successfully deleted model transfer job",
		"namespace", namespace,
		"jobName", jobName,
	)

	return nil
}

func (kc *SharedClientLogic) GetSecret(ctx context.Context, namespace string, name string) (*corev1.Secret, error) {
	sessionLogger := ctx.Value(constants.TraceLoggerKey).(*slog.Logger)

	secret, err := kc.Client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			sessionLogger.Debug("secret not found (may be expected when probing for free name)", "namespace", namespace, "name", name)
		} else {
			sessionLogger.Error("failed to get secret", "namespace", namespace, "name", name, "error", err)
		}
		return nil, fmt.Errorf("failed to get secret %s: %w", name, err)
	}

	return secret, nil
}

func (kc *SharedClientLogic) GetConfigMap(ctx context.Context, namespace string, name string) (*corev1.ConfigMap, error) {
	sessionLogger := ctx.Value(constants.TraceLoggerKey).(*slog.Logger)

	configMap, err := kc.Client.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			sessionLogger.Debug("configmap not found (may be expected when probing for free name)", "namespace", namespace, "name", name)
		} else {
			sessionLogger.Error("failed to get configmap", "namespace", namespace, "name", name, "error", err)
		}
		return nil, fmt.Errorf("failed to get configmap %s: %w", name, err)
	}

	return configMap, nil
}

func (kc *SharedClientLogic) CreateConfigMap(ctx context.Context, namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	sessionLogger := ctx.Value(constants.TraceLoggerKey).(*slog.Logger)

	configMapCreated, err := kc.Client.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		nameForLog := configMap.Name
		if nameForLog == "" {
			nameForLog = configMap.GenerateName + "<generated>"
		}
		sessionLogger.Error("failed to create configMap",
			"namespace", namespace,
			"name", nameForLog,
			"error", err,
		)
		return nil, fmt.Errorf("failed to create configMap %s: %w", nameForLog, err)
	}

	sessionLogger.Info("successfully created configMap",
		"namespace", namespace,
		"name", configMapCreated.Name,
	)
	return configMapCreated, nil
}

func (kc *SharedClientLogic) DeleteConfigMap(ctx context.Context, namespace string, name string) error {
	sessionLogger := ctx.Value(constants.TraceLoggerKey).(*slog.Logger)

	err := kc.Client.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})

	if err != nil {
		sessionLogger.Warn("failed to delete configMap (may not exist)",
			"namespace", namespace,
			"name", name,
			"error", err,
		)
		return fmt.Errorf("failed to delete configMap %s: %w", name, err)
	}

	sessionLogger.Info("successfully deleted configMap",
		"namespace", namespace,
		"name", name,
	)

	return nil

}

func (kc *SharedClientLogic) PatchConfigMapOwnerReference(ctx context.Context, namespace string, name string, ownerRef metav1.OwnerReference) error {
	sessionLogger := ctx.Value(constants.TraceLoggerKey).(*slog.Logger)

	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"ownerReferences": []metav1.OwnerReference{ownerRef},
		},
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		sessionLogger.Error("failed to marshal patch for configmap",
			"namespace", namespace,
			"name", name,
			"error", err)
		return fmt.Errorf("failed to marshal patch: %w", err)
	}

	_, err = kc.Client.CoreV1().ConfigMaps(namespace).Patch(
		ctx,
		name,
		types.StrategicMergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	if err != nil {
		sessionLogger.Error("failed to patch configmap owner reference",
			"namespace", namespace,
			"name", name,
			"error", err)
		return fmt.Errorf("failed to patch configmap owner reference: %w", err)
	}

	sessionLogger.Info("successfully patched configmap owner reference", "name", name)
	return nil
}

func (kc *SharedClientLogic) GetTransferJobPods(ctx context.Context, namespace string, jobNames []string) (*corev1.PodList, error) {
	if namespace == "" || len(jobNames) == 0 {
		return &corev1.PodList{}, nil
	}

	sessionLogger := ctx.Value(constants.TraceLoggerKey).(*slog.Logger)

	labelSelector := "job-name in (" + strings.Join(jobNames, ",") + ")"

	podList, err := kc.Client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		sessionLogger.Error("failed to list pods for jobs", "namespace", namespace, "jobNames", jobNames, "error", err)
		return &corev1.PodList{}, fmt.Errorf("failed to list pods for jobs: %w", err)
	}

	return podList, nil
}

func (kc *SharedClientLogic) GetEventsForPods(ctx context.Context, namespace string, podNames []string) (*corev1.EventList, error) {
	if namespace == "" || len(podNames) == 0 {
		return &corev1.EventList{}, nil
	}

	sessionLogger := ctx.Value(constants.TraceLoggerKey).(*slog.Logger)

	var allEvents []corev1.Event
	for _, podName := range podNames {
		fieldSelector := fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Pod", podName)
		events, err := kc.Client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
			FieldSelector: fieldSelector,
		})
		if err != nil {
			sessionLogger.Warn("failed to list events for pod", "pod", podName, "error", err)
			continue
		}
		allEvents = append(allEvents, events.Items...)
	}

	return &corev1.EventList{Items: allEvents}, nil
}

func (kc *SharedClientLogic) PatchSecretOwnerReference(ctx context.Context, namespace string, name string, ownerRef metav1.OwnerReference) error {
	sessionLogger := ctx.Value(constants.TraceLoggerKey).(*slog.Logger)

	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"ownerReferences": []metav1.OwnerReference{ownerRef},
		},
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		sessionLogger.Error("failed to marshal patch for secret",
			"namespace", namespace,
			"name", name,
			"error", err)
		return fmt.Errorf("failed to marshal patch: %w", err)
	}

	_, err = kc.Client.CoreV1().Secrets(namespace).Patch(
		ctx,
		name,
		types.StrategicMergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	if err != nil {
		sessionLogger.Error("failed to patch secret owner reference",
			"namespace", namespace,
			"name", name,
			"error", err)
		return fmt.Errorf("failed to patch secret owner reference: %w", err)
	}

	sessionLogger.Info("successfully patched secret owner reference", "name", name)
	return nil

}
