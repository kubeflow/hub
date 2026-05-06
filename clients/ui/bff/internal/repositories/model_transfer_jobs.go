package repositories

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/kubeflow/hub/ui/bff/internal/constants"
	helper "github.com/kubeflow/hub/ui/bff/internal/helpers"
	k8s "github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	// DefaultAsyncUploadImage is the default container image for async-upload jobs.
	DefaultAsyncUploadImage  = "ghcr.io/kubeflow/hub/job/async-upload:latest"
	asyncUploadConfigMapName = "model-registry-ui-config"
	asyncUploadConfigMapKey  = "images-jobs-async-upload"
)

var (
	ErrJobNotFound           = errors.New("model transfer job not found")
	ErrJobValidationFailed   = errors.New("validation failed")
	ErrModelRegistryNotFound = errors.New("model registry not found in the selected namespace")
)

// recoverFromAnnotation copies annotation value to target if target is empty
func recoverFromAnnotation(target *string, annotations map[string]string, key string) {
	if *target == "" {
		*target = annotations[key]
	}
}

// recoverEnumFromAnnotation copies annotation value to target enum if target is empty
func recoverEnumFromAnnotation[T ~string](target *T, annotations map[string]string, key string) {
	if *target == "" {
		if val := annotations[key]; val != "" {
			*target = T(val)
		}
	}
}

