package mcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog"
	openapi "github.com/kubeflow/hub/catalog/pkg/openapi"
	"github.com/kubeflow/hub/pkg/api"
)

type mockProvider struct {
	servers  map[string]*openapi.MCPServer
	getError error
}

func (m *mockProvider) GetMCPServer(_ context.Context, serverID string, _ bool, _ int32) (*openapi.MCPServer, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	s, ok := m.servers[serverID]
	if !ok {
		return nil, nil
	}
	return s, nil
}

func (m *mockProvider) ListMCPServers(context.Context, mcpcatalog.ListMCPServersParams) (openapi.MCPServerList, error) {
	return openapi.MCPServerList{}, nil
}
func (m *mockProvider) ListMCPServerTools(context.Context, string, mcpcatalog.ListMCPServerToolsParams) (openapi.MCPToolsList, error) {
	return openapi.MCPToolsList{}, nil
}
func (m *mockProvider) GetMCPServerTool(context.Context, string, string) (*openapi.MCPTool, error) {
	return nil, nil
}
func (m *mockProvider) GetFilterOptions(context.Context) (*openapi.FilterOptionsList, error) {
	return nil, nil
}

func newTestRouter(provider mcpcatalog.MCPCatalogProvider) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/api/mcp_catalog/v1alpha1/mcp_servers/{server_id}/logo", LogoHandler(provider))
	return r
}

func serverWithLogo(logo string) *openapi.MCPServer {
	s := &openapi.MCPServer{}
	s.SetLogo(logo)
	return s
}

