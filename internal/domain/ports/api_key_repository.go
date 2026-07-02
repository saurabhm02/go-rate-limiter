package ports

import (
	"context"

	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
)

// APIKeyRepository looks up API key metadata by hash.
// Implemented by the Postgres adapter in M3.
type APIKeyRepository interface {
	FindByHash(ctx context.Context, keyHash string) (*entity.APIKey, error)
}