func isK8sJobFailed(job *batchv1.Job) bool {
	if job == nil {
		return false
	}
	if job.Status.Failed > 0 {
		return true
	}
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// ErrForbidden indicates the user does not have permission for the requested operation.
var ErrForbidden = errors.New("forbidden")

func (m *ModelRegistryRepository) GetAllModelTransferJobs(ctx context.Context, client k8s.KubernetesClientInterface, namespace string, modelRegistryID string, jobNamespace string) (*models.ModelTransferJobList, error) {
	if modelRegistryID == "" {
		return &models.ModelTransferJobList{Items: []models.ModelTransferJob{}, Size: 0, PageSize: 0}, nil
	}

	logger := helper.GetContextLogger(ctx)

	// If no specific namespace is provided, check if the user can list jobs cluster-wide.
	if jobNamespace == "" {
		identity, ok := ctx.Value(constants.RequestIdentityKey).(*k8s.RequestIdentity)
		if !ok || identity == nil {
			return nil, fmt.Errorf("request identity not found in context")
		}
		canList, err := client.CanListJobsClusterWide(ctx, identity)
		if err != nil {
			return nil, fmt.Errorf("failed to check job list permission: %w", err)
		}
		if !canList {
			return nil, fmt.Errorf("%w: user does not have permission to list jobs across all namespaces; provide a jobNamespace query parameter to scope the request", ErrForbidden)
		}
	}

	jobList, err := client.GetAllModelTransferJobs(ctx, namespace, modelRegistryID, jobNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch model transfer jobs: %w", err)
	}

	transferJobs := make([]models.ModelTransferJob, 0, len(jobList.Items))
	jobNamesByNamespace := make(map[string][]string)
	for _, job := range jobList.Items {
		if job.DeletionTimestamp != nil {
			continue
		}
		transferJobs = append(transferJobs, convertK8sJobToModel(&job))
		jobNamesByNamespace[job.Namespace] = append(jobNamesByNamespace[job.Namespace], job.Name)
	}

	if len(transferJobs) > 0 {
		type terminationResult struct {
			RegisteredModel *struct {
				ID string `json:"id"`
			} `json:"RegisteredModel"`
			ModelVersion *struct {
				ID string `json:"id"`
			} `json:"ModelVersion"`
			ModelArtifact *struct {
				ID string `json:"id"`
			} `json:"ModelArtifact"`
		}

		podErrorsByJob := make(map[string]string)
		podTerminationByJob := make(map[string]*terminationResult)

		for ns, jobNames := range jobNamesByNamespace {
			podList, err := client.GetTransferJobPods(ctx, ns, jobNames)
			if err != nil {
				logger.Warn("failed to fetch pods for transfer jobs", "namespace", ns, "error", err)
				continue
			}
			if len(podList.Items) == 0 {
				continue
			}
			for _, pod := range podList.Items {
				jobName := pod.Labels["job-name"]
				key := ns + "/" + jobName

				for _, cs := range pod.Status.ContainerStatuses {
					if cs.State.Waiting != nil {
						reason := cs.State.Waiting.Reason
						if reason == "ImagePullBackOff" || reason == "ErrImagePull" ||
							reason == "CrashLoopBackOff" || reason == "CreateContainerConfigError" ||
							reason == "InvalidImageName" {
							msg := cs.State.Waiting.Message
							if msg == "" {
								msg = reason
							}
							podErrorsByJob[key] = fmt.Sprintf("%s: %s", reason, msg)
							break
						}
					}
					if cs.State.Terminated != nil {
						if cs.State.Terminated.ExitCode != 0 {
							msg := cs.State.Terminated.Message
							if msg == "" {
								msg = cs.State.Terminated.Reason
							}
							podErrorsByJob[key] = fmt.Sprintf("Container exited with code %d: %s", cs.State.Terminated.ExitCode, msg)
						}
						if cs.State.Terminated.Message != "" {
							var result terminationResult
							if err := json.Unmarshal([]byte(cs.State.Terminated.Message), &result); err == nil {
								podTerminationByJob[key] = &result
							}
						}
						break
					}
				}
			}
		}

		for i := range transferJobs {
			key := transferJobs[i].Namespace + "/" + transferJobs[i].Name
			if errMsg, ok := podErrorsByJob[key]; ok {
				if transferJobs[i].ErrorMessage == "" {
					transferJobs[i].ErrorMessage = errMsg
				}
				if transferJobs[i].Status == models.ModelTransferJobStatusRunning ||
					transferJobs[i].Status == models.ModelTransferJobStatusPending {
					transferJobs[i].Status = models.ModelTransferJobStatusFailed
				}
			}
			if result, ok := podTerminationByJob[key]; ok {
				if result.RegisteredModel != nil && result.RegisteredModel.ID != "" {
					transferJobs[i].RegisteredModelId = result.RegisteredModel.ID
				}
				if result.ModelVersion != nil && result.ModelVersion.ID != "" {
					transferJobs[i].ModelVersionId = result.ModelVersion.ID
				}
				if result.ModelArtifact != nil && result.ModelArtifact.ID != "" {
					transferJobs[i].ModelArtifactId = result.ModelArtifact.ID
				}
			}
		}
	}

	return &models.ModelTransferJobList{
		Items:    transferJobs,
		Size:     len(transferJobs),
		PageSize: len(transferJobs),
	}, nil
}

func (m *ModelRegistryRepository) GetModelTransferJob(ctx context.Context, client k8s.KubernetesClientInterface, namespace string, jobName string, modelRegistryID string) (*models.ModelTransferJob, error) {
	job, err := client.GetModelTransferJob(ctx, namespace, jobName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("%w: %s", ErrJobNotFound, jobName)
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	jobRegistry := job.Labels["modelregistry.kubeflow.org/model-registry-name"]
	if jobRegistry != modelRegistryID {
		return nil, fmt.Errorf("%w: %s", ErrJobNotFound, jobName)
	}

	result := convertK8sJobToModel(job)
	return &result, nil
}

func (m *ModelRegistryRepository) GetModelTransferJobEvents(ctx context.Context, client k8s.KubernetesClientInterface, jobNamespace string, jobName string, modelRegistryID string) ([]models.ModelTransferJobEvent, error) {
	job, err := client.GetModelTransferJob(ctx, jobNamespace, jobName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("%w: %s", ErrJobNotFound, jobName)
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	jobRegistry := job.Labels["modelregistry.kubeflow.org/model-registry-name"]
	if jobRegistry != modelRegistryID {
		return nil, fmt.Errorf("%w: %s", ErrJobNotFound, jobName)
	}

	podList, err := client.GetTransferJobPods(ctx, jobNamespace, []string{jobName})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pods for transfer job: %w", err)
	}

	if len(podList.Items) == 0 {
		return []models.ModelTransferJobEvent{}, nil
	}

	podNames := make([]string, 0, len(podList.Items))
	for _, pod := range podList.Items {
		podNames = append(podNames, pod.Name)
	}

	eventList, err := client.GetEventsForPods(ctx, jobNamespace, podNames)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events for pods: %w", err)
	}

	return convertK8sEventsToModelEvents(eventList), nil
}

func convertK8sEventsToModelEvents(eventList *corev1.EventList) []models.ModelTransferJobEvent {
	events := make([]models.ModelTransferJobEvent, 0, len(eventList.Items))
	for _, event := range eventList.Items {
		ts := event.LastTimestamp.Time
		if ts.IsZero() {
			ts = event.EventTime.Time
		}
		if ts.IsZero() {
			ts = event.FirstTimestamp.Time
		}
		events = append(events, models.ModelTransferJobEvent{
			Timestamp: ts.Format("2006-01-02T15:04:05Z"),
			Type:      event.Type,
			Reason:    event.Reason,
			Message:   event.Message,
		})
	}
	return events
}

func (m *ModelRegistryRepository) CreateModelTransferJob(ctx context.Context, client k8s.KubernetesClientInterface, namespace string, payload models.ModelTransferJob, modelRegistryID string, isFederatedMode bool, podNamespace string) (*models.ModelTransferJob, error) {
	return m.createModelTransferJobResources(ctx, client, namespace, payload, modelRegistryID, "", isFederatedMode, podNamespace)
}

func (m *ModelRegistryRepository) createModelTransferJobResources(
	ctx context.Context,
	client k8s.KubernetesClientInterface,
	namespace string,
	payload models.ModelTransferJob,
	modelRegistryID string,
	existingDestSecretName string,
	isFederatedMode bool,
	podNamespace string,
) (*models.ModelTransferJob, error) {
	payload.Source.Bucket = strings.TrimSpace(payload.Source.Bucket)
	payload.Source.Key = strings.TrimSpace(payload.Source.Key)
	payload.Source.URI = strings.TrimSpace(payload.Source.URI)
	payload.Destination.URI = strings.TrimSpace(payload.Destination.URI)
	payload.Destination.Registry = strings.TrimSpace(payload.Destination.Registry)

	skipDestCredsValidation := existingDestSecretName != ""
	if err := validateCreatePayload(payload, skipDestCredsValidation); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	if payload.Destination.Registry == "" && payload.Destination.URI != "" {
		payload.Destination.Registry, _ = extractRegistryFromURI(payload.Destination.URI)
	}

	logger := helper.GetContextLogger(ctx)

	modelRegistryAddress, err := m.getModelRegistryAddress(ctx, client, namespace, modelRegistryID, isFederatedMode)
	if err != nil {
		return nil, err
	}

	jobID := uuid.NewString()
	jobName := payload.Name

	var configMapName, sourceSecretName, destSecretName string
	destSecretName = existingDestSecretName

	configMap := buildModelMetadataConfigMap(jobName+"-metadata-configmap-", payload, jobID, jobName)
	configMapCreated, err := client.CreateConfigMap(ctx, payload.Namespace, configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create metadata configmap: %w", err)
	}
	configMapName = configMapCreated.Name

	if payload.Source.Type == models.ModelTransferJobSourceTypeS3 {
		sourceSecret := buildSourceSecret(jobName+"-source-creds-", payload, jobID)
		sourceSecretCreated, err := client.CreateSecret(ctx, payload.Namespace, sourceSecret)
		if err != nil {
			cleanupCreatedResources(ctx, client, payload.Namespace, configMapName, "", "")
			return nil, fmt.Errorf("failed to create source secret: %w", err)
		}
		sourceSecretName = sourceSecretCreated.Name
	}

	if existingDestSecretName == "" {
		destSecret, err := buildDestinationSecret(jobName+"-dest-creds-", payload, jobID)
		if err != nil {
			cleanupCreatedResources(ctx, client, payload.Namespace, configMapName, sourceSecretName, "")
			return nil, fmt.Errorf("failed to build destination secret: %w", err)
		}
		destSecretCreated, err := client.CreateSecret(ctx, payload.Namespace, destSecret)
		if err != nil {
			cleanupCreatedResources(ctx, client, payload.Namespace, configMapName, sourceSecretName, "")
			return nil, fmt.Errorf("failed to create destination secret: %w", err)
		}
		destSecretName = destSecretCreated.Name
	}

	imageURI := resolveAsyncUploadImage(ctx, client, isFederatedMode, podNamespace)
	job := buildK8sJob(jobName, jobID, payload, configMapName, sourceSecretName, destSecretName, modelRegistryAddress, modelRegistryID, imageURI)
	jobCreated, err := client.CreateModelTransferJob(ctx, payload.Namespace, job)
	if err != nil {
		cleanupCreatedResources(ctx, client, payload.Namespace, configMapName, sourceSecretName, destSecretName)
		if apierrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("%w: job '%s' already exists", ErrJobValidationFailed, jobName)
		}
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	if jobCreated == nil {
		logger.Error("created job is nil - unexpected K8s client behavior")
		cleanupCreatedResources(ctx, client, payload.Namespace, configMapName, sourceSecretName, destSecretName)
		if err := client.DeleteModelTransferJob(ctx, payload.Namespace, jobName); err != nil && !apierrors.IsNotFound(err) {
			logger.Warn("failed to cleanup job after nil response", "jobName", jobName, "error", err)
		}
		return nil, fmt.Errorf("unexpected Kubernetes API behavior: created job object was nil")
	}

	ownerRef := metav1.OwnerReference{
		APIVersion: "batch/v1",
		Kind:       "Job",
		Name:       jobCreated.Name,
		UID:        jobCreated.UID,
	}

	if err := client.PatchConfigMapOwnerReference(ctx, payload.Namespace, configMapName, ownerRef); err != nil {
		logger.Warn("failed to set ownerReference on configmap", "error", err)
	}
	if sourceSecretName != "" {
		if err := client.PatchSecretOwnerReference(ctx, payload.Namespace, sourceSecretName, ownerRef); err != nil {
			logger.Warn("failed to set ownerReference on source secret", "error", err)
		}
	}
	if err := client.PatchSecretOwnerReference(ctx, payload.Namespace, destSecretName, ownerRef); err != nil {
		logger.Warn("failed to set ownerReference on destination secret", "error", err)
	}

	result := convertK8sJobToModel(jobCreated)
	return &result, nil
}

func (m *ModelRegistryRepository) UpdateModelTransferJob(
	ctx context.Context,
	client k8s.KubernetesClientInterface,
	namespace string,
	oldJobName string,
	newPayload models.ModelTransferJob,
	deleteOldJob bool,
	modelRegistryID string,
	isFederatedMode bool,
	podNamespace string,
) (*models.ModelTransferJob, error) {

	logger := helper.GetContextLogger(ctx)

	if newPayload.Namespace == "" {
		return nil, fmt.Errorf("%w: namespace is required in the request body for retry", ErrJobValidationFailed)
	}

	newJobName := newPayload.Name
	if newJobName == "" {
		return nil, fmt.Errorf("%w: new job name is required", ErrJobValidationFailed)
	}
	if len(newJobName) > 63 {
		return nil, fmt.Errorf("%w: job name must be 63 characters or less", ErrJobValidationFailed)
	}
	if errs := validation.IsDNS1123Subdomain(newJobName); len(errs) > 0 {
		return nil, fmt.Errorf("%w: invalid job name: %s", ErrJobValidationFailed, strings.Join(errs, ", "))
	}
	if newJobName == oldJobName {
		return nil, fmt.Errorf("%w: new job name must be different from old job name", ErrJobValidationFailed)
	}

	oldJob, err := client.GetModelTransferJob(ctx, newPayload.Namespace, oldJobName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("%w: %s", ErrJobNotFound, oldJobName)
		}
		return nil, fmt.Errorf("failed to get existing job: %w", err)
	}

	oldAnnotations := oldJob.Annotations
	if oldAnnotations == nil {
		oldAnnotations = map[string]string{}
	}

	jobRegistry := oldJob.Labels["modelregistry.kubeflow.org/model-registry-name"]
	if jobRegistry != modelRegistryID {
		return nil, fmt.Errorf("%w: %s", ErrJobNotFound, oldJobName)
	}

	if !isK8sJobFailed(oldJob) {
		return nil, fmt.Errorf("%w: retry is only allowed for failed jobs; current job has not failed", ErrJobValidationFailed)
	}

	oldConfigMapName := oldAnnotations["modelregistry.kubeflow.org/configmap-name"]
	if oldConfigMapName == "" {
		return nil, fmt.Errorf("old job missing required annotation: configmap-name (job may not have been created via this API)")
	}

	oldDestSecretName := oldAnnotations["modelregistry.kubeflow.org/dest-secret"]
	if oldDestSecretName == "" {
		return nil, fmt.Errorf("old job missing required annotation: dest-secret")
	}

	oldSourceSecretName := oldAnnotations["modelregistry.kubeflow.org/source-secret"]

	// Recover metadata from annotations
	recoverEnumFromAnnotation(&newPayload.Source.Type, oldAnnotations, "modelregistry.kubeflow.org/source-type")
	recoverFromAnnotation(&newPayload.Source.Bucket, oldAnnotations, "modelregistry.kubeflow.org/source-bucket")
	recoverFromAnnotation(&newPayload.Source.Key, oldAnnotations, "modelregistry.kubeflow.org/source-key")
	recoverFromAnnotation(&newPayload.Source.URI, oldAnnotations, "modelregistry.kubeflow.org/source-uri")
	recoverEnumFromAnnotation(&newPayload.Destination.Type, oldAnnotations, "modelregistry.kubeflow.org/dest-type")
	recoverFromAnnotation(&newPayload.Destination.Registry, oldAnnotations, "modelregistry.kubeflow.org/dest-registry")
	recoverFromAnnotation(&newPayload.Destination.URI, oldAnnotations, "modelregistry.kubeflow.org/dest-uri")
	recoverEnumFromAnnotation(&newPayload.UploadIntent, oldAnnotations, "modelregistry.kubeflow.org/upload-intent")
	recoverFromAnnotation(&newPayload.RegisteredModelName, oldAnnotations, "modelregistry.kubeflow.org/model-name")
	recoverFromAnnotation(&newPayload.ModelVersionName, oldAnnotations, "modelregistry.kubeflow.org/version-name")
	recoverFromAnnotation(&newPayload.RegisteredModelId, oldAnnotations, "modelregistry.kubeflow.org/registered-model-id")
	recoverFromAnnotation(&newPayload.ModelVersionId, oldAnnotations, "modelregistry.kubeflow.org/model-version-id")
	recoverFromAnnotation(&newPayload.ModelArtifactId, oldAnnotations, "modelregistry.kubeflow.org/model-artifact-id")
	recoverFromAnnotation(&newPayload.Author, oldAnnotations, "modelregistry.kubeflow.org/author")
	recoverFromAnnotation(&newPayload.Description, oldAnnotations, "modelregistry.kubeflow.org/description")
	recoverFromAnnotation(&newPayload.JobDisplayName, oldAnnotations, "modelregistry.kubeflow.org/display-name")
	if newPayload.JobDisplayName == "" {
		newPayload.JobDisplayName = oldJobName
	}

	oldConfigMap, err := client.GetConfigMap(ctx, newPayload.Namespace, oldConfigMapName)
	if err != nil {
		logger.Warn("failed to get old configmap", "name", oldConfigMapName, "error", err)
	}

	var oldSourceSecret *corev1.Secret
	if oldSourceSecretName != "" {
		oldSourceSecret, err = client.GetSecret(ctx, newPayload.Namespace, oldSourceSecretName)
		if err != nil {
			logger.Warn("failed to get old source secret", "name", oldSourceSecretName, "error", err)
		}
	}

	oldDestSecret, err := client.GetSecret(ctx, newPayload.Namespace, oldDestSecretName)
	if err != nil {
		return nil, fmt.Errorf("failed to get old dest secret: %w", err)
	}

	if newPayload.Source.Type == models.ModelTransferJobSourceTypeS3 {
		if newPayload.Source.AwsAccessKeyId == "" && oldSourceSecret != nil && oldSourceSecret.Data != nil {
			if val, ok := oldSourceSecret.Data["AWS_ACCESS_KEY_ID"]; ok {
				newPayload.Source.AwsAccessKeyId = string(val)
			}

			if val, ok := oldSourceSecret.Data["AWS_SECRET_ACCESS_KEY"]; ok {
				newPayload.Source.AwsSecretAccessKey = string(val)
			}

			if newPayload.Source.Region == "" {
				if val, ok := oldSourceSecret.Data["AWS_DEFAULT_REGION"]; ok {
					newPayload.Source.Region = string(val)
				}
			}
			if newPayload.Source.Endpoint == "" {
				if val, ok := oldSourceSecret.Data["AWS_S3_ENDPOINT"]; ok {
					newPayload.Source.Endpoint = string(val)
				}
			}
		}
	}

	reuseDestCreds := (newPayload.Destination.Username == "" && newPayload.Destination.Password == "") &&
		oldDestSecret != nil && len(oldDestSecret.Data[".dockerconfigjson"]) > 0

	var existingDestSecretName string
	if reuseDestCreds {
		jobID := uuid.NewString()
		clonedSecret := cloneDestSecretFromExisting(newPayload.Name+"-dest-creds-", newPayload.Namespace, jobID, oldDestSecret)
		if clonedSecret == nil {
			return nil, fmt.Errorf("could not clone destination secret for reuse")
		}
		destSecretCreated, err := client.CreateSecret(ctx, newPayload.Namespace, clonedSecret)
		if err != nil {
			return nil, fmt.Errorf("failed to create cloned destination secret: %w", err)
		}
		existingDestSecretName = destSecretCreated.Name
	}

	if oldConfigMap != nil && oldConfigMap.Data != nil {
		if newPayload.ModelArtifactName == "" {
			if val, ok := oldConfigMap.Data["ModelArtifact.name"]; ok {
				newPayload.ModelArtifactName = val
			}
		}
		if newPayload.VersionDescription == "" {
			if val, ok := oldConfigMap.Data["ModelVersion.description"]; ok {
				newPayload.VersionDescription = val
			}
		}
		if newPayload.SourceModelFormat == "" {
			if val, ok := oldConfigMap.Data["ModelArtifact.model_format_name"]; ok {
				newPayload.SourceModelFormat = val
			}
		}
		if newPayload.SourceModelFormatVersion == "" {
			if val, ok := oldConfigMap.Data["ModelArtifact.model_format_version"]; ok {
				newPayload.SourceModelFormatVersion = val
			}
		}
		// Recover custom properties if not provided in new payload
		if newPayload.ModelCustomProperties == nil {
			if val, ok := oldConfigMap.Data["RegisteredModel.customProperties"]; ok && val != "" {
				var props map[string]interface{}
				if err := json.Unmarshal([]byte(val), &props); err == nil {
					newPayload.ModelCustomProperties = props
				} else {
					logger.Warn("failed to unmarshal model custom properties", "key", "RegisteredModel.customProperties", "error", err)
				}
			}
		}
		if newPayload.VersionCustomProperties == nil {
			if val, ok := oldConfigMap.Data["ModelVersion.customProperties"]; ok && val != "" {
				var props map[string]interface{}
				if err := json.Unmarshal([]byte(val), &props); err == nil {
					newPayload.VersionCustomProperties = props
				} else {
					logger.Warn("failed to unmarshal version custom properties", "key", "ModelVersion.customProperties", "error", err)
				}
			}
		}
	}

	if err := validateCreatePayload(newPayload, reuseDestCreds); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	result, err := m.createModelTransferJobResources(ctx, client, namespace, newPayload, modelRegistryID, existingDestSecretName, isFederatedMode, podNamespace)
	if err != nil {
		if reuseDestCreds && existingDestSecretName != "" {
			if delErr := client.DeleteSecret(ctx, newPayload.Namespace, existingDestSecretName); delErr != nil {
				logger.Warn("failed to cleanup cloned destination secret after create failure", "name", existingDestSecretName, "error", delErr)
			}
		}
		return nil, err
	}

	if deleteOldJob {
		if err := client.DeleteModelTransferJob(ctx, newPayload.Namespace, oldJobName); err != nil {
			logger.Warn("failed to delete old job", "name", oldJobName, "error", err)
		}
	}
	return result, nil
}

