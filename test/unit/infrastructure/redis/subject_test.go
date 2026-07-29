package redis_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
	redisinfra "github.com/saurabh/distributed-rate-limiter/internal/infrastructure/redis"
)

func slidingRule(limit int64) entity.Rule {
	return entity.Rule{
		TenantID:      uuid.New(),
		RoutePattern:  "/admitdesk/intake/*",
		Algorithm:     entity.AlgorithmSlidingWindow,
		Enabled:       true,
		LimitCount:    limit,
		WindowSeconds: 600,
	}
}

// drain consumes n allowed requests and fails if any is denied.
func drain(t *testing.T, rl *redisinfra.RateLimiter, key entity.RateLimitKey, rule entity.Rule, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		dec, err := rl.Check(context.Background(), key, rule, 1)
		if err != nil {
			t.Fatalf("subject %q check %d: %v", key.Subject, i, err)
		}
		if !dec.Allowed {
			t.Fatalf("subject %q request %d should be allowed", key.Subject, i+1)
		}
	}
}

func TestSubjectsGetIndependentBuckets(t *testing.T) {
	_, client := setupRedis(t)
	rl, err := redisinfra.NewRateLimiter(client)
	if err != nil {
		t.Fatalf("NewRateLimiter: %v", err)
	}

	tenantID := uuid.New()
	rule := slidingRule(5)
	route := "/admitdesk/intake/apply"

	alice := entity.RateLimitKey{TenantID: tenantID, Route: route, Subject: "203.0.113.7"}
	bob := entity.RateLimitKey{TenantID: tenantID, Route: route, Subject: "198.51.100.9"}

	// Alice exhausts her budget on the shared route.
	drain(t, rl, alice, rule, 5)
	dec, err := rl.Check(context.Background(), alice, rule, 1)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Allowed {
		t.Fatal("alice's 6th request should be denied")
	}

	// Bob, same tenant and same route, is untouched.
	drain(t, rl, bob, rule, 5)
	dec, err = rl.Check(context.Background(), bob, rule, 1)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Allowed {
		t.Fatal("bob's 6th request should be denied")
	}
}

func TestSubjectAlsoSplitsTokenBuckets(t *testing.T) {
	_, client := setupRedis(t)
	rl, err := redisinfra.NewRateLimiter(client)
	if err != nil {
		t.Fatalf("NewRateLimiter: %v", err)
	}

	tenantID := uuid.New()
	rule := entity.Rule{
		TenantID:       tenantID,
		RoutePattern:   "/api/payments*",
		Algorithm:      entity.AlgorithmTokenBucket,
		Enabled:        true,
		BucketCapacity: 2,
		RefillRate:     1,
	}
	route := "/api/payments/1"

	a := entity.RateLimitKey{TenantID: tenantID, Route: route, Subject: "acct-a"}
	b := entity.RateLimitKey{TenantID: tenantID, Route: route, Subject: "acct-b"}

	drain(t, rl, a, rule, 2)
	if dec, _ := rl.Check(context.Background(), a, rule, 1); dec.Allowed {
		t.Fatal("acct-a should be out of tokens")
	}
	if dec, err := rl.Check(context.Background(), b, rule, 1); err != nil || !dec.Allowed {
		t.Fatalf("acct-b should still have tokens: dec=%+v err=%v", dec, err)
	}
}

func TestEmptySubjectKeepsLegacyKeyFormat(t *testing.T) {
	tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	route := "/api/payments/1"
	base := redisinfra.TenantRouteKey{TenantID: tenantID, Route: route}

	wantTB := "rl:tb:" + tenantID.String() + ":" + redisinfra.RouteHash(route)
	if got := redisinfra.TokenBucketKey(base); got != wantTB {
		t.Fatalf("token bucket key = %q, want %q", got, wantTB)
	}

	curr, prev, idx := redisinfra.SlidingWindowKeys(base)
	wantSW := "rl:sw:" + tenantID.String() + ":" + redisinfra.RouteHash(route)
	if curr != wantSW+":curr" || prev != wantSW+":prev" || idx != wantSW+":idx" {
		t.Fatalf("sliding window keys = %q %q %q, want %s:{curr,prev,idx}", curr, prev, idx, wantSW)
	}

	// Whitespace-only subject is treated as absent.
	if got := redisinfra.TokenBucketKey(redisinfra.TenantRouteKey{TenantID: tenantID, Route: route, Subject: "  "}); got != wantTB {
		t.Fatalf("blank subject changed the key: %q", got)
	}

	// A real subject appends exactly one segment.
	withSubject := redisinfra.TenantRouteKey{TenantID: tenantID, Route: route, Subject: "203.0.113.7"}
	if got := redisinfra.TokenBucketKey(withSubject); got != wantTB+":"+redisinfra.SubjectHash("203.0.113.7") {
		t.Fatalf("subject key = %q", got)
	}
}

// Subjects are case-sensitive; routes are not.
func TestSubjectHashIsCaseSensitive(t *testing.T) {
	if redisinfra.SubjectHash("UserA") == redisinfra.SubjectHash("usera") {
		t.Fatal("subject hash must not fold case")
	}
	if redisinfra.RouteHash("/API/Payments") != redisinfra.RouteHash("/api/payments") {
		t.Fatal("route hash should stay case-insensitive")
	}
}
