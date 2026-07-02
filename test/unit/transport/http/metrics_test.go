package httptransport_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
	"github.com/saurabh/distributed-rate-limiter/internal/observability"
	httptransport "github.com/saurabh/distributed-rate-limiter/internal/transport/http"
)

func TestMetricsEndpoint(t *testing.T) {
	observability.RecordHTTPRequest("GET", "/metrics", 200, 0)
	observability.RecordRateLimitCheck(entity.RateLimitDecision{Allowed: true}, nil, 0)

	mux := httptransport.NewMux(httptransport.Dependencies{})

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	for _, name := range []string{"http_requests_total", "ratelimit_checks_total", "go_goroutines"} {
		if !strings.Contains(body, name) {
			t.Fatalf("missing metric %q in body (first 500 chars): %s", name, body[:min(500, len(body))])
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
