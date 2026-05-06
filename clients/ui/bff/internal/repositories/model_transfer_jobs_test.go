package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/kubeflow/hub/ui/bff/internal/constants"
	k8s "github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/mocks"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// fakeKubernetesClient is a lightweight test double for KubernetesClientInterface that only
// implements the methods used by GetAllModelTransferJobs. All other methods either return
// zero values or panic if unexpectedly called.
type fakeKubernetesClient struct {
	jobs              *batchv1.JobList
	podsByNamespace   map[string]*corev1.PodList
	jobsByNamespace   map[string]map[string]*batchv1.Job
	eventsByNamespace map[string]*corev1.EventList
}

// testContext returns a context with a RequestIdentity set, as required by GetAllModelTransferJobs.
func testContext() context.Context {
	identity := &k8s.RequestIdentity{UserID: "test-user", Groups: []string{"system:authenticated"}}
	return context.WithValue(context.Background(), constants.RequestIdentityKey, identity)
}

func mustGetSingleJob(t *testing.T, repo *ModelRegistryRepository, client k8s.KubernetesClientInterface, namespace, modelRegistryID string) models.ModelTransferJob {
	t.Helper()

	list, err := repo.GetAllModelTransferJobs(testContext(), client, namespace, modelRegistryID, "")
	if err != nil {
		t.Fatalf("GetAllModelTransferJobs returned error: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected 1 job, got %d", len(list.Items))
	}

	return list.Items[0]
}

func buildSingleJobFixture(jobName string, jobStatus batchv1.JobStatus, containerState corev1.ContainerState) *fakeKubernetesClient {
	job := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: "kubeflow",
		},
		Status: jobStatus,
	}

	jobs := &batchv1.JobList{
		Items: []batchv1.Job{job},
	}

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName + "-pod",
			Namespace: "kubeflow",
			Labels: map[string]string{
				"job-name": job.Name,
			},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					State: containerState,
				},
			},
		},
	}

	return &fakeKubernetesClient{
		jobs: jobs,
		podsByNamespace: map[string]*corev1.PodList{
			"kubeflow": {
				Items: []corev1.Pod{pod},
			},
		},
	}
}

func (f *fakeKubernetesClient) GetAllModelTransferJobs(ctx context.Context, namespace string, modelRegistryID string, jobNamespace string) (*batchv1.JobList, error) {
	if f.jobs == nil {
		return &batchv1.JobList{}, nil
	}
	return f.jobs, nil
}

func (f *fakeKubernetesClient) GetTransferJobPods(ctx context.Context, namespace string, jobNames []string) (*corev1.PodList, error) {
	if f.podsByNamespace == nil {
		return &corev1.PodList{}, nil
	}
	if pods, ok := f.podsByNamespace[namespace]; ok {
		return pods, nil
	}
	return &corev1.PodList{}, nil
}

// The remaining methods are not used by GetAllModelTransferJobs in these tests.

func (f *fakeKubernetesClient) GetServiceNames(ctx context.Context, namespace string) ([]string, error) {
	return nil, nil
}

func (f *fakeKubernetesClient) GetServiceDetailsByName(ctx context.Context, namespace, serviceName string, serviceType string) (k8s.ServiceDetails, error) {
	return k8s.ServiceDetails{}, nil
}

func (f *fakeKubernetesClient) GetServiceDetails(ctx context.Context, namespace string) ([]k8s.ServiceDetails, error) {
	return nil, nil
}

//nolint:staticcheck // Use corev1.Endpoints here to satisfy the existing KubernetesClientInterface, consistent with production code.
func (f *fakeKubernetesClient) GetServiceEndpoints(ctx context.Context, namespace, serviceName string) (*corev1.Endpoints, error) {
	return nil, nil
}

func (f *fakeKubernetesClient) GetNamespaces(ctx context.Context, identity *k8s.RequestIdentity) ([]corev1.Namespace, error) {
	return nil, nil
}

func (f *fakeKubernetesClient) CanListServicesInNamespace(ctx context.Context, identity *k8s.RequestIdentity, namespace string) (bool, error) {
	return false, nil
}

func (f *fakeKubernetesClient) CanAccessServiceInNamespace(ctx context.Context, identity *k8s.RequestIdentity, namespace, serviceName string) (bool, error) {
	return false, nil
}

