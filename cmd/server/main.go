package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/saurabh/distributed-rate-limiter/internal/application"
	"github.com/saurabh/distributed-rate-limiter/internal/config"
	"github.com/saurabh/distributed-rate-limiter/internal/infrastructure/cache"
	"github.com/saurabh/distributed-rate-limiter/internal/infrastructure/postgres"
	redisinfra "github.com/saurabh/distributed-rate-limiter/internal/infrastructure/redis"
	"github.com/saurabh/distributed-rate-limiter/internal/observability"
	httptransport "github.com/saurabh/distributed-rate-limiter/internal/transport/http"
	"github.com/saurabh/distributed-rate-limiter/internal/transport/http/handlers"
	"github.com/saurabh/distributed-rate-limiter/internal/transport/http/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}

	logger := observability.NewLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	ctx := context.Background()

	pgPool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("postgres connect failed", "error", err)
		os.Exit(1)
	}
	defer pgPool.Close()

	redisClient, err := redisinfra.NewClient(ctx, cfg.RedisURL)
	if err != nil {
		slog.Error("redis connect failed", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	tenantRepo := postgres.NewTenantRepository(pgPool)
	apiKeyRepo := postgres.NewAPIKeyRepository(pgPool)
	ruleRepo := cache.NewRuleCache(postgres.NewRuleRepository(pgPool), cfg.RuleCacheTTL)

	authService := application.NewAuthService(apiKeyRepo, tenantRepo)
	rateLimiter, err := redisinfra.NewRateLimiter(redisClient)
	if err != nil {
		slog.Error("rate limiter init failed", "error", err)
		os.Exit(1)
	}
	checkService := application.NewCheckService(ruleRepo, application.NewRuleResolver(), rateLimiter)
	provisioning := application.NewProvisioningService(postgres.NewProjectRepository(pgPool))

	if cfg.AdminToken == "" {
		slog.Warn("ADMIN_TOKEN not set — the project write API is disabled")
	}

	deps := httptransport.Dependencies{
		HealthCheckers: []handlers.HealthChecker{
			postgres.PoolPinger{PingFn: func(ctx context.Context) error { return postgres.Ping(ctx, pgPool) }},
			redisinfra.NewClientPinger(redisClient),
		},
		Auth:    middleware.NewAuthMiddleware(authService),
		Tenants: tenantRepo,
		Rules:   ruleRepo,
		Check:   checkService,

		Projects:         provisioning,
		Console:          provisioning,
		AuthService:      authService,
		BehindProxy:      cfg.BehindProxy,
		AdminToken:       cfg.AdminToken,
		DashboardOrigins: cfg.DashboardOrigins,
	}

	server := &http.Server{
		Addr:    cfg.Addr(),
		Handler: httptransport.NewMux(deps),
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", cfg.Addr())
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		slog.Error("server error", "error", err)
		os.Exit(1)
	case sig := <-stop:
		slog.Info("shutdown signal received", "signal", sig.String())
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown failed", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}
