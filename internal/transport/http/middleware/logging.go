package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/saurabh/distributed-rate-limiter/pkg/httputil"
)

// Logging logs one structured line per request. Health and metrics probes are skipped.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if skipRequestLog(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		sw := httputil.NewStatusWriter(w)
		next.ServeHTTP(sw, r)

		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.Status,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	})
}

func skipRequestLog(path string) bool {
	switch path {
	case "/health/live", "/metrics":
		return true
	default:
		return false
	}
}
