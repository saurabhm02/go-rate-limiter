package ports

import (
	"context"

	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
)

// RateLimiter evaluates whether a request is allowed under a rule.
// Implemented by the Redis adapter in M2.
type RateLimiter interface {
	Check(ctx context.Context, key entity.RateLimitKey, rule entity.Rule, cost int64) (entity.RateLimitDecision, error)
}
