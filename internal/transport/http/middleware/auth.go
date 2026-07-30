package middleware

import (
	"net/http"

	"github.com/saurabh/distributed-rate-limiter/internal/application"
	"github.com/saurabh/distributed-rate-limiter/pkg/httputil"
)

type AuthMiddleware struct {
	auth *application.AuthService
}

func NewAuthMiddleware(auth *application.AuthService) *AuthMiddleware {
	return &AuthMiddleware{auth: auth}
}

func (m *AuthMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tc, err := m.auth.Authenticate(r.Context(), r.Header.Get("X-API-Key"))
		if err != nil {
			status, msg := application.MapAuthError(err)
			httputil.WriteError(w, status, msg)
			return
		}

		ctx := httputil.WithTenantID(r.Context(), tc.Tenant.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// PublicPaths skips auth for health and metrics endpoints.
func PublicPaths(path string) bool {
	switch path {
	case "/health/live", "/health/ready", "/metrics":
		return true
	default:
		return false
	}
}

func AuthOrPublic(auth *AuthMiddleware, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if PublicPaths(r.URL.Path) || IsAdminPath(r.URL.Path) || IsProjectPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		auth.Wrap(next).ServeHTTP(w, r)
	})
}
