package repositories

import (
	"context"
	"testing"

	k8s "github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type baseImageJobClient struct {
	fakeKubernetesClient
	createdJob *batchv1.Job
}

func (c *baseImageJobClient) GetServiceDetailsByName(context.Context, string, string, string) (k8s.ServiceDetails, error) {
	return k8s.ServiceDetails{Name: "registry", ClusterIP: "127.0.0.1", HTTPPort: 8080}, nil
}

func (c *baseImageJobClient) CreateModelTransferJob(_ context.Context, _ string, job *batchv1.Job) (*batchv1.Job, error) {
	c.createdJob = job
	return job, nil
}

func TestOCIBaseImageCreateAndRetry(t *testing.T) {
	for _, configured := range []bool{false, true} {
		name := "default"
		image, source := defaultOCIBaseImage, "bff-default"
		client := &baseImageJobClient{}
		if configured {
			name, image, source = "configured", "mirror.example.com/base:1", "deployment-config"
			client.configMapsByNamespace = map[string]map[string]*corev1.ConfigMap{
				"config": {asyncUploadConfigMapName: {Data: map[string]string{ociBaseImageConfigMapKey: image}}},
			}
		}
		t.Run(name, func(t *testing.T) {
			repo := &ModelRegistryRepository{OCIBaseImageConfigNamespace: "config"}
			payload := models.ModelTransferJob{
				Name: "upload", Namespace: "jobs", JobDisplayName: "upload",
				Source: models.ModelTransferJobSource{Type: models.ModelTransferJobSourceTypeURI, URI: "https://example.com/model"},
				Destination: models.ModelTransferJobDestination{Type: models.ModelTransferJobDestinationTypeOCI,
					URI: "destination.example.com/model:1", Registry: "destination.example.com", Username: "user", Password: "password"},
				UploadIntent: models.ModelTransferJobUploadIntentUpdateArtifact, ModelArtifactId: "1",
			}
			_, err := repo.CreateModelTransferJob(testContext(), client, "registry", payload, "registry", false, "pod", nil)
			if err != nil {
				t.Fatal(err)
			}
			assertEnv := func() {
				t.Helper()
				vars := map[string]string{}
				for _, env := range client.createdJob.Spec.Template.Spec.Containers[0].Env {
					vars[env.Name] = env.Value
				}
				if vars["MODEL_SYNC_DESTINATION_OCI_BASE_IMAGE"] != image || vars["MODEL_SYNC_DESTINATION_OCI_BASE_IMAGE_SOURCE"] != source {
					t.Fatalf("unexpected base-image env: %v", vars)
				}
				if client.createdJob.Namespace != "jobs" {
					t.Fatal("configuration namespace changed the Job namespace")
				}
			}
			assertEnv()
			oldJob := client.createdJob.DeepCopy()
			oldJob.Status.Failed = 1
			client.jobsByNamespace = map[string]map[string]*batchv1.Job{"jobs": {"upload": oldJob}}
			secretName := oldJob.Annotations["modelregistry.kubeflow.org/dest-secret"]
			client.secretsByNamespace = map[string]map[string]*corev1.Secret{"jobs": {secretName: {Data: map[string][]byte{".dockerconfigjson": []byte(`{"auths":{}}`)}}}}
			payload.Name = "retry"
			_, err = repo.UpdateModelTransferJob(testContext(), client, "registry", "upload", payload, false, "registry", false, "pod", nil)
			if err != nil {
				t.Fatal(err)
			}
			assertEnv()
		})
	}
}

func TestOCIBaseImageConfigNamespace(t *testing.T) {
	for _, tc := range []struct{ override, pod, want string }{
		{" config ", "pod", "config"}, {"", "pod", "pod"}, {"", "", ""},
	} {
		repo := &ModelRegistryRepository{OCIBaseImageConfigNamespace: tc.override}
		if got := repo.ociBaseImageNamespace(tc.pod); got != tc.want {
			t.Fatalf("namespace = %q, want %q", got, tc.want)
		}
	}
}

func TestOCIBaseImageSourceAndReadFailures(t *testing.T) {
	resource := schema.GroupResource{Resource: "configmaps"}
	for _, tc := range []struct {
		name, value, source string
		readError           error
		wantError           bool
	}{
		{name: "absent", source: "bff-default", readError: apierrors.NewNotFound(resource, asyncUploadConfigMapName)},
		{name: "forbidden", readError: apierrors.NewForbidden(resource, asyncUploadConfigMapName, nil), wantError: true},
		{name: "timeout", readError: apierrors.NewTimeoutError("timeout", 1), wantError: true},
		{name: "configured", value: " mirror.example.com/base:1 ", source: "deployment-config"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &fakeKubernetesClient{getConfigMapErr: tc.readError,
				configMapsByNamespace: map[string]map[string]*corev1.ConfigMap{
					"config": {asyncUploadConfigMapName: {Data: map[string]string{ociBaseImageConfigMapKey: tc.value}}},
				},
			}
			image, source, err := resolveOCIBaseImageWithSource(testContext(), client, "config")
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v", err)
			}
			if source != tc.source {
				t.Fatalf("source = %q, want %q", source, tc.source)
			}
			if tc.name == "configured" && image != "mirror.example.com/base:1" {
				t.Fatalf("unexpected image %q", image)
			}
		})
	}
}
