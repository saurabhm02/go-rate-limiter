package cache

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/ports"
)

// RuleCache wraps a RuleRepository with a per-tenant TTL cache.
type RuleCache struct {
	repo ports.RuleRepository
	ttl  time.Duration

	mu    sync.RWMutex
	items map[uuid.UUID]cachedRules
}

type cachedRules struct {
	rules     []entity.Rule
	expiresAt time.Time
}

func NewRuleCache(repo ports.RuleRepository, ttl time.Duration) *RuleCache {
	return &RuleCache{
		repo:  repo,
		ttl:   ttl,
		items: make(map[uuid.UUID]cachedRules),
	}
}

var _ ports.RuleRepository = (*RuleCache)(nil)

func (c *RuleCache) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]entity.Rule, error) {
	if rules, ok := c.get(tenantID); ok {
		return rules, nil
	}

	rules, err := c.repo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	c.set(tenantID, rules)
	return rules, nil
}

func (c *RuleCache) ListAll(ctx context.Context) ([]entity.Rule, error) {
	return c.repo.ListAll(ctx)
}

func (c *RuleCache) get(tenantID uuid.UUID) ([]entity.Rule, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, ok := c.items[tenantID]
	if !ok || time.Now().After(item.expiresAt) {
		return nil, false
	}
	return item.rules, true
}

func (c *RuleCache) set(tenantID uuid.UUID, rules []entity.Rule) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[tenantID] = cachedRules{
		rules:     rules,
		expiresAt: time.Now().Add(c.ttl),
	}
}