func (m *ModelRegistryRepository) DeleteModelTransferJob(ctx context.Context, client k8s.KubernetesClientInterface, namespace string, jobName string, modelRegistryID string) (*models.ModelTransferJob, error) {
	if modelRegistryID == "" {
		return nil, fmt.Errorf("%w: model registry name is required", ErrJobValidationFailed)
	}
	job, err := client.GetModelTransferJob(ctx, namespace, jobName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("%w: %s", ErrJobNotFound, jobName)
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	jobRegistry := job.Labels["modelregistry.kubeflow.org/model-registry-name"]
	if jobRegistry != modelRegistryID {
		return nil, fmt.Errorf("%w: %s", ErrJobNotFound, jobName)
	}

	if err := client.DeleteModelTransferJob(ctx, namespace, jobName); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("%w: %s", ErrJobNotFound, jobName)
		}
		return nil, fmt.Errorf("failed to delete model transfer job %s: %w", jobName, err)
	}

	result := convertK8sJobToModel(job)
	return &result, nil
}

// getModelRegistryAddress returns the registry address for use in transfer job env.
// It uses federated/external address when available (e.g. Route URL from Service annotation)
// so the job pod can reach the registry via the ingress path when NetworkPolicy restricts direct ClusterIP access.
func (m *ModelRegistryRepository) getModelRegistryAddress(ctx context.Context, client k8s.KubernetesClientInterface, namespace, modelRegistryID string, isFederatedMode bool) (string, error) {
	modelRegistry, err := m.GetModelRegistryWithMode(ctx, client, namespace, modelRegistryID, isFederatedMode)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", ErrModelRegistryNotFound
		}
		return "", fmt.Errorf("failed to get model registry: %w", err)
	}
	return modelRegistry.ServerAddress, nil
}

