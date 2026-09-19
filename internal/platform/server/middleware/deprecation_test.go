package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func stubHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func registryDeprecation(sunset time.Time) DeprecationConfig {
	return DeprecationConfig{
		SunsetDate: sunset,
		Paths:      []DeprecatedPath{{Prefix: "/api/model_registry/v1alpha3/", Successor: "/api/model_registry/v1/"}},
	}
}

func TestDeprecationMiddleware_AlphaPath(t *testing.T) {
	sunset := time.Date(2027, 4, 29, 0, 0, 0, 0, time.UTC)
	handler := DeprecationMiddleware(registryDeprecation(sunset))(stubHandler())

	req := httptest.NewRequest("GET", "/api/model_registry/v1alpha3/registered_models", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "true", rr.Header().Get("Deprecation"))
	assert.Equal(t, sunset.Format(http.TimeFormat), rr.Header().Get("Sunset"))
	assert.Equal(t, `</api/model_registry/v1/>; rel="successor-version"`, rr.Header().Get("Link"))
}

func TestDeprecationMiddleware_V1Path(t *testing.T) {
	sunset := time.Date(2027, 4, 29, 0, 0, 0, 0, time.UTC)
	handler := DeprecationMiddleware(registryDeprecation(sunset))(stubHandler())

	req := httptest.NewRequest("GET", "/api/model_registry/v1/registered_models", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, rr.Header().Get("Deprecation"))
	assert.Empty(t, rr.Header().Get("Sunset"))
	assert.Empty(t, rr.Header().Get("Link"))
}

func TestDeprecationMiddleware_NonAPIPath(t *testing.T) {
	sunset := time.Date(2027, 4, 29, 0, 0, 0, 0, time.UTC)
	handler := DeprecationMiddleware(registryDeprecation(sunset))(stubHandler())

	req := httptest.NewRequest("GET", "/readyz/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, rr.Header().Get("Deprecation"))
	assert.Empty(t, rr.Header().Get("Sunset"))
	assert.Empty(t, rr.Header().Get("Link"))
}

func TestDeprecationMiddleware_PreservesDownstreamLinkHeader(t *testing.T) {
	sunset := time.Date(2027, 4, 29, 0, 0, 0, 0, time.UTC)
	downstream := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Link", `<https://example.com/page2>; rel="next"`)
		w.WriteHeader(http.StatusOK)
	})
	handler := DeprecationMiddleware(registryDeprecation(sunset))(downstream)

	req := httptest.NewRequest("GET", "/api/model_registry/v1alpha3/registered_models", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	links := rr.Header().Values("Link")
	assert.Contains(t, links, `</api/model_registry/v1/>; rel="successor-version"`)
	assert.Contains(t, links, `<https://example.com/page2>; rel="next"`)
}

func TestDeprecationMiddleware_CallsInnerHandler(t *testing.T) {
	sunset := time.Date(2027, 4, 29, 0, 0, 0, 0, time.UTC)
	called := false
	downstream := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := DeprecationMiddleware(registryDeprecation(sunset))(downstream)

	req := httptest.NewRequest("GET", "/api/model_registry/v1alpha3/registered_models", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.True(t, called)
}

func TestDeprecationMiddleware_MultiplePaths(t *testing.T) {
	sunset := time.Date(2027, 4, 29, 0, 0, 0, 0, time.UTC)
	handler := DeprecationMiddleware(DeprecationConfig{
		SunsetDate: sunset,
		Paths: []DeprecatedPath{
			{Prefix: "/api/model_catalog/v1alpha1/", Successor: "/api/model_catalog/v1/"},
			{Prefix: "/api/mcp_catalog/v1alpha1/", Successor: "/api/mcp_catalog/v1/"},
		},
	})(stubHandler())

	tests := []struct {
		path     string
		wantLink string
	}{
		{"/api/model_catalog/v1alpha1/models", `</api/model_catalog/v1/>; rel="successor-version"`},
		{"/api/mcp_catalog/v1alpha1/mcp_servers", `</api/mcp_catalog/v1/>; rel="successor-version"`},
		{"/api/model_catalog/v1/models", ""},
		{"/api/skill_catalog/v1/skills", ""},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, httptest.NewRequest("GET", tt.path, nil))

			assert.Equal(t, http.StatusOK, rr.Code)
			if tt.wantLink == "" {
				assert.Empty(t, rr.Header().Get("Deprecation"))
				assert.Empty(t, rr.Header().Get("Sunset"))
				assert.Empty(t, rr.Header().Get("Link"))
				return
			}
			assert.Equal(t, "true", rr.Header().Get("Deprecation"))
			assert.Equal(t, sunset.Format(http.TimeFormat), rr.Header().Get("Sunset"))
			assert.Equal(t, []string{tt.wantLink}, rr.Header().Values("Link"))
		})
	}
}

func TestDeprecationMiddleware_NoPaths(t *testing.T) {
	sunset := time.Date(2027, 4, 29, 0, 0, 0, 0, time.UTC)
	handler := DeprecationMiddleware(DeprecationConfig{SunsetDate: sunset})(stubHandler())

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/api/model_registry/v1alpha3/registered_models", nil))

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, rr.Header().Get("Deprecation"))
}
