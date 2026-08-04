package skillcatalog

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

func newTestK8sSecretResolver(baseURL string) *K8sSecretResolver {
	return &K8sSecretResolver{
		baseURL:    baseURL,
		namespace:  "test-ns",
		readToken:  func() (string, error) { return "test-token", nil },
		httpClient: http.DefaultClient,
	}
}

func TestCredentialsFromSecretData_TokenField(t *testing.T) {
	creds, err := credentialsFromSecretData("s", map[string]string{
		"token":    b64("secret-token"),
		"username": b64("bot"),
	})
	require.NoError(t, err)
	assert.Equal(t, "bot", creds.Username)
	assert.Equal(t, "secret-token", creds.Token)
}

func TestCredentialsFromSecretData_PasswordFieldFallback(t *testing.T) {
	creds, err := credentialsFromSecretData("s", map[string]string{
		"password": b64("pat-value"),
	})
	require.NoError(t, err)
	assert.Equal(t, "pat-value", creds.Token)
}

func TestCredentialsFromSecretData_UsernameDefaultsWhenAbsent(t *testing.T) {
	creds, err := credentialsFromSecretData("s", map[string]string{"token": b64("t")})
	require.NoError(t, err)
	assert.Equal(t, "x-access-token", creds.Username)
}

func TestCredentialsFromSecretData_MissingBothFieldsErrors(t *testing.T) {
	_, err := credentialsFromSecretData("git-creds", map[string]string{"username": b64("bot")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git-creds")
}

func TestCredentialsFromSecretData_UndecodableValueTreatedAsAbsent(t *testing.T) {
	_, err := credentialsFromSecretData("s", map[string]string{"token": "not-valid-base64!!"})
	require.Error(t, err, "an undecodable token field must not be treated as a valid credential")
}

func TestCredentialsFromSecretData_EmptyValueTreatedAsAbsent(t *testing.T) {
	_, err := credentialsFromSecretData("git-creds", map[string]string{"token": b64("")})
	require.Error(t, err, "an empty token must not count as a credential (would silently fall back to anonymous)")
}

func TestK8sSecretResolver_ReReadsTokenPerRequest(t *testing.T) {
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"token":"` + b64("t") + `"}}`))
	}))
	defer server.Close()

	tokens := []string{"tok-1", "tok-2"}
	i := 0
	r := newTestK8sSecretResolver(server.URL)
	r.readToken = func() (string, error) {
		tok := tokens[i]
		i++
		return tok, nil
	}

	_, err := r.GetCredentials(context.Background(), "s")
	require.NoError(t, err)
	_, err = r.GetCredentials(context.Background(), "s")
	require.NoError(t, err)

	assert.Equal(t, []string{"Bearer tok-1", "Bearer tok-2"}, seen,
		"each request must use a freshly read token, not a cached one")
}

func TestK8sSecretResolver_GetCredentials_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/namespaces/test-ns/secrets/git-creds", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"token":"` + b64("s3cr3t") + `","username":"` + b64("deploy-bot") + `"}}`))
	}))
	defer server.Close()

	r := newTestK8sSecretResolver(server.URL)
	creds, err := r.GetCredentials(context.Background(), "git-creds")
	require.NoError(t, err)
	assert.Equal(t, "deploy-bot", creds.Username)
	assert.Equal(t, "s3cr3t", creds.Token)
}

func TestK8sSecretResolver_GetCredentials_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	r := newTestK8sSecretResolver(server.URL)
	_, err := r.GetCredentials(context.Background(), "missing-secret")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestK8sSecretResolver_GetCredentials_InvalidSecretNameRejected(t *testing.T) {
	r := newTestK8sSecretResolver("http://unused")
	for _, name := range []string{
		"../other-ns/admin-token",
		"../../etc/passwd",
		"UPPERCASE",
		"has spaces",
		"trailing-",
		"-leading",
		"",
	} {
		_, err := r.GetCredentials(context.Background(), name)
		require.Errorf(t, err, "expected error for secret name %q", name)
		assert.Contains(t, err.Error(), "invalid secret name", "name=%q", name)
	}
}

func TestK8sSecretResolver_GetCredentials_ServerErrorPropagates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("no RBAC access"))
	}))
	defer server.Close()

	r := newTestK8sSecretResolver(server.URL)
	_, err := r.GetCredentials(context.Background(), "git-creds")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}
