package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func dummyHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
}

func TestCORSMiddleware_Disabled(t *testing.T) {
	tests := []struct {
		name    string
		origins []string
	}{
		{"nil origins", nil},
		{"empty slice", []string{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := CORSMiddleware(tc.origins)(dummyHandler())

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", "https://evil.com")
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}

func TestCORSMiddleware_AllowsConfiguredOrigin(t *testing.T) {
	handler := CORSMiddleware([]string{"https://dashboard.example.com"})(dummyHandler())

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://dashboard.example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "https://dashboard.example.com", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_RejectsUnconfiguredOrigin(t *testing.T) {
	handler := CORSMiddleware([]string{"https://dashboard.example.com"})(dummyHandler())

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_PreflightAllowed(t *testing.T) {
	handler := CORSMiddleware([]string{"https://dashboard.example.com"})(dummyHandler())

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://dashboard.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "https://dashboard.example.com", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, rr.Header().Get("Access-Control-Allow-Methods"), "POST")
}

func TestCORSMiddleware_PreflightDisabled(t *testing.T) {
	handler := CORSMiddleware(nil)(dummyHandler())

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_WildcardOrigin(t *testing.T) {
	handler := CORSMiddleware([]string{"*"})(dummyHandler())

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://any-site.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_MultipleOrigins(t *testing.T) {
	origins := []string{"https://dashboard.example.com", "https://admin.example.com"}
	handler := CORSMiddleware(origins)(dummyHandler())

	for _, origin := range origins {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", origin)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, origin, rr.Header().Get("Access-Control-Allow-Origin"),
			"expected origin %s to be allowed", origin)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://other.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_CredentialsDisabled(t *testing.T) {
	handler := CORSMiddleware([]string{"https://dashboard.example.com"})(dummyHandler())

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://dashboard.example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORSMiddleware_PreflightPATCH(t *testing.T) {
	handler := CORSMiddleware([]string{"https://dashboard.example.com"})(dummyHandler())

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://dashboard.example.com")
	req.Header.Set("Access-Control-Request-Method", "PATCH")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "https://dashboard.example.com", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, rr.Header().Get("Access-Control-Allow-Methods"), "PATCH")
}