func resolveAsyncUploadImage(ctx context.Context, client k8s.KubernetesClientInterface, isFederatedMode bool, podNamespace string) string {
	if !isFederatedMode || podNamespace == "" {
		return DefaultAsyncUploadImage
	}
	logger := helper.GetContextLogger(ctx)
	cm, err := client.GetConfigMap(ctx, podNamespace, asyncUploadConfigMapName)
	if err != nil {
		logger.Info("ConfigMap not found, using default async-upload image",
			"configmap", asyncUploadConfigMapName, "error", err)
		return DefaultAsyncUploadImage
	}
	if img, ok := cm.Data[asyncUploadConfigMapKey]; ok && strings.TrimSpace(img) != "" {
		return strings.TrimSpace(img)
	}
	logger.Warn("ConfigMap key not found or empty, using default async-upload image",
		"configmap", asyncUploadConfigMapName, "key", asyncUploadConfigMapKey)
	return DefaultAsyncUploadImage
}

func buildK8sJob(jobName, jobID string, payload models.ModelTransferJob,
	configMapName, sourceSecretName, destSecretName, modelRegistryAddress, modelRegistryID, imageURI string) *batchv1.Job {

	backoffLimit := int32(3)

	volumes := []corev1.Volume{
		{
			Name: "destination-credentials",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: destSecretName,
					Items: []corev1.KeyToPath{
						{Key: ".dockerconfigjson", Path: ".dockerconfigjson"},
					},
				},
			},
		},
		{
			Name: "model-metadata",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configMapName,
					},
				},
			},
		},
	}

	volumeMounts := []corev1.VolumeMount{
		{Name: "destination-credentials", MountPath: "/opt/creds/destination", ReadOnly: true},
		{Name: "model-metadata", MountPath: "/etc/model-metadata", ReadOnly: true},
	}
	registryPort, registrySecure := parseRegistryServerAddress(modelRegistryAddress)
	envVars := []corev1.EnvVar{
		{Name: "MODEL_SYNC_SOURCE_TYPE", Value: string(payload.Source.Type)},
		{Name: "MODEL_SYNC_DESTINATION_TYPE", Value: string(payload.Destination.Type)},
		{Name: "MODEL_SYNC_DESTINATION_OCI_URI", Value: payload.Destination.URI},
		{Name: "MODEL_SYNC_DESTINATION_OCI_REGISTRY", Value: payload.Destination.Registry},
		{Name: "MODEL_SYNC_DESTINATION_OCI_CREDENTIALS_PATH", Value: "/opt/creds/destination/.dockerconfigjson"},
		{Name: "MODEL_SYNC_REGISTRY_SERVER_ADDRESS", Value: registryOriginOnly(modelRegistryAddress)},
		{Name: "MODEL_SYNC_REGISTRY_PORT", Value: registryPort},
		{Name: "MODEL_SYNC_REGISTRY_IS_SECURE", Value: strconv.FormatBool(registrySecure)},
		{Name: "MODEL_SYNC_METADATA_CONFIGMAP_PATH", Value: "/etc/model-metadata"},
		{Name: "MODEL_SYNC_MODEL_UPLOAD_INTENT", Value: string(payload.UploadIntent)},
	}

	if payload.UploadIntent == models.ModelTransferJobUploadIntentCreateVersion && payload.RegisteredModelId != "" {
		envVars = append(envVars, corev1.EnvVar{Name: "MODEL_SYNC_MODEL_ID", Value: payload.RegisteredModelId})
	}

	if payload.Destination.Type == models.ModelTransferJobDestinationTypeOCI && payload.Destination.Registry == "quay.io" {
		envVars = append(envVars, corev1.EnvVar{Name: "MODEL_SYNC_DESTINATION_OCI_BASE_IMAGE", Value: "quay.io/quay/busybox:latest"})
	}

	annotations := map[string]string{
		"modelregistry.kubeflow.org/display-name":        payload.JobDisplayName,
		"modelregistry.kubeflow.org/source-type":         string(payload.Source.Type),
		"modelregistry.kubeflow.org/dest-type":           string(payload.Destination.Type),
		"modelregistry.kubeflow.org/dest-uri":            payload.Destination.URI,
		"modelregistry.kubeflow.org/dest-registry":       payload.Destination.Registry,
		"modelregistry.kubeflow.org/upload-intent":       string(payload.UploadIntent),
		"modelregistry.kubeflow.org/model-name":          payload.RegisteredModelName,
		"modelregistry.kubeflow.org/version-name":        payload.ModelVersionName,
		"modelregistry.kubeflow.org/artifact-name":       payload.ModelVersionName,
		"modelregistry.kubeflow.org/description":         payload.Description,
		"modelregistry.kubeflow.org/author":              payload.Author,
		"modelregistry.kubeflow.org/configmap-name":      configMapName,
		"modelregistry.kubeflow.org/dest-secret":         destSecretName,
		"modelregistry.kubeflow.org/registered-model-id": payload.RegisteredModelId,
		"modelregistry.kubeflow.org/model-version-id":    payload.ModelVersionId,
		"modelregistry.kubeflow.org/model-artifact-id":   payload.ModelArtifactId,
	}

	switch payload.Source.Type {
	case models.ModelTransferJobSourceTypeS3:
		volumes = append(volumes, corev1.Volume{
			Name: "source-credentials",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: sourceSecretName,
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name: "source-credentials", MountPath: "/opt/creds/source", ReadOnly: true,
		})
		envVars = append(envVars,
			corev1.EnvVar{Name: "MODEL_SYNC_SOURCE_AWS_KEY", Value: payload.Source.Key},
			corev1.EnvVar{Name: "MODEL_SYNC_SOURCE_S3_CREDENTIALS_PATH", Value: "/opt/creds/source"},
		)
		annotations["modelregistry.kubeflow.org/source-bucket"] = payload.Source.Bucket
		annotations["modelregistry.kubeflow.org/source-key"] = payload.Source.Key
		annotations["modelregistry.kubeflow.org/source-secret"] = sourceSecretName
		annotations["modelregistry.kubeflow.org/model-registry-name"] = modelRegistryID

	case models.ModelTransferJobSourceTypeURI:
		envVars = append(envVars,
			corev1.EnvVar{Name: "MODEL_SYNC_SOURCE_URI", Value: payload.Source.URI},
		)
		annotations["modelregistry.kubeflow.org/source-uri"] = payload.Source.URI
		annotations["modelregistry.kubeflow.org/model-registry-name"] = modelRegistryID
	}

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: payload.Namespace,
			Labels: map[string]string{
				"modelregistry.kubeflow.org/job-type":            "async-upload",
				"modelregistry.kubeflow.org/job-id":              jobID,
				"modelregistry.kubeflow.org/model-registry-name": modelRegistryID,
			},
			Annotations: annotations,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"modelregistry.kubeflow.org/job-type": "async-upload",
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: func() *bool { b := true; return &b }(),
					},
					Volumes: volumes,
					Containers: []corev1.Container{
						{
							Name:            "async-upload",
							Image:           imageURI,
							ImagePullPolicy: corev1.PullIfNotPresent,
							VolumeMounts:    volumeMounts,
							Env:             envVars,
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: func() *bool { b := false; return &b }(),
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
								SeccompProfile: &corev1.SeccompProfile{
									Type: corev1.SeccompProfileTypeRuntimeDefault,
								},
							},
						},
					},
				},
			},
		},
	}
}

