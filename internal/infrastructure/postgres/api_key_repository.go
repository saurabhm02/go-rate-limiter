package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
	domainerrors "github.com/saurabh/distributed-rate-limiter/internal/domain/errors"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/ports"
)

var _ ports.TenantRepository = (*TenantRepository)(nil)

var _ ports.APIKeyRepository = (*APIKeyRepository)(nil)

type APIKeyRepository struct {
	pool *pgxpool.Pool
}

func NewAPIKeyRepository(pool *pgxpool.Pool) *APIKeyRepository {
	return &APIKeyRepository{pool: pool}
}

func (r *APIKeyRepository) FindByHash(ctx context.Context, keyHash string) (*entity.APIKey, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, key_prefix, status
		FROM api_keys
		WHERE key_hash = $1
	`, keyHash)

	var key entity.APIKey
	var status string
	if err := row.Scan(&key.ID, &key.TenantID, &key.KeyPrefix, &status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("find api key: %w", err)
	}
	key.Status = entity.APIKeyStatus(status)
	return &key, nil
}

type TenantRepository struct {
	pool *pgxpool.Pool
}

func NewTenantRepository(pool *pgxpool.Pool) *TenantRepository {
	return &TenantRepository{pool: pool}
}

func (r *TenantRepository) List(ctx context.Context) ([]entity.Tenant, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, status
		FROM tenants
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []entity.Tenant
	for rows.Next() {
		var t entity.Tenant
		var status string
		if err := rows.Scan(&t.ID, &t.Name, &status); err != nil {
			return nil, fmt.Errorf("scan tenant: %w", err)
		}
		t.Status = entity.TenantStatus(status)
		tenants = append(tenants, t)
	}
	return tenants, rows.Err()
}

func (r *TenantRepository) GetByID(ctx context.Context, tenantID uuid.UUID) (*entity.Tenant, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, name, status
		FROM tenants
		WHERE id = $1
	`, tenantID)

	var t entity.Tenant
	var status string
	if err := row.Scan(&t.ID, &t.Name, &status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.ErrTenantNotFound
		}
		return nil, fmt.Errorf("get tenant: %w", err)
	}
	t.Status = entity.TenantStatus(status)
	return &t, nil
}