func (f *fakeKubernetesClient) CanNamespaceAccessRegistry(ctx context.Context, identity *k8s.RequestIdentity, jobNamespace, registryName, registryNamespace string) (bool, error) {
	return false, nil
}

func (f *fakeKubernetesClient) GetSelfSubjectRulesReview(ctx context.Context, identity *k8s.RequestIdentity, namespace string) ([]string, error) {
	return nil, nil
}

func (f *fakeKubernetesClient) IsClusterAdmin(identity *k8s.RequestIdentity) (bool, error) {
	return false, nil
}

func (f *fakeKubernetesClient) BearerToken() (string, error) {
	return "", nil
}

func (f *fakeKubernetesClient) GetUser(identity *k8s.RequestIdentity) (string, error) {
	return "", nil
}

func (f *fakeKubernetesClient) GetGroups(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (f *fakeKubernetesClient) GetAllCatalogSourceConfigs(ctx context.Context, namespace string) (corev1.ConfigMap, corev1.ConfigMap, error) {
	return corev1.ConfigMap{}, corev1.ConfigMap{}, nil
}

func (f *fakeKubernetesClient) UpdateCatalogSourceConfig(ctx context.Context, namespace string, configMap *corev1.ConfigMap) error {
	return nil
}

func (f *fakeKubernetesClient) CreateSecret(ctx context.Context, namespace string, secret *corev1.Secret) (*corev1.Secret, error) {
	return nil, nil
}

func (f *fakeKubernetesClient) PatchSecret(ctx context.Context, namespace string, secretName string, data map[string]string) error {
	return nil
}

func (f *fakeKubernetesClient) DeleteSecret(ctx context.Context, namespace string, secretName string) error {
	return nil
}

func (f *fakeKubernetesClient) CreateModelTransferJob(ctx context.Context, namespace string, job *batchv1.Job) (*batchv1.Job, error) {
	return nil, nil
}

func (f *fakeKubernetesClient) GetEventsForPods(ctx context.Context, namespace string, podNames []string) (*corev1.EventList, error) {
	if f.eventsByNamespace == nil {
		return &corev1.EventList{}, nil
	}
	if events, ok := f.eventsByNamespace[namespace]; ok {
		return events, nil
	}
	return &corev1.EventList{}, nil
}

func (f *fakeKubernetesClient) DeleteModelTransferJob(ctx context.Context, namespace string, jobName string) error {
	return nil
}

func (f *fakeKubernetesClient) CreateConfigMap(ctx context.Context, namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	return nil, nil
}

func (f *fakeKubernetesClient) DeleteConfigMap(ctx context.Context, namespace string, name string) error {
	return nil
}

func (f *fakeKubernetesClient) GetModelTransferJob(ctx context.Context, namespace string, jobName string) (*batchv1.Job, error) {
	if f.jobsByNamespace != nil {
		if byNS, ok := f.jobsByNamespace[namespace]; ok {
			if job, ok := byNS[jobName]; ok {
				return job, nil
			}
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{Group: "batch", Resource: "jobs"}, jobName)
}

func (f *fakeKubernetesClient) GetConfigMap(ctx context.Context, namespace string, name string) (*corev1.ConfigMap, error) {
	return nil, nil
}

func (f *fakeKubernetesClient) GetSecret(ctx context.Context, namespace string, name string) (*corev1.Secret, error) {
	return nil, nil
}

func (f *fakeKubernetesClient) PatchSecretOwnerReference(ctx context.Context, namespace string, name string, ownerRef metav1.OwnerReference) error {
	return nil
}

func (f *fakeKubernetesClient) PatchConfigMapOwnerReference(ctx context.Context, namespace string, name string, ownerRef metav1.OwnerReference) error {
	return nil
}

func (f *fakeKubernetesClient) CanListJobsClusterWide(ctx context.Context, identity *k8s.RequestIdentity) (bool, error) {
	return true, nil
}

func TestGetAllModelTransferJobs_PodWaitingFailuresOverrideStatusToFailed(t *testing.T) {
	repo := NewModelRegistryRepository()

	waitReasons := []string{"ImagePullBackOff", "ErrImagePull", "CrashLoopBackOff", "CreateContainerConfigError"}

	for _, reason := range waitReasons {
		t.Run(reason, func(t *testing.T) {
			jobStatus := batchv1.JobStatus{
				Active: 1, // initial status: Running
			}
			containerState := corev1.ContainerState{
				Waiting: &corev1.ContainerStateWaiting{
					Reason:  reason,
					Message: "simulated waiting error",
				},
			}

			client := buildSingleJobFixture("job-waiting-"+reason, jobStatus, containerState)

			jobModel := mustGetSingleJob(t, repo, client, "kubeflow", "model-registry-id")
			if jobModel.Status != models.ModelTransferJobStatusFailed {
				t.Fatalf("expected job status Failed, got %s", jobModel.Status)
			}
			if jobModel.ErrorMessage == "" {
				t.Fatalf("expected error message to be set for reason %s", reason)
			}
		})
	}
}

func TestGetAllModelTransferJobs_TerminatedNonZeroExitCodeOverridesStatusAndMessage(t *testing.T) {
	repo := NewModelRegistryRepository()

	jobStatus := batchv1.JobStatus{
		Active: 1, // initial status: Running
	}
	containerState := corev1.ContainerState{
		Terminated: &corev1.ContainerStateTerminated{
			ExitCode: 1,
			Message:  "terminated due to error",
			Reason:   "Error",
		},
	}

	client := buildSingleJobFixture("job-terminated", jobStatus, containerState)

	jobModel := mustGetSingleJob(t, repo, client, "kubeflow", "model-registry-id")
	if jobModel.Status != models.ModelTransferJobStatusFailed {
		t.Fatalf("expected job status Failed, got %s", jobModel.Status)
	}
	expectedPrefix := "Container exited with code 1:"
	if jobModel.ErrorMessage == "" || jobModel.ErrorMessage[:len(expectedPrefix)] != expectedPrefix {
		t.Fatalf("expected error message to start with %q, got %q", expectedPrefix, jobModel.ErrorMessage)
	}
}

func TestGetAllModelTransferJobs_AlreadyFailedJobGetsTerminationMessage(t *testing.T) {
	repo := NewModelRegistryRepository()

	jobStatus := batchv1.JobStatus{
		Failed: 1, // K8s Job controller already marked it failed
	}
	containerState := corev1.ContainerState{
		Terminated: &corev1.ContainerStateTerminated{
			ExitCode: 1,
			Message:  "terminated due to error",
			Reason:   "Error",
		},
	}

	client := buildSingleJobFixture("job-already-failed", jobStatus, containerState)

	jobModel := mustGetSingleJob(t, repo, client, "kubeflow", "model-registry-id")
	if jobModel.Status != models.ModelTransferJobStatusFailed {
		t.Fatalf("expected job status Failed, got %s", jobModel.Status)
	}
	expectedPrefix := "Container exited with code 1:"
	if jobModel.ErrorMessage == "" || jobModel.ErrorMessage[:len(expectedPrefix)] != expectedPrefix {
		t.Fatalf("expected error message to start with %q, got %q", expectedPrefix, jobModel.ErrorMessage)
	}
}

func TestGetAllModelTransferJobs_AlreadyFailedJobPrefersTerminationOverEmptyCondition(t *testing.T) {
	repo := NewModelRegistryRepository()

	job := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job-failed-empty-condition",
			Namespace: "kubeflow",
		},
		Status: batchv1.JobStatus{
			Failed: 1,
			Conditions: []batchv1.JobCondition{
				{
					Type:   batchv1.JobFailed,
					Status: corev1.ConditionTrue,
					// Message intentionally empty — common real-world case
				},
			},
		},
	}

	jobs := &batchv1.JobList{Items: []batchv1.Job{job}}

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job-failed-empty-condition-pod",
			Namespace: "kubeflow",
			Labels:    map[string]string{"job-name": job.Name},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode: 1,
							Message:  "specific error from container",
						},
					},
				},
			},
		},
	}

	client := &fakeKubernetesClient{
		jobs: jobs,
		podsByNamespace: map[string]*corev1.PodList{
			"kubeflow": {Items: []corev1.Pod{pod}},
		},
	}

	jobModel := mustGetSingleJob(t, repo, client, "kubeflow", "model-registry-id")
	if jobModel.Status != models.ModelTransferJobStatusFailed {
		t.Fatalf("expected job status Failed, got %s", jobModel.Status)
	}
	expectedPrefix := "Container exited with code 1:"
	if jobModel.ErrorMessage == "" || jobModel.ErrorMessage[:len(expectedPrefix)] != expectedPrefix {
		t.Fatalf("expected error message from pod termination, got %q", jobModel.ErrorMessage)
	}
}

