package redis

import (
	"context"
	"embed"
	"fmt"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
	domainerrors "github.com/saurabh/distributed-rate-limiter/internal/domain/errors"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/ports"
)

//go:embed scripts/*.lua
var luaScripts embed.FS

// RateLimiter implements ports.RateLimiter using Redis Lua scripts.
type RateLimiter struct {
	client          *goredis.Client
	tokenBucketSHA  string
	slidingWindowSHA string
}

var _ ports.RateLimiter = (*RateLimiter)(nil)

// NewRateLimiter loads Lua scripts and returns a Redis-backed rate limiter.
func NewRateLimiter(client *goredis.Client) (*RateLimiter, error) {
	tb, err := luaScripts.ReadFile("scripts/token_bucket.lua")
	if err != nil {
		return nil, fmt.Errorf("read token bucket script: %w", err)
	}
	sw, err := luaScripts.ReadFile("scripts/sliding_window.lua")
	if err != nil {
		return nil, fmt.Errorf("read sliding window script: %w", err)
	}

	ctx := context.Background()
	tbSHA, err := client.ScriptLoad(ctx, string(tb)).Result()
	if err != nil {
		return nil, fmt.Errorf("load token bucket script: %w", err)
	}
	swSHA, err := client.ScriptLoad(ctx, string(sw)).Result()
	if err != nil {
		return nil, fmt.Errorf("load sliding window script: %w", err)
	}

	return &RateLimiter{
		client:           client,
		tokenBucketSHA:   tbSHA,
		slidingWindowSHA: swSHA,
	}, nil
}

// Check evaluates the rule for the given key and cost.
func (r *RateLimiter) Check(ctx context.Context, key entity.RateLimitKey, rule entity.Rule, cost int64) (entity.RateLimitDecision, error) {
	if cost <= 0 {
		cost = 1
	}

	trk := TenantRouteKey{TenantID: key.TenantID, Route: key.Route}

	switch rule.Algorithm {
	case entity.AlgorithmTokenBucket:
		return r.checkTokenBucket(ctx, trk, rule, cost)
	case entity.AlgorithmSlidingWindow:
		return r.checkSlidingWindow(ctx, trk, rule, cost)
	default:
		return entity.RateLimitDecision{}, fmt.Errorf("%w: unsupported algorithm %s", domainerrors.ErrInvalidRule, rule.Algorithm)
	}
}

func (r *RateLimiter) checkTokenBucket(ctx context.Context, key TenantRouteKey, rule entity.Rule, cost int64) (entity.RateLimitDecision, error) {
	nowMs := time.Now().UnixMilli()
	redisKey := TokenBucketKey(key)

	result, err := r.client.EvalSha(ctx, r.tokenBucketSHA, []string{redisKey},
		rule.BucketCapacity,
		rule.RefillRate,
		nowMs,
		cost,
	).Int64Slice()
	if err != nil {
		return entity.RateLimitDecision{}, fmt.Errorf("%w: %v", domainerrors.ErrRateLimitBackend, err)
	}
	return parseDecision(result, rule)
}

func (r *RateLimiter) checkSlidingWindow(ctx context.Context, key TenantRouteKey, rule entity.Rule, cost int64) (entity.RateLimitDecision, error) {
	nowSec := time.Now().Unix()
	curr, prev, idx := SlidingWindowKeys(key)

	result, err := r.client.EvalSha(ctx, r.slidingWindowSHA, []string{curr, prev, idx},
		rule.LimitCount,
		rule.WindowSeconds,
		nowSec,
		cost,
	).Int64Slice()
	if err != nil {
		return entity.RateLimitDecision{}, fmt.Errorf("%w: %v", domainerrors.ErrRateLimitBackend, err)
	}
	return parseDecision(result, rule)
}

func parseDecision(result []int64, rule entity.Rule) (entity.RateLimitDecision, error) {
	if len(result) < 4 {
		return entity.RateLimitDecision{}, fmt.Errorf("%w: unexpected lua response %v", domainerrors.ErrRateLimitBackend, result)
	}

	allowed := result[0] == 1
	remaining := result[1]
	if remaining < 0 {
		remaining = 0
	}

	return entity.RateLimitDecision{
		Allowed:    allowed,
		Limit:      rule.Limit(),
		Remaining:  remaining,
		ResetAt:    result[2],
		RetryAfter: result[3],
		Algorithm:  rule.Algorithm,
	}, nil
}

// ParseLuaInt64Slice helps tests decode script output.
func ParseLuaInt64Slice(values []interface{}) ([]int64, error) {
	out := make([]int64, len(values))
	for i, v := range values {
		switch n := v.(type) {
		case int64:
			out[i] = n
		case string:
			parsed, err := strconv.ParseInt(n, 10, 64)
			if err != nil {
				return nil, err
			}
			out[i] = parsed
		default:
			return nil, fmt.Errorf("unexpected type %T", v)
		}
	}
	return out, nil
}