func convertK8sJobToModel(job *batchv1.Job) models.ModelTransferJob {
	if job == nil {
		return models.ModelTransferJob{}
	}

	annotations := job.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}
	labels := job.Labels
	if labels == nil {
		labels = map[string]string{}
	}

	status := models.ModelTransferJobStatusPending
	if job.Status.Succeeded > 0 {
		status = models.ModelTransferJobStatusCompleted
	} else if job.Status.Failed > 0 {
		status = models.ModelTransferJobStatusFailed
	} else if job.Status.Active > 0 {
		status = models.ModelTransferJobStatusRunning
	}

	errorMessage := ""
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			errorMessage = condition.Message
			break
		}
	}

	lastUpdateTime := fmt.Sprintf("%d", job.CreationTimestamp.UnixMilli())
	if job.Status.CompletionTime != nil {
		lastUpdateTime = fmt.Sprintf("%d", job.Status.CompletionTime.UnixMilli())
	}

	return models.ModelTransferJob{
		Id:             labels["modelregistry.kubeflow.org/job-id"],
		Name:           job.Name,
		JobDisplayName: annotations["modelregistry.kubeflow.org/display-name"],
		Description:    annotations["modelregistry.kubeflow.org/description"],
		Source: models.ModelTransferJobSource{
			Type:   models.ModelTransferJobSourceType(annotations["modelregistry.kubeflow.org/source-type"]),
			Bucket: annotations["modelregistry.kubeflow.org/source-bucket"],
			Key:    annotations["modelregistry.kubeflow.org/source-key"],
			URI:    annotations["modelregistry.kubeflow.org/source-uri"],
		},
		Destination: models.ModelTransferJobDestination{
			Type:     models.ModelTransferJobDestinationType(annotations["modelregistry.kubeflow.org/dest-type"]),
			URI:      annotations["modelregistry.kubeflow.org/dest-uri"],
			Registry: annotations["modelregistry.kubeflow.org/dest-registry"],
		},
		UploadIntent:             models.ModelTransferJobUploadIntent(annotations["modelregistry.kubeflow.org/upload-intent"]),
		RegisteredModelName:      annotations["modelregistry.kubeflow.org/model-name"],
		ModelVersionName:         annotations["modelregistry.kubeflow.org/version-name"],
		ModelArtifactName:        annotations["modelregistry.kubeflow.org/version-name"],
		RegisteredModelId:        annotations["modelregistry.kubeflow.org/registered-model-id"],
		ModelVersionId:           annotations["modelregistry.kubeflow.org/model-version-id"],
		ModelArtifactId:          annotations["modelregistry.kubeflow.org/model-artifact-id"],
		Author:                   annotations["modelregistry.kubeflow.org/author"],
		Status:                   status,
		ErrorMessage:             errorMessage,
		CreateTimeSinceEpoch:     fmt.Sprintf("%d", job.CreationTimestamp.UnixMilli()),
		LastUpdateTimeSinceEpoch: lastUpdateTime,
		Namespace:                job.Namespace,
		SourceSecretName:         annotations["modelregistry.kubeflow.org/source-secret"],
		DestSecretName:           annotations["modelregistry.kubeflow.org/dest-secret"],
	}
}

