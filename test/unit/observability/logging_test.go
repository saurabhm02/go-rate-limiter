package observability_test

import (
	"testing"

	"github.com/saurabh/distributed-rate-limiter/internal/observability"
)

func TestNewLoggerLevels(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "error", "verbose"} {
		if observability.NewLogger(level) == nil {
			t.Fatalf("nil logger for level %q", level)
		}
	}
}

func TestNormalizeRoute(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "/health/live", want: "/health/live"},
		{path: "/metrics", want: "/metrics"},
		{path: "/v1/check", want: "/v1/check"},
		{path: "/v1/config/tenants/550e8400-e29b-41d4-a716-446655440000/rules", want: "/v1/config/tenants/{id}/rules"},
		{path: "/other", want: "unknown"},
	}

	for _, tt := range tests {
		if got := observability.NormalizeRoute(tt.path); got != tt.want {
			t.Fatalf("NormalizeRoute(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
