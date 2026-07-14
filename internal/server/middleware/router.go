package middleware

import (
	"net/http"

	platformmw "github.com/kubeflow/hub/internal/platform/server/middleware"
	"github.com/kubeflow/hub/internal/server/openapi"
)

// WrapWithValidation wraps the auto-generated router with CORS and validation middleware.
// If corsAllowedOrigins is empty, CORS is disabled (no cross-origin headers are added).
func WrapWithValidation(corsAllowedOrigins []string, routers ...openapi.Router) http.Handler {
	baseRouter := openapi.NewRouter(routers...)

	handler := platformmw.CORSMiddleware(corsAllowedOrigins)(baseRouter)

	return platformmw.ValidationMiddleware(handler)
}
