package kubernetes

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const ComponentLabelValue = "model-registry"
const ComponentLabelValueCatalog = "model-catalog"

const CatalogSourceKey = "sources.yaml"
const CatalogSourceDefaultConfigMapName = "default-catalog-sources"
const CatalogSourceUserConfigMapName = "model-catalog-sources"

const McpCatalogSourceKey = "sources.yaml"
const McpCatalogSourceDefaultConfigMapName = CatalogSourceDefaultConfigMapName
const McpCatalogSourceUserConfigMapName = "mcp-catalog-sources"

type KubernetesClientInterface interface {
	// Service discovery
	GetServiceNames(ctx context.Context, namespace string) ([]string, error)
	GetServiceDetailsByName(ctx context.Context, namespace, serviceName string, serviceType string) (ServiceDetails, error)
	GetServiceDetails(ctx context.Context, namespace string) ([]ServiceDetails, error)
	// GetServiceEndpoints returns the Endpoints for a service (used to detect availability). On permission error callers should treat the registry as available.
	//nolint:staticcheck // intentionally using deprecated corev1.Endpoints for RBAC compatibility; see tech debt ticket for EndpointSlice migration
	GetServiceEndpoints(ctx context.Context, namespace, serviceName string) (*corev1.Endpoints, error)

	// Namespace access
	GetNamespaces(ctx context.Context, identity *RequestIdentity) ([]corev1.Namespace, error)

	// Permission checks (abstracted SAR/SelfSAR)
	CanListServicesInNamespace(ctx context.Context, identity *RequestIdentity, namespace string) (bool, error)
	CanAccessServiceInNamespace(ctx context.Context, identity *RequestIdentity, namespace, serviceName string) (bool, error)
	CanNamespaceAccessRegistry(ctx context.Context, identity *RequestIdentity, jobNamespace, registryName, registryNamespace string) (bool, error)
	GetSelfSubjectRulesReview(ctx context.Context, identity *RequestIdentity, namespace string) ([]string, error)

	// Meta
	IsClusterAdmin(identity *RequestIdentity) (bool, error)
	BearerToken() (string, error)
	GetUser(identity *RequestIdentity) (string, error)

	// Model Registry Settings
	GetGroups(ctx context.Context) ([]string, error)

	//Model Catalog Settings
	GetAllCatalogSourceConfigs(ctx context.Context, namespace string) (corev1.ConfigMap, corev1.ConfigMap, error)
	UpdateCatalogSourceConfig(ctx context.Context, namespace string, configMap *corev1.ConfigMap) error
	CreateSecret(ctx context.Context, namespace string, secret *corev1.Secret) (*corev1.Secret, error)
	PatchSecret(ctx context.Context, namespace string, secretName string, data map[string]string) error
	DeleteSecret(ctx context.Context, namespace string, secretName string) error

	// MCP Catalog Settings
	GetAllMcpCatalogSourceConfigs(ctx context.Context, namespace string) (corev1.ConfigMap, corev1.ConfigMap, error)
	UpdateMcpCatalogSourceConfig(ctx context.Context, namespace string, configMap *corev1.ConfigMap) error

	// Model transfer jobs
	CanListJobsClusterWide(ctx context.Context, identity *RequestIdentity) (bool, error)
	GetAllModelTransferJobs(ctx context.Context, namespace string, modelRegistryID string, jobNamespace string) (*batchv1.JobList, error)
	CreateModelTransferJob(ctx context.Context, namespace string, job *batchv1.Job) (*batchv1.Job, error)
	GetTransferJobPods(ctx context.Context, namespace string, jobNames []string) (*corev1.PodList, error)
	GetEventsForPods(ctx context.Context, namespace string, podNames []string) (*corev1.EventList, error)
	DeleteModelTransferJob(ctx context.Context, namespace string, jobName string) error
	CreateConfigMap(ctx context.Context, namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error)
	DeleteConfigMap(ctx context.Context, namespace string, name string) error
	GetModelTransferJob(ctx context.Context, namespace string, jobName string) (*batchv1.Job, error)
	GetConfigMap(ctx context.Context, namespace string, name string) (*corev1.ConfigMap, error)
	GetSecret(ctx context.Context, namespace string, name string) (*corev1.Secret, error)
	PatchSecretOwnerReference(ctx context.Context, namespace string, name string, ownerRef metav1.OwnerReference) error
	PatchConfigMapOwnerReference(ctx context.Context, namespace string, name string, ownerRef metav1.OwnerReference) error
}
