package k8mocks

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	k8s "github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var DefaultTestUsers = []TestUser{
	{
		UserName: "user@example.com",
		Token:    "FAKE_CLUSTER_ADMIN_TOKEN",
		Groups:   []string{},
	},
	{
		UserName: "doraNonAdmin@example.com",
		Token:    "FAKE_DORA_TOKEN",
		Groups:   []string{"dora-namespace-group", "dora-service-group"},
	},
	{
		UserName: "bellaNonAdmin@example.com",
		Token:    "FAKE_BELLA_TOKEN",
		Groups:   []string{},
	},
}

type TestUser struct {
	UserName string
	Token    string
	Groups   []string
}

type TestEnvInput struct {
	Users  []TestUser
	Logger *slog.Logger
	Ctx    context.Context
	Cancel context.CancelFunc
}

func SetupEnvTest(input TestEnvInput) (*envtest.Environment, kubernetes.Interface, error) {
	var binaryAssetsDir string

	// Check for explicit envtest assets directory (used in Docker)
	if envDir := os.Getenv("ENVTEST_ASSETS_DIR"); envDir != "" {
		// Construct full path with OS/ARCH suffix
		binaryAssetsDir = filepath.Join(envDir, "k8s",
			fmt.Sprintf("1.29.3-%s-%s", runtime.GOOS, runtime.GOARCH))
	} else {
		// Fall back to project root detection (local development)
		projectRoot, err := getProjectRoot()
		if err != nil {
			input.Logger.Error("failed to find project root", slog.String("error", err.Error()))
			input.Cancel()
			os.Exit(1)
		}
		binaryAssetsDir = filepath.Join(projectRoot, "bin", "k8s",
			fmt.Sprintf("1.29.3-%s-%s", runtime.GOOS, runtime.GOARCH))
	}

	// Fix for #2136: Configure ControlPlane to bind to 127.0.0.1 instead of
	// platform-specific addresses (e.g., 192.168.127.254 on macOS) which don't
	// exist inside Docker containers.
	//
	// Pre-allocate a free port on 127.0.0.1 and pass it as a real port number.
	// envtest treats Port:"0" as a literal string (not auto-select), which causes
	// kube-apiserver to reject it with "--secure-port 0 must be between 1 and 65535".
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to allocate free port on 127.0.0.1: %w", err)
	}
	freePort := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	testEnv := &envtest.Environment{
		BinaryAssetsDirectory: binaryAssetsDir,
		ControlPlane: envtest.ControlPlane{
			APIServer: &envtest.APIServer{
				SecureServing: envtest.SecureServing{
					ListenAddr: envtest.ListenAddr{
						Address: "127.0.0.1",
						Port:    strconv.Itoa(freePort),
					},
				},
			},
		},
	}

	cfg, err := testEnv.Start()
	if err != nil {
		input.Logger.Error("failed to start envtest", slog.String("error", err.Error()))
		input.Cancel()
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		input.Logger.Error("failed to create clientset", slog.String("error", err.Error()))
		input.Cancel()
		os.Exit(1)
	}

	// bootstrap resources
	err = setupMock(clientset, input.Ctx)
	if err != nil {
		input.Logger.Error("failed to setup mock data", slog.String("error", err.Error()))
		input.Cancel()
		os.Exit(1)
	}

	return testEnv, clientset, nil
}

