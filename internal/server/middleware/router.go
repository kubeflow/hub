package middleware

import (
	"net/http"

	platformmw "github.com/kubeflow/hub/internal/platform/server/middleware"
)

// WrapWithValidation wraps a handler with CORS and validation middleware.
// If corsAllowedOrigins is empty, CORS is disabled (no cross-origin headers are added).
func WrapWithValidation(corsAllowedOrigins []string, handler http.Handler) http.Handler {
	handler = platformmw.CORSMiddleware(corsAllowedOrigins)(handler)

	return platformmw.ValidationMiddleware(handler)
}
