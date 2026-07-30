package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
)

type NewProject struct {
	TenantID  uuid.UUID
	Name      string
	KeyHash   string
	KeyPrefix string
	Rules     []entity.Rule
}

type KeySummary struct {
	ID        uuid.UUID
	Prefix    string
	Status    string
	CreatedAt time.Time
}

type ProjectSummary struct {
	ID        uuid.UUID
	Name      string
	Status    string
	CreatedAt time.Time
	RuleCount int
	Rules     []entity.Rule
	Keys      []KeySummary
}

type ProjectStore interface {
	CreateProject(ctx context.Context, p NewProject) error
	ListProjects(ctx context.Context) ([]ProjectSummary, error)
	AddAPIKey(ctx context.Context, tenantID uuid.UUID, keyHash, keyPrefix string) error
	RevokeAPIKey(ctx context.Context, tenantID, keyID uuid.UUID) error
}