func setupMock(mockK8sClient kubernetes.Interface, ctx context.Context) error {

	err := createNamespace(mockK8sClient, ctx, "kubeflow")
	if err != nil {
		return err
	}

	err = createNamespace(mockK8sClient, ctx, "dora-namespace")
	if err != nil {
		return err
	}

	err = createNamespace(mockK8sClient, ctx, "bella-namespace")
	if err != nil {
		return err
	}

	err = createNamespace(mockK8sClient, ctx, "bento-namespace")
	if err != nil {
		return err
	}

	err = createService(mockK8sClient, ctx, "model-registry", "kubeflow", "Model Registry", "Model Registry Description", "10.0.0.10", "model-registry")
	if err != nil {
		return err
	}
	err = createEndpoints(mockK8sClient, ctx, "kubeflow", "model-registry", true)
	if err != nil {
		return err
	}
	err = createService(mockK8sClient, ctx, "model-registry-one", "kubeflow", "Model Registry One", "Model Registry One description", "10.0.0.11", "model-registry")
	if err != nil {
		return err
	}
	err = createEndpoints(mockK8sClient, ctx, "kubeflow", "model-registry-one", false)
	if err != nil {
		return err
	}
	err = createService(mockK8sClient, ctx, "model-registry-dora", "dora-namespace", "Model Registry Dora", "Model Registry Dora description", "10.0.0.12", "model-registry")
	if err != nil {
		return err
	}
	err = createService(mockK8sClient, ctx, "model-registry-bella", "bella-namespace", "Model Registry Bella", "Model Registry Bella description", "10.0.0.13", "model-registry")
	if err != nil {
		return err
	}
	err = createService(mockK8sClient, ctx, "model-registry-bella-prod", "bella-namespace", "Bella Production Registry", "Production model registry for Bella team", "10.0.0.20", "model-registry")
	if err != nil {
		return err
	}
	err = createService(mockK8sClient, ctx, "model-registry-bella-staging", "bella-namespace", "Bella Staging Registry", "Staging model registry for Bella team", "10.0.0.21", "model-registry")
	if err != nil {
		return err
	}
	err = createService(mockK8sClient, ctx, "model-registry-bella-dev", "bella-namespace", "Bella Dev Registry", "Development model registry for Bella team", "10.0.0.22", "model-registry")
	if err != nil {
		return err
	}
	err = createService(mockK8sClient, ctx, "non-model-registry", "kubeflow", "Not a Model Registry", "Not a Model Registry Bella description", "10.0.0.14", "")
	if err != nil {
		return err
	}

	err = createModelCatalogService(mockK8sClient, ctx, "model-catalog", "kubeflow", "10.0.0.15")
	if err != nil {
		return err
	}

	err = createModelCatalogService(mockK8sClient, ctx, "model-catalog", "bella-namespace", "10.0.0.16")
	if err != nil {
		return err
	}

	err = createClusterAdminRBAC(mockK8sClient, ctx, DefaultTestUsers[0].UserName)
	if err != nil {
		return fmt.Errorf("failed to create cluster admin RBAC: %w", err)
	}

	err = createNamespaceRestrictedRBAC(mockK8sClient, ctx, DefaultTestUsers[1].UserName, "dora-namespace")
	if err != nil {
		return fmt.Errorf("failed to create namespace-restricted RBAC: %w", err)
	}

	err = createNamespaceRestrictedRBAC(mockK8sClient, ctx, DefaultTestUsers[2].UserName, "bella-namespace")
	if err != nil {
		return fmt.Errorf("failed to create namespace-restricted RBAC: %w", err)
	}

	err = createGroupAccessRBAC(mockK8sClient, ctx, DefaultTestUsers[1].Groups[1], "dora-namespace", "model-registry-dora")
	if err != nil {
		return fmt.Errorf("failed to create group-based RBAC: %w", err)
	}

	err = createNamespaceDefaultSARegistryAccessRBAC(mockK8sClient, ctx, "dora-namespace", "model-registry-dora")
	if err != nil {
		return fmt.Errorf("failed to create namespace default SA registry access RBAC: %w", err)
	}

	err = createNamespaceDefaultSARegistryAccessRBAC(mockK8sClient, ctx, "kubeflow", "model-registry")
	if err != nil {
		return fmt.Errorf("failed to create kubeflow default SA registry access RBAC: %w", err)
	}
	err = createCrossNamespaceRegistryAccessBinding(mockK8sClient, ctx, "kubeflow", "namespace-sa-registry-access", "dora-namespace")
	if err != nil {
		return fmt.Errorf("failed to create dora-namespace cross-namespace access to model-registry: %w", err)
	}

	err = createNamespaceDefaultSARegistryAccessRBAC(mockK8sClient, ctx, "bella-namespace", "model-registry-bella")
	if err != nil {
		return fmt.Errorf("failed to create bella-namespace default SA registry access RBAC: %w", err)
	}
	err = createCrossNamespaceRegistryAccessBinding(mockK8sClient, ctx, "bella-namespace", "namespace-sa-registry-access", "dora-namespace")
	if err != nil {
		return fmt.Errorf("failed to create dora-namespace cross-namespace access to model-registry-bella: %w", err)
	}

	err = createGroupNamespaceAccessRBAC(mockK8sClient, ctx, DefaultTestUsers[1].Groups[0], "dora-namespace")
	if err != nil {
		return fmt.Errorf("failed to set up group access to namespace: %w", err)
	}

	err = createModelCatalogDefaultSourcesConfigMap(mockK8sClient, ctx, "bella-namespace")
	if err != nil {
		return err
	}

	err = createModelCatalogSourcesConfigMap(mockK8sClient, ctx, "bella-namespace")
	if err != nil {
		return err
	}

	err = createModelCatalogDefaultSourcesConfigMap(mockK8sClient, ctx, "kubeflow")
	if err != nil {
		return err
	}

	err = createModelCatalogSourcesConfigMap(mockK8sClient, ctx, "kubeflow")
	if err != nil {
		return err
	}

	err = addMcpDataToDefaultCatalogSourcesConfigMap(mockK8sClient, ctx, "kubeflow")
	if err != nil {
		return err
	}

	err = createMcpCatalogSourcesConfigMap(mockK8sClient, ctx, "kubeflow")
	if err != nil {
		return err
	}

	err = addMcpDataToDefaultCatalogSourcesConfigMap(mockK8sClient, ctx, "bella-namespace")
	if err != nil {
		return err
	}

	err = createMcpCatalogSourcesConfigMap(mockK8sClient, ctx, "bella-namespace")
	if err != nil {
		return err
	}

	err = createHuggingFaceSecret(mockK8sClient, ctx, "kubeflow")
	if err != nil {
		return err
	}

	err = createHuggingFaceSecret(mockK8sClient, ctx, "bella-namespace")
	if err != nil {
		return err
	}

	err = createModelTransferJob(mockK8sClient, ctx, "kubeflow", "model-registry")
	if err != nil {
		return err
	}

	err = createModelTransferJob(mockK8sClient, ctx, "bella-namespace", "model-registry-bella")
	if err != nil {
		return err
	}

	err = createTransferJobPods(mockK8sClient, ctx, "kubeflow")
	if err != nil {
		return err
	}

	err = createTransferJobPods(mockK8sClient, ctx, "bella-namespace")
	if err != nil {
		return err
	}

	err = createTransferJobPodEvents(mockK8sClient, ctx, "kubeflow")
	if err != nil {
		return err
	}

	err = createTransferJobPodEvents(mockK8sClient, ctx, "bella-namespace")
	if err != nil {
		return err
	}

	return nil
}

