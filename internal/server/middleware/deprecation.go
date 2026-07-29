package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/golang/glog"
)

const (
	alphaPathPrefix   = "/api/model_registry/v1alpha3/"
	v1SuccessorPath   = "/api/model_registry/v1/"
	deprecationLogGap = 60 * time.Second
)

// DeprecationConfig configures the deprecation middleware.
type DeprecationConfig struct {
	// SunsetDate is the date after which the deprecated alpha API may be removed.
	SunsetDate time.Time
}

// DeprecationMiddleware injects RFC 8594 deprecation headers (Deprecation,
// Sunset, Link) on responses to alpha (v1alpha3) API requests. Requests to
// any other path pass through unchanged.
func DeprecationMiddleware(cfg DeprecationConfig) func(http.Handler) http.Handler {
	sunsetValue := cfg.SunsetDate.UTC().Format(http.TimeFormat)
	linkValue := fmt.Sprintf("<%s>; rel=\"successor-version\"", v1SuccessorPath)

	var lastLogUnix atomic.Int64

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, alphaPathPrefix) {
				w.Header().Set("Deprecation", "true")
				w.Header().Set("Sunset", sunsetValue)
				w.Header().Add("Link", linkValue)

				now := time.Now().Unix()
				if last := lastLogUnix.Load(); now-last >= int64(deprecationLogGap.Seconds()) {
					if lastLogUnix.CompareAndSwap(last, now) {
						glog.Warningf("deprecated alpha API called: %s %s (sunset: %s)", r.Method, r.URL.Path, sunsetValue)
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