func buildModelMetadataConfigMap(generateNamePrefix string, payload models.ModelTransferJob, jobID string, jobName string) *corev1.ConfigMap {
	data := map[string]string{
		"ModelVersion.name":   payload.ModelVersionName,
		"ModelVersion.author": payload.Author,
		"ModelArtifact.name":  payload.ModelVersionName,
	}

	if payload.VersionDescription != "" {
		data["ModelVersion.description"] = payload.VersionDescription
		data["ModelArtifact.description"] = payload.VersionDescription
	}
	if payload.SourceModelFormat != "" {
		data["ModelArtifact.model_format_name"] = payload.SourceModelFormat
	}
	if payload.SourceModelFormatVersion != "" {
		data["ModelArtifact.model_format_version"] = payload.SourceModelFormatVersion
	}
	if len(payload.ModelCustomProperties) > 0 {
		if b, err := json.Marshal(payload.ModelCustomProperties); err == nil {
			data["RegisteredModel.customProperties"] = string(b)
		}
	}
	if len(payload.VersionCustomProperties) > 0 {
		if b, err := json.Marshal(payload.VersionCustomProperties); err == nil {
			data["ModelVersion.customProperties"] = string(b)
			data["ModelArtifact.customProperties"] = string(b)
		}
	}

	switch payload.UploadIntent {
	case models.ModelTransferJobUploadIntentCreateModel:
		data["RegisteredModel.name"] = payload.RegisteredModelName
		data["RegisteredModel.description"] = payload.Description
		data["RegisteredModel.owner"] = payload.Author
	case models.ModelTransferJobUploadIntentCreateVersion:
		data["RegisteredModel.id"] = payload.RegisteredModelId
	case models.ModelTransferJobUploadIntentUpdateArtifact:
		data["RegisteredModel.id"] = payload.RegisteredModelId
		data["ModelVersion.id"] = payload.ModelVersionId
		data["ModelArtifact.id"] = payload.ModelArtifactId
	}

	data["ModelArtifact.model_source_kind"] = "transfer_job"
	data["ModelArtifact.model_source_class"] = "async-upload"
	data["ModelArtifact.model_source_group"] = payload.Namespace
	data["ModelArtifact.model_source_name"] = jobName

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: generateNamePrefix,
			Namespace:    payload.Namespace,
			Labels: map[string]string{
				"modelregistry.kubeflow.org/job-type": "async-upload",
				"modelregistry.kubeflow.org/job-id":   jobID,
			},
		},
		Data: data,
	}
}

