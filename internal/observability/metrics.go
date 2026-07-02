package observability

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests handled by the rate limiter API.",
	}, []string{"method", "route", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})

	rateLimitChecksTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ratelimit_checks_total",
		Help: "Total rate limit check evaluations.",
	}, []string{"result"})

	rateLimitCheckDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ratelimit_check_duration_seconds",
		Help:    "Rate limit check evaluation latency in seconds.",
		Buckets: []float64{.001, .0025, .005, .01, .025, .05, .1, .25, .5, 1},
	})
)

// RecordHTTPRequest records request count and duration for Prometheus.
func RecordHTTPRequest(method, path string, status int, duration time.Duration) {
	route := NormalizeRoute(path)
	httpRequestsTotal.WithLabelValues(method, route, statusClass(status)).Inc()
	httpRequestDuration.WithLabelValues(method, route).Observe(duration.Seconds())
}

// RecordRateLimitCheck records business metrics for POST /v1/check evaluations.
func RecordRateLimitCheck(decision entity.RateLimitDecision, err error, duration time.Duration) {
	rateLimitCheckDuration.Observe(duration.Seconds())
	rateLimitChecksTotal.WithLabelValues(classifyCheckResult(decision, err)).Inc()
}

func classifyCheckResult(decision entity.RateLimitDecision, err error) string {
	if err != nil {
		return "error"
	}
	if decision.Algorithm == "" && decision.Limit <= 0 {
		return "no_rule"
	}
	if decision.Allowed {
		return "allowed"
	}
	return "denied"
}

// NormalizeRoute maps dynamic paths to stable labels for low-cardinality metrics.
func NormalizeRoute(path string) string {
	switch {
	case path == "/health/live", path == "/health/ready", path == "/metrics":
		return path
	case path == "/v1/check", path == "/v1/config/tenants":
		return path
	case strings.HasPrefix(path, "/v1/config/tenants/") && strings.HasSuffix(path, "/rules"):
		return "/v1/config/tenants/{id}/rules"
	default:
		return "unknown"
	}
}

func statusClass(code int) string {
	switch {
	case code >= 500:
		return "5xx"
	case code >= 400:
		return "4xx"
	case code >= 300:
		return "3xx"
	case code >= 200:
		return "2xx"
	default:
		return "other"
	}
}