func createModelCatalogDefaultSourcesConfigMap(
	k8sClient kubernetes.Interface,
	ctx context.Context,
	namespace string,
) error {
	raw := strings.TrimSpace(`
catalogs:
  - name: Dora AI
    id: dora_ai_models
    type: yaml
    enabled: true
    properties:
      yamlCatalogPath: dora_ai_models.yaml
    labels:
      - Dora AI

  - name: Bella AI validated
    id: bella_ai_validated_models
    type: yaml
    enabled: true
    properties:
      yamlCatalogPath: bella_ai_validated_models.yaml
    labels:
      - Bella AI validated
`)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8s.CatalogSourceDefaultConfigMapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			k8s.CatalogSourceKey:  raw,
			"dora_ai_models.yaml": "models:\n - name: ai_model1",
		},
	}

	if _, err := k8sClient.CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("failed to create default-catalog-sources configmap: %w", err)
	}

	return nil
}

func createModelCatalogSourcesConfigMap(
	k8sClient kubernetes.Interface,
	ctx context.Context,
	namespace string,
) error {
	raw := strings.TrimSpace(`
catalogs:
  - name: Custom yaml
    id: custom_yaml_models
    type: yaml
    enabled: true
    properties:
      yamlCatalogPath: custom_yaml_models.yaml
    includedModels:
      - model-*
      - model-2-*
    excludedModels:
      - sample-model-*
    labels:
      - Dora AI

  - name: Sample source
    id: sample_source_models
    type: yaml
    enabled: false
    properties:
      yamlCatalogPath: sample_source_models.yaml
    includedModels:
      - model-*
      - model-2-*
    excludedModels:
      - sample-model-*
    labels:
      - Bella AI validated
      - Dora AI

  - name: Hugging face source
    id: hugging_face_source
    type: hf
    enabled: true
    properties:
      apiKey: hugging-face-source-secret
      allowedOrganization: org
    includedModels:
      - model-*
      - model-2-*
    excludedModels:
      - sample-model-*
    labels:
      - Bella AI validated
`)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8s.CatalogSourceUserConfigMapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			k8s.CatalogSourceKey:        raw,
			"custom_yaml_models.yaml":   "models:\n - name: model1",
			"sample_source_models.yaml": "models:\n - name: model2",
		},
	}

	if _, err := k8sClient.CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("failed to create model-catalog-sources configmap: %w", err)
	}

	return nil
}

func createHuggingFaceSecret(k8sClient kubernetes.Interface, ctx context.Context, namespace string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hugging-face-source-secret",
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"apiKey": "hf_test_api_key_12345",
		},
	}

	if _, err := k8sClient.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("failed to create huggingface secret: %w", err)
	}
	return nil
}

func createNamespace(k8sClient kubernetes.Interface, ctx context.Context, namespace string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	_, err := k8sClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create namespace %s: %w", namespace, err)
	}

	return nil
}

func createClusterAdminRBAC(k8sClient kubernetes.Interface, ctx context.Context, username string) error {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("cluster-admin-binding-%s", username),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "User",
				Name: username,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	_, err := k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create cluster admin ClusterRoleBinding: %w", err)
	}

	return nil
}

func createNamespaceRestrictedRBAC(k8sClient kubernetes.Interface, ctx context.Context, username, namespace string) error {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "namespace-restricted-role",
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"services", "namespaces"},
				Verbs:     []string{"get", "list"},
			},
		},
	}

	_, err := k8sClient.RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create Role: %w", err)
	}

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "namespace-restricted-binding",
			Namespace: namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "User",
				Name: username,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     "namespace-restricted-role",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	_, err = k8sClient.RbacV1().RoleBindings(namespace).Create(ctx, roleBinding, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create RoleBinding: %w", err)
	}

	return nil
}

func createGroupAccessRBAC(k8sClient kubernetes.Interface, ctx context.Context, groupName, namespace, serviceName string) error {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "group-model-registry-access",
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"services"},
				Verbs:     []string{"get", "list"},
				ResourceNames: []string{
					serviceName,
				},
			},
		},
	}

	_, err := k8sClient.RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create Role for group: %w", err)
	}

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "group-access-binding",
			Namespace: namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "Group",
				Name: groupName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     "group-model-registry-access",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	_, err = k8sClient.RbacV1().RoleBindings(namespace).Create(ctx, roleBinding, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create RoleBinding for group: %w", err)
	}
	return nil
}