func TestLogoHandler_ServerNotFound(t *testing.T) {
	provider := &mockProvider{servers: map[string]*openapi.MCPServer{}}
	r := newTestRouter(provider)

	req := httptest.NewRequest(http.MethodGet, "/api/mcp_catalog/v1alpha1/mcp_servers/999/logo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLogoHandler_NoLogo(t *testing.T) {
	provider := &mockProvider{
		servers: map[string]*openapi.MCPServer{
			"1": {},
		},
	}
	r := newTestRouter(provider)

	req := httptest.NewRequest(http.MethodGet, "/api/mcp_catalog/v1alpha1/mcp_servers/1/logo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLogoHandler_DataURI_SVG(t *testing.T) {
	svgContent := `<svg xmlns="http://www.w3.org/2000/svg"><circle r="10"/></svg>`
	dataURI := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(svgContent))

	provider := &mockProvider{
		servers: map[string]*openapi.MCPServer{
			"1": serverWithLogo(dataURI),
		},
	}
	r := newTestRouter(provider)

	req := httptest.NewRequest(http.MethodGet, "/api/mcp_catalog/v1alpha1/mcp_servers/1/logo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/svg+xml", w.Header().Get("Content-Type"))
	assert.Equal(t, "public, max-age=3600", w.Header().Get("Cache-Control"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Contains(t, w.Header().Get("Content-Security-Policy"), "default-src 'none'")
	assert.Contains(t, w.Header().Get("Content-Security-Policy"), "sandbox")
	assert.Equal(t, svgContent, w.Body.String())
}

func TestLogoHandler_DataURI_PNG(t *testing.T) {
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	dataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngData)

	provider := &mockProvider{
		servers: map[string]*openapi.MCPServer{
			"1": serverWithLogo(dataURI),
		},
	}
	r := newTestRouter(provider)

	req := httptest.NewRequest(http.MethodGet, "/api/mcp_catalog/v1alpha1/mcp_servers/1/logo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
	assert.Equal(t, "inline", w.Header().Get("Content-Disposition"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	// CSP is only needed for active formats like SVG; raster images don't get it.
	assert.Empty(t, w.Header().Get("Content-Security-Policy"))
	assert.Equal(t, pngData, w.Body.Bytes())
}

func TestLogoHandler_PlainURL_Redirect(t *testing.T) {
	logoURL := "https://example.com/logo.png"
	provider := &mockProvider{
		servers: map[string]*openapi.MCPServer{
			"1": serverWithLogo(logoURL),
		},
	}
	r := newTestRouter(provider)

	req := httptest.NewRequest(http.MethodGet, "/api/mcp_catalog/v1alpha1/mcp_servers/1/logo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, logoURL, w.Header().Get("Location"))
}

func TestLogoHandler_MalformedDataURI(t *testing.T) {
	provider := &mockProvider{
		servers: map[string]*openapi.MCPServer{
			"1": serverWithLogo("data:image/png;base64"),
		},
	}
	r := newTestRouter(provider)

	req := httptest.NewRequest(http.MethodGet, "/api/mcp_catalog/v1alpha1/mcp_servers/1/logo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogoHandler_InvalidBase64(t *testing.T) {
	provider := &mockProvider{
		servers: map[string]*openapi.MCPServer{
			"1": serverWithLogo("data:image/png;base64,!!!not-valid-base64!!!"),
		},
	}
	r := newTestRouter(provider)

	req := httptest.NewRequest(http.MethodGet, "/api/mcp_catalog/v1alpha1/mcp_servers/1/logo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogoHandler_ProviderError(t *testing.T) {
	provider := &mockProvider{
		getError: fmt.Errorf("database connection failed"),
	}
	r := newTestRouter(provider)

	req := httptest.NewRequest(http.MethodGet, "/api/mcp_catalog/v1alpha1/mcp_servers/1/logo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestLogoHandler_ProviderNotFoundError(t *testing.T) {
	provider := &mockProvider{
		getError: fmt.Errorf("server not found: %w", api.ErrNotFound),
	}
	r := newTestRouter(provider)

	req := httptest.NewRequest(http.MethodGet, "/api/mcp_catalog/v1alpha1/mcp_servers/999/logo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLogoHandler_ProviderBadRequestError(t *testing.T) {
	provider := &mockProvider{
		getError: fmt.Errorf("invalid ID: %w", api.ErrBadRequest),
	}
	r := newTestRouter(provider)

	req := httptest.NewRequest(http.MethodGet, "/api/mcp_catalog/v1alpha1/mcp_servers/abc/logo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogoHandler_DataURI_PlainText(t *testing.T) {
	// A non-base64 data URI percent-encodes its payload (RFC 2397); it must be
	// decoded to the real bytes rather than served with literal %XX escapes.
	dataURI := "data:image/svg+xml,%3Csvg%3E%3C/svg%3E"
	provider := &mockProvider{
		servers: map[string]*openapi.MCPServer{
			"1": serverWithLogo(dataURI),
		},
	}
	r := newTestRouter(provider)

	req := httptest.NewRequest(http.MethodGet, "/api/mcp_catalog/v1alpha1/mcp_servers/1/logo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/svg+xml", w.Header().Get("Content-Type"))
	assert.Equal(t, "<svg></svg>", w.Body.String())
}

func TestLogoHandler_DataURI_PlainText_TooLarge(t *testing.T) {
	// An oversized percent-encoded (non-base64) payload must be rejected before
	// unescaping, based on the encoded length.
	big := strings.Repeat("A", (1<<20)+1)
	dataURI := "data:image/svg+xml," + big
	provider := &mockProvider{
		servers: map[string]*openapi.MCPServer{
			"1": serverWithLogo(dataURI),
		},
	}
	r := newTestRouter(provider)

	req := httptest.NewRequest(http.MethodGet, "/api/mcp_catalog/v1alpha1/mcp_servers/1/logo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

func TestLogoHandler_DataURI_RejectsNonImageMediaType(t *testing.T) {
	// A logo that claims to be HTML must not be served with that Content-Type,
	// otherwise it becomes stored XSS in the catalog origin.
	htmlPayload := `<script>alert(document.domain)</script>`
	dataURI := "data:text/html;base64," + base64.StdEncoding.EncodeToString([]byte(htmlPayload))
	provider := &mockProvider{
		servers: map[string]*openapi.MCPServer{
			"1": serverWithLogo(dataURI),
		},
	}
	r := newTestRouter(provider)

	req := httptest.NewRequest(http.MethodGet, "/api/mcp_catalog/v1alpha1/mcp_servers/1/logo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)
	assert.NotContains(t, w.Body.String(), "<script>")
}

func TestLogoHandler_DataURI_CaseInsensitiveScheme(t *testing.T) {
	// The URI scheme is case-insensitive; an uppercase DATA: must still be
	// decoded and served, not treated as a redirect target.
	svgContent := `<svg xmlns="http://www.w3.org/2000/svg"></svg>`
	dataURI := "DATA:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(svgContent))
	provider := &mockProvider{
		servers: map[string]*openapi.MCPServer{
			"1": serverWithLogo(dataURI),
		},
	}
	r := newTestRouter(provider)

	req := httptest.NewRequest(http.MethodGet, "/api/mcp_catalog/v1alpha1/mcp_servers/1/logo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/svg+xml", w.Header().Get("Content-Type"))
	assert.Equal(t, svgContent, w.Body.String())
}

func TestLogoHandler_RejectsNonHTTPRedirect(t *testing.T) {
	for _, logo := range []string{
		"javascript:alert(document.domain)",
		"ftp://example.com/logo.png",
		"//evil.example/logo.png",
	} {
		provider := &mockProvider{
			servers: map[string]*openapi.MCPServer{
				"1": serverWithLogo(logo),
			},
		}
		r := newTestRouter(provider)

		req := httptest.NewRequest(http.MethodGet, "/api/mcp_catalog/v1alpha1/mcp_servers/1/logo", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "logo %q should be rejected", logo)
		assert.Empty(t, w.Header().Get("Location"), "logo %q should not redirect", logo)
	}
}

func TestLogoHandler_DataURI_TooLarge(t *testing.T) {
	// A decoded payload above the cap is rejected without being served.
	big := strings.Repeat("A", (1<<20)+1)
	dataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte(big))
	provider := &mockProvider{
		servers: map[string]*openapi.MCPServer{
			"1": serverWithLogo(dataURI),
		},
	}
	r := newTestRouter(provider)

	req := httptest.NewRequest(http.MethodGet, "/api/mcp_catalog/v1alpha1/mcp_servers/1/logo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

func TestRegisterLogoRoutes_ServesAllAPIVersions(t *testing.T) {
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	dataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngData)
	provider := &mockProvider{
		servers: map[string]*openapi.MCPServer{
			"1": serverWithLogo(dataURI),
		},
	}
	r := chi.NewRouter()
	registerLogoRoutes(r, provider)

	for _, version := range []string{"v1alpha1", "v1"} {
		t.Run(version, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/mcp_catalog/%s/mcp_servers/1/logo", version), nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
			assert.Equal(t, pngData, w.Body.Bytes())
		})
	}
}