func TestGetAllModelTransferJobs_TerminationMessageParsesIDs(t *testing.T) {
	repo := NewModelRegistryRepository()

	terminationJSON := `{
  "RegisteredModel": { "id": "rm-123" },
  "ModelVersion":   { "id": "mv-456" },
  "ModelArtifact":  { "id": "ma-789" }
}`

	jobStatus := batchv1.JobStatus{
		Succeeded: 1, // Completed job
	}
	containerState := corev1.ContainerState{
		Terminated: &corev1.ContainerStateTerminated{
			ExitCode: 0,
			Message:  terminationJSON,
		},
	}

	client := buildSingleJobFixture("job-with-termination-json", jobStatus, containerState)

	jobModel := mustGetSingleJob(t, repo, client, "kubeflow", "model-registry-id")
	if jobModel.RegisteredModelId != "rm-123" {
		t.Fatalf("expected RegisteredModelId rm-123, got %q", jobModel.RegisteredModelId)
	}
	if jobModel.ModelVersionId != "mv-456" {
		t.Fatalf("expected ModelVersionId mv-456, got %q", jobModel.ModelVersionId)
	}
	if jobModel.ModelArtifactId != "ma-789" {
		t.Fatalf("expected ModelArtifactId ma-789, got %q", jobModel.ModelArtifactId)
	}
}

