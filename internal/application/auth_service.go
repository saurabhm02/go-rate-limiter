package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
	domainerrors "github.com/saurabh/distributed-rate-limiter/internal/domain/errors"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/ports"
	"github.com/saurabh/distributed-rate-limiter/pkg/httputil"
)

// TenantContext is the authenticated tenant identity for a request.
type TenantContext struct {
	Tenant entity.Tenant
	APIKey entity.APIKey
}

// AuthService validates API keys and resolves tenant identity.
type AuthService struct {
	keys    ports.APIKeyRepository
	tenants ports.TenantRepository
}

func NewAuthService(keys ports.APIKeyRepository, tenants ports.TenantRepository) *AuthService {
	return &AuthService{keys: keys, tenants: tenants}
}

// Authenticate verifies the raw API key and returns tenant context.
func (s *AuthService) Authenticate(ctx context.Context, rawAPIKey string) (TenantContext, error) {
	if rawAPIKey == "" {
		return TenantContext{}, domainerrors.ErrAPIKeyNotFound
	}

	key, err := s.keys.FindByHash(ctx, httputil.HashAPIKey(rawAPIKey))
	if err != nil {
		return TenantContext{}, err
	}
	if !key.IsActive() {
		return TenantContext{}, domainerrors.ErrAPIKeyRevoked
	}

	tenant, err := s.tenants.GetByID(ctx, key.TenantID)
	if err != nil {
		return TenantContext{}, err
	}
	if !tenant.IsActive() {
		return TenantContext{}, domainerrors.ErrTenantSuspended
	}

	return TenantContext{Tenant: *tenant, APIKey: *key}, nil
}

// MapAuthError converts domain auth errors to HTTP status codes.
func MapAuthError(err error) (int, string) {
	switch {
	case errors.Is(err, domainerrors.ErrAPIKeyNotFound), errors.Is(err, domainerrors.ErrAPIKeyRevoked):
		return 401, "unauthorized"
	case errors.Is(err, domainerrors.ErrTenantSuspended):
		return 403, "tenant_suspended"
	case errors.Is(err, domainerrors.ErrTenantNotFound):
		return 401, "unauthorized"
	default:
		return 503, "service_unavailable"
	}
}

// AuthenticateOrError wraps Authenticate with formatted infrastructure errors.
func (s *AuthService) AuthenticateOrError(ctx context.Context, rawAPIKey string) (TenantContext, error) {
	tc, err := s.Authenticate(ctx, rawAPIKey)
	if err != nil {
		return TenantContext{}, fmt.Errorf("authenticate: %w", err)
	}
	return tc, nil
}
