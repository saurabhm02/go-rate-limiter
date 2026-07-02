package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
)

// RuleRepository loads rate limit rules for tenants.
// Implemented by the Postgres adapter in M3.
type RuleRepository interface {
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]entity.Rule, error)
	ListAll(ctx context.Context) ([]entity.Rule, error)
}