func buildSourceSecret(generateNamePrefix string, payload models.ModelTransferJob, jobID string) *corev1.Secret {
	stringData := map[string]string{
		"AWS_ACCESS_KEY_ID":     payload.Source.AwsAccessKeyId,
		"AWS_SECRET_ACCESS_KEY": payload.Source.AwsSecretAccessKey,
		"AWS_S3_ENDPOINT":       payload.Source.Endpoint,
		"AWS_S3_BUCKET":         payload.Source.Bucket,
	}

	if payload.Source.Region != "" {
		stringData["AWS_DEFAULT_REGION"] = payload.Source.Region
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: generateNamePrefix,
			Namespace:    payload.Namespace,
			Labels: map[string]string{
				"modelregistry.kubeflow.org/job-type": "async-upload",
				"modelregistry.kubeflow.org/job-id":   jobID,
			},
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: stringData,
	}
}

func buildDestinationSecret(generateNamePrefix string, payload models.ModelTransferJob, jobID string) (*corev1.Secret, error) {
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", payload.Destination.Username, payload.Destination.Password)))

	registry := payload.Destination.Registry
	if registry == "" {
		registry, _ = extractRegistryFromURI(payload.Destination.URI)
	}

	dockerConfig := fmt.Sprintf(`{"auths":{"%s":{"auth":"%s"}}}`,
		registry, auth)

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: generateNamePrefix,
			Namespace:    payload.Namespace,
			Labels: map[string]string{
				"modelregistry.kubeflow.org/job-type": "async-upload",
				"modelregistry.kubeflow.org/job-id":   jobID,
			},
		},
		Type: corev1.SecretTypeDockerConfigJson,
		StringData: map[string]string{
			".dockerconfigjson": dockerConfig,
		},
	}, nil
}

