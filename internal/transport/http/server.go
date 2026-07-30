package httptransport

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/ports"
	"github.com/saurabh/distributed-rate-limiter/internal/transport/http/handlers"
	"github.com/saurabh/distributed-rate-limiter/internal/transport/http/middleware"
)

// Dependencies groups HTTP-layer collaborators.
type Dependencies struct {
	HealthCheckers []handlers.HealthChecker
	Auth           *middleware.AuthMiddleware
	Tenants        ports.TenantRepository
	Rules          ports.RuleRepository
	Check          handlers.RateLimitChecker

	Projects         handlers.ProjectCreator
	AdminToken       string
	DashboardOrigins []string
}

func NewMux(deps Dependencies) http.Handler {
	mux := http.NewServeMux()

	health := handlers.NewHealthHandler(deps.HealthCheckers...)
	mux.HandleFunc("GET /health/live", health.Live)
	mux.HandleFunc("GET /health/ready", health.Ready)

	if deps.Check != nil {
		check := handlers.NewCheckHandler(deps.Check)
		mux.Handle("POST /v1/check", check)
	}

	config := handlers.NewConfigHandler(deps.Tenants, deps.Rules)
	mux.HandleFunc("GET /v1/config/tenants", config.ListTenants)
	mux.HandleFunc("GET /v1/config/tenants/{id}/rules", config.ListTenantRules)

	if deps.Projects != nil && deps.AdminToken != "" {
		admin := handlers.NewAdminHandler(deps.Projects)
		guard := func(h http.HandlerFunc) http.Handler {
			return middleware.AdminAuth(deps.AdminToken, h)
		}
		mux.Handle("GET /v1/admin/projects", guard(admin.ListProjects))
		mux.Handle("POST /v1/admin/projects", guard(admin.CreateProject))
		mux.Handle("POST /v1/admin/projects/{id}/keys", guard(admin.AddKey))
		mux.Handle("DELETE /v1/admin/projects/{id}/keys/{keyId}", guard(admin.RevokeKey))
	}

	mux.Handle("GET /metrics", promhttp.Handler())

	var handler http.Handler = mux
	handler = middleware.Metrics(handler)
	handler = middleware.Logging(handler)
	if deps.Auth != nil {
		handler = middleware.AuthOrPublic(deps.Auth, handler)
	}
	if len(deps.DashboardOrigins) > 0 {
		handler = middleware.CORS(deps.DashboardOrigins, handler)
	}
	return handler
}
