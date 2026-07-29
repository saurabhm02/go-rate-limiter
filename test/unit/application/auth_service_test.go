package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/saurabh/distributed-rate-limiter/internal/application"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
	domainerrors "github.com/saurabh/distributed-rate-limiter/internal/domain/errors"
)

type stubAPIKeyRepo struct {
	key *entity.APIKey
	err error
}

func (s stubAPIKeyRepo) FindByHash(ctx context.Context, hash string) (*entity.APIKey, error) {
	return s.key, s.err
}

type stubTenantRepo struct {
	tenant *entity.Tenant
	err    error
}

func (s stubTenantRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Tenant, error) {
	return s.tenant, s.err
}

func TestAuthServiceAuthenticateSuccess(t *testing.T) {
	tenantID := uuid.New()
	svc := application.NewAuthService(
		stubAPIKeyRepo{key: &entity.APIKey{TenantID: tenantID, Status: entity.APIKeyStatusActive}},
		stubTenantRepo{tenant: &entity.Tenant{ID: tenantID, Status: entity.TenantStatusActive}},
	)

	tc, err := svc.Authenticate(context.Background(), "any-key")
	if err != nil {
		t.Fatal(err)
	}
	if tc.Tenant.ID != tenantID {
		t.Fatalf("tenant mismatch")
	}
}

func TestAuthServiceRevokedKey(t *testing.T) {
	svc := application.NewAuthService(
		stubAPIKeyRepo{key: &entity.APIKey{Status: entity.APIKeyStatusRevoked}},
		stubTenantRepo{},
	)
	_, err := svc.Authenticate(context.Background(), "key")
	if !errors.Is(err, domainerrors.ErrAPIKeyRevoked) {
		t.Fatalf("got %v", err)
	}
}

func TestAuthServiceSuspendedTenant(t *testing.T) {
	tenantID := uuid.New()
	svc := application.NewAuthService(
		stubAPIKeyRepo{key: &entity.APIKey{TenantID: tenantID, Status: entity.APIKeyStatusActive}},
		stubTenantRepo{tenant: &entity.Tenant{ID: tenantID, Status: entity.TenantStatusSuspended}},
	)
	_, err := svc.Authenticate(context.Background(), "key")
	if !errors.Is(err, domainerrors.ErrTenantSuspended) {
		t.Fatalf("got %v", err)
	}
}

type stubRuleRepo struct {
	rules []entity.Rule
}

func (s stubRuleRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]entity.Rule, error) {
	return s.rules, nil
}

func (s stubRuleRepo) ListAll(ctx context.Context) ([]entity.Rule, error) {
	return s.rules, nil
}

type stubLimiter struct {
	decision entity.RateLimitDecision
	err      error
	called   bool
}

func (s *stubLimiter) Check(ctx context.Context, key entity.RateLimitKey, rule entity.Rule, cost int64) (entity.RateLimitDecision, error) {
	s.called = true
	return s.decision, s.err
}

func TestCheckServiceNoRuleAllows(t *testing.T) {
	limiter := &stubLimiter{}
	svc := application.NewCheckService(
		stubRuleRepo{rules: []entity.Rule{}},
		application.NewRuleResolver(),
		limiter,
	)

	dec, err := svc.Check(context.Background(), uuid.New(), "/unknown", "", 1)
	if err != nil || !dec.Allowed {
		t.Fatalf("dec=%+v err=%v", dec, err)
	}
	if limiter.called {
		t.Fatal("limiter should not be called when no rule matches")
	}
}

func TestCheckServiceUsesLimiterWhenRuleMatches(t *testing.T) {
	tenantID := uuid.New()
	limiter := &stubLimiter{decision: entity.RateLimitDecision{Allowed: true, Limit: 10, Remaining: 9}}
	svc := application.NewCheckService(
		stubRuleRepo{rules: []entity.Rule{{
			TenantID:      tenantID,
			RoutePattern:  "/api/payments*",
			Algorithm:     entity.AlgorithmSlidingWindow,
			Enabled:       true,
			LimitCount:    10,
			WindowSeconds: 60,
		}}},
		application.NewRuleResolver(),
		limiter,
	)

	dec, err := svc.Check(context.Background(), tenantID, "/api/payments/1", "", 1)
	if err != nil || !dec.Allowed || dec.Remaining != 9 {
		t.Fatalf("dec=%+v err=%v", dec, err)
	}
	if !limiter.called {
		t.Fatal("expected limiter call")
	}
}