func createNamespaceDefaultSARegistryAccessRBAC(k8sClient kubernetes.Interface, ctx context.Context, namespace, serviceName string) error {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "namespace-sa-registry-access",
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"services"},
				Verbs:         []string{"get"},
				ResourceNames: []string{serviceName},
			},
		},
	}
	_, err := k8sClient.RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create Role for namespace default SA: %w", err)
	}
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "namespace-sa-registry-access-binding",
			Namespace: namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     "Group",
				Name:     "system:serviceaccounts:" + namespace,
				APIGroup: "rbac.authorization.k8s.io",
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     "namespace-sa-registry-access",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
	_, err = k8sClient.RbacV1().RoleBindings(namespace).Create(ctx, roleBinding, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create RoleBinding for namespace default SA: %w", err)
	}
	return nil
}

func createCrossNamespaceRegistryAccessBinding(k8sClient kubernetes.Interface, ctx context.Context, registryNamespace, roleName, jobNamespace string) error {
	bindingName := "registry-access-binding-" + strings.ReplaceAll(jobNamespace, ":", "-")
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bindingName,
			Namespace: registryNamespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     "Group",
				Name:     "system:serviceaccounts:" + jobNamespace,
				APIGroup: "rbac.authorization.k8s.io",
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     roleName,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
	_, err := k8sClient.RbacV1().RoleBindings(registryNamespace).Create(ctx, roleBinding, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create cross-namespace registry access RoleBinding: %w", err)
	}
	return nil
}

func createGroupNamespaceAccessRBAC(k8sClient kubernetes.Interface, ctx context.Context, groupName, namespace string) error {

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "group-namespace-access-role",
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces", "services"},
				Verbs:     []string{"get", "list"},
			},
		},
	}

	_, err := k8sClient.RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create Role for group namespace access: %w", err)
	}

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "group-namespace-access-binding",
			Namespace: namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "Group",
				Name: groupName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     "group-namespace-access-role",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	_, err = k8sClient.RbacV1().RoleBindings(namespace).Create(ctx, roleBinding, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create RoleBinding for group namespace access: %w", err)
	}

	return nil
}

func createService(k8sClient kubernetes.Interface, ctx context.Context, name string, namespace string, displayName string, description string, clusterIP string, componentLabel string) error {

	annotations := map[string]string{}

	if displayName != "" {
		annotations["displayName"] = displayName
	}

	if description != "" {
		annotations["description"] = description
	}

	labels := map[string]string{}
	if componentLabel != "" {
		labels["component"] = componentLabel
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"component": k8s.ComponentLabelValue,
			},
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: clusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:        "http-api",
					Port:        8080,
					Protocol:    corev1.ProtocolTCP,
					AppProtocol: strPtr("http"),
				},
				{
					Name:        "grpc-api",
					Port:        9090,
					Protocol:    corev1.ProtocolTCP,
					AppProtocol: strPtr("grpc"),
				},
			},
		},
	}

	// Create the service using kubernetes.Interface
	_, err := k8sClient.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	return nil
}

// createEndpoints creates an Endpoints resource for a service so that EndpointsHasReadyAddresses returns hasReady.
//
//nolint:staticcheck // intentionally using deprecated corev1.Endpoints for RBAC compatibility; see tech debt ticket for EndpointSlice migration
func createEndpoints(k8sClient kubernetes.Interface, ctx context.Context, namespace, serviceName string, hasReady bool) error {
	ep := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Subsets: nil,
	}
	if hasReady {
		ep.Subsets = []corev1.EndpointSubset{
			{Addresses: []corev1.EndpointAddress{{IP: "10.0.0.1"}}},
		}
	}
	_, err := k8sClient.CoreV1().Endpoints(namespace).Create(ctx, ep, metav1.CreateOptions{})
	return err
}

func createModelCatalogService(k8sClient kubernetes.Interface, ctx context.Context, name, namespace, clusterIP string) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app":                         "model-catalog-service",
				"app.kubernetes.io/component": "model-catalog",
				"app.kubernetes.io/instance":  "model-catalog-service",
				"app.kubernetes.io/name":      "model-catalog",
				"app.kubernetes.io/part-of":   "model-catalog",
				"component":                   "model-catalog",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"component": "model-catalog-server",
			},
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: clusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:        "http-api",
					Port:        8081,
					Protocol:    corev1.ProtocolTCP,
					AppProtocol: strPtr("http"),
				},
			},
		},
	}

	_, err := k8sClient.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create model-catalog service: %w", err)
	}

	return nil
}

