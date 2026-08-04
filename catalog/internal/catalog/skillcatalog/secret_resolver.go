package skillcatalog

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// validK8sName matches Kubernetes resource names (RFC 1123 DNS subdomain).
// Validated before constructing the secret URL to prevent path traversal.
var validK8sName = regexp.MustCompile(`^[a-z0-9]([a-z0-9.\-]*[a-z0-9])?$`)

// Standard in-cluster service account mount paths.
const (
	serviceAccountTokenPath     = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	serviceAccountCACertPath    = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	serviceAccountNamespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

// K8sSecretResolver fetches git credentials from Kubernetes Secrets in the
// pod's own namespace via the Kubernetes API server, using the pod's service
// account token. It talks to the API server over plain net/http rather than
// client-go, since the catalog service otherwise has no Kubernetes API
// dependency.
type K8sSecretResolver struct {
	baseURL   string
	namespace string
	// readToken returns the current bearer token. It re-reads the mounted
	// service-account token on every call so a rotated (projected) token — which
	// kubelet rewrites on disk, ~hourly by default — is picked up rather than a
	// stale cached value that would start returning 401s.
	readToken  func() (string, error)
	httpClient *http.Client
}

// NewK8sSecretResolver builds a resolver using the pod's in-cluster service
// account (token, CA certificate, namespace) and the standard
// KUBERNETES_SERVICE_HOST/PORT environment variables. It returns an error when
// not running in a cluster (e.g. local development), so the caller can log a
// warning and continue without Secret-backed auth rather than failing to start.
func NewK8sSecretResolver() (*K8sSecretResolver, error) {
	// Read once up front to fail fast when not running in a cluster; the token is
	// re-read per request (see readToken) so rotation is handled.
	if _, err := os.ReadFile(serviceAccountTokenPath); err != nil {
		return nil, fmt.Errorf("reading service account token: %w", err)
	}
	caCert, err := os.ReadFile(serviceAccountCACertPath)
	if err != nil {
		return nil, fmt.Errorf("reading service account CA certificate: %w", err)
	}
	namespace, err := os.ReadFile(serviceAccountNamespacePath)
	if err != nil {
		return nil, fmt.Errorf("reading service account namespace: %w", err)
	}

	host := os.Getenv("KUBERNETES_SERVICE_HOST")
	port := os.Getenv("KUBERNETES_SERVICE_PORT")
	if host == "" || port == "" {
		return nil, fmt.Errorf("KUBERNETES_SERVICE_HOST/PORT not set; not running in a cluster")
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("no valid certificates found in service account CA file")
	}

	return &K8sSecretResolver{
		baseURL:   "https://" + net.JoinHostPort(host, port),
		namespace: strings.TrimSpace(string(namespace)),
		readToken: func() (string, error) {
			b, err := os.ReadFile(serviceAccountTokenPath)
			if err != nil {
				return "", fmt.Errorf("reading service account token: %w", err)
			}
			return strings.TrimSpace(string(b)), nil
		},
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool}},
		},
	}, nil
}

// secretResponse is the subset of a corev1.Secret this resolver reads. Secret
// data values are base64-encoded, per the Kubernetes API.
type secretResponse struct {
	Data map[string]string `json:"data"`
}

// GetCredentials fetches the named Secret from the resolver's namespace and
// extracts git credentials from it: a "token" or "password" key is the
// credential; an optional "username" key defaults to "x-access-token"
// (matching common PAT-as-password conventions) when absent.
func (r *K8sSecretResolver) GetCredentials(ctx context.Context, secretName string) (*Credentials, error) {
	if !validK8sName.MatchString(secretName) {
		return nil, fmt.Errorf("invalid secret name %q: must be a valid Kubernetes resource name", secretName)
	}
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/secrets/%s", r.baseURL, r.namespace, secretName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building request for secret %q: %w", secretName, err)
	}
	token, err := r.readToken()
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching secret %q: %w", secretName, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("secret %q not found in namespace %q", secretName, r.namespace)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetching secret %q: unexpected status %d: %s", secretName, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var secret secretResponse
	if err := json.NewDecoder(resp.Body).Decode(&secret); err != nil {
		return nil, fmt.Errorf("decoding secret %q: %w", secretName, err)
	}

	return credentialsFromSecretData(secretName, secret.Data)
}

// credentialsFromSecretData extracts git credentials from a Secret's data map.
func credentialsFromSecretData(secretName string, data map[string]string) (*Credentials, error) {
	token, ok := decodeSecretField(data, "token")
	if !ok {
		token, ok = decodeSecretField(data, "password")
	}
	if !ok {
		return nil, fmt.Errorf("secret %q has neither a %q nor a %q field", secretName, "token", "password")
	}

	username, ok := decodeSecretField(data, "username")
	if !ok {
		username = "x-access-token"
	}

	return &Credentials{Username: username, Token: token}, nil
}

// decodeSecretField base64-decodes a single field from Secret data.
func decodeSecretField(data map[string]string, key string) (string, bool) {
	raw, ok := data[key]
	if !ok {
		return "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(decoded) == 0 {
		return "", false // an undecodable or empty value is treated as absent
	}
	return string(decoded), true
}
