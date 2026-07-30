package middleware

import (
	"net/http"
	"strings"

	"github.com/saurabh/distributed-rate-limiter/internal/application"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
	"github.com/saurabh/distributed-rate-limiter/pkg/httputil"
)

func ProjectAuth(auth *application.AuthService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, ok := bearerToken(r)
		if !ok {
			httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		tc, err := auth.Authenticate(r.Context(), raw)
		if err != nil {
			status, msg := application.MapAuthError(err)
			httputil.WriteError(w, status, msg)
			return
		}

		if !tc.APIKey.CanManageProject() {
			httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		ctx := httputil.WithTenantID(r.Context(), tc.Tenant.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func bearerToken(r *http.Request) (string, bool) {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		return "", false
	}
	token = strings.TrimSpace(token)
	return token, token != ""
}

func RequireCheckRole(auth *application.AuthService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tc, err := auth.Authenticate(r.Context(), r.Header.Get("X-API-Key"))
		if err != nil {
			status, msg := application.MapAuthError(err)
			httputil.WriteError(w, status, msg)
			return
		}
		if tc.APIKey.Role == entity.RoleAdmin {
			httputil.WriteError(w, http.StatusForbidden, "admin_key_cannot_check")
			return
		}
		next.ServeHTTP(w, r)
	})
}

const ProjectPathPrefix = "/v1/projects"

func IsProjectPath(path string) bool {
	return path == ProjectPathPrefix || strings.HasPrefix(path, ProjectPathPrefix+"/")
}
