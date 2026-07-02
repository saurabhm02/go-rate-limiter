package entity

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Rule defines how requests are limited for a tenant and route pattern.
type Rule struct {
	ID             uuid.UUID
	TenantID       uuid.UUID
	RoutePattern   string
	Algorithm      Algorithm
	Enabled        bool
	LimitCount     int64   // sliding window: max requests in window
	WindowSeconds  int64   // sliding window: window size
	BucketCapacity int64   // token bucket: burst capacity
	RefillRate     float64 // token bucket: tokens per second
}

// Validate checks invariants for the rule's algorithm parameters.
func (r Rule) Validate() error {
	if r.TenantID == uuid.Nil {
		return fmt.Errorf("tenant id is required")
	}
	if strings.TrimSpace(r.RoutePattern) == "" {
		return fmt.Errorf("route pattern is required")
	}
	if !r.Algorithm.Valid() {
		return fmt.Errorf("invalid algorithm %q", r.Algorithm)
	}

	switch r.Algorithm {
	case AlgorithmSlidingWindow:
		if r.LimitCount <= 0 {
			return fmt.Errorf("limit count must be positive")
		}
		if r.WindowSeconds <= 0 {
			return fmt.Errorf("window seconds must be positive")
		}
	case AlgorithmTokenBucket:
		if r.BucketCapacity <= 0 {
			return fmt.Errorf("bucket capacity must be positive")
		}
		if r.RefillRate <= 0 {
			return fmt.Errorf("refill rate must be positive")
		}
	}
	return nil
}

// Matches reports whether the rule applies to the given request route.
func (r Rule) Matches(route string) bool {
	pattern := r.RoutePattern
	if pattern == "*" { // means match all routes
		return true
	}
	if strings.HasSuffix(pattern, "*") { // means match prefix
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(route, prefix)
	}
	return route == pattern // means exact match
}

// MatchKind classifies how a rule matches a route for precedence resolution.
type MatchKind int

const (
	MatchNone MatchKind = iota
	MatchDefault
	MatchPrefix
	MatchExact
)

// MatchKindFor returns how this rule matches the route.
func (r Rule) MatchKindFor(route string) MatchKind {
	if !r.Matches(route) {
		return MatchNone
	}
	if r.RoutePattern == "*" {
		return MatchDefault
	}
	if strings.HasSuffix(r.RoutePattern, "*") {
		return MatchPrefix
	}
	return MatchExact
}

// Specificity returns a score used to pick the best rule among matches.
// Higher score wins. Exact beats prefix; longer patterns beat shorter ones.
func (r Rule) Specificity(route string) int {
	switch r.MatchKindFor(route) {
	case MatchExact:
		return 1000 + len(r.RoutePattern)
	case MatchPrefix:
		return len(strings.TrimSuffix(r.RoutePattern, "*"))
	case MatchDefault:
		return 1
	default:
		return 0
	}
}

// Limit returns the configured limit for display headers, algorithm-dependent.
func (r Rule) Limit() int64 {
	if r.Algorithm == AlgorithmTokenBucket {
		return r.BucketCapacity
	}
	return r.LimitCount
}
