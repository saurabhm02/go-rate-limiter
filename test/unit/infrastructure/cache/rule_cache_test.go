package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
	"github.com/saurabh/distributed-rate-limiter/internal/infrastructure/cache"
)

type stubRuleRepo struct {
	calls int
}

func (s *stubRuleRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]entity.Rule, error) {
	s.calls++
	return []entity.Rule{{RoutePattern: "*"}}, nil
}

func (s *stubRuleRepo) ListAll(ctx context.Context) ([]entity.Rule, error) {
	return nil, nil
}

func TestRuleCacheTTL(t *testing.T) {
	stub := &stubRuleRepo{}
	c := cache.NewRuleCache(stub, 50*time.Millisecond)
	tenantID := uuid.New()

	if _, err := c.ListByTenant(context.Background(), tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ListByTenant(context.Background(), tenantID); err != nil {
		t.Fatal(err)
	}
	if stub.calls != 1 {
		t.Fatalf("calls = %d, want 1 cache hit", stub.calls)
	}

	time.Sleep(60 * time.Millisecond)
	if _, err := c.ListByTenant(context.Background(), tenantID); err != nil {
		t.Fatal(err)
	}
	if stub.calls != 2 {
		t.Fatalf("calls = %d, want refresh after TTL", stub.calls)
	}
}