func createModelTransferJob(k8sClient kubernetes.Interface, ctx context.Context, namespace string, registryName string) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "transfer-job-001-config",
			Namespace: namespace,
			Labels: map[string]string{
				"modelregistry.kubeflow.org/job-type": "async-upload",
				"modelregistry.kubeflow.org/job-id":   "001",
			},
		},
		Data: map[string]string{
			"RegisteredModel.name":            "Model One",
			"RegisteredModel.description":     "This model does things and stuff",
			"RegisteredModel.owner":           "Sherlock Holmes",
			"ModelVersion.name":               "Version One",
			"ModelVersion.author":             "Sherlock Holmes",
			"ModelArtifact.model_format_name": "onnx",
		},
	}

	_, err := k8sClient.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create configmap: %w", err)
	}

	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "transfer-job-001-source-secret",
			Namespace: namespace,
			Labels: map[string]string{
				"modelregistry.kubeflow.org/job-type": "async-upload",
				"modelregistry.kubeflow.org/job-id":   "001",
			},
		},
		StringData: map[string]string{
			"AWS_ACCESS_KEY_ID":     "mock-access-key",
			"AWS_SECRET_ACCESS_KEY": "mock-secret-key",
			"AWS_DEFAULT_REGION":    "us-east-1",
			"AWS_S3_BUCKET":         "source-bucket",
		},
	}

	_, err = k8sClient.CoreV1().Secrets(namespace).Create(ctx, sourceSecret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create source secret: %w", err)
	}

	destSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "transfer-job-001-dest-secret",
			Namespace: namespace,
			Labels: map[string]string{
				"modelregistry.kubeflow.org/job-type": "async-upload",
				"modelregistry.kubeflow.org/job-id":   "001",
			},
		},
		Type: corev1.SecretTypeDockerConfigJson,
		StringData: map[string]string{
			".dockerconfigjson": `{"auths":{"quay.io":{"auth":"bW9jazptb2Nr"}}}`,
			"username":          "",
			"password":          "",
			"registry":          "",
		},
	}

	_, err = k8sClient.CoreV1().Secrets(namespace).Create(ctx, destSecret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create destination secret: %w", err)
	}

	backoffLimit := int32(3)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "transfer-job-001",
			Namespace: namespace,
			Labels: map[string]string{
				"modelregistry.kubeflow.org/job-type":            "async-upload",
				"modelregistry.kubeflow.org/job-id":              "001",
				"modelregistry.kubeflow.org/model-registry-name": registryName,
			},
			Annotations: map[string]string{
				"modelregistry.kubeflow.org/display-name":        "Transfer job 001",
				"modelregistry.kubeflow.org/registered-model-id": "1",
				"modelregistry.kubeflow.org/model-name":          "Model One",
				"modelregistry.kubeflow.org/model-version-id":    "1",
				"modelregistry.kubeflow.org/version-name":        "Version One",
				"modelregistry.kubeflow.org/source-type":         "s3",
				"modelregistry.kubeflow.org/source-bucket":       "source-bucket",
				"modelregistry.kubeflow.org/source-key":          "models/my-model",
				"modelregistry.kubeflow.org/dest-type":           "oci",
				"modelregistry.kubeflow.org/dest-uri":            "quay.io/test/model:v1",
				"modelregistry.kubeflow.org/upload-intent":       "create_model",
				"modelregistry.kubeflow.org/author":              "Sherlock Holmes",
				"modelregistry.kubeflow.org/description":         "Transfer job for Model One",
				"modelregistry.kubeflow.org/configmap-name":      "transfer-job-001-config",
				"modelregistry.kubeflow.org/source-secret":       "transfer-job-001-source-secret",
				"modelregistry.kubeflow.org/dest-secret":         "transfer-job-001-dest-secret",
			},
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
					Containers: []corev1.Container{
						{
							Name:  "async-upload",
							Image: "ghcr.io/kubeflow/hub/job/async-upload:latest",
							Env: []corev1.EnvVar{
								{Name: "MODEL_SYNC_SOURCE_TYPE", Value: "s3"},
								{Name: "MODEL_SYNC_SOURCE_AWS_KEY", Value: "models/my-model"},
								{Name: "MODEL_SYNC_DESTINATION_TYPE", Value: "oci"},
								{Name: "MODEL_SYNC_DESTINATION_OCI_URI", Value: "quay.io/test/model:v1"},
								{Name: "MODEL_SYNC_DESTINATION_OCI_REGISTRY", Value: "quay.io"},
								{Name: "MODEL_SYNC_MODEL_UPLOAD_INTENT", Value: "create_model"},
								{Name: "MODEL_SYNC_METADATA_CONFIGMAP_PATH", Value: "/etc/model-metadata"},
								{Name: "MODEL_SYNC_SOURCE_S3_CREDENTIALS_PATH", Value: "/opt/creds/source"},
								{Name: "MODEL_SYNC_DESTINATION_OCI_CREDENTIALS_PATH", Value: "/opt/creds/destination/.dockerconfigjson"},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "source-credentials", MountPath: "/opt/creds/source", ReadOnly: true},
								{Name: "destination-credentials", MountPath: "/opt/creds/destination", ReadOnly: true},
								{Name: "model-metadata", MountPath: "/etc/model-metadata", ReadOnly: true},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "source-credentials",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "transfer-job-001-source-secret",
								},
							},
						},
						{
							Name: "destination-credentials",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "transfer-job-001-dest-secret",
								},
							},
						},
						{
							Name: "model-metadata",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "transfer-job-001-config",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	createdJob1, err := k8sClient.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	createdJob1.Status = batchv1.JobStatus{
		Failed: 1,
		Conditions: []batchv1.JobCondition{
			{Type: batchv1.JobFailed, Status: corev1.ConditionTrue, Message: "Simulated failure for testing retry"},
		},
	}
	_, err = k8sClient.BatchV1().Jobs(namespace).UpdateStatus(ctx, createdJob1, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	job2ConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "transfer-job-002-config",
			Namespace: namespace,
			Labels: map[string]string{
				"modelregistry.kubeflow.org/job-type": "async-upload",
				"modelregistry.kubeflow.org/job-id":   "002",
			},
		},
		Data: map[string]string{
			"RegisteredModel.name":        "Model Two",
			"RegisteredModel.description": "Another model for testing",
			"RegisteredModel.owner":       "John Watson",
			"ModelVersion.name":           "Version Three",
			"ModelVersion.author":         "John Watson",
		},
	}

	_, err = k8sClient.CoreV1().ConfigMaps(namespace).Create(ctx, job2ConfigMap, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create job2 configmap: %w", err)
	}

	job2 := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "transfer-job-002",
			Namespace: namespace,
			Labels: map[string]string{
				"modelregistry.kubeflow.org/job-type":            "async-upload",
				"modelregistry.kubeflow.org/job-id":              "002",
				"modelregistry.kubeflow.org/model-registry-name": registryName,
			},
			Annotations: map[string]string{
				"modelregistry.kubeflow.org/display-name":        "Transfer job 002",
				"modelregistry.kubeflow.org/registered-model-id": "2",
				"modelregistry.kubeflow.org/model-name":          "Model Two",
				"modelregistry.kubeflow.org/model-version-id":    "3",
				"modelregistry.kubeflow.org/version-name":        "Version Three",
				"modelregistry.kubeflow.org/source-type":         "s3",
				"modelregistry.kubeflow.org/dest-type":           "oci",
				"modelregistry.kubeflow.org/dest-uri":            "quay.io/test/model-two:v3",
				"modelregistry.kubeflow.org/upload-intent":       "create_model",
				"modelregistry.kubeflow.org/author":              "John Watson",
				"modelregistry.kubeflow.org/description":         "Create new model - completed successfully",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "async-upload",
							Image: "ghcr.io/kubeflow/hub/job/async-upload:latest",
						},
					},
				},
			},
		},
	}

	createdJob2, err := k8sClient.BatchV1().Jobs(namespace).Create(ctx, job2, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create job2: %w", err)
	}

	createdJob2.Status = batchv1.JobStatus{
		Succeeded: 1,
		Conditions: []batchv1.JobCondition{
			{Type: batchv1.JobComplete, Status: corev1.ConditionTrue},
		},
	}

	_, err = k8sClient.BatchV1().Jobs(namespace).UpdateStatus(ctx, createdJob2, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update job2 status: %w", err)
	}

	job3ConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "transfer-job-003-config",
			Namespace: namespace,
			Labels: map[string]string{
				"modelregistry.kubeflow.org/job-type": "async-upload",
				"modelregistry.kubeflow.org/job-id":   "003",
			},
		},
		Data: map[string]string{
			"RegisteredModel.name": "Model One",
			"ModelVersion.name":    "Version Two",
			"ModelVersion.author":  "Sherlock Holmes",
		},
	}

	_, err = k8sClient.CoreV1().ConfigMaps(namespace).Create(ctx, job3ConfigMap, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create job3 configmap: %w", err)
	}

	job3 := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "transfer-job-003",
			Namespace: namespace,
			Labels: map[string]string{
				"modelregistry.kubeflow.org/job-type":            "async-upload",
				"modelregistry.kubeflow.org/job-id":              "003",
				"modelregistry.kubeflow.org/model-registry-name": registryName,
			},
			Annotations: map[string]string{
				"modelregistry.kubeflow.org/display-name":        "Transfer job 003",
				"modelregistry.kubeflow.org/registered-model-id": "1",
				"modelregistry.kubeflow.org/model-name":          "Model One",
				"modelregistry.kubeflow.org/model-version-id":    "2",
				"modelregistry.kubeflow.org/version-name":        "Version Two",
				"modelregistry.kubeflow.org/source-type":         "s3",
				"modelregistry.kubeflow.org/dest-type":           "oci",
				"modelregistry.kubeflow.org/dest-uri":            "quay.io/test/model-one:v2",
				"modelregistry.kubeflow.org/upload-intent":       "create_version",
				"modelregistry.kubeflow.org/author":              "Sherlock Holmes",
				"modelregistry.kubeflow.org/description":         "Create new version - failed due to connection timeout",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "async-upload",
							Image: "ghcr.io/kubeflow/hub/job/async-upload:latest",
						},
					},
				},
			},
		},
	}

	createdJob3, err := k8sClient.BatchV1().Jobs(namespace).Create(ctx, job3, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create job3: %w", err)
	}

	createdJob3.Status = batchv1.JobStatus{
		Failed: 1,
		Conditions: []batchv1.JobCondition{
			{Type: batchv1.JobFailed, Status: corev1.ConditionTrue, Message: "Connection timeout to destination registry"},
		},
	}
	_, err = k8sClient.BatchV1().Jobs(namespace).UpdateStatus(ctx, createdJob3, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update job3 status: %w", err)
	}

	job4 := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "transfer-job-004",
			Namespace: namespace,
			Labels: map[string]string{
				"modelregistry.kubeflow.org/job-type":            "async-upload",
				"modelregistry.kubeflow.org/job-id":              "004",
				"modelregistry.kubeflow.org/model-registry-name": registryName,
			},
			Annotations: map[string]string{
				"modelregistry.kubeflow.org/display-name":   "Transfer job 004",
				"modelregistry.kubeflow.org/configmap-name": "transfer-job-001-config",
				"modelregistry.kubeflow.org/dest-secret":    "transfer-job-001-dest-secret",
				"modelregistry.kubeflow.org/source-secret":  "transfer-job-001-source-secret",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{Name: "async-upload", Image: "ghcr.io/kubeflow/hub/job/async-upload:latest"},
					},
				},
			},
		},
	}
	createdJob4, err := k8sClient.BatchV1().Jobs(namespace).Create(ctx, job4, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create job4: %w", err)
	}
	createdJob4.Status = batchv1.JobStatus{Active: 1}
	_, err = k8sClient.BatchV1().Jobs(namespace).UpdateStatus(ctx, createdJob4, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update job4 status: %w", err)
	}

	return nil
}

