//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/saurabh/distributed-rate-limiter/internal/application"
	redisinfra "github.com/saurabh/distributed-rate-limiter/internal/infrastructure/redis"
	pgstore "github.com/saurabh/distributed-rate-limiter/internal/infrastructure/postgres"
	"github.com/saurabh/distributed-rate-limiter/internal/infrastructure/cache"
)

func TestCheckServiceWithSeedRules(t *testing.T) {
	ctx := context.Background()
	pool := setupPostgres(t)
	redisClient, _ := setupMiniredis(t)

	limiter, err := redisinfra.NewRateLimiter(redisClient)
	if err != nil {
		t.Fatal(err)
	}

	ruleRepo := cache.NewRuleCache(pgstore.NewRuleRepository(pool), cacheTTL)
	checkSvc := application.NewCheckService(ruleRepo, application.NewRuleResolver(), limiter)

	tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	for i := 0; i < 10; i++ {
		dec, err := checkSvc.Check(ctx, tenantID, "/api/payments/1", 1)
		if err != nil {
			t.Fatalf("check %d: %v", i, err)
		}
		if !dec.Allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	dec, err := checkSvc.Check(ctx, tenantID, "/api/payments/1", 1)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Allowed {
		t.Fatal("11th payment request should be denied")
	}

	dec, err = checkSvc.Check(ctx, tenantID, "/unknown-route", 1)
	if err != nil || !dec.Allowed {
		t.Fatalf("default rule should allow /unknown-route dec=%+v err=%v", dec, err)
	}
}

func TestAuthServiceWithSeedData(t *testing.T) {
	ctx := context.Background()
	pool := setupPostgres(t)

	authSvc := application.NewAuthService(
		pgstore.NewAPIKeyRepository(pool),
		pgstore.NewTenantRepository(pool),
	)

	tc, err := authSvc.Authenticate(ctx, "rl_demo_abc123xyz")
	if err != nil {
		t.Fatal(err)
	}
	if tc.Tenant.Name != "demo-corp" {
		t.Fatalf("tenant=%s", tc.Tenant.Name)
	}
}