func TestGetAllModelTransferJobs_TerminationMessageMalformedJSONHandledGracefully(t *testing.T) {
	repo := NewModelRegistryRepository()

	jobStatus := batchv1.JobStatus{
		Succeeded: 1,
	}

	cases := []struct {
		name    string
		jobName string
		message string
	}{
		{
			name:    "malformed JSON",
			jobName: "job-malformed-json",
			message: "{not valid json",
		},
		{
			name:    "empty message",
			jobName: "job-empty-message",
			message: "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			containerState := corev1.ContainerState{
				Terminated: &corev1.ContainerStateTerminated{
					ExitCode: 0,
					Message:  tc.message,
				},
			}

			client := buildSingleJobFixture(tc.jobName, jobStatus, containerState)

			jobModel := mustGetSingleJob(t, repo, client, "kubeflow", "model-registry-id")
			if jobModel.RegisteredModelId != "" || jobModel.ModelVersionId != "" || jobModel.ModelArtifactId != "" {
				t.Fatalf("expected no IDs to be set for termination message %q, got rm=%q mv=%q ma=%q",
					tc.message, jobModel.RegisteredModelId, jobModel.ModelVersionId, jobModel.ModelArtifactId)
			}
		})
	}
}

func TestGetModelTransferJobEvents_UsesTimestampFallbacks(t *testing.T) {
	repo := NewModelRegistryRepository()

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job-events",
			Namespace: "kubeflow",
			Labels: map[string]string{
				"modelregistry.kubeflow.org/model-registry-name": "mr-1",
			},
		},
	}

	// Pod list just needs to be non-empty for GetModelTransferJobEvents to proceed.
	podList := &corev1.PodList{
		Items: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-events-1",
					Namespace: "kubeflow",
				},
			},
		},
	}

	// Three events exercising the timestamp fallback chain with distinct times:
	// 1) LastTimestamp set
	// 2) LastTimestamp zero, EventTime set
	// 3) LastTimestamp & EventTime zero, FirstTimestamp set
	lastTsTime := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	eventTimeTime := time.Date(2025, 1, 1, 11, 0, 0, 0, time.UTC)
	firstTsTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	lastTs := metav1.NewTime(lastTsTime)
	eventTime := metav1.NewMicroTime(eventTimeTime)
	firstTs := metav1.NewTime(firstTsTime)

	eventList := &corev1.EventList{
		Items: []corev1.Event{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "event-last",
					Namespace: "kubeflow",
				},
				LastTimestamp: lastTs,
				Type:          "Normal",
				Reason:        "Pulling",
				Message:       "Using image pull policy",
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "event-time",
					Namespace: "kubeflow",
				},
				EventTime: metav1.MicroTime{Time: eventTime.Time},
				Type:      "Normal",
				Reason:    "Started",
				Message:   "Container started",
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "event-first",
					Namespace: "kubeflow",
				},
				FirstTimestamp: firstTs,
				Type:           "Warning",
				Reason:         "BackOff",
				Message:        "Back-off restarting failed container",
			},
		},
	}

	client := &fakeKubernetesClient{
		jobsByNamespace: map[string]map[string]*batchv1.Job{
			"kubeflow": {
				"job-events": job,
			},
		},
		podsByNamespace: map[string]*corev1.PodList{
			"kubeflow": podList,
		},
		eventsByNamespace: map[string]*corev1.EventList{
			"kubeflow": eventList,
		},
	}

	events, err := repo.GetModelTransferJobEvents(context.Background(), client, "kubeflow", "job-events", "mr-1")
	if err != nil {
		t.Fatalf("GetModelTransferJobEvents returned error: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	// Verify basic mapping and that timestamps are formatted
	if events[0].Reason != "Pulling" || events[0].Type != "Normal" || events[0].Message == "" {
		t.Fatalf("event[0] not mapped correctly: %+v", events[0])
	}
	if events[1].Reason != "Started" || events[1].Type != "Normal" || events[1].Message == "" {
		t.Fatalf("event[1] not mapped correctly: %+v", events[1])
	}
	if events[2].Reason != "BackOff" || events[2].Type != "Warning" || events[2].Message == "" {
		t.Fatalf("event[2] not mapped correctly: %+v", events[2])
	}
	if events[0].Timestamp == "" || events[1].Timestamp == "" || events[2].Timestamp == "" {
		t.Fatalf("expected all events to have timestamps, got: %+v", events)
	}

	// Verify that the fallback chain picked the expected source for each timestamp.
	if events[0].Timestamp != lastTsTime.Format("2006-01-02T15:04:05Z") {
		t.Fatalf("expected events[0] timestamp from LastTimestamp, got %q", events[0].Timestamp)
	}
	if events[1].Timestamp != eventTimeTime.Format("2006-01-02T15:04:05Z") {
		t.Fatalf("expected events[1] timestamp from EventTime, got %q", events[1].Timestamp)
	}
	if events[2].Timestamp != firstTsTime.Format("2006-01-02T15:04:05Z") {
		t.Fatalf("expected events[2] timestamp from FirstTimestamp, got %q", events[2].Timestamp)
	}
}

func TestGetAllModelTransferJobs_NormalRunningJobNotOverridden(t *testing.T) {
	repo := NewModelRegistryRepository()

	job := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job-normal",
			Namespace: "kubeflow",
		},
		Status: batchv1.JobStatus{
			Active: 1, // Running
		},
	}

	jobs := &batchv1.JobList{
		Items: []batchv1.Job{job},
	}

	// Pod with a running container and no failure reasons or termination
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-normal",
			Namespace: "kubeflow",
			Labels: map[string]string{
				"job-name": job.Name,
			},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
				},
			},
		},
	}

	client := &fakeKubernetesClient{
		jobs: jobs,
		podsByNamespace: map[string]*corev1.PodList{
			"kubeflow": {
				Items: []corev1.Pod{pod},
			},
		},
	}

	jobModel := mustGetSingleJob(t, repo, client, "kubeflow", "model-registry-id")
	if jobModel.Status != models.ModelTransferJobStatusRunning {
		t.Fatalf("expected job status Running to be preserved, got %s", jobModel.Status)
	}
	if jobModel.ErrorMessage != "" {
		t.Fatalf("expected no error message for normal running job, got %q", jobModel.ErrorMessage)
	}
}