func createTransferJobPods(k8sClient kubernetes.Interface, ctx context.Context, namespace string) error {
	// Pod for job1 (RUNNING) - active pod, no termination message
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "transfer-job-001-pod-abc12",
			Namespace: namespace,
			Labels: map[string]string{
				"job-name":                            "transfer-job-001",
				"modelregistry.kubeflow.org/job-type": "async-upload",
				"batch.kubernetes.io/job-name":        "transfer-job-001",
				"batch.kubernetes.io/controller-uid":  "uid-001",
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:  "async-upload",
					Image: "quay.io/opendatahub/model-registry-job-async-upload:latest",
				},
			},
		},
	}

	createdPod1, err := k8sClient.CoreV1().Pods(namespace).Create(ctx, pod1, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create pod for job1: %w", err)
	}

	createdPod1.Status = corev1.PodStatus{
		Phase: corev1.PodRunning,
		ContainerStatuses: []corev1.ContainerStatus{
			{
				Name:  "async-upload",
				State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
			},
		},
	}
	_, err = k8sClient.CoreV1().Pods(namespace).UpdateStatus(ctx, createdPod1, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update pod1 status: %w", err)
	}

	// Pod for job2 (COMPLETED) - succeeded with termination message containing created IDs
	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "transfer-job-002-pod-def34",
			Namespace: namespace,
			Labels: map[string]string{
				"job-name":                            "transfer-job-002",
				"modelregistry.kubeflow.org/job-type": "async-upload",
				"batch.kubernetes.io/job-name":        "transfer-job-002",
				"batch.kubernetes.io/controller-uid":  "uid-002",
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:  "async-upload",
					Image: "quay.io/opendatahub/model-registry-job-async-upload:latest",
				},
			},
		},
	}

	createdPod2, err := k8sClient.CoreV1().Pods(namespace).Create(ctx, pod2, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create pod for job2: %w", err)
	}

	createdPod2.Status = corev1.PodStatus{
		Phase: corev1.PodSucceeded,
		ContainerStatuses: []corev1.ContainerStatus{
			{
				Name: "async-upload",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 0,
						Message:  `{"RegisteredModel":{"id":"2"},"ModelVersion":{"id":"3"},"ModelArtifact":{"id":"5"}}`,
					},
				},
			},
		},
	}
	_, err = k8sClient.CoreV1().Pods(namespace).UpdateStatus(ctx, createdPod2, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update pod2 status: %w", err)
	}

	// Pod for job3 (FAILED) - failed with error in termination message
	pod3 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "transfer-job-003-pod-ghi56",
			Namespace: namespace,
			Labels: map[string]string{
				"job-name":                            "transfer-job-003",
				"modelregistry.kubeflow.org/job-type": "async-upload",
				"batch.kubernetes.io/job-name":        "transfer-job-003",
				"batch.kubernetes.io/controller-uid":  "uid-003",
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:  "async-upload",
					Image: "quay.io/opendatahub/model-registry-job-async-upload:latest",
				},
			},
		},
	}

	createdPod3, err := k8sClient.CoreV1().Pods(namespace).Create(ctx, pod3, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create pod for job3: %w", err)
	}

	createdPod3.Status = corev1.PodStatus{
		Phase: corev1.PodFailed,
		ContainerStatuses: []corev1.ContainerStatus{
			{
				Name: "async-upload",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 1,
						Message:  "Connection timeout: failed to push model artifact to quay.io/test/model-one:v2 after 3 retries",
					},
				},
			},
		},
	}
	_, err = k8sClient.CoreV1().Pods(namespace).UpdateStatus(ctx, createdPod3, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update pod3 status: %w", err)
	}

	return nil
}

