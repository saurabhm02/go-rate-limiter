package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/ports"
)

var _ ports.RuleRepository = (*RuleRepository)(nil)

type RuleRepository struct {
	pool *pgxpool.Pool
}

func NewRuleRepository(pool *pgxpool.Pool) *RuleRepository {
	return &RuleRepository{pool: pool}
}

func (r *RuleRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]entity.Rule, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, route_pattern, algorithm,
		       COALESCE(limit_count, 0), COALESCE(window_seconds, 0),
		       COALESCE(bucket_capacity, 0), COALESCE(refill_rate, 0),
		       enabled
		FROM rules
		WHERE tenant_id = $1
		ORDER BY route_pattern
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list rules by tenant: %w", err)
	}
	defer rows.Close()

	var rules []entity.Rule
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (r *RuleRepository) ListAll(ctx context.Context) ([]entity.Rule, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, route_pattern, algorithm,
		       COALESCE(limit_count, 0), COALESCE(window_seconds, 0),
		       COALESCE(bucket_capacity, 0), COALESCE(refill_rate, 0),
		       enabled
		FROM rules
		ORDER BY tenant_id, route_pattern
	`)
	if err != nil {
		return nil, fmt.Errorf("list all rules: %w", err)
	}
	defer rows.Close()

	var rules []entity.Rule
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

type ruleRow interface {
	Scan(dest ...any) error
}

func scanRule(row ruleRow) (entity.Rule, error) {
	var rule entity.Rule
	var algorithm string
	var refillRate float64

	err := row.Scan(
		&rule.ID,
		&rule.TenantID,
		&rule.RoutePattern,
		&algorithm,
		&rule.LimitCount,
		&rule.WindowSeconds,
		&rule.BucketCapacity,
		&refillRate,
		&rule.Enabled,
	)
	if err != nil {
		return entity.Rule{}, fmt.Errorf("scan rule: %w", err)
	}

	parsed, err := entity.ParseAlgorithm(algorithm)
	if err != nil {
		return entity.Rule{}, err
	}
	rule.Algorithm = parsed
	rule.RefillRate = refillRate
	return rule, nil
}
