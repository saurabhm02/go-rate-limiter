package redis

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const keyPrefix = "rl"

// RouteHash returns a bounded hash segment for a normalized route.
func RouteHash(route string) string {
	normalized := strings.ToLower(strings.TrimSpace(route))
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:8]) // 16 hex chars
}

// TokenBucketKey builds the Redis key for token bucket state.
// Format: rl:tb:{tenant_id}:{route_hash}
func TokenBucketKey(key TenantRouteKey) string {
	return fmt.Sprintf("%s:tb:%s:%s", keyPrefix, key.TenantID, RouteHash(key.Route))
}

// SlidingWindowKeys builds Redis keys for sliding window counters.
// Format: rl:sw:{tenant_id}:{route_hash}:{curr|prev|idx}
func SlidingWindowKeys(key TenantRouteKey) (curr, prev, idx string) {
	base := fmt.Sprintf("%s:sw:%s:%s", keyPrefix, key.TenantID, RouteHash(key.Route))
	return base + ":curr", base + ":prev", base + ":idx"
}

// TenantRouteKey is the infrastructure view of a rate limit counter identity.
type TenantRouteKey struct {
	TenantID uuid.UUID
	Route    string
}