func createTransferJobPodEvents(k8sClient kubernetes.Interface, ctx context.Context, namespace string) error {
	baseTime := time.Date(2025, 5, 27, 15, 38, 28, 0, time.UTC)

	podEvents := []struct {
		podName   string
		eventName string
		events    []struct {
			reason  string
			evType  string
			message string
			offset  time.Duration
		}
	}{
		{
			podName:   "transfer-job-001-pod-abc12",
			eventName: "transfer-job-001-pod-abc12",
			events: []struct {
				reason  string
				evType  string
				message string
				offset  time.Duration
			}{
				{"Pulling", "Normal", "Pulling image \"quay.io/opendatahub/model-registry-job-async-upload:latest\"", 0},
				{"Pulled", "Normal", "Successfully pulled image \"quay.io/opendatahub/model-registry-job-async-upload:latest\"", 5 * time.Second},
				{"Created", "Normal", "Created container async-upload", 6 * time.Second},
				{"Started", "Normal", "Started container async-upload", 7 * time.Second},
			},
		},
		{
			podName:   "transfer-job-002-pod-def34",
			eventName: "transfer-job-002-pod-def34",
			events: []struct {
				reason  string
				evType  string
				message string
				offset  time.Duration
			}{
				{"Pulling", "Normal", "Pulling image \"quay.io/opendatahub/model-registry-job-async-upload:latest\"", 0},
				{"Pulled", "Normal", "Successfully pulled image \"quay.io/opendatahub/model-registry-job-async-upload:latest\"", 4 * time.Second},
				{"Created", "Normal", "Created container async-upload", 5 * time.Second},
				{"Started", "Normal", "Started container async-upload", 6 * time.Second},
			},
		},
		{
			podName:   "transfer-job-003-pod-ghi56",
			eventName: "transfer-job-003-pod-ghi56",
			events: []struct {
				reason  string
				evType  string
				message string
				offset  time.Duration
			}{
				{"Pulling", "Normal", "Pulling image \"quay.io/opendatahub/model-registry-job-async-upload:latest\"", 0},
				{"Pulled", "Normal", "Successfully pulled image \"quay.io/opendatahub/model-registry-job-async-upload:latest\"", 3 * time.Second},
				{"Created", "Normal", "Created container async-upload", 4 * time.Second},
				{"Started", "Normal", "Started container async-upload", 5 * time.Second},
				{"BackOff", "Warning", "Back-off restarting failed container async-upload", 30 * time.Second},
			},
		},
	}

	for _, pe := range podEvents {
		for i, ev := range pe.events {
			eventTime := baseTime.Add(ev.offset)
			event := &corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-event-%d", pe.eventName, i),
					Namespace: namespace,
				},
				InvolvedObject: corev1.ObjectReference{
					Kind:      "Pod",
					Name:      pe.podName,
					Namespace: namespace,
				},
				Reason:         ev.reason,
				Message:        ev.message,
				Type:           ev.evType,
				LastTimestamp:  metav1.NewTime(eventTime),
				FirstTimestamp: metav1.NewTime(eventTime),
			}
			_, err := k8sClient.CoreV1().Events(namespace).Create(ctx, event, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create event for pod %s: %w", pe.podName, err)
			}
		}
	}

	return nil
}

