package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func stubHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
}

func TestWrapWithValidation_CORSDisabled(t *testing.T) {
	handler := WrapWithValidation(nil, stubHandler())

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestWrapWithValidation_CORSEnabled(t *testing.T) {
	handler := WrapWithValidation([]string{"https://dashboard.example.com"}, stubHandler())

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://dashboard.example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "https://dashboard.example.com", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestWrapWithValidation_CORSAndValidation(t *testing.T) {
	handler := WrapWithValidation([]string{"https://dashboard.example.com"}, stubHandler())

	req := httptest.NewRequest("GET", "/test?name=test%00invalid", nil)
	req.Header.Set("Origin", "https://dashboard.example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
