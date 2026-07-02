package httptransport

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/ports"
	"github.com/saurabh/distributed-rate-limiter/internal/infrastructure/postgres"
	"github.com/saurabh/distributed-rate-limiter/internal/transport/http/handlers"
	"github.com/saurabh/distributed-rate-limiter/internal/transport/http/middleware"
)

// Dependencies groups HTTP-layer collaborators.
type Dependencies struct {
	HealthCheckers []handlers.HealthChecker
	Auth           *middleware.AuthMiddleware
	Tenants        *postgres.TenantRepository
	Rules          ports.RuleRepository
	Check          handlers.RateLimitChecker
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

	mux.Handle("GET /metrics", promhttp.Handler())

	var handler http.Handler = mux
	handler = middleware.Metrics(handler)
	handler = middleware.Logging(handler)
	if deps.Auth != nil {
		handler = middleware.AuthOrPublic(deps.Auth, handler)
	}
	return handler
}
