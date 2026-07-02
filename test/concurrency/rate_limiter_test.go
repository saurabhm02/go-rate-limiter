package concurrency_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/google/uuid"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
	redisinfra "github.com/saurabh/distributed-rate-limiter/internal/infrastructure/redis"
)

// Concurrent requests must not exceed limit by more than 1 (documented tolerance).
func TestConcurrentSlidingWindowNeverExceedsLimitMuch(t *testing.T) {
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
		mr.Close()
	})

	rl, err := redisinfra.NewRateLimiter(client)
	if err != nil {
		t.Fatal(err)
	}

	const limit = 100
	rule := entity.Rule{
		Algorithm:     entity.AlgorithmSlidingWindow,
		LimitCount:    limit,
		WindowSeconds: 60,
	}
	key := entity.RateLimitKey{TenantID: uuid.New(), Route: "/v1/check"}

	var allowed int64
	var wg sync.WaitGroup
	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dec, err := rl.Check(context.Background(), key, rule, 1)
			if err != nil {
				t.Errorf("check: %v", err)
				return
			}
			if dec.Allowed {
				atomic.AddInt64(&allowed, 1)
			}
		}()
	}
	wg.Wait()

	if allowed > limit+1 {
		t.Fatalf("allowed %d requests, limit %d (tolerance +1)", allowed, limit)
	}
	if allowed < limit-1 {
		t.Fatalf("allowed only %d, expected close to %d", allowed, limit)
	}
}

func TestConcurrentTokenBucketNeverExceedsCapacityMuch(t *testing.T) {
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
		mr.Close()
	})

	rl, err := redisinfra.NewRateLimiter(client)
	if err != nil {
		t.Fatal(err)
	}

	const capacity = 50
	rule := entity.Rule{
		Algorithm:      entity.AlgorithmTokenBucket,
		BucketCapacity: capacity,
		RefillRate:     0.01, // negligible refill during test
	}
	key := entity.RateLimitKey{TenantID: uuid.New(), Route: "/api/orders"}

	var allowed int64
	var wg sync.WaitGroup
	for i := 0; i < 300; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dec, err := rl.Check(context.Background(), key, rule, 1)
			if err != nil {
				t.Errorf("check: %v", err)
				return
			}
			if dec.Allowed {
				atomic.AddInt64(&allowed, 1)
			}
		}()
	}
	wg.Wait()

	if allowed > capacity+1 {
		t.Fatalf("allowed %d, capacity %d", allowed, capacity)
	}
}
