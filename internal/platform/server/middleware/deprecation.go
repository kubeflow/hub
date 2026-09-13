package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/golang/glog"
)

const deprecationLogGap = 60 * time.Second

// DeprecatedPath pairs a deprecated API path prefix with the prefix of the
// version that replaces it. Both should end with a slash, for example
// "/api/model_registry/v1alpha3/" and "/api/model_registry/v1/".
type DeprecatedPath struct {
	// Prefix is matched against the request path.
	Prefix string
	// Successor is advertised in the Link header with rel="successor-version".
	Successor string
}

// DeprecationConfig configures the deprecation middleware.
type DeprecationConfig struct {
	// SunsetDate is the date after which the deprecated APIs may be removed.
	SunsetDate time.Time
	// Paths lists the deprecated prefixes and their successors.
	Paths []DeprecatedPath
}

// DeprecationMiddleware injects RFC 8594 deprecation headers (Deprecation,
// Sunset, Link) on responses to requests under any configured deprecated
// prefix, and logs a warning at most once per minute. Requests to any other
// path pass through unchanged.
func DeprecationMiddleware(cfg DeprecationConfig) func(http.Handler) http.Handler {
	sunsetValue := cfg.SunsetDate.UTC().Format(http.TimeFormat)

	links := make([]string, len(cfg.Paths))
	for i, p := range cfg.Paths {
		links[i] = fmt.Sprintf("<%s>; rel=\"successor-version\"", p.Successor)
	}

	var lastLogUnix atomic.Int64

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for i, p := range cfg.Paths {
				if !strings.HasPrefix(r.URL.Path, p.Prefix) {
					continue
				}

				w.Header().Set("Deprecation", "true")
				w.Header().Set("Sunset", sunsetValue)
				w.Header().Add("Link", links[i])

				now := time.Now().Unix()
				if last := lastLogUnix.Load(); now-last >= int64(deprecationLogGap.Seconds()) {
					if lastLogUnix.CompareAndSwap(last, now) {
						glog.Warningf("deprecated alpha API called: %s %s (sunset: %s)", r.Method, r.URL.Path, sunsetValue)
					}
				}
				break
			}

			next.ServeHTTP(w, r)
		})
	}
}
