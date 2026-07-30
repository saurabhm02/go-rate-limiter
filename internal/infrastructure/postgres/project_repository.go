package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
	domainerrors "github.com/saurabh/distributed-rate-limiter/internal/domain/errors"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/ports"
)

var _ ports.ProjectStore = (*ProjectRepository)(nil)

type ProjectRepository struct {
	pool *pgxpool.Pool
}

func NewProjectRepository(pool *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{pool: pool}
}

// CreateProject writes the tenant, its first key and its rules together.
// If any part fails the whole thing rolls back. A project with no key cannot be
// called, and one with no rules limits nothing, so half of it is never useful.
func (r *ProjectRepository) CreateProject(ctx context.Context, p ports.NewProject) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO tenants (id, name, status) VALUES ($1, $2, 'active')
	`, p.TenantID, p.Name); err != nil {
		return wrapUnique(err, "insert tenant")
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO api_keys (tenant_id, key_hash, key_prefix, status)
		VALUES ($1, $2, $3, 'active')
	`, p.TenantID, p.KeyHash, p.KeyPrefix); err != nil {
		return wrapUnique(err, "insert api key")
	}

	if len(p.Rules) > 0 {
		batch := &pgx.Batch{}
		for _, rule := range p.Rules {
			batch.Queue(`
				INSERT INTO rules (tenant_id, route_pattern, algorithm, limit_count,
				                   window_seconds, bucket_capacity, refill_rate, enabled)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			`,
				p.TenantID, rule.RoutePattern, string(rule.Algorithm),
				nullIfZero(rule.LimitCount), nullIfZero(rule.WindowSeconds),
				nullIfZero(rule.BucketCapacity), nullIfZeroFloat(rule.RefillRate),
				rule.Enabled,
			)
		}
		results := tx.SendBatch(ctx, batch)
		for range p.Rules {
			if _, err := results.Exec(); err != nil {
				_ = results.Close()
				return wrapUnique(err, "insert rule")
			}
		}
		if err := results.Close(); err != nil {
			return fmt.Errorf("close rule batch: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// ListProjects returns every project with its rules and key info for the
// dashboard. It runs three queries instead of one per project, so adding
// projects does not slow the page down.
func (r *ProjectRepository) ListProjects(ctx context.Context) ([]ports.ProjectSummary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, status, created_at
		FROM tenants
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	var out []ports.ProjectSummary
	index := map[uuid.UUID]int{}
	for rows.Next() {
		var p ports.ProjectSummary
		if err := rows.Scan(&p.ID, &p.Name, &p.Status, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan tenant: %w", err)
		}
		index[p.ID] = len(out)
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return out, nil
	}

	keyRows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, key_prefix, status, created_at
		FROM api_keys
		ORDER BY created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("list keys: %w", err)
	}
	defer keyRows.Close()
	for keyRows.Next() {
		var tenantID uuid.UUID
		var k ports.KeySummary
		if err := keyRows.Scan(&k.ID, &tenantID, &k.Prefix, &k.Status, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan key: %w", err)
		}
		if i, ok := index[tenantID]; ok {
			out[i].Keys = append(out[i].Keys, k)
		}
	}
	if err := keyRows.Err(); err != nil {
		return nil, err
	}

	ruleRows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, route_pattern, algorithm,
		       COALESCE(limit_count, 0), COALESCE(window_seconds, 0),
		       COALESCE(bucket_capacity, 0), COALESCE(refill_rate, 0), enabled
		FROM rules
		ORDER BY route_pattern
	`)
	if err != nil {
		return nil, fmt.Errorf("list rules: %w", err)
	}
	defer ruleRows.Close()
	for ruleRows.Next() {
		var rule entity.Rule
		var algorithm string
		if err := ruleRows.Scan(&rule.ID, &rule.TenantID, &rule.RoutePattern, &algorithm,
			&rule.LimitCount, &rule.WindowSeconds, &rule.BucketCapacity, &rule.RefillRate,
			&rule.Enabled); err != nil {
			return nil, fmt.Errorf("scan rule: %w", err)
		}
		rule.Algorithm = entity.Algorithm(algorithm)
		if i, ok := index[rule.TenantID]; ok {
			out[i].Rules = append(out[i].Rules, rule)
			out[i].RuleCount++
		}
	}
	return out, ruleRows.Err()
}

// AddAPIKey adds another key to a project that already exists. This is how you
// rotate a key without downtime: add the new one, ship it, then revoke the old.
func (r *ProjectRepository) AddAPIKey(ctx context.Context, tenantID uuid.UUID, keyHash, keyPrefix string) error {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO api_keys (tenant_id, key_hash, key_prefix, status)
		SELECT $1, $2, $3, 'active'
		WHERE EXISTS (SELECT 1 FROM tenants WHERE id = $1)
	`, tenantID, keyHash, keyPrefix)
	if err != nil {
		return wrapUnique(err, "insert api key")
	}
	if tag.RowsAffected() == 0 {
		return domainerrors.ErrTenantNotFound
	}
	return nil
}

// RevokeAPIKey stops a key working but keeps the row, so you can still see when
// it existed and when it was turned off.
func (r *ProjectRepository) RevokeAPIKey(ctx context.Context, tenantID, keyID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE api_keys SET status = 'revoked'
		WHERE id = $1 AND tenant_id = $2
	`, keyID, tenantID)
	if err != nil {
		return fmt.Errorf("revoke key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domainerrors.ErrAPIKeyNotFound
	}
	return nil
}

// wrapUnique spots a duplicate-key error from Postgres and turns it into our
// own error, so the HTTP layer can answer 409 instead of a generic 503.
func wrapUnique(err error, op string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("%s: %w", op, domainerrors.ErrProjectExists)
	}
	return fmt.Errorf("%s: %w", op, err)
}

// The rules table leaves the columns another algorithm does not use as NULL.
// Writing 0 would read like someone set a limit of zero.
func nullIfZero(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}

func nullIfZeroFloat(v float64) any {
	if v == 0 {
		return nil
	}
	return v
}
