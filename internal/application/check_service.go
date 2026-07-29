package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
	domainerrors "github.com/saurabh/distributed-rate-limiter/internal/domain/errors"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/ports"
)

// CheckService orchestrates rule resolution and rate limit evaluation.
type CheckService struct {
	rules    ports.RuleRepository
	resolver *RuleResolver
	limiter  ports.RateLimiter
}

func NewCheckService(rules ports.RuleRepository, resolver *RuleResolver, limiter ports.RateLimiter) *CheckService {
	return &CheckService{
		rules:    rules,
		resolver: resolver,
		limiter:  limiter,
	}
}

// Check evaluates whether a tenant request is allowed for the given route.
func (s *CheckService) Check(ctx context.Context, tenantID uuid.UUID, route, subject string, cost int64) (entity.RateLimitDecision, error) {
	if cost <= 0 {
		cost = 1
	}

	rules, err := s.rules.ListByTenant(ctx, tenantID)
	if err != nil {
		return entity.RateLimitDecision{}, fmt.Errorf("load rules: %w", err)
	}

	rule := s.resolver.Resolve(rules, route)
	if rule == nil {
		return entity.RateLimitDecision{Allowed: true}, nil
	}

	key := entity.RateLimitKey{TenantID: tenantID, Route: route, Subject: subject}
	decision, err := s.limiter.Check(ctx, key, *rule, cost)
	if err != nil {
		if errors.Is(err, domainerrors.ErrRateLimitBackend) {
			return entity.RateLimitDecision{}, err
		}
		return entity.RateLimitDecision{}, fmt.Errorf("%w: %v", domainerrors.ErrRateLimitBackend, err)
	}
	return decision, nil
}
