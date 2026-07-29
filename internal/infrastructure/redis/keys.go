package redis

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const keyPrefix = "rl"

func hashSegment(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8]) // 16 hex chars
}

// RouteHash returns a bounded hash segment for a normalized route.
func RouteHash(route string) string {
	return hashSegment(strings.ToLower(strings.TrimSpace(route)))
}

func SubjectHash(subject string) string {
	return hashSegment(strings.TrimSpace(subject))
}

func subjectSegment(subject string) string {
	if strings.TrimSpace(subject) == "" {
		return ""
	}
	return ":" + SubjectHash(subject)
}

// TokenBucketKey builds the Redis key for token bucket state.
// Format: rl:tb:{tenant_id}:{route_hash}[:{subject_hash}]
func TokenBucketKey(key TenantRouteKey) string {
	return fmt.Sprintf("%s:tb:%s:%s%s", keyPrefix, key.TenantID, RouteHash(key.Route), subjectSegment(key.Subject))
}

// SlidingWindowKeys builds Redis keys for sliding window counters.
// Format: rl:sw:{tenant_id}:{route_hash}[:{subject_hash}]:{curr|prev|idx}
func SlidingWindowKeys(key TenantRouteKey) (curr, prev, idx string) {
	base := fmt.Sprintf("%s:sw:%s:%s%s", keyPrefix, key.TenantID, RouteHash(key.Route), subjectSegment(key.Subject))
	return base + ":curr", base + ":prev", base + ":idx"
}

// TenantRouteKey is the infrastructure view of a rate limit counter identity.
type TenantRouteKey struct {
	TenantID uuid.UUID
	Route    string
	Subject  string
}
