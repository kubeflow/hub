package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kubeflow/hub/internal/server/openapi"
	"github.com/stretchr/testify/assert"
)

type stubRouter struct{}

func (s *stubRouter) Routes() openapi.Routes {
	return openapi.Routes{
		"test": {Name: "test", Method: "GET", Pattern: "/test", HandlerFunc: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		}},
	}
}

func (s *stubRouter) OrderedRoutes() []openapi.Route {
	routes := s.Routes()
	result := make([]openapi.Route, 0, len(routes))
	for _, r := range routes {
		result = append(result, r)
	}
	return result
}

func TestWrapWithValidation_CORSDisabled(t *testing.T) {
	handler := WrapWithValidation(nil, &stubRouter{})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestWrapWithValidation_CORSEnabled(t *testing.T) {
	handler := WrapWithValidation([]string{"https://dashboard.example.com"}, &stubRouter{})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://dashboard.example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "https://dashboard.example.com", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestWrapWithValidation_CORSAndValidation(t *testing.T) {
	handler := WrapWithValidation([]string{"https://dashboard.example.com"}, &stubRouter{})

	req := httptest.NewRequest("GET", "/test?name=test%00invalid", nil)
	req.Header.Set("Origin", "https://dashboard.example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
