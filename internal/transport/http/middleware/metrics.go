package middleware

import (
	"net/http"
	"time"

	"github.com/saurabh/distributed-rate-limiter/internal/observability"
	"github.com/saurabh/distributed-rate-limiter/pkg/httputil"
)

// Metrics records Prometheus HTTP request metrics.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := httputil.NewStatusWriter(w)
		next.ServeHTTP(sw, r)
		observability.RecordHTTPRequest(r.Method, r.URL.Path, sw.Status, time.Since(start))
	})
}