func validateCreatePayload(payload models.ModelTransferJob, skipDestCredsValidation bool) error {
	if payload.Name == "" {
		return fmt.Errorf("%w: job resource name is required", ErrJobValidationFailed)
	}

	if payload.JobDisplayName == "" {
		return fmt.Errorf("%w: job display name is required", ErrJobValidationFailed)
	}

	if payload.Namespace == "" {
		return fmt.Errorf("%w: job namespace is required", ErrJobValidationFailed)
	}

	if len(payload.Name) > 63 {
		return fmt.Errorf("%w: job name must be 63 characters or less (label limit)", ErrJobValidationFailed)
	}

	if errs := validation.IsDNS1123Subdomain(payload.Name); len(errs) > 0 {
		return fmt.Errorf("%w: %s", ErrJobValidationFailed, strings.Join(errs, ", "))
	}

	if payload.Source.Type == "" {
		return fmt.Errorf("%w: source type is required", ErrJobValidationFailed)
	}

	switch payload.Source.Type {
	case models.ModelTransferJobSourceTypeS3:
		if payload.Source.Bucket == "" {
			return fmt.Errorf("%w: source bucket is required for S3 source type", ErrJobValidationFailed)
		}
		if payload.Source.Key == "" {
			return fmt.Errorf("%w: source key is required for S3 source type", ErrJobValidationFailed)
		}
		if payload.Source.AwsAccessKeyId == "" {
			return fmt.Errorf("%w: AWS access key ID is required for S3 source type", ErrJobValidationFailed)
		}
		if payload.Source.AwsSecretAccessKey == "" {
			return fmt.Errorf("%w: AWS secret access key is required for S3 source type", ErrJobValidationFailed)
		}
	case models.ModelTransferJobSourceTypeURI:
		if payload.Source.URI == "" {
			return fmt.Errorf("%w: source URI is required for URI source type", ErrJobValidationFailed)
		}
	default:
		return fmt.Errorf("%w: invalid source type: %s", ErrJobValidationFailed, payload.Source.Type)
	}

	if payload.Destination.Type == "" {
		return fmt.Errorf("%w: destination type is required", ErrJobValidationFailed)
	}
	if payload.Destination.Type == models.ModelTransferJobDestinationTypeOCI {
		if payload.Destination.URI == "" {
			return fmt.Errorf("%w: destination URI is required for OCI destination type", ErrJobValidationFailed)
		}

		registry := payload.Destination.Registry
		if registry == "" {
			if _, err := extractRegistryFromURI(payload.Destination.URI); err != nil {
				return fmt.Errorf("%w: cannot extract registry from destination URI: %w", ErrJobValidationFailed, err)
			}
		}

		if !skipDestCredsValidation {
			if payload.Destination.Username == "" {
				return fmt.Errorf("%w: destination username is required for OCI destination type", ErrJobValidationFailed)
			}
			if payload.Destination.Password == "" {
				return fmt.Errorf("%w: destination password is required for OCI destination type", ErrJobValidationFailed)
			}
		}
	} else {
		return fmt.Errorf("%w: invalid destination type: %s", ErrJobValidationFailed, payload.Destination.Type)
	}

	if payload.UploadIntent == "" {
		return fmt.Errorf("%w: upload intent is required", ErrJobValidationFailed)
	}
	validIntents := map[models.ModelTransferJobUploadIntent]bool{
		models.ModelTransferJobUploadIntentCreateModel:    true,
		models.ModelTransferJobUploadIntentCreateVersion:  true,
		models.ModelTransferJobUploadIntentUpdateArtifact: true,
	}
	if !validIntents[payload.UploadIntent] {
		return fmt.Errorf("%w: invalid upload intent: %s", ErrJobValidationFailed, payload.UploadIntent)
	}

	switch payload.UploadIntent {
	case models.ModelTransferJobUploadIntentCreateModel:
		if payload.RegisteredModelName == "" {
			return fmt.Errorf("%w: registered model name is required for create_model intent", ErrJobValidationFailed)
		}
		if payload.ModelVersionName == "" {
			return fmt.Errorf("%w: model version name is required for create_model intent", ErrJobValidationFailed)
		}
	case models.ModelTransferJobUploadIntentCreateVersion:
		if payload.RegisteredModelId == "" {
			return fmt.Errorf("%w: registered model ID is required for create_version intent", ErrJobValidationFailed)
		}
		if payload.ModelVersionName == "" {
			return fmt.Errorf("%w: model version name is required for create_version intent", ErrJobValidationFailed)
		}
	case models.ModelTransferJobUploadIntentUpdateArtifact:
		if payload.ModelArtifactId == "" {
			return fmt.Errorf("%w: model artifact ID is required for update_artifact intent", ErrJobValidationFailed)
		}
	}

	return nil
}

func cleanupCreatedResources(ctx context.Context, client k8s.KubernetesClientInterface, namespace, configMapName, sourceSecretName, destSecretName string) {
	logger := helper.GetContextLogger(ctx)

	if configMapName != "" {
		if err := client.DeleteConfigMap(ctx, namespace, configMapName); err != nil {
			logger.Warn("failed to cleanup configmap", "name", configMapName, "error", err)
		}
	}
	if sourceSecretName != "" {
		if err := client.DeleteSecret(ctx, namespace, sourceSecretName); err != nil {
			logger.Warn("failed to cleanup source secret", "name", sourceSecretName, "error", err)
		}
	}
	if destSecretName != "" {
		if err := client.DeleteSecret(ctx, namespace, destSecretName); err != nil {
			logger.Warn("failed to cleanup dest secret", "name", destSecretName, "error", err)
		}
	}
}

func extractRegistryFromURI(uri string) (string, error) {
	parts := strings.Split(uri, "/")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0], nil
	}
	return "", fmt.Errorf("cannot extract registry from URI: %s", uri)
}

func parseRegistryServerAddress(serverAddress string) (port string, isSecure bool) {
	u, err := url.Parse(serverAddress)
	if err != nil {
		return "8080", false
	}
	port = u.Port()
	if port == "" {
		if strings.ToLower(u.Scheme) == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	isSecure = strings.ToLower(u.Scheme) == "https"
	return port, isSecure
}

func registryOriginOnly(serverAddress string) string {
	u, err := url.Parse(serverAddress)
	if err != nil {
		return serverAddress
	}
	host := u.Hostname()
	if host == "" {
		return serverAddress
	}
	scheme := u.Scheme
	if scheme == "" {
		scheme = "http"
	}

	// Ensure port is explicit so the Python ModelRegistry client (which uses
	// urlparse) does not append it after the path, producing a malformed URL.
	port := u.Port()
	if port == "" {
		if scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	hostPort := host + ":" + port

	// Preserve any routing prefix (e.g. /model-registry/<name>) that the
	// gateway needs, but strip the MR API path suffix (/api/...).
	path := u.Path
	if idx := strings.Index(path, "/api/"); idx >= 0 {
		path = path[:idx]
	}
	path = strings.TrimRight(path, "/")

	return scheme + "://" + hostPort + path
}

func cloneDestSecretFromExisting(generateNamePrefix, namespace, jobID string, oldSecret *corev1.Secret) *corev1.Secret {
	if oldSecret == nil {
		return nil
	}
	data := make(map[string][]byte)
	for k, v := range oldSecret.Data {
		data[k] = append([]byte(nil), v...)
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: generateNamePrefix,
			Namespace:    namespace,
			Labels: map[string]string{
				"modelregistry.kubeflow.org/job-type": "async-upload",
				"modelregistry.kubeflow.org/job-id":   jobID,
			},
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: data,
	}
}
