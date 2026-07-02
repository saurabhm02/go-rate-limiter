package httputil

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const tenantIDKey contextKey = "tenant_id"

// WithTenantID stores the authenticated tenant on the request context.
func WithTenantID(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

// TenantIDFromContext returns the tenant ID set by auth middleware.
func TenantIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(tenantIDKey).(uuid.UUID)
	return v, ok
}
