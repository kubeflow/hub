package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/kubeflow/hub/ui/bff/internal/config"
	k8s "github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"

	"github.com/kubeflow/hub/ui/bff/internal/constants"
	"github.com/kubeflow/hub/ui/bff/internal/mocks"
	"github.com/kubeflow/hub/ui/bff/internal/repositories"
)

// serveApiTest builds the test App with mocked clients, serves the request and returns
// the raw response together with its (already read) body. Use it directly for non-JSON
// responses (e.g. binary payloads); setupApiTest wraps it for JSON envelope responses.
func serveApiTest(method string, url string, body interface{}, k8Factory kubernetes.KubernetesClientFactory, requestIdentity kubernetes.RequestIdentity, namespace string) (*http.Response, []byte, error) {
	mockMRClient, err := mocks.NewModelRegistryClient(nil)
	if err != nil {
		return nil, nil, err
	}
	mockModelCatalogClient, err := mocks.NewModelCatalogClientMock(nil)
	if err != nil {
		return nil, nil, err
	}

	mockClient := new(mocks.MockHTTPClient)

	cfg := config.EnvConfig{
		AuthMethod: config.AuthMethodInternal,
	}
	//if token is set, use token auth
	if requestIdentity.Token != "" {
		cfg.AuthMethod = config.AuthMethodUser
	}
	testApp := App{
		repositories:            repositories.NewRepositories(mockMRClient, mockModelCatalogClient),
		kubernetesClientFactory: k8Factory,
		logger:                  slog.Default(),
		config:                  cfg,
		podNamespace:            "bella-namespace",
	}

	var req *http.Request
	if body != nil {
		r, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		bytes.NewReader(r)
		req, err = http.NewRequest(method, url, bytes.NewReader(r))
		if err != nil {
			return nil, nil, err
		}
	} else {
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			return nil, nil, err
		}
	}

	// Set the kubeflow-userid header (middleware work)
	if requestIdentity.UserID != "" {
		req.Header.Set(constants.KubeflowUserIDHeader, requestIdentity.UserID)
	}

	ctx := mocks.NewMockSessionContext(req.Context())

	ctx = context.WithValue(ctx, constants.ModelRegistryHttpClientKey, mockClient)
	ctx = context.WithValue(ctx, constants.RequestIdentityKey, requestIdentity)
	ctx = context.WithValue(ctx, constants.NamespaceHeaderParameterKey, namespace)
	mrHttpClient := k8s.HTTPClient{}
	modelCatalogHttpClient := k8s.HTTPClient{}
	ctx = context.WithValue(ctx, constants.ModelRegistryHttpClientKey, mrHttpClient)
	ctx = context.WithValue(ctx, constants.ModelCatalogHttpClientKey, modelCatalogHttpClient)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	testApp.Routes().ServeHTTP(rr, req)

	rs := rr.Result()
	defer rs.Body.Close()
	respBody, err := io.ReadAll(rs.Body)
	if err != nil {
		return nil, nil, err
	}

	return rs, respBody, nil
}

func setupApiTest[T any](method string, url string, body interface{}, k8Factory kubernetes.KubernetesClientFactory, requestIdentity kubernetes.RequestIdentity, namespace string) (T, *http.Response, error) {
	rs, respBody, err := serveApiTest(method, url, body, k8Factory, requestIdentity, namespace)
	if err != nil {
		return *new(T), nil, err
	}

	var entity T
	err = json.Unmarshal(respBody, &entity)
	if err != nil {
		if err == io.EOF {
			// There's no body to parse.
			return *new(T), rs, nil
		}
		return *new(T), nil, err
	}

	return entity, rs, nil
}

func resolveStaticAssetsDirOnTests() string {
	// Fall back to finding project root for testing
	projectRoot, err := findProjectRootOnTests()
	if err != nil {
		panic("Failed to find project root: ")
	}

	return filepath.Join(projectRoot, "static")
}

// on tests findProjectRoot searches for the project root by locating go.mod
func findProjectRootOnTests() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Traverse up until go.mod is found
	for currentDir != "/" {
		if _, err := os.Stat(filepath.Join(currentDir, "go.mod")); err == nil {
			return currentDir, nil
		}
		currentDir = filepath.Dir(currentDir)
	}

	return "", os.ErrNotExist
}
