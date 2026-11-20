package inferenceservicecontroller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/go-logr/logr"
	kservev1beta1 "github.com/kserve/kserve/pkg/apis/serving/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	tokenTestRegistriesNS   = "kubeflow"
	tokenTestUserNS         = "user-ns"
	tokenTestToken          = "controller-sa-token"
	tokenTestNamespaceLabel = "modelregistry.kubeflow.org/namespace"
	tokenTestURLAnnotation  = "modelregistry.kubeflow.org/url"
	tokenTestRegModelLabel  = "modelregistry.kubeflow.org/registered-model-id"
)

// rewriteToServer sends every request to the test server, whatever host the
// controller resolved, and keeps the original path.
type rewriteToServer struct {
	target *url.URL
	next   http.RoundTripper
}

func (t rewriteToServer) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = t.target.Scheme
	req.URL.Host = t.target.Host
	return t.next.RoundTrip(req)
}

func mrService(namespace string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "model-registry",
			Namespace: namespace,
			Labels:    map[string]string{"component": "model-registry"},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{Name: "http-api", Port: 8080}},
		},
	}
}

// reconcileAuthHeaders reconciles isvc and returns the Authorization headers
// the controller sent to the model registry.
func reconcileAuthHeaders(t *testing.T, isvc *kservev1beta1.InferenceService) []string {
	t.Helper()

	var mu sync.Mutex
	var headers []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		headers = append(headers, r.Header.Get("Authorization"))
		mu.Unlock()
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	target, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(kservev1beta1.AddToScheme(scheme))
	cli := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(isvc, mrService(tokenTestRegistriesNS), mrService(tokenTestUserNS)).
		Build()

	r := NewInferenceServiceController(
		cli,
		logr.Discard(),
		false,
		tokenTestToken,
		"modelregistry.kubeflow.org/inference-service-id",
		tokenTestRegModelLabel,
		"modelregistry.kubeflow.org/model-version-id",
		tokenTestNamespaceLabel,
		"modelregistry.kubeflow.org/name",
		tokenTestURLAnnotation,
		"modelregistry.kubeflow.org/finalizer",
		"routing.kubeflow.org/external-address-rest",
		tokenTestRegistriesNS,
	)
	r.OverrideHTTPClient(&http.Client{Transport: rewriteToServer{target: target, next: http.DefaultTransport}})

	// The mock registry fails every call, so Reconcile returns an error; only
	// the headers it sent matter here.
	_, _ = r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: isvc.Name, Namespace: isvc.Namespace},
	})

	mu.Lock()
	defer mu.Unlock()
	if len(headers) == 0 {
		t.Fatal("controller made no request to the model registry")
	}
	return headers
}

func newISVC(labels, annotations map[string]string) *kservev1beta1.InferenceService {
	labels[tokenTestRegModelLabel] = "1"
	return &kservev1beta1.InferenceService{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "isvc",
			Namespace:   tokenTestUserNS,
			Labels:      labels,
			Annotations: annotations,
		},
	}
}

func TestBearerTokenNotSentToISVCSuppliedRegistry(t *testing.T) {
	tests := []struct {
		name string
		isvc *kservev1beta1.InferenceService
	}{
		{
			name: "url annotation",
			isvc: newISVC(map[string]string{}, map[string]string{tokenTestURLAnnotation: "https://attacker.example"}),
		},
		{
			name: "namespace label outside the registries namespace",
			isvc: newISVC(map[string]string{tokenTestNamespaceLabel: tokenTestUserNS}, nil),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, h := range reconcileAuthHeaders(t, tt.isvc) {
				if h != "" {
					t.Fatalf("controller sent Authorization %q to a registry chosen by the InferenceService", h)
				}
			}
		})
	}
}

func TestBearerTokenSentToRegistriesNamespace(t *testing.T) {
	for _, h := range reconcileAuthHeaders(t, newISVC(map[string]string{}, nil)) {
		if h != "Bearer "+tokenTestToken {
			t.Fatalf("expected the controller token on requests to the registries namespace, got %q", h)
		}
	}
}