func TestRegistryOriginOnly(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ClusterIP with explicit port",
			input:    "http://10.43.0.100:8080/api/model_registry/v1alpha3",
			expected: "http://10.43.0.100:8080",
		},
		{
			name:     "Route-based HTTPS (no explicit port)",
			input:    "https://my-registry-rest.apps.example.com/api/model_registry/v1alpha3",
			expected: "https://my-registry-rest.apps.example.com:443",
		},
		{
			name:     "Route-based HTTPS with explicit port",
			input:    "https://my-registry-rest.apps.example.com:443/api/model_registry/v1alpha3",
			expected: "https://my-registry-rest.apps.example.com:443",
		},
		{
			name:     "Gateway-based URL preserves path prefix",
			input:    "https://gateway.apps.example.com/model-registry/my-registry/api/model_registry/v1alpha3",
			expected: "https://gateway.apps.example.com:443/model-registry/my-registry",
		},
		{
			name:     "Gateway-based URL with explicit port preserves path prefix",
			input:    "https://gateway.apps.example.com:443/model-registry/my-registry/api/model_registry/v1alpha3",
			expected: "https://gateway.apps.example.com:443/model-registry/my-registry",
		},
		{
			name:     "HTTP defaults to port 80",
			input:    "http://gateway.apps.example.com/model-registry/my-registry/api/model_registry/v1alpha3",
			expected: "http://gateway.apps.example.com:80/model-registry/my-registry",
		},
		{
			name:     "URL with no path",
			input:    "https://my-registry-rest.apps.example.com",
			expected: "https://my-registry-rest.apps.example.com:443",
		},
		{
			name:     "unparseable input returned as-is",
			input:    "://bad-url",
			expected: "://bad-url",
		},
		{
			name:     "bare hostname returned as-is",
			input:    "just-a-hostname",
			expected: "just-a-hostname",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := registryOriginOnly(tc.input)
			if got != tc.expected {
				t.Errorf("registryOriginOnly(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

var _ = Describe("resolveAsyncUploadImage", func() {
	var (
		client  k8s.KubernetesClientInterface
		mockCtx = mocks.NewMockSessionContextNoParent()
	)

	BeforeEach(func() {
		var err error
		client, err = kubernetesMockedStaticClientFactory.GetClient(mocks.NewMockSessionContextNoParent())
		Expect(err).NotTo(HaveOccurred())
	})

	Context("when not in federated mode", func() {
		It("should return the default image", func() {
			img := resolveAsyncUploadImage(mockCtx, client, false, "")
			Expect(img).To(Equal(DefaultAsyncUploadImage))
		})
	})

	Context("when in federated mode with empty namespace", func() {
		It("should return the default image", func() {
			img := resolveAsyncUploadImage(mockCtx, client, true, "")
			Expect(img).To(Equal(DefaultAsyncUploadImage))
		})
	})

	Context("when in federated mode with ConfigMap missing", func() {
		It("should fall back to the default image", func() {
			img := resolveAsyncUploadImage(mockCtx, client, true, "bento-namespace")
			Expect(img).To(Equal(DefaultAsyncUploadImage))
		})
	})

	Context("when in federated mode with ConfigMap present", func() {
		const testNamespace = "kubeflow"

		AfterEach(func() {
			_ = client.DeleteConfigMap(mockCtx, testNamespace, asyncUploadConfigMapName)
		})

		It("should return the configured image when the key is set", func() {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      asyncUploadConfigMapName,
					Namespace: testNamespace,
				},
				Data: map[string]string{
					asyncUploadConfigMapKey: "registry.example.com/custom-image:v1",
				},
			}
			_, err := client.CreateConfigMap(mockCtx, testNamespace, cm)
			Expect(err).NotTo(HaveOccurred())

			img := resolveAsyncUploadImage(mockCtx, client, true, testNamespace)
			Expect(img).To(Equal("registry.example.com/custom-image:v1"))
		})

		It("should fall back to the default image when the key is missing", func() {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      asyncUploadConfigMapName,
					Namespace: testNamespace,
				},
				Data: map[string]string{
					"some-other-key": "some-value",
				},
			}
			_, err := client.CreateConfigMap(mockCtx, testNamespace, cm)
			Expect(err).NotTo(HaveOccurred())

			img := resolveAsyncUploadImage(mockCtx, client, true, testNamespace)
			Expect(img).To(Equal(DefaultAsyncUploadImage))
		})

		It("should fall back to the default image when the key is empty", func() {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      asyncUploadConfigMapName,
					Namespace: testNamespace,
				},
				Data: map[string]string{
					asyncUploadConfigMapKey: "",
				},
			}
			_, err := client.CreateConfigMap(mockCtx, testNamespace, cm)
			Expect(err).NotTo(HaveOccurred())

			img := resolveAsyncUploadImage(mockCtx, client, true, testNamespace)
			Expect(img).To(Equal(DefaultAsyncUploadImage))
		})

		It("should fall back to the default image when the key is whitespace-only", func() {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      asyncUploadConfigMapName,
					Namespace: testNamespace,
				},
				Data: map[string]string{
					asyncUploadConfigMapKey: "   ",
				},
			}
			_, err := client.CreateConfigMap(mockCtx, testNamespace, cm)
			Expect(err).NotTo(HaveOccurred())

			img := resolveAsyncUploadImage(mockCtx, client, true, testNamespace)
			Expect(img).To(Equal(DefaultAsyncUploadImage))
		})
	})
})
