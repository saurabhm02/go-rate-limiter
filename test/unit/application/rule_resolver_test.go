package application_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/saurabh/distributed-rate-limiter/internal/application"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
)

func slidingRule(pattern string, enabled bool) entity.Rule {
	return entity.Rule{
		ID:            uuid.New(),
		TenantID:      uuid.New(),
		RoutePattern:  pattern,
		Algorithm:     entity.AlgorithmSlidingWindow,
		Enabled:       enabled,
		LimitCount:    100,
		WindowSeconds: 60,
	}
}

func TestRuleResolverExactBeatsPrefix(t *testing.T) {
	resolver := application.NewRuleResolver()
	rules := []entity.Rule{
		slidingRule("/api/*", true),
		slidingRule("/api/payments*", true),
		slidingRule("/api/payments/123", true),
	}

	got := resolver.Resolve(rules, "/api/payments/123")
	if got == nil {
		t.Fatal("expected a matching rule")
	}
	if got.RoutePattern != "/api/payments/123" {
		t.Fatalf("got %q, want exact /api/payments/123", got.RoutePattern)
	}
}

func TestRuleResolverLongestPrefixWins(t *testing.T) {
	resolver := application.NewRuleResolver()
	rules := []entity.Rule{
		slidingRule("/api/*", true),
		slidingRule("/api/payments*", true),
	}

	got := resolver.Resolve(rules, "/api/payments/42")
	if got == nil || got.RoutePattern != "/api/payments*" {
		t.Fatalf("got %#v, want /api/payments*", got)
	}
}

func TestRuleResolverDefaultFallback(t *testing.T) {
	resolver := application.NewRuleResolver()
	rules := []entity.Rule{
		slidingRule("*", true),
	}

	got := resolver.Resolve(rules, "/unknown/path")
	if got == nil || got.RoutePattern != "*" {
		t.Fatalf("got %#v, want default *", got)
	}
}

func TestRuleResolverNoMatchReturnsNil(t *testing.T) {
	resolver := application.NewRuleResolver()
	rules := []entity.Rule{
		slidingRule("/v1/check", true),
	}

	got := resolver.Resolve(rules, "/other")
	if got != nil {
		t.Fatalf("got %#v, want nil (no rule -> allow)", got)
	}
}

func TestRuleResolverSkipsDisabled(t *testing.T) {
	resolver := application.NewRuleResolver()
	rules := []entity.Rule{
		slidingRule("/v1/check", false),
		slidingRule("*", true),
	}

	got := resolver.Resolve(rules, "/v1/check")
	if got == nil || got.RoutePattern != "*" {
		t.Fatalf("got %#v, want enabled default *", got)
	}
}

func TestRuleResolverV1CheckScenario(t *testing.T) {
	resolver := application.NewRuleResolver()
	rules := []entity.Rule{
		slidingRule("POST /v1/check", true),
		slidingRule("/api/payments*", true),
		slidingRule("*", true),
	}

	got := resolver.Resolve(rules, "/api/payments/1")
	if got.RoutePattern != "/api/payments*" {
		t.Fatalf("got %q", got.RoutePattern)
	}

	got = resolver.Resolve(rules, "POST /v1/check")
	if got.RoutePattern != "POST /v1/check" {
		t.Fatalf("got %q", got.RoutePattern)
	}
}
