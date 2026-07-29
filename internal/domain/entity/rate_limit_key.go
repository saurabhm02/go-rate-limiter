package entity

import "github.com/google/uuid"

// RateLimitKey identifies the counter namespace for a tenant and route.
// Infrastructure adapters (Redis) use this to build storage keys in M2.
type RateLimitKey struct {
	TenantID uuid.UUID
	Route    string
	Subject  string
}
