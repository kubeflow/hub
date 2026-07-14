package middleware

import (
	"net/http"

	"github.com/go-chi/cors"
)

// CORSMiddleware returns a CORS middleware handler. If allowedOrigins is empty,
// the returned middleware is a no-op, making CORS effectively disabled.
// This is the secure default: deployments that do not explicitly configure
// allowed origins will not accept cross-origin requests.
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	if len(allowedOrigins) == 0 {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	})
}
