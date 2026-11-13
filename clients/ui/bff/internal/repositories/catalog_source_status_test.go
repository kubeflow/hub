package repositories

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"
	"github.com/stretchr/testify/require"
)

func TestCatalogSourceStatusClear(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodDelete, r.Method)
		require.Equal(t, "/sources/mcp_source/status", r.URL.Path)
		require.Equal(t, int64(0), r.ContentLength)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := httpclient.NewHTTPClient(slog.Default(), server.URL, nil, false, nil)
	require.NoError(t, err)

	err = (CatalogSourceStatus{}).ClearSourceStatus(client, "mcp_source")
	require.NoError(t, err)
}

func TestCatalogSourceStatusClearReturnsUpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/sources/mcp_source/status", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := httpclient.NewHTTPClient(slog.Default(), server.URL, nil, false, nil)
	require.NoError(t, err)

	err = (CatalogSourceStatus{}).ClearSourceStatus(client, "mcp_source")
	require.Error(t, err)
	require.ErrorContains(t, err, "HTTP 500")
}
