package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
)

// TenantRepository loads tenant records.
type TenantRepository interface {
	GetByID(ctx context.Context, tenantID uuid.UUID) (*entity.Tenant, error)
}
