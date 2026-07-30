package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/saurabh/distributed-rate-limiter/pkg/httputil"
)

// AdminAuth blocks the request unless the Authorization header carries the
// admin token. If no token is configured it turns the routes off instead of
// leaving them open, since nothing else guards them.
func AdminAuth(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token == "" {
			httputil.WriteError(w, http.StatusServiceUnavailable, "admin_api_disabled")
			return
		}
		presented, hasScheme := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !hasScheme {
			httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(presented)), []byte(token)) != 1 {
			httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		next.ServeHTTP(w, r)
	})
}

const AdminPathPrefix = "/v1/admin/"

func IsAdminPath(path string) bool {
	return strings.HasPrefix(path, AdminPathPrefix)
}
