package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/google/uuid"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
	redisinfra "github.com/saurabh/distributed-rate-limiter/internal/infrastructure/redis"
)

func setupRedis(t *testing.T) (*miniredis.Miniredis, *goredis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
		mr.Close()
	})
	return mr, client
}

func testKey() entity.RateLimitKey {
	return entity.RateLimitKey{TenantID: uuid.New(), Route: "/api/payments"}
}

func TestTokenBucketAllowsBurstThenDenies(t *testing.T) {
	_, client := setupRedis(t)
	rl, err := redisinfra.NewRateLimiter(client)
	if err != nil {
		t.Fatalf("NewRateLimiter: %v", err)
	}

	rule := entity.Rule{
		TenantID:       uuid.New(),
		RoutePattern:   "/api/payments*",
		Algorithm:      entity.AlgorithmTokenBucket,
		Enabled:        true,
		BucketCapacity: 3,
		RefillRate:     1,
	}
	key := testKey()

	for i := 0; i < 3; i++ {
		dec, err := rl.Check(context.Background(), key, rule, 1)
		if err != nil {
			t.Fatalf("check %d: %v", i, err)
		}
		if !dec.Allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	dec, err := rl.Check(context.Background(), key, rule, 1)
	if err != nil {
		t.Fatalf("check deny: %v", err)
	}
	if dec.Allowed {
		t.Fatal("4th request should be denied")
	}
	if dec.RetryAfter <= 0 {
		t.Fatalf("expected retry_after, got %d", dec.RetryAfter)
	}
}

func TestTokenBucketRefillsOverTime(t *testing.T) {
	_, client := setupRedis(t)
	rl, err := redisinfra.NewRateLimiter(client)
	if err != nil {
		t.Fatalf("NewRateLimiter: %v", err)
	}

	rule := entity.Rule{
		Algorithm:      entity.AlgorithmTokenBucket,
		BucketCapacity: 1,
		RefillRate:     10,
	}
	key := testKey()

	if _, err := rl.Check(context.Background(), key, rule, 1); err != nil {
		t.Fatal(err)
	}
	dec, err := rl.Check(context.Background(), key, rule, 1)
	if err != nil || dec.Allowed {
		t.Fatalf("expected immediate deny, got allowed=%v err=%v", dec.Allowed, err)
	}

	time.Sleep(150 * time.Millisecond)

	dec, err = rl.Check(context.Background(), key, rule, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !dec.Allowed {
		t.Fatal("expected refill to allow request")
	}
}

func TestSlidingWindowEnforcesLimit(t *testing.T) {
	_, client := setupRedis(t)
	rl, err := redisinfra.NewRateLimiter(client)
	if err != nil {
		t.Fatalf("NewRateLimiter: %v", err)
	}

	rule := entity.Rule{
		Algorithm:     entity.AlgorithmSlidingWindow,
		LimitCount:    5,
		WindowSeconds: 60,
	}
	key := testKey()

	for i := 0; i < 5; i++ {
		dec, err := rl.Check(context.Background(), key, rule, 1)
		if err != nil {
			t.Fatalf("check %d: %v", i, err)
		}
		if !dec.Allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	dec, err := rl.Check(context.Background(), key, rule, 1)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Allowed {
		t.Fatal("6th request should be denied")
	}
	if dec.Remaining != 0 {
		t.Fatalf("remaining = %d, want 0", dec.Remaining)
	}
}

func TestSlidingWindowDeniesWhenCostExceedsRemaining(t *testing.T) {
	_, client := setupRedis(t)
	rl, err := redisinfra.NewRateLimiter(client)
	if err != nil {
		t.Fatal(err)
	}

	rule := entity.Rule{
		Algorithm:     entity.AlgorithmSlidingWindow,
		LimitCount:    5,
		WindowSeconds: 60,
	}
	key := testKey()

	for i := 0; i < 4; i++ {
		if _, err := rl.Check(context.Background(), key, rule, 1); err != nil {
			t.Fatal(err)
		}
	}

	dec, err := rl.Check(context.Background(), key, rule, 2)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Allowed {
		t.Fatal("cost 2 should be denied when only 1 slot remains")
	}
}

func TestRouteHashStable(t *testing.T) {
	h1 := redisinfra.RouteHash("/api/payments")
	h2 := redisinfra.RouteHash("/api/payments")
	if h1 != h2 || len(h1) != 16 {
		t.Fatalf("hash = %q", h1)
	}
}
