package entity_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
)

func TestRuleMatches(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		route   string
		want    bool
	}{
		{name: "default matches anything", pattern: "*", route: "/anything", want: true},
		{name: "exact match", pattern: "/v1/check", route: "/v1/check", want: true},
		{name: "exact mismatch", pattern: "/v1/check", route: "/v1/other", want: false},
		{name: "prefix match", pattern: "/api/payments*", route: "/api/payments/123", want: true},
		{name: "prefix boundary", pattern: "/api/payments*", route: "/api/payments", want: true},
		{name: "prefix too short", pattern: "/api/payments*", route: "/api/orders", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := entity.Rule{RoutePattern: tt.pattern, Algorithm: entity.AlgorithmSlidingWindow, LimitCount: 10, WindowSeconds: 60}
			if got := rule.Matches(tt.route); got != tt.want {
				t.Fatalf("Matches(%q) = %v, want %v", tt.route, got, tt.want)
			}
		})
	}
}

func TestRuleSpecificity(t *testing.T) {
	exact := entity.Rule{RoutePattern: "/v1/check", Algorithm: entity.AlgorithmSlidingWindow, LimitCount: 1, WindowSeconds: 1}
	prefix := entity.Rule{RoutePattern: "/api/*", Algorithm: entity.AlgorithmSlidingWindow, LimitCount: 1, WindowSeconds: 1}
	def := entity.Rule{RoutePattern: "*", Algorithm: entity.AlgorithmSlidingWindow, LimitCount: 1, WindowSeconds: 1}

	route := "/v1/check"
	if exact.Specificity(route) <= prefix.Specificity(route) {
		t.Fatalf("exact should beat prefix on specificity")
	}
	if prefix.Specificity("/api/foo") <= def.Specificity("/api/foo") {
		t.Fatalf("prefix should beat default on specificity")
	}
}

func TestRuleValidate(t *testing.T) {
	base := entity.Rule{
		TenantID:      uuid.New(),
		RoutePattern:  "/v1/check",
		Algorithm:     entity.AlgorithmSlidingWindow,
		LimitCount:    100,
		WindowSeconds: 60,
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid rule: %v", err)
	}

	tb := base
	tb.Algorithm = entity.AlgorithmTokenBucket
	tb.LimitCount = 0
	tb.WindowSeconds = 0
	tb.BucketCapacity = 10
	tb.RefillRate = 2
	if err := tb.Validate(); err != nil {
		t.Fatalf("valid token bucket rule: %v", err)
	}
}