func addMcpDataToDefaultCatalogSourcesConfigMap(
	k8sClient kubernetes.Interface,
	ctx context.Context,
	namespace string,
) error {
	cm, err := k8sClient.CoreV1().ConfigMaps(namespace).Get(ctx, k8s.CatalogSourceDefaultConfigMapName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get default-catalog-sources configmap for MCP data: %w", err)
	}

	mcpRaw := strings.TrimSpace(`
mcp_catalogs:
  - name: Community MCP Servers
    id: community_mcp_servers
    type: yaml
    enabled: true
    properties:
      yamlCatalogPath: community_mcp_servers.yaml
    labels:
      - Community

  - name: Verified MCP Servers
    id: verified_mcp_servers
    type: yaml
    enabled: true
    properties:
      yamlCatalogPath: verified_mcp_servers.yaml
    labels:
      - Verified
`)

	existingSources := cm.Data[k8s.McpCatalogSourceKey]
	cm.Data[k8s.McpCatalogSourceKey] = existingSources + "\n" + mcpRaw
	cm.Data["community_mcp_servers.yaml"] = "servers:\n - name: community_server_1"

	if _, err := k8sClient.CoreV1().ConfigMaps(namespace).Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update default-catalog-sources configmap with MCP data: %w", err)
	}

	return nil
}

func createMcpCatalogSourcesConfigMap(
	k8sClient kubernetes.Interface,
	ctx context.Context,
	namespace string,
) error {
	raw := strings.TrimSpace(`
mcp_catalogs:
  - name: Custom MCP Servers
    id: custom_mcp_servers
    type: yaml
    enabled: true
    properties:
      yamlCatalogPath: custom_mcp_servers.yaml
    includedServers:
      - server-*
    excludedServers:
      - test-server-*
    labels:
      - Custom

  - name: Organization MCP
    id: org_mcp_servers
    type: yaml
    enabled: false
    properties:
      yamlCatalogPath: org_mcp_servers.yaml
    labels:
      - Organization
`)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8s.McpCatalogSourceUserConfigMapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			k8s.McpCatalogSourceKey:   raw,
			"custom_mcp_servers.yaml": "servers:\n - name: custom_server_1",
			"org_mcp_servers.yaml":    "servers:\n - name: org_server_1",
		},
	}

	if _, err := k8sClient.CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("failed to create mcp-catalog-sources configmap: %w", err)
	}

	return nil
}

func strPtr(s string) *string {
	return &s
}

func getProjectRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(currentDir, "go.mod")); err == nil {
			// Found the project root where go.mod is located
			return currentDir, nil
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// We reached the root directory and did not find the go.mod
			return "", fmt.Errorf("could not find project root")
		}

		currentDir = parentDir
	}
}
